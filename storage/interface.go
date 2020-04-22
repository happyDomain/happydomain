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
