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

package domainwatch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

// EventKind identifies the notifications about watched domains.
const EventKind = "domain_watch"

const (
	// lookupTimeout bounds a lookup made while the user waits.
	lookupTimeout = 20 * time.Second
	// checkThrottle is how long after a lookup asking for another one
	// returns the last result instead.
	checkThrottle = time.Minute
	// maxJitter spreads the lookups of the domains over time.
	maxJitter = 30 * time.Minute
	// maxNoteLength bounds the note of a watch.
	maxNoteLength = 1000
)

// UserGetter finds the owner of a watch, to notify them.
type UserGetter interface {
	GetUser(id happydns.Identifier) (*happydns.User, error)
}

// Notifier tells a user about an event of one of their watches.
type Notifier interface {
	NotifyUser(user *happydns.User, payload *notifPkg.NotificationPayload)
}

// Service manages the domains users watch, and looks them up when due.
type Service struct {
	store      DomainWatchStorage
	users      UserGetter
	lookup     domaininfo.Getter
	notifier   Notifier
	maxPerUser int

	// mu serializes the read-modify-write of watches, so that a lookup
	// does not overwrite a note being edited, nor the reverse. Lookups
	// themselves run outside of it.
	mu sync.Mutex

	// Overridable for tests.
	nowFn    func() time.Time
	jitterFn func() time.Duration
}

var _ happydns.DomainWatchUsecase = (*Service)(nil)

// NewService builds the domain watch service. lookup retrieves the
// registration information of a domain; notifier, which may be nil, tells
// users about the phase changes of their domains; maxPerUser caps the number
// of watches of a user, 0 meaning unlimited.
func NewService(store DomainWatchStorage, users UserGetter, lookup domaininfo.Getter, notifier Notifier, maxPerUser int) *Service {
	return &Service{
		store:      store,
		users:      users,
		lookup:     lookup,
		notifier:   notifier,
		maxPerUser: maxPerUser,
		nowFn:      time.Now,
		jitterFn:   func() time.Duration { return rand.N(maxJitter) },
	}
}

// NormalizeName turns what a user typed, a domain, a subdomain or a URL, into
// the registrable domain to watch, in ASCII and in Unicode.
func NormalizeName(input string) (ascii, display string, err error) {
	name := strings.TrimSpace(input)
	if strings.Contains(name, "://") {
		if u, err := url.Parse(name); err == nil && u.Hostname() != "" {
			name = u.Hostname()
		}
	}

	name, err = idna.Lookup.ToASCII(strings.TrimSuffix(name, "."))
	if err != nil {
		return "", "", happydns.ValidationError{Msg: fmt.Sprintf("%q is not a valid domain name", input)}
	}

	fqdn, err := happydns.NormalizeDomainName(name)
	if err != nil {
		return "", "", happydns.ValidationError{Msg: fmt.Sprintf("%q is not a valid domain name: %s", input, err.Error())}
	}

	ascii, err = publicsuffix.EffectiveTLDPlusOne(strings.TrimSuffix(fqdn, "."))
	if err != nil {
		return "", "", happydns.ValidationError{Msg: fmt.Sprintf("%q is a public suffix: no one can register it", input)}
	}

	display, err = idna.Display.ToUnicode(ascii)
	if err != nil {
		display = ascii
	}
	return ascii, display, nil
}

func (s *Service) ListDomainWatches(user *happydns.User) ([]*happydns.DomainWatch, error) {
	watches, err := s.store.ListDomainWatchesByUser(user.Id)
	if err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to list the watched domains: %w", err), UserMessage: "Unable to list the watched domains."}
	}
	if watches == nil {
		watches = []*happydns.DomainWatch{}
	}
	return watches, nil
}

func (s *Service) GetDomainWatch(user *happydns.User, id happydns.Identifier) (*happydns.DomainWatch, error) {
	w, err := s.store.GetDomainWatch(id)
	if err != nil {
		return nil, err
	}
	// Not found rather than forbidden: do not leak the watches of others.
	if !w.UserId.Equals(user.Id) {
		return nil, happydns.ErrDomainWatchNotFound
	}
	return w, nil
}

func (s *Service) CreateDomainWatch(ctx context.Context, user *happydns.User, input happydns.DomainWatchInput) (*happydns.DomainWatch, error) {
	ascii, display, err := NormalizeName(input.Domain)
	if err != nil {
		return nil, err
	}
	if err := checkNote(input.Note); err != nil {
		return nil, err
	}

	w, err := s.create(user, ascii, display, input.Note)
	if err != nil {
		return nil, err
	}

	// Look it up at once, so the user sees where it stands. The watch is
	// added all the same when this fails: the worker looks it up soon.
	checked, err := s.check(ctx, user, w.Id)
	if err != nil {
		log.Printf("DomainWatch: unable to look %s up once added: %v", w.DomainName, err)
		return w, nil
	}
	return checked, nil
}

