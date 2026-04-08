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

package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// respondWithMetrics writes metrics in the format requested by the Accept header.
// JSON is the default (preserving the previous API contract for clients that
// send Accept: */* or omit the header). Prometheus text exposition is only
// returned when explicitly requested via Accept: text/plain.
func respondWithMetrics(c *gin.Context, metrics []happydns.CheckMetric) {
	if metrics == nil {
		metrics = []happydns.CheckMetric{}
	}

	if wantsPrometheusText(c.GetHeader("Accept")) {
		c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(renderPrometheus(metrics)))
		return
	}

	c.JSON(http.StatusOK, metrics)
}

const maxLimit = 1000

// wantsPrometheusText returns true when the Accept header explicitly asks for
// text/plain (or the Prometheus exposition media type) without also accepting
// JSON. This keeps the JSON API the default for browsers and generic clients
// while letting `curl -H 'Accept: text/plain'` opt into the Prometheus format.
func wantsPrometheusText(accept string) bool {
	if accept == "" {
		return false
	}
	if strings.Contains(accept, "application/json") {
		return false
	}
	return strings.Contains(accept, "text/plain") ||
		strings.Contains(accept, "application/openmetrics-text")
}

// escapePromLabelValue escapes a label value for the Prometheus text exposition
// format. The spec only allows three escape sequences inside label values:
// `\\`, `\"` and `\n`. Using fmt's %q is unsafe because it can emit \xNN or
// \uNNNN sequences that Prometheus rejects.
func escapePromLabelValue(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// renderPrometheus formats metrics as Prometheus text exposition format
// (version 0.0.4). It only emits constructs allowed by that format: HELP/TYPE
// metadata and untyped samples — no OpenMetrics-only directives such as # UNIT.
func renderPrometheus(metrics []happydns.CheckMetric) string {
	type metricMeta struct {
		unit string
	}
	seen := map[string]metricMeta{}
	var names []string

	for _, m := range metrics {
		if _, ok := seen[m.Name]; !ok {
			seen[m.Name] = metricMeta{unit: m.Unit}
			names = append(names, m.Name)
		}
	}
	sort.Strings(names)

	var b strings.Builder
	nameIdx := map[string]int{}
	for i, name := range names {
		nameIdx[name] = i
	}

	// Sort metrics by name order, then by timestamp.
	sorted := make([]happydns.CheckMetric, len(metrics))
	copy(sorted, metrics)
	sort.Slice(sorted, func(i, j int) bool {
		ni, nj := nameIdx[sorted[i].Name], nameIdx[sorted[j].Name]
		if ni != nj {
			return ni < nj
		}
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	currentName := ""
	for _, m := range sorted {
		if m.Name != currentName {
			currentName = m.Name
			meta := seen[m.Name]
			if meta.unit != "" {
				// Surface the unit as a HELP comment so it stays parseable
				// under Prometheus text 0.0.4 (which has no # UNIT directive).
				fmt.Fprintf(&b, "# HELP %s unit: %s\n", m.Name, meta.unit)
			}
			fmt.Fprintf(&b, "# TYPE %s untyped\n", m.Name)
		}

		b.WriteString(m.Name)
		if len(m.Labels) > 0 {
			b.WriteByte('{')
			first := true
			labelKeys := make([]string, 0, len(m.Labels))
			for k := range m.Labels {
				labelKeys = append(labelKeys, k)
			}
			sort.Strings(labelKeys)
			for _, k := range labelKeys {
				if !first {
					b.WriteByte(',')
				}
				fmt.Fprintf(&b, "%s=\"%s\"", k, escapePromLabelValue(m.Labels[k]))
				first = false
			}
			b.WriteByte('}')
		}

		fmt.Fprintf(&b, " %g", m.Value)
		if !m.Timestamp.IsZero() {
			fmt.Fprintf(&b, " %d", m.Timestamp.UnixMilli())
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func getLimitParam(c *gin.Context, defaultLimit int) int {
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > maxLimit {
				return maxLimit
			}
			return parsed
		}
	}
	return defaultLimit
}

// GetUserMetrics returns metrics across all checkers for the authenticated user.
//
//	@Summary		Get all user metrics
//	@Description	Returns metrics from all recent executions for the authenticated user. Format depends on Accept header: application/json for JSON, otherwise Prometheus text.
//	@Tags			checkers
//	@Produce		json,plain
//	@Param			limit	query	int	false	"Maximum number of executions to extract metrics from (default: 100)"
//	@Success		200	{array}	checker.CheckMetric
//	@Router			/checkers/metrics [get]
func (cc *CheckerController) GetUserMetrics(c *gin.Context) {
	target := targetFromContext(c)
	userID := happydns.TargetIdentifier(target.UserId)
	if userID == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "Not authenticated"})
		return
	}

	limit := getLimitParam(c, 100)
	metrics, err := cc.statusUC.GetMetricsByUser(*userID, limit)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	respondWithMetrics(c, metrics)
}

// GetDomainMetrics returns metrics for a domain and its service children.
//
//	@Summary		Get domain metrics
//	@Description	Returns metrics from recent executions for a domain and all its services. Format depends on Accept header: application/json for JSON, otherwise Prometheus text.
//	@Tags			checkers
//	@Produce		json,plain
//	@Param			domain	path	string	true	"Domain identifier"
//	@Param			limit	query	int		false	"Maximum number of executions (default: 100)"
//	@Success		200	{array}	checker.CheckMetric
//	@Router			/domains/{domain}/checkers/metrics [get]
func (cc *CheckerController) GetDomainMetrics(c *gin.Context) {
	target := targetFromContext(c)
	domainID := happydns.TargetIdentifier(target.DomainId)
	if domainID == nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Domain context required"})
		return
	}

	limit := getLimitParam(c, 100)
	metrics, err := cc.statusUC.GetMetricsByDomain(*domainID, limit)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	respondWithMetrics(c, metrics)
}

