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

// DNSQuestion holds a single DNS question entry.
type DNSQuestion struct {
	// Name is the domain name researched.
	Name string `json:"name"`

	// Qtype is the type of record researched.
	Qtype uint16 `json:"qtype"`

	// Qclass is the class of record researched.
	Qclass uint16 `json:"qclass"`
}

// ResolverResponse is the API response for a DNS resolution.
type ResolverResponse struct {
	// Question is the Question section of the DNS response.
	Question []DNSQuestion `json:"question"`

	// Answer is the list of Answer records in the DNS response.
	Answer []dns.RR `json:"answer" swaggertype:"object"`

	// Ns is the list of Authoritative records in the DNS response.
	Ns []dns.RR `json:"ns" swaggertype:"object"`

	// Extra is the list of extra records in the DNS response.
	Extra []dns.RR `json:"extra" swaggertype:"object"`
}

// NewResolverResponseFromMsg converts a dns.Msg to a ResolverResponse.
func NewResolverResponseFromMsg(msg *dns.Msg) *ResolverResponse {
	resp := &ResolverResponse{
		Answer: msg.Answer,
		Ns:     msg.Ns,
		Extra:  msg.Extra,
	}
	for _, q := range msg.Question {
		resp.Question = append(resp.Question, DNSQuestion{
			Name:   q.Name,
			Qtype:  q.Qtype,
			Qclass: q.Qclass,
		})
	}
	return resp
}

type ResolverUsecase interface {
	ResolveQuestion(ResolverRequest) (*dns.Msg, error)
	FlattenSPF(SPFFlattenRequest) (*SPFFlattenResponse, error)
}

// SPFFlattenRequest asks the backend to recursively walk an SPF record and
// count its DNS-lookup budget. When Record is set, the resolver evaluates it
// as the root record (useful while editing) instead of looking up the TXT
// record at Domain.
type SPFFlattenRequest struct {
	// Resolver is the name of the resolver to use (or local or custom).
	Resolver string `json:"resolver,omitempty"`

	// Custom is the address to the recursive server to use.
	Custom string `json:"custom,omitempty"`

	// Domain is the FQDN to resolve. It is also used as the apex when the
	// optional Record field is provided.
	Domain string `json:"domain"`

	// Record overrides the root SPF record being evaluated. When empty, the
	// resolver fetches Domain's TXT and looks for "v=spf1".
	Record string `json:"record,omitempty"`
}

// SPFNode represents a single mechanism or modifier consuming a DNS lookup
// while evaluating an SPF record.
type SPFNode struct {
	// Domain is the resolved domain for this node (the include / redirect
	// target, the qualified mechanism domain, or the parent domain when
	// implicit).
	Domain string `json:"domain"`

	// Mechanism is the raw term as it appears in the parent record
	// (e.g. "include:example.com", "redirect=foo.com", "a", "mx:host.com").
	Mechanism string `json:"mechanism"`

	// Record is the SPF record found at Domain when the node corresponds to
	// an include or redirect. Empty for a/mx/exists/ptr.
	Record string `json:"record,omitempty"`

	// LookupsHere counts the local cost of this node (always 1 for the
	// mechanisms tracked under RFC 7208 §4.6.4).
	LookupsHere int `json:"lookupsHere"`

	// Error is set when the node could not be fully evaluated. Standard
	// values are "no-spf", "nxdomain", "timeout", "loop", "syntax".
	Error string `json:"error,omitempty"`

	// Children are nested includes / redirects.
	Children []*SPFNode `json:"children,omitempty"`
}

// SPFFlattenResponse is the result of a recursive SPF flatten.
type SPFFlattenResponse struct {
	// Record is the root SPF record that was evaluated.
	Record string `json:"record"`

	// LookupCount is the total number of SPF terms that consumed a DNS lookup.
	LookupCount int `json:"lookupCount"`

	// VoidLookups counts NXDOMAIN / no-answer responses observed during the
	// walk (RFC 7208 §4.6.4 caps this at 2).
	VoidLookups int `json:"voidLookups"`

	// Exceeded is true when LookupCount exceeds the 10-lookup hard limit.
	Exceeded bool `json:"exceeded"`

	// VoidExceeded is true when VoidLookups exceeds the 2-void-lookup limit.
	VoidExceeded bool `json:"voidExceeded"`

	// Truncated is true when evaluation was stopped early (depth, cycle,
	// budget overrun).
	Truncated bool `json:"truncated"`

	// Tree is the recursive evaluation tree, rooted at Domain.
	Tree *SPFNode `json:"tree,omitempty"`
}
