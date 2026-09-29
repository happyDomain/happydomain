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

package database_test

import (
	"bytes"
	"errors"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestSecretCheckRecord(t *testing.T) {
	s := newStorage(t)

	if _, err := s.GetSecretCheck(); !errors.Is(err, happydns.ErrNotFound) {
		t.Fatalf("GetSecretCheck on an empty store = %v, want ErrNotFound", err)
	}

	record := []byte{0x01, 0x00, 0xff, 'c', 'h', 'e', 'c', 'k'}
	if err := s.PutSecretCheck(record); err != nil {
		t.Fatalf("PutSecretCheck: %v", err)
	}

	got, err := s.GetSecretCheck()
	if err != nil {
		t.Fatalf("GetSecretCheck: %v", err)
	}
	if !bytes.Equal(got, record) {
		t.Errorf("GetSecretCheck = %x, want %x", got, record)
	}
}
