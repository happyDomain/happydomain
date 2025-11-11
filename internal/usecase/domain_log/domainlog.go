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

// DomainLogAppender is a minimal interface for appending domain logs.
// Used by orchestrator to decouple from the full Service.
type DomainLogAppender interface {
	AppendDomainLog(domain *happydns.Domain, entry *happydns.DomainLog) error
}

type Service struct {
	store DomainLogStorage
}

func NewService(store DomainLogStorage) *Service {
	return &Service{
		store: store,
	}
}

// AppendDomainLog creates a new domain log entry.
func (s *Service) AppendDomainLog(domain *happydns.Domain, entry *happydns.DomainLog) error {
	return s.store.CreateDomainLog(domain, entry)
}

// ListDomainLogs retrieves all logs for a domain, sorted by date (newest first).
func (s *Service) ListDomainLogs(domain *happydns.Domain) ([]*happydns.DomainLog, error) {
	logs, err := s.store.ListDomainLogs(domain)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to retrieve logs for domain %q (did=%s): %w", domain.Domain, domain.Id.String(), err),
			UserMessage: "Unable to access the domain logs. Please try again later.",
		}
	}

	// Sort by date
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Date.After(logs[j].Date)
	})

	return logs, nil
}

// UpdateDomainLog updates an existing domain log entry.
func (s *Service) UpdateDomainLog(domain *happydns.Domain, log *happydns.DomainLog) error {
	return s.store.UpdateDomainLog(domain, log)
}

// DeleteDomainLog removes a domain log entry.
func (s *Service) DeleteDomainLog(domain *happydns.Domain, log *happydns.DomainLog) error {
	return s.store.DeleteDomainLog(domain, log)
}
