package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"service-platform/internal/auth/domain"
)

type fakeRegistrationRepository struct {
	createdUser         domain.User
	createdPasswordHash string
	credentialsUpdated  time.Time
	createErr           error
}

func (r *fakeRegistrationRepository) CreateUserWithCredentials(
	_ context.Context,
	user domain.User,
	passwordHash string,
	updatedAt time.Time,
) error {
	if r.createErr != nil {
		return r.createErr
	}

	r.createdUser = user
	r.createdPasswordHash = passwordHash
	r.credentialsUpdated = updatedAt
	return nil
}

type fakePasswordHasher struct {
	hash string
	err  error
}

func (h fakePasswordHasher) Hash(string) (string, error) {
	return h.hash, h.err
}

func (fakePasswordHasher) Compare(string, string) error {
	return nil
}

func TestAuthServiceRegisterCreatesUserWithPasswordHash(t *testing.T) {
	repository := &fakeRegistrationRepository{}
	service := NewAuthService(repository, fakePasswordHasher{hash: "encoded-hash"})

	user, err := service.Register(t.Context(), RegisterInput{
		Email:    " Alice@Example.com ",
		Password: "secret",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID == "" {
		t.Error("expected generated user ID")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected normalized email, got %q", user.Email)
	}
	if repository.createdPasswordHash != "encoded-hash" {
		t.Errorf("expected password hash to be stored, got %q", repository.createdPasswordHash)
	}
	if repository.createdUser != user {
		t.Errorf("expected repository to receive user %#v, got %#v", user, repository.createdUser)
	}
	if repository.credentialsUpdated.IsZero() {
		t.Error("expected credentials update time")
	}
}

func TestAuthServiceRegisterDoesNotPersistWhenPasswordHashingFails(t *testing.T) {
	repository := &fakeRegistrationRepository{}
	hashErr := errors.New("hash unavailable")
	service := NewAuthService(repository, fakePasswordHasher{err: hashErr})

	_, err := service.Register(t.Context(), RegisterInput{
		Email:    "alice@example.com",
		Password: "secret",
	})

	if !errors.Is(err, hashErr) {
		t.Errorf("expected hashing error, got %v", err)
	}
	if repository.createdUser.ID != "" {
		t.Error("expected repository not to be called when hashing fails")
	}
}

func TestAuthServiceRegisterReturnsRepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")
	repository := &fakeRegistrationRepository{createErr: repositoryErr}
	service := NewAuthService(repository, fakePasswordHasher{hash: "encoded-hash"})

	_, err := service.Register(t.Context(), RegisterInput{
		Email:    "alice@example.com",
		Password: "secret",
	})

	if !errors.Is(err, repositoryErr) {
		t.Errorf("expected repository error, got %v", err)
	}
}

func TestAuthServiceRegisterRejectsInvalidInput(t *testing.T) {
	repository := &fakeRegistrationRepository{}
	service := NewAuthService(repository, fakePasswordHasher{hash: "encoded-hash"})

	_, err := service.Register(t.Context(), RegisterInput{
		Email:    "not-an-email",
		Password: "secret",
	})

	if !errors.Is(err, domain.ErrUserEmailInvalid) {
		t.Errorf("expected ErrUserEmailInvalid, got %v", err)
	}
	if repository.createdUser.ID != "" {
		t.Error("expected repository not to be called for invalid input")
	}
}
