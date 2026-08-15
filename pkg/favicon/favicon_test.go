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

package favicon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// pngBody is the smallest thing http.DetectContentType calls a PNG.
var pngBody = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)

// hostRecordingSource is a minimal Source that records the host it was asked
// about, standing in for a real source until one exists to test against.
type hostRecordingSource struct {
	asked *string
}

func (s hostRecordingSource) Name() string { return "test" }

func (s hostRecordingSource) FetchIcon(site *url.URL) ([]byte, string, error) {
	*s.asked = site.Hostname()
	return pngBody, "image/png", nil
}

func TestSourcesOnNilService(t *testing.T) {
	var fs *FaviconService
	if got := fs.Sources(); got != nil {
		t.Errorf("Sources() on a nil service = %v, want nil", got)
	}
}

func TestNewFaviconServiceRejectsUnknownSource(t *testing.T) {
	if _, err := NewFaviconService(nil, []string{"not-a-real-source"}); err == nil {
		t.Errorf("NewFaviconService with an unknown source = nil error, want an error")
	}
}

func TestFetchRejectsUnsupportedScheme(t *testing.T) {
	fs := &FaviconService{cache: newMemCache()}

	if _, _, err := fs.Fetch("ftp://example.com", time.Minute); err == nil {
		t.Errorf("Fetch on an ftp URL = nil error, want an error")
	}
}

func TestFetchDomainPrependsScheme(t *testing.T) {
	var gotHost string

	fs := &FaviconService{
		cache:   newMemCache(),
		sources: []Source{hostRecordingSource{asked: &gotHost}},
	}

	if _, _, err := fs.FetchDomain("example.com"); err != nil {
		t.Fatalf("FetchDomain: %s", err)
	}
	if gotHost != "example.com" {
		t.Errorf("requested host = %q, want example.com: the domain must be prefixed with https://", gotHost)
	}
}

func TestDownloadIconRejectsBadURL(t *testing.T) {
	if _, _, err := downloadIcon(http.DefaultClient, "not a url"); err == nil {
		t.Errorf("downloadIcon(%q) = nil error, want an error", "not a url")
	}
}

func TestDownloadIconRejectsBrokenBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		// Writes fewer bytes than announced, then the connection is dropped by
		// the handler returning: io.ReadAll must see this as an error, not a
		// short but valid body.
		w.Write(pngBody[:4])
	}))
	defer srv.Close()

	if _, _, err := downloadIcon(srv.Client(), srv.URL+"/icon.png"); err == nil {
		t.Errorf("downloadIcon on a truncated body = nil error, want an error")
	}
}

func TestImageContentType(t *testing.T) {
	if got, err := imageContentType("image/png", nil); err != nil || got != "image/png" {
		t.Errorf("imageContentType(png header) = %q, %v, want image/png, nil", got, err)
	}

	// vnd.microsoft.icon is normalized to the value we actually serve.
	if got, err := imageContentType("image/vnd.microsoft.icon", nil); err != nil || got != "image/x-icon" {
		t.Errorf("imageContentType(ico header) = %q, %v, want image/x-icon, nil", got, err)
	}

	// No header at all: falls back to sniffing the body.
	if got, err := imageContentType("", pngBody); err != nil || got != "image/png" {
		t.Errorf("imageContentType(sniffed) = %q, %v, want image/png, nil", got, err)
	}

	if _, err := imageContentType("text/html", []byte("<script>")); err == nil {
		t.Errorf("imageContentType(text/html) = nil error, want an error")
	}

	if _, err := imageContentType(";;;not a media type", nil); err == nil {
		t.Errorf("imageContentType(malformed header) = nil error, want an error")
	}
}

func TestCacheEvictionDropsOnlyExpiredEntries(t *testing.T) {
	c := newMemCache()

	c.store("expired", &cacheEntry{bytes: make([]byte, maxCacheBytes-10), expiresAt: time.Now().Add(-time.Second)})

	// Storing a second entry that alone fits, but only once the expired one is
	// swept, exercises the branch of evictLocked that trims rather than clears.
	c.store("fresh", &cacheEntry{bytes: make([]byte, 20), expiresAt: time.Now().Add(time.Hour)})

	if _, ok := c.lookup("fresh"); !ok {
		t.Errorf("fresh entry was evicted, want it kept")
	}
	if c.bytes != 20 {
		t.Errorf("bytes = %d, want 20: the expired entry should have been swept, not left counted", c.bytes)
	}
}

