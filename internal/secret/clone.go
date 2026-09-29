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
)

// clone returns a copy of obj, a non-nil pointer to a struct, sharing none of
// the structs holding a Secret with it. Secrets themselves are shared: they
// are never modified in place.
func clone(obj any) (any, error) {
	v := reflect.ValueOf(obj)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("secret clone: want a non-nil pointer to a struct, got %T", obj)
	}

	cp, err := cloneStruct(v)
	if err != nil {
		return nil, err
	}
	return cp.Interface(), nil
}

// cloneStruct copies the struct ptr points at, and returns a pointer to the
// copy.
func cloneStruct(ptr reflect.Value) (reflect.Value, error) {
	cp := reflect.New(ptr.Type().Elem())
	cp.Elem().Set(ptr.Elem())

	if err := deepen(cp.Elem()); err != nil {
		return reflect.Value{}, err
	}
	return cp, nil
}

// deepen replaces, in the struct v, every pointer Walk could follow to a
// Secret by a pointer to a copy.
func deepen(v reflect.Value) error {
	t := v.Type()

	for i := range t.NumField() {
		sf := t.Field(i)
		if _, _, ok := jsonName(sf); !ok {
			continue
		}

		if err := deepenValue(v.Field(i)); err != nil {
			return fmt.Errorf("%s: %w", sf.Name, err)
		}
	}

	return nil
}

func deepenValue(fv reflect.Value) error {
	if fv.Type() == secretType {
		return nil
	}

	switch fv.Kind() {
	case reflect.Struct:
		return deepen(fv)

	case reflect.Pointer:
		if fv.IsNil() || !isStructLike(fv.Type()) || !containsSecret(fv.Type().Elem(), nil) {
			return nil
		}
		return replaceWithClone(fv, fv)

	case reflect.Interface:
		if fv.IsNil() {
			return nil
		}
		e := fv.Elem()
		if e.Kind() != reflect.Pointer || e.IsNil() || !isStructLike(e.Type()) || !containsSecret(e.Type().Elem(), nil) {
			return nil
		}
		return replaceWithClone(fv, e)
	}

	return nil
}

func replaceWithClone(dst, ptr reflect.Value) error {
	if !dst.CanSet() {
		return errors.New("secret clone: cannot copy a pointer held in an unexported field")
	}

	cp, err := cloneStruct(ptr)
	if err != nil {
		return err
	}

	dst.Set(cp)
	return nil
}

// reflectElem returns the struct obj, a pointer to a struct, points at.
func reflectElem(obj any) reflect.Value {
	return reflect.ValueOf(obj).Elem()
}
