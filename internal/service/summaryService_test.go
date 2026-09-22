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

func validCreateInput() service.CreateSummaryInput {
	return service.CreateSummaryInput{
		DrugID:            1,
		WhatIsItFor:       "Dor e febre",
		Posology:          "1 comprimido a cada 8 horas",
		AdverseEffects:    "Nausea",
		Contraindications: "Alergia ao principio ativo",
		SourceURL:         "https://consultas.anvisa.gov.br/bula/1",
	}
}

func TestSummaryService_Create_Success(t *testing.T) {
	fake := testutil.NewFakeSummaryQuerier()
	svc := service.NewSummaryService(fake)

	id, err := svc.Create(context.Background(), validCreateInput())

	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
	assert.Equal(t, 1, fake.CreateCalls)
}

func TestSummaryService_Create_EmptyOptionalBecomesNull(t *testing.T) {
	fake := testutil.NewFakeSummaryQuerier()
	svc := service.NewSummaryService(fake)

	in := validCreateInput()
	in.Storage = "" // optional, not provided
	in.AdverseEffects = "Nausea"

	_, err := svc.Create(context.Background(), in)
	require.NoError(t, err)

	got := fake.LastCreateParams
	assert.False(t, got.Storage.Valid, "storage vazio deveria virar NULL")
	assert.True(t, got.AdverseEffects.Valid, "adverse_effects preenchido deveria ser NOT NULL")
	assert.Equal(t, "Nausea", got.AdverseEffects.String)
}

func TestSummaryService_Create_ErrorMapping(t *testing.T) {
	tests := []struct {
		name     string
		dbErr    error
		wantErr  error
		wantSame bool
	}{
		{
			name:    "drug_id duplicado vira ErrSummaryAlreadyExists",
			dbErr:   testutil.PgError(testutil.CodeUniqueViolation),
			wantErr: service.ErrSummaryAlreadyExists,
		},
		{
			name:    "FK invalida vira ErrDrugNotFound",
			dbErr:   testutil.PgError(testutil.CodeForeignKeyViolation),
			wantErr: service.ErrDrugNotFound,
		},
		{
			name:     "erro desconhecido passa direto",
			dbErr:    errors.New("connection refused"),
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := testutil.NewFakeSummaryQuerier()
			fake.CreateErr = tt.dbErr
			svc := service.NewSummaryService(fake)

			_, err := svc.Create(context.Background(), validCreateInput())

			require.Error(t, err)
			if tt.wantSame {
				assert.Equal(t, tt.dbErr, err)
				return
			}

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSummaryService_Update(t *testing.T) {
	t.Run("resumo existente é atualizado", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		seeded := fake.Seed(db.GetSummaryByIDRow{DrugID: 1, WhatIsItFor: "antigo"})
		svc := service.NewSummaryService(fake)

		err := svc.Update(context.Background(), seeded.ID, service.UpdateSummaryInput{
			WhatIsItFor: "novo",
			Posology:    "2 comprimidos ao dia",
			SourceURL:   "https://exemplo.com/bula",
		})

		require.NoError(t, err)
		row, err := svc.GetByID(context.Background(), seeded.ID)
		require.NoError(t, err)
		assert.Equal(t, "novo", row.WhatIsItFor)
	})

	t.Run("id inexistente vira ErrSummaryNotFound", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		svc := service.NewSummaryService(fake)

		err := svc.Update(context.Background(), 999, service.UpdateSummaryInput{
			WhatIsItFor: "x", Posology: "y", SourceURL: "z",
		})

		assert.ErrorIs(t, err, service.ErrSummaryNotFound)
	})
}

func TestSummaryService_Review(t *testing.T) {
	t.Run("marca o revisor", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		seeded := fake.Seed(db.GetSummaryByIDRow{DrugID: 1})
		svc := service.NewSummaryService(fake)

		require.NoError(t, svc.Review(context.Background(), seeded.ID, 7, "Farmaceutica Responsavel"))

		assert.Equal(t, "Farmaceutica Responsavel", fake.LastReviewParams.ReviewedBy.String)
		assert.True(t, fake.LastReviewParams.ReviewedBy.Valid)
		assert.Equal(t, int64(7), fake.LastReviewParams.ReviewedByUserID.Int64)
	})

	t.Run("id inexistente vira ErrSummaryNotFound", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		svc := service.NewSummaryService(fake)

		err := svc.Review(context.Background(), 999, 7, "alguem")

		assert.ErrorIs(t, err, service.ErrSummaryNotFound)
	})
}

