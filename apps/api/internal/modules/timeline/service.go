package timeline

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is what this service needs of storage.
type Repository interface {
	// Timeline is the couple's items across the given Types, newest first,
	// strictly before `before` when it isn't nil. `limit` is exactly how
	// many rows to return — the service, not the repository, asks for one
	// extra to know whether another page follows.
	Timeline(ctx context.Context, coupleID uuid.UUID, types []Type, before *time.Time, limit int) ([]Item, error)
}

// Couples is the smaller question the Together modules ask: which couple,
// and nothing else — timeline computes a couple's own timezone in SQL
// (joining couples per source, same as journal and prayers already do)
// rather than asking for it here.
type Couples interface {
	CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// Photos resolves a memory's stored photo id to a URL to show it at. nil
// when Cloudinary isn't configured (mirrors memories.Photos /
// memories.Service.PhotosAvailable) — PhotoURL simply omits the photo then,
// rather than erroring a whole page over one memory's picture.
type Photos interface {
	URL(publicID string, version int64) (string, error)
}

type Service struct {
	repo    Repository
	couples Couples
	photos  Photos
	now     func() time.Time
}

func NewService(repo Repository, couples Couples, photos Photos, now func() time.Time) *Service {
	return &Service{repo: repo, couples: couples, photos: photos, now: now}
}

// Page is one page of the timeline: the items, and the cursor for the next
// page — nil once there is nothing further back.
type Page struct {
	Items []Item
	Next  *time.Time
}

// List is the couple's timeline, newest first, one page at a time.
func (s *Service) List(ctx context.Context, userID uuid.UUID, before *time.Time, filter Filter, limit int) (Page, error) {
	coupleID, err := s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return Page{}, err
	}

	// One extra row: its presence, after trimming back to `limit`, is what
	// tells the caller whether another page follows, without a second
	// round trip or a separate count query.
	items, err := s.repo.Timeline(ctx, coupleID, TypesFor(filter), before, limit+1)
	if err != nil {
		return Page{}, err
	}

	var next *time.Time
	if len(items) > limit {
		items = items[:limit]
		at := items[len(items)-1].At
		next = &at
	}
	return Page{Items: items, Next: next}, nil
}

// PhotoURL resolves a memory item's photo, or "" when there is none, or
// photos aren't configured on this server. Needs no couple/user check:
// reaching an Item at all already went through the couple-scoped query in
// Timeline (mirrors memories.Service.PhotoURL).
func (s *Service) PhotoURL(it Item) string {
	if s.photos == nil || it.PhotoID == "" {
		return ""
	}
	url, err := s.photos.URL(it.PhotoID, it.Version.Unix())
	if err != nil {
		return ""
	}
	return url
}
