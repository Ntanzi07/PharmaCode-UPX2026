package router

import (
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
)

func New(drugH *handler.DrugHandler, packageH *handler.PackageHandler) http.Handler {
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

	return mux
}
