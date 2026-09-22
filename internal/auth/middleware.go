package auth

import (
	"errors"
	"log"
	"net/http"
	"time"
)

// CookieName is the admin panel session cookie.
const CookieName = "pharmacode_session"

type Middleware struct {
	svc *Service
}

func NewMiddleware(svc *Service) *Middleware {
	return &Middleware{svc: svc}
}

// Require only lets through requests with a valid session from a user whose
// role is min or higher. No session: 401. Insufficient role: 403.
func (m *Middleware) Require(min Role, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			http.Error(w, ErrUnauthenticated.Error(), http.StatusUnauthorized)
			return
		}
		user, err := m.svc.Authenticate(r.Context(), cookie.Value)
		if errors.Is(err, ErrUnauthenticated) {
			http.Error(w, ErrUnauthenticated.Error(), http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Printf("failed to authenticate session: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if !user.Role.AtLeast(min) {
			http.Error(w, "insufficient permissions", http.StatusForbidden)
			return
		}
		next(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

// SessionCookie builds the session cookie. HttpOnly: JavaScript can't read it.
// SameSite=Strict: the browser doesn't send the cookie on requests coming from
// other sites (CSRF protection). Secure: HTTPS only (turn on in production).
func SessionCookie(token string, expires time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// ClearSessionCookie deletes the cookie in the browser.
func ClearSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}
