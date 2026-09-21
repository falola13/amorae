package couples

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

type service interface {
	Create(ctx context.Context, createdBy uuid.UUID, creatorName string, input CoupleCreateInput) (COUPLES, error)
	Join(ctx context.Context, userID uuid.UUID, code string) (COUPLES, error)
}

type users interface {
	Get(ctx context.Context, id uuid.UUID) (user.User, error)
}

type Handler struct {
	svc   service
	users users
}

func NewHandler(svc service, users users) *Handler {
	return &Handler{svc: svc, users: users}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("POST /couples", http.HandlerFunc(h.create))
	r.HandleAuthed("POST /couples/join", http.HandlerFunc(h.join))
}

type createRequest struct {
	Name                  *string    `json:"name"`
	RelationshipStartDate *time.Time `json:"relationship_start_date"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	createdBy, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	u, err := h.users.Get(r.Context(), createdBy)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	var req createRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	result, err := h.svc.Create(r.Context(), createdBy, u.DisplayName, CoupleCreateInput{
		Name:                  req.Name,
		RelationshipStartDate: req.RelationshipStartDate,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusCreated, ToDTO(result))
}

type joinRequest struct {
	Code string `json:"code"`
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	var req joinRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	result, err := h.svc.Join(r.Context(), userID, req.Code)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, ToDTO(result))
}
