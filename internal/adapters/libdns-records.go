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

package adapter

import (
	"fmt"
	"strings"
	"time"

	"github.com/StackExchange/dnscontrol/v4/pkg/txtutil"
	"github.com/libdns/libdns"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// libdnsToHappyDNSRecord converts a libdns Record to a happydns Record.
// The zone parameter should be the FQDN with trailing dot (e.g. "example.com.").
// For TXT records, it produces happydns.TXT directly (single concatenated string).
func libdnsToHappyDNSRecord(rec libdns.Record, zone string) (happydns.Record, error) {
	rr := rec.RR()

	fqdn := libdns.AbsoluteName(rr.Name, zone)
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}

	ttlSec := uint32(rr.TTL.Seconds())

	// For TXT records, the libdns Data field may be either raw text or
	// RFC1035 presentation-format with quotes and escaping (depends on provider).
	// Use txtutil.ParseQuoted to decode presentation-format data.
	if rr.Type == "TXT" {
		return &happydns.TXT{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttlSec,
			},
			Txt: decodeTXTData(rr.Data),
		}, nil
	}

	// For SPF records (if any provider returns them)
	if rr.Type == "SPF" {
		return &happydns.SPF{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypeSPF,
				Class:  dns.ClassINET,
				Ttl:    ttlSec,
			},
			Txt: decodeTXTData(rr.Data),
		}, nil
	}

	// For all other record types, build a zone-file line and parse it.
	line := fmt.Sprintf("%s %d IN %s %s", fqdn, ttlSec, rr.Type, rr.Data)
	return helpers.ParseRecord(line, zone)
}

// happyDNSRecordToLibdnsRR converts a happydns Record to a libdns RR.
// The zone parameter should be the FQDN with trailing dot (e.g. "example.com.").
func happyDNSRecordToLibdnsRR(record happydns.Record, zone string) libdns.RR {
	hdr := record.Header()

	name := libdns.RelativeName(hdr.Name, zone)
	typStr := dns.TypeToString[hdr.Rrtype]
	ttl := time.Duration(hdr.Ttl) * time.Second

	// For happydns.TXT / happydns.SPF, extract the raw text directly.
	if txt, ok := record.(*happydns.TXT); ok {
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: typStr,
			Data: txt.Txt,
		}
	}
	if spf, ok := record.(*happydns.SPF); ok {
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: typStr,
			Data: spf.Txt,
		}
	}

	// For ConvertibleRecord types, convert to dns.RR first.
	var dnsRR dns.RR
	if cr, ok := record.(happydns.ConvertibleRecord); ok {
		dnsRR = cr.ToRR()
	} else if rr, ok := record.(dns.RR); ok {
		dnsRR = rr
	} else {
		// Fallback: try to extract rdata from string representation.
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: typStr,
			Data: extractRdata(record.String(), typStr),
		}
	}

	return libdns.RR{
		Name: name,
		TTL:  ttl,
		Type: typStr,
		Data: extractRdata(dnsRR.String(), typStr),
	}
}

// decodeTXTData decodes TXT record data that may be in RFC1035 presentation
// format (quoted, with escaping) or raw text. Some libdns providers (e.g.
// PowerDNS) return quoted data like `"value"`, while others (e.g. libdns.TXT)
// return raw unquoted text. ParseQuoted handles quoted data correctly but
// treats unquoted spaces as separators, so we only use it when quotes are present.
func decodeTXTData(s string) string {
	if strings.ContainsRune(s, '"') {
		if decoded, err := txtutil.ParseQuoted(s); err == nil {
			return decoded
		}
	}
	return s
}

// extractRdata extracts the rdata portion from a miekg/dns RR string.
// The format is: "name.\t<TTL>\tIN\t<TYPE>\t<rdata...>"
func extractRdata(rrString string, rrType string) string {
	// miekg/dns uses tab-separated fields
	marker := "\tIN\t" + rrType + "\t"
	idx := strings.Index(rrString, marker)
	if idx != -1 {
		return rrString[idx+len(marker):]
	}

	// Fallback: try space-separated (shouldn't happen with miekg/dns)
	marker = " IN " + rrType + " "
	idx = strings.Index(rrString, marker)
	if idx != -1 {
		return rrString[idx+len(marker):]
	}

	return ""
}

// libdnsRecordsToHappyDNS converts a slice of libdns Records to happydns Records.
func libdnsRecordsToHappyDNS(recs []libdns.Record, zone string) ([]happydns.Record, error) {
	result := make([]happydns.Record, 0, len(recs))
	for _, rec := range recs {
		hdr, err := libdnsToHappyDNSRecord(rec, zone)
		if err != nil {
			return nil, fmt.Errorf("converting libdns record %v: %w", rec.RR(), err)
		}
		result = append(result, hdr)
	}
	return result, nil
}

// happyDNSRecordsToLibdns converts a slice of happydns Records to libdns RR values.
func happyDNSRecordsToLibdns(rrs []happydns.Record, zone string) []libdns.RR {
	result := make([]libdns.RR, len(rrs))
	for i, rr := range rrs {
		result[i] = happyDNSRecordToLibdnsRR(rr, zone)
	}
	return result
}
