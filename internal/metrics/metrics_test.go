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

package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- HTTPMiddleware -------------------------------------------------------

func TestHTTPMiddleware_RecordsRouteTemplateNotRawPath(t *testing.T) {
	// Reset to keep assertions independent from any other test in the package.
	HTTPRequestsTotal.Reset()
	HTTPRequestDuration.Reset()

	r := gin.New()
	r.Use(HTTPMiddleware())
	r.GET("/api/domains/:domain", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/domains/example.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Route template — not the raw URL — must be used as the path label,
	// otherwise cardinality explodes with one series per domain name.
	if got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/domains/:domain", "200")); got != 1 {
		t.Fatalf("expected 1 request recorded for route template, got %v", got)
	}
	if got := testutil.CollectAndCount(HTTPRequestsTotal); got != 1 {
		t.Fatalf("expected exactly one series, got %d (cardinality leak?)", got)
	}
}

func TestHTTPMiddleware_UnmatchedRouteUsesUnknownLabel(t *testing.T) {
	HTTPRequestsTotal.Reset()
	HTTPRequestDuration.Reset()

	r := gin.New()
	r.Use(HTTPMiddleware())

	req := httptest.NewRequest(http.MethodGet, "/no/such/route", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "unknown", "404")); got != 1 {
		t.Fatalf("expected 1 request recorded under 'unknown' path, got %v", got)
	}
}

func TestHTTPMiddleware_InFlightBalanced(t *testing.T) {
	HTTPRequestsInFlight.Set(0)

	r := gin.New()
	r.Use(HTTPMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	if got := testutil.ToFloat64(HTTPRequestsInFlight); got != 0 {
		t.Fatalf("in-flight gauge should return to 0 after requests complete, got %v", got)
	}
}

// --- StorageStatsCollector ------------------------------------------------

type fakeStatsProvider struct {
	users, domains, zones, providers int
	usersErr, domainsErr             error
	zonesPanic                       bool
}

func (f *fakeStatsProvider) CountUsers() (int, error)   { return f.users, f.usersErr }
func (f *fakeStatsProvider) CountDomains() (int, error) { return f.domains, f.domainsErr }
func (f *fakeStatsProvider) CountZones() (int, error) {
	if f.zonesPanic {
		panic("boom")
	}
	return f.zones, nil
}
func (f *fakeStatsProvider) CountProviders() (int, error) { return f.providers, nil }

// collectorFor builds a StorageStatsCollector against a private registry so
// that tests can run in parallel without sharing state with the default
// registry or with each other.
func collectorFor(p StatsProvider) *StorageStatsCollector {
	return &StorageStatsCollector{
		provider: p,
		statsErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "happydomain_storage_stats_errors_total",
			Help: "test",
		}, []string{"entity"}),
		usersDesc: prometheus.NewDesc(
			"happydomain_registered_users", "users", nil, nil),
		domainsDesc: prometheus.NewDesc(
			"happydomain_domains", "domains", nil, nil),
		zonesDesc: prometheus.NewDesc(
			"happydomain_zones", "zones", nil, nil),
		providersDesc: prometheus.NewDesc(
			"happydomain_providers", "providers", nil, nil),
	}
}

func TestStorageStatsCollector_HappyPath(t *testing.T) {
	c := collectorFor(&fakeStatsProvider{users: 3, domains: 7, zones: 11, providers: 2})

	if got := testutil.CollectAndCount(c); got != 4 {
		t.Fatalf("expected 4 metrics, got %d", got)
	}
}

func TestStorageStatsCollector_ErrorSkipsMetricAndIncrementsErrorCounter(t *testing.T) {
	c := collectorFor(&fakeStatsProvider{
		users:      3,
		domainsErr: errors.New("db down"),
		zones:      1, providers: 1,
	})

	// 4 jobs, 1 errors out → 3 metrics emitted.
	if got := testutil.CollectAndCount(c); got != 3 {
		t.Fatalf("expected 3 metrics when one count fails, got %d", got)
	}
	if got := testutil.ToFloat64(c.statsErrorsTotal.WithLabelValues("domain")); got != 1 {
		t.Fatalf("expected stats error counter for 'domain' to be 1, got %v", got)
	}
}

