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

	"git.happydns.org/happyDomain/model"
)

// AdminSchedulerController handles admin operations on the test scheduler
type AdminSchedulerController struct {
	scheduler happydns.SchedulerUsecase
}

func NewAdminSchedulerController(scheduler happydns.SchedulerUsecase) *AdminSchedulerController {
	return &AdminSchedulerController{scheduler: scheduler}
}

// GetSchedulerStatus returns the current scheduler state
//
//	@Summary		Get scheduler status
//	@Description	Returns the current state of the test scheduler including worker count, queue size, and upcoming schedules
//	@Tags			scheduler
//	@Produce		json
//	@Success		200	{object}	happydns.SchedulerStatus
//	@Router			/scheduler [get]
func (ctrl *AdminSchedulerController) GetSchedulerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, ctrl.scheduler.GetSchedulerStatus())
}

// EnableScheduler enables the test scheduler at runtime
//
//	@Summary		Enable scheduler
//	@Description	Enables the test scheduler at runtime without restarting the server
//	@Tags			scheduler
//	@Success		200	{object}	happydns.SchedulerStatus
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/scheduler/enable [post]
func (ctrl *AdminSchedulerController) EnableScheduler(c *gin.Context) {
	if err := ctrl.scheduler.SetEnabled(true); err != nil {
		c.JSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ctrl.scheduler.GetSchedulerStatus())
}

// DisableScheduler disables the test scheduler at runtime
//
//	@Summary		Disable scheduler
//	@Description	Disables the test scheduler at runtime without restarting the server
//	@Tags			scheduler
//	@Success		200	{object}	happydns.SchedulerStatus
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/scheduler/disable [post]
func (ctrl *AdminSchedulerController) DisableScheduler(c *gin.Context) {
	if err := ctrl.scheduler.SetEnabled(false); err != nil {
		c.JSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, ctrl.scheduler.GetSchedulerStatus())
}

// RescheduleUpcoming randomizes the next run time of all enabled schedules
// within their respective intervals to spread load evenly.
//
//	@Summary		Reschedule upcoming tests
//	@Description	Randomizes the next run time of all enabled schedules within their intervals to spread load
//	@Tags			scheduler
//	@Produce		json
//	@Success		200	{object}	map[string]int
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/scheduler/reschedule-upcoming [post]
func (ctrl *AdminSchedulerController) RescheduleUpcoming(c *gin.Context) {
	n, err := ctrl.scheduler.RescheduleUpcomingChecks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, happydns.ErrorResponse{Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rescheduled": n})
}
