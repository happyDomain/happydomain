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
	"strings"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/forms"
	"git.happydns.org/happydns/sources"
)

func init() {
	router.GET("/api/source_specs", ApiHandler(getSourceSpecs))
	router.GET("/api/source_specs/:ssid", ApiHandler(getSourceSpec))
	router.GET("/api/source_specs/:ssid/icon.png", ApiHandler(getSourceSpecIcon))
}

func getSourceSpecs(_ *config.Options, p httprouter.Params, body io.Reader) Response {
	srcs := sources.GetSources()

	ret := map[string]sources.SourceInfos{}
	for k, src := range *srcs {
		ret[k] = src.Infos
	}

	return APIResponse{
		response: ret,
	}
}

func getSourceSpecIcon(_ *config.Options, p httprouter.Params, body io.Reader) Response {
	ssid := string(p.ByName("ssid"))

	if cnt, ok := sources.Icons[strings.TrimSuffix(ssid, ".png")]; ok {
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

type viewSourceSpec struct {
	Fields       []*forms.Field `json:"fields,omitempty"`
	Capabilities []string       `json:"capabilities,omitempty"`
}

func getSourceSpec(_ *config.Options, p httprouter.Params, body io.Reader) Response {
	ssid := string(p.ByName("ssid"))

	src, err := sources.FindSource(ssid)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	return APIResponse{
		response: viewSourceSpec{
			Fields:       forms.GenStructFields(src),
			Capabilities: sources.GetSourceCapabilities(src),
		},
	}
}
