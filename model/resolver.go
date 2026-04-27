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
	FetchMTASTSPolicy(MTASTSPolicyRequest) (*MTASTSPolicyResponse, error)
	CheckDMARCReportAuth(DMARCReportAuthRequest) (*DMARCReportAuthResponse, error)
}

// DMARCReportAuthRequest asks the backend to check whether ExternalDomain
// authorizes receiving DMARC reports for Owner, by resolving the TXT record
// at <Owner>._report._dmarc.<ExternalDomain> per RFC 7489 sec. 7.1.
type DMARCReportAuthRequest struct {
	// Resolver is the name of the resolver to use (or local or custom).
	Resolver string `json:"resolver,omitempty"`

	// Custom is the address to the recursive server to use.
	Custom string `json:"custom,omitempty"`

	// Owner is the protected domain whose DMARC record references the
	// external reporting destination (e.g. "example.com").
	Owner string `json:"owner"`

	// ExternalDomain is the FQDN of the reporting destination, taken from
	// the host part of a rua/ruf URI (e.g. "reports.thirdparty.tld").
	ExternalDomain string `json:"externalDomain"`
}

// DMARCReportAuthResponse reports whether ExternalDomain authorizes Owner to
// send it DMARC aggregate or forensic reports.
type DMARCReportAuthResponse struct {
	// QueriedName is the FQDN that was looked up (echoes the synthesized
	// name so the UI can surface it without rebuilding it).
	QueriedName string `json:"queriedName"`

	// Status is the high-level outcome:
	//   "ok"               at least one TXT starting with "v=DMARC1" was found
	//   "no-dmarc-record"  TXT records exist but none start with v=DMARC1
	//   "not-found"        NXDOMAIN or no TXT at the synthesized name
	//   "dns-error"        resolver returned an error or refused
	//   "resolver-error"   the resolver could not be contacted
	Status string `json:"status"`

	// ErrorMsg gives a short human-readable reason when Status != "ok".
	ErrorMsg string `json:"errorMsg,omitempty"`

	// Records is the list of TXT records returned (may be empty).
	Records []string `json:"records,omitempty"`
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

// MTASTSPolicyRequest asks the backend to fetch and parse the MTA-STS policy
// file published at https://mta-sts.<Domain>/.well-known/mta-sts.txt
// (RFC 8461 sec. 3.3).
type MTASTSPolicyRequest struct {
	// Domain is the FQDN whose MTA-STS policy should be fetched.
	Domain string `json:"domain"`
}

// MTASTSPolicyResponse is the result of an MTA-STS policy fetch.
type MTASTSPolicyResponse struct {
	// URL is the policy URL that was fetched.
	URL string `json:"url"`

	// Status is the high-level outcome:
	//   "ok"          policy fetched and parsed
	//   "dns-error"   could not resolve mta-sts.<domain>
	//   "tls-error"   TLS handshake failed
	//   "not-found"   HTTP 404 (no policy published)
	//   "http-error"  any other non-2xx status
	//   "fetch-error" connection refused, timeout, etc.
	//   "too-large"   body exceeded the size cap before parsing
	Status string `json:"status"`

	// HTTPCode is the response status code when an HTTP exchange completed.
	HTTPCode int `json:"httpCode,omitempty"`

	// ErrorMsg gives a short human-readable reason when Status != "ok".
	ErrorMsg string `json:"errorMsg,omitempty"`

	// Body is the raw response body (truncated to the size cap) returned for
	// diagnostic display. Set even when parsing failed.
	Body string `json:"body,omitempty"`

	// Parsed policy fields. Empty unless Status is "ok".
	Version string   `json:"version,omitempty"`
	Mode    string   `json:"mode,omitempty"`
	MX      []string `json:"mx,omitempty"`
	MaxAge  int      `json:"maxAge,omitempty"`

	// Redirected is true when the server tried to redirect us; per RFC 8461
	// sec. 3.3 we MUST NOT follow.
	Redirected bool `json:"redirected,omitempty"`
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
