package authuser_test

import (
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

func TestGenAccountRecoveryHash(t *testing.T) {
	recoveryKey := make([]byte, 64)
	hash := authuser.GenAccountRecoveryHash(recoveryKey, false)
	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestCanRecoverAccount(t *testing.T) {
	recoveryKey := make([]byte, 64)
	user := &happydns.UserAuth{PasswordRecoveryKey: recoveryKey}
	hash := authuser.GenAccountRecoveryHash(recoveryKey, false)

	err := authuser.CanRecoverAccount(user, hash)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

type dummyMailer struct {
	err error
}

func (m *dummyMailer) SendMail(to *mail.Address, subject, content string) (err error) {
	return m.err
}

func TestGenerateLink(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}

	uc := authuser.NewRecoverAccountUsecase(store, nil, config, nil)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com"}

	link, err := uc.GenerateLink(user)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if link == "" {
		t.Error("Expected non-empty link")
	}
}

func TestSendLink(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	mailer := &dummyMailer{}
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}

	uc := authuser.NewRecoverAccountUsecase(store, mailer, config, nil)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com"}

	err := uc.SendLink(user)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestSendLink_Error(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	mailer := &dummyMailer{err: errors.New("SMTP Error")}
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}

	uc := authuser.NewRecoverAccountUsecase(store, mailer, config, nil)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com"}

	err := uc.SendLink(user)
	if err == nil || err.Error() != "SMTP Error" {
		t.Errorf("Expected SMTP Error, got %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	mailer := &dummyMailer{}
	config := &happydns.Options{ExternalURL: url.URL{Scheme: "http", Host: "example.com"}}
	changePassword := authuser.NewChangePasswordUsecase(store, authuser.NewCheckPasswordConstraintsUsecase())

	uc := authuser.NewRecoverAccountUsecase(store, mailer, config, changePassword)
	user := &happydns.UserAuth{Id: []byte("user1"), Email: "user@example.com", PasswordRecoveryKey: make([]byte, 64)}

	err := uc.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      authuser.GenAccountRecoveryHash(user.PasswordRecoveryKey, false),
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Previous recovery hash should work too
	err = uc.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      authuser.GenAccountRecoveryHash(user.PasswordRecoveryKey, true),
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Invalid key should not work
	otherKey := make([]byte, 64)
	otherKey[1] = byte('a')
	err = uc.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      authuser.GenAccountRecoveryHash(otherKey, true),
		Password: "StrongPassword123!",
	})
	if err == nil || !strings.HasPrefix(err.Error(), "The account recovery link you follow is invalid or has expired (it is valid during ") {
		t.Errorf("Expected invalid recovery link, got %v", err)
	}
}
