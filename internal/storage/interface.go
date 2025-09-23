// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package storage // import "git.happydns.org/happyDomain/internal/storage"

import (
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/internal/usecase/check"
	"git.happydns.org/happyDomain/internal/usecase/domain"
	"git.happydns.org/happyDomain/internal/usecase/domain_log"
	"git.happydns.org/happyDomain/internal/usecase/insight"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/internal/usecase/user"
	"git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type ProviderAndDomainStorage interface {
	provider.ProviderStorage
	domain.DomainStorage
}

type Storage interface {
	authuser.AuthUserStorage
	domain.DomainStorage
	domainlog.DomainLogStorage
	insight.InsightStorage
	check.CheckerStorage
	provider.ProviderStorage
	session.SessionStorage
	user.UserStorage
	zone.ZoneStorage

	// SchemaVersion returns the version of the migration currently in use.
	SchemaVersion() int

	// DoMigration is the first function called.
	MigrateSchema() error

	// Close shutdown the connection with the database and releases all structure.
	Close() error
}

type Iterator interface {
	Release()
	Next() bool
	Valid() bool
	Key() string
	Value() any
}

type KVStorage interface {
	Close() error
	DecodeData(i any, v any) error
	Has(key string) (bool, error)
	Get(key string, v any) error
	Put(key string, v any) error
	FindIdentifierKey(prefix string) (key string, id happydns.Identifier, err error)
	Delete(key string) error
	Search(prefix string) Iterator
}
