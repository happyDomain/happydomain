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
	SchedulerQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "happydomain_scheduler_queue_depth",
		Help: "Number of items currently in the check scheduler queue.",
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

	// Build info
	BuildInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "happydomain_build_info",
		Help: "Build information about the running happyDomain instance.",
	}, []string{"version"})
)

// SetBuildInfo records the application version in the build info metric.
// Call this once during application startup.
func SetBuildInfo(version string) {
	BuildInfo.WithLabelValues(version).Set(1)
}
