package application

// RefreshTokenManager creates and validates the opaque tokens used by sessions.
type RefreshTokenManager interface {
	Issue(sessionID string) (token string, hash string, err error)
	Parse(token string) (sessionID string, hash string, err error)
}
