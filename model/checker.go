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

import "encoding/json"

// Auto-fill variable identifiers for checker option fields.
const (
	// AutoFillDomainName fills the option with the fully qualified domain name
	// of the domain being tested (e.g. "example.com.").
	AutoFillDomainName = "domain_name"

	// AutoFillSubdomain fills the option with the subdomain relative to the zone
	// (e.g. "www" for "www.example.com." in zone "example.com."). Only
	// applicable for service-scoped tests.
	AutoFillSubdomain = "subdomain"

	// AutoFillServiceType fills the option with the service type identifier
	// (e.g. "abstract.MatrixIM"). Only applicable for service-scoped tests.
	AutoFillServiceType = "service_type"
)

const (
	CheckResultStatusUnknown CheckResultStatus = iota
	CheckResultStatusCritical
	CheckResultStatusWarn
	CheckResultStatusInfo
	CheckResultStatusOK
)

type CheckResultStatus int

type CheckerOptions map[string]any

type Checker interface {
	ID() string
	Name() string
	Availability() CheckerAvailability
	Options() CheckerOptionsDocumentation
	RunCheck(options CheckerOptions, meta map[string]string) (*CheckResult, error)
}

// CheckerHTMLReporter is an optional interface checkers can implement
// to render their stored report as a full HTML document (for iframe embedding).
// Detect support with a type assertion: _, ok := checker.(CheckerHTMLReporter)
type CheckerHTMLReporter interface {
	// GetHTMLReport generates an HTML document from the JSON-encoded report data
	// stored in CheckResult.Report.
	// The raw parameter contains the JSON bytes of the Report field as stored.
	GetHTMLReport(raw json.RawMessage) (string, error)
}

type CheckerResponse struct {
	ID            string                      `json:"id"`
	Name          string                      `json:"name"`
	Availability  CheckerAvailability         `json:"availability"`
	Options       CheckerOptionsDocumentation `json:"options"`
	HasHTMLReport bool                        `json:"has_html_report,omitempty"`
}

type SetCheckerOptionsRequest struct {
	Options CheckerOptions `json:"options"`
}

type CheckerOptionsPositional struct {
	CheckName string
	UserId    *Identifier
	DomainId  *Identifier
	ServiceId *Identifier

	Options CheckerOptions
}

type CheckerAvailability struct {
	ApplyToDomain    bool     `json:"applyToDomain,omitempty"`
	ApplyToService   bool     `json:"applyToService,omitempty"`
	LimitToProviders []string `json:"limitToProviders,omitempty"`
	LimitToServices  []string `json:"limitToServices,omitempty"`
}

type CheckerOptionsDocumentation struct {
	RunOpts     []CheckerOptionDocumentation `json:"runOpts,omitempty"`
	ServiceOpts []CheckerOptionDocumentation `json:"serviceOpts,omitempty"`
	DomainOpts  []CheckerOptionDocumentation `json:"domainOpts,omitempty"`
	UserOpts    []CheckerOptionDocumentation `json:"userOpts,omitempty"`
	AdminOpts   []CheckerOptionDocumentation `json:"adminOpts,omitempty"`
}

type CheckerOptionDocumentation Field

// CheckerStatus represents the current status of a checker for a specific target,
// including whether it is enabled, its schedule, and the most recent result.
type CheckerStatus struct {
	CheckerName string           `json:"checker_name"`
	Enabled     bool             `json:"enabled"`
	Schedule    *CheckerSchedule `json:"schedule,omitempty"`
	LastResult  *CheckResult     `json:"last_result,omitempty"`
}

type CheckerUsecase interface {
	BuildMergedCheckerOptions(string, *Identifier, *Identifier, *Identifier, CheckerOptions) (CheckerOptions, error)
	GetStoredCheckerOptionsNoDefault(string, *Identifier, *Identifier, *Identifier) (CheckerOptions, error)
	GetChecker(string) (Checker, error)
	GetCheckerOptions(string, *Identifier, *Identifier, *Identifier) (*CheckerOptions, error)
	ListCheckers() (*map[string]Checker, error)
	OverwriteSomeCheckerOptions(string, *Identifier, *Identifier, *Identifier, CheckerOptions) error
	SetCheckerOptions(string, *Identifier, *Identifier, *Identifier, CheckerOptions) error
}
