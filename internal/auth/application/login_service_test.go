package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"service-platform/internal/auth/domain"
)

type fakeUserRepository struct {
	user domain.User
	err  error
}

func (r fakeUserRepository) Create(context.Context, domain.User) error { return nil }

func (r fakeUserRepository) GetByID(context.Context, string) (domain.User, error) {
	return r.user, r.err
}

func (r fakeUserRepository) GetByEmail(context.Context, string) (domain.User, error) {
	return r.user, r.err
}

type fakeCredentialRepository struct {
	hash string
	err  error
}

func (r fakeCredentialRepository) Create(context.Context, string, string, time.Time) error {
	return nil
}

func (r fakeCredentialRepository) GetPasswordHash(context.Context, string) (string, error) {
	return r.hash, r.err
}

type loginPasswordHasher struct {
	compareErr error
}

func (loginPasswordHasher) Hash(string) (string, error) { return "", nil }

func (h loginPasswordHasher) Compare(string, string) error { return h.compareErr }

type fakeSessionRepository struct {
	session   domain.Session
	err       error
	rotateErr error
	revokedAt time.Time
}

func (r *fakeSessionRepository) Create(_ context.Context, session domain.Session) error {
	if r.err == nil {
		r.session = session
	}
	return r.err
}

func (r *fakeSessionRepository) GetByID(context.Context, string) (domain.Session, error) {
	return r.session, nil
}

func (r *fakeSessionRepository) Rotate(
	_ context.Context,
	_ string,
	_ string,
	rotatedSession domain.Session,
) error {
	if r.rotateErr == nil {
		r.session = rotatedSession
	}
	return r.rotateErr
}

func (r *fakeSessionRepository) Revoke(_ context.Context, _ string, revokedAt time.Time) error {
	r.revokedAt = revokedAt
	return nil
}

func (*fakeSessionRepository) RevokeFamily(context.Context, string, time.Time) error { return nil }

type fakeRefreshTokenManager struct{}

func (fakeRefreshTokenManager) Issue(sessionID string) (string, string, error) {
	return "refresh-" + sessionID, "refresh-hash", nil
}

func (fakeRefreshTokenManager) Parse(string) (string, string, error) {
	return "", "", domain.ErrRefreshTokenInvalid
}

type fakeAccessTokenService struct {
	input AccessTokenInput
	err   error
}

func (s *fakeAccessTokenService) IssueAccessToken(input AccessTokenInput) (string, error) {
	s.input = input
	return "access-token", s.err
}

func (*fakeAccessTokenService) ParseAccessToken(string) (AccessTokenClaims, error) {
	return AccessTokenClaims{}, nil
}

func TestLoginServiceCreatesSessionAndTokens(t *testing.T) {
	user := domain.User{ID: "user-1", Email: "alice@example.com"}
	sessions := &fakeSessionRepository{}
	accessTokens := &fakeAccessTokenService{}
	service := NewLoginService(
		fakeUserRepository{user: user},
		fakeCredentialRepository{hash: "encoded-hash"},
		loginPasswordHasher{},
		sessions,
		fakeRefreshTokenManager{},
		accessTokens,
	)

	result, err := service.Login(t.Context(), LoginInput{
		Email:    user.Email,
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.User != user {
		t.Fatalf("user = %#v, want %#v", result.User, user)
	}
	if result.AccessToken != "access-token" {
		t.Fatalf("access token = %q", result.AccessToken)
	}
	if result.RefreshToken == "" {
		t.Fatal("expected refresh token")
	}
	if sessions.session.ID == "" {
		t.Fatal("expected session to be persisted")
	}
	if accessTokens.input.UserID != user.ID || accessTokens.input.SessionID != sessions.session.ID {
		t.Fatalf("access token input = %#v", accessTokens.input)
	}
}

func TestLoginServiceReturnsGenericErrorForInvalidPassword(t *testing.T) {
	sessions := &fakeSessionRepository{}
	service := NewLoginService(
		fakeUserRepository{user: domain.User{ID: "user-1"}},
		fakeCredentialRepository{hash: "encoded-hash"},
		loginPasswordHasher{compareErr: errors.New("password mismatch")},
		sessions,
		fakeRefreshTokenManager{},
		&fakeAccessTokenService{},
	)

	_, err := service.Login(t.Context(), LoginInput{Password: "wrong"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
	if sessions.session.ID != "" {
		t.Fatal("expected invalid login not to create a session")
	}
}

func TestLoginServiceReturnsGenericErrorWhenUserDoesNotExist(t *testing.T) {
	service := NewLoginService(
		fakeUserRepository{err: domain.ErrUserNotFound},
		fakeCredentialRepository{},
		loginPasswordHasher{},
		&fakeSessionRepository{},
		fakeRefreshTokenManager{},
		&fakeAccessTokenService{},
	)

	_, err := service.Login(t.Context(), LoginInput{Email: "missing@example.com"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginServiceRejectsDisabledUser(t *testing.T) {
	service := NewLoginService(
		fakeUserRepository{user: domain.User{ID: "user-1", Disabled: true}},
		fakeCredentialRepository{},
		loginPasswordHasher{},
		&fakeSessionRepository{},
		fakeRefreshTokenManager{},
		&fakeAccessTokenService{},
	)

	_, err := service.Login(t.Context(), LoginInput{})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}
