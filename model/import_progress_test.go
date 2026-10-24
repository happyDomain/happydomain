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

package happydns_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

func TestImportProgressNil(t *testing.T) {
	var p *happydns.ImportProgress

	p.SetStep(happydns.ImportStepExploring, "ct", 10, 20)
	p.Advance(0.5)
	p.AddTotal(3)
	p.AddDone(1)
	p.SetCounts(4, 5)

	if s := p.Snapshot(); s != (happydns.ImportProgressSnapshot{}) {
		t.Errorf("the snapshot of a nil progress is %+v, want the zero value", s)
	}
}

func TestImportProgressContext(t *testing.T) {
	if p := happydns.ImportProgressFrom(context.Background()); p != nil {
		t.Errorf("a bare context holds the progress %p, want nil", p)
	}

	p := happydns.NewImportProgress()
	ctx := happydns.WithImportProgress(context.Background(), p)
	if got := happydns.ImportProgressFrom(ctx); got != p {
		t.Errorf("the context holds the progress %p, want %p", got, p)
	}
}

func TestImportProgressSnapshot(t *testing.T) {
	before := time.Now()
	p := happydns.NewImportProgress()

	s := p.Snapshot()
	if s.Step != happydns.ImportStepRetrieving {
		t.Errorf("a new progress is at step %q, want %q", s.Step, happydns.ImportStepRetrieving)
	}
	if s.StartedAt.Before(before) {
		t.Errorf("a new progress started at %s, before its creation at %s", s.StartedAt, before)
	}

	p.SetStep(happydns.ImportStepCollecting, "ct", 0, 0)
	p.AddTotal(10)
	p.AddTotal(5)
	p.AddDone(3)
	p.AddDone(4)
	p.SetCounts(8, 42)

	s = p.Snapshot()
	if s.Step != happydns.ImportStepCollecting || s.Source != "ct" {
		t.Errorf("the progress is at step %q, source %q, want %q, \"ct\"", s.Step, s.Source, happydns.ImportStepCollecting)
	}
	if s.Total != 15 || s.Done != 7 {
		t.Errorf("the progress is at %d of %d, want 7 of 15", s.Done, s.Total)
	}
	if s.Records != 8 || s.Queries != 42 {
		t.Errorf("the progress counts %d records and %d queries, want 8 and 42", s.Records, s.Queries)
	}

	// A step without source drops the previous one.
	p.SetStep(happydns.ImportStepExploring, "", 0, 0)
	if s := p.Snapshot(); s.Source != "" {
		t.Errorf("the progress kept the source %q after a step without one", s.Source)
	}
}

func TestImportProgressConcurrent(t *testing.T) {
	p := happydns.NewImportProgress()

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				p.AddTotal(1)
				p.AddDone(1)
				p.SetStep(happydns.ImportStepExploring, "", 10, 20)
				p.Advance(0.5)
				p.SetCounts(1, 1)
				p.Snapshot()
			}
		}()
	}
	wg.Wait()

	if s := p.Snapshot(); s.Total != 800 || s.Done != 800 {
		t.Errorf("the progress is at %d of %d, want 800 of 800", s.Done, s.Total)
	}
}

func TestImportProgressPercent(t *testing.T) {
	p := happydns.NewImportProgress()

	if s := p.Snapshot(); s.Percent != 0 {
		t.Errorf("a new progress is at %d%%, want 0%%", s.Percent)
	}

	p.SetStep(happydns.ImportStepLocating, "", 10, 20)
	if s := p.Snapshot(); s.Percent != 10 || s.StepEnd != 20 {
		t.Errorf("a step from 10%% to 20%% puts the progress at %d%%, ending at %d%%, want 10%%, ending at 20%%", s.Percent, s.StepEnd)
	}

	p.Advance(0.5)
	if s := p.Snapshot(); s.Percent != 15 {
		t.Errorf("half of a step from 10%% to 20%% puts the progress at %d%%, want 15%%", s.Percent)
	}

	p.Advance(2)
	if s := p.Snapshot(); s.Percent != 20 {
		t.Errorf("more than the whole step puts the progress at %d%%, want its end at 20%%", s.Percent)
	}

	// The progress never goes back, even when a step starts lower.
	p.Advance(0.1)
	p.SetStep(happydns.ImportStepExploring, "", 5, 30)
	if s := p.Snapshot(); s.Percent != 20 {
		t.Errorf("the progress went back to %d%%, want it to stay at 20%%", s.Percent)
	}
	p.Advance(0.8)
	if s := p.Snapshot(); s.Percent != 25 {
		t.Errorf("most of a step from 5%% to 30%% puts the progress at %d%%, want 25%%", s.Percent)
	}

	// A skipped step is jumped over.
	p.SetStep(happydns.ImportStepImporting, "", 95, 100)
	if s := p.Snapshot(); s.Percent != 95 {
		t.Errorf("the last step puts the progress at %d%%, want 95%%", s.Percent)
	}
}
