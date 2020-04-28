package ovh // import "happydns.org/sources/ovh"

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/ovh/go-ovh/ovh"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

type OVHAPI struct {
	Endpoint    string `json:"endpoint,omitempty" happydns:"label=Endpoint,default=ovh-eu,choices=ovh-eu;ovh-us;ovh-ca;soyoustart-eu;soyoustart-ca;kimsufi-eu;kimsufi-ca,required"`
	AppKey      string `json:"appkey,omitempty" happydns:"label=Application Key,placeholder=xxxxxxxxxx,required"`
	AppSecret   string `json:"appsecret,omitempty" happydns:"label=Application Secret,placeholder=xxxxxxxxxx,required,secret"`
	ConsumerKey string `json:"consumerkey,omitempty" happydns:"label=Consumer Key,placeholder=xxxxxxxxxx,required"`
}

func (s *OVHAPI) newClient() (*ovh.Client, error) {
	return ovh.NewClient(
		s.Endpoint,
		s.AppKey,
		s.AppSecret,
		s.ConsumerKey,
	)
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

func init() {
	sources.RegisterSource("git.happydns.org/happydns/sources/ovh/OVHAPI", func() happydns.Source {
		return &OVHAPI{}
	}, sources.SourceInfos{
		Name:        "OVH",
		Description: "Hosting",
	})
}
