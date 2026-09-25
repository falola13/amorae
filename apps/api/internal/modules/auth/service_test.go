package auth

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/ratelimit"
)

// This file lives in package auth (not auth_test): fakeSessionRepo needs to
// return the unexported errSessionNotFound sentinel Service checks for.

type fakeUserRepo struct {
	byEmail map[string]user.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]user.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u user.User) (user.User, error) {
	if _, exists := f.byEmail[u.Email]; exists {
		return user.User{}, user.ErrEmailTaken
	}
	f.byEmail[u.Email] = u
	return u, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (user.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (user.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

// UpdateEmail mirrors the Postgres repository: ErrEmailTaken on a clash.
func (f *fakeUserRepo) UpdateEmail(_ context.Context, id uuid.UUID, email string, at time.Time) (user.User, error) {
	if other, taken := f.byEmail[email]; taken && other.ID != id {
		return user.User{}, user.ErrEmailTaken
	}
	for old, u := range f.byEmail {
		if u.ID == id {
			delete(f.byEmail, old)
			u.Email, u.UpdatedAt = email, at
			f.byEmail[email] = u
			return u, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (f *fakeUserRepo) UpdatePasswordHash(_ context.Context, id uuid.UUID, hash string, at time.Time) error {
	for email, u := range f.byEmail {
		if u.ID == id {
			u.PasswordHash, u.UpdatedAt = hash, at
			f.byEmail[email] = u
			return nil
		}
	}
	return user.ErrNotFound
}

func (f *fakeUserRepo) DeleteMe(_ context.Context, id uuid.UUID) error {
	for email, u := range f.byEmail {
		if u.ID == id {
			delete(f.byEmail, email)
			return nil
		}
	}
	return user.ErrNotFound
}

type fakeSessionRepo struct {
	byHash    map[string]Session
	createErr error // when set, Create fails with it (simulates a DB error)
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{byHash: map[string]Session{}}
}

func (f *fakeSessionRepo) Create(_ context.Context, s Session) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.byHash[string(s.TokenHash)] = s
	return nil
}

// allowAll is an AttemptLimiter that never limits.
type allowAll struct{}

func (allowAll) Allow(string) (bool, time.Duration) { return true, 0 }
func (f *fakeUserRepo) SetLastLoginAt(_ context.Context, id uuid.UUID, at time.Time) error {
	for email, u := range f.byEmail {
		if u.ID == id {
			u.LastLoginAt = &at
			f.byEmail[email] = u
			return nil
		}
	}
	return user.ErrNotFound
}

func (f *fakeSessionRepo) GetByTokenHash(_ context.Context, hash []byte) (Session, error) {
	s, ok := f.byHash[string(hash)]
	if !ok {
		return Session{}, errSessionNotFound
	}
	return s, nil
}

func (f *fakeSessionRepo) Delete(_ context.Context, hash []byte) error {
	delete(f.byHash, string(hash))
	return nil
}

func (f *fakeSessionRepo) ListByUser(_ context.Context, userID uuid.UUID, now time.Time) ([]Session, error) {
	var out []Session
	for _, s := range f.byHash {
		if s.UserID == userID && s.ExpiresAt.After(now) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (f *fakeSessionRepo) DeleteOthers(_ context.Context, userID uuid.UUID, keepHash []byte) (int, error) {
	ended := 0
	for key, s := range f.byHash {
		if s.UserID == userID && key != string(keepHash) {
			delete(f.byHash, key)
			ended++
		}
	}
	return ended, nil
}

func (f *fakeSessionRepo) DeleteAllForUser(_ context.Context, userID uuid.UUID) error {
	for key, s := range f.byHash {
		if s.UserID == userID {
			delete(f.byHash, key)
		}
	}
	return nil
}

func (f *fakeSessionRepo) TouchLastUsed(_ context.Context, hash []byte, at, staleBefore time.Time) error {
	s, ok := f.byHash[string(hash)]
	if !ok {
		return nil
	}
	if s.LastUsedAt == nil || s.LastUsedAt.Before(staleBefore) {
		s.LastUsedAt = &at
		f.byHash[string(hash)] = s
	}
	return nil
}

type fakeTxRunner struct{}

func (fakeTxRunner) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// spyHasher counts Compare calls so a test can assert Login did bcrypt work
// even when the email doesn't exist.
type spyHasher struct {
	inner        PasswordHasher
	compareCalls int
}

func (s *spyHasher) Hash(password string) (string, error) { return s.inner.Hash(password) }

func (s *spyHasher) Compare(hash, password string) error {
	s.compareCalls++
	return s.inner.Compare(hash, password)
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
}

func sequentialToken(tokens ...string) func() (string, error) {
	i := 0
	return func() (string, error) {
		tok := tokens[i]
		i++
		return tok, nil
	}
}

func newTestService(t *testing.T, newToken func() (string, error)) (*Service, *fakeUserRepo, *fakeSessionRepo) {
	t.Helper()
	users := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	hasher := NewBcryptHasher(bcrypt.MinCost)

	svc, err := NewService(users, sessions, hasher, fakeTxRunner{}, time.Hour, fixedNow, newToken, allowAll{}, Options{})
	if err != nil {
		t.Fatalf("NewService() returned an error: %v", err)
	}
	return svc, users, sessions
}

func TestService_Register_OK(t *testing.T) {
	svc, users, sessions := newTestService(t, sequentialToken("token-1"))

	result, err := svc.Register(context.Background(), RegisterInput{AgeConfirmed: true, AcceptedTerms: true,
		Email:       "a@b.com",
		Password:    "password123",
		DisplayName: "Ada",
	})
	if err != nil {
		t.Fatalf("Register() returned an error: %v", err)
	}

	if result.Token != "token-1" {
		t.Errorf("Token = %q, want token-1", result.Token)
	}

	stored, ok := users.byEmail["a@b.com"]
	if !ok {
		t.Fatal("Register() did not create the user")
	}
	if stored.PasswordHash == "" || stored.PasswordHash == "password123" {
		t.Error("Register() stored the plaintext password instead of a hash")
	}

	if _, ok := sessions.byHash[string(hashToken("token-1"))]; !ok {
		t.Fatal("Register() did not create a session for the new token")
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc, _, _ := newTestService(t, sequentialToken("token-1", "token-2"))
	ctx := context.Background()
	input := RegisterInput{AgeConfirmed: true, AcceptedTerms: true, Email: "a@b.com", Password: "password123", DisplayName: "Ada"}

	if _, err := svc.Register(ctx, input); err != nil {
		t.Fatalf("first Register() returned an error: %v", err)
	}

	_, err := svc.Register(ctx, input)
	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindConflict {
		t.Fatalf("second Register() error = %v, want email_taken", err)
	}
}

func TestService_Register_AllFieldErrorsTogether(t *testing.T) {
	svc, _, _ := newTestService(t, sequentialToken("token-1"))

	_, err := svc.Register(context.Background(), RegisterInput{AgeConfirmed: true, AcceptedTerms: true,
		Email:       "not-an-email",
		Password:    "short",
		DisplayName: "",
	})

	appErr, ok := apperr.As(err)
	if !ok || appErr.Code != "validation_failed" {
		t.Fatalf("Register() error = %v, want validation_failed", err)
	}
	for _, field := range []string{"email", "password", "display_name"} {
		if _, ok := appErr.Fields[field]; !ok {
			t.Errorf("Fields is missing %s", field)
		}
	}
}

func TestService_Register_PasswordLengthBoundaries(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{"9 characters rejected", "nine@b.com", strings.Repeat("a", 9), true},
		{"10 characters accepted", "ten@b.com", strings.Repeat("a", 10), false},
		{"72 bytes accepted", "seventytwo@b.com", strings.Repeat("a", 72), false},
		{"73 bytes rejected", "seventythree@b.com", strings.Repeat("a", 73), true},
		// Characters, not bytes, for the minimum: 5 emoji are 20 bytes but 5 characters.
		{"5 emoji rejected", "emoji@b.com", strings.Repeat("🙏", 5), true},
		{"10 accented accepted", "accent@b.com", strings.Repeat("é", 10), false},
		// Bytes, not characters, for the maximum: 37 × "é" is 37 characters but 74 bytes.
		{"74 bytes of accents rejected", "long@b.com", strings.Repeat("é", 37), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService(t, sequentialToken("token"))

			_, err := svc.Register(context.Background(), RegisterInput{AgeConfirmed: true, AcceptedTerms: true,
				Email:       tc.email,
				Password:    tc.password,
				DisplayName: "Name",
			})
			if tc.wantErr && err == nil {
				t.Error("Register() returned nil error, want a validation error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Register() returned an error for a valid password: %v", err)
			}
		})
	}
}

func TestService_Login_OK(t *testing.T) {
	svc, _, _ := newTestService(t, sequentialToken("register-token", "login-token"))
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{AgeConfirmed: true, AcceptedTerms: true, Email: "a@b.com", Password: "password123", DisplayName: "Ada"})
	if err != nil {
		t.Fatalf("Register() returned an error: %v", err)
	}

	result, err := svc.Login(ctx, LoginInput{Email: "a@b.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Login() returned an error: %v", err)
	}
	if result.Token != "login-token" {
		t.Errorf("Token = %q, want login-token", result.Token)
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	svc, _, _ := newTestService(t, sequentialToken("register-token", "login-token"))
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{AgeConfirmed: true, AcceptedTerms: true, Email: "a@b.com", Password: "password123", DisplayName: "Ada"})
	if err != nil {
		t.Fatalf("Register() returned an error: %v", err)
	}

	_, err = svc.Login(ctx, LoginInput{Email: "a@b.com", Password: "wrong-password"})

	appErr, ok := apperr.As(err)
	if !ok || appErr != ErrInvalidCredentials {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_Login_UnknownEmail_StillComparesAgainstDummy(t *testing.T) {
	users := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	spy := &spyHasher{inner: NewBcryptHasher(bcrypt.MinCost)}

	svc, err := NewService(users, sessions, spy, fakeTxRunner{}, time.Hour, fixedNow, NewToken, allowAll{}, Options{})
	if err != nil {
		t.Fatalf("NewService() returned an error: %v", err)
	}

	_, err = svc.Login(context.Background(), LoginInput{Email: "nobody@b.com", Password: "whatever123"})

	appErr, ok := apperr.As(err)
	if !ok || appErr != ErrInvalidCredentials {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if spy.compareCalls != 1 {
		t.Errorf("Compare was called %d times, want exactly 1 (timing equalisation)", spy.compareCalls)
	}
}

func TestService_Authenticate_OK(t *testing.T) {
	svc, _, sessions := newTestService(t, NewToken)
	userID := uuid.New()
	hash := hashToken("token-1")
	sessions.byHash[string(hash)] = Session{
		TokenHash: hash,
		UserID:    userID,
		CreatedAt: fixedNow(),
		ExpiresAt: fixedNow().Add(time.Hour),
	}

	gotID, err := svc.Authenticate(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Authenticate() returned an error: %v", err)
	}
	if gotID != userID {
		t.Errorf("Authenticate() = %v, want %v", gotID, userID)
	}
}

func TestService_Authenticate_ExpiredSession(t *testing.T) {
	svc, _, sessions := newTestService(t, NewToken)
	hash := hashToken("token-1")
	sessions.byHash[string(hash)] = Session{
		TokenHash: hash,
		UserID:    uuid.New(),
		CreatedAt: fixedNow().Add(-2 * time.Hour),
		ExpiresAt: fixedNow().Add(-time.Hour),
	}

	_, err := svc.Authenticate(context.Background(), "token-1")

	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindUnauthenticated {
		t.Fatalf("Authenticate() error = %v, want Unauthenticated", err)
	}
	if _, stillExists := sessions.byHash[string(hash)]; stillExists {
		t.Error("Authenticate() did not delete the expired session")
	}
}

func TestService_Authenticate_UnknownToken(t *testing.T) {
	svc, _, _ := newTestService(t, NewToken)

	_, err := svc.Authenticate(context.Background(), "never-issued")

	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindUnauthenticated {
		t.Fatalf("Authenticate() error = %v, want Unauthenticated", err)
	}
}

func TestService_Logout_Idempotent(t *testing.T) {
	svc, _, sessions := newTestService(t, NewToken)
	hash := hashToken("token-1")
	sessions.byHash[string(hash)] = Session{TokenHash: hash}

	if err := svc.Logout(context.Background(), "token-1"); err != nil {
		t.Fatalf("first Logout() returned an error: %v", err)
	}
	if err := svc.Logout(context.Background(), "token-1"); err != nil {
		t.Fatalf("second Logout() on an already-logged-out token returned an error: %v", err)
	}
}

func TestService_Login_SessionWriteFails_ReturnsErrorNotToken(t *testing.T) {
	svc, _, sessions := newTestService(t, sequentialToken("tok-reg", "tok-login"))
	if _, err := svc.Register(context.Background(), RegisterInput{AgeConfirmed: true, AcceptedTerms: true, Email: "a@b.com", Password: "correct horse", DisplayName: "A"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	sessions.createErr = errors.New("connection reset")
	res, err := svc.Login(context.Background(), LoginInput{Email: "a@b.com", Password: "correct horse"})

	if err == nil {
		t.Fatalf("Login succeeded with token %q although the session was never stored", res.Token)
	}
	if res.Token != "" {
		t.Errorf("Login returned token %q alongside an error", res.Token)
	}
}

func TestService_Login_RateLimitedPerAccountBeforeBcrypt(t *testing.T) {
	users := newFakeUserRepo()
	spy := &spyHasher{inner: NewBcryptHasher(bcrypt.MinCost)}
	limiter := ratelimit.New(2, 15*time.Minute, fixedNow)
	svc, err := NewService(users, newFakeSessionRepo(), spy, fakeTxRunner{}, time.Hour, fixedNow, NewToken, limiter, Options{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	for range 2 {
		_, _ = svc.Login(context.Background(), LoginInput{Email: "victim@example.com", Password: "guess"})
	}
	comparesBefore := spy.compareCalls

	// Different casing is the same account: the key uses the normalised email.
	_, err = svc.Login(context.Background(), LoginInput{Email: " VICTIM@example.com ", Password: "guess"})

	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindRateLimited {
		t.Fatalf("3rd attempt err = %v, want rate_limited", err)
	}
	if appErr.RetryAfter <= 0 {
		t.Errorf("RetryAfter = %s, want > 0", appErr.RetryAfter)
	}
	if spy.compareCalls != comparesBefore {
		t.Error("a rate-limited attempt still ran bcrypt; the limit must be checked first")
	}

	_, err = svc.Login(context.Background(), LoginInput{Email: "other@example.com", Password: "guess"})
	if appErr, _ := apperr.As(err); appErr != nil && appErr.Kind == apperr.KindRateLimited {
		t.Fatal("one account's limit blocked a different account")
	}
}

func registerForEmailChange(t *testing.T) (*Service, *fakeUserRepo, user.User) {
	t.Helper()
	svc, users, _ := newTestService(t, sequentialToken("t1", "t2"))
	res, err := svc.Register(context.Background(), RegisterInput{AgeConfirmed: true, AcceptedTerms: true, Email: "old@example.com", Password: "correct horse", DisplayName: "Ada"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return svc, users, res.User
}

func TestService_ChangeEmail_OK(t *testing.T) {
	svc, users, u := registerForEmailChange(t)

	got, err := svc.ChangeEmail(context.Background(), u.ID, ChangeEmailInput{Email: "  New@Example.com ", CurrentPassword: "correct horse"})
	if err != nil {
		t.Fatalf("ChangeEmail: %v", err)
	}
	if got.Email != "new@example.com" {
		t.Errorf("Email = %q, want normalised new@example.com", got.Email)
	}
	if _, stillOld := users.byEmail["old@example.com"]; stillOld {
		t.Error("old email still maps to the account")
	}
}

func TestService_ChangeEmail_WrongPasswordIsAFieldErrorNot401(t *testing.T) {
	svc, users, u := registerForEmailChange(t)

	_, err := svc.ChangeEmail(context.Background(), u.ID, ChangeEmailInput{Email: "new@example.com", CurrentPassword: "not the password"})

	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindInvalid || appErr.Fields["current_password"] == "" {
		t.Fatalf("err = %v, want validation_failed with a current_password field (a 401 would log the web user out)", err)
	}
	if _, moved := users.byEmail["new@example.com"]; moved {
		t.Fatal("email changed despite the wrong password")
	}
}

func TestService_ChangeEmail_TakenAndInvalidInput(t *testing.T) {
	svc, users, u := registerForEmailChange(t)
	users.byEmail["taken@example.com"] = user.User{ID: uuid.New(), Email: "taken@example.com"}

	_, err := svc.ChangeEmail(context.Background(), u.ID, ChangeEmailInput{Email: "taken@example.com", CurrentPassword: "correct horse"})
	if appErr, ok := apperr.As(err); !ok || appErr.Kind != apperr.KindConflict {
		t.Fatalf("taken email: err = %v, want email_taken", err)
	}

	_, err = svc.ChangeEmail(context.Background(), u.ID, ChangeEmailInput{Email: "not-an-email", CurrentPassword: ""})
	appErr, ok := apperr.As(err)
	if !ok || appErr.Fields["email"] == "" || appErr.Fields["current_password"] == "" {
		t.Fatalf("bad input: err = %v, want both email and current_password field errors", err)
	}
}

func TestService_ChangeEmail_RateLimitedPerAccount(t *testing.T) {
	users := newFakeUserRepo()
	limiter := ratelimit.New(1, 15*time.Minute, fixedNow)
	svc, err := NewService(users, newFakeSessionRepo(), NewBcryptHasher(bcrypt.MinCost), fakeTxRunner{}, time.Hour, fixedNow, NewToken, limiter, Options{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	id := uuid.New()

	_, _ = svc.ChangeEmail(context.Background(), id, ChangeEmailInput{Email: "a@b.com", CurrentPassword: "guess-1"})
	_, err = svc.ChangeEmail(context.Background(), id, ChangeEmailInput{Email: "a@b.com", CurrentPassword: "guess-2"})

	if appErr, ok := apperr.As(err); !ok || appErr.Kind != apperr.KindRateLimited {
		t.Fatalf("second attempt err = %v, want rate_limited", err)
	}
}

const iPhoneUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1"
const windowsUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

func TestService_ListSessions_MarksTheCurrentOneAndLabelsTheDevice(t *testing.T) {
	svc, users, _ := newTestService(t, sequentialToken("phone-token", "laptop-token"))
	ctx := context.Background()

	phone, err := svc.Register(ctx, RegisterInput{AgeConfirmed: true, AcceptedTerms: true,
		Email: "a@b.com", Password: "password123", DisplayName: "Ada", UserAgent: iPhoneUA,
	})
	if err != nil {
		t.Fatalf("Register() returned an error: %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Email: "a@b.com", Password: "password123", UserAgent: windowsUA}); err != nil {
		t.Fatalf("Login() returned an error: %v", err)
	}

	views, err := svc.ListSessions(ctx, users.byEmail["a@b.com"].ID, phone.Token)
	if err != nil {
		t.Fatalf("ListSessions() returned an error: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("got %d sessions, want 2", len(views))
	}

	current, devices := 0, map[string]bool{}
	for _, v := range views {
		devices[v.Device] = true
		if !v.Current {
			continue
		}
		current++
		if v.Device != "Safari on iPhone" {
			t.Errorf("current session Device = %q, want the device that made the request", v.Device)
		}
	}
	if current != 1 {
		t.Errorf("sessions marked current = %d, want exactly 1", current)
	}
	if !devices["Safari on iPhone"] || !devices["Chrome on Windows"] {
		t.Errorf("Device labels = %v, want both sessions labelled", devices)
	}
}

func TestService_SignOutOtherSessions_KeepsTheCallerSignedIn(t *testing.T) {
	svc, users, _ := newTestService(t, sequentialToken("keep-token", "other-token"))
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{AgeConfirmed: true, AcceptedTerms: true, Email: "a@b.com", Password: "password123", DisplayName: "Ada"}); err != nil {
		t.Fatalf("Register() returned an error: %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Email: "a@b.com", Password: "password123"}); err != nil {
		t.Fatalf("Login() returned an error: %v", err)
	}

	ended, err := svc.SignOutOtherSessions(ctx, users.byEmail["a@b.com"].ID, "keep-token")
	if err != nil {
		t.Fatalf("SignOutOtherSessions() returned an error: %v", err)
	}
	if ended != 1 {
		t.Errorf("ended = %d, want 1", ended)
	}

	if _, err := svc.Authenticate(ctx, "keep-token"); err != nil {
		t.Errorf("the session that asked must still authenticate: %v", err)
	}
	if _, err := svc.Authenticate(ctx, "other-token"); err == nil {
		t.Error("the other session should no longer authenticate")
	}
}

func TestService_Authenticate_WritesLastUsedAtMostHourly(t *testing.T) {
	users := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	clock := fixedNow()
	svc, err := NewService(users, sessions, NewBcryptHasher(bcrypt.MinCost), fakeTxRunner{}, 24*time.Hour,
		func() time.Time { return clock }, sequentialToken("token-1"), allowAll{}, Options{})
	if err != nil {
		t.Fatalf("NewService() returned an error: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Register(ctx, RegisterInput{AgeConfirmed: true, AcceptedTerms: true, Email: "a@b.com", Password: "password123", DisplayName: "Ada"}); err != nil {
		t.Fatalf("Register() returned an error: %v", err)
	}
	lastUsed := func() *time.Time {
		s, err := sessions.GetByTokenHash(ctx, hashToken("token-1"))
		if err != nil {
			t.Fatalf("GetByTokenHash() returned an error: %v", err)
		}
		return s.LastUsedAt
	}

	if _, err := svc.Authenticate(ctx, "token-1"); err != nil {
		t.Fatalf("Authenticate() returned an error: %v", err)
	}
	first := lastUsed()
	if first == nil {
		t.Fatal("the first use should record last_used_at")
	}

	clock = clock.Add(30 * time.Minute)
	if _, err := svc.Authenticate(ctx, "token-1"); err != nil {
		t.Fatalf("Authenticate() returned an error: %v", err)
	}
	if got := lastUsed(); !got.Equal(*first) {
		t.Errorf("last_used_at moved within the hour (%v → %v): an active session would write on every request", first, got)
	}

	clock = clock.Add(31 * time.Minute)
	if _, err := svc.Authenticate(ctx, "token-1"); err != nil {
		t.Fatalf("Authenticate() returned an error: %v", err)
	}
	if got := lastUsed(); !got.After(*first) {
		t.Errorf("last_used_at = %v, want it to move once an hour has passed", got)
	}
}
