package auth

import (
	"net/http"
	"strings"

	"pulse-backend/internal/platform/apperr"
	"pulse-backend/internal/platform/authctx"
	"pulse-backend/internal/platform/httpx"
)

type Handler struct {
	service *Service
	tokens  *TokenIssuer
}

func NewHandler(service *Service, tokens *TokenIssuer) *Handler {
	return &Handler{service: service, tokens: tokens}
}

// Register mounts the auth routes. limit wraps the unauthenticated endpoints
// with the per-IP rate limiter.
func (h *Handler) Register(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/auth/signup", limit(h.signup))
	mux.HandleFunc("POST /api/auth/login", limit(h.login))
	mux.HandleFunc("POST /api/auth/refresh", limit(h.refresh))
	mux.HandleFunc("POST /api/auth/logout", h.logout)

	mux.HandleFunc("GET /api/me", h.RequireAuth(h.me))
	mux.HandleFunc("POST /api/me/password", h.RequireAuth(h.changePassword))
	mux.HandleFunc("DELETE /api/me", h.RequireAuth(h.deleteAccount))
}

// RequireAuth rejects requests without a valid "Authorization: Bearer <access token>".
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			httpx.WriteError(w, r, apperr.Unauthorized("missing bearer token"))
			return
		}
		userID, err := h.tokens.Parse(token)
		if err != nil {
			httpx.WriteError(w, r, apperr.Unauthorized("invalid or expired access token"))
			return
		}
		next(w, r.WithContext(authctx.WithUserID(r.Context(), userID)))
	}
}

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) {
	var in SignupInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	session, err := h.service.Signup(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, session)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in LoginInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	session, err := h.service.Login(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var in RefreshInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	session, err := h.service.Refresh(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var in RefreshInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.service.Logout(r.Context(), in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Me(r.Context(), authctx.UserID(r.Context()))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	var in ChangePasswordInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	session, err := h.service.ChangePassword(r.Context(), authctx.UserID(r.Context()), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var in DeleteAccountInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.service.DeleteAccount(r.Context(), authctx.UserID(r.Context()), in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}
