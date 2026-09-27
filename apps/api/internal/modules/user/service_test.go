package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/photos"
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

func (f *fakeRepository) SetPhoto(_ context.Context, id uuid.UUID, photoID string, at time.Time) error {
	u, ok := f.users[id]
	if !ok {
		return ErrNotFound
	}
	u.PhotoID = photoID
	u.UpdatedAt = at
	f.users[id] = u
	return nil
}

// fakePhotos is an in-memory stand-in for Photos so tests run without Cloudinary.
type fakePhotos struct {
	tickets   []string // public ids a ticket was issued for
	destroyed []string // public ids destroyed
}

func (f *fakePhotos) Ticket(publicID string, _ time.Time) (photos.Ticket, error) {
	f.tickets = append(f.tickets, publicID)
	return photos.Ticket{UploadURL: "https://upload.example/" + publicID, Fields: map[string]string{"public_id": publicID}}, nil
}

func (f *fakePhotos) URL(publicID string, version int64) (string, error) {
	return fmt.Sprintf("https://cdn.example/%s?v=%d", publicID, version), nil
}

func (f *fakePhotos) Destroy(_ context.Context, publicID string) error {
	f.destroyed = append(f.destroyed, publicID)
	return nil
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
}

func TestService_Get_NotFound(t *testing.T) {
	svc := NewService(newFakeRepository(), nil, fixedNow)

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
	svc := NewService(newFakeRepository(original), nil, fixedNow)

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
	svc := NewService(newFakeRepository(original), nil, fixedNow)

	_, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{DisplayName: "   "})

	appErr, ok := apperr.As(err)
	if !ok || appErr.Code != "validation_failed" {
		t.Fatalf("UpdateProfile() error = %v, want validation_failed", err)
	}
}

func TestService_UpdateProfile_Timezone(t *testing.T) {
	original := User{ID: uuid.New(), DisplayName: "Ada", Timezone: DefaultTimezone}
	svc := NewService(newFakeRepository(original), nil, fixedNow)

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

func TestService_UpdateProfile_Birthday(t *testing.T) {
	original := User{ID: uuid.New(), DisplayName: "Ada"}
	svc := NewService(newFakeRepository(original), nil, fixedNow)
	year := 1990

	t.Run("absent leaves it alone", func(t *testing.T) {
		got, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{DisplayName: "Ada"})
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if got.BirthMonth != nil || got.BirthDay != nil {
			t.Errorf("BirthMonth/BirthDay = %v/%v, want untouched (nil)", got.BirthMonth, got.BirthDay)
		}
	})

	t.Run("present sets it", func(t *testing.T) {
		got, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{
			DisplayName: "Ada",
			Birthday:    BirthdayPatch{Present: true, Value: Birthday{Month: 9, Day: 30, Year: &year}},
		})
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if got.BirthMonth == nil || *got.BirthMonth != 9 || got.BirthDay == nil || *got.BirthDay != 30 {
			t.Fatalf("BirthMonth/BirthDay = %v/%v, want 9/30", got.BirthMonth, got.BirthDay)
		}
		if got.BirthYear == nil || *got.BirthYear != 1990 {
			t.Errorf("BirthYear = %v, want 1990", got.BirthYear)
		}
	})

	t.Run("explicit clear removes it", func(t *testing.T) {
		got, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{
			DisplayName: "Ada",
			Birthday:    BirthdayPatch{Present: true, Clear: true},
		})
		if err != nil {
			t.Fatalf("UpdateProfile: %v", err)
		}
		if got.BirthMonth != nil || got.BirthDay != nil || got.BirthYear != nil {
			t.Errorf("birthday = %v/%v/%v, want all cleared", got.BirthMonth, got.BirthDay, got.BirthYear)
		}
	})

	t.Run("an impossible date is refused under the birthday field", func(t *testing.T) {
		_, err := svc.UpdateProfile(context.Background(), original.ID, UpdateProfileInput{
			DisplayName: "Ada",
			Birthday:    BirthdayPatch{Present: true, Value: Birthday{Month: 2, Day: 30}},
		})
		f := fieldsOf(t, err)
		if f["birthday"] == "" {
			t.Errorf("fields = %v, want a birthday error", f)
		}
	})
}

