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

package mtasts

import (
	"errors"
	"testing"

	"github.com/miekg/dns"

	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"

	"git.happydns.org/happyDomain/internal/usecase/hostedservice/hostedservicetest"
)

// newUsecase wires an Usecase over a single domain owning a single zone that
// holds the given service.
func newUsecase(t *testing.T, fqdn string, service happydns.ServiceBody) *Usecase {
	t.Helper()

	domains, zones := hostedservicetest.NewStore(t, fqdn, service)

	return NewUsecase(domains, zones)
}

func sampleService() *abstract.MTASTS {
	return &abstract.MTASTS{
		Mode:   "enforce",
		MaxAge: 604800,
		MX:     []string{"mail.example.com"},
		PolicyCNAME: &dns.CNAME{
			Hdr:    dns.RR_Header{Name: "mta-sts", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
			Target: "happydomain.example.com.",
		},
	}
}

func TestPolicy(t *testing.T) {
	uc := newUsecase(t, "example.com.", sampleService())

	want := "version: STSv1\r\nmode: enforce\r\nmx: mail.example.com\r\nmax_age: 604800\r\n"

	// Both the policy host and the bare domain resolve to the same policy.
	for _, fqdn := range []string{"mta-sts.example.com.", "example.com."} {
		body, err := uc.Policy(fqdn)
		if err != nil {
			t.Fatalf("Policy(%q): %v", fqdn, err)
		}
		if string(body) != want {
			t.Errorf("Policy(%q) = %q; want %q", fqdn, body, want)
		}
	}
}

func TestPolicy_UnknownDomain(t *testing.T) {
	uc := newUsecase(t, "example.com.", sampleService())

	if _, err := uc.Policy("mta-sts.unknown.example."); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("Policy(unknown) error = %v; want ErrNotFound", err)
	}
}

// A service that only publishes the DNS half has no policy to serve.
func TestPolicy_NoPolicyConfigured(t *testing.T) {
	uc := newUsecase(t, "example.com.", &abstract.MTASTS{})

	if _, err := uc.Policy("mta-sts.example.com."); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("Policy(no policy) error = %v; want ErrNotFound", err)
	}
}

func TestIsManaged(t *testing.T) {
	uc := newUsecase(t, "example.com.", sampleService())

	for fqdn, want := range map[string]bool{
		"mta-sts.example.com.":     true,
		"mta-sts.example.com":      true,
		"mta-sts.unknown.example.": false,
		// Never authorise a certificate for a name we do not answer on, even
		// when the domain itself is hosted here.
		"example.com.":            false,
		"autoconfig.example.com.": false,
	} {
		got, err := uc.IsManaged(fqdn)
		if err != nil {
			t.Fatalf("IsManaged(%q): %v", fqdn, err)
		}
		if got != want {
			t.Errorf("IsManaged(%q) = %v; want %v", fqdn, got, want)
		}
	}
}

// A service configured for the DNS half only (hosting checkbox left off)
// must never authorise a certificate: happyDomain does not answer on
// mta-sts.<domain> in that case.
func TestIsManaged_NotHosted(t *testing.T) {
	uc := newUsecase(t, "example.com.", &abstract.MTASTS{
		Mode:   "enforce",
		MaxAge: 604800,
		MX:     []string{"mail.example.com"},
	})

	got, err := uc.IsManaged("mta-sts.example.com.")
	if err != nil {
		t.Fatalf("IsManaged: %v", err)
	}
	if got {
		t.Errorf("IsManaged() = true; want false (no PolicyCNAME)")
	}
}
