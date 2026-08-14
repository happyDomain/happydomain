// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/model"
)

func DeclareResolverRoutes(router *gin.RouterGroup, resolverUC happydns.ResolverUsecase) {
	rc := controller.NewResolverController(resolverUC)

	router.POST("/resolver", rc.RunResolver)
	router.POST("/resolver/spf-flatten", rc.FlattenSPF)
	router.POST("/resolver/mta-sts-policy", rc.FetchMTASTSPolicy)
	router.POST("/resolver/dmarc-report-auth", rc.CheckDMARCReportAuth)
}

// DeclareResolverAuthRoutes declares the resolver routes that need an account.
//
// The routes above only ever speak DNS or fetch a well-known HTTPS URL, so they
// are open to anyone. Collecting SSH host keys opens a TCP connection to a host
// and a port the caller picks: netguard already refuses everything that is not
// globally routable, but an anonymous caller would still get a port prober out
// of it. Hence an authenticated route, rate limited on top.
func DeclareResolverAuthRoutes(router *gin.RouterGroup, resolverUC happydns.ResolverUsecase) {
	rc := controller.NewResolverController(resolverUC)

	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Minute,
		Limit: 10,
	})
	limiter := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, happydns.ErrorResponse{
				Message: "Too many requests. Please try again later.",
			})
		},
		KeyFunc: middleware.ClientKey,
	})

	router.POST("/resolver/ssh-hostkeys", limiter, rc.FetchSSHHostKeys)
}
