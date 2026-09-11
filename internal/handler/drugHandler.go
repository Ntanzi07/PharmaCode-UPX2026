package handler

import (
	"PharmaCode_UPX2026/internal/db"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type DrugHandler struct {
	queries *db.Queries
}

func NewDrugHandler(q *db.Queries) *DrugHandler {
	return &DrugHandler{queries: q}
}

func (h *DrugHandler) GetByEAN(w http.ResponseWriter, r *http.Request) {

	ean := r.PathValue("ean")
	if ean == "" {
		http.Error(w, "ean is required", http.StatusBadRequest)
		return
	}

	drug, err := h.queries.GetDrugByEAN(r.Context(), ean)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "ean not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("failed to get drug by ean %s: %v", ean, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(drug)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
