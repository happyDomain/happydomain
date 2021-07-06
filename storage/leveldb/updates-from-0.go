// Copyright or © or Copr. happyDNS (2021)
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
