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
//	@Description	Counts, for each type of object, the secrets stored in clear, sealed by kind of safe, and those that cannot be opened; and, for each key of the instance keyset, the number of safes it wraps. A key wrapping no safe any more can be removed from the keyset.
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

// Rewrap wraps the key of every safe under the primary instance key.
//
//	@Summary	Rewrap safes
//	@Schemes
//	@Description	Wraps the key of every safe under the primary key of the instance keyset, after a rotation. Sealed values are left untouched. A safe that fails, such as one wrapped by a key already removed from the keyset, is reported and left as it was. Idempotent: run it again after an interruption.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	secret.ResealReport
//	@Failure	500	{object}	happydns.ErrorResponse	"Internal server error"
//	@Router		/secrets/rewrap [post]
func (sc *SecretController) Rewrap(c *gin.Context) {
	report, err := sc.secrets.Rewrap(c.Request.Context())
	apiResponse(c, report, err)
}

// Reseal stores every secret the way the current policy stores new ones.
//
//	@Summary	Reseal secrets
//	@Schemes
//	@Description	Stores the secrets of every object the way the current policy stores new ones: sealed under the instance policy, in clear under the plaintext policy. Objects failing are reported and left as they were; objects changed meanwhile, or owned by a deleted user, are reported as skipped. An object is only written back if nothing changed it since it was read. Idempotent: run it again after an interruption.
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
