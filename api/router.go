package api

import (
	"github.com/julienschmidt/httprouter"
)

var router = httprouter.New()

func Router() *httprouter.Router {
	return router
}
