package couples

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

type Repository interface {
	Create(ctx context.Context, inviteCode string, inviteExpiresAt time.Time, input COUPLES) (COUPLES, error)
	Join(ctx context.Context, userID uuid.UUID, code string, at time.Time) error
	GetForUser(ctx context.Context, userID uuid.UUID) (Mine, error)
	UpdateCouples(ctx context.Context, coupleID uuid.UUID, start *time.Time, name *string) error
	UpdateRole(ctx context.Context, id uuid.UUID, coupleID uuid.UUID, role string) error
	UpdateOnboarding(ctx context.Context, id uuid.UUID, coupleID uuid.UUID, install *bool, notifications *bool) error
}

// How long an invite code stays usable. Read through the service clock so a
// test can move time past it.
const inviteTTL = 7 * 24 * time.Hour

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

type CoupleCreateInput struct {
	Name                  *string
	RelationshipStartDate *string
	// The creator's own zone, which seeds the couple's. Nothing lets a couple
	// change it yet, so this is the only chance to get it right.
	Timezone string
}

// The API takes calendar days as "2006-01-02"; encoding/json only decodes
// RFC 3339 into a time.Time, so the string is parsed here instead.
func parseDay(day *string) (*time.Time, error) {
	if day == nil {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*day))
	if err != nil {
		return nil, apperr.Validation(map[string]string{
			"relationship_start_date": "Use a date like 2025-10-25.",
		})
	}
	return &parsed, nil
}

func (s *Service) newInviteCode() (string, error) {
	letters := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	digits := []byte("0123456789")
	code := make([]byte, 6)

	for i := 0; i < 3; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		code[i] = letters[n.Int64()]
	}
	for i := 0; i < 3; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[3+i] = digits[n.Int64()]
	}
	return string(code), nil
}

func (s *Service) Create(ctx context.Context, createdBy uuid.UUID, creatorName string, input CoupleCreateInput) (COUPLES, error) {
	name := ""
	if input.Name != nil {
		name = strings.TrimSpace(*input.Name)
	}
	if name == "" {
		name = strings.TrimSpace(creatorName) + " & partner"
	}

	start, err := parseDay(input.RelationshipStartDate)
	if err != nil {
		return COUPLES{}, err
	}

	timezone := input.Timezone
	if strings.TrimSpace(timezone) == "" {
		timezone = user.DefaultTimezone
	}

	now := s.now()
	value, err := New(name, createdBy, timezone, start, now)
	if err != nil {
		return COUPLES{}, err
	}

	inviteCode, err := s.newInviteCode()
	if err != nil {
		return COUPLES{}, err
	}

	return s.repo.Create(ctx, inviteCode, now.Add(inviteTTL), value)
}

// Joining answers with the same view GetMine returns: the caller has just
// gained a partner, and that partner is the first thing the next screen shows.
func (s *Service) Join(ctx context.Context, userID uuid.UUID, code string) (Mine, error) {
	code = normalizeInviteCode(code)
	if code == "" {
		return Mine{}, apperr.Validation(map[string]string{
			"code": "Enter an invite code.",
		})
	}
	if err := s.repo.Join(ctx, userID, code, s.now()); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}

func normalizeInviteCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Service) GetMine(ctx context.Context, userID uuid.UUID) (Mine, error) {
	return s.repo.GetForUser(ctx, userID)
}

// Both updates return the whole couple, the same shape GetMine returns, so a
// client never has to re-fetch to see what it just changed.
func (s *Service) UpdateCouples(ctx context.Context, userID uuid.UUID, update UpdateDto) (Mine, error) {
	start, err := parseDay(update.RelationshipStartDate)
	if err != nil {
		return Mine{}, err
	}
	mine, err := s.GetMine(ctx, userID)
	if err != nil {
		return Mine{}, err
	}
	if err := s.repo.UpdateCouples(ctx, mine.Couple.ID, start, update.Name); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdateRole(ctx context.Context, userID uuid.UUID, role string) (Mine, error) {
	role, err := ValidateRole(role)
	if err != nil {
		return Mine{}, err
	}

	mine, err := s.GetMine(ctx, userID)
	if err != nil {
		return Mine{}, err
	}
	if err := s.repo.UpdateRole(ctx, userID, mine.Couple.ID, role); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}

// UpdateOnboarding records the caller's own progress. A nil field is left
// alone, so the client can send one step at a time. The couple step is not
// stored: GetMine answering at all means the caller is in a couple.
func (s *Service) UpdateOnboarding(ctx context.Context, userID uuid.UUID, patch OnboardingDto) (Mine, error) {
	mine, err := s.GetMine(ctx, userID)
	if err != nil {
		return Mine{}, err
	}
	if patch.Install == nil && patch.Notifications == nil {
		return mine, nil
	}
	if err := s.repo.UpdateOnboarding(ctx, userID, mine.Couple.ID, patch.Install, patch.Notifications); err != nil {
		return Mine{}, err
	}
	return s.GetMine(ctx, userID)
}
