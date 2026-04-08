// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package checker

import (
	"container/heap"
	"context"
	"hash/fnv"
	"log"
	"slices"
	"sync"
	"time"

	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/internal/metrics"
	"git.happydns.org/happyDomain/model"
)

const (
	minSpacing       = 2 * time.Second
	maxCatchUpWindow = 10 * time.Minute
	defaultInterval  = 24 * time.Hour
)

// SchedulerJob represents a single scheduled checker execution.
type SchedulerJob struct {
	CheckerID string               `json:"checkerID"`
	Target    happydns.CheckTarget `json:"target"`
	PlanID    *happydns.Identifier `json:"planID" swaggertype:"string"`
	Interval  time.Duration        `json:"interval" swaggertype:"integer"`
	NextRun   time.Time            `json:"nextRun"`
	index     int                  // heap index
}

// SchedulerQueue is a min-heap of SchedulerJobs sorted by NextRun.
type SchedulerQueue []*SchedulerJob

func (q SchedulerQueue) Len() int           { return len(q) }
func (q SchedulerQueue) Less(i, j int) bool { return q[i].NextRun.Before(q[j].NextRun) }
func (q SchedulerQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *SchedulerQueue) Push(x any) {
	n := len(*q)
	job := x.(*SchedulerJob)
	job.index = n
	*q = append(*q, job)
}

func (q *SchedulerQueue) Pop() any {
	old := *q
	n := len(old)
	job := old[n-1]
	old[n-1] = nil
	job.index = -1
	*q = old[:n-1]
	return job
}

func (q *SchedulerQueue) Peek() *SchedulerJob {
	if len(*q) == 0 {
		return nil
	}
	return (*q)[0]
}

// SchedulerStatus holds a snapshot of the scheduler's current state.
type SchedulerStatus struct {
	Running  bool            `json:"running"`
	JobCount int             `json:"job_count"`
	NextJobs []*SchedulerJob `json:"next_jobs,omitempty"`
}

// Scheduler manages periodic execution of checkers.
type Scheduler struct {
	queue          SchedulerQueue
	jobKeys        map[string]bool
	engine         happydns.CheckerEngine
	planStore      CheckPlanStorage
	domainStore    DomainLister
	zoneStore      ZoneGetter
	stateStore     SchedulerStateStorage
	cancel         context.CancelFunc
	wake           chan struct{}
	done           chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex
	running        bool
	ctx            context.Context
	maxConcurrency int

	// gate, if set, is consulted before launching each job. Returning false
	// causes the scheduler to skip (and reschedule) the job, e.g. when the
	// owning user is paused, has been inactive for too long, or has
	// exhausted their daily check quota. The job's interval is passed so
	// the gate can make interval-aware decisions (e.g. throttle short
	// intervals before long ones when approaching a budget cap).
	gate func(target happydns.CheckTarget, interval time.Duration) bool

	// onExecute, if set, is invoked after each scheduled execution is
	// successfully created. Used to increment per-user usage counters.
	// Only called for scheduler-driven executions; manual API triggers do
	// not call this.
	onExecute func(target happydns.CheckTarget)
}

// NewScheduler creates a new Scheduler. The optional gate function, if
// non-nil, is consulted before launching each job; returning false causes
// the scheduler to skip (and reschedule) the job. The optional onExecute
// callback, if non-nil, is invoked after each execution is successfully
// created so callers can update per-user counters.
func NewScheduler(
	engine happydns.CheckerEngine,
	maxConcurrency int,
	planStore CheckPlanStorage,
	domainStore DomainLister,
	zoneStore ZoneGetter,
	stateStore SchedulerStateStorage,
	gate func(target happydns.CheckTarget, interval time.Duration) bool,
	onExecute func(target happydns.CheckTarget),
) *Scheduler {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}
	s := &Scheduler{
		engine:         engine,
		planStore:      planStore,
		domainStore:    domainStore,
		zoneStore:      zoneStore,
		stateStore:     stateStore,
		jobKeys:        make(map[string]bool),
		wake:           make(chan struct{}, 1),
		maxConcurrency: maxConcurrency,
		gate:           gate,
		onExecute:      onExecute,
	}
	// The scheduler queue depth is exposed via a Prometheus GaugeFunc that
	// reads the live queue length at scrape time. This avoids having to call
	// gauge.Set after every queue mutation site (Push/Pop/Init/buildQueue/…).
	metrics.RegisterSchedulerQueueDepth(s.queueDepthForMetrics)
	return s
}

