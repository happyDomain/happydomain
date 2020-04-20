package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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

type APIResponse struct {
	response interface{}
}

func (r APIResponse) WriteResponse(w http.ResponseWriter) {
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
}

func (r APIErrorResponse) WriteResponse(w http.ResponseWriter) {
	if r.status == 0 {
		r.status = http.StatusBadRequest
	}

	w.Header().Set("Content-Type", "application/json")
	http.Error(w, fmt.Sprintf("{\"errmsg\":%q}", r.err.Error()), r.status)
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

		if flds := strings.Fields(r.Header.Get("Authorization")); len(flds) != 2 || flds[0] != "Bearer" {
			APIErrorResponse{
				err:    errors.New("Authorization required"),
				status: http.StatusUnauthorized,
			}.WriteResponse(w)
		} else if sessionid, err := base64.StdEncoding.DecodeString(flds[1]); err != nil {
			APIErrorResponse{
				err:    err,
				status: http.StatusUnauthorized,
			}.WriteResponse(w)
		} else if session, err := storage.UsersStore.GetSession(sessionid); err != nil {
			APIErrorResponse{
				err:    err,
				status: http.StatusUnauthorized,
			}.WriteResponse(w)
		} else if std, err := storage.UsersStore.GetUser(int(session.IdUser)); err != nil {
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
