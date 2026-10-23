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

package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// commonHostNamesSHA256 is the hash of the SecLists file, recorded in
// SECLISTS-LICENSE: the embedded copy must stay verbatim.
const commonHostNamesSHA256 = "e331367c140298cb179114fdeefa78f58f696219f0dec017a28bb79487cfcf19"

func TestWordlistFactory(t *testing.T) {
	f := factory(t, "wordlist")

	if f.New(settingsFrom(t, `{}`)) == nil {
		t.Error("the wordlist is disabled by default")
	}
	if source := f.New(settingsFrom(t, `{"disable_wordlist":true}`)); source != nil {
		t.Errorf("disable_wordlist gives %v; want no source", source)
	}
}

func TestWordlistSource(t *testing.T) {
	sum := sha256.Sum256([]byte(commonHostNames))
	if got := hex.EncodeToString(sum[:]); got != commonHostNamesSHA256 {
		t.Errorf("subdomains-top1million-5000.txt hashes to %s; want the SecLists file, unmodified", got)
	}

	names, err := WordlistSource{}.Discover(context.Background(), Target{Domain: "example.com."})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(names) != 5000 {
		t.Errorf("the wordlist yields %d names; want all 5000 of them", len(names))
	}
	if len(names) > 0 && names[0].FQDN != "www.example.com." {
		t.Errorf("the wordlist starts with %s; want the most frequent name first", names[0].FQDN)
	}
}

func TestExploreWordlist(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"blog.example.com. 300 IN A 192.0.2.1",
		"_dmarc.example.com. 300 IN TXT \"v=DMARC1; p=none\"",
		"selector1._domainkey.example.com. 300 IN TXT \"v=DKIM1; p=MIGf\"",
		"_submission._tcp.example.com. 300 IN SRV 0 1 587 mail.example.com.",
		"mail.example.com. 300 IN A 192.0.2.2",
		"intra-only.example.com. 300 IN A 192.0.2.3",
		"custom.example.com. 300 IN A 192.0.2.4",
	)
	opts := zone.serve(t)

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{WellKnownSource{Extra: "custom"}, WordlistSource{}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		"blog.example.com. 300 IN A 192.0.2.1",
		"_dmarc.example.com. 300 IN TXT \"v=DMARC1; p=none\"",
		"selector1._domainkey.example.com. 300 IN TXT \"v=DKIM1; p=MIGf\"",
		"_submission._tcp.example.com. 300 IN SRV 0 1 587 mail.example.com.",
		"mail.example.com. 300 IN A 192.0.2.2",
		"custom.example.com. 300 IN A 192.0.2.4",
	)
	assertHasNoName(t, got, "intra-only.example.com.")
}

// TestExploreWordlistBudget checks that a budget too short for the whole
// wordlist still covers the names of services, explored first, and only
// leaves out the rarest host names.
func TestExploreWordlistBudget(t *testing.T) {
	zone := newTestZone(t, "example.com.",
		"example.com. 3600 IN NS ns1.example.com.",
		"ns1.example.com. 300 IN A 127.0.0.1",
		"_dmarc.example.com. 300 IN TXT \"v=DMARC1; p=none\"",
		"_443._tcp.www.example.com. 300 IN TLSA 3 1 1 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"bilbo.example.com. 300 IN A 192.0.2.1",
	)
	opts := zone.serve(t)
	opts.MaxQueries = 300

	if last := strings.TrimSpace(commonHostNames[strings.LastIndex(strings.TrimSpace(commonHostNames), "\n"):]); last != "bilbo" {
		t.Fatalf("the wordlist ends with %q; want bilbo, the name this test expects left out", last)
	}

	records, err := Explore(context.Background(), "example.com.", nil, []NameSource{WellKnownSource{}, WordlistSource{}}, opts)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	got := recordStrings(records)

	assertHas(t, got,
		"_dmarc.example.com. 300 IN TXT \"v=DMARC1; p=none\"",
		"_443._tcp.www.example.com. 300 IN TLSA 3 1 1 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	)
	assertHasNoName(t, got, "bilbo.example.com.")
}
