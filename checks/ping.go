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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

func init() {
	RegisterChecker("ping", &PingCheck{})
}

// PingReport contains the results of a ping check across one or more targets.
type PingReport struct {
	Targets []PingTargetResult `json:"targets"`
}

// PingTargetResult contains the ping statistics for a single IP address.
type PingTargetResult struct {
	Address    string  `json:"address"`
	RTTMin     float64 `json:"rtt_min"`
	RTTAvg     float64 `json:"rtt_avg"`
	RTTMax     float64 `json:"rtt_max"`
	PacketLoss float64 `json:"packet_loss"`
	Sent       int     `json:"sent"`
	Received   int     `json:"received"`
}

type PingCheck struct{}

func (p *PingCheck) ID() string {
	return "ping"
}

func (p *PingCheck) Name() string {
	return "Ping (ICMP)"
}

func (p *PingCheck) CheckInterval() happydns.CheckIntervalSpec {
	return happydns.CheckIntervalSpec{
		Min:     1 * time.Minute,
		Max:     1 * time.Hour,
		Default: 5 * time.Minute,
	}
}

func (p *PingCheck) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{
		ApplyToService:  true,
		LimitToServices: []string{"abstract.Server"},
	}
}

func (p *PingCheck) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		ServiceOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:      "warningRTT",
				Type:    "number",
				Label:   "Warning RTT threshold (ms)",
				Default: float64(100),
			},
			{
				Id:      "criticalRTT",
				Type:    "uint",
				Label:   "Critical RTT threshold (ms)",
				Default: float64(500),
			},
			{
				Id:      "warningPacketLoss",
				Type:    "uint",
				Label:   "Warning packet loss threshold (%)",
				Default: float64(10),
			},
			{
				Id:      "criticalPacketLoss",
				Type:    "uint",
				Label:   "Critical packet loss threshold (%)",
				Default: float64(50),
			},
			{
				Id:      "count",
				Type:    "uint",
				Label:   "Number of pings to send",
				Default: float64(5),
			},
		},
		RunOpts: []happydns.CheckerOptionDocumentation{
			{
				Id:       "service",
				Label:    "Service",
				AutoFill: happydns.AutoFillService,
			},
		},
	}
}

func getFloatOption(options happydns.CheckerOptions, key string, defaultVal float64) float64 {
	v, ok := options[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case float64:
		return val
	case json.Number:
		f, err := val.Float64()
		if err != nil {
			return defaultVal
		}
		return f
	default:
		return defaultVal
	}
}

func getIntOption(options happydns.CheckerOptions, key string, defaultVal int) int {
	return int(getFloatOption(options, key, float64(defaultVal)))
}

// ipsFromServiceOption extracts the IP addresses directly from the auto-filled
// service body (abstract.Server). Only IPs actually present in the service
// definition are returned, so an IPv4-only service won't trigger IPv6 pings.
// The service JSON has the shape:
//
//	{"_svctype":"abstract.Server","Service":{"A":{...},"AAAA":{...}}}
func ipsFromServiceOption(svc *happydns.ServiceMessage) []net.IP {
	var server abstract.Server
	if err := json.Unmarshal(svc.Service, &server); err != nil {
		return nil
	}

	var ips []net.IP
	if server.A != nil && len(server.A.A) > 0 {
		ips = append(ips, server.A.A)
	}
	if server.AAAA != nil && len(server.AAAA.AAAA) > 0 {
		ips = append(ips, server.AAAA.AAAA)
	}
	return ips
}

