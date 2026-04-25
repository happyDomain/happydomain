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

package database

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// fakeKV is a minimal in-memory KV used only by these tests — inmemory package
// imports kvtpl, so we cannot import it here.
type fakeKV struct {
	data map[string]json.RawMessage
}

func newFakeKV() *fakeKV { return &fakeKV{data: map[string]json.RawMessage{}} }

func (f *fakeKV) Close() error { return nil }
func (f *fakeKV) DecodeData(i any, v any) error {
	b, ok := i.(json.RawMessage)
	if !ok {
		return fmt.Errorf("not a RawMessage (%T)", i)
	}
	return json.Unmarshal(b, v)
}
func (f *fakeKV) Has(key string) (bool, error) {
	_, ok := f.data[key]
	return ok, nil
}
func (f *fakeKV) Get(key string, v any) error {
	raw, ok := f.data[key]
	if !ok {
		return happydns.ErrNotFound
	}
	return json.Unmarshal(raw, v)
}
func (f *fakeKV) Put(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f.data[key] = b
	return nil
}
func (f *fakeKV) FindIdentifierKey(prefix string) (string, happydns.Identifier, error) {
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		return "", nil, err
	}
	return prefix + id.String(), id, nil
}
func (f *fakeKV) Delete(key string) error {
	delete(f.data, key)
	return nil
}
func (f *fakeKV) Search(prefix string) storage.Iterator {
	keys := make([]string, 0)
	for k := range f.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return &fakeIter{data: f.data, keys: keys, idx: -1}
}

type fakeIter struct {
	data map[string]json.RawMessage
	keys []string
	idx  int
}

func (i *fakeIter) Release()      {}
func (i *fakeIter) Next() bool    { i.idx++; return i.idx < len(i.keys) }
func (i *fakeIter) Valid() bool   { return i.idx >= 0 && i.idx < len(i.keys) }
func (i *fakeIter) Key() string   { return i.keys[i.idx] }
func (i *fakeIter) Value() any    { return i.data[i.keys[i.idx]] }
func (i *fakeIter) Err() error    { return nil }

func newDiscoveryTestStore() *KVStorage {
	return &KVStorage{db: newFakeKV()}
}

func TestReplaceDiscoveryEntriesRoundTrip(t *testing.T) {
	s := newDiscoveryTestStore()
	target := happydns.CheckTarget{DomainId: "domA"}

	entries := []happydns.DiscoveryEntry{
		{Type: "tls.endpoint.v1", Ref: "host:443", Payload: json.RawMessage(`{"host":"a"}`)},
		{Type: "tls.endpoint.v1", Ref: "host:465", Payload: json.RawMessage(`{"host":"b"}`)},
	}
	if err := s.ReplaceDiscoveryEntries("checker-srv", target, entries); err != nil {
		t.Fatalf("ReplaceDiscoveryEntries: %v", err)
	}

	byProducer, err := s.ListDiscoveryEntriesByProducer("checker-srv", target)
	if err != nil {
		t.Fatalf("ListDiscoveryEntriesByProducer: %v", err)
	}
	if len(byProducer) != 2 {
		t.Fatalf("want 2 entries, got %d", len(byProducer))
	}

	byTarget, err := s.ListDiscoveryEntriesByTarget(target)
	if err != nil {
		t.Fatalf("ListDiscoveryEntriesByTarget: %v", err)
	}
	if len(byTarget) != 2 {
		t.Fatalf("want 2 entries via target index, got %d", len(byTarget))
	}
}

