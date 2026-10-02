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

// KeyUsage counts the safes wrapped by each key of the instance keyset, by
// key identifier. A key no safe uses any more can be removed from the keyset.
func (m *Manager) KeyUsage() (map[uint32]int, error) {
	usage, _, err := m.keyUsage()
	return usage, err
}

// keyUsage is KeyUsage, also counting the safes whose wrapping key cannot be
// told: a damaged record must not hide the usage of every other key.
func (m *Manager) keyUsage() (usage map[uint32]int, unreadable int, err error) {
	if m == nil || m.safes == nil {
		return nil, 0, errNoManager
	}

	usage = map[uint32]int{}
	err = eachSafe(m.safes.store, func(safe *happydns.Safe) {
		for _, w := range safe.Keyring {
			if w.KEK != KEKInstance {
				continue
			}
			id, err := wrappingKeyId(w)
			if err != nil {
				unreadable++
				continue
			}
			usage[id]++
		}
	}, func(string, error) {
		unreadable++
	})
	return usage, unreadable, err
}

// Rewrap wraps again, under the primary instance key, the key of every safe
// wrapped by another one. The sealed values are not touched. A safe that
// fails is reported and left as it was, and the others are rewrapped; run it
// again to resume.
func (m *Manager) Rewrap(ctx context.Context) (ResealReport, error) {
	report := ResealReport{ObjectType: SafeObjectType}
	if m == nil || m.safes == nil || m.safes.key == nil {
		return report, errors.New("no instance keyset configured")
	}

	var ids []happydns.Identifier
	err := eachSafe(m.safes.store, func(safe *happydns.Safe) {
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

		changed, err := m.rewrapSafe(id)
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

// rewrapSafe rewraps the safe id, as stored now, if it needs it.
func (m *Manager) rewrapSafe(id happydns.Identifier) (bool, error) {
	key := m.safes.key

	safe, err := m.safes.store.GetSafe(id)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}

	needed, err := key.needsRewrap(safe)
	if err != nil || !needed {
		return false, err
	}

	dek, err := key.Unwrap(safe)
	if err != nil {
		return false, err
	}
	w, err := key.Wrap(safe, dek)
	if err != nil {
		return false, err
	}

	// The new wrapping takes the place of the first instance entry; any
	// other one goes, they all wrapped the same key.
	keyring := make([]happydns.WrappedKeyset, 0, len(safe.Keyring))
	placed := false
	for _, e := range safe.Keyring {
		if e.KEK != KEKInstance {
			keyring = append(keyring, e)
		} else if !placed {
			keyring = append(keyring, w)
			placed = true
		}
	}
	safe.Keyring = keyring

	err = m.safes.store.UpdateSafe(safe)
	if errors.Is(err, happydns.ErrSafeNotFound) {
		return false, nil
	}
	return err == nil, err
}

// needsRewrap tells whether an instance entry of safe is wrapped by another
// key than the primary one.
func (k *InstanceKey) needsRewrap(safe *happydns.Safe) (bool, error) {
	for _, w := range safe.Keyring {
		if w.KEK != KEKInstance {
			continue
		}
		id, err := wrappingKeyId(w)
		if err != nil {
			return false, err
		}
		if id != k.PrimaryKeyId() {
			return true, nil
		}
	}
	return false, nil
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

// KeyStatus describes a key of the instance keyset and how many safes it
// wraps.
type KeyStatus struct {
	KeyInfo

	// Safes counts the safes whose key it wraps.
	Safes int `json:"safes"`
}

// KeyStatus lists the keys of the instance keyset with the number of safes
// each wraps. A key wrapping safes but missing from the keyset is listed
// with the status MISSING: those safes do not open. Safes whose wrapping key
// cannot be told, damaged ones, are counted under the status UNREADABLE.
func (m *Manager) KeyStatus() ([]KeyStatus, error) {
	if m == nil || m.safes == nil {
		return nil, errNoManager
	}

	usage, unreadable, err := m.keyUsage()
	if err != nil {
		return nil, err
	}

	var keys []KeyStatus
	if m.safes.key != nil {
		for _, k := range DescribeKeyset(m.safes.key.handle) {
			keys = append(keys, KeyStatus{KeyInfo: k, Safes: usage[k.Id]})
			delete(usage, k.Id)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(usage)) {
		keys = append(keys, KeyStatus{KeyInfo: KeyInfo{Id: id, Status: "MISSING"}, Safes: usage[id]})
	}
	if unreadable > 0 {
		keys = append(keys, KeyStatus{KeyInfo: KeyInfo{Status: "UNREADABLE"}, Safes: unreadable})
	}
	return keys, nil
}
