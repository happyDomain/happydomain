package ui

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/config"
)

var (
	indexTpl       *template.Template
	CustomHeadHTML = ""
	CustomBodyHTML = ""
	HideVoxPeople  = false
)

func init() {
	flag.StringVar(&CustomHeadHTML, "custom-head-html", CustomHeadHTML, "Add custom HTML right before </head>")
	flag.StringVar(&CustomBodyHTML, "custom-body-html", CustomBodyHTML, "Add custom HTML right before </body>")
	flag.BoolVar(&HideVoxPeople, "hide-feedback-button", HideVoxPeople, "Hide the icon on page that permit to give feedback")
}

func DeclareRoutes(cfg *config.Options, router *gin.Engine) {
	if HideVoxPeople {
		CustomHeadHTML += "<style>#voxpeople { display: none !important; }</style>"
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
	router.GET("/_app/*_", serveOrReverse("", cfg))

	router.GET("/", serveOrReverse("/", cfg))
	router.GET("/index.html", serveOrReverse("/", cfg))

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

					for key := range resp.Header {
						c.Writer.Header().Add(key, resp.Header.Get(key))
					}
					c.Writer.WriteHeader(resp.StatusCode)

					io.Copy(c.Writer, resp.Body)
				}
			}
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
