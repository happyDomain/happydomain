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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
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

func (s *SSHFPs) GenComment(origin string) string {
	return fmt.Sprintf("%d fingerprints", len(s.SSHFP))
}

func (s *SSHFPs) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, sshfp := range s.SSHFP {
		rc := utils.NewRecordConfig(domain, "SSHFP", ttl, origin)
		rc.SshfpAlgorithm = sshfp.Algorithm
		rc.SshfpFingerprint = sshfp.Type
		rc.SetTarget(sshfp.FingerPrint)

		rrs = append(rrs, rc)
	}

	return
}

func sshfp_analyze(a *Analyzer) error {
	pool := map[string]models.Records{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSSHFP}) {
		domain := record.NameFQDN

		pool[domain] = append(pool[domain], record)
	}

	for dn, rrs := range pool {
		s := &SSHFPs{}

		for _, rr := range rrs {
			if rr.Type == "SSHFP" {
				s.SSHFP = append(s.SSHFP, &SSHFP{
					Algorithm:   rr.SshfpAlgorithm,
					Type:        rr.SshfpFingerprint,
					FingerPrint: rr.GetTargetField(),
				})

				a.UseRR(rr, dn, s)
			}
		}
	}

	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &SSHFPs{}
		},
		sshfp_analyze,
		ServiceInfos{
			Name:        "SSHFP",
			Description: "Store SSH key fingerprints in DNS.",
			Categories: []string{
				"security",
			},
			RecordTypes: []uint16{
				dns.TypeSSHFP,
			},
			Restrictions: ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeSSHFP,
				},
			},
		},
		1000,
	)
}
