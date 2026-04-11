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
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	notifPkg "git.happydns.org/happyDomain/internal/notification"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

// Caps ?limit= so an unbounded request can't OOM the in-memory slice.
const maxHistoryLimit = 500

// Bounds the persisted annotation to prevent state bloat.
const maxAnnotationLength = 1024

// Storage errors may contain keys/internals; never echo them back.
func internalError(c *gin.Context, err error) {
	log.Printf("notification controller: %v", err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, happydns.ErrorResponse{
		Message: "internal server error",
	})
}

type NotificationController struct {
	dispatcher   *notifUC.Dispatcher
	registry     *notifPkg.Registry
	channelStore notifUC.NotificationChannelStorage
	prefStore    notifUC.NotificationPreferenceStorage
	recordStore  notifUC.NotificationRecordStorage
}

func NewNotificationController(
	dispatcher *notifUC.Dispatcher,
	registry *notifPkg.Registry,
	channelStore notifUC.NotificationChannelStorage,
	prefStore notifUC.NotificationPreferenceStorage,
	recordStore notifUC.NotificationRecordStorage,
) *NotificationController {
	return &NotificationController{
		dispatcher:   dispatcher,
		registry:     registry,
		channelStore: channelStore,
		prefStore:    prefStore,
		recordStore:  recordStore,
	}
}

//	@Summary	List supported notification channel types
//	@Tags		notifications
//	@Produce	json
//	@Success	200	{array}	string
//	@Router		/notifications/channel-types [get]
func (nc *NotificationController) ListChannelTypes(c *gin.Context) {
	c.JSON(http.StatusOK, nc.registry.Types())
}

//	@Summary	List notification channels
//	@Tags		notifications
//	@Produce	json
//	@Success	200	{array}		happydns.NotificationChannel
//	@Router		/notifications/channels [get]
func (nc *NotificationController) ListChannels(c *gin.Context) {
	user := middleware.MyUser(c)
	channels, err := nc.channelStore.ListChannelsByUser(user.Id)
	if err != nil {
		internalError(c, err)
		return
	}
	redacted, err := nc.registry.RedactChannels(channels)
	if err != nil {
		internalError(c, err)
		return
	}
	if redacted == nil {
		redacted = []*happydns.NotificationChannel{}
	}
	c.JSON(http.StatusOK, redacted)
}

//	@Summary	Create a notification channel
//	@Tags		notifications
//	@Accept		json
//	@Produce	json
//	@Param		body	body		happydns.NotificationChannel	true	"Channel configuration"
//	@Success	201		{object}	happydns.NotificationChannel
//	@Router		/notifications/channels [post]
func (nc *NotificationController) CreateChannel(c *gin.Context) {
	user := middleware.MyUser(c)

	var ch happydns.NotificationChannel
	if err := c.ShouldBindJSON(&ch); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	ch.UserId = user.Id

	if _, err := nc.registry.DecodeChannelConfig(&ch); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	if err := nc.channelStore.CreateChannel(&ch); err != nil {
		internalError(c, err)
		return
	}

	redacted, err := nc.registry.RedactChannel(&ch)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, redacted)
}

//	@Summary	Get a notification channel
//	@Tags		notifications
//	@Produce	json
//	@Param		channelId	path		string	true	"Channel ID"
//	@Success	200			{object}	happydns.NotificationChannel
//	@Router		/notifications/channels/{channelId} [get]
func (nc *NotificationController) GetChannel(c *gin.Context) {
	redacted, err := nc.registry.RedactChannel(middleware.MyNotificationChannel(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, redacted)
}

// Absent body fields are preserved so omitting one (e.g. "enabled") doesn't silently zero it.
//
//	@Summary	Update a notification channel
//	@Tags		notifications
//	@Accept		json
//	@Produce	json
//	@Param		channelId	path		string							true	"Channel ID"
//	@Param		body		body		happydns.NotificationChannel	true	"Channel configuration"
//	@Success	200			{object}	happydns.NotificationChannel
//	@Router		/notifications/channels/{channelId} [put]
func (nc *NotificationController) UpdateChannel(c *gin.Context) {
	existing := middleware.MyNotificationChannel(c)

	// Bind onto a copy so json.Unmarshal only overwrites present fields; identity fields are forced back below.
	ch := *existing
	if err := c.ShouldBindJSON(&ch); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	ch.Id = existing.Id
	ch.UserId = existing.UserId

	// Carry forward stored secrets so a GET → PUT round-trip does not wipe them.
	merged, err := nc.registry.MergeChannelForUpdate(existing, &ch)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}
	ch.Config = merged

	if _, err := nc.registry.DecodeChannelConfig(&ch); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	if err := nc.channelStore.UpdateChannel(&ch); err != nil {
		internalError(c, err)
		return
	}

	redacted, err := nc.registry.RedactChannel(&ch)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, redacted)
}

