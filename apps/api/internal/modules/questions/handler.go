package questions

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
	Today(ctx context.Context, userID uuid.UUID) (Entry, error)
	Answer(ctx context.Context, userID uuid.UUID, text string) (Entry, error)
	Past(ctx context.Context, userID uuid.UUID, before *time.Time, limit int) (Page, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /questions/today", http.HandlerFunc(h.today))
	r.HandleAuthed("PUT /questions/today", http.HandlerFunc(h.answer))
	r.HandleAuthed("GET /questions", http.HandlerFunc(h.past))
}

// The shapes in apps/web/src/lib/api/types.ts.
type answerDTO struct {
	Text      string `json:"text"`
	UpdatedAt string `json:"updated_at"`
}

type entryDTO struct {
	AskedOn        string     `json:"asked_on"`
	Question       string     `json:"question"`
	Mine           *answerDTO `json:"mine"`
	Theirs         *answerDTO `json:"theirs"`
	TheirsAnswered bool       `json:"theirs_answered"`
}

type pageDTO struct {
	Items []entryDTO `json:"items"`
	Next  *string    `json:"next"`
}

func toAnswerDTO(a *Answer) *answerDTO {
	if a == nil {
		return nil
	}
	return &answerDTO{Text: a.Text, UpdatedAt: a.UpdatedAt.UTC().Format(time.RFC3339)}
}

func toEntryDTO(e Entry) entryDTO {
	return entryDTO{
		AskedOn:        e.AskedOn.Format(time.DateOnly),
		Question:       e.Question,
		Mine:           toAnswerDTO(e.Mine),
		Theirs:         toAnswerDTO(e.Theirs),
		TheirsAnswered: e.TheirsAnswered,
	}
}

func (h *Handler) today(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	e, err := h.svc.Today(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toEntryDTO(e))
}

type answerRequest struct {
	Text string `json:"text"`
}

func (h *Handler) answer(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req answerRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	e, err := h.svc.Answer(r.Context(), userID, req.Text)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toEntryDTO(e))
}

func (h *Handler) past(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	before, limit, err := parsePaging(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	page, err := h.svc.Past(r.Context(), userID, before, limit)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := pageDTO{Items: make([]entryDTO, 0, len(page.Entries))}
	for _, e := range page.Entries {
		out.Items = append(out.Items, toEntryDTO(e))
	}
	if page.Next != nil {
		next := page.Next.Format(time.DateOnly)
		out.Next = &next
	}
	httpx.Data(w, http.StatusOK, out)
}

// parsePaging reads ?before=YYYY-MM-DD and ?limit=N. Limit defaults to 20 and
// is held to 50.
func parsePaging(r *http.Request) (*time.Time, int, error) {
	var before *time.Time
	if raw := r.URL.Query().Get("before"); raw != "" {
		t, err := time.Parse(time.DateOnly, raw)
		if err != nil {
			return nil, 0, apperr.Validation(map[string]string{"before": "That is not a valid date."})
		}
		before = &t
	}
	limit := DefaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, 0, apperr.Validation(map[string]string{"limit": "That is not a number."})
		}
		limit = min(max(n, 1), MaxLimit)
	}
	return before, limit, nil
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
