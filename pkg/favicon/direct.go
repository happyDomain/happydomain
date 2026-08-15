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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
)

const (
	// maxPageSize is how much of a page we read looking for its icon links.
	// They live in the head, so anything past that is the site's content.
	maxPageSize = 1 << 20 // 1MB

	// maxCandidates bounds how many icons we try before giving up on a site.
	// Declaring an icon that 404s is common enough that stopping at the first
	// one loses sites needlessly, and cheap enough to retry that it is worth
	// the extra request.
	maxCandidates = 4

	// maxPageRedirects is how many hops fetchPage follows before giving up.
	// It duplicates netguard's own bound rather than importing it, since the
	// check performed here (organizational domain) is unrelated to netguard's
	// (address safety, which the shared dialer keeps enforcing regardless).
	maxPageRedirects = 5

	// maxMetaRefreshHops bounds how many <meta http-equiv="refresh"> pages
	// discover follows looking for one that declares an icon. Sites that
	// gate their homepage behind a refresh to a language path (rather than
	// an HTTP redirect, which fetchPage already follows) are what this
	// exists for; it stays small because each hop is a full page fetch.
	maxMetaRefreshHops = 3

	// targetSize is the width we would like, in pixels: the interface displays
	// icons around 16 to 32 CSS pixels, so 64 covers a high-density screen
	// without dragging in the 512px icon meant for a phone home screen.
	targetSize = 64

	// userAgent identifies us to the sites we fetch. Announcing what this is
	// gives an operator something to allow or to refuse knowingly.
	userAgent = "happyDomain-favicon/1 (+https://happydomain.org/)"
)

// maxFetchBudget bounds the wall-clock time FetchIcon spends across all of its
// page fetches (meta-refresh hops) and icon downloads combined. Each
// individual request is already capped at fetchTimeout, but the two loops
// that make up a lookup can chain several of them; without an overall budget
// the worst case is their product, not their sum of one.
//
// A var rather than a const so tests can shrink it instead of taking ten real
// seconds to exercise the budget running out.
var maxFetchBudget = 10 * time.Second

// iconRels are the link relations that designate an icon of the site.
//
// This list is the whole point of doing the discovery here rather than with a
// library: what we want is what a browser would put in a tab, and nothing else.
// The OpenGraph and Twitter images that generic favicon finders also collect
// are share previews, typically 1200x630 photographs weighing hundreds of
// kilobytes. They are not icons, they do not survive being drawn at 16 pixels,
// and a finder that merges them into one flat list gives no way to tell them
// apart afterwards.
//
// mask-icon is left out too: it is a monochrome silhouette meant to be tinted
// by Safari, and it renders as a black blob anywhere else.
var iconRels = map[string]int{
	"icon":                         0,
	"shortcut icon":                0,
	"alternate icon":               10,
	"apple-touch-icon":             20,
	"apple-touch-icon-precomposed": 20,
	"fluid-icon":                   40,
}

// directSource asks the site itself: it fetches the page, reads the icon links
// out of it, and downloads the best one.
//
// It is the only source that works for a domain nobody has crawled, which is
// most of what a happyDomain user manages, and the only one that tells no third
// party which domains this instance holds. It is also the only one that makes
// happyDomain fetch a URL a hostile party controls, hence the guarded client.
type directSource struct {
	client *http.Client
}

func newDirectSource(client *http.Client) *directSource {
	return &directSource{client: client}
}

func (ds *directSource) Name() string {
	return SourceDirect
}

func (ds *directSource) FetchIcon(site *url.URL) ([]byte, string, error) {
	deadline := time.Now().Add(maxFetchBudget)

	// discover always returns at least the well-known /favicon.ico path, so
	// candidates is never empty here.
	candidates := ds.discover(site, deadline)

	errs := make([]error, 0, len(candidates))

	for _, candidate := range candidates {
		if time.Now().After(deadline) {
			errs = append(errs, fmt.Errorf("fetch budget exceeded before trying %s", candidate.url))
			break
		}

		// The icon URL comes out of the remote page, so it is as untrusted as
		// the page itself. downloadIcon rejects anything that is not an
		// http(s) URL before the guarded client gets a chance to make sense
		// of it.
		iconBytes, contentType, err := downloadIcon(ds.client, candidate.url)
		if err == nil {
			return iconBytes, contentType, nil
		}

		errs = append(errs, err)
	}

	return nil, "", errors.Join(errs...)
}

// iconCandidate is one icon a page declares, or the well-known path we try when
// it declares none that works.
type iconCandidate struct {
	url  string
	rel  string
	size int // width in pixels, 0 when the page did not say
	// scalable marks an SVG, which fits any size and is usually the smallest
	// thing on offer.
	scalable bool
}

