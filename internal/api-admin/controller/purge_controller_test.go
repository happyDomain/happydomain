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
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

// partlyFailingPurge deletes some keys and fails with err.
type partlyFailingPurge struct {
	report happydns.PurgeReport
	err    error
}

func (p partlyFailingPurge) Purge(happydns.PurgeOptions) (*happydns.PurgeReport, error) {
	return &p.report, p.err
}

// A failing category does not hide what the others deleted: the
// administrator would otherwise run the whole purge again, blind.
func TestPurgeAnswersThePartialReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)

	pc := NewPurgeController(partlyFailingPurge{
		report: happydns.PurgeReport{Checkers: 42, Compacted: true},
		err:    errors.New("records problem"),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/purge", bytes.NewBufferString(`{"checkers":true,"notification_records":true,"compact":true}`))
	c.Request.Header.Set("Content-Type", "application/json")

	pc.Purge(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Purge = %d, want 500", w.Code)
	}

	var report happydns.PurgeReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("answer %s: %v", w.Body.String(), err)
	}
	want := happydns.PurgeReport{Checkers: 42, Compacted: true, Error: "records problem"}
	if report != want {
		t.Errorf("answer = %+v, want %+v", report, want)
	}
	if !strings.Contains(logs.String(), "records problem") {
		t.Errorf("logs = %q, want the error", logs.String())
	}
}
