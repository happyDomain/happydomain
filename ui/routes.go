package ui

import (
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/config"
)

func DeclareRoutes(cfg *config.Options, router *gin.Engine) {
	router.GET("/", serveOrReverse("/", cfg))

	// Routes handled by the showcase
	router.GET("/en/*_", serveOrReverse("/", cfg))
	router.GET("/fr/*_", serveOrReverse("/", cfg))

	// Routes for real existings files
	router.GET("/css/*path", serveOrReverse("", cfg))
	router.GET("/fonts/*path", serveOrReverse("", cfg))
	router.GET("/img/*path", serveOrReverse("", cfg))
	router.GET("/js/*path", serveOrReverse("", cfg))
	router.GET("/favicon.ico", serveOrReverse("", cfg))
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
}

func serveOrReverse(forced_url string, cfg *config.Options) gin.HandlerFunc {
	if cfg.DevProxy != "" {
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

					for key := range resp.Header {
						c.Writer.Header().Add(key, resp.Header.Get(key))
					}
					c.Writer.WriteHeader(resp.StatusCode)

					io.Copy(c.Writer, resp.Body)
				}
			}
		}
	} else if forced_url != "" {
		return func(c *gin.Context) {
			c.FileFromFS(forced_url, Assets)
		}
	} else {
		return func(c *gin.Context) {
			c.FileFromFS(c.Request.URL.Path, Assets)
		}
	}
}
