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
	"context"
	"encoding/json"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// The types and helpers needed by external checker plugins live in the
// Apache-2.0 licensed checker-sdk-go module. They are re-exported here as
// aliases so the rest of the happyDomain codebase keeps relying on this model.
//
// Host-only types (Execution, CheckPlan, CheckEvaluation, …) remain
// defined in this file because they describe orchestration state that is
// internal to the happyDomain server and never crosses the plugin boundary.

// --- Re-exports from checker-sdk-go ---

type CheckScopeType = sdk.CheckScopeType

const (
	CheckScopeAdmin   = sdk.CheckScopeAdmin
	CheckScopeUser    = sdk.CheckScopeUser
	CheckScopeDomain  = sdk.CheckScopeDomain
	CheckScopeZone    = sdk.CheckScopeZone
	CheckScopeService = sdk.CheckScopeService
)

const (
	AutoFillDomainName       = sdk.AutoFillDomainName
	AutoFillSubdomain        = sdk.AutoFillSubdomain
	AutoFillZone             = sdk.AutoFillZone
	AutoFillServiceType      = sdk.AutoFillServiceType
	AutoFillService          = sdk.AutoFillService
	AutoFillDiscoveryEntries = sdk.AutoFillDiscoveryEntries
)

type (
	CheckTarget                 = sdk.CheckTarget
	CheckerAvailability         = sdk.CheckerAvailability
	CheckerOptions              = sdk.CheckerOptions
	CheckerOptionDocumentation  = sdk.CheckerOptionDocumentation
	CheckerOptionsDocumentation = sdk.CheckerOptionsDocumentation
	Status                      = sdk.Status
	CheckState                  = sdk.CheckState
	CheckMetric                 = sdk.CheckMetric
	ObservationKey              = sdk.ObservationKey
	CheckIntervalSpec           = sdk.CheckIntervalSpec
	ObservationProvider         = sdk.ObservationProvider
	CheckRuleInfo               = sdk.CheckRuleInfo
	CheckRule                   = sdk.CheckRule
	CheckRuleWithOptions        = sdk.CheckRuleWithOptions
	ObservationGetter           = sdk.ObservationGetter
	CheckAggregator             = sdk.CheckAggregator
	CheckerHTMLReporter         = sdk.CheckerHTMLReporter
	CheckerMetricsReporter      = sdk.CheckerMetricsReporter
	CheckerDefinitionProvider   = sdk.CheckerDefinitionProvider
	CheckerDefinition           = sdk.CheckerDefinition
	OptionsValidator            = sdk.OptionsValidator
	ExternalCollectRequest      = sdk.ExternalCollectRequest
	ExternalCollectResponse     = sdk.ExternalCollectResponse
	ExternalEvaluateRequest     = sdk.ExternalEvaluateRequest
	ExternalEvaluateResponse    = sdk.ExternalEvaluateResponse
	ExternalReportRequest       = sdk.ExternalReportRequest
	DiscoveryEntry              = sdk.DiscoveryEntry
	DiscoveryPublisher          = sdk.DiscoveryPublisher
	RelatedObservation          = sdk.RelatedObservation
	ReportContext               = sdk.ReportContext
)

const (
	StatusOK      = sdk.StatusOK
	StatusInfo    = sdk.StatusInfo
	StatusUnknown = sdk.StatusUnknown
	StatusWarn    = sdk.StatusWarn
	StatusCrit    = sdk.StatusCrit
	StatusError   = sdk.StatusError
)

// --- Helpers for converting between target identifier strings and *Identifier ---

// TargetIdentifier parses a target identifier string into an *Identifier.
// Returns nil if the string is empty or cannot be parsed.
func TargetIdentifier(s string) *Identifier {
	if s == "" {
		return nil
	}
	id, err := NewIdentifierFromString(s)
	if err != nil {
		return nil
	}
	return &id
}

// FormatIdentifier returns the string representation of id, or "" if nil.
func FormatIdentifier(id *Identifier) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// --- Host-only types (orchestration state) ---

// CheckerRunRequest is the JSON body for manually triggering a checker.
type CheckerRunRequest struct {
	Options      CheckerOptions  `json:"options,omitempty"`
	EnabledRules map[string]bool `json:"enabledRules,omitempty"`
}

// CheckerOptionsPositional stores options with their positional key components.
type CheckerOptionsPositional struct {
	CheckName string      `json:"checkName"`
	UserId    *Identifier `json:"userId,omitempty"`
	DomainId  *Identifier `json:"domainId,omitempty"`
	ServiceId *Identifier `json:"serviceId,omitempty"`

	Options CheckerOptions `json:"options"`
}

// CheckPlan is an optional user override for a checker on a specific target.
type CheckPlan struct {
	Id        Identifier      `json:"id" swaggertype:"string" binding:"required" readonly:"true"`
	CheckerID string          `json:"checkerId" binding:"required" readonly:"true"`
	Target    CheckTarget     `json:"target" binding:"required" readonly:"true"`
	Interval  *time.Duration  `json:"interval,omitempty" swaggertype:"integer"`
	Enabled   map[string]bool `json:"enabled,omitempty"`
}

