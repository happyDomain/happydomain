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

package app

import (
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// disabledScheduler is a no-op implementation of SchedulerUsecase used when the
// scheduler is disabled in configuration.
type disabledScheduler struct{}

func (d *disabledScheduler) Run()   {}
func (d *disabledScheduler) Close() {}

func (d *disabledScheduler) TriggerOnDemandCheck(checkName string, targetType happydns.CheckScopeType, targetId happydns.Identifier, userId happydns.Identifier, options happydns.CheckerOptions) (happydns.Identifier, error) {
	return happydns.Identifier{}, fmt.Errorf("test scheduler is disabled in configuration")
}
