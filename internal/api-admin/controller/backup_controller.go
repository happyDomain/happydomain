// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

type BackupController struct {
	backup happydns.BackupUsecase
}

func NewBackupController(backup happydns.BackupUsecase) *BackupController {
	return &BackupController{backup: backup}
}

// BackupJSON creates a complete backup of the system.
//
//	@Summary		Create backup
//	@Schemes
//	@Description	Create a complete backup of the system including users, authentication, domains, zones, providers, and sessions.
//	@Tags			backup
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/backup.json [post]
func (bc *BackupController) BackupJSON(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	if err := bc.backup.WriteBackup(c.Writer); err != nil {
		log.Printf("%s: error while streaming backup: %s", c.ClientIP(), err.Error())
	}
}

// RestoreJSON restores a complete backup of the system.
//
//	@Summary		Restore backup
//	@Schemes
//	@Description	Restore a complete backup of the system including users, authentication, domains, zones, providers, and sessions.
//	@Tags			backup
//	@Accept			json
//	@Produce		json
//	@Param			body	body		string	true	"Backup data"
//	@Success		200		{boolean}	true
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse	"Restore errors"
//	@Router			/backup.json [put]
func (bc *BackupController) RestoreJSON(c *gin.Context) {
	// Stream the body so a large dump is never decoded into memory all at once
	// (which could exhaust memory on small hosts before the restore even ran).
	if err := bc.backup.RestoreReader(c.Request.Context(), c.Request.Body); err != nil {
		// A malformed document is the client's fault (400); anything else is a
		// failure while applying records (500).
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) || errors.Is(err, io.ErrUnexpectedEOF) {
			log.Printf("%s sends invalid Backup JSON: %s", c.ClientIP(), err.Error())
			c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
			return
		}

		log.Printf("%s restores a backup with errors: %s", c.ClientIP(), err.Error())
		// An error value encodes in JSON as an empty object: answer its
		// text, every error of the restore included.
		c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, true)
}
