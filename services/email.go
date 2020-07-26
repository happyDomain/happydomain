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

package svcs

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/utils"
)

type MX struct {
	Target     string `json:"target"`
	Preference uint16 `json:"preference,omitempty"`
}

type SPFDirective struct {
	Qualifier byte
	Mechanism string
}

type SPFModifier struct {
	Name      string
	Mechanism string
}

type SPFExplanation struct {
	DomainSpec string
}

type SPFRedirect struct {
	DomainSpec string
}

type SPF struct {
	Content string
}

func (t *SPF) String() string {
	return t.Content
}

type DKIM struct {
	Fields []string
}

func (t *DKIM) String() string {
	return strings.Join(t.Fields, "; ")
}

type DMARC struct {
	Fields []string
}

func (t *DMARC) String() string {
	return strings.Join(t.Fields, ";")
}

type MTA_STS struct {
	Fields []string
}

func (t *MTA_STS) String() string {
	return strings.Join(t.Fields, ";")
}

type TLS_RPT struct {
	Fields []string
}

func (t *TLS_RPT) String() string {
	return strings.Join(t.Fields, ";")
}

type EMail struct {
	MX      []MX             `json:"mx,omitempty" happydns:"label=EMail Servers"`
	SPF     *SPF             `json:"spf,omitempty" happydns:"label=Sender Policy Framework"`
	DKIM    map[string]*DKIM `json:"dkim,omitempty" happydns:"label=Domain Keys"`
	DMARC   *DMARC           `json:"dmarc,omitempty" happydns:"label=DMARC"`
	MTA_STS *MTA_STS         `json:"mta_sts,omitempty" happydns:"label=Strict Transport Security"`
	TLS_RPT *TLS_RPT         `json:"tls_rpt,omitempty" happydns:"label=TLS Reporting"`
}

func (s *EMail) GetNbResources() int {
	return len(s.MX)
}

func (s *EMail) GenComment(origin string) string {
	poolMX := map[string]int{}

	for _, mx := range s.MX {
		labels := dns.SplitDomainName(mx.Target)
		nbLabel := len(labels)

		var dn string
		if len(labels[nbLabel-2]) < 4 {
			dn = strings.Join(labels[nbLabel-3:], ".") + "."
		} else {
			dn = strings.Join(labels[nbLabel-2:], ".") + "."
		}

		poolMX[dn] += 1
	}

	var buffer bytes.Buffer
	first := true

	for dn, nb := range poolMX {
		if !first {
			buffer.WriteString("; ")
		} else {
			first = !first
		}
		buffer.WriteString(strings.TrimSuffix(dn, "."+origin))
		if nb > 1 {
			buffer.WriteString(fmt.Sprintf(" ×%d", nb))
		}
	}

	if s.SPF != nil {
		buffer.WriteString(" + SPF")
	}

	if s.DKIM != nil {
		buffer.WriteString(" + DKIM")
	}

	if s.DMARC != nil {
		buffer.WriteString(" + DMARC")
	}

	if s.MTA_STS != nil {
		buffer.WriteString(" + MTA-STS")
	}

	if s.TLS_RPT != nil {
		buffer.WriteString(" + TLS Reporting")
	}

	return buffer.String()
}

func (s *EMail) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	if len(s.MX) > 0 {
		for _, mx := range s.MX {
			rrs = append(rrs, &dns.MX{
				Hdr: dns.RR_Header{
					Name:   utils.DomainJoin(domain),
					Rrtype: dns.TypeMX,
					Class:  dns.ClassINET,
					Ttl:    ttl,
				},
				Mx:         utils.DomainFQDN(mx.Target, origin),
				Preference: mx.Preference,
			})
		}
	}

	if s.SPF != nil {
		rrs = append(rrs, &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin(domain),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Txt: utils.SplitN("v=spf1 "+s.SPF.String(), 255),
		})
	}

	for selector, d := range s.DKIM {
		rrs = append(rrs, &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin(selector+"._domainkey", domain),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Txt: utils.SplitN(d.String(), 255),
		})
	}

	if s.DMARC != nil {
		rrs = append(rrs, &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin("_dmarc", domain),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Txt: utils.SplitN(s.DMARC.String(), 255),
		})
	}

	if s.MTA_STS != nil {
		rrs = append(rrs, &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin("_mta-sts", domain),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Txt: utils.SplitN(s.MTA_STS.String(), 255),
		})
	}

	if s.TLS_RPT != nil {
		rrs = append(rrs, &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin("_smtp._tls", domain),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Txt: utils.SplitN(s.TLS_RPT.String(), 255),
		})
	}
	return
}

