package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/handler"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/testutil"
)

func newTestHandler(t *testing.T) (*handler.SummaryHandler, *testutil.FakeSummaryQuerier) {
	t.Helper()
	fake := testutil.NewFakeSummaryQuerier()
	return handler.NewSummaryHandler(service.NewSummaryService(fake)), fake
}

const validBody = `{
	"drug_id": 1,
	"what_is_it_for": "Dor e febre",
	"posology": "1 comprimido a cada 8 horas",
	"source_url": "https://consultas.anvisa.gov.br/bula/1"
}`

func TestCreateSummary(t *testing.T) {
	t.Run("payload válido devolve 201 e o id", func(t *testing.T) {
		h, _ := newTestHandler(t)

		req := httptest.NewRequest(http.MethodPost, "/summaries", strings.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.CreateSummary(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var body map[string]int64
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, int64(1), body["id"])
	})

	// Table-driven de novo: todos os casos de 400 numa tabela so.
	t.Run("payloads inválidos devolvem 400", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"json quebrado", `{"drug_id":`},
			{"sem drug_id", `{"what_is_it_for":"a","posology":"b","source_url":"c"}`},
			{"sem what_is_it_for", `{"drug_id":1,"posology":"b","source_url":"c"}`},
			{"sem posology", `{"drug_id":1,"what_is_it_for":"a","source_url":"c"}`},
			{"sem source_url", `{"drug_id":1,"what_is_it_for":"a","posology":"b"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h, fake := newTestHandler(t)

				req := httptest.NewRequest(http.MethodPost, "/summaries", strings.NewReader(tt.body))
				rec := httptest.NewRecorder()

				h.CreateSummary(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				// Validacao invalida nao pode nem encostar no banco.
				assert.Zero(t, fake.CreateCalls)
			})
		}
	})

	t.Run("remédio sem resumo duplicado devolve 409", func(t *testing.T) {
		h, fake := newTestHandler(t)
		fake.Seed(db.GetSummaryByIDRow{DrugID: 1})

		req := httptest.NewRequest(http.MethodPost, "/summaries", strings.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.CreateSummary(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("drug_id inexistente devolve 404", func(t *testing.T) {
		h, fake := newTestHandler(t)
		fake.CreateErr = testutil.PgError(testutil.CodeForeignKeyViolation)

		req := httptest.NewRequest(http.MethodPost, "/summaries", strings.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.CreateSummary(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestGetSummaryByID(t *testing.T) {
	t.Run("encontrado devolve 200 e o corpo", func(t *testing.T) {
		h, fake := newTestHandler(t)
		seeded := fake.Seed(db.GetSummaryByIDRow{DrugID: 7, WhatIsItFor: "Dor"})

		req := httptest.NewRequest(http.MethodGet, "/summaries/1", nil)
		// SetPathValue simula o que o ServeMux faria ao casar "/summaries/{id}".
		// Assim o teste isola o handler, sem depender do router.
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.GetSummaryByID(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, float64(seeded.DrugID), body["drug_id"])
		assert.Equal(t, "Dor", body["what_is_it_for"])
	})

	t.Run("id não numérico devolve 400", func(t *testing.T) {
		h, _ := newTestHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/summaries/abc", nil)
		req.SetPathValue("id", "abc")
		rec := httptest.NewRecorder()

		h.GetSummaryByID(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newTestHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/summaries/999", nil)
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.GetSummaryByID(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestGetSummaryByDrugID(t *testing.T) {
	h, fake := newTestHandler(t)
	fake.Seed(db.GetSummaryByIDRow{DrugID: 42, WhatIsItFor: "Dor"})

	req := httptest.NewRequest(http.MethodGet, "/summaries/drug/42", nil)
	req.SetPathValue("drugID", "42")
	rec := httptest.NewRecorder()

	h.GetSummaryByDrugID(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(42), body["drug_id"])
}

func TestUpdateSummary(t *testing.T) {
	updateBody := `{"what_is_it_for":"novo","posology":"2x ao dia","source_url":"https://exemplo.com"}`

	t.Run("atualiza e devolve 204 sem corpo", func(t *testing.T) {
		h, fake := newTestHandler(t)
		fake.Seed(db.GetSummaryByIDRow{ID: 1, DrugID: 1})

		req := httptest.NewRequest(http.MethodPut, "/summaries/1", strings.NewReader(updateBody))
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.UpdateSummary(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.String(), "204 nao pode ter corpo")
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newTestHandler(t)

		req := httptest.NewRequest(http.MethodPut, "/summaries/999", strings.NewReader(updateBody))
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.UpdateSummary(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("campo obrigatório faltando devolve 400", func(t *testing.T) {
		h, fake := newTestHandler(t)
		fake.Seed(db.GetSummaryByIDRow{ID: 1, DrugID: 1})

		req := httptest.NewRequest(http.MethodPut, "/summaries/1", strings.NewReader(`{"posology":"x"}`))
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.UpdateSummary(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Zero(t, fake.UpdateCalls)
	})
}

func TestReviewSummary(t *testing.T) {
	t.Run("marca como revisado e devolve 204", func(t *testing.T) {
		h, fake := newTestHandler(t)
		fake.Seed(db.GetSummaryByIDRow{ID: 1, DrugID: 1})

		req := httptest.NewRequest(http.MethodPatch, "/summaries/1/review",
			strings.NewReader(`{"reviewed_by":"Farmaceutica Responsavel"}`))
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.ReviewSummary(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "Farmaceutica Responsavel", fake.LastReviewParams.ReviewedBy.String)
	})

	t.Run("reviewed_by vazio devolve 400", func(t *testing.T) {
		h, fake := newTestHandler(t)
		fake.Seed(db.GetSummaryByIDRow{ID: 1, DrugID: 1})

		req := httptest.NewRequest(http.MethodPatch, "/summaries/1/review",
			strings.NewReader(`{"reviewed_by":""}`))
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.ReviewSummary(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Zero(t, fake.ReviewCalls)
	})
}

func TestDeleteSummary(t *testing.T) {
	t.Run("remove e devolve 204", func(t *testing.T) {
		h, fake := newTestHandler(t)
		fake.Seed(db.GetSummaryByIDRow{ID: 1, DrugID: 1})

		req := httptest.NewRequest(http.MethodDelete, "/summaries/1", nil)
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.DeleteSummary(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newTestHandler(t)

		req := httptest.NewRequest(http.MethodDelete, "/summaries/999", nil)
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.DeleteSummary(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestListSummaries(t *testing.T) {
	t.Run("lista vazia devolve array, não null", func(t *testing.T) {
		h, _ := newTestHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/summaries", nil)
		rec := httptest.NewRecorder()

		h.ListSummaries(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"data":[]`)
	})

	t.Run("aplica o default de limit", func(t *testing.T) {
		h, fake := newTestHandler(t)
		for i := int64(1); i <= 3; i++ {
			fake.Seed(db.GetSummaryByIDRow{DrugID: i})
		}

		req := httptest.NewRequest(http.MethodGet, "/summaries", nil)
		rec := httptest.NewRecorder()

		h.ListSummaries(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var body struct {
			Data   []db.ListSummariesRow `json:"data"`
			Limit  int32                 `json:"limit"`
			Offset int32                 `json:"offset"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Len(t, body.Data, 3)
		assert.Equal(t, int32(20), body.Limit)
		assert.Equal(t, int32(0), body.Offset)
	})

	t.Run("query params inválidos devolvem 400", func(t *testing.T) {
		tests := []struct {
			name string
			url  string
		}{
			{"limit não numérico", "/summaries?limit=abc"},
			{"limit zero", "/summaries?limit=0"},
			{"limit negativo", "/summaries?limit=-1"},
			{"offset negativo", "/summaries?offset=-1"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h, _ := newTestHandler(t)

				req := httptest.NewRequest(http.MethodGet, tt.url, nil)
				rec := httptest.NewRecorder()

				h.ListSummaries(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
			})
		}
	})

	t.Run("limit acima de 100 é limitado a 100", func(t *testing.T) {
		h, _ := newTestHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/summaries?limit=500", nil)
		rec := httptest.NewRecorder()

		h.ListSummaries(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"limit":100`)
	})
}

