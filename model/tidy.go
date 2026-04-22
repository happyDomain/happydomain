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

import ()

type TidyUpUseCase interface {
	// TidyAll runs every tidy pass. When dropInvalid is true, iterators
	// that encounter undecodable records (e.g. legacy schema drift) will
	// delete those records; otherwise they are only logged.
	TidyAll(dropInvalid bool) error
	TidyAuthUsers(dropInvalid bool) error
	TidyCheckEvaluations(dropInvalid bool) error
	TidyCheckPlans(dropInvalid bool) error
	TidyCheckerConfigurations(dropInvalid bool) error
	TidyExecutions(dropInvalid bool) error
	TidyObservationCache(dropInvalid bool) error
	TidySnapshots(dropInvalid bool) error
	TidyDomains(dropInvalid bool) error
	TidyDomainLogs(dropInvalid bool) error
	TidyProviders(dropInvalid bool) error
	TidySessions(dropInvalid bool) error
	TidyUsers(dropInvalid bool) error
	TidyZones(dropInvalid bool) error
}
