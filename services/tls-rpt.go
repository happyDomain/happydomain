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
	"fmt"
	"strconv"
	"strings"
)

type TLS_RPT struct {
	Version uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of TLSRPT to use.,default=1,hidden"`
	Rua     []string `json:"rua" happydomain:"label=Aggregate Report URI,placeholder=https://example.com/path|mailto:name@example.com"`
}

func (t *TLS_RPT) Analyze(txt string) error {
	fields := strings.Split(txt, ";")

	if len(fields) < 2 {
		return fmt.Errorf("not a valid TLS-RPT record: should have a version AND a rua, only one field found")
	}
	if len(fields) > 3 || (len(fields) == 3 && fields[2] != "") {
		return fmt.Errorf("not a valid TLS-RPT record: should have exactly 2 fields: seen %d", len(fields))
	}

	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	if !strings.HasPrefix(fields[0], "v=TLSRPTv") {
		return fmt.Errorf("not a valid TLS-RPT record: should begin with v=TLSRPTv1, seen %q", fields[0])
	}

	version, err := strconv.ParseUint(fields[0][9:], 10, 32)
	if err != nil {
		return fmt.Errorf("not a valid TLS-RPT record: bad version number: %w", err)
	}
	t.Version = uint(version)

	if !strings.HasPrefix(fields[1], "rua=") {
		return fmt.Errorf("not a valid TLS-RPT record: expected rua=, found %q", fields[1])
	}

	t.Rua = strings.Split(strings.TrimPrefix(fields[1], "rua="), ",")

	for i := range t.Rua {
		t.Rua[i] = strings.TrimSpace(t.Rua[i])
	}

	return nil
}

func (t *TLS_RPT) String() string {
	return fmt.Sprintf("v=TLSRPTv%d; rua=%s", t.Version, strings.Join(t.Rua, ","))
}
