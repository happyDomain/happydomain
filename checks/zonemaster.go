package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
)

func init() {
	RegisterChecker("zonemaster", &ZonemasterCheck{})
}

type ZonemasterCheck struct{}

func (p *ZonemasterCheck) ID() string {
	return "zonemaster"
}

func (p *ZonemasterCheck) Name() string {
	return "Zonemaster"
}

func (p *ZonemasterCheck) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{
		ApplyToDomain: true,
	}
}

func (p *ZonemasterCheck) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:       "domainName",
				Type:     "string",
				Label:    "Domain name to check",
				AutoFill: happydns.AutoFillDomainName,
				Required: true,
			},
			{
				Id:          "profile",
				Type:        "string",
				Label:       "Profile",
				Placeholder: "default",
				Default:     "default",
			},
		},
		UserOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:      "language",
				Type:    "select",
				Label:   "Result language",
				Default: "en",
				Choices: []string{
					"en", // English
					"fr", // French
					"de", // German
					"es", // Spanish
					"sv", // Swedish
					"da", // Danish
					"fi", // Finnish
					"nb", // Norwegian Bokmål
					"nl", // Dutch
					"pt", // Portuguese
				},
			},
		},
		AdminOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:          "zonemasterAPIURL",
				Type:        "string",
				Label:       "Zonemaster API URL",
				Placeholder: "https://zonemaster.net/api",
				Default:     "https://zonemaster.net/api",
			},
		},
	}
}

// JSON-RPC request/response structures
type jsonRPCRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	ID      int    `json:"id"`
}

type jsonRPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	ID int `json:"id"`
}

// Zonemaster API structures
type startTestParams struct {
	Domain  string `json:"domain"`
	Profile string `json:"profile,omitempty"`
	IPv4    bool   `json:"ipv4,omitempty"`
	IPv6    bool   `json:"ipv6,omitempty"`
}

type testProgressParams struct {
	TestID string `json:"test_id"`
}

type getResultsParams struct {
	ID       string `json:"id"`
	Language string `json:"language"`
}

type testResult struct {
	Module   string `json:"module"`
	Message  string `json:"message"`
	Level    string `json:"level"`
	Testcase string `json:"testcase,omitempty"`
}

type zonemasterResults struct {
	CreatedAt            string            `json:"created_at"`
	HashID               string            `json:"hash_id"`
	Language             string            `json:"language,omitempty"`
	Params               map[string]any    `json:"params"`
	Results              []testResult      `json:"results"`
	TestcaseDescriptions map[string]string `json:"testcase_descriptions,omitempty"`
}

func (p *ZonemasterCheck) callJSONRPC(apiURL, method string, params any) (json.RawMessage, error) {
	reqBody := jsonRPCRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("API error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (p *ZonemasterCheck) RunCheck(options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	// Extract options
	domainName, ok := options["domainName"].(string)
	if !ok || domainName == "" {
		return nil, fmt.Errorf("domainName is required")
	}
	domainName = strings.TrimSuffix(domainName, ".")

	apiURL, ok := options["zonemasterAPIURL"].(string)
	if !ok || apiURL == "" {
		return nil, fmt.Errorf("zonemasterAPIURL is required")
	}
	apiURL = strings.TrimSuffix(apiURL, "/")

	language := "en"
	if lang, ok := options["language"].(string); ok && lang != "" {
		language = lang
	}

	profile := "default"
	if prof, ok := options["profile"].(string); ok && prof != "" {
		profile = prof
	}

	// Step 1: Start the test
	startParams := startTestParams{
		Domain:  domainName,
		Profile: profile,
		IPv4:    true,
		IPv6:    true,
	}

	result, err := p.callJSONRPC(apiURL, "start_domain_test", startParams)
	if err != nil {
		return nil, fmt.Errorf("failed to start test: %w", err)
	}

	var testID string
	if err = json.Unmarshal(result, &testID); err != nil {
		return nil, fmt.Errorf("failed to parse test ID: %w", err)
	}

	if testID == "" {
		return nil, fmt.Errorf("received empty test ID")
	}

	// Step 2: Poll for test completion
	progressParams := testProgressParams{TestID: testID}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("test timeout after 10 minutes (test ID: %s)", testID)

		case <-ticker.C:
			result, err := p.callJSONRPC(apiURL, "test_progress", progressParams)
			if err != nil {
				return nil, fmt.Errorf("failed to test progress: %w", err)
			}

			var progress float64
			if err := json.Unmarshal(result, &progress); err != nil {
				return nil, fmt.Errorf("failed to parse progress: %w", err)
			}

			if progress >= 100 {
				goto testComplete
			}
		}
	}

