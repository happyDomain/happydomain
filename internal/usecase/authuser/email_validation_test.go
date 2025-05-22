package authuser_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

func TestGenEmailValidationHash(t *testing.T) {
	hash := authuser.GenRegistrationHash(&happydns.UserAuth{CreatedAt: time.Now()}, false)
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestGenerateRegistrationLink(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}

	uc := authuser.NewEmailValidationUsecase(store, nil, config)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com"}

	link := uc.GenerateLink(user)
	if link == "" {
		t.Error("Expected non-empty link")
	}
}

func TestSendRegistrationLink(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	mailer := &dummyMailer{}
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}

	uc := authuser.NewEmailValidationUsecase(store, mailer, config)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com"}

	err := uc.SendLink(user)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestSendRegistrationLink_Error(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	mailer := &dummyMailer{err: errors.New("SMTP Error")}
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}

	uc := authuser.NewEmailValidationUsecase(store, mailer, config)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com"}

	err := uc.SendLink(user)
	if err == nil || err.Error() != "SMTP Error" {
		t.Errorf("Expected SMTP Error, got %v", err)
	}
}

func TestValidateEmail(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	mailer := &dummyMailer{}
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}

	uc := authuser.NewEmailValidationUsecase(store, mailer, config)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com", PasswordRecoveryKey: make([]byte, 64)}

	err := uc.Validate(user, happydns.AddressValidationForm{
		Key: authuser.GenRegistrationHash(user, false),
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Previous recovery hash should work too
	err = uc.Validate(user, happydns.AddressValidationForm{
		Key: authuser.GenRegistrationHash(user, true),
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Non-matching user creation time should not work
	user2 := *user
	user2.CreatedAt = time.Now()
	err = uc.Validate(user, happydns.AddressValidationForm{
		Key: authuser.GenRegistrationHash(&user2, false),
	})
	if err == nil || !strings.HasPrefix(err.Error(), "bad email validation key: the validation address link you follow is invalid or has expired (it is valid during ") {
		t.Errorf("Expected invalid validation link, got %v", err)
	}
}
