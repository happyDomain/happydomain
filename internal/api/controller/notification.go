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
	"strconv"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

// NotificationController handles notification-related API endpoints.
type NotificationController struct {
	dispatcher   *notifUC.Dispatcher
	channelStore notifUC.NotificationChannelStorage
	prefStore    notifUC.NotificationPreferenceStorage
	recordStore  notifUC.NotificationRecordStorage
}

// NewNotificationController creates a new NotificationController.
func NewNotificationController(
	dispatcher *notifUC.Dispatcher,
	channelStore notifUC.NotificationChannelStorage,
	prefStore notifUC.NotificationPreferenceStorage,
	recordStore notifUC.NotificationRecordStorage,
) *NotificationController {
	return &NotificationController{
		dispatcher:   dispatcher,
		channelStore: channelStore,
		prefStore:    prefStore,
		recordStore:  recordStore,
	}
}

// --- Channel CRUD ---

// ListChannels returns all notification channels for the authenticated user.
//
//	@Summary	List notification channels
//	@Tags		notifications
//	@Produce	json
//	@Success	200	{array}		happydns.NotificationChannel
//	@Router		/notifications/channels [get]
func (nc *NotificationController) ListChannels(c *gin.Context) {
	user := middleware.MyUser(c)
	channels, err := nc.channelStore.ListChannelsByUser(user.Id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}
	if channels == nil {
		channels = []*happydns.NotificationChannel{}
	}
	c.JSON(http.StatusOK, channels)
}

// CreateChannel creates a new notification channel.
//
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
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	ch.UserId = user.Id

	if err := nc.channelStore.CreateChannel(&ch); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ch)
}

// GetChannel returns a specific notification channel.
//
//	@Summary	Get a notification channel
//	@Tags		notifications
//	@Produce	json
//	@Param		channelId	path		string	true	"Channel ID"
//	@Success	200			{object}	happydns.NotificationChannel
//	@Router		/notifications/channels/{channelId} [get]
func (nc *NotificationController) GetChannel(c *gin.Context) {
	user := middleware.MyUser(c)

	channelId, err := happydns.NewIdentifierFromString(c.Param("channelId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid channel ID"})
		return
	}

	ch, err := nc.channelStore.GetChannel(channelId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Channel not found"})
		return
	}

	if !ch.UserId.Equals(user.Id) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, ch)
}

// UpdateChannel updates a notification channel.
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
	user := middleware.MyUser(c)

	channelId, err := happydns.NewIdentifierFromString(c.Param("channelId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid channel ID"})
		return
	}

	existing, err := nc.channelStore.GetChannel(channelId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Channel not found"})
		return
	}

	if !existing.UserId.Equals(user.Id) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Access denied"})
		return
	}

	var ch happydns.NotificationChannel
	if err := c.ShouldBindJSON(&ch); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	ch.Id = channelId
	ch.UserId = user.Id

	if err := nc.channelStore.UpdateChannel(&ch); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ch)
}

// DeleteChannel deletes a notification channel.
//
//	@Summary	Delete a notification channel
//	@Tags		notifications
//	@Param		channelId	path	string	true	"Channel ID"
//	@Success	204
//	@Router		/notifications/channels/{channelId} [delete]
func (nc *NotificationController) DeleteChannel(c *gin.Context) {
	user := middleware.MyUser(c)

	channelId, err := happydns.NewIdentifierFromString(c.Param("channelId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid channel ID"})
		return
	}

	existing, err := nc.channelStore.GetChannel(channelId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Channel not found"})
		return
	}

	if !existing.UserId.Equals(user.Id) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Access denied"})
		return
	}

	if err := nc.channelStore.DeleteChannel(channelId); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestChannel sends a test notification through a channel.
//
//	@Summary	Send a test notification
//	@Tags		notifications
//	@Param		channelId	path	string	true	"Channel ID"
//	@Success	200	{object}	map[string]string
//	@Router		/notifications/channels/{channelId}/test [post]
func (nc *NotificationController) TestChannel(c *gin.Context) {
	user := middleware.MyUser(c)

	channelId, err := happydns.NewIdentifierFromString(c.Param("channelId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid channel ID"})
		return
	}

	ch, err := nc.channelStore.GetChannel(channelId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Channel not found"})
		return
	}

	if !ch.UserId.Equals(user.Id) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Access denied"})
		return
	}

	if err := nc.dispatcher.SendTestNotification(ch, user); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Test notification sent"})
}

