package importer

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// MaxRows caps how much a single import can carry, so a wrong file can't hold
// the request (and the database transaction) open for minutes.
const MaxRows = 5000

// RowError points at one cell the person has to fix.
type RowError struct {
	Sheet   string `json:"sheet" example:"embalagens"`
	Line    int    `json:"line" example:"7"`
	Column  string `json:"column" example:"ean_1"`
	Message string `json:"message" example:"must have 8 to 13 digits"`
}

func (e RowError) Error() string {
	return fmt.Sprintf("%s line %d (%s): %s", e.Sheet, e.Line, e.Column, e.Message)
}

// DrugRow is one line of the "remedios" sheet.
type DrugRow struct {
	Line               int
	RegistrationNumber string
	BrandName          string
	// ActiveIngredients: the cell is split on "+" and ";", so a combination
	// drug becomes one entry per ingredient.
	ActiveIngredients []string
	Manufacturer      string

	// Leaflet (optional as a whole; when any field is filled the required ones must be too)
	HasSummary         bool
	WhatIsItFor        string
	Posology           string
	MissedDose         string
	Warnings           string
	AdverseEffects     string
	DrugInteractions   string
	Contraindications  string
	SideEffects        string
	WhenToSeekHelp     string
	MechanismOfAction  string
	Storage            string
	SourceURL          string
	LeafletExpedient   string
	LeafletPublishedAt string // always normalized to YYYY-MM-DD
}

// PackageRow is one line of the "embalagens" sheet.
type PackageRow struct {
	Line                     int
	DrugRegistration         string
	PresentationRegistration string
	Description              string
	Eans                     []string
}

// InteractionRow is one line of the "interacoes_ativos" sheet: one rule for a
// pair of active ingredients.
type InteractionRow struct {
	Line           int
	IngredientA    string
	IngredientB    string
	Severity       string
	Description    string
	Recommendation string
	SourceURL      string
}

// File is a parsed spreadsheet, ready to be written to the database.
type File struct {
	Drugs        []DrugRow
	Packages     []PackageRow
	Interactions []InteractionRow
}

var digits = regexp.MustCompile(`^[0-9]+$`)

// ingredientSeparators: how combinations are written in the Bulário ("+") and
// in the CMED (";"). A comma is not a separator: it shows up inside a name.
var ingredientSeparators = regexp.MustCompile(`\s*[+;]\s*`)

