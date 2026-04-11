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

package route

import (
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/controller"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
)

// DeclareNotificationRoutes registers notification routes under /api/notifications.
func DeclareNotificationRoutes(
	apiAuthRoutes *gin.RouterGroup,
	dispatcher *notifUC.Dispatcher,
	channelStore notifUC.NotificationChannelStorage,
	prefStore notifUC.NotificationPreferenceStorage,
	recordStore notifUC.NotificationRecordStorage,
) *controller.NotificationController {
	nc := controller.NewNotificationController(dispatcher, channelStore, prefStore, recordStore)

	notif := apiAuthRoutes.Group("/notifications")

	// Channels
	channels := notif.Group("/channels")
	channels.GET("", nc.ListChannels)
	channels.POST("", nc.CreateChannel)

	channelID := channels.Group("/:channelId")
	channelID.GET("", nc.GetChannel)
	channelID.PUT("", nc.UpdateChannel)
	channelID.DELETE("", nc.DeleteChannel)
	channelID.POST("/test", nc.TestChannel)

	// Preferences
	prefs := notif.Group("/preferences")
	prefs.GET("", nc.ListPreferences)
	prefs.POST("", nc.CreatePreference)

	prefID := prefs.Group("/:prefId")
	prefID.GET("", nc.GetPreference)
	prefID.PUT("", nc.UpdatePreference)
	prefID.DELETE("", nc.DeletePreference)

	// History
	notif.GET("/history", nc.ListHistory)

	return nc
}

