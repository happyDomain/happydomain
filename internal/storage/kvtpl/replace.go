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
	"encoding/json"
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// replace rewrites key with what update returns from the value stored there,
// decoded as In. update runs with no lock held, so it may itself use the
// storage. The write only happens if key still holds what was read:
// otherwise it fails with notFound when key was deleted in between, and
// happydns.ErrChangedMeanwhile when it was changed. update returning nil
// writes nothing.
func replace[In, Out any](db storage.KVStorage, key string, notFound error, update func(*In) (*Out, error)) error {
	var raw json.RawMessage
	err := db.Get(key, &raw)
	if errors.Is(err, happydns.ErrNotFound) {
		return notFound
	}
	if err != nil {
		return err
	}

	var current In
	if err := json.Unmarshal(raw, &current); err != nil {
		return err
	}

	next, err := update(&current)
	if err != nil || next == nil {
		return err
	}

	stored, err := db.PutIfUnchanged(key, raw, next)
	if err != nil || stored {
		return err
	}

	if exists, err := db.Has(key); err != nil {
		return err
	} else if !exists {
		return notFound
	}
	return fmt.Errorf("%s: %w", key, happydns.ErrChangedMeanwhile)
}
