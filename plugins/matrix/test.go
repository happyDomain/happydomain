package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"git.happydns.org/happyDomain/model"
)

type MatrixTester struct {
	TesterURI string
}

func (p *MatrixTester) PluginEnvName() []string {
	return []string{
		"matrixim",
	}
}

func (p *MatrixTester) Version() happydns.PluginVersionInfo {
	return happydns.PluginVersionInfo{
		Name:    "Matrix Federation Tester",
		Version: "0.1",
		AvailableOn: happydns.PluginAvailability{
			ApplyToService:  true,
			LimitToServices: []string{"abstract.MatrixIM"},
		},
	}
}

func (p *MatrixTester) AvailableOptions() happydns.PluginOptionsDocumentation {
	return happydns.PluginOptionsDocumentation{
		RunOpts: []happydns.PluginOptionDocumentation{
			{
				Id:          "serviceDomain",
				Type:        "string",
				Label:       "Matrix domain",
				Placeholder: "matrix.org",
				Default:     "matrix.org",
				Required:    true,
			},
		},
		AdminOpts: []happydns.PluginOptionDocumentation{
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
		Server string `json:"m.server"`
		Result string `json:"result"`
	}
	DNSResult struct {
		SRVError *struct {
			Message string
		}
	}
	ConnectionReports map[string]struct {
		Errors []string
	}
	ConnectionErrors map[string]struct {
		Message string
	}
	Version struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	FederationOK bool `json:"FederationOK"`
}

func (p *MatrixTester) RunTest(options happydns.PluginOptions, meta map[string]string) (*happydns.PluginResult, error) {
	var domain string

	if dn, ok := options["domain"]; ok {
		domain, _ = dn.(string)
	} else if origin, ok := options["origin"]; ok {
		domain, _ = origin.(string)
	}

	if domain == "" {
		return nil, fmt.Errorf("domain not defined")
	}

	domain = strings.TrimSuffix(domain, ".")

	resp, err := http.Get(fmt.Sprintf(p.TesterURI, domain))
	if err != nil {
		return nil, fmt.Errorf("unable to perform the test: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Sorry, the federation tester is broken. Check on https://federationtester.matrix.org/#%s", strings.TrimSuffix(domain, "."))
	}

	var status happydns.PluginResultStatus
	var statusLine string
	var federationTest FederationTesterResponse

	err = json.NewDecoder(resp.Body).Decode(&federationTest)
	if err != nil {
		log.Printf("Error in check_matrix_federation, when decoding json: %s", err.Error())
		return nil, fmt.Errorf("sorry, the federation tester is broken. Check on https://federationtester.matrix.org/#%s", strings.TrimSuffix(domain, "."))
	}

	if federationTest.FederationOK {
		status = happydns.PluginResultStatusOK
		statusLine = "Running " + federationTest.Version.Name + " " + federationTest.Version.Version
	} else {
		status = happydns.PluginResultStatusKO

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

	return &happydns.PluginResult{
		Status:     status,
		StatusLine: statusLine,
		Report:     federationTest,
	}, nil
}
