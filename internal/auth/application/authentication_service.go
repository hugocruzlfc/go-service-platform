package application

import (
	"context"

	"service-platform/internal/auth/domain"
)

// AuthenticationUseCase exposes the authentication operations used by HTTP.
type AuthenticationUseCase interface {
	Register(ctx context.Context, input RegisterInput) (domain.User, error)
	Login(ctx context.Context, input LoginInput) (LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (RefreshResult, error)
	Logout(ctx context.Context, refreshToken string) error
}

// AuthenticationService groups the authentication use cases behind one boundary.
type AuthenticationService struct {
	registration *AuthService
	login        *LoginService
	session      *SessionService
}

// NewAuthenticationService creates the complete authentication use case boundary.
func NewAuthenticationService(
	registration *AuthService,
	login *LoginService,
	session *SessionService,
) *AuthenticationService {
	return &AuthenticationService{
		registration: registration,
		login:        login,
		session:      session,
	}
}

// Register creates a user with credentials.
func (s *AuthenticationService) Register(ctx context.Context, input RegisterInput) (domain.User, error) {
	return s.registration.Register(ctx, input)
}

// Login authenticates a user and creates a session.
func (s *AuthenticationService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	return s.login.Login(ctx, input)
}

// Refresh rotates a refresh token and issues a new access token.
func (s *AuthenticationService) Refresh(ctx context.Context, refreshToken string) (RefreshResult, error) {
	return s.session.Refresh(ctx, refreshToken)
}

// Logout revokes the session associated with a refresh token.
func (s *AuthenticationService) Logout(ctx context.Context, refreshToken string) error {
	return s.session.Logout(ctx, refreshToken)
}

var _ AuthenticationUseCase = (*AuthenticationService)(nil)
