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

package mailer

import (
	"regexp"
	"strings"
	"testing"
)

func TestRenderMailContent(t *testing.T) {
	out, err := renderMailContent("## Hello\n\nSome `code` and a [link](https://example.com/a?b=1&c=2) inline.\n\n[Validate my account](https://example.com/v?u=1&k=2)\n\n- item\n")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`<h2 style="` + mailContentStyles["h2"] + `"`,
		`<p style="` + mailContentStyles["p"] + `">Some`,
		`<code style="` + mailContentStyles["code"] + `">code</code>`,
		`<a style="` + mailContentStyles["a"] + `" href="https://example.com/a?b=1&amp;c=2">link</a>`,
		`<li style="` + mailContentStyles["li"] + `">item</li>`,
		// The lone link paragraph becomes a button plus a copyable address.
		`text-decoration: none; border-radius: 8px;" href="https://example.com/v?u=1&amp;k=2" target="_blank">Validate my account</a>`,
		`word-break: break-all;" href="https://example.com/v?u=1&amp;k=2">https://example.com/v?u=1&amp;k=2</a>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}

	if regexp.MustCompile(`<[^>]* style=[^>]* style=`).MatchString(out) {
		t.Errorf("a tag carries two style attributes:\n%s", out)
	}
	if strings.Contains(out, `<p style="`+mailContentStyles["p"]+`"><a`) {
		t.Errorf("lone link paragraph was not turned into a button:\n%s", out)
	}
}

func TestRenderMailContentKeepsInlineLinks(t *testing.T) {
	out, err := renderMailContent("Go to [the site](https://example.com) now.\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<table") {
		t.Errorf("link inside a sentence must not become a button:\n%s", out)
	}
}

func TestRenderMailContentEscapesHTML(t *testing.T) {
	out, err := renderMailContent("<script>alert(1)</script> and <p>raw</p>\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<script") || strings.Contains(out, "<p>raw") {
		t.Errorf("raw HTML must not pass through:\n%s", out)
	}
}

func TestRenderMailContentNoHardWraps(t *testing.T) {
	out, err := renderMailContent("first line\nsecond line\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<br") {
		t.Errorf("soft line breaks must not become <br>:\n%s", out)
	}
}
