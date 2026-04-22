// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

//go:build ignore
// +build ignore

package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

const (
	subjectColumn = "Subject"
	domainColumn  = "Recognized CAA Domains"
)

// row represents a single CCADB intermediate and the CAA domains it recognizes.
type row struct {
	owner   string
	domains []string
}

// score tracks how well a candidate owner matches a CAA domain across all
// rows that mention that domain.
type score struct {
	rowCount  int
	minSize   int
	nameMatch bool
}

func main() {
	output := flag.String("o", "", "path to write the generated JSON file")
	flag.Parse()

	if *output == "" {
		fatal("missing required -o flag")
	}
	if flag.NArg() < 1 {
		fatal("missing CCADB CSV URL (first positional argument)")
	}
	url := flag.Arg(0)

	rows, err := fetchAndParse(url)
	if err != nil {
		fatal(err.Error())
	}

	mapping := buildDomainToOwner(rows)

	if err := writeJSON(*output, mapping); err != nil {
		fatal(err.Error())
	}
}

func fetchAndParse(url string) ([]row, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	return parseCSV(resp.Body)
}

func parseCSV(r io.Reader) ([]row, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	subjectIdx := indexOf(header, subjectColumn)
	domainIdx := indexOf(header, domainColumn)
	if subjectIdx < 0 {
		return nil, fmt.Errorf("column %q not found in CSV header", subjectColumn)
	}
	if domainIdx < 0 {
		return nil, fmt.Errorf("column %q not found in CSV header", domainColumn)
	}

	var rows []row
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		if subjectIdx >= len(record) || domainIdx >= len(record) {
			continue
		}

		owner := extractOrganization(record[subjectIdx])
		if owner == "" {
			continue
		}

		domains := splitDomains(record[domainIdx])
		if len(domains) == 0 {
			continue
		}

		rows = append(rows, row{owner: owner, domains: domains})
	}
	return rows, nil
}

// buildDomainToOwner inverts the CCADB rows into a CAA-domain → owner mapping.
//
// For each CAA identifier, the "authoritative" owner is selected by:
//  1. Preferring owners whose name contains a significant label of the CAA
//     domain (e.g. "digicert.com" prefers "DigiCert, Inc." over cross-signed
//     subordinates that also list digicert.com in their recognized set).
//  2. Then highest row count: real root CAs have many intermediates all
//     recognizing their own identifier.
//  3. Then smallest minimum Recognized-CAA-Domains set: roots typically
//     recognize just their own identifier, while subordinates inherit larger
//     sets from cross-signing parents.
//  4. Alphabetical for determinism.
//
// Owner names are grouped case-insensitively to collapse CCADB casing
// inconsistencies (e.g. "Cloudflare, Inc." vs "CLOUDFLARE, INC."), preferring
// the variant with the fewest all-caps words.
func buildDomainToOwner(rows []row) map[string]string {
	canonical := canonicalOwners(rows)

	scores := map[string]map[string]*score{}

	for _, r := range rows {
		owner := canonical[strings.ToLower(r.owner)]
		size := len(r.domains)
		for _, dn := range r.domains {
			if _, ok := scores[dn]; !ok {
				scores[dn] = map[string]*score{}
			}
			s, ok := scores[dn][owner]
			if !ok {
				s = &score{minSize: size, nameMatch: ownerMatchesDomain(owner, dn)}
				scores[dn][owner] = s
			}
			s.rowCount++
			if size < s.minSize {
				s.minSize = size
			}
		}
	}

	out := make(map[string]string, len(scores))
	for dn, byOwner := range scores {
		var bestOwner string
		var best *score
		for owner, s := range byOwner {
			if best == nil || scoreBetter(s, owner, best, bestOwner) {
				best = s
				bestOwner = owner
			}
		}
		out[dn] = bestOwner
	}
	return out
}

func scoreBetter(a *score, ao string, b *score, bo string) bool {
	if a.nameMatch != b.nameMatch {
		return a.nameMatch
	}
	if a.rowCount != b.rowCount {
		return a.rowCount > b.rowCount
	}
	if a.minSize != b.minSize {
		return a.minSize < b.minSize
	}
	return ao < bo
}

