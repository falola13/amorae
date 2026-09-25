// Package auth owns authentication use cases. It imports user but user never
// imports auth — see internal/platform/authctx for why that avoids a cycle.
package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"net/url"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// UserRepository lists only what auth's use cases call; the concrete
// postgres repository also implements more, for user.Service (ISP).
type UserRepository interface {
	Create(ctx context.Context, u user.User) (user.User, error)
	GetByEmail(ctx context.Context, email string) (user.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (user.User, error)
	SetLastLoginAt(ctx context.Context, id uuid.UUID, at time.Time) error
	UpdateEmail(ctx context.Context, id uuid.UUID, email string, at time.Time) (user.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, at time.Time) error
	DeleteMe(ctx context.Context, id uuid.UUID) error
}

type SessionRepository interface {
	Create(ctx context.Context, s Session) error
	GetByTokenHash(ctx context.Context, hash []byte) (Session, error)
	Delete(ctx context.Context, hash []byte) error
	ListByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]Session, error)
	DeleteOthers(ctx context.Context, userID uuid.UUID, keepHash []byte) (int, error)
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
	TouchLastUsed(ctx context.Context, hash []byte, at, staleBefore time.Time) error
}

// PasswordResetRepository stores reset links by token hash. Consume returns
// errResetInvalid for an unknown, used or expired link.
type PasswordResetRepository interface {
	Create(ctx context.Context, hash []byte, userID uuid.UUID, expiresAt, at time.Time) error
	Consume(ctx context.Context, hash []byte, at time.Time) (uuid.UUID, error)
}

// ConsentRepository appends consent events; rows are never updated.
type ConsentRepository interface {
	Record(ctx context.Context, c Consent) error
}

// Mailer sends one HTML email. platform/mailer has a logging implementation
// for development and a Resend one for production.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Events counts product events for metrics — counts only, never content.
type Events interface {
	SignedUp()
}

// PolicyVersions are recorded with each consent at sign-up.
type PolicyVersions struct {
	Terms   string
	Privacy string
	Faith   string
}

// Options holds collaborators only some use cases need. Consents and Events
// default to no-ops; missing Resets/Mailer fails only password reset.
type Options struct {
	Resets   PasswordResetRepository
	Mailer   Mailer
	AppURL   string
	Consents ConsentRepository
	Policies PolicyVersions
	Events   Events
}

type noConsents struct{}

func (noConsents) Record(context.Context, Consent) error { return nil }

type noEvents struct{}

func (noEvents) SignedUp() {}

const resetTTL = time.Hour

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TxRunner lets Register create a user and session atomically.
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AttemptLimiter caps login attempts per account against brute force, even from many IPs.
type AttemptLimiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

var (
	ErrInvalidCredentials = apperr.Unauthenticated("invalid_credentials", "Invalid email or password.")
	ErrUnauthenticated    = apperr.Unauthenticated("unauthenticated", "Authentication required.")
)

// Shared with apps/web/src/lib/api/schemas.ts. Max is bytes, not chars:
// bcrypt silently ignores anything past 72 bytes.
const (
	minPasswordChars = 10
	maxPasswordBytes = 72
)

// How stale "last used" has to be before a request writes it again.
const lastUsedResolution = time.Hour

// passwordProblem returns a user-facing reason the password breaks the rule,
// or "" if it's fine.
func passwordProblem(password string) string {
	switch {
	case utf8.RuneCountInString(password) < minPasswordChars:
		return fmt.Sprintf("Use at least %d characters.", minPasswordChars)
	case len(password) > maxPasswordBytes:
		return "That’s a little long. Try a shorter password."
	}
	return ""
}

type Service struct {
	users    UserRepository
	sessions SessionRepository
	hasher   PasswordHasher
	tx       TxRunner
	ttl      time.Duration
	now      func() time.Time
	newToken func() (string, error)
	attempts AttemptLimiter
	resets   PasswordResetRepository
	mailer   Mailer
	appURL   string
	consents ConsentRepository
	policies PolicyVersions
	events   Events

	// Compared against on unknown-email logins so that path costs the same
	// bcrypt work as a wrong password (timing side-channel).
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
	opts Options,
) (*Service, error) {
	dummyHash, err := hasher.Hash("amorae-timing-equalisation-dummy-password")
	if err != nil {
		return nil, fmt.Errorf("hashing dummy password: %w", err)
	}
	if opts.Consents == nil {
		opts.Consents = noConsents{}
	}
	if opts.Events == nil {
		opts.Events = noEvents{}
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
		resets:    opts.Resets,
		mailer:    opts.Mailer,
		appURL:    opts.AppURL,
		consents:  opts.Consents,
		policies:  opts.Policies,
		events:    opts.Events,
		dummyHash: dummyHash,
	}, nil
}

