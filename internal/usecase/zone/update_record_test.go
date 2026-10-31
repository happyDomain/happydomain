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

package zone_test

import (
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/helpers"
	intsvc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
	_ "git.happydns.org/happyDomain/services/abstract"
)

const updateOrigin = "example.com."

// analyzedZone builds a zone from records written relative to updateOrigin,
// as an import would.
func analyzedZone(t *testing.T, records ...string) *happydns.Zone {
	t.Helper()

	var rrs []happydns.Record
	for _, record := range records {
		rr, err := helpers.ParseRecord(record, updateOrigin)
		if err != nil {
			t.Fatalf("ParseRecord(%q): %v", record, err)
		}
		rrs = append(rrs, rr)
	}

	services, _, err := intsvc.AnalyzeZone(updateOrigin, rrs)
	if err != nil {
		t.Fatalf("AnalyzeZone: %v", err)
	}
	return &happydns.Zone{Services: services}
}

// mustRelativeRR parses record as the API does: relative to the origin.
func mustRelativeRR(t *testing.T, record string) happydns.Record {
	t.Helper()

	rr, err := helpers.ParseRecord(record, updateOrigin)
	if err != nil {
		t.Fatalf("ParseRecord(%q): %v", record, err)
	}
	return helpers.RRRelative(rr, updateOrigin)
}

func servicesOfType(services []*happydns.Service, svctype string) []*happydns.Service {
	var found []*happydns.Service
	for _, s := range services {
		if s.Type == svctype {
			found = append(found, s)
		}
	}
	return found
}

func recordsOfService(t *testing.T, svc *happydns.Service) string {
	t.Helper()

	rrs, err := serviceUC.NewListRecordsUsecase().List(svc, updateOrigin, 3600)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var lines []string
	for _, rr := range rrs {
		lines = append(lines, rr.String())
	}
	return strings.Join(lines, "\n")
}

func newUpdateRecordUsecase() *zoneUC.UpdateRecordUsecase {
	list := serviceUC.NewListRecordsUsecase()
	return zoneUC.NewUpdateRecordUsecase(
		zoneUC.NewAddRecordUsecase(list),
		zoneUC.NewDeleteRecordUsecase(list, serviceUC.NewSearchRecordUsecase(list)),
		list,
		serviceUC.NewSearchRecordUsecase(list),
	)
}

// Changing the SOA keeps a single origin, with its name servers and its id.
func Test_UpdateRecordSOA(t *testing.T) {
	zone := analyzedZone(t,
		"@ 86400 IN SOA ns.example.com. hostmaster.example.com. 2026081109 172800 3600 2419200 86400",
		"@ 3600 IN NS ns0.example.com.",
		"@ 3600 IN NS ns1.example.com.",
		"@ 3600 IN MX 10 mx.example.com.",
	)
	origins := servicesOfType(zone.Services[""], "abstract.Origin")
	if len(origins) != 1 {
		t.Fatalf("expected 1 origin before the update, got %d", len(origins))
	}
	id := origins[0].Id

	err := newUpdateRecordUsecase().Update(zone, updateOrigin,
		mustRelativeRR(t, "@ 86400 IN SOA ns.example.com. hostmaster.example.com. 2026081109 172800 3600 2419200 86400"),
		mustRelativeRR(t, "@ 86400 IN SOA ns2.example.com. hostmaster.example.com. 2026081109 172800 3600 2419200 86400"),
	)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if nsonly := servicesOfType(zone.Services[""], "abstract.NSOnlyOrigin"); len(nsonly) != 0 {
		t.Errorf("expected no NSOnlyOrigin, got %d", len(nsonly))
	}
	origins = servicesOfType(zone.Services[""], "abstract.Origin")
	if len(origins) != 1 {
		t.Fatalf("expected 1 origin after the update, got %d", len(origins))
	}
	if !origins[0].Id.Equals(id) {
		t.Errorf("origin id changed: %x, want %x", origins[0].Id, id)
	}
	got := recordsOfService(t, origins[0])
	if !strings.Contains(got, "ns2.example.com.") || strings.Contains(got, "\tns.example.com.") {
		t.Errorf("SOA not updated:\n%s", got)
	}
	if !strings.Contains(got, "ns0.example.com.") || !strings.Contains(got, "ns1.example.com.") {
		t.Errorf("name servers lost:\n%s", got)
	}
}

// Changing the only record of a service keeps the service where it was, with
// its id and the comment of the user.
func Test_UpdateRecordKeepsServiceInPlace(t *testing.T) {
	zone := analyzedZone(t,
		"@ 3600 IN MX 10 mx.example.com.",
		"@ 3600 IN TXT \"v=spf1 -all\"",
		"@ 3600 IN CAA 0 issue \"letsencrypt.org\"",
	)
	// Put the MX service first, so that it would move if it was recreated.
	const idx = 0
	services := zone.Services[""]
	for i, s := range services {
		if s.Type == "svcs.MXs" {
			services[0], services[i] = services[i], services[0]
		}
	}
	mx := services[idx]
	if mx.Type != "svcs.MXs" {
		t.Fatalf("no MX service in %v", services)
	}
	mx.UserComment = "mail"

	err := newUpdateRecordUsecase().Update(zone, updateOrigin,
		mustRelativeRR(t, "@ 3600 IN MX 10 mx.example.com."),
		mustRelativeRR(t, "@ 3600 IN MX 20 mx.example.com."),
	)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if len(zone.Services[""]) != 3 {
		t.Fatalf("expected 3 services, got %d", len(zone.Services[""]))
	}
	got := zone.Services[""][idx]
	if got.Type != "svcs.MXs" || !got.Id.Equals(mx.Id) {
		t.Fatalf("service at %d is %s %x, want svcs.MXs %x", idx, got.Type, got.Id, mx.Id)
	}
	if got.UserComment != "mail" {
		t.Errorf("user comment lost: %q", got.UserComment)
	}
	if rrs := recordsOfService(t, got); !strings.Contains(rrs, "MX\t20 ") {
		t.Errorf("MX not updated:\n%s", rrs)
	}
}

