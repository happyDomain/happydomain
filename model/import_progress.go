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

package happydns

import (
	"context"
	"sync"
	"time"
)

// ImportStep is the step an import of a zone from its provider is at.
type ImportStep string

const (
	// ImportStepRetrieving is the provider retrieving the zone, telling
	// nothing more about it.
	ImportStepRetrieving ImportStep = "retrieving"
	// ImportStepLocating is the discovery finding the authoritative
	// servers of the domain.
	ImportStepLocating ImportStep = "locating"
	// ImportStepTransfer is the discovery asking a technique for the whole
	// zone, such as a zone transfer.
	ImportStepTransfer ImportStep = "transfer"
	// ImportStepCollecting is the discovery collecting the names to
	// explore from a technique.
	ImportStepCollecting ImportStep = "collecting"
	// ImportStepExploring is the discovery asking about each name.
	ImportStepExploring ImportStep = "exploring"
	// ImportStepFollowing is the discovery following the in-zone targets
	// of the answers.
	ImportStepFollowing ImportStep = "following"
	// ImportStepImporting is the zone retrieved being analysed into
	// services.
	ImportStepImporting ImportStep = "importing"
)

// ImportRetrievedPercent is the share of the whole import, in percent, the
// retrieval of the zone spans: its analysis into services spans the rest.
const ImportRetrievedPercent = 95

// ImportProgressSnapshot tells how far an import of a zone is.
type ImportProgressSnapshot struct {
	Step ImportStep `json:"step"`
	// Source is the technique the step is about, if any.
	Source string `json:"source,omitempty"`
	// Done is the number of names explored.
	Done int64 `json:"done"`
	// Total is the number of names to explore, 0 when unknown.
	Total int64 `json:"total"`
	// Records is the number of records found.
	Records int64 `json:"records"`
	// Queries is the number of DNS questions asked.
	Queries int64 `json:"queries"`
	// Percent is how far the whole import is, from 0 to 100. It never
	// goes back, but jumps over the steps that are skipped or tell
	// nothing of their own progress.
	Percent int `json:"percent"`
	// StepEnd is the percent the current step ends at, for the progress
	// to creep towards while the step tells nothing.
	StepEnd   int       `json:"step_end"`
	StartedAt time.Time `json:"started_at"`
}

// ImportProgress records how far an import of a zone is, for the user
// waiting for it. Its methods are safe for concurrent use, and do nothing on
// a nil ImportProgress, so the code reporting does not have to check
// whether anyone listens.
type ImportProgress struct {
	mu sync.Mutex
	s  ImportProgressSnapshot

	// from and to bound the share of the whole import the current step
	// spans, in percent.
	from, to float64
	// percent is how far the whole import is, unrounded.
	percent float64
}

// NewImportProgress returns the progress of an import starting now.
func NewImportProgress() *ImportProgress {
	return &ImportProgress{
		s: ImportProgressSnapshot{
			Step:      ImportStepRetrieving,
			StartedAt: time.Now(),
		},
	}
}

// SetStep tells the import reached step, about the technique source if
// not empty, which spans from from to to percent of the whole import.
func (p *ImportProgress) SetStep(step ImportStep, source string, from, to float64) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.s.Step = step
	p.s.Source = source
	p.from, p.to = from, to
	p.s.StepEnd = int(to)
	p.raise(from)
}

// Advance tells the current step is done for the fraction, from 0 to 1.
func (p *ImportProgress) Advance(fraction float64) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.raise(p.from + (p.to-p.from)*min(max(fraction, 0), 1))
}

// raise brings the progress of the whole import to percent, unless it is
// already further. p.mu must be held.
func (p *ImportProgress) raise(percent float64) {
	if percent > p.percent {
		p.percent = min(percent, 100)
		p.s.Percent = int(p.percent)
	}
}

// AddTotal adds n names to explore.
func (p *ImportProgress) AddTotal(n int64) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.s.Total += n
}

// AddDone adds n names explored.
func (p *ImportProgress) AddDone(n int64) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.s.Done += n
}

// SetCounts tells the numbers of records found and questions asked so far.
func (p *ImportProgress) SetCounts(records, queries int64) {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.s.Records = records
	p.s.Queries = queries
}

// Snapshot returns how far the import is now.
func (p *ImportProgress) Snapshot() ImportProgressSnapshot {
	if p == nil {
		return ImportProgressSnapshot{}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.s
}

type importProgressKey struct{}

// WithImportProgress returns a copy of ctx carrying p, for the code
// retrieving the zone to report on, without its interfaces knowing.
func WithImportProgress(ctx context.Context, p *ImportProgress) context.Context {
	return context.WithValue(ctx, importProgressKey{}, p)
}

// ImportProgressFrom returns the progress ctx carries, nil if none.
func ImportProgressFrom(ctx context.Context) *ImportProgress {
	p, _ := ctx.Value(importProgressKey{}).(*ImportProgress)
	return p
}
