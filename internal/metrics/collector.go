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
)

// StatsProvider is the minimal interface required by StorageStatsCollector to
// count business entities. It is implemented by internal/app.storageStatsProvider.
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
}

// NewStorageStatsCollector creates a new collector backed by the given StatsProvider
// and registers it with the default Prometheus registry.
func NewStorageStatsCollector(p StatsProvider) *StorageStatsCollector {
	c := &StorageStatsCollector{
		provider: p,
		usersDesc: prometheus.NewDesc(
			"happydomain_registered_users_total",
			"Current number of registered user accounts.",
			nil, nil,
		),
		domainsDesc: prometheus.NewDesc(
			"happydomain_domains_total",
			"Current number of domains managed across all users.",
			nil, nil,
		),
		zonesDesc: prometheus.NewDesc(
			"happydomain_zones_total",
			"Current number of zone snapshots stored.",
			nil, nil,
		),
		providersDesc: prometheus.NewDesc(
			"happydomain_providers_total",
			"Current number of provider configurations across all users.",
			nil, nil,
		),
	}
	prometheus.MustRegister(c)
	return c
}

// Describe implements prometheus.Collector.
func (c *StorageStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usersDesc
	ch <- c.domainsDesc
	ch <- c.zonesDesc
	ch <- c.providersDesc
}

// Collect implements prometheus.Collector. It queries storage live so the
// values always reflect the actual database state.
func (c *StorageStatsCollector) Collect(ch chan<- prometheus.Metric) {
	if n, err := c.provider.CountUsers(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.usersDesc, prometheus.GaugeValue, float64(n))
	}
	if n, err := c.provider.CountDomains(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.domainsDesc, prometheus.GaugeValue, float64(n))
	}
	if n, err := c.provider.CountZones(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.zonesDesc, prometheus.GaugeValue, float64(n))
	}
	if n, err := c.provider.CountProviders(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.providersDesc, prometheus.GaugeValue, float64(n))
	}
}
