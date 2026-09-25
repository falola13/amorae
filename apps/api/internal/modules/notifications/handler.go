package notifications

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

type service interface {
	Get(ctx context.Context, userID uuid.UUID) (Preferences, error)
	Update(ctx context.Context, userID uuid.UUID, patch Patch) (Preferences, error)
	Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) error
	Nudge(ctx context.Context, senderID uuid.UUID) error
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /notifications/preferences", http.HandlerFunc(h.get))
	r.HandleAuthed("PATCH /notifications/preferences", http.HandlerFunc(h.update))
	r.HandleAuthed("POST /notifications/subscribe", http.HandlerFunc(h.subscribe))
	r.HandleAuthed("POST /nudge", http.HandlerFunc(h.nudge))
}

// The shape in apps/web/src/lib/api/types.ts.
type prefsDTO struct {
	NewWeek        bool   `json:"new_week"`
	PrayerReminder bool   `json:"prayer_reminder"`
	ReminderTime   string `json:"reminder_time"`
	EventReminders bool   `json:"event_reminders"`
	ImportantDates bool   `json:"important_dates"`
	Appreciation   bool   `json:"appreciation"`
	Journal        bool   `json:"journal"`
	Goals          bool   `json:"goals"`
	Challenges     bool   `json:"challenges"`
	PrayerAnswered bool   `json:"prayer_answered"`
	Together       bool   `json:"together"`
	QuietFrom      string `json:"quiet_from"`
	QuietTo        string `json:"quiet_to"`
	DailyCap       int    `json:"daily_cap"`
	MaxDailyCap    int    `json:"max_daily_cap"`
}

func toDTO(p Preferences) prefsDTO {
	return prefsDTO{
		NewWeek:        p.NewWeek,
		PrayerReminder: p.PrayerReminder,
		ReminderTime:   p.ReminderTime,
		EventReminders: p.EventReminders,
		ImportantDates: p.ImportantDates,
		Appreciation:   p.Appreciation,
		Journal:        p.Journal,
		Goals:          p.Goals,
		Challenges:     p.Challenges,
		PrayerAnswered: p.PrayerAnswered,
		Together:       p.Together,
		QuietFrom:      p.QuietFrom,
		QuietTo:        p.QuietTo,
		DailyCap:       p.DailyCap,
		MaxDailyCap:    maxDailyCap,
	}
}

// Every field is a pointer: the client sends one switch at a time, and a
// missing field means "leave it alone" rather than "turn it off".
type patchRequest struct {
	NewWeek        *bool   `json:"new_week"`
	PrayerReminder *bool   `json:"prayer_reminder"`
	ReminderTime   *string `json:"reminder_time"`
	EventReminders *bool   `json:"event_reminders"`
	ImportantDates *bool   `json:"important_dates"`
	Appreciation   *bool   `json:"appreciation"`
	Journal        *bool   `json:"journal"`
	Goals          *bool   `json:"goals"`
	Challenges     *bool   `json:"challenges"`
	PrayerAnswered *bool   `json:"prayer_answered"`
	Together       *bool   `json:"together"`
	QuietFrom      *string `json:"quiet_from"`
	QuietTo        *string `json:"quiet_to"`
	DailyCap       *int    `json:"daily_cap"`
}

// The browser's own PushSubscription.toJSON(), posted as it comes.
type subscribeRequest struct {
	Endpoint       string  `json:"endpoint"`
	ExpirationTime *string `json:"expirationTime"`
	Keys           struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	prefs, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toDTO(prefs))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req patchRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	prefs, err := h.svc.Update(r.Context(), userID, Patch(req))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, toDTO(prefs))
}

func (h *Handler) subscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req subscribeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.Subscribe(r.Context(), userID, req.Endpoint, req.Keys.P256dh, req.Keys.Auth); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}

func (h *Handler) nudge(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	if err := h.svc.Nudge(r.Context(), userID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}
