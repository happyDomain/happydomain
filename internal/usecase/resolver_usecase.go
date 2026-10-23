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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
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

func (ru *resolverUsecase) ResolveQuestion(request happydns.ResolverRequest) (*dns.Msg, error) {
	request.DomainName = dns.Fqdn(request.DomainName)

	// The resolver choice, wherever it came from in the request, is picked and
	// vetted in one place shared with the other resolver endpoints.
	resolvers, err := ru.pickResolver(context.Background(), request.Resolver, request.Custom)
	if err != nil {
		return nil, err
	}

	client := dns.Client{Timeout: time.Second * 5}

	var r *dns.Msg
	rrType := dns.StringToType[request.Type]
	if _, isPseudo := happydns.PseudoTypeByRrtype(rrType); isPseudo {
		// Pseudo-types are registered in dns.StringToType so that DNSControl
		// handles them, but they only exist in the API of a provider: there is
		// nothing to ask a resolver about.
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("%s is not a real DNS record type, it cannot be resolved.", request.Type)}
	}

	if rrType == dns.TypeANY {
		r, err = helpers.MultiQuestion(context.Background(), &client, resolvers, request.DomainName, RRToAskForANY, helpers.QueryOptions{Recursive: true})
	} else {
		r, err = helpers.Question(context.Background(), &client, resolvers, request.DomainName, rrType, helpers.QueryOptions{Recursive: true})
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

// pickResolver turns the resolver a request to the resolver tool asked for
// into the dialable "ip:port" to ask, or, when it asked for none or for
// "local", the default resolvers to try in turn.
//
// Everything the request supplies goes through the resolver guard, and that
// includes the plain `name` form: the frontend sends "8.8.8.8" or "9.9.9.10"
// straight in that field (see web/src/lib/resolver.ts), so it is exactly as
// caller-controlled as `custom` is. Only the default resolvers are exempt,
// because the operator chose them, routinely on loopback.
func (ru *resolverUsecase) pickResolver(ctx context.Context, name, custom string) ([]string, error) {
	resolver := name
	switch resolver {
	case "", "local":
		return ru.config.DefaultResolvers, nil
	case "custom":
		if custom == "" {
			return nil, happydns.ValidationError{Msg: "custom resolver address required"}
		}
		resolver = custom
	}

	ctx, cancel := context.WithTimeout(ctx, resolverLookupTimeout)
	defer cancel()

	// ResolveAddrPort hands back an IP literal, never the name it was given:
	// miekg/dns would otherwise resolve it a second time, and the second answer
	// is the one an attacker gets to choose.
	target, err := ru.resolverGuard.ResolveAddrPort(ctx, resolver, 53)
	if err != nil {
		// Our own resolver failing to answer says nothing about the server the
		// caller asked for, so it must not be reported as a refused one.
		if errors.Is(err, netguard.ErrTemporary) {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to check the requested resolver: %w", err),
				UserMessage: ru.resolverGuard.Unavailable("The requested DNS server"),
			}
		}

		return nil, happydns.ValidationError{Msg: ru.resolverGuard.Refusal("The requested DNS server")}
	}

	return []string{target}, nil
}
