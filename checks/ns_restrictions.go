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
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

func init() {
	RegisterChecker("ns_restrictions", &NSRestrictionsCheck{})
}

// NSRestrictionsReport contains the results of NS security restriction checks.
type NSRestrictionsReport struct {
	Servers []NSServerResult `json:"servers"`
}

// NSServerResult holds the check results for a single nameserver IP.
type NSServerResult struct {
	Name    string        `json:"name"`
	Address string        `json:"address"`
	Checks  []NSCheckItem `json:"checks"`
}

// NSCheckItem represents one security check for an NS server.
type NSCheckItem struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type NSRestrictionsCheck struct{}

func (c *NSRestrictionsCheck) ID() string {
	return "ns_restrictions"
}

func (c *NSRestrictionsCheck) Name() string {
	return "NS Security Restrictions"
}

func (c *NSRestrictionsCheck) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{
		ApplyToService:  true,
		LimitToServices: []string{"abstract.Origin", "abstract.NSOnlyOrigin"},
	}
}

func (c *NSRestrictionsCheck) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:       "service",
				Label:    "Service",
				AutoFill: happydns.AutoFillService,
			},
			{
				Id:       "domainName",
				Label:    "Domain name",
				AutoFill: happydns.AutoFillDomainName,
			},
		},
	}
}

// nsFromServiceOption extracts the list of NS records from an Origin or NSOnlyOrigin service.
func nsFromServiceOption(svc *happydns.ServiceMessage) []*dns.NS {
	if svc.Type == "abstract.Origin" {
		var origin abstract.Origin
		if err := json.Unmarshal(svc.Service, &origin); err != nil {
			return nil
		}
		return origin.NameServers
	}

	var origin abstract.NSOnlyOrigin
	if err := json.Unmarshal(svc.Service, &origin); err != nil {
		return nil
	}
	return origin.NameServers
}

// checkAXFR returns (ok bool, detail string).
// ok=false means the server accepted the zone transfer (CRITICAL).
func checkAXFR(ctx context.Context, domain, addr string) (bool, string) {
	msg := new(dns.Msg)
	msg.SetAxfr(dns.Fqdn(domain))

	t := &dns.Transfer{}
	t.DialTimeout = 5 * time.Second
	t.ReadTimeout = 10 * time.Second

	ch, err := t.In(msg, net.JoinHostPort(addr, "53"))
	if err != nil {
		// Connection refused or similar — transfer was refused, good.
		return true, fmt.Sprintf("transfer refused: %s", err)
	}

	for env := range ch {
		if env.Error != nil {
			return true, fmt.Sprintf("transfer error: %s", env.Error)
		}
		for _, rr := range env.RR {
			if rr.Header().Rrtype == dns.TypeSOA {
				// Zone transfer succeeded — CRITICAL.
				return false, "AXFR zone transfer accepted"
			}
		}
	}

	return true, "AXFR refused"
}

// checkIXFR returns (ok bool, detail string).
// ok=false means the server answered with records (WARN).
func checkIXFR(ctx context.Context, domain, addr string) (bool, string) {
	msg := new(dns.Msg)
	msg.SetIxfr(dns.Fqdn(domain), 0, "", "")

	cl := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	resp, _, err := cl.ExchangeContext(ctx, msg, net.JoinHostPort(addr, "53"))
	if err != nil {
		return true, fmt.Sprintf("query failed: %s", err)
	}

	if resp.Rcode != dns.RcodeSuccess {
		return true, fmt.Sprintf("IXFR refused (rcode=%s)", dns.RcodeToString[resp.Rcode])
	}
	if len(resp.Answer) > 0 {
		return false, fmt.Sprintf("IXFR accepted with %d answer(s)", len(resp.Answer))
	}

	return true, "IXFR refused or empty"
}

// checkNoRecursion returns (ok bool, detail string).
// ok=false means the server offers recursion (WARN).
func checkNoRecursion(ctx context.Context, domain, addr string) (bool, string) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeSOA)
	msg.RecursionDesired = true

	cl := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	resp, _, err := cl.ExchangeContext(ctx, msg, net.JoinHostPort(addr, "53"))
	if err != nil {
		return true, fmt.Sprintf("query failed: %s", err)
	}

	if resp.RecursionAvailable {
		return false, "recursion available (RA bit set)"
	}
	return true, "recursion not available"
}

