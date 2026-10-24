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

package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// axfrTimeout bounds each step of a zone transfer: the connection, then
// every read.
const axfrTimeout = 5 * time.Second

// maxAXFRRecords caps the records kept from a transfer: the servers are
// chosen by whoever runs the domain, and could send an endless zone.
var maxAXFRRecords = 100000

func init() {
	RegisterSource("axfr", 10, func(s *Settings) NameSource {
		if s.DisableAXFR {
			return nil
		}
		return AXFRSource{}
	})
}

// AXFRSettings are the settings of the zone transfer technique.
type AXFRSettings struct {
	DisableAXFR bool `json:"disable_axfr,omitempty" happydomain:"label=Do not try a zone transfer,description=By default a zone transfer (AXFR) is tried first: few servers allow it but those that do hand the whole zone."`
}

// AXFRSource asks the authoritative servers for a zone transfer. Most
// servers refuse it to strangers, but those that allow it hand the whole
// zone, sparing the exploration.
type AXFRSource struct{}

func (AXFRSource) ID() string {
	return "axfr"
}

// Discover yields no name: the transfer hands records, see Records.
func (AXFRSource) Discover(ctx context.Context, t Target) ([]Name, error) {
	return nil, nil
}

// Records tries a transfer from each authoritative server in turn, and
// returns the records of the first that succeeds. Servers refusing it, or
// not even accepting the connection, are not an error: that is what they
// usually do.
func (AXFRSource) Records(ctx context.Context, t Target) ([]dns.RR, error) {
	progress := happydns.ImportProgressFrom(ctx)
	for i, server := range t.Authoritative {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if rrs, err := transfer(ctx, t.Domain, server); err == nil {
			return rrs, nil
		}
		progress.Advance(float64(i+1) / float64(len(t.Authoritative)))
	}

	return nil, nil
}

// transfer runs one zone transfer from server, an already vetted
// "ip:port".
func transfer(ctx context.Context, domain, server string) ([]dns.RR, error) {
	dialer := net.Dialer{Timeout: axfrTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", server)
	if err != nil {
		return nil, fmt.Errorf("zone transfer from %s: %w", server, err)
	}
	// The read timeout is renewed at each message: only the context bounds
	// a server dripping records.
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	tr := &dns.Transfer{
		Conn:         &dns.Conn{Conn: conn},
		ReadTimeout:  axfrTimeout,
		WriteTimeout: axfrTimeout,
	}

	m := new(dns.Msg)
	m.SetAxfr(domain)

	envelopes, err := tr.In(m, server)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("zone transfer from %s: %w", server, err)
	}

	var rrs []dns.RR
	for envelope := range envelopes {
		if envelope.Error != nil {
			return nil, fmt.Errorf("zone transfer from %s: %w", server, envelope.Error)
		}
		rrs = append(rrs, envelope.RR...)
		if len(rrs) > maxAXFRRecords {
			// Unblock the reader, which waits for us to take its envelopes.
			conn.Close()
			for range envelopes {
			}
			return nil, fmt.Errorf("zone transfer from %s: more than %d records", server, maxAXFRRecords)
		}
	}

	return rrs, nil
}
