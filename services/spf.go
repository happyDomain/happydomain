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
	"strings"

	"github.com/StackExchange/dnscontrol/v4/pkg/spflib"
)

type SPF struct {
	Version    uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of SPF to use.,default=1,hidden"`
	Directives []string `json:"directives" happydomain:"label=Directives,placeholder=ip4:203.0.113.12"`
}

func (t *SPF) Analyze(txt string) error {
	_, err := spflib.Parse(txt, nil)
	if err != nil {
		return err
	}

	t.Version = 1

	fields := strings.Fields(txt)

	// Avoid doublon
	for _, directive := range fields[1:] {
		exists := false
		for _, known := range t.Directives {
			if known == directive {
				exists = true
				break
			}
		}

		if !exists {
			t.Directives = append(t.Directives, directive)
		}
	}

	return nil
}

func (t *SPF) String() string {
	directives := append([]string{fmt.Sprintf("v=spf%d", t.Version)}, t.Directives...)
	return strings.Join(directives, " ")
}
