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

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
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

type notifier struct {
	mu   sync.Mutex
	sent []*notifPkg.NotificationPayload
	to   []*happydns.User
}

func (n *notifier) NotifyUser(user *happydns.User, p *notifPkg.NotificationPayload) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, p)
	n.to = append(n.to, user)
}

type fixture struct {
	db       storage.Storage
	registry *registry
	notifier *notifier
	service  *domainwatch.Service
	worker   *domainwatch.Worker
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
		notifier: &notifier{},
		now:      time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
	}
	f.alice = &happydns.User{Id: happydns.Identifier{0x0a}, Email: "alice@example.com"}
	f.bob = &happydns.User{Id: happydns.Identifier{0x0b}, Email: "bob@example.com"}
	for _, u := range []*happydns.User{f.alice, f.bob} {
		if err := db.CreateOrUpdateUser(u); err != nil {
			t.Fatal(err)
		}
	}

	f.service = domainwatch.NewService(db, db, f.registry.lookup, f.notifier, maxPerUser)
	f.service.SetClock(func() time.Time { return f.now }, func() time.Duration { return 0 })
	f.worker = domainwatch.NewWorker(f.service)
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
	if len(f.notifier.sent) != 0 {
		t.Errorf("the first lookup notified: %+v", f.notifier.sent)
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

	f.now = f.now.Add(8 * day)
	if n := f.worker.RunOnce(ctx); n != 1 {
		t.Fatalf("worker checked %d watches, want 1", n)
	}
	if got, _ := f.service.GetDomainWatch(f.alice, w.Id); got.Note != "++" {
		t.Errorf("note = %q after the worker, want the one edited during its lookup", got.Note)
	}
}

// The whole life of a domain, from expiring to available, as the worker sees
// it, with one notification per step.
func TestWorkerFollowsTheLifeOfADomain(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()

	f.registry.set("example.com", f.expiresIn(40*day), nil)
	if _, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}

	// Nothing due yet.
	if n := f.worker.RunOnce(ctx); n != 0 {
		t.Fatalf("worker checked %d watches, want 0", n)
	}

	steps := []struct {
		after time.Duration
		info  *domaininfo.DomainInfo
		err   error
		want  happydns.DomainWatchPhase
		title string
	}{
		// Due when it starts expiring, 10 days later.
		{10 * day, f.expiresIn(40 * day), nil, happydns.DomainWatchExpiring, "example.com expires in 30 days"},
		{41 * day, f.expiresIn(40 * day), nil, happydns.DomainWatchGrace, "example.com has expired"},
		{45 * day, f.expiresIn(40*day, "redemption period"), nil, happydns.DomainWatchRedemption, "example.com entered its redemption period"},
		{75 * day, f.expiresIn(40*day, "pending delete"), nil, happydns.DomainWatchPendingDelete, "example.com will be released within days"},
		{80 * day, nil, types.ErrDomainDoesNotExist, happydns.DomainWatchAvailable, "example.com is available!"},
		{81 * day, f.expiresIn(445 * day), nil, happydns.DomainWatchRegistered, "example.com was registered"},
	}

	start := f.now
	for i, st := range steps {
		f.now = start.Add(st.after)
		f.registry.set("example.com", st.info, st.err)

		if n := f.worker.RunOnce(ctx); n != 1 {
			t.Fatalf("step %d: worker checked %d watches, want 1", i, n)
		}
		list, _ := f.service.ListDomainWatches(f.alice)
		if len(list) != 1 || list[0].Phase != st.want {
			t.Fatalf("step %d: watches %+v, want phase %q", i, list, st.want)
		}
		if len(f.notifier.sent) != i+1 {
			t.Fatalf("step %d: %d notifications, want %d", i, len(f.notifier.sent), i+1)
		}
		last := f.notifier.sent[i]
		if last.Event == nil || last.Event.Title != st.title || last.CheckerID != domainwatch.EventKind || last.DomainName != "example.com" {
			t.Errorf("step %d: notification %+v", i, last.Event)
		}
		if !f.notifier.to[i].Id.Equals(f.alice.Id) {
			t.Errorf("step %d: notified %v, want alice", i, f.notifier.to[i].Id)
		}

		// Checked again right away: not due anymore.
		if n := f.worker.RunOnce(ctx); n != 0 {
			t.Fatalf("step %d: worker checked %d watches again, want 0", i, n)
		}
	}

	list, _ := f.service.ListDomainWatches(f.alice)
	if len(list[0].History) != len(steps) {
		t.Errorf("history holds %d events, want %d", len(list[0].History), len(steps))
	}
}

