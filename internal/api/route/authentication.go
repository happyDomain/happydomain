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
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/controller"
	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/model"
)

func DeclareAuthenticationRoutes(cfg *config.Options, baserouter, apirouter *gin.RouterGroup, dependancies happydns.UsecaseDependancies) *controller.LoginController {
	lc := controller.NewLoginController(dependancies.AuthenticationUsecase())

	apirouter.POST("/auth", lc.Login)
	apirouter.POST("/auth/logout", lc.Logout)

	if cfg.GetOIDCProviderURL() != "" {
		oidcp := controller.NewOIDCProvider(cfg, dependancies.AuthenticationUsecase())

		authRoutes := baserouter.Group("/auth")

		providerurl, _ := url.Parse(cfg.GetOIDCProviderURL())
		authRoutes.GET("has_oidc", func(c *gin.Context) {
			parts := strings.Split(strings.TrimSuffix(providerurl.Host, "."), ".")
			if len(parts) > 2 {
				c.JSON(http.StatusOK, gin.H{"provider": strings.Join(parts[len(parts)-2:len(parts)], ".")})
			} else {
				c.JSON(http.StatusOK, gin.H{"provider": strings.Join(parts, ".")})
			}
		})

		authRoutes.GET("oidc", oidcp.RedirectOIDC)
		authRoutes.GET("callback", oidcp.CompleteOIDC)
	}

	return lc
}

func DeclareAuthenticationCheckRoutes(apiAuthRoutes *gin.RouterGroup, dependancies happydns.UsecaseDependancies, lc *controller.LoginController) {
	apiAuthRoutes.GET("/auth", lc.GetLoggedUser)
}
