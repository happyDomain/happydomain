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

package gandi // import "happydns.org/sources/gandi"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

type GandiAPI struct {
	APIKey string `json:"api_key,omitempty" happydns:"label=API Key,placeholder=xxxxxxxxxx,required,description=Get your API Key in the Security section under https://account.gandi.net/. Copy the corresponding key."`
}

func (s *GandiAPI) newRequest(method, url string, body io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}
	req.Header.Add("authorization", "Apikey "+s.APIKey)
	req.Header.Add("content-type", "application/json")
	return
}

type gandiError struct {
	Cause   string `json:"cause"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Object  string `json:"object,omitempty"`
}

func (e gandiError) Error() string {
	return fmt.Sprintf("Error %d: %s (%s)", e.Code, e.Message, e.Cause)
}

func doJSON(req *http.Request, v interface{}) (err error) {
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		gerr := gandiError{}
		err = json.NewDecoder(resp.Body).Decode(&gerr)
		if err != nil {
			return
		} else {
			return gerr
		}
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return
}

func (s *GandiAPI) ListAvailableTypes() (types []uint16) {
	types = []uint16{
		dns.TypeA,
		dns.TypeAAAA,
		//dns.TypeALIAS,
		dns.TypeCAA,
		dns.TypeCDS,
		dns.TypeCNAME,
		dns.TypeDNAME,
		dns.TypeDS,
		dns.TypeKEY,
		dns.TypeLOC,
		dns.TypeMX,
		dns.TypeNS,
		dns.TypeOPENPGPKEY,
		dns.TypePTR,
		dns.TypeSPF,
		dns.TypeSOA,
		dns.TypeSRV,
		dns.TypeSSHFP,
		dns.TypeTLSA,
		dns.TypeTXT,
		//dns.TypeWKS,
	}

	req, err := s.newRequest("GET", "https://api.gandi.net/v5/livedns/dns/rrtypes", nil)
	if err != nil {
		return
	}

	rrtypes := []string{}

	err = doJSON(req, &rrtypes)
	if err != nil {
		return
	}

	types = []uint16{}
	for _, r := range rrtypes {
		if t, ok := dns.StringToType[r]; ok {
			types = append(types, t)
		}
	}

	return
}

type gandiDomainInfo struct {
	FQDN string `json:"fqdn"`
	Href string `json:"domain_href"`
}

func (s *GandiAPI) ListDomains() (zones []string, err error) {
	var req *http.Request
	req, err = s.newRequest("GET", "https://api.gandi.net/v5/livedns/domains", nil)
	if err != nil {
		return
	}

	domains := []gandiDomainInfo{}

	err = doJSON(req, &domains)
	if err != nil {
		return
	}

	for _, d := range domains {
		zones = append(zones, dns.Fqdn(d.FQDN))
	}

	return
}

func (s *GandiAPI) Validate() (err error) {
	var req *http.Request
	req, err = s.newRequest("GET", "https://api.gandi.net/v5/billing/info", nil)
	if err != nil {
		return
	}

	err = doJSON(req, nil)
	return
}

func (s *GandiAPI) DomainExists(fqdn string) (err error) {
	var req *http.Request

	// Search domain in liveDNS API first
	req, err = s.newRequest("GET", "https://api.gandi.net/v5/livedns/domains/"+strings.TrimSuffix(fqdn, "."), nil)
	if err != nil {
		return
	}

	err = doJSON(req, nil)
	if err == nil {
		return
	} else if gerr, ok := err.(gandiError); !ok || gerr.Code == 404 {
		return
	}

	// Determine if the domain exists in the old API
	req, err = s.newRequest("GET", "https://api.gandi.net/v5/domain/domains/"+strings.TrimSuffix(fqdn, "."), nil)
	if err != nil {
		return
	}

	err = doJSON(req, nil)
	if err != nil {
		return
	}

	return fmt.Errorf("Your domain %q uses the Gandi's classic DNS interface. You need to switch your domain to the new Gandi's LiveDNS to be able to use it in happyDNS. Please follow thoses simple instructions to make the change in a minute: https://docs.gandi.net/en/domain_names/common_operations/changing_nameservers.html#how-to-switch-to-livedns", fqdn)
}

type gandiRecord struct {
	Name   string   `json:"rrset_name,omitempty"`
	Type   string   `json:"rrset_type,omitempty"`
	Values []string `json:"rrset_values"`
	TTL    uint32   `json:"rrset_ttl,omitempty"`
}

func (r *gandiRecord) toRRs(origin string) (rrs []dns.RR, err error) {
	if len(r.Name) == 0 || r.Name == "@" {
		r.Name = origin
	} else {
		r.Name += "." + origin
	}

	for _, value := range r.Values {
		var rr dns.RR
		rr, err = dns.NewRR(fmt.Sprintf("$ORIGIN %s\n%s %d IN %s %s", origin, r.Name, r.TTL, r.Type, value))
		rrs = append(rrs, rr)
	}

	return
}

func (s *GandiAPI) ImportZone(dn *happydns.Domain) (rrs []dns.RR, err error) {
	var req *http.Request
	req, err = s.newRequest("GET", "https://api.gandi.net/v5/livedns/domains/"+strings.TrimSuffix(dn.DomainName, ".")+"/records", nil)
	if err != nil {
		return
	}

	records := []*gandiRecord{}

	err = doJSON(req, &records)
	if err != nil {
		return
	}

	for _, r := range records {
		var rr []dns.RR
		rr, err = r.toRRs(dn.DomainName)
		if err != nil {
			return
		}

		rrs = append(rrs, rr...)
	}

	return
}

func (s *GandiAPI) changeRR(dn *happydns.Domain, rr dns.RR, cbChange func(*gandiRecord) error) (err error) {
	var req *http.Request
	rrtype := dns.Type(rr.Header().Rrtype).String()
	var rrname string
	if rr.Header().Name == dn.DomainName {
		rrname = "@"
	} else {
		rrname = strings.TrimSuffix(rr.Header().Name, "."+dn.DomainName)
	}
	url := "https://api.gandi.net/v5/livedns/domains/" + strings.TrimSuffix(dn.DomainName, ".") + "/records/" + rrname + "/" + rrtype

	// Get already existing records for this type
	req, err = s.newRequest("GET", url, nil)
	if err != nil {
		return
	}

	record := &gandiRecord{
		Type: rrtype,
	}
	if rr.Header().Ttl != 0 {
		record.TTL = rr.Header().Ttl
	}

	err = doJSON(req, record)
	if err != nil {
		if gerr, ok := err.(gandiError); !ok || (gerr.Code != 404 && gerr.Object == "dns-record") {
			return
		}
	}

	// Do the callback
	err = cbChange(record)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	reqType := "PUT"
	if len(record.Values) == 0 {
		reqType = "DELETE"
	} else {
		err = json.NewEncoder(&buf).Encode(record)
		if err != nil {
			return
		}
	}

	// Send the new content
	req, err = s.newRequest(reqType, url, &buf)
	if err != nil {
		return
	}

	err = doJSON(req, nil)
	return
}

func (s *GandiAPI) AddRR(dn *happydns.Domain, rr dns.RR) error {
	return s.changeRR(dn, rr, func(record *gandiRecord) error {
		// Add the new value
		record.Values = append(record.Values, strings.TrimPrefix(rr.String(), rr.Header().String()))

		return nil
	})
}

func (s *GandiAPI) DeleteRR(dn *happydns.Domain, rr dns.RR) (err error) {
	return s.changeRR(dn, rr, func(record *gandiRecord) error {
		str := strings.TrimPrefix(rr.String(), rr.Header().String())
		for i, v := range record.Values {
			if v == str {
				record.Values = append(record.Values[:i], record.Values[i+1:]...)
				return nil
			}
		}

		return fmt.Errorf("Record to delete not found.")
	})
}

func (s *GandiAPI) UpdateSOA(dn *happydns.Domain, newSOA *dns.SOA, refreshSerial bool) (err error) {
	return fmt.Errorf("SOA record is not supported by Gandi's API")
}

func init() {
	sources.RegisterSource(func() happydns.Source {
		return &GandiAPI{}
	}, sources.SourceInfos{
		Name:        "Gandi",
		Description: "French hosting provider.",
	})
}
