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

// discovery.go persists the two host-side indexes behind the cross-checker
// discovery mechanism described in docs/checker-discovery.md:
//
//   - dscent|{producer}|{target}|{type}|{ref}         primary record
//   - dscent-tgt|{target}|{producer}|{type}|{ref}     target lookup (auto-fill)
//   - dscobs|{producer}|{target}|{ref}|{consumer}|{k} observation lineage
//   - dscobs-snap|{snapshotId}|...                     cascade on snapshot delete
//
// Refs and observation keys are opaque to the host; we trust producers not
// to embed "|" in them (the SDK doc recommends short, deterministic values
// such as "host:port" or a sha1 digest).

package database

import (
	"fmt"
	"strings"

	"git.happydns.org/happyDomain/model"
)

const (
	discoveryPrimaryPrefix = "dscent|"
	discoveryTargetIndex   = "dscent-tgt|"
)

func dscEntryKey(producerID string, target happydns.CheckTarget, typ, ref string) string {
	return fmt.Sprintf("%s%s|%s|%s|%s", discoveryPrimaryPrefix, producerID, target.String(), typ, ref)
}

func dscEntryTargetIndexKey(producerID string, target happydns.CheckTarget, typ, ref string) string {
	return fmt.Sprintf("%s%s|%s|%s|%s", discoveryTargetIndex, target.String(), producerID, typ, ref)
}

func dscObsKey(producerID string, target happydns.CheckTarget, ref, consumerID string, obsKey happydns.ObservationKey) string {
	return fmt.Sprintf("dscobs|%s|%s|%s|%s|%s", producerID, target.String(), ref, consumerID, obsKey)
}

func dscObsSnapIndexKey(snapshotID happydns.Identifier, primary string) string {
	// The primary key is appended verbatim so cascade delete can recover it
	// without parsing the suffix; the value carries it too for safety.
	return fmt.Sprintf("dscobs-snap|%s|%s", snapshotID.String(), primary)
}

// --- DiscoveryEntry storage -------------------------------------------------

// dscEntryTargetSearchPrefix returns the key prefix that matches the given
// target scope plus any narrower scope. RawURLEncoded identifiers never
// contain "/" or "|", so slash boundaries in the encoded "u/d/s" target form
// are unambiguous for prefix matching:
//
//   - service scope ("u/d/s") → "dscent-tgt|u/d/s|"        (exact)
//   - domain  scope ("u/d/")  → "dscent-tgt|u/d/"          (this domain + any service under it)
//   - user    scope ("u//")   → "dscent-tgt|u/"            (this user + any domain/service)
//   - empty   scope ("//")    → "dscent-tgt|"              (all)
func dscEntryTargetSearchPrefix(target happydns.CheckTarget) string {
	const base = discoveryTargetIndex
	switch {
	case target.ServiceId != "":
		return base + target.String() + "|"
	case target.DomainId != "":
		return base + target.UserId + "/" + target.DomainId + "/"
	case target.UserId != "":
		return base + target.UserId + "/"
	default:
		return base
	}
}

// parseTargetFromIndexKey splits the encoded "u/d/s" portion of a target-index
// key back into a CheckTarget. The encoding is always 3 fields separated by
// "/", with empty strings for unset scopes.
func parseTargetFromIndexKey(s string) (happydns.CheckTarget, bool) {
	parts := strings.SplitN(s, "/", 3)
	if len(parts) != 3 {
		return happydns.CheckTarget{}, false
	}
	return happydns.CheckTarget{
		UserId:    parts[0],
		DomainId:  parts[1],
		ServiceId: parts[2],
	}, true
}

