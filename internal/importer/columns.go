// Package importer reads the project's spreadsheet template (.xlsx) and turns
// it into rows the import service can write to the database.
package importer

import (
	"strings"
	"unicode"
)

// Sheet names expected in the template.
const (
	SheetDrugs    = "remedios"
	SheetPackages = "embalagens"
)

type column struct {
	Name     string // header, as it appears in the template
	Required bool
	Help     string // shown in the "instrucoes" sheet
}

// DrugColumns is the "remedios" sheet: one row per drug, with the optional
// simplified leaflet in the same row.
var DrugColumns = []column{
	{"registro_anvisa", true, "Numero da Regularizacao no Bulario (9 digitos). Identifica o remedio."},
	{"nome_comercial", false, "Nome do Produto no Bulario. Ex.: Advil 12h"},
	{"principio_ativo", true, "Principio Ativo. Ex.: ibuprofeno"},
	{"fabricante", true, "Empresa Detentora da Regularizacao"},
	{"para_que_serve", false, "Bula: para que o remedio serve (obrigatorio se preencher a bula)"},
	{"posologia", false, "Bula: como tomar (obrigatorio se preencher a bula)"},
	{"esqueceu_dose", false, "Bula: o que fazer se esquecer de usar"},
	{"advertencias", false, "Bula: advertencias e precaucoes"},
	{"reacoes_adversas", false, "Bula: reacoes adversas"},
	{"interacoes", false, "Bula: interacoes medicamentosas"},
	{"contraindicacoes", false, "Bula: contraindicacoes"},
	{"efeitos_colaterais", false, "Bula: efeitos colaterais"},
	{"quando_procurar_ajuda", false, "Bula: quando procurar ajuda"},
	{"como_funciona", false, "Bula: mecanismo de acao"},
	{"armazenamento", false, "Bula: como guardar"},
	{"fonte_url", false, "Link da bula oficial (obrigatorio se preencher a bula)"},
	{"bula_expediente", false, "Numero do expediente da bula publicada"},
	{"bula_publicada_em", false, "Data de publicacao da bula (AAAA-MM-DD ou DD/MM/AAAA)"},
}

// PackageColumns is the "embalagens" sheet: one row per presentation.
var PackageColumns = []column{
	{"registro_anvisa", true, "Registro do remedio (9 digitos), o mesmo da aba remedios"},
	{"registro_apresentacao", true, "REGISTRO da apresentacao na CMED (13 digitos)"},
	{"descricao", true, "APRESENTACAO da CMED. Ex.: 600 mg, caixa com 20"},
	{"ean_1", true, "Codigo de barras principal (8 a 13 digitos)"},
	{"ean_2", false, "Segundo EAN da mesma caixa, se houver"},
	{"ean_3", false, "Terceiro EAN da mesma caixa, se houver"},
}

// normalizeHeader makes header matching forgiving: ignores case, accents,
// spaces and punctuation, so "Registro Anvisa" matches "registro_anvisa".
func normalizeHeader(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(replaceAccent(r))
		case r == ' ' || r == '_' || r == '-':
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

// accents maps the accented letters we care about to their plain version.
var accents = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

func replaceAccent(r rune) rune {
	if plain, ok := accents[r]; ok {
		return plain
	}
	return r
}
