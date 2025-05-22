package authuser_test

import (
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

func TestGetAuthUserUsecase(t *testing.T) {
	memStore, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatalf("Failed to create in-memory storage: %v", err)
	}

	now := time.Now()
	user := &happydns.UserAuth{
		Email:             "test@example.com",
		EmailVerification: &now,
		CreatedAt:         now,
		LastLoggedIn:      &now,
		Password:          []byte("fakehash"),
	}

	// Add new user in memory (and assign an ID)
	err = memStore.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("Failed to create auth user: %v", err)
	}
	if user.Id == nil {
		t.Fatalf("Expected non-nil user ID, got %s", user.Id)
	}

	uc := authuser.NewGetAuthUserUsecase(memStore)

	t.Run("ByID returns the correct user", func(t *testing.T) {
		got, err := uc.ByID(user.Id)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if got.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %s", got.Email)
		}
	})

	t.Run("ByEmail returns the correct user", func(t *testing.T) {
		got, err := uc.ByEmail("test@example.com")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !got.Id.Equals(user.Id) {
			t.Errorf("Expected ID '%s', got %s", user.Id, got.Id)
		}
	})

	t.Run("ByID returns error for unknown ID", func(t *testing.T) {
		_, err := uc.ByID([]byte("unknown-id"))
		if err == nil {
			t.Error("Expected error for unknown ID, got nil")
		}
	})

	t.Run("ByEmail returns error for unknown email", func(t *testing.T) {
		_, err := uc.ByEmail("unknown@example.com")
		if err == nil {
			t.Error("Expected error for unknown email, got nil")
		}
	})
}
