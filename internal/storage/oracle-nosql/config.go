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
	"io"
	"os"

	"git.happydns.org/happyDomain/internal/storage"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
)

type OCINoSQLConfig struct {
	Region               string
	Table                string
	tenancy              string
	user                 string
	fingerprint          string
	compartment          string
	privateKeyFile       string
	privateKeyPassphrase string
}

var cfg OCINoSQLConfig

func init() {
	storage.StorageEngines["oracle-nosql"] = Instantiate

	flag.StringVar(&cfg.Region, "oci-region", "us-phoenix-1", "OCI region where the NoSQL database is located")
	flag.StringVar(&cfg.Table, "oci-table", "happydomain", "Table name where values are stored")
	flag.StringVar(&cfg.tenancy, "oci-tenancy", cfg.tenancy, "OCI tenancy ID where is located the NoSQL database")
	flag.StringVar(&cfg.user, "oci-user", cfg.user, "OCI user ID accessing the NoSQL database")
	flag.StringVar(&cfg.fingerprint, "oci-fingerprint", cfg.fingerprint, "OCI user API key fingerprint")
	flag.StringVar(&cfg.compartment, "oci-compartment", cfg.compartment, "OCI compartment ID where the NoSQL database lies")
	flag.StringVar(&cfg.privateKeyFile, "oci-private-key-file", cfg.privateKeyFile, "Path to the OCI private key for the given user")
}

func Instantiate() (storage.Storage, error) {
	db, err := NewOCINoSQLStorage(&cfg)
	if err != nil {
		return nil, err
	}

	return kv.NewKVDatabase(db)
}

func (cfg *OCINoSQLConfig) privateKey() (string, error) {
	fd, err := os.Open(cfg.privateKeyFile)
	if err != nil {
		return "", err
	}

	cnt, err := io.ReadAll(fd)
	if err != nil {
		return "", err
	}

	return string(cnt), err
}