// IsFullyDisabled returns true if the enabled map is non-empty and every entry is false.
func (p *CheckPlan) IsFullyDisabled() bool {
	if len(p.Enabled) == 0 {
		return false
	}
	for _, v := range p.Enabled {
		if v {
			return false
		}
	}
	return true
}

// IsRuleEnabled returns whether a specific rule is enabled.
// A nil or empty map means all rules are enabled. A missing key means enabled.
func (p *CheckPlan) IsRuleEnabled(ruleName string) bool {
	if len(p.Enabled) == 0 {
		return true
	}
	v, ok := p.Enabled[ruleName]
	if !ok {
		return true
	}
	return v
}

// CheckerStatus combines a checker definition with its latest execution and plan for a target.
type CheckerStatus struct {
	*CheckerDefinition
	LatestExecution *Execution      `json:"latestExecution,omitempty"`
	Plan            *CheckPlan      `json:"plan,omitempty"`
	Enabled         bool            `json:"enabled"`
	EnabledRules    map[string]bool `json:"enabledRules"`
}

// CheckEvaluation is the result of running a checker on observed data.
type CheckEvaluation struct {
	Id          Identifier   `json:"id" swaggertype:"string" binding:"required" readonly:"true"`
	PlanID      *Identifier  `json:"planId,omitempty" swaggertype:"string"`
	CheckerID   string       `json:"checkerId" binding:"required"`
	Target      CheckTarget  `json:"target" binding:"required"`
	SnapshotID  Identifier   `json:"snapshotId" swaggertype:"string" binding:"required" readonly:"true"`
	EvaluatedAt time.Time    `json:"evaluatedAt" binding:"required" readonly:"true" format:"date-time"`
	States      []CheckState `json:"states" binding:"required" readonly:"true"`
}

// ObservationSnapshot holds data collected during an execution.
type ObservationSnapshot struct {
	Id          Identifier                         `json:"id" swaggertype:"string" binding:"required" readonly:"true"`
	Target      CheckTarget                        `json:"target" binding:"required" readonly:"true"`
	CollectedAt time.Time                          `json:"collectedAt" binding:"required" readonly:"true" format:"date-time"`
	Data        map[ObservationKey]json.RawMessage `json:"data" binding:"required" readonly:"true" swaggertype:"object,object"`
}

// ObservationCacheEntry is a lightweight pointer to cached observation data in a snapshot.
type ObservationCacheEntry struct {
	SnapshotID  Identifier `json:"snapshotId"`
	CollectedAt time.Time  `json:"collectedAt"`
}

// ExecutionStatus represents the lifecycle state of an execution.
type ExecutionStatus int

const (
	ExecutionPending ExecutionStatus = iota
	ExecutionRunning
	ExecutionDone
	ExecutionFailed
	// ExecutionRateLimited indicates a planned execution that will be
	// skipped because the user's MaxChecksPerDay quota is exhausted.
	// Only used for synthetic planned executions returned by
	// ListPlannedExecutions; never persisted.
	ExecutionRateLimited
)

// TriggerType represents what initiated an execution.
type TriggerType int

const (
	TriggerManual TriggerType = iota
	TriggerSchedule
)

// TriggerInfo describes the trigger for an execution.
type TriggerInfo struct {
	Type   TriggerType `json:"type"`
	PlanID *Identifier `json:"planId,omitempty" swaggertype:"string"`
}

// Execution represents a single run of a checker pipeline.
type Execution struct {
	Id           Identifier      `json:"id" swaggertype:"string" binding:"required" readonly:"true"`
	CheckerID    string          `json:"checkerId" binding:"required" readonly:"true"`
	PlanID       *Identifier     `json:"planId,omitempty" swaggertype:"string" readonly:"true"`
	Target       CheckTarget     `json:"target" binding:"required" readonly:"true"`
	Trigger      TriggerInfo     `json:"trigger" binding:"required" readonly:"true"`
	StartedAt    time.Time       `json:"startedAt" binding:"required" readonly:"true" format:"date-time"`
	EndedAt      *time.Time      `json:"endedAt,omitempty" readonly:"true" format:"date-time"`
	Status       ExecutionStatus `json:"status" binding:"required" readonly:"true"`
	Error        string          `json:"error,omitempty" readonly:"true"`
	Result       CheckState      `json:"result" readonly:"true"`
	EvaluationID *Identifier     `json:"evaluationId,omitempty" swaggertype:"string" readonly:"true"`
}

// CheckerEngine orchestrates the full checker pipeline.
type CheckerEngine interface {
	CreateExecution(checkerID string, target CheckTarget, plan *CheckPlan) (*Execution, error)
	RunExecution(ctx context.Context, exec *Execution, plan *CheckPlan, runOpts CheckerOptions) (*CheckEvaluation, error)
}
