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
	TokenHash []byte    `json:"-"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	// Nil until the session is used again after it was created.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// The browser's user agent, kept only so a person can recognise their own
	// sessions. Never returned as-is: handlers send deviceLabel(UserAgent).
	UserAgent string `json:"-"`
}

// SessionView is one row of "where you're signed in", for the owner of the
// session. It carries no token hash and no raw user agent: nothing here
// identifies a session to anyone but the person looking at their own list.
type SessionView struct {
	Current    bool
	Device     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  time.Time
}

// errSessionNotFound is repository_postgres.go's signal that no row
// matched a token hash. It stays unexported: everything outside this
// package only ever sees the apperr.Unauthenticated that Service.Authenticate
// maps it to — "no such session" and "expired session" should look
// identical to a caller.
var errSessionNotFound = errors.New("session not found")

// errResetInvalid covers an unknown, used and expired reset link alike: the
// person can only ask for a new one, so there's nothing to tell apart.
var errResetInvalid = errors.New("password reset not found")

// Consent kinds, one row each in user_consents.
const (
	ConsentTerms   = "terms"
	ConsentPrivacy = "privacy"
	ConsentAge18   = "age_18"
	ConsentFaith   = "faith_content"
)

// Consent is one thing a person agreed to, with the policy version they saw.
type Consent struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Kind          string
	PolicyVersion string
	CreatedAt     time.Time
}