// queueDepthForMetrics returns the current queue length under the read lock.
// It is invoked from the Prometheus scrape goroutine.
func (s *Scheduler) queueDepthForMetrics() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return float64(s.queue.Len())
}

// Start begins the scheduler loop in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.ctx = ctx
	s.cancel = cancel
	s.running = true
	s.done = make(chan struct{})
	s.buildQueue()
	s.spreadOverdueJobs()
	s.mu.Unlock()
	go s.run(ctx)
}

// Stop halts the scheduler and waits for in-flight workers to finish.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	s.running = false
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	// Drop the queue-depth accessor so a stopped scheduler does not keep its
	// closure (and the captured queue) reachable for the lifetime of the
	// process. This is essential in tests that spin schedulers up and down.
	metrics.RegisterSchedulerQueueDepth(nil)
}

// GetStatus returns a snapshot of the scheduler's current state.
func (s *Scheduler) GetStatus() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := SchedulerStatus{
		Running:  s.running,
		JobCount: s.queue.Len(),
	}

	n := min(20, s.queue.Len())
	if n > 0 {
		tmp := make(SchedulerQueue, s.queue.Len())
		copy(tmp, s.queue)
		for i, job := range tmp {
			cp := *job
			cp.index = i
			tmp[i] = &cp
		}
		status.NextJobs = make([]*SchedulerJob, 0, n)
		for range n {
			status.NextJobs = append(status.NextJobs, heap.Pop(&tmp).(*SchedulerJob))
		}
	}

	return status
}

// SetEnabled starts or stops the scheduler. The provided ctx is used as the
// parent context for the new scheduler loop when enabled is true.
func (s *Scheduler) SetEnabled(ctx context.Context, enabled bool) error {
	s.mu.RLock()
	wasRunning := s.running
	s.mu.RUnlock()

	if wasRunning {
		s.Stop()
	}
	if enabled {
		s.Start(ctx)
	}
	return nil
}

// RebuildQueue rebuilds the scheduler queue and returns the new job count.
func (s *Scheduler) RebuildQueue() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buildQueue()
	s.spreadOverdueJobs()
	return s.queue.Len()
}

func (s *Scheduler) run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Scheduler: panic in run loop: %v", r)
		}
		s.wg.Wait()
		close(s.done)
	}()

	sem := make(chan struct{}, s.maxConcurrency)

	for {
		ready, cancelled := s.waitForNextJob(ctx)
		if cancelled {
			return
		}
		if !ready {
			continue
		}

		s.mu.Lock()
		if s.queue.Len() == 0 {
			s.mu.Unlock()
			continue
		}
		job := heap.Pop(&s.queue).(*SchedulerJob)
		gate := s.gate
		s.mu.Unlock()

		// Honour the user-level gate before doing any work.
		if gate != nil && !gate(job.Target, job.Interval) {
			// log.Printf("Scheduler: skipping checker %s on %s (gated by user policy)", job.CheckerID, job.Target.String())
			s.rescheduleJob(job)
			continue
		}

		// Find plan if applicable.
		var plan *happydns.CheckPlan
		if job.PlanID != nil {
			p, err := s.planStore.GetCheckPlan(*job.PlanID)
			if err == nil {
				plan = p
			}
		}

		if !s.acquireWorkerSlot(ctx, sem, job) {
			return
		}

		s.wg.Add(1)
		go s.executeJob(ctx, job, plan, sem)

		// Advance to next cycle and re-enqueue.
		s.rescheduleJob(job)
	}
}

// waitForNextJob blocks until the next job is due, the queue is woken, or the
// context is cancelled. It returns (true, false) when a job is ready to run,
// (false, false) when the loop should re-evaluate (wake or queue rebuild), and
// (false, true) when the context is done.
func (s *Scheduler) waitForNextJob(ctx context.Context) (ready, cancelled bool) {
	s.mu.RLock()
	qLen := s.queue.Len()
	s.mu.RUnlock()

	if qLen == 0 {
		select {
		case <-ctx.Done():
			return false, true
		case <-s.wake:
			return false, false
		case <-time.After(1 * time.Minute):
			s.mu.Lock()
			s.buildQueue()
			s.mu.Unlock()
			return false, false
		}
	}

	s.mu.RLock()
	next := s.queue.Peek()
	var delay time.Duration
	if next != nil {
		delay = time.Until(next.NextRun)
	}
	s.mu.RUnlock()

	if delay <= 0 {
		return true, false
	}

	timer := time.NewTimer(delay)
	select {
	case <-ctx.Done():
		timer.Stop()
		return false, true
	case <-s.wake:
		timer.Stop()
		return false, false
	case <-timer.C:
		return true, false
	}
}

