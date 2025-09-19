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

package check

import (
	"fmt"

	"git.happydns.org/happyDomain/checks"
	"git.happydns.org/happyDomain/model"
)

type checkerUsecase struct {
	config *happydns.Options
}

func NewCheckerUsecase(cfg *happydns.Options) happydns.CheckerUsecase {
	return &checkerUsecase{
		config: cfg,
	}
}

func (tu *checkerUsecase) GetChecker(cname string) (happydns.Checker, error) {
	checker, err := checks.FindChecker(cname)
	if err != nil {
		return nil, fmt.Errorf("unable to find check named %q: %w", cname, err)
	}

	return checker, nil
}

func (tu *checkerUsecase) ListCheckers() (*map[string]happydns.Checker, error) {
	return checks.GetCheckers(), nil
}
