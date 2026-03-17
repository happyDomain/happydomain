package checks

import (
	"context"
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
	RegisterChecker("domain-registration", &DomainRegistrationCheck{})
}

type DomainRegistrationCheck struct{}

func (p *DomainRegistrationCheck) ID() string   { return "domain-registration" }
func (p *DomainRegistrationCheck) Name() string { return "Domain Registration" }

func (p *DomainRegistrationCheck) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{ApplyToDomain: true}
}

func (p *DomainRegistrationCheck) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{Id: "domainName", Type: "string", Label: "Domain name", AutoFill: happydns.AutoFillDomainName, Required: true},
		},
		UserOpts: []happydns.CheckerOptionDocumentation{
			{Id: "warningDays", Type: "number", Label: "Days before expiration to warn", Default: DEFAULT_WARNING_DAYS},
			{Id: "criticalDays", Type: "number", Label: "Days before expiration to alert", Default: DEFAULT_CRITICAL_DAYS},
			{Id: "requiredStatuses", Type: "string", Label: "Required lock statuses (comma-separated)", Default: "clientTransferProhibited"},
		},
	}
}

func (p *DomainRegistrationCheck) RunCheck(ctx context.Context, options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	// 1. Extract domainName
	domainName, ok := options["domainName"].(string)
	if !ok || domainName == "" {
		return nil, fmt.Errorf("domainName is required")
	}
	domainName = strings.TrimSuffix(domainName, ".")

	// 2. Extract thresholds (with defaults)
	warningDays := extractInt(options, "warningDays", DEFAULT_WARNING_DAYS)
	criticalDays := extractInt(options, "criticalDays", DEFAULT_CRITICAL_DAYS)

	// 3. Extract required statuses
	requiredStatusesStr := "clientTransferProhibited"
	if v, ok := options["requiredStatuses"].(string); ok && v != "" {
		requiredStatusesStr = v
	}

	// 4. Try RDAP, fallback to WHOIS
	info, err := domaininfo.GetDomainRDAPInfo(ctx, happydns.Origin(domainName))
	if err != nil {
		info, err = domaininfo.GetDomainWhoisInfo(ctx, happydns.Origin(domainName))
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve domain info: %w", err)
		}
	}

	// 5. Evaluate expiration status
	expirationStatus := happydns.CheckResultStatusUnknown
	var expirationLine string

	if info.ExpirationDate == nil {
		expirationStatus = happydns.CheckResultStatusUnknown
		expirationLine = "Expiration date not available"
	} else {
		daysUntil := int(math.Ceil(time.Until(*info.ExpirationDate).Hours() / 24))

		switch {
		case daysUntil < 0:
			expirationStatus = happydns.CheckResultStatusCritical
			expirationLine = fmt.Sprintf("Domain expired %d day(s) ago (expired on %s)", -daysUntil, info.ExpirationDate.Format("2006-01-02"))
		case daysUntil < criticalDays:
			expirationStatus = happydns.CheckResultStatusCritical
			expirationLine = fmt.Sprintf("Domain expires in %d day(s) (on %s)", daysUntil, info.ExpirationDate.Format("2006-01-02"))
		case daysUntil < warningDays:
			expirationStatus = happydns.CheckResultStatusWarn
			expirationLine = fmt.Sprintf("Domain expires in %d day(s) (on %s)", daysUntil, info.ExpirationDate.Format("2006-01-02"))
		default:
			expirationStatus = happydns.CheckResultStatusOK
			expirationLine = fmt.Sprintf("Domain valid until %s (%d days)", info.ExpirationDate.Format("2006-01-02"), daysUntil)
		}
	}

	// 6. Evaluate lock status
	lockStatus := happydns.CheckResultStatusOK
	var lockLine string

	var requiredStatuses []string
	for _, s := range strings.Split(requiredStatusesStr, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			requiredStatuses = append(requiredStatuses, s)
		}
	}

	if len(requiredStatuses) > 0 {
		statusSet := make(map[string]bool, len(info.Status))
		for _, s := range info.Status {
			statusSet[s] = true
		}

		var missing []string
		for _, req := range requiredStatuses {
			if !statusSet[req] {
				missing = append(missing, req)
			}
		}

		if len(missing) > 0 {
			lockStatus = happydns.CheckResultStatusCritical
			lockLine = fmt.Sprintf("Missing lock status: %s", strings.Join(missing, ", "))
		} else {
			lockLine = fmt.Sprintf("All required statuses present: %s", strings.Join(requiredStatuses, ", "))
		}
	}

	// 7. Combine results: worst status wins (lower value = more severe)
	finalStatus := expirationStatus
	if lockStatus < finalStatus {
		finalStatus = lockStatus
	}

	statusLine := expirationLine
	if lockLine != "" {
		statusLine = expirationLine + "; " + lockLine
	}

	return &happydns.CheckResult{
		Status:     finalStatus,
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
