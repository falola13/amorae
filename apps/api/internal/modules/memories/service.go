package memories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/photos"
)

type Repository interface {
	List(ctx context.Context, coupleID uuid.UUID) ([]Memory, error)
	ByID(ctx context.Context, coupleID, id uuid.UUID) (Memory, error)
	Create(ctx context.Context, m Memory, at time.Time) (uuid.UUID, error)
	SetPhoto(ctx context.Context, coupleID, id uuid.UUID, photoID string, at time.Time) error
	Delete(ctx context.Context, coupleID, id uuid.UUID) error
}

// Photos is what this module needs of picture storage. Nil when Cloudinary is
// not configured, and the handler answers accordingly rather than the service
// pretending.
type Photos interface {
	Ticket(publicID string, at time.Time) (photos.Ticket, error)
	URL(publicID string, version int64) (string, error)
	Destroy(ctx context.Context, publicID string) error
}

// Couples answers the one question this module asks of pairing.
type Couples interface {
	CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type Service struct {
	repo    Repository
	couples Couples
	photos  Photos
	now     func() time.Time
}

func NewService(repo Repository, couples Couples, pics Photos, now func() time.Time) *Service {
	return &Service{repo: repo, couples: couples, photos: pics, now: now}
}

// PhotosAvailable reports whether pictures can be attached at all.
func (s *Service) PhotosAvailable() bool { return s.photos != nil }

// PhotoTicket is permission to upload one file, to one name this server
// chooses, for the next hour. The memory must exist and be this couple's —
// checked here, so a ticket is never issued for somebody else's memory.
func (s *Service) PhotoTicket(ctx context.Context, userID, id uuid.UUID) (photos.Ticket, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return photos.Ticket{}, err
	}
	if _, err := s.repo.ByID(ctx, coupleID, id); err != nil {
		return photos.Ticket{}, err
	}
	if s.photos == nil {
		return photos.Ticket{}, ErrNoPhotos
	}
	return s.photos.Ticket(photos.PublicID(coupleID, id), s.now())
}

// AttachPhoto records that the upload happened. It takes no id from the
// request: the only name a ticket could have written to is the one derived
// here, so there is nothing for a client to choose.
func (s *Service) AttachPhoto(ctx context.Context, userID, id uuid.UUID) (Memory, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Memory{}, err
	}
	if s.photos == nil {
		return Memory{}, ErrNoPhotos
	}
	if err := s.repo.SetPhoto(ctx, coupleID, id, photos.PublicID(coupleID, id), s.now()); err != nil {
		return Memory{}, err
	}
	return s.repo.ByID(ctx, coupleID, id)
}

// RemovePhoto deletes the file before clearing the pointer, so a refusal at
// Cloudinary leaves the photo visible rather than claiming it is gone.
func (s *Service) RemovePhoto(ctx context.Context, userID, id uuid.UUID) (Memory, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Memory{}, err
	}
	m, err := s.repo.ByID(ctx, coupleID, id)
	if err != nil {
		return Memory{}, err
	}
	if err := s.destroyPhoto(ctx, m); err != nil {
		return Memory{}, err
	}
	if err := s.repo.SetPhoto(ctx, coupleID, id, "", s.now()); err != nil {
		return Memory{}, err
	}
	return s.repo.ByID(ctx, coupleID, id)
}

// Delete removes a moment and its picture. The photo goes first: the row
// holds the only reference to it.
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return err
	}
	m, err := s.repo.ByID(ctx, coupleID, id)
	if err != nil {
		return err
	}
	if err := s.destroyPhoto(ctx, m); err != nil {
		return err
	}
	return s.repo.Delete(ctx, coupleID, id)
}

func (s *Service) destroyPhoto(ctx context.Context, m Memory) error {
	if s.photos == nil || !m.HasPhoto() {
		return nil
	}
	return s.photos.Destroy(ctx, m.PhotoID)
}

// PhotoURL is a delivery address for one memory's picture, or "" if it has
// none.
//
// It needs no couple and no user: the memory only reached this point through
// a couple-scoped query, so being able to name it is already the permission.
// Generated per request rather than stored, so the address lives as long as
// the response and no longer.
func (s *Service) PhotoURL(m Memory) string {
	if s.photos == nil || !m.HasPhoto() {
		return ""
	}
	// updated_at moves when the photo does, which is what versions the URL.
	url, err := s.photos.URL(m.PhotoID, m.UpdatedAt.Unix())
	if err != nil {
		return ""
	}
	return url
}

// List is the couple's memories, the same for both of them (FR-MEM-002).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Memory, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, coupleID)
}

// Create keeps a moment. Either partner may, and it belongs to them both
// (DEC-16) — an archive with an author beside each entry is a feed.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (Memory, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Memory{}, err
	}
	clean, err := Validate(in)
	if err != nil {
		return Memory{}, err
	}

	id, err := s.repo.Create(ctx, Memory{
		CoupleID: coupleID,
		Title:    clean.Title,
		Date:     clean.Date,
		Location: clean.Location,
		Note:     clean.Note,
	}, s.now())
	if err != nil {
		return Memory{}, err
	}
	return s.repo.ByID(ctx, coupleID, id)
}
