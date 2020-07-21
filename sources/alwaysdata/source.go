// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

package alwaysdata // import "happydns.org/sources/alwaysdata"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
)

type AlwaysdataAPI struct {
	Token string `json:"token,omitempty" happydns:"label=Token API,placeholder=xxxxxxxxxx,required,description=Get your token at https://admin.alwaysdata.com/token/add/; indicate happyDNS as Application; and nothing in the second field. Copy the corresponding key."`
}

func (s *AlwaysdataAPI) newRequest(method, url string, body io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}
	req.SetBasicAuth(s.Token, "")
	return
}

func doJSON(req *http.Request, v interface{}) (err error) {
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var txt []byte
		txt, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		return fmt.Errorf("Error %d: %v", resp.StatusCode, strings.TrimSpace(string(txt)))
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return
}

func doTxt(req *http.Request) (txt []byte, err error) {
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	txt, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode >= 300 {
		return txt, fmt.Errorf("Error %d: %s", resp.StatusCode, txt)
	}

	return
}

type alwaysdataInfo struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
	Href string `json:"href"`
}

func (s *AlwaysdataAPI) ListDomains() (zones []string, err error) {
	var req *http.Request
	req, err = s.newRequest("GET", "https://api.alwaysdata.com/v1/domain/", nil)
	if err != nil {
		return
	}

	domains := []alwaysdataInfo{}

	err = doJSON(req, &domains)
	if err != nil {
		return
	}

	for _, d := range domains {
		zones = append(zones, dns.Fqdn(d.Name))
	}

	return
}

func (s *AlwaysdataAPI) Validate() (err error) {
	var req *http.Request
	req, err = s.newRequest("GET", "https://api.alwaysdata.com/v1/account/", nil)
	if err != nil {
		return
	}

	accounts := []alwaysdataInfo{}

	err = doJSON(req, &accounts)
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		return fmt.Errorf("API doesn't report any account.")
	}

	return nil
}

func (s *AlwaysdataAPI) getDomainInfos(fqdn string) (dn alwaysdataInfo, err error) {
	var req *http.Request
	req, err = s.newRequest("GET", "https://api.alwaysdata.com/v1/domain/", nil)
	if err != nil {
		return
	}

	domains := []alwaysdataInfo{}

	err = doJSON(req, &domains)
	if err != nil {
		return
	}

	fqdn = strings.TrimSuffix(fqdn, ".")
	for _, d := range domains {
		if d.Name == fqdn {
			return d, nil
		}
	}

	err = fmt.Errorf("Domain not found in your alwaysdata account.")
	return
}

func (s *AlwaysdataAPI) DomainExists(fqdn string) (err error) {
	_, err = s.getDomainInfos(fqdn)
	return
}

type alwaysdataRecord struct {
	Id            int64          `json:"id"`
	Href          string         `json:"href"`
	Domain        alwaysdataInfo `json:"domain"`
	Type          string         `json:"type"`
	Name          string         `json:"name"`
	Value         string         `json:"value"`
	Priority      *uint16        `json:"priority"`
	TTL           uint32         `json:"ttl"`
	IsUserDefined bool           `json:"is_user_defined"`
	IsActive      bool           `json:"is_active"`
}
type alwaysdataRecordOut struct {
	Id            int64   `json:"id"`
	Href          string  `json:"href"`
	Domain        int64   `json:"domain"`
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	Value         string  `json:"value"`
	Priority      *uint16 `json:"priority"`
	TTL           uint32  `json:"ttl"`
	IsUserDefined bool    `json:"is_user_defined"`
	IsActive      bool    `json:"is_active"`
}

func (r *alwaysdataRecord) toRR(origin string) (dns.RR, error) {
	if len(r.Name) == 0 {
		r.Name = origin
	} else {
		r.Name += "." + origin
	}

	if r.Type == "TXT" {
		r.Value = "\"" + r.Value + "\""
	}

	str := fmt.Sprintf("$ORIGIN .\n$TTL %d\n%s %d IN %s ", 300, r.Name, r.TTL, r.Type)

	if r.Priority != nil {
		str += fmt.Sprintf("%d %s", *r.Priority, r.Value)
	} else {
		str += r.Value
	}

	return dns.NewRR(str)
}

