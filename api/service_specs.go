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
	"bytes"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/services"
)

func init() {
	router.GET("/api/service_specs", ApiHandler(getServiceSpecs))
	router.GET("/api/service_specs/:ssid", ApiHandler(getServiceSpec))
}

type service_field struct {
	Id          string   `json:"id"`
	Type        string   `json:"type"`
	Label       string   `json:"label,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Default     string   `json:"default,omitempty"`
	Choices     []string `json:"choices,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
	Description string   `json:"description,omitempty"`
}

func getServiceSpecs(_ *config.Options, p httprouter.Params, body io.Reader) Response {
	services := svcs.GetServices()

	ret := map[string]svcs.ServiceInfos{}
	for k, service := range *services {
		ret[k] = service.Infos
	}

	return APIResponse{
		response: ret,
	}
}

func getServiceSpecImg(ssid string) Response {
	if cnt, ok := svcs.Icons[strings.TrimSuffix(ssid, ".png")]; ok {
		return &FileResponse{
			contentType: "image/png",
			content:     bytes.NewBuffer(cnt),
		}
	} else {
		return APIErrorResponse{
			status: http.StatusNotFound,
			err:    errors.New("Icon not found."),
		}
	}
}

type viewServiceSpec struct {
	Fields []service_field `json:"fields,omitempty"`
}

func getServiceSpec(_ *config.Options, p httprouter.Params, body io.Reader) Response {
	ssid := string(p.ByName("ssid"))

	svc, err := svcs.FindSubService(ssid)
	if err != nil {
		return APIErrorResponse{
			err:    err,
			status: http.StatusNotFound,
		}
	}

	svcType := reflect.Indirect(reflect.ValueOf(svc)).Type()

	fields := []service_field{}
	for i := 0; i < svcType.NumField(); i += 1 {
		jsonTag := svcType.Field(i).Tag.Get("json")
		jsonTuples := strings.Split(jsonTag, ",")

		f := service_field{
			Type: svcType.Field(i).Type.String(),
		}

		if len(jsonTuples) > 0 && len(jsonTuples[0]) > 0 {
			f.Id = jsonTuples[0]
		} else {
			f.Id = svcType.Field(i).Name
		}

		tag := svcType.Field(i).Tag.Get("happydns")
		tuples := strings.Split(tag, ",")

		for _, t := range tuples {
			kv := strings.SplitN(t, "=", 2)
			if len(kv) > 1 {
				switch strings.ToLower(kv[0]) {
				case "label":
					f.Label = kv[1]
				case "placeholder":
					f.Placeholder = kv[1]
				case "default":
					f.Default = kv[1]
				case "description":
					f.Description = kv[1]
				case "choices":
					f.Choices = strings.Split(kv[1], ";")
				}
			} else {
				switch strings.ToLower(kv[0]) {
				case "required":
					f.Required = true
				case "secret":
					f.Secret = true
				default:
					f.Label = kv[0]
				}
			}
		}
		fields = append(fields, f)
	}

	return APIResponse{
		response: viewServiceSpec{fields},
	}
}
