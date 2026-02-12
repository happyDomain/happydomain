// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package app

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/usecase/checkresult"
	"git.happydns.org/happyDomain/model"
)

const (
	SchedulerCheckInterval     = 1 * time.Minute // How often to check for due tests
	SchedulerCleanupInterval   = 24 * time.Hour  // How often to clean up old executions
	SchedulerDiscoveryInterval = 1 * time.Hour   // How often to auto-discover new targets
	CheckExecutionTimeout      = 5 * time.Minute // Max time for a single check
	MaxRetries                 = 3               // Max retry attempts for failed checks
)

// Priority levels for test execution queue
const (
	PriorityOnDemand  = iota // On-demand tests (highest priority)
	PriorityOverdue          // Overdue scheduled tests
	PriorityScheduled        // Regular scheduled tests
)

// checkScheduler manages background test execution
type checkScheduler struct {
	cfg              *happydns.Options
	store            storage.Storage
	checkerUsecase   happydns.CheckerUsecase
	resultUsecase    *checkresult.CheckResultUsecase
	scheduleUsecase  *checkresult.CheckScheduleUsecase
	stop             chan struct{}   // closed to stop the main Run loop
	stopWorkers      chan struct{}   // closed to stop all workers simultaneously
	runNowChan       chan *queueItem // on-demand items routed through the main loop
	workAvail        chan struct{}   // non-blocking signals that queue has new work
	queue            *priorityQueue
	activeExecutions map[string]*activeExecution
	workers          []*worker
	mu               sync.RWMutex
	wg               sync.WaitGroup
	runtimeEnabled   bool
	running          bool
}

// activeExecution tracks a running test execution
type activeExecution struct {
	execution *happydns.CheckExecution
	cancel    context.CancelFunc
	startTime time.Time
}

// queueItem represents a test execution request in the queue
type queueItem struct {
	schedule  *happydns.CheckerSchedule
	execution *happydns.CheckExecution
	priority  int
	queuedAt  time.Time
	retries   int
}

// --- container/heap implementation for priorityQueue ---

// priorityHeap is the underlying heap, ordered by priority then arrival time.
type priorityHeap []*queueItem

func (h priorityHeap) Len() int { return len(h) }
func (h priorityHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority < h[j].priority
	}
	return h[i].queuedAt.Before(h[j].queuedAt)
}
func (h priorityHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *priorityHeap) Push(x any)   { *h = append(*h, x.(*queueItem)) }
func (h *priorityHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil // avoid memory leak
	*h = old[:n-1]
	return x
}

// priorityQueue is a thread-safe min-heap of queueItems.
type priorityQueue struct {
	h  priorityHeap
	mu sync.Mutex
}

func newPriorityQueue() *priorityQueue {
	pq := &priorityQueue{}
	heap.Init(&pq.h)
	return pq
}

// Push adds an item to the queue.
func (q *priorityQueue) Push(item *queueItem) {
	q.mu.Lock()
	defer q.mu.Unlock()
	heap.Push(&q.h, item)
}

// Pop removes and returns the highest-priority item, or nil if empty.
func (q *priorityQueue) Pop() *queueItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.h.Len() == 0 {
		return nil
	}
	return heap.Pop(&q.h).(*queueItem)
}

// Len returns the queue length.
func (q *priorityQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.h.Len()
}

// worker processes tests from the queue
type worker struct {
	id        int
	scheduler *checkScheduler
}

// disabledScheduler is a no-op implementation used when scheduler is disabled
type disabledScheduler struct{}

func (d *disabledScheduler) Run()   {}
func (d *disabledScheduler) Close() {}

// TriggerOnDemandCheck returns an error indicating the scheduler is disabled
func (d *disabledScheduler) TriggerOnDemandCheck(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, userId happydns.Identifier, options happydns.CheckerOptions) (happydns.Identifier, error) {
	return happydns.Identifier{}, fmt.Errorf("test scheduler is disabled in configuration")
}

