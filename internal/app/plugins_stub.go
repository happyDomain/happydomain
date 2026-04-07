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

//go:build !linux && !darwin && !freebsd

package app

import "log"

// initPlugins is a no-op on platforms where Go's plugin package is not
// supported (Windows, plan9, …). If the operator configured plugin
// directories anyway we log a clear warning rather than silently ignoring
// them, so the misconfiguration is visible at startup.
func (a *App) initPlugins() error {
	if len(a.cfg.PluginsDirectories) > 0 {
		log.Printf("Warning: plugin loading is not supported on this platform; ignoring %d configured plugin directories", len(a.cfg.PluginsDirectories))
	}
	return nil
}
