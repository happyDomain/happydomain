package api

import (
	"io"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
)

func init() {
	router.GET("/api/version", apiHandler(showVersion))
}

func showVersion(_ *config.Options, _ httprouter.Params, _ io.Reader) Response {
	return APIResponse{
		response: map[string]interface{}{"version": 0.1},
	}
}
