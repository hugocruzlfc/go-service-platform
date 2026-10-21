package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

var (
	// ErrUserIDRequired indicates that a user ID was not provided.
	ErrUserIDRequired = errors.New("auth: user ID is required")
	// ErrUserEmailRequired indicates that a user email was not provided.
	ErrUserEmailRequired = errors.New("auth: user email is required")
	// ErrUserEmailInvalid indicates that a user email is not valid.
	ErrUserEmailInvalid = errors.New("auth: user email is invalid")
	// ErrUserEmailTaken indicates that a user email is already registered.
	ErrUserEmailTaken = errors.New("auth: user email is already registered")
	// ErrUserNotFound indicates that a user does not exist.
	ErrUserNotFound = errors.New("auth: user not found")
	// ErrCredentialsNotFound indicates that user credentials do not exist.
	ErrCredentialsNotFound = errors.New("auth: user credentials not found")
	// ErrPasswordRequired indicates that a password was not provided.
	ErrPasswordRequired = errors.New("auth: password is required")
	// ErrPasswordMismatch indicates that a password does not match its hash.
	ErrPasswordMismatch = errors.New("auth: password does not match")
	// ErrPasswordHashInvalid indicates that a stored password hash is invalid.
	ErrPasswordHashInvalid = errors.New("auth: password hash is invalid")
	// ErrRoleNameRequired indicates that a role name was not provided.
	ErrRoleNameRequired = errors.New("auth: role name is required")
	// ErrPermissionNameRequired indicates that a permission name was not provided.
	ErrPermissionNameRequired = errors.New("auth: permission name is required")
	// ErrSessionIDRequired indicates that a session ID was not provided.
	ErrSessionIDRequired = errors.New("auth: session ID is required")
	// ErrSessionUserIDRequired indicates that a session user ID was not provided.
	ErrSessionUserIDRequired = errors.New("auth: session user ID is required")
	// ErrSessionFamilyRequired indicates that a session token family was not provided.
	ErrSessionFamilyRequired = errors.New("auth: session token family is required")
	// ErrSessionExpiryInvalid indicates that a session expires before it is created.
	ErrSessionExpiryInvalid = errors.New("auth: session expiry must be after creation")
)

// User represents an identity that can authenticate with the service.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	Disabled  bool      `json:"disabled"`
}

// NewUser creates a user with a normalized and validated email address.
func NewUser(id, email string, createdAt time.Time) (User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return User{}, ErrUserIDRequired
	}

	normalizedEmail := normalizeEmail(email)
	if normalizedEmail == "" {
		return User{}, ErrUserEmailRequired
	}

	parsedEmail, err := mail.ParseAddress(normalizedEmail)
	if err != nil || parsedEmail.Address != normalizedEmail {
		return User{}, fmt.Errorf("%w: %q", ErrUserEmailInvalid, email)
	}

	return User{
		ID:        id,
		Email:     normalizedEmail,
		CreatedAt: createdAt.UTC(),
	}, nil
}

// Role represents a named group of permissions.
type Role struct {
	Name string `json:"name"`
}

// NewRole creates a role with a normalized name.
func NewRole(name string) (Role, error) {
	name = normalizeName(name)
	if name == "" {
		return Role{}, ErrRoleNameRequired
	}

	return Role{Name: name}, nil
}

// Permission represents one action a principal may perform.
type Permission struct {
	Name string `json:"name"`
}

// NewPermission creates a permission with a normalized name.
func NewPermission(name string) (Permission, error) {
	name = normalizeName(name)
	if name == "" {
		return Permission{}, ErrPermissionNameRequired
	}

	return Permission{Name: name}, nil
}

// Session represents a refresh-token session for an authenticated user.
type Session struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	TokenFamily string     `json:"-"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  time.Time  `json:"lastUsedAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	RefreshHash string     `json:"-"`
}

// NewSession creates an active refresh-token session.
func NewSession(
	id string,
	userID string,
	tokenFamily string,
	refreshHash string,
	createdAt time.Time,
	expiresAt time.Time,
) (Session, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Session{}, ErrSessionIDRequired
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Session{}, ErrSessionUserIDRequired
	}

	tokenFamily = strings.TrimSpace(tokenFamily)
	if tokenFamily == "" {
		return Session{}, ErrSessionFamilyRequired
	}

	createdAt = createdAt.UTC()
	expiresAt = expiresAt.UTC()
	if !expiresAt.After(createdAt) {
		return Session{}, ErrSessionExpiryInvalid
	}

	return Session{
		ID:          id,
		UserID:      userID,
		TokenFamily: tokenFamily,
		RefreshHash: strings.TrimSpace(refreshHash),
		CreatedAt:   createdAt,
		LastUsedAt:  createdAt,
		ExpiresAt:   expiresAt,
	}, nil
}

// IsExpired reports whether the session has expired at now.
func (s Session) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

// IsRevoked reports whether the session has been revoked.
func (s Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// Revoke marks the session as revoked at now.
func (s *Session) Revoke(now time.Time) {
	revokedAt := now.UTC()
	s.RevokedAt = &revokedAt
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
