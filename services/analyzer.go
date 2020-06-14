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
	"crypto/sha1"
	"errors"
	"io"
	"reflect"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
)

type Analyzer struct {
	origin     string
	zone       []dns.RR
	services   map[string][]*happydns.ServiceCombined
	aliases    map[string][]string
	defaultTTL uint32
}

type AnalyzerRecordFilter struct {
	Prefix       string
	Domain       string
	SubdomainsOf string
	Contains     string
	Type         uint16
	Class        uint16
	Ttl          uint32
}

func (a *Analyzer) searchRR(arrs ...AnalyzerRecordFilter) (rrs []dns.RR) {
	for _, record := range a.zone {
		for _, arr := range arrs {
			if strings.HasPrefix(record.Header().Name, arr.Prefix) &&
				strings.HasSuffix(record.Header().Name, arr.SubdomainsOf) &&
				(arr.Domain == "" || record.Header().Name == arr.Domain) &&
				(arr.Type == 0 || record.Header().Rrtype == arr.Type) &&
				(arr.Class == 0 || record.Header().Class == arr.Class) &&
				(arr.Ttl == 0 || record.Header().Ttl == arr.Ttl) &&
				(arr.Contains == "" || strings.Contains(record.String(), arr.Contains)) {
				rrs = append(rrs, record)
			}
		}
	}

	return
}

func (a *Analyzer) useRR(rr dns.RR, domain string, svc happydns.Service) error {
	found := false
	for k, record := range a.zone {
		if record == rr {
			found = true
			a.zone[k] = a.zone[len(a.zone)-1]
			a.zone = a.zone[:len(a.zone)-1]
		}
	}

	if !found {
		return errors.New("Record not found.")
	}

	// svc nil, just drop the record from the zone (probably handle another way)
	if svc == nil {
		return nil
	}

	// Remove origin to get an relative domain here
	domain = strings.TrimSuffix(strings.TrimSuffix(domain, "."+a.origin), a.origin)

	for _, service := range a.services[domain] {
		if service.Service == svc {
			service.Comment = svc.GenComment(a.origin)
			service.NbResources = svc.GetNbResources()
			return nil
		}
	}

	hash := sha1.New()
	io.WriteString(hash, rr.String())

	var ttl uint32 = 0
	if rr.Header().Ttl != a.defaultTTL {
		ttl = rr.Header().Ttl
	}

	a.services[domain] = append(a.services[domain], &happydns.ServiceCombined{svc, happydns.ServiceType{
		Id:          hash.Sum(nil),
		Type:        reflect.Indirect(reflect.ValueOf(svc)).Type().String(),
		Domain:      domain,
		Ttl:         ttl,
		Comment:     svc.GenComment(a.origin),
		NbResources: svc.GetNbResources(),
	}})

	return nil
}

func getMostUsedTTL(zone []dns.RR) uint32 {
	ttls := map[uint32]int{}
	for _, rr := range zone {
		ttls[rr.Header().Ttl] += 1
	}

	var max uint32 = 0
	for k, v := range ttls {
		if w, ok := ttls[max]; !ok || v > w {
			max = k
		}
	}

	return max
}

func AnalyzeZone(origin string, zone []dns.RR) (svcs map[string][]*happydns.ServiceCombined, aliases map[string][]string, defaultTTL uint32, err error) {
	defaultTTL = getMostUsedTTL(zone)

	a := Analyzer{
		origin:     origin,
		zone:       zone,
		services:   map[string][]*happydns.ServiceCombined{},
		aliases:    map[string][]string{},
		defaultTTL: defaultTTL,
	}

	// Find services between all registered ones
	for _, service := range OrderedServices() {
		if service.Analyzer == nil {
			continue
		}

		if err = service.Analyzer(&a); err != nil {
			return
		}
	}

	svcs = a.services

	// Consider records not used by services as Orphan
	for _, record := range a.zone {
		// Skip DNSSEC records
		if record.Header().Rrtype == dns.TypeNSEC ||
			record.Header().Rrtype == dns.TypeNSEC3 ||
			record.Header().Rrtype == dns.TypeDNSKEY ||
			record.Header().Rrtype == dns.TypeRRSIG {
			continue
		}

		domain := strings.TrimSuffix(strings.TrimSuffix(record.Header().Name, "."+a.origin), a.origin)

		hash := sha1.New()
		io.WriteString(hash, record.String())

		orphan := &Orphan{record.String()[strings.LastIndex(record.Header().String(), "\tIN\t")+4:]}
		svcs[domain] = append(svcs[domain], &happydns.ServiceCombined{orphan, happydns.ServiceType{
			Id:          hash.Sum(nil),
			Type:        reflect.Indirect(reflect.ValueOf(orphan)).Type().String(),
			Domain:      domain,
			Ttl:         record.Header().Ttl,
			NbResources: 1,
			Comment:     orphan.GenComment(a.origin),
		}})
	}

	aliases = a.aliases

	return
}
