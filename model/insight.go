// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

type Insights struct {
	InsightsID string          `json:"id"`
	Version    VersionResponse `json:"version"`
	Build      struct {
		// build settings used by the Go compiler
		Settings  map[string]string `json:"settings"`
		GoVersion string            `json:"goVersion"`
	} `json:"build"`
	OS struct {
		Type   string `json:"type"`
		Arch   string `json:"arch"`
		NumCPU int    `json:"numCPU"`
	} `json:"os"`
	Config struct {
		DisableEmbeddedLogin bool   `json:"disableEmbeddedLogin,omitempty"`
		DisableProviders     bool   `json:"disableProviders,omitempty"`
		DisableRegistration  bool   `json:"disableRegistration,omitempty"`
		HasBaseURL           bool   `json:"hasBaseURL,omitempty"`
		HasDevProxy          bool   `json:"hasDevProxy,omitempty"`
		HasExternalAuth      bool   `json:"hasExternalAuth,omitempty"`
		HasListmonkURL       bool   `json:"hasListmonkURL,omitempty"`
		LocalBind            bool   `json:"localBind,omitempty"`
		NbOidcProviders      int    `json:"nbOidcProviders,omitempty"`
		NoAuthActive         bool   `json:"noAuthActive,omitempty"`
		NoMail               bool   `json:"noMail,omitempty"`
		NonUnixAdminBind     bool   `json:"nonUnixAdminBind,omitempty"`
		StorageEngine        string `json:"storageEngine,omitempty"`
	} `json:"config"`
	Database struct {
		Version     int            `json:"schemaVersion"`
		NbAuthUsers int            `json:"nbAuthUsers"`
		NbDomains   int            `json:"nbDomains"`
		Providers   map[string]int `json:"providers"`
		NbUsers     int            `json:"nbUsers"`
		NbZones     int            `json:"nbZones"`
	} `json:"db"`
}
