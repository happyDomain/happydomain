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

package usecase

import (
	"context"
	"net"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// startSuggestionResolver serves handler on a local port, and returns its
// address.
func startSuggestionResolver(t *testing.T, handler dns.HandlerFunc) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	srv := &dns.Server{PacketConn: pc, Handler: handler, NotifyStartedFunc: func() { close(started) }}
	go srv.ActivateAndServe()
	<-started
	t.Cleanup(func() { srv.Shutdown() })
	return pc.LocalAddr().String()
}

func mustRR(t *testing.T, s string) dns.RR {
	t.Helper()
	rr, err := dns.NewRR(s)
	if err != nil {
		t.Fatal(err)
	}
	return rr
}

func TestSuggestProviders(t *testing.T) {
	zones := map[string][]dns.RR{
		"ovh.example.": {
			mustRR(t, "ovh.example. 60 IN NS DNS103.ovh.net."),
			mustRR(t, "ovh.example. 60 IN NS ns103.ovh.net."),
		},
		"mixed.example.": {
			mustRR(t, "mixed.example. 60 IN NS ns103.ovh.net."),
			mustRR(t, "mixed.example. 60 IN NS kim.ns.cloudflare.com."),
		},
		"shared.example.":            {mustRR(t, "shared.example. 60 IN NS dns1.p01.nsone.net.")},
		"whitelabel.example.":        {mustRR(t, "whitelabel.example. 60 IN NS ns1.whitelabel.example.")},
		"xn--caf-dma.example.":       {mustRR(t, "xn--caf-dma.example. 60 IN NS ns103.ovh.net.")},
		"0/25.2.0.192.in-addr.arpa.": {mustRR(t, "0/25.2.0.192.in-addr.arpa. 60 IN NS ns103.ovh.net.")},
		// The name servers of the target of an alias are not those of the alias.
		"alias.example.": {
			mustRR(t, "alias.example. 60 IN CNAME ovh.example."),
			mustRR(t, "ovh.example. 60 IN NS ns103.ovh.net."),
		},
		// Not delegated: only the parent has name servers.
		"sub.ovh.example.": {},
	}
	resolver := startSuggestionResolver(t, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		name := dns.CanonicalName(r.Question[0].Name)
		answer, ok := zones[name]
		switch {
		case name == "broken.example.":
			m.Rcode = dns.RcodeServerFailure
		case !ok:
			m.Rcode = dns.RcodeNameError
		case len(answer) == 0:
			m.Ns = []dns.RR{mustRR(t, "ovh.example. 60 IN NS ns103.ovh.net.")}
		}
		m.Answer = answer
		w.WriteMsg(m)
	})

	uc := NewProviderSuggestionUsecase(&happydns.Options{DefaultResolvers: []string{resolver}})

	for domain, want := range map[string]struct {
		nameservers []string
		providers   []string
	}{
		"ovh.example":               {[]string{"dns103.ovh.net.", "ns103.ovh.net."}, []string{"OVHAPI"}},
		" OVH.example. ":            {[]string{"dns103.ovh.net.", "ns103.ovh.net."}, []string{"OVHAPI"}},
		"mixed.example":             {[]string{"kim.ns.cloudflare.com.", "ns103.ovh.net."}, []string{"CloudflareAPI", "OVHAPI"}},
		"shared.example":            {[]string{"dns1.p01.nsone.net."}, []string{"NS1API", "NetlifyAPI"}},
		"whitelabel.example":        {[]string{"ns1.whitelabel.example."}, []string{}},
		"café.example":              {[]string{"ns103.ovh.net."}, []string{"OVHAPI"}},
		"0/25.2.0.192.in-addr.arpa": {[]string{"ns103.ovh.net."}, []string{"OVHAPI"}},
		"alias.example":             {[]string{}, []string{}},
		"sub.ovh.example":           {[]string{}, []string{}},
		"unknown.example":           {[]string{}, []string{}},
		"broken.example":            {[]string{}, []string{}},
	} {
		t.Run(domain, func(t *testing.T) {
			got, err := uc.SuggestProviders(context.Background(), domain)
			if err != nil {
				t.Fatal(err)
			}
			if got.Nameservers == nil || !slices.Equal(got.Nameservers, want.nameservers) {
				t.Errorf("name servers: got %#v, want %#v", got.Nameservers, want.nameservers)
			}
			if got.Providers == nil || !slices.Equal(got.Providers, want.providers) {
				t.Errorf("providers: got %#v, want %#v", got.Providers, want.providers)
			}
		})
	}
}

func TestSuggestProvidersInvalidName(t *testing.T) {
	var asked atomic.Int32
	resolver := startSuggestionResolver(t, func(w dns.ResponseWriter, r *dns.Msg) { asked.Add(1) })
	uc := NewProviderSuggestionUsecase(&happydns.Options{DefaultResolvers: []string{resolver}})

	for _, domain := range []string{"", ".", "  ", "a..example"} {
		_, err := uc.SuggestProviders(context.Background(), domain)
		if _, ok := err.(happydns.ValidationError); !ok {
			t.Errorf("%q: got error %v, want a ValidationError", domain, err)
		}
	}
	if n := asked.Load(); n != 0 {
		t.Errorf("the resolver has been asked %d times", n)
	}
}

func TestSuggestProvidersTimeout(t *testing.T) {
	resolver := startSuggestionResolver(t, func(w dns.ResponseWriter, r *dns.Msg) {})
	uc := &providerSuggestionUsecase{
		config:  &happydns.Options{DefaultResolvers: []string{resolver, resolver, resolver}},
		timeout: 100 * time.Millisecond,
	}

	start := time.Now()
	got, err := uc.SuggestProviders(context.Background(), "ovh.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Providers) != 0 || got.Providers == nil {
		t.Errorf("got %#v, want an empty list", got.Providers)
	}
	// The budget covers all the resolvers, not each of them.
	if d := time.Since(start); d > time.Second {
		t.Errorf("took %s", d)
	}
}
