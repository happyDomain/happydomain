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

	"git.happydns.org/happyDomain/internal/api/middleware"
	happydns "git.happydns.org/happyDomain/model"
)

// perMinuteRateLimiter builds a per-client-IP limiter allowing limit requests
// per minute.
//
// It keys on middleware.ClientKey, which honours -trusted-proxy: these
// endpoints are meant to sit behind a reverse proxy, and without that the
// whole Internet would share the proxy's budget.
func perMinuteRateLimiter(limit uint) gin.HandlerFunc {
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Minute,
		Limit: limit,
	})

	return ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, happydns.ErrorResponse{
				Message: "Too many requests. Please try again later.",
			})
		},
		KeyFunc: middleware.ClientKey,
	})
}