// acquireWorkerSlot blocks until a concurrency slot is available or the context
// is cancelled. Returns false when the context is done.
func (s *Scheduler) acquireWorkerSlot(ctx context.Context, sem chan struct{}, job *SchedulerJob) bool {
	select {
	case sem <- struct{}{}:
		return true
	default:
		log.Printf("Scheduler: all %d workers busy, waiting for a slot (checker %s on %s)", s.maxConcurrency, job.CheckerID, job.Target.String())
		select {
		case sem <- struct{}{}:
			return true
		case <-ctx.Done():
			return false
		}
	}
}

// executeJob runs a single checker execution in its own goroutine.
// The caller must have incremented s.wg and acquired a slot from sem.
func (s *Scheduler) executeJob(ctx context.Context, job *SchedulerJob, plan *happydns.CheckPlan, sem chan struct{}) {
	defer func() { <-sem; s.wg.Done() }()
	metrics.SchedulerActiveWorkers.Inc()
	checkStart := time.Now()
	defer func() {
		metrics.SchedulerActiveWorkers.Dec()
		metrics.SchedulerCheckDuration.WithLabelValues(job.CheckerID).Observe(time.Since(checkStart).Seconds())
		if r := recover(); r != nil {
			log.Printf("Scheduler: panic in worker for checker %s on %s: %v", job.CheckerID, job.Target.String(), r)
		}
	}()
	log.Printf("Scheduler: running checker %s on %s", job.CheckerID, job.Target.String())
	exec, err := s.engine.CreateExecution(job.CheckerID, job.Target, plan)
	if err != nil {
		metrics.SchedulerChecksTotal.WithLabelValues(job.CheckerID, "error").Inc()
		log.Printf("Scheduler: checker %s on %s failed to create execution: %v", job.CheckerID, job.Target.String(), err)
		return
	}
	if s.onExecute != nil {
		s.onExecute(job.Target)
	}
	_, err = s.engine.RunExecution(ctx, exec, plan, nil)
	status := "success"
	if err != nil {
		status = "error"
		log.Printf("Scheduler: checker %s on %s failed: %v", job.CheckerID, job.Target.String(), err)
	}
	metrics.SchedulerChecksTotal.WithLabelValues(job.CheckerID, status).Inc()
	if s.stateStore != nil {
		if err := s.stateStore.SetLastSchedulerRun(time.Now()); err != nil {
			log.Printf("Scheduler: failed to persist last run time: %v", err)
		}
	}
}

// rescheduleJob advances job.NextRun past the current time, adds jitter,
// and pushes the job back onto the scheduler queue.
func (s *Scheduler) rescheduleJob(job *SchedulerJob) {
	now := time.Now()
	for job.NextRun.Before(now) {
		job.NextRun = job.NextRun.Add(job.Interval)
	}
	job.NextRun = job.NextRun.Add(computeJitter(job.CheckerID, job.Target.String(), job.NextRun, job.Interval))
	key := job.CheckerID + "|" + job.Target.String()
	s.mu.Lock()
	heap.Push(&s.queue, job)
	s.jobKeys[key] = true
	s.mu.Unlock()
}

