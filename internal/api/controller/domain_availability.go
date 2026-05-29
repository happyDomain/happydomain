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
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

type DomainAvailabilityWatchController struct {
	watchService happydns.DomainAvailabilityWatchUsecase
	engine       happydns.CheckerEngine
	statusUC     *checkerUC.CheckStatusUsecase
}

// NewDomainAvailabilityWatchController builds the controller. engine and
// statusUC may be nil when the checker system is disabled; in that case the
// status and on-demand check endpoints are not registered.
func NewDomainAvailabilityWatchController(watchService happydns.DomainAvailabilityWatchUsecase, engine happydns.CheckerEngine, statusUC *checkerUC.CheckStatusUsecase) *DomainAvailabilityWatchController {
	return &DomainAvailabilityWatchController{
		watchService: watchService,
		engine:       engine,
		statusUC:     statusUC,
	}
}

// DomainAvailabilityWatchStatus reports the latest known availability result
// for a watch, derived from the most recent checker execution.
type DomainAvailabilityWatchStatus struct {
	// Available is the last observed availability: true once the domain is free
	// to register, false while still registered. nil when the watch has never
	// been checked or the last check could not determine availability.
	Available *bool `json:"available,omitempty"`

	// Checking is true while an availability check is currently running.
	Checking bool `json:"checking"`

	// LastChecked is when the most recent finished check completed.
	LastChecked *time.Time `json:"last_checked,omitempty"`

	// Error carries the failure message when the last check could not run.
	Error string `json:"error,omitempty"`
}

// ListDomainAvailabilityWatches retrieves all availability watches owned by the user.
//
//	@Summary	Retrieve user's availability watches
//	@Schemes
//	@Description	Retrieve all domain availability watches belonging to the user.
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Security		securitydefinitions.basic
//	@Success		200	{array}		happydns.DomainAvailabilityWatch
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		500	{object}	happydns.ErrorResponse	"Unable to retrieve watches"
//	@Router			/availability [get]
func (wc *DomainAvailabilityWatchController) ListDomainAvailabilityWatches(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	watches, err := wc.watchService.ListUserDomainAvailabilityWatches(user)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, watches)
}

// AddDomainAvailabilityWatch registers a new availability watch.
//
//	@Summary	Add a new availability watch
//	@Schemes
//	@Description	Register a new domain availability watch for the user.
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			body	body	happydns.DomainAvailabilityWatchCreationInput	true	"Watch to add"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.DomainAvailabilityWatch
//	@Failure		400	{object}	happydns.ErrorResponse	"Error in received data"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		500	{object}	happydns.ErrorResponse	"Database writing error"
//	@Router			/availability [post]
func (wc *DomainAvailabilityWatchController) AddDomainAvailabilityWatch(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("No user specified."))
		return
	}

	var input happydns.DomainAvailabilityWatchCreationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": fmt.Sprintf("Unable to decode given watch: %s", err.Error())})
		return
	}

	watch, err := wc.watchService.CreateDomainAvailabilityWatch(c.Request.Context(), user, &input)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, watch)
}

// GetDomainAvailabilityWatch retrieves a single availability watch owned by the user.
//
//	@Summary	Retrieve an availability watch
//	@Schemes
//	@Description	Retrieve a single domain availability watch owned by the user.
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			watchId	path	string	true	"Watch identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	happydns.DomainAvailabilityWatch
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Watch not found"
//	@Router			/availability/{watchId} [get]
func (wc *DomainAvailabilityWatchController) GetDomainAvailabilityWatch(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	id, err := happydns.NewIdentifierFromString(c.Param("watchId"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("Invalid watch identifier: %w", err))
		return
	}

	watch, err := wc.watchService.GetUserDomainAvailabilityWatch(user, id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, watch)
}

// DeleteDomainAvailabilityWatch removes an availability watch owned by the user.
//
//	@Summary	Delete an availability watch
//	@Schemes
//	@Description	Delete a domain availability watch owned by the user.
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			watchId	path	string	true	"Watch identifier"
//	@Security		securitydefinitions.basic
//	@Success		204	"Watch deleted"
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Watch not found"
//	@Failure		500	{object}	happydns.ErrorResponse	"Database writing error"
//	@Router			/availability/{watchId} [delete]
func (wc *DomainAvailabilityWatchController) DeleteDomainAvailabilityWatch(c *gin.Context) {
	user := middleware.MyUser(c)
	if user == nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("User not defined."))
		return
	}

	id, err := happydns.NewIdentifierFromString(c.Param("watchId"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("Invalid watch identifier: %w", err))
		return
	}

	if err := wc.watchService.DeleteDomainAvailabilityWatch(user, id); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// resolveWatchTarget validates the user and watch ownership and returns the