// genericDomainLabels are labels that appear in CAA domains but don't identify
// a specific CA brand (e.g. "pki.goog" is Google, not "pki"). Anything shorter
// than 3 characters (TLDs) is also skipped.
var genericDomainLabels = map[string]bool{
	"com": true, "net": true, "org": true, "gov": true, "edu": true,
	"co": true,
	"pki": true, "tls": true, "ssl": true, "www": true, "eca": true,
	"publicca": true, "epki": true, "cert": true, "trust": true,
	"certificate": true, "ca": true,
}

// ownerMatchesDomain returns true if a significant label of the CAA domain
// appears as a substring of the owner name (alphanumeric-only, lowercased).
// Used to prefer self-referential owners (e.g. "DigiCert" for "digicert.com")
// over cross-signed subordinates that also list the domain.
func ownerMatchesDomain(owner, caaDomain string) bool {
	normName := alphaNumLower(owner)
	for _, label := range strings.Split(caaDomain, ".") {
		label = strings.ToLower(label)
		if len(label) < 3 || genericDomainLabels[label] {
			continue
		}
		if strings.Contains(normName, label) {
			return true
		}
	}
	return false
}

func alphaNumLower(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// canonicalOwners returns a map from lowercased owner name to the preferred
// display variant. Preference: fewer all-caps words, then lexicographically
// smallest (for determinism).
func canonicalOwners(rows []row) map[string]string {
	variants := map[string]map[string]struct{}{}
	for _, r := range rows {
		key := strings.ToLower(r.owner)
		if _, ok := variants[key]; !ok {
			variants[key] = map[string]struct{}{}
		}
		variants[key][r.owner] = struct{}{}
	}

	out := make(map[string]string, len(variants))
	for key, vs := range variants {
		picks := make([]string, 0, len(vs))
		for v := range vs {
			picks = append(picks, v)
		}
		sort.Slice(picks, func(i, j int) bool {
			ai, aj := allCapsWords(picks[i]), allCapsWords(picks[j])
			if ai != aj {
				return ai < aj
			}
			return picks[i] < picks[j]
		})
		out[key] = picks[0]
	}
	return out
}

// allCapsWords counts words (whitespace-delimited) that contain at least one
// letter and are entirely uppercase — a proxy for "ALL CAPS" shouting that we
// want to avoid when choosing a canonical display form.
func allCapsWords(s string) int {
	n := 0
	for _, w := range strings.Fields(s) {
		hasLetter := false
		allUpper := true
		for _, r := range w {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				hasLetter = true
				if r >= 'a' && r <= 'z' {
					allUpper = false
				}
			}
		}
		if hasLetter && allUpper {
			n++
		}
	}
	return n
}

func splitDomains(cell string) []string {
	var out []string
	for _, dn := range strings.FieldsFunc(cell, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}) {
		dn = strings.ToLower(strings.TrimSpace(dn))
		if isDomainName(dn) {
			out = append(out, dn)
		}
	}
	return out
}

// isDomainName filters free-form text leaking from the CCADB "Recognized CAA
// Domains" cell (tokens like "None", "N/A", "Comma-separated", "list.", or
// sentence fragments). Requires at least two non-empty DNS-like labels and a
// TLD of at least two letters.
func isDomainName(s string) bool {
	labels := strings.Split(s, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if l == "" || !isDomainLabel(l) {
			return false
		}
	}
	tld := labels[len(labels)-1]
	if len(tld) < 2 {
		return false
	}
	for _, r := range tld {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

func isDomainLabel(l string) bool {
	if l[0] == '-' || l[len(l)-1] == '-' {
		return false
	}
	for _, r := range l {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

func writeJSON(path string, data map[string]string) error {
	buf, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	buf = append(buf, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// extractOrganization returns the O= (organization) value from an RFC-4514-ish
// DN string as provided by CCADB (fields separated by "; ").
func extractOrganization(subject string) string {
	for _, field := range strings.Split(subject, "; ") {
		field = strings.TrimSpace(field)
		if v, ok := strings.CutPrefix(field, "O="); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func indexOf(header []string, name string) int {
	for i, h := range header {
		if strings.TrimSpace(h) == name {
			return i
		}
	}
	return -1
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "gen_caa_issuers: "+msg)
	os.Exit(1)
}
