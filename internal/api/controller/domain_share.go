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

// GetDomainShares lists the users a domain is shared with.
//
//	@Summary	List the users a domain is shared with.
//	@Schemes
//	@Description	List the users the given Domain has been shared with. Owner-only.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string	true	"Domain identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		happydns.DomainShareUser
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Domain not found"
//	@Router			/domains/{domainId}/share [get]
func (dc *DomainController) GetDomainShares(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusUnauthorized, fmt.Errorf("User not defined."))
		return
	}
	domain := c.MustGet("domain").(*happydns.Domain)

	shares, err := dc.domainService.ListDomainShares(user, domain.Id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, shares)
}

// ShareDomain shares a domain with another user.
//
//	@Summary	Share a domain with another user.
//	@Schemes
//	@Description	Grant another user (identified by id or email) access to the Domain. Owner-only.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string				true	"Domain identifier"
//	@Param			body		body	happydns.DomainShareInput	true	"The user to invite"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.DomainShareUser
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Domain or user not found"
//	@Router			/domains/{domainId}/share [post]
func (dc *DomainController) ShareDomain(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusUnauthorized, fmt.Errorf("User not defined."))
		return
	}
	domain := c.MustGet("domain").(*happydns.Domain)

	var input happydns.DomainShareInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	grantee, err := dc.domainService.ShareDomain(user, domain.Id, input.User, input.WithProvider)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, &happydns.DomainShareUser{Id: grantee.Id, Email: grantee.Email})
}

// DelDomainShare revokes a user's access to a domain.
//
//	@Summary	Revoke a user's access to a domain.
//	@Schemes
//	@Description	Revoke a user's access to the Domain. The owner may revoke anyone; an invited user may remove only their own access.
//	@Tags			domains
//	@Accept			json
//	@Produce		json
//	@Param			domainId	path	string	true	"Domain identifier"
//	@Param			userId		path	string	true	"Grantee user identifier"
//	@Security		securitydefinitions.basic
//	@Success		204	"Access revoked"
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Domain not found"
//	@Router			/domains/{domainId}/share/{userId} [delete]
func (dc *DomainController) DelDomainShare(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusUnauthorized, fmt.Errorf("User not defined."))
		return
	}
	domain := c.MustGet("domain").(*happydns.Domain)

	granteeID, err := happydns.NewIdentifierFromString(c.Param("userid"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Invalid user identifier: %s", err.Error())})
		return
	}

	if err := dc.domainService.UnshareDomain(user, domain.Id, granteeID); err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.Status(http.StatusNoContent)
}
