package anvisa_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/anvisa"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
)

// latin1 writes the text the way Anvisa publishes it, so the tests cover the
// encoding the real files come in.
func latin1(t *testing.T, s string) *bytes.Reader {
	t.Helper()
	out, err := charmap.Windows1252.NewEncoder().Bytes([]byte(s))
	require.NoError(t, err)
	return bytes.NewReader(out)
}

const medicamentosHeader = "TIPO_PRODUTO;NOME_PRODUTO;NUMERO_REGISTRO_PRODUTO;" +
	"EMPRESA_DETENTORA_REGISTRO;PRINCIPIO_ATIVO;SITUACAO_REGISTRO\n"

func TestParseMedicamentos(t *testing.T) {
	file := medicamentosHeader +
		"MEDICAMENTO;ADVIL 12H;1.0043.2105;PF CONSUMER HEALTHCARE;IBUPROFENO;VÁLIDO\n" +
		"MEDICAMENTO;NEOSALDINA;100432106;TAKEDA;DIPIRONA SÓDICA + MUCATO DE ISOMETEPTENO + CAFEÍNA;Válido\n" +
		"MEDICAMENTO;FORA DE LINHA;100432107;X;PARACETAMOL;CANCELADO\n" +
		";;;;;\n"

	out, errs, err := anvisa.ParseMedicamentos(latin1(t, file), false)

	require.NoError(t, err)
	assert.Empty(t, errs)
	require.Len(t, out.Drugs, 3)

	assert.Equal(t, "100432105", out.Drugs[0].RegistrationNumber, "dots are removed")
	assert.Equal(t, "ADVIL 12H", out.Drugs[0].BrandName)
	assert.Equal(t, []string{"IBUPROFENO"}, out.Drugs[0].ActiveIngredients)
	assert.Equal(t, "PF CONSUMER HEALTHCARE", out.Drugs[0].Manufacturer)
	assert.False(t, out.Drugs[0].HasSummary, "Anvisa does not give us the simplified leaflet")

	assert.Equal(t, []string{"DIPIRONA SÓDICA", "MUCATO DE ISOMETEPTENO", "CAFEÍNA"},
		out.Drugs[1].ActiveIngredients, "accents survive and the combination is split")
}

func TestParseMedicamentos_OnlyValid(t *testing.T) {
	file := medicamentosHeader +
		"MEDICAMENTO;ADVIL 12H;100432105;PF;IBUPROFENO;VÁLIDO\n" +
		"MEDICAMENTO;FORA DE LINHA;100432107;X;PARACETAMOL;CANCELADO\n"

	out, _, err := anvisa.ParseMedicamentos(latin1(t, file), true)

	require.NoError(t, err)
	require.Len(t, out.Drugs, 1)
	assert.Equal(t, "100432105", out.Drugs[0].RegistrationNumber)
}

func TestParseMedicamentos_BadRows(t *testing.T) {
	file := medicamentosHeader +
		"MEDICAMENTO;SEM REGISTRO;123;X;IBUPROFENO;VÁLIDO\n" +
		"MEDICAMENTO;SEM PRINCIPIO;100432105;X;;VÁLIDO\n" +
		"MEDICAMENTO;SEM EMPRESA;100432106;;IBUPROFENO;VÁLIDO\n" +
		"MEDICAMENTO;OK;100432107;X;IBUPROFENO;VÁLIDO\n"

	out, errs, err := anvisa.ParseMedicamentos(latin1(t, file), false)

	require.NoError(t, err)
	require.Len(t, out.Drugs, 1, "a broken row does not stop the file")
	require.Len(t, errs, 3)
	for _, e := range errs {
		assert.Equal(t, anvisa.MedicamentosFile, e.Sheet)
		assert.NotZero(t, e.Line)
	}
}

