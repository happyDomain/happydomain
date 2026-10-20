// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package common_test

import (
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/services/common"
)

func newSSHFP(t *testing.T, name string) *dns.SSHFP {
	t.Helper()

	rr, err := dns.NewRR(name + " 3600 IN SSHFP 4 2 0123456789abcdef")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	return rr.(*dns.SSHFP)
}

func TestSSHFPRelative(t *testing.T) {
	in := []*dns.SSHFP{
		newSSHFP(t, "srv.example.com."),
		newSSHFP(t, "srv.example.com."),
	}

	out := common.SSHFPRelative(in, "example.com.", "srv.example.com.")

	if len(out) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(out))
	}
	for i, rr := range out {
		if rr.Hdr.Name != "" {
			t.Errorf("record %d: name = %q; want it relative to the subdomain", i, rr.Hdr.Name)
		}
		if rr.Algorithm != 4 || rr.Type != 2 || rr.FingerPrint != "0123456789abcdef" {
			t.Errorf("record %d: rdata changed: %v", i, rr)
		}
	}
}

func TestSSHFPRelativeEmpty(t *testing.T) {
	if out := common.SSHFPRelative(nil, "example.com.", "srv.example.com."); len(out) != 0 {
		t.Errorf("Expected no record, got %v", out)
	}
}
