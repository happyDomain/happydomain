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

package web

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/config"
)

var (
	indexTpl       *template.Template
	CustomHeadHTML = ""
	CustomBodyHTML = ""
	HideVoxPeople  = false
	MsgHeaderColor = "danger"
	MsgHeaderText  = ""
)

func init() {
	flag.StringVar(&CustomHeadHTML, "custom-head-html", CustomHeadHTML, "Add custom HTML right before </head>")
	flag.StringVar(&CustomBodyHTML, "custom-body-html", CustomBodyHTML, "Add custom HTML right before </body>")
	flag.BoolVar(&HideVoxPeople, "hide-feedback-button", HideVoxPeople, "Hide the icon on page that permit to give feedback")
	flag.StringVar(&MsgHeaderText, "msg-header-text", MsgHeaderText, "Custom message banner to add at the top of the app")
	flag.StringVar(&MsgHeaderColor, "msg-header-color", MsgHeaderColor, "Background color of the banner added at the top of the app")
}

func DeclareRoutes(cfg *config.Options, router *gin.Engine) {
	if cfg.DisableProviders {
		CustomHeadHTML += `<script type="text/javascript">window.disable_providers = true;</script>`
	}

	if cfg.DisableRegistration {
		CustomHeadHTML += `<script type="text/javascript">window.disable_registration = true;</script>`
	}

	if cfg.DisableEmbeddedLogin {
		CustomHeadHTML += `<script type="text/javascript">window.disable_embedded_login = true;</script>`
	}

	if config.OIDCProviderURL != "" {
		CustomHeadHTML += `<script type="text/javascript">window.oidc_configured = true;</script>`
	}

	if HideVoxPeople {
		CustomHeadHTML += "<style>#voxpeople { display: none !important; }</style>"
	}

	if len(MsgHeaderText) != 0 {
		CustomHeadHTML += fmt.Sprintf(`<script type="text/javascript">window.msg_header = { text: %q, color: %q };</script>`, MsgHeaderText, MsgHeaderColor)
	}

	if cfg.DevProxy != "" {
		router.GET("/.svelte-kit/*_", serveOrReverse("", cfg))
		router.GET("/node_modules/*_", serveOrReverse("", cfg))
		router.GET("/@vite/*_", serveOrReverse("", cfg))
		router.GET("/@id/*_", serveOrReverse("", cfg))
		router.GET("/@fs/*_", serveOrReverse("", cfg))
		router.GET("/src/*_", serveOrReverse("", cfg))
		router.GET("/home/*_", serveOrReverse("", cfg))
	}
	router.GET("/_app/*_", func(c *gin.Context) { c.Writer.Header().Set("Cache-Control", "public, max-age=604800, immutable") }, serveOrReverse("", cfg))

	router.GET("/", serveOrReverse("/", cfg))
	router.GET("/index.html", serveOrReverse("/", cfg))

	// Routes handled by the showcase
	router.GET("/en/*_", serveOrReverse("/", cfg))
	router.GET("/fr/*_", serveOrReverse("/", cfg))

	// Routes for real existings files
	router.GET("/fonts/*path", func(c *gin.Context) { c.Writer.Header().Set("Cache-Control", "public, max-age=604800, immutable") }, serveOrReverse("", cfg))
	router.GET("/img/*path", func(c *gin.Context) { c.Writer.Header().Set("Cache-Control", "public, max-age=604800, immutable") }, serveOrReverse("", cfg))
	router.GET("/favicon.ico", func(c *gin.Context) { c.Writer.Header().Set("Cache-Control", "public, max-age=604800, immutable") }, serveOrReverse("", cfg))
	router.GET("/manifest.json", serveOrReverse("", cfg))
	router.GET("/robots.txt", serveOrReverse("", cfg))
	router.GET("/service-worker.js", serveOrReverse("", cfg))

	// Routes to virtual content
	router.GET("/domains/*_", serveOrReverse("/", cfg))
	router.GET("/email-validation", serveOrReverse("/", cfg))
	router.GET("/forgotten-password", serveOrReverse("/", cfg))
	router.GET("/join", serveOrReverse("/", cfg))
	router.GET("/login", serveOrReverse("/", cfg))
	router.GET("/me", serveOrReverse("/", cfg))
	router.GET("/onboarding/*_", serveOrReverse("/", cfg))
	router.GET("/providers/*_", serveOrReverse("/", cfg))
	router.GET("/services/*_", serveOrReverse("/", cfg))
	router.GET("/tools/*_", serveOrReverse("/", cfg))
	router.GET("/resolver/*_", serveOrReverse("/", cfg))
	router.GET("/zones/*_", serveOrReverse("/", cfg))

	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") || strings.Contains(c.Request.Header.Get("Accept"), "application/json") {
			c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "errmsg": "Page not found"})
		} else {
			serveOrReverse("/", cfg)(c)
		}
	})
}

