// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

package abstract

import (
	"fmt"
	"strings"
	"sync"

	"github.com/miekg/dns"

	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

// autoconfigHostMu guards autoconfigHost. The host is set once at app
// startup, but tests may rewrite it.
var (
	autoconfigHostMu sync.RWMutex
	autoconfigHost   string
)

// SetAutoconfigHost configures the FQDN that autoconfig.<domain> and
// autodiscover.<domain> CNAMEs should point to. Called by the app at startup
// from the resolved Options.MailAutoconfigHost (or ExternalURL.Host).
func SetAutoconfigHost(host string) {
	autoconfigHostMu.Lock()
	defer autoconfigHostMu.Unlock()
	autoconfigHost = strings.TrimSuffix(host, ".")
}

// GetAutoconfigHost returns the configured autoconfig FQDN, with a trailing
// dot. Empty string when no host is configured.
func GetAutoconfigHost() string {
	autoconfigHostMu.RLock()
	defer autoconfigHostMu.RUnlock()
	if autoconfigHost == "" {
		return ""
	}
	return autoconfigHost + "."
}

// EmailAutoConfig publishes the three de-facto mail-client auto-configuration
// standards in one shot:
//   - RFC 6186 SRV records for the chosen incoming and outgoing protocols
//   - Mozilla Autoconfig: a CNAME at autoconfig.<domain>
//   - Microsoft Autodiscover: a CNAME at autodiscover.<domain> and
//     _autodiscover._tcp SRV
//
// The struct stores the raw DNS records verbatim. All parsing and
// reconstruction (protocol/host/port from SRV, target normalisation, etc.) is
// done by the frontend editor so that records analyzed from a zone are
// preserved exactly as-is and never re-emitted with subtle differences.
//
// The non-record fields below (DisplayName, IncomingAuth, …) are not
// published in DNS — they configure the HTTP responder that serves the
// Mozilla / Microsoft XML.
type EmailAutoConfig struct {
	DisplayName      string `json:"displayName,omitempty" happydomain:"label=Provider Display Name,placeholder=Example Mail"`
	DisplayShortName string `json:"displayShortName,omitempty" happydomain:"label=Short Name,placeholder=Example"`

	IncomingAuth   string `json:"incomingAuth,omitempty" happydomain:"label=Incoming Authentication,choices=password-cleartext;password-encrypted;OAuth2;NTLM,default=password-cleartext"`
	OutgoingAuth   string `json:"outgoingAuth,omitempty" happydomain:"label=Outgoing Authentication,choices=password-cleartext;password-encrypted;OAuth2;NTLM,default=password-cleartext"`
	UsernameFormat string `json:"usernameFormat,omitempty" happydomain:"label=Username Format,choices=%EMAILADDRESS%;%EMAILLOCALPART%,default=%EMAILADDRESS%"`
	ExchangeServer string `json:"exchangeServer,omitempty" happydomain:"label=Exchange Server (optional),placeholder=mail.example.com,description=Hostname of an on-prem Exchange server. Enables MAPI/EWS in Microsoft Autodiscover responses."`

	IncomingSRV       *dns.SRV   `json:"incomingSRV,omitempty"`
	OutgoingSRV       *dns.SRV   `json:"outgoingSRV,omitempty"`
	AutoconfigCNAME   *dns.CNAME `json:"autoconfigCNAME,omitempty"`
	AutodiscoverCNAME *dns.CNAME `json:"autodiscoverCNAME,omitempty"`
	AutodiscoverSRV   *dns.SRV   `json:"autodiscoverSRV,omitempty"`
}

// srvProtocol pulls the protocol identifier (e.g. "imaps", "submission") out
// of an SRV record's owner name. The owner name may be either a relative
// label like "_imaps._tcp" or a full FQDN like "_imaps._tcp.example.com.".
func srvProtocol(srv *dns.SRV) string {
	if srv == nil {
		return ""
	}
	name := strings.TrimPrefix(srv.Hdr.Name, "_")
	if i := strings.Index(name, "."); i >= 0 {
		name = name[:i]
	}
	return name
}

// srvHost returns the SRV target with any trailing dot stripped.
func srvHost(srv *dns.SRV) string {
	if srv == nil {
		return ""
	}
	return strings.TrimSuffix(srv.Target, ".")
}

// IncomingType returns the incoming protocol identifier derived from the
// stored SRV record, or "" when no incoming SRV is configured.
func (s *EmailAutoConfig) IncomingType() string { return srvProtocol(s.IncomingSRV) }

// IncomingHost returns the incoming hostname (no trailing dot).
func (s *EmailAutoConfig) IncomingHost() string { return srvHost(s.IncomingSRV) }

// IncomingPort returns the incoming port, or 0 when no incoming SRV is set.
func (s *EmailAutoConfig) IncomingPort() uint16 {
	if s.IncomingSRV == nil {
		return 0
	}
	return s.IncomingSRV.Port
}

// OutgoingType returns the outgoing protocol identifier derived from the
// stored SRV record.
func (s *EmailAutoConfig) OutgoingType() string { return srvProtocol(s.OutgoingSRV) }

// OutgoingHost returns the outgoing hostname (no trailing dot).
func (s *EmailAutoConfig) OutgoingHost() string { return srvHost(s.OutgoingSRV) }

// OutgoingPort returns the outgoing port, or 0 when no outgoing SRV is set.
func (s *EmailAutoConfig) OutgoingPort() uint16 {
	if s.OutgoingSRV == nil {
		return 0
	}
	return s.OutgoingSRV.Port
}

func (s *EmailAutoConfig) GetNbResources() int {
	n := 0
	if srvConfigured(s.IncomingSRV) {
		n++
	}
	if srvConfigured(s.OutgoingSRV) {
		n++
	}
	if cnameConfigured(s.AutoconfigCNAME) {
		n++
	}
	if cnameConfigured(s.AutodiscoverCNAME) {
		n++
	}
	if srvConfigured(s.AutodiscoverSRV) {
		n++
	}
	return n
}

func (s *EmailAutoConfig) GenComment() string {
	var b strings.Builder

	if srvConfigured(s.IncomingSRV) {
		fmt.Fprintf(&b, "%s %s:%d", strings.ToUpper(s.IncomingType()), s.IncomingHost(), s.IncomingPort())
	}
	if srvConfigured(s.OutgoingSRV) {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		fmt.Fprintf(&b, "%s %s:%d", s.OutgoingType(), s.OutgoingHost(), s.OutgoingPort())
	}
	if cnameConfigured(s.AutoconfigCNAME) || cnameConfigured(s.AutodiscoverCNAME) {
		if b.Len() > 0 {
			b.WriteString(" + ")
		}
		b.WriteString("autoconfig/autodiscover")
	}

	return b.String()
}

// srvConfigured reports whether the SRV pointer was actually filled in by the
// frontend, as opposed to being a zero stub left over from the service-spec
// auto-initializer (which pre-allocates pointer-to-DNS fields with an empty
// Hdr.Name and Target).
func srvConfigured(srv *dns.SRV) bool {
	return srv != nil && srv.Hdr.Name != ""
}

func cnameConfigured(c *dns.CNAME) bool {
	return c != nil && c.Hdr.Name != ""
}

// GetRecords returns the stored records verbatim. The frontend editor is
// responsible for filling every field — the backend never synthesises or
// rewrites records, so what was analyzed (or what the user typed) is exactly
// what gets published.
func (s *EmailAutoConfig) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	var rrs []happydns.Record

	if srvConfigured(s.IncomingSRV) {
		rrs = append(rrs, s.IncomingSRV)
	}
	if srvConfigured(s.OutgoingSRV) {
		rrs = append(rrs, s.OutgoingSRV)
	}
	if cnameConfigured(s.AutoconfigCNAME) {
		rrs = append(rrs, s.AutoconfigCNAME)
	}
	if cnameConfigured(s.AutodiscoverCNAME) {
		rrs = append(rrs, s.AutodiscoverCNAME)
	}
	if srvConfigured(s.AutodiscoverSRV) {
		rrs = append(rrs, s.AutodiscoverSRV)
	}

	return rrs, nil
}