func TestSummaryService_Delete(t *testing.T) {
	t.Run("remove o resumo", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		seeded := fake.Seed(db.GetSummaryByIDRow{DrugID: 1})
		svc := service.NewSummaryService(fake)

		require.NoError(t, svc.Delete(context.Background(), seeded.ID))

		_, err := svc.GetByID(context.Background(), seeded.ID)
		assert.ErrorIs(t, err, service.ErrSummaryNotFound)
	})

	t.Run("id inexistente vira ErrSummaryNotFound", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		svc := service.NewSummaryService(fake)

		assert.ErrorIs(t, svc.Delete(context.Background(), 999), service.ErrSummaryNotFound)
	})
}

func TestSummaryService_Get(t *testing.T) {
	fake := testutil.NewFakeSummaryQuerier()
	seeded := fake.Seed(db.GetSummaryByIDRow{DrugID: 42, WhatIsItFor: "Dor"})
	svc := service.NewSummaryService(fake)

	t.Run("GetByID encontra", func(t *testing.T) {
		row, err := svc.GetByID(context.Background(), seeded.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(42), row.DrugID)
	})

	t.Run("GetByDrugID encontra", func(t *testing.T) {
		row, err := svc.GetByDrugID(context.Background(), 42)
		require.NoError(t, err)
		assert.Equal(t, "Dor", row.WhatIsItFor)
	})

	t.Run("GetByDrugID sem resultado vira ErrSummaryNotFound", func(t *testing.T) {
		_, err := svc.GetByDrugID(context.Background(), 999)
		assert.ErrorIs(t, err, service.ErrSummaryNotFound)
	})
}

func TestSummaryService_List(t *testing.T) {
	fake := testutil.NewFakeSummaryQuerier()
	for i := int64(1); i <= 5; i++ {
		fake.Seed(db.GetSummaryByIDRow{DrugID: i})
	}
	svc := service.NewSummaryService(fake)

	t.Run("respeita limit e offset", func(t *testing.T) {
		rows, err := svc.List(context.Background(), 2, 1)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.Equal(t, int64(2), rows[0].DrugID)
	})

	t.Run("offset além do fim devolve vazio", func(t *testing.T) {
		rows, err := svc.List(context.Background(), 10, 99)
		require.NoError(t, err)
		assert.Empty(t, rows)
	})
}

func TestSummaryService_NovasSecoesEVersaoDaBula(t *testing.T) {
	t.Run("missed_dose, warnings e versão da bula vão para o banco", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		svc := service.NewSummaryService(fake)

		in := validCreateInput()
		in.MissedDose = "Tome assim que lembrar"
		in.Warnings = "Não use com álcool"
		in.LeafletExpedient = "0123456/24-5"
		in.LeafletPublishedAt = "2024-05-31"

		_, err := svc.Create(context.Background(), in)
		require.NoError(t, err)

		got := fake.LastCreateParams
		assert.Equal(t, "Tome assim que lembrar", got.MissedDose.String)
		assert.Equal(t, "Não use com álcool", got.Warnings.String)
		assert.Equal(t, "0123456/24-5", got.LeafletExpedient.String)
		require.True(t, got.LeafletPublishedAt.Valid)
		assert.Equal(t, "2024-05-31", got.LeafletPublishedAt.Time.Format("2006-01-02"))
	})

	t.Run("data vazia vira NULL", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		svc := service.NewSummaryService(fake)

		_, err := svc.Create(context.Background(), validCreateInput())

		require.NoError(t, err)
		assert.False(t, fake.LastCreateParams.LeafletPublishedAt.Valid)
		assert.False(t, fake.LastCreateParams.MissedDose.Valid)
	})

	t.Run("data em formato errado vira ErrInvalidLeafletDate e não chega no banco", func(t *testing.T) {
		fake := testutil.NewFakeSummaryQuerier()
		svc := service.NewSummaryService(fake)

		in := validCreateInput()
		in.LeafletPublishedAt = "31/05/2024"
		_, err := svc.Create(context.Background(), in)

		assert.ErrorIs(t, err, service.ErrInvalidLeafletDate)
		assert.Zero(t, fake.CreateCalls)

		err = svc.Update(context.Background(), 1, service.UpdateSummaryInput{
			WhatIsItFor: "x", Posology: "y", SourceURL: "z", LeafletPublishedAt: "2024-13-01",
		})
		assert.ErrorIs(t, err, service.ErrInvalidLeafletDate)
		assert.Zero(t, fake.UpdateCalls)
	})
}
