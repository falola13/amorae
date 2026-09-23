package prayers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// service is the handler's view of the service: only what HTTP calls.
type service interface {
	Current(ctx context.Context, userID uuid.UUID) (Record, CoupleContext, error)
	History(ctx context.Context, userID uuid.UUID) ([]Record, CoupleContext, error)
	Week(ctx context.Context, userID, weekID uuid.UUID) (Record, CoupleContext, error)
	SavePoints(ctx context.Context, userID uuid.UUID, points []Point) (Record, CoupleContext, error)
	Publish(ctx context.Context, userID uuid.UUID) (Record, CoupleContext, error)
	SetCompletion(ctx context.Context, userID, pointID uuid.UUID, done bool) (Record, CoupleContext, error)
	SetReflection(ctx context.Context, userID, weekID uuid.UUID, body string) (Record, CoupleContext, error)
}

type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.HandleAuthed("GET /prayers/current", http.HandlerFunc(h.current))
	r.HandleAuthed("GET /prayers/history", http.HandlerFunc(h.history))
	r.HandleAuthed("GET /prayers/weeks/{id}", http.HandlerFunc(h.week))
	r.HandleAuthed("PUT /prayers/current/points", http.HandlerFunc(h.savePoints))
	r.HandleAuthed("POST /prayers/current/publish", http.HandlerFunc(h.publish))
	r.HandleAuthed("POST /prayers/points/{id}/complete", http.HandlerFunc(h.complete))
	r.HandleAuthed("DELETE /prayers/points/{id}/complete", http.HandlerFunc(h.uncomplete))
	r.HandleAuthed("PATCH /prayers/weeks/{id}/reflection", http.HandlerFunc(h.reflection))
}

// Every write answers with the whole week, the same shape a read returns, so
// a client never has to re-fetch to see what it just changed.

func (h *Handler) current(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	rec, cc, err := h.svc.Current(r.Context(), userID)
	h.respond(w, r, rec, cc, userID, err, http.StatusOK)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	records, cc, err := h.svc.History(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, http.StatusOK, ToDTOs(records, userID, cc.Partner(userID)))
}

func (h *Handler) week(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	weekID, ok := pathID(w, r, "id", "week")
	if !ok {
		return
	}
	rec, cc, err := h.svc.Week(r.Context(), userID, weekID)
	h.respond(w, r, rec, cc, userID, err, http.StatusOK)
}

type savePointsRequest struct {
	Points []struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Text      string `json:"text"`
		Scripture string `json:"scripture"`
		Verse     string `json:"verse"`
		// Accepted and ignored. The client sends back the position it read,
		// but the order of the array is what decides — ValidatePoints
		// numbers them from it. Declaring the field keeps the decoder, which
		// refuses unknown ones, from rejecting an otherwise good request.
		Position int `json:"position"`
	} `json:"points"`
}

func (h *Handler) savePoints(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	var req savePointsRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	points := make([]Point, 0, len(req.Points))
	for _, p := range req.Points {
		// An id the client cannot supply is a new point. A malformed one is
		// treated the same way rather than refused: the position in the array
		// is what identifies a point to the person writing it.
		id, err := uuid.Parse(p.ID)
		if err != nil {
			id = uuid.UUID{}
		}
		points = append(points, Point{
			ID:        id,
			Title:     p.Title,
			Body:      p.Text,
			Scripture: p.Scripture,
			Verse:     p.Verse,
		})
	}

	rec, cc, err := h.svc.SavePoints(r.Context(), userID, points)
	h.respond(w, r, rec, cc, userID, err, http.StatusOK)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	rec, cc, err := h.svc.Publish(r.Context(), userID)
	h.respond(w, r, rec, cc, userID, err, http.StatusOK)
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	h.setCompletion(w, r, true)
}

func (h *Handler) uncomplete(w http.ResponseWriter, r *http.Request) {
	h.setCompletion(w, r, false)
}

func (h *Handler) setCompletion(w http.ResponseWriter, r *http.Request, done bool) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	pointID, ok := pathID(w, r, "id", "prayer")
	if !ok {
		return
	}
	rec, cc, err := h.svc.SetCompletion(r.Context(), userID, pointID, done)
	h.respond(w, r, rec, cc, userID, err, http.StatusOK)
}

type reflectionRequest struct {
	Reflection string `json:"reflection"`
}

func (h *Handler) reflection(w http.ResponseWriter, r *http.Request) {
	userID, ok := caller(w, r)
	if !ok {
		return
	}
	weekID, ok := pathID(w, r, "id", "week")
	if !ok {
		return
	}
	var req reflectionRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	rec, cc, err := h.svc.SetReflection(r.Context(), userID, weekID, req.Reflection)
	h.respond(w, r, rec, cc, userID, err, http.StatusOK)
}

func (h *Handler) respond(
	w http.ResponseWriter, r *http.Request,
	rec Record, cc CoupleContext, userID uuid.UUID, err error, status int,
) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Data(w, status, ToDTO(rec, userID, cc.Partner(userID)))
}

func caller(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated("unauthenticated", "Authentication required."))
		return uuid.UUID{}, false
	}
	return userID, true
}

// pathID reads a uuid from the path. `thing` names it for the person reading
// the message, who should never see the words "uuid" or "path parameter".
func pathID(w http.ResponseWriter, r *http.Request, name, thing string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		httpx.Error(w, r, apperr.NotFound("not_found", "That "+thing+" isn’t here."))
		return uuid.UUID{}, false
	}
	return id, true
}
