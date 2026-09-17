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

func newPackageHandler(t *testing.T) (*handler.PackageHandler, *testutil.FakePackageQuerier) {
	t.Helper()
	fake := testutil.NewFakePackageQuerier()
	return handler.NewPackageHandler(service.NewPackageService(fake)), fake
}

const validPackageBody = `{
	"registration_number": "1023401230014",
	"ean": "7891234567890",
	"description": "Caixa com 20 comprimidos"
}`

func TestCreatePackage(t *testing.T) {
	t.Run("payload válido devolve 201 e o id", func(t *testing.T) {
		h, fake := newPackageHandler(t)
		fake.SeedDrug("1023401230014", 7)

		req := httptest.NewRequest(http.MethodPost, "/packages", strings.NewReader(validPackageBody))
		rec := httptest.NewRecorder()

		h.CreatePackage(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var body map[string]int64
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, int64(1), body["id"])
	})

	t.Run("remédio inexistente devolve 404", func(t *testing.T) {
		h, _ := newPackageHandler(t)

		req := httptest.NewRequest(http.MethodPost, "/packages", strings.NewReader(validPackageBody))
		rec := httptest.NewRecorder()

		h.CreatePackage(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("EAN duplicado devolve 409", func(t *testing.T) {
		h, fake := newPackageHandler(t)
		fake.SeedDrug("1023401230014", 7)
		fake.SeedPackage(db.ListPackagesRow{DrugID: 7, Ean: "7891234567890"})

		req := httptest.NewRequest(http.MethodPost, "/packages", strings.NewReader(validPackageBody))
		rec := httptest.NewRecorder()

		h.CreatePackage(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("payloads inválidos devolvem 400", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"json quebrado", `{"ean":`},
			{"sem registration_number", `{"ean":"7891234567890","description":"x"}`},
			{"sem ean", `{"registration_number":"1","description":"x"}`},
			{"sem description", `{"registration_number":"1","ean":"7891234567890"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h, fake := newPackageHandler(t)

				req := httptest.NewRequest(http.MethodPost, "/packages", strings.NewReader(tt.body))
				rec := httptest.NewRecorder()

				h.CreatePackage(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Zero(t, fake.CreateCalls)
			})
		}
	})
}

func TestGetPackageByID(t *testing.T) {
	t.Run("encontrado devolve 200", func(t *testing.T) {
		h, fake := newPackageHandler(t)
		fake.SeedPackage(db.ListPackagesRow{ID: 1, DrugID: 3, Ean: "7891234567890", Description: "Caixa"})

		req := httptest.NewRequest(http.MethodGet, "/packages/1", nil)
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.GetPackageByID(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "7891234567890", body["ean"])
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newPackageHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/packages/999", nil)
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.GetPackageByID(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("id não numérico devolve 400", func(t *testing.T) {
		h, _ := newPackageHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/packages/abc", nil)
		req.SetPathValue("id", "abc")
		rec := httptest.NewRecorder()

		h.GetPackageByID(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUpdatePackage(t *testing.T) {
	updateBody := `{"drug_id":1,"ean":"7892222222222","description":"novo"}`

	t.Run("atualiza e devolve 204", func(t *testing.T) {
		h, fake := newPackageHandler(t)
		fake.SeedPackage(db.ListPackagesRow{ID: 1, DrugID: 1, Ean: "7891111111111"})

		req := httptest.NewRequest(http.MethodPut, "/packages/1", strings.NewReader(updateBody))
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.UpdatePackage(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newPackageHandler(t)

		req := httptest.NewRequest(http.MethodPut, "/packages/999", strings.NewReader(updateBody))
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.UpdatePackage(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("EAN de outra embalagem devolve 409", func(t *testing.T) {
		h, fake := newPackageHandler(t)
		fake.SeedPackage(db.ListPackagesRow{ID: 1, DrugID: 1, Ean: "7892222222222"})
		fake.SeedPackage(db.ListPackagesRow{ID: 2, DrugID: 1, Ean: "7891111111111"})

		req := httptest.NewRequest(http.MethodPut, "/packages/2", strings.NewReader(updateBody))
		req.SetPathValue("id", "2")
		rec := httptest.NewRecorder()

		h.UpdatePackage(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("entradas inválidas devolvem 400", func(t *testing.T) {
		tests := []struct {
			name string
			id   string
			body string
		}{
			{"id não numérico", "abc", updateBody},
			{"json quebrado", "1", `{"ean":`},
			{"sem drug_id", "1", `{"ean":"7891111111111","description":"x"}`},
			{"sem ean", "1", `{"drug_id":1,"description":"x"}`},
			{"sem description", "1", `{"drug_id":1,"ean":"7891111111111"}`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h, fake := newPackageHandler(t)

				req := httptest.NewRequest(http.MethodPut, "/packages/x", strings.NewReader(tt.body))
				req.SetPathValue("id", tt.id)
				rec := httptest.NewRecorder()

				h.UpdatePackage(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				assert.Zero(t, fake.UpdateCalls)
			})
		}
	})
}

func TestDeletePackage(t *testing.T) {
	t.Run("remove e devolve 204", func(t *testing.T) {
		h, fake := newPackageHandler(t)
		fake.SeedPackage(db.ListPackagesRow{ID: 1, DrugID: 1, Ean: "7891111111111"})

		req := httptest.NewRequest(http.MethodDelete, "/packages/1", nil)
		req.SetPathValue("id", "1")
		rec := httptest.NewRecorder()

		h.DeletePackage(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("id inexistente devolve 404", func(t *testing.T) {
		h, _ := newPackageHandler(t)

		req := httptest.NewRequest(http.MethodDelete, "/packages/999", nil)
		req.SetPathValue("id", "999")
		rec := httptest.NewRecorder()

		h.DeletePackage(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestListPackages(t *testing.T) {
	t.Run("lista vazia devolve array, não null", func(t *testing.T) {
		h, _ := newPackageHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/packages", nil)
		rec := httptest.NewRecorder()

		h.ListPackages(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"data":[]`)
	})

	t.Run("limit acima de 100 é limitado a 100", func(t *testing.T) {
		h, _ := newPackageHandler(t)

		req := httptest.NewRequest(http.MethodGet, "/packages?limit=500", nil)
		rec := httptest.NewRecorder()

		h.ListPackages(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"limit":100`)
	})

	t.Run("query params inválidos devolvem 400", func(t *testing.T) {
		for _, url := range []string{"/packages?limit=abc", "/packages?limit=-5", "/packages?offset=-1"} {
			h, _ := newPackageHandler(t)

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			h.ListPackages(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code, url)
		}
	})

	t.Run("erro inesperado devolve 500", func(t *testing.T) {
		h, fake := newPackageHandler(t)
		fake.ListErr = errors.New("connection refused")

		req := httptest.NewRequest(http.MethodGet, "/packages", nil)
		rec := httptest.NewRecorder()

		h.ListPackages(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "connection refused")
	})
}
