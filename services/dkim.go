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
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

type DKIM struct {
	Version        uint     `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of DKIM to use.,default=1,hidden"`
	AcceptableHash []string `json:"h" happydomain:"label=Hash Algorithms,choices=*;sha1;sha256"`
	KeyType        string   `json:"k" happydomain:"label=Key Type,choices=rsa"`
	Notes          string   `json:"n" happydomain:"label=Notes,description=Notes intended for a foreign postmaster"`
	PublicKey      []byte   `json:"p" happydomain:"label=Public Key,placeholder=a0b1c2d3e4f5==,required"`
	ServiceType    []string `json:"s" happydomain:"label=Service Types,choices=*;email"`
	Flags          []string `json:"t" happydomain:"label=Flags,choices=y;s"`
}

func (t *DKIM) Analyze(txt string) error {
	fields := analyseFields(txt)

	if v, ok := fields["v"]; ok {
		if !strings.HasPrefix(v, "DKIM") {
			return fmt.Errorf("not a valid DKIM record: should begin with v=DKIMv1, seen v=%q", v)
		}

		version, err := strconv.ParseUint(v[4:], 10, 32)
		if err != nil {
			return fmt.Errorf("not a valid DKIM record: bad version number: %w", err)
		}
		t.Version = uint(version)
	} else {
		return fmt.Errorf("not a valid DKIM record: version not found")
	}

	if h, ok := fields["h"]; ok {
		t.AcceptableHash = strings.Split(h, ":")
	} else {
		t.AcceptableHash = []string{"*"}
	}
	if k, ok := fields["k"]; ok {
		t.KeyType = k
	}
	if n, ok := fields["n"]; ok {
		t.Notes = n
	}
	if p, ok := fields["p"]; ok {
		var err error
		t.PublicKey, err = base64.StdEncoding.DecodeString(p)
		if err != nil {
			return fmt.Errorf("not a valid DKIM record: public key is not base64 valid: %w", err)
		}
	}
	if s, ok := fields["s"]; ok {
		t.ServiceType = strings.Split(s, ":")
	} else {
		t.ServiceType = []string{"*"}
	}
	if f, ok := fields["t"]; ok {
		t.Flags = strings.Split(f, ":")
	}

	return nil
}

func (t *DKIM) String() string {
	fields := []string{
		fmt.Sprintf("v=DKIM%d", t.Version),
	}

	if len(t.AcceptableHash) > 1 || (len(t.AcceptableHash) > 0 && t.AcceptableHash[0] != "*") {
		fields = append(fields, fmt.Sprintf("h=%s", strings.Join(t.AcceptableHash, ":")))
	}
	if t.KeyType != "" {
		fields = append(fields, fmt.Sprintf("k=%s", t.KeyType))
	}
	if t.Notes != "" {
		fields = append(fields, fmt.Sprintf("n=%s", t.Notes))
	}
	if len(t.PublicKey) > 0 {
		fields = append(fields, fmt.Sprintf("p=%s", base64.StdEncoding.EncodeToString(t.PublicKey)))
	}
	if len(t.ServiceType) > 1 || (len(t.ServiceType) > 0 && t.ServiceType[0] != "*") {
		fields = append(fields, fmt.Sprintf("s=%s", strings.Join(t.ServiceType, ":")))
	}
	if len(t.Flags) > 0 {
		fields = append(fields, fmt.Sprintf("t=%s", strings.Join(t.Flags, ":")))
	}

	return strings.Join(fields, ";")
}
