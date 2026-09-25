package appreciation

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
	List(ctx context.Context, userID uuid.UUID) ([]Appreciation, error)
	Send(ctx context.Context, userID uuid.UUID, text string) (Appreciation, error)
	Undo(ctx context.Context, userID, id uuid.UUID) error
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /appreciations", http.HandlerFunc(h.list))
	r.HandleAuthed("POST /appreciations", http.HandlerFunc(h.send))
	r.HandleAuthed("DELETE /appreciations/{id}", http.HandlerFunc(h.undo))
}

// The shape in apps/web/src/lib/api/types.ts. No recipient (BR-APPR-01).
type appreciationDTO struct {
	ID     string `json:"id"`
	FromID string `json:"from_id"`
	Date   string `json:"date"`
	Text   string `json:"text"`
}

func toDTO(a Appreciation) appreciationDTO {
	return appreciationDTO{
		ID:     a.ID.String(),
		FromID: a.FromID.String(),
		Date:   a.Date.Format(time.DateOnly),
		Text:   a.Text,
	}
}

type sendRequest struct {
	Text string `json:"text"`
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
	out := make([]appreciationDTO, 0, len(found))
	for _, a := range found {
		out = append(out, toDTO(a))
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req sendRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	a, err := h.svc.Send(r.Context(), userID, req.Text)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusCreated, toDTO(a))
}

func (h *Handler) undo(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNotFound)
		return
	}
	if err := h.svc.Undo(r.Context(), userID, id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
