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

// Package secret seals and opens the credentials happyDomain stores on behalf
// of its users. See docs/secret-management.md for the design.
package secret

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// ErrMalformedSealed is returned for a value carrying the sealed prefix that
// cannot be parsed.
var ErrMalformedSealed = errors.New("malformed sealed secret")

// aadVersion is the first component of every associated data. It follows the
// sealed prefix, so a new format never shares associated data with this one.
const aadVersion = "hds:1"

var b64 = base64.RawURLEncoding.Strict()

// Sealed is a parsed sealed value: hds:1:<safe id>:<payload>.
type Sealed struct {
	// SafeId names the safe that opens the payload.
	SafeId happydns.Identifier

	// Payload depends on the store of the safe: the ciphertext for the
	// inline store.
	Payload []byte
}

// IsSealed reports whether value carries the sealed prefix. It says nothing
// of whether the rest parses: see ParseSealed.
func IsSealed(value string) bool {
	return strings.HasPrefix(value, happydns.SealedSecretPrefix)
}

// FormatSealed returns the sealed value naming safeId and holding payload.
func FormatSealed(safeId happydns.Identifier, payload []byte) string {
	return happydns.SealedSecretPrefix + b64.EncodeToString(safeId) + ":" + b64.EncodeToString(payload)
}

// ParseSealed parses a value produced by FormatSealed. It accepts only the
// canonical encoding, so what it accepts formats back to the same string.
func ParseSealed(value string) (Sealed, error) {
	rest, ok := strings.CutPrefix(value, happydns.SealedSecretPrefix)
	if !ok {
		return Sealed{}, fmt.Errorf("%w: missing %q prefix", ErrMalformedSealed, happydns.SealedSecretPrefix)
	}

	id, payload, ok := strings.Cut(rest, ":")
	if !ok {
		return Sealed{}, fmt.Errorf("%w: missing payload", ErrMalformedSealed)
	}
	if id == "" {
		return Sealed{}, fmt.Errorf("%w: empty safe identifier", ErrMalformedSealed)
	}
	if payload == "" {
		return Sealed{}, fmt.Errorf("%w: empty payload", ErrMalformedSealed)
	}

	var sv Sealed
	var err error

	if sv.SafeId, err = decodeCanonical(id); err != nil {
		return Sealed{}, fmt.Errorf("%w: safe identifier: %w", ErrMalformedSealed, err)
	}
	if sv.Payload, err = decodeCanonical(payload); err != nil {
		return Sealed{}, fmt.Errorf("%w: payload: %w", ErrMalformedSealed, err)
	}

	return sv, nil
}

func decodeCanonical(s string) ([]byte, error) {
	b, err := b64.DecodeString(s)
	if err != nil {
		return nil, err
	}
	// Strict already refuses non-zero trailing bits; this also catches
	// anything else that would not format back byte for byte.
	if b64.EncodeToString(b) != s {
		return nil, errors.New("non-canonical encoding")
	}
	return b, nil
}

// SecretContext tells where a secret belongs. It is bound to the ciphertext as
// associated data, so a sealed value copied to another field, object or user
// does not open.
type SecretContext struct {
	// Owner is the user the secret belongs to, or InstanceOwner() for the
	// secrets of the instance itself: it is not always a user identifier.
	Owner happydns.Identifier

	// ObjectType names the kind of object holding the secret: "provider"...
	ObjectType string

	// ObjectId identifies the object among those of its type. A string, so
	// that objects without a single identifier (checker options, keyed by
	// their target) fit.
	ObjectId string

	// Field is the path of JSON names to the secret in the object, stable
	// across Go renames: "ApiToken", "auth.token".
	Field string
}

// Validate refuses a context missing any component: associated data built
// from it would bind the secret to less than it should.
func (sc SecretContext) Validate() error {
	if err := sc.validateObject(); err != nil {
		return err
	}
	if sc.Field == "" {
		return errors.New("secret context: missing field")
	}
	return nil
}

// validateObject is Validate without the field, which Walk fills.
func (sc SecretContext) validateObject() error {
	switch {
	case len(sc.Owner) == 0:
		return errors.New("secret context: missing owner")
	case sc.ObjectType == "":
		return errors.New("secret context: missing object type")
	case sc.ObjectId == "":
		return errors.New("secret context: missing object identifier")
	}
	return nil
}

// AssociatedData returns the AEAD associated data of a secret sealed in
// safeId, in the lengthPrefixed layout so that no two different contexts give
// the same bytes.
//
// This is a persisted format: changing it makes every stored secret
// unreadable.
func AssociatedData(safeId happydns.Identifier, sc SecretContext) []byte {
	return lengthPrefixed(
		[]byte(aadVersion),
		safeId,
		sc.Owner,
		[]byte(sc.ObjectType),
		[]byte(sc.ObjectId),
		[]byte(sc.Field),
	)
}

// lengthPrefixed concatenates parts, each prefixed with its length as a
// big-endian uint32.
//
// This is the layout of persisted associated data: changing it makes every
// stored secret and safe unreadable.
func lengthPrefixed(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += 4 + len(p)
	}

	ad := make([]byte, 0, n)
	for _, p := range parts {
		ad = binary.BigEndian.AppendUint32(ad, uint32(len(p)))
		ad = append(ad, p...)
	}
	return ad
}

// binding tells where a secret belongs, whatever its safe: an opened Secret
// keeps it, so that sealing it again elsewhere is noticed.
func (sc SecretContext) binding() string {
	return string(AssociatedData(nil, sc))
}
