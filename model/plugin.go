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

import (
	"time"
)

const (
	PluginResultStatusKO PluginResultStatus = iota
	PluginResultStatusWarn
	PluginResultStatusInfo
	PluginResultStatusOK
)

const (
	PluginStateStopped PluginStateEnum = iota
	PluginStateStarting
	PluginStateReady
	PluginStateError
)

type PluginResultStatus int
type PluginStateEnum int

type TestPlugin interface {
	PluginEnvName() []string
	Version() PluginVersionInfo

	RunTest(options map[string]interface{}, meta map[string]string) (*PluginResult, error)
}

type LaunchableTestPlugin interface {
	StartPlugin(options map[string]interface{}) error
	StopPlugin() error
	PluginStatus() PluginState
}

type PluginVersionInfo struct {
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	AvailableOn PluginAvailability `json:"availableOn"`
}

type PluginAvailability struct {
	ApplyToDomain    bool     `json:"applyToDomain,omitempty"`
	ApplyToService   bool     `json:"applyToDomain,omitempty"`
	LimitToProviders []string `json:"limitToProviders,omitempty"`
	LimitToServices  []string `json:"limitToServices,omitempty"`
}

type PluginResult struct {
	Status     PluginResultStatus `json:"status"`
	StatusLine string             `json:"statusLine,omitempty"`
	Report     interface{}        `json:"report"`
}

type PluginState struct {
	State  PluginStateEnum `json:"state"`
	Uptime time.Time       `json:"uptime,omitempty"`
}
