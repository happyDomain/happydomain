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

package checks

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/abstract"
)

func init() {
	RegisterChecker("caa_cert_issuer", &CAACertIssuerCheck{})
}

// CAACertIssuerReport contains the results of CAA vs certificate issuer checks.
type CAACertIssuerReport struct {
	DisallowIssue  bool                    `json:"disallow_issue,omitempty"`
	AllowedIssuers []string                `json:"allowed_issuers"`
	Servers        []CAACertServerResult   `json:"servers"`
}

// CAACertServerResult holds the check results for a single server.
type CAACertServerResult struct {
	Subdomain string              `json:"subdomain"`
	Address   string              `json:"address"`
	Error     string              `json:"error,omitempty"`
	CertInfo  *CAACertInfo        `json:"cert_info,omitempty"`
}

// CAACertInfo holds certificate details and match status.
type CAACertInfo struct {
	Subject   string `json:"subject"`
	IssuerCN  string `json:"issuer_cn"`
	IssuerOrg string `json:"issuer_org"`
	Matched   bool   `json:"matched"`
	MatchedCA string `json:"matched_ca,omitempty"`
}

type CAACertIssuerCheck struct{}

func (c *CAACertIssuerCheck) ID() string {
	return "caa_cert_issuer"
}

func (c *CAACertIssuerCheck) Name() string {
	return "CAA Certificate Issuer"
}

func (c *CAACertIssuerCheck) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{
		ApplyToService:  true,
		LimitToServices: []string{"svcs.CAAPolicy"},
	}
}

func (c *CAACertIssuerCheck) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:       "service",
				Label:    "Service",
				AutoFill: happydns.AutoFillService,
			},
			{
				Id:       "zone",
				Label:    "Zone",
				AutoFill: happydns.AutoFillZone,
			},
			{
				Id:       "subdomain",
				Label:    "Subdomain",
				AutoFill: happydns.AutoFillSubdomain,
			},
			{
				Id:       "domainName",
				Label:    "Domain name",
				AutoFill: happydns.AutoFillDomainName,
			},
		},
	}
}

// normalizeForMatch strips non-alphanumeric characters and lowercases a string
// for fuzzy CA name matching.
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// caaIssuerDomainBase extracts the base name from a CAA issuer domain
// (e.g., "letsencrypt.org" → "letsencrypt", "pki.goog" → "pki").
func caaIssuerDomainBase(issuerDomain string) string {
	issuerDomain = strings.ToLower(strings.TrimSpace(issuerDomain))
	parts := strings.Split(issuerDomain, ".")
	if len(parts) >= 2 {
		return parts[0]
	}
	return issuerDomain
}

// matchCAAIssuer checks if a certificate's issuer matches any allowed CAA issuer domain.
func matchCAAIssuer(issuerCN, issuerOrg string, allowedIssuers []string) (bool, string) {
	normCN := normalizeForMatch(issuerCN)
	normOrg := normalizeForMatch(issuerOrg)

	for _, issuer := range allowedIssuers {
		base := normalizeForMatch(caaIssuerDomainBase(issuer))
		if base == "" {
			continue
		}

		// Check if the base name appears in the CN or Org
		if strings.Contains(normCN, base) || strings.Contains(normOrg, base) {
			return true, issuer
		}

		// Also check the full domain (without TLD) normalized
		fullNorm := normalizeForMatch(issuer)
		if strings.Contains(normCN, fullNorm) || strings.Contains(normOrg, fullNorm) {
			return true, issuer
		}
	}

	return false, ""
}

