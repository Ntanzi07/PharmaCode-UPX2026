package testutil

import (
	"context"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type FakeDrugQuerier struct {
	mu     sync.Mutex
	nextID int64
	drugs  map[int64]db.ListDrugsRow

	eans              map[string]int64
	reviewedSummaries map[int64]string

	// Active ingredients, as in the real tables: one row per name (by
	// normalized name) and the links drug -> ingredient.
	ingredients map[string]int64 // normalized name -> id
	names       map[int64]string // id -> name as written
	links       map[int64][]int64
	rules       map[[2]int64]fakeRule

	LastIngredientNames []string

	CreateErr error
	UpdateErr error
	DeleteErr error
	ListErr   error
	GetErr    error

	LastCreateParams db.CreateDrugParams
	LastUpdateParams db.UpdateDrugParams

	CreateCalls int
	UpdateCalls int
	DeleteCalls int
}

func NewFakeDrugQuerier() *FakeDrugQuerier {
	return &FakeDrugQuerier{
		nextID:            1,
		drugs:             make(map[int64]db.ListDrugsRow),
		eans:              make(map[string]int64),
		reviewedSummaries: make(map[int64]string),
		ingredients:       make(map[string]int64),
		names:             make(map[int64]string),
		links:             make(map[int64][]int64),
		rules:             make(map[[2]int64]fakeRule),
	}
}

func (f *FakeDrugQuerier) SeedDrug(row db.ListDrugsRow) db.ListDrugsRow {
	f.mu.Lock()
	defer f.mu.Unlock()

	if row.ID == 0 {
		row.ID = f.nextID
		f.nextID++
	} else if row.ID >= f.nextID {
		f.nextID = row.ID + 1
	}
	f.drugs[row.ID] = row
	return row
}

func (f *FakeDrugQuerier) SeedEAN(ean string, drugID int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eans[ean] = drugID
}

func (f *FakeDrugQuerier) SeedReviewedSummary(drugID int64, whatIsItFor string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviewedSummaries[drugID] = whatIsItFor
}

func (f *FakeDrugQuerier) CreateDrug(ctx context.Context, arg db.CreateDrugParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.CreateCalls++
	f.LastCreateParams = arg

	if f.CreateErr != nil {
		return 0, f.CreateErr
	}
	for _, d := range f.drugs {
		if d.RegistrationNumber == arg.RegistrationNumber {
			return 0, PgError(CodeUniqueViolation)
		}
	}

	id := f.nextID
	f.nextID++
	f.drugs[id] = db.ListDrugsRow{
		ID:                 id,
		RegistrationNumber: arg.RegistrationNumber,
		BrandName:          arg.BrandName,
		Manufacturer:       arg.Manufacturer,
	}
	return id, nil
}

func (f *FakeDrugQuerier) UpdateDrug(ctx context.Context, arg db.UpdateDrugParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.UpdateCalls++
	f.LastUpdateParams = arg

	if f.UpdateErr != nil {
		return 0, f.UpdateErr
	}

	row, ok := f.drugs[arg.ID]
	if !ok {
		return 0, nil
	}
	for id, d := range f.drugs {
		if id != arg.ID && d.RegistrationNumber == arg.RegistrationNumber {
			return 0, PgError(CodeUniqueViolation)
		}
	}

	row.RegistrationNumber = arg.RegistrationNumber
	row.BrandName = arg.BrandName
	row.Manufacturer = arg.Manufacturer
	f.drugs[arg.ID] = row

	return 1, nil
}

func (f *FakeDrugQuerier) DeleteDrug(ctx context.Context, id int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.DeleteCalls++

	if f.DeleteErr != nil {
		return 0, f.DeleteErr
	}
	if _, ok := f.drugs[id]; !ok {
		return 0, nil
	}
	delete(f.drugs, id)
	return 1, nil
}

func (f *FakeDrugQuerier) ListDrugs(ctx context.Context, arg db.ListDrugsParams) ([]db.ListDrugsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ListErr != nil {
		return nil, f.ListErr
	}

	ids := make([]int64, 0, len(f.drugs))
	for id := range f.drugs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var out []db.ListDrugsRow
	for _, id := range ids {
		row := f.drugs[id]
		row.ActiveIngredients = f.ingredientNames(id)
		if matchesQuery(row, arg.Q) {
			out = append(out, row)
		}
	}

	offset := int(arg.Offset)
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit := int(arg.Limit); limit >= 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// matchesQuery mirrors the WHERE of ListDrugs: brand name, company,
// registration and ingredients, ignoring case and accents.
func matchesQuery(row db.ListDrugsRow, query string) bool {
	query = normalizeIngredient(query)
	if query == "" {
		return true
	}
	text := normalizeIngredient(strings.Join(append([]string{
		row.BrandName.String, row.Manufacturer, row.RegistrationNumber,
	}, row.ActiveIngredients...), " "))
	return strings.Contains(text, query)
}

func (f *FakeDrugQuerier) SearchIngredients(ctx context.Context, arg db.SearchIngredientsParams) ([]db.SearchIngredientsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	query := normalizeIngredient(arg.Q)
	var out []db.SearchIngredientsRow
	for id, name := range f.names {
		if query != "" && !strings.Contains(normalizeIngredient(name), query) {
			continue
		}
		var drugs int64
		for _, linked := range f.links {
			for _, linkedID := range linked {
				if linkedID == id {
					drugs++
				}
			}
		}
		out = append(out, db.SearchIngredientsRow{ID: id, Name: name, Drugs: drugs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if limit := int(arg.Limit); limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeDrugQuerier) GetDrugByID(ctx context.Context, id int64) (db.GetDrugByIDRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetDrugByIDRow{}, f.GetErr
	}
	d, ok := f.drugs[id]
	if !ok {
		return db.GetDrugByIDRow{}, pgx.ErrNoRows
	}
	return db.GetDrugByIDRow{
		ID:                 d.ID,
		RegistrationNumber: d.RegistrationNumber,
		BrandName:          d.BrandName,
		ActiveIngredients:  f.ingredientNames(d.ID),
		Manufacturer:       d.Manufacturer,
		UpdatedAt:          d.UpdatedAt,
	}, nil
}

func (f *FakeDrugQuerier) GetDrugByEAN(ctx context.Context, ean string) (db.GetDrugByEANRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetDrugByEANRow{}, f.GetErr
	}

	drugID, ok := f.eans[ean]
	if !ok {
		return db.GetDrugByEANRow{}, pgx.ErrNoRows
	}
	d, ok := f.drugs[drugID]
	if !ok {
		return db.GetDrugByEANRow{}, pgx.ErrNoRows
	}

	return db.GetDrugByEANRow{
		Ean:                ean,
		ID:                 d.ID,
		RegistrationNumber: d.RegistrationNumber,
		BrandName:          d.BrandName,
		ActiveIngredients:  f.ingredientNames(d.ID),
		Manufacturer:       d.Manufacturer,
	}, nil
}

func (f *FakeDrugQuerier) GetSummaryByEAN(ctx context.Context, ean string) (db.GetSummaryByEANRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetSummaryByEANRow{}, f.GetErr
	}

	drugID, ok := f.eans[ean]
	if !ok {
		return db.GetSummaryByEANRow{}, pgx.ErrNoRows
	}
	d, ok := f.drugs[drugID]
	if !ok {
		return db.GetSummaryByEANRow{}, pgx.ErrNoRows
	}

	row := db.GetSummaryByEANRow{
		Ean:                ean,
		Description:        "Caixa com 20 comprimidos",
		RegistrationNumber: d.RegistrationNumber,
		BrandName:          d.BrandName,
		ActiveIngredients:  f.ingredientNames(d.ID),
		Manufacturer:       d.Manufacturer,
	}
	if what, ok := f.reviewedSummaries[drugID]; ok {
		row.WhatIsItFor = pgtype.Text{String: what, Valid: true}
		row.Posology = pgtype.Text{String: "1 comprimido a cada 8 horas", Valid: true}
		row.SourceUrl = pgtype.Text{String: "https://consultas.anvisa.gov.br/bula/1", Valid: true}
	}
	return row, nil
}

// ---- active ingredients ----

// normalizeIngredient mimics the normalize_ingredient_name function of the
// database: lowercase, no accents, single spaces.
func normalizeIngredient(name string) string {
	var b strings.Builder
	lastSpace := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if plain, ok := accents[r]; ok {
			r = plain
		}
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteRune(' ')
			}
			lastSpace = true
			continue
		}
		lastSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

var accents = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

// ingredientNames returns the names linked to a drug, sorted like the database does.
func (f *FakeDrugQuerier) ingredientNames(drugID int64) []string {
	out := []string{}
	for _, id := range f.links[drugID] {
		out = append(out, f.names[id])
	}
	sort.Strings(out)
	return out
}

// Ingredients is a helper for the asserts: the ingredients of a drug.
func (f *FakeDrugQuerier) Ingredients(drugID int64) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ingredientNames(drugID)
}