// ListDiscoveryEntriesByTarget returns every entry published at the given
// target scope or any narrower scope. A domain-scoped consumer therefore
// receives entries published at that domain itself and at any service under
// it; a user-scoped consumer additionally sees entries from any domain it
// owns. This mirrors the way checkers are layered — service-scoped producers
// (checker-dane, checker-smtp, …) routinely emit tls.endpoint.v1 entries
// that domain-scoped consumers (checker-tls, checker-caa) need to aggregate.
func (s *KVStorage) ListDiscoveryEntriesByTarget(target happydns.CheckTarget) ([]*happydns.StoredDiscoveryEntry, error) {
	iterPrefix := dscEntryTargetSearchPrefix(target)
	iter := s.db.Search(iterPrefix)
	defer iter.Release()

	const indexPrefix = discoveryTargetIndex
	var out []*happydns.StoredDiscoveryEntry
	for iter.Next() {
		rest := strings.TrimPrefix(iter.Key(), indexPrefix)
		// rest = "{u}/{d}/{s}|{producer}|{type}|{ref}"
		parts := strings.SplitN(rest, "|", 4)
		if len(parts) != 4 {
			continue
		}
		actualTarget, ok := parseTargetFromIndexKey(parts[0])
		if !ok {
			continue
		}
		entry := &happydns.StoredDiscoveryEntry{}
		if err := s.db.Get(dscEntryKey(parts[1], actualTarget, parts[2], parts[3]), entry); err != nil {
			// Stale index entry — ignore; tidy will eventually clean it.
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *KVStorage) ListDiscoveryEntriesByProducer(producerID string, target happydns.CheckTarget) ([]*happydns.StoredDiscoveryEntry, error) {
	prefix := fmt.Sprintf("%s%s|%s|", discoveryPrimaryPrefix, producerID, target.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var out []*happydns.StoredDiscoveryEntry
	for iter.Next() {
		entry := &happydns.StoredDiscoveryEntry{}
		if err := s.db.DecodeData(iter.Value(), entry); err != nil {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *KVStorage) ListAllDiscoveryEntries() (happydns.Iterator[happydns.StoredDiscoveryEntry], error) {
	iter := s.db.Search(discoveryPrimaryPrefix)
	return NewKVIterator[happydns.StoredDiscoveryEntry](s.db, iter), nil
}

// ReplaceDiscoveryEntries atomically replaces the set of entries stored for
// (producerID, target): everything previously stored is deleted, then the
// new set is written, in a single batch. Passing an empty `entries` slice
// simply clears.
func (s *KVStorage) ReplaceDiscoveryEntries(producerID string, target happydns.CheckTarget, entries []happydns.DiscoveryEntry) error {
	batch := s.db.NewBatch()

	// Stage deletes for every existing entry (primary + target index).
	prefix := fmt.Sprintf("%s%s|%s|", discoveryPrimaryPrefix, producerID, target.String())
	iter := s.db.Search(prefix)
	for iter.Next() {
		rest := strings.TrimPrefix(iter.Key(), prefix)
		parts := strings.SplitN(rest, "|", 2)
		if len(parts) == 2 {
			batch.Delete(dscEntryTargetIndexKey(producerID, target, parts[0], parts[1]))
		}
		batch.Delete(iter.Key())
	}
	iter.Release()

	for _, e := range entries {
		stored := &happydns.StoredDiscoveryEntry{
			ProducerID: producerID,
			Target:     target,
			Type:       e.Type,
			Ref:        e.Ref,
			Payload:    e.Payload,
		}
		if err := batch.Put(dscEntryKey(producerID, target, e.Type, e.Ref), stored); err != nil {
			return err
		}
		if err := batch.Put(dscEntryTargetIndexKey(producerID, target, e.Type, e.Ref), true); err != nil {
			return err
		}
	}
	return batch.Commit()
}

// RestoreDiscoveryEntry writes an entry at its canonical key and rebuilds
// its target index. Used by the backup restore path.
func (s *KVStorage) RestoreDiscoveryEntry(entry *happydns.StoredDiscoveryEntry) error {
	batch := s.db.NewBatch()
	if err := batch.Put(dscEntryKey(entry.ProducerID, entry.Target, entry.Type, entry.Ref), entry); err != nil {
		return err
	}
	if err := batch.Put(dscEntryTargetIndexKey(entry.ProducerID, entry.Target, entry.Type, entry.Ref), true); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) DeleteDiscoveryEntriesByProducer(producerID string, target happydns.CheckTarget) error {
	prefix := fmt.Sprintf("%s%s|%s|", discoveryPrimaryPrefix, producerID, target.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	batch := s.db.NewBatch()
	for iter.Next() {
		rest := strings.TrimPrefix(iter.Key(), prefix)
		parts := strings.SplitN(rest, "|", 2)
		if len(parts) == 2 {
			batch.Delete(dscEntryTargetIndexKey(producerID, target, parts[0], parts[1]))
		}
		batch.Delete(iter.Key())
	}
	return batch.Commit()
}

func (s *KVStorage) ClearDiscoveryEntries() error {
	if err := s.clearByPrefix(discoveryTargetIndex); err != nil {
		return err
	}
	return s.clearByPrefix(discoveryPrimaryPrefix)
}

// --- DiscoveryObservationRef storage ----------------------------------------

// PutDiscoveryObservationRef writes ref at its canonical primary key and
// maintains the per-snapshot index used by cascade delete. The function
// reads the existing primary so it can drop the previous snap-index entry
// when the snapshot id changes: without that cleanup, a later cascade
// delete of the older snapshot would wipe the primary this call just wrote.
//
// The Get and the batch commit are serialized per primary key by a sharded
// mutex. Without that, two concurrent writers for the same primary could
// both observe the same old snapshot id, each stage a delete for it, and
// then commit their own snap-index entries: the loser's snap-index would
// outlive its primary write and the next cascade delete of THAT snapshot
// would erase a primary that no longer belongs to it.
func (s *KVStorage) PutDiscoveryObservationRef(ref *happydns.DiscoveryObservationRef) error {
	primary := dscObsKey(ref.ProducerID, ref.Target, ref.Ref, ref.ConsumerID, ref.Key)

	mu := s.lockForObsRef(primary)
	mu.Lock()
	defer mu.Unlock()

	batch := s.db.NewBatch()

	// If a previous ref exists at the same primary key under a different
	// snapshot, drop its stale snap-index so a later cascade delete for that
	// earlier snapshot doesn't wipe the primary this call just wrote.
	old := &happydns.DiscoveryObservationRef{}
	if err := s.db.Get(primary, old); err == nil && !old.SnapshotID.Equals(ref.SnapshotID) {
		batch.Delete(dscObsSnapIndexKey(old.SnapshotID, primary))
	}

	if err := batch.Put(primary, ref); err != nil {
		return err
	}
	if err := batch.Put(dscObsSnapIndexKey(ref.SnapshotID, primary), primary); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) ListDiscoveryObservationRefs(producerID string, target happydns.CheckTarget, ref string) ([]*happydns.DiscoveryObservationRef, error) {
	prefix := fmt.Sprintf("dscobs|%s|%s|%s|", producerID, target.String(), ref)
	iter := s.db.Search(prefix)
	defer iter.Release()

	var out []*happydns.DiscoveryObservationRef
	for iter.Next() {
		r := &happydns.DiscoveryObservationRef{}
		if err := s.db.DecodeData(iter.Value(), r); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *KVStorage) ListAllDiscoveryObservationRefs() (happydns.Iterator[happydns.DiscoveryObservationRef], error) {
	iter := s.db.Search("dscobs|")
	return NewKVIterator[happydns.DiscoveryObservationRef](s.db, iter), nil
}

// RestoreDiscoveryObservationRef writes a ref at its canonical key and
// rebuilds its snapshot index. Used by the backup restore path.
func (s *KVStorage) RestoreDiscoveryObservationRef(ref *happydns.DiscoveryObservationRef) error {
	primary := dscObsKey(ref.ProducerID, ref.Target, ref.Ref, ref.ConsumerID, ref.Key)
	batch := s.db.NewBatch()
	if err := batch.Put(primary, ref); err != nil {
		return err
	}
	if err := batch.Put(dscObsSnapIndexKey(ref.SnapshotID, primary), primary); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) DeleteDiscoveryObservationRefsForSnapshot(snapshotID happydns.Identifier) error {
	prefix := fmt.Sprintf("dscobs-snap|%s|", snapshotID.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	batch := s.db.NewBatch()
	for iter.Next() {
		var primary string
		if err := s.db.DecodeData(iter.Value(), &primary); err != nil || primary == "" {
			// Fall back to extracting from the key suffix.
			primary = strings.TrimPrefix(iter.Key(), prefix)
		}
		batch.Delete(primary)
		batch.Delete(iter.Key())
	}
	return batch.Commit()
}

func (s *KVStorage) ClearDiscoveryObservationRefs() error {
	if err := s.clearByPrefix("dscobs-snap|"); err != nil {
		return err
	}
	return s.clearByPrefix("dscobs|")
}
