package usecase_test

import (
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	userUC "git.happydns.org/happyDomain/internal/usecase/user"
	"git.happydns.org/happyDomain/model"
)

type testUserInfo struct {
	id         happydns.Identifier
	email      string
	newsletter bool
}

func (u testUserInfo) GetUserId() happydns.Identifier { return u.id }
func (u testUserInfo) GetEmail() string               { return u.email }
func (u testUserInfo) JoinNewsletter() bool           { return u.newsletter }

func Test_CompleteAuthentication(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	userUsecase := userUC.NewUserUsecases(mem, nil, nil, nil)
	authenticationUsecase := usecase.NewAuthenticationUsecase(&happydns.Options{}, mem, userUsecase)

	uinfo := testUserInfo{
		id:         happydns.Identifier([]byte("user-123")),
		email:      "john@example.com",
		newsletter: false,
	}

	user, err := authenticationUsecase.CompleteAuthentication(uinfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %s", user.Email)
	}

	// Check the user is correctly stored in db
	stored, err := mem.GetUser(happydns.Identifier([]byte("user-123")))
	if err != nil {
		t.Fatalf("expected stored user, got error: %v", err)
	}
	if stored.Email != "john@example.com" {
		t.Errorf("expected stored email to be john@example.com, got %s", stored.Email)
	}
}

type testNewsletterSubscription struct {
	userSubscribed happydns.UserInfo
}

func (ds *testNewsletterSubscription) SubscribeToNewsletter(u happydns.UserInfo) error {
	ds.userSubscribed = u

	return nil
}

func Test_CompleteAuthentication_WithNewsletter(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	mockNewsletterSubscription := &testNewsletterSubscription{}
	userUsecase := userUC.NewUserUsecases(mem, mockNewsletterSubscription, nil, nil)
	authenticationUsecase := usecase.NewAuthenticationUsecase(&happydns.Options{}, mem, userUsecase)

	uinfo := testUserInfo{
		id:         happydns.Identifier([]byte("user-123")),
		email:      "john@example.com",
		newsletter: true,
	}

	_, err := authenticationUsecase.CompleteAuthentication(uinfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check the user has been subscribed
	if mockNewsletterSubscription.userSubscribed == nil || mockNewsletterSubscription.userSubscribed.GetEmail() != uinfo.GetEmail() {
		t.Errorf("user not subscribed to newsletter after first login")
	}

	// Reset the subscription state
	mockNewsletterSubscription.userSubscribed = nil

	// Redo the authentication, now that the user is already registered
	_, err = authenticationUsecase.CompleteAuthentication(uinfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockNewsletterSubscription.userSubscribed != nil {
		t.Errorf("user has been re-subscribed to newsletter beyond first login")
	}
}

func Test_AuthenticateUserWithPassword_WrongPassword(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()

	authUser := &happydns.UserAuth{
		Email: "a@b.c",
	}
	err := authUser.DefinePassword("secure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mem.CreateAuthUser(authUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userUsecase := userUC.NewUserUsecases(mem, nil, nil, nil)
	authenticationUsecase := usecase.NewAuthenticationUsecase(&happydns.Options{}, mem, userUsecase)

	_, err = authenticationUsecase.AuthenticateUserWithPassword(happydns.LoginRequest{
		Email:    "a@b.c",
		Password: "wrong-password",
	})
	if err == nil || err.Error() != `tries to login as "a@b.c", but sent an invalid password` {
		t.Errorf("unexpected error: %v", err)
	}
}

func Test_AuthenticateUserWithPassword_WeakPassword(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()

	authUser := &happydns.UserAuth{
		Email: "a@b.c",
	}
	err := authUser.DefinePassword("weak")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mem.CreateAuthUser(authUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userUsecase := userUC.NewUserUsecases(mem, nil, nil, nil)
	authenticationUsecase := usecase.NewAuthenticationUsecase(&happydns.Options{}, mem, userUsecase)

	_, err = authenticationUsecase.AuthenticateUserWithPassword(happydns.LoginRequest{
		Email:    "a@b.c",
		Password: "weak",
	})
	if err == nil || err.Error() != `tries to login as "a@b.c", but sent an invalid password` {
		t.Errorf("unexpected error: %v", err)
	}
}

func Test_AuthenticateUserWithPassword_UnverifiedEmail(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()

	authUser := &happydns.UserAuth{
		Email: "a@b.c",
	}
	err := authUser.DefinePassword("v3rySecure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mem.CreateAuthUser(authUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userUsecase := userUC.NewUserUsecases(mem, nil, nil, nil)
	authenticationUsecase := usecase.NewAuthenticationUsecase(&happydns.Options{}, mem, userUsecase)

	_, err = authenticationUsecase.AuthenticateUserWithPassword(happydns.LoginRequest{
		Email:    "a@b.c",
		Password: "v3rySecure",
	})
	if err == nil || err.Error() != `tries to login as "a@b.c", but has not verified email` {
		t.Errorf("unexpected error: %v", err)
	}
}

func Test_AuthenticateUserWithPassword_NoEmail(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()

	authUser := &happydns.UserAuth{
		Email: "a@b.c",
	}
	err := authUser.DefinePassword("v3rySecure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mem.CreateAuthUser(authUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userUsecase := userUC.NewUserUsecases(mem, nil, nil, nil)
	authenticationUsecase := usecase.NewAuthenticationUsecase(&happydns.Options{NoMail: true}, mem, userUsecase)

	_, err = authenticationUsecase.AuthenticateUserWithPassword(happydns.LoginRequest{
		Email:    "a@b.c",
		Password: "v3rySecure",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func Test_AuthenticateUserWithPassword(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()

	now := time.Now()
	authUser := &happydns.UserAuth{
		Email:             "a@b.c",
		EmailVerification: &now,
	}
	err := authUser.DefinePassword("v3rySecure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = mem.CreateAuthUser(authUser)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	userUsecase := userUC.NewUserUsecases(mem, nil, nil, nil)
	authenticationUsecase := usecase.NewAuthenticationUsecase(&happydns.Options{}, mem, userUsecase)

	_, err = authenticationUsecase.AuthenticateUserWithPassword(happydns.LoginRequest{
		Email:    "a@b.c",
		Password: "v3rySecure",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