func TestErroInesperadoDoBancoVira500(t *testing.T) {
	boom := errors.New("connection refused")

	tests := []struct {
		name  string
		setup func(f *testutil.FakeSummaryQuerier)
		call  func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder)
	}{
		{
			name:  "create",
			setup: func(f *testutil.FakeSummaryQuerier) { f.CreateErr = boom },
			call: func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder) {
				h.CreateSummary(rec, httptest.NewRequest(http.MethodPost, "/summaries", strings.NewReader(validBody)))
			},
		},
		{
			name:  "list",
			setup: func(f *testutil.FakeSummaryQuerier) { f.ListErr = boom },
			call: func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder) {
				h.ListSummaries(rec, httptest.NewRequest(http.MethodGet, "/summaries", nil))
			},
		},
		{
			name:  "get by id",
			setup: func(f *testutil.FakeSummaryQuerier) { f.GetErr = boom },
			call: func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodGet, "/summaries/1", nil)
				req.SetPathValue("id", "1")
				h.GetSummaryByID(rec, req)
			},
		},
		{
			name:  "get by drug id",
			setup: func(f *testutil.FakeSummaryQuerier) { f.GetErr = boom },
			call: func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodGet, "/summaries/drug/1", nil)
				req.SetPathValue("drugID", "1")
				h.GetSummaryByDrugID(rec, req)
			},
		},
		{
			name:  "update",
			setup: func(f *testutil.FakeSummaryQuerier) { f.UpdateErr = boom },
			call: func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder) {
				body := `{"what_is_it_for":"a","posology":"b","source_url":"c"}`
				req := httptest.NewRequest(http.MethodPut, "/summaries/1", strings.NewReader(body))
				req.SetPathValue("id", "1")
				h.UpdateSummary(rec, req)
			},
		},
		{
			name:  "review",
			setup: func(f *testutil.FakeSummaryQuerier) { f.ReviewErr = boom },
			call: func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodPatch, "/summaries/1/review", strings.NewReader(`{"reviewed_by":"x"}`))
				req.SetPathValue("id", "1")
				h.ReviewSummary(rec, req)
			},
		},
		{
			name:  "delete",
			setup: func(f *testutil.FakeSummaryQuerier) { f.DeleteErr = boom },
			call: func(h *handler.SummaryHandler, rec *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodDelete, "/summaries/1", nil)
				req.SetPathValue("id", "1")
				h.DeleteSummary(rec, req)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, fake := newTestHandler(t)
			tt.setup(fake)

			rec := httptest.NewRecorder()
			tt.call(h, rec)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.NotContains(t, rec.Body.String(), "connection refused",
				"detalhe interno nao pode vazar na resposta")
		})
	}
}

func TestIDInvalidoDevolve400(t *testing.T) {
	h, _ := newTestHandler(t)

	t.Run("update", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/summaries/abc", strings.NewReader(`{}`))
		req.SetPathValue("id", "abc")
		h.UpdateSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("review", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/summaries/abc/review", strings.NewReader(`{}`))
		req.SetPathValue("id", "abc")
		h.ReviewSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("delete", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/summaries/abc", nil)
		req.SetPathValue("id", "abc")
		h.DeleteSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("json quebrado no review", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/summaries/1/review", strings.NewReader(`{`))
		req.SetPathValue("id", "1")
		h.ReviewSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("json quebrado no update", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/summaries/1", strings.NewReader(`{`))
		req.SetPathValue("id", "1")
		h.UpdateSummary(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("drugID não numérico", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/summaries/drug/abc", nil)
		req.SetPathValue("drugID", "abc")
		h.GetSummaryByDrugID(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
