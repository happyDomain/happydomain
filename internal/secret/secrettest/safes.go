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

// Package secrettest provides helpers to test code sealing secrets.
package secrettest

import (
	"fmt"
	"sync"

	"git.happydns.org/happyDomain/model"
)

// Safes keeps safes in memory, enforcing one safe of a kind per owner as
// the real storage does.
type Safes struct {
	mu    sync.Mutex
	safes map[string]happydns.Safe
}

func NewSafes() *Safes {
	return &Safes{safes: map[string]happydns.Safe{}}
}

func (m *Safes) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []*happydns.Safe
	for _, s := range m.safes {
		c := s
		all = append(all, &c)
	}
	return &sliceIterator{items: all, idx: -1}, nil
}

func (m *Safes) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.safes[id.String()]
	if !ok {
		return nil, happydns.ErrSafeNotFound
	}
	return &s, nil
}

func (m *Safes) GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.safes {
		if s.Owner.Equals(owner) && s.Kind == kind {
			c := s
			return &c, nil
		}
	}
	return nil, happydns.ErrSafeNotFound
}

func (m *Safes) CreateSafe(safe *happydns.Safe) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.safes {
		if s.Owner.Equals(safe.Owner) && s.Kind == safe.Kind {
			return fmt.Errorf("owner already has a %s safe", safe.Kind)
		}
	}
	m.safes[safe.Id.String()] = *safe
	return nil
}

func (m *Safes) UpdateSafe(safe *happydns.Safe) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.safes[safe.Id.String()]; !ok {
		return happydns.ErrSafeNotFound
	}
	m.safes[safe.Id.String()] = *safe
	return nil
}

func (m *Safes) DeleteSafe(id happydns.Identifier) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.safes, id.String())
	return nil
}

type sliceIterator struct {
	items []*happydns.Safe
	idx   int
}

func (it *sliceIterator) Next() bool           { it.idx++; return it.idx < len(it.items) }
func (it *sliceIterator) NextWithError() bool  { return it.Next() }
func (it *sliceIterator) Item() *happydns.Safe { return it.items[it.idx] }
func (it *sliceIterator) DropItem() error      { return nil }
func (it *sliceIterator) Key() string          { return "" }
func (it *sliceIterator) Raw() any             { return nil }
func (it *sliceIterator) Err() error           { return nil }
func (it *sliceIterator) Close()               {}
