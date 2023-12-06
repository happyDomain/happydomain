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
