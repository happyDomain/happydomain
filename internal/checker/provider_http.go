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

package checker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
)

// httpClient is a shared client with a sensible timeout for remote checker
// endpoints.  The per-request context can shorten this further.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// maxErrorBodySize is the maximum number of bytes read from an error response
// body to include in the error message.
const maxErrorBodySize = 4096

// maxResponseBodySize is the maximum number of bytes read from a successful
// response body.  This prevents a misbehaving endpoint from causing OOM.
const maxResponseBodySize = 10 << 20 // 10 MiB

// HTTPObservationProvider is an ObservationProvider that delegates data
// collection to a remote HTTP endpoint via POST /collect.
type HTTPObservationProvider struct {
	observationKey happydns.ObservationKey
	endpoint       string // base URL without trailing slash

	lastEntries []happydns.DiscoveryEntry // entries from the last Collect response, surfaced via DiscoverEntries
}

// NewHTTPObservationProvider creates a new HTTP-backed observation provider.
// endpoint is the base URL of the remote checker (e.g. "http://checker-ping:8080").
func NewHTTPObservationProvider(key happydns.ObservationKey, endpoint string) *HTTPObservationProvider {
	return &HTTPObservationProvider{
		observationKey: key,
		endpoint:       strings.TrimSuffix(endpoint, "/"),
	}
}

// Key returns the observation key this provider handles.
func (p *HTTPObservationProvider) Key() happydns.ObservationKey {
	return p.observationKey
}

// Collect sends the observation request to the remote endpoint and returns
// the raw JSON data. The returned value is a json.RawMessage which
// ObservationContext.Get() will marshal without double-encoding.
func (p *HTTPObservationProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	reqBody := happydns.ExternalCollectRequest{
		Key:     p.observationKey,
		Options: opts,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("HTTP provider %s: failed to marshal request: %w", p.observationKey, err)
	}

	url := p.endpoint + "/collect"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP provider %s: failed to create request: %w", p.observationKey, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP provider %s: request failed: %w", p.observationKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		return nil, fmt.Errorf("HTTP provider %s: endpoint returned status %d: %s", p.observationKey, resp.StatusCode, string(respBody))
	}

	var result happydns.ExternalCollectResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBodySize)).Decode(&result); err != nil {
		return nil, fmt.Errorf("HTTP provider %s: failed to decode response: %w", p.observationKey, err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("HTTP provider %s: remote error: %s", p.observationKey, result.Error)
	}

	if result.Data == nil {
		return nil, fmt.Errorf("HTTP provider %s: remote returned empty data", p.observationKey)
	}

	p.lastEntries = result.Entries

	// Return json.RawMessage directly - it implements json.Marshaler,
	// so ObservationContext.Get() won't double-encode it.
	return result.Data, nil
}

// DiscoverEntries implements sdk.DiscoveryPublisher: it exposes the entries
// carried in the last /collect response so the engine can ingest them
// through the same path as in-process providers.
func (p *HTTPObservationProvider) DiscoverEntries(_ any) ([]happydns.DiscoveryEntry, error) {
	return p.lastEntries, nil
}

// report posts an ExternalReportRequest to the remote /report endpoint and
// returns the raw response body. The related map is built from the
// ReportContext's Related(key) for the caller-supplied keys so the remote
// reporter can consume cross-checker observations without an extra lookup.
func (p *HTTPObservationProvider) report(ctx context.Context, rc happydns.ReportContext, keys []happydns.ObservationKey) ([]byte, error) {
	related := make(map[happydns.ObservationKey][]happydns.RelatedObservation, len(keys))
	for _, k := range keys {
		if rs := rc.Related(k); len(rs) > 0 {
			related[k] = rs
		}
	}
	reqBody := happydns.ExternalReportRequest{
		Key:     p.observationKey,
		Data:    rc.Data(),
		Related: related,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("HTTP provider %s: marshal report request: %w", p.observationKey, err)
	}

	url := p.endpoint + "/report"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP provider %s: create report request: %w", p.observationKey, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP provider %s: report request failed: %w", p.observationKey, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotImplemented {
		return nil, fmt.Errorf("HTTP provider %s: remote does not support /report", p.observationKey)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		return nil, fmt.Errorf("HTTP provider %s: report returned status %d: %s", p.observationKey, resp.StatusCode, string(respBody))
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
}

// GetHTMLReport implements happydns.CheckerHTMLReporter by forwarding to
// POST /report. Related observations present in rc are forwarded under the
// provider's own observation key — the only key that can meaningfully be
// consumed by the remote reporter.
func (p *HTTPObservationProvider) GetHTMLReport(rc happydns.ReportContext) (string, error) {
	body, err := p.report(context.Background(), rc, []happydns.ObservationKey{p.observationKey})
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ExtractMetrics implements happydns.CheckerMetricsReporter by forwarding to
// POST /report and expecting a JSON array of happydns.CheckMetric.
func (p *HTTPObservationProvider) ExtractMetrics(rc happydns.ReportContext, _ time.Time) ([]happydns.CheckMetric, error) {
	body, err := p.report(context.Background(), rc, []happydns.ObservationKey{p.observationKey})
	if err != nil {
		return nil, err
	}
	var metrics []happydns.CheckMetric
	if err := json.Unmarshal(body, &metrics); err != nil {
		return nil, fmt.Errorf("HTTP provider %s: decode metrics response: %w", p.observationKey, err)
	}
	return metrics, nil
}
