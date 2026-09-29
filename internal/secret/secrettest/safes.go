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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"git.happydns.org/happyDomain/model"
)

// Safes keeps safes in memory, following the rules of the real storage
// (KVStorage) so that a test passing on it does not hide a bug the real one
// would reveal:
//
//   - records are kept encoded, so what is handed out or given is a copy;
//   - an owner index points to the safe of a kind each owner has, and
//     GetSafeByOwner follows it;
//   - a safe needs an owner, a kind and a well-formed identifier, not already
//     taken;
//   - the identifier, owner and kind of a safe cannot change;
//   - deleting or updating an unknown safe fails.
//
// Records written by Put bypass these rules, to stand for damaged or
// concurrent ones.
type Safes struct {
	mu      sync.Mutex
	records map[string][]byte
	// index maps the owner and kind of a safe to its identifier.
	index map[string]string

	// The counters count the successful writes through the storage
	// interface; Put is not counted.
	creates  int
	updates  int
	replaces int
	deletes  int
}

func NewSafes() *Safes {
	return &Safes{records: map[string][]byte{}, index: map[string]string{}}
}

func indexKey(owner happydns.Identifier, kind string) string {
	return owner.String() + "|" + kind
}

func decode(record []byte) (*happydns.Safe, error) {
	var safe happydns.Safe
	if err := json.Unmarshal(record, &safe); err != nil {
		return nil, err
	}
	return &safe, nil
}

// get returns the safe id; the caller holds the lock.
func (m *Safes) get(id string) (*happydns.Safe, error) {
	record, ok := m.records[id]
	if !ok {
		return nil, happydns.ErrSafeNotFound
	}
	return decode(record)
}

func (m *Safes) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// In the order of their keys, as the real storage lists them.
	ids := make([]string, 0, len(m.records))
	for id := range m.records {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	all := make([]*happydns.Safe, 0, len(ids))
	for _, id := range ids {
		safe, err := m.get(id)
		if err != nil {
			return nil, err
		}
		all = append(all, safe)
	}
	return &sliceIterator{items: all, idx: -1}, nil
}

func (m *Safes) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.get(id.String())
}

func (m *Safes) GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.index[indexKey(owner, kind)]
	if !ok {
		return nil, happydns.ErrSafeNotFound
	}
	safe, err := m.get(id)
	if err != nil {
		return nil, err
	}
	if !safe.Owner.Equals(owner) || safe.Kind != kind {
		return nil, fmt.Errorf("safe index %s points to a safe of another owner or kind", indexKey(owner, kind))
	}
	return safe, nil
}

func (m *Safes) CreateSafe(safe *happydns.Safe) error {
	if safe.Owner.IsEmpty() || safe.Kind == "" {
		return errors.New("a safe needs an owner and a kind")
	}
	if len(safe.Id) != happydns.IDENTIFIER_LEN {
		return fmt.Errorf("%w: %d bytes, want %d", happydns.ErrInvalidIdentifier, len(safe.Id), happydns.IDENTIFIER_LEN)
	}
	record, err := json.Marshal(safe)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, taken := m.records[safe.Id.String()]; taken {
		return fmt.Errorf("safe %s: %w", safe.Id.String(), happydns.ErrAlreadyExists)
	}
	if _, indexed := m.index[indexKey(safe.Owner, safe.Kind)]; indexed {
		return fmt.Errorf("owner already has a %s safe: %w", safe.Kind, happydns.ErrAlreadyExists)
	}
	m.records[safe.Id.String()] = record
	m.index[indexKey(safe.Owner, safe.Kind)] = safe.Id.String()
	m.creates++
	return nil
}

func (m *Safes) UpdateSafe(safe *happydns.Safe) error {
	record, err := json.Marshal(safe)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	old, err := m.get(safe.Id.String())
	if err != nil {
		return err
	}
	if !old.Owner.Equals(safe.Owner) || old.Kind != safe.Kind {
		return errors.New("the owner and kind of a safe cannot change")
	}
	m.records[safe.Id.String()] = record
	m.updates++
	return nil
}

// ReplaceSafe applies update to a copy of the stored safe, and stores the
// result unless the safe changed or was deleted in between.
func (m *Safes) ReplaceSafe(id happydns.Identifier, update func(*happydns.Safe) (*happydns.Safe, error)) error {
	m.mu.Lock()
	stored, ok := m.records[id.String()]
	m.mu.Unlock()
	if !ok {
		return happydns.ErrSafeNotFound
	}

	old, err := decode(stored)
	if err != nil {
		return err
	}
	owner, kind := slices.Clone(old.Owner), old.Kind

	next, err := update(old)
	if err != nil || next == nil {
		return err
	}
	if !next.Id.Equals(id) || !next.Owner.Equals(owner) || next.Kind != kind {
		return errors.New("the identifier, owner and kind of a safe cannot change")
	}
	record, err := json.Marshal(next)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.records[id.String()]
	if !ok {
		return happydns.ErrSafeNotFound
	}
	if !bytes.Equal(current, stored) {
		return fmt.Errorf("safe %s: %w", id.String(), happydns.ErrChangedMeanwhile)
	}
	m.records[id.String()] = record
	m.replaces++
	return nil
}

// DeleteSafe removes a safe and its owner index.
func (m *Safes) DeleteSafe(id happydns.Identifier) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	safe, err := m.get(id.String())
	if err != nil {
		return err
	}
	delete(m.index, indexKey(safe.Owner, safe.Kind))
	delete(m.records, id.String())
	m.deletes++
	return nil
}

// Put stores safe as is, as a record written behind the storage's back,
// damaged or concurrent: it bypasses the rules and the counters, and leaves
// the owner index alone.
func (m *Safes) Put(safe happydns.Safe) {
	record, err := json.Marshal(safe)
	if err != nil {
		panic(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records[safe.Id.String()] = record
}

// Len returns the number of safes stored.
func (m *Safes) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.records)
}

// Creates returns the number of safes CreateSafe stored.
func (m *Safes) Creates() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.creates
}

// Updates returns the number of safes UpdateSafe rewrote.
func (m *Safes) Updates() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updates
}

// Deletes returns the number of safes DeleteSafe removed.
func (m *Safes) Deletes() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deletes
}

// Replaces returns the number of safes ReplaceSafe rewrote.
func (m *Safes) Replaces() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.replaces
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