type RegisterInput struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	// FR-AUTH-010/011: both required. FaithConsent is separate, optional (FR-AUTH-012).
	AgeConfirmed  bool `json:"age_confirmed"`
	AcceptedTerms bool `json:"accepted_terms"`
	FaithConsent  bool `json:"faith_consent"`
	// Set by the handler from the request, never by the client's JSON body.
	UserAgent string `json:"-"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// Set by the handler from the request, never by the client's JSON body.
	UserAgent string `json:"-"`
}

type AuthResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      user.User `json:"user"`
}

// Register creates the user and their first session in one transaction: a
// user should never exist without a way to log in.
func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	fields := map[string]string{}
	if problem := passwordProblem(input.Password); problem != "" {
		fields["password"] = problem
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
	if !input.AgeConfirmed {
		fields["age_confirmed"] = "You need to be 18 or older to use Amorae."
	}
	if !input.AcceptedTerms {
		fields["accepted_terms"] = "Agree to the Terms and Privacy Policy to continue."
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
		UserAgent: input.UserAgent,
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		created, err := s.users.Create(ctx, u)
		if err != nil {
			return err
		}
		u = created
		if err := s.recordConsents(ctx, u.ID, input.FaithConsent); err != nil {
			return err
		}
		return s.sessions.Create(ctx, session)
	})
	if err != nil {
		return AuthResult{}, err
	}

	s.events.SignedUp()
	return AuthResult{Token: token, ExpiresAt: session.ExpiresAt, User: u}, nil
}

// recordConsents runs inside the transaction that creates the account
// (FR-AUTH-011.AC2).
func (s *Service) recordConsents(ctx context.Context, userID uuid.UUID, faith bool) error {
	kinds := map[string]string{
		ConsentTerms:   s.policies.Terms,
		ConsentPrivacy: s.policies.Privacy,
		// Age check is versioned with the Terms, which set the minimum age.
		ConsentAge18: s.policies.Terms,
	}
	if faith {
		kinds[ConsentFaith] = s.policies.Faith
	}
	for kind, version := range kinds {
		id, err := uuid.NewV7()
		if err != nil {
			return apperr.Internal(fmt.Errorf("generating consent id: %w", err))
		}
		if err := s.consents.Record(ctx, Consent{
			ID: id, UserID: userID, Kind: kind, PolicyVersion: version, CreatedAt: s.now(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// Login never reveals whether an email is registered: unknown email and
// wrong password both return ErrInvalidCredentials with equal bcrypt work
// (see dummyHash), so neither response nor timing leaks which occurred.
func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	email := user.NormalizeEmail(input.Email)

	// Checked before DB/bcrypt work; keyed on email as typed either way, so
	// hitting the limit reveals nothing about which accounts exist.
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
		UserAgent: input.UserAgent,
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.users.SetLastLoginAt(ctx, u.ID, now); err != nil {

			return err
		}
		return s.sessions.Create(ctx, session)
	})
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{Token: token, ExpiresAt: session.ExpiresAt, User: u}, nil
}

// Authenticate resolves a bearer token to its owner's user id. A missing
// session and an expired one produce the identical error.
func (s *Service) Authenticate(ctx context.Context, token string) (uuid.UUID, error) {
	hash := hashToken(token)

	session, err := s.sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			return uuid.UUID{}, ErrUnauthenticated
		}
		return uuid.UUID{}, err
	}

	now := s.now()
	if !now.Before(session.ExpiresAt) {
		_ = s.sessions.Delete(ctx, hash) // best-effort; must never authenticate regardless
		return uuid.UUID{}, ErrUnauthenticated
	}

	// Best-effort: must never be the reason a valid request fails.
	_ = s.sessions.TouchLastUsed(ctx, hash, now, now.Add(-lastUsedResolution))

	return session.UserID, nil
}

// ListSessions marks the session making the request via currentToken.
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID, currentToken string) ([]SessionView, error) {
	sessions, err := s.sessions.ListByUser(ctx, userID, s.now())
	if err != nil {
		return nil, err
	}

	current := hashToken(currentToken)
	views := make([]SessionView, 0, len(sessions))
	for _, session := range sessions {
		views = append(views, SessionView{
			Current:    bytes.Equal(session.TokenHash, current),
			Device:     deviceLabel(session.UserAgent),
			CreatedAt:  session.CreatedAt,
			LastUsedAt: session.LastUsedAt,
			ExpiresAt:  session.ExpiresAt,
		})
	}
	return views, nil
}

// SignOutOtherSessions keeps the caller signed in.
func (s *Service) SignOutOtherSessions(ctx context.Context, userID uuid.UUID, currentToken string) (int, error) {
	return s.sessions.DeleteOthers(ctx, userID, hashToken(currentToken))
}

type ChangeEmailInput struct {
	Email           string
	CurrentPassword string
}

// errWrongCurrentPassword is a field error, not a 401: the caller IS signed
// in, and web clients treat any 401 as "session expired, log out".
var errWrongCurrentPassword = apperr.Validation(map[string]string{"current_password": "That password isn’t right."})

// ChangeEmail re-checks the current password so a stolen session alone can't
// take over the account's login identity; rate-limited against guessing.
func (s *Service) ChangeEmail(ctx context.Context, userID uuid.UUID, input ChangeEmailInput) (user.User, error) {
	if ok, retryAfter := s.attempts.Allow("reauth:" + userID.String()); !ok {
		return user.User{}, apperr.RateLimited(retryAfter)
	}

	fields := map[string]string{}
	email, err := user.ValidateEmail(input.Email)
	if err != nil {
		appErr, ok := apperr.As(err)
		if !ok {
			return user.User{}, err
		}
		for k, v := range appErr.Fields {
			fields[k] = v
		}
	}
	if input.CurrentPassword == "" {
		fields["current_password"] = "Enter your current password."
	}
	if len(fields) > 0 {
		return user.User{}, apperr.Validation(fields)
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return user.User{}, err
	}
	if err := s.hasher.Compare(u.PasswordHash, input.CurrentPassword); err != nil {
		return user.User{}, errWrongCurrentPassword
	}
	if email == u.Email {
		return u, nil
	}
	return s.users.UpdateEmail(ctx, userID, email, s.now())
}

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

// ChangePassword ends every other session (FR-ACCT-004.AC1) so a leaked
// password can't keep another holder signed in; the caller stays in.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentToken string, input ChangePasswordInput) error {
	if ok, retryAfter := s.attempts.Allow("reauth:" + userID.String()); !ok {
		return apperr.RateLimited(retryAfter)
	}

	fields := map[string]string{}
	if problem := passwordProblem(input.NewPassword); problem != "" {
		fields["new_password"] = problem
	}
	if input.CurrentPassword == "" {
		fields["current_password"] = "Enter your current password."
	}
	if len(fields) > 0 {
		return apperr.Validation(fields)
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.hasher.Compare(u.PasswordHash, input.CurrentPassword); err != nil {
		return errWrongCurrentPassword
	}

	hash, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return apperr.Internal(fmt.Errorf("hashing password: %w", err))
	}
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.users.UpdatePasswordHash(ctx, userID, hash, s.now()); err != nil {
			return err
		}
		_, err := s.sessions.DeleteOthers(ctx, userID, hashToken(currentToken))
		return err
	})
}

// DeleteAccount re-checks the password (FR-ACCT-005.AC2); see
// user.PostgresRepository.DeleteMe for the cascade.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, currentPassword string) error {
	if ok, retryAfter := s.attempts.Allow("reauth:" + userID.String()); !ok {
		return apperr.RateLimited(retryAfter)
	}
	if currentPassword == "" {
		return apperr.Validation(map[string]string{"current_password": "Enter your current password."})
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.hasher.Compare(u.PasswordHash, currentPassword); err != nil {
		return errWrongCurrentPassword
	}
	return s.users.DeleteMe(ctx, userID)
}

var errResetUnavailable = apperr.Internal(errors.New("password reset is not configured"))

// ForgotPassword answers identically whether or not the account exists
// (FR-AUTH-008.AC1), so it can't be used to enumerate registered emails.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	if s.resets == nil || s.mailer == nil {
		return errResetUnavailable
	}
	email = user.NormalizeEmail(email)
	if email == "" {
		return apperr.Validation(map[string]string{"email": "Enter your email."})
	}
	if ok, retryAfter := s.attempts.Allow("forgot:" + email); !ok {
		return apperr.RateLimited(retryAfter)
	}

	u, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, user.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	token, err := s.newToken()
	if err != nil {
		return apperr.Internal(fmt.Errorf("generating token: %w", err))
	}
	now := s.now()
	if err := s.resets.Create(ctx, hashToken(token), u.ID, now.Add(resetTTL), now); err != nil {
		return err
	}

	link := s.appURL + "/reset?token=" + url.QueryEscape(token)
	body := `<p>Someone asked to reset the password for your Amorae account.</p>` +
		`<p><a href="` + html.EscapeString(link) + `">Choose a new password</a></p>` +
		`<p>The link works once and expires in one hour. If you didn't ask for this, you can ignore this email.</p>`
	if err := s.mailer.Send(ctx, u.Email, "Reset your Amorae password", body); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// ResetPassword signs out every device (FR-AUTH-008.AC2); the link is
// consumed in the same transaction.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if s.resets == nil {
		return errResetUnavailable
	}
	fields := map[string]string{}
	if token == "" {
		fields["token"] = "This reset link is incomplete. Ask for a new one."
	}
	if problem := passwordProblem(newPassword); problem != "" {
		fields["new_password"] = problem
	}
	if len(fields) > 0 {
		return apperr.Validation(fields)
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return apperr.Internal(fmt.Errorf("hashing password: %w", err))
	}
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		now := s.now()
		userID, err := s.resets.Consume(ctx, hashToken(token), now)
		if errors.Is(err, errResetInvalid) {
			return apperr.Validation(map[string]string{
				"token": "This reset link has expired or was already used. Ask for a new one.",
			})
		}
		if err != nil {
			return err
		}
		if err := s.users.UpdatePasswordHash(ctx, userID, hash, now); err != nil {
			return err
		}
		return s.sessions.DeleteAllForUser(ctx, userID)
	})
}

// Logout is idempotent: deleting a nonexistent row isn't a SQL error.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}