// discover returns the icons the site declares, best first, always ending with
// the well-known /favicon.ico.
//
// A page that cannot be fetched or parsed is not fatal: /favicon.ico alone
// covers a fair number of sites, and costs one request.
func (ds *directSource) discover(site *url.URL, deadline time.Time) []iconCandidate {
	var candidates []iconCandidate

	// A page that declares no icon but points elsewhere with a meta refresh
	// (rather than an HTTP redirect, which fetchPage already follows) is
	// common enough for homepages that just select a language: the icon
	// lives on the page it points to, not on the stub itself.
	current := site
	for hop := 0; hop <= maxMetaRefreshHops; hop++ {
		if hop > 0 && time.Now().After(deadline) {
			break
		}

		body, base, err := ds.fetchPage(current)
		if err != nil {
			break
		}

		found, refresh := parseIconLinks(body, base)
		body.Close()

		if len(found) > 0 {
			candidates = found
			break
		}

		if refresh == nil || !sameOrganizationalDomain(site, refresh) {
			break
		}

		current = refresh
	}

	rankCandidates(candidates)

	// The well-known path keeps the last slot rather than competing for one: a
	// page declaring four icons would otherwise use up the budget and never
	// let us try the one place every site used to put it.
	candidates = dedupCandidates(candidates, maxCandidates-1)

	wellKnownURL := *site
	wellKnownURL.Path = "/favicon.ico"
	wellKnownURL.RawQuery = ""
	wellKnownURL.Fragment = ""

	wellKnown := iconCandidate{url: wellKnownURL.String(), rel: "well-known"}

	if !slices.ContainsFunc(candidates, func(c iconCandidate) bool { return c.url == wellKnown.url }) {
		candidates = append(candidates, wellKnown)
	}

	return candidates
}

func (ds *directSource) fetchPage(site *url.URL) (io.ReadCloser, *url.URL, error) {
	req, err := http.NewRequest(http.MethodGet, site.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	// A copy so the redirect policy below applies only to this request: the
	// shared client keeps netguard's CheckRedirect for downloadIcon and every
	// other caller. Sharing the Transport, and with it DialContext, keeps
	// every hop guarded against a private or link-local address regardless of
	// which domain it names.
	client := *ds.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxPageRedirects {
			return errors.New("too many redirects")
		}

		// A site may freely redirect within itself, http to https, apex to
		// www, a country path, and so on. A redirect that leaves the
		// organizational domain is a different site: following it would read
		// icon links out of a page the caller never asked about, and could
		// leak which domains this instance holds to whoever operates it.
		if !sameOrganizationalDomain(site, req.URL) {
			return fmt.Errorf("redirected outside %s's domain to %s", site.Hostname(), req.URL)
		}

		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, site)
	}

	// Relative icon links resolve against where the page ended up, not against
	// where we asked for it: a site redirecting to www. or to a country path
	// would otherwise have every relative icon pointing at the wrong host.
	base := resp.Request.URL

	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(resp.Body, maxPageSize), resp.Body}, base, nil
}

// sameOrganizationalDomain reports whether a and b's hosts belong to the same
// organization: example.org and www.example.org do, example.org and
// example.net, or example.org and evil.example.org.attacker.net, do not.
//
// A host that is not a registrable domain (an IP literal, a single-label
// name, an unlisted TLD) has no organizational domain to compare, so it only
// matches itself.
func sameOrganizationalDomain(a, b *url.URL) bool {
	ah, bh := strings.ToLower(a.Hostname()), strings.ToLower(b.Hostname())
	if ah == bh {
		return true
	}

	aOrg, aErr := publicsuffix.EffectiveTLDPlusOne(ah)
	bOrg, bErr := publicsuffix.EffectiveTLDPlusOne(bh)
	if aErr != nil || bErr != nil {
		return false
	}

	return aOrg == bOrg
}

// parseIconLinks reads the icon links of a document. It stops at the end of
// the head, since everything it looks for lives there. It also reports a
// <meta http-equiv="refresh"> target, for the caller to follow when the page
// declares no icon of its own: a stub page that only redirects has nothing
// else worth looking at.
func parseIconLinks(r io.Reader, base *url.URL) ([]iconCandidate, *url.URL) {
	var candidates []iconCandidate
	var metaRefresh *url.URL

	tokenizer := html.NewTokenizer(r)

	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return candidates, metaRefresh

		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := tokenizer.TagName()

			switch string(name) {
			case "base":
				if hasAttr {
					if href := attrs(tokenizer)["href"]; href != "" {
						if u, err := base.Parse(href); err == nil {
							base = u
						}
					}
				}

			case "link":
				if hasAttr {
					if candidate, ok := linkCandidate(attrs(tokenizer), base); ok {
						candidates = append(candidates, candidate)
					}
				}

			case "meta":
				if hasAttr && metaRefresh == nil {
					a := attrs(tokenizer)
					if strings.EqualFold(strings.TrimSpace(a["http-equiv"]), "refresh") {
						if target, ok := parseMetaRefresh(a["content"], base); ok {
							metaRefresh = target
						}
					}
				}

			case "body":
				return candidates, metaRefresh
			}

		case html.EndTagToken:
			if name, _ := tokenizer.TagName(); string(name) == "head" {
				return candidates, metaRefresh
			}
		}
	}
}

