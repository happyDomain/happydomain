package authuser_test

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/mailer"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

type dummyEmailValidation struct {
	err error
}

func (d *dummyEmailValidation) GenerateLink(u *happydns.UserAuth) string {
	return ""
}

func (d *dummyEmailValidation) SendLink(u *happydns.UserAuth) error {
	return d.err
}

func (d *dummyEmailValidation) Validate(user *happydns.UserAuth, form happydns.AddressValidationForm) error {
	return d.err
}

func TestCreateAuthUser_Success(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	pwChecker := authuser.NewCheckPasswordConstraintsUsecase()
	emailValidation := &dummyEmailValidation{}
	usecase := authuser.NewCreateAuthUserUsecase(store, &mailer.Mailer{}, pwChecker, emailValidation)

	reg := happydns.UserRegistration{
		Email:      "test@example.com",
		Password:   "StrongPassword123!",
		Newsletter: true,
	}

	user, err := usecase.Create(reg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != reg.Email {
		t.Errorf("expected email %s, got %s", reg.Email, user.Email)
	}
	if user.Password == nil {
		t.Errorf("expected defined password, got %s", user.Password)
	}
	if !user.AllowCommercials {
		t.Error("expected user to have AllowCommercials = true")
	}
}

func TestCreateAuthUser_InvalidEmail(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	pwChecker := authuser.NewCheckPasswordConstraintsUsecase()
	usecase := authuser.NewCreateAuthUserUsecase(store, &mailer.Mailer{}, pwChecker, &dummyEmailValidation{})

	reg := happydns.UserRegistration{
		Email:    "bademail",
		Password: "StrongPassword123!",
	}

	_, err := usecase.Create(reg)
	if err == nil || err.Error() != "the given email is invalid" {
		t.Errorf("expected validation error for email, got: %v", err)
	}
}

func TestCreateAuthUser_WeakPassword(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	pwChecker := authuser.NewCheckPasswordConstraintsUsecase()
	usecase := authuser.NewCreateAuthUserUsecase(store, &mailer.Mailer{}, pwChecker, &dummyEmailValidation{})

	reg := happydns.UserRegistration{
		Email:    "test@example.com",
		Password: "123",
	}

	_, err := usecase.Create(reg)
	if err == nil || err.Error() != "password must be at least 8 characters long" {
		t.Errorf("expected password constraint error, got: %v", err)
	}

	reg.Password = "Secur3$"
	_, err = usecase.Create(reg)
	if err == nil || err.Error() != "password must be at least 8 characters long" {
		t.Errorf("expected password constraint error, got: %v", err)
	}

	reg.Password = "secure123"
	_, err = usecase.Create(reg)
	if err == nil || err.Error() != "Password must contain upper case letters." {
		t.Errorf("expected password constraint error, got: %v", err)
	}

	reg.Password = "Secure123"
	_, err = usecase.Create(reg)
	if err == nil || err.Error() != "Password must be longer or contain symbols." {
		t.Errorf("expected password constraint error, got: %v", err)
	}
}

func TestCreateAuthUser_EmailAlreadyUsed(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	pwChecker := authuser.NewCheckPasswordConstraintsUsecase()
	usecase := authuser.NewCreateAuthUserUsecase(store, &mailer.Mailer{}, pwChecker, &dummyEmailValidation{})

	// Create a user first
	reg := happydns.UserRegistration{
		Email:    "used@example.com",
		Password: "StrongPassword123!",
	}
	_, err := usecase.Create(reg)
	if err != nil {
		t.Fatalf("setup user creation failed: %v", err)
	}

	// Try creating again with the same email
	_, err = usecase.Create(reg)
	if err == nil || err.Error() != "an account already exists with the given address. Try logging in." {
		t.Errorf("expected duplicate email error, got: %v", err)
	}
}

func TestCreateAuthUser_EmailValidationFails(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	pwChecker := authuser.NewCheckPasswordConstraintsUsecase()
	emailValidation := &dummyEmailValidation{err: errors.New("SMTP error")}
	usecase := authuser.NewCreateAuthUserUsecase(store, &mailer.Mailer{}, pwChecker, emailValidation)

	reg := happydns.UserRegistration{
		Email:    "fail@example.com",
		Password: "StrongPassword123!",
	}

	_, err := usecase.Create(reg)
	if err == nil || err.Error() != "unable to send validation email: SMTP error" {
		t.Errorf("expected internal error for email sending, got: %v", err)
	}
}
