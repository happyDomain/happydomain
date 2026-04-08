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
	"strconv"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "happydomain_http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "happydomain_http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	HTTPRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "happydomain_http_requests_in_flight",
		Help: "Current number of HTTP requests being served.",
	})

	// Scheduler metrics
	//
	// schedulerQueueDepthFn is consulted at scrape time by the GaugeFunc
	// registered below. The scheduler installs its accessor via
	// RegisterSchedulerQueueDepth at construction, which avoids sprinkling
	// gauge.Set calls across every queue mutation site.
	schedulerQueueDepthFn atomic.Pointer[func() float64]

	// SchedulerQueueDepth is kept as a package-level var (rather than the
	// blank identifier) so it is discoverable via grep alongside the other
	// metric vars and easy to reference from tests.
	SchedulerQueueDepth = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "happydomain_scheduler_queue_depth",
		Help: "Number of items currently in the check scheduler queue.",
	}, func() float64 {
		if fn := schedulerQueueDepthFn.Load(); fn != nil {
			return (*fn)()
		}
		return 0
	})

	SchedulerActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "happydomain_scheduler_active_workers",
		Help: "Number of check scheduler workers currently executing a check.",
	})

	SchedulerChecksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "happydomain_scheduler_checks_total",
		Help: "Total number of checks executed by the scheduler.",
	}, []string{"checker", "status"})

	SchedulerCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "happydomain_scheduler_check_duration_seconds",
		Help:    "Duration of individual check executions in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"checker"})

	// DNS provider API metrics
	ProviderAPICallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "happydomain_provider_api_calls_total",
		Help: "Total number of DNS provider API calls.",
	}, []string{"provider", "operation", "status"})

	ProviderAPIDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "happydomain_provider_api_duration_seconds",
		Help:    "Duration of DNS provider API calls in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"provider", "operation"})

	// Storage metrics
	StorageOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "happydomain_storage_operations_total",
		Help: "Total number of storage operations.",
	}, []string{"operation", "entity", "status"})

	StorageOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "happydomain_storage_operation_duration_seconds",
		Help:    "Duration of storage operations in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation", "entity"})

	// Build info. Always 1; the metadata is carried in the labels so that
	// dashboards and alerts can group/diff across deployments.
	BuildInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "happydomain_build_info",
		Help: "Build information about the running happyDomain instance. Always 1; metadata is in the labels.",
	}, []string{"version", "revision", "dirty", "build_date"})
)

// SetBuildInfo records the application build metadata in the build info
// metric. Call this once during application startup. buildDate should be
// formatted as RFC3339 (UTC) and may be empty if unknown.
func SetBuildInfo(version, revision, buildDate string, dirty bool) {
	BuildInfo.WithLabelValues(version, revision, strconv.FormatBool(dirty), buildDate).Set(1)
}

// RegisterSchedulerQueueDepth installs the accessor used at scrape time to
// report the current scheduler queue depth. The function is invoked from the
// Prometheus scrape goroutine, so it must be safe to call concurrently with
// queue mutations and must not block for long. Passing nil unregisters the
// accessor (the gauge will then report 0).
func RegisterSchedulerQueueDepth(fn func() float64) {
	if fn == nil {
		schedulerQueueDepthFn.Store(nil)
		return
	}
	schedulerQueueDepthFn.Store(&fn)
}