func (c *CAACertIssuerCheck) RunCheck(ctx context.Context, options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	service, ok := options["service"].(*happydns.ServiceMessage)
	if !ok {
		return nil, fmt.Errorf("service not defined")
	}
	if service.Type != "svcs.CAAPolicy" {
		return nil, fmt.Errorf("service is %s, expected svcs.CAAPolicy", service.Type)
	}

	var caaPolicy svcs.CAAPolicy
	if err := json.Unmarshal(service.Service, &caaPolicy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CAA policy: %w", err)
	}

	// Parse CAA records to get allowed issuers.
	caaFields := svcs.CAAFields{}
	for _, rec := range caaPolicy.Records {
		caaFields.Analyze(rec.Flag, rec.Tag, rec.Value)
	}

	report := CAACertIssuerReport{}

	// If issuance is disallowed, just report that.
	if caaFields.DisallowIssue {
		report.DisallowIssue = true
		return &happydns.CheckResult{
			Status:     happydns.CheckResultStatusOK,
			StatusLine: "Certificate issuance disallowed by CAA policy",
			Report:     report,
		}, nil
	}

	// Collect allowed issuer domains from Issue + IssueWild.
	var allowedIssuers []string
	seen := map[string]bool{}
	for _, iss := range caaFields.Issue {
		if iss.IssuerDomainName != "" && !seen[iss.IssuerDomainName] {
			allowedIssuers = append(allowedIssuers, iss.IssuerDomainName)
			seen[iss.IssuerDomainName] = true
		}
	}
	for _, iss := range caaFields.IssueWild {
		if iss.IssuerDomainName != "" && !seen[iss.IssuerDomainName] {
			allowedIssuers = append(allowedIssuers, iss.IssuerDomainName)
			seen[iss.IssuerDomainName] = true
		}
	}
	report.AllowedIssuers = allowedIssuers

	if len(allowedIssuers) == 0 {
		return &happydns.CheckResult{
			Status:     happydns.CheckResultStatusInfo,
			StatusLine: "No CAA issue/issuewild records found",
			Report:     report,
		}, nil
	}

	// Get the zone to find Server services.
	zone, ok := options["zone"].(*happydns.ZoneMessage)
	if !ok {
		return nil, fmt.Errorf("zone not defined")
	}

	domainName := ""
	if dn, ok := options["domainName"].(string); ok {
		domainName = dn
	}

	// Iterate over all services in the zone to find Server services.
	for subdomain, services := range zone.Services {
		for _, svc := range services {
			if svc.Type != "abstract.Server" {
				continue
			}

			var server abstract.Server
			if err := json.Unmarshal(svc.Service, &server); err != nil {
				continue
			}

			// Build the FQDN for SNI.
			var fqdn string
			if string(subdomain) == "" || string(subdomain) == "@" {
				fqdn = strings.TrimSuffix(domainName, ".")
			} else {
				fqdn = string(subdomain) + "." + strings.TrimSuffix(domainName, ".")
			}

			// Collect IPs from A and AAAA records.
			var ips []net.IP
			if server.A != nil && len(server.A.A) != 0 {
				ips = append(ips, server.A.A)
			}
			if server.AAAA != nil && len(server.AAAA.AAAA) != 0 {
				ips = append(ips, server.AAAA.AAAA)
			}

			for _, ip := range ips {
				result := CAACertServerResult{
					Subdomain: string(subdomain),
					Address:   ip.String(),
				}

				certInfo, err := fetchCertInfo(ctx, fqdn, ip.String())
				if err != nil {
					result.Error = err.Error()
				} else {
					matched, matchedCA := matchCAAIssuer(certInfo.IssuerCN, certInfo.IssuerOrg, allowedIssuers)
					certInfo.Matched = matched
					certInfo.MatchedCA = matchedCA
					result.CertInfo = certInfo
				}

				report.Servers = append(report.Servers, result)
			}
		}
	}

	if len(report.Servers) == 0 {
		return &happydns.CheckResult{
			Status:     happydns.CheckResultStatusInfo,
			StatusLine: "No servers found in zone to check certificates",
			Report:     report,
		}, nil
	}

	// Determine overall status.
	overallStatus := happydns.CheckResultStatusOK
	matchCount := 0
	errorCount := 0
	totalCount := len(report.Servers)

	for _, srv := range report.Servers {
		if srv.Error != "" {
			errorCount++
		} else if srv.CertInfo != nil && srv.CertInfo.Matched {
			matchCount++
		}
	}

	var summaryParts []string
	for _, srv := range report.Servers {
		label := srv.Address
		if srv.Subdomain != "" {
			label = srv.Subdomain + "/" + srv.Address
		}
		if srv.Error != "" {
			summaryParts = append(summaryParts, fmt.Sprintf("%s: error", label))
		} else if srv.CertInfo != nil && srv.CertInfo.Matched {
			summaryParts = append(summaryParts, fmt.Sprintf("%s: OK (%s)", label, srv.CertInfo.MatchedCA))
		} else {
			summaryParts = append(summaryParts, fmt.Sprintf("%s: mismatch", label))
		}
	}

	checkedCount := totalCount - errorCount
	if checkedCount == 0 {
		overallStatus = happydns.CheckResultStatusCritical
	} else if matchCount < checkedCount {
		overallStatus = happydns.CheckResultStatusWarn
	}

	return &happydns.CheckResult{
		Status:     overallStatus,
		StatusLine: strings.Join(summaryParts, " | "),
		Report:     report,
	}, nil
}

