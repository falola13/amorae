package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
	"github.com/falola13/amorae/apps/api/internal/platform/photos"
)

type service interface {
	Get(ctx context.Context, id uuid.UUID) (User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (User, error)
	PhotoTicket(ctx context.Context, userID uuid.UUID) (photos.Ticket, error)
	AttachPhoto(ctx context.Context, userID uuid.UUID) (User, error)
	RemovePhoto(ctx context.Context, userID uuid.UUID) (User, error)
	PhotoURL(u User) string
}

// Handler is transport only; business rules live in Service.
type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /users/me", http.HandlerFunc(h.getMe))
	r.HandleAuthed("PATCH /users/me", http.HandlerFunc(h.updateMe))
	// DELETE /users/me is registered by auth: it re-checks the password.
	// The browser uploads straight to Cloudinary; this only signs permission
	// and then records that it happened (docs/API.md), same as memories'.
	r.HandleAuthed("POST /users/me/photo/ticket", http.HandlerFunc(h.photoTicket))
	r.HandleAuthed("PUT /users/me/photo", http.HandlerFunc(h.attachPhoto))
	r.HandleAuthed("DELETE /users/me/photo", http.HandlerFunc(h.removePhoto))
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	id, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	u, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, ToDTOWithPhoto(u, h.svc.PhotoURL(u)))
}

func (h *Handler) photoTicket(w http.ResponseWriter, r *http.Request) {
	id, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}
	ticket, err := h.svc.PhotoTicket(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, ticket)
}

func (h *Handler) attachPhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}
	// No body on purpose: the ticket's name is server-derived, so there is nothing to send.
	u, err := h.svc.AttachPhoto(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, ToDTOWithPhoto(u, h.svc.PhotoURL(u)))
}

func (h *Handler) removePhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}
	u, err := h.svc.RemovePhoto(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, ToDTOWithPhoto(u, h.svc.PhotoURL(u)))
}

// Email is deliberately absent: changing it needs the current password, via
// PUT /users/me/email in auth; sending "email" here is rejected as unknown.
//
// Birthday is a json.RawMessage rather than *birthdayDTO because a pointer
// can't tell "the key was absent" from "the key was sent as null" apart —
// both decode to nil — and those two mean different things here: leave it
// alone, versus clear it.
type updateMeRequest struct {
	DisplayName string          `json:"display_name"`
	Timezone    string          `json:"timezone"`
	Birthday    json.RawMessage `json:"birthday"`
}

func (h *Handler) updateMe(w http.ResponseWriter, r *http.Request) {
	id, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	var req updateMeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	input := UpdateProfileInput{DisplayName: req.DisplayName, Timezone: req.Timezone}
	if req.Birthday != nil {
		patch, err := decodeBirthdayPatch(req.Birthday)
		if err != nil {
			httpx.Error(w, r, err)
			return
		}
		input.Birthday = patch
	}

	u, err := h.svc.UpdateProfile(r.Context(), id, input)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, ToDTOWithPhoto(u, h.svc.PhotoURL(u)))
}

// decodeBirthdayPatch reads a present "birthday" key: the JSON literal null
// clears it, anything else must decode as a birthdayDTO to set it.
func decodeBirthdayPatch(raw json.RawMessage) (BirthdayPatch, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return BirthdayPatch{Present: true, Clear: true}, nil
	}

	var b birthdayDTO
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		return BirthdayPatch{}, apperr.Validation(map[string]string{"birthday": "That isn’t a date."})
	}
	return BirthdayPatch{Present: true, Value: Birthday{Month: b.Month, Day: b.Day, Year: b.Year}}, nil
}
