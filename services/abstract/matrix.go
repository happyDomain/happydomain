// Copyright or © or Copr. happyDNS (2020)
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

package abstract

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/services"
	"git.happydns.org/happydomain/utils"
)

type MatrixIM struct {
	Matrix []*svcs.SRV `json:"matrix"`
}

func (s *MatrixIM) GetNbResources() int {
	return len(s.Matrix)
}

func (s *MatrixIM) GenComment(origin string) string {
	dest := map[string][]uint16{}

destloop:
	for _, srv := range s.Matrix {
		for _, port := range dest[srv.Target] {
			if port == srv.Port {
				continue destloop
			}
		}
		dest[srv.Target] = append(dest[srv.Target], srv.Port)
	}

	var buffer bytes.Buffer
	first := true
	for dn, ports := range dest {
		dn = strings.TrimSuffix(dn, "."+origin)
		if !first {
			buffer.WriteString("; ")
		} else {
			first = !first
		}
		buffer.WriteString(dn)
		buffer.WriteString(" (")
		firstport := true
		for _, port := range ports {
			if !firstport {
				buffer.WriteString(", ")
			} else {
				firstport = !firstport
			}
			buffer.WriteString(strconv.Itoa(int(port)))
		}
		buffer.WriteString(")")
	}

	return buffer.String()
}

func (s *MatrixIM) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	for _, matrix := range s.Matrix {
		rrs = append(rrs, matrix.GenRRs(utils.DomainJoin("_matrix._tcp", domain), ttl, origin)...)
	}
	return
}

func matrix_analyze(a *svcs.Analyzer) error {
	matrixDomains := map[string]*MatrixIM{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_matrix._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.Header().Name, "_matrix._tcp.")

			if _, ok := matrixDomains[domain]; !ok {
				matrixDomains[domain] = &MatrixIM{}
			}

			matrixDomains[domain].Matrix = append(matrixDomains[domain].Matrix, srv)

			a.UseRR(
				record,
				domain,
				matrixDomains[domain],
			)
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &MatrixIM{}
		},
		matrix_analyze,
		svcs.ServiceInfos{
			Name:        "Matrix IM",
			Description: "Communicate on Matrix using your domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"im",
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		1,
	)
}
