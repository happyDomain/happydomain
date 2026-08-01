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
	"encoding/json"
	"fmt"

	"github.com/miekg/dns"
)

// UnmarshalRecord rebuilds the record a service stored, dispatching on the
// record type its header carries.
//
// The pseudo-types happyDomain represents need a hand: they are *dns.PrivateRR,
// whose rdata sits behind a non-empty interface, and encoding/json refuses to
// unmarshal into one even when it already holds a concrete value.
func UnmarshalRecord(payload []byte) (dns.RR, error) {
	var header struct {
		Hdr dns.RR_Header
	}

	if err := json.Unmarshal(payload, &header); err != nil {
		return nil, err
	}

	newrr, ok := dns.TypeToRR[header.Hdr.Rrtype]
	if !ok {
		return nil, fmt.Errorf("unknown rr type %d", header.Hdr.Rrtype)
	}

	rr := newrr()

	prr, isPrivate := rr.(*dns.PrivateRR)
	if !isPrivate {
		return rr, json.Unmarshal(payload, rr)
	}

	var private struct {
		Hdr  dns.RR_Header
		Data json.RawMessage
	}

	if err := json.Unmarshal(payload, &private); err != nil {
		return nil, err
	}

	prr.Hdr = private.Hdr
	if private.Data != nil {
		if err := json.Unmarshal(private.Data, prr.Data); err != nil {
			return nil, err
		}
	}

	return prr, nil
}
