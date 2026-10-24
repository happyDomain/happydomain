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

package domainwatch

import (
	"git.happydns.org/happyDomain/model"
)

// DomainWatchStorage stores the domains users watch.
type DomainWatchStorage interface {
	// ListAllDomainWatches lists the watches of every user, for the
	// background checks and the administrative passes.
	ListAllDomainWatches() (happydns.Iterator[happydns.DomainWatch], error)
	ListDomainWatchesByUser(userId happydns.Identifier) ([]*happydns.DomainWatch, error)
	GetDomainWatch(id happydns.Identifier) (*happydns.DomainWatch, error)
	// CreateDomainWatch stores a new watch, giving it its identifier.
	CreateDomainWatch(w *happydns.DomainWatch) error
	UpdateDomainWatch(w *happydns.DomainWatch) error
	DeleteDomainWatch(id happydns.Identifier) error
	// RestoreDomainWatch stores w under its identifier, replacing any
	// watch stored under it, as a backup restore does.
	RestoreDomainWatch(w *happydns.DomainWatch) error
	// TidyDomainWatchIndexes removes stale user index entries.
	TidyDomainWatchIndexes() error
}
