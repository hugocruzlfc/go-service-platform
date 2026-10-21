package argon2

import (
	"errors"
	"strings"
	"testing"

	"service-platform/internal/auth/domain"
)

func TestPasswordHasherHashAndCompare(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "correct horse battery staple"

	encodedHash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("expected no error hashing password, got %v", err)
	}

	if encodedHash == password {
		t.Fatal("expected password not to be stored in plaintext")
	}
	if !strings.HasPrefix(encodedHash, "$argon2id$") {
		t.Fatalf("expected Argon2id PHC hash, got %q", encodedHash)
	}
	if err := hasher.Compare(password, encodedHash); err != nil {
		t.Fatalf("expected password to match, got %v", err)
	}
}

func TestPasswordHasherUsesDifferentSaltForEachHash(t *testing.T) {
	hasher := NewPasswordHasher()

	firstHash, err := hasher.Hash("same password")
	if err != nil {
		t.Fatalf("hashing first password: %v", err)
	}
	secondHash, err := hasher.Hash("same password")
	if err != nil {
		t.Fatalf("hashing second password: %v", err)
	}

	if firstHash == secondHash {
		t.Fatal("expected hashes with random salts to differ")
	}
}

func TestPasswordHasherRejectsWrongPassword(t *testing.T) {
	hasher := NewPasswordHasher()

	encodedHash, err := hasher.Hash("correct password")
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	if err := hasher.Compare("wrong password", encodedHash); !errors.Is(err, domain.ErrPasswordMismatch) {
		t.Errorf("expected ErrPasswordMismatch, got %v", err)
	}
}

func TestPasswordHasherRejectsEmptyPassword(t *testing.T) {
	hasher := NewPasswordHasher()

	if _, err := hasher.Hash(""); !errors.Is(err, domain.ErrPasswordRequired) {
		t.Errorf("expected ErrPasswordRequired from Hash, got %v", err)
	}
	if err := hasher.Compare("", "invalid"); !errors.Is(err, domain.ErrPasswordRequired) {
		t.Errorf("expected ErrPasswordRequired from Compare, got %v", err)
	}
}

func TestPasswordHasherRejectsMalformedHash(t *testing.T) {
	hasher := NewPasswordHasher()

	if err := hasher.Compare("password", "not-a-hash"); !errors.Is(err, domain.ErrPasswordHashInvalid) {
		t.Errorf("expected ErrPasswordHashInvalid, got %v", err)
	}
}
