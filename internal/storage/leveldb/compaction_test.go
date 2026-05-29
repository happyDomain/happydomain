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

package database

import (
	"fmt"
	"testing"
	"time"

	goerrors "errors"

	"git.happydns.org/happyDomain/model"
)

func openTemp(t *testing.T) *LevelDBStorage {
	t.Helper()
	s, err := NewLevelDBStorage(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("NewLevelDBStorage: %v", err)
	}
	return s
}

func TestCompactReclaimsDeletedKeys(t *testing.T) {
	s := openTemp(t)
	defer s.Close()

	// Write then delete a batch of keys so compaction has tombstones to clear.
	for i := range 100 {
		if err := s.Put(fmt.Sprintf("k-%03d", i), i); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}
	for i := range 100 {
		if err := s.Delete(fmt.Sprintf("k-%03d", i)); err != nil {
			t.Fatalf("Delete: %v", err)
		}
	}

	if err := s.Compact(); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	// The deleted keys must remain logically gone after compaction.
	var v int
	if err := s.Get("k-000", &v); !goerrors.Is(err, happydns.ErrNotFound) {
		t.Errorf("Get after compaction: got err=%v, want ErrNotFound", err)
	}
}

func TestCompactionWorkerStopsOnClose(t *testing.T) {
	s := openTemp(t)
	s.StartCompactionWorker(20 * time.Millisecond)

	// Let the worker tick at least once.
	time.Sleep(60 * time.Millisecond)

	// Close must stop the worker and return without hanging.
	done := make(chan error, 1)
	go func() { done <- s.Close() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close hung: compaction worker did not stop")
	}
}

func TestCompactionWorkerDisabled(t *testing.T) {
	s := openTemp(t)

	// interval <= 0 must start no goroutine and leave Close working.
	s.StartCompactionWorker(0)
	if s.compactionStop != nil || s.compactionDone != nil {
		t.Fatal("StartCompactionWorker(0) should not create worker channels")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
