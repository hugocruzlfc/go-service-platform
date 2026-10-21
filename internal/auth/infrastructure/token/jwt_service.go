package token

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"service-platform/internal/auth/application"
	"service-platform/internal/auth/domain"
)

var _ application.TokenService = (*Service)(nil)

// Service signs and validates JWT access tokens with Ed25519.
type Service struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	issuer     string
	audience   string
	accessTTL  time.Duration
}

// NewService creates a JWT service with the supplied signing configuration.
func NewService(
	privateKey ed25519.PrivateKey,
	issuer string,
	audience string,
	accessTTL time.Duration,
) (*Service, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("token: invalid Ed25519 private key")
	}
	if strings.TrimSpace(issuer) == "" {
		return nil, errors.New("token: issuer is required")
	}
	if strings.TrimSpace(audience) == "" {
		return nil, errors.New("token: audience is required")
	}
	if accessTTL <= 0 {
		return nil, errors.New("token: access token lifetime must be positive")
	}

	privateKeyCopy := append(ed25519.PrivateKey(nil), privateKey...)
	publicKey := privateKeyCopy.Public().(ed25519.PublicKey)

	return &Service{
		privateKey: privateKeyCopy,
		publicKey:  append(ed25519.PublicKey(nil), publicKey...),
		issuer:     issuer,
		audience:   audience,
		accessTTL:  accessTTL,
	}, nil
}

// IssueAccessToken creates a short-lived JWT for an authenticated session.
func (s *Service) IssueAccessToken(input application.AccessTokenInput) (string, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return "", domain.ErrTokenSubjectRequired
	}

	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return "", domain.ErrTokenSessionRequired
	}

	tokenID, err := generateTokenID()
	if err != nil {
		return "", fmt.Errorf("generate token ID: %w", err)
	}

	issuedAt := time.Now().UTC()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ID:        tokenID,
		},
		SessionID:   sessionID,
		Roles:       append([]string(nil), input.Roles...),
		Permissions: append([]string(nil), input.Permissions...),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	encodedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return encodedToken, nil
}

// ParseAccessToken validates an encoded JWT and returns its claims.
func (s *Service) ParseAccessToken(encodedToken string) (application.AccessTokenClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(
		encodedToken,
		&jwtClaims{},
		func(_ *jwt.Token) (any, error) {
			return s.publicKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(s.audience),
	)
	if err != nil || !parsedToken.Valid {
		return application.AccessTokenClaims{}, domain.ErrAccessTokenInvalid
	}

	claims, ok := parsedToken.Claims.(*jwtClaims)
	if !ok || claims.Subject == "" || claims.SessionID == "" || claims.ID == "" {
		return application.AccessTokenClaims{}, domain.ErrAccessTokenInvalid
	}

	return application.AccessTokenClaims{
		UserID:      claims.Subject,
		SessionID:   claims.SessionID,
		TokenID:     claims.ID,
		Roles:       append([]string(nil), claims.Roles...),
		Permissions: append([]string(nil), claims.Permissions...),
	}, nil
}

type jwtClaims struct {
	SessionID   string   `json:"sid"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

func generateTokenID() (string, error) {
	var bytes [16]byte

	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
