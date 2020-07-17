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

package ovh // import "happydns.org/sources/ovh"

import (
	"flag"
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/ovh/go-ovh/ovh"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

var (
	appKey    string
	appSecret string
)

type OVHAPI struct {
	Endpoint    string `json:"endpoint,omitempty" happydns:"label=Endpoint,default=ovh-eu,choices=ovh-eu;ovh-us;ovh-ca;soyoustart-eu;soyoustart-ca;kimsufi-eu;kimsufi-ca,required"`
	ConsumerKey string `json:"consumerkey,omitempty" happydns:"label=Consumer Key,placeholder=xxxxxxxxxx,required,description=The endpoint depends on your service's seller (OVH/SoYouStart/Kimsufi) and the datacenter location (eu/us/ca). Choose 'ovh-eu' if unsure."`
}

func (s *OVHAPI) newClient() (*ovh.Client, error) {
	return ovh.NewClient(
		s.Endpoint,
		appKey,
		appSecret,
		s.ConsumerKey,
	)
}

func (s *OVHAPI) ListDomains() (zones []string, err error) {
	var client *ovh.Client
	client, err = s.newClient()
	if err != nil {
		return
	}

	err = client.Get("/domain/zone", &zones)
	if err != nil {
		return
	}

	for i, zone := range zones {
		zones[i] = dns.Fqdn(zone)
	}

	return
}

func (s *OVHAPI) Validate() error {
	client, err := s.newClient()
	if err != nil {
		return err
	}

	me := struct {
		State string `json:"state"`
	}{}

	err = client.Get("/me", &me)
	if err != nil {
		return err
	}

	if me.State != "complete" {
		return fmt.Errorf("API state returns is %q, expected \"complete\"", me.State)
	}

	return nil
}

func (s *OVHAPI) DomainExists(fqdn string) (err error) {
	var client *ovh.Client
	client, err = s.newClient()
	if err != nil {
		return
	}

	var zone struct{ Name string }

	err = client.Get(fmt.Sprintf("/domain/zone/%s", strings.TrimSuffix(fqdn, ".")), &zone)
	if err != nil {
		return
	}

	return
}

func (s *OVHAPI) ImportZone(dn *happydns.Domain) (rrs []dns.RR, err error) {
	var client *ovh.Client
	client, err = s.newClient()
	if err != nil {
		return
	}

	var zone string

	err = client.Get(fmt.Sprintf("/domain/zone/%s/export", strings.TrimSuffix(dn.DomainName, ".")), &zone)
	if err != nil {
		return
	}

	zp := dns.NewZoneParser(strings.NewReader(zone), dn.DomainName, "")

	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rrs = append(rrs, rr)
	}

	if err = zp.Err(); err != nil {
		return
	}

	return
}

type OVHRecord struct {
	Id        int64  `json:"id,omitempty"`
	FieldType string `json:"fieldType"`
	TTL       uint32 `json:"ttl"`
	Target    string `json:"target"`
	Zone      string `json:"zone,omitempty"`
	SubDomain string `json:"subDomain"`
}

func (s *OVHAPI) AddRR(dn *happydns.Domain, rr dns.RR) (err error) {
	var client *ovh.Client
	client, err = s.newClient()
	if err != nil {
		return
	}

	sup := OVHRecord{
		SubDomain: ovhSubdomain(dn, rr),
		FieldType: dns.Type(rr.Header().Rrtype).String(),
		TTL:       rr.Header().Ttl,
		Target:    strings.TrimPrefix(rr.String(), rr.Header().String()),
	}

	var res OVHRecord

	err = client.Post(
		fmt.Sprintf("/domain/zone/%s/record", strings.TrimSuffix(dn.DomainName, ".")),
		sup, &res)
	if err != nil {
		return
	}

	err = client.Post(fmt.Sprintf("/domain/zone/%s/refresh", strings.TrimSuffix(dn.DomainName, ".")), nil, nil)
	if err != nil {
		return
	}

	return
}

