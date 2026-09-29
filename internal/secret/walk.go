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

package secret

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// ErrUnsupportedSecret is returned when a happydns.Secret sits where Walk
// cannot reach it for sure: in a map, a slice, an array, behind a pointer to
// the Secret itself, or in an interface holding a struct by value.
var ErrUnsupportedSecret = errors.New("secret in an unsupported position")

var secretType = reflect.TypeFor[happydns.Secret]()

// WalkFunc is called for every Secret found by Walk, with the path of JSON
// names leading to it. Its error aborts the walk.
type WalkFunc func(path string, s *happydns.Secret) error

// Walk calls fn on every happydns.Secret held by obj, a non-nil pointer to a
// struct, in field order. It follows nested and embedded structs, non-nil
// pointers to structs and interfaces holding one; it skips what encoding/json
// skips (unexported fields, `json:"-"`).
//
// Secrets are found by their type, no tag needed. One it cannot reach for
// sure is an error rather than skipped: a secret silently left out would be
// stored unsealed.
func Walk(obj any, fn WalkFunc) error {
	v := reflect.ValueOf(obj)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("secret walk: want a non-nil pointer to a struct, got %T", obj)
	}

	return walkStruct(v.Elem(), "", fn)
}

func walkStruct(v reflect.Value, prefix string, fn WalkFunc) error {
	t := v.Type()

	for i := range t.NumField() {
		sf := t.Field(i)

		name, tagged, ok := jsonName(sf)
		if !ok {
			continue
		}

		path := joinPath(prefix, name)
		if sf.Anonymous && !tagged && isStructLike(sf.Type) {
			// Promoted into the parent object, as encoding/json does.
			path = prefix
		}

		if err := walkValue(v.Field(i), path, fn); err != nil {
			return err
		}
	}

	return nil
}

func walkValue(fv reflect.Value, path string, fn WalkFunc) error {
	if fv.Type() == secretType {
		if err := fn(path, fv.Addr().Interface().(*happydns.Secret)); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	}

	switch fv.Kind() {
	case reflect.Struct:
		return walkStruct(fv, path, fn)

	case reflect.Pointer:
		if fv.Type().Elem().Kind() == reflect.Struct && fv.Type().Elem() != secretType {
			if fv.IsNil() {
				return nil
			}
			return walkStruct(fv.Elem(), path, fn)
		}

	case reflect.Interface:
		if fv.IsNil() {
			return nil
		}
		e := fv.Elem()
		if e.Kind() == reflect.Pointer && !e.IsNil() && e.Elem().Kind() == reflect.Struct && e.Type().Elem() != secretType {
			return walkStruct(e.Elem(), path, fn)
		}
		if containsSecret(e.Type(), nil) {
			return fmt.Errorf("%s: %w", path, ErrUnsupportedSecret)
		}
		return nil
	}

	if containsSecret(fv.Type(), nil) {
		return fmt.Errorf("%s: %w", path, ErrUnsupportedSecret)
	}
	return nil
}

// containsSecret reports whether a value of type t may hold a Secret. An
// interface is not known statically: Walk looks into its value instead.
func containsSecret(t reflect.Type, seen map[reflect.Type]bool) bool {
	if t == secretType {
		return true
	}
	if seen[t] {
		return false
	}
	if seen == nil {
		seen = map[reflect.Type]bool{}
	}
	seen[t] = true

	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return containsSecret(t.Elem(), seen)
	case reflect.Map:
		return containsSecret(t.Key(), seen) || containsSecret(t.Elem(), seen)
	case reflect.Struct:
		for i := range t.NumField() {
			sf := t.Field(i)
			if _, _, ok := jsonName(sf); ok && containsSecret(sf.Type, seen) {
				return true
			}
		}
	}
	return false
}

// jsonName returns the name encoding/json gives to sf, whether it comes from
// a tag, and false when encoding/json skips the field.
func jsonName(sf reflect.StructField) (name string, tagged bool, ok bool) {
	if !sf.IsExported() {
		// encoding/json still promotes the fields of an embedded struct of
		// unexported type.
		if !sf.Anonymous || !isStructLike(sf.Type) {
			return "", false, false
		}
	}

	tag := sf.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}

	name, _, _ = strings.Cut(tag, ",")
	if name != "" {
		return name, true, true
	}
	return sf.Name, false, true
}

func isStructLike(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct && t != secretType
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
