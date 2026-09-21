package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/ratelimit"
)

// This file lives in package auth (not auth_test) because fakeSessionRepo
// needs to return errSessionNotFound — the unexported sentinel Service
// checks for — to correctly simulate "no such session" the way the real
// postgres repository does.

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

// allowAll is an AttemptLimiter that never limits, for tests about
// everything except rate limiting.
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

type fakeTxRunner struct{}

func (fakeTxRunner) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// spyHasher wraps a real hasher and counts Compare calls, so a test can
// assert that Login did the bcrypt work even when the email doesn't exist.
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

	svc, err := NewService(users, sessions, hasher, fakeTxRunner{}, time.Hour, fixedNow, newToken, allowAll{})
	if err != nil {
		t.Fatalf("NewService() returned an error: %v", err)
	}
	return svc, users, sessions
}

func TestService_Register_OK(t *testing.T) {
	svc, users, sessions := newTestService(t, sequentialToken("token-1"))

	result, err := svc.Register(context.Background(), RegisterInput{
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
	input := RegisterInput{Email: "a@b.com", Password: "password123", DisplayName: "Ada"}

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

	_, err := svc.Register(context.Background(), RegisterInput{
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
		{"7 bytes rejected", "seven@b.com", strings.Repeat("a", 7), true},
		{"8 bytes accepted", "eight@b.com", strings.Repeat("a", 8), false},
		{"72 bytes accepted", "seventytwo@b.com", strings.Repeat("a", 72), false},
		{"73 bytes rejected", "seventythree@b.com", strings.Repeat("a", 73), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newTestService(t, sequentialToken("token"))

			_, err := svc.Register(context.Background(), RegisterInput{
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

	_, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password123", DisplayName: "Ada"})
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

	_, err := svc.Register(ctx, RegisterInput{Email: "a@b.com", Password: "password123", DisplayName: "Ada"})
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

	svc, err := NewService(users, sessions, spy, fakeTxRunner{}, time.Hour, fixedNow, NewToken, allowAll{})
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
	if _, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "correct horse", DisplayName: "A"}); err != nil {
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
	svc, err := NewService(users, newFakeSessionRepo(), spy, fakeTxRunner{}, time.Hour, fixedNow, NewToken, limiter)
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

	// Another account is unaffected.
	_, err = svc.Login(context.Background(), LoginInput{Email: "other@example.com", Password: "guess"})
	if appErr, _ := apperr.As(err); appErr != nil && appErr.Kind == apperr.KindRateLimited {
		t.Fatal("one account's limit blocked a different account")
	}
}
