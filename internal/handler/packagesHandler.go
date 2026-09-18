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

type updatePackageRequest struct {
	DrugID      int64  `json:"drug_id"`
	Ean         string `json:"ean"`
	Description string `json:"description"`
}

// CreatePackage godoc
// @Summary      Cadastra uma embalagem (EAN) de um remédio
// @Tags         packages
// @Accept       json
// @Produce      json
// @Param        body  body      createPackageRequest  true  "Dados da embalagem"
// @Success      201   {object}  idResponse
// @Failure      400   {string}  string  "json inválido ou campo obrigatório faltando"
// @Failure      404   {string}  string  "drug not found for this registration_number"
// @Failure      409   {string}  string  "ean already exists"
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

// ListPackages godoc
// @Summary      Lista embalagens (paginado)
// @Tags         packages
// @Produce      json
// @Param        limit   query     int     false  "Itens por página (padrão 20, máx 100)"
// @Param        offset  query     int     false  "Quantos itens pular (padrão 0)"
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
// @Summary      Busca uma embalagem pelo ID
// @Tags         packages
// @Produce      json
// @Param        id   path      int     true  "ID da embalagem"
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
// @Summary      Atualiza uma embalagem
// @Tags         packages
// @Accept       json
// @Param        id   path      int     true  "ID da embalagem"
// @Param        body  body      updatePackageRequest  true  "Dados da embalagem"
// @Success      204
// @Failure      400  {string}  string  "id ou json inválido"
// @Failure      404  {string}  string  "package not found"
// @Failure      409  {string}  string  "ean already exists"
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
	if req.Ean == "" {
		http.Error(w, "ean is required", http.StatusBadRequest)
		return
	}
	if req.Description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}

	err = h.service.Update(r.Context(), id, service.UpdatePackageInput(req))
	if err != nil {
		if errors.Is(err, service.ErrPackageNotFound) {
			http.Error(w, "package not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrDuplicateEAN) {
			http.Error(w, "ean already exists", http.StatusConflict)
			return
		}
		log.Printf("failed to update package %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeletePackage godoc
// @Summary      Remove uma embalagem
// @Tags         packages
// @Param        id   path      int     true  "ID da embalagem"
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
