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
	"unsafe"
)

// clone returns a copy of obj, a non-nil pointer to a struct, sharing none of
// the structs holding a Secret with it. Secrets themselves are shared: they
// are never modified in place. What holds no secret, found by looking at the
// values as Walk does, is shared too: an http.Client reached through a pointer
// keeps its locks and pools to itself.
//
// A struct holding secrets reached twice, through a cycle or two pointers, is
// an error, as it is for Walk.
func clone(obj any) (any, error) {
	v := reflect.ValueOf(obj)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("secret clone: want a non-nil pointer to a struct, got %T", obj)
	}

	c := &cloner{visited: map[visitKey]bool{}}
	c.visited[visitKey{v.Pointer(), v.Type()}] = true

	cp, err := c.cloneStruct(v)
	if err != nil {
		return nil, err
	}
	return cp.Interface(), nil
}

// cloner is one clone. It remembers the structs it copied, by their address
// in the original.
type cloner struct {
	visited map[visitKey]bool
}

// cloneStruct copies the struct ptr points at, and returns a pointer to the
// copy.
func (c *cloner) cloneStruct(ptr reflect.Value) (reflect.Value, error) {
	cp := reflect.New(ptr.Type().Elem())
	cp.Elem().Set(ptr.Elem())

	if err := c.deepen(cp.Elem()); err != nil {
		return reflect.Value{}, err
	}
	return cp, nil
}

// deepen replaces, in the struct v, every pointer through which Walk finds a
// Secret by a pointer to a copy.
func (c *cloner) deepen(v reflect.Value) error {
	t := v.Type()

	for i := range t.NumField() {
		sf := t.Field(i)
		if _, _, ok := jsonName(sf); !ok {
			continue
		}

		if err := c.deepenValue(v.Field(i)); err != nil {
			return fmt.Errorf("%s: %w", sf.Name, err)
		}
	}

	return nil
}

func (c *cloner) deepenValue(fv reflect.Value) error {
	if fv.Type() == secretType {
		return nil
	}

	switch fv.Kind() {
	case reflect.Struct:
		return c.deepen(fv)

	case reflect.Pointer:
		if fv.IsNil() || !isStructPointer(fv.Type()) {
			return nil
		}
		return c.replaceWithClone(fv)

	case reflect.Interface:
		if fv.IsNil() {
			return nil
		}
		e := fv.Elem()
		if !isStructPointer(e.Type()) || e.IsNil() {
			return nil
		}
		return c.replaceWithClone(fv)
	}

	return nil
}

// replaceWithClone replaces the pointer dst holds, directly or in an
// interface, by a pointer to a copy, when a Secret can be found through it.
func (c *cloner) replaceWithClone(dst reflect.Value) error {
	if !dst.CanSet() {
		// An embedded pointer to a struct of unexported type: Walk follows
		// it as encoding/json does, so it must be copied too. dst lies in
		// the copy cloneStruct allocated, never in the original.
		if !dst.CanAddr() {
			return errors.New("secret clone: cannot copy a pointer held in an unexported field")
		}
		dst = reflect.NewAt(dst.Type(), unsafe.Pointer(dst.UnsafeAddr())).Elem()
	}

	ptr := dst
	if dst.Kind() == reflect.Interface {
		ptr = dst.Elem()
	}

	if !reachesSecret(ptr) {
		return nil
	}

	key := visitKey{ptr.Pointer(), ptr.Type()}
	if c.visited[key] {
		return fmt.Errorf("%w: a struct holding secrets is reached twice", ErrUnsupportedSecret)
	}
	c.visited[key] = true

	cp, err := c.cloneStruct(ptr)
	if err != nil {
		return err
	}

	dst.Set(cp)
	return nil
}
