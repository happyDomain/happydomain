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
	"fmt"
	"io"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.POST("/api/source_settings/*ssid", apiAuthHandler(getSourceSettingsState))
}

type SourceSettingsState struct {
	happydns.Source
	Id       int64   `json:"_id,omitempty"`
	Name     string  `json:"_comment"`
	State    int32   `json:"state"`
	Recall   *int64  `json:"recall,omitempty"`
	Redirect *string `json:"redirect,omitempty"`
}

type SourceSettingsResponse struct {
	happydns.Source `json:"Source,omitempty"`
	From            *sources.CustomForm `json:"form,omitempty"`
	Redirect        *string             `json:"redirect,omitempty"`
}

func getSourceSettingsState(cfg *config.Options, req *RequestResources, body io.Reader) Response {
	ssid := string(req.Ps.ByName("ssid"))
	// Remove the leading slash
	if len(ssid) > 1 {
		ssid = ssid[1:]
	}

	src, err := sources.FindSource(ssid)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	var uss SourceSettingsState
	uss.Source = src
	err = json.NewDecoder(body).Decode(&uss)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if uss.Recall != nil {
		req.Session.GetValue(fmt.Sprintf("source-creation-%d", *uss.Recall), src)
		req.Session.GetValue(fmt.Sprintf("source-creation-%d-name", *uss.Recall), &uss.Name)
		req.Session.GetValue(fmt.Sprintf("source-creation-%d-id", *uss.Recall), &uss.Id)
		req.Session.GetValue(fmt.Sprintf("source-creation-%d-next", *uss.Recall), &uss.Redirect)
	}

	var form *sources.CustomForm

	csf, ok := src.(sources.CustomSettingsForm)
	if !ok {
		if uss.State == 1 {
			err = sources.DoneForm
		} else {
			form = sources.GenDefaultSettingsForm(src)
		}
	} else {
		form, err = csf.DisplaySettingsForm(uss.State, cfg, req.Session, func() int64 {
			key, recallid := req.Session.FindNewKey("source-creation-")
			req.Session.SetValue(key, src)
			req.Session.SetValue(key+"-name", uss.Name)
			req.Session.SetValue(key+"-id", uss.Id)
			if uss.Redirect != nil {
				req.Session.SetValue(key+"-next", *uss.Redirect)
			}
			return recallid
		})
	}

	if err != nil {
		if err != sources.DoneForm {
			return APIErrorResponse{
				err: err,
			}
		} else if err = src.Validate(); err != nil {
			return APIErrorResponse{
				err: err,
			}
		} else if uss.Id == 0 {
			// Create a new Source
			if s, err := storage.MainStore.CreateSource(req.User, src, uss.Name); err != nil {
				return APIErrorResponse{
					err: err,
				}
			} else {
				return APIResponse{
					response: SourceSettingsResponse{
						Source:   s,
						Redirect: uss.Redirect,
					},
				}
			}
		} else {
			// Update an existing Source
			if s, err := storage.MainStore.GetSource(req.User, uss.Id); err != nil {
				return APIErrorResponse{
					err: err,
				}
			} else {
				s.Comment = uss.Name
				s.Source = uss.Source

				if err := storage.MainStore.UpdateSource(s); err != nil {
					return APIErrorResponse{
						err: err,
					}
				} else {
					return APIResponse{
						response: SourceSettingsResponse{
							Source:   s,
							Redirect: uss.Redirect,
						},
					}
				}
			}
		}
	}

	return APIResponse{
		response: SourceSettingsResponse{
			From: form,
		},
	}
}
