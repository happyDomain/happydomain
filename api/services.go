package api

import (
	"io"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/model"
)

func init() {
	router.GET("/api/services", apiHandler(listServices))
	//router.POST("/api/services", apiHandler(newService))
}

func listServices(_ httprouter.Params, _ io.Reader) Response {
	if services, err := happydns.GetServices(); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: services,
		}
	}
}
