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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- Stub types ---

// stubCheckerEngine implements happydns.CheckerEngine for testing.
type stubCheckerEngine struct {
	exec *happydns.Execution
	eval *happydns.CheckEvaluation
	err  error
}

func (s *stubCheckerEngine) CreateExecution(checkerID string, target happydns.CheckTarget, plan *happydns.CheckPlan) (*happydns.Execution, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.exec != nil {
		return s.exec, nil
	}
	id, _ := happydns.NewRandomIdentifier()
	return &happydns.Execution{
		Id:        id,
		CheckerID: checkerID,
		Target:    target,
		StartedAt: time.Now(),
		Status:    happydns.ExecutionPending,
	}, nil
}

func (s *stubCheckerEngine) RunExecution(ctx context.Context, exec *happydns.Execution, plan *happydns.CheckPlan, runOpts happydns.CheckerOptions) (*happydns.CheckEvaluation, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.eval != nil {
		return s.eval, nil
	}
	return &happydns.CheckEvaluation{
		CheckerID: exec.CheckerID,
		Target:    exec.Target,
		States:    []happydns.CheckState{{Status: happydns.StatusOK, Code: "ok"}},
	}, nil
}

// testObservationProvider is a no-op provider for tests.
type testObservationProvider struct{}

func (p *testObservationProvider) Key() happydns.ObservationKey { return "test_ctrl_obs" }
func (p *testObservationProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	return map[string]any{"v": 1}, nil
}

// testHTMLObservationProvider implements CheckerHTMLReporter for HTML report tests.
type testHTMLObservationProvider struct{}

func (p *testHTMLObservationProvider) Key() happydns.ObservationKey { return "test_html_obs" }
func (p *testHTMLObservationProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	return map[string]any{"html": true}, nil
}
func (p *testHTMLObservationProvider) GetHTMLReport(raw json.RawMessage) (string, error) {
	return "<html><body>test report</body></html>", nil
}

// testCheckRule produces a fixed status.
type testCheckRule struct {
	name   string
	status happydns.Status
}

func (r *testCheckRule) Name() string        { return r.name }
func (r *testCheckRule) Description() string { return "test rule: " + r.name }
func (r *testCheckRule) Evaluate(ctx context.Context, obs happydns.ObservationGetter, opts happydns.CheckerOptions) happydns.CheckState {
	return happydns.CheckState{Status: r.status, Code: r.name}
}

// registerTestChecker registers a checker for controller tests and returns its ID.
// Uses a unique name to avoid collisions with other tests.
var testCheckerSeq int

func registerTestChecker() string {
	testCheckerSeq++
	id := fmt.Sprintf("ctrl_test_checker_%d", testCheckerSeq)
	checkerPkg.RegisterObservationProvider(&testObservationProvider{})
	checkerPkg.RegisterChecker(&happydns.CheckerDefinition{
		ID:   id,
		Name: "Controller Test Checker",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_a", status: happydns.StatusOK},
		},
	})
	return id
}

// newTestController creates a CheckerController with in-memory storage.
func newTestController(engine happydns.CheckerEngine) *CheckerController {
	cc, _ := newTestControllerWithStorage(engine)
	return cc
}

// newTestControllerWithStorage creates a CheckerController and returns the underlying storage.
func newTestControllerWithStorage(engine happydns.CheckerEngine) (*CheckerController, storage.Storage) {
	store, err := inmemory.Instantiate()
	if err != nil {
		panic(err)
	}
	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	planUC := checkerUC.NewCheckPlanUsecase(store)
	statusUC := checkerUC.NewCheckStatusUsecase(store, store, store, store)
	return NewCheckerController(engine, optionsUC, planUC, statusUC, nil), store
}

// --- targetFromContext tests ---

func TestTargetFromContext_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	target := targetFromContext(c)

	if target.UserId != "" {
		t.Errorf("expected empty UserId, got %q", target.UserId)
	}
	if target.DomainId != "" {
		t.Errorf("expected empty DomainId, got %q", target.DomainId)
	}
	if target.ServiceId != "" {
		t.Errorf("expected empty ServiceId, got %q", target.ServiceId)
	}
}

