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

// CheckerScheduleController handles test schedule operations
type CheckerScheduleController struct {
	testScheduleUC happydns.CheckerScheduleUsecase
}

func NewCheckerScheduleController(testScheduleUC happydns.CheckerScheduleUsecase) *CheckerScheduleController {
	return &CheckerScheduleController{
		testScheduleUC: testScheduleUC,
	}
}

// ListCheckerSchedules retrieves schedules for the authenticated user
//
//	@Summary		List test schedules
//	@Description	Retrieves test schedules for the authenticated user with optional pagination
//	@Tags			test-schedules
//	@Produce		json
//	@Param			limit	query		int	false	"Maximum number of schedules to return (0 = all)"
//	@Param			offset	query		int	false	"Number of schedules to skip (default: 0)"
//	@Success		200	{array}		happydns.CheckerSchedule
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/plugins/tests/schedules [get]
func (tc *CheckerScheduleController) ListCheckerSchedules(c *gin.Context) {
	user := middleware.MyUser(c)

	schedules, err := tc.testScheduleUC.ListUserSchedules(user.Id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	// Apply pagination
	limit := 0
	offset := 0
	fmt.Sscanf(c.Query("limit"), "%d", &limit)
	fmt.Sscanf(c.Query("offset"), "%d", &offset)

	if offset > len(schedules) {
		offset = len(schedules)
	}
	schedules = schedules[offset:]
	if limit > 0 && len(schedules) > limit {
		schedules = schedules[:limit]
	}

	c.JSON(http.StatusOK, schedules)
}

// CreateCheckerSchedule creates a new test schedule
//
//	@Summary		Create test schedule
//	@Description	Creates a new test schedule for the authenticated user
//	@Tags			test-schedules
//	@Accept			json
//	@Produce		json
//	@Param			body	body		happydns.CheckerSchedule	true	"Check schedule to create"
//	@Success		201		{object}	happydns.CheckerSchedule
//	@Failure		400		{object}	happydns.ErrorResponse
//	@Failure		500		{object}	happydns.ErrorResponse
//	@Router			/plugins/tests/schedules [post]
func (tc *CheckerScheduleController) CreateCheckerSchedule(c *gin.Context) {
	user := middleware.MyUser(c)

	var schedule happydns.CheckerSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	// Set user ID
	schedule.OwnerId = user.Id

	if err := tc.testScheduleUC.CreateSchedule(&schedule); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

// GetCheckerSchedule retrieves a specific schedule
//
//	@Summary		Get test schedule
//	@Description	Retrieves a specific test schedule by ID
//	@Tags			test-schedules
//	@Produce		json
//	@Param			schedule_id	path		string	true	"Schedule ID"
//	@Success		200			{object}	happydns.CheckerSchedule
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/plugins/tests/schedules/{schedule_id} [get]
func (tc *CheckerScheduleController) GetCheckerSchedule(c *gin.Context) {
	user := middleware.MyUser(c)
	scheduleIdStr := c.Param("schedule_id")

	scheduleId, err := happydns.NewIdentifierFromString(scheduleIdStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid schedule ID"))
		return
	}

	// Verify ownership
	if err := tc.testScheduleUC.ValidateScheduleOwnership(scheduleId, user.Id); err != nil {
		middleware.ErrorResponse(c, http.StatusForbidden, err)
		return
	}

	schedule, err := tc.testScheduleUC.GetSchedule(scheduleId)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// UpdateCheckerSchedule updates an existing schedule
//
//	@Summary		Update test schedule
//	@Description	Updates an existing test schedule
//	@Tags			test-schedules
//	@Accept			json
//	@Produce		json
//	@Param			schedule_id	path		string					true	"Schedule ID"
//	@Param			body		body		happydns.CheckerSchedule	true	"Updated schedule"
//	@Success		200			{object}	happydns.CheckerSchedule
//	@Failure		400			{object}	happydns.ErrorResponse
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/plugins/tests/schedules/{schedule_id} [put]
func (tc *CheckerScheduleController) UpdateCheckerSchedule(c *gin.Context) {
	user := middleware.MyUser(c)
	scheduleIdStr := c.Param("schedule_id")

	scheduleId, err := happydns.NewIdentifierFromString(scheduleIdStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid schedule ID"))
		return
	}

	// Verify ownership
	if err := tc.testScheduleUC.ValidateScheduleOwnership(scheduleId, user.Id); err != nil {
		middleware.ErrorResponse(c, http.StatusForbidden, err)
		return
	}

	var schedule happydns.CheckerSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	// Ensure ID matches
	schedule.Id = scheduleId
	schedule.OwnerId = user.Id

	if err := tc.testScheduleUC.UpdateSchedule(&schedule); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// DeleteCheckerSchedule deletes a schedule
//
//	@Summary		Delete test schedule
//	@Description	Deletes a test schedule
//	@Tags			test-schedules
//	@Produce		json
//	@Param			schedule_id	path	string	true	"Schedule ID"
//	@Success		204			"No Content"
//	@Failure		404			{object}	happydns.ErrorResponse
//	@Failure		500			{object}	happydns.ErrorResponse
//	@Router			/plugins/tests/schedules/{schedule_id} [delete]
func (tc *CheckerScheduleController) DeleteCheckerSchedule(c *gin.Context) {
	user := middleware.MyUser(c)
	scheduleIdStr := c.Param("schedule_id")

	scheduleId, err := happydns.NewIdentifierFromString(scheduleIdStr)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid schedule ID"))
		return
	}

	// Verify ownership
	if err := tc.testScheduleUC.ValidateScheduleOwnership(scheduleId, user.Id); err != nil {
		middleware.ErrorResponse(c, http.StatusForbidden, err)
		return
	}

	if err := tc.testScheduleUC.DeleteSchedule(scheduleId); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}
