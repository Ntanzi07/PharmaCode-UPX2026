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

type SummaryHandler struct {
	service *service.SummaryService
}

func NewSummaryHandler(s *service.SummaryService) *SummaryHandler {
	return &SummaryHandler{service: s}
}

type createSummaryRequest struct {
	DrugID            int64  `json:"drug_id"`
	WhatIsItFor       string `json:"what_is_it_for"`
	Posology          string `json:"posology"`
	MissedDose        string `json:"missed_dose"`
	Warnings          string `json:"warnings"`
	AdverseEffects    string `json:"adverse_effects"`
	DrugInteractions  string `json:"drug_interactions"`
	Contraindications string `json:"contraindications"`
	SideEffects       string `json:"side_effects"`
	WhenToSeekHelp    string `json:"when_to_seek_help"`
	MechanismOfAction string `json:"mechanism_of_action"`
	Storage           string `json:"storage"`
	SourceURL         string `json:"source_url"`
	// Versão da bula oficial que foi resumida
	LeafletExpedient   string `json:"leaflet_expedient" example:"0123456/24-5"`
	LeafletPublishedAt string `json:"leaflet_published_at" example:"2024-05-31" format:"date"`
}

type updateSummaryRequest struct {
	WhatIsItFor       string `json:"what_is_it_for"`
	Posology          string `json:"posology"`
	MissedDose        string `json:"missed_dose"`
	Warnings          string `json:"warnings"`
	AdverseEffects    string `json:"adverse_effects"`
	DrugInteractions  string `json:"drug_interactions"`
	Contraindications string `json:"contraindications"`
	SideEffects       string `json:"side_effects"`
	WhenToSeekHelp    string `json:"when_to_seek_help"`
	MechanismOfAction string `json:"mechanism_of_action"`
	Storage           string `json:"storage"`
	SourceURL         string `json:"source_url"`
	// Versão da bula oficial que foi resumida
	LeafletExpedient   string `json:"leaflet_expedient" example:"0123456/24-5"`
	LeafletPublishedAt string `json:"leaflet_published_at" example:"2024-05-31" format:"date"`
}

type reviewSummaryRequest struct {
	ReviewedBy string `json:"reviewed_by"`
}

// summaryRequiredFields validates the columns the table declares as NOT NULL.
func summaryRequiredFields(whatIsItFor, posology, sourceURL string) error {
	if whatIsItFor == "" {
		return errors.New("what_is_it_for is required")
	}
	if posology == "" {
		return errors.New("posology is required")
	}
	if sourceURL == "" {
		return errors.New("source_url is required")
	}
	return nil
}

