// Copyright or © or Copr. happyDNS (2022)
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

package happydns

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

const IDENTIFIER_LEN = 16

type Identifier []byte

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
		return errors.New("Unvalid character found to encapsulate the JSON value")
	}

	*i = make([]byte, base64.RawURLEncoding.DecodedLen(len(src)-2))
	_, err := base64.RawURLEncoding.Decode(*i, src[1:len(src)-1])

	return err
}
