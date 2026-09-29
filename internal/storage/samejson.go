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


package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

// SameJSON reports whether a and b hold the same JSON value. Object keys may
// come in any order and spacing may differ; numbers are compared as written,
// so that no precision is lost on large integers.
func SameJSON(a, b []byte) (bool, error) {
	va, err := decodeJSONValue(a)
	if err != nil {
		return false, err
	}
	vb, err := decodeJSONValue(b)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(va, vb), nil
}

func decodeJSONValue(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the JSON value")
	}
	return v, nil
}
