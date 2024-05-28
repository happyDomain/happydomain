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

package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/actions"
	"git.happydns.org/happyDomain/config"
)

func declareApiCompatRoutes(cfg *config.Options, router *gin.RouterGroup) {
	router.GET("/nic/update", noipUpdateRoute)
}

func noipUpdateRoute(c *gin.Context) {
	if auth_method, ok := c.Get("AuthMethod"); !ok || (auth_method != "basic" && auth_method != "bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "To avoid security issues (CSRF), you can only use /nic/update with the HTTP Bearer Authorization header. Generate a key in your account settings."})
		return
	}

	user := myUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"errmsg": "User not defined"})
		return
	}

	hostnames := strings.Split(c.Query("hostname"), ",")
	if len(hostnames) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "hostname parameter is required"})
		return
	}

	// Standardize hostnames
	for i := range hostnames {
		hostnames[i] = dns.CanonicalName(hostnames[i])
	}

	myips := strings.Split(c.Query("myip"), ",")
	myipv6 := c.Query("myipv6")

	if len(myips) > 2 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errmsg": "myip should not contains more than 2 IP (1 ipv4 and 1 ipv6)"})
		return
	}

	var myipv4 string
	for _, ip := range myips {
		if strings.Contains(ip, ":") && myipv6 == "" {
			myipv6 = ip
		} else {
			myipv4 = ip
		}
	}

	err := actions.DynamicUpdate(user, hostnames, myipv4, myipv6)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errmsg": fmt.Sprintf("Unable to update your domain: %s", err.Error())})
		return
	}

	if offline := c.Query("offline"); offline != "" {
		// This is just a warning, not error
		c.AbortWithStatusJSON(http.StatusOK, gin.H{"errmsg": "Please note that offline parameter is not handled by happyDomain."})
		return
	}

	c.Status(http.StatusOK)
}
