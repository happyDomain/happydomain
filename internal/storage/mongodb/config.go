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

package database

import (
	"flag"

	"git.happydns.org/happyDomain/internal/storage"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
)

var (
	uri    string
	dbName string
)

func init() {
	storage.StorageEngines["mongodb"] = Instantiate

	flag.StringVar(&uri, "mongodb-uri", "mongodb://localhost:27017", "MongoDB connection URI")
	flag.StringVar(&dbName, "mongodb-database", "happydomain", "MongoDB database name")
}

func Instantiate() (storage.Storage, error) {
	db, err := NewMongoDBStorage(uri, dbName)
	if err != nil {
		return nil, err
	}

	return kv.NewKVDatabase(db)
}
