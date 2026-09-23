package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/falola13/amorae/apps/api/internal/modules/user"
	"github.com/falola13/amorae/apps/api/internal/platform/apperr"
	"github.com/falola13/amorae/apps/api/internal/platform/authctx"
	"github.com/falola13/amorae/apps/api/internal/platform/httpx"
)

// service is declared here, by Handler, so this file depends on the three
// use cases it actually calls rather than *Service.
type service interface {
	Register(ctx context.Context, input RegisterInput) (AuthResult, error)
	Login(ctx context.Context, input LoginInput) (AuthResult, error)
	Logout(ctx context.Context, token string) error
	ChangeEmail(ctx context.Context, userID uuid.UUID, input ChangeEmailInput) (user.User, error)
	ListSessions(ctx context.Context, userID uuid.UUID, currentToken string) ([]SessionView, error)
	SignOutOtherSessions(ctx context.Context, userID uuid.UUID, currentToken string) (int, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, currentToken string, input ChangePasswordInput) error
	DeleteAccount(ctx context.Context, userID uuid.UUID, currentPassword string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
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
	r.Handle("POST /auth/password/forgot", http.HandlerFunc(h.forgotPassword))
	r.Handle("POST /auth/password/reset", http.HandlerFunc(h.resetPassword))
	// These live here, not in the user module, because each re-checks the
	// password, which only auth knows how to do.
	r.HandleAuthed("PUT /users/me/email", http.HandlerFunc(h.changeEmail))
	r.HandleAuthed("PUT /users/me/password", http.HandlerFunc(h.changePassword))
	r.HandleAuthed("DELETE /users/me", http.HandlerFunc(h.deleteMe))
	// Your own sessions, and a way to end the rest of them.
	r.HandleAuthed("GET /sessions", http.HandlerFunc(h.listSessions))
	r.HandleAuthed("DELETE /sessions/others", http.HandlerFunc(h.signOutOthers))
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
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"display_name"`
	AgeConfirmed  bool   `json:"age_confirmed"`
	AcceptedTerms bool   `json:"accepted_terms"`
	FaithConsent  bool   `json:"faith_consent"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	result, err := h.svc.Register(r.Context(), RegisterInput{
		Email:         req.Email,
		Password:      req.Password,
		DisplayName:   req.DisplayName,
		AgeConfirmed:  req.AgeConfirmed,
		AcceptedTerms: req.AcceptedTerms,
		FaithConsent:  req.FaithConsent,
		UserAgent:     r.UserAgent(),
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
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: r.UserAgent(),
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

type changeEmailRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"current_password"`
}

func (h *Handler) changeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}

	var req changeEmailRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	u, err := h.svc.ChangeEmail(r.Context(), userID, ChangeEmailInput{Email: req.Email, CurrentPassword: req.CurrentPassword})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, user.ToDTO(u))
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// changePassword needs the caller's own token so their session survives
// while every other one ends.
func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}
	token, ok := bearerToken(r)
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}

	var req changePasswordRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	if err := h.svc.ChangePassword(r.Context(), userID, token, ChangePasswordInput{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// deleteConfirmation is the word the person types to delete their account.
// The web form applies the same rule (isDeleteConfirmation in schemas.ts).
const deleteConfirmation = "delete"

func isDeleteConfirmation(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), deleteConfirmation)
}

type deleteMeRequest struct {
	Confirm         string `json:"confirm"`
	CurrentPassword string `json:"current_password"`
}

// deleteMe needs both the typed word, which guards against a slip, and the
// password, which proves the person at the keyboard owns the account.
func (h *Handler) deleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}

	var req deleteMeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if !isDeleteConfirmation(req.Confirm) {
		httpx.Error(w, r, apperr.Validation(map[string]string{
			"confirm": `Type "` + deleteConfirmation + `" to confirm.`,
		}))
		return
	}

	if err := h.svc.DeleteAccount(r.Context(), userID, req.CurrentPassword); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), req.Email); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// sessionDTO is one row of "where you're signed in". No token hash, no raw
// user agent, no IP: only what the owner needs to recognise their own devices.
type sessionDTO struct {
	Current    bool       `json:"current"`
	Device     string     `json:"device"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}
	token, ok := bearerToken(r)
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}

	sessions, err := h.svc.ListSessions(r.Context(), userID, token)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]sessionDTO, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sessionDTO{
			Current:    s.Current,
			Device:     s.Device,
			CreatedAt:  s.CreatedAt,
			LastUsedAt: s.LastUsedAt,
			ExpiresAt:  s.ExpiresAt,
		})
	}
	httpx.Data(w, http.StatusOK, out)
}

func (h *Handler) signOutOthers(w http.ResponseWriter, r *http.Request) {
	userID, ok := authctx.UserID(r.Context())
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}
	token, ok := bearerToken(r)
	if !ok {
		httpx.Error(w, r, ErrUnauthenticated)
		return
	}

	ended, err := h.svc.SignOutOtherSessions(r.Context(), userID, token)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.Data(w, http.StatusOK, struct {
		SignedOut int `json:"signed_out"`
	}{SignedOut: ended})
}
