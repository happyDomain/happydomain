// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

package backup

import (
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/internal/storage"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	happydns "git.happydns.org/happyDomain/model"
)

type Usecase struct {
	store storage.Storage
}

func NewUsecase(store storage.Storage) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) Backup() happydns.Backup {
	ret := happydns.Backup{
		Version:     u.store.SchemaVersion(),
		DomainsLogs: map[string][]*happydns.DomainLog{},
	}

	// UserAuth
	uai, err := u.store.ListAllAuthUsers()
	if err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve AuthUsers: %s", err.Error()))
	} else {
		defer uai.Close()
		for uai.Next() {
			ret.UsersAuth = append(ret.UsersAuth, uai.Item())
		}
	}

	// Users
	iter, err := u.store.ListAllUsers()
	if err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Users: %s", err.Error()))
	} else {
		defer iter.Close()

		for iter.Next() {
			user := iter.Item()

			ret.Users = append(ret.Users, user)

			// Domains
			ds, err := u.store.ListDomains(user)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Domain names of %s (%s): %s", user.Id.String(), user.Email, err.Error()))
			} else {
				ret.Domains = append(ret.Domains, ds...)

				for _, dn := range ds {
					// Domain logs
					ls, logErr := u.store.ListDomainLogs(dn)
					if logErr != nil {
						ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve domain's logs %s/%s (%s): %s", user.Id.String(), dn.Id.String(), dn.DomainName, logErr.Error()))
					} else {
						ret.DomainsLogs[dn.Id.String()] = ls
					}

					// Zones
					for _, zid := range dn.ZoneHistory {
						z, zoneErr := u.store.GetZone(zid)
						if zoneErr != nil {
							ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve domain's zone %s/%s (%s): zoneid=%s: %s", user.Id.String(), dn.Id.String(), dn.DomainName, zid.String(), zoneErr.Error()))
						} else {
							ret.Zones = append(ret.Zones, z)
						}
					}
				}
			}

			// Providers
			ps, err := u.store.ListProviders(user)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Providers: %s", err.Error()))
			} else {
				ret.Providers = append(ret.Providers, ps...)
			}

			// Sessions
			ss, err := u.store.ListUserSessions(user.Id)
			if err != nil {
				ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Sessions: %s", err.Error()))
			} else {
				ret.Sessions = append(ret.Sessions, ss...)
			}
		}
	}

	// Checker configurations (positional, one entry per (checker, user?, domain?, service?)).
	if cfgIter, err := u.store.ListAllCheckerConfigurations(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckerConfigurations: %s", err.Error()))
	} else {
		defer cfgIter.Close()
		for cfgIter.Next() {
			ret.CheckerConfigurations = append(ret.CheckerConfigurations, cfgIter.Item())
		}
	}

	// Check plans.
	if planIter, err := u.store.ListAllCheckPlans(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckPlans: %s", err.Error()))
	} else {
		defer planIter.Close()
		for planIter.Next() {
			ret.CheckPlans = append(ret.CheckPlans, planIter.Item())
		}
	}

	// Check evaluations.
	if evalIter, err := u.store.ListAllEvaluations(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckEvaluations: %s", err.Error()))
	} else {
		defer evalIter.Close()
		for evalIter.Next() {
			ret.CheckEvaluations = append(ret.CheckEvaluations, evalIter.Item())
		}
	}

	// Executions.
	if execIter, err := u.store.ListAllExecutions(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Executions: %s", err.Error()))
	} else {
		defer execIter.Close()
		for execIter.Next() {
			ret.Executions = append(ret.Executions, execIter.Item())
		}
	}

	// Discovery entries.
	if entryIter, err := u.store.ListAllDiscoveryEntries(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve DiscoveryEntries: %s", err.Error()))
	} else {
		defer entryIter.Close()
		for entryIter.Next() {
			ret.DiscoveryEntries = append(ret.DiscoveryEntries, entryIter.Item())
		}
	}

	// Discovery observation refs.
	if refIter, err := u.store.ListAllDiscoveryObservationRefs(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve DiscoveryObservationRefs: %s", err.Error()))
	} else {
		defer refIter.Close()
		for refIter.Next() {
			ret.DiscoveryObservationRefs = append(ret.DiscoveryObservationRefs, refIter.Item())
		}
	}

	return ret
}

func (u *Usecase) Restore(backup *happydns.Backup) error {
	var errs error

	// UserAuth
	for _, ua := range backup.UsersAuth {
		errs = errors.Join(errs, u.store.UpdateAuthUser(ua))
	}

	// Users
	for _, user := range backup.Users {
		if err := u.store.CreateOrUpdateUser(user); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	// Providers
	for _, provider := range backup.Providers {
		p, err := providerUC.ParseProvider(provider)
		if err != nil {
			errs = errors.Join(errs, err)
		}

		errs = errors.Join(errs, u.store.UpdateProvider(p))
	}

	// Domains
	for _, domain := range backup.Domains {
		if err := u.store.UpdateDomain(domain); err != nil {
			errs = errors.Join(errs, err)
		} else {
			// Domain logs
			for _, l := range backup.DomainsLogs[domain.Id.String()] {
				errs = errors.Join(errs, u.store.UpdateDomainLog(domain, l))
			}
		}
	}

	// Zones
	for _, zmsg := range backup.Zones {
		zone, err := zoneUC.ParseZone(zmsg)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			errs = errors.Join(errs, u.store.UpdateZone(zone))
		}
	}

	// Sessions
	for _, session := range backup.Sessions {
		errs = errors.Join(errs, u.store.UpdateSession(session))
	}

	// Checker configurations.
	for _, cfg := range backup.CheckerConfigurations {
		if cfg == nil {
			continue
		}
		errs = errors.Join(errs, u.store.UpdateCheckerConfiguration(cfg.CheckName, cfg.UserId, cfg.DomainId, cfg.ServiceId, cfg.Options))
	}

	// Check plans.
	for _, plan := range backup.CheckPlans {
		errs = errors.Join(errs, u.store.RestoreCheckPlan(plan))
	}

	// Check evaluations (reference plans, restored above).
	for _, eval := range backup.CheckEvaluations {
		errs = errors.Join(errs, u.store.RestoreEvaluation(eval))
	}

	// Executions.
	for _, exec := range backup.Executions {
		errs = errors.Join(errs, u.store.RestoreExecution(exec))
	}

	// Discovery entries. Restored after snapshots (referenced indirectly via
	// target + producer, no FK), before observation refs which carry snapshot
	// pointers that must resolve at lookup time.
	for _, entry := range backup.DiscoveryEntries {
		errs = errors.Join(errs, u.store.RestoreDiscoveryEntry(entry))
	}

	// Discovery observation refs.
	for _, ref := range backup.DiscoveryObservationRefs {
		errs = errors.Join(errs, u.store.RestoreDiscoveryObservationRef(ref))
	}

	return errs
}
