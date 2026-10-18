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

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

// KeystoreController lets the administrator manage the keystores of any
// owner, those of the instance included.
type KeystoreController struct {
	keystores *usecase.KeystoreUsecase
}

func NewKeystoreController(keystores *usecase.KeystoreUsecase) *KeystoreController {
	return &KeystoreController{keystores: keystores}
}

// OwnerHandler reads the owner of the path: a user identifier, or "instance"
// for the instance itself, owner of the options of the admin scope.
func (kc *KeystoreController) OwnerHandler(c *gin.Context) {
	param := c.Param("owner")
	owner := secret.InstanceOwner()
	if param != owner.String() && param != "instance" {
		id, err := happydns.NewIdentifierFromString(param)
		if err != nil || id.IsEmpty() {
			middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid owner %q: a user identifier or \"instance\"", param))
			return
		}
		owner = id
	}
	c.Set("keystoreOwner", owner)
	c.Next()
}

func keystoreOwner(c *gin.Context) happydns.Identifier {
	return c.MustGet("keystoreOwner").(happydns.Identifier)
}

// ListKeystores lists the keystores of an owner.
//
//	@Summary	List the keystores of an owner
//	@Schemes
//	@Description	List the keystores of an owner, a user or the instance itself: those of the instance, then their personal ones, each with the objects whose secrets it holds.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Param		owner	path		string	true	"User identifier, or instance"
//	@Success	200		{array}		usecase.KeystoreUsage
//	@Failure	400		{object}	happydns.ErrorResponse	"Invalid owner"
//	@Router		/secrets/owners/{owner}/keystores [get]
func (kc *KeystoreController) ListKeystores(c *gin.Context) {
	list, err := kc.keystores.List(c.Request.Context(), keystoreOwner(c))
	apiResponse(c, list, err)
}

// CreateKeystore creates a personal keystore for an owner.
//
//	@Summary	Create a keystore for an owner
//	@Schemes
//	@Description	Create a personal keystore of the given kind for an owner.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Param		owner	path		string						true	"User identifier, or instance"
//	@Param		body	body		controller.KeystoreCreation	true	"Kind, name and parameters"
//	@Success	201		{object}	secret.Keystore
//	@Failure	400		{object}	happydns.ErrorResponse	"Invalid keystore"
//	@Router		/secrets/owners/{owner}/keystores [post]
func (kc *KeystoreController) CreateKeystore(c *gin.Context) {
	var req controller.KeystoreCreation
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	ks, err := kc.keystores.Create(c.Request.Context(), keystoreOwner(c), req.Kind, req.Name, req.Params)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, ks)
}

// DeleteKeystore deletes a keystore of an owner holding no secret.
//
//	@Summary	Delete a keystore of an owner
//	@Schemes
//	@Description	Delete a keystore of an owner, one of the instance included. One still holding secrets is refused: migrate them first.
//	@Tags		admin
//	@Security	securitydefinitions.basic
//	@Param		owner	path	string	true	"User identifier, or instance"
//	@Param		ref		path	string	true	"Keystore reference"
//	@Success	204
//	@Failure	404	{object}	happydns.ErrorResponse	"Keystore not found"
//	@Failure	409	{object}	happydns.ErrorResponse	"Keystore still holding secrets"
//	@Router		/secrets/owners/{owner}/keystores/{ref} [delete]
func (kc *KeystoreController) DeleteKeystore(c *gin.Context) {
	if err := kc.keystores.Delete(c.Request.Context(), keystoreOwner(c), c.Param("ref"), true); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MigrateKeystore moves the secrets of a keystore of an owner into another.
//
//	@Summary	Migrate a keystore of an owner
//	@Schemes
//	@Description	Move the secrets an owner keeps in a keystore into another of theirs. An object that cannot move, as one used in the background toward a keystore needing its owner, is reported and left where it is.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Param		owner	path		string							true	"User identifier, or instance"
//	@Param		ref		path		string							true	"Keystore reference"
//	@Param		body	body		controller.KeystoreMigration	true	"Destination"
//	@Success	200		{array}		secret.ResealReport
//	@Failure	400		{object}	happydns.ErrorResponse	"Invalid destination"
//	@Failure	404		{object}	happydns.ErrorResponse	"Keystore not found"
//	@Router		/secrets/owners/{owner}/keystores/{ref}/migrate [post]
func (kc *KeystoreController) MigrateKeystore(c *gin.Context) {
	var req controller.KeystoreMigration
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	reports, err := kc.keystores.Migrate(c.Request.Context(), keystoreOwner(c), c.Param("ref"), req.To)
	apiResponse(c, reports, err)
}

// SetDefaultKeystore chooses the keystore new objects of an owner go to.
//
//	@Summary	Choose the default keystore of an owner
//	@Schemes
//	@Description	Choose the keystore the secrets of new objects of an owner go to when the request chooses none. An empty reference goes back to the policy of the instance.
//	@Tags		admin
//	@Accept		json
//	@Security	securitydefinitions.basic
//	@Param		owner	path	string						true	"User identifier, or instance"
//	@Param		body	body	controller.KeystoreChoice	true	"Keystore reference"
//	@Success	204
//	@Failure	400	{object}	happydns.ErrorResponse	"Invalid keystore"
//	@Failure	404	{object}	happydns.ErrorResponse	"Keystore not found"
//	@Router		/secrets/owners/{owner}/keystores/default [put]
func (kc *KeystoreController) SetDefaultKeystore(c *gin.Context) {
	var req controller.KeystoreChoice
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	if err := kc.keystores.SetDefault(c.Request.Context(), keystoreOwner(c), req.Ref); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListKeystoreKinds lists the kinds of personal keystore owners may create.
//
//	@Summary	List keystore kinds
//	@Schemes
//	@Description	List the kinds of personal keystore this instance lets owners create, each with the form of its parameters.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{array}	secret.KindInfo
//	@Router		/secrets/keystore-kinds [get]
func (kc *KeystoreController) ListKeystoreKinds(c *gin.Context) {
	c.JSON(http.StatusOK, kc.keystores.Kinds())
}
