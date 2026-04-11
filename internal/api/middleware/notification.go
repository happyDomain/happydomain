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

package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

const (
	ctxKeyNotificationChannel    = "notification_channel"
	ctxKeyNotificationPreference = "notification_preference"
)

// Centralizes ownership check so per-channel endpoints cannot forget it.
func NotificationChannelHandler(store notifUC.NotificationChannelStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := MyUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
			return
		}

		channelId, err := happydns.NewIdentifierFromString(c.Param("channelId"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid channel ID"})
			return
		}

		ch, err := store.GetChannel(channelId)
		if err != nil {
			if errors.Is(err, happydns.ErrNotificationChannelNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Channel not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
			return
		}

		if !ch.UserId.Equals(user.Id) {
			// 404 not 403: do not leak the existence of channels owned by others.
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Channel not found"})
			return
		}

		c.Set(ctxKeyNotificationChannel, ch)
		c.Next()
	}
}

// Panics if middleware not installed — wiring bug, not runtime.
func MyNotificationChannel(c *gin.Context) *happydns.NotificationChannel {
	return c.MustGet(ctxKeyNotificationChannel).(*happydns.NotificationChannel)
}

func NotificationPreferenceHandler(store notifUC.NotificationPreferenceStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := MyUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined."})
			return
		}

		prefId, err := happydns.NewIdentifierFromString(c.Param("prefId"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "Invalid preference ID"})
			return
		}

		pref, err := store.GetPreference(prefId)
		if err != nil {
			if errors.Is(err, happydns.ErrNotificationPreferenceNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Preference not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": err.Error()})
			return
		}

		if !pref.UserId.Equals(user.Id) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"errmsg": "Preference not found"})
			return
		}

		c.Set(ctxKeyNotificationPreference, pref)
		c.Next()
	}
}

func MyNotificationPreference(c *gin.Context) *happydns.NotificationPreference {
	return c.MustGet(ctxKeyNotificationPreference).(*happydns.NotificationPreference)
}
