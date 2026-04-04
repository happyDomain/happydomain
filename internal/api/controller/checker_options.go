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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

// GetCheckerOptions returns layered options for a checker, from least to most specific scope.
// The scope is determined by the route context (user-only at /api/checkers, domain/service at scoped routes).
//
//	@Summary	Get checker options by scope
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{array}	happydns.CheckerOptionsPositional
//	@Router		/checkers/{checkerId}/options [get]
//	@Router		/domains/{domain}/checkers/{checkerId}/options [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/options [get]
func (cc *CheckerController) GetCheckerOptions(c *gin.Context) {
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")
	positionals, err := cc.OptionsUC.GetCheckerOptionsPositional(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if positionals == nil {
		positionals = []*happydns.CheckerOptionsPositional{}
	}

	// Append auto-fill resolved values so the frontend can display them.
	autoFillOpts, err := cc.OptionsUC.GetAutoFillOptions(checkerID, target)
	if err == nil && autoFillOpts != nil {
		positionals = append(positionals, &happydns.CheckerOptionsPositional{
			CheckName: checkerID,
			UserId:    happydns.TargetIdentifier(target.UserId),
			DomainId:  happydns.TargetIdentifier(target.DomainId),
			ServiceId: happydns.TargetIdentifier(target.ServiceId),
			Options:   autoFillOpts,
		})
	}

	c.JSON(http.StatusOK, positionals)
}

// AddCheckerOptions partially merges options at the current scope.
//
//	@Summary	Merge checker options
//	@Tags		checkers
//	@Accept		json
//	@Produce	json
//	@Param		checkerId	path	string					true	"Checker ID"
//	@Param		options		body	checker.CheckerOptions	true	"Options to merge"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	checker.CheckerOptions
//	@Router		/checkers/{checkerId}/options [post]
//	@Router		/domains/{domain}/checkers/{checkerId}/options [post]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/options [post]
func (cc *CheckerController) AddCheckerOptions(c *gin.Context) {
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")
	var opts happydns.CheckerOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}
	merged, err := cc.OptionsUC.MergeCheckerOptions(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), opts)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if err := cc.OptionsUC.ValidateOptions(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), merged, false); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}
	if _, err := cc.OptionsUC.AddCheckerOptions(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), opts); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, merged)
}

// ChangeCheckerOptions fully replaces options at the current scope.
//
//	@Summary	Replace checker options
//	@Tags		checkers
//	@Accept		json
//	@Produce	json
//	@Param		checkerId	path	string					true	"Checker ID"
//	@Param		options		body	checker.CheckerOptions	true	"Options to set"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	checker.CheckerOptions
//	@Router		/checkers/{checkerId}/options [put]
//	@Router		/domains/{domain}/checkers/{checkerId}/options [put]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/options [put]
func (cc *CheckerController) ChangeCheckerOptions(c *gin.Context) {
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")
	var opts happydns.CheckerOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}
	if err := cc.OptionsUC.ValidateOptions(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), opts, false); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}
	if err := cc.OptionsUC.SetCheckerOptions(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), opts); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, opts)
}

// GetCheckerOption returns a single option value at the current scope.
//
//	@Summary	Get a single checker option
//	@Tags		checkers
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		optname		path	string	true	"Option name"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	any
//	@Router		/checkers/{checkerId}/options/{optname} [get]
//	@Router		/domains/{domain}/checkers/{checkerId}/options/{optname} [get]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/options/{optname} [get]
func (cc *CheckerController) GetCheckerOption(c *gin.Context) {
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")
	optname := c.Param("optname")
	val, err := cc.OptionsUC.GetCheckerOption(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), optname)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	if val == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Option not set"})
		return
	}
	c.JSON(http.StatusOK, val)
}

// SetCheckerOption sets a single option value at the current scope.
//
//	@Summary	Set a single checker option
//	@Tags		checkers
//	@Accept		json
//	@Produce	json
//	@Param		checkerId	path	string	true	"Checker ID"
//	@Param		optname		path	string	true	"Option name"
//	@Param		value		body	any		true	"Option value"
//	@Param		domain		path	string	true	"Domain identifier"
//	@Param		zoneid		path	string	true	"Zone identifier"
//	@Param		subdomain	path	string	true	"Subdomain"
//	@Param		serviceid	path	string	true	"Service identifier"
//	@Success	200	{object}	any
//	@Router		/checkers/{checkerId}/options/{optname} [put]
//	@Router		/domains/{domain}/checkers/{checkerId}/options/{optname} [put]
//	@Router		/domains/{domain}/zone/{zoneid}/{subdomain}/services/{serviceid}/checkers/{checkerId}/options/{optname} [put]
func (cc *CheckerController) SetCheckerOption(c *gin.Context) {
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")
	optname := c.Param("optname")
	var value any
	if err := c.ShouldBindJSON(&value); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}
	// Validate the full merged options after inserting the key.
	existing, err := cc.OptionsUC.GetCheckerOptions(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	existing[optname] = value
	if err := cc.OptionsUC.ValidateOptions(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), existing, false); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}
	if err := cc.OptionsUC.SetCheckerOption(checkerID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), optname, value); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
