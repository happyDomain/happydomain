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

package svcs

import (
	"fmt"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type SSHFP struct {
	Algorithm   uint8  `json:"algorithm"`
	Type        uint8  `json:"type"`
	FingerPrint string `json:"fingerprint"`
}

type SSHFPs struct {
	SSHFP []*SSHFP `json:"SSHFP,omitempty" happydomain:"label=SSH Fingerprint,description=Server's SSH fingerprint"`
}

func (s *SSHFPs) GetNbResources() int {
	return len(s.SSHFP)
}

func (s *SSHFPs) GenComment() string {
	return fmt.Sprintf("%d fingerprints", len(s.SSHFP))
}

func (s *SSHFPs) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, err error) {
	for _, sshfp := range s.SSHFP {
		rr := utils.NewRecord(domain, "SSHFP", ttl, origin)
		rr.(*dns.SSHFP).Algorithm = sshfp.Algorithm
		rr.(*dns.SSHFP).Type = sshfp.Type
		rr.(*dns.SSHFP).FingerPrint = sshfp.FingerPrint

		rrs = append(rrs, rr)
	}

	return
}

func sshfp_analyze(a *Analyzer) error {
	pool := map[string][]happydns.Record{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSSHFP}) {
		domain := record.Header().Name

		pool[domain] = append(pool[domain], record)
	}

	for dn, rrs := range pool {
		s := &SSHFPs{}

		for _, rr := range rrs {
			if sshfp, ok := rr.(*dns.SSHFP); ok {
				s.SSHFP = append(s.SSHFP, &SSHFP{
					Algorithm:   sshfp.Algorithm,
					Type:        sshfp.Type,
					FingerPrint: sshfp.FingerPrint,
				})

				a.UseRR(rr, dn, s)
			}
		}
	}

	return nil
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &SSHFPs{}
		},
		sshfp_analyze,
		happydns.ServiceInfos{
			Name:        "SSHFP",
			Description: "Store SSH key fingerprints in DNS.",
			Categories: []string{
				"security",
			},
			RecordTypes: []uint16{
				dns.TypeSSHFP,
			},
			Restrictions: happydns.ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeSSHFP,
				},
			},
		},
		1000,
	)
}
