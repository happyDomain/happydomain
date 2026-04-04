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

package checker

import (
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestWorstStatusAggregator_Empty(t *testing.T) {
	agg := WorstStatusAggregator{}
	got := agg.Aggregate(nil)
	if got.Status != happydns.StatusUnknown {
		t.Errorf("Aggregate(nil) status = %v, want StatusUnknown", got.Status)
	}
	if got.Message != "" {
		t.Errorf("Aggregate(nil) message = %q, want empty", got.Message)
	}
}

func TestWorstStatusAggregator_Single(t *testing.T) {
	agg := WorstStatusAggregator{}
	got := agg.Aggregate([]happydns.CheckState{
		{Status: happydns.StatusOK, Message: "all good"},
	})
	if got.Status != happydns.StatusOK {
		t.Errorf("status = %v, want StatusOK", got.Status)
	}
	if got.Message != "all good" {
		t.Errorf("message = %q, want %q", got.Message, "all good")
	}
}

func TestWorstStatusAggregator_PicksWorst(t *testing.T) {
	agg := WorstStatusAggregator{}
	tests := []struct {
		name     string
		states   []happydns.CheckState
		wantStat happydns.Status
	}{
		{
			name: "ok and warn",
			states: []happydns.CheckState{
				{Status: happydns.StatusOK},
				{Status: happydns.StatusWarn},
			},
			wantStat: happydns.StatusWarn,
		},
		{
			name: "crit among ok and warn",
			states: []happydns.CheckState{
				{Status: happydns.StatusOK},
				{Status: happydns.StatusCrit},
				{Status: happydns.StatusWarn},
			},
			wantStat: happydns.StatusCrit,
		},
		{
			name: "error is worst",
			states: []happydns.CheckState{
				{Status: happydns.StatusCrit},
				{Status: happydns.StatusError},
				{Status: happydns.StatusOK},
			},
			wantStat: happydns.StatusError,
		},
		{
			name: "info and ok",
			states: []happydns.CheckState{
				{Status: happydns.StatusInfo},
				{Status: happydns.StatusOK},
			},
			wantStat: happydns.StatusInfo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agg.Aggregate(tt.states)
			if got.Status != tt.wantStat {
				t.Errorf("status = %v, want %v", got.Status, tt.wantStat)
			}
		})
	}
}

func TestWorstStatusAggregator_ConcatenatesMessages(t *testing.T) {
	agg := WorstStatusAggregator{}
	got := agg.Aggregate([]happydns.CheckState{
		{Status: happydns.StatusOK, Message: "check A passed"},
		{Status: happydns.StatusWarn, Message: ""},
		{Status: happydns.StatusCrit, Message: "check C failed"},
	})
	want := "check A passed; check C failed"
	if got.Message != want {
		t.Errorf("message = %q, want %q", got.Message, want)
	}
}
