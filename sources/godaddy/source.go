// Copyright or © or Copr. happyDNS (2021)
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

package godaddy // import "happydns.org/sources/godaddy"

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"
	"github.com/oze4/godaddygo"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
	"git.happydns.org/happydns/utils"
)

type GoDaddyAPI struct {
	ApiKey    string `json:"apiKey,omitempty" happydns:"label=Api Key,placeholder=xxxxxxxxxx,required"`
	ApiSecret string `json:"apiSecret,omitempty" happydns:"label=Api Secret,placeholder=xxxxxxxxxx,required"`
}

func (s *GoDaddyAPI) ListAvailableTypes() (types []uint16) {
	return []uint16{
		dns.TypeA,
		dns.TypeAAAA,
		dns.TypeCNAME,
		dns.TypeMX,
		dns.TypeNS,
		dns.TypeSOA,
		dns.TypeSRV,
		dns.TypeTXT,
	}
}

func (s *GoDaddyAPI) newClient() (godaddygo.API, error) {
	return godaddygo.NewProduction(
		s.ApiKey,
		s.ApiSecret,
	)
}

func (s *GoDaddyAPI) ListDomains() (zones []string, err error) {
	var client godaddygo.API
	client, err = s.newClient()
	if err != nil {
		return
	}

	var domains []godaddygo.DomainSummary
	domains, err = client.V1().ListDomains(context.Background())
	if err != nil {
		return
	}

	for _, domain := range domains {
		zones = append(zones, dns.Fqdn(domain.Domain))
	}

	return
}

func (s *GoDaddyAPI) Validate() error {
	var client godaddygo.API
	client, err := s.newClient()
	if err != nil {
		return err
	}

	avlb, err := client.V1().CheckAvailability(context.Background(), "happydns.org", false)
	if err != nil {
		return err
	}

	if avlb.Domain != "happydns.org" {
		return fmt.Errorf("GoDaddy API doesn't returns something usefull. Try update happyDNS or report bug.")
	}

	return nil
}

func (s *GoDaddyAPI) DomainExists(fqdn string) (err error) {
	var client godaddygo.API
	client, err = s.newClient()
	if err != nil {
		return
	}

	_, err = client.V1().Domain(strings.TrimSuffix(fqdn, ".")).GetDetails(context.Background())
	return
}

func toRR(r *godaddygo.Record, origin string) (dns.RR, error) {
	if r.Name == "@" {
		r.Name = origin
	} else {
		r.Name = r.Name + "." + origin
	}

	if r.Protocol != "" {
		r.Name = r.Protocol + "." + r.Name
	}
	if r.Service != "" {
		r.Name = r.Service + "." + r.Name
	}

	if r.Type == "TXT" {
		r.Data = "\"" + r.Data + "\""
	}

	str := fmt.Sprintf("$ORIGIN .\n%s %d IN %s ", r.Name, r.TTL, r.Type)

	if r.Type == "SRV" {
		str += fmt.Sprintf("%d %d %d %s", r.Priority, r.Weight, r.Port, r.Data)
	} else if r.Type == "MX" {
		str += fmt.Sprintf("%d %s", r.Priority, r.Data)
	} else {
		str += r.Data
	}

	return dns.NewRR(str)

}

func (s *GoDaddyAPI) ImportZone(dn *happydns.Domain) (rrs []dns.RR, err error) {
	var client godaddygo.API
	client, err = s.newClient()
	if err != nil {
		return
	}

	var records []godaddygo.Record
	records, err = client.V1().Domain(strings.TrimSuffix(dn.DomainName, ".")).Records().List(context.Background())

	for _, r := range records {
		var rr dns.RR
		rr, err = toRR(&r, dn.DomainName)
		if err != nil {
			return
		}

		rrs = append(rrs, rr)
	}

	return
}

func getSubdomain(dn *happydns.Domain, rr dns.RR) string {
	str := strings.TrimSuffix(strings.TrimSuffix(rr.Header().Name, dn.DomainName), ".")
	if len(str) == 0 {
		return "@"
	}
	return str
}

