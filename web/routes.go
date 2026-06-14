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
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

var (
	CustomHeadHTML = ""
	CustomBodyHTML = ""
	HideVoxPeople  = false
	MsgHeaderColor = "danger"
	MsgHeaderText  = ""
)

// staleReloadJS is served (with a 200) in place of a 404 when a client requests
// a content-hashed bundle under /_app/immutable/ that no longer exists. That can
// only happen to a browser running a *stale, cached* HTML page after a deploy:
// the embedded binary ships new bundle hashes and drops the old ones, so the
// page's bootstrapping import() would normally 404 and the SPA would render but
// never finish starting (visible but frozen).
//
// Because the browser evaluates this as the very module it was trying to import,
// returning a tiny self-healing script here lets already-stuck clients recover on
// their own: it forces a one-shot cache-busting reload so the browser fetches the
// current HTML (which points at bundles that actually exist).
//
// Reload-loop guard: the primary guard is storage-independent — if the URL
// already carries our _fresh marker, we already redirected once and the bundle
// is *still* missing, so the deploy is genuinely broken; we stop reloading and
// let the import reject rather than spin. The sessionStorage check is only a
// best-effort secondary guard across separate navigations, and its failure (e.g.
// Safari private mode throwing on setItem) can no longer cause an ungated loop.
//
// Two things make this robust against "kit.start is not a function":
//   1. The reload is issued SYNCHRONOUSLY during evaluation, so navigation is
//      committed before SvelteKit's bootstrap `.then(([kit]) => kit.start(...))`
//      microtask can run against this (non-)module.
//   2. We THROW at the end of evaluation, so the failing `import()` rejects
//      instead of resolving to this module. Promise.all rejects, the bootstrap
//      `.then` is skipped, and kit.start is never reached. On a guarded second
//      load (when no reload is issued) this surfaces as a clean import rejection
//      rather than a confusing TypeError.
//
// Note: happyDomain ships a service worker, but we deliberately do NOT touch the
// Cache Storage API here. The service worker already manages its own cache
// lifecycle (it drops old-version caches on activate), the synchronous reload
// fetches fresh HTML through the worker's network-first path anyway, and doing
// async `caches.*` work would only push the reload into a microtask chain that
// loses the race against kit.start.
const staleReloadJS = `(function () {
  var KEY = '__hd_stale_reload__';
  var now = Date.now();
  // Storage-independent guard: if we already redirected with _fresh and the
  // bundle is still missing, stop here (no reload loop, even without storage).
  try {
    if (new URL(window.location.href).searchParams.has('_fresh')) return;
  } catch (e) {}
  // Best-effort secondary guard across separate navigations; failure is ignored.
  try {
    if (now - parseInt(sessionStorage.getItem(KEY) || '0', 10) < 10000) return;
    sessionStorage.setItem(KEY, String(now));
  } catch (e) {}
  try {
    var u = new URL(window.location.href);
    u.searchParams.set('_fresh', String(now));
    window.location.replace(u.toString());
  } catch (e) {
    try { window.location.reload(); } catch (e2) {}
  }
})();
throw new Error('happydomain: stale bundle, reloading');
`

func init() {
	flag.StringVar(&CustomHeadHTML, "custom-head-html", CustomHeadHTML, "Add custom HTML right before </head>")
	flag.StringVar(&CustomBodyHTML, "custom-body-html", CustomBodyHTML, "Add custom HTML right before </body>")
	flag.BoolVar(&HideVoxPeople, "hide-feedback-button", HideVoxPeople, "Hide the icon on page that permit to give feedback")
	flag.StringVar(&MsgHeaderText, "msg-header-text", MsgHeaderText, "Custom message banner to add at the top of the app")
	flag.StringVar(&MsgHeaderColor, "msg-header-color", MsgHeaderColor, "Background color of the banner added at the top of the app")
}

