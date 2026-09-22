package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
)

type AuthHandler struct {
	auth         *auth.Service
	users        *service.UserService
	limiter      *auth.LoginLimiter
	secureCookie bool
}

func NewAuthHandler(a *auth.Service, u *service.UserService, l *auth.LoginLimiter, secureCookie bool) *AuthHandler {
	return &AuthHandler{auth: a, users: u, limiter: l, secureCookie: secureCookie}
}

type loginRequest struct {
	Email    string `json:"email" example:"ana@farmacia.com"`
	Password string `json:"password" example:"uma-senha-forte"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// clientIP returns the caller's IP. Behind the frontend's nginx it comes in X-Real-IP.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Login godoc
// @Summary      Log in to the admin panel
// @Description  Checks email and password and sets the session cookie (HttpOnly). In Swagger, after logging in the other routes work in this same browser.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Credentials"
// @Success      200   {object}  auth.User
// @Failure      400   {string}  string  "invalid json"
// @Failure      401   {string}  string  "invalid email or password"
// @Failure      429   {string}  string  "too many attempts"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	key := clientIP(r) + "|" + auth.NormalizeEmail(req.Email)
	if !h.limiter.Allowed(key) {
		http.Error(w, "too many attempts, try again in a few minutes", http.StatusTooManyRequests)
		return
	}

	token, user, expires, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		h.limiter.Fail(key)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Printf("failed to login: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	h.limiter.Reset(key)

	http.SetCookie(w, auth.SessionCookie(token, expires, h.secureCookie))
	writeJSON(w, http.StatusOK, user)
}

// Logout godoc
// @Summary      Log out of the admin panel
// @Tags         auth
// @Success      204
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := h.auth.Logout(r.Context(), c.Value); err != nil {
			log.Printf("failed to logout: %v", err)
		}
	}
	http.SetCookie(w, auth.ClearSessionCookie(h.secureCookie))
	w.WriteHeader(http.StatusNoContent)
}

// Me godoc
// @Summary      Current user
// @Tags         auth
// @Produce      json
// @Success      200  {object}  auth.User
// @Failure      401  {string}  string  "authentication required"
// @Router       /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	writeJSON(w, http.StatusOK, user)
}

// ChangePassword godoc
// @Summary      Change your own password
// @Description  Requires the current password. The user's other sessions are ended; the current one stays.
// @Tags         auth
// @Accept       json
// @Param        body  body  changePasswordRequest  true  "Current and new password (8 to 72 characters)"
// @Success      204
// @Failure      400  {string}  string  "weak new password / invalid json"
// @Failure      401  {string}  string  "authentication required"
// @Failure      403  {string}  string  "current password is wrong"
// @Router       /auth/password [put]
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	cookie, _ := r.Cookie(auth.CookieName)
	token := ""
	if cookie != nil {
		token = cookie.Value
	}

	// Same protection as login: repeatedly wrong current passwords get blocked.
	key := "pwd|" + clientIP(r) + "|" + user.Email
	if !h.limiter.Allowed(key) {
		http.Error(w, "too many attempts, try again in a few minutes", http.StatusTooManyRequests)
		return
	}

	err := h.users.ChangeOwnPassword(r.Context(), user.ID, token, req.CurrentPassword, req.NewPassword)
	switch {
	case err == nil:
		h.limiter.Reset(key)
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, service.ErrWrongPassword):
		h.limiter.Fail(key)
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, auth.ErrWeakPassword):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		log.Printf("failed to change password of user %d: %v", user.ID, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
