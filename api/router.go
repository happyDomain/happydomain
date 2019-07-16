package api

import (
	"github.com/julienschmidt/httprouter"
)

var DefaultNameServer = "127.0.0.1:53"

var router = httprouter.New()

func Router() *httprouter.Router {
	return router
}
