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

type InteractionHandler struct {
	service *service.DrugService
}

func NewInteractionHandler(s *service.DrugService) *InteractionHandler {
	return &InteractionHandler{service: s}
}

type interactionRequest struct {
	Eans []string `json:"eans" example:"7896015592752,7891058001155"`
}

// CheckInteractions godoc
// @Summary      Check interactions between the scanned boxes
// @Description  Public route used by the app: send the barcodes the person scanned and it compares every pair of active ingredients coming from different boxes against the registered rules. Unknown barcodes come back in "not_found" instead of failing the request. An empty "findings" means no rule is registered for those pairs, not that the combination is safe.
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Param        body  body      interactionRequest  true  "Scanned barcodes (2 to 10)"
// @Success      200   {object}  service.InteractionReport
// @Failure      400   {string}  string  "invalid json / fewer than 2 or more than 10 barcodes / malformed barcode"
// @Failure      500   {string}  string  "internal server error"
// @Router       /interactions [post]
func (h *InteractionHandler) CheckInteractions(w http.ResponseWriter, r *http.Request) {
	var req interactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	eans, err := normalizeEANs(req.Eans)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(eans) < 2 {
		http.Error(w, "send at least 2 different barcodes", http.StatusBadRequest)
		return
	}
	if len(eans) > service.MaxInteractionEANs {
		http.Error(w, "send at most 10 barcodes", http.StatusBadRequest)
		return
	}

	report, err := h.service.CheckInteractions(r.Context(), eans)
	if err != nil {
		log.Printf("failed to check interactions: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

type interactionRuleRequest struct {
	IngredientA    string `json:"ingredient_a" example:"ibuprofeno"`
	IngredientB    string `json:"ingredient_b" example:"varfarina"`
	Severity       string `json:"severity" enums:"grave,moderada,leve" example:"grave"`
	Description    string `json:"description" example:"Juntos aumentam o risco de sangramento."`
	Recommendation string `json:"recommendation" example:"Não use junto sem orientação médica."`
	SourceURL      string `json:"source_url" example:"https://consultas.anvisa.gov.br/#/bulario/"`
}

// ruleErrorStatus maps the rule errors to HTTP status codes.
func ruleErrorStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, service.ErrInteractionRuleNotFound):
		return http.StatusNotFound, true
	case errors.Is(err, service.ErrInvalidSeverity),
		errors.Is(err, service.ErrInvalidIngredientPair),
		errors.Is(err, service.ErrInvalidInteractionText),
		errors.Is(err, service.ErrInvalidInteractionSource):
		return http.StatusBadRequest, true
	}
	return 0, false
}

// ListInteractionRules godoc
// @Summary      List the registered interaction rules
// @Tags         interactions
// @Produce      json
// @Param        limit   query     int  false  "Items per page (default 20, max 100)"
// @Param        offset  query     int  false  "How many items to skip (default 0)"
// @Success      200  {object}  listResponse{data=[]db.ListIngredientInteractionsRow}
// @Failure      401  {string}  string  "authentication required"
// @Router       /interaction-rules [get]
func (h *InteractionHandler) ListInteractionRules(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}

	rules, err := h.service.ListInteractionRules(r.Context(), limit, offset)
	if err != nil {
		log.Printf("failed to list interaction rules: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if rules == nil {
		rules = []db.ListIngredientInteractionsRow{}
	}
	writeJSON(w, http.StatusOK, listResponse{Data: rules, Limit: limit, Offset: offset})
}

// SaveInteractionRule godoc
// @Summary      Create or update the rule of a pair (pharmacist)
// @Description  The pair is identified by the two ingredient names, in any order. An ingredient that does not exist yet is created. Sending the same pair again updates the rule.
// @Tags         interactions
// @Accept       json
// @Produce      json
// @Param        body  body      interactionRuleRequest  true  "Interaction rule"
// @Success      200   {string}  string  "rule updated"
// @Success      201   {string}  string  "rule created"
// @Failure      400   {string}  string  "invalid data"
// @Failure      403   {string}  string  "insufficient permissions"
// @Router       /interaction-rules [put]
func (h *InteractionHandler) SaveInteractionRule(w http.ResponseWriter, r *http.Request) {
	var req interactionRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	created, err := h.service.SaveInteractionRule(r.Context(), service.InteractionRuleInput{
		IngredientA:    req.IngredientA,
		IngredientB:    req.IngredientB,
		Severity:       req.Severity,
		Description:    req.Description,
		Recommendation: req.Recommendation,
		SourceURL:      req.SourceURL,
	})
	if err != nil {
		if status, ok := ruleErrorStatus(err); ok {
			http.Error(w, err.Error(), status)
			return
		}
		log.Printf("failed to save interaction rule: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if created {
		w.WriteHeader(http.StatusCreated)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteInteractionRule godoc
// @Summary      Delete the rule of a pair (pharmacist)
// @Tags         interactions
// @Param        ingredientA  path  int  true  "Id of one of the ingredients"
// @Param        ingredientB  path  int  true  "Id of the other ingredient"
// @Success      204
// @Failure      400  {string}  string  "invalid id"
// @Failure      404  {string}  string  "interaction rule not found"
// @Router       /interaction-rules/{ingredientA}/{ingredientB} [delete]
func (h *InteractionHandler) DeleteInteractionRule(w http.ResponseWriter, r *http.Request) {
	a, err := strconv.ParseInt(r.PathValue("ingredientA"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	b, err := strconv.ParseInt(r.PathValue("ingredientB"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteInteractionRule(r.Context(), a, b); err != nil {
		if status, ok := ruleErrorStatus(err); ok {
			http.Error(w, err.Error(), status)
			return
		}
		log.Printf("failed to delete interaction rule: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
