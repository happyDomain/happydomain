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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// checkNewIdentifier refuses to create an object under a missing or malformed
// identifier. The caller generates it, with happydns.NewRandomIdentifier, and
// never takes it from a request: the storage cannot tell the two apart, but it
// can at least keep its keys well formed.
func checkNewIdentifier(id happydns.Identifier) error {
	if len(id) != happydns.IDENTIFIER_LEN {
		return fmt.Errorf("%w: %d bytes, want %d", happydns.ErrInvalidIdentifier, len(id), happydns.IDENTIFIER_LEN)
	}
	return nil
}

// createNew stores v under key, the primary key of the new what identified by
// id, then writes its indexes with index, removing v again if that fails. It
// fails with happydns.ErrInvalidIdentifier when id is missing or malformed,
// and with happydns.ErrAlreadyExists when it is taken.
func (s *KVStorage) createNew(what string, id happydns.Identifier, key string, v any, index func() error) error {
	if err := checkNewIdentifier(id); err != nil {
		return err
	}

	// Checking first then writing would let a concurrent creation of the
	// same identifier land in between: the write itself refuses it.
	if stored, err := s.db.PutIfAbsent(key, v); err != nil {
		return err
	} else if !stored {
		return fmt.Errorf("%s %s: %w", what, id.String(), happydns.ErrAlreadyExists)
	}

	if err := index(); err != nil {
		return errors.Join(err, s.db.Delete(key))
	}
	return nil
}
