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

package checker

import (
	"strings"

	"git.happydns.org/happyDomain/model"
)

// WorstStatusAggregator aggregates check states by taking the worst status.
type WorstStatusAggregator struct{}

func (a WorstStatusAggregator) Aggregate(states []happydns.CheckState) happydns.CheckState {
	if len(states) == 0 {
		return happydns.CheckState{Status: happydns.StatusUnknown}
	}
	worst := states[0].Status
	var messages []string
	for _, s := range states {
		if s.Status > worst {
			worst = s.Status
		}
		if s.Message != "" {
			messages = append(messages, s.Message)
		}
	}
	return happydns.CheckState{
		Status:  worst,
		Message: strings.Join(messages, "; "),
	}
}
