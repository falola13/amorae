package events

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

type service interface {
	List(ctx context.Context, userID uuid.UUID) ([]Event, error)
	Get(ctx context.Context, userID, eventID uuid.UUID) (Event, error)
	Create(ctx context.Context, userID uuid.UUID, in Input) (Event, error)
	Update(ctx context.Context, userID, eventID uuid.UUID, in Input) (Event, error)
	SetDone(ctx context.Context, userID, eventID uuid.UUID, done bool) (Event, error)
	SetChecklistItem(ctx context.Context, userID, eventID, itemID uuid.UUID, done bool) (Event, error)
	Delete(ctx context.Context, userID, eventID uuid.UUID) error
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /events", http.HandlerFunc(h.list))
	r.HandleAuthed("POST /events", http.HandlerFunc(h.create))
	r.HandleAuthed("GET /events/{id}", http.HandlerFunc(h.get))
	r.HandleAuthed("PATCH /events/{id}", http.HandlerFunc(h.update))
	r.HandleAuthed("DELETE /events/{id}", http.HandlerFunc(h.remove))
	r.HandleAuthed("POST /events/{id}/complete", http.HandlerFunc(h.complete))
	r.HandleAuthed("DELETE /events/{id}/complete", http.HandlerFunc(h.uncomplete))
	r.HandleAuthed("PATCH /events/{id}/checklist/{item}", http.HandlerFunc(h.checklistItem))
}

// The shape in apps/web/src/lib/api/types.ts.
type checklistDTO struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type eventDTO struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Date      string         `json:"date"`
	StartTime string         `json:"start_time,omitempty"`
	EndTime   string         `json:"end_time,omitempty"`
	Location  string         `json:"location,omitempty"`
	Reminder  string         `json:"reminder,omitempty"`
	Notes     string         `json:"notes,omitempty"`
	Checklist []checklistDTO `json:"checklist"`
	Done      bool           `json:"done"`
	Kind      string         `json:"kind"`
	// Null on an event from before ownership existed.
	CreatedBy *string `json:"created_by"`
}

func toDTO(e Event) eventDTO {
	out := eventDTO{
		ID:        e.ID.String(),
		Title:     e.Title,
		Date:      e.Date.Format(time.DateOnly),
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		Location:  e.Location,
		Reminder:  e.Reminder,
		Notes:     e.Notes,
		// Always an array, never null, so a client can map over it.
		Checklist: make([]checklistDTO, 0, len(e.Checklist)),
		Done:      e.Done,
		Kind:      e.Kind,
	}
	if e.CreatedBy != nil {
		id := e.CreatedBy.String()
		out.CreatedBy = &id
	}
	for _, item := range e.Checklist {
		out.Checklist = append(out.Checklist, checklistDTO{
			ID: item.ID.String(), Text: item.Text, Done: item.Done,
		})
	}
	return out
}

// Every field is a pointer: a PATCH mentioning one leaves the rest alone; "" clears it.
type inputRequest struct {
	Title     *string   `json:"title"`
	Date      *string   `json:"date"`
	StartTime *string   `json:"start_time"`
	EndTime   *string   `json:"end_time"`
	Location  *string   `json:"location"`
	Reminder  *string   `json:"reminder"`
	Notes     *string   `json:"notes"`
	Checklist *[]string `json:"checklist"`
	Kind      *string   `json:"kind"`
}

func (r inputRequest) input() Input { return Input(r) }

type doneRequest struct {
	Done bool `json:"done"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	found, err := h.svc.List(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]eventDTO, 0, len(found))
	for _, e := range found {
		out = append(out, toDTO(e))
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, eventID, ok := callerAndEvent(w, r)
	if !ok {
		return
	}
	e, err := h.svc.Get(r.Context(), userID, eventID)
	respond(w, r, e, err, http.StatusOK)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req inputRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	e, err := h.svc.Create(r.Context(), userID, req.input())
	respond(w, r, e, err, http.StatusCreated)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, eventID, ok := callerAndEvent(w, r)
	if !ok {
		return
	}
	var req inputRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	e, err := h.svc.Update(r.Context(), userID, eventID, req.input())
	respond(w, r, e, err, http.StatusOK)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	userID, eventID, ok := callerAndEvent(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), userID, eventID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request)   { h.setDone(w, r, true) }
func (h *Handler) uncomplete(w http.ResponseWriter, r *http.Request) { h.setDone(w, r, false) }

func (h *Handler) setDone(w http.ResponseWriter, r *http.Request, done bool) {
	userID, eventID, ok := callerAndEvent(w, r)
	if !ok {
		return
	}
	e, err := h.svc.SetDone(r.Context(), userID, eventID, done)
	respond(w, r, e, err, http.StatusOK)
}

func (h *Handler) checklistItem(w http.ResponseWriter, r *http.Request) {
	userID, eventID, ok := callerAndEvent(w, r)
	if !ok {
		return
	}
	itemID, ok := pathID(w, r, "item", "item")
	if !ok {
		return
	}
	var req doneRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	e, err := h.svc.SetChecklistItem(r.Context(), userID, eventID, itemID, req.Done)
	respond(w, r, e, err, http.StatusOK)
}

func respond(w http.ResponseWriter, r *http.Request, e Event, err error, status int) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, status, toDTO(e))
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}

func callerAndEvent(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := caller(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	eventID, ok := pathID(w, r, "id", "event")
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return userID, eventID, true
}

// pathID's "thing" names the resource in the error message; never "uuid" or "path parameter".
func pathID(w http.ResponseWriter, r *http.Request, name, thing string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		httpx.Error(w, r, apperr.NotFound("not_found", "That "+thing+" isn’t here."))
		return uuid.UUID{}, false
	}
	return id, true
}
