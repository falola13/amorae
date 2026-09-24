package memories

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/photos"
)

type service interface {
	List(ctx context.Context, userID uuid.UUID) ([]Memory, error)
	Create(ctx context.Context, userID uuid.UUID, in Input) (Memory, error)
	PhotoTicket(ctx context.Context, userID, id uuid.UUID) (photos.Ticket, error)
	AttachPhoto(ctx context.Context, userID, id uuid.UUID) (Memory, error)
	RemovePhoto(ctx context.Context, userID, id uuid.UUID) (Memory, error)
	PhotoURL(m Memory) string
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
	// The browser uploads straight to Cloudinary; this only signs permission
	// and then records that it happened (docs/API.md).
	r.HandleAuthed("POST /memories/{id}/photo/ticket", http.HandlerFunc(h.photoTicket))
	r.HandleAuthed("PUT /memories/{id}/photo", http.HandlerFunc(h.attachPhoto))
	r.HandleAuthed("DELETE /memories/{id}/photo", http.HandlerFunc(h.removePhoto))
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
	// Derived from whether there is a photo, not stored beside it.
	HasPhoto bool `json:"has_photo"`
	// A signed, unguessable delivery address, generated for this request and
	// only for a member of this couple. Absent when there is no photo.
	PhotoURL string `json:"photo_url,omitempty"`
}

func toDTO(m Memory, photoURL string) memoryDTO {
	return memoryDTO{
		ID:       m.ID.String(),
		Title:    m.Title,
		Date:     m.Date.Format(time.DateOnly),
		Location: m.Location,
		Note:     m.Note,
		HasPhoto: m.HasPhoto(),
		PhotoURL: photoURL,
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
		out = append(out, toDTO(m, h.svc.PhotoURL(m)))
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
	httpx.Data(w, http.StatusCreated, toDTO(m, h.svc.PhotoURL(m)))
}

func (h *Handler) photoTicket(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := callerAndMemory(w, r)
	if !ok {
		return
	}
	ticket, err := h.svc.PhotoTicket(r.Context(), userID, id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, ticket)
}

func (h *Handler) attachPhoto(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := callerAndMemory(w, r)
	if !ok {
		return
	}
	// No body on purpose: the only name the ticket could have written to is
	// the one the server derives, so there is nothing to send.
	m, err := h.svc.AttachPhoto(r.Context(), userID, id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toDTO(m, h.svc.PhotoURL(m)))
}

func (h *Handler) removePhoto(w http.ResponseWriter, r *http.Request) {
	userID, id, ok := callerAndMemory(w, r)
	if !ok {
		return
	}
	m, err := h.svc.RemovePhoto(r.Context(), userID, id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toDTO(m, h.svc.PhotoURL(m)))
}

func callerAndMemory(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := caller(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNotFound)
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return userID, id, true
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