// CreateSummary godoc
// @Summary      Cadastra a bula simplificada de um remédio
// @Tags         summaries
// @Accept       json
// @Produce      json
// @Param        body  body      createSummaryRequest  true  "Resumo da bula"
// @Success      201   {object}  idResponse
// @Failure      400   {string}  string  "json inválido ou campo obrigatório faltando"
// @Failure      404   {string}  string  "drug not found for this drug_id"
// @Failure      409   {string}  string  "this drug already has a summary"
// @Failure      500  {string}  string  "internal server error"
// @Router       /summaries [post]
func (h *SummaryHandler) CreateSummary(w http.ResponseWriter, r *http.Request) {
	var req createSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.DrugID == 0 {
		http.Error(w, "drug_id is required", http.StatusBadRequest)
		return
	}
	if err := summaryRequiredFields(req.WhatIsItFor, req.Posology, req.SourceURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.service.Create(r.Context(), service.CreateSummaryInput(req))
	if err != nil {
		if errors.Is(err, service.ErrInvalidLeafletDate) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrDrugNotFound) {
			http.Error(w, "drug not found for this drug_id", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrSummaryAlreadyExists) {
			http.Error(w, "this drug already has a summary", http.StatusConflict)
			return
		}
		log.Printf("failed to create summary for drug %d: %v", req.DrugID, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]int64{"id": id}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// ListSummaries godoc
// @Summary      Lista bulas simplificadas (paginado)
// @Tags         summaries
// @Produce      json
// @Param        limit   query     int     false  "Itens por página (padrão 20, máx 100)"
// @Param        offset  query     int     false  "Quantos itens pular (padrão 0)"
// @Success      200  {object}  listResponse{data=[]db.ListSummariesRow}
// @Failure      400  {string}  string  "invalid limit / invalid offset"
// @Failure      500  {string}  string  "internal server error"
// @Router       /summaries [get]
func (h *SummaryHandler) ListSummaries(w http.ResponseWriter, r *http.Request) {
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

	summaries, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		log.Printf("failed to list summaries: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if summaries == nil {
		summaries = []db.ListSummariesRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(listResponse{
		Data:   summaries,
		Limit:  limit,
		Offset: offset,
	}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// GetSummaryByID godoc
// @Summary      Busca uma bula simplificada pelo ID
// @Tags         summaries
// @Produce      json
// @Param        id   path      int     true  "ID do resumo"
// @Success      200  {object}  db.GetSummaryByIDRow
// @Failure      400  {string}  string  "invalid id"
// @Failure      404  {string}  string  "summary not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /summaries/{id} [get]
func (h *SummaryHandler) GetSummaryByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	summary, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrSummaryNotFound) {
			http.Error(w, "summary not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to get summary %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// GetSummaryByDrugID godoc
// @Summary      Busca a bula simplificada de um remédio
// @Tags         summaries
// @Produce      json
// @Param        drugID  path      int     true  "ID do remédio"
// @Success      200  {object}  db.GetSummaryByDrugIDRow
// @Failure      400  {string}  string  "invalid drug id"
// @Failure      404  {string}  string  "summary not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /summaries/drug/{drugID} [get]
func (h *SummaryHandler) GetSummaryByDrugID(w http.ResponseWriter, r *http.Request) {
	drugID, err := strconv.ParseInt(r.PathValue("drugID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid drug id", http.StatusBadRequest)
		return
	}

	summary, err := h.service.GetByDrugID(r.Context(), drugID)
	if err != nil {
		if errors.Is(err, service.ErrSummaryNotFound) {
			http.Error(w, "summary not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to get summary of drug %d: %v", drugID, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

// UpdateSummary godoc
// @Summary      Atualiza uma bula simplificada
// @Tags         summaries
// @Accept       json
// @Param        id   path      int     true  "ID do resumo"
// @Param        body  body      updateSummaryRequest  true  "Resumo da bula"
// @Success      204
// @Failure      400  {string}  string  "id ou json inválido / campo obrigatório faltando"
// @Failure      404  {string}  string  "summary not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /summaries/{id} [put]
func (h *SummaryHandler) UpdateSummary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req updateSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if err := summaryRequiredFields(req.WhatIsItFor, req.Posology, req.SourceURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.Update(r.Context(), id, service.UpdateSummaryInput(req)); err != nil {
		if errors.Is(err, service.ErrInvalidLeafletDate) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrSummaryNotFound) {
			http.Error(w, "summary not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to update summary %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReviewSummary godoc
// @Summary      Marca uma bula simplificada como revisada
// @Tags         summaries
// @Accept       json
// @Param        id   path      int     true  "ID do resumo"
// @Param        body  body      reviewSummaryRequest  true  "Quem revisou"
// @Success      204
// @Failure      400  {string}  string  "id ou json inválido / reviewed_by is required"
// @Failure      404  {string}  string  "summary not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /summaries/{id}/review [patch]
func (h *SummaryHandler) ReviewSummary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req reviewSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.ReviewedBy == "" {
		http.Error(w, "reviewed_by is required", http.StatusBadRequest)
		return
	}

	if err := h.service.Review(r.Context(), id, req.ReviewedBy); err != nil {
		if errors.Is(err, service.ErrSummaryNotFound) {
			http.Error(w, "summary not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to review summary %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSummary godoc
// @Summary      Remove uma bula simplificada
// @Tags         summaries
// @Param        id   path      int     true  "ID do resumo"
// @Success      204
// @Failure      400  {string}  string  "invalid id"
// @Failure      404  {string}  string  "summary not found"
// @Failure      500  {string}  string  "internal server error"
// @Router       /summaries/{id} [delete]
func (h *SummaryHandler) DeleteSummary(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrSummaryNotFound) {
			http.Error(w, "summary not found", http.StatusNotFound)
			return
		}
		log.Printf("failed to delete summary %d: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
