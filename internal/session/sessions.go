// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package session // import "git.happydns.org/happyDomain/internal/session"

import (
	"encoding/base32"
	"fmt"
	"net/http"
	"strings"
	"time"

	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/mileusna/useragent"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

const COOKIE_NAME = "happydomain_session"

// SessionStore is an implementation of Gorilla Sessions, using happyDomain storage.
type SessionStore struct {
	Codecs  []securecookie.Codec
	options *sessions.Options
	storage storage.Storage
}

func NewSessionStore(opts *config.Options, storage storage.Storage, keyPairs ...[]byte) *SessionStore {
	store := &SessionStore{
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		options: &sessions.Options{
			Path:     opts.GetBasePath() + "/",
			MaxAge:   86400 * 30,
			Secure:   opts.DevProxy == "" && opts.ExternalURL.URL.Scheme != "http",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
		storage: storage,
	}
	store.MaxAge(store.options.MaxAge)
	return store
}

// Get Fetches a session for a given name after it has been added to the registry.
func (s *SessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a new session for the given name without adding it to the registry.
func (s *SessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(s, name)
	options := *s.options
	session.Options = &options
	session.IsNew = true

	if _, ok := r.Header["Authorization"]; ok && len(r.Header["Authorization"]) > 0 {
		if flds := strings.Fields(r.Header["Authorization"][0]); len(flds) == 2 && flds[0] == "Bearer" {
			session.ID = flds[1]
		} else if user, _, ok := r.BasicAuth(); ok {
			session.ID = user
		}
	} else if cookie, err := r.Cookie(name); err == nil {
		err := securecookie.DecodeMulti(name, cookie.Value, &session.ID, s.Codecs...)
		if err != nil {
			// Value could not be decrypted, consider this is a new session
			return session, err
		}
	}

	if len(session.ID) == 0 {
		// Cookie not found, this is a new session
		return session, nil
	}

	err := s.load(session)
	session.IsNew = false
	return session, err
}

// Save saves the given session into the database and deletes cookies if needed.
func (s *SessionStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	var cookieValue string

	if s.options.MaxAge < 0 || session.Options.MaxAge < 0 {
		s.storage.DeleteSession(session.ID)
	} else {
		if session.ID == "" {
			session.ID = NewSessionId()
		}
		encrypted, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
		if err != nil {
			return err
		}
		cookieValue = encrypted

		err = s.save(session, r.UserAgent())
		if err != nil {
			return err
		}
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), cookieValue, session.Options))
	return nil
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
func (s *SessionStore) MaxAge(age int) {
	s.options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range s.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

func (s *SessionStore) Options(options ginsessions.Options) {
	s.options = options.ToGorillaOptions()
}

func (s *SessionStore) load(session *sessions.Session) error {
	mysession, err := s.storage.GetSession(session.ID)
	if err != nil {
		return err
	}

	if len(mysession.Content) > 0 {
		err = securecookie.DecodeMulti(session.Name(), mysession.Content, &session.Values, s.Codecs...)
		if err != nil {
			return err
		}
	}

	if len(mysession.IdUser) > 0 {
		session.Values["iduser"] = happydns.Identifier(mysession.IdUser)
	}
	if len(mysession.Description) > 0 {
		session.Values["description"] = mysession.Description
	}
	if _, ok := session.Values["created_on"].(time.Time); !ok && !mysession.IssuedAt.IsZero() {
		session.Values["created_on"] = mysession.IssuedAt
	}
	if !mysession.ExpiresOn.IsZero() {
		session.Values["expires_on"] = mysession.ExpiresOn
	}

	return nil
}

// save writes encoded session.Values to a database record.
func (s *SessionStore) save(session *sessions.Session, ua string) error {
	var iduser happydns.Identifier
	if iu, ok := session.Values["iduser"].(happydns.Identifier); ok {
		iduser = iu
	} else if iu, ok := session.Values["iduser"].([]byte); ok {
		iduser = happydns.Identifier(iu)
	}
	delete(session.Values, "iduser")

	var description string
	if descr, ok := session.Values["description"].(string); ok {
		description = descr
	} else {
		browser := useragent.Parse(ua)
		description = fmt.Sprintf("%s on %s", browser.Name, browser.OS)
		session.Values["description"] = description
	}
	delete(session.Values, "description")

	crOn := session.Values["created_on"]
	delete(session.Values, "created_on")
	exOn := session.Values["expires_on"]
	delete(session.Values, "expires_on")

	var expiresOn time.Time

	createdOn, ok := crOn.(time.Time)
	if !ok {
		createdOn = time.Now()
	}

	if exOn == nil {
		expiresOn = time.Now().Add(time.Second * time.Duration(session.Options.MaxAge))
	} else {
		expiresOn = exOn.(time.Time)
		if expiresOn.Sub(time.Now().Add(time.Second*time.Duration(session.Options.MaxAge))) < 0 {
			expiresOn = time.Now().Add(time.Second * time.Duration(session.Options.MaxAge))
		}
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}

	mysession := &happydns.Session{
		Id:          session.ID,
		IdUser:      iduser,
		Content:     encoded,
		Description: description,
		IssuedAt:    createdOn,
		ExpiresOn:   expiresOn,
		ModifiedOn:  time.Now(),
	}

	return s.storage.UpdateSession(mysession)
}

func NewSessionId() string {
	return strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(64)), "=")
}
