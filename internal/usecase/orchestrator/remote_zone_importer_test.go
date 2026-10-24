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

package orchestrator_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	"git.happydns.org/happyDomain/internal/usecase/orchestrator"
	"git.happydns.org/happyDomain/model"
)

// blockingRetriever holds every retrieval until release is closed, and
// tells the context each one saw once released.
type blockingRetriever struct {
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
	ctxErrs chan error
}

func newBlockingRetriever() *blockingRetriever {
	return &blockingRetriever{
		started: make(chan struct{}, 8),
		release: make(chan struct{}),
		ctxErrs: make(chan error, 8),
	}
}

func (r *blockingRetriever) RetrieveZone(ctx context.Context, _ *happydns.Provider, _ string) ([]happydns.Record, error) {
	r.calls.Add(1)
	r.started <- struct{}{}
	<-r.release
	r.ctxErrs <- ctx.Err()
	return nil, nil
}

// countingZoneImporter counts the zones it imports.
type countingZoneImporter struct {
	calls atomic.Int32
}

func (i *countingZoneImporter) Import(_ *happydns.User, _ *happydns.Domain, _ []happydns.Record) (*happydns.Zone, error) {
	n := i.calls.Add(1)
	return &happydns.Zone{ZoneMeta: happydns.ZoneMeta{Id: happydns.Identifier([]byte{byte(n)})}}, nil
}

func newBlockingImporter(retriever *blockingRetriever, importer *countingZoneImporter) *orchestrator.RemoteZoneImporterUsecase {
	return orchestrator.NewRemoteZoneImporterUsecase(
		domainlogUC.NoopDomainLogAppender{},
		&mockProviderGetter{provider: &happydns.Provider{}},
		importer,
		retriever,
	)
}

func TestRemoteZoneImporter_JoinsInFlightImport(t *testing.T) {
	retriever := newBlockingRetriever()
	importer := &countingZoneImporter{}
	uc := newBlockingImporter(retriever, importer)

	domain := &happydns.Domain{Id: happydns.Identifier([]byte("d1")), DomainName: "example.com."}

	type result struct {
		zone *happydns.Zone
		err  error
	}
	results := make(chan result, 2)
	importOnce := func() {
		zone, err := uc.Import(context.Background(), &happydns.User{}, domain)
		results <- result{zone, err}
	}

	go importOnce()
	<-retriever.started
	go importOnce()

	// Leave the second caller the time to join the import in flight.
	time.Sleep(50 * time.Millisecond)
	close(retriever.release)

	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("Import: %v, %v", first.err, second.err)
	}
	if n := retriever.calls.Load(); n != 1 {
		t.Errorf("the zone was retrieved %d times; want once for both callers", n)
	}
	if n := importer.calls.Load(); n != 1 {
		t.Errorf("the zone was imported %d times; want once for both callers", n)
	}
	if !first.zone.Id.Equals(second.zone.Id) {
		t.Errorf("callers got zones %s and %s; want the same", first.zone.Id.String(), second.zone.Id.String())
	}
}

func TestRemoteZoneImporter_SurvivesCallerCancel(t *testing.T) {
	retriever := newBlockingRetriever()
	importer := &countingZoneImporter{}
	uc := newBlockingImporter(retriever, importer)

	domain := &happydns.Domain{Id: happydns.Identifier([]byte("d1")), DomainName: "example.com."}

	ctx, cancel := context.WithCancel(context.Background())
	errs := make(chan error, 1)
	go func() {
		_, err := uc.Import(ctx, &happydns.User{}, domain)
		errs <- err
	}()

	<-retriever.started
	cancel()

	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Import returned %v; want the cancellation of the caller", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Import kept the caller waiting after it gave up")
	}

	close(retriever.release)
	if err := <-retriever.ctxErrs; err != nil {
		t.Errorf("the retrieval saw its context ended (%v); want it to outlive the caller", err)
	}

	deadline := time.Now().Add(time.Second)
	for importer.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if importer.calls.Load() != 1 {
		t.Error("the zone retrieved was not imported once the caller had left")
	}
}

func TestRemoteZoneImporter_DistinctDomains(t *testing.T) {
	retriever := newBlockingRetriever()
	uc := newBlockingImporter(retriever, &countingZoneImporter{})

	errs := make(chan error, 2)
	for _, id := range []string{"d1", "d2"} {
		go func() {
			_, err := uc.Import(context.Background(), &happydns.User{}, &happydns.Domain{Id: happydns.Identifier([]byte(id)), DomainName: id + ".example."})
			errs <- err
		}()
	}

	<-retriever.started
	<-retriever.started
	close(retriever.release)

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("Import: %v", err)
		}
	}
	if n := retriever.calls.Load(); n != 2 {
		t.Errorf("two domains were retrieved %d times; want twice", n)
	}
}

// panickingRetriever panics on every retrieval, as a faulty provider would.
type panickingRetriever struct{}

func (panickingRetriever) RetrieveZone(context.Context, *happydns.Provider, string) ([]happydns.Record, error) {
	panic("provider bug")
}

func TestRemoteZoneImporter_RecoversPanic(t *testing.T) {
	uc := orchestrator.NewRemoteZoneImporterUsecase(
		domainlogUC.NoopDomainLogAppender{},
		&mockProviderGetter{provider: &happydns.Provider{}},
		&countingZoneImporter{},
		panickingRetriever{},
	)

	domain := &happydns.Domain{Id: happydns.Identifier([]byte("d1")), DomainName: "example.com."}

	zone, err := uc.Import(context.Background(), &happydns.User{}, domain)
	if err == nil {
		t.Fatalf("Import returned zone %v; want the panic as an error", zone)
	}
}
