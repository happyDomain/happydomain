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
	"encoding/hex"
	"errors"
)

type HexaString []byte

func (hs *HexaString) MarshalJSON() (dst []byte, err error) {
	dst = make([]byte, hex.EncodedLen(len(*hs)))
	hex.Encode(dst, *hs)
	dst = append([]byte{'"'}, dst...)
	dst = append(dst, '"')
	return
}

func (hs *HexaString) UnmarshalJSON(b []byte) (err error) {
	if len(b) == 0 || b[0] != '"' || b[len(b)-1] != '"' {
		return errors.New("Expected JSON string")
	}

	*hs = make([]byte, hex.DecodedLen(len(b)-2))
	_, err = hex.Decode(*hs, b[1:len(b)-1])

	return
}
