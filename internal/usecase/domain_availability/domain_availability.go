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

package domain_availability

import (
	"context"
	"fmt"
	"sync"

	"git.happydns.org/happyDomain/model"
)

type Service struct {
	store             DomainAvailabilityWatchStorage
	schedulerNotifier happydns.SchedulerWatchNotifier

	// createMu serializes CreateDomainAvailabilityWatch calls so the
	// list-then-create duplicate check below is not racy under concurrent
	// requests from the same process.
	createMu sync.Mutex
}

func NewService(store DomainAvailabilityWatchStorage) *Service {
	return &Service{store: store}
}

// SetSchedulerNotifier sets the optional scheduler notifier for incremental
// queue updates on watch creation/deletion.
func (s *Service) SetSchedulerNotifier(notifier happydns.SchedulerWatchNotifier) {
	s.schedulerNotifier = notifier
}

func (s *Service) CreateDomainAvailabilityWatch(ctx context.Context, user *happydns.User, input *happydns.DomainAvailabilityWatchCreationInput) (*happydns.DomainAvailabilityWatch, error) {
	watch, err := happydns.NewDomainAvailabilityWatch(user, input.DomainName)
	if err != nil {
		return nil, happydns.ValidationError{Msg: err.Error()}
	}
	watch.Interval = input.Interval

	// Prevent the same user from watching the same domain twice. Serialize
	// against other calls in this process so two concurrent requests for the
	// same domain cannot both pass the check before either has written.
	s.createMu.Lock()
	defer s.createMu.Unlock()

	exists, err := s.store.ExistsDomainAvailabilityWatch(user.Id, watch.DomainName)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to ExistsDomainAvailabilityWatch: %w", err),
			UserMessage: "Sorry, we are unable to create your availability watch now.",
		}
	}
	if exists {
		return nil, happydns.ValidationError{Msg: "you are already watching this domain."}
	}

	if err := s.store.CreateDomainAvailabilityWatch(watch); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateDomainAvailabilityWatch: %w", err),
			UserMessage: "Sorry, we are unable to create your availability watch now.",
		}
	}

	if s.schedulerNotifier != nil {
		s.schedulerNotifier.NotifyWatchChange(watch)
	}

	return watch, nil
}

func (s *Service) GetUserDomainAvailabilityWatch(user *happydns.User, id happydns.Identifier) (*happydns.DomainAvailabilityWatch, error) {
	watch, err := s.store.GetDomainAvailabilityWatch(id)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(watch.Owner) {
		return nil, happydns.ErrDomainAvailabilityWatchNotFound
	}

	return watch, nil
}

func (s *Service) ListUserDomainAvailabilityWatches(user *happydns.User) ([]*happydns.DomainAvailabilityWatch, error) {
	watches, err := s.store.ListDomainAvailabilityWatches(user)
	if err != nil {
		return nil, fmt.Errorf("unable to ListUserDomainAvailabilityWatches: %w", err)
	}

	if len(watches) == 0 {
		return []*happydns.DomainAvailabilityWatch{}, nil
	}

	return watches, nil
}

func (s *Service) DeleteDomainAvailabilityWatch(user *happydns.User, id happydns.Identifier) error {
	// Ensure the watch exists and is owned by the caller before deleting.
	if _, err := s.GetUserDomainAvailabilityWatch(user, id); err != nil {
		return err
	}

	if err := s.store.DeleteDomainAvailabilityWatch(id); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteDomainAvailabilityWatch: %w", err),
			UserMessage: "Sorry, we are unable to delete your availability watch now.",
		}
	}

	if s.schedulerNotifier != nil {
		s.schedulerNotifier.NotifyWatchRemoved(id)
	}

	return nil
}