// GetSchedulerStatus returns a status indicating the scheduler is disabled
func (d *disabledScheduler) GetSchedulerStatus() happydns.SchedulerStatus {
	return happydns.SchedulerStatus{
		ConfigEnabled:  false,
		RuntimeEnabled: false,
		Running:        false,
	}
}

// SetEnabled returns an error since the scheduler is disabled in configuration
func (d *disabledScheduler) SetEnabled(enabled bool) error {
	return fmt.Errorf("scheduler is disabled in configuration, cannot enable at runtime")
}

// RescheduleUpcomingChecks returns an error since the scheduler is disabled
func (d *disabledScheduler) RescheduleUpcomingChecks() (int, error) {
	return 0, fmt.Errorf("test scheduler is disabled in configuration")
}

// newCheckScheduler creates a new test scheduler
func newCheckScheduler(
	cfg *happydns.Options,
	store storage.Storage,
	checkerUsecase happydns.CheckerUsecase,
) *checkScheduler {
	numWorkers := cfg.TestWorkers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	scheduler := &checkScheduler{
		cfg:              cfg,
		store:            store,
		checkerUsecase:   checkerUsecase,
		resultUsecase:    checkresult.NewCheckResultUsecase(store, cfg),
		scheduleUsecase:  checkresult.NewCheckScheduleUsecase(store, cfg, store, checkerUsecase),
		stop:             make(chan struct{}),
		stopWorkers:      make(chan struct{}),
		runNowChan:       make(chan *queueItem, 100),
		workAvail:        make(chan struct{}, numWorkers),
		queue:            newPriorityQueue(),
		activeExecutions: make(map[string]*activeExecution),
		workers:          make([]*worker, numWorkers),
		runtimeEnabled:   true,
	}

	for i := 0; i < numWorkers; i++ {
		scheduler.workers[i] = &worker{
			id:        i,
			scheduler: scheduler,
		}
	}

	return scheduler
}

// enqueue pushes an item to the priority queue and wakes one idle worker.
func (s *checkScheduler) enqueue(item *queueItem) {
	s.queue.Push(item)
	select {
	case s.workAvail <- struct{}{}:
	default:
		// All workers are already busy or already notified; they will drain
		// the queue on their own after finishing the current item.
	}
}

// Close stops the scheduler and waits for all workers to finish.
func (s *checkScheduler) Close() {
	log.Println("Stopping test scheduler...")

	// Unblock the main Run loop.
	close(s.stop)

	// Unblock all workers simultaneously.
	close(s.stopWorkers)

	// Cancel all active test executions.
	s.mu.Lock()
	for _, exec := range s.activeExecutions {
		exec.cancel()
	}
	s.mu.Unlock()

	// Wait for all workers to finish their current item.
	s.wg.Wait()

	log.Println("Check scheduler stopped")
}

// Run starts the scheduler main loop. It must not be called more than once.
func (s *checkScheduler) Run() {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	log.Printf("Starting test scheduler with %d workers...\n", len(s.workers))

	// Reschedule overdue tests before starting workers so that tests missed
	// during a server suspend or shutdown are spread into the near future
	// instead of all firing at once.
	if n, err := s.scheduleUsecase.RescheduleOverdueChecks(); err != nil {
		log.Printf("Warning: failed to reschedule overdue tests: %v\n", err)
	} else if n > 0 {
		log.Printf("Rescheduled %d overdue test(s) into the near future\n", n)
	}

	// Start workers
	for _, w := range s.workers {
		s.wg.Add(1)
		go w.run(&s.wg)
	}

	// Main scheduling loop
	checkTicker := time.NewTicker(SchedulerCheckInterval)
	cleanupTicker := time.NewTicker(SchedulerCleanupInterval)
	discoveryTicker := time.NewTicker(SchedulerDiscoveryInterval)
	defer checkTicker.Stop()
	defer cleanupTicker.Stop()
	defer discoveryTicker.Stop()

	// Initial discovery: create default schedules for all existing targets
	if err := s.scheduleUsecase.DiscoverAndEnsureSchedules(); err != nil {
		log.Printf("Warning: schedule discovery encountered errors: %v\n", err)
	}
	// Initial check
	s.checkSchedules()

	for {
		select {
		case <-checkTicker.C:
			s.checkSchedules()

		case <-cleanupTicker.C:
			s.cleanup()

		case <-discoveryTicker.C:
			if err := s.scheduleUsecase.DiscoverAndEnsureSchedules(); err != nil {
				log.Printf("Warning: schedule discovery encountered errors: %v\n", err)
			}

		case item := <-s.runNowChan:
			s.enqueue(item)

		case <-s.stop:
			return
		}
	}
}

