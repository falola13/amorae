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
	Active(ctx context.Context, userID uuid.UUID) ([]Viewer, error)
	Current(ctx context.Context, userID uuid.UUID) (Viewer, error)
	Get(ctx context.Context, userID, id uuid.UUID) (Viewer, error)
	Past(ctx context.Context, userID uuid.UUID) ([]Summary, error)
	Start(ctx context.Context, userID uuid.UUID, in StartInput) (Viewer, error)
	Edit(ctx context.Context, userID, id uuid.UUID, in EditInput) (Viewer, error)
	ReplacePlan(ctx context.Context, userID, id uuid.UUID, prompts []string) (Viewer, error)
	Mark(ctx context.Context, userID, id uuid.UUID, n int, done, skipped *bool, note *string) (Viewer, error)
	Leave(ctx context.Context, userID, id uuid.UUID) error
	Reflect(ctx context.Context, userID, id uuid.UUID, text string) (Viewer, error)
}

// partners answers which of a couple's two members is not the caller, for labeling marks "theirs".
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
	// The literal paths (templates, active, current, past) win over {id} in the mux, so they are not read as an id.
	r.HandleAuthed("GET /challenges/templates", http.HandlerFunc(h.templates))
	r.HandleAuthed("GET /challenges/active", http.HandlerFunc(h.active))
	// current, and the two /current routes below, are aliases for older
	// clients from when a couple had one challenge: they act on the most
	// recently started one that is going.
	r.HandleAuthed("GET /challenges/current", http.HandlerFunc(h.current))
	r.HandleAuthed("GET /challenges/past", http.HandlerFunc(h.past))
	r.HandleAuthed("GET /challenges/{id}", http.HandlerFunc(h.byID))
	r.HandleAuthed("PUT /challenges/{id}/reflection", http.HandlerFunc(h.reflect))
	r.HandleAuthed("PATCH /challenges/{id}", http.HandlerFunc(h.edit))
	r.HandleAuthed("PUT /challenges/{id}/plan", http.HandlerFunc(h.plan))
	r.HandleAuthed("POST /challenges", http.HandlerFunc(h.start))
	r.HandleAuthed("PATCH /challenges/{id}/days/{n}", http.HandlerFunc(h.markDay))
	r.HandleAuthed("DELETE /challenges/{id}", http.HandlerFunc(h.leave))
	r.HandleAuthed("DELETE /challenges/current", http.HandlerFunc(h.leaveCurrent))
	r.HandleAuthed("PATCH /challenges/current/days/{n}", http.HandlerFunc(h.markCurrentDay))
}

// The shape in apps/web/src/lib/api/types.ts. done/skipped/note are the
// caller's own; the partner's sit alongside — both can see, neither can
// change the other's (DEC-30).
type dayDTO struct {
	N              int    `json:"n"`
	Text           string `json:"text"`
	Date           string `json:"date"`
	Open           bool   `json:"open"`
	Done           bool   `json:"done"`
	Skipped        bool   `json:"skipped,omitempty"`
	PartnerDone    bool   `json:"partner_done,omitempty"`
	PartnerSkipped bool   `json:"partner_skipped,omitempty"`
	Note           string `json:"note"`
	PartnerNote    string `json:"partner_note"`
}

