// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

// failingRestore fails every restore with err.
type failingRestore struct {
	happydns.BackupUsecase
	err error
}

func (f failingRestore) Restore(*happydns.Backup) error { return f.err }

// What went wrong reaches the administrator, and the server logs: an error
// value encodes in JSON as an empty object.
func TestRestoreJSONAnswersTheErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)

	bc := NewBackupController(failingRestore{err: errors.Join(errors.New("first problem"), errors.New("second problem"))})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/backup.json", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	bc.RestoreJSON(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("RestoreJSON = %d, want 500", w.Code)
	}

	var resp happydns.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("answer %s: %v", w.Body.String(), err)
	}
	if !strings.Contains(resp.Message, "first problem") || !strings.Contains(resp.Message, "second problem") {
		t.Errorf("answer = %s, want every error", w.Body.String())
	}
	if !strings.Contains(logs.String(), "first problem") || !strings.Contains(logs.String(), "second problem") {
		t.Errorf("logs = %q, want every error", logs.String())
	}
}
