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
	"crypto/sha256"
	"strings"

	dnscontrol "github.com/StackExchange/dnscontrol/v4/models"
	"github.com/StackExchange/dnscontrol/v4/pkg/diff2"
	"github.com/StackExchange/dnscontrol/v4/pkg/domaintags"
	"github.com/StackExchange/dnscontrol/v4/pkg/dnsrr"
	_ "github.com/StackExchange/dnscontrol/v4/pkg/rtype"
	"github.com/StackExchange/dnscontrol/v4/pkg/rtypecontrol"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// DNSControlFromCorrectionType converts a DNSControl diff verb (CREATE, CHANGE, DELETE, REPORT)
// to a happyDomain correction kind (Addition, Update, Deletion, Other).
func DNSControlFromCorrectionType(in diff2.Verb) (kind happydns.CorrectionKind) {
	switch in {
	case diff2.CREATE:
		kind = happydns.CorrectionKindAddition
	case diff2.CHANGE:
		kind = happydns.CorrectionKindUpdate
	case diff2.DELETE:
		kind = happydns.CorrectionKindDeletion
	case diff2.REPORT:
		kind = happydns.CorrectionKindOther
	}

	return
}

// DNSControlCorrectionKindFromMessage parses a DNSControl correction message to determine
// the type of change. It looks for prefixes like "+ CREATE", "± MODIFY", "- DELETE" in the message.
// Returns the corresponding correction kind (Addition, Update, Deletion, Other).
func DNSControlCorrectionKindFromMessage(msg string) (kind happydns.CorrectionKind) {
	if strings.HasPrefix(msg, "+ CREATE") {
		kind = happydns.CorrectionKindAddition
	} else if strings.HasPrefix(msg, "± MODIFY") {
		kind = happydns.CorrectionKindUpdate
	} else if strings.HasPrefix(msg, "- DELETE") {
		kind = happydns.CorrectionKindDeletion
	} else {
		kind = happydns.CorrectionKindOther
	}

	return
}

// DNSControlDiffByRecord computes the differences between two sets of DNS records using DNSControl's
// diff engine. It converts happyDomain records to DNSControl format, computes the diff, and returns
// the corrections needed to transform oldrrs into newrrs.
// Returns a slice of corrections, the total number of corrections, and any error.
func DNSControlDiffByRecord(oldrrs []happydns.Record, newrrs []happydns.Record, origin string) ([]*happydns.Correction, int, error) {
	oldrecords, err := DNSControlRRtoRC(oldrrs, origin)
	if err != nil {
		return nil, 0, err
	}

	newdc, err := NewDNSControlDomainConfig(origin, newrrs)
	if err != nil {
		return nil, 0, err
	}

	corrections, nbCorrections, err := diff2.ByRecord(oldrecords, newdc, nil)
	if err != nil {
		return nil, nbCorrections, err
	}

	ret := make([]*happydns.Correction, len(corrections))
	for i, correction := range corrections {
		id := sha256.Sum224([]byte(correction.MsgsJoined))

		var oldRecords []happydns.Record
		for _, rc := range correction.Old {
			oldRecords = append(oldRecords, rc.ToRR())
		}

		var newRecords []happydns.Record
		for _, rc := range correction.New {
			newRecords = append(newRecords, rc.ToRR())
		}

		ret[i] = &happydns.Correction{
			Id:         id[:],
			Msg:        correction.MsgsJoined,
			Kind:       DNSControlFromCorrectionType(correction.Type),
			OldRecords: oldRecords,
			NewRecords: newRecords,
		}
	}

	return ret, nbCorrections, nil
}

// DNSControlRRtoRC converts a slice of happyDomain records to DNSControl's RecordConfig format.
// It handles conversion of custom record types (like happydns.TXT, happydns.SPF) to standard dns.RR
// before converting to DNSControl format.
// The origin parameter specifies the zone name (with or without trailing dot).
func DNSControlRRtoRC(rrs []happydns.Record, origin string) (dnscontrol.Records, error) {
	originNoTrailingDot := strings.TrimSuffix(origin, ".")
	records := make([]*dnscontrol.RecordConfig, len(rrs))

	for i, rr := range rrs {
		// Convert happydns.TXT, happydns.SPF, ... to corresponding dns.RR
		if record, ok := rr.(happydns.ConvertibleRecord); ok {
			rr = record.ToRR()
		}

		typeName := dns.TypeToString[rr.Header().Rrtype]

		var rc dnscontrol.RecordConfig
		var err error

		if _, ok := rtypecontrol.Func[typeName]; ok {
			dcn := domaintags.MakeDomainNameVarieties(originNoTrailingDot)
			rcPtr, e := rtypecontrol.NewRecordConfigFromStruct(rr.Header().Name, rr.Header().Ttl, typeName, rr, dcn)
			if e != nil {
				return nil, e
			}
			rc = *rcPtr
		} else {
			rc, err = dnsrr.RRtoRC(rr.(dns.RR), originNoTrailingDot)
			if err != nil {
				return nil, err
			}
		}

		records[i] = &rc
	}

	return records, nil
}

// NewDNSControlDomainConfig creates a DNSControl DomainConfig from happyDomain records.
// This is used to represent a desired zone state when computing corrections or validating records.
// The origin parameter specifies the zone name (with or without trailing dot).
func NewDNSControlDomainConfig(origin string, rrs []happydns.Record) (*dnscontrol.DomainConfig, error) {
	records, err := DNSControlRRtoRC(rrs, origin)

	return &dnscontrol.DomainConfig{
		Name:    strings.TrimSuffix(origin, "."),
		Records: records,
	}, err
}

// recordKey returns a canonical string for matching records by name, type, and rdata.
func recordKey(r happydns.Record) string {
	return r.String()
}

// BuildTargetRecords computes the target record set by applying the selected
// corrections to the provider's current records. It starts with a copy of
// providerRecords, then for each correction whose ID is in selectedIDs:
//   - Addition: appends NewRecords
//   - Deletion: removes matching OldRecords
//   - Update: removes matching OldRecords, appends NewRecords
func BuildTargetRecords(
	providerRecords []happydns.Record,
	corrections []*happydns.Correction,
	selectedIDs []happydns.Identifier,
) []happydns.Record {
	// Build a set of selected IDs for fast lookup.
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[string(id)] = true
	}

	// Start with a copy of provider records.
	result := make([]happydns.Record, len(providerRecords))
	copy(result, providerRecords)

	for _, cr := range corrections {
		if !selected[string(cr.Id)] {
			continue
		}

		switch cr.Kind {
		case happydns.CorrectionKindAddition:
			result = append(result, cr.NewRecords...)

		case happydns.CorrectionKindDeletion:
			result = removeRecords(result, cr.OldRecords)

		case happydns.CorrectionKindUpdate:
			result = removeRecords(result, cr.OldRecords)
			result = append(result, cr.NewRecords...)
		}
	}

	return result
}

// removeRecords removes records from the slice that match any of the toRemove
// records by their canonical string representation. Each toRemove record
// removes at most one match.
func removeRecords(records []happydns.Record, toRemove []happydns.Record) []happydns.Record {
	removeKeys := make(map[string]int, len(toRemove))
	for _, r := range toRemove {
		removeKeys[recordKey(r)]++
	}

	result := make([]happydns.Record, 0, len(records))
	for _, r := range records {
		key := recordKey(r)
		if removeKeys[key] > 0 {
			removeKeys[key]--
			continue
		}
		result = append(result, r)
	}
	return result
}
