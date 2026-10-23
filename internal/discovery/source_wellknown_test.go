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
	"slices"
	"strings"
	"testing"
)

func fqdns(names []Name) (ret []string) {
	for _, name := range names {
		ret = append(ret, name.FQDN)
	}
	return
}

func TestWellKnownFactory(t *testing.T) {
	f := factory(t, "wellknown")

	source, ok := f.New(settingsFrom(t, `{"extra_names":"intranet"}`)).(WellKnownSource)
	if !ok || source.Extra != "intranet" {
		t.Errorf("source = %#v; want the well-known names, with the extra ones", source)
	}
}

func TestWellKnownSource(t *testing.T) {
	target := Target{Domain: "example.com."}

	names, err := WellKnownSource{Extra: "custom\n# a comment\n\n  spaced  \nnot a name!\n"}.Discover(context.Background(), target)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	got := fqdns(names)
	if len(got) < 50 {
		t.Errorf("the builtin list yields %d names; want it loaded", len(got))
	}
	for _, fqdn := range got {
		if !strings.HasSuffix(fqdn, ".example.com.") || strings.Contains(fqdn, "#") {
			t.Errorf("builtin list yields %q", fqdn)
		}
	}
	if !slices.Contains(got, "_dmarc.example.com.") || !slices.Contains(got, "selector1._domainkey.example.com.") {
		t.Errorf("the builtin list misses usual services: %v", got)
	}
	if extra := got[len(got)-2:]; !slices.Equal(extra, []string{"custom.example.com.", "spaced.example.com."}) {
		t.Errorf("extra names = %v; want custom and spaced only, after the builtin ones", extra)
	}
	for _, name := range names {
		if !slices.Equal(name.Sources, []string{"wellknown"}) {
			t.Errorf("%s comes from %v; want wellknown", name.FQDN, name.Sources)
		}
	}
}

// TestWellKnownList checks every name of the builtin list, which a typo
// would otherwise silently drop.
func TestWellKnownList(t *testing.T) {
	for line := range strings.Lines(wellKnownNames) {
		label := strings.TrimSpace(line)
		if label != "" && !strings.HasPrefix(label, "#") && !isHostLikeName(label) {
			t.Errorf("the builtin list holds %q, which is no name", label)
		}
	}
}

func TestIsHostLikeName(t *testing.T) {
	for name, want := range map[string]bool{
		"www":                      true,
		"_dmarc":                   true,
		"selector1._domainkey":     true,
		"Mail-01":                  true,
		"www.example.com.":         true,
		"":                         false,
		".":                        false,
		"..":                       false,
		"a..b":                     false,
		".www":                     false,
		"not a name":               false,
		"tab\tname":                false,
		"café":                     false,
		"*":                        false,
		"@":                        false,
		strings.Repeat("a", 63):    true,
		strings.Repeat("a", 64):    false,
		strings.Repeat("abc.", 70): false,
	} {
		if got := isHostLikeName(name); got != want {
			t.Errorf("isHostLikeName(%q) = %v; want %v", name, got, want)
		}
	}
}

func TestNamesFromLists(t *testing.T) {
	names := namesFromLists("test", "example.com.", "www\r\nMail.Example.com.\n", "# comment\n  ftp  \n")

	if got, want := fqdns(names), []string{"www.example.com.", "mail.example.com.", "ftp.example.com."}; !slices.Equal(got, want) {
		t.Errorf("names = %v; want %v", got, want)
	}
}
