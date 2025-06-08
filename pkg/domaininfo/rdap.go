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
	"time"

	"git.happydns.org/happyDomain/model"

	"github.com/openrdap/rdap"
)

func GetDomainRDAPInfo(domain happydns.Origin) (*happydns.DomainInfo, error) {
	client := &rdap.Client{}
	domainInfo, err := client.QueryDomain(string(domain))
	if err != nil {
		if ce, ok := err.(*rdap.ClientError); ok && ce.Type == rdap.ObjectDoesNotExist {
			return nil, happydns.DomainDoesNotExist
		}
		return nil, err
	}

	// Registrar
	registrar := "Unknown"
	var registrar_url *string
	for _, ent := range domainInfo.Entities {
		if ent.Roles != nil {
			for _, role := range ent.Roles {
				if role == "registrar" && ent.VCard != nil && len(ent.VCard.Get("fn")) > 0 {
					registrar = ent.VCard.Get("fn")[0].Value.(string)
					if len(ent.VCard.Get("url")) > 0 {
						url := ent.VCard.Get("url")[0].Value.(string)
						registrar_url = &url
					}
				}
			}
		}
	}

	// Dates
	var expiration *time.Time
	var creation *time.Time
	for _, event := range domainInfo.Events {
		if (event.Action == "expiration" || event.Action == "registration") && event.Date != "" {
			date, err := time.Parse(time.RFC3339, event.Date)
			if err != nil {
				return nil, err
			}

			if event.Action == "expiration" {
				expiration = &date
			} else if event.Action == "registration" {
				creation = &date
			}
		}
	}

	// Nameservers
	var nameservers []string
	for _, nameserver := range domainInfo.Nameservers {
		if nameserver.UnicodeName != "" {
			nameservers = append(nameservers, nameserver.UnicodeName)
		} else {
			nameservers = append(nameservers, nameserver.LDHName)
		}
	}

	name := domainInfo.UnicodeName
	if name == "" {
		name = domainInfo.LDHName
	}

	return &happydns.DomainInfo{
		Name:           name,
		Nameservers:    nameservers,
		CreationDate:   creation,
		ExpirationDate: expiration,
		Registrar:      registrar,
		RegistrarURL:   registrar_url,
		Status:         domainInfo.Status,
	}, nil
}
