package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

// Repository lists only what Service calls; Create lives on the concrete
// type instead, for auth's use (see PostgresRepository's ISP note).
type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	Update(ctx context.Context, u User) (User, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.GetByID(ctx, id)
}

// BirthdayPatch mirrors what the request body can say about "birthday":
// the key absent (Present false, leave it alone), present as `null` (Present
// true, Clear true, remove it), or present with a value (Present true, Value
// set). A plain *Birthday can't tell "absent" from "explicit null" apart —
// both would leave a nil pointer — so the handler has to carry Present itself.
type BirthdayPatch struct {
	Present bool
	Clear   bool
	Value   Birthday
}

// Email is intentionally absent: changing it needs the current password
// (auth.Service.ChangeEmail).
type UpdateProfileInput struct {
	DisplayName string `json:"display_name"`
	// Timezone is optional: empty leaves it unchanged.
	Timezone string
	Birthday BirthdayPatch
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (User, error) {
	fields := map[string]string{}

	displayName, err := ValidateDisplayName(input.DisplayName)
	if err := collectFields(fields, err); err != nil {
		return User{}, err
	}

	var timezone string
	if input.Timezone != "" {
		timezone, err = ValidateTimezone(input.Timezone)
		if err := collectFields(fields, err); err != nil {
			return User{}, err
		}
	}

	var birthday Birthday
	if input.Birthday.Present && !input.Birthday.Clear {
		birthday, err = ValidateBirthday(input.Birthday.Value.Month, input.Birthday.Value.Day, input.Birthday.Value.Year, s.now())
		if err := collectFields(fields, err); err != nil {
			return User{}, err
		}
	}

	if len(fields) > 0 {
		return User{}, apperr.Validation(fields)
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	u.DisplayName = displayName
	if timezone != "" {
		u.Timezone = timezone
	}
	switch {
	case input.Birthday.Present && input.Birthday.Clear:
		u.BirthMonth, u.BirthDay, u.BirthYear = nil, nil, nil
	case input.Birthday.Present:
		month, day := birthday.Month, birthday.Day
		u.BirthMonth, u.BirthDay, u.BirthYear = &month, &day, birthday.Year
	}
	u.UpdatedAt = s.now()

	return s.repo.Update(ctx, u)
}

// collectFields merges a validation error's fields so multiple invalid fields are reported together.
func collectFields(fields map[string]string, err error) error {
	if err == nil {
		return nil
	}
	appErr, ok := apperr.As(err)
	if !ok || appErr.Kind != apperr.KindInvalid {
		return err
	}
	for k, v := range appErr.Fields {
		fields[k] = v
	}
	return nil
}
