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

import "time"

// NotificationChannelType identifies the transport used to deliver a notification.
type NotificationChannelType string

const (
	NotificationChannelEmail       NotificationChannelType = "email"
	NotificationChannelWebhook     NotificationChannelType = "webhook"
	NotificationChannelUnifiedPush NotificationChannelType = "unifiedpush"
)

// NotificationChannelConfig holds channel-specific configuration.
type NotificationChannelConfig struct {
	// EmailAddress overrides the user's account email. Empty means use account email.
	EmailAddress string `json:"emailAddress,omitempty"`

	// WebhookURL is the HTTP endpoint to POST to.
	WebhookURL string `json:"webhookUrl,omitempty"`

	// WebhookHeaders are extra headers sent with webhook requests.
	WebhookHeaders map[string]string `json:"webhookHeaders,omitempty"`

	// WebhookSecret is used to compute an HMAC-SHA256 signature header.
	WebhookSecret string `json:"webhookSecret,omitempty"`

	// UnifiedPushEndpoint is the push server endpoint URL.
	UnifiedPushEndpoint string `json:"unifiedPushEndpoint,omitempty"`
}

// NotificationChannel represents a single configured notification destination.
type NotificationChannel struct {
	// Id is the channel's unique identifier.
	Id Identifier `json:"id" swaggertype:"string" readonly:"true"`

	// UserId is the owner of the channel.
	UserId Identifier `json:"userId" swaggertype:"string" readonly:"true"`

	// Type is the transport type (email, webhook, unifiedpush).
	Type NotificationChannelType `json:"type" binding:"required"`

	// Name is a human-readable label for the channel.
	Name string `json:"name"`

	// Enabled controls whether notifications are sent through this channel.
	Enabled bool `json:"enabled"`

	// Config holds channel-specific settings.
	Config NotificationChannelConfig `json:"config"`
}

// NotificationPreference controls what notifications a user receives for a given scope.
// Scope resolution: ServiceId set > DomainId set > global (both nil).
type NotificationPreference struct {
	// Id is the preference's unique identifier.
	Id Identifier `json:"id" swaggertype:"string" readonly:"true"`

	// UserId is the owner of the preference.
	UserId Identifier `json:"userId" swaggertype:"string" readonly:"true"`

	// DomainId, if set, scopes this preference to a specific domain.
	DomainId *Identifier `json:"domainId,omitempty" swaggertype:"string"`

	// ServiceId, if set, scopes this preference to a specific service.
	ServiceId *Identifier `json:"serviceId,omitempty" swaggertype:"string"`

	// ChannelIds restricts which channels to use. Empty means all enabled channels.
	ChannelIds []Identifier `json:"channelIds,omitempty" swaggertype:"array,string"`

	// MinStatus is the minimum severity that triggers a notification.
	MinStatus Status `json:"minStatus"`

	// NotifyRecovery controls whether recovery (back to OK) notifications are sent.
	NotifyRecovery bool `json:"notifyRecovery"`

	// QuietStart is the start hour (0-23, UTC) of a quiet window.
	QuietStart *int `json:"quietStart,omitempty"`

	// QuietEnd is the end hour (0-23, UTC) of a quiet window.
	QuietEnd *int `json:"quietEnd,omitempty"`

	// Enabled is the master switch for this preference scope.
	Enabled bool `json:"enabled"`
}

// NotificationState tracks the last notified status for a (checker, target, user) tuple.
// Used for deduplication: only state transitions trigger notifications.
type NotificationState struct {
	// CheckerID identifies the checker.
	CheckerID string `json:"checkerId"`

	// Target is the checked scope.
	Target CheckTarget `json:"target"`

	// UserId is the user who owns the target.
	UserId Identifier `json:"userId" swaggertype:"string"`

	// LastStatus is the status from the last notification.
	LastStatus Status `json:"lastStatus"`

	// LastNotifiedAt is when the last notification was sent.
	LastNotifiedAt time.Time `json:"lastNotifiedAt" format:"date-time"`

	// Acknowledged indicates the user has acknowledged the current issue.
	Acknowledged bool `json:"acknowledged"`

	// AcknowledgedAt is when the issue was acknowledged.
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty" format:"date-time"`

	// AcknowledgedBy describes who acknowledged (user email or "api").
	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`

	// Annotation is a user-provided note on the acknowledgement.
	Annotation string `json:"annotation,omitempty"`
}

// NotificationRecord logs a sent notification for audit purposes.
type NotificationRecord struct {
	// Id is the record's unique identifier.
	Id Identifier `json:"id" swaggertype:"string" readonly:"true"`

	// UserId is the recipient user.
	UserId Identifier `json:"userId" swaggertype:"string"`

	// ChannelType is the transport used.
	ChannelType NotificationChannelType `json:"channelType"`

	// ChannelId is the channel through which the notification was sent.
	ChannelId Identifier `json:"channelId" swaggertype:"string"`

	// CheckerID is the checker that triggered the notification.
	CheckerID string `json:"checkerId"`

	// Target is the checked scope.
	Target CheckTarget `json:"target"`

	// OldStatus is the previous status before the transition.
	OldStatus Status `json:"oldStatus"`

	// NewStatus is the new status that triggered the notification.
	NewStatus Status `json:"newStatus"`

	// SentAt is when the notification was dispatched.
	SentAt time.Time `json:"sentAt" format:"date-time"`

	// Success indicates whether the send succeeded.
	Success bool `json:"success"`

	// Error holds the error message if the send failed.
	Error string `json:"error,omitempty"`
}

// AcknowledgeRequest is the JSON body for acknowledging a checker issue.
type AcknowledgeRequest struct {
	Annotation string `json:"annotation,omitempty"`
}