// emailautoconfig_analyze reconstructs an EmailAutoConfig from a zone import.
// It only claims records when both the autoconfig. and autodiscover. CNAMEs
// are present and point to the same target — that's the unambiguous signal
// that this domain was previously published with the high-level service.
// Otherwise it leaves the SRV records for rfc6186_analyze to pick up.
func emailautoconfig_analyze(a *svc.Analyzer) error {
	candidates := map[string]*dns.CNAME{}

	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeCNAME, Prefix: "autoconfig."}) {
		cname, ok := record.(*dns.CNAME)
		if !ok {
			continue
		}
		domain := strings.TrimPrefix(cname.Header().Name, "autoconfig.")
		candidates[domain] = cname
	}

	for domain, autoconfigCNAME := range candidates {
		// Find a matching autodiscover. CNAME with the same target.
		var autodiscoverCNAME *dns.CNAME
		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeCNAME, Prefix: "autodiscover." + domain}) {
			if cname, ok := record.(*dns.CNAME); ok && cname.Header().Name == "autodiscover."+domain && cname.Target == autoconfigCNAME.Target {
				autodiscoverCNAME = cname
				break
			}
		}
		if autodiscoverCNAME == nil {
			continue
		}

		ec := &EmailAutoConfig{
			AutoconfigCNAME:   autoconfigCNAME,
			AutodiscoverCNAME: autodiscoverCNAME,
		}
		consumed := []happydns.Record{autoconfigCNAME, autodiscoverCNAME}

		// Find at most one incoming + one outgoing SRV under this domain.
		for _, p := range []string{"_imaps._tcp.", "_imap._tcp.", "_pop3s._tcp.", "_pop3._tcp."} {
			if ec.IncomingSRV != nil {
				break
			}
			for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeSRV, Prefix: p + domain}) {
				if srv, ok := record.(*dns.SRV); ok && srv.Header().Name == p+domain {
					ec.IncomingSRV = srv
					consumed = append(consumed, srv)
					break
				}
			}
		}
		for _, p := range []string{"_submissions._tcp.", "_submission._tcp."} {
			if ec.OutgoingSRV != nil {
				break
			}
			for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeSRV, Prefix: p + domain}) {
				if srv, ok := record.(*dns.SRV); ok && srv.Header().Name == p+domain {
					ec.OutgoingSRV = srv
					consumed = append(consumed, srv)
					break
				}
			}
		}

		// We need at least one incoming and one outgoing protocol — otherwise
		// the analysis isn't strong enough. Leave the records.
		if ec.IncomingSRV == nil || ec.OutgoingSRV == nil {
			continue
		}

		// Optionally consume the _autodiscover._tcp SRV.
		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeSRV, Prefix: "_autodiscover._tcp." + domain}) {
			if srv, ok := record.(*dns.SRV); ok && srv.Header().Name == "_autodiscover._tcp."+domain {
				ec.AutodiscoverSRV = srv
				consumed = append(consumed, srv)
				break
			}
		}

		for _, rr := range consumed {
			if err := a.UseRR(rr, domain, ec); err != nil {
				return err
			}
		}
	}

	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody { return &EmailAutoConfig{} },
		emailautoconfig_analyze,
		happydns.ServiceInfos{
			Name:        "Email Auto-configuration",
			Description: "Publish IMAP/POP/SMTP settings for mail clients via RFC 6186, Mozilla Autoconfig, and Microsoft Autodiscover.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories:  []string{"email"},
			RecordTypes: []uint16{dns.TypeSRV, dns.TypeCNAME},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{dns.TypeSRV},
			},
		},
		1,
	)
}
