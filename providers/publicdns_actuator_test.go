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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"context"
	"errors"
	"net"
	"slices"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/discovery"
	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

type testTechnique struct{}

func (testTechnique) ID() string { return "test-publicdns" }

func (testTechnique) Discover(ctx context.Context, t discovery.Target) ([]discovery.Name, error) {
	return nil, nil
}

func init() {
	discovery.RegisterSource("test-publicdns", 999, func(s *discovery.Settings) discovery.NameSource {
		return testTechnique{}
	})
}

func TestPublicDNSDiscoveryTechniques(t *testing.T) {
	if !slices.ContainsFunc((&PublicDNSDiscovery{}).Sources(), func(s discovery.NameSource) bool { return s.ID() == "test-publicdns" }) {
		t.Error("a registered technique is not used")
	}
}

func TestPublicDNSDiscoveryActuator(t *testing.T) {
	body := &PublicDNSDiscovery{}

	guard, err := netguard.New("test", "-test-allowed-target", nil)
	if err != nil {
		t.Fatalf("guard: %v", err)
	}
	instance, err := body.InstantiateGuardedProvider(guard, []string{"9.9.9.9:53"})
	if err != nil {
		t.Fatalf("InstantiateGuardedProvider: %v", err)
	}
	actuator := instance.(*publicDNSActuator)
	if opts := actuator.opts; opts.Guard != guard || !slices.Equal(opts.Resolvers, []string{"9.9.9.9:53"}) {
		t.Errorf("options = %+v; want the guard and the resolvers handed", opts)
	}

	// Without a guard, the exploration refuses every non-public server.
	if instance, err = body.InstantiateProvider(); err != nil || instance.(*publicDNSActuator).opts.Guard != nil {
		t.Errorf("InstantiateProvider = %v, %v; want an actuator without guard", instance, err)
	}

	if actuator.CanCreateDomain() || actuator.CanListZones() {
		t.Error("the discovery claims to create domains or list zones")
	}
	if err := actuator.CreateDomain("example.com."); err == nil {
		t.Error("CreateDomain succeeded")
	}
	if _, err := actuator.ListZones(); err == nil {
		t.Error("ListZones succeeded")
	}
	if _, _, err := actuator.GetZoneCorrections("example.com.", nil); err == nil {
		t.Error("GetZoneCorrections succeeded")
	}
}

// serveRecursive answers like a recursive resolver to which example.com. is
// served by ns1.example.com., at 127.0.0.1, and returns its address.
func serveRecursive(t *testing.T) string {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &dns.Server{PacketConn: pc, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)
		var answer string
		switch q := req.Question[0]; {
		case q.Name == "example.com." && q.Qtype == dns.TypeNS:
			answer = "example.com. 300 IN NS ns1.example.com."
		case q.Name == "ns1.example.com." && q.Qtype == dns.TypeA:
			answer = "ns1.example.com. 300 IN A 127.0.0.1"
		}
		if answer == "" {
			m.Rcode = dns.RcodeNameError
		} else if rr, err := dns.NewRR(answer); err == nil {
			m.Answer = append(m.Answer, rr)
		}
		w.WriteMsg(m)
	})}
	go srv.ActivateAndServe()
	t.Cleanup(func() { srv.Shutdown() })

	return pc.LocalAddr().String()
}

func TestPublicDNSDomainExistence(t *testing.T) {
	resolver := serveRecursive(t)
	guard, err := netguard.New("test", "-test-allowed-target", []string{"127.0.0.0/8"})
	if err != nil {
		t.Fatalf("guard: %v", err)
	}

	instance, err := (&PublicDNSDiscovery{}).InstantiateGuardedProvider(guard, []string{resolver})
	if err != nil {
		t.Fatalf("InstantiateGuardedProvider: %v", err)
	}
	actuator := instance.(*publicDNSActuator)
	if err := actuator.TestDomainExistence(context.Background(), "example.com."); err != nil {
		t.Errorf("TestDomainExistence(example.com.) = %v; want the domain found", err)
	}
	if err := actuator.TestDomainExistence(context.Background(), "unregistered.example."); err == nil {
		t.Error("TestDomainExistence succeeded for a domain without name server")
	}

	// Without the guard allowing it, the loopback name server is refused.
	instance, err = (&PublicDNSDiscovery{}).InstantiateGuardedProvider(nil, []string{resolver})
	if err != nil {
		t.Fatalf("InstantiateGuardedProvider: %v", err)
	}
	if err := instance.(*publicDNSActuator).TestDomainExistence(context.Background(), "example.com."); err == nil {
		t.Error("TestDomainExistence succeeded with a name server the guard refuses")
	}
}

func TestPublicDNSZoneRecords(t *testing.T) {
	type key struct{}
	var (
		gotCtx     context.Context
		gotDomain  string
		gotKnown   []string
		gotSources []discovery.NameSource
		gotOpts    discovery.Options

		err error
	)
	record, _ := dns.NewRR("example.com. 300 IN A 192.0.2.1")
	defer func(orig func(context.Context, string, []string, []discovery.NameSource, discovery.Options) ([]happydns.Record, error)) {
		explore = orig
	}(explore)
	explore = func(ctx context.Context, domain string, known []string, sources []discovery.NameSource, opts discovery.Options) ([]happydns.Record, error) {
		gotCtx, gotDomain, gotKnown, gotSources, gotOpts = ctx, domain, known, sources, opts
		return []happydns.Record{record}, err
	}

	guard, _ := netguard.New("test", "-test-allowed-target", nil)
	instance, _ := (&PublicDNSDiscovery{}).InstantiateGuardedProvider(guard, []string{"9.9.9.9:53"})
	actuator := instance.(*publicDNSActuator)

	ctx := context.WithValue(context.Background(), key{}, "import")
	records, e := actuator.GetZoneRecordsWithContext(ctx, "example.com.", []string{"www"})
	if e != nil || len(records) != 1 || records[0] != record {
		t.Fatalf("GetZoneRecordsWithContext = %v, %v; want the records of the exploration", records, e)
	}
	if gotCtx.Value(key{}) != "import" || gotDomain != "example.com." || !slices.Equal(gotKnown, []string{"www"}) {
		t.Errorf("Explore got %v, %q, %v; want the context, the domain and the known names handed", gotCtx, gotDomain, gotKnown)
	}
	if len(gotSources) != len(actuator.sources) || gotOpts.Guard != guard || !slices.Equal(gotOpts.Resolvers, []string{"9.9.9.9:53"}) {
		t.Errorf("Explore got %d sources and %+v; want the sources and the options of the provider", len(gotSources), gotOpts)
	}

	if _, e := actuator.GetZoneRecords("example.com."); e != nil || gotKnown != nil {
		t.Errorf("GetZoneRecords = %v, with known names %v; want no known name", e, gotKnown)
	}

	err = errors.New("unreachable")
	if _, e := actuator.GetZoneRecordsWithContext(ctx, "example.com.", nil); !errors.Is(e, err) {
		t.Errorf("GetZoneRecordsWithContext error = %v; want that of the exploration", e)
	}
}