// GetCheckerMetrics returns metrics for a specific checker on a target.
//
//	@Summary		Get checker metrics
//	@Description	Returns metrics from recent executions of a specific checker on a target. Format depends on Accept header: application/json for JSON, otherwise Prometheus text.
//	@Tags			checkers
//	@Produce		json,plain
//	@Param			checkerId	path	string	true	"Checker ID"
//	@Param			domain		path	string	true	"Domain identifier"
//	@Param			zoneid		path	string	false	"Zone identifier"
//	@Param			subdomain	path	string	false	"Subdomain"
//	@Param			serviceid	path	string	false	"Service identifier"
//	@Param			limit		query	int		false	"Maximum number of executions (default: 100)"
//	@Success		200	{array}	checker.CheckMetric
//	@Router			/domains/{domain}/checkers/{checkerId}/metrics [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/metrics [get]
func (cc *CheckerController) GetCheckerMetrics(c *gin.Context) {
	checkerID := c.Param("checkerId")
	target := targetFromContext(c)

	limit := getLimitParam(c, 100)
	metrics, err := cc.statusUC.GetMetricsByChecker(checkerID, target, limit)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	respondWithMetrics(c, metrics)
}

// GetExecutionMetrics returns metrics for a single execution.
//
//	@Summary		Get execution metrics
//	@Description	Returns metrics extracted from a single execution's observation snapshot. Format depends on Accept header: application/json for JSON, otherwise Prometheus text.
//	@Tags			checkers
//	@Produce		json,plain
//	@Param			checkerId	path	string	true	"Checker ID"
//	@Param			executionId	path	string	true	"Execution ID"
//	@Param			domain		path	string	true	"Domain identifier"
//	@Param			zoneid		path	string	false	"Zone identifier"
//	@Param			subdomain	path	string	false	"Subdomain"
//	@Param			serviceid	path	string	false	"Service identifier"
//	@Success		200	{array}	checker.CheckMetric
//	@Router			/domains/{domain}/checkers/{checkerId}/executions/{executionId}/metrics [get]
//	@Router			/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/executions/{executionId}/metrics [get]
func (cc *CheckerController) GetExecutionMetrics(c *gin.Context) {
	execID, err := happydns.NewIdentifierFromString(c.Param("executionId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid execution ID"})
		return
	}

	target := targetFromContext(c)

	exec, err := cc.statusUC.GetExecution(target, execID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Execution not found"})
		return
	}

	metrics, err := cc.statusUC.GetMetricsByExecution(target, exec.Id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	respondWithMetrics(c, metrics)
}
