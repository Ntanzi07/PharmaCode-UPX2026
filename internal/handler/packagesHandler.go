package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
)

type PackageHandler struct {
	service *service.PackageService
}

func NewPackageHandler(s *service.PackageService) *PackageHandler {
	return &PackageHandler{service: s}
}

type createPackageRequest struct {
	RegistrationNumber       string   `json:"registration_number" example:"1234567890123"`
	Eans                     []string `json:"eans" example:"7891058001155,7891058017392"`
	Description              string   `json:"description" example:"400 mg, caixa com 20 cápsulas"`
	PresentationRegistration string   `json:"presentation_registration,omitempty" example:"1234567890014"`
}

type updatePackageRequest struct {
	DrugID                   int64    `json:"drug_id" example:"1"`
	Eans                     []string `json:"eans" example:"7891058001155,7891058017392"`
	Description              string   `json:"description" example:"400 mg, caixa com 20 cápsulas"`
	PresentationRegistration string   `json:"presentation_registration,omitempty" example:"1234567890014"`
}

// normalizeEANs trims spaces, drops empty and duplicate values, and validates the format.
// Accepts 8 (EAN-8) to 13 (EAN-13) digits.
func normalizeEANs(raw []string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	eans := make([]string, 0, len(raw))
	for _, e := range raw {
		e = strings.TrimSpace(e)
		if e == "" || seen[e] {
			continue
		}
		if !isDigits(e) || len(e) < 8 || len(e) > 13 {
			return nil, fmt.Errorf("invalid ean %q: must have 8 to 13 digits", e)
		}
		seen[e] = true
		eans = append(eans, e)
	}
	if len(eans) == 0 {
		return nil, errors.New("at least one ean is required")
	}
	return eans, nil
}

// validatePresentationRegistration: optional, but when present it must have 13 digits.
func validatePresentationRegistration(s string) error {
	if s != "" && (len(s) != 13 || !isDigits(s)) {
		return errors.New("presentation_registration must have exactly 13 digits")
	}
	return nil
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// packageErrorStatus maps the service's write errors to HTTP status codes.
func packageErrorStatus(err error) (int, string, bool) {
	switch {
	case errors.Is(err, service.ErrPackageNotFound):
		return http.StatusNotFound, "package not found", true
	case errors.Is(err, service.ErrDrugNotFound):
		return http.StatusNotFound, "drug not found", true
	case errors.Is(err, service.ErrDuplicateEAN):
		return http.StatusConflict, "ean already registered in another package", true
	case errors.Is(err, service.ErrDuplicatePresentation):
		return http.StatusConflict, "presentation_registration already exists", true
	}
	return 0, "", false
}

// CreatePackage godoc
// @Summary      Create a package (presentation) of a drug
// @Description  A package can have several EANs (e.g. an old and a new code on the shelf at the same time).
// @Tags         packages
// @Accept       json
// @Produce      json
// @Param        body  body      createPackageRequest  true  "Package data"
// @Success      201   {object}  idResponse
// @Failure      400   {string}  string  "invalid json or missing required field"
// @Failure      404   {string}  string  "drug not found"
// @Failure      409   {string}  string  "ean or presentation_registration already registered"
// @Failure      500  {string}  string  "internal server error"
// @Router       /packages [post]
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
	eans, err := normalizeEANs(req.Eans)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Eans = eans
	if req.Description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}
	req.PresentationRegistration = strings.TrimSpace(req.PresentationRegistration)
	if err := validatePresentationRegistration(req.PresentationRegistration); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.service.Create(r.Context(), service.CreatePackageInput(req))
	if err != nil {
		if status, msg, ok := packageErrorStatus(err); ok {
			http.Error(w, msg, status)
			return
		}
		log.Printf("failed to create package %v: %v", req.Eans, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]int64{"id": id}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// ListPackages godoc
// @Summary      List packages (paginated)
// @Tags         packages
// @Produce      json
// @Param        limit   query     int     false  "Items per page (default 20, max 100)"
// @Param        offset  query     int     false  "How many items to skip (default 0)"
// @Success      200  {object}  listResponse{data=[]db.ListPackagesRow}
// @Failure      400  {string}  string  "invalid limit / invalid offset"
// @Failure      500  {string}  string  "internal server error"
// @Router       /packages [get]
func (h *PackageHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
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

	packages, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		log.Printf("failed to list packages: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if packages == nil {
		packages = []db.ListPackagesRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(listResponse{
		Data:   packages,
		Limit:  limit,
		Offset: offset,
	}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// GetPackageByID godoc
// @Summary      Get a package by ID
// @Tags         packages
// @Produce      json
// @Param        id   path      int     true  "Package ID"
// @Success      200  {object}  db.GetPackageByIdRow
// @Failure      400  {string}  string  "invalid id"
// @Failure      404  {string}  string  "package not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /packages/{id} [get]
func (h *PackageHandler) GetPackageByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	pkg, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			http.Error(w, "package not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to get package %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pkg); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// UpdatePackage godoc
// @Summary      Update a package
// @Tags         packages
// @Accept       json
// @Param        id   path      int     true  "Package ID"
// @Param        body  body      updatePackageRequest  true  "Package data"
// @Success      204
// @Failure      400  {string}  string  "invalid id or json"
// @Failure      404  {string}  string  "package not found / drug not found"
// @Failure      409  {string}  string  "ean or presentation_registration already registered"
// @Failure      500  {string}  string  "internal server error"
// @Router       /packages/{id} [put]
func (h *PackageHandler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req updatePackageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.DrugID == 0 {
		http.Error(w, "drug_id is required", http.StatusBadRequest)
		return
	}
	eans, err := normalizeEANs(req.Eans)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Eans = eans
	if req.Description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}
	req.PresentationRegistration = strings.TrimSpace(req.PresentationRegistration)
	if err := validatePresentationRegistration(req.PresentationRegistration); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.service.Update(r.Context(), id, service.UpdatePackageInput(req))
	if err != nil {
		if status, msg, ok := packageErrorStatus(err); ok {
			http.Error(w, msg, status)
			return
		}
		log.Printf("failed to update package %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeletePackage godoc
// @Summary      Delete a package
// @Tags         packages
// @Param        id   path      int     true  "Package ID"
// @Success      204
// @Failure      400  {string}  string  "invalid id"
// @Failure      404  {string}  string  "package not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /packages/{id} [delete]
func (h *PackageHandler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			http.Error(w, "package not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to delete package %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
