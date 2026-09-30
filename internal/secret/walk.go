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
	"slices"
	"strconv"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// ErrUnsupportedSecret is returned when a happydns.Secret sits where Walk
// cannot reach it for sure: in a slice, an array, a map other than one of
// Secrets with string keys, behind a pointer to the Secret itself, or in an
// interface holding a struct by value.
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
// The Secrets of a map with string keys are visited in key order, under the
// path of the map followed by the quoted key: `headers["X-Token"]`. What fn
// does to one is stored back in the map.
//
// Secrets are found by their type, no tag needed. One it cannot reach for
// sure is an error rather than skipped: a secret silently left out would be
// stored unsealed.
func Walk(obj any, fn WalkFunc) error {
	v := reflect.ValueOf(obj)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("secret walk: want a non-nil pointer to a struct, got %T", obj)
	}

	w := &walker{visit: func(path string, fv reflect.Value) error {
		return fn(path, fv.Addr().Interface().(*happydns.Secret))
	}}
	return w.follow(v, "")
}

// walker is one walk. It remembers the structs it followed a pointer to, so
// that a cycle ends and a struct reached twice is noticed.
type walker struct {
	visit func(path string, fv reflect.Value) error

	// lenient skips a struct reached twice instead of refusing it.
	lenient bool

	visited map[visitKey]bool
}

type visitKey struct {
	addr uintptr
	typ  reflect.Type
}

// follow walks the struct ptr, a non-nil pointer to a struct, points at.
func (w *walker) follow(ptr reflect.Value, path string) error {
	key := visitKey{ptr.Pointer(), ptr.Type()}
	if w.visited[key] {
		// A secret reached under two paths would be sealed for one and read
		// back for the other: its associated data cannot match both.
		if w.lenient || !reachesSecret(ptr) {
			return nil
		}
		return fmt.Errorf("%s: %w: a struct holding secrets is reached twice", path, ErrUnsupportedSecret)
	}
	if w.visited == nil {
		w.visited = map[visitKey]bool{}
	}
	w.visited[key] = true

	return w.walkStruct(ptr.Elem(), path)
}

func (w *walker) walkStruct(v reflect.Value, prefix string) error {
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

		if err := w.walkValue(v.Field(i), path); err != nil {
			return err
		}
	}

	return nil
}

func (w *walker) walkValue(fv reflect.Value, path string) error {
	if fv.Type() == secretType {
		if err := w.visit(path, fv); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	}

	switch fv.Kind() {
	case reflect.Struct:
		return w.walkStruct(fv, path)

	case reflect.Map:
		if isSecretMap(fv.Type()) {
			return w.walkMap(fv, path)
		}

	case reflect.Pointer:
		if isStructPointer(fv.Type()) {
			if fv.IsNil() {
				return nil
			}
			return w.follow(fv, path)
		}

	case reflect.Interface:
		if fv.IsNil() {
			return nil
		}
		e := fv.Elem()
		if isStructPointer(e.Type()) && !e.IsNil() {
			return w.follow(e, path)
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

// walkMap visits the Secrets of fv, a map of Secrets with string keys, in
// key order, and stores back those the visit changed.
func (w *walker) walkMap(fv reflect.Value, path string) error {
	keys := fv.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })

	for _, k := range keys {
		// Map values cannot be addressed: the visit gets a copy.
		orig := fv.MapIndex(k)
		val := reflect.New(secretType).Elem()
		val.Set(orig)

		kpath := path + "[" + strconv.Quote(k.String()) + "]"
		if err := w.visit(kpath, val); err != nil {
			return fmt.Errorf("%s: %w", kpath, err)
		}

		// Only when changed: a walk that only reads leaves the map, maybe
		// shared, untouched.
		if !reflect.DeepEqual(val.Interface(), orig.Interface()) {
			fv.SetMapIndex(k, val)
		}
	}

	return nil
}

// isSecretMap reports whether t is a map of Secrets with string keys, the
// only map Walk goes into.
func isSecretMap(t reflect.Type) bool {
	return t.Kind() == reflect.Map && t.Key().Kind() == reflect.String && t.Elem() == secretType
}

// errFound stops reachesSecret at the first secret.
var errFound = errors.New("secret found")

// reachesSecret reports whether Walk would find a Secret in the struct ptr, a
// non-nil pointer to a struct, points at, or refuse one it cannot reach. It
// looks at the values, not only the types: an interface only counts for what
// it holds.
func reachesSecret(ptr reflect.Value) bool {
	w := &walker{
		lenient: true,
		visit:   func(string, reflect.Value) error { return errFound },
	}
	return w.follow(ptr, "") != nil
}

func isStructPointer(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct && t.Elem() != secretType
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
