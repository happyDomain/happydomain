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

package usecase

import (
	"errors"
	"fmt"
	"strings"

	"git.happydns.org/happyDomain/model"
)

type domainInfoUsecase struct {
	getters []happydns.DomainInfoGetter
}

func NewDomainInfoUsecase(getters ...happydns.DomainInfoGetter) happydns.DomainInfoUsecase {
	return &domainInfoUsecase{getters: getters}
}

func (diu *domainInfoUsecase) GetDomainInfo(fqdn happydns.Origin) (*happydns.DomainInfo, error) {
	domain := happydns.Origin(strings.TrimSuffix(string(fqdn), "."))

	var lastErr error
	for _, getter := range diu.getters {
		infos, err := getter(domain)
		if err != nil {
			if errors.Is(err, happydns.DomainDoesNotExist) {
				return nil, err
			}
			lastErr = err
			continue
		}
		if infos == nil {
			lastErr = fmt.Errorf("no information found")
			continue
		}
		return infos, nil
	}

	return nil, fmt.Errorf("unable to retrieve RDAP/WHOIS info about the domain name: %w", lastErr)
}