// Moving a record to another subdomain takes it out of its service and puts it
// with the services of its new name.
func Test_UpdateRecordOtherSubdomain(t *testing.T) {
	zone := analyzedZone(t,
		"www 3600 IN A 192.0.2.1",
		"www 3600 IN AAAA 2001:db8::1",
	)

	err := newUpdateRecordUsecase().Update(zone, updateOrigin,
		mustRelativeRR(t, "www 3600 IN AAAA 2001:db8::1"),
		mustRelativeRR(t, "web 3600 IN AAAA 2001:db8::1"),
	)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if len(zone.Services["www"]) != 1 || strings.Contains(recordsOfService(t, zone.Services["www"][0]), "AAAA") {
		t.Errorf("AAAA still under www: %v", zone.Services["www"])
	}
	if len(zone.Services["web"]) != 1 || !strings.Contains(recordsOfService(t, zone.Services["web"][0]), "2001:db8::1") {
		t.Errorf("AAAA not under web: %v", zone.Services["web"])
	}
}

func Test_UpdateRecordNotFound(t *testing.T) {
	zone := analyzedZone(t, "www 3600 IN A 192.0.2.1")

	err := newUpdateRecordUsecase().Update(zone, updateOrigin,
		mustRelativeRR(t, "www 3600 IN A 192.0.2.2"),
		mustRelativeRR(t, "www 3600 IN A 192.0.2.3"),
	)
	if err == nil {
		t.Fatalf("expected an error for a record not in the zone")
	}
}

// A name server added to a zone with an origin joins that origin.
func Test_AddRecordNSJoinsOrigin(t *testing.T) {
	zone := analyzedZone(t,
		"@ 86400 IN SOA ns.example.com. hostmaster.example.com. 1 172800 3600 2419200 86400",
		"@ 3600 IN NS ns0.example.com.",
	)
	id := zone.Services[""][0].Id

	err := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase()).Add(zone, updateOrigin, mustRelativeRR(t, "@ 3600 IN NS ns1.example.com."))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if len(zone.Services[""]) != 1 {
		t.Fatalf("expected 1 service, got %d", len(zone.Services[""]))
	}
	if s := zone.Services[""][0]; s.Type != "abstract.Origin" || !s.Id.Equals(id) {
		t.Errorf("got %s %x, want abstract.Origin %x", s.Type, s.Id, id)
	}
}

// A SOA added to a zone with only name servers turns them into an origin.
func Test_AddRecordSOAJoinsNSOnlyOrigin(t *testing.T) {
	zone := analyzedZone(t,
		"@ 3600 IN NS ns0.example.com.",
		"@ 3600 IN NS ns1.example.com.",
	)
	if zone.Services[""][0].Type != "abstract.NSOnlyOrigin" {
		t.Fatalf("expected an NSOnlyOrigin, got %s", zone.Services[""][0].Type)
	}
	id := zone.Services[""][0].Id

	err := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase()).Add(zone, updateOrigin, mustRelativeRR(t, "@ 86400 IN SOA ns.example.com. hostmaster.example.com. 1 172800 3600 2419200 86400"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if len(zone.Services[""]) != 1 {
		t.Fatalf("expected 1 service, got %d", len(zone.Services[""]))
	}
	s := zone.Services[""][0]
	if s.Type != "abstract.Origin" || !s.Id.Equals(id) {
		t.Errorf("got %s %x, want abstract.Origin %x", s.Type, s.Id, id)
	}
	if got := recordsOfService(t, s); !strings.Contains(got, "ns1.example.com.") {
		t.Errorf("name servers lost:\n%s", got)
	}
}

// A TXT record added next to an SPF policy stays a service of its own.
func Test_AddRecordDoesNotMergeUnrelated(t *testing.T) {
	zone := analyzedZone(t, "@ 3600 IN TXT \"v=spf1 -all\"")

	err := zoneUC.NewAddRecordUsecase(serviceUC.NewListRecordsUsecase()).Add(zone, updateOrigin, mustRelativeRR(t, "@ 3600 IN TXT \"hello\""))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if len(zone.Services[""]) != 2 {
		t.Fatalf("expected 2 services, got %d", len(zone.Services[""]))
	}
}

// Deleting the SOA leaves the name servers in the same service.
func Test_DeleteRecordSOAKeepsServiceId(t *testing.T) {
	zone := analyzedZone(t,
		"@ 86400 IN SOA ns.example.com. hostmaster.example.com. 1 172800 3600 2419200 86400",
		"@ 3600 IN NS ns0.example.com.",
	)
	id := zone.Services[""][0].Id

	list := serviceUC.NewListRecordsUsecase()
	err := zoneUC.NewDeleteRecordUsecase(list, serviceUC.NewSearchRecordUsecase(list)).Delete(zone, updateOrigin, mustRelativeRR(t, "@ 86400 IN SOA ns.example.com. hostmaster.example.com. 1 172800 3600 2419200 86400"))
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if len(zone.Services[""]) != 1 {
		t.Fatalf("expected 1 service, got %d", len(zone.Services[""]))
	}
	if s := zone.Services[""][0]; s.Type != "abstract.NSOnlyOrigin" || !s.Id.Equals(id) {
		t.Errorf("got %s %x, want abstract.NSOnlyOrigin %x", s.Type, s.Id, id)
	}
}
