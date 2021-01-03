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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

type ResponseLogger struct {
	Status int
	Log    string
	Err    error
	Size   int64
}

func logResponse(r *http.Request, l ResponseLogger) {
	log.Printf("%s %d \"%s %s\" %d \"%s\" %d\n", r.RemoteAddr, l.Status, r.Method, r.URL.Path, r.ContentLength, r.UserAgent(), l.Size)
	if l.Log != "" {
		log.Println("  " + strings.TrimSpace(l.Log))
	}
	if l.Err != nil {
		log.Println("  " + l.Err.Error())
	}
}

type Response interface {
	WriteResponse(http.ResponseWriter) ResponseLogger
}

type FileResponse struct {
	contentType string
	content     io.WriterTo
}

func (r *FileResponse) WriteResponse(w http.ResponseWriter) (res ResponseLogger) {
	w.Header().Set("Content-Type", r.contentType)
	w.WriteHeader(http.StatusOK)
	n, err := r.content.WriteTo(w)

	return ResponseLogger{
		Status: http.StatusOK,
		Err:    err,
		Size:   n,
	}
}

type APIResponse struct {
	response interface{}
	cookies  []*http.Cookie
}

func (r APIResponse) WriteResponse(w http.ResponseWriter) ResponseLogger {
	log := ""
	for _, cookie := range r.cookies {
		log += fmt.Sprintf(" cookie=%s,expires=%d", cookie.Name, cookie.Expires)
		http.SetCookie(w, cookie)
	}

	if str, found := r.response.(string); found {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, str)
	} else if bts, found := r.response.([]byte); found {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment")
		w.Header().Set("Content-Transfer-Encoding", "binary")
		w.WriteHeader(http.StatusOK)
		w.Write(bts)
	} else if j, err := json.Marshal(r.response); err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, fmt.Sprintf("{\"errmsg\":%q}", err), http.StatusInternalServerError)

		return ResponseLogger{
			Status: http.StatusInternalServerError,
			Log:    log,
			Err:    err,
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(j)
	}

	return ResponseLogger{
		Status: http.StatusOK,
		Log:    log,
	}
}

func NewAPIResponse(response interface{}, err error) Response {
	if err != nil {
		return APIErrorResponse{
			status: http.StatusBadRequest,
			err:    err,
		}
	} else {
		return APIResponse{
			response: response,
		}
	}
}

type APIErrorResponse struct {
	status  int
	err     error
	href    string
	cookies []*http.Cookie
	cause   error
}

func (r APIErrorResponse) WriteResponse(w http.ResponseWriter) ResponseLogger {
	log := ""

	if r.status == 0 {
		r.status = http.StatusBadRequest
	}

	for _, cookie := range r.cookies {
		http.SetCookie(w, cookie)
		log += fmt.Sprintf(" cookie=%s,expires=%d", cookie.Name, cookie.Expires)
	}

	w.Header().Set("Content-Type", "application/json")
	if len(r.href) == 0 {
		http.Error(w, fmt.Sprintf("{\"errmsg\":%q}", r.err.Error()), r.status)
	} else {
		http.Error(w, fmt.Sprintf("{\"errmsg\":%q,\"href\":%q}", r.err.Error(), r.href), r.status)
		log += fmt.Sprintf(" href=%s", r.href)
	}

	return ResponseLogger{
		Status: r.status,
		Log:    log + " " + r.err.Error(),
		Err:    r.cause,
	}
}

func NewAPIErrorResponse(status int, err error) APIErrorResponse {
	return APIErrorResponse{
		status: status,
		err:    err,
	}
}

func ApiHandler(f func(*config.Options, httprouter.Params, io.Reader) Response) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
			r.RemoteAddr = addr
		}

		// Read the body
		if r.ContentLength < 0 || r.ContentLength > 6553600 {
			http.Error(w, fmt.Sprintf("{errmsg:\"Request too large or request size unknown\"}"), http.StatusRequestEntityTooLarge)
			return
		}

		opts := r.Context().Value("opts").(*config.Options)
		logResponse(r, f(opts, ps, r.Body).WriteResponse(w))
	}
}

