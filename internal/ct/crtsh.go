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

package ct

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var crtSh = Backend{
	ID:   "crtsh",
	Name: "crt.sh",
	New: func(cfg Config) Searcher {
		return CrtSh{cfg}
	},
}

// crtShTimeLayout is the layout of the dates crt.sh returns, in UTC.
const crtShTimeLayout = "2006-01-02T15:04:05"

// CrtSh searches crt.sh, by Sectigo: https://crt.sh/
//
// It needs no API key, but is often slow, or failing, on the domains
// holding many certificates.
type CrtSh struct {
	Config
}

type crtShEntry struct {
	ID           int64  `json:"id"`
	IssuerName   string `json:"issuer_name"`
	NameValue    string `json:"name_value"`
	NotBefore    string `json:"not_before"`
	NotAfter     string `json:"not_after"`
	SerialNumber string `json:"serial_number"`
}

func (s CrtSh) Search(ctx context.Context, q Query) ([]Issuance, error) {
	domain := strings.TrimSuffix(q.Domain, ".")
	// crt.sh reads the query as a LIKE pattern.
	if strings.ContainsAny(domain, "%_") {
		return nil, fmt.Errorf("crt.sh cannot search %q: it would read it as a pattern", domain)
	}

	params := url.Values{}
	if q.IncludeSubdomains {
		// % matches one label or more: the certificates naming the apex
		// alone are left out, sparing a second query to a slow service.
		params.Set("q", "%."+domain)
	} else {
		params.Set("q", domain)
	}
	params.Set("output", "json")
	params.Set("deduplicate", "Y")
	if q.ExcludeExpired {
		params.Set("exclude", "expired")
	}

	body, err := s.get(ctx, s.baseURL("https://crt.sh")+"/?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var entries []crtShEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("unable to read the answer of crt.sh: %w", err)
	}

	issuances := make([]Issuance, 0, len(entries))
	for _, e := range entries {
		// A failed parse leaves the zero time: the names matter more.
		notBefore, _ := time.Parse(crtShTimeLayout, e.NotBefore)
		notAfter, _ := time.Parse(crtShTimeLayout, e.NotAfter)

		issuances = append(issuances, Issuance{
			ID:        strconv.FormatInt(e.ID, 10),
			DNSNames:  strings.Fields(e.NameValue),
			Issuer:    e.IssuerName,
			Serial:    e.SerialNumber,
			NotBefore: notBefore,
			NotAfter:  notAfter,
		})
	}

	return issuances, nil
}