func TestReplaceDiscoveryEntriesReplacesAtomically(t *testing.T) {
	s := newDiscoveryTestStore()
	target := happydns.CheckTarget{DomainId: "domA"}

	if err := s.ReplaceDiscoveryEntries("p", target, []happydns.DiscoveryEntry{
		{Type: "t", Ref: "r1"},
		{Type: "t", Ref: "r2"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := s.ReplaceDiscoveryEntries("p", target, []happydns.DiscoveryEntry{
		{Type: "t", Ref: "r3"},
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.ListDiscoveryEntriesByProducer("p", target)
	if len(got) != 1 || got[0].Ref != "r3" {
		t.Fatalf("replace did not clear previous set: %#v", got)
	}
	viaTarget, _ := s.ListDiscoveryEntriesByTarget(target)
	if len(viaTarget) != 1 || viaTarget[0].Ref != "r3" {
		t.Fatalf("target index diverged from primary: %#v", viaTarget)
	}
}

func TestListDiscoveryEntriesByTargetAggregatesProducers(t *testing.T) {
	s := newDiscoveryTestStore()
	target := happydns.CheckTarget{DomainId: "domA"}

	if err := s.ReplaceDiscoveryEntries("p1", target, []happydns.DiscoveryEntry{{Type: "t", Ref: "a"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceDiscoveryEntries("p2", target, []happydns.DiscoveryEntry{{Type: "t", Ref: "b"}}); err != nil {
		t.Fatal(err)
	}

	got, _ := s.ListDiscoveryEntriesByTarget(target)
	if len(got) != 2 {
		t.Fatalf("want 2 entries from two producers, got %d", len(got))
	}
}

func TestListDiscoveryEntriesByTargetWidensToNarrowerScopes(t *testing.T) {
	s := newDiscoveryTestStore()

	// Service-scoped publisher (e.g. checker-dane on a TLSA service).
	svc := happydns.CheckTarget{UserId: "u1", DomainId: "d1", ServiceId: "svc1"}
	if err := s.ReplaceDiscoveryEntries("checker-dane", svc, []happydns.DiscoveryEntry{
		{Type: "tls.endpoint.v1", Ref: "host:443"},
	}); err != nil {
		t.Fatal(err)
	}

	// Another service under the same domain.
	svc2 := happydns.CheckTarget{UserId: "u1", DomainId: "d1", ServiceId: "svc2"}
	if err := s.ReplaceDiscoveryEntries("checker-smtp", svc2, []happydns.DiscoveryEntry{
		{Type: "tls.endpoint.v1", Ref: "mx:25"},
	}); err != nil {
		t.Fatal(err)
	}

	// A service under a different domain — must not leak.
	other := happydns.CheckTarget{UserId: "u1", DomainId: "d2", ServiceId: "svcX"}
	if err := s.ReplaceDiscoveryEntries("checker-srv", other, []happydns.DiscoveryEntry{
		{Type: "tls.endpoint.v1", Ref: "x:443"},
	}); err != nil {
		t.Fatal(err)
	}

	// Domain-scoped consumer (e.g. checker-tls) sees both services under d1
	// but not the one under d2.
	dom := happydns.CheckTarget{UserId: "u1", DomainId: "d1"}
	got, err := s.ListDiscoveryEntriesByTarget(dom)
	if err != nil {
		t.Fatalf("ListDiscoveryEntriesByTarget(domain): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("domain-scoped lookup: want 2 entries from services under d1, got %d: %#v", len(got), got)
	}
	for _, e := range got {
		if e.Target.DomainId != "d1" {
			t.Errorf("leaked entry from another domain: %#v", e)
		}
	}

	// Service-scoped lookup stays exact.
	exact, err := s.ListDiscoveryEntriesByTarget(svc)
	if err != nil {
		t.Fatalf("ListDiscoveryEntriesByTarget(service): %v", err)
	}
	if len(exact) != 1 || exact[0].Ref != "host:443" {
		t.Fatalf("service-scoped lookup widened unexpectedly: %#v", exact)
	}

	// User-scoped lookup widens to every domain/service under that user.
	usr := happydns.CheckTarget{UserId: "u1"}
	all, err := s.ListDiscoveryEntriesByTarget(usr)
	if err != nil {
		t.Fatalf("ListDiscoveryEntriesByTarget(user): %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("user-scoped lookup: want 3 entries across u1, got %d", len(all))
	}
}

func TestDiscoveryObservationRefCascadeOnSnapshotDelete(t *testing.T) {
	s := newDiscoveryTestStore()
	target := happydns.CheckTarget{DomainId: "domA"}
	snap := happydns.Identifier{1, 2, 3}
	other := happydns.Identifier{4, 5, 6}

	refs := []*happydns.DiscoveryObservationRef{
		{ProducerID: "p", Target: target, Ref: "r1", ConsumerID: "c", Key: "k", SnapshotID: snap, CollectedAt: time.Now()},
		{ProducerID: "p", Target: target, Ref: "r2", ConsumerID: "c", Key: "k", SnapshotID: snap, CollectedAt: time.Now()},
		{ProducerID: "p", Target: target, Ref: "r1", ConsumerID: "c", Key: "k", SnapshotID: other, CollectedAt: time.Now()},
	}
	for _, r := range refs {
		if err := s.PutDiscoveryObservationRef(r); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.DeleteDiscoveryObservationRefsForSnapshot(snap); err != nil {
		t.Fatalf("cascade delete: %v", err)
	}

	remaining, _ := s.ListDiscoveryObservationRefs("p", target, "r1")
	if len(remaining) != 1 || !remaining[0].SnapshotID.Equals(other) {
		t.Fatalf("cascade delete left stale data: %#v", remaining)
	}
	remaining2, _ := s.ListDiscoveryObservationRefs("p", target, "r2")
	if len(remaining2) != 0 {
		t.Fatalf("cascade delete missed snapshot refs: %#v", remaining2)
	}
}

func TestPutDiscoveryObservationRefUpsert(t *testing.T) {
	s := newDiscoveryTestStore()
	target := happydns.CheckTarget{DomainId: "domA"}

	first := &happydns.DiscoveryObservationRef{
		ProducerID: "p", Target: target, Ref: "r", ConsumerID: "c", Key: "k",
		SnapshotID: happydns.Identifier{1}, CollectedAt: time.Now().Add(-time.Hour),
	}
	second := &happydns.DiscoveryObservationRef{
		ProducerID: "p", Target: target, Ref: "r", ConsumerID: "c", Key: "k",
		SnapshotID: happydns.Identifier{2}, CollectedAt: time.Now(),
	}
	if err := s.PutDiscoveryObservationRef(first); err != nil {
		t.Fatal(err)
	}
	if err := s.PutDiscoveryObservationRef(second); err != nil {
		t.Fatal(err)
	}

	got, _ := s.ListDiscoveryObservationRefs("p", target, "r")
	if len(got) != 1 {
		t.Fatalf("upsert should keep a single ref per tuple, got %d", len(got))
	}
	if !got[0].SnapshotID.Equals(second.SnapshotID) {
		t.Fatalf("latest ref should win, got SnapshotID=%v", got[0].SnapshotID)
	}
}
