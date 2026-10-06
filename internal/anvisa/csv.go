// Package anvisa reads the open data files Anvisa publishes and turns them into
// the same rows the spreadsheet importer produces, so both paths share one
// upsert routine.
//
// Two files are used:
//
//	DADOS_ABERTOS_MEDICAMENTOS.csv  -> drugs + active ingredients
//	TA_PRECO_MEDICAMENTO.csv        -> packages + EANs (the CMED price list)
//
// Nothing here writes to the database: the result is the row types of the
// importer package, which the import service already knows how to apply.
package anvisa

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
)

// ErrNoRows is returned when a file has a usable header but no data.
var ErrNoRows = errors.New("the file has no data rows")

// headerSearchLimit is how many leading lines are searched for the header. The
// CMED list opens with a long legal notice: its header sits on line 60.
const headerSearchLimit = 100

var nonDigits = regexp.MustCompile(`[^0-9]`)

// onlyDigits cleans numbers that come formatted ("1.0043.2105-1") or padded.
func onlyDigits(s string) string { return nonDigits.ReplaceAllString(s, "") }

// reader reads one Anvisa file. They are semicolon separated and have been
// published both in UTF-8 and in Windows-1252, so the encoding is detected
// instead of assumed.
type reader struct {
	name    string // file name, used in the error rows
	csv     *csv.Reader
	columns map[string]int
	line    int
}

// newReader finds the header and indexes its columns. Column names are matched
// the forgiving way (case, accents and punctuation ignored), so
// "PRINCÍPIO_ATIVO", "principio ativo" and "PrincipioAtivo" all match.
//
// want lists the column groups the caller needs, each group being the accepted
// names of one column. A line only counts as the header when it carries at
// least one name of every group, which is what lets the leading title rows of
// the CMED spreadsheet be skipped.
func newReader(name string, r io.Reader, want [][]string) (*reader, error) {
	c := csv.NewReader(bufio.NewReaderSize(decode(r), 64*1024))
	c.Comma = ';'
	// The files are not quoted consistently: a lone " inside a product name is
	// common and must not break the row.
	c.LazyQuotes = true
	// Rows with a trailing empty field happen; let each row have its own length.
	c.FieldsPerRecord = -1

	var last map[string]int
	for line := 1; line <= headerSearchLimit; line++ {
		cells, err := c.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: could not read the header: %w", name, err)
		}

		columns := index(cells)
		if len(columns) > len(last) {
			last = columns
		}
		if complete(columns, want) {
			return &reader{name: name, csv: c, columns: columns, line: line}, nil
		}
	}
	return nil, missingColumns(name, last, want)
}

// index maps every non-empty header cell to its position, keeping the first of
// a repeated name.
func index(cells []string) map[string]int {
	columns := make(map[string]int, len(cells))
	for i, cell := range cells {
		key := importer.NormalizeHeader(strings.TrimPrefix(cell, "\ufeff"))
		if key == "" {
			continue
		}
		if _, repeated := columns[key]; !repeated {
			columns[key] = i
		}
	}
	return columns
}

// complete reports whether the header carries one accepted name of every group.
func complete(columns map[string]int, want [][]string) bool {
	for _, names := range want {
		if !anyOf(columns, names) {
			return false
		}
	}
	return len(columns) > 0
}

func anyOf(columns map[string]int, names []string) bool {
	for _, name := range names {
		if _, ok := columns[name]; ok {
			return true
		}
	}
	return false
}

// missingColumns builds a readable error: which column is missing and what the
// file actually has. The usual cause is simply the wrong file.
func missingColumns(name string, columns map[string]int, want [][]string) error {
	var missing []string
	for _, names := range want {
		if !anyOf(columns, names) {
			missing = append(missing, strings.Join(names, " / "))
		}
	}
	have := make([]string, 0, len(columns))
	for column := range columns {
		have = append(have, column)
	}
	if len(have) == 0 {
		return fmt.Errorf("%s: no header found in the first %d lines", name, headerSearchLimit)
	}
	return fmt.Errorf("%s: column %s not found; the file has: %s",
		name, strings.Join(missing, ", "), strings.Join(have, ", "))
}

// row is one line of the file, read by column name.
type row struct {
	line   int
	values []string
	cols   map[string]int
}

// next reads the following line. It returns io.EOF when the file ends.
func (r *reader) next() (row, error) {
	values, err := r.csv.Read()
	if err != nil {
		return row{}, err
	}
	r.line++
	return row{line: r.line, values: values, cols: r.columns}, nil
}

// get returns the first of the given column names that is filled in, trimmed.
func (v row) get(names ...string) string {
	for _, name := range names {
		i, ok := v.cols[name]
		if !ok || i >= len(v.values) {
			continue
		}
		if value := strings.TrimSpace(v.values[i]); value != "" {
			return value
		}
	}
	return ""
}

// empty reports whether every cell of the row is blank.
func (v row) empty() bool {
	for _, cell := range v.values {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// decode makes the stream UTF-8. Anvisa publishes these files in Windows-1252
// (so "PRINCÍPIO" arrives as invalid UTF-8), but has published UTF-8 versions
// too, so the first chunk decides instead of a fixed guess.
func decode(r io.Reader) io.Reader {
	br := bufio.NewReaderSize(r, 32*1024)
	head, _ := br.Peek(8192)
	if utf8.Valid(head) {
		return br
	}
	return charmap.Windows1252.NewDecoder().Reader(br)
}
