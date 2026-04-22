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
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/storage"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type BackupController struct {
	config *happydns.Options
	store  storage.Storage
}

func NewBackupController(cfg *happydns.Options, store storage.Storage) *BackupController {
	return &BackupController{
		config: cfg,
		store:  store,
	}
}

func (bc *BackupController) DoBackup() (ret happydns.Backup) {
	ret.Version = bc.store.SchemaVersion()
	ret.DomainsLogs = map[string][]*happydns.DomainLog{}

	// UserAuth
	uai, err := bc.store.ListAllAuthUsers()
	if err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve AuthUsers: %s", err.Error()))
	} else {
		defer uai.Close()
		for uai.Next() {
			ret.UsersAuth = append(ret.UsersAuth, uai.Item())
		}
	}

	// Users
	iter, err := bc.store.ListAllUsers()
	if err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Users: %s", err.Error()))
	} else {
		defer iter.Close()

		for iter.Next() {
			u := iter.Item()

			ret.Users = append(ret.Users, u)

			// Domains
			ds, err := bc.store.ListDomains(u)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Domain names of %s (%s): %s", u.Id.String(), u.Email, err.Error()))
			} else {
				ret.Domains = append(ret.Domains, ds...)

				for _, dn := range ds {
					// Domain logs
					ls, logErr := bc.store.ListDomainLogs(dn)
					if logErr != nil {
						ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve domain's logs %s/%s (%s): %s", u.Id.String(), dn.Id.String(), dn.DomainName, logErr.Error()))
					} else {
						ret.DomainsLogs[dn.Id.String()] = ls
					}

					// Zones
					for _, zid := range dn.ZoneHistory {
						z, zoneErr := bc.store.GetZone(zid)
						if zoneErr != nil {
							ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve domain's zone %s/%s (%s): zoneid=%s: %s", u.Id.String(), dn.Id.String(), dn.DomainName, zid.String(), zoneErr.Error()))
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
			ss, err := bc.store.ListUserSessions(u.Id)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Sessions: %s", err.Error()))
			} else {
				ret.Sessions = append(ret.Sessions, ss...)
			}
		}
	}

	// Checker configurations (positional, one entry per (checker, user?, domain?, service?)).
	if cfgIter, err := bc.store.ListAllCheckerConfigurations(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckerConfigurations: %s", err.Error()))
	} else {
		defer cfgIter.Close()
		for cfgIter.Next() {
			ret.CheckerConfigurations = append(ret.CheckerConfigurations, cfgIter.Item())
		}
	}

	// Check plans.
	if planIter, err := bc.store.ListAllCheckPlans(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckPlans: %s", err.Error()))
	} else {
		defer planIter.Close()
		for planIter.Next() {
			ret.CheckPlans = append(ret.CheckPlans, planIter.Item())
		}
	}

	// Check evaluations.
	if evalIter, err := bc.store.ListAllEvaluations(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckEvaluations: %s", err.Error()))
	} else {
		defer evalIter.Close()
		for evalIter.Next() {
			ret.CheckEvaluations = append(ret.CheckEvaluations, evalIter.Item())
		}
	}

	// Executions.
	if execIter, err := bc.store.ListAllExecutions(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Executions: %s", err.Error()))
	} else {
		defer execIter.Close()
		for execIter.Next() {
			ret.Executions = append(ret.Executions, execIter.Item())
		}
	}

	// Discovery entries.
	if entryIter, err := bc.store.ListAllDiscoveryEntries(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve DiscoveryEntries: %s", err.Error()))
	} else {
		defer entryIter.Close()
		for entryIter.Next() {
			ret.DiscoveryEntries = append(ret.DiscoveryEntries, entryIter.Item())
		}
	}

	// Discovery observation refs.
	if refIter, err := bc.store.ListAllDiscoveryObservationRefs(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve DiscoveryObservationRefs: %s", err.Error()))
	} else {
		defer refIter.Close()
		for refIter.Next() {
			ret.DiscoveryObservationRefs = append(ret.DiscoveryObservationRefs, refIter.Item())
		}
	}

	return
}

func (bc *BackupController) DoRestore(backup *happydns.Backup) (errs error) {
	// UserAuth
	for _, ua := range backup.UsersAuth {
		errs = errors.Join(errs, bc.store.UpdateAuthUser(ua))
	}

	// Users
	for _, user := range backup.Users {
		err := bc.store.CreateOrUpdateUser(user)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	// Providers
	for _, provider := range backup.Providers {
		p, err := providerUC.ParseProvider(provider)
		if err != nil {
			errs = errors.Join(errs, err)
		}

		errs = errors.Join(errs, bc.store.UpdateProvider(p))
	}

	// Domains
	for _, domain := range backup.Domains {
		err := bc.store.UpdateDomain(domain)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			// Domain logs
			for _, log := range backup.DomainsLogs[domain.Id.String()] {
				errs = errors.Join(errs, bc.store.UpdateDomainLog(domain, log))
			}
		}
	}

	// Zones
	for _, zmsg := range backup.Zones {
		zone, err := zoneUC.ParseZone(zmsg)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			errs = errors.Join(errs, bc.store.UpdateZone(zone))
		}
	}

	// Sessions
	for _, session := range backup.Sessions {
		errs = errors.Join(errs, bc.store.UpdateSession(session))
	}

	// Checker configurations.
	for _, cfg := range backup.CheckerConfigurations {
		if cfg == nil {
			continue
		}
		errs = errors.Join(errs, bc.store.UpdateCheckerConfiguration(cfg.CheckName, cfg.UserId, cfg.DomainId, cfg.ServiceId, cfg.Options))
	}

	// Check plans.
	for _, plan := range backup.CheckPlans {
		errs = errors.Join(errs, bc.store.RestoreCheckPlan(plan))
	}

	// Check evaluations (reference plans, restored above).
	for _, eval := range backup.CheckEvaluations {
		errs = errors.Join(errs, bc.store.RestoreEvaluation(eval))
	}

	// Executions.
	for _, exec := range backup.Executions {
		errs = errors.Join(errs, bc.store.RestoreExecution(exec))
	}

	// Discovery entries. Restored after snapshots (referenced indirectly via
	// target + producer, no FK), before observation refs which carry snapshot
	// pointers that must resolve at lookup time.
	for _, entry := range backup.DiscoveryEntries {
		errs = errors.Join(errs, bc.store.RestoreDiscoveryEntry(entry))
	}

	// Discovery observation refs.
	for _, ref := range backup.DiscoveryObservationRefs {
		errs = errors.Join(errs, bc.store.RestoreDiscoveryObservationRef(ref))
	}

	return
}

// BackupJSON creates a complete backup of the system.
//
//	@Summary		Create backup
//	@Schemes
//	@Description	Create a complete backup of the system including users, authentication, domains, zones, providers, and sessions.
//	@Tags			backup
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		500	{object}	happydns.ErrorResponse
//	@Router			/backup.json [post]
func (bc *BackupController) BackupJSON(c *gin.Context) {
	c.JSON(http.StatusOK, bc.DoBackup())
}

// RestoreJSON restores a complete backup of the system.
//
//	@Summary		Restore backup
//	@Schemes
//	@Description	Restore a complete backup of the system including users, authentication, domains, zones, providers, and sessions.
//	@Tags			backup
//	@Accept			json
//	@Produce		json
//	@Param			body	body		string	true	"Backup data"
//	@Success		200		{boolean}	true
//	@Failure		400		{object}	happydns.ErrorResponse	"Invalid input"
//	@Failure		500		{object}	happydns.ErrorResponse	"Restore errors"
//	@Router			/backup.json [put]
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