func TestWorkerSharesLookupsAndSurvivesFailures(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()

	f.registry.set("example.com", f.expiresIn(200*day), nil)
	for _, u := range []*happydns.User{f.alice, f.bob} {
		if _, err := f.service.CreateDomainWatch(ctx, u, happydns.DomainWatchInput{Domain: "example.com"}); err != nil {
			t.Fatal(err)
		}
	}

	f.now = f.now.Add(8 * day)
	f.registry.set("example.com", nil, errors.New("RDAP unreachable"))
	before := f.registry.count("example.com")
	if n := f.worker.RunOnce(ctx); n != 2 {
		t.Fatalf("worker checked %d watches, want 2", n)
	}
	if n := f.registry.count("example.com") - before; n != 1 {
		t.Errorf("%d lookups for 2 watches of the same domain, want 1", n)
	}

	for _, u := range []*happydns.User{f.alice, f.bob} {
		list, _ := f.service.ListDomainWatches(u)
		if list[0].Phase != happydns.DomainWatchRegistered || list[0].LastError == "" {
			t.Errorf("a failed lookup changed the watch: %+v", list[0])
		}
	}
	if len(f.notifier.sent) != 0 {
		t.Errorf("a failed lookup notified: %+v", f.notifier.sent)
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

func TestWorkerDoesNotNotifyAUserGone(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()

	f.registry.set("example.com", f.expiresIn(200*day), nil)
	if _, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := f.db.DeleteUser(f.alice.Id); err != nil {
		t.Fatal(err)
	}

	f.now = f.now.Add(8 * day)
	f.registry.set("example.com", nil, types.ErrDomainDoesNotExist)
	if n := f.worker.RunOnce(ctx); n != 1 {
		t.Fatalf("worker checked %d watches, want 1", n)
	}
	if len(f.notifier.sent) != 0 {
		t.Errorf("notified the owner gone: %+v", f.notifier.sent)
	}
}

func TestWorkerSkipsAWatchRemovedDuringTheLookup(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()

	f.registry.set("example.com", f.expiresIn(200*day), nil)
	w, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	f.now = f.now.Add(8 * day)
	f.registry.onLookup = func(string) {
		if err := f.service.DeleteDomainWatch(f.alice, w.Id); err != nil {
			t.Errorf("delete during the lookup: %v", err)
		}
	}
	if n := f.worker.RunOnce(ctx); n != 0 {
		t.Errorf("worker checked %d watches, want 0: the only one was removed", n)
	}
	if list, _ := f.service.ListDomainWatches(f.alice); len(list) != 0 {
		t.Errorf("the removed watch came back: %+v", list)
	}
}

func TestWorkerLeavesDueTheWatchesStoppedDuringTheLookup(t *testing.T) {
	f := newFixture(t, 0)

	f.registry.set("example.com", f.expiresIn(200*day), nil)
	if _, err := f.service.CreateDomainWatch(context.Background(), f.alice, happydns.DomainWatchInput{Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(8 * day)

	ctx, cancel := context.WithCancel(context.Background())
	f.registry.onLookup = func(string) { cancel() }
	if n := f.worker.RunOnce(ctx); n != 0 {
		t.Errorf("worker checked %d watches stopped during their lookup, want 0", n)
	}
	list, _ := f.service.ListDomainWatches(f.alice)
	if list[0].LastError != "" {
		t.Errorf("the lookup cut short was recorded as a failure: %q", list[0].LastError)
	}

	f.registry.onLookup = nil
	if n := f.worker.RunOnce(context.Background()); n != 1 {
		t.Errorf("worker checked %d watches, want 1: the watch is still due", n)
	}
}

func TestWorkerLeavesDueTheWatchesNotLookedUp(t *testing.T) {
	f := newFixture(t, 0)

	f.registry.set("example.com", f.expiresIn(200*day), nil)
	if _, err := f.service.CreateDomainWatch(context.Background(), f.alice, happydns.DomainWatchInput{Domain: "example.com"}); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(8 * day)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if n := f.worker.RunOnce(ctx); n != 0 {
		t.Errorf("worker checked %d watches once stopped, want 0", n)
	}

	if n := f.worker.RunOnce(context.Background()); n != 1 {
		t.Errorf("worker checked %d watches, want 1: the watch is still due", n)
	}
}

// Checked on demand, the domain changing phase notifies the user asking.
func TestCheckNotifiesThePhaseChange(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()
	f.registry.set("example.com", f.expiresIn(200*day), nil)
	w, err := f.service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	f.now = f.now.Add(time.Hour)
	f.registry.set("example.com", nil, types.ErrDomainDoesNotExist)
	if _, err := f.service.CheckDomainWatch(ctx, f.alice, w.Id); err != nil {
		t.Fatal(err)
	}
	if len(f.notifier.sent) != 1 || f.notifier.sent[0].Event.Title != "example.com is available!" {
		t.Fatalf("notifications = %+v, want the domain available", f.notifier.sent)
	}
	if f.notifier.to[0] != f.alice {
		t.Errorf("notified %+v, want the user who checked", f.notifier.to[0])
	}
}

func TestServiceWithoutNotifier(t *testing.T) {
	f := newFixture(t, 0)
	ctx := context.Background()
	service := domainwatch.NewService(f.db, f.db, f.registry.lookup, nil, 0)
	service.SetClock(func() time.Time { return f.now }, func() time.Duration { return 0 })

	f.registry.set("example.com", f.expiresIn(200*day), nil)
	w, err := service.CreateDomainWatch(ctx, f.alice, happydns.DomainWatchInput{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	f.now = f.now.Add(time.Hour)
	f.registry.set("example.com", nil, types.ErrDomainDoesNotExist)
	if got, err := service.CheckDomainWatch(ctx, f.alice, w.Id); err != nil || got.Phase != happydns.DomainWatchAvailable {
		t.Errorf("CheckDomainWatch = %+v, %v; want the domain available", got, err)
	}
}