// samplePage is the head of a site that declares its icons the way real ones
// do: a share preview that is not an icon, a couple of sizes, and an SVG.
const samplePage = `<!DOCTYPE html><html><head>
<meta property="og:image" content="https://example.com/share-preview-1200x630.png">
<meta property="og:image:width" content="1200">
<meta name="twitter:image" content="https://example.com/twitter-card.jpg">
<link rel="mask-icon" href="/safari-pinned-tab.svg" color="#000000">
<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">
<link rel="icon" type="image/png" sizes="16x16" href="/favicon-16.png">
<link rel="icon" type="image/png" sizes="32x32" href="favicon-32.png">
<link rel="icon" type="image/svg+xml" href="/icon.svg">
<link rel="stylesheet" href="/style.css">
</head><body><img src="/not-an-icon.png"></body></html>`

func parseSample(t *testing.T, page, baseURL string) []iconCandidate {
	t.Helper()

	base, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parsing base: %s", err)
	}

	candidates, _ := parseIconLinks(strings.NewReader(page), base)
	rankCandidates(candidates)

	return candidates
}

func TestParseIconLinksIgnoresShareImages(t *testing.T) {
	candidates := parseSample(t, samplePage, "https://example.com/")

	for _, candidate := range candidates {
		for _, unwanted := range []string{"share-preview", "twitter-card", "safari-pinned-tab", "style.css", "not-an-icon"} {
			if strings.Contains(candidate.url, unwanted) {
				t.Errorf("candidate %q should not have been collected (rel=%q)", candidate.url, candidate.rel)
			}
		}
	}

	if len(candidates) != 4 {
		t.Errorf("collected %d candidates, want 4: %v", len(candidates), candidates)
	}
}

func TestParseIconLinksResolvesRelativeHref(t *testing.T) {
	candidates := parseSample(t, samplePage, "https://example.com/en/")

	var found bool
	for _, candidate := range candidates {
		if candidate.url == "https://example.com/en/favicon-32.png" {
			found = true
		}
	}

	if !found {
		t.Errorf("relative href was not resolved against the page URL: %v", candidates)
	}
}

func TestParseIconLinksHonoursBaseTag(t *testing.T) {
	page := `<html><head><base href="https://cdn.example.net/assets/">
	<link rel="icon" href="icon.png"></head></html>`

	candidates := parseSample(t, page, "https://example.com/")

	if len(candidates) != 1 || candidates[0].url != "https://cdn.example.net/assets/icon.png" {
		t.Errorf("candidates = %v, want the icon resolved against <base>", candidates)
	}
}

func TestParseIconLinksReachesEOFWithoutHeadClose(t *testing.T) {
	// Truncated markup, no </head> and no <body>: parseIconLinks must still
	// return what it collected instead of looping forever.
	page := `<html><head><link rel="icon" href="/a.png">`
	base, _ := url.Parse("https://example.com/")

	candidates, _ := parseIconLinks(strings.NewReader(page), base)
	if len(candidates) != 1 || candidates[0].url != "https://example.com/a.png" {
		t.Errorf("candidates = %v, want the one icon collected before EOF", candidates)
	}
}

func TestParseIconLinksStopsAtBodyWithoutHeadClose(t *testing.T) {
	// A <body> reached with no </head> in between must stop the scan the same
	// way an explicit </head> would.
	page := `<html><head><link rel="icon" href="/a.png"><body><link rel="icon" href="/late.png"></body></html>`
	base, _ := url.Parse("https://example.com/")

	candidates, _ := parseIconLinks(strings.NewReader(page), base)
	if len(candidates) != 1 || candidates[0].url != "https://example.com/a.png" {
		t.Errorf("candidates = %v, want only the icon declared before <body>", candidates)
	}
}

func TestRankCandidatesPrefersScalableThenClosestSize(t *testing.T) {
	candidates := parseSample(t, samplePage, "https://example.com/")

	if got := candidates[0].url; got != "https://example.com/icon.svg" {
		t.Errorf("best candidate = %q, want the SVG", got)
	}

	// The 180x180 only needs to be downscaled to be displayed crisply, while
	// the 32x32 would need to be upscaled: downscaling wins.
	if got := candidates[1].url; got != "https://example.com/apple-touch-icon.png" {
		t.Errorf("second candidate = %q, want the 180x180", got)
	}
}

func TestMimeFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/icon.svg", "image/svg+xml"},
		{"/ICON.SVG", "image/svg+xml"},
		{"/icon.png", "image/png"},
		{"/favicon.ico", "image/x-icon"},
		{"/icon.gif", "image/gif"},
		{"/icon.jpg", "image/jpeg"},
		{"/icon.jpeg", "image/jpeg"},
		{"/icon.webp", "image/webp"},
		{"/icon.bmp", ""},
		{"/icon", ""},
		{"/", ""},
	}

	for _, tt := range tests {
		if got := mimeFromPath(tt.path); got != tt.want {
			t.Errorf("mimeFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestBestDeclaredSizePicksClosestToTarget(t *testing.T) {
	tests := []struct {
		sizes string
		want  int
	}{
		{"32x32", 32},
		{"16x16 32x32 48x48", 48},
		{"64x64 128x128", 64},
		{"any", 0},
		{"not-a-size", 0},
		{"0x0", 0},
		{"-5x-5", 0},
		{"", 0},
	}

	for _, tt := range tests {
		if got := bestDeclaredSize(tt.sizes); got != tt.want {
			t.Errorf("bestDeclaredSize(%q) = %d, want %d", tt.sizes, got, tt.want)
		}
	}
}

func TestCandidateScorePrefersScalableThenDeclaredSize(t *testing.T) {
	scalable := candidateScore(iconCandidate{rel: "icon", scalable: true})
	sized := candidateScore(iconCandidate{rel: "icon", size: targetSize})
	undeclared := candidateScore(iconCandidate{rel: "icon"})

	if scalable >= sized {
		t.Errorf("scalable score %d, want lower than sized score %d", scalable, sized)
	}
	if sized >= undeclared {
		t.Errorf("exact-size score %d, want lower than undeclared score %d", sized, undeclared)
	}
}

func TestDedupCandidatesRespectsLimitAndOrder(t *testing.T) {
	candidates := []iconCandidate{
		{url: "https://example.com/a.png"},
		{url: "https://example.com/a.png"}, // duplicate, dropped before the limit is hit
		{url: "https://example.com/b.png"},
		{url: "https://example.com/c.png"}, // past the limit
	}

	got := dedupCandidates(candidates, 2)

	want := []string{"https://example.com/a.png", "https://example.com/b.png"}
	if len(got) != len(want) {
		t.Fatalf("dedupCandidates = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i].url != w {
			t.Errorf("dedupCandidates[%d] = %q, want %q", i, got[i].url, w)
		}
	}
}

func TestLinkCandidateRejectsUnknownRelAndEmptyHref(t *testing.T) {
	base, _ := url.Parse("https://example.com/")

	if _, ok := linkCandidate(map[string]string{"rel": "stylesheet", "href": "/style.css"}, base); ok {
		t.Errorf("linkCandidate accepted an unknown rel")
	}

	if _, ok := linkCandidate(map[string]string{"rel": "icon", "href": ""}, base); ok {
		t.Errorf("linkCandidate accepted an empty href")
	}

	if _, ok := linkCandidate(map[string]string{"rel": "icon", "href": "%zz"}, base); ok {
		t.Errorf("linkCandidate accepted an unparsable href")
	}
}

func TestDiscoverAlwaysEndsWithWellKnown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head></head><body>no icon here</body></html>"))
	}))
	defer srv.Close()

	site, _ := url.Parse(srv.URL + "/some/page?a=b")
	candidates := newDirectSource(srv.Client()).discover(site, time.Now().Add(maxFetchBudget))

	if len(candidates) != 1 || candidates[0].url != srv.URL+"/favicon.ico" {
		t.Errorf("candidates = %v, want only the well-known path", candidates)
	}
}

func TestDirectSourceTriesNextCandidate(t *testing.T) {
	var asked []string

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)

		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			// The best icon on offer is declared but missing, which is common
			// enough that it must not cost the site its icon.
			w.Write([]byte(`<html><head>
				<link rel="icon" type="image/svg+xml" href="/gone.svg">
				<link rel="icon" type="image/png" sizes="32x32" href="/here.png">
				</head></html>`))
		case "/here.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(pngBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	site, _ := url.Parse(srv.URL)
	body, contentType, err := newDirectSource(srv.Client()).FetchIcon(site)
	if err != nil {
		t.Fatalf("FetchIcon: %s", err)
	}
	if contentType != "image/png" || len(body) != len(pngBody) {
		t.Errorf("got %d bytes of %q, want the PNG", len(body), contentType)
	}

	want := []string{"/", "/gone.svg", "/here.png"}
	if strings.Join(asked, " ") != strings.Join(want, " ") {
		t.Errorf("requested %v, want %v", asked, want)
	}
}

func TestDirectSourceNoCandidatesWhenPageAndFaviconBothFail(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	site, _ := url.Parse(srv.URL)
	if _, _, err := newDirectSource(srv.Client()).FetchIcon(site); err == nil {
		t.Errorf("FetchIcon = nil, want an error when every candidate fails")
	}
}

