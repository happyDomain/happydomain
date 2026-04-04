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

package checker

import (
	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/model"
)

// The checker definition registry lives in the Apache-2.0 licensed
// checker-sdk-go module, so external plugins can register themselves
// without depending on AGPL code. These wrappers preserve the existing
// happyDomain call sites.

// RegisterChecker registers a checker definition globally.
func RegisterChecker(c *happydns.CheckerDefinition) {
	sdk.RegisterChecker(c)
}

// RegisterExternalizableChecker registers a checker that supports being
// delegated to a remote HTTP endpoint. It appends an "endpoint" AdminOpt
// so the administrator can optionally configure a remote URL.
// When the endpoint is left empty, the checker runs locally as usual.
func RegisterExternalizableChecker(c *happydns.CheckerDefinition) {
	sdk.RegisterExternalizableChecker(c)
}

// RegisterObservationProvider registers an observation provider globally.
func RegisterObservationProvider(p happydns.ObservationProvider) {
	sdk.RegisterObservationProvider(p)
}

// GetCheckers returns all registered checker definitions.
func GetCheckers() map[string]*happydns.CheckerDefinition {
	return sdk.GetCheckers()
}

// FindChecker returns the checker definition with the given ID, or nil.
func FindChecker(id string) *happydns.CheckerDefinition {
	return sdk.FindChecker(id)
}
