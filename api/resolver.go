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

package api

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
)

var (
	RRToAskForANY = []uint16{dns.TypeSOA, dns.TypeA, dns.TypeAAAA, dns.TypeNS, dns.TypeMX, dns.TypeTXT}
)

func declareResolverRoutes(router *gin.RouterGroup) {
	router.POST("/resolver", runResolver)
}

// resolverRequest holds the resolution parameters
type resolverRequest struct {
	// Resolver is the name of the resolver to use (or local or custom).
	Resolver string `json:"resolver"`

	// Custom is the address to the recursive server to use.
	Custom string `json:"custom,omitempty"`

	// DomainName is the FQDN to resolve.
	DomainName string `json:"domain"`

	// Type is the type of record to retrieve.
	Type string `json:"type"`
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

// DNSMsg is the documentation structur corresponding to dns.Msg
type DNSMsg struct {
	// Question is the Question section of the DNS response.
	Question []DNSQuestion

	// Answer is the list of Answer records in the DNS response.
	Answer []interface{} `swaggertype:"object"`

	// Ns is the list of Authoritative records in the DNS response.
	Ns []interface{} `swaggertype:"object"`

	// Extra is the list of extra records in the DNS response.
	Extra []interface{} `swaggertype:"object"`
}

type DNSQuestion struct {
	// Name is the domain name researched.
	Name string

	// Qtype is the type of record researched.
	Qtype uint16

	// Qclass is the class of record researched.
	Qclass uint16
}

// runResolver performs a NS resolution for a given domain, with options.
//
//	@Summary	Perform a DNS resolution.
//	@Schemes
//	@Description	Perform a NS resolution	for a given domain, with options.
//	@Tags			resolver
//	@Accept			json
//	@Produce		json
//	@Param			body	body		resolverRequest	true	"Options to the resolution"
//	@Success		200		{object}	DNSMsg
//	@Success		204		{object}	happydns.Error	"No content"
//	@Failure		400		{object}	happydns.Error	"Invalid input"
//	@Failure		401		{object}	happydns.Error	"Authentication failure"
//	@Failure		403		{object}	happydns.Error	"The resolver refused to treat our request"
//	@Failure		404		{object}	happydns.Error	"The domain doesn't exist"
//	@Failure		406		{object}	happydns.Error	"The resolver returned an error"
//	@Failure		500		{object}	happydns.Error
//	@Router			/resolver [post]
func runResolver(c *gin.Context) {
	var urr resolverRequest
	if err := c.ShouldBindJSON(&urr); err != nil {
		log.Printf("%s sends invalid ResolverRequest JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	urr.DomainName = dns.Fqdn(urr.DomainName)

	if urr.Resolver == "custom" {
		urr.Resolver = urr.Custom
	} else if urr.Resolver == "local" {
		cConf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			log.Printf("%s unable to load ClientConfigFromFile: %s", c.ClientIP(), err.Error())
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Sorry, we are currently unable to perform the request. Please try again later."})
			return
		}

		urr.Resolver = cConf.Servers[rand.Intn(len(cConf.Servers))]
	}

	if strings.Count(urr.Resolver, ":") > 0 && urr.Resolver[0] != '[' {
		urr.Resolver = "[" + urr.Resolver + "]"
	}

	client := dns.Client{Timeout: time.Second * 5}

	var r *dns.Msg
	var err error
	rrType := dns.StringToType[urr.Type]
	if rrType == dns.TypeANY {
		r, err = resolverANYQuestion(client, urr.Resolver+":53", urr.DomainName)
	} else {
		r, err = resolverQuestion(client, urr.Resolver+":53", urr.DomainName, rrType)
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	if r == nil {
		c.AbortWithStatusJSON(http.StatusNoContent, gin.H{"errmsg": "No response to display."})
		return
	} else if r.Rcode == dns.RcodeFormatError {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "DNS request mal formated."})
		return
	} else if r.Rcode == dns.RcodeServerFailure {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Resolver returns an error (most likely something is wrong in %q).", urr.DomainName)})
		return
	} else if r.Rcode == dns.RcodeNameError {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": fmt.Sprintf("The domain %q was not found.", urr.DomainName)})
		return
	} else if r.Rcode == dns.RcodeNotImplemented {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": "Resolver returns the request hits non implemented code."})
		return
	} else if r.Rcode == dns.RcodeRefused {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Resolver refused to treat our request."})
		return
	} else if r.Rcode != dns.RcodeSuccess {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"errmsg": fmt.Errorf("Resolver returns %s.", dns.RcodeToString[r.Rcode])})
		return
	}

	c.JSON(http.StatusOK, r)
}