// hostRedirectTransport lets a test drive requests for arbitrary hostnames to
// a single httptest server, so a redirect's Location header can name a
// different host than the site under test without touching the network.
type hostRedirectTransport struct {
	addr string
	base http.RoundTripper
}

func (t hostRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Host = req.URL.Host
	req.URL.Scheme = "http"
	req.URL.Host = t.addr
	return t.base.RoundTrip(req)
}

func TestSameOrganizationalDomain(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"https://example.org/", "https://example.org/", true},
		{"https://example.org/", "https://www.example.org/", true},
		{"https://example.org/", "https://EXAMPLE.ORG/", true},
		{"https://shop.example.org/", "https://www.example.org/", true},
		{"https://example.org/", "https://example.net/", false},
		{"https://example.org/", "https://example.org.attacker.net/", false},
		{"https://127.0.0.1/", "https://127.0.0.1/", true},
		{"https://127.0.0.1/", "https://127.0.0.2/", false},
	}

	for _, tt := range tests {
		a, _ := url.Parse(tt.a)
		b, _ := url.Parse(tt.b)
		if got := sameOrganizationalDomain(a, b); got != tt.want {
			t.Errorf("sameOrganizationalDomain(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDirectSourceFollowsRedirectWithinOrganization(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "example.org" {
			http.Redirect(w, r, "http://www.example.org/", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><link rel="icon" type="image/png" sizes="32x32" href="/here.png"></head></html>`))
	})
	mux.HandleFunc("/here.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBody)
	})

	client := srv.Client()
	client.Transport = hostRedirectTransport{addr: srv.Listener.Addr().String(), base: client.Transport}

	site, _ := url.Parse("http://example.org/")
	body, contentType, err := newDirectSource(client).FetchIcon(site)
	if err != nil {
		t.Fatalf("FetchIcon: %s", err)
	}
	if contentType != "image/png" || len(body) != len(pngBody) {
		t.Errorf("got %d bytes of %q, want the PNG from the redirected host", len(body), contentType)
	}
}

func TestDirectSourceIgnoresRedirectOutsideOrganization(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "example.org" {
			http.Redirect(w, r, "http://evil.example.net/", http.StatusFound)
			return
		}
		// Reached only if the redirect was wrongly followed.
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><link rel="icon" type="image/png" sizes="32x32" href="/here.png"></head></html>`))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(pngBody)
	})

	client := srv.Client()
	client.Transport = hostRedirectTransport{addr: srv.Listener.Addr().String(), base: client.Transport}

	site, _ := url.Parse("http://example.org/")
	candidates := newDirectSource(client).discover(site, time.Now().Add(maxFetchBudget))

	if len(candidates) != 1 || candidates[0].url != "http://example.org/favicon.ico" {
		t.Errorf("candidates = %v, want only the well-known path: the cross-domain redirect must not be followed", candidates)
	}
}

func TestDirectSourceFollowsMetaRefresh(t *testing.T) {
	// Mirrors a real site: / declares no icon and only bounces to /en/ with
	// a <meta http-equiv="refresh">, not an HTTP redirect, the way a browser
	// tab would still show the icon declared on the page it lands on.
	var asked []string

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><meta http-equiv="refresh" content="0; url=/en/"></head></html>`))
	})
	mux.HandleFunc("/en/", func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><link rel="icon" type="image/png" sizes="32x32" href="/here.png"></head></html>`))
	})
	mux.HandleFunc("/here.png", func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBody)
	})

	site, _ := url.Parse(srv.URL)
	body, contentType, err := newDirectSource(srv.Client()).FetchIcon(site)
	if err != nil {
		t.Fatalf("FetchIcon: %s", err)
	}
	if contentType != "image/png" || len(body) != len(pngBody) {
		t.Errorf("got %d bytes of %q, want the PNG from the page reached via meta refresh", len(body), contentType)
	}

	want := []string{"/", "/en/", "/here.png"}
	if strings.Join(asked, " ") != strings.Join(want, " ") {
		t.Errorf("requested %v, want %v", asked, want)
	}
}

func TestDirectSourceIgnoresMetaRefreshOutsideOrganization(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "example.org" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><meta http-equiv="refresh" content="0; url=http://evil.example.net/"></head></html>`))
			return
		}
		// Reached only if the refresh was wrongly followed.
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><link rel="icon" type="image/png" sizes="32x32" href="/here.png"></head></html>`))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(pngBody)
	})

	client := srv.Client()
	client.Transport = hostRedirectTransport{addr: srv.Listener.Addr().String(), base: client.Transport}

	site, _ := url.Parse("http://example.org/")
	candidates := newDirectSource(client).discover(site, time.Now().Add(maxFetchBudget))

	if len(candidates) != 1 || candidates[0].url != "http://example.org/favicon.ico" {
		t.Errorf("candidates = %v, want only the well-known path: the cross-domain meta refresh must not be followed", candidates)
	}
}

