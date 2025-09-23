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
	"cmp"
	"fmt"
	"maps"
	"slices"

	"git.happydns.org/happyDomain/checks"
	"git.happydns.org/happyDomain/model"
)

type checkerUsecase struct {
	config *happydns.Options
	store  CheckerStorage
}

func NewCheckerUsecase(cfg *happydns.Options, store CheckerStorage) happydns.CheckerUsecase {
	return &checkerUsecase{
		config: cfg,
		store:  store,
	}
}

func (tu *checkerUsecase) GetChecker(cname string) (happydns.Checker, error) {
	checker, err := checks.FindChecker(cname)
	if err != nil {
		return nil, fmt.Errorf("unable to find check named %q: %w", cname, err)
	}

	return checker, nil
}

// copyNonEmpty copies key/value pairs from src into dst, skipping nil or empty-string values.
func copyNonEmpty(dst, src happydns.CheckerOptions) {
	for k, v := range src {
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		dst[k] = v
	}
}

func compareIdentifiers(a, b *happydns.Identifier) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	if a.Equals(*b) {
		return 0
	}

	return a.Compare(*b)
}

// CompareCheckerOptionsPositional defines the merge precedence ordering for
// checker option configs: admin < user < domain < service.
func CompareCheckerOptionsPositional(a, b *happydns.CheckerOptionsPositional) int {
	if a.CheckName != b.CheckName {
		return cmp.Compare(a.CheckName, b.CheckName)
	}
	if res := compareIdentifiers(a.UserId, b.UserId); res != 0 {
		return res
	}
	if res := compareIdentifiers(a.DomainId, b.DomainId); res != 0 {
		return res
	}
	return compareIdentifiers(a.ServiceId, b.ServiceId)
}

func (tu *checkerUsecase) GetCheckerOptions(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (*happydns.CheckerOptions, error) {
	configs, err := tu.store.GetCheckerConfiguration(cname, userid, domainid, serviceid)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(configs, CompareCheckerOptionsPositional)

	opts := make(happydns.CheckerOptions)

	for _, c := range configs {
		maps.Copy(opts, c.Options)
	}

	return &opts, nil
}

func (tu *checkerUsecase) ListCheckers() (*map[string]happydns.Checker, error) {
	return checks.GetCheckers(), nil
}

func (tu *checkerUsecase) SetCheckerOptions(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.CheckerOptions) error {
	return tu.store.UpdateCheckerConfiguration(cname, userid, domainid, serviceid, opts)
}

func (tu *checkerUsecase) OverwriteSomeCheckerOptions(cname string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.CheckerOptions) error {
	current, err := tu.GetCheckerOptions(cname, userid, domainid, serviceid)
	if err != nil {
		return err
	}

	maps.Copy(*current, opts)

	return tu.store.UpdateCheckerConfiguration(cname, userid, domainid, serviceid, *current)
}
