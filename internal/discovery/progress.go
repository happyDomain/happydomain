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

package discovery

import (
	"time"

	"git.happydns.org/happyDomain/model"
)

// band is the share of the whole import a stage of the exploration spans,
// in percent.
type band struct{ from, to float64 }

// bands are the shares of the whole import the stages of the exploration
// span.
type bands struct {
	locating, transfer, apex, collecting, exploring, following band
}

// planBands shares the retrieval of the zone among the stages of the
// exploration, after how long they usually take. The transfer is left out
// when no source hands the whole zone.
func planBands(transfer bool) (b bands) {
	stages := []struct {
		weight float64
		band   *band
	}{
		{5, &b.locating},
		{10, &b.transfer},
		{5, &b.apex},
		{20, &b.collecting},
		{45, &b.exploring},
		{10, &b.following},
	}
	if !transfer {
		stages[1].weight = 0
	}

	var total float64
	for _, stage := range stages {
		total += stage.weight
	}

	var at float64
	for _, stage := range stages {
		width := stage.weight * happydns.ImportRetrievedPercent / total
		*stage.band = band{at, at + width}
		at += width
	}
	return
}

// progressInterval is how often the numbers of records found and of
// questions asked are reported, rather than on each of them.
var progressInterval = 500 * time.Millisecond

// reportCounts reports the numbers of records found and of questions asked
// on e.progress, at each progressInterval until the returned function is
// called, which reports them a last time.
func (e *explorer) reportCounts() (stop func()) {
	report := func() {
		e.mu.Lock()
		records := int64(len(e.records))
		e.mu.Unlock()

		// The questions over the budget are refused, not asked.
		e.progress.SetCounts(records, min(e.queries.Load(), int64(e.opts.MaxQueries)))
	}

	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)

		ticker := time.NewTicker(progressInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				report()
			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
		<-finished
		report()
	}
}
