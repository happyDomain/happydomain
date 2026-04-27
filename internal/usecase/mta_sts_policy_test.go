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
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestParseMTASTSBody_Minimal(t *testing.T) {
	resp := &happydns.MTASTSPolicyResponse{}
	parseMTASTSBody("version: STSv1\nmode: enforce\nmx: mail.example.com\nmax_age: 86400\n", resp)
	if resp.Version != "STSv1" {
		t.Errorf("version = %q, want STSv1", resp.Version)
	}
	if resp.Mode != "enforce" {
		t.Errorf("mode = %q, want enforce", resp.Mode)
	}
	if len(resp.MX) != 1 || resp.MX[0] != "mail.example.com" {
		t.Errorf("mx = %v, want [mail.example.com]", resp.MX)
	}
	if resp.MaxAge != 86400 {
		t.Errorf("maxAge = %d, want 86400", resp.MaxAge)
	}
}

func TestParseMTASTSBody_MultipleMX(t *testing.T) {
	resp := &happydns.MTASTSPolicyResponse{}
	parseMTASTSBody("version: STSv1\nmode: testing\nmx: mail1.example.com\nmx: *.mail.example.com\nmax_age: 604800\n", resp)
	if len(resp.MX) != 2 {
		t.Fatalf("mx = %v, want 2 entries", resp.MX)
	}
	if resp.MX[0] != "mail1.example.com" || resp.MX[1] != "*.mail.example.com" {
		t.Errorf("mx = %v, unexpected order/values", resp.MX)
	}
}

func TestParseMTASTSBody_CRLFAndComments(t *testing.T) {
	body := "# example policy\r\nversion: STSv1\r\nmode: enforce\r\n\r\nmx: mail.example.com\r\nmax_age: 3600\r\n"
	resp := &happydns.MTASTSPolicyResponse{}
	parseMTASTSBody(body, resp)
	if resp.Version != "STSv1" || resp.Mode != "enforce" || resp.MaxAge != 3600 || len(resp.MX) != 1 {
		t.Errorf("CRLF/comment handling failed: %+v", resp)
	}
}

func TestParseMTASTSBody_CaseInsensitiveKeys(t *testing.T) {
	resp := &happydns.MTASTSPolicyResponse{}
	parseMTASTSBody("Version: STSv1\nMode: NONE\nMax_Age: 0\n", resp)
	if resp.Version != "STSv1" {
		t.Errorf("version = %q", resp.Version)
	}
	// Mode is lowercased on parse to ease matching.
	if resp.Mode != "none" {
		t.Errorf("mode = %q, want lowercase 'none'", resp.Mode)
	}
	if resp.MaxAge != 0 {
		t.Errorf("maxAge = %d, want 0", resp.MaxAge)
	}
}

func TestParseMTASTSBody_IgnoresUnknownKeys(t *testing.T) {
	resp := &happydns.MTASTSPolicyResponse{}
	parseMTASTSBody("version: STSv1\nfoo: bar\nmode: enforce\n", resp)
	if resp.Version != "STSv1" || resp.Mode != "enforce" {
		t.Errorf("unexpected: %+v", resp)
	}
}

func TestParseMTASTSBody_NonNumericMaxAgeIgnored(t *testing.T) {
	resp := &happydns.MTASTSPolicyResponse{}
	parseMTASTSBody("version: STSv1\nmode: enforce\nmax_age: not-a-number\n", resp)
	if resp.MaxAge != 0 {
		t.Errorf("maxAge = %d, want 0 when value is non-numeric", resp.MaxAge)
	}
}

func TestParseMTASTSBody_LinesWithoutColonIgnored(t *testing.T) {
	resp := &happydns.MTASTSPolicyResponse{}
	parseMTASTSBody("version: STSv1\nthis is junk\nmode: enforce\n", resp)
	if resp.Version != "STSv1" || resp.Mode != "enforce" {
		t.Errorf("unexpected: %+v", resp)
	}
}

func TestFetchMTASTSPolicy_EmptyDomain(t *testing.T) {
	us := &resolverUsecase{}
	_, err := us.FetchMTASTSPolicy(happydns.MTASTSPolicyRequest{Domain: ""})
	if err == nil {
		t.Fatal("expected error on empty domain")
	}
	if !strings.Contains(err.Error(), "domain") {
		t.Errorf("error %q does not mention domain", err)
	}
}
