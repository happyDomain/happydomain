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

package database

// PurgeCheckerHistory deletes everything the checkers recreate on their next
// run. Observation cache entries and discovery observation refs point at
// snapshots, so they go with them. Check plans and checker options are kept.
func (s *KVStorage) PurgeCheckerHistory() (int, error) {
	// Indexes go before the primaries they point at, so that a purge that
	// fails, or a write landing while it runs, leaves no index pointing at
	// a missing key.
	return s.deleteByPrefixes(
		ExecutionByCheckerIndexPrefix,
		ExecutionByDomainIndexPrefix,
		ExecutionByPlanIndexPrefix,
		ExecutionByUserIndexPrefix,
		ExecutionPrimaryPrefix,
		evaluationByPlanIndexPrefix,
		evaluationByCheckerIndexPrefix,
		evaluationPrimaryPrefix,
		discoveryTargetIndex,
		discoveryPrimaryPrefix,
		"dscobs-snap|",
		"dscobs|",
		observationSnapshotPrefix,
		observationCachePrefix,
	)
}
