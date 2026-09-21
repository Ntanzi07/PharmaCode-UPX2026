package router

import (
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Handlers struct {
	Drug    *handler.DrugHandler
	Package *handler.PackageHandler
	Summary *handler.SummaryHandler
	Auth    *handler.AuthHandler
	User    *handler.UserHandler
}

// New monta as rotas. Quem pode chamar cada uma:
//
//	pública  -> GET /drugs/ean/{ean} (o app), login e a documentação
//	editor   -> leitura do painel e cadastro/edição/remoção de dados
//	reviewer -> + marcar bula como revisada
//	admin    -> + gerenciar usuários
func New(h Handlers, mw *auth.Middleware) http.Handler {
	mux := http.NewServeMux()

	editor := func(f http.HandlerFunc) http.Handler { return mw.Require(auth.RoleEditor, f) }
	reviewer := func(f http.HandlerFunc) http.Handler { return mw.Require(auth.RoleReviewer, f) }
	admin := func(f http.HandlerFunc) http.Handler { return mw.Require(auth.RoleAdmin, f) }

	// --- público: é o que o app usa, só devolve bula revisada ---
	mux.HandleFunc("GET /drugs/ean/{ean}", h.Drug.GetSummaryByEAN)

	// --- autenticação ---
	mux.HandleFunc("POST /auth/login", h.Auth.Login)
	mux.HandleFunc("POST /auth/logout", h.Auth.Logout)
	mux.Handle("GET /auth/me", editor(h.Auth.Me))
	mux.Handle("PUT /auth/password", editor(h.Auth.ChangePassword))

	// --- remédios ---
	mux.Handle("GET /drugs", editor(h.Drug.ListDrugs))
	mux.Handle("POST /drugs", editor(h.Drug.CreateDrug))
	mux.Handle("PUT /drugs/{id}", editor(h.Drug.UpdateDrug))
	mux.Handle("DELETE /drugs/{id}", editor(h.Drug.DeleteDrug))

	// --- embalagens ---
	mux.Handle("GET /packages", editor(h.Package.ListPackages))
	mux.Handle("POST /packages", editor(h.Package.CreatePackage))
	mux.Handle("GET /packages/{id}", editor(h.Package.GetPackageByID))
	mux.Handle("PUT /packages/{id}", editor(h.Package.UpdatePackage))
	mux.Handle("DELETE /packages/{id}", editor(h.Package.DeletePackage))

	// --- bulas ---
	mux.Handle("GET /summaries", editor(h.Summary.ListSummaries))
	mux.Handle("POST /summaries", editor(h.Summary.CreateSummary))
	mux.Handle("GET /summaries/{id}", editor(h.Summary.GetSummaryByID))
	mux.Handle("GET /summaries/drug/{drugID}", editor(h.Summary.GetSummaryByDrugID))
	mux.Handle("PUT /summaries/{id}", editor(h.Summary.UpdateSummary))
	mux.Handle("PATCH /summaries/{id}/review", reviewer(h.Summary.ReviewSummary))
	mux.Handle("DELETE /summaries/{id}", editor(h.Summary.DeleteSummary))

	// --- usuários do painel ---
	mux.Handle("GET /users", admin(h.User.ListUsers))
	mux.Handle("POST /users", admin(h.User.CreateUser))
	mux.Handle("PUT /users/{id}", admin(h.User.UpdateUser))
	mux.Handle("PUT /users/{id}/password", admin(h.User.SetUserPassword))

	// Swagger UI: http://localhost:<porta>/docs
	redirectToDocs := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	}
	mux.HandleFunc("GET /docs", redirectToDocs)
	mux.HandleFunc("GET /docs/{$}", redirectToDocs)
	mux.Handle("GET /docs/", httpSwagger.WrapHandler)

	return mux
}
