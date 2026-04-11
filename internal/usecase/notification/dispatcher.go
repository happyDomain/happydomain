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

// Dispatcher evaluates notification state transitions after checker executions
// and dispatches notifications through configured channels.
type Dispatcher struct {
	channelStore NotificationChannelStorage
	prefStore    NotificationPreferenceStorage
	stateStore   NotificationStateStorage
	recordStore  NotificationRecordStorage
	userStore    UserGetter
	domainStore  DomainGetter
	senders      map[happydns.NotificationChannelType]notifPkg.ChannelSender
	baseURL      string
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(
	channelStore NotificationChannelStorage,
	prefStore NotificationPreferenceStorage,
	stateStore NotificationStateStorage,
	recordStore NotificationRecordStorage,
	userStore UserGetter,
	domainStore DomainGetter,
	senders map[happydns.NotificationChannelType]notifPkg.ChannelSender,
	baseURL string,
) *Dispatcher {
	return &Dispatcher{
		channelStore: channelStore,
		prefStore:    prefStore,
		stateStore:   stateStore,
		recordStore:  recordStore,
		userStore:    userStore,
		domainStore:  domainStore,
		senders:      senders,
		baseURL:      baseURL,
	}
}

// OnExecutionComplete is the callback invoked after a checker execution finishes.
// It determines whether a notification should be sent based on state transitions,
// user preferences, acknowledgements, and quiet hours.
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
		log.Printf("notification: failed to load user %s: %v", exec.Target.UserId, err)
		return
	}

	newStatus := exec.Result.Status

	// Load or create notification state.
	state, err := d.stateStore.GetState(exec.CheckerID, exec.Target, *userId)
	if errors.Is(err, happydns.ErrNotificationStateNotFound) {
		state = &happydns.NotificationState{
			CheckerID:  exec.CheckerID,
			Target:     exec.Target,
			UserId:     *userId,
			LastStatus: happydns.StatusUnknown,
		}
	} else if err != nil {
		log.Printf("notification: failed to load state for %s/%s: %v", exec.CheckerID, exec.Target.String(), err)
		return
	}

	oldStatus := state.LastStatus

	// No state transition: skip notification.
	if oldStatus == newStatus {
		return
	}

	// Clear acknowledgement on any state change.
	state.Acknowledged = false
	state.AcknowledgedAt = nil
	state.AcknowledgedBy = ""
	state.Annotation = ""

	isRecovery := newStatus < happydns.StatusWarn && oldStatus >= happydns.StatusWarn

	// Resolve the effective preference for this target.
	pref := d.resolvePreference(user, exec.Target)
	if pref == nil || !pref.Enabled {
		// No preference or disabled: still update state, but don't notify.
		d.updateState(state, newStatus)
		return
	}

	// Check minimum severity threshold.
	if !isRecovery && newStatus < pref.MinStatus {
		d.updateState(state, newStatus)
		return
	}

	// Check recovery notification preference.
	if isRecovery && !pref.NotifyRecovery {
		d.updateState(state, newStatus)
		return
	}

	// Check quiet hours.
	if d.isQuietHour(pref) {
		d.updateState(state, newStatus)
		return
	}

	// Resolve domain name for the notification payload.
	domainName := exec.Target.DomainId
	if did := happydns.TargetIdentifier(exec.Target.DomainId); did != nil {
		if domain, err := d.domainStore.GetDomain(*did); err == nil {
			domainName = domain.DomainName
		}
	}

	var states []happydns.CheckState
	if eval != nil {
		states = eval.States
	}

	payload := &notifPkg.NotificationPayload{
		User:       user,
		CheckerID:  exec.CheckerID,
		Target:     exec.Target,
		DomainName: domainName,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		States:     states,
		BaseURL:    d.baseURL,
	}

	// Resolve channels and send.
	channels := d.resolveChannels(user, pref)
	for _, ch := range channels {
		d.sendAndRecord(ch, payload, user)
	}

	d.updateState(state, newStatus)
}

