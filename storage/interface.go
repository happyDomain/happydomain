package storage // import "happydns.org/storage"

import (
	"git.happydns.org/happydns/model"
)

type Storage interface {
	DoMigration() error
	Close() error

	GetSession(id []byte) (*happydns.Session, error)
	CreateSession(session *happydns.Session) error
	UpdateSession(session *happydns.Session) error
	DeleteSession(session *happydns.Session) error
	ClearSessions() error

	GetUsers() (happydns.Users, error)
	GetUser(id int) (*happydns.User, error)
	GetUserByEmail(email string) (*happydns.User, error)
	UserExists(email string) bool
	CreateUser(user *happydns.User) error
	UpdateUser(user *happydns.User) error
	DeleteUser(user *happydns.User) error
	ClearUsers() error

	GetZones(u *happydns.User) (happydns.Zones, error)
	GetZone(u *happydns.User, id int) (*happydns.Zone, error)
	GetZoneByDN(u *happydns.User, dn string) (*happydns.Zone, error)
	ZoneExists(dn string) bool
	CreateZone(u *happydns.User, z *happydns.Zone) error
	UpdateZone(z *happydns.Zone) error
	UpdateZoneOwner(z *happydns.Zone, newOwner *happydns.User) error
	DeleteZone(z *happydns.Zone) error
	ClearZones() error
}

type UserStorage interface {
	DoMigration() error
	Close() error

	GetSession(id []byte) (*happydns.Session, error)
	CreateSession(session *happydns.Session) error
	UpdateSession(session *happydns.Session) error
	DeleteSession(session *happydns.Session) error
	ClearSessions() error

	GetUsers() (happydns.Users, error)
	GetUser(id int) (*happydns.User, error)
	GetUserByEmail(email string) (*happydns.User, error)
	UserExists(email string) bool
	CreateUser(user *happydns.User) error
	UpdateUser(user *happydns.User) error
	DeleteUser(user *happydns.User) error
	ClearUsers() error
}
