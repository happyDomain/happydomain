package checks

import (
	"fmt"
	"math"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

const (
	DEFAULT_WARNING_DAYS  = 30
	DEFAULT_CRITICAL_DAYS = 7
)

func init() {
	RegisterChecker("domain-expiration", &DomainExpirationCheck{})
}

type DomainExpirationCheck struct{}

func (p *DomainExpirationCheck) ID() string   { return "domain-expiration" }
func (p *DomainExpirationCheck) Name() string { return "Domain Expiration" }

func (p *DomainExpirationCheck) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{ApplyToDomain: true}
}

func (p *DomainExpirationCheck) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{Id: "domainName", Type: "string", Label: "Domain name", AutoFill: happydns.AutoFillDomainName, Required: true},
		},
		UserOpts: []happydns.CheckerOptionDocumentation{
			{Id: "warningDays", Type: "number", Label: "Days before expiration to warn", Default: DEFAULT_WARNING_DAYS},
			{Id: "criticalDays", Type: "number", Label: "Days before expiration to alert", Default: DEFAULT_CRITICAL_DAYS},
		},
	}
}

func (p *DomainExpirationCheck) RunCheck(options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	// 1. Extract domainName
	domainName, ok := options["domainName"].(string)
	if !ok || domainName == "" {
		return nil, fmt.Errorf("domainName is required")
	}
	domainName = strings.TrimSuffix(domainName, ".")

	// 2. Extract thresholds (with defaults)
	warningDays := extractInt(options, "warningDays", DEFAULT_WARNING_DAYS)
	criticalDays := extractInt(options, "criticalDays", DEFAULT_CRITICAL_DAYS)

	// 3. Try RDAP, fallback to WHOIS
	info, err := domaininfo.GetDomainRDAPInfo(happydns.Origin(domainName))
	if err != nil {
		info, err = domaininfo.GetDomainWhoisInfo(happydns.Origin(domainName))
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve domain info: %w", err)
		}
	}

	// 4. Check expiration date presence
	if info.ExpirationDate == nil {
		return &happydns.CheckResult{
			Status:     happydns.CheckResultStatusUnknown,
			StatusLine: "Expiration date not available",
			Report:     info,
		}, nil
	}

	// 5. Compute days remaining
	daysUntil := int(math.Ceil(time.Until(*info.ExpirationDate).Hours() / 24))

	// 6. Determine status
	var status happydns.CheckResultStatus
	var statusLine string
	switch {
	case daysUntil < 0:
		status = happydns.CheckResultStatusCritical
		statusLine = fmt.Sprintf("Domain expired %d day(s) ago (expired on %s)", -daysUntil, info.ExpirationDate.Format("2006-01-02"))
	case daysUntil < criticalDays:
		status = happydns.CheckResultStatusCritical
		statusLine = fmt.Sprintf("Domain expires in %d day(s) (on %s)", daysUntil, info.ExpirationDate.Format("2006-01-02"))
	case daysUntil < warningDays:
		status = happydns.CheckResultStatusWarn
		statusLine = fmt.Sprintf("Domain expires in %d day(s) (on %s)", daysUntil, info.ExpirationDate.Format("2006-01-02"))
	default:
		status = happydns.CheckResultStatusOK
		statusLine = fmt.Sprintf("Domain valid until %s (%d days)", info.ExpirationDate.Format("2006-01-02"), daysUntil)
	}

	return &happydns.CheckResult{
		Status:     status,
		StatusLine: statusLine,
		Report:     info,
	}, nil
}

// extractInt reads an int/float64 option with a default fallback.
func extractInt(options happydns.CheckerOptions, key string, def int) int {
	if v, ok := options[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return def
}