// resolvePreference finds the most specific enabled preference for the given target.
// Specificity: service > domain > global.
func (d *Dispatcher) resolvePreference(user *happydns.User, target happydns.CheckTarget) *happydns.NotificationPreference {
	prefs, err := d.prefStore.ListPreferencesByUser(user.Id)
	if err != nil {
		log.Printf("notification: failed to load preferences for user %s: %v", user.Id, err)
		return nil
	}

	var global, domainMatch, serviceMatch *happydns.NotificationPreference
	for _, p := range prefs {
		if p.ServiceId != nil && p.ServiceId.String() == target.ServiceId {
			serviceMatch = p
		} else if p.DomainId != nil && p.DomainId.String() == target.DomainId && p.ServiceId == nil {
			domainMatch = p
		} else if p.DomainId == nil && p.ServiceId == nil {
			global = p
		}
	}

	if serviceMatch != nil {
		return serviceMatch
	}
	if domainMatch != nil {
		return domainMatch
	}
	return global
}

// resolveChannels returns the channels to use for a notification.
func (d *Dispatcher) resolveChannels(user *happydns.User, pref *happydns.NotificationPreference) []*happydns.NotificationChannel {
	allChannels, err := d.channelStore.ListChannelsByUser(user.Id)
	if err != nil {
		log.Printf("notification: failed to load channels for user %s: %v", user.Id, err)
		return nil
	}

	// Build a set of allowed channel IDs from the preference.
	var allowed map[string]bool
	if len(pref.ChannelIds) > 0 {
		allowed = make(map[string]bool, len(pref.ChannelIds))
		for _, id := range pref.ChannelIds {
			allowed[id.String()] = true
		}
	}

	var result []*happydns.NotificationChannel
	for _, ch := range allChannels {
		if !ch.Enabled {
			continue
		}
		if allowed != nil && !allowed[ch.Id.String()] {
			continue
		}
		result = append(result, ch)
	}
	return result
}

// isQuietHour returns true if the current UTC hour falls within the quiet window.
func (d *Dispatcher) isQuietHour(pref *happydns.NotificationPreference) bool {
	if pref.QuietStart == nil || pref.QuietEnd == nil {
		return false
	}

	hour := time.Now().UTC().Hour()
	start := *pref.QuietStart
	end := *pref.QuietEnd

	if start <= end {
		return hour >= start && hour < end
	}
	// Wraps midnight, e.g. 22:00 - 06:00.
	return hour >= start || hour < end
}

// sendAndRecord dispatches through the appropriate sender and logs the result.
func (d *Dispatcher) sendAndRecord(ch *happydns.NotificationChannel, payload *notifPkg.NotificationPayload, user *happydns.User) {
	sender, ok := d.senders[ch.Type]
	if !ok {
		log.Printf("notification: no sender for channel type %q", ch.Type)
		return
	}

	rec := &happydns.NotificationRecord{
		UserId:      user.Id,
		ChannelType: ch.Type,
		ChannelId:   ch.Id,
		CheckerID:   payload.CheckerID,
		Target:      payload.Target,
		OldStatus:   payload.OldStatus,
		NewStatus:   payload.NewStatus,
		SentAt:      time.Now(),
	}

	if err := sender.Send(ch, payload); err != nil {
		log.Printf("notification: failed to send via %s channel %s: %v", ch.Type, ch.Id, err)
		rec.Success = false
		rec.Error = err.Error()
	} else {
		rec.Success = true
	}

	if err := d.recordStore.CreateRecord(rec); err != nil {
		log.Printf("notification: failed to log record: %v", err)
	}
}

func (d *Dispatcher) updateState(state *happydns.NotificationState, newStatus happydns.Status) {
	state.LastStatus = newStatus
	state.LastNotifiedAt = time.Now()
	if err := d.stateStore.PutState(state); err != nil {
		log.Printf("notification: failed to update state: %v", err)
	}
}
