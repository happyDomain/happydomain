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

func (u *Usecase) backupOneUser(user *happydns.User, ret *happydns.Backup) {
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
			u.backupOneUser(iter.Item(), &ret)
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

	// Observation snapshots.
	if snapIter, err := u.store.ListAllSnapshots(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve ObservationSnapshots: %s", err.Error()))
	} else {
		defer snapIter.Close()
		for snapIter.Next() {
			snap := snapIter.Item()
			ret.ObservationSnapshots = append(ret.ObservationSnapshots, snap)
		}
	}

	return ret
}

func (u *Usecase) BackupUser(user *happydns.User) happydns.Backup {
	uid := user.Id.String()

	ret := happydns.Backup{
		Version:     u.store.SchemaVersion(),
		DomainsLogs: map[string][]*happydns.DomainLog{},
	}

	// UserAuth for this user — strip credentials before export.
	if ua, err := u.store.GetAuthUser(user.Id); err == nil {
		ua.Password = nil
		ua.PasswordRecoveryKey = nil
		ret.UsersAuth = append(ret.UsersAuth, ua)
	} else if !errors.Is(err, happydns.ErrAuthUserNotFound) {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve UserAuth: %s", err.Error()))
	}

	u.backupOneUser(user, &ret)

	// Strip session IDs — they are live credentials, not portable data.
	for _, s := range ret.Sessions {
		s.Id = ""
	}

	// Checker configurations scoped to this user.
	if cfgIter, err := u.store.ListAllCheckerConfigurations(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckerConfigurations: %s", err.Error()))
	} else {
		defer cfgIter.Close()
		for cfgIter.Next() {
			cfg := cfgIter.Item()
			if cfg.UserId != nil && cfg.UserId.Equals(user.Id) {
				ret.CheckerConfigurations = append(ret.CheckerConfigurations, cfg)
			}
		}
	}

	// Check plans scoped to this user (indexed lookup).
	if plans, err := u.store.ListCheckPlansByUser(user.Id); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckPlans: %s", err.Error()))
	} else {
		ret.CheckPlans = append(ret.CheckPlans, plans...)
	}

	// Check evaluations scoped to this user.
	if evalIter, err := u.store.ListAllEvaluations(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve CheckEvaluations: %s", err.Error()))
	} else {
		defer evalIter.Close()
		for evalIter.Next() {
			eval := evalIter.Item()
			if eval.Target.UserId == uid {
				ret.CheckEvaluations = append(ret.CheckEvaluations, eval)
			}
		}
	}

	// Executions scoped to this user (indexed lookup, no limit).
	if execs, err := u.store.ListExecutionsByUser(user.Id, 0, nil); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve Executions: %s", err.Error()))
	} else {
		ret.Executions = append(ret.Executions, execs...)
	}

	// Discovery entries scoped to this user.
	if entryIter, err := u.store.ListAllDiscoveryEntries(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve DiscoveryEntries: %s", err.Error()))
	} else {
		defer entryIter.Close()
		for entryIter.Next() {
			entry := entryIter.Item()
			if entry.Target.UserId == uid {
				ret.DiscoveryEntries = append(ret.DiscoveryEntries, entry)
			}
		}
	}

	// Discovery observation refs scoped to this user.
	if refIter, err := u.store.ListAllDiscoveryObservationRefs(); err != nil {
		ret.Errors = append(ret.Errors, fmt.Sprintf("unable to retrieve DiscoveryObservationRefs: %s", err.Error()))
	} else {
		defer refIter.Close()
		for refIter.Next() {
			ref := refIter.Item()
			if ref.Target.UserId == uid {
				ret.DiscoveryObservationRefs = append(ret.DiscoveryObservationRefs, ref)
			}
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

	// Observation snapshots (restored last; rebuild cache from snapshot data).
	for _, snap := range backup.ObservationSnapshots {
		if err := u.store.RestoreSnapshot(snap); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for key := range snap.Data {
			existing, _ := u.store.GetCachedObservation(snap.Target, key)
			if existing == nil || snap.CollectedAt.After(existing.CollectedAt) {
				if err := u.store.PutCachedObservation(snap.Target, key, &happydns.ObservationCacheEntry{
					SnapshotID:  snap.Id,
					CollectedAt: snap.CollectedAt,
				}); err != nil {
					errs = errors.Join(errs, err)
				}
			}
		}
	}

	return errs
}
