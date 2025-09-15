package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"git.happydns.org/happyDomain/model"
)

type MatrixTester struct {
	TesterURI string
}

func (p *MatrixTester) ID() string {
	return "matrixim"
}

func (p *MatrixTester) Name() string {
	return "Matrix Federation Tester"
}

func (p *MatrixTester) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{
		ApplyToService:  true,
		LimitToServices: []string{"abstract.MatrixIM"},
	}
}

func (p *MatrixTester) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:          "serviceDomain",
				Type:        "string",
				Label:       "Matrix domain",
				Placeholder: "matrix.org",
				Default:     "matrix.org",
				AutoFill:    happydns.AutoFillDomainName,
				Required:    true,
			},
		},
		AdminOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:          "federationTesterServer",
				Type:        "string",
				Label:       "Federation Tester Server",
				Placeholder: "https://federationtester.matrix.org/",
				Default:     "https://federationtester.matrix.org/",
				Required:    true,
			},
		},
	}
}

type FederationTesterResponse struct {
	WellKnownResult struct {
		Server         string `json:"m.server"`
		Result         string `json:"result"`
		CacheExpiresAt int64  `json:"CacheExpiresAt"`
	}
	DNSResult struct {
		SRVSkipped bool   `json:"SRVSkipped"`
		SRVCName   string `json:"SRVCName"`
		SRVRecords []struct {
			Target   string `json:"Target"`
			Port     uint16 `json:"Port"`
			Priority uint16 `json:"Priority"`
			Weight   uint16 `json:"Weight"`
		} `json:"SRVRecords"`
		SRVError *struct {
			Message string `json:"Message"`
		} `json:"SRVError"`
		Hosts map[string]struct {
			CName string   `json:"CName"`
			Addrs []string `json:"Addrs"`
		} `json:"Hosts"`
		Addrs []string `json:"Addrs"`
	}
	ConnectionReports map[string]struct {
		Certificates []struct {
			SubjectCommonName string   `json:"SubjectCommonName"`
			IssuerCommonName  string   `json:"IssuerCommonName"`
			SHA256Fingerprint string   `json:"SHA256Fingerprint"`
			DNSNames          []string `json:"DNSNames"`
		}
		Cipher struct {
			Version     string `json:"Version"`
			CipherSuite string `json:"CipherSuite"`
		}
		Checks struct {
			AllChecksOK        bool `json:"AllChecksOK"`
			MatchingServerName bool `json:"MatchingServerName"`
			FutureValidUntilTS bool `json:"FutureValidUntilTS"`
			HasEd25519Key      bool `json:"HasEd25519Key"`
			AllEd25519ChecksOK bool `json:"AllEd25519ChecksOK"`
			ValidCertificates  bool `json:"ValidCertificates"`
		}
		Errors []string
	}
	ConnectionErrors map[string]struct {
		Message string
	}
	Version struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Error   string `json:"error,omitempty"`
	}
	FederationOK bool `json:"FederationOK"`
}

func (p *MatrixTester) RunCheck(ctx context.Context, options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	var domain string

	if dn, ok := options["serviceDomain"]; ok {
		domain, _ = dn.(string)
	}

	if domain == "" {
		return nil, fmt.Errorf("domain not defined")
	}

	domain = strings.TrimSuffix(domain, ".")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(p.TesterURI, domain), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to build the request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to perform the test: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Sorry, the federation tester is broken. Check on https://federationtester.matrix.org/#%s", strings.TrimSuffix(domain, "."))
	}

	var status happydns.CheckResultStatus
	var statusLine string
	var federationTest FederationTesterResponse

	err = json.NewDecoder(resp.Body).Decode(&federationTest)
	if err != nil {
		log.Printf("Error in check_matrix_federation, when decoding json: %s", err.Error())
		return nil, fmt.Errorf("sorry, the federation tester is broken. Check on https://federationtester.matrix.org/#%s", strings.TrimSuffix(domain, "."))
	}

	if federationTest.FederationOK {
		status = happydns.CheckResultStatusOK
		statusLine = "Running " + federationTest.Version.Name + " " + federationTest.Version.Version
	} else {
		status = happydns.CheckResultStatusCritical

		if federationTest.DNSResult.SRVError != nil && federationTest.WellKnownResult.Result != "" {
			statusLine = fmt.Sprintf("%s OR %s", federationTest.DNSResult.SRVError.Message, federationTest.WellKnownResult.Result)
		} else if len(federationTest.ConnectionErrors) > 0 {
			var msg strings.Builder
			for srv, cerr := range federationTest.ConnectionErrors {
				if msg.Len() > 0 {
					msg.WriteString("; ")
				}
				msg.WriteString(srv)
				msg.WriteString(": ")
				msg.WriteString(cerr.Message)
			}
			statusLine = fmt.Sprintf("Connection errors: %s", msg.String())
		} else if federationTest.WellKnownResult.Server != strings.TrimSuffix(domain, ".") {
			statusLine = fmt.Sprintf("Bad homeserver_name: got %s, expected %s.", federationTest.WellKnownResult.Server, strings.TrimSuffix(domain, "."))
		} else {
			statusLine = fmt.Sprintf("An unimplemented error occurs. Please report this to happydomain team. But know that federation seems to be broken. Check https://federationtester.matrix.org/#%s", strings.TrimSuffix(domain, "."))
		}
	}

	return &happydns.CheckResult{
		Status:     status,
		StatusLine: statusLine,
		Report:     federationTest,
	}, nil
}

