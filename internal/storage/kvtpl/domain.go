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

package database

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// Secondary indexes for the domain entity.
//
//	domain.owner|{ownerId}|{domainId}     -> ""   reverse lookup by owner
//	domain.fqdn|{hash(fqdn)}|{domainId}   -> ""   reverse lookup by FQDN
//
// The FQDN is hashed (truncated SHA-256, 21 bytes) so the key length is
// bounded regardless of how long the domain name is.
const (
	domainPrimaryPrefix    = "domain-"
	domainOwnerIndexPrefix = "domain.owner|"
	domainFQDNIndexPrefix  = "domain.fqdn|"
)

// normalizeDomainName lowercases and trailing-dot-normalizes a domain name so
// the FQDN index is case-insensitive (RFC 4343) and stable regardless of how
// callers spelled the name.
func normalizeDomainName(name string) string {
	return strings.ToLower(dns.Fqdn(name))
}

// hashFQDN returns a fixed-length, URL-safe digest of the normalized FQDN.
// SHA-256 truncated to 21 bytes encodes to 28 base64url chars, keeping the
// full index key well under 64 chars (prefix 12 + 28 + 1 + 22 = 63).
func hashFQDN(fqdn string) string {
	sum := sha256.Sum256([]byte(normalizeDomainName(fqdn)))
	return base64.RawURLEncoding.EncodeToString(sum[:21])
}

func domainOwnerIndexKey(ownerId, domainId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", domainOwnerIndexPrefix, ownerId.String(), domainId.String())
}

func domainFQDNIndexKey(fqdn string, domainId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", domainFQDNIndexPrefix, hashFQDN(fqdn), domainId.String())
}

// putDomainIndexes writes the owner and FQDN secondary indexes for d.
func (s *KVStorage) putDomainIndexes(d *happydns.Domain) error {
	if err := s.db.Put(domainOwnerIndexKey(d.Owner, d.Id), ""); err != nil {
		return err
	}
	return s.db.Put(domainFQDNIndexKey(d.DomainName, d.Id), "")
}

func (s *KVStorage) ListAllDomains() (happydns.Iterator[happydns.Domain], error) {
	iter := s.db.Search(domainPrimaryPrefix)
	return NewKVIterator[happydns.Domain](s.db, iter), nil
}

func (s *KVStorage) CountDomains() (int, error) {
	return s.countByPrefix(domainPrimaryPrefix)
}

func (s *KVStorage) ListDomains(u *happydns.User) (domains []*happydns.Domain, err error) {
	prefix := fmt.Sprintf("%s%s|", domainOwnerIndexPrefix, u.Id.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	for iter.Next() {
		id, kerr := lastKeySegment(iter.Key())
		if kerr != nil {
			continue
		}
		d, gerr := s.GetDomain(id)
		if gerr != nil {
			// Stale index entry, e.g. after a crashed delete. Skip and let
			// TidyDomainIndexes (or the next UpdateDomain) clean it up.
			log.Printf("ListDomains: stale owner index %q -> missing domain: %v", iter.Key(), gerr)
			continue
		}
		domains = append(domains, d)
	}

	err = iter.Err()
	return
}

func (s *KVStorage) getDomain(id string) (*happydns.Domain, error) {
	domain := &happydns.Domain{}
	err := s.db.Get(id, domain)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrDomainNotFound
	}
	return domain, err
}

func (s *KVStorage) GetDomain(id happydns.Identifier) (*happydns.Domain, error) {
	return s.getDomain(fmt.Sprintf("%s%s", domainPrimaryPrefix, id.String()))
}

func (s *KVStorage) GetDomainByDN(u *happydns.User, dn string) ([]*happydns.Domain, error) {
	domains, err := s.ListDomains(u)
	if err != nil {
		return nil, err
	}

	target := normalizeDomainName(dn)
	var ret []*happydns.Domain
	for _, domain := range domains {
		if normalizeDomainName(domain.DomainName) == target {
			ret = append(ret, domain)
		}
	}

	if len(ret) == 0 {
		return nil, happydns.ErrNotFound
	}

	return ret, nil
}

