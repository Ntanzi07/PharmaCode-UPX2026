package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

// MaxInteractionEANs caps how many barcodes one check can carry.
const MaxInteractionEANs = 10

// Severity levels a rule can have, from the worst to the mildest.
const (
	SeverityHigh     = "grave"
	SeverityModerate = "moderada"
	SeverityLow      = "leve"
)

// ValidSeverity reports whether the severity is one of the three known levels.
func ValidSeverity(severity string) bool {
	switch severity {
	case SeverityHigh, SeverityModerate, SeverityLow:
		return true
	}
	return false
}

// InteractionDrug is one of the scanned drugs.
type InteractionDrug struct {
	Ean               string   `json:"ean" example:"7896015592752"`
	DrugID            int64    `json:"drug_id" example:"1"`
	BrandName         string   `json:"brand_name" example:"Advil 12h"`
	Manufacturer      string   `json:"manufacturer" example:"Pfizer"`
	ActiveIngredients []string `json:"active_ingredients" example:"ibuprofeno"`
}

// DrugRef names a drug inside a finding.
type DrugRef struct {
	DrugID    int64  `json:"drug_id" example:"1"`
	Ean       string `json:"ean" example:"7896015592752"`
	BrandName string `json:"brand_name" example:"Advil 12h"`
}

// InteractionFinding is one pair of active ingredients, coming from two
// different boxes, that has a registered rule.
type InteractionFinding struct {
	IngredientA    string  `json:"ingredient_a" example:"ibuprofeno"`
	IngredientB    string  `json:"ingredient_b" example:"varfarina"`
	Severity       string  `json:"severity" enums:"grave,moderada,leve" example:"grave"`
	Description    string  `json:"description"`
	Recommendation string  `json:"recommendation"`
	SourceURL      string  `json:"source_url"`
	DrugA          DrugRef `json:"drug_a"`
	DrugB          DrugRef `json:"drug_b"`
}

// InteractionReport is the answer of POST /interactions.
type InteractionReport struct {
	Drugs []InteractionDrug `json:"drugs"`
	// NotFound: barcodes that are not registered. They are reported instead of
	// failing the whole request, so one unknown box doesn't break the app.
	NotFound []string             `json:"not_found"`
	Findings []InteractionFinding `json:"findings"`
	// Severity of the worst finding ("grave", "moderada", "leve"), or empty
	// when nothing was found: it is what the app colours the screen by.
	WorstSeverity string `json:"worst_severity" enums:"grave,moderada,leve," example:"grave"`
}

// CheckInteractions compares every pair of active ingredients coming from
// different boxes against the registered rules. It does not look at the
// leaflet, so a drug whose leaflet is not reviewed is still checked.
//
// Nothing found does NOT mean the combination is safe: it means there is no
// rule registered for it. The answer says so in the panel and in the docs.
func (s *DrugService) CheckInteractions(ctx context.Context, eans []string) (InteractionReport, error) {
	report := InteractionReport{Drugs: []InteractionDrug{}, NotFound: []string{}, Findings: []InteractionFinding{}}

	rows, err := s.queries.ListDrugsByEANs(ctx, eans)
	if err != nil {
		return report, err
	}

	found := make(map[string]bool, len(rows))
	byID := make(map[int64]DrugRef, len(rows))
	for _, row := range rows {
		found[row.Ean] = true
		report.Drugs = append(report.Drugs, InteractionDrug{
			Ean:               row.Ean,
			DrugID:            row.DrugID,
			BrandName:         row.BrandName.String,
			Manufacturer:      row.Manufacturer,
			ActiveIngredients: row.ActiveIngredients,
		})
		byID[row.DrugID] = DrugRef{DrugID: row.DrugID, Ean: row.Ean, BrandName: row.BrandName.String}
	}
	for _, ean := range eans {
		if !found[ean] {
			report.NotFound = append(report.NotFound, ean)
		}
	}

	findings, err := s.queries.FindInteractionsBetweenEANs(ctx, eans)
	if err != nil {
		return report, err
	}
	for _, row := range findings {
		report.Findings = append(report.Findings, InteractionFinding{
			IngredientA:    row.IngredientA,
			IngredientB:    row.IngredientB,
			Severity:       row.Severity,
			Description:    row.Description,
			Recommendation: row.Recommendation.String,
			SourceURL:      row.SourceUrl,
			DrugA:          byID[row.DrugAID],
			DrugB:          byID[row.DrugBID],
		})
		report.WorstSeverity = worstSeverity(report.WorstSeverity, row.Severity)
	}
	return report, nil
}

