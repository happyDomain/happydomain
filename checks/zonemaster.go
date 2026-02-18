package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	if err := json.Unmarshal(result, &testID); err != nil {
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
