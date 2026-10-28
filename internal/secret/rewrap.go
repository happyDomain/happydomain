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
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"git.happydns.org/happyDomain/model"
)

// SafeObjectType names safes in the reports.
const SafeObjectType = "safe"

// Rewrap wraps again, under the current key of its store, every keyring entry
// made under an older one. The sealed values are not touched. A safe that
// fails is reported and left as it was, and the others are rewrapped; run it
// again to resume.
func (m *Manager) Rewrap(ctx context.Context) (ResealReport, error) {
	report := ResealReport{ObjectType: SafeObjectType}
	if m == nil || m.safes == nil {
		return report, errNoManager
	}
	if !slices.ContainsFunc(slices.Collect(maps.Values(m.stores)), func(s Store) bool {
		_, ok := s.(Rewrapper)
		return ok
	}) {
		return report, errors.New("no secret store configured here rewraps its keys")
	}

	var ids []happydns.Identifier
	err := eachSafe(m.safes, func(safe *happydns.Safe) {
		ids = append(ids, safe.Id)
	}, func(key string, err error) {
		report.Processed++
		report.Failed++
		report.Errors = append(report.Errors, fmt.Sprintf("record %s: %s", key, err.Error()))
	})
	if err != nil {
		return report, err
	}

	for _, id := range ids {
		report.Processed++

		changed, err := m.rewrapSafe(ctx, id)
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("safe %s: %s", id.String(), err.Error()))
			continue
		}
		if changed {
			report.Changed++
		}
	}

	return report, nil
}

// rewrapSafe rewraps the entries of the safe id, as stored now, that need it.
func (m *Manager) rewrapSafe(ctx context.Context, id happydns.Identifier) (bool, error) {
	safe, err := m.safes.GetSafe(id)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}

	changed := false
	kept := map[string]bool{}
	keyring := make([]happydns.WrappedKeyset, 0, len(safe.Keyring))
	for _, w := range safe.Keyring {
		r, ok := m.stores[w.KEK].(Rewrapper)
		if !ok {
			keyring = append(keyring, w)
			continue
		}

		// The entries of one store all wrap the same key: only the first
		// stays, so that a stale one does not leave an older key able to
		// open the safe.
		if kept[w.KEK] {
			changed = true
			continue
		}
		kept[w.KEK] = true

		rewrapped, err := r.Rewrap(ctx, safe, &w)
		if err != nil {
			return false, err
		}
		changed = changed || rewrapped
		keyring = append(keyring, w)
	}
	if !changed {
		return false, nil
	}
	safe.Keyring = keyring

	err = m.safes.UpdateSafe(safe)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		return false, nil
	}
	return err == nil, err
}

// KeyInfo describes a key of a keyset, without its material.
type KeyInfo struct {
	Id      uint32 `json:"id"`
	Primary bool   `json:"primary"`
	Status  string `json:"status"`
	Type    string `json:"type,omitempty"`
}

func (k KeyInfo) String() string {
	primary := ""
	if k.Primary {
		primary = " (primary)"
	}
	return fmt.Sprintf("%d%s %s %s", k.Id, primary, k.Status, k.Type)
}

// KeyStatus describes a key of a store and how many safes it wraps.
type KeyStatus struct {
	// KEK names the store the key belongs to, as keyring entries do.
	KEK string `json:"kek"`

	KeyInfo

	// Safes counts the safes whose key it wraps.
	Safes int `json:"safes"`
}

// KeyStatus lists the keys of every store that reports them, with the number
// of safes each wraps. The entries made by a key no longer in its store, or
// by a store not configured here, are listed with the status MISSING: they do
// not open. Those of a store configured here that reports no keys, as one
// whose key its owner holds, are left out. Those whose wrapping
// key cannot be told, and the safe records that do not decode, are counted
// under the status UNREADABLE, the latter without a store.
func (m *Manager) KeyStatus() ([]KeyStatus, error) {
	if m == nil || m.safes == nil {
		return nil, errNoManager
	}

	usage := map[string]map[uint32]int{}
	unreported := map[string]int{}
	unreadable := map[string]int{}
	err := eachSafe(m.safes, func(safe *happydns.Safe) {
		for _, w := range safe.Keyring {
			kek := w.KEK
			store, configured := m.stores[w.KEK]
			var wrappingKey func(happydns.WrappedKeyset) (uint32, error)
			if r, ok := store.(KeyReporter); ok {
				wrappingKey = r.WrappingKey
			} else if configured {
				// A store reporting no keys, as one whose key its owner
				// holds, has none to miss.
				continue
			}
			if wrappingKey == nil {
				unreported[w.KEK]++
				continue
			}
			id, err := wrappingKey(w)
			if err != nil {
				unreadable[w.KEK]++
				continue
			}
			if usage[kek] == nil {
				usage[kek] = map[uint32]int{}
			}
			usage[kek][id]++
		}
	}, func(string, error) {
		unreadable[""]++
	})
	if err != nil {
		return nil, err
	}

	var keys []KeyStatus
	for _, kek := range slices.Sorted(maps.Keys(m.stores)) {
		r, ok := m.stores[kek].(KeyReporter)
		if !ok {
			continue
		}
		used := usage[kek]
		for _, k := range r.Keys() {
			keys = append(keys, KeyStatus{KEK: kek, KeyInfo: k, Safes: used[k.Id]})
			delete(used, k.Id)
		}
		for _, id := range slices.Sorted(maps.Keys(used)) {
			keys = append(keys, KeyStatus{KEK: kek, KeyInfo: KeyInfo{Id: id, Status: "MISSING"}, Safes: used[id]})
		}
	}
	for _, kek := range slices.Sorted(maps.Keys(unreported)) {
		keys = append(keys, KeyStatus{KEK: kek, KeyInfo: KeyInfo{Status: "MISSING"}, Safes: unreported[kek]})
	}
	for _, kek := range slices.Sorted(maps.Keys(unreadable)) {
		keys = append(keys, KeyStatus{KEK: kek, KeyInfo: KeyInfo{Status: "UNREADABLE"}, Safes: unreadable[kek]})
	}
	return keys, nil
}
