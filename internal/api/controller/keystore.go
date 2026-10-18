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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/usecase"
)

// KeystoreController lets a user manage the keystores their secrets are
// sealed in.
type KeystoreController struct {
	keystores *usecase.KeystoreUsecase
}

func NewKeystoreController(keystores *usecase.KeystoreUsecase) *KeystoreController {
	return &KeystoreController{keystores: keystores}
}

// KeystoreCreation asks for a personal keystore.
type KeystoreCreation struct {
	// Kind is the kind of keystore, among those GET /keystores/kinds lists.
	Kind string `json:"kind" binding:"required"`

	// Name tells it apart from the other keystores of the user.
	Name string `json:"name" binding:"required"`

	// Params holds what the kind needs, following its form.
	Params json.RawMessage `json:"params,omitempty" swaggertype:"object"`
}

// KeystoreMigration asks to move the secrets of a keystore into another.
type KeystoreMigration struct {
	// To is the reference of the keystore receiving the secrets.
	To string `json:"to" binding:"required"`
}

// KeystoreChoice names a keystore.
type KeystoreChoice struct {
	// Ref is the reference of the keystore; empty, the choice is
	// forgotten.
	Ref string `json:"ref"`
}

// ListKeystores lists the keystores of the user.
//
//	@Summary		List keystores
//	@Description	List the keystores the secrets of the user can be sealed in: those of the instance, then their personal ones, each with the objects whose secrets it holds.
//	@Tags			keystores
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		usecase.KeystoreUsage
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/keystores [get]
func (kc *KeystoreController) ListKeystores(c *gin.Context) {
	list, err := kc.keystores.List(c.Request.Context(), middleware.MyUser(c).Id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

// ListKeystoreKinds lists the kinds of personal keystore the user may create.
//
//	@Summary		List keystore kinds
//	@Description	List the kinds of personal keystore this instance lets users create, each with the form of its parameters.
//	@Tags			keystores
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		secret.KindInfo
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/keystores/kinds [get]
func (kc *KeystoreController) ListKeystoreKinds(c *gin.Context) {
	c.JSON(http.StatusOK, kc.keystores.Kinds(middleware.MyUser(c).Id))
}

// CreateKeystore creates a personal keystore.
//
//	@Summary		Create a keystore
//	@Description	Create a personal keystore of the given kind.
//	@Tags			keystores
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			body	body		KeystoreCreation	true	"Kind, name and parameters"
//	@Success		201		{object}	secret.Keystore
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid keystore"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Router			/keystores [post]
func (kc *KeystoreController) CreateKeystore(c *gin.Context) {
	var req KeystoreCreation
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	ks, err := kc.keystores.Create(c.Request.Context(), middleware.MyUser(c).Id, req.Kind, req.Name, req.Params)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, ks)
}

// DeleteKeystore deletes a personal keystore holding no secret.
//
//	@Summary		Delete a keystore
//	@Description	Delete a personal keystore. One still holding secrets is refused: migrate them first.
//	@Tags			keystores
//	@Security		securitydefinitions.basic
//	@Param			ref	path	string	true	"Keystore reference"
//	@Success		204
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		403	{object}	happydns.ErrorResponse	"Keystore of the instance"
//	@Failure		404	{object}	happydns.ErrorResponse	"Keystore not found"
//	@Failure		409	{object}	happydns.ErrorResponse	"Keystore still holding secrets"
//	@Router			/keystores/{ref} [delete]
func (kc *KeystoreController) DeleteKeystore(c *gin.Context) {
	if err := kc.keystores.Delete(c.Request.Context(), middleware.MyUser(c).Id, c.Param("ref"), false); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MigrateKeystore moves the secrets of a keystore into another.
//
//	@Summary		Migrate a keystore
//	@Description	Move the secrets held in a keystore into another. An object that cannot move, as one used in the background toward a keystore needing its owner, is reported and left where it is.
//	@Tags			keystores
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Param			ref		path		string				true	"Keystore reference"
//	@Param			body	body		KeystoreMigration	true	"Destination"
//	@Success		200		{array}		secret.ResealReport
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid destination"
//	@Failure		401		{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404		{object}	happydns.ErrorResponse	"Keystore not found"
//	@Router			/keystores/{ref}/migrate [post]
func (kc *KeystoreController) MigrateKeystore(c *gin.Context) {
	var req KeystoreMigration
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	reports, err := kc.keystores.Migrate(c.Request.Context(), middleware.MyUser(c).Id, c.Param("ref"), req.To)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, reports)
}

// SetDefaultKeystore chooses the keystore new objects of the user go to.
//
//	@Summary		Choose the default keystore
//	@Description	Choose the keystore the secrets of new objects go to when the request chooses none. An empty reference goes back to the policy of the instance.
//	@Tags			keystores
//	@Accept			json
//	@Security		securitydefinitions.basic
//	@Param			body	body	KeystoreChoice	true	"Keystore reference"
//	@Success		204
//	@Failure		400	{object}	happydns.ErrorResponse	"Invalid keystore"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Keystore not found"
//	@Router			/keystores/default [put]
func (kc *KeystoreController) SetDefaultKeystore(c *gin.Context) {
	var req KeystoreChoice
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	if err := kc.keystores.SetDefault(c.Request.Context(), middleware.MyUser(c).Id, req.Ref); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// keystoreChosen answers a write whose keystore, chosen by the request, does
// not fit.
func keystoreChosen(err error) bool {
	return errors.Is(err, secret.ErrUnknownKeystore) || errors.Is(err, secret.ErrKeystoreNotUnattended)
}
