package importer_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
)

// build writes a spreadsheet with the given sheets, where each sheet is a list
// of rows (the first one being the header).
func build(t *testing.T, sheets map[string][][]string) *bytes.Reader {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	for name, rows := range sheets {
		_, err := f.NewSheet(name)
		require.NoError(t, err)
		for r, row := range rows {
			for c, value := range row {
				cell, err := excelize.CoordinatesToCellName(c+1, r+1)
				require.NoError(t, err)
				require.NoError(t, f.SetCellStr(name, cell, value))
			}
		}
	}
	f.DeleteSheet("Sheet1")
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	return bytes.NewReader(buf.Bytes())
}

var drugHeader = []string{"registro_anvisa", "nome_comercial", "principio_ativo", "fabricante",
	"para_que_serve", "posologia", "fonte_url", "bula_publicada_em"}

var packageHeader = []string{"registro_anvisa", "registro_apresentacao", "descricao", "ean_1", "ean_2", "ean_3"}

func TestParse_Success(t *testing.T) {
	file, errs, err := importer.Parse(build(t, map[string][][]string{
		"remedios": {drugHeader,
			{"192900007", "Advil 12h", "ibuprofeno", "Pfizer", "Dor e febre", "1 a cada 12h", "https://x", "31/05/2024"},
			{"", "", "", "", "", "", "", ""}, // empty rows are ignored
		},
		"embalagens": {packageHeader,
			{"192900007", "1929000070034", "600 mg, caixa com 10", " 7896015592752 ", "7896015592769", ""},
		},
	}))

	require.NoError(t, err)
	assert.Empty(t, errs)
	require.Len(t, file.Drugs, 1)
	require.Len(t, file.Packages, 1)

	d := file.Drugs[0]
	assert.Equal(t, "192900007", d.RegistrationNumber)
	assert.True(t, d.HasSummary)
	assert.Equal(t, "2024-05-31", d.LeafletPublishedAt, "DD/MM/AAAA is converted")

	p := file.Packages[0]
	assert.Equal(t, []string{"7896015592752", "7896015592769"}, p.Eans, "spaces are trimmed, empty EANs dropped")
}

func TestParse_DrugWithoutLeaflet(t *testing.T) {
	file, errs, err := importer.Parse(build(t, map[string][][]string{
		"remedios":   {drugHeader, {"192900007", "Advil 12h", "ibuprofeno", "Pfizer", "", "", "", ""}},
		"embalagens": {packageHeader},
	}))

	require.NoError(t, err)
	assert.Empty(t, errs)
	assert.False(t, file.Drugs[0].HasSummary, "leaflet is optional")
}

func TestParse_RowErrors(t *testing.T) {
	_, errs, err := importer.Parse(build(t, map[string][][]string{
		"remedios": {drugHeader,
			{"192900007", "Advil 12h", "", "Pfizer", "", "", "", ""},              // no active ingredient
			{"192900007", "Advil 12h", "ibuprofeno", "Pfizer", "Dor", "", "", ""}, // repeated + half-filled leaflet
		},
		"embalagens": {packageHeader,
			{"192900007", "12345", "caixa", "7896015592752", "", ""},                        // registration too short
			{"192900007", "1929000070069", "caixa", "789601559275X", "7896015592752", ""},   // invalid EAN + repeated EAN
			{"000000000", "0000000000034", "caixa", "7896015592700", "", ""},                // drug not in the sheet (caught when importing)
		},
	}))

	require.NoError(t, err)
	got := map[string]string{}
	for _, e := range errs {
		got[e.Column] = e.Message
		assert.NotZero(t, e.Line)
	}
	assert.Contains(t, got, "principio_ativo")
	assert.Contains(t, got, "registro_anvisa")   // repeated
	assert.Contains(t, got, "posologia")         // required once the leaflet is filled
	assert.Contains(t, got, "fonte_url")         // same
	assert.Contains(t, got, "registro_apresentacao")
	assert.Contains(t, got, "ean_1")
	assert.Contains(t, got, "ean_2")
}

func TestParse_BrokenFiles(t *testing.T) {
	t.Run("missing sheet", func(t *testing.T) {
		_, _, err := importer.Parse(build(t, map[string][][]string{"remedios": {drugHeader}}))
		assert.ErrorContains(t, err, "embalagens")
	})

	t.Run("missing column in the header", func(t *testing.T) {
		_, errs, err := importer.Parse(build(t, map[string][][]string{
			"remedios":   {{"registro_anvisa", "principio_ativo"}}, // no fabricante
			"embalagens": {packageHeader},
		}))
		require.NoError(t, err)
		require.NotEmpty(t, errs)
		assert.Equal(t, "fabricante", errs[0].Column)
		assert.Equal(t, 1, errs[0].Line)
	})

	t.Run("not a spreadsheet", func(t *testing.T) {
		_, _, err := importer.Parse(bytes.NewReader([]byte("isso nao e um xlsx")))
		assert.Error(t, err)
	})

	t.Run("only headers", func(t *testing.T) {
		_, _, err := importer.Parse(build(t, map[string][][]string{
			"remedios": {drugHeader}, "embalagens": {packageHeader},
		}))
		assert.ErrorContains(t, err, "no data rows")
	})
}

func TestBuildTemplate(t *testing.T) {
	data, err := importer.BuildTemplate()
	require.NoError(t, err)

	// The template must be importable as it comes, example rows and all.
	file, errs, err := importer.Parse(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Empty(t, errs)
	assert.Len(t, file.Drugs, 1)
	assert.Len(t, file.Packages, 2)
	assert.True(t, file.Drugs[0].HasSummary)
}

// Headers written in a different case, with accents or spaces still match.
func TestParse_ForgivingHeaders(t *testing.T) {
	file, errs, err := importer.Parse(build(t, map[string][][]string{
		"Remédios": {{"Registro Anvisa", "Nome Comercial", "Princípio Ativo", "Fabricante"},
			{"192900007", "Advil 12h", "ibuprofeno", "Pfizer"}},
		"EMBALAGENS": {{"registro anvisa", "REGISTRO APRESENTACAO", "Descrição", "EAN 1"},
			{"192900007", "1929000070034", "caixa com 10", "7896015592752"}},
	}))

	require.NoError(t, err)
	assert.Empty(t, errs)
	assert.Len(t, file.Drugs, 1)
	assert.Len(t, file.Packages, 1)
}