func ovhSubdomain(dn *happydns.Domain, rr dns.RR) string {
	return strings.TrimSuffix(strings.TrimSuffix(rr.Header().Name, dn.DomainName), ".")
}

func (s *OVHAPI) DeleteRR(dn *happydns.Domain, rr dns.RR) (err error) {
	var client *ovh.Client
	client, err = s.newClient()
	if err != nil {
		return
	}

	// Get all matching IDs
	var ids []int64
	err = client.Get(
		fmt.Sprintf("/domain/zone/%s/record?fieldType=%s&subDomain=%s", strings.TrimSuffix(dn.DomainName, "."), dns.Type(rr.Header().Rrtype).String(), ovhSubdomain(dn, rr)),
		&ids)
	if err != nil {
		return
	}

	// Find the right ID
	for _, id := range ids {
		var rec OVHRecord

		err = client.Get(
			fmt.Sprintf("/domain/zone/%s/record/%d", strings.TrimSuffix(dn.DomainName, "."), id),
			&rec)
		if err != nil {
			return
		}

		if rec.Target == strings.TrimPrefix(rr.String(), rr.Header().String()) {
			err = client.Delete(fmt.Sprintf("/domain/zone/%s/record/%d", strings.TrimSuffix(dn.DomainName, "."), id), nil)
			if err != nil {
				return
			}

			err = client.Post(fmt.Sprintf("/domain/zone/%s/refresh", strings.TrimSuffix(dn.DomainName, ".")), nil, nil)
			if err != nil {
				return
			}

			return
		}
	}

	return
}

type OVH_SOA struct {
	Server  string `json:"server"`
	Email   string `json:"email"`
	Serial  uint32 `json:"serial"`
	Refresh uint32 `json:"refresh"`
	Expire  uint32 `json:"expire"`
	NxTtl   uint32 `json:"nxDomainTtl"`
	Ttl     uint32 `json:"ttl"`
}

func (s *OVHAPI) UpdateSOA(dn *happydns.Domain, newSOA *dns.SOA, refreshSerial bool) (err error) {
	var client *ovh.Client
	client, err = s.newClient()
	if err != nil {
		return
	}

	// Get current SOA
	var curSOA OVH_SOA
	err = client.Get(
		fmt.Sprintf("/domain/zone/%s/soa", strings.TrimSuffix(dn.DomainName, ".")),
		&curSOA)
	if err != nil {
		return
	}

	// Is there any change?
	changes := false
	if curSOA.Server != newSOA.Ns {
		curSOA.Server = newSOA.Ns
		changes = true
	}
	if curSOA.Email != newSOA.Mbox {
		curSOA.Email = newSOA.Mbox
		changes = true
	}
	if curSOA.Refresh != newSOA.Refresh {
		curSOA.Refresh = newSOA.Refresh
		changes = true
	}
	if curSOA.Expire != newSOA.Expire {
		curSOA.Expire = newSOA.Expire
		changes = true
	}
	if curSOA.NxTtl != newSOA.Minttl {
		curSOA.NxTtl = newSOA.Minttl
		changes = true
	}

	// OVH handles automatically serial update, so only force non-refresh
	if !refreshSerial && curSOA.Serial != newSOA.Serial {
		curSOA.Serial = newSOA.Serial
		changes = true
	}
	newSOA.Serial = curSOA.Serial

	if changes {
		err = client.Post(fmt.Sprintf("/domain/zone/%s/refresh", strings.TrimSuffix(dn.DomainName, ".")), nil, nil)
		if err != nil {
			return
		}
	}

	return
}

func init() {
	flag.StringVar(&appKey, "ovh-application-key", "", "Application Key for using the OVH API")
	flag.StringVar(&appSecret, "ovh-application-secret", "", "Application Secret for using the OVH API")

	sources.RegisterSource(func() happydns.Source {
		return &OVHAPI{}
	}, sources.SourceInfos{
		Name:        "OVH",
		Description: "Hosting",
	})
}
