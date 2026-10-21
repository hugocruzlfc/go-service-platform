package application

// PasswordHasher hashes passwords and verifies them against encoded hashes.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(password, encodedHash string) error
}