// ── HTML report ───────────────────────────────────────────────────────────────

type matrixCertData struct {
	SubjectCommonName string
	IssuerCommonName  string
	SHA256Fingerprint string
	DNSNames          []string
}

type matrixConnectionData struct {
	Address      string
	TLSVersion   string
	CipherSuite  string
	Certs        []matrixCertData
	AllChecksOK  bool
	CheckDetails []matrixCheckItem
	Errors       []string
	Open         bool // details element open when checks failed
}

type matrixCheckItem struct {
	Label string
	OK    bool
}

type matrixConnErrData struct {
	Address string
	Message string
}

type matrixSRVRecord struct {
	Target   string
	Port     uint16
	Priority uint16
	Weight   uint16
}

type matrixHostData struct {
	Name  string
	CName string
	Addrs []string
}

type matrixTemplateData struct {
	FederationOK     bool
	Version          string
	VersionError     string
	WellKnownServer  string
	WellKnownResult  string
	SRVSkipped       bool
	SRVCName         string
	SRVRecords       []matrixSRVRecord
	SRVError         string
	Hosts            []matrixHostData
	Addrs            []string
	Connections      []matrixConnectionData
	ConnectionErrors []matrixConnErrData
}

var matrixHTMLTemplate = template.Must(
	template.New("matrix").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Matrix Federation Report</title>
<style>
*, *::before, *::after { box-sizing: border-box; }
:root {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 14px;
  line-height: 1.5;
  color: #1f2937;
  background: #f3f4f6;
}
body { margin: 0; padding: 1rem; }
code { font-family: ui-monospace, monospace; font-size: .9em; }
h2 { font-size: 1rem; font-weight: 700; margin: 0 0 .6rem; }
h3 { font-size: .9rem; font-weight: 600; margin: 0 0 .4rem; }

.hd {
  background: #fff;
  border-radius: 10px;
  padding: 1rem 1.25rem;
  margin-bottom: .75rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.08);
}
.hd h1 { margin: 0 0 .4rem; font-size: 1.15rem; font-weight: 700; }