func newAlwaysdataRecord(rr dns.RR, domain alwaysdataInfo) (ar *alwaysdataRecordOut) {
	ar = &alwaysdataRecordOut{
		Domain: domain.Id,
		Type:   dns.Type(rr.Header().Rrtype).String(),
		Name:   strings.TrimSuffix(strings.TrimSuffix(rr.Header().Name, domain.Name+"."), "."),
		TTL:    rr.Header().Ttl,
	}

	if mx, ok := rr.(*dns.MX); ok {
		ar.Priority = &mx.Preference
		ar.Value = mx.Mx
	} else if srv, ok := rr.(*dns.SRV); ok {
		ar.Priority = &srv.Priority
		ar.Value = fmt.Sprintf("%d %d %s", srv.Weight, srv.Port, srv.Target)
	} else {
		ar.Value = strings.TrimPrefix(rr.String(), rr.Header().String())
	}

	return
}

func (s *AlwaysdataAPI) ImportZone(dn *happydns.Domain) (rrs []dns.RR, err error) {
	var domaininfo alwaysdataInfo
	domaininfo, err = s.getDomainInfos(dn.DomainName)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = s.newRequest("GET", "https://api.alwaysdata.com/v1/record/", nil)
	if err != nil {
		return
	}

	records := []*alwaysdataRecord{}

	err = doJSON(req, &records)
	if err != nil {
		return
	}

	for _, r := range records {
		// Skip non-related records
		if r.Domain.Href != domaininfo.Href {
			continue
		}

		var rr dns.RR
		rr, err = r.toRR(dn.DomainName)
		if err != nil {
			return
		}

		rrs = append(rrs, rr)
	}

	return
}

func (s *AlwaysdataAPI) AddRR(dn *happydns.Domain, rr dns.RR) (err error) {
	var domaininfo alwaysdataInfo
	domaininfo, err = s.getDomainInfos(dn.DomainName)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(newAlwaysdataRecord(rr, domaininfo))
	if err != nil {
		return
	}

	var req *http.Request
	req, err = s.newRequest("POST", "https://api.alwaysdata.com/v1/record/", &buf)
	if err != nil {
		return
	}

	err = doJSON(req, nil)
	return
}

func (s *AlwaysdataAPI) DeleteRR(dn *happydns.Domain, rr dns.RR) (err error) {
	var domaininfo alwaysdataInfo
	domaininfo, err = s.getDomainInfos(dn.DomainName)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = s.newRequest("GET", "https://api.alwaysdata.com/v1/record/", nil)
	if err != nil {
		return
	}

	records := []*alwaysdataRecord{}

	err = doJSON(req, &records)
	if err != nil {
		return
	}

	for _, r := range records {
		// Skip non-related records
		if r.Domain.Href != domaininfo.Href {
			continue
		}

		var rr_test dns.RR
		rr_test, err = r.toRR(dn.DomainName)
		if err != nil {
			return
		}

		if rr_test.String() == rr.String() {
			var req *http.Request
			req, err = s.newRequest("DELETE", fmt.Sprintf("https://api.alwaysdata.com%s", r.Href), nil)
			if err != nil {
				return
			}

			err = doJSON(req, nil)
			return
		}
	}

	return fmt.Errorf("Record not found")
}

func (s *AlwaysdataAPI) UpdateSOA(dn *happydns.Domain, newSOA *dns.SOA, refreshSerial bool) (err error) {
	return fmt.Errorf("Not implemented yet")
}

func init() {
	sources.RegisterSource(func() happydns.Source {
		return &AlwaysdataAPI{}
	}, sources.SourceInfos{
		Name:        "Alwaysdata",
		Description: "French hosting provider.",
	})
}
