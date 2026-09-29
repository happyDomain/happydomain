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

package secret

import (
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// maxProblems bounds how many objects a report names: when thousands fail
// the same way, it names some of them and says how many others there are.
const maxProblems = 20

// problems gathers what a report says went wrong, up to maxProblems.
type problems struct {
	items []string
	more  int
}

func (p *problems) add(format string, args ...any) {
	if len(p.items) >= maxProblems {
		p.more++
		return
	}
	p.items = append(p.items, fmt.Sprintf(format, args...))
}

func (p *problems) list() []string {
	if p.more > 0 {
		return append(p.items, fmt.Sprintf("and %d more", p.more))
	}
	return p.items
}

// skipped tells whether err leaves an object for a later run rather than
// being a failure: it changed meanwhile, or its owner no longer exists.
func skipped(err error) bool {
	return errors.Is(err, happydns.ErrChangedMeanwhile) || errors.Is(err, ErrUnknownOwner)
}

// ResealAll runs reseal on every object iter lists, and reports what it did.
// reseal tells whether it wrote the object back.
//
// The listing is read to its end, and closed, before the first reseal:
// writing while iterating is not safe on every storage. An object that does
// not decode, or whose reseal fails, is reported and left as it was, and the
// others go on: running it again is how a reseal resumes. Only a failure to
// list stops it.
func ResealAll[T any](objectType string, iter happydns.Iterator[T], name func(*T) string, reseal func(*T) (bool, error)) (ResealReport, error) {
	report := ResealReport{ObjectType: objectType}
	var probs problems

	var items []*T
	for iter.NextWithError() {
		if err := iter.Err(); err != nil {
			report.Processed++
			report.Failed++
			probs.add("%s: %s", iter.Key(), err.Error())
			continue
		}
		items = append(items, iter.Item())
	}
	err := iter.Err()
	iter.Close()
	if err != nil {
		return ResealReport{ObjectType: objectType}, err
	}

	for _, item := range items {
		report.Processed++

		changed, err := reseal(item)
		switch {
		case err == nil:
			if changed {
				report.Changed++
			}
		case skipped(err):
			report.Skipped++
			probs.add("%s: skipped, run again: %s", name(item), err.Error())
		default:
			report.Failed++
			probs.add("%s: %s", name(item), err.Error())
		}
	}

	report.Errors = probs.list()
	return report, nil
}

// InspectAll adds up what inspect tells of every object iter lists.
//
// An object that does not decode, or that inspect cannot read, is counted as
// undecodable rather than stopping the report: it is needed most when the
// stored data is damaged. Only a failure to list stops it.
func InspectAll[T any](iter happydns.Iterator[T], name func(*T) string, inspect func(*T, *Counts) error) (Counts, error) {
	defer iter.Close()

	c := Counts{Sealed: map[string]int{}}
	var probs problems

	for iter.NextWithError() {
		if err := iter.Err(); err != nil {
			c.Undecodable++
			probs.add("%s: %s", iter.Key(), err.Error())
			continue
		}

		// Counted apart, so that an object failing halfway adds nothing
		// but its failure.
		var one Counts
		if err := inspect(iter.Item(), &one); err != nil {
			c.Undecodable++
			probs.add("%s: %s", name(iter.Item()), err.Error())
			continue
		}
		c.add(one)
	}
	if err := iter.Err(); err != nil {
		return Counts{}, err
	}

	c.Problems = probs.list()
	return c, nil
}

func (c *Counts) add(o Counts) {
	c.Clear += o.Clear
	c.Unreadable += o.Unreadable
	c.Undecodable += o.Undecodable
	if c.Sealed == nil {
		c.Sealed = map[string]int{}
	}
	for kind, n := range o.Sealed {
		c.Sealed[kind] += n
	}
}
