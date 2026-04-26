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

package checkers

import (
	matrix "git.happydns.org/checker-matrix/checker"
	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/internal/checker"
)

func init() {
	prvd := matrix.Provider()
	checker.RegisterObservationProvider(prvd)
	// Not Externalizable checker as it already calls a HTTP API
	checker.RegisterChecker(prvd.(sdk.CheckerDefinitionProvider).Definition())
}
