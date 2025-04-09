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

package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/multierr"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/usecase"
)

type BackupController struct {
	config *config.Options
	store  storage.Storage
}

func NewBackupController(cfg *config.Options, store storage.Storage) *BackupController {
	return &BackupController{
		config: cfg,
		store:  store,
	}
}

func (bc *BackupController) DoBackup() (ret happydns.Backup) {
	ret.Version = bc.store.SchemaVersion()
	ret.DomainsLogs = map[string][]*happydns.DomainLog{}

	// UserAuth
	uas, err := bc.store.GetAuthUsers()
	if err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve AuthUsers: %s", err.Error()))
	} else {
		ret.UsersAuth = uas
	}

	// Users
	us, err := bc.store.GetUsers()
	if err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Users: %s", err.Error()))
	} else {
		ret.Users = us

		for _, u := range us {
			// Domains
			ds, err := bc.store.GetDomains(u)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Domain names of %s (%s): %s", u.Id.String(), u.Email, err.Error()))
			} else {
				ret.Domains = append(ret.Domains, ds...)

				for _, dn := range ds {
					// Domain logs
					ls, err := bc.store.GetDomainLogs(dn)
					if err != nil {
						ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve domain's logs %s/%s (%s): %s", u.Id.String(), dn.Id.String(), dn.DomainName, err.Error()))
					} else {
						ret.DomainsLogs[dn.Id.String()] = ls
					}

					// Zones
					for _, zid := range dn.ZoneHistory {
						z, err := bc.store.GetZone(zid)
						if err != nil {
							ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve domain's zone %s/%s (%s): zoneid=%s: %s", u.Id.String(), dn.Id.String(), dn.DomainName, zid.String(), err.Error()))
						} else {
							ret.Zones = append(ret.Zones, z)
						}
					}
				}
			}

			// Providers
			ps, err := bc.store.ListProviders(u)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Providers: %s", err.Error()))
			} else {
				ret.Providers = append(ret.Providers, ps...)
			}

			// Sessions
			ss, err := bc.store.GetUserSessions(u.Id)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Sessions: %s", err.Error()))
			} else {
				ret.Sessions = append(ret.Sessions, ss...)
			}
		}
	}

	return
}

func (bc *BackupController) DoRestore(backup *happydns.Backup) (errs error) {
	// UserAuth
	for _, ua := range backup.UsersAuth {
		errs = multierr.Combine(errs, bc.store.UpdateAuthUser(ua))
	}

	// Users
	for _, user := range backup.Users {
		err := bc.store.CreateOrUpdateUser(user)
		if err != nil {
			errs = multierr.Combine(errs, err)
		}
	}

	// Providers
	for _, provider := range backup.Providers {
		p, err := usecase.ParseProvider(provider)
		if err != nil {
			errs = multierr.Combine(errs, err)
		}

		errs = multierr.Combine(errs, bc.store.UpdateProvider(p))
	}

	// Domains
	for _, domain := range backup.Domains {
		err := bc.store.UpdateDomain(domain)
		if err != nil {
			errs = multierr.Combine(errs, err)
		} else {
			// Domain logs
			for _, log := range backup.DomainsLogs[domain.Id.String()] {
				errs = multierr.Combine(errs, bc.store.UpdateDomainLog(domain, log))
			}
		}
	}

	// Zones
	for _, zmsg := range backup.Zones {
		zone, err := usecase.ParseZone(zmsg)
		if err != nil {
			errs = multierr.Combine(errs, err)
		} else {
			errs = multierr.Combine(errs, bc.store.UpdateZone(zone))
		}
	}

	// Sessions
	for _, session := range backup.Sessions {
		errs = multierr.Combine(errs, bc.store.UpdateSession(session))
	}

	return
}

func (bc *BackupController) BackupJSON(c *gin.Context) {
	c.JSON(http.StatusOK, bc.DoBackup())
}

func (bc *BackupController) RestoreJSON(c *gin.Context) {
	var backup happydns.Backup
	err := c.ShouldBindJSON(&backup)
	if err != nil {
		log.Printf("%s sends invalid Backup JSON: %s", c.ClientIP(), err.Error())
		c.AbortWithStatusJSON(http.StatusBadRequest, happydns.ErrorResponse{Message: fmt.Sprintf("Something is wrong in received data: %s", err.Error())})
		return
	}

	err = bc.DoRestore(&backup)
	if err == nil {
		c.JSON(http.StatusOK, true)
	} else {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": err})
	}
}
