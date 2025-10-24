// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package domainlog_test

import (
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/domain_log"
	"git.happydns.org/happyDomain/model"
)

func createTestUser(t *testing.T, store *inmemory.InMemoryStorage, email string) *happydns.User {
	user := &happydns.User{
		Id:    happydns.Identifier([]byte("user-" + email)),
		Email: email,
	}
	if err := store.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestDomain(t *testing.T, store *inmemory.InMemoryStorage, user *happydns.User, domainName string) *happydns.Domain {
	domain := &happydns.Domain{
		Owner:      user.Id,
		DomainName: domainName,
	}
	if err := store.CreateDomain(domain); err != nil {
		t.Fatalf("failed to create test domain: %v", err)
	}
	return domain
}

func Test_AppendDomainLog(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	logService := domainlog.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")
	domain := createTestDomain(t, mem, user, "example.com")

	log := happydns.NewDomainLog(user, happydns.LOG_INFO, "Test log entry")

	err := logService.AppendDomainLog(domain, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.Id.IsEmpty() {
		t.Error("expected log ID to be set")
	}
	if !log.IdUser.Equals(user.Id) {
		t.Errorf("expected log IdUser to be %v, got %v", user.Id, log.IdUser)
	}
	if log.Content != "Test log entry" {
		t.Errorf("expected content 'Test log entry', got %s", log.Content)
	}
	if log.Level != happydns.LOG_INFO {
		t.Errorf("expected level LOG_INFO, got %d", log.Level)
	}
	if log.Date.IsZero() {
		t.Error("expected Date to be set")
	}

	// Verify log is stored in database
	logs, err := mem.ListDomainLogs(domain)
	if err != nil {
		t.Fatalf("expected stored logs, got error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Content != "Test log entry" {
		t.Errorf("expected stored content to be 'Test log entry', got %s", logs[0].Content)
	}
}

func Test_ListDomainLogs(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	logService := domainlog.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")
	domain := createTestDomain(t, mem, user, "example.com")

	// Create multiple logs with different timestamps
	log1 := happydns.NewDomainLog(user, happydns.LOG_INFO, "First log")
	log1.Date = time.Now().Add(-2 * time.Hour)
	err := logService.AppendDomainLog(domain, log1)
	if err != nil {
		t.Fatalf("unexpected error creating log 1: %v", err)
	}

	log2 := happydns.NewDomainLog(user, happydns.LOG_WARN, "Second log")
	log2.Date = time.Now().Add(-1 * time.Hour)
	err = logService.AppendDomainLog(domain, log2)
	if err != nil {
		t.Fatalf("unexpected error creating log 2: %v", err)
	}

	log3 := happydns.NewDomainLog(user, happydns.LOG_DEBUG, "Third log")
	log3.Date = time.Now()
	err = logService.AppendDomainLog(domain, log3)
	if err != nil {
		t.Fatalf("unexpected error creating log 3: %v", err)
	}

	// List logs
	logs, err := logService.ListDomainLogs(domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}

	// Verify logs are sorted by date (newest first)
	if len(logs) >= 3 {
		if logs[0].Content != "Third log" {
			t.Errorf("expected first log to be 'Third log', got %s", logs[0].Content)
		}
		if logs[1].Content != "Second log" {
			t.Errorf("expected second log to be 'Second log', got %s", logs[1].Content)
		}
		if logs[2].Content != "First log" {
			t.Errorf("expected third log to be 'First log', got %s", logs[2].Content)
		}
	}
}

func Test_ListDomainLogs_MultipleDomains(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	logService := domainlog.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")
	domain1 := createTestDomain(t, mem, user, "example.com")
	domain2 := createTestDomain(t, mem, user, "test.com")

	// Create logs for domain1
	log1 := happydns.NewDomainLog(user, happydns.LOG_INFO, "Domain1 Log 1")
	err := logService.AppendDomainLog(domain1, log1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log2 := happydns.NewDomainLog(user, happydns.LOG_INFO, "Domain1 Log 2")
	err = logService.AppendDomainLog(domain1, log2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create log for domain2
	log3 := happydns.NewDomainLog(user, happydns.LOG_INFO, "Domain2 Log 1")
	err = logService.AppendDomainLog(domain2, log3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List logs for domain1
	domain1Logs, err := logService.ListDomainLogs(domain1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domain1Logs) != 2 {
		t.Errorf("expected 2 logs for domain1, got %d", len(domain1Logs))
	}

	// List logs for domain2
	domain2Logs, err := logService.ListDomainLogs(domain2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domain2Logs) != 1 {
		t.Errorf("expected 1 log for domain2, got %d", len(domain2Logs))
	}
}

func Test_UpdateDomainLog(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	logService := domainlog.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")
	domain := createTestDomain(t, mem, user, "example.com")

	// Create a log
	log := happydns.NewDomainLog(user, happydns.LOG_INFO, "Original content")
	err := logService.AppendDomainLog(domain, log)
	if err != nil {
		t.Fatalf("unexpected error creating log: %v", err)
	}

	// Update the log
	log.Content = "Updated content"
	log.Level = happydns.LOG_WARN
	err = logService.UpdateDomainLog(domain, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the log was updated
	logs, err := logService.ListDomainLogs(domain)
	if err != nil {
		t.Fatalf("unexpected error retrieving logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got %s", logs[0].Content)
	}
	if logs[0].Level != happydns.LOG_WARN {
		t.Errorf("expected level LOG_WARN, got %d", logs[0].Level)
	}
}

func Test_DeleteDomainLog(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	logService := domainlog.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")
	domain := createTestDomain(t, mem, user, "example.com")

	// Create multiple logs
	log1 := happydns.NewDomainLog(user, happydns.LOG_INFO, "Log 1")
	err := logService.AppendDomainLog(domain, log1)
	if err != nil {
		t.Fatalf("unexpected error creating log 1: %v", err)
	}

	log2 := happydns.NewDomainLog(user, happydns.LOG_INFO, "Log 2")
	err = logService.AppendDomainLog(domain, log2)
	if err != nil {
		t.Fatalf("unexpected error creating log 2: %v", err)
	}

	// Delete the first log
	err = logService.DeleteDomainLog(domain, log1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the log was deleted
	logs, err := logService.ListDomainLogs(domain)
	if err != nil {
		t.Fatalf("unexpected error listing logs: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log after deletion, got %d", len(logs))
	}
	if len(logs) == 1 && logs[0].Content != "Log 2" {
		t.Errorf("expected remaining log to be 'Log 2', got %s", logs[0].Content)
	}
}

func Test_AppendDomainLog_DifferentLogLevels(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	logService := domainlog.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")
	domain := createTestDomain(t, mem, user, "example.com")

	levels := []int8{
		happydns.LOG_CRIT,
		happydns.LOG_FATAL,
		happydns.LOG_ERR,
		happydns.LOG_WARN,
		happydns.LOG_INFO,
		happydns.LOG_DEBUG,
	}

	for _, level := range levels {
		log := happydns.NewDomainLog(user, level, "Test log")
		err := logService.AppendDomainLog(domain, log)
		if err != nil {
			t.Fatalf("unexpected error for level %d: %v", level, err)
		}
		if log.Level != level {
			t.Errorf("expected level %d, got %d", level, log.Level)
		}
	}

	// Verify all logs were created
	logs, err := logService.ListDomainLogs(domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != len(levels) {
		t.Errorf("expected %d logs, got %d", len(levels), len(logs))
	}
}

func Test_ListDomainLogs_EmptyDomain(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	logService := domainlog.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")
	domain := createTestDomain(t, mem, user, "example.com")

	// List logs for a domain with no logs
	logs, err := logService.ListDomainLogs(domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs for empty domain, got %d", len(logs))
	}
}