// checkSchedules checks for due tests and queues them
func (s *checkScheduler) checkSchedules() {
	s.mu.RLock()
	enabled := s.runtimeEnabled
	s.mu.RUnlock()
	if !enabled {
		return
	}

	dueSchedules, err := s.scheduleUsecase.ListDueSchedules()
	if err != nil {
		log.Printf("Error listing due schedules: %v\n", err)
		return
	}

	now := time.Now()
	for _, schedule := range dueSchedules {
		// Determine priority based on how overdue the test is
		priority := PriorityScheduled
		if schedule.NextRun.Add(schedule.Interval).Before(now) {
			priority = PriorityOverdue
		}

		// Create execution record
		execution := &happydns.CheckExecution{
			ScheduleId:  &schedule.Id,
			CheckerName: schedule.CheckerName,
			OwnerId:     schedule.OwnerId,
			TargetType:  schedule.TargetType,
			TargetId:    schedule.TargetId,
			Status:      happydns.CheckExecutionPending,
			StartedAt:   time.Now(),
			Options:     schedule.Options,
		}

		if err := s.resultUsecase.CreateCheckExecution(execution); err != nil {
			log.Printf("Error creating execution for schedule %s: %v\n", schedule.Id.String(), err)
			continue
		}

		s.enqueue(&queueItem{
			schedule:  schedule,
			execution: execution,
			priority:  priority,
			queuedAt:  now,
			retries:   0,
		})
	}

	// Mark scheduler run
	if err := s.store.CheckSchedulerRun(); err != nil {
		log.Printf("Error marking scheduler run: %v\n", err)
	}
}

// TriggerOnDemandCheck triggers an immediate test execution.
// It creates the execution record synchronously (so the caller gets an ID back)
// and then routes the item through runNowChan so the main loop controls
// all queue insertions.
func (s *checkScheduler) TriggerOnDemandCheck(checkerName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, ownerId happydns.Identifier, options happydns.CheckerOptions) (happydns.Identifier, error) {
	schedule := &happydns.CheckerSchedule{
		CheckerName: checkerName,
		OwnerId:     ownerId,
		TargetType:  targetType,
		TargetId:    targetId,
		Interval:    0, // On-demand, no interval
		Enabled:     true,
		Options:     options,
	}

	execution := &happydns.CheckExecution{
		ScheduleId:  nil,
		CheckerName: checkerName,
		OwnerId:     ownerId,
		TargetType:  targetType,
		TargetId:    targetId,
		Status:      happydns.CheckExecutionPending,
		StartedAt:   time.Now(),
		Options:     options,
	}

	if err := s.resultUsecase.CreateCheckExecution(execution); err != nil {
		return happydns.Identifier{}, err
	}

	item := &queueItem{
		schedule:  schedule,
		execution: execution,
		priority:  PriorityOnDemand,
		queuedAt:  time.Now(),
		retries:   0,
	}

	// Route through the main loop when possible; fall back to direct enqueue
	// if the channel is full so that the caller never blocks.
	select {
	case s.runNowChan <- item:
	default:
		s.enqueue(item)
	}

	return execution.Id, nil
}