// parseMetaRefresh reads a meta refresh's content attribute, "0;
// url=https://example.com/en/" or "5; URL='/en/'", and resolves the URL part
// against base. The delay before it is unused: a page that redirects at all
// is treated the same whether it does so immediately or after a pause.
func parseMetaRefresh(content string, base *url.URL) (*url.URL, bool) {
	_, rest, ok := strings.Cut(content, ";")
	if !ok {
		return nil, false
	}

	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(strings.ToLower(rest), "url=") {
		return nil, false
	}

	href := strings.Trim(rest[len("url="):], `'" `)
	if href == "" {
		return nil, false
	}

	target, err := base.Parse(href)
	if err != nil {
		return nil, false
	}

	return target, true
}

func attrs(tokenizer *html.Tokenizer) map[string]string {
	values := map[string]string{}

	for {
		key, value, more := tokenizer.TagAttr()
		values[strings.ToLower(string(key))] = string(value)

		if !more {
			return values
		}
	}
}

func linkCandidate(attributes map[string]string, base *url.URL) (iconCandidate, bool) {
	rel := strings.ToLower(strings.Join(strings.Fields(attributes["rel"]), " "))
	if _, ok := iconRels[rel]; !ok {
		return iconCandidate{}, false
	}

	href := strings.TrimSpace(attributes["href"])
	if href == "" {
		return iconCandidate{}, false
	}

	target, err := base.Parse(href)
	if err != nil {
		return iconCandidate{}, false
	}

	mime := strings.ToLower(strings.TrimSpace(attributes["type"]))
	if mime == "" {
		mime = mimeFromPath(target.Path)
	}

	sizes := strings.ToLower(strings.TrimSpace(attributes["sizes"]))

	return iconCandidate{
		url:      target.String(),
		rel:      rel,
		size:     bestDeclaredSize(sizes),
		scalable: strings.Contains(mime, "svg") || sizes == "any" || strings.HasSuffix(strings.ToLower(target.Path), ".svg"),
	}, true
}

func mimeFromPath(p string) string {
	switch strings.ToLower(path.Ext(p)) {
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	}

	return ""
}

// bestDeclaredSize reads a sizes attribute ("32x32", "16x16 32x32 48x48") and
// keeps the width closest to what we display.
func bestDeclaredSize(sizes string) int {
	best := 0

	for _, size := range strings.Fields(sizes) {
		width, _, found := strings.Cut(size, "x")
		if !found {
			continue
		}

		value, err := strconv.Atoi(width)
		if err != nil || value <= 0 {
			continue
		}

		if best == 0 || distanceToTarget(value) < distanceToTarget(best) {
			best = value
		}
	}

	return best
}

// distanceToTarget scores how much a given size will hurt once drawn at
// targetSize. The two directions are not symmetric: an oversized icon is
// simply downscaled, which is close to free, while an undersized one is
// upscaled and comes out visibly soft. A 180px apple-touch-icon serves a
// crisp 64px icon; a 32px favicon.ico does not.
func distanceToTarget(size int) int {
	if size >= targetSize {
		// A gentle tie-breaker among oversized candidates: still prefer the
		// one closest to target (less to downscale, less to transfer), but
		// never let it outweigh the cost of an undersized one.
		return (size - targetSize) / 4
	}

	return 4 * (targetSize - size)
}

// rankCandidates sorts the icons a page declares, best first.
func rankCandidates(candidates []iconCandidate) {
	// Stable, so that a page declaring several equivalent icons keeps them in
	// the order it listed them.
	slices.SortStableFunc(candidates, func(a, b iconCandidate) int {
		return candidateScore(a) - candidateScore(b)
	})
}

// undeclaredSize is the width assumed for a candidate whose page declares no
// sizes attribute, per relation. apple-touch-icon rarely bothers declaring
// one because Apple's own convention is a single large master image
// (traditionally up to 180x180); icon and shortcut icon left unsized are, by
// the same convention, the historical 16x16 or 32x32 favicon.ico. Getting
// this guess right matters: it is what let a real, undersized favicon.ico
// outscore a real, oversized apple-touch-icon before either declared a size.
var undeclaredSize = map[string]int{
	"apple-touch-icon":             180,
	"apple-touch-icon-precomposed": 180,
}

// candidateScore is what decides which icon gets served. Lower is better.
func candidateScore(candidate iconCandidate) int {
	score := iconRels[candidate.rel]

	switch {
	case candidate.scalable:
		// One drawing for every size, and usually the lightest file.
		score -= 5

	case candidate.size > 0:
		score += distanceToTarget(candidate.size)

	default:
		size, ok := undeclaredSize[candidate.rel]
		if !ok {
			size = 32
		}
		score += distanceToTarget(size)
	}

	return score
}

func dedupCandidates(candidates []iconCandidate, limit int) []iconCandidate {
	seen := map[string]bool{}
	kept := make([]iconCandidate, 0, limit)

	for _, candidate := range candidates {
		if seen[candidate.url] {
			continue
		}
		seen[candidate.url] = true

		kept = append(kept, candidate)
		if len(kept) == limit {
			break
		}
	}

	return kept
}
