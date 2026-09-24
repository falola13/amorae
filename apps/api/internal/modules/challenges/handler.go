package challenges

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

type service interface {
	Current(ctx context.Context, userID uuid.UUID) (Viewer, error)
	Start(ctx context.Context, userID uuid.UUID, key string) (Viewer, error)
	Mark(ctx context.Context, userID uuid.UUID, n int, done, skipped *bool) (Viewer, error)
	Leave(ctx context.Context, userID uuid.UUID) error
}

// partners answers which of a couple's two members is not the caller, so a
// view can label one set of marks "theirs".
type partners interface {
	PartnerOf(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type Handler struct {
	svc      service
	partners partners
}

func NewHandler(svc service, p partners) *Handler {
	return &Handler{svc: svc, partners: p}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	// Before /challenges/current, so the literal path is not read as an id
	// by a future route that takes one.
	r.HandleAuthed("GET /challenges/templates", http.HandlerFunc(h.templates))
	r.HandleAuthed("GET /challenges/current", http.HandlerFunc(h.current))
	r.HandleAuthed("POST /challenges", http.HandlerFunc(h.start))
	r.HandleAuthed("DELETE /challenges/current", http.HandlerFunc(h.leave))
	r.HandleAuthed("PATCH /challenges/current/days/{n}", http.HandlerFunc(h.markDay))
}

// The shape in apps/web/src/lib/api/types.ts. `done` and `skipped` are the
// caller's own, exactly as my_completed is on a prayer week; the partner's
// sit alongside so both can see both, and neither can change the other's
// (DEC-30).
type dayDTO struct {
	N              int    `json:"n"`
	Text           string `json:"text"`
	Done           bool   `json:"done"`
	Skipped        bool   `json:"skipped,omitempty"`
	PartnerDone    bool   `json:"partner_done,omitempty"`
	PartnerSkipped bool   `json:"partner_skipped,omitempty"`
}

type challengeDTO struct {
	ID        string   `json:"id"`
	Template  string   `json:"template"`
	Title     string   `json:"title"`
	StartedOn string   `json:"started_on"`
	Days      []dayDTO `json:"days"`
}

type templateDTO struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Blurb string `json:"blurb"`
	Days  int    `json:"days"`
}

func toDTO(v Viewer) challengeDTO {
	out := challengeDTO{
		ID:        v.ID.String(),
		Template:  v.Template,
		Title:     v.Title,
		StartedOn: v.StartedOn.Format(time.DateOnly),
		Days:      make([]dayDTO, 0, len(v.Days)),
	}
	for _, d := range v.Days {
		mine, _ := d.MarkFor(v.Me)
		theirs, _ := d.MarkFor(v.Partner)
		out.Days = append(out.Days, dayDTO{
			N:              d.N,
			Text:           d.Prompt,
			Done:           mine == MarkDone,
			Skipped:        mine == MarkSkipped,
			PartnerDone:    theirs == MarkDone,
			PartnerSkipped: theirs == MarkSkipped,
		})
	}
	return out
}

type startRequest struct {
	Template string `json:"template"`
}

type markRequest struct {
	Done    *bool `json:"done"`
	Skipped *bool `json:"skipped"`
}

func (h *Handler) templates(w http.ResponseWriter, r *http.Request) {
	if _, ok := caller(w, r); !ok {
		return
	}
	all := Templates()
	out := make([]templateDTO, 0, len(all))
	for _, t := range all {
		out = append(out, templateDTO{Key: t.Key, Title: t.Title, Blurb: t.Blurb, Days: len(t.Prompts)})
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) current(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Current(r.Context(), userID)
	h.respond(w, r, v, err, http.StatusOK)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req startRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v, err := h.svc.Start(r.Context(), userID, req.Template)
	h.respond(w, r, v, err, http.StatusCreated)
}

func (h *Handler) leave(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	if err := h.svc.Leave(r.Context(), userID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) markDay(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 {
		httpx.Error(w, r, ErrUnknownDay)
		return
	}
	var req markRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v, err := h.svc.Mark(r.Context(), userID, n, req.Done, req.Skipped)
	h.respond(w, r, v, err, http.StatusOK)
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, v Viewer, err error, status int) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	// Who the partner is only decides how the view is labelled, so a couple
	// still waiting for one reads as "nobody else has marked anything"
	// rather than failing.
	if partner, perr := h.partners.PartnerOf(r.Context(), v.Me); perr == nil {
		v.Partner = partner
	}
	httpx.Data(w, status, toDTO(v))
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
