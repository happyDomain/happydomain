package api

import (
	"io"

	"github.com/julienschmidt/httprouter"
)

func init() {
	router.GET("/api/version", apiHandler(showVersion))
}

func showVersion(_ httprouter.Params, _ io.Reader) Response {
	return APIResponse{
		response: map[string]interface{}{"version": 0.1},
	}
}
