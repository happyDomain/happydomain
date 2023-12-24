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

package database

import (
	"fmt"
	"log"
	"strings"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func migrateFrom5(s *LevelDBStorage) (err error) {
	err = migrateFrom5_analyseEMailSvc(s)
	if err != nil {
		return
	}

	err = s.Tidy()
	if err != nil {
		return
	}

	return
}

type ServiceCombined struct {
	Service map[string]interface{}
	happydns.ServiceMeta
}

type Zone struct {
	happydns.ZoneMeta
	Services map[string][]*ServiceCombined `json:"services"`
}

func migrateFrom5_analyseEMailSvc(s *LevelDBStorage) (err error) {
	var zone *Zone

	iter := s.search("domain.zone-")
	for iter.Next() {
		zone = &Zone{}
		err = s.get(string(iter.Key()), &zone)

		changed := false
		for dn, zServices := range zone.Services {
			for ksvc, svc := range zServices {
				if svc.Type == "abstract.EMail" {
					// SPF
					if oldspf, ok := svc.Service["spf"].(map[string]interface{}); ok {
						if content, ok := oldspf["Content"]; ok {
							if cnt, ok := content.(string); ok {
								newspf := &svcs.SPF{}
								newspf.Analyze("v=spf1 " + cnt)
								zone.Services[dn][ksvc].Service["spf"] = newspf
								changed = true
							}
						}
					}

					// DKIM
					newdkim := map[string]*svcs.DKIM{}

					if olddkim, ok := svc.Service["dkim"].(map[string]interface{}); ok {
						for k, v := range olddkim {
							if oldv, ok := v.(map[string]interface{}); ok {
								var fields []string
								for _, f := range oldv["Fields"].([]interface{}) {
									fields = append(fields, f.(string))
								}

								newdkim[k] = &svcs.DKIM{}
								newdkim[k].Analyze(strings.Join(fields, ";"))
								changed = true
							}
						}
					}

					zone.Services[dn][ksvc].Service["dkim"] = newdkim

					// DMARC
					if olddmarc, ok := svc.Service["dmarc"].(map[string]interface{}); ok {
						var fields []string
						for _, f := range olddmarc["Fields"].([]interface{}) {
							fields = append(fields, f.(string))
						}

						newdmarc := &svcs.DMARC{}
						newdmarc.Analyze("v=DMARC1;" + strings.Join(fields, ";"))
						zone.Services[dn][ksvc].Service["dmarc"] = newdmarc
						changed = true
					}

					// MTA-STS
					if oldsts, ok := svc.Service["mta_sts"].(map[string]interface{}); ok {
						var fields []string
						for _, f := range oldsts["Fields"].([]interface{}) {
							fields = append(fields, f.(string))
						}

						newsts := &svcs.MTA_STS{}
						newsts.Analyze(strings.Join(fields, ";"))
						zone.Services[dn][ksvc].Service["mta_sts"] = newsts
						changed = true
					}

					// TLS-RPT
					if oldrpt, ok := svc.Service["tls_rpt"].(map[string]interface{}); ok {
						var fields []string
						for _, f := range oldrpt["Fields"].([]interface{}) {
							fields = append(fields, f.(string))
						}

						newrpt := &svcs.TLS_RPT{}
						newrpt.Analyze(strings.Join(fields, ";"))
						zone.Services[dn][ksvc].Service["tls_rpt"] = newrpt
						changed = true
					}
				}
			}
		}

		if changed {
			err = s.put(string(iter.Key()), zone)
			if err != nil {
				return fmt.Errorf("unable to write %s: %w", iter.Key(), err)
			}
			log.Printf("Migrating v4 -> v5: %s (update abstract.EMail)...", iter.Key())
		}
	}

	return nil
}