type challengeDTO struct {
	ID        string  `json:"id"`
	Template  string  `json:"template"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	Kind      string  `json:"kind"`
	StartedOn string  `json:"started_on"`
	EndedAt   *string `json:"ended_at"`
	TodayN    int     `json:"today_n"`
	// Days until it begins; 0 once it has.
	StartsIn int `json:"starts_in"`
	// Whether the caller may change it: still going, and theirs to touch.
	CanEdit   bool    `json:"can_edit"`
	CreatedBy *string `json:"created_by"`
	// Only once it is over, and only when there is something written.
	Reflection        string   `json:"reflection,omitempty"`
	PartnerReflection string   `json:"partner_reflection,omitempty"`
	Days              []dayDTO `json:"days"`
}

type summaryDTO struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Template    string  `json:"template"`
	Status      string  `json:"status"`
	Kind        string  `json:"kind"`
	CreatedBy   *string `json:"created_by"`
	StartedOn   string  `json:"started_on"`
	EndedAt     *string `json:"ended_at"`
	Days        int     `json:"days"`
	MyDone      int     `json:"my_done"`
	PartnerDone int     `json:"partner_done"`
}

type templateDTO struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Blurb    string `json:"blurb"`
	Days     int    `json:"days"`
	Season   string `json:"season"`
	Category string `json:"category"`
}

func toDTO(v Viewer) challengeDTO {
	out := challengeDTO{
		ID:        v.ID.String(),
		Template:  v.Template,
		Title:     v.Title,
		Status:    string(v.Status),
		Kind:      string(v.Kind),
		StartedOn: v.StartedOn.Format(time.DateOnly),
		EndedAt:   instant(v.EndedAt),
		TodayN:    v.TodayN(),
		StartsIn:  v.StartsIn(),
		CanEdit:   v.CanEdit(),
		Days:      make([]dayDTO, 0, len(v.Days)),
	}
	if v.CreatedBy != uuid.Nil {
		id := v.CreatedBy.String()
		out.CreatedBy = &id
	}
	if v.Status != StatusActive {
		out.Reflection = v.Reflections[v.Me]
		out.PartnerReflection = v.Reflections[v.Partner]
	}
	for _, d := range v.Days {
		mine, _ := d.MarkFor(v.Me)
		theirs, _ := d.MarkFor(v.Partner)
		out.Days = append(out.Days, dayDTO{
			N:              d.N,
			Text:           d.Prompt,
			Date:           v.DateOf(d.N).Format(time.DateOnly),
			Open:           v.Opened(d.N),
			Done:           mine == MarkDone,
			Skipped:        mine == MarkSkipped,
			PartnerDone:    theirs == MarkDone,
			PartnerSkipped: theirs == MarkSkipped,
			Note:           d.Notes[v.Me],
			PartnerNote:    d.Notes[v.Partner],
		})
	}
	return out
}

func instant(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

type customRequest struct {
	Title   string   `json:"title"`
	Prompts []string `json:"prompts"`
}

type startRequest struct {
	Template  string         `json:"template"`
	Custom    *customRequest `json:"custom"`
	Again     string         `json:"again"`
	Kind      string         `json:"kind"`
	StartedOn string         `json:"started_on"`
}

type markRequest struct {
	Done    *bool   `json:"done"`
	Skipped *bool   `json:"skipped"`
	Note    *string `json:"note"`
}

type editRequest struct {
	Title     *string `json:"title"`
	StartedOn *string `json:"started_on"`
	Kind      *string `json:"kind"`
}

type planRequest struct {
	Prompts []string `json:"prompts"`
}

type reflectionRequest struct {
	Text string `json:"text"`
}

func (h *Handler) templates(w http.ResponseWriter, r *http.Request) {
	if _, ok := caller(w, r); !ok {
		return
	}
	all := Templates()
	out := make([]templateDTO, 0, len(all))
	for _, t := range all {
		out = append(out, templateDTO{
			Key: t.Key, Title: t.Title, Blurb: t.Blurb, Days: len(t.Prompts),
			Season: t.Season, Category: t.Category,
		})
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) active(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Active(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]challengeDTO, 0, len(list))
	for _, v := range list {
		out = append(out, toDTO(h.withPartner(r, v)))
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

func (h *Handler) byID(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNoSuchChallenge)
		return
	}
	v, err := h.svc.Get(r.Context(), userID, id)
	h.respond(w, r, v, err, http.StatusOK)
}

func (h *Handler) past(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	list, err := h.svc.Past(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]summaryDTO, 0, len(list))
	for _, s := range list {
		var createdBy *string
		if s.CreatedBy != uuid.Nil {
			id := s.CreatedBy.String()
			createdBy = &id
		}
		out = append(out, summaryDTO{
			ID: s.ID.String(), Title: s.Title, Template: s.Template, Status: string(s.Status), Kind: string(s.Kind), CreatedBy: createdBy,
			StartedOn: s.StartedOn.Format(time.DateOnly), EndedAt: instant(s.EndedAt),
			Days: s.Days, MyDone: s.MyDone, PartnerDone: s.PartnerDone,
		})
	}
	httpx.Data(w, http.StatusOK, out)
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
	in := StartInput{Template: req.Template, Again: req.Again, Kind: req.Kind, StartedOn: req.StartedOn}
	if req.Custom != nil {
		in.Custom = &Custom{Title: req.Custom.Title, Prompts: req.Custom.Prompts}
	}
	v, err := h.svc.Start(r.Context(), userID, in)
	h.respond(w, r, v, err, http.StatusCreated)
}

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNoSuchChallenge)
		return
	}
	var req editRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v, err := h.svc.Edit(r.Context(), userID, id, EditInput{Title: req.Title, StartedOn: req.StartedOn, Kind: req.Kind})
	h.respond(w, r, v, err, http.StatusOK)
}

func (h *Handler) plan(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNoSuchChallenge)
		return
	}
	var req planRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v, err := h.svc.ReplacePlan(r.Context(), userID, id, req.Prompts)
	h.respond(w, r, v, err, http.StatusOK)
}

func (h *Handler) leave(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNoSuchChallenge)
		return
	}
	h.leaveByID(w, r, userID, id)
}

// leaveCurrent is DELETE /challenges/current, for older clients.
func (h *Handler) leaveCurrent(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Current(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.leaveByID(w, r, userID, v.ID)
}

func (h *Handler) leaveByID(w http.ResponseWriter, r *http.Request, userID, id uuid.UUID) {
	if err := h.svc.Leave(r.Context(), userID, id); err != nil {
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
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNoSuchChallenge)
		return
	}
	h.mark(w, r, userID, id)
}

// markCurrentDay is PATCH /challenges/current/days/{n}, for older clients.
func (h *Handler) markCurrentDay(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	v, err := h.svc.Current(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.mark(w, r, userID, v.ID)
}

func (h *Handler) mark(w http.ResponseWriter, r *http.Request, userID, id uuid.UUID) {
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
	v, err := h.svc.Mark(r.Context(), userID, id, n, req.Done, req.Skipped, req.Note)
	h.respond(w, r, v, err, http.StatusOK)
}

func (h *Handler) reflect(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, r, ErrNoSuchChallenge)
		return
	}
	var req reflectionRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v, err := h.svc.Reflect(r.Context(), userID, id, req.Text)
	h.respond(w, r, v, err, http.StatusOK)
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request, v Viewer, err error, status int) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, status, toDTO(h.withPartner(r, v)))
}

// withPartner labels the viewer's partner. PartnerOf only affects labeling; an
// error here just means no partner marks shown, not a failure.
func (h *Handler) withPartner(r *http.Request, v Viewer) Viewer {
	if partner, perr := h.partners.PartnerOf(r.Context(), v.Me); perr == nil {
		v.Partner = partner
	}
	return v
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}
