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

package abstract

import (
	"strings"
	"sync"
)

// Some services do more than publish records: happyDomain also serves the
// content those records point at (mail-client auto-configuration XML, the
// MTA-STS policy file, …). Every such service emits a CNAME toward the same
// public hostname, which is a property of the deployment rather than of the
// zone — hence a single package-global, set once at startup.
var (
	serviceHostingHostMu sync.RWMutex
	serviceHostingHost   string
)

// SetServiceHostingHost configures the FQDN that the hosting CNAMEs
// (autoconfig., autodiscover., mta-sts., …) should point to. Called by the app
// at startup from the resolved Options.ServiceHostingHost.
func SetServiceHostingHost(host string) {
	serviceHostingHostMu.Lock()
	defer serviceHostingHostMu.Unlock()
	serviceHostingHost = strings.TrimSuffix(host, ".")
}

// GetServiceHostingHost returns the configured hosting FQDN, with a trailing
// dot. Empty string when no host is configured.
func GetServiceHostingHost() string {
	serviceHostingHostMu.RLock()
	defer serviceHostingHostMu.RUnlock()
	if serviceHostingHost == "" {
		return ""
	}
	return serviceHostingHost + "."
}
