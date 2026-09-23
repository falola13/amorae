package couples

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

type service interface {
	Create(ctx context.Context, createdBy uuid.UUID, creatorName string, input CoupleCreateInput) (COUPLES, error)
	Join(ctx context.Context, userID uuid.UUID, code string) (Mine, error)
	GetMine(ctx context.Context, userID uuid.UUID) (Mine, error)
	UpdateCouples(ctx context.Context, userID uuid.UUID, update UpdateDto) (Mine, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role string) (Mine, error)
	UpdateOnboarding(ctx context.Context, userID uuid.UUID, patch OnboardingDto) (Mine, error)
	RegenerateInvite(ctx context.Context, userID uuid.UUID) (Mine, error)
	LeaveCouple(ctx context.Context, userID uuid.UUID) ([]Mine, error)
	Archived(ctx context.Context, userID uuid.UUID) ([]Mine, error)
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
	r.HandleAuthed("GET /couples/me", http.HandlerFunc(h.getMe))
	r.HandleAuthed("POST /couples", http.HandlerFunc(h.create))
	r.HandleAuthed("POST /couples/join", http.HandlerFunc(h.join))
	r.HandleAuthed("PATCH /couples/me", http.HandlerFunc(h.updateCouple))
	r.HandleAuthed("PATCH /couples/role", http.HandlerFunc(h.updateRole))
	r.HandleAuthed("PATCH /couples/me/onboarding", http.HandlerFunc(h.updateOnboarding))
	r.HandleAuthed("POST /couples/invite", http.HandlerFunc(h.regenerateInvite))
	r.HandleAuthed("DELETE /couples/me", http.HandlerFunc(h.leaveCouple))
	r.HandleAuthed("GET /couples/archived", http.HandlerFunc(h.archived))
}

// Leaving ends the couple for both partners rather than removing one of them
// (FR-PAIR-008). Neither is in a couple afterwards — both are free to start
// again — so it answers with the archive: what the couple was, and how long
// is left to read and download it.
func (h *Handler) leaveCouple(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := authctx.UserID(ctx)
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	archived, err := h.svc.LeaveCouple(ctx, userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.endedDTOs(ctx, archived)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, dto)
}

// archived is the same list on its own, for a client coming back later.
func (h *Handler) archived(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := authctx.UserID(ctx)
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	archived, err := h.svc.Archived(ctx, userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.endedDTOs(ctx, archived)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, dto)
}

// endedDTOs looks up the names of everyone involved. Always an empty array
// rather than null, so a client can render the list without a nil check.
func (h *Handler) endedDTOs(ctx context.Context, archived []Mine) ([]EndedCoupleDTO, error) {
	out := make([]EndedCoupleDTO, 0, len(archived))
	names := map[uuid.UUID]string{}
	for _, mine := range archived {
		for _, m := range mine.Members {
			if _, known := names[m.ID]; known {
				continue
			}
			u, err := h.users.Get(ctx, m.ID)
			if err != nil {
				// A partner who has since deleted their account is simply
				// not listed; the rest of the record still belongs to you.
				continue
			}
			names[m.ID] = u.DisplayName
		}
		out = append(out, ToEndedDTO(mine, names))
	}
	return out, nil
}

func (h *Handler) regenerateInvite(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	mine, err := h.svc.RegenerateInvite(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.mineDTO(r.Context(), userID, mine)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, dto)
}

type createRequest struct {
	Name                  *string `json:"name"`
	RelationshipStartDate *string `json:"relationship_start_date"`
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
	if err := httpx.DecodeOptional(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	if _, err := h.svc.Create(r.Context(), createdBy, u.DisplayName, CoupleCreateInput{
		Name:                  req.Name,
		RelationshipStartDate: req.RelationshipStartDate,
		Timezone:              u.Timezone,
	}); err != nil {
		httpx.Error(w, r, err)
		return
	}

	mine, err := h.svc.GetMine(r.Context(), createdBy)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.mineDTO(r.Context(), createdBy, mine)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusCreated, dto)
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

	mine, err := h.svc.Join(r.Context(), userID, req.Code)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.mineDTO(r.Context(), userID, mine)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, dto)
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	mine, err := h.svc.GetMine(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.mineDTO(r.Context(), userID, mine)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, dto)
}

func (h *Handler) mineDTO(ctx context.Context, userID uuid.UUID, mine Mine) (MineDTO, error) {
	me, err := h.users.Get(ctx, userID)
	if err != nil {
		return MineDTO{}, err
	}

	var partner *user.User
	for _, m := range mine.Members {
		if m.ID == userID {
			continue
		}
		p, err := h.users.Get(ctx, m.ID)
		if err != nil {
			return MineDTO{}, err
		}
		partner = &p
		break
	}

	return ToMineDTO(mine, me, partner), nil
}

func (h *Handler) updateCouple(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := authctx.UserID(ctx)
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	var req UpdateDto
	if err := httpx.DecodeOptional(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	mine, err := h.svc.UpdateCouples(ctx, userID, req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.mineDTO(ctx, userID, mine)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, dto)
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := authctx.UserID(ctx)
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	var role struct {
		Role string `json:"role"`
	}
	if err := httpx.Decode(w, r, &role); err != nil {
		httpx.Error(w, r, err)
		return
	}
	mine, err := h.svc.UpdateRole(ctx, userID, role.Role)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.mineDTO(ctx, userID, mine)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, dto)
}

func (h *Handler) updateOnboarding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := authctx.UserID(ctx)
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	var req OnboardingDto
	if err := httpx.DecodeOptional(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	mine, err := h.svc.UpdateOnboarding(ctx, userID, req)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	dto, err := h.mineDTO(ctx, userID, mine)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, dto)
}
