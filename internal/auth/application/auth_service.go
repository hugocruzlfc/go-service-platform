package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"service-platform/internal/auth/domain"
)

// AuthService implements authentication use cases.
type AuthService struct {
	registrationRepository RegistrationRepository
	passwordHasher         PasswordHasher
}

// RegisterInput contains the credentials needed to register a user.
type RegisterInput struct {
	Email    string
	Password string
}

// NewAuthService creates an authentication service.
func NewAuthService(
	registrationRepository RegistrationRepository,
	passwordHasher PasswordHasher,
) *AuthService {
	return &AuthService{
		registrationRepository: registrationRepository,
		passwordHasher:         passwordHasher,
	}
}

// Register creates a user and stores its password hash atomically.
func (s *AuthService) Register(
	ctx context.Context,
	input RegisterInput,
) (domain.User, error) {
	createdAt := time.Now().UTC()
	userID, err := generateUserID()
	if err != nil {
		return domain.User{}, fmt.Errorf("generate user ID: %w", err)
	}

	user, err := domain.NewUser(userID, input.Email, createdAt)
	if err != nil {
		return domain.User{}, err
	}

	passwordHash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	if err := s.registrationRepository.CreateUserWithCredentials(
		ctx,
		user,
		passwordHash,
		createdAt,
	); err != nil {
		return domain.User{}, fmt.Errorf("register user: %w", err)
	}

	return user, nil
}

func generateUserID() (string, error) {
	var bytes [16]byte

	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
