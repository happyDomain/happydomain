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
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/usecase"
)

type SecretController struct {
	secrets *usecase.SecretsUsecase
}

func NewSecretController(secrets *usecase.SecretsUsecase) *SecretController {
	return &SecretController{secrets: secrets}
}

// Status tells how the stored secrets are protected.
//
//	@Summary	Secrets status
//	@Schemes
//	@Description	Counts, for each type of object, the secrets stored in clear, sealed by kind of safe, those that cannot be opened, and the objects that could not be looked at; and, for each key of each store, the number of safes it wraps. A key wrapping no safe any more can be removed from its store.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	usecase.SecretsStatus
//	@Failure	500	{object}	happydns.ErrorResponse	"Internal server error"
//	@Router		/secrets/status [get]
func (sc *SecretController) Status(c *gin.Context) {
	status, err := sc.secrets.Status(c.Request.Context())
	apiResponse(c, status, err)
}

// Rewrap wraps the key of every safe under the current key of each store.
//
//	@Summary	Rewrap safes
//	@Schemes
//	@Description	Wraps the key of every safe under the current key of each store, such as the primary key of the instance keyset after a rotation. Sealed values are left untouched. A safe that fails is reported and left as it was. Idempotent: run it again after an interruption.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	secret.ResealReport
//	@Failure	400	{object}	happydns.ErrorResponse	"No store configured rewraps its keys"
//	@Router		/secrets/rewrap [post]
func (sc *SecretController) Rewrap(c *gin.Context) {
	report, err := sc.secrets.Rewrap(c.Request.Context())
	apiResponse(c, report, err)
}

// Reseal stores every secret the way the current policy stores new ones.
//
//	@Summary	Reseal secrets
//	@Schemes
//	@Description	Stores the secrets of every object the way the current policy stores new ones: sealed under the instance policy, in clear under the plaintext policy. Objects failing are reported and left as they were. Idempotent: run it again after an interruption. Run it with a single happyDomain process up.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{array}		secret.ResealReport
//	@Failure	500	{object}	happydns.ErrorResponse	"Internal server error"
//	@Router		/secrets/reseal [post]
func (sc *SecretController) Reseal(c *gin.Context) {
	reports, err := sc.secrets.Reseal(c.Request.Context())
	apiResponse(c, reports, err)
}

// DropSafes deletes the safes once every secret is stored in clear.
//
//	@Summary	Drop safes
//	@Schemes
//	@Description	After going back to the plaintext policy and resealing, deletes the safes and the keyset check record: the keyset is then no longer needed at startup. Refused while a secret is still sealed, or an object could not be looked at.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{integer}	int	"Number of safes deleted"
//	@Failure	400	{object}	happydns.ErrorResponse	"Secrets are still sealed"
//	@Router		/secrets/drop-safes [post]
func (sc *SecretController) DropSafes(c *gin.Context) {
	n, err := sc.secrets.DropSafes(c.Request.Context())
	apiResponse(c, n, err)
}
