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

package domainlog

import (
	"fmt"
	"sort"

	"git.happydns.org/happyDomain/model"
)

type ListDomainLogsUsecase struct {
	store DomainLogStorage
}

func NewListDomainLogsUsecase(store DomainLogStorage) *ListDomainLogsUsecase {
	return &ListDomainLogsUsecase{
		store: store,
	}
}

func (uc *ListDomainLogsUsecase) List(domain *happydns.Domain) ([]*happydns.DomainLog, error) {
	logs, err := uc.store.ListDomainLogs(domain)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to retrieve logs for domain %q (did=%s): %w", domain.DomainName, domain.Id.String(), err),
			UserMessage: "Unable to access the domain logs. Please try again later.",
		}
	}

	// Sort by date
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Date.After(logs[j].Date)
	})

	return logs, nil
}
