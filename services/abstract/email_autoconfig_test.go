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

package abstract_test

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/services/abstract"
)

func mkSRV(name string, port uint16, target string) *dns.SRV {
	return &dns.SRV{
		Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 3600},
		Weight: 1,
		Port:   port,
		Target: target,
	}
}

func mkCNAME(name, target string) *dns.CNAME {
	return &dns.CNAME{
		Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 3600},
		Target: target,
	}
}

func TestEmailAutoConfig_GetRecords(t *testing.T) {
	abstract.SetAutoconfigHost("happydomain.example.com")
	t.Cleanup(func() { abstract.SetAutoconfigHost("") })

	svc := &abstract.EmailAutoConfig{
		IncomingSRV:       mkSRV("_imaps._tcp", 993, "imap.example.com."),
		OutgoingSRV:       mkSRV("_submission._tcp", 587, "smtp.example.com."),
		AutoconfigCNAME:   mkCNAME("autoconfig", "happydomain.example.com."),
		AutodiscoverCNAME: mkCNAME("autodiscover", "happydomain.example.com."),
		AutodiscoverSRV:   mkSRV("_autodiscover._tcp", 443, "happydomain.example.com."),
	}

	rrs, err := svc.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	want := map[string]bool{
		"_imaps._tcp":        false,
		"_submission._tcp":   false,
		"autoconfig":         false,
		"autodiscover":       false,
		"_autodiscover._tcp": false,
	}
	for _, rr := range rrs {
		name := rr.Header().Name
		if _, ok := want[name]; ok {
			want[name] = true
		} else {
			t.Errorf("unexpected record name: %q", name)
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing record %q in output", name)
		}
	}

	if got := svc.GetNbResources(); got != 5 {
		t.Errorf("GetNbResources = %d, want 5", got)
	}

	for _, rr := range rrs {
		switch r := rr.(type) {
		case *dns.CNAME:
			if !strings.HasSuffix(r.Target, "happydomain.example.com.") {
				t.Errorf("CNAME target = %q, want suffix happydomain.example.com.", r.Target)
			}
		case *dns.SRV:
			if r.Header().Name == "_autodiscover._tcp" && r.Port != 443 {
				t.Errorf("autodiscover SRV port = %d, want 443", r.Port)
			}
		}
	}
}

func TestEmailAutoConfig_GetRecords_NoDiscovery(t *testing.T) {
	abstract.SetAutoconfigHost("")
	t.Cleanup(func() { abstract.SetAutoconfigHost("") })

	svc := &abstract.EmailAutoConfig{
		IncomingSRV: mkSRV("_imap._tcp", 143, "imap.example.com."),
		OutgoingSRV: mkSRV("_submission._tcp", 587, "smtp.example.com."),
	}

	rrs, err := svc.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	if len(rrs) != 2 {
		t.Errorf("len(rrs) = %d, want 2", len(rrs))
	}
}

func TestEmailAutoConfig_GenComment(t *testing.T) {
	svc := &abstract.EmailAutoConfig{
		IncomingSRV:       mkSRV("_imaps._tcp", 993, "imap.example.com."),
		OutgoingSRV:       mkSRV("_submission._tcp", 587, "smtp.example.com."),
		AutoconfigCNAME:   mkCNAME("autoconfig", "happydomain.example.com."),
		AutodiscoverCNAME: mkCNAME("autodiscover", "happydomain.example.com."),
	}

	got := svc.GenComment()
	for _, want := range []string{"IMAPS", "imap.example.com:993", "submission", "smtp.example.com:587", "autoconfig"} {
		if !strings.Contains(got, want) {
			t.Errorf("GenComment missing %q in %q", want, got)
		}
	}
}
