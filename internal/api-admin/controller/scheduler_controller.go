// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

	"git.happydns.org/happyDomain/internal/api/middleware"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
)

// AdminSchedulerController handles admin scheduler API endpoints.
type AdminSchedulerController struct {
	scheduler *checkerUC.Scheduler
}

// NewAdminSchedulerController creates a new AdminSchedulerController.
func NewAdminSchedulerController(scheduler *checkerUC.Scheduler) *AdminSchedulerController {
	return &AdminSchedulerController{scheduler: scheduler}
}

// GetSchedulerStatus returns the current scheduler status.
//
//	@Summary	Get scheduler status
//	@Tags		admin-scheduler
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	checkerUC.SchedulerStatus
//	@Router		/scheduler [get]
func (s *AdminSchedulerController) GetSchedulerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.scheduler.GetStatus())
}

// EnableScheduler starts the scheduler and returns updated status.
//
//	@Summary	Enable the scheduler
//	@Tags		admin-scheduler
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	checkerUC.SchedulerStatus
//	@Failure	500	{object}	object
//	@Router		/scheduler/enable [post]
func (s *AdminSchedulerController) EnableScheduler(c *gin.Context) {
	if err := s.scheduler.SetEnabled(c.Request.Context(), true); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, s.scheduler.GetStatus())
}

// DisableScheduler stops the scheduler and returns updated status.
//
//	@Summary	Disable the scheduler
//	@Tags		admin-scheduler
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	checkerUC.SchedulerStatus
//	@Failure	500	{object}	object
//	@Router		/scheduler/disable [post]
func (s *AdminSchedulerController) DisableScheduler(c *gin.Context) {
	if err := s.scheduler.SetEnabled(c.Request.Context(), false); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, s.scheduler.GetStatus())
}

// RescheduleUpcoming rebuilds the job queue and returns the new count.
//
//	@Summary	Rebuild the scheduler queue
//	@Tags		admin-scheduler
//	@Produce	json
//	@Security	securitydefinitions.basic
//	@Success	200	{object}	map[string]int
//	@Router		/scheduler/reschedule-upcoming [post]
func (s *AdminSchedulerController) RescheduleUpcoming(c *gin.Context) {
	n := s.scheduler.RebuildQueue()
	c.JSON(http.StatusOK, gin.H{"rescheduled": n})
}