// CheckTarget used to query/trigger the availability checker. The watch id is
// carried in CheckTarget.DomainId, mirroring the scheduler.
func (wc *DomainAvailabilityWatchController) resolveWatchTarget(c *gin.Context) (happydns.CheckTarget, bool) {
	user := middleware.MyUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return happydns.CheckTarget{}, false
	}

	id, err := happydns.NewIdentifierFromString(c.Param("watchId"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("Invalid watch identifier: %w", err))
		return happydns.CheckTarget{}, false
	}

	watch, err := wc.watchService.GetUserDomainAvailabilityWatch(user, id)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err)
		return happydns.CheckTarget{}, false
	}

	target := happydns.CheckTarget{UserId: user.Id.String(), DomainId: watch.Id.String()}
	return target, true
}

// buildWatchStatus derives the latest availability status from the most recent
// execution of the availability checker for the target.
func (wc *DomainAvailabilityWatchController) buildWatchStatus(target happydns.CheckTarget) DomainAvailabilityWatchStatus {
	var status DomainAvailabilityWatchStatus
	if wc.statusUC == nil {
		return status
	}

	execs, err := wc.statusUC.ListExecutionsByChecker(happydns.DomainAvailabilityCheckerID, target, 1)
	if err != nil || len(execs) == 0 {
		return status
	}

	exec := execs[0]
	switch exec.Status {
	case happydns.ExecutionPending, happydns.ExecutionRunning:
		status.Checking = true
	case happydns.ExecutionFailed:
		status.LastChecked = executionEndTime(exec)
		status.Error = exec.Error
	case happydns.ExecutionRateLimited:
		status.LastChecked = executionEndTime(exec)
		status.Error = "check skipped: rate limited"
	case happydns.ExecutionDone:
		status.LastChecked = executionEndTime(exec)
		// The availability rule inverts the convention: Crit means the watched
		// domain became available, OK means it is still registered.
		switch exec.Result.Status {
		case happydns.StatusCrit:
			available := true
			status.Available = &available
		case happydns.StatusOK:
			available := false
			status.Available = &available
		default:
			status.Error = exec.Result.Message
		}
	}

	return status
}

// executionEndTime returns the execution's completion time, falling back to its
// start time when EndedAt was not recorded.
func executionEndTime(exec *happydns.Execution) *time.Time {
	if exec.EndedAt != nil {
		return exec.EndedAt
	}
	started := exec.StartedAt
	return &started
}

// GetWatchStatus returns the latest known availability state for a watch.
//
//	@Summary	Get the latest availability status of a watch
//	@Schemes
//	@Description	Return the most recent availability result for the watched domain.
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			watchId	path	string	true	"Watch identifier"
//	@Security		securitydefinitions.basic
//	@Success		200	{object}	controller.DomainAvailabilityWatchStatus
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Watch not found"
//	@Router			/availability/{watchId}/status [get]
func (wc *DomainAvailabilityWatchController) GetWatchStatus(c *gin.Context) {
	target, ok := wc.resolveWatchTarget(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, wc.buildWatchStatus(target))
}

// TriggerWatchCheck runs an on-demand availability check for a watch.
//
//	@Summary	Trigger an on-demand availability check
//	@Schemes
//	@Description	Run the availability checker immediately for the watched domain. The check runs asynchronously; poll the status endpoint for the result.
//	@Tags			availability
//	@Accept			json
//	@Produce		json
//	@Param			watchId	path	string	true	"Watch identifier"
//	@Security		securitydefinitions.basic
//	@Success		202	{object}	happydns.Execution
//	@Failure		401	{object}	happydns.ErrorResponse	"Authentication failure"
//	@Failure		404	{object}	happydns.ErrorResponse	"Watch not found"
//	@Failure		500	{object}	happydns.ErrorResponse	"Unable to start the check"
//	@Failure		503	{object}	happydns.ErrorResponse	"Checker system unavailable"
//	@Router			/availability/{watchId}/check [post]
func (wc *DomainAvailabilityWatchController) TriggerWatchCheck(c *gin.Context) {
	target, ok := wc.resolveWatchTarget(c)
	if !ok {
		return
	}

	if wc.engine == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"errmsg": "Checker system is not available"})
		return
	}

	exec, err := wc.engine.CreateExecution(happydns.DomainAvailabilityCheckerID, target, nil)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	go func() {
		if _, err := wc.engine.RunExecution(context.WithoutCancel(c.Request.Context()), exec, nil, nil); err != nil {
			log.Printf("async availability RunExecution error for execution %s: %v", exec.Id.String(), err)
		}
	}()

	c.JSON(http.StatusAccepted, exec)
}