func (s *Service) create(user *happydns.User, ascii, display, note string) (*happydns.DomainWatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.store.ListDomainWatchesByUser(user.Id)
	if err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to list the watched domains: %w", err), UserMessage: "Unable to add the domain to the watchlist."}
	}
	for _, e := range existing {
		if e.DomainName == ascii {
			return nil, happydns.CustomError{Err: fmt.Errorf("you already watch %s", display), Status: http.StatusConflict}
		}
	}
	if s.maxPerUser > 0 && len(existing) >= s.maxPerUser {
		return nil, happydns.CustomError{Err: fmt.Errorf("you cannot watch more than %d domains", s.maxPerUser), Status: http.StatusForbidden}
	}

	now := s.nowFn()
	w := &happydns.DomainWatch{
		UserId:      user.Id,
		DomainName:  ascii,
		DisplayName: display,
		Note:        note,
		CreatedAt:   now,
		Phase:       happydns.DomainWatchUnknown,
		NextCheck:   now,
	}
	if err := s.store.CreateDomainWatch(w); err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to store the domain watch: %w", err), UserMessage: "Unable to add the domain to the watchlist."}
	}
	return w, nil
}

func (s *Service) UpdateDomainWatch(user *happydns.User, id happydns.Identifier, input happydns.DomainWatchInput) (*happydns.DomainWatch, error) {
	if err := checkNote(input.Note); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	w, err := s.GetDomainWatch(user, id)
	if err != nil {
		return nil, err
	}
	w.Note = input.Note
	if err := s.store.UpdateDomainWatch(w); err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to update the domain watch: %w", err), UserMessage: "Unable to update the watched domain."}
	}
	return w, nil
}

func (s *Service) DeleteDomainWatch(user *happydns.User, id happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.GetDomainWatch(user, id); err != nil {
		return err
	}
	if err := s.store.DeleteDomainWatch(id); err != nil {
		return happydns.InternalError{Err: fmt.Errorf("unable to delete the domain watch: %w", err), UserMessage: "Unable to remove the domain from the watchlist."}
	}
	return nil
}

func (s *Service) CheckDomainWatch(ctx context.Context, user *happydns.User, id happydns.Identifier) (*happydns.DomainWatch, error) {
	w, err := s.GetDomainWatch(user, id)
	if err != nil {
		return nil, err
	}
	if w.LastCheck != nil && s.nowFn().Sub(*w.LastCheck) < checkThrottle {
		return w, nil
	}
	return s.check(ctx, user, id)
}

// check looks the domain of the watch up now, on behalf of user.
func (s *Service) check(ctx context.Context, user *happydns.User, id happydns.Identifier) (*happydns.DomainWatch, error) {
	w, err := s.GetDomainWatch(user, id)
	if err != nil {
		return nil, err
	}

	lctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	info, lerr := s.lookup(lctx, w.DomainName)
	// A lookup cut short by the user leaving tells nothing about the
	// domain: it is no failure to record.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return s.record(user, id, domaininfo.Result{Info: info, Err: lerr})
}

// record applies the lookup r to the watch id, stores it and notifies its
// owner, user (looked up when nil), of a phase change.
func (s *Service) record(user *happydns.User, id happydns.Identifier, r domaininfo.Result) (*happydns.DomainWatch, error) {
	s.mu.Lock()
	// Read again: it may have changed, or gone, during the lookup.
	w, err := s.store.GetDomainWatch(id)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	now := s.nowFn()
	from, changed := apply(w, r.Info, r.Err, now, s.jitterFn())
	err = s.store.UpdateDomainWatch(w)
	s.mu.Unlock()

	if err != nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("unable to store the domain watch: %w", err), UserMessage: "Unable to save the result of the check."}
	}

	if changed {
		log.Printf("DomainWatch: %s went from %s to %s", w.DomainName, from, w.Phase)
		s.notify(user, w, from, now)
	}
	return w, nil
}

func (s *Service) notify(user *happydns.User, w *happydns.DomainWatch, from happydns.DomainWatchPhase, now time.Time) {
	if s.notifier == nil {
		return
	}

	ev := eventFor(w, from, now)
	if ev == nil {
		return
	}

	if user == nil {
		var err error
		if user, err = s.users.GetUser(w.UserId); err != nil {
			if !errors.Is(err, happydns.ErrUserNotFound) {
				log.Printf("DomainWatch: unable to find the owner of %s to notify: %v", w.DomainName, err)
			}
			return
		}
	}

	s.notifier.NotifyUser(user, &notifPkg.NotificationPayload{
		CheckerID:  EventKind,
		DomainName: w.DomainName,
		Event:      ev,
	})
}

func checkNote(note string) error {
	if len(note) > maxNoteLength {
		return happydns.ValidationError{Msg: fmt.Sprintf("the note cannot exceed %d characters", maxNoteLength)}
	}
	return nil
}