func (f *FakeDrugQuerier) UpsertIngredient(ctx context.Context, name string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.LastIngredientNames = append(f.LastIngredientNames, name)
	key := normalizeIngredient(name)
	if id, ok := f.ingredients[key]; ok {
		return id, nil
	}
	id := f.nextID
	f.nextID++
	f.ingredients[key] = id
	f.names[id] = strings.TrimSpace(name)
	return id, nil
}

func (f *FakeDrugQuerier) LinkDrugIngredient(ctx context.Context, arg db.LinkDrugIngredientParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, id := range f.links[arg.DrugID] {
		if id == arg.IngredientID {
			return nil
		}
	}
	f.links[arg.DrugID] = append(f.links[arg.DrugID], arg.IngredientID)
	return nil
}

func (f *FakeDrugQuerier) DeleteDrugIngredientsNotIn(ctx context.Context, arg db.DeleteDrugIngredientsNotInParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	kept := f.links[arg.DrugID][:0]
	for _, id := range f.links[arg.DrugID] {
		for _, keep := range arg.IngredientIds {
			if id == keep {
				kept = append(kept, id)
				break
			}
		}
	}
	f.links[arg.DrugID] = kept
	return nil
}

func (f *FakeDrugQuerier) ListIngredientsByDrugID(ctx context.Context, drugID int64) ([]db.ListIngredientsByDrugIDRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []db.ListIngredientsByDrugIDRow
	for _, id := range f.links[drugID] {
		out = append(out, db.ListIngredientsByDrugIDRow{ID: id, Name: f.names[id]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (f *FakeDrugQuerier) ListDrugsByEANs(ctx context.Context, eans []string) ([]db.ListDrugsByEANsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return nil, f.GetErr
	}
	var out []db.ListDrugsByEANsRow
	for _, ean := range eans {
		drugID, ok := f.eans[ean]
		if !ok {
			continue
		}
		d, ok := f.drugs[drugID]
		if !ok {
			continue
		}
		out = append(out, db.ListDrugsByEANsRow{
			Ean:               ean,
			DrugID:            d.ID,
			BrandName:         d.BrandName,
			Manufacturer:      d.Manufacturer,
			ActiveIngredients: f.ingredientNames(d.ID),
		})
	}
	return out, nil
}

// ---- interaction rules ----

type fakeRule struct {
	aID, bID       int64
	severity       string
	description    string
	recommendation string
	sourceURL      string
}

func pairKey(a, b int64) [2]int64 {
	if a > b {
		a, b = b, a
	}
	return [2]int64{a, b}
}

func (f *FakeDrugQuerier) UpsertIngredientInteraction(ctx context.Context, arg db.UpsertIngredientInteractionParams) (db.UpsertIngredientInteractionRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rules == nil {
		f.rules = map[[2]int64]fakeRule{}
	}
	key := pairKey(arg.IngredientAID, arg.IngredientBID)
	_, existed := f.rules[key]
	f.rules[key] = fakeRule{
		aID: key[0], bID: key[1], severity: arg.Severity, description: arg.Description,
		recommendation: arg.Recommendation.String, sourceURL: arg.SourceUrl,
	}
	return db.UpsertIngredientInteractionRow{IngredientAID: key[0], IngredientBID: key[1], Created: !existed}, nil
}

func (f *FakeDrugQuerier) DeleteIngredientInteraction(ctx context.Context, arg db.DeleteIngredientInteractionParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := pairKey(arg.IngredientAID, arg.IngredientBID)
	if _, ok := f.rules[key]; !ok {
		return 0, nil
	}
	delete(f.rules, key)
	return 1, nil
}

func (f *FakeDrugQuerier) ListIngredientInteractions(ctx context.Context, arg db.ListIngredientInteractionsParams) ([]db.ListIngredientInteractionsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []db.ListIngredientInteractionsRow
	for _, r := range f.rules {
		out = append(out, db.ListIngredientInteractionsRow{
			IngredientAID: r.aID, IngredientA: f.names[r.aID],
			IngredientBID: r.bID, IngredientB: f.names[r.bID],
			Severity: r.severity, Description: r.description,
			Recommendation: pgtype.Text{String: r.recommendation, Valid: r.recommendation != ""},
			SourceUrl:      r.sourceURL,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IngredientA != out[j].IngredientA {
			return out[i].IngredientA < out[j].IngredientA
		}
		return out[i].IngredientB < out[j].IngredientB
	})

	offset := int(arg.Offset)
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit := int(arg.Limit); limit >= 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeDrugQuerier) CountIngredientInteractions(ctx context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int64(len(f.rules)), nil
}

// FindInteractionsBetweenEANs mirrors the real query: every pair of ingredients
// coming from two DIFFERENT drugs is looked up in the rules.
func (f *FakeDrugQuerier) FindInteractionsBetweenEANs(ctx context.Context, eans []string) ([]db.FindInteractionsBetweenEANsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return nil, f.GetErr
	}

	drugIDs := []int64{}
	seen := map[int64]bool{}
	for _, ean := range eans {
		if id, ok := f.eans[ean]; ok && !seen[id] {
			seen[id] = true
			drugIDs = append(drugIDs, id)
		}
	}
	sort.Slice(drugIDs, func(i, j int) bool { return drugIDs[i] < drugIDs[j] })

	var out []db.FindInteractionsBetweenEANsRow
	for i, drugA := range drugIDs {
		for _, drugB := range drugIDs[i+1:] {
			for _, ingA := range f.links[drugA] {
				for _, ingB := range f.links[drugB] {
					rule, ok := f.rules[pairKey(ingA, ingB)]
					if !ok || ingA == ingB {
						continue
					}
					out = append(out, db.FindInteractionsBetweenEANsRow{
						IngredientA: f.names[rule.aID],
						IngredientB: f.names[rule.bID],
						Severity:    rule.severity,
						Description: rule.description,
						Recommendation: pgtype.Text{
							String: rule.recommendation,
							Valid:  rule.recommendation != "",
						},
						SourceUrl: rule.sourceURL,
						DrugAID:   drugA,
						DrugBID:   drugB,
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Severity < out[j].Severity })
	return out, nil
}
