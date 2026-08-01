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

package service_test

import (
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
	svcs "git.happydns.org/happyDomain/services"
)

// TestParseServiceRenamedType covers the zones stored while the Alias service
// was the CNAME one: they must load, and come back under the name the service
// goes by now, which is what the frontend picks its editor by.
func TestParseServiceRenamedType(t *testing.T) {
	svc, err := service.ParseService(&happydns.ServiceMessage{
		ServiceMeta: happydns.ServiceMeta{
			Type:   "svcs.CNAME",
			Domain: "www",
		},
		Service: json.RawMessage(`{"cname":{"Hdr":{"Name":"www","Rrtype":5,"Class":1,"Ttl":3600},"Target":"target.example.org."}}`),
	})
	if err != nil {
		t.Fatalf("ParseService failed: %s", err)
	}

	if svc.Type != "svcs.Alias" {
		t.Errorf("the service came back as %q, want %q", svc.Type, "svcs.Alias")
	}

	alias, ok := svc.Service.(*svcs.Alias)
	if !ok {
		t.Fatalf("ParseService returned a %T, want a *svcs.Alias", svc.Service)
	}

	if alias.GenComment() != "target.example.org." {
		t.Errorf("GenComment() = %q, want %q", alias.GenComment(), "target.example.org.")
	}
}
