package router

import (
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
)

func New(drugH *handler.DrugHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /drugs", drugH.ListDrugs)
	mux.HandleFunc("POST /drugs", drugH.CreateDrug)
	mux.HandleFunc("GET /drugs/ean/{ean}", drugH.GetSummaryByEAN)
	mux.HandleFunc("PUT /drugs/{id}", drugH.UpdateDrug)
	mux.HandleFunc("DELETE /drugs/{id}", drugH.DeleteDrug)

	return mux
}
