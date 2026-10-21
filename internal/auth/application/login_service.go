package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"service-platform/internal/auth/domain"
)

const refreshTokenInactivity = 7 * 24 * time.Hour

// LoginService authenticates users and creates their token session.
type LoginService struct {
	users          UserRepository
	credentials    CredentialRepository
	passwordHasher PasswordHasher
	sessions       SessionRepository
	refreshTokens  RefreshTokenManager
	accessTokens   TokenService
	now            func() time.Time
}

// LoginInput contains the credentials used to authenticate a user.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult contains the tokens issued after successful authentication.
type LoginResult struct {
	User         domain.User
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// NewLoginService creates a service for user login and session creation.
func NewLoginService(
	users UserRepository,
	credentials CredentialRepository,
	passwordHasher PasswordHasher,
	sessions SessionRepository,
	refreshTokens RefreshTokenManager,
	accessTokens TokenService,
) *LoginService {
	return &LoginService{
		users:          users,
		credentials:    credentials,
		passwordHasher: passwordHasher,
		sessions:       sessions,
		refreshTokens:  refreshTokens,
		accessTokens:   accessTokens,
		now:            time.Now,
	}
}

// Login verifies credentials, creates a refresh session, and issues tokens.
func (s *LoginService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	user, err := s.users.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return LoginResult{}, domain.ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("get user for login: %w", err)
	}
	if user.Disabled {
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	passwordHash, err := s.credentials.GetPasswordHash(ctx, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialsNotFound) {
			return LoginResult{}, domain.ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("get credentials for login: %w", err)
	}
	if err := s.passwordHasher.Compare(input.Password, passwordHash); err != nil {
		return LoginResult{}, domain.ErrInvalidCredentials
	}

	now := s.now().UTC()
	sessionID, err := generateID()
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate session ID: %w", err)
	}
	tokenFamily, err := generateID()
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate token family: %w", err)
	}
	refreshToken, refreshHash, err := s.refreshTokens.Issue(sessionID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue refresh token: %w", err)
	}

	expiresAt := now.Add(refreshTokenInactivity)
	session, err := domain.NewSession(
		sessionID,
		user.ID,
		tokenFamily,
		refreshHash,
		now,
		expiresAt,
	)
	if err != nil {
		return LoginResult{}, fmt.Errorf("create auth session: %w", err)
	}

	accessToken, err := s.accessTokens.IssueAccessToken(AccessTokenInput{
		UserID:    user.ID,
		SessionID: session.ID,
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue access token: %w", err)
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return LoginResult{}, fmt.Errorf("persist auth session: %w", err)
	}

	return LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func generateID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}
