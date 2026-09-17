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

func newDrugHandler(t *testing.T) (*handler.DrugHandler, *testutil.FakeDrugQuerier) {
	t.Helper()
	fake := testutil.NewFakeDrugQuerier()
	return handler.NewDrugHandler(service.NewDrugService(fake)), fake
}

const validDrugBody = `{
	"registration_number": "1023401230014",
	"brand_name": "Dipirona Sodica",
	"active_ingredient": "Dipirona monoidratada",
	"manufacturer": "Laboratorio Exemplo"
}`

func TestCreateDrug(t *testing.T) {
	t.Run("payload válido devolve 201 e o id", func(t *testing.T) {
		h, _ := newDrugHandler(t)

		req := httptest.NewRequest(http.MethodPost, "/drugs", strings.NewReader(validDrugBody))
		rec := httptest.NewRecorder()

		h.CreateDrug(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var body map[string]int64
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, int64(1), body["id"])
	})

	t.Run("validação falha não pode inserir no banco", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"sem registration_number", `{"active_ingredient":"a","manufacturer":"b"}`},
			{"sem active_ingredient", `{"registration_number":"1","manufacturer":"b"}`},
			{"sem manufacturer", `{"registration_number":"1","active_ingredient":"a"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h, fake := newDrugHandler(t)

				req := httptest.NewRequest(http.MethodPost, "/drugs", strings.NewReader(tt.body))
				rec := httptest.NewRecorder()

				h.CreateDrug(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Zero(t, fake.CreateCalls,
					"respondeu 400 mas chamou o banco assim mesmo — falta o return")
			})
		}
	})

	t.Run("json quebrado devolve 400", func(t *testing.T) {
		h, fake := newDrugHandler(t)

		req := httptest.NewRequest(http.MethodPost, "/drugs", strings.NewReader(`{"registration`))
		rec := httptest.NewRecorder()

		h.CreateDrug(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Zero(t, fake.CreateCalls)
	})

	t.Run("registro duplicado devolve 409", func(t *testing.T) {
		h, fake := newDrugHandler(t)
		fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "1023401230014"})

		req := httptest.NewRequest(http.MethodPost, "/drugs", strings.NewReader(validDrugBody))
		rec := httptest.NewRecorder()

		h.CreateDrug(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestGetSummaryByEAN(t *testing.T) {
	setup := func(t *testing.T) (*handler.DrugHandler, *testutil.FakeDrugQuerier) {
		h, fake := newDrugHandler(t)
		d := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "111", ActiveIngredient: "Dipirona"})
		fake.SeedEAN("7891234567890", d.ID)
		return h, fake
	}

	t.Run("EAN existente devolve 200", func(t *testing.T) {
		h, fake := setup(t)
		fake.SeedReviewedSummary(1, "Dor e febre")

		req := httptest.NewRequest(http.MethodGet, "/drugs/ean/7891234567890", nil)
		req.SetPathValue("ean", "7891234567890")
		rec := httptest.NewRecorder()

		h.GetSummaryByEAN(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "Dor e febre", body["what_is_it_for"])
	})

	t.Run("EAN inexistente devolve 404, não 500", func(t *testing.T) {
		h, _ := setup(t)

		req := httptest.NewRequest(http.MethodGet, "/drugs/ean/0000000000000", nil)
		req.SetPathValue("ean", "0000000000000")
		rec := httptest.NewRecorder()

		h.GetSummaryByEAN(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("EAN vazio devolve 400", func(t *testing.T) {
		h, _ := setup(t)

		req := httptest.NewRequest(http.MethodGet, "/drugs/ean/", nil)
		req.SetPathValue("ean", "")
		rec := httptest.NewRecorder()

		h.GetSummaryByEAN(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("remédio sem resumo revisado devolve 200 com campos nulos", func(t *testing.T) {
		h, _ := setup(t)

		req := httptest.NewRequest(http.MethodGet, "/drugs/ean/7891234567890", nil)
		req.SetPathValue("ean", "7891234567890")
		rec := httptest.NewRecorder()

		h.GetSummaryByEAN(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "Dipirona", body["active_ingredient"])
		assert.Nil(t, body["what_is_it_for"])
	})

	t.Run("erro inesperado devolve 500", func(t *testing.T) {
		h, fake := setup(t)
		fake.GetErr = errors.New("connection refused")

		req := httptest.NewRequest(http.MethodGet, "/drugs/ean/7891234567890", nil)
		req.SetPathValue("ean", "7891234567890")
		rec := httptest.NewRecorder()

		h.GetSummaryByEAN(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "connection refused")
	})
}

func TestUpdateDrug(t *testing.T) {
	t.Run("atualiza e devolve 204", func(t *testing.T) {
		h, fake := newDrugHandler(t)
		fake.SeedDrug(db.ListDrugsRow{ID: 1, RegistrationNumber: "antigo"})

		req := httptest.NewRequest(http.MethodPut, "/drugs/1", strings.NewReader(validDrugBody))
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.UpdateDrug(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.String())
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newDrugHandler(t)

		req := httptest.NewRequest(http.MethodPut, "/drugs/999", strings.NewReader(validDrugBody))
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.UpdateDrug(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("registro de outro remédio devolve 409", func(t *testing.T) {
		h, fake := newDrugHandler(t)
		fake.SeedDrug(db.ListDrugsRow{ID: 1, RegistrationNumber: "1023401230014"})
		fake.SeedDrug(db.ListDrugsRow{ID: 2, RegistrationNumber: "outro"})

		req := httptest.NewRequest(http.MethodPut, "/drugs/2", strings.NewReader(validDrugBody))
		req.SetPathValue("id", "2")
		rec := httptest.NewRecorder()

		h.UpdateDrug(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("entradas inválidas devolvem 400", func(t *testing.T) {
		tests := []struct {
			name string
			id   string
			body string
		}{
			{"id não numérico", "abc", validDrugBody},
			{"id vazio", "", validDrugBody},
			{"json quebrado", "1", `{"registration`},
			{"campo obrigatório faltando", "1", `{"registration_number":"1"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h, fake := newDrugHandler(t)

				req := httptest.NewRequest(http.MethodPut, "/drugs/x", strings.NewReader(tt.body))
				req.SetPathValue("id", tt.id)
				rec := httptest.NewRecorder()

				h.UpdateDrug(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Zero(t, fake.UpdateCalls)
			})
		}
	})
}

func TestDeleteDrug(t *testing.T) {
	t.Run("remove e devolve 204", func(t *testing.T) {
		h, fake := newDrugHandler(t)
		fake.SeedDrug(db.ListDrugsRow{ID: 1, RegistrationNumber: "111"})

		req := httptest.NewRequest(http.MethodDelete, "/drugs/1", nil)
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.DeleteDrug(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newDrugHandler(t)

		req := httptest.NewRequest(http.MethodDelete, "/drugs/999", nil)
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.DeleteDrug(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("id inválido devolve 400", func(t *testing.T) {
		h, _ := newDrugHandler(t)

		for _, id := range []string{"abc", ""} {
			req := httptest.NewRequest(http.MethodDelete, "/drugs/x", nil)
			req.SetPathValue("id", id)
			rec := httptest.NewRecorder()

			h.DeleteDrug(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("erro inesperado devolve 500", func(t *testing.T) {
		h, fake := newDrugHandler(t)
		fake.DeleteErr = errors.New("connection refused")

		req := httptest.NewRequest(http.MethodDelete, "/drugs/1", nil)
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.DeleteDrug(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestListDrugs(t *testing.T) {
	t.Run("lista vazia devolve array, não null", func(t *testing.T) {
		h, _ := newDrugHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/drugs", nil)
		rec := httptest.NewRecorder()

		h.ListDrugs(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"data":[]`)
	})

	t.Run("aplica os defaults de limit e offset", func(t *testing.T) {
		h, fake := newDrugHandler(t)
		fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "111"})

		req := httptest.NewRequest(http.MethodGet, "/drugs", nil)
		rec := httptest.NewRecorder()

		h.ListDrugs(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"limit":20`)
		assert.Contains(t, rec.Body.String(), `"offset":0`)
	})

	t.Run("query params inválidos devolvem 400", func(t *testing.T) {
		for _, url := range []string{"/drugs?limit=abc", "/drugs?limit=0", "/drugs?offset=-1"} {
			h, _ := newDrugHandler(t)

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			h.ListDrugs(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code, url)
		}
	})

	t.Run("erro inesperado devolve 500", func(t *testing.T) {
		h, fake := newDrugHandler(t)
		fake.ListErr = errors.New("connection refused")

		req := httptest.NewRequest(http.MethodGet, "/drugs", nil)
		rec := httptest.NewRecorder()

		h.ListDrugs(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
