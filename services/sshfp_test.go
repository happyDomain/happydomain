// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package svcs_test

import (
	"encoding/json"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"

	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
)

const sshfpFingerprint = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestSSHFPAnalyzeKeepsWholeRecords(t *testing.T) {
	rr, err := dns.NewRR("srv.example.com. 3600 IN SSHFP 4 2 " + sshfpFingerprint)
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	// Two addresses keep abstract.Server from claiming the fingerprint.
	records := []happydns.Record{rr}
	for _, a := range []string{"192.0.2.1", "192.0.2.2"} {
		arr, err := dns.NewRR("srv.example.com. 3600 IN A " + a)
		if err != nil {
			t.Fatalf("dns.NewRR failed: %v", err)
		}
		records = append(records, arr)
	}

	s, _, err := svc.AnalyzeZone("example.com.", records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	var sshfps *svcs.SSHFPs
	for _, service := range s["srv"] {
		if found, ok := service.Service.(*svcs.SSHFPs); ok {
			sshfps = found
		}
	}
	if sshfps == nil {
		t.Fatalf("Expected an SSHFPs service on srv, got %v", s["srv"])
	}
	if len(sshfps.SSHFP) != 1 {
		t.Fatalf("Expected 1 fingerprint, got %d", len(sshfps.SSHFP))
	}

	got := sshfps.SSHFP[0]
	if got.Hdr.Name != "" || got.Hdr.Rrtype != dns.TypeSSHFP {
		t.Errorf("Header = %+v; want a relative name and the SSHFP type", got.Hdr)
	}
	if got.Algorithm != 4 || got.Type != 2 || got.FingerPrint != sshfpFingerprint {
		t.Errorf("Record = %v; want 4 2 %s", got, sshfpFingerprint)
	}
}

func TestSSHFPDecodesLegacyBody(t *testing.T) {
	// The body services were saved with before carrying whole records.
	legacy := `{"SSHFP":[{"algorithm":4,"type":2,"fingerprint":"` + sshfpFingerprint + `"}]}`

	var s svcs.SSHFPs
	if err := json.Unmarshal([]byte(legacy), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	records, err := s.GetRecords("srv", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	rr, ok := records[0].(*dns.SSHFP)
	if !ok {
		t.Fatalf("Expected *dns.SSHFP, got %T", records[0])
	}
	if rr.Hdr.Rrtype != dns.TypeSSHFP || rr.Hdr.Class != dns.ClassINET {
		t.Errorf("Header = %+v; want type SSHFP, class IN", rr.Hdr)
	}
	if rr.Algorithm != 4 || rr.Type != 2 || rr.FingerPrint != sshfpFingerprint {
		t.Errorf("Record = %v; want 4 2 %s", rr, sshfpFingerprint)
	}

	// The stored body is left as it was.
	if s.SSHFP[0].Hdr.Rrtype != 0 {
		t.Errorf("GetRecords modified the stored record: %+v", s.SSHFP[0].Hdr)
	}
}