// --- Preference CRUD ---

// ListPreferences returns all notification preferences for the authenticated user.
//
//	@Summary	List notification preferences
//	@Tags		notifications
//	@Produce	json
//	@Success	200	{array}		happydns.NotificationPreference
//	@Router		/notifications/preferences [get]
func (nc *NotificationController) ListPreferences(c *gin.Context) {
	user := middleware.MyUser(c)
	prefs, err := nc.prefStore.ListPreferencesByUser(user.Id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}
	if prefs == nil {
		prefs = []*happydns.NotificationPreference{}
	}
	c.JSON(http.StatusOK, prefs)
}

// CreatePreference creates a new notification preference.
//
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
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	pref.UserId = user.Id

	if err := nc.prefStore.CreatePreference(&pref); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pref)
}

// GetPreference returns a specific notification preference.
//
//	@Summary	Get a notification preference
//	@Tags		notifications
//	@Produce	json
//	@Param		prefId	path		string	true	"Preference ID"
//	@Success	200		{object}	happydns.NotificationPreference
//	@Router		/notifications/preferences/{prefId} [get]
func (nc *NotificationController) GetPreference(c *gin.Context) {
	user := middleware.MyUser(c)

	prefId, err := happydns.NewIdentifierFromString(c.Param("prefId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid preference ID"})
		return
	}

	pref, err := nc.prefStore.GetPreference(prefId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Preference not found"})
		return
	}

	if !pref.UserId.Equals(user.Id) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

// UpdatePreference updates a notification preference.
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
	user := middleware.MyUser(c)

	prefId, err := happydns.NewIdentifierFromString(c.Param("prefId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid preference ID"})
		return
	}

	existing, err := nc.prefStore.GetPreference(prefId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Preference not found"})
		return
	}

	if !existing.UserId.Equals(user.Id) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Access denied"})
		return
	}

	var pref happydns.NotificationPreference
	if err := c.ShouldBindJSON(&pref); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": err.Error()})
		return
	}

	pref.Id = prefId
	pref.UserId = user.Id

	if err := nc.prefStore.UpdatePreference(&pref); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pref)
}

// DeletePreference deletes a notification preference.
//
//	@Summary	Delete a notification preference
//	@Tags		notifications
//	@Param		prefId	path	string	true	"Preference ID"
//	@Success	204
//	@Router		/notifications/preferences/{prefId} [delete]
func (nc *NotificationController) DeletePreference(c *gin.Context) {
	user := middleware.MyUser(c)

	prefId, err := happydns.NewIdentifierFromString(c.Param("prefId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid preference ID"})
		return
	}

	existing, err := nc.prefStore.GetPreference(prefId)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Preference not found"})
		return
	}

	if !existing.UserId.Equals(user.Id) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"errmsg": "Access denied"})
		return
	}

	if err := nc.prefStore.DeletePreference(prefId); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// --- History ---

// ListHistory returns recent notification records for the authenticated user.
//
//	@Summary	List notification history
//	@Tags		notifications
//	@Produce	json
//	@Param		limit	query		int		false	"Maximum number of records"	default(50)
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

	records, err := nc.recordStore.ListRecordsByUser(user.Id, limit)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}
	if records == nil {
		records = []*happydns.NotificationRecord{}
	}
	c.JSON(http.StatusOK, records)
}

// --- Acknowledgement ---

// AcknowledgeIssue marks a checker issue as acknowledged.
//
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

	if err := nc.dispatcher.AcknowledgeIssue(user.Id, checkerID, target, user.Email, req.Annotation); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	state, err := nc.dispatcher.GetState(user.Id, checkerID, target)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

// ClearAcknowledgement removes an acknowledgement from a checker issue.
//
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	state, err := nc.dispatcher.GetState(user.Id, checkerID, target)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}
