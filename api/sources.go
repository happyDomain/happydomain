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

	"github.com/julienschmidt/httprouter"

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
}

func getSources(_ *config.Options, u *happydns.User, p httprouter.Params, body io.Reader) Response {
	if sources, err := storage.MainStore.GetSourceMetas(u); err != nil {
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

func sourceMetaHandler(f func(*config.Options, *happydns.SourceMeta, *happydns.User, io.Reader) Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) Response {
	return func(opts *config.Options, u *happydns.User, ps httprouter.Params, body io.Reader) Response {
		if sid, err := strconv.ParseInt(string(ps.ByName("sid")), 10, 64); err != nil {
			return APIErrorResponse{err: err}
		} else if srcMeta, err := storage.MainStore.GetSourceMeta(u, sid); err != nil {
			return APIErrorResponse{err: err}
		} else {
			return f(opts, srcMeta, u, body)
		}
	}
}

func sourceHandler(f func(*config.Options, *happydns.SourceCombined, *happydns.User, io.Reader) Response) func(*config.Options, *happydns.User, httprouter.Params, io.Reader) Response {
	return func(opts *config.Options, u *happydns.User, ps httprouter.Params, body io.Reader) Response {
		if sid, err := strconv.ParseInt(string(ps.ByName("sid")), 10, 64); err != nil {
			return APIErrorResponse{err: err}
		} else if source, err := storage.MainStore.GetSource(u, sid); err != nil {
			return APIErrorResponse{err: err}
		} else {
			return f(opts, source, u, body)
		}
	}
}

func getSource(_ *config.Options, s *happydns.SourceCombined, u *happydns.User, body io.Reader) Response {
	return APIResponse{
		response: s,
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

func addSource(_ *config.Options, u *happydns.User, p httprouter.Params, body io.Reader) Response {
	src, err := DecodeSource(body)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	if s, err := storage.MainStore.CreateSource(u, src.Source, src.Comment); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: s,
		}
	}
}

func updateSource(_ *config.Options, s *happydns.SourceCombined, u *happydns.User, body io.Reader) Response {
	src, err := DecodeSource(body)
	if err != nil {
		return APIErrorResponse{
			err: err,
		}
	}

	src.Id = s.Id
	src.OwnerId = s.OwnerId

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

func deleteSource(_ *config.Options, st *happydns.SourceMeta, u *happydns.User, body io.Reader) Response {
	if err := storage.MainStore.DeleteSource(st); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: nil,
		}
	}
}

func getDomainsHostedBySource(_ *config.Options, s *happydns.SourceCombined, u *happydns.User, body io.Reader) Response {
	sr, ok := s.Source.(sources.ListDomainsSource)
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
