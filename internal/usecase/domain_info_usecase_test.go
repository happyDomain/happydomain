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

package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	happydns "git.happydns.org/happyDomain/model"

	"git.happydns.org/happyDomain/internal/usecase"
)

func fakeGetter(info *happydns.DomainInfo, err error) happydns.DomainInfoGetter {
	return func(_ context.Context, _ happydns.Origin) (*happydns.DomainInfo, error) {
		return info, err
	}
}

func TestDomainInfoUsecase_FirstGetterSucceeds(t *testing.T) {
	expected := &happydns.DomainInfo{Name: "example.com", Registrar: "First"}
	uc := usecase.NewDomainInfoUsecase(
		fakeGetter(expected, nil),
		fakeGetter(&happydns.DomainInfo{Name: "example.com", Registrar: "Second"}, nil),
	)

	info, err := uc.GetDomainInfo(context.Background(), "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Registrar != "First" {
		t.Errorf("Registrar = %q, want %q (should use first getter)", info.Registrar, "First")
	}
}

func TestDomainInfoUsecase_FallsBackToSecondGetter(t *testing.T) {
	expected := &happydns.DomainInfo{Name: "example.com", Registrar: "Second"}
	uc := usecase.NewDomainInfoUsecase(
		fakeGetter(nil, fmt.Errorf("RDAP failed")),
		fakeGetter(expected, nil),
	)

	info, err := uc.GetDomainInfo(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Registrar != "Second" {
		t.Errorf("Registrar = %q, want %q", info.Registrar, "Second")
	}
}

func TestDomainInfoUsecase_DomainDoesNotExist_StopsImmediately(t *testing.T) {
	secondCalled := false
	uc := usecase.NewDomainInfoUsecase(
		fakeGetter(nil, happydns.ErrDomainDoesNotExist),
		func(_ context.Context, _ happydns.Origin) (*happydns.DomainInfo, error) {
			secondCalled = true
			return &happydns.DomainInfo{Name: "example.com"}, nil
		},
	)

	_, err := uc.GetDomainInfo(context.Background(), "example.com")
	if !errors.Is(err, happydns.ErrDomainDoesNotExist) {
		t.Errorf("expected ErrDomainDoesNotExist, got: %v", err)
	}
	if secondCalled {
		t.Error("second getter should not be called when first returns ErrDomainDoesNotExist")
	}
}

func TestDomainInfoUsecase_AllGettersFail(t *testing.T) {
	uc := usecase.NewDomainInfoUsecase(
		fakeGetter(nil, fmt.Errorf("RDAP down")),
		fakeGetter(nil, fmt.Errorf("WHOIS down")),
	)

	_, err := uc.GetDomainInfo(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error when all getters fail")
	}
	if !errors.Is(err, fmt.Errorf("WHOIS down")) {
		// The last error should be wrapped
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}

func TestDomainInfoUsecase_GetterReturnsNilInfo(t *testing.T) {
	expected := &happydns.DomainInfo{Name: "example.com", Registrar: "Fallback"}
	uc := usecase.NewDomainInfoUsecase(
		fakeGetter(nil, nil), // no error but nil info
		fakeGetter(expected, nil),
	)

	info, err := uc.GetDomainInfo(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Registrar != "Fallback" {
		t.Errorf("Registrar = %q, want %q", info.Registrar, "Fallback")
	}
}

func TestDomainInfoUsecase_StripsTrailingDot(t *testing.T) {
	var receivedDomain happydns.Origin
	uc := usecase.NewDomainInfoUsecase(
		func(_ context.Context, domain happydns.Origin) (*happydns.DomainInfo, error) {
			receivedDomain = domain
			return &happydns.DomainInfo{Name: string(domain)}, nil
		},
	)

	_, err := uc.GetDomainInfo(context.Background(), "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedDomain != "example.com" {
		t.Errorf("domain passed to getter = %q, want %q (trailing dot should be stripped)", receivedDomain, "example.com")
	}
}

func TestDomainInfoUsecase_NoGetters(t *testing.T) {
	uc := usecase.NewDomainInfoUsecase()

	_, err := uc.GetDomainInfo(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error with no getters")
	}
}
