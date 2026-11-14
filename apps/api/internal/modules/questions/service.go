package questions

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository is what this service needs of storage.
type Repository interface {
	Timezone(ctx context.Context, coupleID uuid.UUID) (string, error)
	AnswersOn(ctx context.Context, coupleID uuid.UUID, day time.Time) ([]Answer, error)
	Keep(ctx context.Context, a Answer, at time.Time) (Answer, error)
	// Before is every answer on the most recent `days` days that have one,
	// strictly earlier than `before`, newest day first.
	Before(ctx context.Context, coupleID uuid.UUID, before time.Time, days int) ([]Answer, error)
}

type Couples interface {
	CoupleFor(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// Poker asks the notifications worker for a pass now, so the other partner
// hears an answer is waiting without waiting for the next cron tick.
type Poker interface{ Poke() }

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type Service struct {
	repo    Repository
	couples Couples
	now     func() time.Time
	poker   Poker
}

func NewService(repo Repository, couples Couples, now func() time.Time, poker Poker) *Service {
	return &Service{repo: repo, couples: couples, now: now, poker: poker}
}

func (s *Service) today(ctx context.Context, userID uuid.UUID) (coupleID uuid.UUID, day time.Time, err error) {
	coupleID, err = s.couples.CoupleFor(ctx, userID)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	zone, err := s.repo.Timezone(ctx, coupleID)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return coupleID, LocalToday(s.now(), zone), nil
}

// Today is today's question and what the caller is allowed to see of it.
func (s *Service) Today(ctx context.Context, userID uuid.UUID) (Entry, error) {
	coupleID, day, err := s.today(ctx, userID)
	if err != nil {
		return Entry{}, err
	}
	answers, err := s.repo.AnswersOn(ctx, coupleID, day)
	if err != nil {
		return Entry{}, err
	}
	return assemble(userID, day, QuestionFor(coupleID, day), answers), nil
}

// Answer keeps the caller's answer to today's question, or rewrites it. Only
// today can be answered: a day that has passed is closed.
func (s *Service) Answer(ctx context.Context, userID uuid.UUID, text string) (Entry, error) {
	clean, err := ValidateAnswer(text)
	if err != nil {
		return Entry{}, err
	}
	coupleID, day, err := s.today(ctx, userID)
	if err != nil {
		return Entry{}, err
	}
	if _, err := s.repo.Keep(ctx, Answer{
		CoupleID: coupleID,
		AuthorID: userID,
		AskedOn:  day,
		Question: QuestionFor(coupleID, day),
		Text:     clean,
	}, s.now()); err != nil {
		return Entry{}, err
	}
	// Only the first answer is announced (the worker reads created_at), so a
	// poke on a rewrite costs a pass and sends nothing.
	if s.poker != nil {
		s.poker.Poke()
	}
	return s.Today(ctx, userID)
}

// Page is a run of past days, newest first. Next is the day to pass as
// "before" for the following page, or nil at the end.
type Page struct {
	Entries []Entry
	Next    *time.Time
}

// Past lists earlier days, each sealed by the same rule as today's.
func (s *Service) Past(ctx context.Context, userID uuid.UUID, before *time.Time, limit int) (Page, error) {
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	coupleID, today, err := s.today(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	// Today is read through Today, never the archive.
	cutoff := today
	if before != nil && before.Before(today) {
		cutoff = *before
	}
	// One extra day tells us whether there is another page.
	answers, err := s.repo.Before(ctx, coupleID, cutoff, limit+1)
	if err != nil {
		return Page{}, err
	}
	var order []time.Time
	byDay := map[time.Time][]Answer{}
	for _, a := range answers {
		if _, seen := byDay[a.AskedOn]; !seen {
			order = append(order, a.AskedOn)
		}
		byDay[a.AskedOn] = append(byDay[a.AskedOn], a)
	}
	page := Page{Entries: []Entry{}}
	for i, day := range order {
		if i == limit {
			next := order[limit-1]
			page.Next = &next
			break
		}
		page.Entries = append(page.Entries, assemble(userID, day, QuestionFor(coupleID, day), byDay[day]))
	}
	return page, nil
}
