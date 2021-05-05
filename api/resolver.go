// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

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

type resolverRequest struct {
	Resolver   string `json:"resolver"`
	Custom     string `json:"custom,omitempty"`
	DomainName string `json:"domain"`
	Type       string `json:"type"`
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

func runResolver(c *gin.Context) {
	var urr resolverRequest
	if err := c.ShouldBindJSON(&urr); err != nil {
		log.Printf("%s sends invalid ResolverRequest JSON: %w", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Something is wrong in received data: %w", err)})
		return
	}

	urr.DomainName = dns.Fqdn(urr.DomainName)

	if urr.Resolver == "custom" {
		urr.Resolver = urr.Custom
	} else if urr.Resolver == "local" {
		cConf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			log.Printf("%s unable to load ClientConfigFromFile: %w", c.ClientIP(), err)
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
