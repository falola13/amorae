package user

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// service is declared here, by the handler that consumes it, so this file
// depends on a two-method interface rather than the concrete *Service —
// handler_test.go implements this with a fake instead of standing up a
// database.
type service interface {
	Get(ctx context.Context, id uuid.UUID) (User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, input UpdateProfileInput) (User, error)
}

// Handler is transport only: decode the request, call the service, map the
// result to a DTO, respond. Every business rule lives in Service instead.
type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /users/me", http.HandlerFunc(h.getMe))
	r.HandleAuthed("PATCH /users/me", http.HandlerFunc(h.updateMe))
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

	httpx.Data(w, http.StatusOK, ToDTO(u))
}

type updateMeRequest struct {
	DisplayName string `json:"display_name"`
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

	u, err := h.svc.UpdateProfile(r.Context(), id, UpdateProfileInput{DisplayName: req.DisplayName})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, ToDTO(u))
}
