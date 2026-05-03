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

package notification

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/model"
)

const (
	dispatchWorkers = 4
	// On overflow, an audit record is written so back-pressure losses are visible in history.
	dispatchQueueSize = 256
	// Caps a single Send so a wedged endpoint cannot starve workers.
	sendTimeout = 15 * time.Second
	// Bounds the persisted error to keep the audit log small.
	maxRecordErrorLen = 512
)

func truncateError(s string) string {
	const marker = "…[truncated]"
	if len(s) <= maxRecordErrorLen {
		return s
	}
	return s[:maxRecordErrorLen-len(marker)] + marker
}

type dispatchJob struct {
	channel *happydns.NotificationChannel
	payload *notifPkg.NotificationPayload
	user    *happydns.User
}

// Async send fan-out; no policy — caller decides whether to enqueue.
type Pool struct {
	registry    *notifPkg.Registry
	recordStore NotificationRecordStorage

	jobs     chan dispatchJob
	wg       sync.WaitGroup
	stopped  atomic.Bool
	stopOnce sync.Once

	// Overridable for tests.
	nowFn func() time.Time
}

func NewPool(registry *notifPkg.Registry, recordStore NotificationRecordStorage) *Pool {
	return &Pool{
		registry:    registry,
		recordStore: recordStore,
		jobs:        make(chan dispatchJob, dispatchQueueSize),
		nowFn:       time.Now,
	}
}

func (p *Pool) Start() {
	for range dispatchWorkers {
		p.wg.Add(1)
		go p.worker()
	}
}

// Idempotent. Post-Stop, Enqueue is a no-op so a racing caller doesn't panic on send-to-closed-channel.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		p.stopped.Store(true)
		close(p.jobs)
	})
	p.wg.Wait()
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for job := range p.jobs {
		p.sendAndRecord(job.channel, job.payload, job.user)
	}
}

// On saturation, persists an audit record so the missed alert surfaces in history.
func (p *Pool) Enqueue(ch *happydns.NotificationChannel, payload *notifPkg.NotificationPayload, user *happydns.User) bool {
	if p.stopped.Load() {
		return false
	}
	job := dispatchJob{channel: ch, payload: payload, user: user}
	select {
	case p.jobs <- job:
		return true
	default:
		// Saturated: record the miss rather than silently drop.
		log.Printf("notification: dispatch queue full, recording back-pressure failure for channel %q (%q)", ch.Id, ch.Type)
		p.recordSaturation(ch, payload, user)
		return false
	}
}

func (p *Pool) recordSaturation(ch *happydns.NotificationChannel, payload *notifPkg.NotificationPayload, user *happydns.User) {
	rec := newRecord(ch, payload, user, p.nowFn())
	rec.Success = false
	rec.Error = "dispatch queue saturated"
	if err := p.recordStore.CreateRecord(rec); err != nil {
		log.Printf("notification: failed to log saturation record: %v", err)
	}
}

func (p *Pool) sendAndRecord(ch *happydns.NotificationChannel, payload *notifPkg.NotificationPayload, user *happydns.User) {
	rec := newRecord(ch, payload, user, p.nowFn())

	if err := p.runSend(ch, payload); err != nil {
		log.Printf("notification: failed to send via %q channel %q: %v", ch.Type, ch.Id, err)
		rec.Success = false
		rec.Error = truncateError(err.Error())
	} else {
		rec.Success = true
	}

	if err := p.recordStore.CreateRecord(rec); err != nil {
		log.Printf("notification: failed to log record: %v", err)
	}
}

func (p *Pool) runSend(ch *happydns.NotificationChannel, payload *notifPkg.NotificationPayload) error {
	sender, ok := p.registry.Get(ch.Type)
	if !ok {
		return fmt.Errorf("no sender for channel type %q", ch.Type)
	}
	cfg, err := sender.DecodeConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("invalid config for channel %s: %w", ch.Id, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	return sender.Send(ctx, cfg, payload)
}

// Caller fills Success/Error after the send attempt.
func newRecord(ch *happydns.NotificationChannel, payload *notifPkg.NotificationPayload, user *happydns.User, sentAt time.Time) *happydns.NotificationRecord {
	return &happydns.NotificationRecord{
		UserId:      user.Id,
		ChannelType: ch.Type,
		ChannelId:   ch.Id,
		CheckerID:   payload.CheckerID,
		Target:      payload.Target,
		OldStatus:   payload.OldStatus,
		NewStatus:   payload.NewStatus,
		SentAt:      sentAt,
	}
}
