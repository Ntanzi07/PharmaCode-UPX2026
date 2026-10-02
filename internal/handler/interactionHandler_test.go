package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/testutil"
)

// pgText is the pgtype.Text of a filled in value.
func pgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// interactionFixture builds two boxes with different active ingredients plus a
// rule for that pair, so the check has something to find.
func interactionFixture(t *testing.T, withRule bool) *handler.InteractionHandler {
	t.Helper()
	fake := testutil.NewFakeDrugQuerier()
	svc := service.NewDrugService(fake, testutil.NewFakeDrugTx(fake))

	link := func(drugID int64, names ...string) {
		for _, name := range names {
			id, err := fake.UpsertIngredient(context.Background(), name)
			require.NoError(t, err)
			require.NoError(t, fake.LinkDrugIngredient(context.Background(),
				db.LinkDrugIngredientParams{DrugID: drugID, IngredientID: id}))
		}
	}

	advil := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "100000001", BrandName: pgText("Advil 12h")})
	link(advil.ID, "ibuprofeno")
	fake.SeedEAN("7890000000011", advil.ID)

	marevan := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "100000002", BrandName: pgText("Marevan")})
	link(marevan.ID, "varfarina")
	fake.SeedEAN("7890000000028", marevan.ID)

	if withRule {
		_, err := svc.SaveInteractionRule(context.Background(), service.InteractionRuleInput{
			IngredientA:    "ibuprofeno",
			IngredientB:    "varfarina",
			Severity:       service.SeverityHigh,
			Description:    "Juntos aumentam o risco de sangramento.",
			Recommendation: "Não use junto sem orientação médica.",
			SourceURL:      "https://consultas.anvisa.gov.br/#/bulario/",
		})
		require.NoError(t, err)
	}

	return handler.NewInteractionHandler(svc)
}

func post(t *testing.T, h *handler.InteractionHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/interactions", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CheckInteractions(rec, req)
	return rec
}

func decodeReport(t *testing.T, rec *httptest.ResponseRecorder) service.InteractionReport {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code)
	var report service.InteractionReport
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	return report
}

