package events

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/database"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Every method takes a couple id, never a user id (DEC-19); an event outside it is simply not found.

func (r *PostgresRepository) List(ctx context.Context, coupleID uuid.UUID) ([]Event, error) {
	return r.load(ctx, `WHERE e.couple_id = $1`, coupleID)
}

func (r *PostgresRepository) ByID(ctx context.Context, coupleID, eventID uuid.UUID) (Event, error) {
	found, err := r.load(ctx, `WHERE e.couple_id = $1 AND e.id = $2`, coupleID, eventID)
	if err != nil {
		return Event{}, err
	}
	if len(found) == 0 {
		return Event{}, ErrNotFound
	}
	return found[0], nil
}

// load reads events and checklists in two queries total, avoiding one query per event.
func (r *PostgresRepository) load(ctx context.Context, where string, args ...any) ([]Event, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `
		SELECT e.id, e.couple_id, e.title, e.date,
		       COALESCE(to_char(e.start_time, 'HH24:MI'), ''),
		       COALESCE(to_char(e.end_time, 'HH24:MI'), ''),
		       COALESCE(e.location, ''), e.reminders, COALESCE(e.notes, ''), e.done, e.didnt_happen,
		       e.created_by, e.kind
		FROM events e `+where+`
		ORDER BY e.date, e.start_time NULLS FIRST, e.created_at
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("loading events: %w", err)
	}

	var out []Event
	at := map[uuid.UUID]int{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.CoupleID, &e.Title, &e.Date, &e.StartTime, &e.EndTime,
			&e.Location, &e.Reminders, &e.Notes, &e.Done, &e.DidntHappen, &e.CreatedBy, &e.Kind); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		at[e.ID] = len(out)
		out = append(out, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loading events: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(out))
	for _, e := range out {
		ids = append(ids, e.ID)
	}

	items, err := r.db.Q(ctx).Query(ctx, `
		SELECT id, event_id, position, text, done
		FROM event_checklist_items WHERE event_id = ANY($1)
		ORDER BY event_id, position
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("loading checklists: %w", err)
	}
	defer items.Close()

	for items.Next() {
		var item ChecklistItem
		var eventID uuid.UUID
		if err := items.Scan(&item.ID, &eventID, &item.Position, &item.Text, &item.Done); err != nil {
			return nil, fmt.Errorf("scanning checklist item: %w", err)
		}
		i := at[eventID]
		out[i].Checklist = append(out[i].Checklist, item)
	}
	if err := items.Err(); err != nil {
		return nil, fmt.Errorf("loading checklists: %w", err)
	}
	return out, nil
}

// Create writes the event and checklist in one transaction so neither exists without the other.
func (r *PostgresRepository) Create(ctx context.Context, e Event, at time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("generating event id: %w", err)
	}
	err = r.db.InTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.Q(ctx).Exec(ctx, `
			INSERT INTO events (id, couple_id, title, date, start_time, end_time, location, reminders, notes, created_by, kind, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
		`, id, e.CoupleID, e.Title, e.Date, nullIfEmpty(e.StartTime), nullIfEmpty(e.EndTime),
			nullIfEmpty(e.Location), remindersOrEmpty(e.Reminders), nullIfEmpty(e.Notes),
			e.CreatedBy, e.Kind, at); err != nil {
			return fmt.Errorf("creating event: %w", err)
		}
		return r.writeChecklist(ctx, id, e.Checklist, at)
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

// Update writes a whole picture (already merged by the service) rather than merging in SQL.
func (r *PostgresRepository) Update(ctx context.Context, e Event, replaceChecklist bool, at time.Time) error {
	return r.db.InTx(ctx, func(ctx context.Context) error {
		tag, err := r.db.Q(ctx).Exec(ctx, `
			UPDATE events
			SET title = $3, date = $4, start_time = $5, end_time = $6,
			    location = $7, reminders = $8, notes = $9, kind = $10, updated_at = $11
			WHERE id = $1 AND couple_id = $2
		`, e.ID, e.CoupleID, e.Title, e.Date, nullIfEmpty(e.StartTime), nullIfEmpty(e.EndTime),
			nullIfEmpty(e.Location), remindersOrEmpty(e.Reminders), nullIfEmpty(e.Notes), e.Kind, at)
		if err != nil {
			return fmt.Errorf("updating event: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		if !replaceChecklist {
			return nil
		}
		if _, err := r.db.Q(ctx).Exec(ctx,
			`DELETE FROM event_checklist_items WHERE event_id = $1`, e.ID); err != nil {
			return fmt.Errorf("clearing checklist: %w", err)
		}
		return r.writeChecklist(ctx, e.ID, e.Checklist, at)
	})
}

func (r *PostgresRepository) writeChecklist(ctx context.Context, eventID uuid.UUID, items []ChecklistItem, at time.Time) error {
	for _, item := range items {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generating checklist id: %w", err)
		}
		if _, err := r.db.Q(ctx).Exec(ctx, `
			INSERT INTO event_checklist_items (id, event_id, position, text, done, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6)
		`, id, eventID, item.Position, item.Text, item.Done, at); err != nil {
			return fmt.Errorf("adding checklist item: %w", err)
		}
	}
	return nil
}

// SetOutcome is idempotent, which lets the toggle be queued offline.
func (r *PostgresRepository) SetOutcome(ctx context.Context, coupleID, eventID uuid.UUID, done, didntHappen bool, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE events SET done = $3, didnt_happen = $4, updated_at = $5 WHERE id = $2 AND couple_id = $1
	`, coupleID, eventID, done, didntHappen, at)
	if err != nil {
		return fmt.Errorf("updating event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetChecklistItem touches one item only (FR-EVT-005.AC1); scoped through the event to the couple.
func (r *PostgresRepository) SetChecklistItem(ctx context.Context, coupleID, eventID, itemID uuid.UUID, done bool, at time.Time) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `
		UPDATE event_checklist_items i
		SET done = $4, updated_at = $5
		FROM events e
		WHERE i.id = $3 AND i.event_id = $2 AND e.id = i.event_id AND e.couple_id = $1
	`, coupleID, eventID, itemID, done, at)
	if err != nil {
		return fmt.Errorf("updating checklist item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes an event and its checklist (FR-EVT-006).
func (r *PostgresRepository) Delete(ctx context.Context, coupleID, eventID uuid.UUID) error {
	tag, err := r.db.Q(ctx).Exec(ctx,
		`DELETE FROM events WHERE id = $2 AND couple_id = $1`, coupleID, eventID)
	if err != nil {
		return fmt.Errorf("deleting event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// remindersOrEmpty keeps a nil list from becoming a NULL in a column that has
// none: no reminders is an empty array.
func remindersOrEmpty(r []string) []string {
	if r == nil {
		return []string{}
	}
	return r
}
