package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

const (
	argon2Version = 19
	memory        = 64 * 1024
	iterations    = 3
	parallelism   = 1
	saltLength    = 16
	keyLength     = 32
)

var _ application.PasswordHasher = (*PasswordHasher)(nil)

// PasswordHasher implements password hashing with Argon2id.
type PasswordHasher struct{}

// NewPasswordHasher creates an Argon2id password hasher.
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

// Hash returns a PHC-encoded Argon2id hash for password.
func (h *PasswordHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", domain.ErrPasswordRequired
	}

	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	derivedKey := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		keyLength,
	)

	return encodeHash(salt, derivedKey), nil
}

// Compare verifies password against an encoded Argon2id hash.
func (h *PasswordHasher) Compare(password, encodedHash string) error {
	if password == "" {
		return domain.ErrPasswordRequired
	}

	parameters, salt, expectedKey, err := parseHash(encodedHash)
	if err != nil {
		return domain.ErrPasswordHashInvalid
	}

	actualKey := argon2.IDKey(
		[]byte(password),
		salt,
		parameters.iterations,
		parameters.memory,
		parameters.parallelism,
		keyLength,
	)

	if subtle.ConstantTimeCompare(actualKey, expectedKey) != 1 {
		return domain.ErrPasswordMismatch
	}

	return nil
}

type parameters struct {
	iterations  uint32
	memory      uint32
	parallelism uint8
}

func encodeHash(salt, derivedKey []byte) string {
	encoding := base64.RawStdEncoding

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version,
		memory,
		iterations,
		parallelism,
		encoding.EncodeToString(salt),
		encoding.EncodeToString(derivedKey),
	)
}

func parseHash(encodedHash string) (parameters, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return parameters{}, nil, nil, errors.New("invalid Argon2id format")
	}

	parsedParameters, err := parseParameters(parts[3])
	if err != nil {
		return parameters{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return parameters{}, nil, nil, errors.New("invalid Argon2id salt")
	}

	derivedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(derivedKey) != keyLength {
		return parameters{}, nil, nil, errors.New("invalid Argon2id key")
	}

	return parsedParameters, salt, derivedKey, nil
}

func parseParameters(value string) (parameters, error) {
	var parsed parameters

	for _, part := range strings.Split(value, ",") {
		key, rawValue, ok := strings.Cut(part, "=")
		if !ok {
			return parameters{}, errors.New("invalid Argon2id parameters")
		}

		value, err := strconv.ParseUint(rawValue, 10, 32)
		if err != nil {
			return parameters{}, errors.New("invalid Argon2id parameter value")
		}

		switch key {
		case "m":
			parsed.memory = uint32(value)
		case "t":
			parsed.iterations = uint32(value)
		case "p":
			if value > 255 {
				return parameters{}, errors.New("invalid Argon2id parallelism")
			}
			parsed.parallelism = uint8(value)
		default:
			return parameters{}, errors.New("unknown Argon2id parameter")
		}
	}

	if parsed.memory == 0 || parsed.iterations == 0 || parsed.parallelism == 0 {
		return parameters{}, errors.New("incomplete Argon2id parameters")
	}

	return parsed, nil
}
