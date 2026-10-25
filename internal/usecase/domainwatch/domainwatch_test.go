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

package domainwatch_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/domainwatch"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
	"git.happydns.org/happyDomain/pkg/domaininfo/types"
)

// registry is a fake registry: it answers the lookups from its domains, and
// counts them.
type registry struct {
	mu      sync.Mutex
	domains map[string]*domaininfo.DomainInfo
	errs    map[string]error
	lookups map[string]int

	// onLookup, when set, runs at each lookup, outside of the lock.
	onLookup func(name string)
}

func newRegistry() *registry {
	return &registry{domains: map[string]*domaininfo.DomainInfo{}, errs: map[string]error{}, lookups: map[string]int{}}
}

func (r *registry) set(name string, info *domaininfo.DomainInfo, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.domains[name], r.errs[name] = info, err
}

func (r *registry) lookup(ctx context.Context, name string) (*domaininfo.DomainInfo, error) {
	if r.onLookup != nil {
		r.onLookup(name)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.lookups[name]++
	if err := r.errs[name]; err != nil {
		return nil, err
	}
	if info, ok := r.domains[name]; ok {
		return info, nil
	}
	return nil, types.ErrDomainDoesNotExist
}

func (r *registry) count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lookups[name]
}

type fixture struct {
	db       storage.Storage
	registry *registry
	service  *domainwatch.Service
	now      time.Time
	alice    *happydns.User
	bob      *happydns.User
}

func newFixture(t *testing.T, maxPerUser int) *fixture {
	t.Helper()

	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	f := &fixture{
		db:       db,
		registry: newRegistry(),
		now:      time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
	}
	f.alice = &happydns.User{Id: happydns.Identifier{0x0a}, Email: "alice@example.com"}
	f.bob = &happydns.User{Id: happydns.Identifier{0x0b}, Email: "bob@example.com"}
	for _, u := range []*happydns.User{f.alice, f.bob} {
		if err := db.CreateOrUpdateUser(u); err != nil {
			t.Fatal(err)
		}
	}

	f.service = domainwatch.NewService(db, f.registry.lookup, maxPerUser)
	f.service.SetClock(func() time.Time { return f.now }, func() time.Duration { return 0 })
	return f
}

func (f *fixture) expiresIn(d time.Duration, status ...string) *domaininfo.DomainInfo {
	t := f.now.Add(d)
	return &domaininfo.DomainInfo{ExpirationDate: &t, Registrar: "Acme", Status: status}
}

const day = 24 * time.Hour

func TestCreateLooksTheDomainUp(t *testing.T) {
	f := newFixture(t, 0)
	f.registry.set("example.com", f.expiresIn(200*day), nil)

	w, err := f.service.CreateDomainWatch(context.Background(), f.alice, happydns.DomainWatchInput{Domain: "www.Example.com", Note: "for my shop"})
	if err != nil {
		t.Fatal(err)
	}
	if w.DomainName != "example.com" || w.Note != "for my shop" || w.Phase != happydns.DomainWatchRegistered || w.Registrar != "Acme" {
		t.Errorf("unexpected watch: %+v", w)
	}
	if w.LastCheck == nil || !w.NextCheck.Equal(f.now.Add(7*day)) {
		t.Errorf("last check %v, next check %v", w.LastCheck, w.NextCheck)
	}
	// Stored as returned.
	stored, err := f.service.GetDomainWatch(f.alice, w.Id)
	if err != nil || stored.Phase != happydns.DomainWatchRegistered {
		t.Errorf("stored watch %+v, %v", stored, err)
	}
}

func TestCreateKeepsTheWatchWhenTheLookupFails(t *testing.T) {
	f := newFixture(t, 0)
	f.registry.set("example.com", nil, errors.New("RDAP unreachable"))

	w, err := f.service.CreateDomainWatch(context.Background(), f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if w.Phase != happydns.DomainWatchUnknown || w.LastError == "" || !w.NextCheck.Equal(f.now.Add(time.Hour)) {
		t.Errorf("unexpected watch: %+v", w)
	}
}

func TestCreateKeepsTheWatchWhenTheUserLeaves(t *testing.T) {
	f := newFixture(t, 0)
	f.registry.set("example.com", f.expiresIn(200*day), nil)

	ctx, cancel := context.WithCancel(context.Background())
	f.registry.onLookup = func(string) { cancel() }
	w, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if w.Phase != happydns.DomainWatchUnknown || w.LastError != "" || !w.NextCheck.Equal(f.now) {
		t.Errorf("unexpected watch: %+v, want one still due, with no error", w)
	}
	if list, _ := f.service.ListDomainWatches(f.alice); len(list) != 1 {
		t.Errorf("listed %d watches, want the one added", len(list))
	}
}

func TestCreateRefusesDuplicatesAndTooMany(t *testing.T) {
	f := newFixture(t, 2)
	ctx := context.Background()

	for _, name := range []string{"a.example", "b.example"} {
		if _, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: name}); err != nil {
			t.Fatal(err)
		}
	}

	var ce happydns.CustomError
	if _, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "https://www.a.example/"}); !errors.As(err, &ce) || ce.Status != http.StatusConflict {
		t.Errorf("duplicate: got %v, want a conflict", err)
	}
	if _, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "c.example"}); !errors.As(err, &ce) || ce.Status != http.StatusForbidden {
		t.Errorf("over the limit: got %v, want forbidden", err)
	}

	// The limit and the duplicates are per user.
	if _, err := f.service.CreateDomainWatch(ctx, f.bob, happydns.DomainWatchInput{Domain: "a.example"}); err != nil {
		t.Errorf("another user watching the same domain: %v", err)
	}

	var ve happydns.ValidationError
	if _, err := f.service.CreateDomainWatch(ctx, f.bob, happydns.DomainWatchInput{Domain: "co.uk"}); !errors.As(err, &ve) {
		t.Errorf("public suffix: got %v, want a validation error", err)
	}
}

