package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
	"github.com/jackc/pgx/v5"
)

type DrugHandler struct {
	service *service.DrugService
}

func NewDrugHandler(s *service.DrugService) *DrugHandler {
	return &DrugHandler{service: s}
}

type createDrugRequest struct {
	RegistrationNumber string `json:"registration_number"`
	BrandName          string `json:"brand_name"`
	ActiveIngredient   string `json:"active_ingredient"`
	Manufacturer       string `json:"manufacturer"`
}

func drugRequestVerification(req createDrugRequest) (error error) {
	if req.RegistrationNumber == "" {
		return errors.New("registration number is required")
	}
	if req.ActiveIngredient == "" {
		return errors.New("active ingredient is required")
	}
	if req.Manufacturer == "" {
		return errors.New("manufacturer is required")
	}
	return nil
}

func (h *DrugHandler) GetSummaryByEAN(w http.ResponseWriter, r *http.Request) {

	ean := r.PathValue("ean")
	if ean == "" {
		http.Error(w, "ean is required", http.StatusBadRequest)
		return
	}

	summary, err := h.service.GetSummaryByEAN(r.Context(), ean)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "ean not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("failed to get summary by ean %s: %v", ean, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(summary)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *DrugHandler) CreateDrug(w http.ResponseWriter, r *http.Request) {
	var req createDrugRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	err := drugRequestVerification(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	id, err := h.service.Create(r.Context(), service.CreateDrugInput(req))
	if err != nil {
		if errors.Is(err, service.ErrDuplicateRegistration) {
			http.Error(w, "registration_number already exists", http.StatusConflict)
			return
		}
		log.Printf("failed to create drug %s: %v", req.RegistrationNumber, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]int64{"id": id}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *DrugHandler) UpdateDrug(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req createDrugRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	err = drugRequestVerification(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.service.Update(r.Context(), id, service.CreateDrugInput(req))
	if err != nil {
		if errors.Is(err, service.ErrDuplicateRegistration) {
			http.Error(w, "registration_number already exists", http.StatusConflict)
			return
		}
		log.Printf("failed to create drug %s: %v", req.RegistrationNumber, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

// name: DeleteDrug :execrows

// name: ListDrugs :many
