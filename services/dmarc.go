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

	"git.happydns.org/happyDomain/services/common"
)

type DMARC struct {
	Version           uint            `json:"version" happydomain:"label=Version,placeholder=1,required,description=The version of DMARC to use.,default=1,hidden"`
	Request           string          `json:"p" happydomain:"label=Requested Mail Receiver policy,choices=none;quarantine;reject,description=Indicates the policy to be enacted by the Receiver,required"`
	SRequest          string          `json:"sp" happydomain:"label=Requested Mail Receiver policy for all subdomains,choices=;none;quarantaine;reject,description=Indicates the policy to be enacted by the Receiver when it receives mail for a subdomain"`
	AURI              []string        `json:"rua" happydomain:"label=RUA,description=Addresses for aggregate feedback,placeholder=mailto:name@example.com"`
	FURI              []string        `json:"ruf" happydomain:"label=RUF,description=Addresses for message-specific failure information,placeholder=mailto:name@example.com"`
	ADKIM             bool            `json:"adkim" happydomain:"label=Strict DKIM Alignment"`
	ASPF              bool            `json:"aspf" happydomain:"label=Strict SPF Alignment"`
	AInterval         common.Duration `json:"ri" happydomain:"label=Interval between aggregate reports"`
	FailureOptions    []string        `json:"fo" happydomain:"label=Failure reporting options,choices=0;1;d;s"`
	RegisteredFormats []string        `json:"rf" happydomain:"label=Format of the failure reports,choices=;afrf"`
	Percent           uint8           `json:"pct" happydomain:"label=Policy applies on,description=Percentage of messages to which the DMARC policy is to be applied.,unit=%"`
}

func analyseFields(txt string) map[string]string {
	ret := map[string]string{}

	for _, f := range strings.Split(txt, ";") {
		f = strings.TrimSpace(f)

		kv := strings.SplitN(f, "=", 2)
		if len(kv) == 1 {
			ret[strings.TrimSpace(kv[0])] = ""
		} else {
			ret[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return ret
}

func (t *DMARC) Analyze(txt string) error {
	fields := analyseFields(txt)

	if v, ok := fields["v"]; ok {
		if !strings.HasPrefix(v, "DMARC") {
			return fmt.Errorf("not a valid DMARC record: should begin with v=DMARCv1, seen v=%q", v)
		}

		version, err := strconv.ParseUint(v[5:], 10, 32)
		if err != nil {
			return fmt.Errorf("not a valid DMARC record: bad version number: %w", err)
		}
		t.Version = uint(version)
	} else {
		return fmt.Errorf("not a valid DMARC record: version not found")
	}

	if p, ok := fields["p"]; ok {
		t.Request = p
	}
	if sp, ok := fields["sp"]; ok {
		t.SRequest = sp
	}
	if rua, ok := fields["rua"]; ok {
		t.AURI = strings.Split(rua, ",")
	}
	if ruf, ok := fields["ruf"]; ok {
		t.FURI = strings.Split(ruf, ",")
	}
	if adkim, ok := fields["adkim"]; ok && adkim == "s" {
		t.ADKIM = true
	}
	if aspf, ok := fields["aspf"]; ok && aspf == "s" {
		t.ASPF = true
	}
	if ri, ok := fields["ri"]; ok {
		v, err := strconv.ParseUint(ri, 10, 32)
		if err != nil {
			return fmt.Errorf("not a valid DMARC record: bad interval value (ri): %w", err)
		}

		t.AInterval = common.Duration(v)
	} else {
		t.AInterval = 86400
	}
	if fo, ok := fields["fo"]; ok {
		t.FailureOptions = strings.Split(fo, ":")
	}
	if rf, ok := fields["rf"]; ok {
		t.RegisteredFormats = strings.Split(rf, ":")
	}
	if pct, ok := fields["pct"]; ok {
		v, err := strconv.ParseUint(pct, 10, 8)
		if err != nil {
			return fmt.Errorf("not a valid DMARC record: bad percent value (prc): %w", err)
		}

		t.Percent = uint8(v)
	} else {
		t.Percent = 100
	}

	return nil
}

func (t *DMARC) String() string {
	fields := []string{
		fmt.Sprintf("v=DMARC%d", t.Version),
	}

	if t.Request != "" {
		fields = append(fields, fmt.Sprintf("p=%s", t.Request))
	}
	if t.SRequest != "" {
		fields = append(fields, fmt.Sprintf("sp=%s", t.SRequest))
	}
	if len(t.AURI) > 0 {
		fields = append(fields, fmt.Sprintf("rua=%s", strings.Join(t.AURI, ",")))
	}
	if len(t.FURI) > 0 {
		fields = append(fields, fmt.Sprintf("ruf=%s", strings.Join(t.FURI, ",")))
	}
	if t.ADKIM {
		fields = append(fields, "adkim=s")
	} else {
		fields = append(fields, "adkim=r")
	}
	if t.ASPF {
		fields = append(fields, "aspf=s")
	} else {
		fields = append(fields, "aspf=r")
	}
	if t.AInterval != 86400 && t.AInterval != 0 {
		fields = append(fields, fmt.Sprintf("ri=%d", t.AInterval))
	}
	if len(t.FailureOptions) > 0 {
		fields = append(fields, fmt.Sprintf("fo=%s", strings.Join(t.FailureOptions, ":")))
	}
	if len(t.RegisteredFormats) > 0 {
		fields = append(fields, fmt.Sprintf("rf=%s", strings.Join(t.RegisteredFormats, ":")))
	}
	if t.Percent != 100 {
		fields = append(fields, fmt.Sprintf("pct=%d", t.Percent))
	}

	return strings.Join(fields, ";")
}
