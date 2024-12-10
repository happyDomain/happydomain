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
	"strconv"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/common"
	"git.happydns.org/happyDomain/utils"
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

func (t *DMARC) GetNbResources() int {
	return 1
}

func (t *DMARC) GenComment(origin string) string {
	var b strings.Builder

	if t.ADKIM && t.ASPF {
		b.WriteString("strict ")
	} else if t.ADKIM {
		b.WriteString("SPF relaxed ")
	} else if t.ASPF {
		b.WriteString("DKIM relaxed ")
	} else {
		b.WriteString("relaxed ")
	}

	if t.Request != "" {
		b.WriteString(t.Request)
		b.WriteString(" ")
	}

	if t.Percent < 100 {
		b.WriteString(strconv.Itoa(int(t.Percent)))
		b.WriteString(" %")
	}

	return b.String()
}

func (t *DMARC) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rc := utils.NewRecordConfig(utils.DomainJoin("_dmarc", domain), "TXT", ttl, origin)
	rc.SetTargetTXT(t.String())

	rrs = append(rrs, rc)

	return
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

func dmarc_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_dmarc"}) {
		service := &DMARC{}

		err = service.Analyze(record.GetTargetTXTJoined())
		if err != nil {
			return
		}

		err = a.UseRR(record, strings.TrimPrefix(record.NameFQDN, "_dmarc."), service)
		if err != nil {
			return
		}
	}

	return
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &DMARC{}
		},
		dmarc_analyze,
		ServiceInfos{
			Name:        "DMARC",
			Description: "Domain-based Message Authentication, Reporting and Conformance.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
