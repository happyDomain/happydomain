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
	"log"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// StatsProvider is the minimal interface required by StorageStatsCollector to
// count business entities. It is implemented by
// internal/storage.StatsProvider, which delegates to the backend's native
// Count* methods so each scrape runs O(prefix scan) rather than O(full decode).
type StatsProvider interface {
	CountUsers() (int, error)
	CountDomains() (int, error)
	CountZones() (int, error)
	CountProviders() (int, error)
}

// StorageStatsCollector is a Prometheus Collector that queries storage at each
// scrape to report accurate business-entity counts.
type StorageStatsCollector struct {
	provider StatsProvider

	usersDesc     *prometheus.Desc
	domainsDesc   *prometheus.Desc
	zonesDesc     *prometheus.Desc
	providersDesc *prometheus.Desc

	// statsErrorsTotal counts failed Count* calls during a Prometheus scrape
	// so silent storage failures remain visible (and alertable) instead of
	// producing gaps in the gauge series.
	statsErrorsTotal *prometheus.CounterVec
}

// NewStorageStatsCollector creates a new collector backed by the given
// StatsProvider and registers it (and its companion error counter) with the
// default Prometheus registry. Re-registration is tolerated, so calling this
// twice — for instance from tests — does not panic.
func NewStorageStatsCollector(p StatsProvider) *StorageStatsCollector {
	c := &StorageStatsCollector{
		provider: p,
		statsErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "happydomain_storage_stats_errors_total",
			Help: "Total number of errors encountered while collecting storage stats for the /metrics endpoint.",
		}, []string{"entity"}),
		usersDesc: prometheus.NewDesc(
			"happydomain_registered_users",
			"Current number of registered user accounts.",
			nil, nil,
		),
		domainsDesc: prometheus.NewDesc(
			"happydomain_domains",
			"Current number of domains managed across all users.",
			nil, nil,
		),
		zonesDesc: prometheus.NewDesc(
			"happydomain_zones",
			"Current number of zone snapshots stored.",
			nil, nil,
		),
		providersDesc: prometheus.NewDesc(
			"happydomain_providers",
			"Current number of provider configurations across all users.",
			nil, nil,
		),
	}

	registerOrLog(c)
	registerOrLog(c.statsErrorsTotal)

	return c
}

// registerOrLog registers a collector with the default registry, tolerating
// "already registered" so test setups and repeated app initialisations are safe.
func registerOrLog(c prometheus.Collector) {
	if err := prometheus.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			return
		}
		log.Printf("metrics: failed to register collector: %v", err)
	}
}

// Describe implements prometheus.Collector.
func (c *StorageStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usersDesc
	ch <- c.domainsDesc
	ch <- c.zonesDesc
	ch <- c.providersDesc
}

// Collect implements prometheus.Collector. It queries storage live so the
// values always reflect the actual database state. Each backend call runs in
// its own goroutine to keep the scrape latency bounded by the slowest count
// rather than their sum.
func (c *StorageStatsCollector) Collect(ch chan<- prometheus.Metric) {
	type job struct {
		entity string
		desc   *prometheus.Desc
		fn     func() (int, error)
	}
	jobs := []job{
		{"user", c.usersDesc, c.provider.CountUsers},
		{"domain", c.domainsDesc, c.provider.CountDomains},
		{"zone", c.zonesDesc, c.provider.CountZones},
		{"provider", c.providersDesc, c.provider.CountProviders},
	}

	type result struct {
		desc *prometheus.Desc
		val  float64
		ok   bool
	}
	results := make([]result, len(jobs))

	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			// A panic inside a backend Count* implementation must not
			// crash the scrape goroutine: convert it into a stats error
			// so the failure is visible via happydomain_storage_stats_errors_total
			// instead of producing an unrecoverable process crash.
			defer func() {
				if r := recover(); r != nil {
					c.statsErrorsTotal.WithLabelValues(j.entity).Inc()
					log.Printf("metrics: panic while collecting %s count: %v", j.entity, r)
				}
			}()
			n, err := j.fn()
			if err != nil {
				c.statsErrorsTotal.WithLabelValues(j.entity).Inc()
				return
			}
			results[i] = result{desc: j.desc, val: float64(n), ok: true}
		}(i, j)
	}
	wg.Wait()

	for _, r := range results {
		if !r.ok {
			continue
		}
		ch <- prometheus.MustNewConstMetric(r.desc, prometheus.GaugeValue, r.val)
	}
}