func (s *Scheduler) buildQueue() {
	s.queue = s.queue[:0]
	s.jobKeys = make(map[string]bool)

	var lastRun time.Time
	if s.stateStore != nil {
		if t, err := s.stateStore.GetLastSchedulerRun(); err != nil {
			log.Printf("Scheduler: failed to read last run time: %v", err)
		} else {
			lastRun = t
		}
	}

	checkers := checkerPkg.GetCheckers()
	plans, err := s.loadAllPlans()
	if err != nil {
		log.Printf("Scheduler: failed to load plans, skipping queue build: %v", err)
		return
	}

	disabledSet, planMap := buildPlanIndex(plans)

	// Collect checkers by scope for efficient iteration.
	var domainCheckers, serviceCheckers []struct {
		id  string
		def *happydns.CheckerDefinition
	}
	for checkerID, def := range checkers {
		if def.Availability.ApplyToDomain {
			domainCheckers = append(domainCheckers, struct {
				id  string
				def *happydns.CheckerDefinition
			}{checkerID, def})
		}
		if def.Availability.ApplyToService {
			serviceCheckers = append(serviceCheckers, struct {
				id  string
				def *happydns.CheckerDefinition
			}{checkerID, def})
		}
	}

	// Auto-discovery: enumerate all domains and schedule applicable checkers.
	domains := s.loadAllDomains()
	for _, domain := range domains {
		uid := domain.Owner
		did := domain.Id
		domainTarget := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

		for _, c := range domainCheckers {
			s.enqueueJob(c.id, c.def, domainTarget, disabledSet, planMap, lastRun)
		}

		// Service-level discovery: load the latest zone and match services.
		if len(serviceCheckers) > 0 {
			services := s.loadDomainServices(domain)
			for _, svc := range services {
				sid := svc.Id
				svcTarget := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String(), ServiceId: sid.String(), ServiceType: svc.Type}

				for _, c := range serviceCheckers {
					if len(c.def.Availability.LimitToServices) > 0 && !slices.Contains(c.def.Availability.LimitToServices, svc.Type) {
						continue
					}
					s.enqueueJob(c.id, c.def, svcTarget, disabledSet, planMap, lastRun)
				}
			}
		}
	}
}

// NotifyDomainChange incrementally adds scheduler jobs for a domain
// without rebuilding the entire queue. Call this after a domain is
// created or its zone is imported/published.
func (s *Scheduler) NotifyDomainChange(domain *happydns.Domain) {
	checkers := checkerPkg.GetCheckers()

	// Load plans relevant to this domain.
	uid := domain.Owner
	did := domain.Id
	domainTarget := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	plans, err := s.planStore.ListCheckPlansByTarget(domainTarget)
	if err != nil {
		log.Printf("Scheduler: NotifyDomainChange: failed to load plans: %v", err)
	}
	disabledSet, planMap := buildPlanIndex(plans)

	// Load services outside the lock to avoid holding the mutex during I/O.
	services := s.loadDomainServices(domain)

	// Build the set of desired job keys for this domain so we can detect stale entries.
	wantKeys := make(map[string]bool)
	didStr := did.String()
	for checkerID, def := range checkers {
		if def.Availability.ApplyToDomain {
			key := checkerID + "|" + domainTarget.String()
			if !disabledSet[key] {
				wantKeys[key] = true
			}
		}
		if def.Availability.ApplyToService {
			for _, svc := range services {
				if len(def.Availability.LimitToServices) > 0 && !slices.Contains(def.Availability.LimitToServices, svc.Type) {
					continue
				}
				svcTarget := happydns.CheckTarget{UserId: uid.String(), DomainId: didStr, ServiceId: svc.Id.String(), ServiceType: svc.Type}
				key := checkerID + "|" + svcTarget.String()
				if !disabledSet[key] {
					wantKeys[key] = true
				}
			}
		}
	}

	var added, removed int
	s.mu.Lock()

	// Remove stale jobs for this domain that are no longer wanted.
	for i := 0; i < len(s.queue); {
		job := s.queue[i]
		if job.Target.DomainId == didStr {
			key := job.CheckerID + "|" + job.Target.String()
			if !wantKeys[key] {
				delete(s.jobKeys, key)
				s.queue[i] = s.queue[len(s.queue)-1]
				s.queue[len(s.queue)-1] = nil
				s.queue = s.queue[:len(s.queue)-1]
				removed++
				continue
			}
		}
		i++
	}
	if removed > 0 {
		heap.Init(&s.queue)
	}

	// Add new jobs for this domain.
	for checkerID, def := range checkers {
		if def.Availability.ApplyToDomain {
			if s.enqueueJob(checkerID, def, domainTarget, disabledSet, planMap, time.Time{}) {
				added++
			}
		}

		if def.Availability.ApplyToService {
			for _, svc := range services {
				if len(def.Availability.LimitToServices) > 0 && !slices.Contains(def.Availability.LimitToServices, svc.Type) {
					continue
				}
				sid := svc.Id
				svcTarget := happydns.CheckTarget{UserId: uid.String(), DomainId: didStr, ServiceId: sid.String(), ServiceType: svc.Type}
				if s.enqueueJob(checkerID, def, svcTarget, disabledSet, planMap, time.Time{}) {
					added++
				}
			}
		}
	}

	s.mu.Unlock()

	if added > 0 || removed > 0 {
		log.Printf("Scheduler: NotifyDomainChange(%s): added %d jobs, removed %d stale jobs", domain.DomainName, added, removed)
		// Wake the run loop so it re-evaluates the queue head.
		select {
		case s.wake <- struct{}{}:
		default:
		}
	}
}