// SplitIngredients turns "dipirona sódica + cafeína" into two entries. Exported
// so every source (spreadsheet, Anvisa open data) splits combinations alike.
func SplitIngredients(cell string) []string {
	var out []string
	for _, part := range ingredientSeparators.Split(cell, -1) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Parse reads the spreadsheet and validates every cell. It always returns all
// the errors it found, not just the first one, so the person can fix the file
// in one pass.
func Parse(r io.Reader) (*File, []RowError, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read the spreadsheet: %w", err)
	}
	defer f.Close()

	var errs []RowError
	file := &File{}

	drugRows, err := sheetRows(f, SheetDrugs)
	if err != nil {
		return nil, nil, err
	}
	pkgRows, err := sheetRows(f, SheetPackages)
	if err != nil {
		return nil, nil, err
	}
	// The interactions sheet is optional: older files don't have it.
	interactionRows, err := optionalSheetRows(f, SheetInteractions)
	if err != nil {
		return nil, nil, err
	}
	if len(drugRows)+len(pkgRows)+len(interactionRows) > MaxRows+3 {
		return nil, nil, fmt.Errorf("the spreadsheet has more than %d rows; split it into smaller files", MaxRows)
	}

	drugCols, errs2 := headerIndex(SheetDrugs, drugRows, DrugColumns)
	errs = append(errs, errs2...)
	pkgCols, errs3 := headerIndex(SheetPackages, pkgRows, PackageColumns)
	errs = append(errs, errs3...)
	var interactionCols map[string]int
	if len(interactionRows) > 0 {
		var errs4 []RowError
		interactionCols, errs4 = headerIndex(SheetInteractions, interactionRows, InteractionColumns)
		errs = append(errs, errs4...)
	}
	if len(errs) > 0 {
		return nil, errs, nil // header is broken: reporting cell errors would only add noise
	}

	seenDrug := map[string]int{}
	for i, row := range drugRows[1:] {
		line := i + 2 // 1-based, and the header is line 1
		get := cellGetter(row, drugCols)
		if isEmptyRow(row) {
			continue
		}

		d := DrugRow{
			Line:               line,
			RegistrationNumber: strings.TrimSpace(get("registro_anvisa")),
			BrandName:          strings.TrimSpace(get("nome_comercial")),
			ActiveIngredients:  SplitIngredients(get("principio_ativo")),
			Manufacturer:       strings.TrimSpace(get("fabricante")),
			WhatIsItFor:        strings.TrimSpace(get("para_que_serve")),
			Posology:           strings.TrimSpace(get("posologia")),
			MissedDose:         strings.TrimSpace(get("esqueceu_dose")),
			Warnings:           strings.TrimSpace(get("advertencias")),
			AdverseEffects:     strings.TrimSpace(get("reacoes_adversas")),
			DrugInteractions:   strings.TrimSpace(get("interacoes")),
			Contraindications:  strings.TrimSpace(get("contraindicacoes")),
			SideEffects:        strings.TrimSpace(get("efeitos_colaterais")),
			WhenToSeekHelp:     strings.TrimSpace(get("quando_procurar_ajuda")),
			MechanismOfAction:  strings.TrimSpace(get("como_funciona")),
			Storage:            strings.TrimSpace(get("armazenamento")),
			SourceURL:          strings.TrimSpace(get("fonte_url")),
			LeafletExpedient:   strings.TrimSpace(get("bula_expediente")),
		}

		if len(d.ActiveIngredients) == 0 {
			errs = append(errs, RowError{SheetDrugs, line, "principio_ativo", "required"})
		}
		for _, req := range []struct{ col, val string }{
			{"registro_anvisa", d.RegistrationNumber},
			{"fabricante", d.Manufacturer},
		} {
			if req.val == "" {
				errs = append(errs, RowError{SheetDrugs, line, req.col, "required"})
			}
		}
		if d.RegistrationNumber != "" {
			if !digits.MatchString(d.RegistrationNumber) {
				errs = append(errs, RowError{SheetDrugs, line, "registro_anvisa", "only digits"})
			}
			if before, dup := seenDrug[d.RegistrationNumber]; dup {
				errs = append(errs, RowError{SheetDrugs, line, "registro_anvisa",
					fmt.Sprintf("repeated in the spreadsheet (also on line %d)", before)})
			}
			seenDrug[d.RegistrationNumber] = line
		}

		// The leaflet is optional, but a half-filled one is not accepted.
		d.HasSummary = anyFilled(d.WhatIsItFor, d.Posology, d.MissedDose, d.Warnings, d.AdverseEffects,
			d.DrugInteractions, d.Contraindications, d.SideEffects, d.WhenToSeekHelp, d.MechanismOfAction,
			d.Storage, d.SourceURL, d.LeafletExpedient, strings.TrimSpace(get("bula_publicada_em")))
		if d.HasSummary {
			for _, req := range []struct{ col, val string }{
				{"para_que_serve", d.WhatIsItFor},
				{"posologia", d.Posology},
				{"fonte_url", d.SourceURL},
			} {
				if req.val == "" {
					errs = append(errs, RowError{SheetDrugs, line, req.col, "required when the leaflet is filled in"})
				}
			}
			published, err := parseDate(strings.TrimSpace(get("bula_publicada_em")))
			if err != nil {
				errs = append(errs, RowError{SheetDrugs, line, "bula_publicada_em", err.Error()})
			}
			d.LeafletPublishedAt = published
		}

		file.Drugs = append(file.Drugs, d)
	}

	seenEan := map[string]int{}
	seenPresentation := map[string]int{}
	for i, row := range pkgRows[1:] {
		line := i + 2
		if isEmptyRow(row) {
			continue
		}
		get := cellGetter(row, pkgCols)

		p := PackageRow{
			Line:                     line,
			DrugRegistration:         strings.TrimSpace(get("registro_anvisa")),
			PresentationRegistration: strings.TrimSpace(get("registro_apresentacao")),
			Description:              strings.TrimSpace(get("descricao")),
		}

		if p.DrugRegistration == "" {
			errs = append(errs, RowError{SheetPackages, line, "registro_anvisa", "required"})
		}
		if p.Description == "" {
			errs = append(errs, RowError{SheetPackages, line, "descricao", "required"})
		}
		if p.PresentationRegistration == "" {
			errs = append(errs, RowError{SheetPackages, line, "registro_apresentacao", "required"})
		} else if len(p.PresentationRegistration) != 13 || !digits.MatchString(p.PresentationRegistration) {
			errs = append(errs, RowError{SheetPackages, line, "registro_apresentacao", "must have exactly 13 digits"})
		} else if before, dup := seenPresentation[p.PresentationRegistration]; dup {
			errs = append(errs, RowError{SheetPackages, line, "registro_apresentacao",
				fmt.Sprintf("repeated in the spreadsheet (also on line %d)", before)})
		} else {
			seenPresentation[p.PresentationRegistration] = line
			if p.DrugRegistration != "" && !strings.HasPrefix(p.PresentationRegistration, p.DrugRegistration) {
				errs = append(errs, RowError{SheetPackages, line, "registro_apresentacao",
					"should start with the drug registration (" + p.DrugRegistration + ")"})
			}
		}

		for _, col := range []string{"ean_1", "ean_2", "ean_3"} {
			ean := strings.TrimSpace(get(col))
			if ean == "" {
				if col == "ean_1" {
					errs = append(errs, RowError{SheetPackages, line, col, "required"})
				}
				continue
			}
			if !digits.MatchString(ean) || len(ean) < 8 || len(ean) > 13 {
				errs = append(errs, RowError{SheetPackages, line, col, "must have 8 to 13 digits"})
				continue
			}
			if before, dup := seenEan[ean]; dup {
				errs = append(errs, RowError{SheetPackages, line, col,
					fmt.Sprintf("EAN repeated in the spreadsheet (also on line %d)", before)})
				continue
			}
			seenEan[ean] = line
			p.Eans = append(p.Eans, ean)
		}

		file.Packages = append(file.Packages, p)
	}

	seenPair := map[string]int{}
	for i, row := range dataRows(interactionRows) {
		line := i + 2
		if isEmptyRow(row) {
			continue
		}
		get := cellGetter(row, interactionCols)

		r := InteractionRow{
			Line:           line,
			IngredientA:    strings.TrimSpace(get("principio_a")),
			IngredientB:    strings.TrimSpace(get("principio_b")),
			Severity:       normalizeIngredient(get("gravidade")),
			Description:    strings.TrimSpace(get("descricao")),
			Recommendation: strings.TrimSpace(get("recomendacao")),
			SourceURL:      strings.TrimSpace(get("fonte_url")),
		}

		for _, req := range []struct{ col, val string }{
			{"principio_a", r.IngredientA},
			{"principio_b", r.IngredientB},
			{"descricao", r.Description},
			{"fonte_url", r.SourceURL},
		} {
			if req.val == "" {
				errs = append(errs, RowError{SheetInteractions, line, req.col, "required"})
			}
		}
		if !validSeverities[r.Severity] {
			errs = append(errs, RowError{SheetInteractions, line, "gravidade", "use grave, moderada or leve"})
		}

		a, b := normalizeIngredient(r.IngredientA), normalizeIngredient(r.IngredientB)
		if a != "" && a == b {
			errs = append(errs, RowError{SheetInteractions, line, "principio_b",
				"must be a different active ingredient from principio_a"})
		} else if a != "" && b != "" {
			// The pair has no order: a,b and b,a are the same rule.
			key := a + "|" + b
			if a > b {
				key = b + "|" + a
			}
			if before, dup := seenPair[key]; dup {
				errs = append(errs, RowError{SheetInteractions, line, "principio_a",
					fmt.Sprintf("pair repeated in the spreadsheet (also on line %d)", before)})
			}
			seenPair[key] = line
		}

		file.Interactions = append(file.Interactions, r)
	}

	if len(file.Drugs) == 0 && len(file.Packages) == 0 && len(file.Interactions) == 0 {
		return nil, nil, fmt.Errorf("the spreadsheet has no data rows")
	}
	return file, errs, nil
}

