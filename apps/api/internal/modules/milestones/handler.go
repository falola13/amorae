package milestones

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
	List(ctx context.Context, userID uuid.UUID) ([]Milestone, error)
	Create(ctx context.Context, userID uuid.UUID, in Input) (Milestone, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /milestones", http.HandlerFunc(h.list))
	r.HandleAuthed("POST /milestones", http.HandlerFunc(h.create))
}

// The shape in apps/web/src/lib/api/types.ts.
type milestoneDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
	Sub   string `json:"sub,omitempty"`
	// Always sent, never omitted: the list screen branches on it, and an
	// absent field and a false one would have to mean the same thing.
	Reminder bool `json:"reminder"`
}

func toDTO(m Milestone) milestoneDTO {
	return milestoneDTO{
		ID:       m.ID.String(),
		Title:    m.Title,
		Date:     m.Date.Format(time.DateOnly),
		Sub:      m.Sub,
		Reminder: m.Reminder,
	}
}

type createRequest struct {
	Title string `json:"title"`
	Date  string `json:"date"`
	Sub   string `json:"sub"`
	// Absent means yes. The composer sends it either way, but a date worth
	// writing down is usually a date worth coming round again.
	Reminder *bool `json:"reminder"`
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
	out := make([]milestoneDTO, 0, len(found))
	for _, m := range found {
		out = append(out, toDTO(m))
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req createRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	in := Input{Title: req.Title, Sub: req.Sub, Reminder: req.Reminder == nil || *req.Reminder}
	if req.Date != "" {
		date, err := time.Parse(time.DateOnly, req.Date)
		if err != nil {
			httpx.Error(w, r, apperr.Validation(map[string]string{"date": "Pick a date."}))
			return
		}
		in.Date = date
	}

	m, err := h.svc.Create(r.Context(), userID, in)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusCreated, toDTO(m))
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
