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
	router.DELETE("/api/sources/:sid", apiAuthHandler(sourceHandler(deleteSource)))

	router.GET("/api/sources/:sid/domains", apiAuthHandler(sourceHandler(getDomainsHostedBySource)))
}

func getSources(_ *config.Options, u *happydns.User, p httprouter.Params, body io.Reader) Response {
	if sources, err := storage.MainStore.GetSourceTypes(u); err != nil {
		return APIErrorResponse{
			err: err,
		}
	} else {
		return APIResponse{
			response: sources,
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

func decodeSource(body io.Reader) (*happydns.SourceCombined, error) {
	cnt, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var ust happydns.SourceType
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
	src, err := decodeSource(body)
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
	src, err := decodeSource(body)
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

func deleteSource(_ *config.Options, s *happydns.SourceCombined, u *happydns.User, body io.Reader) Response {
	if err := storage.MainStore.DeleteSource(&s.SourceType); err != nil {
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