func TestTargetFromContext_WithUser(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	uid, _ := happydns.NewRandomIdentifier()
	user := &happydns.User{Id: uid}
	c.Set("LoggedUser", user)

	target := targetFromContext(c)

	if target.UserId != uid.String() {
		t.Errorf("expected UserId %q, got %q", uid.String(), target.UserId)
	}
}

func TestTargetFromContext_WithDomain(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	did, _ := happydns.NewRandomIdentifier()
	domain := &happydns.Domain{Id: did}
	c.Set("domain", domain)

	target := targetFromContext(c)

	if target.DomainId != did.String() {
		t.Errorf("expected DomainId %q, got %q", did.String(), target.DomainId)
	}
}

func TestTargetFromContext_WithService(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	sid, _ := happydns.NewRandomIdentifier()
	c.Set("serviceid", happydns.Identifier(sid))

	target := targetFromContext(c)

	if target.ServiceId != sid.String() {
		t.Errorf("expected ServiceId %q, got %q", sid.String(), target.ServiceId)
	}
}

func TestTargetFromContext_WithServiceAndZone(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	sid, _ := happydns.NewRandomIdentifier()
	svc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   sid,
			Type: "svcs.TestType",
		},
	}
	zone := &happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {svc},
		},
	}

	c.Set("serviceid", happydns.Identifier(sid))
	c.Set("zone", zone)

	target := targetFromContext(c)

	if target.ServiceType != "svcs.TestType" {
		t.Errorf("expected ServiceType %q, got %q", "svcs.TestType", target.ServiceType)
	}
}

// --- ListCheckers tests ---