func TestParseMedicamentos_RepeatedRegistration(t *testing.T) {
	file := medicamentosHeader +
		"MEDICAMENTO;NOME ANTIGO;100432105;X;IBUPROFENO;VÁLIDO\n" +
		"MEDICAMENTO;NOME NOVO;100432105;X;IBUPROFENO;VÁLIDO\n"

	out, errs, err := anvisa.ParseMedicamentos(latin1(t, file), false)

	require.NoError(t, err)
	assert.Empty(t, errs)
	require.Len(t, out.Drugs, 1)
	assert.Equal(t, "NOME NOVO", out.Drugs[0].BrandName, "the last line wins")
}

// A quarter of the real file is low risk products, notified instead of
// registered: no registration number, and nothing to complain about.
func TestParseMedicamentos_NotifiedProductsAreSkipped(t *testing.T) {
	file := medicamentosHeader +
		"MEDICAMENTO;SIMETICONA;;IFAL INDUSTRIA;;Ativo\n" +
		"MEDICAMENTO;ADVIL 12H;100432105;PF;IBUPROFENO;Ativo\n"

	out, errs, err := anvisa.ParseMedicamentos(latin1(t, file), false)

	require.NoError(t, err)
	assert.Empty(t, errs, "no registration is not an error, it is another kind of product")
	require.Len(t, out.Drugs, 1)
}

func TestParseMedicamentos_CompanyWithoutCNPJ(t *testing.T) {
	file := medicamentosHeader +
		"MEDICAMENTO;RAVUMA;105730020;60659463002992 - ACHÉ LABORATÓRIOS FARMACÊUTICOS S.A;ciprofibrato;Ativo\n"

	out, _, err := anvisa.ParseMedicamentos(latin1(t, file), false)

	require.NoError(t, err)
	require.Len(t, out.Drugs, 1)
	assert.Equal(t, "ACHÉ LABORATÓRIOS FARMACÊUTICOS S.A", out.Drugs[0].Manufacturer)
}

func TestParseMedicamentos_WrongFile(t *testing.T) {
	_, _, err := anvisa.ParseMedicamentos(strings.NewReader("a;b;c\n1;2;3\n"), false)
	assert.ErrorContains(t, err, "not found")
}

func TestParsePrecos(t *testing.T) {
	// The real CMED list opens with a BOM and ~59 lines of legal notice before
	// the header, and writes an empty barcode as a dash.
	file := "\ufeffSecretaria Executiva - CMED;;;;;;\n" +
		strings.Repeat("Aviso legal comprido;;;;;;\n", 50) +
		";;;;;;\n" +
		"SUBSTÂNCIA;REGISTRO;EAN 1;EAN 2;EAN 3;PRODUTO;APRESENTAÇÃO\n" +
		"IBUPROFENO;1929000070034;7896015592752;    -      ;    -      ;ADVIL 12H;600 MG COM REV   LIB PROL CT BL X 10\n" +
		"IBUPROFENO;1929000070069;7896015592769;7896015592776;    -      ;ADVIL 12H;600 MG COM REV LIB PROL CT BL X 20\n" +
		"IBUPROFENO;1929000070107;    -      ;    -      ;    -      ;ADVIL 12H;600 MG CT BL X 30\n"

	packages, errs, err := anvisa.ParsePrecos(strings.NewReader(file))

	require.NoError(t, err)
	assert.Empty(t, errs)
	require.Len(t, packages, 2, "a presentation without a barcode is skipped")

	assert.Equal(t, "1929000070034", packages[0].PresentationRegistration)
	assert.Equal(t, "192900007", packages[0].DrugRegistration, "the drug is the first 9 digits")
	assert.Equal(t, "600 MG COM REV LIB PROL CT BL X 10", packages[0].Description, "spaces are squeezed")
	assert.Equal(t, []string{"7896015592752"}, packages[0].Eans)
	assert.Equal(t, []string{"7896015592769", "7896015592776"}, packages[1].Eans)
}