func (p *PingCheck) RunCheck(ctx context.Context, options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	service, ok := options["service"].(*happydns.ServiceMessage)
	if !ok {
		return nil, fmt.Errorf("service not defined")
	}
	if service.Type != "abstract.Server" {
		return nil, fmt.Errorf("service is %s, expected abstract.Server", service.Type)
	}

	warningRTT := getFloatOption(options, "warningRTT", 100)
	criticalRTT := getFloatOption(options, "criticalRTT", 500)
	warningPacketLoss := getFloatOption(options, "warningPacketLoss", 10)
	criticalPacketLoss := getFloatOption(options, "criticalPacketLoss", 50)
	count := getIntOption(options, "count", 5)

	if count < 1 {
		count = 1
	}
	if count > 20 {
		count = 20
	}

	// Prefer IPs from the service definition; fall back to live DNS.
	// Using service IPs avoids pinging addresses not defined in the service
	// (e.g. live IPv6 records that the service doesn't have).
	var rawIPs []net.IP
	if serviceIPs := ipsFromServiceOption(service); len(serviceIPs) > 0 {
		rawIPs = serviceIPs
	}
	if len(rawIPs) == 0 {
		return nil, fmt.Errorf("no IP addresses found for %s", service.Domain)
	}

	report := PingReport{}
	var overallStatus happydns.CheckResultStatus = happydns.CheckResultStatusOK
	var summaryParts []string
	var errs []error

	for _, ip := range rawIPs {
		addr := ip.String()

		pinger, err := probing.NewPinger(addr)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create pinger for %s: %w", addr, err))
			continue
		}

		pinger.Count = count
		pinger.Timeout = time.Duration(count)*time.Second + 5*time.Second

		if err = pinger.RunWithContext(ctx); err != nil {
			errs = append(errs, fmt.Errorf("ping failed for %s: %w", addr, err))
			continue
		}

		stats := pinger.Statistics()
		target := PingTargetResult{
			Address:    addr,
			RTTMin:     float64(stats.MinRtt.Microseconds()) / 1000.0,
			RTTAvg:     float64(stats.AvgRtt.Microseconds()) / 1000.0,
			RTTMax:     float64(stats.MaxRtt.Microseconds()) / 1000.0,
			PacketLoss: stats.PacketLoss,
			Sent:       stats.PacketsSent,
			Received:   stats.PacketsRecv,
		}
		report.Targets = append(report.Targets, target)

		if target.PacketLoss >= criticalPacketLoss || target.RTTAvg >= criticalRTT {
			overallStatus = happydns.CheckResultStatusCritical
		} else if (target.PacketLoss >= warningPacketLoss || target.RTTAvg >= warningRTT) && overallStatus > happydns.CheckResultStatusWarn {
			overallStatus = happydns.CheckResultStatusWarn
		}

		summaryParts = append(summaryParts, fmt.Sprintf("%s: %.1fms avg, %.0f%% loss", addr, target.RTTAvg, target.PacketLoss))
	}

	// If no IP responded at all, return the combined errors as a fatal error.
	if len(report.Targets) == 0 {
		return nil, errors.Join(errs...)
	}

	return &happydns.CheckResult{
		Status:     overallStatus,
		StatusLine: strings.Join(summaryParts, " | "),
		Report:     report,
	}, errors.Join(errs...)
}

// ExtractMetrics implements happydns.CheckerMetricsReporter.
func (p *PingCheck) ExtractMetrics(results []*happydns.CheckResult) (*happydns.MetricsReport, error) {
	type seriesKey struct {
		metric  string
		address string
	}
	seriesMap := map[seriesKey]*happydns.MetricSeries{}
	var seriesOrder []seriesKey

	for _, result := range results {
		if result.Report == nil {
			continue
		}

		raw, err := json.Marshal(result.Report)
		if err != nil {
			continue
		}

		var report PingReport
		if err := json.Unmarshal(raw, &report); err != nil {
			continue
		}

		for _, target := range report.Targets {
			ts := result.ExecutedAt

			metrics := []struct {
				suffix string
				label  string
				unit   string
				value  float64
			}{
				{"rtt_avg", "RTT Avg", "ms", target.RTTAvg},
				{"rtt_min", "RTT Min", "ms", target.RTTMin},
				{"rtt_max", "RTT Max", "ms", target.RTTMax},
				{"packet_loss", "Packet Loss", "%", target.PacketLoss},
			}

			for _, m := range metrics {
				key := seriesKey{metric: m.suffix, address: target.Address}
				s, exists := seriesMap[key]
				if !exists {
					s = &happydns.MetricSeries{
						Name:  fmt.Sprintf("%s_%s", m.suffix, target.Address),
						Label: fmt.Sprintf("%s (%s)", m.label, target.Address),
						Unit:  m.unit,
					}
					seriesMap[key] = s
					seriesOrder = append(seriesOrder, key)
				}
				s.Points = append(s.Points, happydns.MetricPoint{
					Timestamp: ts,
					Value:     m.value,
				})
			}
		}
	}

	var series []happydns.MetricSeries
	for _, key := range seriesOrder {
		series = append(series, *seriesMap[key])
	}

	return &happydns.MetricsReport{Series: series}, nil
}
