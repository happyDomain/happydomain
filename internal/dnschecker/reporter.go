// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

// reporter.go exposes the optional reporting surface of observation
// providers: HTML rendering and metrics extraction. Providers opt in by
// implementing CheckerHTMLReporter / CheckerMetricsReporter; the helpers
// here resolve providers (globally or via an ObservationContext override)
// and dispatch report/metric calls. BuildReportContext wires raw observation
// data, check states, and a lazy Related(key) resolver into a SDK-shaped
// ReportContext so reporters can pull cross-checker data on demand.

package dnschecker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/model"
)

// Provider registration is startup-only (see comments on the registries in
// internal/serviceanalyzer/registry.go and internal/providerregistry/registry.go),
// so the "any provider implements X reporter" question has a fixed answer for
// the process lifetime. We compute it once on first call and cache it.
var (
	htmlReporterOnce      sync.Once
	htmlReporterCached    bool
	metricsReporterOnce   sync.Once
	metricsReporterCached bool
)

// HasHTMLReporter returns true if any registered observation provider implements CheckerHTMLReporter.
func HasHTMLReporter() bool {
	htmlReporterOnce.Do(func() {
		for _, p := range sdk.GetObservationProviders() {
			if _, ok := p.(happydns.CheckerHTMLReporter); ok {
				htmlReporterCached = true
				return
			}
		}
	})
	return htmlReporterCached
}

// GetHTMLReport renders an HTML report for the given observation key.
// Returns (html, true, nil) if the provider supports HTML reports, or ("", false, nil) if not.
func GetHTMLReport(key happydns.ObservationKey, rc happydns.ReportContext) (string, bool, error) {
	return getHTMLReport(sdk.FindObservationProvider(key), key, rc)
}

// GetHTMLReportCtx is like GetHTMLReport but resolves the provider through
// the ObservationContext, respecting per-context overrides.
func (oc *ObservationContext) GetHTMLReportCtx(key happydns.ObservationKey, rc happydns.ReportContext) (string, bool, error) {
	return getHTMLReport(oc.getProvider(key), key, rc)
}

func getHTMLReport(provider happydns.ObservationProvider, key happydns.ObservationKey, rc happydns.ReportContext) (string, bool, error) {
	if provider == nil {
		return "", false, fmt.Errorf("no observation provider registered for key %q", key)
	}

	hr, ok := provider.(happydns.CheckerHTMLReporter)
	if !ok {
		return "", false, nil
	}
	html, err := hr.GetHTMLReport(rc)
	return html, true, err
}

// HasMetricsReporter returns true if any registered observation provider implements CheckerMetricsReporter.
func HasMetricsReporter() bool {
	metricsReporterOnce.Do(func() {
		for _, p := range sdk.GetObservationProviders() {
			if _, ok := p.(happydns.CheckerMetricsReporter); ok {
				metricsReporterCached = true
				return
			}
		}
	})
	return metricsReporterCached
}

// GetMetrics extracts metrics for the given observation key.
// Returns (metrics, true, nil) if the provider supports metrics, or (nil, false, nil) if not.
func GetMetrics(key happydns.ObservationKey, rc happydns.ReportContext, collectedAt time.Time) ([]happydns.CheckMetric, bool, error) {
	return getMetrics(sdk.FindObservationProvider(key), key, rc, collectedAt)
}

// GetMetricsCtx is like GetMetrics but resolves the provider through
// the ObservationContext, respecting per-context overrides.
func (oc *ObservationContext) GetMetricsCtx(key happydns.ObservationKey, rc happydns.ReportContext, collectedAt time.Time) ([]happydns.CheckMetric, bool, error) {
	return getMetrics(oc.getProvider(key), key, rc, collectedAt)
}

func getMetrics(provider happydns.ObservationProvider, key happydns.ObservationKey, rc happydns.ReportContext, collectedAt time.Time) ([]happydns.CheckMetric, bool, error) {
	if provider == nil {
		return nil, false, fmt.Errorf("no observation provider registered for key %q", key)
	}

	mr, ok := provider.(happydns.CheckerMetricsReporter)
	if !ok {
		return nil, false, nil
	}
	metrics, err := mr.ExtractMetrics(rc, collectedAt)
	return metrics, true, err
}

// BuildReportContext wires raw, states, and a lazy Related(key) resolver into
// a ReportContext. Pass nil states when none are available; reporters fall
// back to data-only rendering.
func BuildReportContext(ctx context.Context, producerCheckerID string, target happydns.CheckTarget, raw json.RawMessage, lookup RelatedObservationLookup, states []happydns.CheckState) happydns.ReportContext {
	if lookup == nil || producerCheckerID == "" {
		return sdk.NewReportContext(raw, nil, states)
	}
	return &lazyReportContext{
		ctx:      ctx,
		data:     raw,
		lookup:   lookup,
		producer: producerCheckerID,
		target:   target,
		states:   states,
		cache:    make(map[happydns.ObservationKey][]happydns.RelatedObservation),
	}
}

// lazyReportContext resolves Related(key) on first access against a host-side lookup closure.
type lazyReportContext struct {
	mu       sync.Mutex
	ctx      context.Context
	data     json.RawMessage
	lookup   RelatedObservationLookup
	producer string
	target   happydns.CheckTarget
	states   []happydns.CheckState
	cache    map[happydns.ObservationKey][]happydns.RelatedObservation
}

func (l *lazyReportContext) Data() json.RawMessage         { return l.data }
func (l *lazyReportContext) States() []happydns.CheckState { return l.states }
func (l *lazyReportContext) Related(key happydns.ObservationKey) []happydns.RelatedObservation {
	l.mu.Lock()
	defer l.mu.Unlock()
	if cached, ok := l.cache[key]; ok {
		return cached
	}
	out, err := l.lookup(l.ctx, l.producer, l.target, key)
	if err != nil {
		log.Printf("lazyReportContext: Related(%q): %v", key, err)
		return nil
	}
	l.cache[key] = out
	return out
}

// GetAllMetrics extracts metrics from all observation keys in a snapshot.
func GetAllMetrics(snap *happydns.ObservationSnapshot) ([]happydns.CheckMetric, error) {
	var allMetrics []happydns.CheckMetric
	var errs []error
	for key, raw := range snap.Data {
		metrics, supported, err := GetMetrics(key, sdk.StaticReportContext(raw), snap.CollectedAt)
		if err != nil {
			errs = append(errs, fmt.Errorf("observation %q: %w", key, err))
			continue
		}
		if !supported {
			continue
		}
		allMetrics = append(allMetrics, metrics...)
	}
	return allMetrics, errors.Join(errs...)
}
