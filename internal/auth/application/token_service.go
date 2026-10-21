package application

// AccessTokenInput contains the identity and permissions encoded in an access token.
type AccessTokenInput struct {
	UserID      string
	SessionID   string
	Roles       []string
	Permissions []string
}

// AccessTokenClaims contains the validated claims extracted from an access token.
type AccessTokenClaims struct {
	UserID      string
	SessionID   string
	TokenID     string
	Roles       []string
	Permissions []string
}

// TokenService issues and validates access tokens.
type TokenService interface {
	IssueAccessToken(input AccessTokenInput) (string, error)
	ParseAccessToken(encodedToken string) (AccessTokenClaims, error)
}
