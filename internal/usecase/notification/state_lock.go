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

package notification

import (
	"fmt"
	"sync"

	"git.happydns.org/happyDomain/model"
)

// Per-key serialization for state read-modify-write; without it, ack and dispatcher races wipe acks or fire duplicates. In-process only — single instance.
type StateLocker struct {
	mu    sync.Mutex
	locks map[string]*stateLockEntry
}

type stateLockEntry struct {
	mu       sync.Mutex
	refCount int
}

func NewStateLocker() *StateLocker {
	return &StateLocker{locks: make(map[string]*stateLockEntry)}
}

// Always defer the returned unlock — leaking it pins the map entry.
func (l *StateLocker) Lock(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) func() {
	key := stateLockKey(checkerID, target, userId)

	l.mu.Lock()
	entry, ok := l.locks[key]
	if !ok {
		entry = &stateLockEntry{}
		l.locks[key] = entry
	}
	entry.refCount++
	l.mu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()

		l.mu.Lock()
		entry.refCount--
		if entry.refCount == 0 {
			delete(l.locks, key)
		}
		l.mu.Unlock()
	}
}

// Must match the storage tuple exactly; mismatch silently re-introduces the race.
func stateLockKey(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) string {
	return fmt.Sprintf(
		"%s|%s|%s/%s/%s",
		userId.String(),
		checkerID,
		target.UserId,
		target.DomainId,
		target.ServiceId,
	)
}
