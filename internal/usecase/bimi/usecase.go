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

// Package bimi serves the files the BIMI records point to, for the domains
// that asked happyDomain to host them.
package bimi

import (
	"git.happydns.org/happyDomain/internal/usecase/hostedservice"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

type Usecase struct {
	store hostedservice.Store
}

func NewUsecase(domains hostedservice.DomainFinder, zones hostedservice.ZoneGetter) *Usecase {
	return &Usecase{store: hostedservice.NewStore(domains, zones)}
}

// IsManaged reports whether happyDomain hosts the BIMI files for the given
// FQDN. Used by the Caddy on-demand TLS ask endpoint.
func (uc *Usecase) IsManaged(fqdn string) (bool, error) {
	return hostedservice.IsManaged[*abstract.BIMI](uc.store, fqdn, abstract.BIMIHostPrefix)
}

// Logo returns nothing unless name is the one of the zone being edited or of
// the published one.
func (uc *Usecase) Logo(hostFQDN, name string) ([]byte, error) {
	return uc.file(hostFQDN, func(svc *abstract.BIMI) []byte { return svc.LogoFile(name) })
}

// Certificate returns nothing unless name is the one of the zone being edited
// or of the published one.
func (uc *Usecase) Certificate(hostFQDN, name string) ([]byte, error) {
	return uc.file(hostFQDN, func(svc *abstract.BIMI) []byte { return svc.CertificateFile(name) })
}

func (uc *Usecase) file(hostFQDN string, get func(*abstract.BIMI) []byte) ([]byte, error) {
	// Unlike the MTA-STS policy, the files are user content: they are never
	// served from a host name other than the bimi. one, such as happyDomain's.
	domain, ok := hostedservice.StripPrefix(hostFQDN, abstract.BIMIHostPrefix)
	if !ok {
		return nil, happydns.ErrNotFound
	}

	svcs, err := hostedservice.FindServices[*abstract.BIMI](uc.store, domain)
	if err != nil {
		return nil, err
	}

	// The zone being edited and the published one can name a file differently
	// until the first is applied: the live record refers to the second. A stale
	// name, a missing file or a service no longer hosted all leave nothing to
	// serve.
	for _, svc := range svcs {
		if body := get(svc); body != nil {
			return body, nil
		}
	}

	return nil, happydns.ErrNotFound
}