.badge {
  display: inline-flex; align-items: center;
  padding: .2em .65em;
  border-radius: 9999px;
  font-size: .78rem; font-weight: 700;
  letter-spacing: .02em;
}
.ok   { background: #d1fae5; color: #065f46; }
.fail { background: #fee2e2; color: #991b1b; }

.version { color: #6b7280; font-size: .82rem; margin-top: .35rem; }

.section {
  background: #fff;
  border-radius: 8px;
  padding: .85rem 1rem;
  margin-bottom: .6rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.07);
}

details {
  background: #fff;
  border-radius: 8px;
  margin-bottom: .45rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.07);
  overflow: hidden;
}
.section details {
  box-shadow: none;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
  margin-bottom: .4rem;
}
summary {
  display: flex; align-items: center; gap: .5rem;
  padding: .65rem 1rem;
  cursor: pointer;
  user-select: none;
  list-style: none;
}
summary::-webkit-details-marker { display: none; }
summary::before {
  content: "▶";
  font-size: .65rem;
  color: #9ca3af;
  transition: transform .15s;
  flex-shrink: 0;
}
details[open] > summary::before { transform: rotate(90deg); }
.conn-addr { font-weight: 600; flex: 1; font-size: .9rem; font-family: ui-monospace, monospace; }

.details-body { padding: .6rem 1rem .85rem; border-top: 1px solid #f3f4f6; }

table { border-collapse: collapse; width: 100%; font-size: .85rem; }
th, td { text-align: left; padding: .3rem .5rem; border-bottom: 1px solid #f3f4f6; }
th { font-weight: 600; color: #6b7280; }

.check-ok   { color: #059669; }
.check-fail { color: #dc2626; }

.errmsg { color: #dc2626; font-size: .85rem; margin: .25rem 0 0; }
.note   { color: #6b7280; font-size: .85rem; }

ul { margin: .25rem 0; padding-left: 1.2rem; }
li { margin-bottom: .15rem; }
</style>
</head>
<body>

<div class="hd">
  <h1>Matrix Federation</h1>
  {{if .FederationOK}}
  <span class="badge ok">Federation OK</span>
  {{- else}}
  <span class="badge fail">Federation FAIL</span>
  {{- end}}
  {{if .Version}}<div class="version">Server: <code>{{.Version}}</code>{{if .VersionError}} &mdash; {{.VersionError}}{{end}}</div>{{end}}
</div>

{{if .Connections}}
<div class="section">
  <h2>Connections ({{len .Connections}})</h2>
  {{range .Connections}}
  <details{{if .Open}} open{{end}}>
    <summary>
      <span class="conn-addr">{{.Address}}</span>
      {{if .AllChecksOK}}<span class="badge ok">All checks OK</span>{{else}}<span class="badge fail">Checks failed</span>{{end}}
    </summary>
    <div class="details-body">
      {{if or .TLSVersion .CipherSuite}}
      <h3>TLS</h3>
      <p class="note">{{.TLSVersion}}{{if and .TLSVersion .CipherSuite}} &mdash; {{end}}{{.CipherSuite}}</p>
      {{end}}

      {{if .Certs}}
      <h3>Certificates</h3>
      <table>
        <tr><th>Subject</th><th>Issuer</th><th>DNS Names</th><th>Fingerprint (SHA-256)</th></tr>
        {{range .Certs}}
        <tr>
          <td><code>{{.SubjectCommonName}}</code></td>
          <td><code>{{.IssuerCommonName}}</code></td>
          <td>{{range .DNSNames}}<code>{{.}}</code> {{end}}</td>
          <td><code>{{.SHA256Fingerprint}}</code></td>
        </tr>
        {{end}}
      </table>
      {{end}}

      {{if .CheckDetails}}
      <h3 style="margin-top:.7rem">Checks</h3>
      <table>
        {{range .CheckDetails}}
        <tr>
          <td>{{if .OK}}<span class="check-ok">&#10003;</span>{{else}}<span class="check-fail">&#10007;</span>{{end}}</td>
          <td>{{.Label}}</td>
        </tr>
        {{end}}
      </table>
      {{end}}

      {{range .Errors}}<p class="errmsg">&#9888; {{.}}</p>{{end}}
    </div>
  </details>
  {{end}}
</div>
{{end}}

{{if .ConnectionErrors}}
<div class="section">
  <h2>Connection Errors ({{len .ConnectionErrors}})</h2>
  {{range .ConnectionErrors}}
  <p><code>{{.Address}}</code><br><span class="errmsg">{{.Message}}</span></p>
  {{end}}
</div>
{{end}}

<div class="section">
  <h2>Well-Known</h2>
  {{if .WellKnownServer}}
  <p>Server: <code>{{.WellKnownServer}}</code></p>
  {{else if .WellKnownResult}}
  <p class="note">{{.WellKnownResult}}</p>
  {{else}}
  <p class="note">Not found.</p>
  {{end}}
</div>

<div class="section">
  <h2>DNS Resolution</h2>
  {{if .SRVSkipped}}
  <p class="note">SRV lookup skipped{{if .SRVCName}} (CNAME: <code>{{.SRVCName}}</code>){{end}}</p>
  {{else if .SRVError}}
  <p class="errmsg">SRV error: {{.SRVError}}</p>
  {{else if .SRVRecords}}
  <h3>SRV Records</h3>
  <table>
    <tr><th>Target</th><th>Port</th><th>Priority</th><th>Weight</th></tr>
    {{range .SRVRecords}}
    <tr>
      <td><code>{{.Target}}</code></td>
      <td>{{.Port}}</td>
      <td>{{.Priority}}</td>
      <td>{{.Weight}}</td>
    </tr>
    {{end}}
  </table>
  {{else}}
  <p class="note">No SRV records found.</p>
  {{end}}

  {{if .Hosts}}
  <h3 style="margin-top:.6rem">Resolved Hosts</h3>
  {{range .Hosts}}
  <p style="margin:.25rem 0">
    <code>{{.Name}}</code>
    {{if .CName}} &rarr; <code>{{.CName}}</code>{{end}}
    {{if .Addrs}}: {{range .Addrs}}<code>{{.}}</code> {{end}}{{end}}
  </p>
  {{end}}
  {{else if .Addrs}}
  <h3 style="margin-top:.6rem">Addresses</h3>
  <ul>{{range .Addrs}}<li><code>{{.}}</code></li>{{end}}</ul>
  {{end}}
</div>

</body>
</html>`),
)

// GetHTMLReport implements happydns.CheckerHTMLReporter.
func (p *MatrixTester) GetHTMLReport(raw json.RawMessage) (string, error) {
	var r FederationTesterResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("failed to unmarshal matrix report: %w", err)
	}

	data := matrixTemplateData{
		FederationOK:    r.FederationOK,
		WellKnownServer: r.WellKnownResult.Server,
		WellKnownResult: r.WellKnownResult.Result,
		SRVSkipped:      r.DNSResult.SRVSkipped,
		SRVCName:        r.DNSResult.SRVCName,
		Addrs:           r.DNSResult.Addrs,
	}

	// Version
	if r.Version.Name != "" || r.Version.Version != "" {
		data.Version = strings.TrimSpace(r.Version.Name + " " + r.Version.Version)
	}
	data.VersionError = r.Version.Error

	// SRV records
	for _, s := range r.DNSResult.SRVRecords {
		data.SRVRecords = append(data.SRVRecords, matrixSRVRecord{
			Target:   s.Target,
			Port:     s.Port,
			Priority: s.Priority,
			Weight:   s.Weight,
		})
	}

	// SRV error
	if r.DNSResult.SRVError != nil {
		data.SRVError = r.DNSResult.SRVError.Message
	}

	// Hosts
	for name, h := range r.DNSResult.Hosts {
		data.Hosts = append(data.Hosts, matrixHostData{
			Name:  name,
			CName: h.CName,
			Addrs: h.Addrs,
		})
	}

	// Successful connections
	for addr, cr := range r.ConnectionReports {
		conn := matrixConnectionData{
			Address:     addr,
			TLSVersion:  cr.Cipher.Version,
			CipherSuite: cr.Cipher.CipherSuite,
			AllChecksOK: cr.Checks.AllChecksOK,
			Errors:      cr.Errors,
			Open:        !cr.Checks.AllChecksOK,
		}
		for _, cert := range cr.Certificates {
			conn.Certs = append(conn.Certs, matrixCertData{
				SubjectCommonName: cert.SubjectCommonName,
				IssuerCommonName:  cert.IssuerCommonName,
				SHA256Fingerprint: cert.SHA256Fingerprint,
				DNSNames:          cert.DNSNames,
			})
		}
		conn.CheckDetails = []matrixCheckItem{
			{"Matching server name", cr.Checks.MatchingServerName},
			{"Certificate valid until future", cr.Checks.FutureValidUntilTS},
			{"Valid certificates", cr.Checks.ValidCertificates},
			{"Has Ed25519 key", cr.Checks.HasEd25519Key},
			{"All Ed25519 checks OK", cr.Checks.AllEd25519ChecksOK},
		}
		data.Connections = append(data.Connections, conn)
	}

	// Failed connections
	for addr, ce := range r.ConnectionErrors {
		data.ConnectionErrors = append(data.ConnectionErrors, matrixConnErrData{
			Address: addr,
			Message: ce.Message,
		})
	}

	var buf strings.Builder
	if err := matrixHTMLTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render matrix HTML report: %w", err)
	}
	return buf.String(), nil
}
