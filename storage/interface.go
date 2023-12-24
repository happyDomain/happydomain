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

package storage // import "git.happydns.org/happyDomain/storage"

import (
	"git.happydns.org/happyDomain/model"
)

type Storage interface {
	// DoMigration is the first function called.
	DoMigration() error

	// Tidy should optimize the database, looking for orphan records, ...
	Tidy() error

	// Close shutdown the connection with the database and releases all structure.
	Close() error

	// AUTH -------------------------------------------------------

	// GetAuthUsers retrieves the list of known Users.
	GetAuthUsers() (happydns.UserAuths, error)

	// GetAuthUser retrieves the User with the given identifier.
	GetAuthUser(id happydns.Identifier) (*happydns.UserAuth, error)

	// GetAuthUserByEmail retrives the User with the given email address.
	GetAuthUserByEmail(email string) (*happydns.UserAuth, error)

	// AuthUserExists checks if the given email address is already associated to an User.
	AuthUserExists(email string) bool

	// CreateAuthUser creates a record in the database for the given User.
	CreateAuthUser(user *happydns.UserAuth) error

	// UpdateAuthUser updates the fields of the given User.
	UpdateAuthUser(user *happydns.UserAuth) error

	// DeleteAuthUser removes the given User from the database.
	DeleteAuthUser(user *happydns.UserAuth) error

	// ClearAuthUsers deletes all AuthUsers present in the database.
	ClearAuthUsers() error

	// DOMAINS ----------------------------------------------------

	// GetDomains retrieves all Domains associated to the given User.
	GetDomains(u *happydns.User) (happydns.Domains, error)

	// GetDomain retrieves the Domain with the given id and owned by the given User.
	GetDomain(u *happydns.User, id happydns.Identifier) (*happydns.Domain, error)

	// GetDomainByDN is like GetDomain but look for the domain name instead of identifier.
	GetDomainByDN(u *happydns.User, dn string) (*happydns.Domain, error)

	// DomainExists looks if the given domain name alread exists in the database.
	DomainExists(dn string) bool

	// CreateDomain creates a record in the database for the given Domain.
	CreateDomain(u *happydns.User, z *happydns.Domain) error

	// UpdateDomain updates the fields of the given Domain.
	UpdateDomain(z *happydns.Domain) error

	// UpdateDomainOwner updates the owner of the given Domain.
	UpdateDomainOwner(z *happydns.Domain, newOwner *happydns.User) error

	// DeleteDomain removes the given Domain from the database.
	DeleteDomain(z *happydns.Domain) error

	// ClearDomains deletes all Domains present in the database.
	ClearDomains() error

	// DOMAIN LOGS --------------------------------------------------

	GetDomainLogs(*happydns.Domain) ([]*happydns.DomainLog, error)

	CreateDomainLog(*happydns.Domain, *happydns.DomainLog) error

	UpdateDomainLog(*happydns.Domain, *happydns.DomainLog) error

	DeleteDomainLog(*happydns.Domain, *happydns.DomainLog) error

	// PROVIDERS ----------------------------------------------------

	// GetProviderMetas retrieves provider's metadatas of all providers own by the given User.
	GetProviderMetas(u *happydns.User) ([]happydns.ProviderMeta, error)

	// GetProviderMeta retrieves the metadatas for the Provider with the given identifier and owner.
	GetProviderMeta(u *happydns.User, id happydns.Identifier) (*happydns.ProviderMeta, error)

	// GetProvider retrieves the full Provider with the given identifier and owner.
	GetProvider(u *happydns.User, id happydns.Identifier) (*happydns.ProviderCombined, error)

	// CreateProvider creates a record in the database for the given Provider.
	CreateProvider(u *happydns.User, s happydns.Provider, comment string) (*happydns.ProviderCombined, error)

	// UpdateProvider updates the fields of the given Provider.
	UpdateProvider(s *happydns.ProviderCombined) error

	// UpdateProviderOwner updates the owner of the given Provider.
	UpdateProviderOwner(s *happydns.ProviderCombined, newOwner *happydns.User) error

	// DeleteProvider removes the given Provider from the database.
	DeleteProvider(s *happydns.ProviderMeta) error

	// ClearProviders deletes all Providers present in the database.
	ClearProviders() error

	// SESSIONS ---------------------------------------------------

	// GetSession retrieves the Session with the given identifier.
	GetSession(id happydns.Identifier) (*happydns.Session, error)

	// GetAuthUserSessions retrieves all Session for the given AuthUser.
	GetAuthUserSessions(user *happydns.UserAuth) ([]*happydns.Session, error)

	// GetUserSessions retrieves all Session for the given User.
	GetUserSessions(user *happydns.User) ([]*happydns.Session, error)

	// CreateSession creates a record in the database for the given Session.
	CreateSession(session *happydns.Session) error

	// UpdateSession updates the fields of the given Session.
	UpdateSession(session *happydns.Session) error

	// DeleteSession removes the given Session from the database.
	DeleteSession(session *happydns.Session) error

	// ClearSessions deletes all Sessions present in the database.
	ClearSessions() error

	// USERS ------------------------------------------------------

	// GetUsers retrieves the list of known Users.
	GetUsers() (happydns.Users, error)

	// GetUser retrieves the User with the given identifier.
	GetUser(id happydns.Identifier) (*happydns.User, error)

	// GetUserByEmail retrives the User with the given email address.
	GetUserByEmail(email string) (*happydns.User, error)

	// CreateUser creates a record in the database for the given User.
	CreateUser(user *happydns.User) error

	// UpdateUser updates the fields of the given User.
	UpdateUser(user *happydns.User) error

	// DeleteUser removes the given User from the database.
	DeleteUser(user *happydns.User) error

	// ClearUsers deletes all Users present in the database.
	ClearUsers() error

	// ZONES ------------------------------------------------------

	// GetZoneMeta retrives metadatas of the Zone with the given identifier.
	GetZoneMeta(id happydns.Identifier) (*happydns.ZoneMeta, error)

	// GetZone retrieves the full Zone (including Services and metadatas) which have the given identifier.
	GetZone(id happydns.Identifier) (*happydns.Zone, error)

	// CreateZone creates a record in the database for the given Zone.
	CreateZone(zone *happydns.Zone) error

	// UpdateZone updates the fields of the given Zone.
	UpdateZone(zone *happydns.Zone) error

	// DeleteZone removes the given Zone from the database.
	DeleteZone(zone *happydns.Zone) error

	// ClearZones deletes all Zones present in the database.
	ClearZones() error
}
