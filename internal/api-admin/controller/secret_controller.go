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

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
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
//	@Description	Counts, for each type of object, the secrets stored in clear, sealed by kind of safe, and those that cannot be opened; and, for each key of the instance keyset, the number of safes it wraps. A key wrapping no safe any more can be removed from the keyset. Lists the safe records that do not decode, with their owner when the owner index tells it, and the number of sealed values that open again only if they are repaired.
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

// DropSafes deletes the instance safes once back to clear.
//
//	@Summary	Drop instance safes
//	@Schemes
//	@Description	Deletes the keys of every user and the keyset check record, so that the instance keyset can be removed from the configuration. Only under the plaintext policy, with the keyset still configured, and once no secret is sealed any more (run a reseal first). A safe that cannot be deleted, such as a damaged record, is left as it was, the others are deleted, the check record is kept and the request fails naming it: keep the keyset until it is dealt with. Idempotent. Every happyDomain process sharing the database must run under the plaintext policy.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	secret.ResealReport
//	@Failure	400	{object}	happydns.ErrorResponse	"Secrets still sealed, wrong policy, or safes left"
//	@Failure	500	{object}	happydns.ErrorResponse	"Internal server error"
//	@Router		/secrets/drop-safes [post]
func (sc *SecretController) DropSafes(c *gin.Context) {
	report, err := sc.secrets.DropSafes(c.Request.Context())
	apiResponse(c, report, err)
}

// ForgetSafe gives up a damaged safe.
//
//	@Summary	Forget a damaged safe
//	@Schemes
//	@Description	Deletes a safe record that does not decode, listed under damagedSafes in the status, along with its owner index. The values sealed in it never open again: their users have to enter them again. Refused for a safe that decodes. Repair it from a backup instead when one holds it.
//	@Tags		admin
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Param		safeId	path	string	true	"Safe identifier"
//	@Success	200	{boolean}	bool
//	@Failure	400	{object}	happydns.ErrorResponse	"Invalid identifier, no such safe, or a safe that decodes"
//	@Router		/secrets/safes/{safeId}/forget [post]
func (sc *SecretController) ForgetSafe(c *gin.Context) {
	id, err := happydns.NewIdentifierFromString(c.Param("safeId"))
	if err != nil {
		apiResponse(c, nil, fmt.Errorf("invalid safe identifier: %w", err))
		return
	}
	apiResponse(c, true, sc.secrets.ForgetSafe(c.Request.Context(), id))
}

// RepairSafe puts a damaged safe back from a backup.
//
//	@Summary	Repair a damaged safe
//	@Schemes
//	@Description	Puts the safe of the given identifier, taken from an administrative backup, in place of its record that does not decode, listed under damagedSafes in the status: the values sealed in it open again. Only the safe that was damaged can take its place: it has to open with the instance keyset, its key being bound to its identifier and owner. Refused for a safe that decodes.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Param		safeId	path	string			true	"Safe identifier"
//	@Param		body	body	happydns.Backup	true	"An administrative backup holding the safe"
//	@Success	200	{boolean}	bool
//	@Failure	400	{object}	happydns.ErrorResponse	"Invalid identifier or backup, no such safe in the backup or in the database, a safe that decodes, or one that does not open"
//	@Router		/secrets/safes/{safeId}/repair [post]
func (sc *SecretController) RepairSafe(c *gin.Context) {
	id, err := happydns.NewIdentifierFromString(c.Param("safeId"))
	if err != nil {
		apiResponse(c, nil, fmt.Errorf("invalid safe identifier: %w", err))
		return
	}

	var backup happydns.Backup
	if err := c.ShouldBindJSON(&backup); err != nil {
		apiResponse(c, nil, fmt.Errorf("invalid backup: %w", err))
		return
	}

	apiResponse(c, true, sc.secrets.RepairSafe(c.Request.Context(), id, &backup))
}
