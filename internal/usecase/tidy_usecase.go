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

// iterateTidy drives an iterator using NextWithError so Tidy can decide
// whether to delete undecodable records (via DropItem) or just log them.
// handle is only invoked for successfully decoded items.
func iterateTidy[T any](iter happydns.Iterator[T], dropInvalid bool, handle func(*T) error) error {
	for iter.NextWithError() {
		item := iter.Item()
		if item == nil {
			key := iter.Key()
			log.Printf("KVIterator: error decoding item at key %q: %s", key, iter.Err())
			if dropInvalid {
				if err := iter.DropItem(); err != nil {
					log.Printf("KVIterator: failed to delete invalid item at key %q: %s", key, err)
				} else {
					log.Printf("KVIterator: dropped invalid item at key %q", key)
				}
			}
			continue
		}
		if err := handle(item); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (tu *tidyUpUsecase) TidyAll(dropInvalid bool) error {
	for _, tidy := range []func(bool) error{
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
		tu.TidyObservationCache,
	} {
		if err := tidy(dropInvalid); err != nil {
			return err
		}
	}
	return nil
}

func (tu *tidyUpUsecase) TidyAuthUsers(dropInvalid bool) error {
	iter, err := tu.store.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(userAuth *happydns.UserAuth) error {
		_, err := tu.store.GetUser(userAuth.Id)
		if errors.Is(err, happydns.ErrUserNotFound) && time.Since(userAuth.CreatedAt) > 24*time.Hour {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan authuser (user %s not found): %v\n", userAuth.Id.String(), userAuth)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyCheckEvaluations(dropInvalid bool) error {
	iter, err := tu.store.ListAllEvaluations()
	if err != nil {
		return err
	}
	defer iter.Close()

	err = iterateTidy(iter, dropInvalid, func(eval *happydns.CheckEvaluation) error {
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
			if _, err := tu.store.GetCheckPlan(*eval.PlanID); errors.Is(err, happydns.ErrCheckPlanNotFound) {
				log.Printf("Deleting orphan check evaluation (plan %s not found): %s\n", eval.PlanID.String(), eval.Id.String())
				drop = true
			}
		}

		if drop {
			if err := tu.store.DeleteEvaluation(eval.Id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return tu.store.TidyEvaluationIndexes()
}

func (tu *tidyUpUsecase) TidyCheckPlans(dropInvalid bool) error {
	iter, err := tu.store.ListAllCheckPlans()
	if err != nil {
		return err
	}
	defer iter.Close()

	err = iterateTidy(iter, dropInvalid, func(plan *happydns.CheckPlan) error {
		if plan.Target.UserId != "" {
			userId, err := happydns.NewIdentifierFromString(plan.Target.UserId)
			if err == nil {
				_, err = tu.store.GetUser(userId)
				if errors.Is(err, happydns.ErrUserNotFound) {
					log.Printf("Deleting orphan check plan (user %s not found): %s\n", plan.Target.UserId, plan.Id.String())
					_ = tu.store.DeleteEvaluationsByChecker(plan.CheckerID, plan.Target)
					_ = tu.store.DeleteExecutionsByChecker(plan.CheckerID, plan.Target)
					return iter.DropItem()
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
					return iter.DropItem()
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return tu.store.TidyCheckPlanIndexes()
}

func (tu *tidyUpUsecase) TidyCheckerConfigurations(dropInvalid bool) error {
	iter, err := tu.store.ListAllCheckerConfigurations()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(cfg *happydns.CheckerOptionsPositional) error {
		if cfg.UserId != nil {
			if _, err := tu.store.GetUser(*cfg.UserId); errors.Is(err, happydns.ErrUserNotFound) {
				log.Printf("Deleting orphan checker configuration (user %s not found): %s\n", cfg.UserId.String(), cfg.CheckName)
				return iter.DropItem()
			} else if err != nil {
				return err
			}
		}

		if cfg.DomainId != nil {
			domain, err := tu.store.GetDomain(*cfg.DomainId)
			if errors.Is(err, happydns.ErrDomainNotFound) {
				log.Printf("Deleting orphan checker configuration (domain %s not found): %s\n", cfg.DomainId.String(), cfg.CheckName)
				return iter.DropItem()
			} else if err != nil {
				return err
			}

			if cfg.ServiceId != nil && len(domain.ZoneHistory) > 0 {
				// Check both the WIP zone ([0]) and the latest published
				// zone ([1]) so we keep configs for services that are
				// either being worked on or currently live.
				found := false
				for _, idx := range []int{0, 1} {
					if idx >= len(domain.ZoneHistory) {
						break
					}
					zone, err := tu.store.GetZone(domain.ZoneHistory[idx])
					if err != nil {
						return err
					}
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
					if found {
						break
					}
				}
				if !found {
					log.Printf("Deleting orphan checker configuration (service %s not found in domain %s): %s\n", cfg.ServiceId.String(), cfg.DomainId.String(), cfg.CheckName)
					return iter.DropItem()
				}
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyExecutions(dropInvalid bool) error {
	iter, err := tu.store.ListAllExecutions()
	if err != nil {
		return err
	}
	defer iter.Close()

	err = iterateTidy(iter, dropInvalid, func(exec *happydns.Execution) error {
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
			if _, err := tu.store.GetCheckPlan(*exec.PlanID); errors.Is(err, happydns.ErrCheckPlanNotFound) {
				log.Printf("Deleting orphan execution (plan %s not found): %s\n", exec.PlanID.String(), exec.Id.String())
				drop = true
			}
		}

		if drop {
			if err := tu.store.DeleteExecution(exec.Id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return tu.store.TidyExecutionIndexes()
}

func (tu *tidyUpUsecase) TidyObservationCache(dropInvalid bool) error {
	iter, err := tu.store.ListAllCachedObservations()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(entry *happydns.ObservationCacheEntry) error {
		if _, err := tu.store.GetSnapshot(entry.SnapshotID); errors.Is(err, happydns.ErrSnapshotNotFound) {
			log.Printf("Deleting stale observation cache entry (snapshot %s not found)\n", entry.SnapshotID.String())
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyDomains(dropInvalid bool) error {
	iter, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(domain *happydns.Domain) error {
		if _, err := tu.store.GetUser(domain.Owner); errors.Is(err, happydns.ErrUserNotFound) {
			// Drop domain of unexistant users
			log.Printf("Deleting orphan domain (user %s not found): %v\n", domain.Owner.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}

		if _, err := tu.store.GetProvider(domain.ProviderId); errors.Is(err, happydns.ErrProviderNotFound) {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain (provider %s not found): %v\n", domain.ProviderId.String(), domain)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyDomainLogs(dropInvalid bool) error {
	iter, err := tu.store.ListAllDomainLogs()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(l *happydns.DomainLogWithDomainId) error {
		if _, err := tu.store.GetDomain(l.DomainId); errors.Is(err, happydns.ErrDomainNotFound) {
			// Drop domain of unexistant provider
			log.Printf("Deleting orphan domain log (domain %s not found): %v\n", l.DomainId.String(), l)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyProviders(dropInvalid bool) error {
	iter, err := tu.store.ListAllProviders()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(prvd *happydns.ProviderMessage) error {
		_, err := tu.store.GetUser(prvd.Owner)
		if errors.Is(err, happydns.ErrUserNotFound) {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan provider (user %s not found): %v\n", prvd.Owner.String(), prvd)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidySessions(dropInvalid bool) error {
	iter, err := tu.store.ListAllSessions()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(session *happydns.Session) error {
		_, err := tu.store.GetUser(session.IdUser)
		if errors.Is(err, happydns.ErrUserNotFound) {
			// Drop session from unexistant users
			log.Printf("Deleting orphan session (user %s not found): %v\n", session.IdUser.String(), session)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidySnapshots(dropInvalid bool) error {
	// Collect all snapshot IDs referenced by evaluations.
	evalIter, err := tu.store.ListAllEvaluations()
	if err != nil {
		return err
	}
	defer evalIter.Close()

	referencedSnapshots := make(map[string]struct{})
	if err = iterateTidy(evalIter, dropInvalid, func(eval *happydns.CheckEvaluation) error {
		if !eval.SnapshotID.IsEmpty() {
			referencedSnapshots[eval.SnapshotID.String()] = struct{}{}
		}
		return nil
	}); err != nil {
		return err
	}

	// Delete snapshots not referenced by any evaluation.
	iter, err := tu.store.ListAllSnapshots()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(snap *happydns.ObservationSnapshot) error {
		if _, ok := referencedSnapshots[snap.Id.String()]; !ok {
			log.Printf("Deleting orphan snapshot: %s\n", snap.Id.String())
			if err := iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyUsers(dropInvalid bool) error {
	iter, err := tu.store.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(authUser *happydns.UserAuth) error {
		if authUser.EmailVerification == nil && authUser.LastLoggedIn == nil && time.Since(authUser.CreatedAt) > 7*24*time.Hour {
			log.Printf("Deleting user with unverified email and no login (created %s): %s\n", authUser.CreatedAt.Format(time.RFC3339), authUser.Email)
			if err := tu.store.DeleteUser(authUser.Id); err != nil && !errors.Is(err, happydns.ErrUserNotFound) {
				return err
			}
			if err := iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyZones(dropInvalid bool) error {
	iterdn, err := tu.store.ListAllDomains()
	if err != nil {
		return err
	}
	defer iterdn.Close()

	var referencedZones []happydns.Identifier
	if err = iterateTidy(iterdn, dropInvalid, func(domain *happydns.Domain) error {
		referencedZones = append(referencedZones, domain.ZoneHistory...)
		return nil
	}); err != nil {
		return err
	}

	iter, err := tu.store.ListAllZones()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(zone *happydns.ZoneMessage) error {
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
			if err := iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}
