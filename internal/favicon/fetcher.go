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

package favicon

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.deanishe.net/favicon"
)

type cacheEntry struct {
	bytes       []byte
	contentType string
	fetchedAt   time.Time
}

type FaviconService struct {
	cache  sync.Map
	client *http.Client
}

func NewFaviconService() *FaviconService {
	return &FaviconService{
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// Fetch retrieves the favicon for the given URL. It returns the icon bytes,
// content type, and any error. Results are cached with the given TTL.
func (fs *FaviconService) Fetch(rawURL string, ttl time.Duration) ([]byte, string, error) {
	if cached, ok := fs.cache.Load(rawURL); ok {
		entry := cached.(*cacheEntry)
		if time.Since(entry.fetchedAt) < ttl {
			return entry.bytes, entry.contentType, nil
		}
	}

	icons, err := favicon.Find(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("finding favicons for %s: %w", rawURL, err)
	}

	if len(icons) == 0 {
		return nil, "", fmt.Errorf("no favicon found for %s", rawURL)
	}

	icon := pickBestIcon(icons)

	iconBytes, contentType, err := fs.downloadIcon(icon.URL)
	if err != nil {
		return nil, "", fmt.Errorf("downloading favicon for %s: %w", rawURL, err)
	}

	fs.cache.Store(rawURL, &cacheEntry{
		bytes:       iconBytes,
		contentType: contentType,
		fetchedAt:   time.Now(),
	})

	return iconBytes, contentType, nil
}

// FetchDomain retrieves the favicon for a domain name, with SSRF protection.
func (fs *FaviconService) FetchDomain(domain string) ([]byte, string, error) {
	if err := validateDomain(domain); err != nil {
		return nil, "", err
	}

	return fs.Fetch("https://"+domain, 24*time.Hour)
}

func pickBestIcon(icons []*favicon.Icon) *favicon.Icon {
	best := icons[0]
	for _, icon := range icons[1:] {
		if isBetterIcon(icon, best) {
			best = icon
		}
	}
	return best
}

func isBetterIcon(a, b *favicon.Icon) bool {
	aMime := a.MimeType
	bMime := b.MimeType

	// Prefer SVG
	if strings.Contains(aMime, "svg") && !strings.Contains(bMime, "svg") {
		return true
	}
	if !strings.Contains(aMime, "svg") && strings.Contains(bMime, "svg") {
		return false
	}

	// Prefer PNG over other raster formats
	if strings.Contains(aMime, "png") && !strings.Contains(bMime, "png") {
		return true
	}
	if !strings.Contains(aMime, "png") && strings.Contains(bMime, "png") {
		return false
	}

	// Prefer larger icons
	return a.Width > b.Width
}

func (fs *FaviconService) downloadIcon(iconURL string) ([]byte, string, error) {
	resp, err := fs.client.Get(iconURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, iconURL)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	return body, contentType, nil
}

func validateDomain(domain string) error {
	if strings.ContainsAny(domain, "/:@") {
		return fmt.Errorf("invalid domain: %q", domain)
	}

	// Resolve to check for private IPs
	ips, err := net.LookupIP(domain)
	if err != nil {
		return fmt.Errorf("cannot resolve domain %q: %w", domain, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("domain %q resolves to private IP", domain)
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateURL checks that a URL is safe to fetch (no private IPs).
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	return validateDomain(u.Hostname())
}
