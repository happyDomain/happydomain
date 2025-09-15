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

func (p *MatrixTester) RunCheck(options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
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