func email_analyze(a *Analyzer) (err error) {
	services := map[string]*EMail{}

	// Handle only MX records
	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeMX}) {
		if mx, ok := record.(*dns.MX); ok {
			dn := mx.Header().Name

			if _, ok := services[dn]; !ok {
				services[dn] = &EMail{}
			}

			services[dn].MX = append(
				services[dn].MX,
				MX{
					Target:     mx.Mx,
					Preference: mx.Preference,
				},
			)

			err = a.useRR(
				record,
				dn,
				services[dn],
			)
			if err != nil {
				return
			}
		}
	}

	for domain, service := range services {
		// Is there SPF record?
		for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, Domain: domain, Contains: "v=spf1"}) {
			if service.SPF == nil {
				service.SPF = &SPF{}
			}

			if txt, ok := record.(*dns.TXT); ok {
				fields := strings.Fields(service.SPF.Content + " " + strings.TrimPrefix(strings.TrimSpace(strings.Join(txt.Txt, "")), "v=spf1"))

				for i := 0; i < len(fields); i += 1 {
					for j := i + 1; j < len(fields); j += 1 {
						if fields[i] == fields[j] {
							fields = append(fields[:j], fields[j+1:]...)
							j -= 1
						}
					}
				}

				service.SPF.Content = strings.Join(fields, " ")
			}

			err = a.useRR(record, domain, service)
			if err != nil {
				return
			}
		}

		service.DKIM = map[string]*DKIM{}
		// Is there DKIM record?
		for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, SubdomainsOf: "_domainkey." + domain}) {
			selector := strings.TrimSuffix(record.Header().Name, "._domainkey."+domain)

			if _, ok := service.DKIM[selector]; !ok {
				service.DKIM[selector] = &DKIM{}
			}

			if txt, ok := record.(*dns.TXT); ok {
				service.DKIM[selector].Fields = append(service.DKIM[selector].Fields, strings.Split(strings.Join(txt.Txt, ""), ";")...)
			}

			err = a.useRR(record, domain, service)
			if err != nil {
				return
			}
		}

		// Is there DMARC record?
		for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, Domain: "_dmarc." + domain}) {
			if service.DMARC == nil {
				service.DMARC = &DMARC{}
			}

			if txt, ok := record.(*dns.TXT); ok {
				service.DMARC.Fields = append(service.DMARC.Fields, strings.Split(strings.Join(txt.Txt, ""), ";")...)
			}

			err = a.useRR(record, domain, service)
			if err != nil {
				return
			}
		}

		// Is there MTA-STS record?
		for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, Domain: "_mta-sts." + domain}) {
			if service.MTA_STS == nil {
				service.MTA_STS = &MTA_STS{}
			}

			if txt, ok := record.(*dns.TXT); ok {
				service.MTA_STS.Fields = append(service.MTA_STS.Fields, strings.Split(strings.Join(txt.Txt, ""), ";")...)
			}

			err = a.useRR(record, domain, service)
			if err != nil {
				return
			}
		}

		// Is there MTA-STS record?
		for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, Domain: "_smtp._tls." + domain}) {
			if service.TLS_RPT == nil {
				service.TLS_RPT = &TLS_RPT{}
			}

			if txt, ok := record.(*dns.TXT); ok {
				service.TLS_RPT.Fields = append(service.TLS_RPT.Fields, strings.Split(strings.Join(txt.Txt, ""), ";")...)
			}

			err = a.useRR(record, domain, service)
			if err != nil {
				return
			}
		}
	}

	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &EMail{}
		},
		email_analyze,
		ServiceInfos{
			Name:        "E-Mail",
			Description: "Send and receive e-mail with this domain.",
			Categories: []string{
				"email",
			},
			Tabs: true,
			Restrictions: ServiceRestrictions{
				Single: true,
				NeedTypes: []uint16{
					dns.TypeMX,
				},
			},
		},
		1,
	)
}
