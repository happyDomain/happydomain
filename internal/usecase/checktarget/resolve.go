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

// Package checktarget resolves a CheckTarget's DomainId to the name it
// refers to. A CheckTarget.DomainId is overloaded: it names either a real
// Domain or a DomainAvailabilityWatch (watches are not real domains, so they
// share the same identifier space rather than getting a distinct field).
// Every consumer of CheckTarget that needs the underlying name has to
// perform the same "try the domain store, fall back to the watch store"
// resolution, so it lives here once instead of being reimplemented per
// caller.
package checktarget // import "git.happydns.org/happyDomain/internal/usecase/checktarget"

import (
	"git.happydns.org/happyDomain/model"
)

// DomainGetter is the minimal interface needed to resolve a CheckTarget's
// DomainId against the real domain store.
type DomainGetter interface {
	GetDomain(id happydns.Identifier) (*happydns.Domain, error)
}

// WatchGetter is the minimal interface needed to resolve a CheckTarget's
// DomainId against the availability-watch store.
type WatchGetter interface {
	GetDomainAvailabilityWatch(id happydns.Identifier) (*happydns.DomainAvailabilityWatch, error)
}

// Resolve looks up id as a real Domain first. When the domain store fails to
// return it, for any reason (not found, or otherwise), id is retried against
// watchStore, since it may refer to a DomainAvailabilityWatch instead.
//
// domain is non-nil only when id resolved to a real Domain, so callers that
// also need e.g. zone/service data can use it; watches carry no such data.
// err is the original domain-lookup error, returned only when neither
// resolution succeeded.
func Resolve(id happydns.Identifier, domainStore DomainGetter, watchStore WatchGetter) (domain *happydns.Domain, domainName string, err error) {
	domain, err = domainStore.GetDomain(id)
	if err == nil {
		return domain, domain.DomainName, nil
	}

	if watchStore != nil {
		if watch, werr := watchStore.GetDomainAvailabilityWatch(id); werr == nil {
			return nil, watch.DomainName, nil
		}
	}

	return nil, "", err
}
