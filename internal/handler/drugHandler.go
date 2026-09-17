package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
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

type listResponse struct {
	Data   any   `json:"data"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
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

	summary, err := h.service.GetSummaryByEANService(r.Context(), ean)
	if errors.Is(err, service.ErrDrugNotFound) {
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
		return
	}

	id, err := h.service.CreateDrugService(r.Context(), service.CreateDrugInput(req))
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

	err = h.service.UpdateDrugService(r.Context(), id, service.CreateDrugInput(req))
	if err != nil {
		if errors.Is(err, service.ErrDrugNotFound) {
			http.Error(w, "drug not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrDuplicateRegistration) {
			http.Error(w, "registration_number already exists", http.StatusConflict)
			return
		}
		log.Printf("failed to update drug %s: %v", req.RegistrationNumber, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DrugHandler) DeleteDrug(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.DeleteDrugService(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrDrugNotFound) {
			http.Error(w, "drug not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to delete drug %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DrugHandler) ListDrugs(w http.ResponseWriter, r *http.Request) {
	limit := int32(20)
	if s := r.URL.Query().Get("limit"); s != "" {
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil || v <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if v > 100 {
			v = 100
		}
		limit = int32(v)
	}

	offset := int32(0)
	if s := r.URL.Query().Get("offset"); s != "" {
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil || v < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = int32(v)
	}

	drugs, err := h.service.ListDrugsService(r.Context(), limit, offset)
	if err != nil {
		log.Printf("failed to get the drug list: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if drugs == nil {
		drugs = []db.ListDrugsRow{}
	}

	resp := listResponse{
		Data:   drugs,
		Limit:  limit,
		Offset: offset,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
