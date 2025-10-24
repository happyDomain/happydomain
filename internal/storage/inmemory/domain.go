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

package inmemory

import (
	"slices"

	"git.happydns.org/happyDomain/model"
)

func (s *InMemoryStorage) ListAllDomains() (happydns.Iterator[happydns.Domain], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.Domain](&s.domains), nil
}

// ListDomains retrieves all Domains associated to the given User.
func (s *InMemoryStorage) ListDomains(u *happydns.User) ([]*happydns.Domain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var domains []*happydns.Domain
	for _, domain := range s.domains {
		if domain.Owner.Equals(u.Id) {
			domains = append(domains, domain)
		}
	}

	return domains, nil
}

// GetDomain retrieves the Domain with the given id and owned by the given User.
func (s *InMemoryStorage) GetDomain(id happydns.Identifier) (*happydns.Domain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	domain, exists := s.domains[id.String()]
	if !exists {
		return nil, happydns.ErrDomainNotFound
	}

	return domain, nil
}

// GetDomainByDN is like GetDomain but look for the domain name instead of identifier.
func (s *InMemoryStorage) GetDomainByDN(u *happydns.User, dn string) (ret []*happydns.Domain, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, domain := range s.domains {
		if domain.DomainName == dn {
			ret = append(ret, domain)
		}
	}

	if len(ret) == 0 {
		return nil, happydns.ErrDomainNotFound
	}

	return
}

// CreateDomain creates a record in the database for the given Domain.
func (s *InMemoryStorage) CreateDomain(domain *happydns.Domain) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	domain.Id, err = happydns.NewRandomIdentifier()
	s.domains[domain.Id.String()] = domain

	return
}

// UpdateDomain updates the fields of the given Domain.
func (s *InMemoryStorage) UpdateDomain(domain *happydns.Domain) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.domains[domain.Id.String()] = domain

	return nil
}

// DeleteDomain removes the given Domain from the database.
func (s *InMemoryStorage) DeleteDomain(domainid happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.domains, domainid.String())
	return nil
}

// ClearDomains deletes all Domains present in the database.
func (s *InMemoryStorage) ClearDomains() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.domains = make(map[string]*happydns.Domain)
	return nil
}

// DOMAIN LOGS --------------------------------------------------

func (s *InMemoryStorage) ListAllDomainLogs() (happydns.Iterator[happydns.DomainLogWithDomainId], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return NewInMemoryIterator[happydns.DomainLogWithDomainId](&s.domainLogs), nil
}

// ListDomainLogs retrieves the logs for the given Domain.
func (s *InMemoryStorage) ListDomainLogs(domain *happydns.Domain) (dlogs []*happydns.DomainLog, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, logid := range s.domainLogsByDomains[domain.Id.String()] {
		dlogs = append(dlogs, &s.domainLogs[logid.String()].DomainLog)
	}

	return
}

// CreateDomainLog creates a log entry for the given Domain.
func (s *InMemoryStorage) CreateDomainLog(domain *happydns.Domain, log *happydns.DomainLog) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Id, err = happydns.NewRandomIdentifier()
	if err != nil {
		return
	}

	s.domainLogs[log.Id.String()] = &happydns.DomainLogWithDomainId{DomainLog: *log, DomainId: domain.Id}
	s.domainLogsByDomains[domain.Id.String()] = append(s.domainLogsByDomains[domain.Id.String()], &log.Id)
	return
}

// UpdateDomainLog updates a log entry for the given Domain.
func (s *InMemoryStorage) UpdateDomainLog(domain *happydns.Domain, log *happydns.DomainLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.domainLogs[log.Id.String()] = &happydns.DomainLogWithDomainId{DomainLog: *log, DomainId: domain.Id}
	return nil
}

// DeleteDomainLog deletes a log entry for the given Domain.
func (s *InMemoryStorage) DeleteDomainLog(domain *happydns.Domain, log *happydns.DomainLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.domainLogs, log.Id.String())

	i := slices.IndexFunc(s.domainLogsByDomains[domain.Id.String()], func(e *happydns.Identifier) bool {
		return e.Equals(log.Id)
	})
	if i == -1 {
		return happydns.ErrDomainLogNotFound
	}
	s.domainLogsByDomains[domain.Id.String()] = append(s.domainLogsByDomains[domain.Id.String()][:i], s.domainLogsByDomains[domain.Id.String()][i+1:]...)

	return nil
}