testComplete:
	// Step 3: Get test results
	resultsParams := getResultsParams{
		ID:       testID,
		Language: language,
	}

	result, err = p.callJSONRPC(apiURL, "get_test_results", resultsParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}

	var results zonemasterResults
	if err := json.Unmarshal(result, &results); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}
	results.Language = language

	// Analyze results to determine overall status
	var (
		errorCount   int
		warningCount int
		infoCount    int
		criticalMsgs []string
	)

	for _, r := range results.Results {
		switch strings.ToUpper(r.Level) {
		case "CRITICAL", "ERROR":
			errorCount++
			if len(criticalMsgs) < 5 { // Keep first 5 critical messages
				criticalMsgs = append(criticalMsgs, r.Message)
			}
		case "WARNING":
			warningCount++
		case "INFO", "NOTICE":
			infoCount++
		}
	}

	// Determine status
	var status happydns.CheckResultStatus
	var statusLine string

	if errorCount > 0 {
		status = happydns.CheckResultStatusCritical
		statusLine = fmt.Sprintf("%d error(s), %d warning(s) found", errorCount, warningCount)
		if len(criticalMsgs) > 0 {
			statusLine += ": " + strings.Join(criticalMsgs[:min(2, len(criticalMsgs))], "; ")
		}
	} else if warningCount > 0 {
		status = happydns.CheckResultStatusWarn
		statusLine = fmt.Sprintf("%d warning(s) found", warningCount)
	} else {
		status = happydns.CheckResultStatusOK
		statusLine = fmt.Sprintf("All checks passed (%d checks)", len(results.Results))
	}

	return &happydns.CheckResult{
		Status:     status,
		StatusLine: statusLine,
		Report:     results,
	}, nil
}

// ── HTML report ───────────────────────────────────────────────────────────────

// zmLevelDisplayOrder defines the severity order used for sorting and display.
var zmLevelDisplayOrder = []string{"CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG"}

var zmLevelRank = func() map[string]int {
	m := make(map[string]int, len(zmLevelDisplayOrder))
	for i, l := range zmLevelDisplayOrder {
		m[l] = len(zmLevelDisplayOrder) - i
	}
	return m
}()

type zmLevelCount struct {
	Level string
	Count int
}

type zmModuleGroup struct {
	Name     string
	Position int // first-seen index, used as tiebreaker in sort
	Results  []testResult
	Levels   []zmLevelCount // sorted by severity desc, zeros omitted
	Worst    string
	Open     bool
}

type zmTemplateData struct {
	Domain    string
	CreatedAt string
	HashID    string
	Language  string
	Modules   []zmModuleGroup
	Totals    []zmLevelCount // sorted by severity desc, zeros omitted
}

var zonemasterHTMLTemplate = template.Must(
	template.New("zonemaster").
		Funcs(template.FuncMap{
			"badgeClass": func(level string) string {
				switch strings.ToUpper(level) {
				case "CRITICAL":
					return "badge-critical"
				case "ERROR":
					return "badge-error"
				case "WARNING":
					return "badge-warning"
				case "NOTICE":
					return "badge-notice"
				case "INFO":
					return "badge-info"
				default:
					return "badge-debug"
				}
			},
		}).
		Parse(`<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Zonemaster{{if .Domain}} — {{.Domain}}{{end}}</title>
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
a { color: inherit; }
code { font-family: ui-monospace, monospace; font-size: .9em; }

/* Header card */
.hd {
  background: #fff;
  border-radius: 10px;
  padding: 1rem 1.25rem 1.1rem;
  margin-bottom: .75rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.08);
}
.hd h1 { margin: 0 0 .2rem; font-size: 1.15rem; font-weight: 700; }
.hd .meta { color: #6b7280; font-size: .82rem; margin-bottom: .6rem; }
.totals { display: flex; gap: .35rem; flex-wrap: wrap; }

/* Badges */
.badge {
  display: inline-flex; align-items: center;
  padding: .18em .55em;
  border-radius: 9999px;
  font-size: .72rem; font-weight: 700;
  letter-spacing: .02em; white-space: nowrap;
}
.badge-critical { background: #fee2e2; color: #991b1b; }
.badge-error    { background: #ffedd5; color: #9a3412; }
.badge-warning  { background: #fef3c7; color: #92400e; }
.badge-notice   { background: #e0f2fe; color: #075985; }
.badge-info     { background: #dbeafe; color: #1e40af; }
.badge-debug    { background: #f3f4f6; color: #4b5563; }

/* Accordion */
details {
  background: #fff;
  border-radius: 8px;
  margin-bottom: .45rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.07);
  overflow: hidden;
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
.mod-name { font-weight: 600; flex: 1; font-size: .9rem; }
.mod-badges { display: flex; gap: .25rem; flex-wrap: wrap; }

/* Result rows */
.results { border-top: 1px solid #f3f4f6; }
.row {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: .6rem;
  padding: .45rem 1rem;
  border-bottom: 1px solid #f9fafb;
  align-items: start;
}
.row:last-child { border-bottom: none; }
.row-msg { color: #374151; }
.row-tc  { font-size: .75rem; color: #9ca3af; }
</style>
</head>
<body>

<div class="hd">
  <h1>Zonemaster{{if .Domain}} — <code>{{.Domain}}</code>{{end}}</h1>
  <div class="meta">
    {{- if .CreatedAt}}Run at {{.CreatedAt}}{{end -}}
    {{- if and .CreatedAt .HashID}} &middot; {{end -}}
    {{- if .HashID}}ID: <code>{{.HashID}}</code>{{end -}}
  </div>
  <div class="totals">
    {{- range .Totals}}
    <span class="badge {{badgeClass .Level}}">{{.Level}}&nbsp;{{.Count}}</span>
    {{- end}}
  </div>
</div>

{{range .Modules -}}
<details{{if .Open}} open{{end}}>
  <summary>
    <span class="mod-name">{{.Name}}</span>
    <span class="mod-badges">
      {{- range .Levels}}
      <span class="badge {{badgeClass .Level}}">{{.Count}}</span>
      {{- end}}
    </span>
  </summary>
  <div class="results">
    {{- range .Results}}
    <div class="row">
      <span class="badge {{badgeClass .Level}}">{{.Level}}</span>
      <div>
        <div class="row-msg">{{.Message}}</div>
        {{- if .Testcase}}<div class="row-tc">{{.Testcase}}</div>{{end}}
      </div>
    </div>
    {{- end}}
  </div>
</details>
{{end -}}

</body>
</html>`),
)

