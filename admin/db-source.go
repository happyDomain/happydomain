// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
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

package admin

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/api"
	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.GET("/api/users/:userid/sources", api.ApiHandler(userHandler(getUserSources)))
	router.POST("/api/users/:userid/sources", api.ApiHandler(userHandler(newUserSource)))

	router.GET("/api/users/:userid/sources/:source", api.ApiHandler(userHandler(sourceHandler(getUserSource))))
	router.PUT("/api/users/:userid/sources/:source", api.ApiHandler(userHandler(sourceHandler(updateUserSource))))
	router.DELETE("/api/users/:userid/sources/:source", api.ApiHandler(userHandler(sourceHandler(deleteUserSource))))

	router.DELETE("/api/sources", api.ApiHandler(clearSources))
}

func getUserSources(_ *config.Options, user *happydns.User, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(storage.MainStore.GetSourceTypes(user))
}

func newUserSource(_ *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	us, err := api.DecodeSource(body)
	if err != nil {
		return api.NewAPIErrorResponse(http.StatusBadRequest, fmt.Errorf("Something is wrong in received data: %w", err))
	}
	us.Id = 0

	return api.NewAPIResponse(storage.MainStore.CreateSource(user, us, ""))
}

func sourceHandler(f func(*config.Options, *happydns.SourceCombined, httprouter.Params, io.Reader) api.Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) api.Response {
	return func(opts *config.Options, user *happydns.User, ps httprouter.Params, body io.Reader) api.Response {
		sourceid, err := strconv.ParseInt(ps.ByName("source"), 10, 64)
		if err != nil {
			return api.NewAPIErrorResponse(http.StatusNotFound, err)
		} else {
			source, err := storage.MainStore.GetSource(user, sourceid)
			if err != nil {
				return api.NewAPIErrorResponse(http.StatusNotFound, err)
			} else {
				return f(opts, source, ps, body)
			}
		}
	}
}

func getUserSource(_ *config.Options, source *happydns.SourceCombined, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(source, nil)
}

func updateUserSource(_ *config.Options, source *happydns.SourceCombined, _ httprouter.Params, body io.Reader) api.Response {
	us, err := api.DecodeSource(body)
	if err != nil {
		return api.NewAPIErrorResponse(http.StatusBadRequest, fmt.Errorf("Something is wrong in received data: %w", err))
	}
	us.Id = source.Id

	return api.NewAPIResponse(us, storage.MainStore.UpdateSource(us))
}

func deleteUserSource(_ *config.Options, source *happydns.SourceCombined, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(true, storage.MainStore.DeleteSource(&source.SourceType))
}

func clearSources(_ *config.Options, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(true, storage.MainStore.ClearSources())
}
