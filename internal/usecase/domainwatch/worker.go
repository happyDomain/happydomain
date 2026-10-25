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

package domainwatch

import (
	"context"
	"errors"
	"log"
	"time"

	"git.happydns.org/happyDomain/internal/periodic"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/pkg/domaininfo"
)

const (
	// workerInterval is how often the worker looks for due watches; a
	// lookup happens at most this late.
	workerInterval = 15 * time.Minute
	// lookupConcurrency is how many lookups the worker runs at once.
	lookupConcurrency = 4
)

// Worker looks the watched domains up when they are due, in the background.
type Worker struct {
	service *Service
	runner  periodic.Runner
}

// NewWorker builds the background worker of the service.
func NewWorker(service *Service) *Worker {
	wk := &Worker{service: service}
	wk.runner = periodic.Runner{
		Interval:   workerInterval,
		RunAtStart: true,
		Task:       func(ctx context.Context) { wk.RunOnce(ctx) },
	}
	return wk
}

func (wk *Worker) Start(ctx context.Context) { wk.runner.Start(ctx) }
func (wk *Worker) Stop()                     { wk.runner.Stop() }

// RunOnce looks up every watched domain that is due, and returns how many
// watches it checked.
func (wk *Worker) RunOnce(ctx context.Context) int {
	s := wk.service

	iter, err := s.store.ListAllDomainWatches()
	if err != nil {
		log.Printf("DomainWatch: unable to list the watched domains: %v", err)
		return 0
	}

	now := s.nowFn()
	var due []*happydns.DomainWatch
	var names []string
	for iter.Next() {
		w := iter.Item()
		if !w.NextCheck.After(now) {
			due = append(due, w)
			names = append(names, w.DomainName)
		}
	}
	if err := iter.Err(); err != nil {
		log.Printf("DomainWatch: unable to list every watched domain: %v", err)
	}
	iter.Close()

	if len(due) == 0 {
		return 0
	}

	// Several users watching the same domain share its lookup.
	results := domaininfo.LookupMany(ctx, func(ctx context.Context, name string) (*domaininfo.DomainInfo, error) {
		ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
		defer cancel()
		return s.lookup(ctx, name)
	}, names, lookupConcurrency)

	checked := 0
	for _, w := range due {
		r, ok := results[w.DomainName]
		// Stopped before looking it up, or during the lookup: the failure
		// tells nothing about the domain, which stays due.
		if !ok || r.Err != nil && ctx.Err() != nil {
			continue
		}
		if _, err := s.record(w.Id, r); errors.Is(err, happydns.ErrDomainWatchNotFound) {
			// Removed during the lookup.
			continue
		} else if err != nil {
			log.Printf("DomainWatch: unable to record the check of %s: %v", w.DomainName, err)
			continue
		}
		checked++
	}
	return checked
}