// NotifyDomainRemoved removes all scheduler jobs for the given domain.
func (s *Scheduler) NotifyDomainRemoved(domainID happydns.Identifier) {
	s.mu.Lock()
	n := 0
	for i := 0; i < len(s.queue); {
		job := s.queue[i]
		if job.Target.DomainId == domainID.String() {
			key := job.CheckerID + "|" + job.Target.String()
			delete(s.jobKeys, key)
			// Swap with last and shrink.
			s.queue[i] = s.queue[len(s.queue)-1]
			s.queue[len(s.queue)-1] = nil
			s.queue = s.queue[:len(s.queue)-1]
			n++
		} else {
			i++
		}
	}
	if n > 0 {
		heap.Init(&s.queue)
	}
	s.mu.Unlock()

	if n > 0 {
		log.Printf("Scheduler: NotifyDomainRemoved(%s): removed %d jobs", domainID, n)
	}
}

// buildPlanIndex builds disabled and plan lookup maps from a slice of plans.
func buildPlanIndex(plans []*happydns.CheckPlan) (disabledSet map[string]bool, planMap map[string]*happydns.CheckPlan) {
	disabledSet = make(map[string]bool)
	planMap = make(map[string]*happydns.CheckPlan)
	for _, p := range plans {
		key := p.CheckerID + "|" + p.Target.String()
		planMap[key] = p
		if p.IsFullyDisabled() {
			disabledSet[key] = true
		}
	}
	return
}

// enqueueJob creates and pushes a scheduler job if the key is not already
// present and not disabled. When lastActive is zero (e.g. NotifyDomainChange),
// the job is scheduled at now + jitter; otherwise offset-based grid scheduling
// is used. Must be called with s.mu held. Returns true if a job was added.
func (s *Scheduler) enqueueJob(checkerID string, def *happydns.CheckerDefinition, target happydns.CheckTarget, disabledSet map[string]bool, planMap map[string]*happydns.CheckPlan, lastActive time.Time) bool {
	targetStr := target.String()
	key := checkerID + "|" + targetStr
	if s.jobKeys[key] || disabledSet[key] {
		return false
	}

	plan := planMap[key]
	interval := s.effectiveInterval(def, plan)

	var nextRun time.Time
	if lastActive.IsZero() {
		now := time.Now()
		nextRun = now.Add(computeJitter(checkerID, targetStr, now, interval))
	} else {
		offset := computeOffset(checkerID, targetStr, interval)
		nextRun = computeNextRun(interval, offset, lastActive)
	}

	job := &SchedulerJob{
		CheckerID: checkerID,
		Target:    target,
		Interval:  interval,
		NextRun:   nextRun,
	}
	if plan != nil {
		job.PlanID = &plan.Id
	}
	heap.Push(&s.queue, job)
	s.jobKeys[key] = true
	return true
}

func (s *Scheduler) loadAllPlans() ([]*happydns.CheckPlan, error) {
	iter, err := s.planStore.ListAllCheckPlans()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var plans []*happydns.CheckPlan
	for iter.Next() {
		plans = append(plans, iter.Item())
	}
	return plans, nil
}

func (s *Scheduler) loadAllDomains() []*happydns.Domain {
	if s.domainStore == nil {
		return nil
	}
	iter, err := s.domainStore.ListAllDomains()
	if err != nil {
		log.Printf("Scheduler: failed to list domains for auto-discovery: %v", err)
		return nil
	}
	defer iter.Close()

	var domains []*happydns.Domain
	for iter.Next() {
		d := iter.Item()
		domains = append(domains, d)
	}
	return domains
}

func (s *Scheduler) loadDomainServices(domain *happydns.Domain) []*happydns.ServiceMessage {
	if s.zoneStore == nil || len(domain.ZoneHistory) == 0 {
		return nil
	}

	// Collect services from the WIP zone ([0]) and the latest published
	// zone ([1]).  This lets the scheduler pick up new services the user
	// is configuring while still covering what is live.
	seen := make(map[string]struct{})
	var services []*happydns.ServiceMessage
	for _, idx := range []int{0, 1} {
		if idx >= len(domain.ZoneHistory) {
			break
		}
		zone, err := s.zoneStore.GetZone(domain.ZoneHistory[idx])
		if err != nil {
			log.Printf("Scheduler: failed to load zone %s for domain %s: %v", domain.ZoneHistory[idx], domain.DomainName, err)
			continue
		}
		for _, svcs := range zone.Services {
			for _, svc := range svcs {
				key := svc.Id.String()
				if _, dup := seen[key]; !dup {
					seen[key] = struct{}{}
					services = append(services, svc)
				}
			}
		}
	}
	return services
}

