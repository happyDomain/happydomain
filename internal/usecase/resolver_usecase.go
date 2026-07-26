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

package usecase

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

var (
	RRToAskForANY = []uint16{dns.TypeSOA, dns.TypeA, dns.TypeAAAA, dns.TypeNS, dns.TypeMX, dns.TypeTXT}
)

// resolverLookupTimeout bounds the name resolution done before a DNS server is
// contacted. The endpoints it protects are unauthenticated, so it stays short.
const resolverLookupTimeout = 2 * time.Second

type resolverUsecase struct {
	config *happydns.Options

	// resolverGuard vets the DNS server a caller asks us to query;
	// outboundGuard vets the HTTP destinations these endpoints fetch, today
	// only the MTA-STS policy file.
	resolverGuard *netguard.Guard
	outboundGuard *netguard.Guard
}

func NewResolverUsecase(cfg *happydns.Options, resolverGuard, outboundGuard *netguard.Guard) happydns.ResolverUsecase {
	return &resolverUsecase{
		config:        cfg,
		resolverGuard: resolverGuard,
		outboundGuard: outboundGuard,
	}
}

func resolverANYQuestion(client dns.Client, resolver string, dn string) (r *dns.Msg, err error) {
	var response *dns.Msg

	for _, rrType := range RRToAskForANY {
		m := new(dns.Msg)
		m.Question = append(m.Question, dns.Question{
			Name:   dn,
			Qtype:  rrType,
			Qclass: dns.ClassINET,
		})
		m.RecursionDesired = true
		m.SetEdns0(4096, true)

		response, _, err = client.Exchange(m, resolver)
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

	if r == nil {
		r = response
		r.Question[0].Qtype = dns.TypeANY
	}

	return
}

func resolverQuestion(client dns.Client, resolver string, dn string, rrType uint16) (*dns.Msg, error) {
	m := new(dns.Msg)
	m.SetQuestion(dn, rrType)
	m.RecursionDesired = true
	m.SetEdns0(4096, true)

	r, _, err := client.Exchange(m, resolver)

	return r, err
}

func (ru *resolverUsecase) ResolveQuestion(request happydns.ResolverRequest) (*dns.Msg, error) {
	request.DomainName = dns.Fqdn(request.DomainName)

	// The resolver choice, wherever it came from in the request, is picked and
	// vetted in one place shared with the other resolver endpoints.
	resolver, err := ru.pickResolver(context.Background(), request.Resolver, request.Custom)
	if err != nil {
		return nil, err
	}

	client := dns.Client{Timeout: time.Second * 5}

	var r *dns.Msg
	rrType := dns.StringToType[request.Type]
	if rrType == dns.TypeANY {
		r, err = resolverANYQuestion(client, resolver, request.DomainName)
	} else {
		r, err = resolverQuestion(client, resolver, request.DomainName, rrType)
	}
	if err != nil {
		return nil, happydns.ValidationError{Msg: err.Error()}
	}

	if r == nil {
		return nil, happydns.CustomError{
			Err:    fmt.Errorf("No response to display."),
			Status: http.StatusNoContent,
		}
	} else if r.Rcode == dns.RcodeFormatError {
		return nil, happydns.ValidationError{Msg: "DNS request mal formated."}
	} else if r.Rcode == dns.RcodeServerFailure {
		return nil, happydns.InternalError{
			Err: fmt.Errorf("Resolver returns an error (most likely something is wrong in %q).", request.DomainName),
		}
	} else if r.Rcode == dns.RcodeNameError {
		return nil, happydns.NotFoundError{Msg: fmt.Sprintf("The domain %q was not found.", request.DomainName)}
	} else if r.Rcode == dns.RcodeNotImplemented {
		return nil, happydns.InternalError{
			Err: fmt.Errorf("Resolver returns the request hits non implemented code."),
		}
	} else if r.Rcode == dns.RcodeRefused {
		return nil, happydns.ForbiddenError{Msg: "Resolver refused to treat our request."}
	} else if r.Rcode != dns.RcodeSuccess {
		return nil, happydns.CustomError{
			Err:    fmt.Errorf("Resolver returns %s.", dns.RcodeToString[r.Rcode]),
			Status: http.StatusNotAcceptable,
		}
	}

	return r, nil
}