func DeclareRoutes(cfg *happydns.Options, router *gin.RouterGroup, captchaVerifier happydns.CaptchaVerifier) {
	appConfig := map[string]any{}

	if cfg.DisableCheckerScheduler {
		appConfig["disable_checker_scheduler"] = true
	}

	if cfg.DisableProviders {
		appConfig["disable_providers"] = true
	}

	if cfg.DisableRegistration {
		appConfig["disable_registration"] = true
	}

	if cfg.DisableEmbeddedLogin {
		appConfig["disable_embedded_login"] = true
	}

	if cfg.NoMail {
		appConfig["no_mail"] = true
	}

	if len(cfg.OIDCClients) > 0 {
		appConfig["oidc_configured"] = true
	}

	if len(MsgHeaderText) != 0 {
		appConfig["msg_header"] = map[string]string{
			"text":  MsgHeaderText,
			"color": MsgHeaderColor,
		}
	}

	if HideVoxPeople {
		appConfig["hide_feedback"] = true
	}

	if captchaVerifier.Provider() != "" {
		appConfig["captcha_provider"] = captchaVerifier.Provider()
		appConfig["captcha_site_key"] = captchaVerifier.SiteKey()
	}

	if appcfg, err := json.MarshalIndent(appConfig, "", "  "); err != nil {
		log.Println("Unable to generate JSON config to inject in web application")
	} else {
		CustomHeadHTML += `<script id="app-config" type="application/json">` + string(appcfg) + `</script>`
	}

	serveFile := serveOrReverse("", cfg)
	serveIndex := serveOrReverse("/", cfg)
	serveManifest := serveOrReverse("/manifest.json", cfg)
	immutable := func(c *gin.Context) { c.Writer.Header().Set("Cache-Control", "public, max-age=604800, immutable") }

	if cfg.DevProxy != "" {
		router.GET("/.svelte-kit/*_", serveFile)
		router.GET("/node_modules/*_", serveFile)
		router.GET("/@vite/*_", serveFile)
		router.GET("/@id/*_", serveFile)
		router.GET("/@fs/*_", serveFile)
		router.GET("/src/*_", serveFile)
		router.GET("/home/*_", serveFile)
	}
	router.GET("/_app/*_", immutable, serveFile)

	router.GET("/", serveIndex)
	router.GET("/index.html", serveIndex)

	// Routes handled by the showcase
	router.GET("/en/*_", serveIndex)
	router.GET("/fr/*_", serveIndex)

	// Routes for real existings files
	router.GET("/fonts/*path", immutable, serveFile)
	router.GET("/img/*path", immutable, serveFile)
	router.GET("/favicon.ico", immutable, serveFile)
	router.GET("/manifest.json", serveManifest)
	router.GET("/robots.txt", serveFile)
	router.GET("/service-worker.js", serveFile)

	// Routes to virtual content
	router.GET("/domains/*_", serveIndex)
	router.GET("/email-validation", serveIndex)
	router.GET("/forgotten-password", serveIndex)
	router.GET("/generator/*_", serveIndex)
	router.GET("/login", serveIndex)
	router.GET("/me", serveIndex)
	router.GET("/onboarding/*_", serveIndex)
	router.GET("/providers/*_", serveIndex)
	router.GET("/register", serveIndex)
	router.GET("/services/*_", serveIndex)
	router.GET("/tools/*_", serveIndex)
	router.GET("/resolver/*_", serveIndex)
	router.GET("/zones/*_", serveIndex)
}

func NoRoute(cfg *happydns.Options, router *gin.Engine) {
	serveIndex := serveOrReverse("/", cfg)
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, cfg.BasePath+"/api") || strings.Contains(c.Request.Header.Get("Accept"), "application/json") {
			c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "errmsg": "Page not found"})
		} else if cfg.BasePath != "" && !strings.HasPrefix(c.Request.URL.Path, cfg.BasePath) {
			c.Redirect(http.StatusFound, cfg.BasePath+c.Request.URL.Path)
		} else {
			serveIndex(c)
		}
	})
}

// setNoCache marks a response as revalidate-before-reuse. It is the inverse of
// the immutable middleware and is used for the documents (index.html, manifest,
// and the stale-bundle fallback) that must never pin a browser to dropped
// content-hashed bundles after a deploy. See staleReloadJS.
func setNoCache(c *gin.Context) {
	c.Writer.Header().Set("Cache-Control", "no-cache")
}

