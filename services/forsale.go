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

package svcs

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
)

// RFC 10023 publishes "for sale" statements as TXT records at a _for-sale
// leaf node. Each record carries the version tag and at most one tag-value
// pair; the whole RRset forms a single statement, hence a single Service.
const (
	ForSaleLabel   = "_for-sale"
	ForSaleVersion = "v=FORSALE1;"

	// ForSaleMaxValueLen is the maximum length, in octets, of a fcod or ftxt value.
	ForSaleMaxValueLen = 239
)

// ForSale advertises that the domain name is for sale (RFC 10023).
type ForSale struct {
	Records []*happydns.TXT `json:"txt"`
}

func (s *ForSale) GetNbResources() int {
	return len(s.Records)
}

func (s *ForSale) GenComment() string {
	t := ForSaleFields{}
	for _, rr := range s.Records {
		if rr == nil {
			continue
		}
		t.Analyze(rr.Txt)
	}

	if len(t.Prices) > 0 {
		var prices []string
		for _, p := range t.Prices {
			prices = append(prices, p.String())
		}
		return strings.Join(prices, ", ")
	}

	if len(t.Texts) > 0 {
		return t.Texts[0]
	}

	if len(t.URIs) > 0 {
		return t.URIs[0]
	}

	return "Domain for sale"
}

func (s *ForSale) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, err error) {
	for _, rr := range s.Records {
		rrs = append(rrs, rr)
	}

	return
}

// Initialize seeds a new service with a bare version record, so the _for-sale
// node exists even before the user fills in any detail.
func (s *ForSale) Initialize() (any, error) {
	return &ForSale{
		Records: []*happydns.TXT{
			{
				Hdr: dns.RR_Header{
					Name:   ForSaleLabel,
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
				},
				Txt: ForSaleVersion,
			},
		},
	}, nil
}

// ForSalePrice is the currency/amount couple carried by a fval tag.
type ForSalePrice struct {
	Currency string
	Amount   string
}

func (p ForSalePrice) String() string {
	return p.Currency + " " + p.Amount
}

// ParseForSalePrice splits a fval value, eg. "USD750.50", into its currency
// and amount parts. The currency is 1*(A-Z) and the amount is 1*DIGIT
// optionally followed by a fractional part.
func ParseForSalePrice(value string) (price ForSalePrice, err error) {
	i := 0
	for i < len(value) && value[i] >= 'A' && value[i] <= 'Z' {
		i++
	}

	if i == 0 {
		return price, fmt.Errorf("not a valid for-sale price: missing currency in %q", value)
	}

	price.Currency = value[:i]
	price.Amount = value[i:]

	if price.Amount == "" {
		return price, fmt.Errorf("not a valid for-sale price: missing amount in %q", value)
	}

	frac := false
	for j := 0; j < len(price.Amount); j++ {
		c := price.Amount[j]
		if c == '.' {
			if frac || j == 0 || j == len(price.Amount)-1 {
				return price, fmt.Errorf("not a valid for-sale price: misplaced decimal separator in %q", value)
			}
			frac = true
		} else if c < '0' || c > '9' {
			return price, fmt.Errorf("not a valid for-sale price: unexpected character %q in %q", c, value)
		}
	}

	return price, nil
}

// ParseForSalePair extracts the tag and the value of a for-sale record, eg.
// "v=FORSALE1;ftxt=Call for info." yields ("ftxt", "Call for info.").
// A record limited to the version tag yields two empty strings.
//
// As permitted by RFC 10023 section 3.6, a single space following the version
// tag is tolerated.
func ParseForSalePair(txt string) (tag, value string, err error) {
	content, ok := strings.CutPrefix(strings.TrimSpace(txt), ForSaleVersion)
	if !ok {
		return "", "", fmt.Errorf("not a valid for-sale record: should begin with %s, seen %q", ForSaleVersion, txt)
	}

	content = strings.TrimPrefix(content, " ")
	if content == "" {
		return "", "", nil
	}

	tag, value, ok = strings.Cut(content, "=")
	if !ok {
		return "", "", fmt.Errorf("not a valid for-sale record: content %q is not a tag-value pair", content)
	}

	return tag, value, nil
}

// ForSaleFields is the analyzed view of a whole _for-sale RRset.
type ForSaleFields struct {
	Codes   []string
	Texts   []string
	URIs    []string
	Prices  []ForSalePrice
	Unknown []string
}

// Analyze dispatches one for-sale record into the matching field.
func (analyzed *ForSaleFields) Analyze(txt string) error {
	tag, value, err := ParseForSalePair(txt)
	if err != nil {
		return err
	}

	switch tag {
	case "":
		// Version tag alone: the domain is for sale, without any detail.
	case "fcod":
		analyzed.Codes = append(analyzed.Codes, value)
	case "ftxt":
		analyzed.Texts = append(analyzed.Texts, value)
	case "furi":
		analyzed.URIs = append(analyzed.URIs, value)
	case "fval":
		price, err := ParseForSalePrice(value)
		if err != nil {
			return err
		}
		analyzed.Prices = append(analyzed.Prices, price)
	default:
		// RFC 10023 section 2.2.5: unrecognized tags are ignored, the domain
		// remains for sale.
		analyzed.Unknown = append(analyzed.Unknown, tag+"="+value)
	}

	return nil
}

func forsale_analyze(a *svc.Analyzer) (err error) {
	pool := map[string]*ForSale{}

	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: ForSaleLabel + "."}) {
		txt, ok := record.(*happydns.TXT)

		// rfc10023: 2.4. records without a valid version tag are ignored
		if !ok || !strings.HasPrefix(strings.TrimSpace(txt.Txt), ForSaleVersion) {
			continue
		}

		domain := strings.TrimPrefix(record.Header().Name, ForSaleLabel+".")

		if _, ok := pool[domain]; !ok {
			pool[domain] = &ForSale{}
		}

		pool[domain].Records = append(
			pool[domain].Records,
			helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*happydns.TXT),
		)

		err = a.UseRR(record, domain, pool[domain])
		if err != nil {
			return
		}
	}

	return
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &ForSale{}
		},
		forsale_analyze,
		happydns.ServiceInfos{
			Name:        "Domain For Sale",
			Description: "Advertise that this domain name is for sale (RFC 10023): asking price, free-form message, contact URI and/or broker-specific code.",
			Categories: []string{
				"service",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
