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

// Package types holds the registration information types shared between
// pkg/domaininfo and the happyDomain model. It has no dependency, so the
// model can reference these types without pulling the RDAP and WHOIS
// clients.
package types

import (
	"context"
	"errors"
	"time"
)

// ErrDomainDoesNotExist is returned when the registry reports that the
// domain name is not registered.
var ErrDomainDoesNotExist = errors.New("domain name doesn't exist")

type ContactInfo struct {
	Name         string `json:"name,omitempty"`
	Organization string `json:"organization,omitempty"`
	Email        string `json:"email,omitempty"`
	Street       string `json:"street,omitempty"`
	City         string `json:"city,omitempty"`
	Province     string `json:"province,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	Country      string `json:"country,omitempty"`
	Phone        string `json:"phone,omitempty"`
}

type DomainInfo struct {
	Name           string                  `json:"name"`
	Nameservers    []string                `json:"nameservers"`
	CreationDate   *time.Time              `json:"creation"`
	ExpirationDate *time.Time              `json:"expiration"`
	Registrar      string                  `json:"registrar"`
	RegistrarURL   *string                 `json:"registrar_url"`
	Status         []string                `json:"status"`
	Contacts       map[string]*ContactInfo `json:"contacts,omitempty"`
}

// Getter retrieves the registration information of a domain name.
type Getter func(context.Context, string) (*DomainInfo, error)