// validSeverities mirrors the chk_severity constraint of the database.
var validSeverities = map[string]bool{"grave": true, "moderada": true, "leve": true}

// sheetRows returns every row of a sheet, or an error naming the missing sheet.
func sheetRows(f *excelize.File, name string) ([][]string, error) {
	for _, sheet := range f.GetSheetList() {
		if NormalizeHeader(sheet) != name {
			continue
		}
		rows, err := f.GetRows(sheet)
		if err != nil {
			return nil, fmt.Errorf("could not read sheet %q: %w", sheet, err)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("sheet %q is empty; it needs at least the header row", sheet)
		}
		return rows, nil
	}
	return nil, fmt.Errorf("sheet %q not found; download the template and use it", name)
}

// optionalSheetRows is sheetRows for a sheet that may not be in the file: a
// missing sheet is not an error, it just has no rows.
func optionalSheetRows(f *excelize.File, name string) ([][]string, error) {
	for _, sheet := range f.GetSheetList() {
		if NormalizeHeader(sheet) == name {
			return sheetRows(f, name)
		}
	}
	return nil, nil
}

// dataRows drops the header row, and copes with a sheet that has no rows.
func dataRows(rows [][]string) [][]string {
	if len(rows) < 2 {
		return nil
	}
	return rows[1:]
}

// headerIndex maps each expected column to its position in the header row.
func headerIndex(sheet string, rows [][]string, cols []column) (map[string]int, []RowError) {
	index := map[string]int{}
	for i, cell := range rows[0] {
		index[NormalizeHeader(cell)] = i
	}
	var errs []RowError
	for _, c := range cols {
		if _, ok := index[c.Name]; !ok && c.Required {
			errs = append(errs, RowError{sheet, 1, c.Name, "column missing from the header"})
		}
	}
	return index, errs
}

func cellGetter(row []string, index map[string]int) func(string) string {
	return func(col string) string {
		i, ok := index[col]
		if !ok || i >= len(row) {
			return ""
		}
		return row[i]
	}
}

func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func anyFilled(values ...string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

// parseDate accepts an empty cell, YYYY-MM-DD or DD/MM/YYYY and always returns
// YYYY-MM-DD, which is what the API expects.
func parseDate(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	for _, layout := range []string{time.DateOnly, "02/01/2006", "02/01/06", "2006/01/02"} {
		if d, err := time.Parse(layout, s); err == nil {
			return d.Format(time.DateOnly), nil
		}
	}
	return "", fmt.Errorf("invalid date %q: use AAAA-MM-DD or DD/MM/AAAA", s)
}
