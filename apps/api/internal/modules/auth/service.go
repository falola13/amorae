// Package auth owns authentication use cases: registering, logging in and
// out, and resolving a bearer token back to a user id. It imports user (for
// the User entity and its validation) but user never imports auth — see
// internal/platform/authctx for why that direction doesn't create a cycle.
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

// UserRepository lists only Create and GetByEmail — what auth's use cases
// actually call — even though the concrete postgres repository behind it
// also implements GetByID and Update for user.Service. Each consumer
// declares its own narrow interface (ISP); there's still exactly one
// implementation.
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

// Events counts product events for metrics. Counts only, never content.
type Events interface {
	SignedUp()
}

// PolicyVersions are the Terms, Privacy and faith-content versions recorded
// with each consent at sign-up.
type PolicyVersions struct {
	Terms   string
	Privacy string
	Faith   string
}

// Options holds the collaborators only some use cases need. Consents and
// Events default to no-ops; Resets and Mailer are required for password
// reset and fail that use case, not the whole service, when missing.
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

// How long a password-reset link works.
const resetTTL = time.Hour

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

// The password rule, shared with the web app (apps/web/src/lib/api/schemas.ts):
// at least 10 characters, counted as Unicode characters (so an emoji is one,
// as a person would count it), and at most 72 bytes, because bcrypt ignores
// everything after 72 bytes and a password must never be silently truncated.
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
	// FR-AUTH-010 and 011: both must be true to register. FaithConsent is
	// separate and optional (FR-AUTH-012); the Terms never imply it.
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

// Register validates the input, hashes the password, and creates the user
// and their first session in one transaction — a user should never exist
// without a way to log in, and vice versa.
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

// recordConsents writes what the person agreed to at sign-up, inside the
// transaction that creates them, so an account never exists without its
// record of consent (FR-AUTH-011.AC2).
func (s *Service) recordConsents(ctx context.Context, userID uuid.UUID, faith bool) error {
	kinds := map[string]string{
		ConsentTerms:   s.policies.Terms,
		ConsentPrivacy: s.policies.Privacy,
		// The Terms set the minimum age, so the age check is versioned with them.
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
		UserAgent: input.UserAgent,
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

	now := s.now()
	if !now.Before(session.ExpiresAt) {
		_ = s.sessions.Delete(ctx, hash) // best-effort; an expired session must never authenticate regardless
		return uuid.UUID{}, ErrUnauthenticated
	}

	// Best-effort, and at most hourly (see TouchLastUsed): "last used" is for
	// recognising a session in the list, not an audit trail, and it must never
	// be the reason an otherwise valid request fails.
	_ = s.sessions.TouchLastUsed(ctx, hash, now, now.Add(-lastUsedResolution))

	return session.UserID, nil
}

// ListSessions is "where you're signed in", for the caller's own account. The
// session making the request is marked, so it's obvious which row not to worry
// about. currentToken is the bearer token of that request.
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

// SignOutOtherSessions ends every session except the one asking, and reports
// how many ended. The caller stays signed in: someone securing their account
// from a phone they still hold shouldn't be logged out by it.
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

// ChangeEmail moves the account to a new email, after re-checking the
// current password. The email is the account's identity (it's what login
// asks for), so a stolen session alone must not be enough to take it over.
// Attempts share the per-account login limit, so the password can't be
// guessed through this endpoint either.
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

// ChangePassword re-checks the current password, stores the new one, and
// ends every other session (FR-ACCT-004.AC1): if the password was changed
// because it leaked, whoever used it is signed out. The caller stays in.
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

// DeleteAccount removes the account after re-checking the password
// (FR-ACCT-005.AC2). Sessions, membership and invitations go with the user
// through the cascade; see user.PostgresRepository.DeleteMe for the couple.
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

// ForgotPassword emails a one-hour reset link if the account exists, and
// answers the same way either way (FR-AUTH-008.AC1), so the endpoint can't
// be used to find out which emails are registered.
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

// ResetPassword sets a new password from a reset link and signs out every
// device (FR-AUTH-008.AC2). The link is used up in the same transaction.
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

// Logout deletes the session for token. Deleting a row that doesn't exist
// isn't an error for SQL, which is what makes this idempotent for free —
// no special-casing "already logged out" is needed.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}