func serveOrReverse(forced_url string, cfg *config.Options) gin.HandlerFunc {
	if cfg.DevProxy != "" {
		// Forward to the Vue dev proxy
		return func(c *gin.Context) {
			if u, err := url.Parse(cfg.DevProxy); err != nil {
				http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			} else {
				if forced_url != "" {
					u.Path = path.Join(u.Path, forced_url)
				} else {
					u.Path = path.Join(u.Path, c.Request.URL.Path)
				}

				if r, err := http.NewRequest(c.Request.Method, u.String(), c.Request.Body); err != nil {
					http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
				} else if resp, err := http.DefaultClient.Do(r); err != nil {
					http.Error(c.Writer, err.Error(), http.StatusBadGateway)
				} else {
					defer resp.Body.Close()

					if u.Path != "/" || resp.StatusCode != 200 {
						for key := range resp.Header {
							c.Writer.Header().Add(key, resp.Header.Get(key))
						}
						c.Writer.WriteHeader(resp.StatusCode)

						io.Copy(c.Writer, resp.Body)
					} else {
						for key := range resp.Header {
							if strings.ToLower(key) != "content-length" {
								c.Writer.Header().Add(key, resp.Header.Get(key))
							}
						}

						v, _ := ioutil.ReadAll(resp.Body)

						v2 := strings.Replace(strings.Replace(string(v), "</head>", "{{ .Head }}</head>", 1), "</body>", "{{ .Body }}</body>", 1)

						indexTpl = template.Must(template.New("index.html").Parse(v2))

						if err := indexTpl.ExecuteTemplate(c.Writer, "index.html", map[string]string{
							"Body": CustomBodyHTML,
							"Head": CustomHeadHTML,
						}); err != nil {
							log.Println("Unable to return index.html:", err.Error())
						}
					}
				}
			}
		}
	} else if Assets == nil {
		return func(c *gin.Context) {
			c.String(http.StatusNotFound, "404 Page not found - interface not embedded in binary, please compile with -tags web")
		}
	} else if forced_url == "/" {
		// Serve altered index.html
		return func(c *gin.Context) {
			if indexTpl == nil {
				// Create template from file
				f, _ := Assets.Open("index.html")
				v, _ := ioutil.ReadAll(f)

				v2 := strings.Replace(strings.Replace(string(v), "</head>", "{{ .Head }}</head>", 1), "</body>", "{{ .Body }}</body>", 1)

				indexTpl = template.Must(template.New("index.html").Parse(v2))
			}

			// Serve template
			if err := indexTpl.ExecuteTemplate(c.Writer, "index.html", map[string]string{
				"Body": CustomBodyHTML,
				"Head": CustomHeadHTML,
			}); err != nil {
				log.Println("Unable to return index.html:", err.Error())
			}
		}
	} else if forced_url != "" {
		// Serve forced_url
		return func(c *gin.Context) {
			c.FileFromFS(forced_url, Assets)
		}
	} else {
		// Serve requested file
		return func(c *gin.Context) {
			c.FileFromFS(c.Request.URL.Path, Assets)
		}
	}
}
