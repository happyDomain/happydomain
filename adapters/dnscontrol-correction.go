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
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

func DNSControlDiffByRecord(oldrrs []happydns.Record, newrrs []happydns.Record, origin string) ([]*happydns.Correction, error) {
	oldrecords, err := DNSControlRRtoRC(oldrrs, origin)
	if err != nil {
		return nil, err
	}

	newdc, err := NewDNSControlDomainConfig(origin, newrrs)
	if err != nil {
		return nil, err
	}

	corrections, err := diff2.ByRecord(oldrecords, newdc, nil)
	if err != nil {
		return nil, err
	}

	ret := make([]*happydns.Correction, len(corrections))
	for i, correction := range corrections {
		var kind happydns.CorrectionKind

		// Convert Change Type to Correction Kind
		switch correction.Type {
		case diff2.CREATE:
			kind = happydns.CorrectionKindAddition
		case diff2.CHANGE:
			kind = happydns.CorrectionKindUpdate
		case diff2.DELETE:
			kind = happydns.CorrectionKindDeletion
		case diff2.REPORT:
			kind = happydns.CorrectionKindOther
		}

		ret[i] = &happydns.Correction{
			Msg:  correction.MsgsJoined,
			Kind: kind,
		}
	}

	return ret, nil
}

func DNSControlRRtoRC(rrs []happydns.Record, origin string) (dnscontrol.Records, error) {
	records := make([]*dnscontrol.RecordConfig, len(rrs))

	for i, rr := range rrs {
		rc, err := dnscontrol.RRtoRC(rr.(dns.RR), strings.TrimSuffix(origin, "."))
		if err != nil {
			return nil, err
		}
		records[i] = &rc
	}

	return records, nil
}

func NewDNSControlDomainConfig(origin string, rrs []happydns.Record) (*dnscontrol.DomainConfig, error) {
	records, err := DNSControlRRtoRC(rrs, origin)

	return &dnscontrol.DomainConfig{
		Name:    strings.TrimSuffix(origin, "."),
		Records: records,
	}, err
}
