package anvisa

import (
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
)

// MedicamentosFile is the name of the Anvisa file this parser reads. It is also
// what shows up in the "sheet" field of the row errors.
const MedicamentosFile = "DADOS_ABERTOS_MEDICAMENTOS.csv"

// Column names of DADOS_ABERTOS_MEDICAMENTOS.csv, already normalized. Several
// names are accepted per field because Anvisa has renamed columns between
// versions of the file.
var (
	colRegistration = []string{"numero_registro_produto", "numero_registro", "registro"}
	colProduct      = []string{"nome_produto", "produto"}
	colIngredient   = []string{"principio_ativo", "principios_ativos", "substancia"}
	colCompany      = []string{"empresa_detentora_registro", "empresa", "razao_social_empresa", "laboratorio"}
	colStatus       = []string{"situacao_registro", "situacao"}
)

// Medicamentos is what the drugs file gives us.
type Medicamentos struct {
	Drugs []importer.DrugRow
	// Active: registrations Anvisa still marks as active. The CMED list gives
	// the same barcode to an old and a new registration of the same product,
	// and this is what tells the two apart (see ResolveBarcodes).
	Active map[string]bool
}

// drugRegistration is the 9-digit number the Bulário uses for the drug. The
// open data writes it in several shapes ("1.0043.2105", "100432105",
// "1004321050012"), so it is cleaned and cut to the first 9 digits.
func drugRegistration(s string) string {
	d := onlyDigits(s)
	if len(d) > 9 {
		return d[:9]
	}
	return d
}

// ParseMedicamentos reads DADOS_ABERTOS_MEDICAMENTOS.csv into drug rows.
//
// onlyValid keeps only the registrations Anvisa marks as valid; with it false
// everything in the file comes through, cancelled and expired registrations
// included.
//
// A quarter of the file is products with no registration number at all (low
// risk products, which are notified rather than registered). Those are not
// errors and are skipped without a word: only a row that has a registration but
// cannot become a drug is reported, and reporting never stops the file.
func ParseMedicamentos(r io.Reader, onlyValid bool) (Medicamentos, []importer.RowError, error) {
	out := Medicamentos{Active: map[string]bool{}}

	rd, err := newReader(MedicamentosFile, r, [][]string{colRegistration, colIngredient})
	if err != nil {
		return out, nil, err
	}

	var (
		errs []importer.RowError
		// seen: the file has one line per registration, but a reissued
		// registration can show up twice. The last line wins.
		seen = map[string]int{}
	)

	for {
		v, err := rd.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return out, nil, err
		}
		if v.empty() {
			continue
		}

		active := validRegistration(v.get(colStatus...))
		if onlyValid && !active {
			continue
		}

		raw := v.get(colRegistration...)
		if raw == "" {
			continue // a notified product: it has no registration to key on
		}
		registration := drugRegistration(raw)
		if len(registration) != 9 {
			errs = append(errs, importer.RowError{
				Sheet: MedicamentosFile, Line: v.line, Column: colRegistration[0],
				Message: "registration number does not have 9 digits",
			})
			continue
		}

		ingredients := cleanIngredients(splitIngredients(v.get(colIngredient...)))
		if len(ingredients) == 0 {
			errs = append(errs, importer.RowError{
				Sheet: MedicamentosFile, Line: v.line, Column: colIngredient[0],
				Message: "empty active ingredient",
			})
			continue
		}

		company := cleanCompany(v.get(colCompany...))
		if company == "" {
			errs = append(errs, importer.RowError{
				Sheet: MedicamentosFile, Line: v.line, Column: colCompany[0],
				Message: "empty company",
			})
			continue
		}

		if active {
			out.Active[registration] = true
		}

		drug := importer.DrugRow{
			Line:               v.line,
			RegistrationNumber: registration,
			BrandName:          v.get(colProduct...),
			ActiveIngredients:  ingredients,
			Manufacturer:       company,
		}
		if at, repeated := seen[registration]; repeated {
			out.Drugs[at] = drug
			continue
		}
		seen[registration] = len(out.Drugs)
		out.Drugs = append(out.Drugs, drug)
	}

	if len(out.Drugs) == 0 && len(errs) == 0 {
		return out, nil, ErrNoRows
	}
	return out, errs, nil
}

// validRegistration reads the SITUACAO_REGISTRO column. Anvisa writes it in
// several ways ("VÁLIDO", "Valido", "ATIVO"), so the comparison is loose.
func validRegistration(status string) bool {
	switch importer.NormalizeHeader(status) {
	case "ativo", "ativa", "valido", "valida", "":
		return true
	}
	return false
}

// companyPrefix is the CNPJ the open data glues in front of the company name:
// "60659463002992 - ACHÉ LABORATÓRIOS FARMACÊUTICOS S.A".
var companyPrefix = regexp.MustCompile(`^[0-9./-]{11,20}\s*-\s*`)

// cleanCompany drops that CNPJ, which belongs to the company and not to the
// drug, and is not what the panel should show.
func cleanCompany(s string) string {
	return strings.TrimSpace(companyPrefix.ReplaceAllString(s, ""))
}

// anvisaSeparators: unlike our spreadsheet, where a comma can sit inside a
// name, the open data separates a combination with commas:
// "cafeína anidra, dipirona monoidratada, mucato de isometepteno".
var anvisaSeparators = regexp.MustCompile(`\s*[,;+]\s*`)

// splitIngredients breaks the cell of this file into one name per ingredient.
func splitIngredients(cell string) []string {
	return anvisaSeparators.Split(cell, -1)
}

// cleanIngredients tidies what the open data writes inside the active
// ingredient cell: dosages ("IBUPROFENO 600 MG"), empty markers and repeats.
func cleanIngredients(parts []string) []string {
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		name := strings.TrimSpace(strings.Trim(part, ",.-"))
		if name == "" || name == "-" {
			continue
		}
		key := importer.NormalizeHeader(name)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
	}
	return out
}