// GetSchedulerStatus returns a snapshot of the current scheduler state
func (s *checkScheduler) GetSchedulerStatus() happydns.SchedulerStatus {
	s.mu.RLock()
	activeCount := len(s.activeExecutions)
	running := s.running
	runtimeEnabled := s.runtimeEnabled
	s.mu.RUnlock()

	nextSchedules, _ := s.scheduleUsecase.ListUpcomingSchedules(20)

	return happydns.SchedulerStatus{
		ConfigEnabled:  !s.cfg.DisableScheduler,
		RuntimeEnabled: runtimeEnabled,
		Running:        running,
		WorkerCount:    len(s.workers),
		QueueSize:      s.queue.Len(),
		ActiveCount:    activeCount,
		NextSchedules:  nextSchedules,
	}
}

// SetEnabled enables or disables the scheduler at runtime
func (s *checkScheduler) SetEnabled(enabled bool) error {
	s.mu.Lock()
	wasEnabled := s.runtimeEnabled
	s.runtimeEnabled = enabled
	s.mu.Unlock()

	if enabled && !wasEnabled {
		// Spread out any overdue tests to avoid a thundering herd, then
		// immediately enqueue whatever is now due.
		if n, err := s.scheduleUsecase.RescheduleOverdueChecks(); err != nil {
			log.Printf("Warning: failed to reschedule overdue tests on re-enable: %v\n", err)
		} else if n > 0 {
			log.Printf("Rescheduled %d overdue test(s) after scheduler re-enable\n", n)
		}
		s.checkSchedules()
	}

	return nil
}

// RescheduleUpcomingChecks randomizes the next run time of all enabled schedules
// within their respective intervals, delegating to the schedule usecase.
func (s *checkScheduler) RescheduleUpcomingChecks() (int, error) {
	return s.scheduleUsecase.RescheduleUpcomingChecks()
}

// cleanup removes old execution records and expired test results
func (s *checkScheduler) cleanup() {
	log.Println("Running scheduler cleanup...")

	// Delete completed/failed execution records older than 7 days
	if err := s.resultUsecase.DeleteCompletedExecutions(7 * 24 * time.Hour); err != nil {
		log.Printf("Error cleaning up old executions: %v\n", err)
	}

	// Delete test results older than the configured retention period
	if err := s.resultUsecase.CleanupOldResults(); err != nil {
		log.Printf("Error cleaning up old test results: %v\n", err)
	}

	log.Println("Scheduler cleanup complete")
}

// run is the worker's main loop. It drains the queue eagerly and waits for a
// workAvail signal when idle, rather than sleeping on a fixed timer.
func (w *worker) run(wg *sync.WaitGroup) {
	defer wg.Done()

	log.Printf("Worker %d started\n", w.id)

	for {
		// Drain: try to grab work before blocking.
		if item := w.scheduler.queue.Pop(); item != nil {
			w.executeCheck(item)
			continue
		}

		// Queue is empty; wait for new work or a stop signal.
		select {
		case <-w.scheduler.workAvail:
			// Loop back to attempt a Pop.
		case <-w.scheduler.stopWorkers:
			log.Printf("Worker %d stopped\n", w.id)
			return
		}
	}
}

