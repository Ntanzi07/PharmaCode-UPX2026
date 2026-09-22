package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/testutil"
)

func validDrugInput() service.CreateDrugInput {
	return service.CreateDrugInput{
		RegistrationNumber: "1023401230014",
		BrandName:          "Dipirona Sodica",
		ActiveIngredient:   "Dipirona monoidratada",
		Manufacturer:       "Laboratorio Exemplo",
	}
}

func TestDrugService_Create(t *testing.T) {
	t.Run("cria e devolve o id", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := service.NewDrugService(fake)

		id, err := svc.CreateDrugService(context.Background(), validDrugInput())

		require.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	// brand_name is the only optional column of the drugs table.
	t.Run("brand_name vazio vira NULL", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := service.NewDrugService(fake)

		in := validDrugInput()
		in.BrandName = ""

		_, err := svc.CreateDrugService(context.Background(), in)
		require.NoError(t, err)

		assert.False(t, fake.LastCreateParams.BrandName.Valid,
			"brand_name vazio deveria virar NULL, nao string vazia")
	})

	t.Run("registro duplicado vira ErrDuplicateRegistration", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := service.NewDrugService(fake)

		_, err := svc.CreateDrugService(context.Background(), validDrugInput())
		require.NoError(t, err)

		_, err = svc.CreateDrugService(context.Background(), validDrugInput())

		assert.ErrorIs(t, err, service.ErrDuplicateRegistration)
	})

	t.Run("erro desconhecido passa direto", func(t *testing.T) {
		boom := errors.New("connection refused")
		fake := testutil.NewFakeDrugQuerier()
		fake.CreateErr = boom
		svc := service.NewDrugService(fake)

		_, err := svc.CreateDrugService(context.Background(), validDrugInput())

		assert.Equal(t, boom, err)
	})
}

func TestDrugService_Update(t *testing.T) {
	t.Run("atualiza o remédio existente", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		seeded := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "111", ActiveIngredient: "antigo"})
		svc := service.NewDrugService(fake)

		in := validDrugInput()
		in.ActiveIngredient = "novo"

		require.NoError(t, svc.UpdateDrugService(context.Background(), seeded.ID, in))
		assert.Equal(t, "novo", fake.LastUpdateParams.ActiveIngredient)
	})

	// This case only exists since the query changed from :exec to
	// :execrows. Before, a PUT on a missing id returned success.
	t.Run("id inexistente vira ErrDrugNotFound", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := service.NewDrugService(fake)

		err := svc.UpdateDrugService(context.Background(), 999, validDrugInput())

		assert.ErrorIs(t, err, service.ErrDrugNotFound)
	})

	t.Run("registro de outro remédio vira ErrDuplicateRegistration", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "ja-existe"})
		alvo := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "meu-numero"})
		svc := service.NewDrugService(fake)

		in := validDrugInput()
		in.RegistrationNumber = "ja-existe"

		err := svc.UpdateDrugService(context.Background(), alvo.ID, in)

		assert.ErrorIs(t, err, service.ErrDuplicateRegistration)
	})
}

func TestDrugService_Delete(t *testing.T) {
	t.Run("remove o remédio", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		seeded := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "111"})
		svc := service.NewDrugService(fake)

		require.NoError(t, svc.DeleteDrugService(context.Background(), seeded.ID))
	})

	t.Run("id inexistente vira ErrDrugNotFound", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := service.NewDrugService(fake)

		assert.ErrorIs(t, svc.DeleteDrugService(context.Background(), 999), service.ErrDrugNotFound)
	})
}

func TestDrugService_List(t *testing.T) {
	fake := testutil.NewFakeDrugQuerier()
	for i := 0; i < 5; i++ {
		fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: string(rune('a' + i))})
	}
	svc := service.NewDrugService(fake)

	rows, err := svc.ListDrugsService(context.Background(), 2, 1)

	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

func TestDrugService_GetByEAN(t *testing.T) {
	setup := func() *testutil.FakeDrugQuerier {
		fake := testutil.NewFakeDrugQuerier()
		d := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "111", ActiveIngredient: "Dipirona"})
		fake.SeedEAN("7891234567890", d.ID)
		return fake
	}

	t.Run("GetDrugByEAN encontra", func(t *testing.T) {
		svc := service.NewDrugService(setup())

		row, err := svc.GetDrugByEANService(context.Background(), "7891234567890")

		require.NoError(t, err)
		assert.Equal(t, "Dipirona", row.ActiveIngredient)
	})

	t.Run("EAN inexistente vira ErrDrugNotFound", func(t *testing.T) {
		svc := service.NewDrugService(setup())

		_, err := svc.GetDrugByEANService(context.Background(), "0000000000000")

		assert.ErrorIs(t, err, service.ErrDrugNotFound)
	})

	t.Run("sem resumo revisado devolve linha com campos NULL", func(t *testing.T) {
		svc := service.NewDrugService(setup())

		row, err := svc.GetSummaryByEANService(context.Background(), "7891234567890")

		require.NoError(t, err)
		assert.Equal(t, "Dipirona", row.ActiveIngredient)
		assert.False(t, row.WhatIsItFor.Valid, "sem resumo revisado, what_is_it_for deve ser NULL")
	})

	t.Run("com resumo revisado devolve os campos preenchidos", func(t *testing.T) {
		fake := setup()
		fake.SeedReviewedSummary(1, "Dor e febre")
		svc := service.NewDrugService(fake)

		row, err := svc.GetSummaryByEANService(context.Background(), "7891234567890")

		require.NoError(t, err)
		require.True(t, row.WhatIsItFor.Valid)
		assert.Equal(t, "Dor e febre", row.WhatIsItFor.String)
	})
}
