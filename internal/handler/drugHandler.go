package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	RegistrationNumber string `json:"registration_number" example:"192900007"`
	BrandName          string `json:"brand_name" example:"Advil 12h"`
	// One entry per active ingredient: a combination drug such as Neosaldina
	// has three. Names are matched ignoring case and accents.
	ActiveIngredients []string `json:"active_ingredients" example:"dipirona sódica,cafeína"`
	Manufacturer      string   `json:"manufacturer" example:"Pfizer"`
}

// normalizeIngredients trims the names, drops the empty ones and removes
// repetitions that differ only in case, accents or spacing.
func normalizeIngredients(raw []string) []string {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, name := range raw {
		name = strings.Join(strings.Fields(name), " ")
		if name == "" {
			continue
		}
		key := strings.ToLower(removeAccents(name))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
	}
	return out
}

var ingredientAccents = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

func removeAccents(s string) string {
	var b strings.Builder
	for _, r := range s {
		if plain, ok := ingredientAccents[r]; ok {
			r = plain
		}
		b.WriteRune(r)
	}
	return b.String()
}

// idResponse documents the {"id": 123} body returned by create endpoints.
type idResponse struct {
	ID int64 `json:"id" example:"1"`
}

type listResponse struct {
	Data   any   `json:"data"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func drugRequestVerification(req *createDrugRequest) error {
	if req.RegistrationNumber == "" {
		return errors.New("registration number is required")
	}
	req.ActiveIngredients = normalizeIngredients(req.ActiveIngredients)
	if len(req.ActiveIngredients) == 0 {
		return errors.New("at least one active ingredient is required")
	}
	if req.Manufacturer == "" {
		return errors.New("manufacturer is required")
	}
	return nil
}

// GetSummaryByEAN godoc
// @Summary      Get the simplified leaflet by EAN
// @Description  Returns the drug data and the leaflet summary from the barcode on the box
// @Tags         drugs
// @Produce      json
// @Param        ean  path      string  true  "Package EAN barcode"
// @Success      200  {object}  db.GetSummaryByEANRow
// @Failure      400  {string}  string  "ean is required"
// @Failure      404  {string}  string  "ean not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /drugs/ean/{ean} [get]
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

// CreateDrug godoc
// @Summary      Create a drug
// @Tags         drugs
// @Accept       json
// @Produce      json
// @Param        body  body      createDrugRequest  true  "Drug data"
// @Success      201   {object}  idResponse
// @Failure      400   {string}  string  "invalid json or missing required field"
// @Failure      409   {string}  string  "registration_number already exists"
// @Failure      500  {string}  string  "internal server error"
// @Router       /drugs [post]
func (h *DrugHandler) CreateDrug(w http.ResponseWriter, r *http.Request) {
	var req createDrugRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	err := drugRequestVerification(&req)
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

// UpdateDrug godoc
// @Summary      Update a drug
// @Tags         drugs
// @Accept       json
// @Param        id   path      int     true  "Drug ID"
// @Param        body  body      createDrugRequest  true  "Drug data"
// @Success      204
// @Failure      400  {string}  string  "invalid id or json"
// @Failure      404  {string}  string  "drug not found"
// @Failure      409  {string}  string  "registration_number already exists"
// @Failure      500  {string}  string  "internal server error"
// @Router       /drugs/{id} [put]
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
	err = drugRequestVerification(&req)
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

// DeleteDrug godoc
// @Summary      Delete a drug
// @Tags         drugs
// @Param        id   path      int     true  "Drug ID"
// @Success      204
// @Failure      400  {string}  string  "invalid id"
// @Failure      404  {string}  string  "drug not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /drugs/{id} [delete]
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

// GetDrug godoc
// @Summary      Get one drug by id
// @Tags         drugs
// @Produce      json
// @Param        id   path      int  true  "Drug id"
// @Success      200  {object}  db.GetDrugByIDRow
// @Failure      400  {string}  string  "invalid id"
// @Failure      404  {string}  string  "drug not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /drugs/{id} [get]
func (h *DrugHandler) GetDrug(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	drug, err := h.service.GetDrug(r.Context(), id)
	if errors.Is(err, service.ErrDrugNotFound) {
		http.Error(w, "drug not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("failed to get drug %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, drug)
}

// ListDrugs godoc
// @Summary      List drugs (paginated)
// @Tags         drugs
// @Produce      json
// @Description  With "q" it searches the brand name, the company, the registration number and the active ingredients, ignoring case and accents. Without it, it lists everything.
// @Param        q       query     string  false  "Search term"
// @Param        limit   query     int     false  "Items per page (default 20, max 100)"
// @Param        offset  query     int     false  "How many items to skip (default 0)"
// @Success      200  {object}  listResponse{data=[]db.ListDrugsRow}
// @Failure      400  {string}  string  "invalid limit / invalid offset"
// @Failure      500  {string}  string  "internal server error"
// @Router       /drugs [get]
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

	drugs, err := h.service.ListDrugsService(r.Context(), limit, offset, r.URL.Query().Get("q"))
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
