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
	"fmt"
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
	c.JSON(http.StatusOK, bc.backup.Backup())
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
	var backup happydns.Backup
	if err := c.ShouldBindJSON(&backup); err != nil {
		log.Printf("%s sends invalid Backup JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	if err := bc.backup.Restore(&backup); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err})
		return
	}

	c.JSON(http.StatusOK, true)
}
