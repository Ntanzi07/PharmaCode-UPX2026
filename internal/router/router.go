package router

import (
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
)

func New(drugH *handler.DrugHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /drugs/ean/{ean}", drugH.GetByEAN)
	mux.HandleFunc("GET /summary/ean/{ean}", drugH.GetSummaryByEAN)

	return mux
}