//	@Summary	Delete a notification channel
//	@Tags		notifications
//	@Param		channelId	path	string	true	"Channel ID"
//	@Success	204
//	@Router		/notifications/channels/{channelId} [delete]
func (nc *NotificationController) DeleteChannel(c *gin.Context) {
	ch := middleware.MyNotificationChannel(c)

	if err := nc.channelStore.DeleteChannel(ch.Id); err != nil {
		internalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

//	@Summary	Send a test notification
//	@Tags		notifications
//	@Param		channelId	path	string	true	"Channel ID"
//	@Success	200	{object}	map[string]string
//	@Router		/notifications/channels/{channelId}/test [post]
func (nc *NotificationController) TestChannel(c *gin.Context) {
	user := middleware.MyUser(c)
	ch := middleware.MyNotificationChannel(c)

	if err := nc.dispatcher.SendTestNotification(ch, user); err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test notification sent"})
}

// Hours, when set, must be in 0–23. Timezone, when set, must be a valid IANA name.
func validateQuietHours(p *happydns.NotificationPreference) error {
	if p.QuietStart != nil && (*p.QuietStart < 0 || *p.QuietStart > 23) {
		return fmt.Errorf("quietStart must be between 0 and 23")
	}
	if p.QuietEnd != nil && (*p.QuietEnd < 0 || *p.QuietEnd > 23) {
		return fmt.Errorf("quietEnd must be between 0 and 23")
	}
	if p.Timezone != "" {
		if _, err := time.LoadLocation(p.Timezone); err != nil {
			return fmt.Errorf("timezone %q is not a valid IANA name", p.Timezone)
		}
	}
	return nil
}

//	@Summary	List notification preferences
//	@Tags		notifications
//	@Produce	json
//	@Success	200	{array}		happydns.NotificationPreference
//	@Router		/notifications/preferences [get]
func (nc *NotificationController) ListPreferences(c *gin.Context) {
	user := middleware.MyUser(c)
	prefs, err := nc.prefStore.ListPreferencesByUser(user.Id)
	if err != nil {
		internalError(c, err)
		return
	}
	if prefs == nil {
		prefs = []*happydns.NotificationPreference{}
	}
	c.JSON(http.StatusOK, prefs)
}

//	@Summary	Create a notification preference
//	@Tags		notifications
//	@Accept		json
//	@Produce	json
//	@Param		body	body		happydns.NotificationPreference	true	"Preference configuration"
//	@Success	201		{object}	happydns.NotificationPreference
//	@Router		/notifications/preferences [post]
func (nc *NotificationController) CreatePreference(c *gin.Context) {
	user := middleware.MyUser(c)

	var pref happydns.NotificationPreference
	if err := c.ShouldBindJSON(&pref); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	pref.UserId = user.Id

	if err := validateQuietHours(&pref); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	if err := nc.prefStore.CreatePreference(&pref); err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pref)
}

//	@Summary	Get a notification preference
//	@Tags		notifications
//	@Produce	json
//	@Param		prefId	path		string	true	"Preference ID"
//	@Success	200		{object}	happydns.NotificationPreference
//	@Router		/notifications/preferences/{prefId} [get]
func (nc *NotificationController) GetPreference(c *gin.Context) {
	c.JSON(http.StatusOK, middleware.MyNotificationPreference(c))
}

// Absent body fields preserved (see UpdateChannel).
//
//	@Summary	Update a notification preference
//	@Tags		notifications
//	@Accept		json
//	@Produce	json
//	@Param		prefId	path		string								true	"Preference ID"
//	@Param		body	body		happydns.NotificationPreference		true	"Preference configuration"
//	@Success	200		{object}	happydns.NotificationPreference
//	@Router		/notifications/preferences/{prefId} [put]
func (nc *NotificationController) UpdatePreference(c *gin.Context) {
	existing := middleware.MyNotificationPreference(c)

	pref := *existing
	if err := c.ShouldBindJSON(&pref); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	pref.Id = existing.Id
	pref.UserId = existing.UserId

	if err := validateQuietHours(&pref); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	if err := nc.prefStore.UpdatePreference(&pref); err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, pref)
}

