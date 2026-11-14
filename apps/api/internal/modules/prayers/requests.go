package prayers

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/google/uuid"
)

const maxRequestRunes = 200

type Request struct {
	ID        uuid.UUID
	CoupleID  uuid.UUID
	AuthorID  uuid.UUID
	AskedOn   time.Time
	Text      string
	PrayedBy  *uuid.UUID
	PrayedAt  *time.Time
	CreatedAt time.Time
}

var (
	ErrNoRequest        = apperr.NotFound("no_request_today", "They haven't asked for anything today.")
	ErrAlreadyPrayedFor = apperr.Conflict("already_prayed_for", "It's already been prayed for, so it can't be changed now.")
)

func ValidateRequestText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", apperr.Validation(map[string]string{
			"text": "Write what you'd like prayer for.",
		})
	}
	if utf8.RuneCountInString(text) > maxRequestRunes {
		return "", apperr.Validation(map[string]string{
			"text": fmt.Sprintf("Keep it under %d characters.", maxRequestRunes),
		})
	}
	return text, nil
}