// severityRank orders the levels so the worst one can be picked.
var severityRank = map[string]int{SeverityLow: 1, SeverityModerate: 2, SeverityHigh: 3}

func worstSeverity(current, candidate string) string {
	if severityRank[candidate] > severityRank[current] {
		return candidate
	}
	return current
}

// ---------------------------------------------------------------------------
// Rules management (panel and spreadsheet import)
// ---------------------------------------------------------------------------

// InteractionRuleInput is one rule, with the ingredients written by name. The
// names are matched (or created) exactly like in the drug form.
type InteractionRuleInput struct {
	IngredientA    string
	IngredientB    string
	Severity       string
	Description    string
	Recommendation string
	SourceURL      string
}

// SaveInteractionRule creates or updates the rule of a pair. Returns true when
// the rule did not exist yet.
func (s *DrugService) SaveInteractionRule(ctx context.Context, in InteractionRuleInput) (bool, error) {
	if !ValidSeverity(in.Severity) {
		return false, ErrInvalidSeverity
	}
	if strings.TrimSpace(in.Description) == "" {
		return false, ErrInvalidInteractionText
	}
	if strings.TrimSpace(in.SourceURL) == "" {
		return false, ErrInvalidInteractionSource
	}

	var created bool
	err := s.tx.Run(ctx, func(q DrugQuerier) error {
		aID, bID, err := ingredientPair(ctx, q, in.IngredientA, in.IngredientB)
		if err != nil {
			return err
		}
		row, err := q.UpsertIngredientInteraction(ctx, db.UpsertIngredientInteractionParams{
			IngredientAID:  aID,
			IngredientBID:  bID,
			Severity:       in.Severity,
			Description:    strings.TrimSpace(in.Description),
			Recommendation: optionalText(strings.TrimSpace(in.Recommendation)),
			SourceUrl:      strings.TrimSpace(in.SourceURL),
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.ConstraintName == "chk_severity" {
				return ErrInvalidSeverity
			}
			return err
		}
		created = row.Created
		return nil
	})
	return created, err
}

// ingredientPair turns the two names into ids, creating the ingredients when
// they are new. Refuses a pair of the same ingredient: a rule needs two.
func ingredientPair(ctx context.Context, q DrugQuerier, nameA, nameB string) (int64, int64, error) {
	if strings.TrimSpace(nameA) == "" || strings.TrimSpace(nameB) == "" {
		return 0, 0, ErrInvalidIngredientPair
	}
	aID, err := q.UpsertIngredient(ctx, strings.TrimSpace(nameA))
	if err != nil {
		return 0, 0, err
	}
	bID, err := q.UpsertIngredient(ctx, strings.TrimSpace(nameB))
	if err != nil {
		return 0, 0, err
	}
	if aID == bID {
		return 0, 0, ErrInvalidIngredientPair
	}
	return aID, bID, nil
}

// DeleteInteractionRule removes the rule of a pair, in any order of the ids.
func (s *DrugService) DeleteInteractionRule(ctx context.Context, ingredientAID, ingredientBID int64) error {
	rows, err := s.queries.DeleteIngredientInteraction(ctx, db.DeleteIngredientInteractionParams{
		IngredientAID: ingredientAID,
		IngredientBID: ingredientBID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInteractionRuleNotFound
	}
	return nil
}

// ListInteractionRules lists the registered rules, paginated.
func (s *DrugService) ListInteractionRules(ctx context.Context, limit, offset int32) ([]db.ListIngredientInteractionsRow, error) {
	return s.queries.ListIngredientInteractions(ctx, db.ListIngredientInteractionsParams{Limit: limit, Offset: offset})
}

// SearchIngredients suggests ingredient names for the rule form, with how many
// drugs use each one. Seeing that "varfarina sódica" has 7 drugs and
// "varfarina" has 1 is what stops a rule being written against the name no
// drug actually uses.
func (s *DrugService) SearchIngredients(ctx context.Context, query string, limit int32) ([]db.SearchIngredientsRow, error) {
	return s.queries.SearchIngredients(ctx, db.SearchIngredientsParams{
		Q:     strings.TrimSpace(query),
		Limit: limit,
	})
}

// CountInteractionRules is shown in the panel: how much of the knowledge base
// is filled in.
func (s *DrugService) CountInteractionRules(ctx context.Context) (int64, error) {
	return s.queries.CountIngredientInteractions(ctx)
}
