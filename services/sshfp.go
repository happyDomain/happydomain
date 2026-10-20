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

	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/common"
)

type SSHFPs struct {
	SSHFP []*dns.SSHFP `json:"SSHFP,omitempty" happydomain:"label=SSH Fingerprint,description=Server's SSH fingerprint"`
}

func (s *SSHFPs) GetNbResources() int {
	return len(s.SSHFP)
}

func (s *SSHFPs) GenComment() string {
	return fmt.Sprintf("%d fingerprints", len(s.SSHFP))
}

func (s *SSHFPs) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, err error) {
	for _, sshfp := range s.SSHFP {
		// Services saved before the body carried whole records only hold
		// algorithm, type and fingerprint, which decode into an empty header.
		if sshfp.Hdr.Rrtype == 0 {
			rr := *sshfp
			rr.Hdr.Rrtype = dns.TypeSSHFP
			rr.Hdr.Class = dns.ClassINET
			sshfp = &rr
		}

		rrs = append(rrs, sshfp)
	}

	return
}

func sshfp_analyze(a *svc.Analyzer) error {
	pool := map[string][]happydns.Record{}

	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeSSHFP}) {
		domain := record.Header().Name

		pool[domain] = append(pool[domain], record)
	}

	for dn, rrs := range pool {
		s := &SSHFPs{}

		for _, rr := range rrs {
			if sshfp, ok := rr.(*dns.SSHFP); ok {
				s.SSHFP = append(s.SSHFP, common.SSHFPRelative([]*dns.SSHFP{sshfp}, a.GetOrigin(), dn)...)

				if err := a.UseRR(rr, dn, s); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &SSHFPs{}
		},
		sshfp_analyze,
		happydns.ServiceInfos{
			Name:        "SSHFP",
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
