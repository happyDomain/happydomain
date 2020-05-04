// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package storage // import "happydns.org/storage"

import (
	"git.happydns.org/happydns/model"
)

type Storage interface {
	DoMigration() error
	Close() error

	GetDomains(u *happydns.User) (happydns.Domains, error)
	GetDomain(u *happydns.User, id int) (*happydns.Domain, error)
	GetDomainByDN(u *happydns.User, dn string) (*happydns.Domain, error)
	DomainExists(dn string) bool
	CreateDomain(u *happydns.User, z *happydns.Domain) error
	UpdateDomain(z *happydns.Domain) error
	UpdateDomainOwner(z *happydns.Domain, newOwner *happydns.User) error
	DeleteDomain(z *happydns.Domain) error
	ClearDomains() error

	GetSession(id []byte) (*happydns.Session, error)
	CreateSession(session *happydns.Session) error
	UpdateSession(session *happydns.Session) error
	DeleteSession(session *happydns.Session) error
	ClearSessions() error

	GetSourceTypes(u *happydns.User) ([]happydns.SourceType, error)
	GetSource(u *happydns.User, id int64) (*happydns.SourceCombined, error)
	CreateSource(u *happydns.User, s happydns.Source, comment string) (*happydns.SourceCombined, error)
	UpdateSource(s *happydns.SourceCombined) error
	UpdateSourceOwner(s *happydns.SourceCombined, newOwner *happydns.User) error
	DeleteSource(s *happydns.SourceType) error
	ClearSources() error

	GetUsers() (happydns.Users, error)
	GetUser(id int) (*happydns.User, error)
	GetUserByEmail(email string) (*happydns.User, error)
	UserExists(email string) bool
	CreateUser(user *happydns.User) error
	UpdateUser(user *happydns.User) error
	DeleteUser(user *happydns.User) error
	ClearUsers() error
}