func TestListCheckers_ReturnsRegistered(t *testing.T) {
	checkerID := registerTestChecker()
	cc := newTestController(&stubCheckerEngine{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/checkers", nil)

	cc.ListCheckers(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if _, ok := result[checkerID]; !ok {
		t.Errorf("expected checker %q in response, got keys: %v", checkerID, keysOf(result))
	}
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// --- CheckerHandler tests ---

func TestCheckerHandler_NotFound(t *testing.T) {
	cc := newTestController(&stubCheckerEngine{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/checkers/nonexistent", nil)
	c.Params = gin.Params{{Key: "checkerId", Value: "nonexistent_checker_xyz"}}

	cc.CheckerHandler(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if _, ok := resp["errmsg"]; !ok {
		t.Error("expected errmsg in response")
	}
}

func TestCheckerHandler_Found(t *testing.T) {
	checkerID := registerTestChecker()
	cc := newTestController(&stubCheckerEngine{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/checkers/"+checkerID, nil)
	c.Params = gin.Params{{Key: "checkerId", Value: checkerID}}

	// CheckerHandler calls c.Next(), so we need to verify context is set.
	// Use a gin engine to test the middleware chain.
	router := gin.New()
	router.GET("/checkers/:checkerId", cc.CheckerHandler, cc.GetChecker)

	req := httptest.NewRequest("GET", "/checkers/"+checkerID, nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var def map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &def); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if def["id"] != checkerID {
		t.Errorf("expected checker id %q, got %v", checkerID, def["id"])
	}
}

// --- TriggerCheck tests ---

func TestTriggerCheck_Sync_Returns200(t *testing.T) {
	checkerID := registerTestChecker()

	eval := &happydns.CheckEvaluation{
		CheckerID: checkerID,
		States:    []happydns.CheckState{{Status: happydns.StatusOK, Code: "ok"}},
	}
	engine := &stubCheckerEngine{eval: eval}
	cc := newTestController(engine)

	uid, _ := happydns.NewRandomIdentifier()
	user := &happydns.User{Id: uid}

	body, _ := json.Marshal(happydns.CheckerRunRequest{})
	req := httptest.NewRequest("POST", "/checkers/"+checkerID+"/executions?sync=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/checkers/:checkerId/executions", func(c *gin.Context) {
		c.Set("LoggedUser", user)
		c.Next()
	}, cc.TriggerCheck)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["checkerId"] != checkerID {
		t.Errorf("expected checkerId %q, got %v", checkerID, result["checkerId"])
	}
}

func TestTriggerCheck_Async_Returns202(t *testing.T) {
	checkerID := registerTestChecker()

	engine := &stubCheckerEngine{}
	cc := newTestController(engine)

	uid, _ := happydns.NewRandomIdentifier()
	user := &happydns.User{Id: uid}

	body, _ := json.Marshal(happydns.CheckerRunRequest{})
	req := httptest.NewRequest("POST", "/checkers/"+checkerID+"/executions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/checkers/:checkerId/executions", func(c *gin.Context) {
		c.Set("LoggedUser", user)
		c.Next()
	}, cc.TriggerCheck)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerCheck_EngineError_Returns500(t *testing.T) {
	checkerID := registerTestChecker()

	engine := &stubCheckerEngine{err: fmt.Errorf("engine failure")}
	cc := newTestController(engine)

	uid, _ := happydns.NewRandomIdentifier()
	user := &happydns.User{Id: uid}

	body, _ := json.Marshal(happydns.CheckerRunRequest{})
	req := httptest.NewRequest("POST", "/checkers/"+checkerID+"/executions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := gin.New()
	router.POST("/checkers/:checkerId/executions", func(c *gin.Context) {
		c.Set("LoggedUser", user)
		c.Next()
	}, cc.TriggerCheck)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetExecutionStatus tests ---

func TestGetExecutionStatus_ReturnsExecution(t *testing.T) {
	cc := newTestController(&stubCheckerEngine{})

	execID, _ := happydns.NewRandomIdentifier()
	exec := &happydns.Execution{
		Id:        execID,
		CheckerID: "test",
		Status:    happydns.ExecutionDone,
		Result:    happydns.CheckState{Status: happydns.StatusOK, Message: "done"},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/executions/"+execID.String(), nil)
	c.Set("execution", exec)

	cc.GetExecutionStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["checkerId"] != "test" {
		t.Errorf("expected checkerId %q, got %v", "test", result["checkerId"])
	}
}

// --- GetChecker tests ---

func TestGetChecker_ReturnsDefinition(t *testing.T) {
	checkerID := registerTestChecker()
	cc := newTestController(&stubCheckerEngine{})

	def := checkerPkg.FindChecker(checkerID)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/checkers/"+checkerID, nil)
	c.Set("checker", def)

	cc.GetChecker(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["id"] != checkerID {
		t.Errorf("expected id %q, got %v", checkerID, result["id"])
	}
}

// --- ExecutionHandler tests ---

func TestExecutionHandler_InvalidID(t *testing.T) {
	cc := newTestController(&stubCheckerEngine{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/executions/not-valid", nil)
	c.Params = gin.Params{{Key: "executionId", Value: "not-valid"}}

	cc.ExecutionHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecutionHandler_NotFound(t *testing.T) {
	cc := newTestController(&stubCheckerEngine{})

	fakeID, _ := happydns.NewRandomIdentifier()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/executions/"+fakeID.String(), nil)
	c.Params = gin.Params{{Key: "executionId", Value: fakeID.String()}}

	cc.ExecutionHandler(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- PlanHandler tests ---

func TestPlanHandler_InvalidID(t *testing.T) {
	cc := newTestController(&stubCheckerEngine{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/plans/not-valid", nil)
	c.Params = gin.Params{{Key: "planId", Value: "not-valid"}}

	cc.PlanHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPlanHandler_NotFound(t *testing.T) {
	cc := newTestController(&stubCheckerEngine{})

	fakeID, _ := happydns.NewRandomIdentifier()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/plans/"+fakeID.String(), nil)
	c.Params = gin.Params{{Key: "planId", Value: fakeID.String()}}

	cc.PlanHandler(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetExecutionHTMLReport tests ---

// seedExecutionWithObservations creates an execution backed by a snapshot containing the given
// observation data. It returns the execution (with ID assigned by the store).
func seedExecutionWithObservations(t *testing.T, store storage.Storage, target happydns.CheckTarget, data map[happydns.ObservationKey]json.RawMessage) *happydns.Execution {
	t.Helper()

	snap := &happydns.ObservationSnapshot{
		Target:      target,
		CollectedAt: time.Now(),
		Data:        data,
	}
	if err := store.CreateSnapshot(snap); err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}

	eval := &happydns.CheckEvaluation{
		CheckerID:  "html_test_checker",
		Target:     target,
		SnapshotID: snap.Id,
	}
	if err := store.CreateEvaluation(eval); err != nil {
		t.Fatalf("CreateEvaluation: %v", err)
	}

	exec := &happydns.Execution{
		CheckerID:    "html_test_checker",
		Target:       target,
		Status:       happydns.ExecutionDone,
		EvaluationID: &eval.Id,
	}
	if err := store.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	return exec
}

func init() {
	// Register the HTML observation provider once for tests.
	checkerPkg.RegisterObservationProvider(&testHTMLObservationProvider{})
}

func TestGetExecutionHTMLReport_ObservationsNotAvailable(t *testing.T) {
	cc := newTestController(&stubCheckerEngine{})

	// Create an execution with no evaluation/snapshot backing.
	fakeExecID, _ := happydns.NewRandomIdentifier()
	exec := &happydns.Execution{
		Id:        fakeExecID,
		CheckerID: "html_test_checker",
		Status:    happydns.ExecutionDone,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/report", nil)
	c.Set("execution", exec)
	c.Params = gin.Params{{Key: "obsKey", Value: "test_html_obs"}}

	cc.GetExecutionHTMLReport(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetExecutionHTMLReport_ObservationKeyNotFound(t *testing.T) {
	cc, store := newTestControllerWithStorage(&stubCheckerEngine{})

	target := happydns.CheckTarget{DomainId: "d1"}
	exec := seedExecutionWithObservations(t, store, target, map[happydns.ObservationKey]json.RawMessage{
		"test_html_obs": json.RawMessage(`{"v":1}`),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/report", nil)
	c.Set("execution", exec)
	c.Params = gin.Params{{Key: "obsKey", Value: "nonexistent_key"}}

	cc.GetExecutionHTMLReport(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// testNoHTMLObservationProvider is a provider that does NOT implement CheckerHTMLReporter.
type testNoHTMLObservationProvider struct{}

func (p *testNoHTMLObservationProvider) Key() happydns.ObservationKey { return "test_no_html_obs" }
func (p *testNoHTMLObservationProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	return map[string]any{"v": 1}, nil
}

func init() {
	checkerPkg.RegisterObservationProvider(&testNoHTMLObservationProvider{})
}

func TestGetExecutionHTMLReport_ProviderDoesNotSupportHTML(t *testing.T) {
	cc, store := newTestControllerWithStorage(&stubCheckerEngine{})

	target := happydns.CheckTarget{DomainId: "d1"}
	exec := seedExecutionWithObservations(t, store, target, map[happydns.ObservationKey]json.RawMessage{
		"test_no_html_obs": json.RawMessage(`{"v":1}`),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/report", nil)
	c.Set("execution", exec)
	c.Params = gin.Params{{Key: "obsKey", Value: "test_no_html_obs"}}

	cc.GetExecutionHTMLReport(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 (unsupported), got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetExecutionHTMLReport_Success(t *testing.T) {
	cc, store := newTestControllerWithStorage(&stubCheckerEngine{})

	target := happydns.CheckTarget{DomainId: "d1"}
	exec := seedExecutionWithObservations(t, store, target, map[happydns.ObservationKey]json.RawMessage{
		"test_html_obs": json.RawMessage(`{"v":1}`),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/report", nil)
	c.Set("execution", exec)
	c.Params = gin.Params{{Key: "obsKey", Value: "test_html_obs"}}

	cc.GetExecutionHTMLReport(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if body != "<html><body>test report</body></html>" {
		t.Errorf("unexpected body: %s", body)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header to be set")
	}

	xcto := w.Header().Get("X-Content-Type-Options")
	if xcto != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %q", xcto)
	}
}

// --- getLimitParam tests ---

func newContextWithQuery(query string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?"+query, nil)
	return c
}

func TestGetLimitParam(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		defaultLimit int
		expected     int
	}{
		{"empty query returns default", "", 100, 100},
		{"valid limit", "limit=50", 100, 50},
		{"zero returns default", "limit=0", 100, 100},
		{"negative returns default", "limit=-5", 100, 100},
		{"non-numeric returns default", "limit=abc", 100, 100},
		{"large value capped to maxLimit", "limit=1500", 100, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newContextWithQuery(tt.query)
			got := getLimitParam(c, tt.defaultLimit)
			if got != tt.expected {
				t.Errorf("getLimitParam(%q, %d) = %d, want %d", tt.query, tt.defaultLimit, got, tt.expected)
			}
		})
	}
}
