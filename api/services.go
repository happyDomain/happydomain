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

package api

import (
	"io"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.GET("/api/services", apiHandler(listServices))
	//router.POST("/api/services", apiHandler(newService))

	router.POST("/api/domains/:domain/analyze", apiAuthHandler(domainHandler(analyzeDomain)))
}

func listServices(_ *config.Options, _ httprouter.Params, _ io.Reader) Response {
	ret := map[string]svcs.ServiceInfos{}

	for k, svc := range *svcs.GetServices() {
		ret[k] = svc.Infos
	}

	return APIResponse{
		response: ret,
	}
}

func analyzeDomain(opts *config.Options, domain *happydns.Domain, body io.Reader) Response {
	source, err := storage.MainStore.GetSource(&happydns.User{Id: domain.IdUser}, domain.IdSource)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	zone, err := source.ImportZone(domain)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	services, aliases, err := svcs.AnalyzeZone(domain.DomainName, zone)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: map[string]interface{}{
			"aliases":  aliases,
			"services": services,
		},
	}
}
