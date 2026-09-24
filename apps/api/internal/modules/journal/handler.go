package journal

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
	List(ctx context.Context, userID uuid.UUID) ([]Entry, error)
	Add(ctx context.Context, userID uuid.UUID, tag, text string) (Entry, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /journal", http.HandlerFunc(h.list))
	r.HandleAuthed("POST /journal", http.HandlerFunc(h.add))
}

// The shape in apps/web/src/lib/api/types.ts.
type entryDTO struct {
	ID       string `json:"id"`
	AuthorID string `json:"author_id"`
	Date     string `json:"date"`
	Tag      Tag    `json:"tag"`
	Text     string `json:"text"`
}

func toDTO(e Entry) entryDTO {
	return entryDTO{
		ID:       e.ID.String(),
		AuthorID: e.AuthorID.String(),
		Date:     e.Date.Format(time.DateOnly),
		Tag:      e.Tag,
		Text:     e.Text,
	}
}

type addRequest struct {
	Tag  string `json:"tag"`
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
	out := make([]entryDTO, 0, len(found))
	for _, e := range found {
		out = append(out, toDTO(e))
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req addRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	e, err := h.svc.Add(r.Context(), userID, req.Tag, req.Text)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusCreated, toDTO(e))
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