// checkANYHandled returns (ok bool, detail string).
// ok=false means the server returned a full record set for ANY (WARN).
// Per RFC 8482, servers should return HINFO or minimal response.
func checkANYHandled(ctx context.Context, domain, addr string) (bool, string) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeANY)

	cl := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	resp, _, err := cl.ExchangeContext(ctx, msg, net.JoinHostPort(addr, "53"))
	if err != nil {
		return true, fmt.Sprintf("query failed: %s", err)
	}

	if resp.Rcode != dns.RcodeSuccess {
		return true, fmt.Sprintf("ANY refused (rcode=%s)", dns.RcodeToString[resp.Rcode])
	}

	// If there's only a HINFO record, it's RFC 8482 compliant.
	if len(resp.Answer) == 1 {
		if _, ok := resp.Answer[0].(*dns.HINFO); ok {
			return true, "RFC 8482 compliant HINFO response"
		}
	}

	// Empty answer or TC (truncated) with no answers — also acceptable.
	if len(resp.Answer) == 0 {
		return true, "ANY returned empty answer"
	}

	return false, fmt.Sprintf("ANY returned %d records (not RFC 8482 compliant)", len(resp.Answer))
}

// checkIsAuthoritative returns (ok bool, detail string).
// ok=false means the server is not authoritative for the zone (INFO).
func checkIsAuthoritative(ctx context.Context, domain, addr string) (bool, string) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeSOA)

	cl := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	resp, _, err := cl.ExchangeContext(ctx, msg, net.JoinHostPort(addr, "53"))
	if err != nil {
		return false, fmt.Sprintf("query failed: %s", err)
	}

	if resp.Authoritative {
		return true, "server is authoritative (AA bit set)"
	}
	return false, "server is not authoritative (AA bit not set)"
}

// checkServerAddr runs all NS security checks against a single IP address.
// Returns the result and the worst status encountered.
func checkServerAddr(ctx context.Context, domain, nsHost, addr string) (NSServerResult, happydns.CheckResultStatus) {
	result := NSServerResult{Name: nsHost, Address: addr}
	status := happydns.CheckResultStatusOK

	type checkDef struct {
		name      string
		fn        func(context.Context, string, string) (bool, string)
		failLevel happydns.CheckResultStatus
	}
	checks := []checkDef{
		{"AXFR refused", checkAXFR, happydns.CheckResultStatusCritical},
		{"IXFR refused", checkIXFR, happydns.CheckResultStatusWarn},
		{"No recursion", checkNoRecursion, happydns.CheckResultStatusWarn},
		{"ANY handled (RFC 8482)", checkANYHandled, happydns.CheckResultStatusWarn},
		{"Is authoritative", checkIsAuthoritative, happydns.CheckResultStatusInfo},
	}

	for _, ch := range checks {
		ok, detail := ch.fn(ctx, domain, addr)
		result.Checks = append(result.Checks, NSCheckItem{Name: ch.name, OK: ok, Detail: detail})
		if !ok && status > ch.failLevel {
			status = ch.failLevel
		}
	}

	return result, status
}

