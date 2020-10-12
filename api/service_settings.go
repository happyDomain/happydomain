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
	"encoding/json"
	"io"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/forms"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.POST("/api/domains/:domain/zone/:zoneid/:subdomain/services/*psid", apiAuthHandler(domainHandler(zoneHandler(getServiceSettingsState))))
}

type ServiceSettingsState struct {
	FormState
	happydns.Service
}

type ServiceSettingsResponse struct {
	FormResponse
	Services map[string][]*happydns.ServiceCombined `json:"services,omitempty"`
}

func getServiceSettingsState(cfg *config.Options, req *RequestResources, body io.Reader) Response {
	psid := string(req.Ps.ByName("psid"))
	// Remove the leading slash
	if len(psid) > 1 {
		psid = psid[1:]
	}

	pvr, err := svcs.FindService(psid)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	var ups ServiceSettingsState
	ups.Service = pvr
	err = json.NewDecoder(body).Decode(&ups)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	form, err := formDoState(cfg, req, &ups.FormState, ups.Service, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != forms.DoneForm {
			return APIErrorResponse{
				err: err,
			}
		} else if ups.Id == 0 {
			// Append a new Service
			if err = req.Zone.AppendService(string(req.Ps.ByName("subdomain")), req.Domain.DomainName, &happydns.ServiceCombined{Service: ups.Service}); err != nil {
				return APIErrorResponse{
					err: err,
				}
			} else if err = storage.MainStore.UpdateZone(req.Zone); err != nil {
				return APIErrorResponse{
					err: err,
				}
			} else {
				return APIResponse{
					response: ServiceSettingsResponse{
						Services:     req.Zone.Services,
						FormResponse: FormResponse{Redirect: ups.Redirect},
					},
				}
			}
		} else {
			// Update an existing Service
			if err = req.Zone.EraseServiceWithoutMeta(string(req.Ps.ByName("subdomain")), req.Domain.DomainName, ups.IdB, ups); err != nil {
				return APIErrorResponse{
					err: err,
				}
			} else if err = storage.MainStore.UpdateZone(req.Zone); err != nil {
				return APIErrorResponse{
					err: err,
				}
			} else {
				return APIResponse{
					response: ServiceSettingsResponse{
						Services:     req.Zone.Services,
						FormResponse: FormResponse{Redirect: ups.Redirect},
					},
				}
			}
		}
	}

	return APIResponse{
		response: SourceSettingsResponse{
			FormResponse: FormResponse{From: form},
		},
	}
}
