package anvisa

import (
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
)

// PrecosFile is the CMED price list: one line per presentation, with the
// barcodes printed on the box.
const PrecosFile = "TA_PRECO_MEDICAMENTO.csv"

// Column names of the CMED list. The .csv of the open data and the .xlsx
// published on gov.br name the same fields differently, so both are accepted.
var (
	colPresentation = []string{"registro", "numero_registro_produto", "registro_produto"}
	colDescription  = []string{"apresentacao", "descricao_apresentacao", "descricao"}
	colEans         = [][]string{
		{"ean_1", "ean1", "ean", "codigo_ean_1", "codigo_barras"},
		{"ean_2", "ean2", "codigo_ean_2"},
		{"ean_3", "ean3", "codigo_ean_3"},
	}
)

// ParsePrecos reads the CMED list into package rows. Every presentation of the
// file becomes a package of the drug whose registration is the first 9 digits
// of its own 13-digit registration.
//
// A presentation without a usable barcode is skipped: the app finds the box by
// its EAN, so a package without one would never be read.
//
// A barcode claimed by two presentations is NOT decided here, because the file
// alone cannot say which one is right: ResolveBarcodes does that, with the
// registration status of the drugs file in hand.
func ParsePrecos(r io.Reader) ([]importer.PackageRow, []importer.RowError, error) {
	rd, err := newReader(PrecosFile, r, [][]string{colPresentation, colEans[0]})
	if err != nil {
		return nil, nil, err
	}

	var (
		packages []importer.PackageRow
		errs     []importer.RowError
		seenPkg  = map[string]int{}
	)

	for {
		v, err := rd.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if v.empty() {
			continue
		}

		presentation := onlyDigits(v.get(colPresentation...))
		if len(presentation) != 13 {
			errs = append(errs, importer.RowError{
				Sheet: PrecosFile, Line: v.line, Column: colPresentation[0],
				Message: "presentation registration does not have 13 digits",
			})
			continue
		}

		description := cleanDescription(v.get(colDescription...))
		if description == "" {
			description = "Apresentação " + presentation
		}

		pkg := importer.PackageRow{
			Line:                     v.line,
			DrugRegistration:         presentation[:9],
			PresentationRegistration: presentation,
			Description:              description,
		}

		for i, names := range colEans {
			ean := onlyDigits(v.get(names...))
			if ean == "" {
				continue
			}
			if len(ean) < 8 || len(ean) > 13 {
				errs = append(errs, importer.RowError{
					Sheet: PrecosFile, Line: v.line, Column: colEans[i][0],
					Message: "barcode does not have 8 to 13 digits",
				})
				continue
			}
			pkg.Eans = append(pkg.Eans, ean)
		}

		if len(pkg.Eans) == 0 {
			continue // nothing for the app to scan
		}

		if at, repeated := seenPkg[presentation]; repeated {
			// Same presentation twice (one line per price bracket): keep the
			// first and only add barcodes the first line did not have.
			packages[at].Eans = append(packages[at].Eans, pkg.Eans...)
			continue
		}
		seenPkg[presentation] = len(packages)
		packages = append(packages, pkg)
	}

	if len(packages) == 0 && len(errs) == 0 {
		return nil, nil, ErrNoRows
	}
	return packages, errs, nil
}

// cleanDescription tidies the CMED description, which comes in caps and with
// runs of spaces: "600 MG COM REV LIB PROL   CT BL AL PLAS INC X 20".
func cleanDescription(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// ResolveBarcodes settles the barcodes two presentations both claim.
//
// It happens 170-odd times in the real list: a product keeps its box (and its
// barcode) when the registration is renewed or transferred to another company,
// so the old registration and the new one are both printed with the same code.
// Our package_eans has a unique barcode, so one of them has to win.
//
// The winner is the presentation whose drug Anvisa still marks as active; with
// no way to tell them apart, the first one in the file keeps it. A presentation
// left with no barcode at all is dropped: the app would never reach it.
func ResolveBarcodes(packages []importer.PackageRow, active map[string]bool) ([]importer.PackageRow, []importer.RowError) {
	// owner: barcode -> index of the package that holds it so far.
	owner := make(map[string]int, len(packages))
	var errs []importer.RowError

	for i := range packages {
		kept := packages[i].Eans[:0]
		for _, ean := range packages[i].Eans {
			at, taken := owner[ean]
			if !taken {
				owner[ean] = i
				kept = append(kept, ean)
				continue
			}
			if active[packages[i].DrugRegistration] && !active[packages[at].DrugRegistration] {
				// This one is the active registration: take the barcode over.
				drop(&packages[at], ean)
				owner[ean] = i
				kept = append(kept, ean)
				errs = append(errs, conflict(packages[at], ean))
				continue
			}
			errs = append(errs, conflict(packages[i], ean))
		}
		packages[i].Eans = kept
	}

	out := packages[:0]
	for _, p := range packages {
		if len(p.Eans) > 0 {
			out = append(out, p)
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Line < errs[j].Line })
	return out, errs
}

// drop removes one barcode from a package that lost it.
func drop(p *importer.PackageRow, ean string) {
	for i, own := range p.Eans {
		if own == ean {
			p.Eans = append(p.Eans[:i], p.Eans[i+1:]...)
			return
		}
	}
}

func conflict(p importer.PackageRow, ean string) importer.RowError {
	return importer.RowError{
		Sheet: PrecosFile, Line: p.Line, Column: "ean",
		Message: "barcode " + ean + " kept on the active registration of the same product",
	}
}