func TestParsePrecos_RepeatedRows(t *testing.T) {
	file := "REGISTRO;APRESENTACAO;EAN_1\n" +
		"1929000070034;caixa com 10;7896015592752\n" +
		"1929000070034;caixa com 10;7896015592769\n" + // same presentation, another price bracket
		"12345;curto;7896015592783\n"

	packages, errs, err := anvisa.ParsePrecos(strings.NewReader(file))

	require.NoError(t, err)
	require.Len(t, packages, 1, "the repeated presentation is merged")
	assert.Equal(t, []string{"7896015592752", "7896015592769"}, packages[0].Eans)

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, "13 digits")
}

// The real list prints the same barcode on an old and a new registration of the
// same product, and our package_eans lets only one of them have it.
func TestResolveBarcodes(t *testing.T) {
	packages := []importer.PackageRow{
		{Line: 10, DrugRegistration: "101070346", PresentationRegistration: "1010703460032",
			Eans: []string{"7896015592752"}}, // inactive registration, first in the file
		{Line: 20, DrugRegistration: "192900007", PresentationRegistration: "1929000070034",
			Eans: []string{"7896015592752", "7896015592769"}}, // the active one
		{Line: 30, DrugRegistration: "100000001", PresentationRegistration: "1000000010011",
			Eans: []string{"7896015592783"}},
	}
	active := map[string]bool{"192900007": true, "100000001": true}

	out, errs := anvisa.ResolveBarcodes(packages, active)

	require.Len(t, out, 2, "the presentation left with no barcode is dropped")
	assert.Equal(t, "1929000070034", out[0].PresentationRegistration)
	assert.ElementsMatch(t, []string{"7896015592752", "7896015592769"}, out[0].Eans,
		"the active registration takes the shared barcode")
	assert.Equal(t, "1000000010011", out[1].PresentationRegistration)
	require.Len(t, errs, 1)
	assert.Equal(t, 10, errs[0].Line, "the one reported is the loser")
}

func TestResolveBarcodes_NeitherActive(t *testing.T) {
	packages := []importer.PackageRow{
		{Line: 10, DrugRegistration: "101070346", Eans: []string{"7896015592752"}},
		{Line: 20, DrugRegistration: "192900007", Eans: []string{"7896015592752"}},
	}

	out, errs := anvisa.ResolveBarcodes(packages, map[string]bool{})

	require.Len(t, out, 1, "with nothing to tell them apart the first one keeps it")
	assert.Equal(t, 10, out[0].Line)
	require.Len(t, errs, 1)
	assert.Equal(t, 20, errs[0].Line)
}

func TestChunks(t *testing.T) {
	drugs := []importer.DrugRow{
		{RegistrationNumber: "100000001"},
		{RegistrationNumber: "100000002"},
		{RegistrationNumber: "100000003"},
	}
	packages := []importer.PackageRow{
		{DrugRegistration: "100000001", PresentationRegistration: "1000000010011"},
		{DrugRegistration: "100000003", PresentationRegistration: "1000000030011"},
		{DrugRegistration: "999999999", PresentationRegistration: "9999999990011"}, // drug not in the file
	}

	files := anvisa.Chunks(drugs, packages, 2)

	require.Len(t, files, 3)
	// Chunk 1: two drugs, and only the packages of those two.
	assert.Len(t, files[0].Drugs, 2)
	require.Len(t, files[0].Packages, 1)
	assert.Equal(t, "1000000010011", files[0].Packages[0].PresentationRegistration)
	// Chunk 2: the third drug with its own package.
	assert.Len(t, files[1].Drugs, 1)
	require.Len(t, files[1].Packages, 1)
	assert.Equal(t, "1000000030011", files[1].Packages[0].PresentationRegistration)
	// Chunk 3: what is left over, with no drug of its own.
	assert.Empty(t, files[2].Drugs)
	require.Len(t, files[2].Packages, 1)
	assert.Equal(t, "9999999990011", files[2].Packages[0].PresentationRegistration)
}
