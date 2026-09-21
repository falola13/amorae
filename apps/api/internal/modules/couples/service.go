package couples

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
)

type Repository interface {
	Create(ctx context.Context, inviteCode string, creatorName string, input COUPLES) (COUPLES, error)
	Join(ctx context.Context, userID uuid.UUID, code string, at time.Time) (COUPLES, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now}
}

type CoupleCreateInput struct {
	Name                  *string
	RelationshipStartDate *time.Time
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

	value, err := New(name, createdBy, input.RelationshipStartDate, s.now())
	if err != nil {
		return COUPLES{}, err
	}

	inviteCode, err := s.newInviteCode()
	if err != nil {
		return COUPLES{}, err
	}

	return s.repo.Create(ctx, inviteCode, creatorName, value)
}

func (s *Service) Join(ctx context.Context, userID uuid.UUID, code string) (COUPLES, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return COUPLES{}, apperr.Validation(map[string]string{
			"code": "Enter an invite code.",
		})
	}
	return s.repo.Join(ctx, userID, code, s.now())
}