func TestParseMetaRefresh(t *testing.T) {
	base, _ := url.Parse("https://example.com/en/")

	tests := []struct {
		content string
		want    string // "" means ok should be false
	}{
		{"0; url=/fr/", "https://example.com/fr/"},
		{"5; URL='/fr/'", "https://example.com/fr/"},
		{`0;url="/fr/"`, "https://example.com/fr/"},
		{"0;url=https://other.example.com/", "https://other.example.com/"},
		{"no semicolon here", ""},
		{"0; not-a-url=/fr/", ""},
		{"0; url=", ""},
		{"0; url=%zz", ""},
	}

	for _, tt := range tests {
		got, ok := parseMetaRefresh(tt.content, base)
		if tt.want == "" {
			if ok {
				t.Errorf("parseMetaRefresh(%q) = %v, true, want false", tt.content, got)
			}
			continue
		}

		if !ok || got.String() != tt.want {
			t.Errorf("parseMetaRefresh(%q) = %v, %v, want %q, true", tt.content, got, ok, tt.want)
		}
	}
}

func TestFetchIconFailsWhenBudgetExpiresBeforeFirstCandidate(t *testing.T) {
	old := maxFetchBudget
	maxFetchBudget = 30 * time.Millisecond
	defer func() { maxFetchBudget = old }()

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><link rel="icon" href="/here.png"></head></html>`))
	})

	site, _ := url.Parse(srv.URL)
	if _, _, err := newDirectSource(srv.Client()).FetchIcon(site); err == nil {
		t.Errorf("FetchIcon = nil error, want the budget-exceeded error")
	}
}

func TestDiscoverStopsFollowingMetaRefreshWhenBudgetExpires(t *testing.T) {
	old := maxFetchBudget
	maxFetchBudget = 60 * time.Millisecond
	defer func() { maxFetchBudget = old }()

	var hops int

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Every hop bounces to the next one and sleeps long enough that, after a
	// couple of hops, the budget runs out before discover starts another one.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hops++
		time.Sleep(40 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><meta http-equiv="refresh" content="0; url=/"></head></html>`))
	})

	site, _ := url.Parse(srv.URL)
	candidates := newDirectSource(srv.Client()).discover(site, time.Now().Add(maxFetchBudget))

	if len(candidates) != 1 || candidates[0].url != srv.URL+"/favicon.ico" {
		t.Errorf("candidates = %v, want only the well-known path", candidates)
	}
	if hops >= maxMetaRefreshHops+1 {
		t.Errorf("hops = %d, want discover to stop early once the budget ran out", hops)
	}
}

func TestDirectSourceFallsBackToWellKnownWhenPageFails(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// A site refusing the page fetch, as many large ones do.
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(pngBody)
	})

	site, _ := url.Parse(srv.URL)
	if _, _, err := newDirectSource(srv.Client()).FetchIcon(site); err != nil {
		t.Errorf("FetchIcon: %s, want the well-known path to save it", err)
	}
}

func TestFetchPageRejectsUnparsableURL(t *testing.T) {
	// A host with a raw space cannot round-trip through String() and back
	// through url.Parse the way http.NewRequest requires.
	site := &url.URL{Scheme: "http", Host: "exa mple.com"}

	if _, _, err := newDirectSource(http.DefaultClient).fetchPage(site); err == nil {
		t.Errorf("fetchPage on an unparsable URL = nil error, want an error")
	}
}

func TestFetchPageRejectsTooManyRedirects(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})

	site, _ := url.Parse(srv.URL)
	if _, _, err := newDirectSource(srv.Client()).fetchPage(site); err == nil {
		t.Errorf("fetchPage on a redirect loop = nil error, want an error")
	}
}

func TestBuildSourcesSkipsBlankAndRejectsUnknown(t *testing.T) {
	sources, err := buildSources([]string{" ", "direct", ""}, nil)
	if err != nil {
		t.Fatalf("buildSources: %s", err)
	}
	if len(sources) != 1 || sources[0].Name() != SourceDirect {
		t.Errorf("sources = %v, want just direct", sources)
	}

	if _, err := buildSources([]string{"not-a-real-source"}, nil); err == nil {
		t.Errorf("buildSources with an unknown name = nil error, want an error")
	}
}
