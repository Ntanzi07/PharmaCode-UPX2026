package router

import (
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func New(drugH *handler.DrugHandler, packageH *handler.PackageHandler, summaryH *handler.SummaryHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /drugs", drugH.ListDrugs)
	mux.HandleFunc("POST /drugs", drugH.CreateDrug)
	mux.HandleFunc("GET /drugs/ean/{ean}", drugH.GetSummaryByEAN)
	mux.HandleFunc("PUT /drugs/{id}", drugH.UpdateDrug)
	mux.HandleFunc("DELETE /drugs/{id}", drugH.DeleteDrug)

	mux.HandleFunc("GET /packages", packageH.ListPackages)
	mux.HandleFunc("POST /packages", packageH.CreatePackage)
	mux.HandleFunc("GET /packages/{id}", packageH.GetPackageByID)
	mux.HandleFunc("PUT /packages/{id}", packageH.UpdatePackage)
	mux.HandleFunc("DELETE /packages/{id}", packageH.DeletePackage)

	mux.HandleFunc("GET /summaries", summaryH.ListSummaries)
	mux.HandleFunc("POST /summaries", summaryH.CreateSummary)
	mux.HandleFunc("GET /summaries/{id}", summaryH.GetSummaryByID)
	mux.HandleFunc("GET /summaries/drug/{drugID}", summaryH.GetSummaryByDrugID)
	mux.HandleFunc("PUT /summaries/{id}", summaryH.UpdateSummary)
	mux.HandleFunc("PATCH /summaries/{id}/review", summaryH.ReviewSummary)
	mux.HandleFunc("DELETE /summaries/{id}", summaryH.DeleteSummary)

	// Swagger UI: http://localhost:<porta>/swagger/index.html
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	return mux
}
