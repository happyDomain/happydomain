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

package happydns

import (
	"encoding/json"
	"time"
)

// Values are owned by sender implementations; the model does not enumerate them.
type NotificationChannelType string

// Config is opaque to the model: decoded by the sender registered for Type.
type NotificationChannel struct {
	Id      Identifier              `json:"id" swaggertype:"string" readonly:"true"`
	UserId  Identifier              `json:"userId" swaggertype:"string" readonly:"true"`
	Type    NotificationChannelType `json:"type" binding:"required"`
	Name    string                  `json:"name"`
	Enabled bool                    `json:"enabled"`
	Config  json.RawMessage         `json:"config" swaggertype:"object"`
}

// Scope resolution: ServiceId set > DomainId set > global (both nil).
type NotificationPreference struct {
	Id        Identifier   `json:"id" swaggertype:"string" readonly:"true"`
	UserId    Identifier   `json:"userId" swaggertype:"string" readonly:"true"`
	DomainId  *Identifier  `json:"domainId,omitempty" swaggertype:"string"`
	ServiceId *Identifier  `json:"serviceId,omitempty" swaggertype:"string"`
	// Empty means all enabled channels.
	ChannelIds     []Identifier `json:"channelIds,omitempty" swaggertype:"array,string"`
	MinStatus      Status       `json:"minStatus"`
	NotifyRecovery bool         `json:"notifyRecovery"`
	// Hours 0-23, interpreted in Timezone (IANA name; empty means UTC).
	QuietStart *int   `json:"quietStart,omitempty"`
	QuietEnd   *int   `json:"quietEnd,omitempty"`
	Timezone   string `json:"timezone,omitempty"`
	Enabled    bool   `json:"enabled"`
}

// Used for deduplication: only state transitions trigger notifications.
type NotificationState struct {
	CheckerID      string      `json:"checkerId"`
	Target         CheckTarget `json:"target"`
	UserId         Identifier  `json:"userId" swaggertype:"string"`
	LastStatus     Status      `json:"lastStatus"`
	LastNotifiedAt time.Time   `json:"lastNotifiedAt" format:"date-time"`
	Acknowledged   bool        `json:"acknowledged"`
	AcknowledgedAt *time.Time  `json:"acknowledgedAt,omitempty" format:"date-time"`
	// User email or "api".
	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`
	Annotation     string `json:"annotation,omitempty"`
}

type NotificationRecord struct {
	Id          Identifier              `json:"id" swaggertype:"string" readonly:"true"`
	UserId      Identifier              `json:"userId" swaggertype:"string"`
	ChannelType NotificationChannelType `json:"channelType"`
	ChannelId   Identifier              `json:"channelId" swaggertype:"string"`
	CheckerID   string                  `json:"checkerId"`
	Target      CheckTarget             `json:"target"`
	OldStatus   Status                  `json:"oldStatus"`
	NewStatus   Status                  `json:"newStatus"`
	SentAt      time.Time               `json:"sentAt" format:"date-time"`
	Success     bool                    `json:"success"`
	Error       string                  `json:"error,omitempty"`
}

type AcknowledgeRequest struct {
	Annotation string `json:"annotation,omitempty"`
}

// Called both on explicit user clear and when a transition invalidates the ack.
func (s *NotificationState) ClearAcknowledgement() {
	s.Acknowledged = false
	s.AcknowledgedAt = nil
	s.AcknowledgedBy = ""
	s.Annotation = ""
}

// Implicit fallback when no preference is configured: opt-in at Warn+ on all enabled channels. Returned with zero Id/UserId; not persisted.
func DefaultNotificationPreference() *NotificationPreference {
	return &NotificationPreference{
		MinStatus:      StatusWarn,
		NotifyRecovery: false,
		Enabled:        true,
	}
}

// Returns 2 service / 1 domain / 0 global / -1 no-match.
func (p *NotificationPreference) MatchesTarget(target CheckTarget) int {
	if p.ServiceId != nil {
		if p.ServiceId.String() == target.ServiceId {
			return 2
		}
		return -1
	}
	if p.DomainId != nil {
		if p.DomainId.String() == target.DomainId {
			return 1
		}
		return -1
	}
	return 0
}
