package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const tokenBytes = 32

// NewToken generates a session token: 32 bytes from crypto/rand,
// base64url-encoded. It's the default Service uses in production; tests
// inject a deterministic function instead so token values are predictable.
func NewToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken returns the SHA-256 digest of a token. Only this digest is ever
// stored in the sessions table (see repository_postgres.go) — a database
// leak alone can't be turned back into a usable token, since SHA-256 isn't
// reversible.
func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