func (s *KVStorage) FindDomainsByName(fqdn string) ([]*happydns.Domain, error) {
	prefix := fmt.Sprintf("%s%s|", domainFQDNIndexPrefix, hashFQDN(fqdn))
	iter := s.db.Search(prefix)
	defer iter.Release()

	var ret []*happydns.Domain
	for iter.Next() {
		id, kerr := lastKeySegment(iter.Key())
		if kerr != nil {
			continue
		}
		d, gerr := s.GetDomain(id)
		if gerr != nil {
			log.Printf("FindDomainsByName: stale fqdn index %q -> missing domain: %v", iter.Key(), gerr)
			continue
		}
		ret = append(ret, d)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, happydns.ErrNotFound
	}

	return ret, nil
}

func (s *KVStorage) CreateDomain(z *happydns.Domain) error {
	key, id, err := s.db.FindIdentifierKey(domainPrimaryPrefix)
	if err != nil {
		return err
	}

	z.Id = id
	if err := s.db.Put(key, z); err != nil {
		return err
	}
	return s.putDomainIndexes(z)
}

func (s *KVStorage) UpdateDomain(z *happydns.Domain) error {
	primaryKey := fmt.Sprintf("%s%s", domainPrimaryPrefix, z.Id.String())

	// Load the previous record to detect index-affecting changes. UpdateDomain
	// is also used by the backup restore path where the primary may not exist
	// yet, so a missing old record is not an error.
	old, err := s.GetDomain(z.Id)
	if err != nil && !errors.Is(err, happydns.ErrDomainNotFound) {
		return err
	}

	if err := s.db.Put(primaryKey, z); err != nil {
		return err
	}

	if old != nil {
		if !old.Owner.Equals(z.Owner) {
			if delErr := s.db.Delete(domainOwnerIndexKey(old.Owner, old.Id)); delErr != nil {
				log.Printf("UpdateDomain: failed to delete stale owner index for owner %s: %v", old.Owner.String(), delErr)
			}
		}
		if normalizeDomainName(old.DomainName) != normalizeDomainName(z.DomainName) {
			if delErr := s.db.Delete(domainFQDNIndexKey(old.DomainName, old.Id)); delErr != nil {
				log.Printf("UpdateDomain: failed to delete stale fqdn index for %s: %v", old.DomainName, delErr)
			}
		}
	}

	return s.putDomainIndexes(z)
}

func (s *KVStorage) DeleteDomain(zId happydns.Identifier) error {
	// Best-effort index cleanup: if the primary is already gone we still want
	// the caller's Delete to succeed, and any orphan index entry will be
	// skipped harmlessly by readers and reaped by tidy.
	if d, err := s.GetDomain(zId); err == nil {
		if delErr := s.db.Delete(domainOwnerIndexKey(d.Owner, d.Id)); delErr != nil {
			log.Printf("DeleteDomain: failed to delete owner index for owner %s: %v", d.Owner.String(), delErr)
		}
		if delErr := s.db.Delete(domainFQDNIndexKey(d.DomainName, d.Id)); delErr != nil {
			log.Printf("DeleteDomain: failed to delete fqdn index for %s: %v", d.DomainName, delErr)
		}
	}

	return s.db.Delete(fmt.Sprintf("%s%s", domainPrimaryPrefix, zId.String()))
}

func (s *KVStorage) ClearDomains() error {
	if err := s.ClearZones(); err != nil {
		return err
	}

	if err := s.clearByPrefix(domainOwnerIndexPrefix); err != nil {
		return err
	}
	if err := s.clearByPrefix(domainFQDNIndexPrefix); err != nil {
		return err
	}

	iter, err := s.ListAllDomains()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		err = s.db.Delete(iter.Key())
		if err != nil {
			return err
		}
	}

	return iter.Err()
}