// fetchCertInfo connects to the given IP on port 443 with the specified SNI
// and returns the certificate information.
func fetchCertInfo(ctx context.Context, sni, ip string) (*CAACertInfo, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}

	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(ip, "443"), &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, fmt.Errorf("TLS connection failed: %w", err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates presented")
	}

	leaf := certs[0]
	info := &CAACertInfo{
		Subject:  leaf.Subject.CommonName,
		IssuerCN: leaf.Issuer.CommonName,
	}
	if len(leaf.Issuer.Organization) > 0 {
		info.IssuerOrg = strings.Join(leaf.Issuer.Organization, ", ")
	}

	return info, nil
}

var caaCertIssuerHTMLTemplate = template.Must(template.New("caa_cert_issuer").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>CAA Certificate Issuer Check</title>
<style>
body { font-family: sans-serif; margin: 1em; }
table { border-collapse: collapse; width: 100%; margin-bottom: 1.5em; }
th, td { border: 1px solid #ccc; padding: 0.4em 0.8em; text-align: left; }
th { background: #f0f0f0; }
.ok { color: #2a7a2a; font-weight: bold; }
.fail { color: #c0392b; font-weight: bold; }
.error { color: #856404; font-style: italic; }
.info { margin-bottom: 1em; }
</style>
</head>
<body>
<h1>CAA Certificate Issuer Report</h1>
{{if .DisallowIssue}}
<p class="info">Certificate issuance is <strong>disallowed</strong> by CAA policy.</p>
{{else}}
<p class="info">Allowed issuers: <strong>{{range $i, $v := .AllowedIssuers}}{{if $i}}, {{end}}{{$v}}{{end}}</strong></p>
{{if .Servers}}
<table>
<thead><tr><th>Subdomain</th><th>Address</th><th>Certificate Subject</th><th>Issuer CN</th><th>Issuer Org</th><th>CAA Match</th></tr></thead>
<tbody>
{{range .Servers}}
<tr>
  <td>{{if .Subdomain}}{{.Subdomain}}{{else}}@{{end}}</td>
  <td>{{.Address}}</td>
  {{if .Error}}
  <td colspan="4"><span class="error">{{.Error}}</span></td>
  {{else if .CertInfo}}
  <td>{{.CertInfo.Subject}}</td>
  <td>{{.CertInfo.IssuerCN}}</td>
  <td>{{.CertInfo.IssuerOrg}}</td>
  <td>{{if .CertInfo.Matched}}<span class="ok">&#10003; {{.CertInfo.MatchedCA}}</span>{{else}}<span class="fail">&#10007; No match</span>{{end}}</td>
  {{end}}
</tr>
{{end}}
</tbody>
</table>
{{else}}
<p class="info">No servers found in zone.</p>
{{end}}
{{end}}
</body>
</html>
`))

// GetHTMLReport implements happydns.CheckerHTMLReporter.
func (c *CAACertIssuerCheck) GetHTMLReport(raw json.RawMessage) (string, error) {
	var report CAACertIssuerReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return "", fmt.Errorf("failed to parse report: %w", err)
	}

	var buf bytes.Buffer
	if err := caaCertIssuerHTMLTemplate.Execute(&buf, report); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}
