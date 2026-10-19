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
	"context"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestSealHonoursCancelledContext(t *testing.T) {
	m, _, store := instanceManagers(t)
	sc := objectContext()

	obj := &managedObject{ApiKey: happydns.NewSecret("my-api-key")}
	if err := m.SealObject(cancelledContext(), sc, obj); !errors.Is(err, context.Canceled) {
		t.Errorf("SealObject: got %v, want context.Canceled", err)
	}
	if !obj.ApiKey.IsClear() {
		t.Error("SealObject changed the secret despite the cancelled context")
	}

	fsc := sc
	fsc.Field = "apikey"
	s := happydns.NewSecret("my-api-key")
	if err := m.SealSecret(cancelledContext(), fsc, &s); !errors.Is(err, context.Canceled) {
		t.Errorf("SealSecret: got %v, want context.Canceled", err)
	}
	if !s.IsClear() {
		t.Error("SealSecret changed the secret despite the cancelled context")
	}

	if _, err := store.GetSafeByOwner(sc.Owner, KindInstance); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("a safe was created despite the cancelled context: %v", err)
	}
}

func TestOpenHonoursCancelledContext(t *testing.T) {
	m, _, _ := instanceManagers(t)
	sc := objectContext()
	fsc := sc
	fsc.Field = "apikey"

	obj := &managedObject{ApiKey: happydns.NewSecret("my-api-key")}
	if err := m.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	token := obj.ApiKey.Token()

	stored := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.OpenObject(cancelledContext(), sc, stored); !errors.Is(err, context.Canceled) {
		t.Errorf("OpenObject: got %v, want context.Canceled", err)
	}
	if !stored.ApiKey.IsSealed() {
		t.Error("OpenObject opened the secret despite the cancelled context")
	}

	if _, err := m.OpenCopy(cancelledContext(), sc, stored); !errors.Is(err, context.Canceled) {
		t.Errorf("OpenCopy: got %v, want context.Canceled", err)
	}

	s := sealedFromStorage(t, token)
	if err := m.OpenSecret(cancelledContext(), fsc, &s); !errors.Is(err, context.Canceled) {
		t.Errorf("OpenSecret: got %v, want context.Canceled", err)
	}
}

// A cancelled request must not report credentials as unreadable, nor rewrite
// any of them.
func TestAdminHonoursCancelledContext(t *testing.T) {
	m, plaintext, _ := instanceManagers(t)
	sc := objectContext()

	obj := &managedObject{ApiKey: happydns.NewSecret("my-api-key")}
	if err := m.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	token := obj.ApiKey.Token()

	var counts Counts
	stored := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.Inspect(cancelledContext(), sc, stored, &counts); !errors.Is(err, context.Canceled) {
		t.Errorf("Inspect: got %v, want context.Canceled", err)
	}
	if counts.Unreadable != 0 {
		t.Errorf("a cancelled inspection counted %d unreadable secrets", counts.Unreadable)
	}

	if _, err := plaintext.ResealObject(cancelledContext(), sc, stored); !errors.Is(err, context.Canceled) {
		t.Errorf("ResealObject: got %v, want context.Canceled", err)
	}
	if stored.ApiKey.Token() != token {
		t.Error("ResealObject changed the object despite the cancelled context")
	}
}
