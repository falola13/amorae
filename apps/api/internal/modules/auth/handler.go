package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// service is declared here, by Handler, so this file depends on the three
// use cases it actually calls rather than *Service.
type service interface {
	Register(ctx context.Context, input RegisterInput) (AuthResult, error)
	Login(ctx context.Context, input LoginInput) (AuthResult, error)
	Logout(ctx context.Context, token string) error
}

// Handler is transport only: decode, call the service, map to a DTO,
// respond. There's no dto.go in this package — AuthResultDTO is the only
// shape this module sends and lives next to the handler that builds it.
type Handler struct {
	svc service
}

func NewHandler(svc service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes follows the same shape every module uses. Logout is
// deliberately *not* HandleAuthed: holding a token is all the authority
// needed to destroy it, and putting it behind RequireAuth would turn
// "log out an already-expired session" into a 401 instead of the idempotent
// 204 the contract promises.
func (h *Handler) RegisterRoutes(r *httpx.Router) {
	r.Handle("POST /auth/register", http.HandlerFunc(h.register))
	r.Handle("POST /auth/login", http.HandlerFunc(h.login))
	r.Handle("POST /auth/logout", http.HandlerFunc(h.logout))
}

// authResultDTO matches AuthResult{"token","expires_at","user"} in the HTTP
// contract exactly; User is user.DTO so the password hash can't leak here
// either.
type authResultDTO struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      user.DTO  `json:"user"`
}

func toAuthResultDTO(r AuthResult) authResultDTO {
	return authResultDTO{
		Token:     r.Token,
		ExpiresAt: r.ExpiresAt,
		User:      user.ToDTO(r.User),
	}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	result, err := h.svc.Register(r.Context(), RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusCreated, toAuthResultDTO(result))
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	result, err := h.svc.Login(r.Context(), LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, toAuthResultDTO(result))
}

// logout reads the bearer token itself rather than going through
// RequireAuth (see RegisterRoutes): it needs the raw token to delete the
// matching session, and a token that is already expired or unknown should
// still get a 204, because the end state the caller wants already holds.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}

	if err := h.svc.Logout(r.Context(), token); err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.NoContent(w)
}
