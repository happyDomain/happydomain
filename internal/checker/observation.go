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

// observation.go implements the observation subsystem, which is the data
// collection layer for the checker framework. An observation represents a
// piece of raw data gathered about a check target (e.g. DNS records, HTTP
// headers, TLS certificate details). Observations are identified by an
// ObservationKey and collected on demand by registered ObservationProviders.
//
// The ObservationContext provides lazy-loading, cached, thread-safe access to
// observations: the first checker that requests a given observation triggers
// its collection, and subsequent checkers reuse the cached result. This
// design decouples data collection from evaluation: checkers declare which
// observations they need, and the context ensures each is collected at most
// once per check run. Observations can also be persisted as snapshots and
// reused across runs when freshness requirements allow.
//
// Observation providers may optionally implement reporting interfaces
// (CheckerHTMLReporter, CheckerMetricsReporter) to produce human-readable
// reports or extract time-series metrics from collected data.

package checker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	sdk "git.happydns.org/checker-sdk-go/checker"
	"git.happydns.org/happyDomain/model"
)

// ObservationCacheLookup resolves a cached observation for a target+key.
// Returns the raw data and collection time, or an error if not cached.
type ObservationCacheLookup func(target happydns.CheckTarget, key happydns.ObservationKey) (json.RawMessage, time.Time, error)

// ObservationContext provides lazy-loading, cached, thread-safe access to observation data.
// Collected data is serialized to json.RawMessage immediately after collection.
//
// Concurrency model: the outer mu protects only the cache/errors/inflight
// maps and is held for short critical sections. Provider collection runs
// *without* mu held, so two calls to Get for *different* keys can collect
// concurrently. Two calls for the *same* key are deduplicated: the first
// installs an inflight channel, runs the collection, then closes the
// channel; the others wait on it and read the cached result afterwards.
type ObservationContext struct {
	target           happydns.CheckTarget
	opts             happydns.CheckerOptions
	cache            map[happydns.ObservationKey]json.RawMessage
	errors           map[happydns.ObservationKey]error
	inflight         map[happydns.ObservationKey]chan struct{}
	mu               sync.Mutex
	cacheLookup      ObservationCacheLookup // nil = no DB cache
	freshness        time.Duration          // 0 = always collect
	providerOverride map[happydns.ObservationKey]happydns.ObservationProvider
}

// NewObservationContext creates a new ObservationContext for the given target and options.
// cacheLookup and freshness enable cross-checker observation reuse from stored snapshots.
// Pass nil and 0 to disable DB-based caching.
func NewObservationContext(target happydns.CheckTarget, opts happydns.CheckerOptions, cacheLookup ObservationCacheLookup, freshness time.Duration) *ObservationContext {
	return &ObservationContext{
		target:      target,
		opts:        opts,
		cache:       make(map[happydns.ObservationKey]json.RawMessage),
		errors:      make(map[happydns.ObservationKey]error),
		inflight:    make(map[happydns.ObservationKey]chan struct{}),
		cacheLookup: cacheLookup,
		freshness:   freshness,
	}
}

// SetProviderOverride registers a per-context provider that takes precedence
// over the global registry for the given observation key.  This is used to
// substitute local providers with HTTP-backed ones when an endpoint is configured.
func (oc *ObservationContext) SetProviderOverride(key happydns.ObservationKey, p happydns.ObservationProvider) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	if oc.providerOverride == nil {
		oc.providerOverride = make(map[happydns.ObservationKey]happydns.ObservationProvider)
	}
	oc.providerOverride[key] = p
}

// getProvider returns the observation provider for the given key, checking
// per-context overrides first, then falling back to the global registry.
// Safe to call without holding oc.mu - it acquires the lock internally.
func (oc *ObservationContext) getProvider(key happydns.ObservationKey) happydns.ObservationProvider {
	oc.mu.Lock()
	override := oc.providerOverride
	oc.mu.Unlock()
	if override != nil {
		if p, ok := override[key]; ok {
			return p
		}
	}
	return sdk.FindObservationProvider(key)
}

