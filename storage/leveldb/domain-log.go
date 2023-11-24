// Copyright or © or Copr. happyDNS (2023)
//
// contact@happydomain.org
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
	"fmt"
	"log"
	"strings"

	"git.happydns.org/happyDomain/model"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) GetDomainLogs(domain *happydns.Domain) (logs []*happydns.DomainLog, err error) {
	iter := s.search(fmt.Sprintf("domain.log|%s|", domain.Id.String()))
	defer iter.Release()

	for iter.Next() {
		var z happydns.DomainLog

		err = decodeData(iter.Value(), &z)
		if err != nil {
			return
		}

		logs = append(logs, &z)
	}

	return
}

func (s *LevelDBStorage) getDomainLog(id string) (l *happydns.DomainLog, d *happydns.Domain, err error) {
	l = &happydns.DomainLog{}
	err = s.get(id, l)

	st := strings.Split(id, "|")
	if len(st) < 3 {
		return
	}

	d = &happydns.Domain{}
	err = s.get(id, fmt.Sprintf("domain-%s", st[1]))

	return
}

func (s *LevelDBStorage) CreateDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	key, id, err := s.findIdentifierKey(fmt.Sprintf("domain.log|%s|", d.Id.String()))
	if err != nil {
		return err
	}

	l.Id = id
	return s.put(key, l)
}

func (s *LevelDBStorage) UpdateDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	return s.put(fmt.Sprintf("domain.log|%s|%s", d.Id.String(), l.Id.String()), l)
}

func (s *LevelDBStorage) DeleteDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	return s.delete(fmt.Sprintf("domain.log|%s|%s", d.Id.String(), l.Id.String()))
}

func (s *LevelDBStorage) TidyDomainLogs() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("domain.log|")), nil)
	defer iter.Release()

	for iter.Next() {
		l, _, err := s.getDomainLog(string(iter.Key()))

		if err != nil {
			if l != nil {
				log.Printf("Deleting log without valid domain: %s (%s)\n", l.Id.String(), err.Error())
			} else {
				log.Printf("Deleting unreadable log (%s): %v\n", err.Error(), l)
			}
			err = tx.Delete(iter.Key(), nil)
		}

		if err != nil {
			tx.Discard()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Discard()
		return err
	}

	return nil
}
