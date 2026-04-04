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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// PlanHandler is a middleware that validates the planId path parameter,
// checks target scope, and sets "plan" in context.
func (cc *CheckerController) PlanHandler(c *gin.Context) {
	planID, err := happydns.NewIdentifierFromString(c.Param("planId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid plan ID"})
		return
	}

	plan, err := cc.planUC.GetCheckPlan(targetFromContext(c), planID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Check plan not found"})
		return
	}

	c.Set("plan", plan)
	c.Next()
}

// ListCheckPlans returns all check plans for a domain.
//
//	@Summary	List check plans for a domain
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{array}	happydns.CheckPlan
//	@Router		/domains/{domain}/checkers/{checkerId}/plans [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/plans [get]
func (cc *CheckerController) ListCheckPlans(c *gin.Context) {
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")

	plans, err := cc.planUC.ListCheckPlansByTargetAndChecker(target, checkerID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, plans)
}

// CreateCheckPlan creates a new check plan.
//
//	@Summary	Create a check plan
//	@Tags		checkers
//	@Accept		json
//	@Produce	json
//	@Param		checkerId	path	string				true	"Checker ID"
//	@Param		plan		body	happydns.CheckPlan	true	"Check plan to create"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	201	{object}	happydns.CheckPlan
//	@Failure	400	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/plans [post]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/plans [post]
func (cc *CheckerController) CreateCheckPlan(c *gin.Context) {
	target := targetFromContext(c)

	var plan happydns.CheckPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	plan.Target = target
	plan.CheckerID = c.Param("checkerId")

	if err := cc.planUC.CreateCheckPlan(&plan); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("cannot create check plan: %w", err))
		return
	}

	c.JSON(http.StatusCreated, plan)
}

// GetCheckPlan returns a specific check plan.
//
//	@Summary	Get a check plan
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		planId		path	string	true	"Plan ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	happydns.CheckPlan
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/plans/{planId} [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/plans/{planId} [get]
func (cc *CheckerController) GetCheckPlan(c *gin.Context) {
	plan := c.MustGet("plan").(*happydns.CheckPlan)
	c.JSON(http.StatusOK, plan)
}

// UpdateCheckPlan updates an existing check plan.
//
//	@Summary	Update a check plan
//	@Tags		checkers
//	@Accept		json
//	@Produce	json
//	@Param		checkerId	path	string				true	"Checker ID"
//	@Param		planId		path	string				true	"Plan ID"
//	@Param		plan		body	happydns.CheckPlan	true	"Updated check plan"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	happydns.CheckPlan
//	@Failure	400	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/plans/{planId} [put]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/plans/{planId} [put]
func (cc *CheckerController) UpdateCheckPlan(c *gin.Context) {
	existing := c.MustGet("plan").(*happydns.CheckPlan)

	var plan happydns.CheckPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	plan.Target = targetFromContext(c)
	plan.CheckerID = c.Param("checkerId")

	updated, err := cc.planUC.UpdateCheckPlan(plan.Target, existing.Id, &plan)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("cannot update check plan: %w", err))
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteCheckPlan deletes a check plan.
//
//	@Summary	Delete a check plan
//	@Tags		checkers
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		planId		path	string	true	"Plan ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	204
//	@Failure	404	{object}	happydns.ErrorResponse
//	@Router		/domains/{domain}/checkers/{checkerId}/plans/{planId} [delete]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/plans/{planId} [delete]
func (cc *CheckerController) DeleteCheckPlan(c *gin.Context) {
	plan := c.MustGet("plan").(*happydns.CheckPlan)

	if err := cc.planUC.DeleteCheckPlan(targetFromContext(c), plan.Id); err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Check plan not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