func TestWatchesArePrivate(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()

	w, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.service.GetDomainWatch(f.bob, w.Id); !errors.Is(err, happydns.ErrDomainWatchNotFound) {
		t.Errorf("get by another user: %v", err)
	}
	if _, err := f.service.UpdateDomainWatch(f.bob, w.Id, happydns.DomainWatchInput{Note: "mine"}); !errors.Is(err, happydns.ErrDomainWatchNotFound) {
		t.Errorf("update by another user: %v", err)
	}
	if err := f.service.DeleteDomainWatch(f.bob, w.Id); !errors.Is(err, happydns.ErrDomainWatchNotFound) {
		t.Errorf("delete by another user: %v", err)
	}
	if _, err := f.service.CheckDomainWatch(ctx, f.bob, w.Id); !errors.Is(err, happydns.ErrDomainWatchNotFound) {
		t.Errorf("check by another user: %v", err)
	}
	if list, _ := f.service.ListDomainWatches(f.bob); len(list) != 0 {
		t.Errorf("listed for another user: %+v", list)
	}

	updated, err := f.service.UpdateDomainWatch(f.alice, w.Id, happydns.DomainWatchInput{Domain: "ignored.example", Note: "really want it"})
	if err != nil || updated.Note != "really want it" || updated.DomainName != "example.com" {
		t.Errorf("update: %+v, %v", updated, err)
	}
	if err := f.service.DeleteDomainWatch(f.alice, w.Id); err != nil {
		t.Errorf("delete: %v", err)
	}
	if list, _ := f.service.ListDomainWatches(f.alice); len(list) != 0 {
		t.Errorf("listed after deletion: %+v", list)
	}
}

func TestCheckIsThrottled(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()
	f.registry.set("example.com", f.expiresIn(200*day), nil)

	w, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	f.now = f.now.Add(10 * time.Second)
	if _, err := f.service.CheckDomainWatch(ctx, f.alice, w.Id); err != nil {
		t.Fatal(err)
	}
	if n := f.registry.count("example.com"); n != 1 {
		t.Errorf("%d lookups, want 1: checked again too soon", n)
	}

	f.now = f.now.Add(time.Minute)
	if _, err := f.service.CheckDomainWatch(ctx, f.alice, w.Id); err != nil {
		t.Fatal(err)
	}
	if n := f.registry.count("example.com"); n != 2 {
		t.Errorf("%d lookups, want 2", n)
	}
}

// A lookup cut short by the user leaving is not a failure of the lookup.
func TestCheckDoesNotRecordALookupCutShort(t *testing.T) {
	f := newFixture(t, 0)
	f.registry.set("example.com", f.expiresIn(200*day), nil)
	w, err := f.service.CreateDomainWatch(context.Background(), f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	f.now = f.now.Add(time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	f.registry.onLookup = func(string) { cancel() }
	if _, err := f.service.CheckDomainWatch(ctx, f.alice, w.Id); !errors.Is(err, context.Canceled) {
		t.Errorf("CheckDomainWatch = %v, want context.Canceled", err)
	}

	got, _ := f.service.GetDomainWatch(f.alice, w.Id)
	if got.LastError != "" || !got.LastCheck.Equal(*w.LastCheck) || !got.NextCheck.Equal(w.NextCheck) {
		t.Errorf("the lookup cut short was recorded: %+v", got)
	}
}

// A note edited while the domain is looked up is not overwritten by the
// result of the lookup.
func TestLookupKeepsTheNoteEditedMeanwhile(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()
	f.registry.set("example.com", f.expiresIn(200*day), nil)
	w, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com", Note: "first"})
	if err != nil {
		t.Fatal(err)
	}

	note := ""
	f.registry.onLookup = func(string) {
		note += "+"
		if _, err := f.service.UpdateDomainWatch(f.alice, w.Id, happydns.DomainWatchInput{Note: note}); err != nil {
			t.Errorf("update during the lookup: %v", err)
		}
	}

	f.now = f.now.Add(time.Hour)
	if got, err := f.service.CheckDomainWatch(ctx, f.alice, w.Id); err != nil || got.Note != "+" {
		t.Errorf("CheckDomainWatch = %+v, %v; want the note edited during the lookup", got, err)
	}
}

func TestNoteIsBounded(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()
	long := strings.Repeat("a", 1001)

	var ve happydns.ValidationError
	if _, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com", Note: long}); !errors.As(err, &ve) {
		t.Errorf("create with a long note: got %v, want a validation error", err)
	}

	w, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com", Note: strings.Repeat("a", 1000)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.UpdateDomainWatch(f.alice, w.Id, happydns.DomainWatchInput{Note: long}); !errors.As(err, &ve) {
		t.Errorf("update with a long note: got %v, want a validation error", err)
	}
}
