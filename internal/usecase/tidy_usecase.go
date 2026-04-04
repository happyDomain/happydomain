// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package usecase

import (
	"errors"
	"log"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type tidyUpUsecase struct {
	store storage.Storage
}

func NewTidyUpUsecase(store storage.Storage) happydns.TidyUpUseCase {
	return &tidyUpUsecase{
		store: store,
	}
}

func (tu *tidyUpUsecase) TidyAll() error {
	for _, tidy := range []func() error{
		tu.TidySessions,
		tu.TidyAuthUsers,
		tu.TidyUsers,
		tu.TidyProviders,
		tu.TidyDomains,
		tu.TidyZones,
		tu.TidyDomainLogs,
		tu.TidyCheckPlans,
		tu.TidyCheckerConfigurations,
		tu.TidyExecutions,
		tu.TidyCheckEvaluations,
		tu.TidySnapshots,
	} {
		if err := tidy(); err != nil {
			return err
		}
	}
	return nil
}

func (tu *tidyUpUsecase) TidyAuthUsers() error {
	iter, err := tu.store.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		userAuth := iter.Item()

		_, err = tu.store.GetUser(userAuth.Id)
		if errors.Is(err, happydns.ErrUserNotFound) && time.Since(userAuth.CreatedAt) > 24*time.Hour {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan authuser (user %s not found): %v\n", userAuth.Id.String(), userAuth)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidyCheckEvaluations() error {
	iter, err := tu.store.ListAllEvaluations()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		eval := iter.Item()

		drop := false

		if eval.Target.UserId != "" {
			userId, err := happydns.NewIdentifierFromString(eval.Target.UserId)
			if err == nil {
				if _, err = tu.store.GetUser(userId); errors.Is(err, happydns.ErrUserNotFound) {
					log.Printf("Deleting orphan check evaluation (user %s not found): %s\n", eval.Target.UserId, eval.Id.String())
					drop = true
				}
			}
		}

		if !drop && eval.Target.DomainId != "" {
			domainId, err := happydns.NewIdentifierFromString(eval.Target.DomainId)
			if err == nil {
				if _, err = tu.store.GetDomain(domainId); errors.Is(err, happydns.ErrDomainNotFound) {
					log.Printf("Deleting orphan check evaluation (domain %s not found): %s\n", eval.Target.DomainId, eval.Id.String())
					drop = true
				}
			}
		}

		if !drop && eval.PlanID != nil {
			if _, err = tu.store.GetCheckPlan(*eval.PlanID); errors.Is(err, happydns.ErrCheckPlanNotFound) {
				log.Printf("Deleting orphan check evaluation (plan %s not found): %s\n", eval.PlanID.String(), eval.Id.String())
				drop = true
			}
		}

		if drop {
			if err = tu.store.DeleteEvaluation(eval.Id); err != nil {
				return err
			}
		}
	}

	if err = iter.Err(); err != nil {
		return err
	}

	return tu.store.TidyEvaluationIndexes()
}

func (tu *tidyUpUsecase) TidyCheckPlans() error {
	iter, err := tu.store.ListAllCheckPlans()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		plan := iter.Item()

		if plan.Target.UserId != "" {
			userId, err := happydns.NewIdentifierFromString(plan.Target.UserId)
			if err == nil {
				_, err = tu.store.GetUser(userId)
				if errors.Is(err, happydns.ErrUserNotFound) {
					log.Printf("Deleting orphan check plan (user %s not found): %s\n", plan.Target.UserId, plan.Id.String())
					_ = tu.store.DeleteEvaluationsByChecker(plan.CheckerID, plan.Target)
					_ = tu.store.DeleteExecutionsByChecker(plan.CheckerID, plan.Target)
					if err = iter.DropItem(); err != nil {
						return err
					}
					continue
				}
			}
		}

		if plan.Target.DomainId != "" {
			domainId, err := happydns.NewIdentifierFromString(plan.Target.DomainId)
			if err == nil {
				_, err = tu.store.GetDomain(domainId)
				if errors.Is(err, happydns.ErrDomainNotFound) {
					log.Printf("Deleting orphan check plan (domain %s not found): %s\n", plan.Target.DomainId, plan.Id.String())
					_ = tu.store.DeleteEvaluationsByChecker(plan.CheckerID, plan.Target)
					_ = tu.store.DeleteExecutionsByChecker(plan.CheckerID, plan.Target)
					if err = iter.DropItem(); err != nil {
						return err
					}
					continue
				}
			}
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return tu.store.TidyCheckPlanIndexes()
}

func (tu *tidyUpUsecase) TidyCheckerConfigurations() error {
	iter, err := tu.store.ListAllCheckerConfigurations()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		cfg := iter.Item()

		if cfg.UserId != nil {
			if _, err = tu.store.GetUser(*cfg.UserId); errors.Is(err, happydns.ErrUserNotFound) {
				log.Printf("Deleting orphan checker configuration (user %s not found): %s\n", cfg.UserId.String(), cfg.CheckName)
				if err = iter.DropItem(); err != nil {
					return err
				}
				continue
			} else if err != nil {
				return err
			}
		}

		if cfg.DomainId != nil {
			domain, err := tu.store.GetDomain(*cfg.DomainId)
			if errors.Is(err, happydns.ErrDomainNotFound) {
				log.Printf("Deleting orphan checker configuration (domain %s not found): %s\n", cfg.DomainId.String(), cfg.CheckName)
				if err = iter.DropItem(); err != nil {
					return err
				}
				continue
			} else if err != nil {
				return err
			}

			if cfg.ServiceId != nil && len(domain.ZoneHistory) > 0 {
				zone, err := tu.store.GetZone(domain.ZoneHistory[len(domain.ZoneHistory)-1])
				if err != nil {
					return err
				}
				found := false
				for _, svcs := range zone.Services {
					for _, svc := range svcs {
						if svc.Id.Equals(*cfg.ServiceId) {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					log.Printf("Deleting orphan checker configuration (service %s not found in domain %s): %s\n", cfg.ServiceId.String(), cfg.DomainId.String(), cfg.CheckName)
					if err = iter.DropItem(); err != nil {
						return err
					}
					continue
				}
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidyExecutions() error {
	iter, err := tu.store.ListAllExecutions()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		exec := iter.Item()

		drop := false

		if exec.Target.UserId != "" {
			userId, err := happydns.NewIdentifierFromString(exec.Target.UserId)
			if err == nil {
				if _, err = tu.store.GetUser(userId); errors.Is(err, happydns.ErrUserNotFound) {
					log.Printf("Deleting orphan execution (user %s not found): %s\n", exec.Target.UserId, exec.Id.String())
					drop = true
				}
			}
		}

		if !drop && exec.Target.DomainId != "" {
			domainId, err := happydns.NewIdentifierFromString(exec.Target.DomainId)
			if err == nil {
				if _, err = tu.store.GetDomain(domainId); errors.Is(err, happydns.ErrDomainNotFound) {
					log.Printf("Deleting orphan execution (domain %s not found): %s\n", exec.Target.DomainId, exec.Id.String())
					drop = true
				}
			}
		}

		if !drop && exec.PlanID != nil {
			if _, err = tu.store.GetCheckPlan(*exec.PlanID); errors.Is(err, happydns.ErrCheckPlanNotFound) {
				log.Printf("Deleting orphan execution (plan %s not found): %s\n", exec.PlanID.String(), exec.Id.String())
				drop = true
			}
		}

		if drop {
			if err = tu.store.DeleteExecution(exec.Id); err != nil {
				return err
			}
		}
	}

	if err = iter.Err(); err != nil {
		return err
	}

	return tu.store.TidyExecutionIndexes()
}

func (tu *tidyUpUsecase) TidyDomains() error {
	iter, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		domain := iter.Item()

		if _, err = tu.store.GetUser(domain.Owner); errors.Is(err, happydns.ErrUserNotFound) {
			// Drop domain of unexistant users
			log.Printf("Deleting orphan domain (user %s not found): %v\n", domain.Owner.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}

		if _, err = tu.store.GetProvider(domain.ProviderId); errors.Is(err, happydns.ErrProviderNotFound) {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain (provider %s not found): %v\n", domain.ProviderId.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidyDomainLogs() error {
	iter, err := tu.store.ListAllDomainLogs()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		l := iter.Item()

		if _, err = tu.store.GetDomain(l.DomainId); errors.Is(err, happydns.ErrDomainNotFound) {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain log (domain %s not found): %v\n", l.DomainId.String(), l)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidyProviders() error {
	iter, err := tu.store.ListAllProviders()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		prvd := iter.Item()

		_, err = tu.store.GetUser(prvd.Owner)
		if errors.Is(err, happydns.ErrUserNotFound) {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan provider (user %s not found): %v\n", prvd.Owner.String(), prvd)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidySessions() error {
	iter, err := tu.store.ListAllSessions()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		session := iter.Item()

		_, err = tu.store.GetUser(session.IdUser)
		if errors.Is(err, happydns.ErrUserNotFound) {
			// Drop session from unexistant users
			log.Printf("Deleting orphan session (user %s not found): %v\n", session.IdUser.String(), session)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidySnapshots() error {
	// Collect all snapshot IDs referenced by evaluations.
	evalIter, err := tu.store.ListAllEvaluations()
	if err != nil {
		return err
	}
	defer evalIter.Close()

	referencedSnapshots := make(map[string]struct{})
	for evalIter.Next() {
		eval := evalIter.Item()
		if !eval.SnapshotID.IsEmpty() {
			referencedSnapshots[eval.SnapshotID.String()] = struct{}{}
		}
	}
	if err = evalIter.Err(); err != nil {
		return err
	}

	// Delete snapshots not referenced by any evaluation.
	iter, err := tu.store.ListAllSnapshots()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		snap := iter.Item()
		if _, ok := referencedSnapshots[snap.Id.String()]; !ok {
			log.Printf("Deleting orphan snapshot: %s\n", snap.Id.String())
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidyUsers() error {
	iter, err := tu.store.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		authUser := iter.Item()

		if authUser.EmailVerification == nil && authUser.LastLoggedIn == nil && time.Since(authUser.CreatedAt) > 7*24*time.Hour {
			log.Printf("Deleting user with unverified email and no login (created %s): %s\n", authUser.CreatedAt.Format(time.RFC3339), authUser.Email)
			if err = tu.store.DeleteUser(authUser.Id); err != nil && !errors.Is(err, happydns.ErrUserNotFound) {
				return err
			}
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}

func (tu *tidyUpUsecase) TidyZones() error {
	iterdn, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iterdn.Close()

	var referencedZones []happydns.Identifier

	for iterdn.Next() {
		domain := iterdn.Item()
		for _, zh := range domain.ZoneHistory {
			referencedZones = append(referencedZones, zh)
		}
	}

	if err = iterdn.Err(); err != nil {
		return err
	}

	iter, err := tu.store.ListAllZones()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		zone := iter.Item()

		foundZone := false
		for _, zid := range referencedZones {
			if zid.Equals(zone.Id) {
				foundZone = true
				break
			}
		}

		if !foundZone {
			// Drop orphan zones
			log.Printf("Deleting orphan zone: %s\n", zone.Id.String())
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
	}

	return iter.Err()
}
