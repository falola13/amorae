// Package export builds "download my data" (FR-ACCT-006). It owns no table:
// each section comes from the module that owns that data, through a small
// interface declared here, so later modules add a section without touching
// the others.
package export

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/auth"
	"github.com/falola13/amorae/apps/api/internal/modules/couples"
	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

type users interface {
	Get(ctx context.Context, id uuid.UUID) (user.User, error)
}

type couplesReader interface {
	GetMine(ctx context.Context, userID uuid.UUID) (couples.Mine, error)
}

type consents interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]auth.Consent, error)
}

type Handler struct {
	users    users
	couples  couplesReader
	consents consents
	now      func() time.Time
}

func NewHandler(users users, couples couplesReader, consents consents, now func() time.Time) *Handler {
	return &Handler{users: users, couples: couples, consents: consents, now: now}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /users/me/export", http.HandlerFunc(h.export))
}

type consentDTO struct {
	Kind          string    `json:"kind"`
	PolicyVersion string    `json:"policy_version"`
	CreatedAt     time.Time `json:"created_at"`
}

// memberDTO names a member by display name and role only. The partner's
// email and anything about their credentials stay out (FR-ACCT-006.AC2).
type memberDTO struct {
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	You         bool   `json:"you"`
}

type coupleDTO struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	StartedOn  *string     `json:"started_on,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	Members    []memberDTO `json:"members"`
	InviteCode string      `json:"invite_code,omitempty"`
}

type exportDTO struct {
	ExportedAt time.Time    `json:"exported_at"`
	User       user.DTO     `json:"user"`
	Consents   []consentDTO `json:"consents"`
	Couple     *coupleDTO   `json:"couple"`
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := authctx.UserID(ctx)
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return
	}

	me, err := h.users.Get(ctx, userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	rows, err := h.consents.ListByUser(ctx, userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := exportDTO{ExportedAt: h.now(), User: user.ToDTO(me), Consents: make([]consentDTO, 0, len(rows))}
	for _, c := range rows {
		out.Consents = append(out.Consents, consentDTO{Kind: c.Kind, PolicyVersion: c.PolicyVersion, CreatedAt: c.CreatedAt})
	}

	couple, err := h.couple(ctx, userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out.Couple = couple

	w.Header().Set("Content-Disposition", `attachment; filename="amorae-export.json"`)
	httpx.Data(w, http.StatusOK, out)
}

// couple is nil for someone not in a couple, which is not an error here.
func (h *Handler) couple(ctx context.Context, userID uuid.UUID) (*coupleDTO, error) {
	mine, err := h.couples.GetMine(ctx, userID)
	if errors.Is(err, couples.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c := &coupleDTO{
		ID:         mine.Couple.ID.String(),
		Name:       mine.Couple.Name,
		CreatedAt:  mine.Couple.CreatedAt,
		Members:    make([]memberDTO, 0, len(mine.Members)),
		InviteCode: mine.InviteCode,
	}
	if !mine.Couple.RelationshipStartDate.IsZero() {
		day := mine.Couple.RelationshipStartDate.Format("2006-01-02")
		c.StartedOn = &day
	}
	for _, m := range mine.Members {
		u, err := h.users.Get(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		c.Members = append(c.Members, memberDTO{DisplayName: u.DisplayName, Role: m.Role, You: m.ID == userID})
	}
	return c, nil
}
