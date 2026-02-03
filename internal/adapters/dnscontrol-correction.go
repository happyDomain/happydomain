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
	"strings"

	dnscontrol "github.com/StackExchange/dnscontrol/v4/models"
	"github.com/StackExchange/dnscontrol/v4/pkg/diff2"
	"github.com/StackExchange/dnscontrol/v4/pkg/dnsrr"
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
		ret[i] = &happydns.Correction{
			Msg:  correction.MsgsJoined,
			Kind: DNSControlFromCorrectionType(correction.Type),
		}
	}

	return ret, nbCorrections, nil
}

// DNSControlRRtoRC converts a slice of happyDomain records to DNSControl's RecordConfig format.
// It handles conversion of custom record types (like happydns.TXT, happydns.SPF) to standard dns.RR
// before converting to DNSControl format.
// The origin parameter specifies the zone name (with or without trailing dot).
func DNSControlRRtoRC(rrs []happydns.Record, origin string) (dnscontrol.Records, error) {
	records := make([]*dnscontrol.RecordConfig, len(rrs))

	for i, rr := range rrs {
		// Convert happydns.TXT, happydns.SPF, ... to corresponding dns.RR
		if record, ok := rr.(happydns.ConvertibleRecord); ok {
			rr = record.ToRR()
		}

		rc, err := dnsrr.RRtoRC(rr.(dns.RR), strings.TrimSuffix(origin, "."))
		if err != nil {
			return nil, err
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
