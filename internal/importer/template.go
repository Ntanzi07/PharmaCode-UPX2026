package importer

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// TemplateFileName is what the browser saves when the panel downloads the model.
const TemplateFileName = "pharmacode-modelo.xlsx"

// BuildTemplate creates the spreadsheet model: one sheet for drugs (with the
// leaflet), one for packages, and one explaining every column. The example row
// uses Advil 12h, the same drug used in db/seed.
func BuildTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	header, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E3E8EE"}},
	})
	if err != nil {
		return nil, err
	}

	drugExample := []string{
		"192900007", "Advil 12h", "ibuprofeno", "PF Consumer Healthcare Brazil",
		"Alivia dores leves a moderadas e baixa a febre.",
		"1 comprimido a cada 12 horas, no maximo 2 por dia.",
		"Tome assim que lembrar; nunca tome duas doses juntas.",
		"Fale com o medico se tem pressao alta ou problemas no estomago.",
		"Azia, enjoo, dor no estomago.",
		"Evite junto com outros anti-inflamatorios e anticoagulantes.",
		"Nao use se tem alergia a anti-inflamatorios ou ulcera ativa.",
		"Os mais comuns afetam o estomago.",
		"Procure ajuda se houver vomito com sangue ou falta de ar.",
		"Reduz as substancias que o corpo produz na inflamacao.",
		"Temperatura ambiente, longe da umidade.",
		"https://consultas.anvisa.gov.br/#/bulario/q/?numeroRegistro=192900007",
		"0123456/24-5", "2024-05-31",
	}
	packageExamples := [][]string{
		{"192900007", "1929000070034", "600 mg, caixa com 10", "7896015592752", "", ""},
		{"192900007", "1929000070069", "600 mg, caixa com 20", "7896015592769", "", ""},
	}

	if err := writeSheet(f, SheetDrugs, DrugColumns, [][]string{drugExample}, header); err != nil {
		return nil, err
	}
	if err := writeSheet(f, SheetPackages, PackageColumns, packageExamples, header); err != nil {
		return nil, err
	}
	if err := writeInstructions(f, header); err != nil {
		return nil, err
	}

	// Sheet1 is the empty sheet excelize starts with.
	if i, _ := f.GetSheetIndex(SheetDrugs); i >= 0 {
		f.SetActiveSheet(i)
	}
	f.DeleteSheet("Sheet1")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeSheet(f *excelize.File, name string, cols []column, examples [][]string, headerStyle int) error {
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	for i, c := range cols {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		title := c.Name
		if c.Required {
			title += " *"
		}
		if err := f.SetCellStr(name, cell, title); err != nil {
			return err
		}
		col, _ := excelize.ColumnNumberToName(i + 1)
		width := 22.0
		if !c.Required && i > 3 {
			width = 34 // leaflet texts are long
		}
		if err := f.SetColWidth(name, col, col, width); err != nil {
			return err
		}
	}
	last, _ := excelize.ColumnNumberToName(len(cols))
	if err := f.SetCellStyle(name, "A1", last+"1", headerStyle); err != nil {
		return err
	}
	// Every column is text: keeps long registration numbers and EANs from
	// turning into 1,92900007E+08 in Excel.
	text, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		return err
	}
	if err := f.SetColStyle(name, "A:"+last, text); err != nil {
		return err
	}
	if err := f.SetCellStyle(name, "A1", last+"1", headerStyle); err != nil {
		return err
	}

	for r, row := range examples {
		for i, value := range row {
			cell, err := excelize.CoordinatesToCellName(i+1, r+2)
			if err != nil {
				return err
			}
			if err := f.SetCellStr(name, cell, value); err != nil {
				return err
			}
		}
	}
	return f.SetPanes(name, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
}

func writeInstructions(f *excelize.File, headerStyle int) error {
	const name = "instrucoes"
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	rows := [][]string{
		{"aba", "coluna", "obrigatorio", "o que preencher"},
	}
	for _, c := range DrugColumns {
		rows = append(rows, []string{SheetDrugs, c.Name, req(c.Required), c.Help})
	}
	for _, c := range PackageColumns {
		rows = append(rows, []string{SheetPackages, c.Name, req(c.Required), c.Help})
	}
	rows = append(rows,
		[]string{"", "", "", ""},
		[]string{"regra", "linha de exemplo", "", "As linhas de exemplo podem ser apagadas antes de importar."},
		[]string{"regra", "bula", "", "A bula fica na aba remedios. Se preencher qualquer campo dela, para_que_serve, posologia e fonte_url passam a ser obrigatorios."},
		[]string{"regra", "revisao", "", "Bula importada entra como NAO revisada: um farmaceutico precisa revisar no painel para ela aparecer no app."},
		[]string{"regra", "repetidos", "", "Importar o mesmo arquivo de novo atualiza os registros existentes, nao duplica."},
	)
	for r, row := range rows {
		for i, value := range row {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+1)
			if err := f.SetCellStr(name, cell, value); err != nil {
				return err
			}
		}
	}
	if err := f.SetCellStyle(name, "A1", "D1", headerStyle); err != nil {
		return err
	}
	for col, width := range map[string]float64{"A": 14, "B": 26, "C": 12, "D": 90} {
		if err := f.SetColWidth(name, col, col, width); err != nil {
			return fmt.Errorf("column %s: %w", col, err)
		}
	}
	return nil
}

func req(required bool) string {
	if required {
		return "sim"
	}
	return "nao"
}
