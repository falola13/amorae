package timeline

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// service is the handler's view of the service: only what HTTP calls.
type service interface {
	List(ctx context.Context, userID uuid.UUID, before *time.Time, filter Filter, limit int) (Page, error)
	PhotoURL(it Item) string
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /timeline", http.HandlerFunc(h.list))
}

// The shape in apps/web/src/lib/api/types.ts.
type itemDTO struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	At    string `json:"at"`
	Date  string `json:"date"`
	Title string `json:"title"`
	Sub   string `json:"sub,omitempty"`
	Path  string `json:"path"`
	// Present only for a memory that actually has one — resolving it needs
	// Cloudinary configured, which isn't guaranteed (Service.PhotoURL).
	PhotoURL string `json:"photo_url,omitempty"`
	// Present only when this item has a single actor; a memory, a whole
	// prayer week, or a finished goal has none.
	ActorID string `json:"actor_id,omitempty"`
}

type pageDTO struct {
	Items []itemDTO `json:"items"`
	// null once there is nothing further back — never an empty string, so a
	// client can tell "no more pages" apart from "cursor of zero value".
	Next *string `json:"next"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}

	before, err := ParseCursor(r.URL.Query().Get("before"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	limit, err := ValidateLimit(r.URL.Query().Get("limit"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	filter := ValidateFilter(r.URL.Query().Get("filter"))

	page, err := h.svc.List(r.Context(), userID, before, filter, limit)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := pageDTO{Items: make([]itemDTO, 0, len(page.Items))}
	for _, it := range page.Items {
		out.Items = append(out.Items, toDTO(it, h.svc.PhotoURL(it)))
	}
	if page.Next != nil {
		next := page.Next.UTC().Format(time.RFC3339)
		out.Next = &next
	}
	httpx.Data(w, http.StatusOK, out)
}

func toDTO(it Item, photoURL string) itemDTO {
	out := itemDTO{
		ID:       it.ID.String(),
		Type:     string(it.Type),
		At:       it.At.UTC().Format(time.RFC3339),
		Date:     it.Date.Format(time.DateOnly),
		Title:    it.Title,
		Sub:      it.Sub,
		Path:     it.Path,
		PhotoURL: photoURL,
	}
	if it.ActorID != uuid.Nil {
		out.ActorID = it.ActorID.String()
	}
	return out
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
