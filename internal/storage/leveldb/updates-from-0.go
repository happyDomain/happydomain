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

package database

import (
	"bytes"
	"log"
)

func migrateFrom0(s *LevelDBStorage) (err error) {
	err = migrateFrom0_sourcesProvider(s)
	if err != nil {
		return
	}

	err = migrateFrom0_reparentDomains(s)
	if err != nil {
		return
	}

	return
}

type sourceMeta struct {
	Type    string `json:"_srctype"`
	Id      int64  `json:"_id"`
	OwnerId int64  `json:"_ownerid"`
	Comment string `json:"_comment,omitempty"`
}

func migrateFrom0_sourcesProvider(s *LevelDBStorage) (err error) {
	iter := s.search("source-")
	defer iter.Release()

	for iter.Next() {
		src := iter.Value()
		for src[0] == '"' {
			err = decodeData(src, &src)
			if err != nil {
				return
			}
		}

		src = bytes.Replace(src, []byte("\"Source\":"), []byte("\"Provider\":"), 1)

		var srcMeta sourceMeta
		err = decodeData(src, &srcMeta)
		if err != nil {
			return
		}

		newType := ""

		switch srcMeta.Type {
		case "ddns.DDNSServer":
			newType = "DDNSServer"
		case "DDNSServer":
			newType = "DDNSServer"
		case "gandi.GandiAPI":
			newType = "GandiAPI"
		case "GandiAPI":
			newType = "GandiAPI"
		case "ovh.OVHAPI":
			newType = "OVHAPI"
		case "OVHAPI":
			newType = "OVHAPI"
		default:
			// Keep other source type to update in future version
			log.Printf("Migrating v0 -> v1: skip %s (%s)...", iter.Key(), srcMeta.Type)
			continue
		}

		if newType != "" {
			src = bytes.Replace(src, []byte(srcMeta.Type), []byte(newType), 1)

			if newType == "DDNSServer" {
				src = bytes.Replace(src, []byte("\"hmac-md5.sig-alg.reg.int.\""), []byte("\"hmac-md5\""), 1)
				src = bytes.Replace(src, []byte("\"hmac-sha1.\""), []byte("\"hmac-sha1\""), 1)
				src = bytes.Replace(src, []byte("\"hmac-sha256.\""), []byte("\"hmac-sha256\""), 1)
				src = bytes.Replace(src, []byte("\"hmac-sha512.\""), []byte("\"hmac-sha512\""), 1)
				src = bytes.Replace(src, []byte(".\""), []byte("\""), 1)
			}
		}

		newKey := bytes.Replace(iter.Key(), []byte("source-"), []byte("provider-"), 1)

		log.Printf("Migrating v0 -> v1: %s to %s (%s)...", iter.Key(), newKey, newType)

		err = s.db.Put(newKey, src, nil)
		if err != nil {
			return
		}

		err = s.delete(string(iter.Key()))
		if err != nil {
			return
		}
	}

	return
}

func migrateFrom0_reparentDomains(s *LevelDBStorage) (err error) {
	iter := s.search("domain-")
	defer iter.Release()

	for iter.Next() {
		domstr := iter.Value()

		domstr = bytes.Replace(domstr, []byte("\"id_source\":"), []byte("\"id_provider\":"), 1)

		log.Printf("Migrating v0 -> v1: %s...", iter.Key())

		err = s.db.Put(iter.Key(), domstr, nil)
		if err != nil {
			return
		}
	}

	return
}
