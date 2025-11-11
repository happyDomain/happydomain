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

package happydns

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"errors"
)

const IDENTIFIER_LEN = 16

func NewIdentifierFromString(src string) (id Identifier, err error) {
	return base64.RawURLEncoding.DecodeString(src)
}

func NewRandomIdentifier() (id Identifier, err error) {
	id = make([]byte, IDENTIFIER_LEN)

	if _, err = rand.Read(id); err != nil {
		return
	}

	return
}

func (i *Identifier) IsEmpty() bool {
	return len(*i) == 0
}

func (i Identifier) Equals(other Identifier) bool {
	return bytes.Equal(i, other)
}

func (i *Identifier) String() string {
	return base64.RawURLEncoding.EncodeToString(*i)
}

func (i Identifier) MarshalJSON() (dst []byte, err error) {
	dst = make([]byte, base64.RawURLEncoding.EncodedLen(len(i)))
	base64.RawURLEncoding.Encode(dst, i)
	dst = append([]byte{'"'}, dst...)
	dst = append(dst, '"')
	return
}

func (i *Identifier) UnmarshalJSON(src []byte) error {
	if len(src) < 2 || src[0] != '"' || src[len(src)-1] != '"' {
		return errors.New("Invalid character encapsulating the JSON value")
	}

	*i = make([]byte, base64.RawURLEncoding.DecodedLen(len(src)-2))
	_, err := base64.RawURLEncoding.Decode(*i, src[1:len(src)-1])

	return err
}

func init() {
	gob.Register(Identifier{})
}
