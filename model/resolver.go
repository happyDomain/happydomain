// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package happydns

import (
	"github.com/miekg/dns"
)

// ResolverRequest holds the resolution parameters
type ResolverRequest struct {
	// Resolver is the name of the resolver to use (or local or custom).
	Resolver string `json:"resolver"`

	// Custom is the address to the recursive server to use.
	Custom string `json:"custom,omitempty"`

	// DomainName is the FQDN to resolve.
	DomainName string `json:"domain"`

	// Type is the type of record to retrieve.
	Type string `json:"type"`
}

type DNSQuestion struct {
	// Name is the domain name researched.
	Name string

	// Qtype is the type of record researched.
	Qtype uint16

	// Qclass is the class of record researched.
	Qclass uint16
}

// DNSMsg is the documentation struct corresponding to dns.Msg
type DNSMsg struct {
	// Question is the Question section of the DNS response.
	Question []DNSQuestion

	// Answer is the list of Answer records in the DNS response.
	Answer []any `swaggertype:"object"`

	// Ns is the list of Authoritative records in the DNS response.
	Ns []any `swaggertype:"object"`

	// Extra is the list of extra records in the DNS response.
	Extra []any `swaggertype:"object"`
}

type ResolverUsecase interface {
	ResolveQuestion(ResolverRequest) (*dns.Msg, error)
}
