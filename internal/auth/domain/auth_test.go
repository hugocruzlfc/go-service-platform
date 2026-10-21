package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewUserNormalizesEmailAndTimestamp(t *testing.T) {
	createdAt := time.Date(
		2026, time.October, 21,
		10, 0, 0, 0,
		time.FixedZone("UTC+2", 2*60*60),
	)

	user, err := NewUser("user-1", "  Alice@Example.COM ", createdAt)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected normalized email %q, got %q", "alice@example.com", user.Email)
	}
	if user.CreatedAt.Location() != time.UTC {
		t.Error("expected creation time to be normalized to UTC")
	}
	if user.Disabled {
		t.Error("expected a new user to be enabled")
	}
}

func TestNewUserRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		email   string
		wantErr error
	}{
		{name: "missing ID", id: " ", email: "alice@example.com", wantErr: ErrUserIDRequired},
		{name: "missing email", id: "user-1", email: " ", wantErr: ErrUserEmailRequired},
		{name: "invalid email", id: "user-1", email: "not-an-email", wantErr: ErrUserEmailInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewUser(test.id, test.email, time.Now())

			if !errors.Is(err, test.wantErr) {
				t.Errorf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestNewRoleAndPermissionNormalizeNames(t *testing.T) {
	role, err := NewRole(" Admin ")
	if err != nil {
		t.Fatalf("expected no error creating role, got %v", err)
	}
	if role.Name != "admin" {
		t.Errorf("expected role name %q, got %q", "admin", role.Name)
	}

	permission, err := NewPermission(" Tasks:Read ")
	if err != nil {
		t.Fatalf("expected no error creating permission, got %v", err)
	}
	if permission.Name != "tasks:read" {
		t.Errorf("expected permission name %q, got %q", "tasks:read", permission.Name)
	}
}

func TestNewSessionStartsActiveAndExpiresAfterCreation(t *testing.T) {
	createdAt := time.Date(2026, time.October, 21, 10, 0, 0, 0, time.UTC)
	expiresAt := createdAt.Add(24 * time.Hour)

	session, err := NewSession(
		"session-1",
		"user-1",
		"family-1",
		"refresh-hash",
		createdAt,
		expiresAt,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if session.LastUsedAt != createdAt {
		t.Errorf("expected last used time %v, got %v", createdAt, session.LastUsedAt)
	}
	if session.IsExpired(createdAt.Add(time.Hour)) {
		t.Error("expected session to be active before expiry")
	}
	if !session.IsExpired(expiresAt) {
		t.Error("expected session to expire at its expiry time")
	}
	if session.IsRevoked() {
		t.Error("expected new session not to be revoked")
	}
}

func TestSessionRevokeMarksRevocationTime(t *testing.T) {
	createdAt := time.Date(2026, time.October, 21, 10, 0, 0, 0, time.UTC)
	session, err := NewSession(
		"session-1",
		"user-1",
		"family-1",
		"refresh-hash",
		createdAt,
		createdAt.Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	session.Revoke(createdAt.Add(time.Hour))

	if !session.IsRevoked() {
		t.Error("expected session to be revoked")
	}
	if session.RevokedAt == nil || !session.RevokedAt.Equal(createdAt.Add(time.Hour)) {
		t.Errorf("expected revocation time %v, got %v", createdAt.Add(time.Hour), session.RevokedAt)
	}
}

func TestNewSessionRejectsInvalidExpiry(t *testing.T) {
	createdAt := time.Date(2026, time.October, 21, 10, 0, 0, 0, time.UTC)

	_, err := NewSession(
		"session-1",
		"user-1",
		"family-1",
		"refresh-hash",
		createdAt,
		createdAt,
	)

	if !errors.Is(err, ErrSessionExpiryInvalid) {
		t.Errorf("expected ErrSessionExpiryInvalid, got %v", err)
	}
}