func TestCheckInteractions(t *testing.T) {
	t.Run("par com regra cadastrada devolve o alerta", func(t *testing.T) {
		rec := post(t, interactionFixture(t, true), `{"eans":["7890000000011","7890000000028"]}`)

		report := decodeReport(t, rec)
		require.Len(t, report.Findings, 1)
		finding := report.Findings[0]
		assert.Equal(t, "ibuprofeno", finding.IngredientA)
		assert.Equal(t, "varfarina", finding.IngredientB)
		assert.Equal(t, service.SeverityHigh, finding.Severity)
		assert.Equal(t, service.SeverityHigh, report.WorstSeverity)
		assert.Len(t, report.Drugs, 2)
		assert.Empty(t, report.NotFound)
		// The two drugs of the finding are the two scanned boxes.
		assert.NotEqual(t, finding.DrugA.DrugID, finding.DrugB.DrugID)
		assert.NotEmpty(t, finding.DrugA.BrandName)
		assert.NotEmpty(t, finding.DrugB.BrandName)
	})

	t.Run("sem regra cadastrada o relatório vem vazio", func(t *testing.T) {
		rec := post(t, interactionFixture(t, false), `{"eans":["7890000000011","7890000000028"]}`)

		report := decodeReport(t, rec)
		assert.Empty(t, report.Findings)
		assert.Empty(t, report.WorstSeverity)
		assert.Len(t, report.Drugs, 2)
	})

	t.Run("EAN desconhecido entra em not_found", func(t *testing.T) {
		rec := post(t, interactionFixture(t, true), `{"eans":["7890000000011","7899999999994"]}`)

		report := decodeReport(t, rec)
		assert.Equal(t, []string{"7899999999994"}, report.NotFound)
		assert.Empty(t, report.Findings)
	})

	t.Run("EAN repetido no corpo não conta como duas caixas", func(t *testing.T) {
		rec := post(t, interactionFixture(t, true), `{"eans":["7890000000011"," 7890000000011 "]}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "at least 2")
	})

	t.Run("entradas inválidas devolvem 400", func(t *testing.T) {
		tests := []struct{ name, body string }{
			{"json quebrado", `{"eans":`},
			{"lista vazia", `{"eans":[]}`},
			{"um EAN só", `{"eans":["7890000000011"]}`},
			{"EAN com letra", `{"eans":["789000000001X","7890000000028"]}`},
			{"EAN curto", `{"eans":["123","7890000000028"]}`},
			{"mais de 10 EANs", `{"eans":["7890000000011","7890000000028","7890000000035","7890000000042",
				"7890000000059","7890000000066","7890000000073","7890000000080","7890000000097",
				"7890000000103","7890000000110"]}`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, http.StatusBadRequest, post(t, interactionFixture(t, true), tt.body).Code)
			})
		}
	})
}

// listRules calls GET /interaction-rules and decodes the page.
func listRules(t *testing.T, h *handler.InteractionHandler) []db.ListIngredientInteractionsRow {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/interaction-rules", nil)
	rec := httptest.NewRecorder()
	h.ListInteractionRules(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data []db.ListIngredientInteractionsRow `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Data
}

func TestInteractionRulesHandler(t *testing.T) {
	ruleBody := `{"ingredient_a":"ibuprofeno","ingredient_b":"varfarina","severity":"grave",
		"description":"Risco de sangramento.","source_url":"https://consultas.anvisa.gov.br/#/bulario/"}`

	put := func(h *handler.InteractionHandler, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/interaction-rules", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.SaveInteractionRule(rec, req)
		return rec
	}

	t.Run("cria com 201 e atualiza com 200", func(t *testing.T) {
		h := interactionFixture(t, false)

		assert.Equal(t, http.StatusCreated, put(h, ruleBody).Code)
		assert.Equal(t, http.StatusOK, put(h, ruleBody).Code)
	})

	t.Run("dados inválidos devolvem 400", func(t *testing.T) {
		tests := []struct{ name, body string }{
			{"json quebrado", `{"ingredient_a":`},
			{"gravidade desconhecida", `{"ingredient_a":"a","ingredient_b":"b","severity":"urgente",
				"description":"x","source_url":"https://anvisa.gov.br"}`},
			{"sem descrição", `{"ingredient_a":"a","ingredient_b":"b","severity":"leve",
				"description":"  ","source_url":"https://anvisa.gov.br"}`},
			{"sem fonte", `{"ingredient_a":"a","ingredient_b":"b","severity":"leve","description":"x"}`},
			{"mesmo princípio nos dois lados", `{"ingredient_a":"Ibuprofeno","ingredient_b":"ibuprofeno",
				"severity":"leve","description":"x","source_url":"https://anvisa.gov.br"}`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, http.StatusBadRequest, put(interactionFixture(t, false), tt.body).Code)
			})
		}
	})

	t.Run("lista as regras cadastradas", func(t *testing.T) {
		rules := listRules(t, interactionFixture(t, true))

		require.Len(t, rules, 1)
		assert.Equal(t, "ibuprofeno", rules[0].IngredientA)
		assert.Equal(t, "varfarina", rules[0].IngredientB)
		assert.Equal(t, "grave", rules[0].Severity)
	})

	t.Run("apaga a regra e depois devolve 404", func(t *testing.T) {
		h := interactionFixture(t, true)
		rules := listRules(t, h)
		require.Len(t, rules, 1)

		del := func() *httptest.ResponseRecorder {
			a := strconv.FormatInt(rules[0].IngredientAID, 10)
			b := strconv.FormatInt(rules[0].IngredientBID, 10)
			req := httptest.NewRequest(http.MethodDelete, "/interaction-rules/"+a+"/"+b, nil)
			req.SetPathValue("ingredientA", a)
			req.SetPathValue("ingredientB", b)
			rec := httptest.NewRecorder()
			h.DeleteInteractionRule(rec, req)
			return rec
		}

		assert.Equal(t, http.StatusNoContent, del().Code)
		assert.Equal(t, http.StatusNotFound, del().Code)
	})

	t.Run("id inválido na rota devolve 400", func(t *testing.T) {
		h := interactionFixture(t, true)
		req := httptest.NewRequest(http.MethodDelete, "/interaction-rules/abc/2", nil)
		req.SetPathValue("ingredientA", "abc")
		req.SetPathValue("ingredientB", "2")
		rec := httptest.NewRecorder()
		h.DeleteInteractionRule(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
