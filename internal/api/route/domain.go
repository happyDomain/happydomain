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

	"github.com/gin-gonic/gin"
	ratelimit "github.com/JGLTechnologies/gin-rate-limit"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/netguard"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

func DeclareDomainRoutes(
	router *gin.RouterGroup,
	domainUC happydns.DomainUsecase,
	domainLogUC happydns.DomainLogUsecase,
	remoteZoneImporter happydns.RemoteZoneImporterUsecase,
	zoneImporter happydns.ZoneImporterUsecase,
	zoneUC happydns.ZoneUsecase,
	zoneCorrApplier happydns.ZoneCorrectionApplierUsecase,
	zoneServiceUC happydns.ZoneServiceUsecase,
	serviceUC happydns.ServiceUsecase,
	cc *controller.CheckerController,
	checkStatusUC *checkerUC.CheckStatusUsecase,
	domainInfoUC happydns.DomainInfoUsecase,
	nc *controller.NotificationController,
	outboundGuard *netguard.Guard,
) {
	dc := controller.NewDomainController(
		domainUC,
		remoteZoneImporter,
		zoneImporter,
		checkStatusUC,
	)

	router.GET("/domains", dc.GetDomains)
	router.POST("/domains", dc.AddDomain)

	apiDomainsRoutes := router.Group("/domains/:domain")
	apiDomainsRoutes.Use(middleware.DomainHandler(domainUC, false))

	apiDomainsRoutes.GET("", dc.GetDomain)
	apiDomainsRoutes.PUT("", dc.UpdateDomain)
	apiDomainsRoutes.DELETE("", dc.DelDomain)

	// Rate-limit sharing changes per user: each invite call may reveal whether
	// an arbitrary email is registered and creates a share grant plus a
	// notification, and both endpoints let an owner be probed by repeated
	// share/unshare cycling.
	shareRLStore := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Minute,
		Limit: 5,
	})
	shareRLMiddleware := ratelimit.RateLimiter(shareRLStore, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, happydns.ErrorResponse{
				Message: "Too many share invites. Please try again later.",
			})
		},
		KeyFunc: func(c *gin.Context) string {
			user := middleware.MyUser(c)
			if user == nil {
				return middleware.ClientKey(c)
			}
			return user.Id.String()
		},
	})

	apiDomainsRoutes.GET("/share", dc.GetDomainShares)
	apiDomainsRoutes.POST("/share", shareRLMiddleware, dc.ShareDomain)
	apiDomainsRoutes.DELETE("/share/:userid", shareRLMiddleware, dc.DelDomainShare)

	DeclareDomainInfoRoutes(apiDomainsRoutes.Group("/info"), domainInfoUC)
	DeclareDomainLogRoutes(apiDomainsRoutes, domainLogUC)

	apiDomainsRoutes.POST("/zone", dc.ImportZone)
	apiDomainsRoutes.POST("/retrieve_zone", dc.RetrieveZone)

	certCtrl := controller.NewCertificateController(outboundGuard)
	apiDomainsRoutes.POST("/fetch-certificate", certCtrl.FetchCertificate)

	emailIdCtrl := controller.NewEmailIdentifierController()
	apiDomainsRoutes.POST("/email-identifier", emailIdCtrl.ComputeEmailIdentifier)

	// Mount domain-scoped checker routes.
	if cc != nil {
		DeclareScopedCheckerRoutes(apiDomainsRoutes, cc, nc)
	}

	DeclareZoneRoutes(
		apiDomainsRoutes,
		zoneUC,
		domainUC,
		zoneCorrApplier,
		zoneServiceUC,
		serviceUC,
		cc,
		nc,
	)
}
