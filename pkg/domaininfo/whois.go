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

package domaininfo

import (
	"context"
	"errors"
	"time"

	"github.com/likexian/whois"
	"github.com/likexian/whois-parser"

	"git.happydns.org/happyDomain/pkg/domaininfo/types"
)

func GetDomainWhoisInfo(ctx context.Context, domain string) (*DomainInfo, error) {
	client := whois.NewClient()

	// The whois library has no context support; derive a timeout from the
	// context deadline so we at least honour it approximately.
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			client.SetTimeout(remaining)
		}
	}

	raw, err := client.Whois(domain)
	if err != nil {
		return nil, err
	}

	result, err := whoisparser.Parse(raw)
	if err != nil {
		if errors.Is(err, whoisparser.ErrNotFoundDomain) {
			return nil, types.ErrDomainDoesNotExist
		}
		return nil, err
	}

	// Some registries (e.g. Verisign for .com) return a "No match" response
	// that the parser accepts without error but produces an empty Domain
	// field. Treat this as a non-existent domain.
	if result.Domain == nil || result.Domain.Domain == "" {
		return nil, types.ErrDomainDoesNotExist
	}

	return mapWhoisResult(&result), nil
}

// mapWhoisResult converts a parsed whois response into a DomainInfo. Kept
// separate from the network call so the mapping can be unit-tested without
// touching the registry.
func mapWhoisResult(result *whoisparser.WhoisInfo) *DomainInfo {
	registrar := "Unknown"
	var registrarURL *string
	if result.Registrar != nil {
		registrar = result.Registrar.Name
		registrarURL = sanitizeURL(result.Registrar.ReferralURL)
	}

	var (
		name        string
		nameservers []string
		created     *time.Time
		expires     *time.Time
		status      []string
	)
	if result.Domain != nil {
		name = result.Domain.Domain
		nameservers = result.Domain.NameServers
		created = result.Domain.CreatedDateInTime
		expires = result.Domain.ExpirationDateInTime
		status = result.Domain.Status
	}

	// Contacts
	contacts := make(map[string]*ContactInfo)
	whoisContacts := map[string]*whoisparser.Contact{
		"registrant": result.Registrant,
		"admin":      result.Administrative,
		"tech":       result.Technical,
	}
	for key, wc := range whoisContacts {
		if wc == nil {
			continue
		}
		contacts[key] = &ContactInfo{
			Name:         wc.Name,
			Organization: wc.Organization,
			Email:        wc.Email,
			Street:       wc.Street,
			City:         wc.City,
			Province:     wc.Province,
			PostalCode:   wc.PostalCode,
			Country:      wc.Country,
			Phone:        wc.Phone,
		}
	}

	var contactsPtr map[string]*ContactInfo
	if len(contacts) > 0 {
		contactsPtr = contacts
	}

	return &DomainInfo{
		Name:           name,
		Nameservers:    nameservers,
		CreationDate:   created,
		ExpirationDate: expires,
		Registrar:      registrar,
		RegistrarURL:   registrarURL,
		Status:         status,
		Contacts:       contactsPtr,
	}
}
