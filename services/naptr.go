// Copyright or © or Copr. happyDNS (2023)
//
// contact@happydomain.org
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
	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type NAPTR struct {
	Order       uint16 `json:"order" happydomain:"label=Order,description=The order in which the records must be processed"`
	Preference  uint16 `json:"preference" happydomain:"label=Preference,description=The order in which the records with same order should be processed"`
	Flags       string `json:"flags" happydomain:"label=Flags,choices=S;A;U;P"`
	Service     string `json:"service" happydomain:"label=Service"`
	Regexp      string `json:"regexp" happydomain:"label=Regexp"`
	Replacement string `json:"replacement" happydomain:"label=Replacement"`
}

func (ss *NAPTR) GetNbResources() int {
	return 1
}

func (ss *NAPTR) GenComment(origin string) string {
	return ss.Service
}

func (ss *NAPTR) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(domain, "NAPTR", ttl, origin)
	rr.SetTargetNAPTR(ss.Order, ss.Preference, ss.Flags, ss.Service, ss.Regexp, ss.Replacement)
	rrs = append(rrs, rr)
	return
}

func naptr_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeNAPTR}) {
		if record.Type == "NAPTR" {
			err = a.UseRR(
				record,
				record.NameFQDN,
				&NAPTR{
					Order:       record.NaptrOrder,
					Preference:  record.NaptrPreference,
					Flags:       record.NaptrFlags,
					Service:     record.NaptrService,
					Regexp:      record.NaptrRegexp,
					Replacement: record.GetTargetField(),
				},
			)
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
			return &NAPTR{}
		},
		naptr_analyze,
		ServiceInfos{
			Name: "Naming Authority Pointer",
			RecordTypes: []uint16{
				dns.TypeNAPTR,
			},
			Restrictions: ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeNAPTR,
				},
			},
		},
		100,
	)
}
