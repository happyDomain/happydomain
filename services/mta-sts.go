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

package svcs

import (
	"fmt"
	"strconv"
	"strings"
)

type MTA_STS struct {
	Version uint   `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of MTA-STS to use.,default=1,hidden"`
	Id      string `json:"id" happydomain:"label=Policy Identifier,placeholder=,description=A short string used to track policy updates."`
}

func (t *MTA_STS) Analyze(txt string) error {
	fields := strings.Split(txt, ";")

	if len(fields) < 2 {
		return fmt.Errorf("not a valid MTA-STS record: should have a version AND a id, only one field found")
	}
	if len(fields) > 3 || (len(fields) == 3 && fields[2] != "") {
		return fmt.Errorf("not a valid MTA-STS record: should have exactly 2 fields: seen %d", len(fields))
	}

	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	if !strings.HasPrefix(fields[0], "v=STSv") {
		return fmt.Errorf("not a valid MTA-STS record: should begin with v=STSv1, seen %q", fields[0])
	}

	version, err := strconv.ParseUint(fields[0][6:], 10, 32)
	if err != nil {
		return fmt.Errorf("not a valid MTA-STS record: bad version number: %w", err)
	}
	t.Version = uint(version)

	if !strings.HasPrefix(fields[1], "id=") {
		return fmt.Errorf("not a valid MTA-STS record: expected id=, found %q", fields[1])
	}

	t.Id = strings.TrimSpace(strings.TrimPrefix(fields[1], "id="))

	return nil
}

func (t *MTA_STS) String() string {
	return fmt.Sprintf("v=STSv%d; id=%s", t.Version, t.Id)
}
