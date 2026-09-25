package user

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// fakeRepository is an in-memory stand-in for Repository so tests run without a database.
type fakeRepository struct {
	users map[uuid.UUID]User
}

func newFakeRepository(users ...User) *fakeRepository {
	m := make(map[uuid.UUID]User, len(users))
	for _, u := range users {
		m[u.ID] = u
	}
	return &fakeRepository{users: m}
}

func (f *fakeRepository) GetByID(_ context.Context, id uuid.UUID) (User, error) {
	u, ok := f.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeRepository) Update(_ context.Context, u User) (User, error) {
	if _, ok := f.users[u.ID]; !ok {
		return User{}, ErrNotFound
	}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeRepository) UpdateLoginTime(_ context.Context, u User) (User, error) {
	if _, ok := f.users[u.ID]; !ok {
		return User{}, ErrNotFound
	}
	f.users[u.ID] = u
	return u, nil
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
}

func TestService_Get_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository(), fixedNow)

	_, err := svc.Get(context.Background(), uuid.New())

	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindNotFound {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestService_UpdateProfile_BumpsUpdatedAt(t *testing.T) {
	original := User{
		ID:          uuid.New(),
		Email:       "a@b.com",
		DisplayName: "Old Name",
		CreatedAt:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	svc := NewService(newFakeRepository(original), fixedNow)

	updated, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{DisplayName: "  New Name  "})
	if err != nil {
		t.Fatalf("UpdateProfile() returned an error: %v", err)
	}

	if updated.DisplayName != "New Name" {
		t.Errorf("DisplayName = %q, want trimmed New Name", updated.DisplayName)
	}
	if !updated.UpdatedAt.Equal(fixedNow()) {
		t.Errorf("UpdatedAt = %v, want %v", updated.UpdatedAt, fixedNow())
	}
	if !updated.CreatedAt.Equal(original.CreatedAt) {
		t.Error("UpdateProfile() changed CreatedAt")
	}
}

func TestService_UpdateProfile_RejectsInvalidDisplayName(t *testing.T) {
	original := User{ID: uuid.New(), Email: "a@b.com", DisplayName: "Name"}
	svc := NewService(newFakeRepository(original), fixedNow)

	_, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{DisplayName: "   "})

	appErr, ok := apperr.As(err)
	if !ok || appErr.Code != "validation_failed" {
		t.Fatalf("UpdateProfile() error = %v, want validation_failed", err)
	}
}

func TestService_UpdateProfile_Timezone(t *testing.T) {
	original := User{ID: uuid.New(), DisplayName: "Ada", Timezone: DefaultTimezone}
	svc := NewService(newFakeRepository(original), fixedNow)

	got, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{DisplayName: "Ada", Timezone: "Africa/Lagos"})
	if err != nil {
		t.Fatalf("valid zone: %v", err)
	}
	if got.Timezone != "Africa/Lagos" {
		t.Errorf("Timezone = %q, want Africa/Lagos", got.Timezone)
	}

	got, err = svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{DisplayName: "Ada"})
	if err != nil || got.Timezone != "Africa/Lagos" {
		t.Errorf("empty timezone should leave it unchanged: got %q, err %v", got.Timezone, err)
	}

	for _, bad := range []string{"Mars/Olympus", "Local", "   "} {
		_, err = svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{DisplayName: "", Timezone: bad})
		f := fieldsOf(t, err)
		if f["timezone"] == "" || f["display_name"] == "" {
			t.Errorf("timezone %q: fields = %v, want both timezone and display_name errors at once", bad, f)
		}
	}
}

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	appErr, ok := apperr.As(err)
	if !ok {
		t.Fatalf("err = %v, want a validation error", err)
	}
	return appErr.Fields
}