// Get collects observation data for the given key (lazily) and unmarshals it into dest.
// Thread-safe: concurrent calls for the same key are deduplicated; concurrent
// calls for different keys collect in parallel.
func (oc *ObservationContext) Get(ctx context.Context, key happydns.ObservationKey, dest any) error {
	for {
		oc.mu.Lock()
		if raw, ok := oc.cache[key]; ok {
			oc.mu.Unlock()
			return json.Unmarshal(raw, dest)
		}
		if err, ok := oc.errors[key]; ok {
			oc.mu.Unlock()
			return err
		}
		if ch, ok := oc.inflight[key]; ok {
			// Another goroutine is already collecting this key. Release
			// the lock, wait for it to finish, then re-check the cache.
			oc.mu.Unlock()
			select {
			case <-ch:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// We are the leader for this key. Install the inflight channel
		// before releasing the lock so concurrent callers wait on us.
		ch := make(chan struct{})
		oc.inflight[key] = ch
		oc.mu.Unlock()

		raw, collectErr := oc.collect(ctx, key)

		// Collection errors are cached for the lifetime of this
		// ObservationContext (i.e. a single execution run). This is
		// intentional: within one run the same transient failure would
		// keep recurring, and retrying would slow down the pipeline.
		// A new execution creates a fresh context, giving the provider
		// another chance.
		oc.mu.Lock()
		if collectErr != nil {
			oc.errors[key] = collectErr
		} else {
			oc.cache[key] = raw
		}
		delete(oc.inflight, key)
		close(ch)
		oc.mu.Unlock()

		if collectErr != nil {
			return collectErr
		}
		return json.Unmarshal(raw, dest)
	}
}

// collect runs the DB-cache lookup and provider collection for a single key
// without holding oc.mu, so collections for different keys can run in
// parallel. Callers are responsible for installing the result into the cache
// or errors map and signalling waiters.
func (oc *ObservationContext) collect(ctx context.Context, key happydns.ObservationKey) (json.RawMessage, error) {
	if oc.cacheLookup != nil && oc.freshness > 0 {
		if raw, collectedAt, err := oc.cacheLookup(oc.target, key); err == nil {
			if time.Since(collectedAt) < oc.freshness {
				return raw, nil
			}
		}
	}

	provider := oc.getProvider(key)
	if provider == nil {
		return nil, fmt.Errorf("no observation provider registered for key %q", key)
	}

	val, err := provider.Collect(ctx, oc.opts)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(val)
	if err != nil {
		return nil, fmt.Errorf("observation %q: marshal failed: %w", key, err)
	}
	return json.RawMessage(raw), nil
}

// Data returns all cached observation data as pre-serialized JSON.
func (oc *ObservationContext) Data() map[happydns.ObservationKey]json.RawMessage {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	data := make(map[happydns.ObservationKey]json.RawMessage, len(oc.cache))
	for k, v := range oc.cache {
		data[k] = v
	}
	return data
}

// Provider registration is startup-only (see comments on the registries in
// internal/service/registry.go and internal/provider/registry.go), so the
// "any provider implements X reporter" question has a fixed answer for the
// process lifetime. We compute it once on first call and cache it.
var (
	htmlReporterOnce    sync.Once
	htmlReporterCached  bool
	metricsReporterOnce sync.Once
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

// GetHTMLReport renders an HTML report for the given observation key and raw JSON data.
// Returns (html, true, nil) if the provider supports HTML reports, or ("", false, nil) if not.
func GetHTMLReport(key happydns.ObservationKey, raw json.RawMessage) (string, bool, error) {
	return getHTMLReport(sdk.FindObservationProvider(key), key, raw)
}

// GetHTMLReportCtx is like GetHTMLReport but resolves the provider through
// the ObservationContext, respecting per-context overrides.
func (oc *ObservationContext) GetHTMLReportCtx(key happydns.ObservationKey, raw json.RawMessage) (string, bool, error) {
	return getHTMLReport(oc.getProvider(key), key, raw)
}

func getHTMLReport(provider happydns.ObservationProvider, key happydns.ObservationKey, raw json.RawMessage) (string, bool, error) {
	if provider == nil {
		return "", false, fmt.Errorf("no observation provider registered for key %q", key)
	}

	hr, ok := provider.(happydns.CheckerHTMLReporter)
	if !ok {
		return "", false, nil
	}
	html, err := hr.GetHTMLReport(raw)
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

// GetMetrics extracts metrics for the given observation key and raw JSON data.
// Returns (metrics, true, nil) if the provider supports metrics, or (nil, false, nil) if not.
func GetMetrics(key happydns.ObservationKey, raw json.RawMessage, collectedAt time.Time) ([]happydns.CheckMetric, bool, error) {
	return getMetrics(sdk.FindObservationProvider(key), key, raw, collectedAt)
}

// GetMetricsCtx is like GetMetrics but resolves the provider through
// the ObservationContext, respecting per-context overrides.
func (oc *ObservationContext) GetMetricsCtx(key happydns.ObservationKey, raw json.RawMessage, collectedAt time.Time) ([]happydns.CheckMetric, bool, error) {
	return getMetrics(oc.getProvider(key), key, raw, collectedAt)
}

func getMetrics(provider happydns.ObservationProvider, key happydns.ObservationKey, raw json.RawMessage, collectedAt time.Time) ([]happydns.CheckMetric, bool, error) {
	if provider == nil {
		return nil, false, fmt.Errorf("no observation provider registered for key %q", key)
	}

	mr, ok := provider.(happydns.CheckerMetricsReporter)
	if !ok {
		return nil, false, nil
	}
	metrics, err := mr.ExtractMetrics(raw, collectedAt)
	return metrics, true, err
}

// GetAllMetrics extracts metrics from all observation keys in a snapshot.
func GetAllMetrics(snap *happydns.ObservationSnapshot) ([]happydns.CheckMetric, error) {
	var allMetrics []happydns.CheckMetric
	var errs []error
	for key, raw := range snap.Data {
		metrics, supported, err := GetMetrics(key, raw, snap.CollectedAt)
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
