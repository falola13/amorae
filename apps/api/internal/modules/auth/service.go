// Package auth owns authentication use cases: registering, logging in and
// out, and resolving a bearer token back to a user id. It imports user (for
// the User entity and its validation) but user never imports auth — see
// internal/platform/authctx for why that direction doesn't create a cycle.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// UserRepository lists only Create and GetByEmail — what auth's use cases
// actually call — even though the concrete postgres repository behind it
// also implements GetByID and Update for user.Service. Each consumer
// declares its own narrow interface (ISP); there's still exactly one
// implementation.
type UserRepository interface {
	Create(ctx context.Context, u user.User) (user.User, error)
	GetByEmail(ctx context.Context, email string) (user.User, error)
	SetLastLoginAt(ctx context.Context, id uuid.UUID, at time.Time) error
}

type SessionRepository interface {
	Create(ctx context.Context, s Session) error
	GetByTokenHash(ctx context.Context, hash []byte) (Session, error)
	Delete(ctx context.Context, hash []byte) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TxRunner is what Register needs to create a user and a session
// atomically. *database.DB satisfies this directly — no adapter needed.
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AttemptLimiter caps login attempts per account, so a password can't be
// guessed by brute force even from many IPs. ratelimit.Limiter satisfies it.
type AttemptLimiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

var (
	ErrInvalidCredentials = apperr.Unauthenticated("invalid_credentials", "Invalid email or password.")
	ErrUnauthenticated    = apperr.Unauthenticated("unauthenticated", "Authentication required.")
)

const (
	minPasswordBytes = 8
	maxPasswordBytes = 72 // bcrypt ignores bytes beyond 72; reject instead of silently truncating
)

type Service struct {
	users    UserRepository
	sessions SessionRepository
	hasher   PasswordHasher
	tx       TxRunner
	ttl      time.Duration
	now      func() time.Time
	newToken func() (string, error)
	attempts AttemptLimiter

	// dummyHash is compared against on every failed login where the email
	// doesn't exist, so that path costs the same bcrypt work as a wrong
	// password on a real account — see Login.
	dummyHash string
}

func NewService(
	users UserRepository,
	sessions SessionRepository,
	hasher PasswordHasher,
	tx TxRunner,
	ttl time.Duration,
	now func() time.Time,
	newToken func() (string, error),
	attempts AttemptLimiter,
) (*Service, error) {
	dummyHash, err := hasher.Hash("amorae-timing-equalisation-dummy-password")
	if err != nil {
		return nil, fmt.Errorf("hashing dummy password: %w", err)
	}

	return &Service{
		users:     users,
		sessions:  sessions,
		hasher:    hasher,
		tx:        tx,
		ttl:       ttl,
		now:       now,
		newToken:  newToken,
		attempts:  attempts,
		dummyHash: dummyHash,
	}, nil
}

type RegisterInput struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      user.User `json:"user"`
}

// Register validates the input, hashes the password, and creates the user
// and their first session in one transaction — a user should never exist
// without a way to log in, and vice versa.
func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	fields := map[string]string{}
	if n := len(input.Password); n < minPasswordBytes || n > maxPasswordBytes {
		fields["password"] = fmt.Sprintf("Must be between %d and %d characters.", minPasswordBytes, maxPasswordBytes)
	}

	u, err := user.New(input.Email, input.DisplayName, "", s.now())
	if err != nil {
		appErr, ok := apperr.As(err)
		if !ok || appErr.Kind != apperr.KindInvalid {
			return AuthResult{}, err
		}
		for field, msg := range appErr.Fields {
			fields[field] = msg
		}
	}

	if len(fields) > 0 {
		return AuthResult{}, apperr.Validation(fields)
	}

	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return AuthResult{}, apperr.Internal(fmt.Errorf("hashing password: %w", err))
	}
	u.PasswordHash = hash

	token, err := s.newToken()
	if err != nil {
		return AuthResult{}, apperr.Internal(fmt.Errorf("generating token: %w", err))
	}

	session := Session{
		TokenHash: hashToken(token),
		UserID:    u.ID,
		CreatedAt: s.now(),
		ExpiresAt: s.now().Add(s.ttl),
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		created, err := s.users.Create(ctx, u)
		if err != nil {
			return err
		}
		u = created
		return s.sessions.Create(ctx, session)
	})
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{Token: token, ExpiresAt: session.ExpiresAt, User: u}, nil
}

// Login never reveals whether an email is registered: an unknown email and
// a wrong password both return ErrInvalidCredentials, and both do the same
// amount of bcrypt work first (see dummyHash), so neither the response nor
// its timing leaks which case occurred.
func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	email := user.NormalizeEmail(input.Email)

	// Checked before any database or bcrypt work, so a blocked guess costs
	// nothing. The key is the email as typed, registered or not, so hitting
	// the limit reveals nothing about which accounts exist.
	if ok, retryAfter := s.attempts.Allow("login:" + email); !ok {
		return AuthResult{}, apperr.RateLimited(retryAfter)
	}

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, user.ErrNotFound) {
			return AuthResult{}, err
		}
		_ = s.hasher.Compare(s.dummyHash, input.Password)
		return AuthResult{}, ErrInvalidCredentials
	}

	if err := s.hasher.Compare(u.PasswordHash, input.Password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	token, err := s.newToken()
	if err != nil {
		return AuthResult{}, apperr.Internal(fmt.Errorf("generating token: %w", err))
	}

	now := s.now()
	session := Session{
		TokenHash: hashToken(token),
		UserID:    u.ID,
		CreatedAt: s.now(),
		ExpiresAt: s.now().Add(s.ttl),
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.users.SetLastLoginAt(ctx, u.ID, now); err != nil {

			return err
		}
		return s.sessions.Create(ctx, session)
	})
	if err != nil {
		// The transaction rolled back, so the session doesn't exist: handing
		// out the token anyway would "log in" a user whose every request 401s.
		return AuthResult{}, err
	}

	return AuthResult{Token: token, ExpiresAt: session.ExpiresAt, User: u}, nil
}

// Authenticate resolves a bearer token to the user id that owns it. A
// missing session and an expired one produce the identical error — telling
// them apart isn't useful to a caller and would just be another way to
// leak information about what did or didn't exist.
func (s *Service) Authenticate(ctx context.Context, token string) (uuid.UUID, error) {
	hash := hashToken(token)

	session, err := s.sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			return uuid.UUID{}, ErrUnauthenticated
		}
		return uuid.UUID{}, err
	}

	if !s.now().Before(session.ExpiresAt) {
		_ = s.sessions.Delete(ctx, hash) // best-effort; an expired session must never authenticate regardless
		return uuid.UUID{}, ErrUnauthenticated
	}

	return session.UserID, nil
}

// Logout deletes the session for token. Deleting a row that doesn't exist
// isn't an error for SQL, which is what makes this idempotent for free —
// no special-casing "already logged out" is needed.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}
