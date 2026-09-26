package notifications

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// inboxRetention is how many of one person's own rows are kept. A glance
// back over what recently happened, not a permanent record — that's what
// notification_sends is for, and it has no title or body to show anyone.
const inboxRetention = 30

// InboxItem is one notification as the person who received it can look back
// on it — everything Deliver wrote down at the moment it decided to send.
type InboxItem struct {
	ID        uuid.UUID
	Kind      string
	Title     string
	Body      string
	Path      string
	CreatedAt time.Time
	Read      bool
}

// InboxRepository is what listing and reading someone's own inbox needs.
type InboxRepository interface {
	Inbox(ctx context.Context, userID uuid.UUID) ([]InboxItem, error)
	MarkInboxRead(ctx context.Context, userID uuid.UUID, at time.Time) error
}

// Inbox is this person's own history, newest first, capped at
// inboxRetention by RecordInbox rather than by this read.
func (s *Service) Inbox(ctx context.Context, userID uuid.UUID) ([]InboxItem, error) {
	return s.repo.Inbox(ctx, userID)
}

// MarkInboxRead marks every one of the caller's rows read at once, as of
// now. There is no per-row "mark this one read" — opening the list is what
// reading it means.
func (s *Service) MarkInboxRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkInboxRead(ctx, userID, s.now())
}
