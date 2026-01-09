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

package database

import (
	"flag"

	"git.happydns.org/happyDomain/internal/storage"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
)

type PostgreSQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Table    string
	SSLMode  string
}

var cfg PostgreSQLConfig

func init() {
	storage.StorageEngines["postgresql"] = Instantiate

	flag.StringVar(&cfg.Host, "postgres-host", "localhost", "PostgreSQL server hostname")
	flag.IntVar(&cfg.Port, "postgres-port", 5432, "PostgreSQL server port")
	flag.StringVar(&cfg.User, "postgres-user", "happydomain", "PostgreSQL username")
	flag.StringVar(&cfg.Password, "postgres-password", "", "PostgreSQL password")
	flag.StringVar(&cfg.Database, "postgres-database", "happydomain", "PostgreSQL database name")
	flag.StringVar(&cfg.Table, "postgres-table", "happydomain_kv", "PostgreSQL table name for key-value storage")
	flag.StringVar(&cfg.SSLMode, "postgres-ssl-mode", "disable", "PostgreSQL SSL mode (disable, require, verify-ca, verify-full)")
}

func Instantiate() (storage.Storage, error) {
	db, err := NewPostgreSQLStorage(&cfg)
	if err != nil {
		return nil, err
	}

	return kv.NewKVDatabase(db)
}
