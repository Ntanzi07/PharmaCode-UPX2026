package auth

import (
	"errors"
	"log"
	"net/http"
	"time"
)

// CookieName é o cookie da sessão do painel.
const CookieName = "pharmacode_session"

type Middleware struct {
	svc *Service
}

func NewMiddleware(svc *Service) *Middleware {
	return &Middleware{svc: svc}
}

// Require só deixa passar requisições com sessão válida de um usuário com
// papel igual ou acima de min. Sem sessão: 401. Papel insuficiente: 403.
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

// SessionCookie monta o cookie da sessão. HttpOnly: o JavaScript não lê.
// SameSite=Strict: o navegador não manda o cookie em requisições vindas de
// outros sites (protege contra CSRF). Secure: só em HTTPS (ligar em produção).
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

// ClearSessionCookie apaga o cookie no navegador.
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
