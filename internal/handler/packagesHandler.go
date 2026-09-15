package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
)

type PackageHandler struct {
	service *service.PackageService
}

func NewPackageHandler(s *service.PackageService) *PackageHandler {
	return &PackageHandler{service: s}
}

type createPackageRequest struct {
	RegistrationNumber string `json:"registration_number"`
	Ean                string `json:"ean"`
	Description        string `json:"description"`
}

func (h *PackageHandler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var req createPackageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.RegistrationNumber == "" {
		http.Error(w, "registration_number is required", http.StatusBadRequest)
		return
	}
	if req.Ean == "" {
		http.Error(w, "ean is required", http.StatusBadRequest)
		return
	}
	if req.Description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}

	id, err := h.service.Create(r.Context(), service.CreatePackageInput(req))
	if err != nil {
		if errors.Is(err, service.ErrDrugNotFound) {
			http.Error(w, "drug not found for this registration_number", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrDuplicateEAN) {
			http.Error(w, "ean already exists", http.StatusConflict)
			return
		}
		log.Printf("failed to create package %s: %v", req.Ean, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]int64{"id": id}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
