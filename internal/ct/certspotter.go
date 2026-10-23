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
	"net/http"
	"net/url"
	"strings"
	"time"
)

var certSpotter = Backend{
	ID:          "certspotter",
	Name:        "Cert Spotter",
	TakesAPIKey: true,
	New: func(cfg Config) Searcher {
		return CertSpotter{cfg}
	},
}

// certSpotterMaxPages caps the pages of issuances fetched by one search: past
// it, the search returns what it has got, without error.
const certSpotterMaxPages = 50

// CertSpotter searches the issuances API of Cert Spotter, by SSLMate:
// https://sslmate.com/help/reference/ct_search_api_v1
//
// It only knows the certificates not expired yet. It answers a few queries
// an hour without APIKey, more with one.
type CertSpotter struct {
	Config
}

type certSpotterIssuance struct {
	ID         string    `json:"id"`
	CertSHA256 string    `json:"cert_sha256"`
	DNSNames   []string  `json:"dns_names"`
	NotBefore  time.Time `json:"not_before"`
	NotAfter   time.Time `json:"not_after"`
	Issuer     struct {
		FriendlyName string `json:"friendly_name"`
		Name         string `json:"name"`
	} `json:"issuer"`
}

func (s CertSpotter) Search(ctx context.Context, q Query) (issuances []Issuance, err error) {
	params := url.Values{}
	params.Set("domain", strings.TrimSuffix(q.Domain, "."))
	if q.IncludeSubdomains {
		params.Set("include_subdomains", "true")
	}
	params.Add("expand", "dns_names")
	params.Add("expand", "issuer")

	header := http.Header{}
	if s.APIKey != "" {
		header.Set("Authorization", "Bearer "+s.APIKey)
	}

	// The issuances come in pages: the next one starts after the last ID
	// of the previous one, until a page comes empty.
	for page := 0; page < certSpotterMaxPages; page++ {
		body, err := s.get(ctx, s.baseURL("https://api.certspotter.com")+"/v1/issuances?"+params.Encode(), header)
		if err != nil {
			return issuances, err
		}

		var raw []certSpotterIssuance
		if err := json.Unmarshal(body, &raw); err != nil {
			return issuances, fmt.Errorf("unable to read the answer of Cert Spotter: %w", err)
		}
		if len(raw) == 0 {
			return issuances, nil
		}

		for _, r := range raw {
			issuer := r.Issuer.FriendlyName
			if issuer == "" {
				issuer = r.Issuer.Name
			}

			issuances = append(issuances, Issuance{
				ID:         r.ID,
				DNSNames:   r.DNSNames,
				Issuer:     issuer,
				NotBefore:  r.NotBefore,
				NotAfter:   r.NotAfter,
				CertSHA256: r.CertSHA256,
			})
		}

		// Without a new ID to start after, the same page would come again.
		last := raw[len(raw)-1].ID
		if last == "" || last == params.Get("after") {
			return issuances, nil
		}
		params.Set("after", last)
	}

	return issuances, nil
}
