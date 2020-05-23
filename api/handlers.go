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

	"github.com/julienschmidt/httprouter"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/storage"
)

type Response interface {
	WriteResponse(http.ResponseWriter)
}

type FileResponse struct {
	contentType string
	content     io.WriterTo
}

func (r *FileResponse) WriteResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", r.contentType)
	w.WriteHeader(http.StatusOK)
	r.content.WriteTo(w)
}

type APIResponse struct {
	response interface{}
	cookies  []*http.Cookie
}

func (r APIResponse) WriteResponse(w http.ResponseWriter) {
	for _, cookie := range r.cookies {
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
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(j)
	}
}

type APIErrorResponse struct {
	status int
	err    error
	href   string
}

func (r APIErrorResponse) WriteResponse(w http.ResponseWriter) {
	if r.status == 0 {
		r.status = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	if len(r.href) == 0 {
		http.Error(w, fmt.Sprintf("{\"errmsg\":%q}", r.err.Error()), r.status)
	} else {
		http.Error(w, fmt.Sprintf("{\"errmsg\":%q,\"href\":%q}", r.err.Error(), r.href), r.status)
	}
}

func apiHandler(f func(*config.Options, httprouter.Params, io.Reader) Response) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
			r.RemoteAddr = addr
		}
		log.Printf("%s \"%s %s\" [%s]\n", r.RemoteAddr, r.Method, r.URL.Path, r.UserAgent())

		// Read the body
		if r.ContentLength < 0 || r.ContentLength > 6553600 {
			http.Error(w, fmt.Sprintf("{errmsg:\"Request too large or request size unknown\"}"), http.StatusRequestEntityTooLarge)
			return
		}

		opts := r.Context().Value("opts").(*config.Options)
		f(opts, ps, r.Body).WriteResponse(w)
	}
}

func apiAuthHandler(f func(*config.Options, *happydns.User, httprouter.Params, io.Reader) Response) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
			r.RemoteAddr = addr
		}
		log.Printf("%s \"%s %s\" [%s]\n", r.RemoteAddr, r.Method, r.URL.Path, r.UserAgent())

		// Read the body
		if r.ContentLength < 0 || r.ContentLength > 6553600 {
			http.Error(w, fmt.Sprintf("{errmsg:\"Request too large or request size unknown\"}"), http.StatusRequestEntityTooLarge)
			return
		}

		var sessionid []byte

		if cookie, err := r.Cookie("happydns_session"); err == nil {
			if sessionid, err = base64.StdEncoding.DecodeString(cookie.Value); err != nil {
				APIErrorResponse{
					err:    fmt.Errorf("Unable to authenticate request due to invalid cookie value: %w", err),
					status: http.StatusUnauthorized,
				}.WriteResponse(w)
				return
			}
		} else if flds := strings.Fields(r.Header.Get("Authorization")); len(flds) == 2 && flds[0] == "Bearer" {
			if sessionid, err = base64.StdEncoding.DecodeString(flds[1]); err != nil {
				APIErrorResponse{
					err:    fmt.Errorf("Unable to authenticate request due to invalid Authorization header value: %w", err),
					status: http.StatusUnauthorized,
				}.WriteResponse(w)
			}
		}

		if sessionid == nil || len(sessionid) == 0 {
			APIErrorResponse{
				err:    fmt.Errorf("Authorization required"),
				status: http.StatusUnauthorized,
			}.WriteResponse(w)
		} else if session, err := storage.MainStore.GetSession(sessionid); err != nil {
			APIErrorResponse{
				err:    err,
				status: http.StatusUnauthorized,
			}.WriteResponse(w)
		} else if std, err := storage.MainStore.GetUser(session.IdUser); err != nil {
			APIErrorResponse{
				err:    err,
				status: http.StatusUnauthorized,
			}.WriteResponse(w)
		} else {
			opts := r.Context().Value("opts").(*config.Options)
			f(opts, std, ps, r.Body).WriteResponse(w)
		}
	}
}
