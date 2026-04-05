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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// --- Stub types for domain tests ---

// stubDomainUsecase implements happydns.DomainUsecase for testing.
type stubDomainUsecase struct {
	domains []*happydns.Domain
	err     error
}

func (s *stubDomainUsecase) CreateDomain(ctx context.Context, user *happydns.User, input *happydns.DomainCreationInput) (*happydns.Domain, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubDomainUsecase) DeleteDomain(id happydns.Identifier) error {
	return fmt.Errorf("not implemented")
}
func (s *stubDomainUsecase) ExtendsDomainWithZoneMeta(d *happydns.Domain) (*happydns.DomainWithZoneMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubDomainUsecase) GetUserDomain(user *happydns.User, id happydns.Identifier) (*happydns.Domain, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubDomainUsecase) GetUserDomainByFQDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubDomainUsecase) ListUserDomains(user *happydns.User) ([]*happydns.Domain, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.domains, nil
}
func (s *stubDomainUsecase) UpdateDomain(id happydns.Identifier, user *happydns.User, fn func(*happydns.Domain)) error {
	return fmt.Errorf("not implemented")
}

// newDomainTestContext creates a gin context with a logged-in user and a recorder.
func newDomainTestContext(user *happydns.User) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/domains", nil)
	if user != nil {
		c.Set("LoggedUser", user)
	}
	return w, c
}

// --- GetDomains tests ---

func TestGetDomains_Unauthenticated(t *testing.T) {
	dc := NewDomainController(&stubDomainUsecase{}, nil, nil, nil)

	w, c := newDomainTestContext(nil)
	dc.GetDomains(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetDomains_ListError(t *testing.T) {
	stub := &stubDomainUsecase{err: fmt.Errorf("db failure")}
	dc := NewDomainController(stub, nil, nil, nil)

	uid, _ := happydns.NewRandomIdentifier()
	user := &happydns.User{Id: uid}
	w, c := newDomainTestContext(user)
	dc.GetDomains(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestGetDomains_EmptyList(t *testing.T) {
	stub := &stubDomainUsecase{domains: []*happydns.Domain{}}
	dc := NewDomainController(stub, nil, nil, nil)

	uid, _ := happydns.NewRandomIdentifier()
	user := &happydns.User{Id: uid}
	w, c := newDomainTestContext(user)
	dc.GetDomains(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result []happydns.DomainWithCheckStatus
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 domains, got %d", len(result))
	}
}

func TestGetDomains_NilCheckStatusUC(t *testing.T) {
	did1, _ := happydns.NewRandomIdentifier()
	did2, _ := happydns.NewRandomIdentifier()
	stub := &stubDomainUsecase{
		domains: []*happydns.Domain{
			{Id: did1, DomainName: "example.com."},
			{Id: did2, DomainName: "example.org."},
		},
	}
	dc := NewDomainController(stub, nil, nil, nil)

	uid, _ := happydns.NewRandomIdentifier()
	user := &happydns.User{Id: uid}
	w, c := newDomainTestContext(user)
	dc.GetDomains(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result []happydns.DomainWithCheckStatus
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(result))
	}

	for _, d := range result {
		if d.LastCheckStatus != nil {
			t.Errorf("expected nil LastCheckStatus when checkStatusUC is nil, got %v for domain %s", *d.LastCheckStatus, d.DomainName)
		}
	}
}

func TestGetDomains_WithCheckStatuses(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()
	did1, _ := happydns.NewRandomIdentifier()
	did2, _ := happydns.NewRandomIdentifier()
	did3, _ := happydns.NewRandomIdentifier()

	stub := &stubDomainUsecase{
		domains: []*happydns.Domain{
			{Id: did1, DomainName: "warn.example.com.", Owner: uid},
			{Id: did2, DomainName: "ok.example.com.", Owner: uid},
			{Id: did3, DomainName: "unchecked.example.com.", Owner: uid},
		},
	}

	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	statusUC := checkerUC.NewCheckStatusUsecase(store, store, store, store)

	// Create executions: domain 1 has WARN, domain 2 has OK, domain 3 has none.
	for _, tc := range []struct {
		domainId happydns.Identifier
		status   happydns.Status
	}{
		{did1, happydns.StatusOK},
		{did1, happydns.StatusWarn},
		{did2, happydns.StatusOK},
	} {
		exec := &happydns.Execution{
			CheckerID: "test_checker",
			Target:    happydns.CheckTarget{UserId: uid.String(), DomainId: tc.domainId.String()},
			StartedAt: time.Now(),
			Status:    happydns.ExecutionDone,
			Result:    happydns.CheckState{Status: tc.status},
		}
		if err := store.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution() error: %v", err)
		}
	}

	dc := NewDomainController(stub, nil, nil, statusUC)

	user := &happydns.User{Id: uid}
	w, c := newDomainTestContext(user)
	dc.GetDomains(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result []happydns.DomainWithCheckStatus
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 domains, got %d", len(result))
	}

	statusByDomain := make(map[string]*happydns.Status)
	for _, d := range result {
		statusByDomain[d.Id.String()] = d.LastCheckStatus
	}

	// Domain 1: worst is WARN.
	if s := statusByDomain[did1.String()]; s == nil {
		t.Error("expected non-nil status for domain 1 (warn.example.com)")
	} else if *s != happydns.StatusWarn {
		t.Errorf("expected WARN for domain 1, got %v", *s)
	}

	// Domain 2: worst is OK.
	if s := statusByDomain[did2.String()]; s == nil {
		t.Error("expected non-nil status for domain 2 (ok.example.com)")
	} else if *s != happydns.StatusOK {
		t.Errorf("expected OK for domain 2, got %v", *s)
	}

	// Domain 3: no executions → nil.
	if s := statusByDomain[did3.String()]; s != nil {
		t.Errorf("expected nil status for domain 3 (unchecked.example.com), got %v", *s)
	}
}

func TestGetDomains_ResponseIncludesDomainFields(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	pid, _ := happydns.NewRandomIdentifier()

	stub := &stubDomainUsecase{
		domains: []*happydns.Domain{
			{Id: did, DomainName: "test.example.com.", Owner: uid, ProviderId: pid, Group: "mygroup"},
		},
	}
	dc := NewDomainController(stub, nil, nil, nil)

	user := &happydns.User{Id: uid}
	w, c := newDomainTestContext(user)
	dc.GetDomains(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var result []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(result))
	}

	// Verify the JSON contains the expected domain fields (embedded from *Domain).
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(result[0], &fields); err != nil {
		t.Fatalf("failed to unmarshal domain entry: %v", err)
	}

	for _, key := range []string{"id", "id_owner", "id_provider", "domain", "group"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("expected field %q in response JSON", key)
		}
	}

	// last_check_status should be omitted when nil (omitempty).
	if _, ok := fields["last_check_status"]; ok {
		t.Error("expected last_check_status to be omitted when nil")
	}
}
