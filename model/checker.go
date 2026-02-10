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

type CheckerResponse struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	Availability CheckerAvailability         `json:"availability"`
	Options      CheckerOptionsDocumentation `json:"options"`
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
	GetChecker(string) (Checker, error)
	GetCheckerOptions(string, *Identifier, *Identifier, *Identifier) (*CheckerOptions, error)
	ListCheckers() (*map[string]Checker, error)
	OverwriteSomeCheckerOptions(string, *Identifier, *Identifier, *Identifier, CheckerOptions) error
	SetCheckerOptions(string, *Identifier, *Identifier, *Identifier, CheckerOptions) error
}
