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

package happydns

const (
	PluginResultStatusKO PluginResultStatus = iota
	PluginResultStatusWarn
	PluginResultStatusInfo
	PluginResultStatusOK
)

type PluginResultStatus int

type PluginOptions map[string]any

type SetPluginOptionsRequest struct {
	Options PluginOptions `json:"options"`
}

type PluginOptionsPositional struct {
	PluginName string
	UserId     *Identifier
	DomainId   *Identifier
	ServiceId  *Identifier

	Options PluginOptions
}

type TestPlugin interface {
	PluginEnvName() []string
	Version() PluginVersionInfo
	AvailableOptions() PluginOptionsDocumentation

	RunTest(options PluginOptions, meta map[string]string) (*PluginResult, error)
}

type PluginVersionInfo struct {
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	AvailableOn PluginAvailability `json:"availableOn"`
}

type PluginAvailability struct {
	ApplyToDomain    bool     `json:"applyToDomain,omitempty"`
	ApplyToService   bool     `json:"applyToService,omitempty"`
	LimitToProviders []string `json:"limitToProviders,omitempty"`
	LimitToServices  []string `json:"limitToServices,omitempty"`
}

type PluginOptionsDocumentation struct {
	RunOpts     []PluginOptionDocumentation `json:"runOpts,omitempty"`
	ServiceOpts []PluginOptionDocumentation `json:"serviceOpts,omitempty"`
	DomainOpts  []PluginOptionDocumentation `json:"domainOpts,omitempty"`
	UserOpts    []PluginOptionDocumentation `json:"userOpts,omitempty"`
	AdminOpts   []PluginOptionDocumentation `json:"adminOpts,omitempty"`
}

type PluginOptionDocumentation Field

type PluginStatus struct {
	PluginVersionInfo
	Opts PluginOptionsDocumentation `json:"options"`
}

type PluginResult struct {
	Status     PluginResultStatus `json:"status"`
	StatusLine string             `json:"statusLine,omitempty"`
	Report     any                `json:"report"`
}

type PluginManager interface {
	GetTestPlugins() []TestPlugin
	GetTestPlugin(string) (TestPlugin, bool)
}

type TestPluginUsecase interface {
	GetTestPlugin(string) (TestPlugin, error)
	GetTestPluginOptions(string, *Identifier, *Identifier, *Identifier) (*PluginOptions, error)
	ListTestPlugins() ([]TestPlugin, error)
	OverwriteSomeTestPluginOptions(string, *Identifier, *Identifier, *Identifier, PluginOptions) error
	SetTestPluginOptions(string, *Identifier, *Identifier, *Identifier, PluginOptions) error
}
