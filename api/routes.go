// Copyright or © or Copr. happyDNS (2021)
//
// contact@happydomain.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"git.happydns.org/happyDomain/config"
	docs "git.happydns.org/happyDomain/docs"
)

//	@title			happyDomain API
//	@version		0.1
//	@description	Finally a simple interface for domain names.

//	@contact.name	happyDomain team
//	@contact.email	contact+api@happydomain.org

//	@license.name	CeCILL Free Software License Agreement
//	@license.url	https://spdx.org/licenses/CECILL-2.1.html

//	@host		localhost:8081
//	@BasePath	/api

//	@securityDefinitions.basic	BasicAuth

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Description for what is this security definition being used

func DeclareRoutes(cfg *config.Options, router *gin.Engine) {
	apiRoutes := router.Group("/api")

	declareAuthenticationRoutes(cfg, apiRoutes)
	declareProviderSpecsRoutes(apiRoutes)
	declareResolverRoutes(apiRoutes)
	declareServiceSpecsRoutes(apiRoutes)
	declareUsersRoutes(cfg, apiRoutes)
	DeclareVersionRoutes(apiRoutes)

	apiAuthRoutes := router.Group("/api")
	apiAuthRoutes.Use(authMiddleware(cfg, false))

	declareDomainsRoutes(cfg, apiAuthRoutes)
	declareProvidersRoutes(cfg, apiAuthRoutes)
	declareProviderSettingsRoutes(cfg, apiAuthRoutes)
	declareUsersAuthRoutes(cfg, apiAuthRoutes)

	// Expose Swagger
	if cfg.ExternalURL.URL.Host != "" {
		tmp := cfg.ExternalURL.URL.String()
		docs.SwaggerInfo.Host = tmp[strings.Index(tmp, "://")+3:]
	} else {
		docs.SwaggerInfo.Host = fmt.Sprintf("localhost%s", cfg.Bind[strings.Index(cfg.Bind, ":"):])
	}
	docs.SwaggerInfo.BasePath = "/api"
	if cfg.BaseURL != "" {
		docs.SwaggerInfo.BasePath = cfg.BaseURL + docs.SwaggerInfo.BasePath
	}
	router.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusFound, "./swagger/index.html") })
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