func TestService_PhotoTicket_Unavailable(t *testing.T) {
	original := User{ID: uuid.New(), DisplayName: "Ada"}
	svc := NewService(newFakeRepository(original), nil, fixedNow) // no Photos configured

	_, err := svc.PhotoTicket(context.Background(), original.ID)

	appErr, ok := apperr.As(err)
	if !ok || appErr.Code != "photos_unavailable" {
		t.Fatalf("PhotoTicket() error = %v, want photos_unavailable", err)
	}
}

func TestService_PhotoTicket_OK(t *testing.T) {
	original := User{ID: uuid.New(), DisplayName: "Ada"}
	pics := &fakePhotos{}
	svc := NewService(newFakeRepository(original), pics, fixedNow)

	ticket, err := svc.PhotoTicket(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("PhotoTicket() error = %v", err)
	}
	if len(pics.tickets) != 1 || pics.tickets[0] != PhotoPublicID(original.ID) {
		t.Errorf("tickets = %v, want one for %s", pics.tickets, PhotoPublicID(original.ID))
	}
	if ticket.UploadURL == "" {
		t.Error("ticket has no upload url")
	}
}

func TestService_AttachPhoto(t *testing.T) {
	original := User{ID: uuid.New(), DisplayName: "Ada"}

	t.Run("unavailable without Photos configured", func(t *testing.T) {
		svc := NewService(newFakeRepository(original), nil, fixedNow)
		_, err := svc.AttachPhoto(context.Background(), original.ID)
		appErr, ok := apperr.As(err)
		if !ok || appErr.Code != "photos_unavailable" {
			t.Fatalf("AttachPhoto() error = %v, want photos_unavailable", err)
		}
	})

	t.Run("records the server-derived public id", func(t *testing.T) {
		svc := NewService(newFakeRepository(original), &fakePhotos{}, fixedNow)
		got, err := svc.AttachPhoto(context.Background(), original.ID)
		if err != nil {
			t.Fatalf("AttachPhoto() error = %v", err)
		}
		if !got.HasPhoto() || got.PhotoID != PhotoPublicID(original.ID) {
			t.Errorf("PhotoID = %q, want %q", got.PhotoID, PhotoPublicID(original.ID))
		}
	})
}

func TestService_RemovePhoto_DestroysThenClears(t *testing.T) {
	original := User{ID: uuid.New(), DisplayName: "Ada", PhotoID: PhotoPublicID(uuid.New())}
	pics := &fakePhotos{}
	svc := NewService(newFakeRepository(original), pics, fixedNow)

	got, err := svc.RemovePhoto(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("RemovePhoto() error = %v", err)
	}
	if got.HasPhoto() {
		t.Error("RemovePhoto() left a photo behind")
	}
	if len(pics.destroyed) != 1 || pics.destroyed[0] != original.PhotoID {
		t.Errorf("destroyed = %v, want [%s]", pics.destroyed, original.PhotoID)
	}
}

func TestService_PhotoURL(t *testing.T) {
	withPhoto := User{ID: uuid.New(), DisplayName: "Ada", PhotoID: "amorae/users/x/avatar", UpdatedAt: fixedNow()}
	withoutPhoto := User{ID: uuid.New(), DisplayName: "Ben"}

	t.Run("present when the user has a photo and Photos is configured", func(t *testing.T) {
		svc := NewService(newFakeRepository(withPhoto), &fakePhotos{}, fixedNow)
		if url := svc.PhotoURL(withPhoto); url == "" {
			t.Error("PhotoURL() = \"\", want a url")
		}
	})

	t.Run("absent without a photo", func(t *testing.T) {
		svc := NewService(newFakeRepository(withoutPhoto), &fakePhotos{}, fixedNow)
		if url := svc.PhotoURL(withoutPhoto); url != "" {
			t.Errorf("PhotoURL() = %q, want empty", url)
		}
	})

	t.Run("absent when Photos isn't configured, even with a photo id", func(t *testing.T) {
		svc := NewService(newFakeRepository(withPhoto), nil, fixedNow)
		if url := svc.PhotoURL(withPhoto); url != "" {
			t.Errorf("PhotoURL() = %q, want empty", url)
		}
	})
}

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	appErr, ok := apperr.As(err)
	if !ok {
		t.Fatalf("err = %v, want a validation error", err)
	}
	return appErr.Fields
}