// executeCheck runs a checker and stores the result
func (w *worker) executeCheck(item *queueItem) {
	ctx, cancel := context.WithTimeout(context.Background(), CheckExecutionTimeout)
	defer cancel()

	execution := item.execution
	schedule := item.schedule

	// Always update schedule NextRun after execution, whether it succeeds or fails.
	// This prevents the schedule from being re-queued on the next tick if the test fails.
	if item.execution.ScheduleId != nil {
		defer func() {
			if err := w.scheduler.scheduleUsecase.UpdateScheduleAfterRun(*item.execution.ScheduleId); err != nil {
				log.Printf("Worker %d: Error updating schedule after run: %v\n", w.id, err)
			}
		}()
	}

	// Mark execution as running
	execution.Status = happydns.CheckExecutionRunning
	if err := w.scheduler.resultUsecase.UpdateCheckExecution(execution); err != nil {
		log.Printf("Worker %d: Error updating execution status: %v\n", w.id, err)
		_ = w.scheduler.resultUsecase.FailCheckExecution(execution.Id, err.Error())
		return
	}

	// Track active execution
	w.scheduler.mu.Lock()
	w.scheduler.activeExecutions[execution.Id.String()] = &activeExecution{
		execution: execution,
		cancel:    cancel,
		startTime: time.Now(),
	}
	w.scheduler.mu.Unlock()

	defer func() {
		w.scheduler.mu.Lock()
		delete(w.scheduler.activeExecutions, execution.Id.String())
		w.scheduler.mu.Unlock()
	}()

	// Get the checker
	checker, err := w.scheduler.checkerUsecase.GetChecker(schedule.CheckerName)
	if err != nil {
		errMsg := fmt.Sprintf("checker not found: %s - %v", schedule.CheckerName, err)
		log.Printf("Worker %d: %s\n", w.id, errMsg)
		_ = w.scheduler.resultUsecase.FailCheckExecution(execution.Id, errMsg)
		return
	}

	// Merge options: global defaults < user opts < domain/service opts < schedule/on-demand opts < auto-fill
	var mergedOptions happydns.CheckerOptions

	var domainId, serviceId *happydns.Identifier
	switch schedule.TargetType {
	case happydns.CheckScopeDomain:
		domainId = &schedule.TargetId
	case happydns.CheckScopeService:
		serviceId = &schedule.TargetId
	}
	var mergeErr error
	mergedOptions, mergeErr = w.scheduler.checkerUsecase.BuildMergedCheckerOptions(schedule.CheckerName, &schedule.OwnerId, domainId, serviceId, schedule.Options)
	if mergeErr != nil {
		// Non-fatal: fall back to schedule-only options
		log.Printf("Worker %d: warning, could not prepare checker options for %s: %v\n", w.id, schedule.CheckerName, mergeErr)
		mergedOptions = schedule.Options
	}

	// Prepare metadata
	meta := make(map[string]string)
	meta["target_type"] = schedule.TargetType.String()
	meta["target_id"] = schedule.TargetId.String()

	// Run the test
	startTime := time.Now()
	resultChan := make(chan *happydns.CheckResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("checker panicked: %v", r)
			}
		}()
		result, err := checker.RunCheck(mergedOptions, meta)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for result or timeout
	var checkResult *happydns.CheckResult
	var testErr error

	select {
	case checkResult = <-resultChan:
		// Check completed successfully
	case testErr = <-errorChan:
		// Check returned an error
	case <-ctx.Done():
		// Timeout
		testErr = fmt.Errorf("test execution timeout after %v", CheckExecutionTimeout)
	}

	duration := time.Since(startTime)

	// Store the result
	result := &happydns.CheckResult{
		CheckerName:    schedule.CheckerName,
		CheckType:      schedule.TargetType,
		TargetId:       schedule.TargetId,
		OwnerId:        schedule.OwnerId,
		ExecutedAt:     time.Now(),
		ScheduledCheck: item.execution.ScheduleId != nil,
		Options:        schedule.Options,
		Duration:       duration,
	}

	if testErr != nil {
		result.Status = happydns.CheckResultStatusUnknown
		result.StatusLine = "Check execution failed"
		result.Error = testErr.Error()
	} else if checkResult != nil {
		result.Status = checkResult.Status
		result.StatusLine = checkResult.StatusLine
		result.Report = checkResult.Report
	} else {
		result.Status = happydns.CheckResultStatusUnknown
		result.StatusLine = "Unknown error"
		result.Error = "No result or error returned from check"
	}

	// Save the result
	if err := w.scheduler.resultUsecase.CreateCheckResult(result); err != nil {
		log.Printf("Worker %d: Error saving test result: %v\n", w.id, err)
		_ = w.scheduler.resultUsecase.FailCheckExecution(execution.Id, err.Error())
		return
	}

	// Complete the execution
	if err := w.scheduler.resultUsecase.CompleteCheckExecution(execution.Id, result.Id); err != nil {
		log.Printf("Worker %d: Error completing execution: %v\n", w.id, err)
		return
	}

	log.Printf("Worker %d: Completed test %s for target %s (status: %d, duration: %v)\n",
		w.id, schedule.CheckerName, schedule.TargetId.String(), result.Status, duration)
}
