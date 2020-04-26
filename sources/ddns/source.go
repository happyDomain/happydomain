package ddns // import "happydns.org/sources/ddns"

import (
	"encoding/base64"
	"net"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

type DDNSServer struct {
	Server  string `json:"server,omitempty" happydns:"label=Server,placeholder=127.0.0.1"`
	KeyName string `json:"keyname,omitempty" happydns:"label=Key Name,placeholder=ddns.,required"`
	KeyAlgo string `json:"algorithm,omitempty" happydns:"label=Key Algorithm,default=hmac-sha256.,choices=hmac-md5.sig-alg.reg.int.;hmac-sha1.;hmac-sha224.;hmac-sha256.;hmac-sha384.;hmac-sha512.,required"`
	KeyBlob []byte `json:"keyblob,omitempty" happydns:"label=Secret Key,placeholder=a0b1c2d3e4f5==,required,secret"`
}

func (s *DDNSServer) base64KeyBlob() string {
	return base64.StdEncoding.EncodeToString(s.KeyBlob)
}

func (s *DDNSServer) Validate() error {
	d := net.Dialer{}
	con, err := d.Dial("tcp", s.Server)
	if err != nil {
		return err
	}
	defer con.Close()

	return nil
}

func (s *DDNSServer) ImportZone(dn *happydns.Domain) (rrs []dns.RR, err error) {
	d := net.Dialer{}
	con, errr := d.Dial("tcp", s.Server)
	if errr != nil {
		err = errr
		return
	}
	defer con.Close()

	m := new(dns.Msg)
	m.SetEdns0(4096, true)
	m.SetAxfr(dn.DomainName)
	m.SetTsig(s.KeyName, s.KeyAlgo, 300, time.Now().Unix())

	dnscon := &dns.Conn{Conn: con}
	transfer := &dns.Transfer{Conn: dnscon, TsigSecret: map[string]string{s.KeyName: s.base64KeyBlob()}}
	c, errr := transfer.In(m, s.Server)

	if errr != nil {
		err = errr
		return
	}

	for {
		response, ok := <-c
		if !ok {
			break
		}

		for _, rr := range response.RR {
			rrs = append(rrs, rr)
		}
	}

	if len(rrs) > 0 {
		rrs = rrs[0 : len(rrs)-1]
	}

	return
}

func (s *DDNSServer) AddRR(domain *happydns.Domain, rr dns.RR) error {
	m := new(dns.Msg)
	m.Id = dns.Id()
	m.Opcode = dns.OpcodeUpdate
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{domain.DomainName, dns.TypeSOA, dns.ClassINET}

	m.Insert([]dns.RR{rr})

	c := new(dns.Client)
	c.TsigSecret = map[string]string{s.KeyName: s.base64KeyBlob()}
	m.SetTsig(s.KeyName, s.KeyAlgo, 300, time.Now().Unix())

	_, _, err := c.Exchange(m, s.Server)
	return err
}

func (s *DDNSServer) DeleteRR(domain *happydns.Domain, rr dns.RR) error {
	m := new(dns.Msg)
	m.Id = dns.Id()
	m.Opcode = dns.OpcodeUpdate
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{domain.DomainName, dns.TypeSOA, dns.ClassINET}

	m.Remove([]dns.RR{rr})

	c := new(dns.Client)
	c.TsigSecret = map[string]string{s.KeyName: s.base64KeyBlob()}
	m.SetTsig(s.KeyName, s.KeyAlgo, 300, time.Now().Unix())

	_, _, err := c.Exchange(m, s.Server)
	return err
}

func init() {
	sources.RegisterSource("git.happydns.org/happydns/sources/ddns/DDNSServer", func() happydns.Source {
		return &DDNSServer{}
	}, sources.SourceInfos{
		Name:        "Dynamic DNS",
		Description: "If your zone is hosted on an authoritative name server that support Dynamic DNS (RFC 2136), such as Bind, Knot, ...",
	})
}
