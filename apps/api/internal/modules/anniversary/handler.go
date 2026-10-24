package anniversary

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
	Get(ctx context.Context, userID uuid.UUID) (View, error)
	Write(ctx context.Context, userID uuid.UUID, text string) (Letter, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /anniversary", http.HandlerFunc(h.get))
	r.HandleAuthed("PUT /anniversary/letter", http.HandlerFunc(h.write))
}

// The shapes in apps/web/src/lib/api/types.ts.
type letterDTO struct {
	ID        string `json:"id"`
	AuthorID  string `json:"author_id"`
	OpensOn   string `json:"opens_on"`
	Text      string `json:"text"`
	UpdatedAt string `json:"updated_at"`
}

type yearDTO struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Prayed      int    `json:"prayed"`
	Answered    int    `json:"answered"`
	PrayerWeeks int    `json:"prayer_weeks"`
	Memories    int    `json:"memories"`
	Events      int    `json:"events"`
	Challenges  int    `json:"challenges"`
	Notes       int    `json:"notes"`
	Journal     int    `json:"journal"`
}

type viewDTO struct {
	On           string      `json:"on"`
	Years        int         `json:"years"`
	DaysUntil    int         `json:"days_until"`
	Writable     bool        `json:"writable"`
	WritableFrom string      `json:"writable_from"`
	Mine         *letterDTO  `json:"mine"`
	Theirs       *letterDTO  `json:"theirs"`
	TheirsSealed bool        `json:"theirs_sealed"`
	Past         []letterDTO `json:"past"`
	Year         yearDTO     `json:"year"`
}

func toLetterDTO(l *Letter) *letterDTO {
	if l == nil {
		return nil
	}
	return &letterDTO{
		ID:        l.ID.String(),
		AuthorID:  l.AuthorID.String(),
		OpensOn:   l.OpensOn.Format(time.DateOnly),
		Text:      l.Text,
		UpdatedAt: l.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toViewDTO(v View) viewDTO {
	past := make([]letterDTO, 0, len(v.Past))
	for i := range v.Past {
		past = append(past, *toLetterDTO(&v.Past[i]))
	}
	o := v.Occasion
	return viewDTO{
		On:           o.On.Format(time.DateOnly),
		Years:        o.Years,
		DaysUntil:    o.DaysUntil,
		Writable:     o.Writable(),
		WritableFrom: o.On.AddDate(0, 0, -WriteAhead).Format(time.DateOnly),
		Mine:         toLetterDTO(v.Mine),
		Theirs:       toLetterDTO(v.Theirs),
		TheirsSealed: v.TheirsSealed,
		Past:         past,
		Year: yearDTO{
			From:        v.Year.From.Format(time.DateOnly),
			To:          v.Year.To.Format(time.DateOnly),
			Prayed:      v.Year.Prayed,
			Answered:    v.Year.Answered,
			PrayerWeeks: v.Year.PrayerWeeks,
			Memories:    v.Year.Memories,
			Events:      v.Year.Events,
			Challenges:  v.Year.Challenges,
			Notes:       v.Year.Notes,
			Journal:     v.Year.Journal,
		},
	}
}

type writeRequest struct {
	Text string `json:"text"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toViewDTO(v))
}

func (h *Handler) write(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req writeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	l, err := h.svc.Write(r.Context(), userID, req.Text)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toLetterDTO(&l))
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
