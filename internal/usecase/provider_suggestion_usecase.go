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

package usecase

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/idna"

	"git.happydns.org/happyDomain/internal/helpers"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

// providerSuggestionTimeout bounds the search of the name servers: the user
// is waiting for it to add their domain, and can do without.
const providerSuggestionTimeout = 3 * time.Second

type providerSuggestionUsecase struct {
	config  *happydns.Options
	timeout time.Duration
}

func NewProviderSuggestionUsecase(cfg *happydns.Options) happydns.ProviderSuggestionUsecase {
	return &providerSuggestionUsecase{
		config:  cfg,
		timeout: providerSuggestionTimeout,
	}
}

func (psu *providerSuggestionUsecase) SuggestProviders(ctx context.Context, domain string) (*happydns.ProviderSuggestion, error) {
	domain, err := happydns.NormalizeDomainName(domain)
	if err != nil {
		return nil, happydns.ValidationError{Msg: err.Error()}
	}

	domain, err = idna.ToASCII(domain)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("invalid domain name: %s", err.Error())}
	}

	ctx, cancel := context.WithTimeout(ctx, psu.timeout)
	defer cancel()

	nameservers, err := psu.findNameservers(ctx, domain)
	if err != nil {
		log.Printf("unable to find the name servers of %q to suggest a provider: %s", domain, err.Error())
		nameservers = []string{}
	}

	return &happydns.ProviderSuggestion{
		Nameservers: nameservers,
		Providers:   providerReg.MatchNameservers(nameservers),
	}, nil
}

// findNameservers asks the default resolvers for the name servers of domain.
// The ones of its parent are left out: they tell nothing about where a zone
// not delegated yet is hosted.
func (psu *providerSuggestionUsecase) findNameservers(ctx context.Context, domain string) ([]string, error) {
	client := &dns.Client{Timeout: psu.timeout}
	r, err := helpers.Question(ctx, client, psu.config.DefaultResolvers, domain, dns.TypeNS, helpers.QueryOptions{Recursive: true})
	if err != nil {
		return nil, err
	}

	nameservers := []string{}
	for _, rr := range r.Answer {
		// The answer about an alias holds the name servers of its target.
		if ns, ok := rr.(*dns.NS); ok && dns.CanonicalName(ns.Hdr.Name) == dns.CanonicalName(domain) {
			nameservers = append(nameservers, dns.CanonicalName(ns.Ns))
		}
	}

	slices.Sort(nameservers)
	return slices.Compact(nameservers), nil
}
