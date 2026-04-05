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
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// respondWithMetrics writes metrics as a JSON array.
func respondWithMetrics(c *gin.Context, metrics []happydns.CheckMetric) {
	if metrics == nil {
		metrics = []happydns.CheckMetric{}
	}
	c.JSON(http.StatusOK, metrics)
}

const maxLimit = 1000

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
//	@Description	Returns metrics from all recent executions for the authenticated user as a JSON array.
//	@Tags			checkers
//	@Produce		json
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
//	@Description	Returns metrics from recent executions for a domain and all its services as a JSON array.
//	@Tags			checkers
//	@Produce		json
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
//	@Description	Returns metrics from recent executions of a specific checker on a target as a JSON array.
//	@Tags			checkers
//	@Produce		json
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
//	@Description	Returns metrics extracted from a single execution's observation snapshot as a JSON array.
//	@Tags			checkers
//	@Produce		json
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
