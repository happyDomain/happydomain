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
	"encoding/json"
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
	router.GET("/api/users/:userid/domains", api.ApiHandler(userHandler(getUserDomains)))
	router.POST("/api/users/:userid/domains", api.ApiHandler(userHandler(newUserDomain)))

	router.GET("/api/users/:userid/domains/:domain", api.ApiHandler(userHandler(domainHandler(getUserDomain))))
	router.PUT("/api/users/:userid/domains/:domain", api.ApiHandler(userHandler(domainHandler(updateUserDomain))))
	router.DELETE("/api/users/:userid/domains/:domain", api.ApiHandler(userHandler(deleteUserDomain)))

	router.DELETE("/api/domains", api.ApiHandler(clearDomains))
}

func getUserDomains(_ *config.Options, user *happydns.User, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(storage.MainStore.GetDomains(user))
}

func newUserDomain(_ *config.Options, user *happydns.User, _ httprouter.Params, body io.Reader) api.Response {
	ud := &happydns.Domain{}
	err := json.NewDecoder(body).Decode(&ud)
	if err != nil {
		return api.NewAPIErrorResponse(http.StatusBadRequest, fmt.Errorf("Something is wrong in received data: %w", err))
	}
	ud.Id = 0
	ud.IdUser = user.Id

	return api.NewAPIResponse(ud, storage.MainStore.CreateDomain(user, ud))
}

func domainHandler(f func(*config.Options, *happydns.Domain, httprouter.Params, io.Reader) api.Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) api.Response {
	return func(opts *config.Options, user *happydns.User, ps httprouter.Params, body io.Reader) api.Response {
		domainid, err := strconv.ParseInt(ps.ByName("domain"), 10, 64)
		if err != nil {
			domain, err := storage.MainStore.GetDomainByDN(user, ps.ByName("domain"))
			if err != nil {
				return api.NewAPIErrorResponse(http.StatusNotFound, err)
			} else {
				return f(opts, domain, ps, body)
			}
		} else {
			domain, err := storage.MainStore.GetDomain(user, domainid)
			if err != nil {
				return api.NewAPIErrorResponse(http.StatusNotFound, err)
			} else {
				return f(opts, domain, ps, body)
			}
		}
	}
}

func getUserDomain(_ *config.Options, domain *happydns.Domain, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(domain, nil)
}

func updateUserDomain(_ *config.Options, domain *happydns.Domain, _ httprouter.Params, body io.Reader) api.Response {
	ud := &happydns.Domain{}
	err := json.NewDecoder(body).Decode(&ud)
	if err != nil {
		return api.NewAPIErrorResponse(http.StatusBadRequest, fmt.Errorf("Something is wrong in received data: %w", err))
	}
	ud.Id = domain.Id

	if ud.IdUser != domain.IdUser {
		if err := storage.MainStore.UpdateDomainOwner(domain, &happydns.User{Id: ud.IdUser}); err != nil {
			return api.NewAPIErrorResponse(http.StatusBadRequest, err)
		}
	}

	return api.NewAPIResponse(ud, storage.MainStore.UpdateDomain(ud))
}

func deleteUserDomain(_ *config.Options, user *happydns.User, ps httprouter.Params, _ io.Reader) api.Response {
	domainid, err := strconv.ParseInt(ps.ByName("domain"), 10, 64)
	if err != nil {
		domain, err := storage.MainStore.GetDomainByDN(user, ps.ByName("domain"))
		if err != nil {
			return api.NewAPIErrorResponse(http.StatusNotFound, err)
		} else {
			domainid = domain.Id
		}
	}
	return api.NewAPIResponse(true, storage.MainStore.DeleteDomain(&happydns.Domain{Id: domainid}))
}

func clearDomains(_ *config.Options, _ httprouter.Params, _ io.Reader) api.Response {
	return api.NewAPIResponse(true, storage.MainStore.ClearDomains())
}
