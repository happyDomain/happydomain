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
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	ratelimit "github.com/JGLTechnologies/gin-rate-limit"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	notifPkg "git.happydns.org/happyDomain/internal/notification"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	happydns "git.happydns.org/happyDomain/model"
)

func DeclareNotificationRoutes(
	apiAuthRoutes *gin.RouterGroup,
	dispatcher *notifUC.Dispatcher,
	registry *notifPkg.Registry,
	channelStore notifUC.NotificationChannelStorage,
	prefStore notifUC.NotificationPreferenceStorage,
	recordStore notifUC.NotificationRecordStorage,
) *controller.NotificationController {
	nc := controller.NewNotificationController(dispatcher, registry, channelStore, prefStore, recordStore)

	notif := apiAuthRoutes.Group("/notifications")

	// Channel types (advertised by the registry).
	notif.GET("/channel-types", nc.ListChannelTypes)

	// Channels
	channels := notif.Group("/channels")
	channels.GET("", nc.ListChannels)
	channels.POST("", nc.CreateChannel)

	channelID := channels.Group("/:channelId", middleware.NotificationChannelHandler(channelStore))
	channelID.GET("", nc.GetChannel)
	channelID.PUT("", nc.UpdateChannel)
	channelID.DELETE("", nc.DeleteChannel)

	// Rate-limit per user: each test triggers an outbound request and channels are user-owned.
	testRLStore := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Minute,
		Limit: 5,
	})
	testRLMiddleware := ratelimit.RateLimiter(testRLStore, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, happydns.ErrorResponse{
				Message: "Too many test notifications. Please try again later.",
			})
		},
		KeyFunc: func(c *gin.Context) string {
			user := middleware.MyUser(c)
			if user == nil {
				return c.ClientIP()
			}
			return user.Id.String()
		},
	})
	channelID.POST("/test", testRLMiddleware, nc.TestChannel)

	// Preferences
	prefs := notif.Group("/preferences")
	prefs.GET("", nc.ListPreferences)
	prefs.POST("", nc.CreatePreference)

	prefID := prefs.Group("/:prefId", middleware.NotificationPreferenceHandler(prefStore))
	prefID.GET("", nc.GetPreference)
	prefID.PUT("", nc.UpdatePreference)
	prefID.DELETE("", nc.DeletePreference)

	// History
	notif.GET("/history", nc.ListHistory)

	return nc
}

