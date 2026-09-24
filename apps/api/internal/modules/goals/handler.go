package goals

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
	List(ctx context.Context, userID uuid.UUID) ([]Goal, error)
	Get(ctx context.Context, userID, goalID uuid.UUID) (Goal, error)
	Create(ctx context.Context, userID uuid.UUID, in Input) (Goal, error)
	Update(ctx context.Context, userID, goalID uuid.UUID, in Input) (Goal, error)
	LogProgress(ctx context.Context, userID, goalID uuid.UUID, amount int64) (Goal, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /goals", http.HandlerFunc(h.list))
	r.HandleAuthed("POST /goals", http.HandlerFunc(h.create))
	r.HandleAuthed("GET /goals/{id}", http.HandlerFunc(h.get))
	r.HandleAuthed("PATCH /goals/{id}", http.HandlerFunc(h.update))
	r.HandleAuthed("POST /goals/{id}/progress", http.HandlerFunc(h.progress))
}

// The shape in apps/web/src/lib/api/types.ts.
type progressDTO struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Amount int64  `json:"amount"`
	Date   string `json:"date"`
}

type goalDTO struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Why       string        `json:"why,omitempty"`
	Target    int64         `json:"target"`
	Unit      Unit          `json:"unit"`
	UnitLabel string        `json:"unit_label,omitempty"`
	Progress  []progressDTO `json:"progress"`
	StartDate string        `json:"start_date"`
	EndDate   string        `json:"end_date"`
	Done      bool          `json:"done"`
}

// The running total is not sent: it is the sum of progress, and a client that
// adds it up itself can never disagree with one that was told (BR-GOAL-01).
func toDTO(g Goal) goalDTO {
	out := goalDTO{
		ID:        g.ID.String(),
		Title:     g.Title,
		Why:       g.Why,
		Target:    g.Target,
		Unit:      g.Unit,
		UnitLabel: g.UnitLabel,
		Progress:  make([]progressDTO, 0, len(g.Progress)),
		StartDate: g.StartDate.Format(time.DateOnly),
		EndDate:   g.EndDate.Format(time.DateOnly),
		Done:      g.Done,
	}
	for _, p := range g.Progress {
		out.Progress = append(out.Progress, progressDTO{
			ID: p.ID.String(), UserID: p.UserID.String(),
			Amount: p.Amount, Date: p.Date.Format(time.DateOnly),
		})
	}
	return out
}

type inputRequest struct {
	Title     *string `json:"title"`
	Why       *string `json:"why"`
	Target    *int64  `json:"target"`
	Unit      *string `json:"unit"`
	UnitLabel *string `json:"unit_label"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Done      *bool   `json:"done"`
}

type progressRequest struct {
	Amount int64 `json:"amount"`
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
	out := make([]goalDTO, 0, len(found))
	for _, g := range found {
		out = append(out, toDTO(g))
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, goalID, ok := callerAndGoal(w, r)
	if !ok {
		return
	}
	g, err := h.svc.Get(r.Context(), userID, goalID)
	respond(w, r, g, err, http.StatusOK)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req inputRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	g, err := h.svc.Create(r.Context(), userID, Input(req))
	respond(w, r, g, err, http.StatusCreated)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, goalID, ok := callerAndGoal(w, r)
	if !ok {
		return
	}
	var req inputRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	g, err := h.svc.Update(r.Context(), userID, goalID, Input(req))
	respond(w, r, g, err, http.StatusOK)
}

func (h *Handler) progress(w http.ResponseWriter, r *http.Request) {
	userID, goalID, ok := callerAndGoal(w, r)
	if !ok {
		return
	}
	var req progressRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	g, err := h.svc.LogProgress(r.Context(), userID, goalID, req.Amount)
	respond(w, r, g, err, http.StatusOK)
}

func respond(w http.ResponseWriter, r *http.Request, g Goal, err error, status int) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, status, toDTO(g))
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}

func callerAndGoal(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := caller(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	goalID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, apperr.NotFound("not_found", "That goal isn’t here."))
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return userID, goalID, true
}