func createGoDaddyRecord(rr dns.RR, dn *happydns.Domain) godaddygo.Record {
	record := godaddygo.Record{
		Name: getSubdomain(dn, rr),
		TTL:  int(rr.Header().Ttl),
		Type: dns.Type(rr.Header().Rrtype).String(),
	}

	if mx, ok := rr.(*dns.MX); ok {
		record.Priority = int(mx.Preference)
		record.Data = strings.TrimSuffix(utils.DomainFQDN(mx.Mx, dn.DomainName), ".")
	} else if srv, ok := rr.(*dns.SRV); ok {
		record.Priority = int(srv.Priority)
		record.Weight = int(srv.Weight)
		record.Port = int(srv.Port)
		record.Data = strings.TrimSuffix(utils.DomainFQDN(srv.Target, dn.DomainName), ".")

		sname := strings.SplitN(record.Name, ".", 3)
		if len(sname) == 3 && sname[0][0] == '_' && sname[1][0] == '_' {
			record.Service = sname[0]
			record.Protocol = sname[1]
			record.Name = sname[2]
		} else if len(sname) == 2 && sname[0][0] == '_' && sname[1][0] == '_' {
			record.Service = sname[0]
			record.Protocol = sname[1]
			record.Name = "@"
		}
	} else if cname, ok := rr.(*dns.CNAME); ok {
		record.Data = strings.TrimSuffix(utils.DomainFQDN(cname.Target, dn.DomainName), ".")
	} else if ns, ok := rr.(*dns.NS); ok {
		record.Data = strings.TrimSuffix(utils.DomainFQDN(ns.Ns, dn.DomainName), ".")
	} else {
		record.Data = strings.TrimPrefix(rr.String(), rr.Header().String())
	}

	return record
}

func (s *GoDaddyAPI) AddRR(dn *happydns.Domain, rr dns.RR) (err error) {
	var client godaddygo.API
	client, err = s.newClient()
	if err != nil {
		return
	}

	sup := createGoDaddyRecord(rr, dn)

	err = client.V1().Domain(strings.TrimSuffix(dn.DomainName, ".")).Records().Add(context.Background(), []godaddygo.Record{sup})

	return
}

func (s *GoDaddyAPI) DeleteRR(dn *happydns.Domain, rr dns.RR) (err error) {
	var client godaddygo.API
	client, err = s.newClient()
	if err != nil {
		return
	}

	del := createGoDaddyRecord(rr, dn)

	// Apparently, we can't pass an empty array to ReplaceByType when Name=@, this doesn't delete the last record.
	// So, passing the entire domain each time...
	var records []godaddygo.Record
	records, err = client.V1().Domain(strings.TrimSuffix(dn.DomainName, ".")).Records().List(context.Background())
	if err != nil {
		return
	}

	log.Println(records)
	// Find the right record
	for i, rec := range records {
		if rec.Name == del.Name &&
			rec.Data == del.Data &&
			rec.Port == del.Port &&
			rec.Priority == del.Priority &&
			rec.Protocol == del.Protocol &&
			rec.Service == del.Service &&
			rec.Type == del.Type &&
			rec.Weight == del.Weight {
			records = append(records[:i], records[i+1:]...)
			err = client.V1().Domain(strings.TrimSuffix(dn.DomainName, ".")).Records().Update(context.Background(), records)
			return
		}
	}

	err = fmt.Errorf("Unable to find the good record to delete in the zone.")
	return
}

func (s *GoDaddyAPI) UpdateSOA(dn *happydns.Domain, newSOA *dns.SOA, refreshSerial bool) (err error) {
	return fmt.Errorf("Not implemented yet")
}

func init() {
	sources.RegisterSource(func() happydns.Source {
		return &GoDaddyAPI{}
	}, sources.SourceInfos{
		Name:        "GoDaddy",
		Description: "American hosting provider.",
	})
}