type RequestResources struct {
	Domain     *happydns.Domain
	Ps         httprouter.Params
	Session    *happydns.Session
	Source     *happydns.SourceCombined
	SourceMeta *happydns.SourceMeta
	User       *happydns.User
	Zone       *happydns.Zone
}

func apiOptionalAuthHandler(noauthcb func(*config.Options, *RequestResources, io.Reader) Response, authcb func(*config.Options, *RequestResources, io.Reader) Response) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
			r.RemoteAddr = addr
		}

		// Read the body
		if r.ContentLength < 0 || r.ContentLength > 6553600 {
			http.Error(w, fmt.Sprintf("{errmsg:\"Request too large or request size unknown\"}"), http.StatusRequestEntityTooLarge)
			return
		}

		opts := r.Context().Value("opts").(*config.Options)

		var sessionid []byte

		if cookie, err := r.Cookie("happydns_session"); err == nil {
			if sessionid, err = base64.StdEncoding.DecodeString(cookie.Value); err != nil {
				logResponse(r,
					APIErrorResponse{
						err:    fmt.Errorf("Unable to authenticate request due to invalid cookie value: %w", err),
						status: http.StatusUnauthorized,
						cookies: []*http.Cookie{&http.Cookie{
							Name:     "happydns_session",
							Value:    "",
							Path:     opts.BaseURL + "/",
							Expires:  time.Unix(0, 0),
							Secure:   opts.DevProxy == "",
							HttpOnly: true,
						}},
					}.WriteResponse(w))
				return
			}
		} else if flds := strings.Fields(r.Header.Get("Authorization")); len(flds) == 2 && flds[0] == "Bearer" {
			if sessionid, err = base64.StdEncoding.DecodeString(flds[1]); err != nil {
				logResponse(r, APIErrorResponse{
					err:    fmt.Errorf("Unable to authenticate request due to invalid Authorization header value: %w", err),
					status: http.StatusUnauthorized,
				}.WriteResponse(w))
			}
		}

		var err error
		req := &RequestResources{
			Ps: ps,
		}

		if sessionid == nil || len(sessionid) == 0 {
			logResponse(r, noauthcb(opts, req, r.Body).WriteResponse(w))
		} else if req.Session, err = storage.MainStore.GetSession(sessionid); err != nil {
			logResponse(r, APIErrorResponse{
				err:    err,
				status: http.StatusUnauthorized,
				cookies: []*http.Cookie{&http.Cookie{
					Name:     "happydns_session",
					Value:    "",
					Path:     opts.BaseURL + "/",
					Expires:  time.Unix(0, 0),
					Secure:   opts.DevProxy == "",
					HttpOnly: true,
				}},
			}.WriteResponse(w))
		} else if req.User, err = storage.MainStore.GetUser(req.Session.IdUser); err != nil {
			logResponse(r, APIErrorResponse{
				err:    err,
				status: http.StatusUnauthorized,
				cookies: []*http.Cookie{&http.Cookie{
					Name:     "happydns_session",
					Value:    "",
					Path:     opts.BaseURL + "/",
					Expires:  time.Unix(0, 0),
					Secure:   opts.DevProxy == "",
					HttpOnly: true,
				}},
			}.WriteResponse(w))
		} else if req.User.Email == NO_AUTH_ACCOUNT && !opts.NoAuth {
			logResponse(r, noauthcb(opts, req, r.Body).WriteResponse(w))
		} else {
			logResponse(r, authcb(opts, req, r.Body).WriteResponse(w))
		}
	}

}

func apiAuthHandler(f func(*config.Options, *RequestResources, io.Reader) Response) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return apiOptionalAuthHandler(func(opts *config.Options, req *RequestResources, _ io.Reader) Response {
		return APIErrorResponse{
			err:    fmt.Errorf("Authorization required"),
			status: http.StatusUnauthorized,
		}
	}, func(opts *config.Options, req *RequestResources, r io.Reader) Response {
		response := f(opts, req, r)

		if req.Session.HasChanged() {
			storage.MainStore.UpdateSession(req.Session)
		}

		return response
	})
}
