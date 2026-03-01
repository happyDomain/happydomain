// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/abstract"
)

// abstract.EMail
func explodeAbstractEMail(dn happydns.Subdomain, in *happydns.ServiceMessage) ([]*happydns.ServiceMessage, error) {
	var val struct {
		MX      []map[string]interface{} `json:"mx,omitempty"`
		SPF     map[string]interface{}   `json:"spf,omitempty"`
		DKIM    map[string]*svcs.DKIM    `json:"dkim,omitempty"`
		DMARC   *svcs.DMARCFields        `json:"dmarc,omitempty"`
		MTA_STS *svcs.MTASTSFields       `json:"mta_sts,omitempty"`
		TLS_RPT *svcs.TLS_RPTField       `json:"tls_rpt,omitempty"`
	}

	err := json.Unmarshal(in.Service, &val)
	if err != nil {
		return nil, err
	}

	var ret []*happydns.ServiceMessage

	if len(val.MX) > 0 {
		var mxs svcs.MXs

		var rr dns.RR
		for _, mx := range val.MX {
			if _, ok := mx["preference"]; !ok {
				mx["preference"] = 0.0
			}
			rr, err = dns.NewRR(fmt.Sprintf("zZzZ. 0 IN MX %.0f %s", mx["preference"].(float64), helpers.DomainFQDN(mx["target"].(string), "zZzZ.")))
			if err != nil {
				return nil, err
			} else {
				mxs.MXs = append(mxs.MXs, helpers.RRRelative(rr, "zZzZ.").(*dns.MX))
			}
		}

		sm := *in
		sm.Type = "svcs.MXs"
		sm.Id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return nil, err
		}
		sm.Service, err = json.Marshal(mxs)
		if err != nil {
			return nil, err
		}
		ret = append(ret, &sm)
	}

	if val.SPF != nil {
		if _, ok := val.SPF["directives"].([]interface{}); ok {
			directives := val.SPF["directives"].([]interface{})
			var dir []string
			for _, directive := range directives {
				dir = append(dir, directive.(string))
			}

			rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT %q", fmt.Sprintf("v=spf%.0f %s", val.SPF["version"].(float64), strings.Join(dir, " "))))
			if err != nil {
				return nil, err
			}

			sm := *in
			sm.Type = "svcs.SPF"
			sm.Id, err = happydns.NewRandomIdentifier()
			if err != nil {
				return nil, err
			}
			sm.Service, err = json.Marshal(svcs.SPF{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ.").(*dns.TXT))})
			if err != nil {
				return nil, err
			}
			ret = append(ret, &sm)
		}
	}

	if val.DKIM != nil {
		for _, v := range val.DKIM {
			rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT %q", v.String()))
			if err != nil {
				return nil, err
			}

			sm := *in
			sm.Type = "svcs.DKIM"
			sm.Id, err = happydns.NewRandomIdentifier()
			if err != nil {
				return nil, err
			}
			sm.Service, err = json.Marshal(svcs.DKIMRecord{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ.").(*dns.TXT))})
			if err != nil {
				return nil, err
			}
			ret = append(ret, &sm)
		}
	}

	if val.DMARC != nil {
		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT %q", val.DMARC.String()))
		if err != nil {
			return nil, err
		}

		sm := *in
		sm.Type = "svcs.DMARC"
		sm.Id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return nil, err
		}
		sm.Service, err = json.Marshal(svcs.DMARC{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ.").(*dns.TXT))})
		if err != nil {
			return nil, err
		}
		ret = append(ret, &sm)
	}

	if val.MTA_STS != nil {
		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT %q", val.MTA_STS.String()))
		if err != nil {
			return nil, err
		}

		sm := *in
		sm.Type = "svcs.MTA_STS"
		sm.Id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return nil, err
		}
		sm.Service, err = json.Marshal(svcs.MTA_STS{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ.").(*dns.TXT))})
		if err != nil {
			return nil, err
		}
		ret = append(ret, &sm)
	}

	if val.TLS_RPT != nil {
		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT %q", val.TLS_RPT.String()))
		if err != nil {
			return nil, err
		}

		sm := *in
		sm.Type = "svcs.TLS_RPT"
		sm.Id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return nil, err
		}
		sm.Service, err = json.Marshal(svcs.TLS_RPT{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ.").(*dns.TXT))})
		if err != nil {
			return nil, err
		}
		ret = append(ret, &sm)
	}

	return ret, nil
}

var migrateFrom7SvcType map[string]func(json.RawMessage) (json.RawMessage, error)

func migrateFrom7(s *KVStorage) error {
	migrateFrom7SvcType = make(map[string]func(json.RawMessage) (json.RawMessage, error))

	// abstract.ACMEChallenge
	migrateFrom7SvcType["abstract.ACMEChallenge"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var challenge abstract.ACMEChallenge

		rr, err := dns.NewRR(fmt.Sprintf("_acme-challenge.zZzZ 0 IN TXT %q", val["Challenge"].(string)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			challenge.Record = happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))
		}

		return json.Marshal(challenge)
	}

	// abstract.Delegation
	migrateFrom7SvcType["abstract.Delegation"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var delegation abstract.Delegation

		if nss, ok := val["ns"].([]interface{}); ok {
			for _, ns := range nss {
				rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN NS %s", helpers.DomainFQDN(ns.(string), "zZzZ")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					delegation.NameServers = append(delegation.NameServers, helpers.RRRelative(rr, "zZzZ").(*dns.NS))
				}
			}
		}

		if dss, ok := val["ds"].([]interface{}); ok {
			for _, dsI := range dss {
				ds := dsI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN DS %.0f %.0f %.0f %s", ds["keytag"].(float64), ds["algorithm"].(float64), ds["digestType"].(float64), ds["digest"].(string)))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					delegation.DS = append(delegation.DS, helpers.RRRelative(rr, "zZzZ").(*dns.DS))
				}
			}
		}

		return json.Marshal(delegation)
	}

	// abstract.GithubOrgVerif
	migrateFrom7SvcType["abstract.GithubOrgVerif"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var challenge abstract.GithubOrgVerif

		rr, err := dns.NewRR(fmt.Sprintf("_github-challenge-%s-org.zZzZ. 0 IN TXT %q", val["OrganizationName"].(string), val["Code"].(string)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			challenge.Record = happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))
		}

		return json.Marshal(challenge)
	}

	// abstract.GitlabPageVerif
	migrateFrom7SvcType["abstract.GitlabPageVerif"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var challenge abstract.GitlabPageVerif

		rr, err := dns.NewRR(fmt.Sprintf("_gitlab-pages-verification-code.zZzZ. 0 IN TXT gitlab-pages-verification-code=%q", val["Code"].(string)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			challenge.Record = happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))
		}

		return json.Marshal(challenge)
	}

	// abstract.GoogleVerif
	migrateFrom7SvcType["abstract.GoogleVerif"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var challenge abstract.GoogleVerif

		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT google-site-verification=%s", strings.TrimPrefix(val["SiteVerification"].(string), "google-site-verification=")))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			challenge.Record = happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))
		}

		return json.Marshal(challenge)
	}

	// abstract.KeybaseVerif
	migrateFrom7SvcType["abstract.KeybaseVerif"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var challenge abstract.KeybaseVerif

		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT keybase-site-verification=%s", val["SiteVerification"].(string)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			challenge.Record = happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))
		}

		return json.Marshal(challenge)
	}

	// abstract.MatrixIM
	migrateFrom7SvcType["abstract.MatrixIM"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var matrix abstract.MatrixIM

		if mat, ok := val["matrix"].([]interface{}); ok {
			for _, mI := range mat {
				m := mI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("_matrix._tcp.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", m["priority"].(float64), m["weight"].(float64), m["port"].(float64), helpers.DomainFQDN(m["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					matrix.Records = append(matrix.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}

		return json.Marshal(matrix)
	}

	// abstract.OpenPGP
	migrateFrom7SvcType["abstract.OpenPGP"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var okey abstract.OpenPGP

		rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN OPENPGPKEY %s", helpers.DomainJoin(val["identifier"].(string), "_openpgpkey"), val["pubkey"].(string)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			if _, ok := val["username"]; ok {
				okey.Username = val["username"].(string)
			}
			okey.Record = helpers.RRRelative(rr, "zZzZ").(*dns.OPENPGPKEY)
		}

		return json.Marshal(okey)
	}

	// abstract.SMimeCert
	migrateFrom7SvcType["abstract.SMimeCert"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var okey abstract.SMimeCert

		rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ 0 IN SMIMEA %.0f %.0f %.0f %s", helpers.DomainJoin(val["identifier"].(string), "_smimecert"), val["certusage"].(float64), val["selector"].(float64), val["matchingtype"].(float64), val["certificate"].(string)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			if _, ok := val["username"]; ok {
				okey.Username = val["username"].(string)
			}
			okey.Record = helpers.RRRelative(rr, "zZzZ").(*dns.SMIMEA)
		}

		return json.Marshal(okey)
	}

	// abstract.Origin
	migrateFrom7SvcType["abstract.Origin"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var origin abstract.Origin

		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN SOA %s %s %.0f %.0f %.0f %.0f %.0f", helpers.DomainFQDN(val["mname"].(string), "zZzZ."), helpers.DomainFQDN(val["rname"].(string), "zZzZ."), val["serial"].(float64), val["refresh"].(float64), val["retry"].(float64), val["expire"].(float64), val["nxttl"].(float64)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			origin.SOA = helpers.RRRelative(rr, "zZzZ").(*dns.SOA)
		}

		if _, ok := val["ns"].([]interface{}); ok {
			for _, nsI := range val["ns"].([]interface{}) {
				rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN NS %s", helpers.DomainFQDN(nsI.(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					origin.NameServers = append(origin.NameServers, helpers.RRRelative(rr, "zZzZ").(*dns.NS))
				}
			}
		}

		return json.Marshal(origin)
	}

	// abstract.NSOnlyOrigin
	migrateFrom7SvcType["abstract.NSOnlyOrigin"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var origin abstract.NSOnlyOrigin

		if _, ok := val["ns"].([]interface{}); ok {
			for _, nsI := range val["ns"].([]interface{}) {
				rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN NS %s", helpers.DomainFQDN(nsI.(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					origin.NameServers = append(origin.NameServers, helpers.RRRelative(rr, "zZzZ").(*dns.NS))
				}
			}
		}

		return json.Marshal(origin)
	}

	// abstract.RFC6186
	migrateFrom7SvcType["abstract.RFC6186"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var rfc6186 abstract.RFC6186

		if _, ok := val["submission"].([]interface{}); ok {
			for _, clientI := range val["submission"].([]interface{}) {
				client := clientI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", helpers.DomainJoin("_submission", "_tcp"), client["priority"].(float64), client["weight"].(float64), client["port"].(float64), helpers.DomainFQDN(client["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					rfc6186.Records = append(rfc6186.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}
		if _, ok := val["imaps"].([]interface{}); ok {
			for _, clientI := range val["imaps"].([]interface{}) {
				client := clientI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", helpers.DomainJoin("_imaps", "_tcp"), client["priority"].(float64), client["weight"].(float64), client["port"].(float64), helpers.DomainFQDN(client["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					rfc6186.Records = append(rfc6186.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}
		if _, ok := val["pop3s"].([]interface{}); ok {
			for _, clientI := range val["pop3s"].([]interface{}) {
				client := clientI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", helpers.DomainJoin("_pop3s", "_tcp"), client["priority"].(float64), client["weight"].(float64), client["port"].(float64), helpers.DomainFQDN(client["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					rfc6186.Records = append(rfc6186.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}
		if _, ok := val["submissions"].([]interface{}); ok {
			for _, clientI := range val["submissions"].([]interface{}) {
				client := clientI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", helpers.DomainJoin("_submissions", "_tcp"), client["priority"].(float64), client["weight"].(float64), client["port"].(float64), helpers.DomainFQDN(client["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					rfc6186.Records = append(rfc6186.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}
		if _, ok := val["imap"].([]interface{}); ok {
			for _, clientI := range val["imap"].([]interface{}) {
				client := clientI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", helpers.DomainJoin("_imap", "_tcp"), client["priority"].(float64), client["weight"].(float64), client["port"].(float64), helpers.DomainFQDN(client["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					rfc6186.Records = append(rfc6186.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}
		if _, ok := val["pop3"].([]interface{}); ok {
			for _, clientI := range val["pop3"].([]interface{}) {
				client := clientI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", helpers.DomainJoin("_pop3", "_tcp"), client["priority"].(float64), client["weight"].(float64), client["port"].(float64), helpers.DomainFQDN(client["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					rfc6186.Records = append(rfc6186.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}

		return json.Marshal(rfc6186)
	}

	// abstract.ScalewayChallenge
	migrateFrom7SvcType["abstract.ScalewayChallenge"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var challenge abstract.ScalewayChallenge

		rr, err := dns.NewRR(fmt.Sprintf("_scaleway-challenge.zZzZ. 0 IN TXT %q", val["Code"].(string)))
		if err != nil {
			return nil, err
		}
		if rr != nil {
			challenge.Record = happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))
		}

		return json.Marshal(challenge)
	}

	// abstract.Server
	migrateFrom7SvcType["abstract.Server"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var server abstract.Server

		if _, ok := val["A"].(string); ok {
			rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN A %s", val["A"].(string)))
			if err != nil {
				return nil, err
			}
			if rr != nil {
				server.A = helpers.RRRelative(rr, "zZzZ").(*dns.A)
			}
		}

		if aaaa, ok := val["AAAA"].(string); ok && len(aaaa) > 0 {
			rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN AAAA %s", val["AAAA"].(string)))
			if err != nil {
				return nil, err
			}
			if rr != nil {
				server.AAAA = helpers.RRRelative(rr, "zZzZ").(*dns.AAAA)
			}
		}

		if _, ok := val["SSHFP"].([]interface{}); ok {
			for _, sshfpI := range val["SSHFP"].([]interface{}) {
				sshfp := sshfpI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN SSHFP %.0f %.0f %s", sshfp["algorithm"], sshfp["type"], sshfp["fingerprint"]))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					server.SSHFP = append(server.SSHFP, helpers.RRRelative(rr, "zZzZ").(*dns.SSHFP))
				}
			}
		}

		return json.Marshal(server)
	}

	// abstract.XMPP
	migrateFrom7SvcType["abstract.XMPP"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var xmpp abstract.XMPP

		if _, ok := val["Client"].([]interface{}); ok {
			for _, clientI := range val["Client"].([]interface{}) {
				client := clientI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("_xmpp-client._tcp.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", client["priority"], client["weight"], client["port"], helpers.DomainFQDN(client["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					xmpp.Records = append(xmpp.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}

		if _, ok := val["Server"].([]interface{}); ok {
			for _, serverI := range val["Server"].([]interface{}) {
				server := serverI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("_xmpp-server._tcp.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", server["priority"], server["weight"], server["port"], helpers.DomainFQDN(server["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					xmpp.Records = append(xmpp.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}

		if _, ok := val["Jabber"].([]interface{}); ok {
			for _, jabberI := range val["Jabber"].([]interface{}) {
				jabber := jabberI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("_jabber._tcp.zZzZ. 0 IN SRV %.0f %.0f %.0f %s", jabber["priority"], jabber["weight"], jabber["port"], helpers.DomainFQDN(jabber["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					xmpp.Records = append(xmpp.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}

		return json.Marshal(xmpp)
	}

	// svcs.CAA
	migrateFrom7SvcType["svcs.CAA"] = func(in json.RawMessage) (json.RawMessage, error) {
		s := svcs.CAAFields{}

		err := json.Unmarshal(in, &s)
		if err != nil {
			return nil, err
		}

		var rrs []*dns.CAA

		if s.DisallowIssue {
			rr := helpers.NewRecord("zZzZ", "CAA", 0, "")
			rr.(*dns.CAA).Flag = 0
			rr.(*dns.CAA).Tag = "issue"
			rr.(*dns.CAA).Value = ";"

			rrs = append(rrs, helpers.RRRelative(rr, "zZzZ").(*dns.CAA))
		} else {
			for _, issue := range s.Issue {
				rr := helpers.NewRecord("zZzZ", "CAA", 0, "")
				rr.(*dns.CAA).Flag = 0
				rr.(*dns.CAA).Tag = "issue"
				rr.(*dns.CAA).Value = issue.String()

				rrs = append(rrs, helpers.RRRelative(rr, "zZzZ").(*dns.CAA))
			}

			if s.DisallowWildcardIssue {
				rr := helpers.NewRecord("zZzZ", "CAA", 0, "")
				rr.(*dns.CAA).Flag = 0
				rr.(*dns.CAA).Tag = "issuewild"
				rr.(*dns.CAA).Value = ";"

				rrs = append(rrs, helpers.RRRelative(rr, "zZzZ").(*dns.CAA))
			} else {
				for _, issue := range s.IssueWild {
					rr := helpers.NewRecord("zZzZ", "CAA", 0, "")
					rr.(*dns.CAA).Flag = 0
					rr.(*dns.CAA).Tag = "issuewild"
					rr.(*dns.CAA).Value = issue.String()

					rrs = append(rrs, helpers.RRRelative(rr, "zZzZ").(*dns.CAA))
				}
			}
		}

		if s.DisallowMailIssue {
			rr := helpers.NewRecord("zZzZ", "CAA", 0, "")
			rr.(*dns.CAA).Flag = 0
			rr.(*dns.CAA).Tag = "issuemail"
			rr.(*dns.CAA).Value = ";"

			rrs = append(rrs, helpers.RRRelative(rr, "zZzZ").(*dns.CAA))
		} else {
			for _, issue := range s.IssueMail {
				rr := helpers.NewRecord("zZzZ", "CAA", 0, "")
				rr.(*dns.CAA).Flag = 0
				rr.(*dns.CAA).Tag = "issuemail"
				rr.(*dns.CAA).Value = issue.String()

				rrs = append(rrs, helpers.RRRelative(rr, "zZzZ").(*dns.CAA))
			}
		}

		if len(s.Iodef) > 0 {
			for _, iodef := range s.Iodef {
				rr := helpers.NewRecord("zZzZ", "CAA", 0, "")
				rr.(*dns.CAA).Flag = 0
				rr.(*dns.CAA).Tag = "iodef"
				rr.(*dns.CAA).Value = iodef.String()

				rrs = append(rrs, helpers.RRRelative(rr, "zZzZ").(*dns.CAA))
			}
		}

		return json.Marshal(svcs.CAAPolicy{Records: rrs})
	}

	// svcs.CNAME
	migrateFrom7SvcType["svcs.CNAME"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]string{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN CNAME %s", helpers.DomainFQDN(val["Target"], "zZzZ.")))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.CNAME{Record: helpers.RRRelative(rr, "zZzZ").(*dns.CNAME)})
	}

	// svcs.DKIMRecord
	migrateFrom7SvcType["svcs.DKIMRecord"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := struct {
			Selector string `json:"selector"`
			svcs.DKIM
		}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("%s 0 IN TXT %q", helpers.DomainJoin(val.Selector, "_domainkey", "zZzZ"), val.DKIM.String()))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.DKIMRecord{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))})
	}

	// svcs.SpecialCNAME
	migrateFrom7SvcType["svcs.SpecialCNAME"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]string{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("%s.zZzZ. 0 IN CNAME %s", val["SubDomain"], helpers.DomainFQDN(val["Target"], "zZzZ.")))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.CNAME{Record: helpers.RRRelative(rr, "zZzZ").(*dns.CNAME)})
	}

	// svcs.DMARC
	migrateFrom7SvcType["svcs.DMARC"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := svcs.DMARCFields{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("_dmarc.zZzZ. 0 IN TXT %q", val.String()))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.DMARC{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))})
	}

	// svcs.MXs
	migrateFrom7SvcType["svcs.MXs"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var mxs svcs.MXs

		for _, mxI := range val["mx"].([]interface{}) {
			mx := mxI.(map[string]interface{})
			if _, ok := mx["preference"]; !ok {
				mx["preference"] = 0.0
			}
			rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN MX %.0f %s", mx["preference"].(float64), mx["target"].(string)))
			if err != nil {
				return nil, err
			} else {
				mxs.MXs = append(mxs.MXs, helpers.RRRelative(rr, "zZzZ").(*dns.MX))
			}
		}

		return json.Marshal(mxs)
	}

	// svcs.SPF
	migrateFrom7SvcType["svcs.SPF"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		directives := val["directives"].([]interface{})
		var dir []string
		for _, directive := range directives {
			dir = append(dir, directive.(string))
		}

		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT %q", fmt.Sprintf("v=spf%.0f %s", val["version"].(float64), strings.Join(dir, " "))))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.SPF{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))})
	}

	// svcs.TLSAs
	migrateFrom7SvcType["svcs.TLSAs"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var tlsa svcs.TLSAs

		for _, tlsaI := range val["tlsa"].([]interface{}) {
			t := tlsaI.(map[string]interface{})
			rr, err := dns.NewRR(fmt.Sprintf("%s 0 IN TLSA %.0f %.0f %.0f %s", helpers.DomainJoin(fmt.Sprintf("_%.0f._%s.zZzZ.", t["port"].(float64), t["proto"].(string))), t["certusage"].(float64), t["selector"].(float64), t["matchingtype"].(float64), t["certificate"].(string)))
			if err != nil {
				return nil, err
			}
			if rr != nil {
				tlsa.Records = append(tlsa.Records, helpers.RRRelative(rr, "zZzZ").(*dns.TLSA))
			}
		}

		return json.Marshal(tlsa)
	}

	// svcs.MTA_STS
	migrateFrom7SvcType["svcs.MTA_STS"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := svcs.MTASTSFields{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("_mta-sts.zZzZ. 0 IN TXT %q", val.String()))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.MTA_STS{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))})
	}

	// svcs.TLS_RPT
	migrateFrom7SvcType["svcs.TLS_RPT"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := svcs.TLS_RPTField{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("_smtp._tls.zZzZ. 0 IN TXT %q", val.String()))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.TLS_RPT{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))})
	}

	// svcs.TXT
	migrateFrom7SvcType["svcs.TXT"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]string{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN TXT %q", val["content"]))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.TXT{Record: happydns.NewTXT(helpers.RRRelative(rr, "zZzZ").(*dns.TXT))})
	}

	// svcs.UnknownSRV
	migrateFrom7SvcType["svcs.UnknownSRV"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]interface{}{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		var usrv svcs.UnknownSRV

		if mat, ok := val["srv"].([]interface{}); ok {
			for _, mI := range mat {
				m := mI.(map[string]interface{})
				rr, err := dns.NewRR(fmt.Sprintf("%s 0 IN SRV %.0f %.0f %.0f %s", helpers.DomainJoin("_"+val["name"].(string), "_"+val["proto"].(string), "zZzZ"), m["priority"].(float64), m["weight"].(float64), m["port"].(float64), helpers.DomainFQDN(m["target"].(string), "zZzZ.")))
				if err != nil {
					return nil, err
				}
				if rr != nil {
					usrv.Records = append(usrv.Records, helpers.RRRelative(rr, "zZzZ").(*dns.SRV))
				}
			}
		}

		return json.Marshal(usrv)
	}

	// svcs.Orphan
	migrateFrom7SvcType["svcs.Orphan"] = func(in json.RawMessage) (json.RawMessage, error) {
		val := map[string]string{}

		err := json.Unmarshal(in, &val)
		if err != nil {
			return nil, err
		}

		rr, err := dns.NewRR(fmt.Sprintf("zZzZ. 0 IN %s %s", val["Type"], val["RR"]))
		if err != nil {
			return nil, err
		}

		return json.Marshal(svcs.Orphan{Record: helpers.RRRelative(rr, "zZzZ")})
	}

	zones, err := s.ListAllZones()
	if err != nil {
		return err
	}

	for zones.Next() {
		zone := zones.Item()
		for _, svcs := range zone.Services {
			changed := false

			for i, svc := range svcs {
				if m, ok := migrateFrom7SvcType[svc.Type]; ok {
					svcs[i].Service, err = m(svc.Service)
					if err != nil {
						return err
					}

					if svc.Type == "svcs.CAA" {
						svcs[i].Type = "svcs.CAAPolicy"
					}

					changed = true
				}
			}

			if changed {
				// Save zone
				err = s.UpdateZoneMessage(zone)
				if err != nil {
					return err
				}
				log.Printf("Migrated zone %s", zone.Id.String())
			}
		}
	}

	zones, err = s.ListAllZones()
	if err != nil {
		return err
	}

	for zones.Next() {
		zone := zones.Item()
		for sb, svcs := range zone.Services {
			for i, svc := range svcs {
				// Explode abstract.EMail
				if svc.Type == "abstract.EMail" {
					newsvcs, err := explodeAbstractEMail(sb, svc)
					if err != nil {
						return err
					}

					// Remove svc[i]
					svcs = append(svcs[:i], svcs[i+1:]...)

					// Append each new services
					for _, ns := range newsvcs {
						svcs = append(svcs, ns)
					}

					// Save zone
					err = s.UpdateZoneMessage(zone)
					if err != nil {
						return err
					}
					log.Printf("Migrated zone %s", zone.Id.String())
					break
				}
			}
		}
	}

	return nil
}
