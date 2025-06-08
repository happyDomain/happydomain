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

	"git.happydns.org/happyDomain/model"

	"github.com/likexian/whois"
	"github.com/likexian/whois-parser"
)

func GetDomainWhoisInfo(ctx context.Context, domain happydns.Origin) (*happydns.DomainInfo, error) {
	client := whois.NewClient()

	// The whois library has no context support; derive a timeout from the
	// context deadline so we at least honour it approximately.
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			client.SetTimeout(remaining)
		}
	}

	raw, err := client.Whois(string(domain))
	if err != nil {
		return nil, err
	}

	result, err := whoisparser.Parse(raw)
	if err != nil {
		if errors.Is(err, whoisparser.ErrNotFoundDomain) {
			return nil, happydns.DomainDoesNotExist
		}
		return nil, err
	}

	registrar := "Unknown"
	var registrar_url *string
	if result.Registrar != nil {
		registrar = result.Registrar.Name
		registrar_url = &result.Registrar.ReferralURL
	}

	return &happydns.DomainInfo{
		Name:           result.Domain.Domain,
		Nameservers:    result.Domain.NameServers,
		CreationDate:   result.Domain.CreatedDateInTime,
		ExpirationDate: result.Domain.ExpirationDateInTime,
		Registrar:      registrar,
		RegistrarURL:   registrar_url,
		Status:         result.Domain.Status,
	}, nil
}
