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
	"errors"

	"git.happydns.org/happyDomain/model"
)

// secretCheckKey holds the check record of the instance keyset: see
// secret.InstanceKey.VerifyCheck.
const secretCheckKey = "secret.check"

func (s *KVStorage) GetSecretCheck() ([]byte, error) {
	var record []byte
	err := s.db.Get(secretCheckKey, &record)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *KVStorage) PutSecretCheck(record []byte) error {
	return s.db.Put(secretCheckKey, record)
}

func (s *KVStorage) DeleteSecretCheck() error {
	return s.db.Delete(secretCheckKey)
}