func (s *Scheduler) effectiveInterval(def *happydns.CheckerDefinition, plan *happydns.CheckPlan) time.Duration {
	interval := defaultInterval
	if def.Interval != nil {
		interval = def.Interval.Default
	}

	if plan != nil && plan.Interval != nil {
		interval = *plan.Interval
	}

	// Clamp to bounds.
	if def.Interval != nil {
		if interval < def.Interval.Min {
			interval = def.Interval.Min
		}
		if interval > def.Interval.Max {
			interval = def.Interval.Max
		}
	}

	return interval
}

func (s *Scheduler) spreadOverdueJobs() {
	now := time.Now()
	var overdue []*SchedulerJob

	for s.queue.Len() > 0 && s.queue.Peek().NextRun.Before(now) {
		overdue = append(overdue, heap.Pop(&s.queue).(*SchedulerJob))
	}

	if len(overdue) == 0 {
		return
	}

	window := time.Duration(len(overdue)) * minSpacing
	window = min(window, maxCatchUpWindow)

	for i, job := range overdue {
		delay := window * time.Duration(i) / time.Duration(len(overdue))
		job.NextRun = now.Add(delay)
		heap.Push(&s.queue, job)
	}
}

// GetPlannedJobsForChecker returns a snapshot of scheduled jobs for the given checker and target.
func (s *Scheduler) GetPlannedJobsForChecker(checkerID string, target happydns.CheckTarget) []*SchedulerJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tStr := target.String()
	var result []*SchedulerJob
	for _, job := range s.queue {
		if job.CheckerID == checkerID && job.Target.String() == tStr {
			cp := *job
			result = append(result, &cp)
		}
	}
	return result
}

// computeOffset returns a deterministic offset within the interval.
func computeOffset(checkerID, targetStr string, interval time.Duration) time.Duration {
	h := fnv.New64a()
	h.Write([]byte(checkerID + targetStr))
	return time.Duration(h.Sum64()%uint64(interval.Nanoseconds())) * time.Nanosecond
}

// computeJitter returns a small deterministic jitter (~5% of interval).
func computeJitter(checkerID, targetStr string, cycleTime time.Time, interval time.Duration) time.Duration {
	h := fnv.New64a()
	h.Write([]byte(checkerID + targetStr + cycleTime.Format(time.RFC3339)))
	maxJitter := interval / 20 // 5%
	if maxJitter <= 0 {
		return 0
	}
	return time.Duration(h.Sum64()%uint64(maxJitter.Nanoseconds())) * time.Nanosecond
}

// computeNextRun calculates the next run time based on interval, offset, and
// the last time the scheduler was known to be active. When lastActive is zero
// (first execution), it behaves as before. Otherwise it detects jobs that were
// missed during downtime (slot in (lastActive, now]) and schedules them
// immediately so spreadOverdueJobs can stagger them, while skipping jobs that
// already ran (slot <= lastActive).
func computeNextRun(interval, offset time.Duration, lastActive time.Time) time.Time {
	now := time.Now()

	// Use Unix nanoseconds to avoid time.Duration overflow with ancient epochs.
	nowNano := now.UnixNano()
	intervalNano := int64(interval)
	offsetNano := int64(offset) % intervalNano

	// Find the most recent grid slot <= now.
	cycleN := (nowNano - offsetNano) / intervalNano
	slotNano := cycleN*intervalNano + offsetNano
	if slotNano > nowNano {
		slotNano -= intervalNano
	}
	slot := time.Unix(0, slotNano)

	if lastActive.IsZero() {
		// First execution: schedule at the next future slot.
		if !slot.After(now) {
			return slot.Add(interval)
		}
		return slot
	}

	// Slot was missed during downtime, schedule now for catch-up.
	if slot.After(lastActive) && !slot.After(now) {
		return now
	}

	// Slot already executed before shutdown; advance to next cycle.
	return slot.Add(interval)
}
