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

package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"

	"git.happydns.org/happyDomain/model"
)

func TestRenderPrometheus_ParsesAsValidExposition(t *testing.T) {
	// Include a label value with characters that fmt's %q would have escaped
	// as \xNN / \uNNNN — sequences which are NOT valid in Prometheus text
	// format. The output must still parse cleanly via the upstream parser.
	out := renderPrometheus([]happydns.CheckMetric{
		{
			Name:      "happydomain_check_latency_seconds",
			Unit:      "seconds",
			Value:     0.123,
			Timestamp: time.Unix(1700000000, 0),
			Labels: map[string]string{
				"target": "exämple.com",      // non-ASCII
				"note":   "line1\nline2",     // newline (must become \n)
				"quoted": `he said "hi"`,     // quotes
				"slash":  `a\b`,              // backslash
			},
		},
		{
			Name:  "happydomain_check_latency_seconds",
			Value: 0.456,
			Labels: map[string]string{
				"target": "second.example",
			},
		},
	})

	p := expfmt.NewTextParser(model.LegacyValidation)
	if _, err := p.TextToMetricFamilies(strings.NewReader(out)); err != nil {
		t.Fatalf("renderPrometheus output is not valid Prometheus text format: %v\noutput:\n%s", err, out)
	}
}

func TestRenderPrometheus_EscapesLabelValues(t *testing.T) {
	out := renderPrometheus([]happydns.CheckMetric{{
		Name:  "x",
		Value: 1,
		Labels: map[string]string{
			"a": `\`,
			"b": `"`,
			"c": "\n",
		},
	}})
	if !strings.Contains(out, `a="\\"`) {
		t.Errorf("backslash not escaped: %q", out)
	}
	if !strings.Contains(out, `b="\""`) {
		t.Errorf("quote not escaped: %q", out)
	}
	if !strings.Contains(out, `c="\n"`) {
		t.Errorf("newline not escaped: %q", out)
	}
}

func TestRenderPrometheus_NoOpenMetricsDirectives(t *testing.T) {
	out := renderPrometheus([]happydns.CheckMetric{{
		Name:  "x",
		Unit:  "seconds",
		Value: 1,
	}})
	if strings.Contains(out, "# UNIT") {
		t.Errorf("output contains OpenMetrics-only # UNIT directive incompatible with text/plain;version=0.0.4: %q", out)
	}
}

func TestWantsPrometheusText(t *testing.T) {
	cases := []struct {
		accept string
		want   bool
	}{
		{"", false},
		{"*/*", false},
		{"application/json", false},
		{"application/json, text/plain", false}, // explicit JSON wins
		{"text/plain", true},
		{"text/plain; version=0.0.4", true},
		{"application/openmetrics-text; version=1.0.0", true},
	}
	for _, tc := range cases {
		if got := wantsPrometheusText(tc.accept); got != tc.want {
			t.Errorf("wantsPrometheusText(%q) = %v, want %v", tc.accept, got, tc.want)
		}
	}
}
