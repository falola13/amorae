package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Session is one issued token's server-side record. Only TokenHash is ever
// written to the database (see token.go) — the token itself never touches
// storage.
type Session struct {
	TokenHash []byte
	UserID    uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
}

// errSessionNotFound is repository_postgres.go's signal that no row
// matched a token hash. It stays unexported: everything outside this
// package only ever sees the apperr.Unauthenticated that Service.Authenticate
// maps it to — "no such session" and "expired session" should look
// identical to a caller.
var errSessionNotFound = errors.New("session not found")
