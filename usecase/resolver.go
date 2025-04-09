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
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/model"
)

var (
	RRToAskForANY = []uint16{dns.TypeSOA, dns.TypeA, dns.TypeAAAA, dns.TypeNS, dns.TypeMX, dns.TypeTXT}
)

type resolverUsecase struct {
	config *config.Options
}

func NewResolverUsecase(cfg *config.Options) happydns.ResolverUsecase {
	return &resolverUsecase{
		config: cfg,
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

	if request.Resolver == "custom" {
		request.Resolver = request.Custom
	} else if request.Resolver == "local" {
		cConf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to load ClientConfigFromFile: %s", err.Error()),
				UserMessage: "Sorry, we are currently unable to perform the request. Please try again later.",
				HTTPStatus:  http.StatusInternalServerError,
			}
		}

		request.Resolver = cConf.Servers[rand.Intn(len(cConf.Servers))]
	}

	if strings.Count(request.Resolver, ":") > 0 && request.Resolver[0] != '[' {
		request.Resolver = "[" + request.Resolver + "]"
	}

	client := dns.Client{Timeout: time.Second * 5}

	var r *dns.Msg
	var err error
	rrType := dns.StringToType[request.Type]
	if rrType == dns.TypeANY {
		r, err = resolverANYQuestion(client, request.Resolver+":53", request.DomainName)
	} else {
		r, err = resolverQuestion(client, request.Resolver+":53", request.DomainName, rrType)
	}
	if err != nil {
		return nil, happydns.InternalError{
			Err:        err,
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if r == nil {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("No response to display."),
			HTTPStatus: http.StatusNoContent,
		}
	} else if r.Rcode == dns.RcodeFormatError {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("DNS request mal formated."),
			HTTPStatus: http.StatusBadRequest,
		}
	} else if r.Rcode == dns.RcodeServerFailure {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("Resolver returns an error (most likely something is wrong in %q).", request.DomainName),
			HTTPStatus: http.StatusInternalServerError,
		}
	} else if r.Rcode == dns.RcodeNameError {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("The domain %q was not found.", request.DomainName),
			HTTPStatus: http.StatusNotFound,
		}
	} else if r.Rcode == dns.RcodeNotImplemented {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("Resolver returns the request hits non implemented code."),
			HTTPStatus: http.StatusInternalServerError,
		}
	} else if r.Rcode == dns.RcodeRefused {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("Resolver refused to treat our request."),
			HTTPStatus: http.StatusForbidden,
		}
	} else if r.Rcode != dns.RcodeSuccess {
		return nil, happydns.InternalError{
			Err:        fmt.Errorf("Resolver returns %s.", dns.RcodeToString[r.Rcode]),
			HTTPStatus: http.StatusNotAcceptable,
		}
	}

	return r, nil
}
