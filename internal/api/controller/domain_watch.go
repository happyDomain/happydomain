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

// DomainWatchController handles the domains a user watches for expiration
// and availability.
type DomainWatchController struct {
	domainWatchService happydns.DomainWatchUsecase
}

func NewDomainWatchController(domainWatchService happydns.DomainWatchUsecase) *DomainWatchController {
	return &DomainWatchController{
		domainWatchService: domainWatchService,
	}
}

// watchId reads the identifier of the watch from the path; it answers the
// request itself when it is malformed.
func watchId(c *gin.Context) (happydns.Identifier, bool) {
	id, err := happydns.NewIdentifierFromString(c.Param("watchId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: "Invalid watch identifier."})
		return nil, false
	}
	return id, true
}

// ListDomainWatches lists the domains the user watches.
//
//	@Summary	List watched domains
//	@Schemes
//	@Description	List the domains the user watches for expiration and availability.
//	@Tags			watchlist
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		happydns.DomainWatch
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/watchlist [get]
func (wc *DomainWatchController) ListDomainWatches(c *gin.Context) {
	watches, err := wc.domainWatchService.ListDomainWatches(middleware.MyUser(c))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, watches)
}

// AddDomainWatch starts watching a domain.
//
//	@Summary	Watch a domain
//	@Schemes
//	@Description	Start watching a domain for expiration and availability. A subdomain or a URL is reduced to the registrable domain. The domain is looked up at once.
//	@Tags			watchlist
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			body	body		happydns.DomainWatchInput	true	"Domain to watch"
//	@Success		201		{object}	happydns.DomainWatch
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid domain name"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403		{object}	happydns.ErrorResponse	"Too many watched domains"
//	@Failure		409		{object}	happydns.ErrorResponse	"Domain already watched"
//	@Failure		429		{object}	happydns.ErrorResponse	"Too many requests"
//	@Router			/watchlist [post]
func (wc *DomainWatchController) AddDomainWatch(c *gin.Context) {
	var input happydns.DomainWatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	w, err := wc.domainWatchService.CreateDomainWatch(c.Request.Context(), middleware.MyUser(c), input)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, w)
}

// GetDomainWatch shows a watched domain.
//
//	@Summary	Get a watched domain
//	@Schemes
//	@Tags			watchlist
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			watchId	path		string	true	"Watch identifier"
//	@Success		200		{object}	happydns.DomainWatch
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid watch identifier"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404		{object}	happydns.ErrorResponse	"Watch not found"
//	@Router			/watchlist/{watchId} [get]
func (wc *DomainWatchController) GetDomainWatch(c *gin.Context) {
	id, ok := watchId(c)
	if !ok {
		return
	}

	w, err := wc.domainWatchService.GetDomainWatch(middleware.MyUser(c), id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, w)
}

// UpdateDomainWatch changes the note of a watched domain.
//
//	@Summary	Update a watched domain
//	@Schemes
//	@Description	Change the note of a watched domain; its name cannot change.
//	@Tags			watchlist
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			watchId	path		string						true	"Watch identifier"
//	@Param			body	body		happydns.DomainWatchInput	true	"New note"
//	@Success		200		{object}	happydns.DomainWatch
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid watch identifier or input"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404		{object}	happydns.ErrorResponse	"Watch not found"
//	@Router			/watchlist/{watchId} [put]
func (wc *DomainWatchController) UpdateDomainWatch(c *gin.Context) {
	id, ok := watchId(c)
	if !ok {
		return
	}

	var input happydns.DomainWatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	w, err := wc.domainWatchService.UpdateDomainWatch(middleware.MyUser(c), id, input)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, w)
}

// DeleteDomainWatch stops watching a domain.
//
//	@Summary	Stop watching a domain
//	@Schemes
//	@Tags			watchlist
//	@Security		securitydefinitions.basic
//	@Param			watchId	path	string	true	"Watch identifier"
//	@Success		204
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid watch identifier"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Watch not found"
//	@Router			/watchlist/{watchId} [delete]
func (wc *DomainWatchController) DeleteDomainWatch(c *gin.Context) {
	id, ok := watchId(c)
	if !ok {
		return
	}

	if err := wc.domainWatchService.DeleteDomainWatch(middleware.MyUser(c), id); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CheckDomainWatch looks a watched domain up now.
//
//	@Summary	Check a watched domain now
//	@Schemes
//	@Description	Look the domain up now, rather than at its next scheduled check. Asked again within a minute, it returns the last result.
//	@Tags			watchlist
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			watchId	path		string	true	"Watch identifier"
//	@Success		200		{object}	happydns.DomainWatch
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid watch identifier"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404		{object}	happydns.ErrorResponse	"Watch not found"
//	@Failure		429		{object}	happydns.ErrorResponse	"Too many requests"
//	@Router			/watchlist/{watchId}/check [post]
func (wc *DomainWatchController) CheckDomainWatch(c *gin.Context) {
	id, ok := watchId(c)
	if !ok {
		return
	}

	w, err := wc.domainWatchService.CheckDomainWatch(c.Request.Context(), middleware.MyUser(c), id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, w)
}
