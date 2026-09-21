package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/testutil"
)

func seedUser(t *testing.T, store *testutil.FakeUserStore, email, password string, role auth.Role, active bool) db.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	return store.SeedUser(db.User{Email: email, Name: "Ana", PasswordHash: hash, Role: string(role), Active: active})
}

func TestRole_AtLeast(t *testing.T) {
	assert.True(t, auth.RoleAdmin.AtLeast(auth.RoleReviewer))
	assert.True(t, auth.RoleReviewer.AtLeast(auth.RoleEditor))
	assert.True(t, auth.RoleEditor.AtLeast(auth.RoleEditor))
	assert.False(t, auth.RoleEditor.AtLeast(auth.RoleReviewer))
	assert.False(t, auth.RoleReviewer.AtLeast(auth.RoleAdmin))
	assert.False(t, auth.Role("root").AtLeast(auth.RoleEditor), "papel desconhecido não passa em nada")
}

func TestHashPassword(t *testing.T) {
	_, err := auth.HashPassword("curta")
	assert.ErrorIs(t, err, auth.ErrWeakPassword)

	hash, err := auth.HashPassword("senha-bem-longa")
	require.NoError(t, err)
	assert.NotEqual(t, "senha-bem-longa", hash)
	assert.True(t, auth.CheckPassword(hash, "senha-bem-longa"))
	assert.False(t, auth.CheckPassword(hash, "outra-senha"))
}

func TestService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("credenciais certas criam sessão (email sem diferenciar maiúscula)", func(t *testing.T) {
		store := testutil.NewFakeUserStore()
		u := seedUser(t, store, "ana@farmacia.com", "senha-forte-1", auth.RoleReviewer, true)
		svc := auth.NewService(store, time.Hour)

		token, user, expires, err := svc.Login(ctx, "  ANA@Farmacia.com ", "senha-forte-1")

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, u.ID, user.ID)
		assert.Equal(t, auth.RoleReviewer, user.Role)
		assert.WithinDuration(t, time.Now().Add(time.Hour), expires, time.Minute)
		assert.Equal(t, 1, store.SessionCount(u.ID))
	})

	t.Run("senha errada, email inexistente e usuário inativo dão o mesmo erro", func(t *testing.T) {
		store := testutil.NewFakeUserStore()
		seedUser(t, store, "ana@farmacia.com", "senha-forte-1", auth.RoleEditor, true)
		seedUser(t, store, "inativo@farmacia.com", "senha-forte-1", auth.RoleEditor, false)
		svc := auth.NewService(store, time.Hour)

		for _, c := range [][2]string{
			{"ana@farmacia.com", "errada-123"},
			{"ninguem@farmacia.com", "senha-forte-1"},
			{"inativo@farmacia.com", "senha-forte-1"},
		} {
			_, _, _, err := svc.Login(ctx, c[0], c[1])
			assert.ErrorIs(t, err, auth.ErrInvalidCredentials, c[0])
		}
	})
}

func TestService_AuthenticateAndLogout(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewFakeUserStore()
	seedUser(t, store, "ana@farmacia.com", "senha-forte-1", auth.RoleEditor, true)
	svc := auth.NewService(store, time.Hour)
	token, _, _, err := svc.Login(ctx, "ana@farmacia.com", "senha-forte-1")
	require.NoError(t, err)

	u, err := svc.Authenticate(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "ana@farmacia.com", u.Email)

	_, err = svc.Authenticate(ctx, "token-inventado")
	assert.ErrorIs(t, err, auth.ErrUnauthenticated)

	// sessão vencida
	store.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	_, err = svc.Authenticate(ctx, token)
	assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	store.Now = time.Now

	require.NoError(t, svc.Logout(ctx, token))
	_, err = svc.Authenticate(ctx, token)
	assert.ErrorIs(t, err, auth.ErrUnauthenticated)
}

func TestMiddleware_Require(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewFakeUserStore()
	seedUser(t, store, "editor@x.com", "senha-forte-1", auth.RoleEditor, true)
	seedUser(t, store, "reviewer@x.com", "senha-forte-1", auth.RoleReviewer, true)
	svc := auth.NewService(store, time.Hour)
	mw := auth.NewMiddleware(svc)

	login := func(email string) string {
		token, _, _, err := svc.Login(ctx, email, "senha-forte-1")
		require.NoError(t, err)
		return token
	}

	var gotUser auth.User
	protected := mw.Require(auth.RoleReviewer, func(w http.ResponseWriter, r *http.Request) {
		gotUser, _ = auth.UserFrom(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	call := func(token string) int {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		if token != "" {
			req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
		}
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusUnauthorized, call(""), "sem cookie")
	assert.Equal(t, http.StatusUnauthorized, call("lixo"), "cookie inválido")
	assert.Equal(t, http.StatusForbidden, call(login("editor@x.com")), "papel abaixo do exigido")
	assert.Equal(t, http.StatusNoContent, call(login("reviewer@x.com")))
	assert.Equal(t, "reviewer@x.com", gotUser.Email, "o handler recebe o usuário no contexto")
}

func TestSessionCookie_Flags(t *testing.T) {
	c := auth.SessionCookie("tok", time.Now().Add(time.Hour), true)
	assert.True(t, c.HttpOnly)
	assert.True(t, c.Secure)
	assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
	assert.Equal(t, "/", c.Path)
}

func TestLoginLimiter(t *testing.T) {
	l := auth.NewLoginLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		require.True(t, l.Allowed("k"))
		l.Fail("k")
	}
	assert.False(t, l.Allowed("k"), "bloqueia depois de 3 falhas")
	assert.True(t, l.Allowed("outra-chave"), "outras chaves não são afetadas")

	l.Reset("k")
	assert.True(t, l.Allowed("k"), "login certo zera o contador")
}