// GetHTMLReport implements happydns.CheckerHTMLReporter.
func (p *ZonemasterCheck) GetHTMLReport(raw json.RawMessage) (string, error) {
	var results zonemasterResults
	if err := json.Unmarshal(raw, &results); err != nil {
		return "", fmt.Errorf("failed to unmarshal zonemaster results: %w", err)
	}

	// Group results by module, preserving first-seen order.
	moduleOrder := []string{}
	moduleMap := map[string][]testResult{}
	for _, r := range results.Results {
		if _, seen := moduleMap[r.Module]; !seen {
			moduleOrder = append(moduleOrder, r.Module)
		}
		moduleMap[r.Module] = append(moduleMap[r.Module], r)
	}

	totalCounts := map[string]int{}

	var modules []zmModuleGroup
	for _, name := range moduleOrder {
		rs := moduleMap[name]
		counts := map[string]int{}
		for _, r := range rs {
			lvl := strings.ToUpper(r.Level)
			counts[lvl]++
			totalCounts[lvl]++
		}

		// Find worst level and build sorted level-count slice.
		worst := ""
		worstRank := -1
		var levels []zmLevelCount
		for _, l := range zmLevelDisplayOrder {
			if n, ok := counts[l]; ok && n > 0 {
				levels = append(levels, zmLevelCount{Level: l, Count: n})
				if zmLevelRank[l] > worstRank {
					worstRank = zmLevelRank[l]
					worst = l
				}
			}
		}
		// Append any unknown levels last.
		for l, n := range counts {
			if _, known := zmLevelRank[l]; !known {
				levels = append(levels, zmLevelCount{Level: l, Count: n})
			}
		}

		modules = append(modules, zmModuleGroup{
			Name:     name,
			Position: len(modules),
			Results:  rs,
			Levels:   levels,
			Worst:    worst,
			Open:     worst == "CRITICAL" || worst == "ERROR",
		})
	}

	// Sort modules: most severe first, then by original appearance order.
	sort.Slice(modules, func(i, j int) bool {
		ri, rj := zmLevelRank[modules[i].Worst], zmLevelRank[modules[j].Worst]
		if ri != rj {
			return ri > rj
		}
		return modules[i].Position < modules[j].Position
	})

	// Build sorted totals slice.
	var totals []zmLevelCount
	for _, l := range zmLevelDisplayOrder {
		if n, ok := totalCounts[l]; ok && n > 0 {
			totals = append(totals, zmLevelCount{Level: l, Count: n})
		}
	}

	domain := ""
	if d, ok := results.Params["domain"]; ok {
		domain = fmt.Sprintf("%v", d)
	}

	lang := results.Language
	if lang == "" {
		lang = "en"
	}

	data := zmTemplateData{
		Domain:    domain,
		CreatedAt: results.CreatedAt,
		HashID:    results.HashID,
		Language:  lang,
		Modules:   modules,
		Totals:    totals,
	}

	var buf strings.Builder
	if err := zonemasterHTMLTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render zonemaster HTML report: %w", err)
	}
	return buf.String(), nil
}
