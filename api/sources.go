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
	"io/ioutil"
	"strconv"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/sources"
	"git.happydns.org/happydns/storage"
)

func init() {
	router.GET("/api/sources", apiAuthHandler(getSources))
	router.POST("/api/sources", apiAuthHandler(addSource))

	router.GET("/api/sources/:sid", apiAuthHandler(sourceHandler(getSource)))
	router.PUT("/api/sources/:sid", apiAuthHandler(sourceHandler(updateSource)))
	router.DELETE("/api/sources/:sid", apiAuthHandler(sourceMetaHandler(deleteSource)))

	router.GET("/api/sources/:sid/domains", apiAuthHandler(sourceHandler(getDomainsHostedBySource)))

	router.GET("/api/sources/:sid/available_resource_types", apiAuthHandler(sourceHandler(getAvailableResourceTypes)))
}

func getSources(_ *config.Options, req *RequestResources, body io.Reader) Response {
	if sources, err := storage.MainStore.GetSourceMetas(req.User); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else if len(sources) > 0 {
		return APIResponse{
			response: sources,
		}
	} else {
		return APIResponse{
			response: []happydns.Source{},
		}
	}
}

func sourceMetaHandler(f func(*config.Options, *RequestResources, io.Reader) Response) func(*config.Options, *RequestResources, io.Reader) Response {
	return func(opts *config.Options, req *RequestResources, body io.Reader) Response {
		if sid, err := strconv.ParseInt(string(req.Ps.ByName("sid")), 10, 64); err != nil {
			return APIErrorResponse{err: err}
		} else if req.SourceMeta, err = storage.MainStore.GetSourceMeta(req.User, sid); err != nil {
			return APIErrorResponse{err: err}
		} else {
			return f(opts, req, body)
		}
	}
}

func sourceHandler(f func(*config.Options, *RequestResources, io.Reader) Response) func(*config.Options, *RequestResources, io.Reader) Response {
	return func(opts *config.Options, req *RequestResources, body io.Reader) Response {
		if sid, err := strconv.ParseInt(string(req.Ps.ByName("sid")), 10, 64); err != nil {
			return APIErrorResponse{err: err}
		} else if req.Source, err = storage.MainStore.GetSource(req.User, sid); err != nil {
			return APIErrorResponse{err: err}
		} else {
			req.SourceMeta = &req.Source.SourceMeta
			return f(opts, req, body)
		}
	}
}

func getSource(_ *config.Options, req *RequestResources, body io.Reader) Response {
	return APIResponse{
		response: req.Source,
	}
}

func DecodeSource(body io.Reader) (*happydns.SourceCombined, error) {
	cnt, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var ust happydns.SourceMeta
	err = json.Unmarshal(cnt, &ust)
	if err != nil {
		return nil, err
	}

	us, err := sources.FindSource(ust.Type)
	if err != nil {
		return nil, err
	}

	src := &happydns.SourceCombined{
		us,
		ust,
	}

	err = json.Unmarshal(cnt, &src)
	if err != nil {
		return nil, err
	}

	err = src.Validate()
	if err != nil {
		return nil, err
	}

	return src, nil
}

func addSource(_ *config.Options, req *RequestResources, body io.Reader) Response {
	src, err := DecodeSource(body)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if s, err := storage.MainStore.CreateSource(req.User, src.Source, src.Comment); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: s,
		}
	}
}

func updateSource(_ *config.Options, req *RequestResources, body io.Reader) Response {
	src, err := DecodeSource(body)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	src.Id = req.Source.Id
	src.OwnerId = req.Source.OwnerId

	if err := storage.MainStore.UpdateSource(src); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: src,
		}
	}
}

func deleteSource(_ *config.Options, req *RequestResources, body io.Reader) Response {
	// Check if the source has no more domain associated
	domains, err := storage.MainStore.GetDomains(req.User)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	for _, domain := range domains {
		if domain.IdSource == req.SourceMeta.Id {
			return APIErrorResponse{
				err: fmt.Errorf("You cannot delete this source because there is still some domains associated with it."),
			}
		}
	}

	if err := storage.MainStore.DeleteSource(req.SourceMeta); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: nil,
		}
	}
}

func getDomainsHostedBySource(_ *config.Options, req *RequestResources, body io.Reader) Response {
	sr, ok := req.Source.Source.(sources.ListDomainsSource)
	if !ok {
		return APIErrorResponse{
			err: fmt.Errorf("Source doesn't support domain listing."),
		}
	}

	if domains, err := sr.ListDomains(); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: domains,
		}
	}
}

func getAvailableResourceTypes(_ *config.Options, req *RequestResources, body io.Reader) Response {
	lrt, ok := req.Source.Source.(sources.LimitedResourceTypesSource)
	if !ok {
		// Return all types known to be supported by happyDNS
		return APIResponse{
			response: sources.DefaultAvailableTypes,
		}
	} else {
		return APIResponse{
			response: lrt.ListAvailableTypes(),
		}
	}
}