func serveOrReverse(forced_url string, cfg *happydns.Options) gin.HandlerFunc {
	if cfg.DevProxy != "" {
		// Parse once at creation time, not per request
		devURL, err := url.Parse(cfg.DevProxy)
		if err != nil {
			return func(c *gin.Context) {
				http.Error(c.Writer, "invalid dev proxy URL: "+err.Error(), http.StatusInternalServerError)
			}
		}

		// Forward to the Vue dev proxy
		return func(c *gin.Context) {
			u := *devURL // copy to avoid mutating shared state across requests
			if forced_url != "" {
				u.Path = path.Join(u.Path, forced_url)
			} else {
				u.Path = path.Join(u.Path, c.Request.URL.Path)
			}
			u.RawQuery = c.Request.URL.RawQuery

			r, err := http.NewRequest(c.Request.Method, u.String(), c.Request.Body)
			if err != nil {
				http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
				return
			}
			resp, err := http.DefaultClient.Do(r)
			if err != nil {
				http.Error(c.Writer, err.Error(), http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()

			if u.Path != "/" || resp.StatusCode != 200 {
				for key, vals := range resp.Header {
					for _, v := range vals {
						c.Writer.Header().Add(key, v)
					}
				}
				c.Writer.WriteHeader(resp.StatusCode)
				io.Copy(c.Writer, resp.Body)
			} else {
				for key, vals := range resp.Header {
					if !strings.EqualFold(key, "content-length") {
						for _, v := range vals {
							c.Writer.Header().Add(key, v)
						}
					}
				}

				v, err := io.ReadAll(resp.Body)
				if err != nil {
					http.Error(c.Writer, err.Error(), http.StatusBadGateway)
					return
				}

				// Local template per request — no race condition on a package-level var
				tpl, err := template.New("index.html").Parse(
					strings.Replace(strings.Replace(string(v),
						"</head>", "{{ .Head }}</head>", 1),
						"</body>", "{{ .Body }}</body>", 1),
				)
				if err != nil {
					http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := tpl.Execute(c.Writer, map[string]string{
					"Body": CustomBodyHTML,
					"Head": CustomHeadHTML,
				}); err != nil {
					log.Println("Unable to return index.html:", err.Error())
				}
			}
		}
	} else if Assets == nil {
		return func(c *gin.Context) {
			c.String(http.StatusNotFound, "404 Page not found - interface not embedded in binary, please compile with -tags web")
		}
	} else if forced_url == "/" {
		// Pre-render index.html once at handler creation time
		f, err := Assets.Open("index.html")
		if err != nil {
			log.Println("Unable to open embedded index.html:", err)
			return func(c *gin.Context) {
				c.String(http.StatusInternalServerError, "index.html not found in embedded assets")
			}
		}
		v, err := io.ReadAll(f)
		if err != nil {
			log.Println("Unable to read embedded index.html:", err)
			return func(c *gin.Context) {
				c.String(http.StatusInternalServerError, "failed to read embedded index.html")
			}
		}

		rendered := []byte(strings.Replace(strings.Replace(string(v), "</head>", CustomHeadHTML+"</head>", 1), "</body>", CustomBodyHTML+"</body>", 1))

		if cfg.BasePath != "" {
			rendered = bytes.ReplaceAll(
				bytes.ReplaceAll(
					bytes.ReplaceAll(
						bytes.ReplaceAll(
							rendered,
							[]byte(`href="/`),
							append([]byte(`href="`), append([]byte(cfg.BasePath), '/')...),
						),
						[]byte(`import("/`),
						append([]byte(`import("`), append([]byte(cfg.BasePath), '/')...),
					),
					[]byte(`base: "`),
					append([]byte(`base: "`), []byte(cfg.BasePath)...),
				),
				[]byte("</head>"),
				[]byte(`<base href="`+cfg.BasePath+`"></head>`),
			)
		}

		return func(c *gin.Context) {
			// Revalidate the HTML so clients always load a page pointing at
			// content-hashed bundles that still exist after a deploy, rather
			// than a cached page referencing dropped hashes. See staleReloadJS.
			setNoCache(c)
			c.Data(http.StatusOK, "text/html; charset=utf-8", rendered)
		}
	} else if forced_url == "/manifest.json" {
		// Serve altered manifest.json
		return func(c *gin.Context) {
			f, err := Assets.Open("manifest.json")
			if err != nil {
				c.String(http.StatusInternalServerError, "manifest.json not found in embedded assets")
				return
			}
			v, err := io.ReadAll(f)
			if err != nil {
				c.String(http.StatusInternalServerError, "failed to read manifest.json")
				return
			}
			v2 := strings.Replace(strings.Replace(string(v), "\"id\": \"/\"", "\"id\": \""+cfg.BasePath+"\"/", 1), "\"start_url\": \"/\"", "\"start_url\": \""+cfg.BasePath+"/\"", 1)

			setNoCache(c)
			c.Data(http.StatusOK, "application/manifest+json", []byte(v2))
		}
	} else if forced_url != "" {
		// Serve forced_url
		return func(c *gin.Context) {
			c.FileFromFS(forced_url, Assets)
		}
	} else {
		// Serve requested file
		return func(c *gin.Context) {
			reqPath := strings.TrimPrefix(c.Request.URL.Path, cfg.BasePath)

			// A missing content-hashed bundle means a stale cached client: hand
			// it a self-healing reload module (200) instead of a dead 404 so an
			// already-stuck client recovers on its own.
			if strings.HasPrefix(reqPath, "/_app/immutable/") && strings.HasSuffix(reqPath, ".js") {
				f, err := Assets.Open(reqPath)
				if err != nil {
					setNoCache(c)
					c.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(staleReloadJS))
					return
				}
				defer f.Close()

				// Serve straight from the handle we just opened rather than
				// letting FileFromFS reopen and re-stat the same file: this is
				// the hot path (every JS chunk of every page load).
				if fi, err := f.Stat(); err == nil {
					http.ServeContent(c.Writer, c.Request, reqPath, fi.ModTime(), f)
					return
				}
			}

			c.FileFromFS(reqPath, Assets)
		}
	}
}
