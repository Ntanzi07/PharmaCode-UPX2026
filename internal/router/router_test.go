package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/router"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/testutil"
)

const pwd = "senha-forte-1"

// newServer builds the whole API with fakes and one user per role.
func newServer(t *testing.T) (*httptest.Server, *testutil.FakeSummaryQuerier) {
	t.Helper()
	store := testutil.NewFakeUserStore()
	hash, err := auth.HashPassword(pwd)
	require.NoError(t, err)
	for _, role := range []string{"editor", "reviewer", "admin"} {
		store.SeedUser(db.User{Name: role, Email: role + "@x.com", PasswordHash: hash, Role: role, Active: true})
	}

	drugs := testutil.NewFakeDrugQuerier()
	drugs.SeedDrug(db.ListDrugsRow{ID: 1, RegistrationNumber: "1", ActiveIngredient: "Paracetamol", Manufacturer: "X"})
	drugs.SeedEAN("7891234567890", 1)
	drugs.SeedReviewedSummary(1, "Dor e febre")
	summaries := testutil.NewFakeSummaryQuerier()
	summaries.Seed(db.GetSummaryByIDRow{ID: 1, DrugID: 1})

	authSvc := auth.NewService(store, time.Hour)
	userSvc := service.NewUserService(store)
	h := router.New(router.Handlers{
		Drug:    handler.NewDrugHandler(service.NewDrugService(drugs)),
		Package: handler.NewPackageHandler(service.NewPackageService(testutil.NewFakePackageQuerier())),
		Summary: handler.NewSummaryHandler(service.NewSummaryService(summaries)),
		Auth:    handler.NewAuthHandler(authSvc, userSvc, auth.NewLoginLimiter(5, time.Minute), false),
		User:    handler.NewUserHandler(userSvc),
	}, auth.NewMiddleware(authSvc))

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, summaries
}

// login returns the session cookie of the user with that role.
func login(t *testing.T, srv *httptest.Server, role string) *http.Cookie {
	t.Helper()
	res, err := http.Post(srv.URL+"/auth/login", "application/json",
		strings.NewReader(`{"email":"`+role+`@x.com","password":"`+pwd+`"}`))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	for _, c := range res.Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	t.Fatal("login sem cookie de sessão")
	return nil
}

func do(t *testing.T, srv *httptest.Server, method, path string, cookie *http.Cookie, body string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, srv.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	res.Body.Close()
	return res.StatusCode
}

func TestPermissoes(t *testing.T) {
	srv, summaries := newServer(t)
	editor := login(t, srv, "editor")
	reviewer := login(t, srv, "reviewer")
	admin := login(t, srv, "admin")

	drugBody := `{"registration_number":"R%s","active_ingredient":"x","manufacturer":"y"}`

	t.Run("a consulta do app é pública", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(t, srv, "GET", "/drugs/ean/7891234567890", nil, ""))
	})

	t.Run("sem login, nada além da consulta pública", func(t *testing.T) {
		for _, r := range [][2]string{
			{"GET", "/drugs"}, {"POST", "/drugs"}, {"DELETE", "/drugs/1"},
			{"GET", "/summaries"}, {"PATCH", "/summaries/1/review"},
			{"GET", "/users"}, {"GET", "/auth/me"},
		} {
			assert.Equal(t, http.StatusUnauthorized, do(t, srv, r[0], r[1], nil, "{}"), r[0]+" "+r[1])
		}
	})

	t.Run("editor lê e escreve dados, mas não revisa nem gerencia usuários", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(t, srv, "GET", "/drugs", editor, ""))
		assert.Equal(t, http.StatusCreated, do(t, srv, "POST", "/drugs", editor, strings.Replace(drugBody, "%s", "1", 1)))
		assert.Equal(t, http.StatusForbidden, do(t, srv, "PATCH", "/summaries/1/review", editor, ""))
		assert.Equal(t, http.StatusForbidden, do(t, srv, "GET", "/users", editor, ""))
		assert.Zero(t, summaries.ReviewCalls)
	})

	t.Run("reviewer revisa, e a revisão fica no nome dele", func(t *testing.T) {
		assert.Equal(t, http.StatusNoContent, do(t, srv, "PATCH", "/summaries/1/review", reviewer, ""))
		assert.Equal(t, "reviewer", summaries.LastReviewParams.ReviewedBy.String)
		assert.Equal(t, http.StatusForbidden, do(t, srv, "POST", "/users", reviewer, "{}"))
	})

	t.Run("admin gerencia usuários", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(t, srv, "GET", "/users", admin, ""))
		assert.Equal(t, http.StatusCreated, do(t, srv, "POST", "/users", admin,
			`{"name":"Nova","email":"nova@x.com","password":"senha-forte-1","role":"editor"}`))
	})

	t.Run("logout invalida a sessão", func(t *testing.T) {
		c := login(t, srv, "editor")
		assert.Equal(t, http.StatusNoContent, do(t, srv, "POST", "/auth/logout", c, ""))
		assert.Equal(t, http.StatusUnauthorized, do(t, srv, "GET", "/auth/me", c, ""))
	})
}

func TestLogin(t *testing.T) {
	srv, _ := newServer(t)

	t.Run("cookie de sessão é HttpOnly e SameSite=Strict, e /auth/me devolve o usuário", func(t *testing.T) {
		c := login(t, srv, "reviewer")
		assert.True(t, c.HttpOnly)
		assert.Equal(t, http.SameSiteStrictMode, c.SameSite)

		req, _ := http.NewRequest("GET", srv.URL+"/auth/me", nil)
		req.AddCookie(c)
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		var me map[string]any
		require.NoError(t, json.NewDecoder(res.Body).Decode(&me))
		assert.Equal(t, "reviewer@x.com", me["email"])
		assert.Equal(t, "reviewer", me["role"])
		assert.NotContains(t, me, "password_hash")
	})

	t.Run("senha errada 401 e, depois de 5 erros, 429", func(t *testing.T) {
		body := `{"email":"editor@x.com","password":"errada-123"}`
		for i := 0; i < 5; i++ {
			assert.Equal(t, http.StatusUnauthorized, do(t, srv, "POST", "/auth/login", nil, body))
		}
		assert.Equal(t, http.StatusTooManyRequests, do(t, srv, "POST", "/auth/login", nil,
			`{"email":"editor@x.com","password":"`+pwd+`"}`), "bloqueado mesmo com a senha certa")
	})
}