func TestStorageStatsCollector_PanicIsRecovered(t *testing.T) {
	c := collectorFor(&fakeStatsProvider{users: 1, domains: 1, providers: 1, zonesPanic: true})

	// Must not crash the test process; panicking job is dropped, others succeed.
	got := testutil.CollectAndCount(c)
	if got != 3 {
		t.Fatalf("expected 3 metrics when zones panics, got %d", got)
	}
	if v := testutil.ToFloat64(c.statsErrorsTotal.WithLabelValues("zone")); v != 1 {
		t.Fatalf("expected zone stats error counter to be 1, got %v", v)
	}
}

// --- Scheduler queue depth gauge -----------------------------------------

func TestRegisterSchedulerQueueDepth(t *testing.T) {
	t.Cleanup(func() { RegisterSchedulerQueueDepth(nil) })

	RegisterSchedulerQueueDepth(func() float64 { return 42 })

	// The gauge func is registered against the default registry by promauto.
	// Gather and look for our specific metric.
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var found bool
	for _, mf := range mfs {
		if mf.GetName() != "happydomain_scheduler_queue_depth" {
			continue
		}
		found = true
		if v := mf.GetMetric()[0].GetGauge().GetValue(); v != 42 {
			t.Fatalf("expected queue depth 42, got %v", v)
		}
	}
	if !found {
		t.Fatal("happydomain_scheduler_queue_depth not registered")
	}

	// nil clears the accessor → gauge falls back to 0.
	RegisterSchedulerQueueDepth(nil)
	mfs, _ = prometheus.DefaultGatherer.Gather()
	for _, mf := range mfs {
		if mf.GetName() != "happydomain_scheduler_queue_depth" {
			continue
		}
		if v := mf.GetMetric()[0].GetGauge().GetValue(); v != 0 {
			t.Fatalf("expected queue depth 0 after clearing accessor, got %v", v)
		}
	}
}

// --- SetBuildInfo --------------------------------------------------------

func TestSetBuildInfo(t *testing.T) {
	BuildInfo.Reset()
	SetBuildInfo("1.2.3-test", "abcdef0", "2026-04-08T00:00:00Z", true)

	if got := testutil.ToFloat64(BuildInfo.WithLabelValues("1.2.3-test", "abcdef0", "true", "2026-04-08T00:00:00Z")); got != 1 {
		t.Fatalf("expected build_info{...}=1, got %v", got)
	}
}

// --- /metrics endpoint exposition format ---------------------------------

// TestMetricsEndpointParses guards against the whole exposition pipeline
// emitting something that an actual Prometheus scraper would reject.
func TestMetricsEndpointParses(t *testing.T) {
	// Drive at least one observation through every metric family touched by
	// instrumentation so the endpoint isn't trivially empty.
	HTTPRequestsTotal.WithLabelValues("GET", "/x", "200").Inc()
	StorageOperationsTotal.WithLabelValues("get", "user", "success").Inc()
	SchedulerChecksTotal.WithLabelValues("dns", "success").Inc()
	ProviderAPICallsTotal.WithLabelValues("dummy", "list", "success").Inc()
	SetBuildInfo("test", "deadbee", "2026-04-08T00:00:00Z", false)

	srv := httptest.NewServer(promhttp.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	parser := expfmt.NewTextParser(model.LegacyValidation)
	mfs, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Fatalf("invalid prometheus exposition format: %v", err)
	}

	// Sanity-check a few of the metrics we expect to find.
	for _, name := range []string{
		"happydomain_http_requests_total",
		"happydomain_storage_operations_total",
		"happydomain_scheduler_checks_total",
		"happydomain_provider_api_calls_total",
		"happydomain_build_info",
	} {
		if _, ok := mfs[name]; !ok {
			t.Errorf("expected metric %q in /metrics output", name)
		}
	}
}