//	@Summary	Delete a notification preference
//	@Tags		notifications
//	@Param		prefId	path	string	true	"Preference ID"
//	@Success	204
//	@Router		/notifications/preferences/{prefId} [delete]
func (nc *NotificationController) DeletePreference(c *gin.Context) {
	pref := middleware.MyNotificationPreference(c)

	if err := nc.prefStore.DeletePreference(pref.Id); err != nil {
		internalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

//	@Summary	List notification history
//	@Tags		notifications
//	@Produce	json
//	@Param		limit	query		int		false	"Maximum number of records (capped at 500)"	default(50)
//	@Success	200		{array}		happydns.NotificationRecord
//	@Router		/notifications/history [get]
func (nc *NotificationController) ListHistory(c *gin.Context) {
	user := middleware.MyUser(c)

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}

	records, err := nc.recordStore.ListRecordsByUser(user.Id, limit)
	if err != nil {
		internalError(c, err)
		return
	}
	if records == nil {
		records = []*happydns.NotificationRecord{}
	}
	c.JSON(http.StatusOK, records)
}

//	@Summary	Acknowledge a checker issue
//	@Tags		checkers
//	@Accept		json
//	@Produce	json
//	@Param		domain		path		string							true	"Domain identifier"
//	@Param		checkerId	path		string							true	"Checker ID"
//	@Param		body		body		happydns.AcknowledgeRequest		true	"Acknowledgement"
//	@Success	200			{object}	happydns.NotificationState
//	@Router		/domains/{domain}/checkers/{checkerId}/acknowledge [post]
func (nc *NotificationController) AcknowledgeIssue(c *gin.Context) {
	user := middleware.MyUser(c)
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")

	var req happydns.AcknowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body is optional for acknowledgement.
		req = happydns.AcknowledgeRequest{}
	}
	if len(req.Annotation) > maxAnnotationLength {
		req.Annotation = req.Annotation[:maxAnnotationLength]
	}

	if err := nc.dispatcher.AcknowledgeIssue(user.Id, checkerID, target, user.Email, req.Annotation); err != nil {
		if errors.Is(err, happydns.ErrNotificationStateNotFound) {
			middleware.ErrorResponse(c, http.StatusNotFound, err)
			return
		}
		internalError(c, err)
		return
	}

	state, err := nc.dispatcher.GetState(user.Id, checkerID, target)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, state)
}

//	@Summary	Clear acknowledgement
//	@Tags		checkers
//	@Produce	json
//	@Param		domain		path		string	true	"Domain identifier"
//	@Param		checkerId	path		string	true	"Checker ID"
//	@Success	200			{object}	happydns.NotificationState
//	@Router		/domains/{domain}/checkers/{checkerId}/acknowledge [delete]
func (nc *NotificationController) ClearAcknowledgement(c *gin.Context) {
	user := middleware.MyUser(c)
	target := targetFromContext(c)
	checkerID := c.Param("checkerId")

	if err := nc.dispatcher.ClearAcknowledgement(user.Id, checkerID, target); err != nil {
		if errors.Is(err, happydns.ErrNotificationStateNotFound) {
			middleware.ErrorResponse(c, http.StatusNotFound, err)
			return
		}
		internalError(c, err)
		return
	}

	state, err := nc.dispatcher.GetState(user.Id, checkerID, target)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, state)
}
