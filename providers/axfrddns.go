// Copyright or © or Copr. happyDNS (2021)
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
// As a counterpart to the access to the provider code and rights to copy, modify
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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"encoding/base64"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/axfrddns"

	"git.happydns.org/happyDomain/model"
)

type DDNSServer struct {
	Server  string `json:"server,omitempty" happydomain:"label=Server,placeholder=127.0.0.1"`
	KeyName string `json:"keyname,omitempty" happydomain:"label=Key Name,placeholder=ddns,required"`
	KeyAlgo string `json:"algorithm,omitempty" happydomain:"label=Key Algorithm,default=hmac-sha256,choices=hmac-md5;hmac-sha1;hmac-sha256;hmac-sha512,required"`
	KeyBlob []byte `json:"keyblob,omitempty" happydomain:"label=Secret Key,placeholder=a0b1c2d3e4f5==,required,secret"`
}

func (s *DDNSServer) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"master": s.Server,
	}

	if s.Server == "" {
		config["master"] = "127.0.0.1"
	}

	if s.KeyName != "" {
		config["transfer-key"] = strings.Join([]string{s.KeyAlgo, s.KeyName, base64.StdEncoding.EncodeToString(s.KeyBlob)}, ":")
		config["update-key"] = config["transfer-key"]
	}

	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *DDNSServer) DNSControlName() string {
	return "AXFRDDNS"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &DDNSServer{}
	}, ProviderInfos{
		Name:        "Dynamic DNS",
		Description: "If your zone is hosted on an authoritative name server that support Dynamic DNS (RFC 2136), such as Bind, Knot, ...",
	})
}
