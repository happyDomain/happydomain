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
	"errors"
	"strings"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
	"git.happydns.org/happyDomain/pkg/domaininfo/types"
)

var now = time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

func at(d time.Duration) *time.Time {
	t := now.Add(d)
	return &t
}

func registeredInfo(expiresIn time.Duration, status ...string) *domaininfo.DomainInfo {
	return &domaininfo.DomainInfo{ExpirationDate: at(expiresIn), Registrar: "Acme", Status: status}
}

func TestPhaseOf(t *testing.T) {
	cases := []struct {
		name string
		info *domaininfo.DomainInfo
		err  error
		want happydns.DomainWatchPhase
		ok   bool
	}{
		{"registered", registeredInfo(200 * day), nil, happydns.DomainWatchRegistered, true},
		{"no expiration date", &domaininfo.DomainInfo{}, nil, happydns.DomainWatchRegistered, true},
		{"expiring", registeredInfo(20 * day), nil, happydns.DomainWatchExpiring, true},
		{"auto-renewed by the registry", registeredInfo(347*day, "auto renew period"), nil, happydns.DomainWatchRegistered, true},
		{"expired", registeredInfo(-2 * day), nil, happydns.DomainWatchGrace, true},
		{"redemption", registeredInfo(-40*day, "redemptionPeriod"), nil, happydns.DomainWatchRedemption, true},
		{"pending delete", registeredInfo(-70*day, "pending delete"), nil, happydns.DomainWatchPendingDelete, true},
		{"available", nil, types.ErrDomainDoesNotExist, happydns.DomainWatchAvailable, true},
		{"lookup failure", nil, errors.New("timeout"), "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := phaseOf(tc.info, tc.err, now)
			if got != tc.want || ok != tc.ok {
				t.Errorf("phaseOf() = %q, %v, want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestApplyFirstLookupDoesNotNotify(t *testing.T) {
	w := &happydns.DomainWatch{Phase: happydns.DomainWatchUnknown}

	if _, changed := apply(w, registeredInfo(200*day), nil, now, 0); changed {
		t.Error("the first phase found must not be notified")
	}
	if w.Phase != happydns.DomainWatchRegistered || w.Registrar != "Acme" || w.ExpirationDate == nil || w.PhaseSince == nil {
		t.Errorf("watch not filled: %+v", w)
	}
	if len(w.History) != 0 {
		t.Errorf("history = %+v, want empty", w.History)
	}
}

func TestApplyTransition(t *testing.T) {
	w := &happydns.DomainWatch{Phase: happydns.DomainWatchGrace}

	from, changed := apply(w, registeredInfo(-40*day, "redemptionPeriod"), nil, now, 0)
	if !changed || from != happydns.DomainWatchGrace || w.Phase != happydns.DomainWatchRedemption {
		t.Fatalf("apply() = %q, %v; phase %q", from, changed, w.Phase)
	}
	if len(w.History) != 1 || w.History[0] != (happydns.DomainWatchEvent{At: now, From: happydns.DomainWatchGrace, To: happydns.DomainWatchRedemption}) {
		t.Errorf("history = %+v", w.History)
	}
	if w.EstimatedRelease == nil || !w.EstimatedRelease.Equal(now.Add(35*day)) {
		t.Errorf("estimated release = %v, want in 35 days", w.EstimatedRelease)
	}

	// Same phase again: nothing to tell.
	if _, changed := apply(w, registeredInfo(-41*day, "redemptionPeriod"), nil, now.Add(12*time.Hour), 0); changed {
		t.Error("an unchanged phase must not be notified")
	}
	if !w.PhaseSince.Equal(now) {
		t.Errorf("phase since moved to %v", w.PhaseSince)
	}
}

func TestApplyAvailableForgetsTheRegistration(t *testing.T) {
	w := &happydns.DomainWatch{Phase: happydns.DomainWatchPendingDelete, Registrar: "Acme", ExpirationDate: at(-80 * day), Status: []string{"pendingDelete"}}

	if _, changed := apply(w, nil, types.ErrDomainDoesNotExist, now, 0); !changed {
		t.Fatal("available must be notified")
	}
	if w.Registrar != "" || w.ExpirationDate != nil || w.Status != nil || w.EstimatedRelease != nil {
		t.Errorf("registration facts kept: %+v", w)
	}
}

func TestApplyLookupFailureKeepsThePhase(t *testing.T) {
	w := &happydns.DomainWatch{Phase: happydns.DomainWatchPendingDelete, Registrar: "Acme"}

	if _, changed := apply(w, nil, errors.New("RDAP server unreachable"), now, 0); changed {
		t.Error("a failed lookup must not be notified")
	}
	if w.Phase != happydns.DomainWatchPendingDelete || w.Registrar != "Acme" {
		t.Errorf("a failed lookup changed the watch: %+v", w)
	}
	if w.LastError != "RDAP server unreachable" || !w.NextCheck.Equal(now.Add(errorRetry)) {
		t.Errorf("last error %q, next check %v", w.LastError, w.NextCheck)
	}

	// The next successful lookup clears the error.
	apply(w, registeredInfo(-80*day, "pendingDelete"), nil, now, 0)
	if w.LastError != "" {
		t.Errorf("last error kept: %q", w.LastError)
	}

	// An answer telling nothing is no better.
	apply(w, nil, nil, now, 0)
	if w.LastError != "no registration information returned" || w.Phase != happydns.DomainWatchPendingDelete {
		t.Errorf("empty answer: last error %q, phase %q", w.LastError, w.Phase)
	}

	// Never looked up successfully: retry soon.
	w = &happydns.DomainWatch{Phase: happydns.DomainWatchUnknown}
	apply(w, nil, errors.New("timeout"), now, 0)
	if !w.NextCheck.Equal(now.Add(firstErrorRetry)) {
		t.Errorf("next check %v, want in %v", w.NextCheck, firstErrorRetry)
	}
}

func TestApplyHistoryIsBounded(t *testing.T) {
	w := &happydns.DomainWatch{Phase: happydns.DomainWatchRegistered}
	for i := range 30 {
		if i%2 == 0 {
			apply(w, nil, types.ErrDomainDoesNotExist, now, 0)
		} else {
			apply(w, registeredInfo(200*day), nil, now, 0)
		}
	}
	if len(w.History) != maxHistory {
		t.Errorf("history holds %d events, want %d", len(w.History), maxHistory)
	}
}

func TestNextCheck(t *testing.T) {
	cases := []struct {
		name   string
		phase  happydns.DomainWatchPhase
		exp    *time.Time
		status []string
		want   time.Time
	}{
		{"registered far from expiring: weekly", happydns.DomainWatchRegistered, at(300 * day), nil, now.Add(stableInterval)},
		{"registered without expiration date: weekly", happydns.DomainWatchRegistered, nil, nil, now.Add(stableInterval)},
		{"registered, starts expiring in 3 days", happydns.DomainWatchRegistered, at(33 * day), nil, now.Add(3 * day)},
		{"expiring in 20 days: weekly", happydns.DomainWatchExpiring, at(20 * day), nil, now.Add(stableInterval)},
		{"expiring in 2 days: right after expiration", happydns.DomainWatchExpiring, at(2 * day), nil, now.Add(2*day + afterExpiration)},
		{"grace: twice a day", happydns.DomainWatchGrace, at(-3 * day), nil, now.Add(closeInterval)},
		{"redemption: twice a day", happydns.DomainWatchRedemption, at(-40 * day), nil, now.Add(closeInterval)},
		{"pending delete: twice a day", happydns.DomainWatchPendingDelete, nil, nil, now.Add(closeInterval)},
		{"auto-renewed, awaiting the registrar: twice a day", happydns.DomainWatchRegistered, at(347 * day), []string{"autoRenewPeriod"}, now.Add(closeInterval)},
		{"available: twice a day", happydns.DomainWatchAvailable, nil, nil, now.Add(closeInterval)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &happydns.DomainWatch{Phase: tc.phase, ExpirationDate: tc.exp, Status: tc.status}
			if got := nextCheck(w, now); !got.Equal(tc.want) {
				t.Errorf("nextCheck() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEventFor(t *testing.T) {
	cases := []struct {
		from, to happydns.DomainWatchPhase
		title    string
		body     string
	}{
		{happydns.DomainWatchRegistered, happydns.DomainWatchExpiring, "example.com expires in 20 days", "ends on November 13, 2026"},
		{happydns.DomainWatchExpiring, happydns.DomainWatchGrace, "example.com has expired", "can still renew it"},
		{happydns.DomainWatchGrace, happydns.DomainWatchRedemption, "example.com entered its redemption period", "by November 28, 2026"},
		{happydns.DomainWatchRedemption, happydns.DomainWatchPendingDelete, "example.com will be released within days", "by October 29, 2026"},
		{happydns.DomainWatchPendingDelete, happydns.DomainWatchAvailable, "example.com is available!", "register it now"},
		{happydns.DomainWatchAvailable, happydns.DomainWatchRegistered, "example.com was registered", "through `Acme`"},
		{happydns.DomainWatchRedemption, happydns.DomainWatchRegistered, "example.com was renewed", "renewed it, until"},
		{happydns.DomainWatchGrace, happydns.DomainWatchExpiring, "example.com was renewed", "will tell you if it expires again"},
		{happydns.DomainWatchExpiring, happydns.DomainWatchRegistered, "example.com was renewed", "renewed it"},
		{happydns.DomainWatchRegistered, happydns.DomainWatchPendingDelete, "example.com will be released within days", "pending deletion"},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			w := &happydns.DomainWatch{DomainName: "example.com", DisplayName: "example.com", Phase: tc.to, PhaseSince: &now, Registrar: "Acme"}
			if tc.to == happydns.DomainWatchExpiring {
				w.ExpirationDate = at(20*day + time.Hour)
			} else if tc.to != happydns.DomainWatchAvailable {
				w.ExpirationDate = at(300 * day)
			}
			w.EstimatedRelease = estimateRelease(w)

			ev := eventFor(w, tc.from, now)
			if ev == nil {
				t.Fatal("no event")
			}
			if ev.Title != tc.title {
				t.Errorf("title = %q, want %q", ev.Title, tc.title)
			}
			if !strings.Contains(ev.Body, tc.body) {
				t.Errorf("body = %q, want it to contain %q", ev.Body, tc.body)
			}
			if !strings.Contains(ev.Body, "`example.com`") {
				t.Errorf("body does not quote the domain: %q", ev.Body)
			}
			if ev.Kind != EventKind || ev.Path != "/watchlist/" {
				t.Errorf("kind %q, path %q", ev.Kind, ev.Path)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		in, ascii, display string
	}{
		{"example.com", "example.com", "example.com"},
		{"  Example.COM. ", "example.com", "example.com"},
		{"www.shop.example.co.uk", "example.co.uk", "example.co.uk"},
		{"https://www.example.org/some/page?x=1", "example.org", "example.org"},
		{"bücher.de", "xn--bcher-kva.de", "bücher.de"},
		{"xn--bcher-kva.de", "xn--bcher-kva.de", "bücher.de"},
	}
	for _, tc := range cases {
		ascii, display, err := NormalizeName(tc.in)
		if err != nil || ascii != tc.ascii || display != tc.display {
			t.Errorf("NormalizeName(%q) = %q, %q, %v, want %q, %q", tc.in, ascii, display, err, tc.ascii, tc.display)
		}
	}

	for _, in := range []string{"", "com", "co.uk", "exa mple.com", "-.com"} {
		if _, _, err := NormalizeName(in); err == nil {
			t.Errorf("NormalizeName(%q) succeeded, want an error", in)
		}
	}
}
