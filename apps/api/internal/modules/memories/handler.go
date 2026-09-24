package memories

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
	List(ctx context.Context, userID uuid.UUID) ([]Memory, error)
	Create(ctx context.Context, userID uuid.UUID, in Input) (Memory, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /memories", http.HandlerFunc(h.list))
	r.HandleAuthed("POST /memories", http.HandlerFunc(h.create))
}

// The shape in apps/web/src/lib/api/types.ts.
type memoryDTO struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Date     string `json:"date"`
	Location string `json:"location,omitempty"`
	Note     string `json:"note,omitempty"`
	// Always sent: the screen branches on it to decide whether to leave room
	// for a picture, and an absent field would have to mean false anyway.
	HasPhoto bool `json:"has_photo"`
}

func toDTO(m Memory) memoryDTO {
	return memoryDTO{
		ID:       m.ID.String(),
		Title:    m.Title,
		Date:     m.Date.Format(time.DateOnly),
		Location: m.Location,
		Note:     m.Note,
		HasPhoto: m.HasPhoto,
	}
}

type createRequest struct {
	Title    string `json:"title"`
	Date     string `json:"date"`
	Location string `json:"location"`
	Note     string `json:"note"`
	// Accepted and ignored. The client posts the whole Memory shape it holds,
	// has_photo included, and httpx.Decode refuses a field it has never heard
	// of — so leaving this out would turn every save into invalid_json.
	// Whether a photo exists is the server's to say (FR-MEM-003).
	HasPhoto bool `json:"has_photo"`
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
	out := make([]memoryDTO, 0, len(found))
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

	in := Input{Title: req.Title, Location: req.Location, Note: req.Note}
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
