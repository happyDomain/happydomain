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
	"errors"
	"log"
	"time"

	notifPkg "git.happydns.org/happyDomain/internal/notification"
	"git.happydns.org/happyDomain/model"
)

// Glue between checker execution and notification system; owns no I/O — all of it lives in collaborators.
type Dispatcher struct {
	stateStore  NotificationStateStorage
	userStore   UserGetter
	domainStore DomainGetter

	resolver *Resolver
	pool     *Pool
	tester   *Tester
	ack      *AckService
	locker   *StateLocker

	// Overridable for tests.
	nowFn func() time.Time
}

// Caller owns Pool lifecycle.
func NewDispatcher(
	stateStore NotificationStateStorage,
	userStore UserGetter,
	domainStore DomainGetter,
	resolver *Resolver,
	pool *Pool,
	tester *Tester,
	ack *AckService,
	locker *StateLocker,
) *Dispatcher {
	return &Dispatcher{
		stateStore:  stateStore,
		userStore:   userStore,
		domainStore: domainStore,
		resolver:    resolver,
		pool:        pool,
		tester:      tester,
		ack:         ack,
		locker:      locker,
		nowFn:       time.Now,
	}
}

func (d *Dispatcher) Start() { d.pool.Start() }
func (d *Dispatcher) Stop()  { d.pool.Stop() }

func (d *Dispatcher) OnExecutionComplete(exec *happydns.Execution, eval *happydns.CheckEvaluation) {
	if exec == nil || exec.Status != happydns.ExecutionDone {
		return
	}

	userId := happydns.TargetIdentifier(exec.Target.UserId)
	if userId == nil {
		return
	}

	user, err := d.userStore.GetUser(*userId)
	if err != nil {
		log.Printf("notification: failed to load user %q: %v", userId, err)
		return
	}

	newStatus := exec.Result.Status

	// Serialize with AckService so concurrent updates can't wipe an ack or fire duplicates.
	unlock := d.locker.Lock(exec.CheckerID, exec.Target, *userId)
	defer unlock()

	state, err := d.loadOrInitState(exec, *userId)
	if err != nil {
		return
	}

	oldStatus := state.LastStatus
	pref := d.resolver.ResolvePreference(user, exec.Target)

	dec := decide(state, pref, newStatus, d.nowFn())

	// Recovery/escalation invalidates ack: incident is over or has worsened.
	if dec.ClearAck {
		state.ClearAcknowledgement()
	}

	switch dec.Action {
	case actionSkip:
		return
	case actionAdvance:
		d.advanceState(state, newStatus)
		return
	}

	payload := d.buildPayload(user, exec, eval, oldStatus, newStatus)

	// Mark before enqueue so a rapid re-run sees oldStatus == newStatus and skips.
	d.markNotified(state, newStatus)

	for _, ch := range d.resolver.ResolveChannels(user, pref) {
		d.pool.Enqueue(ch, payload, user)
	}
}

func (d *Dispatcher) loadOrInitState(exec *happydns.Execution, userId happydns.Identifier) (*happydns.NotificationState, error) {
	state, err := d.stateStore.GetState(exec.CheckerID, exec.Target, userId)
	if errors.Is(err, happydns.ErrNotificationStateNotFound) {
		return &happydns.NotificationState{
			CheckerID:  exec.CheckerID,
			Target:     exec.Target,
			UserId:     userId,
			LastStatus: happydns.StatusUnknown,
		}, nil
	}
	if err != nil {
		log.Printf("notification: failed to load state for %q/%q: %v", exec.CheckerID, exec.Target.String(), err)
		return nil, err
	}
	return state, nil
}

func (d *Dispatcher) buildPayload(user *happydns.User, exec *happydns.Execution, eval *happydns.CheckEvaluation, oldStatus, newStatus happydns.Status) *notifPkg.NotificationPayload {
	var domainName string
	if did := happydns.TargetIdentifier(exec.Target.DomainId); did != nil {
		if domain, err := d.domainStore.GetDomain(*did); err == nil {
			domainName = domain.DomainName
		}
	}
	if domainName == "" {
		domainName = "(unknown domain)"
	}

	var states []happydns.CheckState
	if eval != nil {
		states = eval.States
	}

	return &notifPkg.NotificationPayload{
		Recipient:  notifPkg.Recipient{Email: user.Email},
		CheckerID:  exec.CheckerID,
		Target:     exec.Target,
		DomainName: domainName,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		States:     states,
	}
}

// Persist the observed status without claiming a notification was sent (policy suppressed it).
func (d *Dispatcher) advanceState(state *happydns.NotificationState, newStatus happydns.Status) {
	state.LastStatus = newStatus
	if err := d.stateStore.PutState(state); err != nil {
		log.Printf("notification: failed to update state: %v", err)
	}
}

func (d *Dispatcher) markNotified(state *happydns.NotificationState, newStatus happydns.Status) {
	state.LastStatus = newStatus
	state.LastNotifiedAt = d.nowFn()
	if err := d.stateStore.PutState(state); err != nil {
		log.Printf("notification: failed to update state: %v", err)
	}
}

func (d *Dispatcher) SendTestNotification(ch *happydns.NotificationChannel, user *happydns.User) error {
	return d.tester.Send(ch, user)
}

func (d *Dispatcher) AcknowledgeIssue(userId happydns.Identifier, checkerID string, target happydns.CheckTarget, acknowledgedBy string, annotation string) error {
	return d.ack.AcknowledgeIssue(userId, checkerID, target, acknowledgedBy, annotation)
}

func (d *Dispatcher) ClearAcknowledgement(userId happydns.Identifier, checkerID string, target happydns.CheckTarget) error {
	return d.ack.ClearAcknowledgement(userId, checkerID, target)
}

func (d *Dispatcher) GetState(userId happydns.Identifier, checkerID string, target happydns.CheckTarget) (*happydns.NotificationState, error) {
	return d.ack.GetState(userId, checkerID, target)
}

func (d *Dispatcher) ListStatesByUser(userId happydns.Identifier) ([]*happydns.NotificationState, error) {
	return d.ack.ListStatesByUser(userId)
}