// checkNameServer resolves nsHost and runs checks on each address.
// Returns results and summary parts for each address.
func checkNameServer(ctx context.Context, domain, nsHost string) ([]NSServerResult, []string, happydns.CheckResultStatus) {
	worstStatus := happydns.CheckResultStatusOK

	addrs, err := net.LookupHost(nsHost)
	if err != nil {
		return []NSServerResult{{
			Name:    nsHost,
			Address: "",
			Checks:  []NSCheckItem{{Name: "DNS resolution", OK: false, Detail: fmt.Sprintf("lookup failed: %s", err)}},
		}}, []string{fmt.Sprintf("%s: resolution failed", nsHost)}, happydns.CheckResultStatusWarn
	}

	var results []NSServerResult
	var summaryParts []string

	for _, addr := range addrs {
		// Skip IPv6 addresses when there is no IPv6 connectivity.
		if ip := net.ParseIP(addr); ip != nil && ip.To4() == nil {
			conn, err := net.DialTimeout("udp", net.JoinHostPort(addr, "53"), 3*time.Second)
			if errors.Is(err, syscall.ENETUNREACH) {
				results = append(results, NSServerResult{
					Name:    nsHost,
					Address: addr,
					Checks:  []NSCheckItem{{Name: "IPv6 connectivity", Detail: "unable to test due to the lack of IPv6 connectivity"}},
				})
				summaryParts = append(summaryParts, fmt.Sprintf("%s (%s): skipped (no IPv6)", nsHost, addr))
				continue
			}
			if conn != nil {
				conn.Close()
			}
		}

		serverResult, serverStatus := checkServerAddr(ctx, domain, nsHost, addr)
		results = append(results, serverResult)

		if serverStatus < worstStatus {
			worstStatus = serverStatus
		}

		switch serverStatus {
		case happydns.CheckResultStatusCritical:
			summaryParts = append(summaryParts, fmt.Sprintf("%s (%s): CRITICAL", nsHost, addr))
		case happydns.CheckResultStatusWarn:
			summaryParts = append(summaryParts, fmt.Sprintf("%s (%s): WARN", nsHost, addr))
		default:
			summaryParts = append(summaryParts, fmt.Sprintf("%s (%s): OK", nsHost, addr))
		}
	}

	return results, summaryParts, worstStatus
}

func (c *NSRestrictionsCheck) RunCheck(ctx context.Context, options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	service, ok := options["service"].(*happydns.ServiceMessage)
	if !ok {
		return nil, fmt.Errorf("service not defined")
	}
	if service.Type != "abstract.Origin" && service.Type != "abstract.NSOnlyOrigin" {
		return nil, fmt.Errorf("service is %s, expected abstract.Origin or abstract.NSOnlyOrigin", service.Type)
	}

	domainName := ""
	if dn, ok := options["domainName"].(string); ok {
		domainName = dn
	}
	if domainName == "" {
		domainName = service.Domain
	}

	nameServers := nsFromServiceOption(service)
	if len(nameServers) == 0 {
		return nil, fmt.Errorf("no nameservers found in service")
	}

	report := NSRestrictionsReport{}
	overallStatus := happydns.CheckResultStatusOK
	var summaryParts []string

	for _, ns := range nameServers {
		nsHost := strings.TrimSuffix(ns.Ns, ".")
		results, parts, status := checkNameServer(ctx, domainName, nsHost)
		report.Servers = append(report.Servers, results...)
		summaryParts = append(summaryParts, parts...)
		if status < overallStatus {
			overallStatus = status
		}
	}

	return &happydns.CheckResult{
		Status:     overallStatus,
		StatusLine: strings.Join(summaryParts, " | "),
		Report:     report,
	}, nil
}

var nsRestrictionsHTMLTemplate = template.Must(template.New("ns_restrictions").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>NS Security Restrictions</title>
<style>
body { font-family: sans-serif; margin: 1em; }
table { border-collapse: collapse; width: 100%; margin-bottom: 1.5em; }
th, td { border: 1px solid #ccc; padding: 0.4em 0.8em; text-align: left; }
th { background: #f0f0f0; }
.ok { color: #2a7a2a; font-weight: bold; }
.fail { color: #c0392b; font-weight: bold; }
h2 { margin-top: 1.5em; }
</style>
</head>
<body>
<h1>NS Security Restrictions Report</h1>
{{range .Servers}}
<h2>{{.Name}} ({{.Address}})</h2>
<table>
<thead><tr><th>Check</th><th>Result</th><th>Detail</th></tr></thead>
<tbody>
{{range .Checks}}
<tr>
  <td>{{.Name}}</td>
  <td>{{if .OK}}<span class="ok">&#10003; OK</span>{{else}}<span class="fail">&#10007; FAIL</span>{{end}}</td>
  <td>{{.Detail}}</td>
</tr>
{{end}}
</tbody>
</table>
{{end}}
</body>
</html>
`))

// GetHTMLReport implements happydns.CheckerHTMLReporter.
func (c *NSRestrictionsCheck) GetHTMLReport(raw json.RawMessage) (string, error) {
	var report NSRestrictionsReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return "", fmt.Errorf("failed to parse report: %w", err)
	}

	var buf bytes.Buffer
	if err := nsRestrictionsHTMLTemplate.Execute(&buf, report); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}
