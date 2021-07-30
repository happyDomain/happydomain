// Copyright or © or Copr. happyDNS (2021)
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
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happydns/config"
	"git.happydns.org/happydns/storage"
)

const COOKIE_NAME = "happydns_session"

func authMiddleware(opts *config.Options, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sessionid []byte

		// Retrieve the session from cookie or header
		if cookie, err := c.Cookie(COOKIE_NAME); err == nil {
			if sessionid, err = base64.StdEncoding.DecodeString(cookie); err != nil {
				c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": fmt.Sprintf("Unable to authenticate request due to invalid cookie value: %s", err.Error())})
				return
			}
		} else if flds := strings.Fields(c.GetHeader("Authorization")); len(flds) == 2 && flds[0] == "Bearer" {
			if sessionid, err = base64.StdEncoding.DecodeString(flds[1]); err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": fmt.Sprintf("Unable to authenticate request due to invalid Authorization header value: %s", err.Error())})
				return
			}
		}

		// Stop here if there is no cookie and we allow no auth
		if optional && (sessionid == nil || len(sessionid) == 0) {
			c.Next()
			return
		}

		session, err := storage.MainStore.GetSession(sessionid)
		if err != nil {
			log.Printf("%s tries an invalid session: %s", c.ClientIP(), err.Error())
			c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": fmt.Sprintf("Your session has expired. Please reconnect.")})
			return
		}

		c.Set("MySession", session)

		user, err := storage.MainStore.GetUser(session.IdUser)
		if err != nil {
			log.Printf("%s has a correct session, but related user is invalid: %s", c.ClientIP(), err.Error())
			c.SetCookie(COOKIE_NAME, "", -1, opts.BaseURL+"/", "", opts.DevProxy == "", true)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": fmt.Sprintf("Something goes wrong with your session. Please reconnect.")})
			return
		}

		c.Set("LoggedUser", user)

		c.Next()

		if session.HasChanged() {
			storage.MainStore.UpdateSession(session)
		}
	}
}
