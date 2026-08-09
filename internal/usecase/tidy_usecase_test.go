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

package usecase_test

import (
	"encoding/json"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

func TestTidyObservationCache_RemovesStaleEntries(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	// Create a snapshot and a cache entry pointing to it.
	snap := &happydns.ObservationSnapshot{
		Target:      target,
		CollectedAt: time.Now(),
		Data: map[happydns.ObservationKey]json.RawMessage{
			"obs_a": json.RawMessage(`{"x":1}`),
		},
	}
	if err := store.CreateSnapshot(snap); err != nil {
		t.Fatalf("CreateSnapshot() error: %v", err)
	}

	validEntry := &happydns.ObservationCacheEntry{
		SnapshotID:  snap.Id,
		CollectedAt: snap.CollectedAt,
	}
	if err := store.PutCachedObservation(target, "obs_a", validEntry); err != nil {
		t.Fatalf("PutCachedObservation() error: %v", err)
	}

	// Create a stale cache entry pointing to a non-existent snapshot.
	staleSnapID, _ := happydns.NewRandomIdentifier()
	staleEntry := &happydns.ObservationCacheEntry{
		SnapshotID:  staleSnapID,
		CollectedAt: time.Now().Add(-time.Hour),
	}
	if err := store.PutCachedObservation(target, "obs_stale", staleEntry); err != nil {
		t.Fatalf("PutCachedObservation() error: %v", err)
	}

	// Verify both entries exist before tidy.
	if _, err := store.GetCachedObservation(target, "obs_a"); err != nil {
		t.Fatalf("expected valid cache entry to exist: %v", err)
	}
	if _, err := store.GetCachedObservation(target, "obs_stale"); err != nil {
		t.Fatalf("expected stale cache entry to exist: %v", err)
	}

	// Run tidy.
	tu := usecase.NewTidyUpUsecase(store)
	if err := tu.TidyObservationCache(true); err != nil {
		t.Fatalf("TidyObservationCache() error: %v", err)
	}

	// Valid entry should still exist.
	if _, err := store.GetCachedObservation(target, "obs_a"); err != nil {
		t.Errorf("expected valid cache entry to survive tidy: %v", err)
	}

	// Stale entry should be removed.
	if _, err := store.GetCachedObservation(target, "obs_stale"); err == nil {
		t.Error("expected stale cache entry to be removed by tidy")
	}
}

func TestTidyDomains_RemovesOrphanShares(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	owner := &happydns.User{Id: happydns.Identifier("owner"), Email: "owner@example.com"}
	grantee := &happydns.User{Id: happydns.Identifier("grantee"), Email: "grantee@example.com"}
	for _, u := range []*happydns.User{owner, grantee} {
		if err := store.CreateOrUpdateUser(u); err != nil {
			t.Fatalf("CreateOrUpdateUser() error: %v", err)
		}
	}

	// The domain needs a live provider, otherwise tidy deletes it as an orphan
	// and takes its shares along with it.
	prvd := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Owner: owner.Id, Type: "dummy"}}
	if err := store.CreateProvider(prvd); err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}

	dom := &happydns.Domain{Owner: owner.Id, ProviderId: prvd.Id, DomainName: "example.com."}
	if err := store.CreateDomain(dom); err != nil {
		t.Fatalf("CreateDomain() error: %v", err)
	}

	missingDomain, _ := happydns.NewRandomIdentifier()
	missingGrantee, _ := happydns.NewRandomIdentifier()

	// A valid share, plus two orphans (missing domain, missing grantee).
	if err := store.AddDomainShare(dom.Id, grantee.Id); err != nil {
		t.Fatalf("AddDomainShare(valid) error: %v", err)
	}
	if err := store.AddDomainShare(missingDomain, grantee.Id); err != nil {
		t.Fatalf("AddDomainShare(orphan domain) error: %v", err)
	}
	if err := store.AddDomainShare(dom.Id, missingGrantee); err != nil {
		t.Fatalf("AddDomainShare(orphan grantee) error: %v", err)
	}

	tu := usecase.NewTidyUpUsecase(store)
	if err := tu.TidyDomains(true); err != nil {
		t.Fatalf("TidyDomains() error: %v", err)
	}

	// The valid share survives.
	if shared, _ := store.IsDomainSharedWith(dom.Id, grantee.Id); !shared {
		t.Error("expected valid domain share to survive tidy")
	}
	// Both orphans are gone, from both index views.
	if shared, _ := store.IsDomainSharedWith(missingDomain, grantee.Id); shared {
		t.Error("expected orphan share (missing domain) to be removed")
	}
	if shared, _ := store.IsDomainSharedWith(dom.Id, missingGrantee); shared {
		t.Error("expected orphan share (missing grantee) to be removed")
	}
	if ids, _ := store.ListSharedDomainIDs(missingGrantee); len(ids) != 0 {
		t.Errorf("expected grant index cleaned for missing grantee, got %d entries", len(ids))
	}
}

func TestTidyProviders_RemovesOrphanShares(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	owner := &happydns.User{Id: happydns.Identifier("owner"), Email: "owner@example.com"}
	grantee := &happydns.User{Id: happydns.Identifier("grantee"), Email: "grantee@example.com"}
	for _, u := range []*happydns.User{owner, grantee} {
		if err := store.CreateOrUpdateUser(u); err != nil {
			t.Fatalf("CreateOrUpdateUser() error: %v", err)
		}
	}

	prvd := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Owner: owner.Id, Comment: "P"}}
	if err := store.CreateProvider(prvd); err != nil {
		t.Fatalf("CreateProvider() error: %v", err)
	}

	missingProvider, _ := happydns.NewRandomIdentifier()
	missingGrantee, _ := happydns.NewRandomIdentifier()

	if err := store.AddProviderShare(prvd.Id, grantee.Id); err != nil {
		t.Fatalf("AddProviderShare(valid) error: %v", err)
	}
	if err := store.AddProviderShare(missingProvider, grantee.Id); err != nil {
		t.Fatalf("AddProviderShare(orphan provider) error: %v", err)
	}
	if err := store.AddProviderShare(prvd.Id, missingGrantee); err != nil {
		t.Fatalf("AddProviderShare(orphan grantee) error: %v", err)
	}

	tu := usecase.NewTidyUpUsecase(store)
	if err := tu.TidyProviders(true); err != nil {
		t.Fatalf("TidyProviders() error: %v", err)
	}

	if shared, _ := store.IsProviderSharedWith(prvd.Id, grantee.Id); !shared {
		t.Error("expected valid provider share to survive tidy")
	}
	if shared, _ := store.IsProviderSharedWith(missingProvider, grantee.Id); shared {
		t.Error("expected orphan share (missing provider) to be removed")
	}
	if shared, _ := store.IsProviderSharedWith(prvd.Id, missingGrantee); shared {
		t.Error("expected orphan share (missing grantee) to be removed")
	}
}

func TestTidyObservationCache_EmptyCache(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	tu := usecase.NewTidyUpUsecase(store)
	if err := tu.TidyObservationCache(true); err != nil {
		t.Fatalf("TidyObservationCache() on empty cache error: %v", err)
	}
}
