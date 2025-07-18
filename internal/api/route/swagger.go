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

//go:build swagger

package route

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	docs "git.happydns.org/happyDomain/docs"
	"git.happydns.org/happyDomain/model"
)

func declareRouteSwagger(cfg *happydns.Options, router *gin.RouterGroup) {
	// Expose Swagger
	if cfg.ExternalURL.Host != "" {
		tmp := cfg.ExternalURL.String()
		docs.SwaggerInfo.Host = tmp[strings.Index(tmp, "://")+3:]
	} else {
		docs.SwaggerInfo.Host = fmt.Sprintf("localhost%s", cfg.Bind[strings.Index(cfg.Bind, ":"):])
	}
	docs.SwaggerInfo.BasePath = "/api"
	if cfg.BasePath != "" {
		docs.SwaggerInfo.BasePath = cfg.BasePath + docs.SwaggerInfo.BasePath
	}
	router.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusFound, "./swagger/index.html") })
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
