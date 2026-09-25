package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Session is one issued token's server-side record. Only TokenHash is ever
// written to the database — the token itself never touches storage.
type Session struct {
	TokenHash []byte    `json:"-"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	// Nil until the session is used again after creation.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	// Never returned as-is: handlers send deviceLabel(UserAgent).
	UserAgent string `json:"-"`
}

// SessionView carries no token hash and no raw user agent, for the owner's
// own session list.
type SessionView struct {
	Current    bool
	Device     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  time.Time
}

// errSessionNotFound stays unexported: Service.Authenticate maps it to
// apperr.Unauthenticated so "no such session" and "expired session" look
// identical to callers (enumeration resistance).
var errSessionNotFound = errors.New("session not found")

// errResetInvalid covers unknown, used, and expired reset links alike, to
// avoid leaking which case applies.
var errResetInvalid = errors.New("password reset not found")

// Consent kinds, one row each in user_consents.
const (
	ConsentTerms   = "terms"
	ConsentPrivacy = "privacy"
	ConsentAge18   = "age_18"
	ConsentFaith   = "faith_content"
)

type Consent struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Kind          string
	PolicyVersion string
	CreatedAt     time.Time
}
