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

package helpers

import (
	"context"

	"github.com/miekg/dns"
)

// QueryOptions tunes a single DNS question.
type QueryOptions struct {
	// Recursive sets the RD bit. It is left unset for questions sent
	// straight to the authoritative servers of a zone.
	Recursive bool
}

// Question asks server for the records of type qtype at name. The
// question carries an EDNS0 OPT record with the DO bit, so DNSSEC records
// come along with the answer.
func Question(ctx context.Context, client *dns.Client, server, name string, qtype uint16, opts QueryOptions) (*dns.Msg, error) {
	m := new(dns.Msg)
	m.SetQuestion(name, qtype)
	m.RecursionDesired = opts.Recursive
	m.SetEdns0(4096, true)

	r, _, err := client.ExchangeContext(ctx, m, server)

	return r, err
}

// MultiQuestion asks server one question per type in qtypes and merges
// the answers into a single response, whose question type is set to ANY.
// It stands in for ANY queries, that most servers refuse (RFC 8482).
func MultiQuestion(ctx context.Context, client *dns.Client, server, name string, qtypes []uint16, opts QueryOptions) (r *dns.Msg, err error) {
	var response *dns.Msg

	for _, qtype := range qtypes {
		response, err = Question(ctx, client, server, name, qtype, opts)
		if err != nil {
			return
		}

		if len(response.Answer) > 0 {
			if r == nil {
				r = response
				r.Question[0].Qtype = dns.TypeANY
			} else {
				r.Answer = append(r.Answer, response.Answer...)
			}
		}
	}

	if r == nil && response != nil {
		r = response
		r.Question[0].Qtype = dns.TypeANY
	}

	return
}
