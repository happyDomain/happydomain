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

package forms // import "git.happydns.org/happyDomain/forms"

import (
	"bytes"
	"reflect"

	"git.happydns.org/happyDomain/model"
)

// RedactSecrets replaces the value of every field tagged `secret` with
// happydns.RedactedSecret, recursing into embedded and nested structs the same
// way Endpoints does.
//
// It exists because the credentials a user entrusts to happyDomain live in a
// struct whose shape differs for each of the 60+ providers, so there is no one
// type to write a MarshalJSON on. Reading the tags back is the only place we
// can decide, once, that a stored credential never leaves over the user API.
//
// It mutates data in place, so callers must own what they pass. That holds for
// the provider handlers: GetUserProvider and the provider middleware both
// ParseProvider a fresh body out of the store on every request, so no one else
// holds a reference to it.
//
// A field that is empty stays empty: the sentinel means "a value is stored and
// withheld", and claiming that for a credential the user never filled in would
// make an empty field look set.
func RedactSecrets(data any) {
	v := reflect.Indirect(reflect.ValueOf(data))
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return
	}

	redactStruct(v)
}

func redactStruct(v reflect.Value) {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !isWalkable(sf) {
			continue
		}

		fv := v.Field(i)

		if inner := structValue(fv); inner.IsValid() {
			redactStruct(inner)
			if sf.Anonymous {
				continue
			}
		}

		if !GenField(sf).Secret {
			continue
		}

		redactValue(fv)
	}
}

func redactValue(fv reflect.Value) {
	if !fv.CanSet() {
		return
	}

	switch {
	case fv.Kind() == reflect.String:
		if fv.Len() == 0 {
			return
		}
		fv.SetString(happydns.RedactedSecret)
	case isByteSlice(fv.Type()):
		if fv.Len() == 0 {
			return
		}
		// Base64 in JSON, so the sentinel round-trips like any other blob.
		fv.SetBytes([]byte(happydns.RedactedSecret))
	default:
		if fv.IsZero() {
			return
		}
		// Fail closed. No secret of another kind exists today; should one
		// appear, it is emitted zeroed rather than in clear, and MergeSecrets
		// reads that zero back as "unchanged".
		fv.SetZero()
	}
}

// MergeSecrets carries stored secrets forward across a write: wherever incoming
// still holds the sentinel RedactSecrets put there, the value from existing is
// restored. Without it, the redacted body a client just read back and submitted
// again would write the placeholder into the database.
//
// existing may be nil, which is what creation looks like. There is then nothing
// to carry forward, so the sentinel is cleared instead: a `required` field
// reports itself as empty rather than letting the placeholder reach a provider
// API as if it were a credential.
//
// Only incoming is mutated. When the two bodies are of different types, which
// means the user changed the provider type, existing is ignored for the same
// reason.
func MergeSecrets(existing, incoming any) {
	iv := reflect.Indirect(reflect.ValueOf(incoming))
	if !iv.IsValid() || iv.Kind() != reflect.Struct {
		return
	}

	ev := reflect.Indirect(reflect.ValueOf(existing))
	if !ev.IsValid() || ev.Type() != iv.Type() {
		ev = reflect.Value{}
	}

	mergeStruct(ev, iv)
}

// ev may be invalid, meaning "nothing stored to carry forward".
func mergeStruct(ev, iv reflect.Value) {
	t := iv.Type()

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !isWalkable(sf) {
			continue
		}

		ifv := iv.Field(i)

		var efv reflect.Value
		if ev.IsValid() {
			efv = ev.Field(i)
		}

		if inner := structValue(ifv); inner.IsValid() {
			var existingInner reflect.Value
			if efv.IsValid() {
				existingInner = structValue(efv)
			}
			mergeStruct(existingInner, inner)
			if sf.Anonymous {
				continue
			}
		}

		if !GenField(sf).Secret {
			continue
		}

		mergeValue(efv, ifv)
	}
}

func mergeValue(efv, ifv reflect.Value) {
	if !ifv.CanSet() {
		return
	}

	stored := efv.IsValid() && efv.Type() == ifv.Type()

	switch {
	case ifv.Kind() == reflect.String:
		if ifv.String() != happydns.RedactedSecret {
			return
		}
		if stored {
			ifv.SetString(efv.String())
		} else {
			ifv.SetString("")
		}

	case isByteSlice(ifv.Type()):
		if !bytes.Equal(ifv.Bytes(), []byte(happydns.RedactedSecret)) {
			return
		}
		if stored {
			// Copy: the two bodies must not share a backing array, the caller
			// is about to keep one and drop the other.
			b := make([]byte, efv.Len())
			copy(b, efv.Bytes())
			ifv.SetBytes(b)
		} else {
			ifv.SetBytes(nil)
		}

	default:
		// Mirror of redactValue's fail-closed branch.
		if !ifv.IsZero() {
			return
		}
		if stored {
			ifv.Set(efv)
		}
	}
}

// isWalkable reports whether sf is worth descending into.
//
// A plain unexported field is skipped: reflect refuses to set it, and no form
// ever fills one. An embedded one is not, even when its type is unexported:
// reflect drops the read-only mark one level down, so the exported fields it
// promotes are settable, and those are the ones carrying the tags.
func isWalkable(sf reflect.StructField) bool {
	return sf.PkgPath == "" || sf.Anonymous
}

// structValue unwraps fv to the struct it is or points at, or returns an
// invalid Value when there is none to walk into.
func structValue(fv reflect.Value) reflect.Value {
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return reflect.Value{}
		}
		fv = fv.Elem()
	}

	if fv.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	return fv
}

func isByteSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}
