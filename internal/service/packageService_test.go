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

func validPackageInput() service.CreatePackageInput {
	return service.CreatePackageInput{
		RegistrationNumber: "1023401230014",
		Eans:               []string{"7891234567890"},
		Description:        "Caixa com 20 comprimidos",
	}
}

func TestPackageService_Create(t *testing.T) {
	t.Run("cria a embalagem ligada ao remédio", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		fake.SeedDrug("1023401230014", 7)
		svc := service.NewPackageService(fake)

		id, err := svc.Create(context.Background(), validPackageInput())

		require.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	t.Run("registration_number inexistente vira ErrDrugNotFound", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		svc := service.NewPackageService(fake)

		_, err := svc.Create(context.Background(), validPackageInput())

		assert.ErrorIs(t, err, service.ErrDrugNotFound)
	})

	t.Run("EAN duplicado vira ErrDuplicateEAN", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		fake.SeedDrug("1023401230014", 7)
		svc := service.NewPackageService(fake)

		_, err := svc.Create(context.Background(), validPackageInput())
		require.NoError(t, err)

		_, err = svc.Create(context.Background(), validPackageInput())

		assert.ErrorIs(t, err, service.ErrDuplicateEAN)
	})

	t.Run("EAN já usado em outra embalagem, mesmo junto com um novo, vira ErrDuplicateEAN", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		fake.SeedDrug("1023401230014", 7)
		fake.SeedPackage(db.ListPackagesRow{DrugID: 7, Eans: []string{"7891234567890"}})
		svc := service.NewPackageService(fake)

		in := validPackageInput()
		in.Eans = []string{"7899999999999", "7891234567890"}
		_, err := svc.Create(context.Background(), in)

		assert.ErrorIs(t, err, service.ErrDuplicateEAN)
	})

	t.Run("presentation_registration repetido vira ErrDuplicatePresentation", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		fake.SeedDrug("1023401230014", 7)
		svc := service.NewPackageService(fake)

		in := validPackageInput()
		in.PresentationRegistration = "1234567890014"
		_, err := svc.Create(context.Background(), in)
		require.NoError(t, err)

		in.Eans = []string{"7899999999999"}
		_, err = svc.Create(context.Background(), in)

		assert.ErrorIs(t, err, service.ErrDuplicatePresentation)
	})

	t.Run("presentation_registration vazio vai como NULL", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		fake.SeedDrug("1023401230014", 7)
		svc := service.NewPackageService(fake)

		_, err := svc.Create(context.Background(), validPackageInput())

		require.NoError(t, err)
		assert.False(t, fake.LastCreateParams.PresentationRegistration.Valid)
	})

	t.Run("erro desconhecido passa direto", func(t *testing.T) {
		boom := errors.New("connection refused")
		fake := testutil.NewFakePackageQuerier()
		fake.CreateErr = boom
		svc := service.NewPackageService(fake)

		_, err := svc.Create(context.Background(), validPackageInput())

		assert.Equal(t, boom, err)
	})
}

func TestPackageService_Update(t *testing.T) {
	t.Run("atualiza a embalagem existente", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		seeded := fake.SeedPackage(db.ListPackagesRow{DrugID: 1, Eans: []string{"7891111111111"}, Description: "antigo"})
		svc := service.NewPackageService(fake)

		err := svc.Update(context.Background(), seeded.ID, service.UpdatePackageInput{
			DrugID:      1,
			Eans:        []string{"7892222222222"},
			Description: "novo",
		})

		require.NoError(t, err)
		assert.Equal(t, "novo", fake.LastUpdateParams.Description)
	})

	// Same gain as in drugs: comes from switching :exec to :execrows.
	t.Run("id inexistente vira ErrPackageNotFound", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		svc := service.NewPackageService(fake)

		err := svc.Update(context.Background(), 999, service.UpdatePackageInput{
			DrugID: 1, Eans: []string{"7891111111111"}, Description: "x",
		})

		assert.ErrorIs(t, err, service.ErrPackageNotFound)
	})

	t.Run("drug_id inexistente (FK) vira ErrDrugNotFound", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		seeded := fake.SeedPackage(db.ListPackagesRow{DrugID: 1, Eans: []string{"7891111111111"}})
		fake.UpdateErr = testutil.PgError(testutil.CodeForeignKeyViolation)
		svc := service.NewPackageService(fake)

		err := svc.Update(context.Background(), seeded.ID, service.UpdatePackageInput{
			DrugID: 999, Eans: []string{"7891111111111"}, Description: "x",
		})

		assert.ErrorIs(t, err, service.ErrDrugNotFound)
	})

	t.Run("EAN de outra embalagem vira ErrDuplicateEAN", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		fake.SeedPackage(db.ListPackagesRow{DrugID: 1, Eans: []string{"7891111111111"}})
		alvo := fake.SeedPackage(db.ListPackagesRow{DrugID: 1, Eans: []string{"7892222222222"}})
		svc := service.NewPackageService(fake)

		err := svc.Update(context.Background(), alvo.ID, service.UpdatePackageInput{
			DrugID: 1, Eans: []string{"7891111111111"}, Description: "x",
		})

		assert.ErrorIs(t, err, service.ErrDuplicateEAN)
	})
}

func TestPackageService_Delete(t *testing.T) {
	t.Run("remove a embalagem", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		seeded := fake.SeedPackage(db.ListPackagesRow{DrugID: 1, Eans: []string{"7891111111111"}})
		svc := service.NewPackageService(fake)

		require.NoError(t, svc.Delete(context.Background(), seeded.ID))
	})

	t.Run("id inexistente vira ErrPackageNotFound", func(t *testing.T) {
		fake := testutil.NewFakePackageQuerier()
		svc := service.NewPackageService(fake)

		assert.ErrorIs(t, svc.Delete(context.Background(), 999), service.ErrPackageNotFound)
	})
}

func TestPackageService_GetAndList(t *testing.T) {
	fake := testutil.NewFakePackageQuerier()
	seeded := fake.SeedPackage(db.ListPackagesRow{DrugID: 3, Eans: []string{"7891111111111"}, Description: "Caixa"})
	fake.SeedPackage(db.ListPackagesRow{DrugID: 3, Eans: []string{"7892222222222"}})
	svc := service.NewPackageService(fake)

	t.Run("GetByID encontra", func(t *testing.T) {
		row, err := svc.GetByID(context.Background(), seeded.ID)
		require.NoError(t, err)
		assert.Equal(t, "Caixa", row.Description)
	})

	t.Run("GetByID sem resultado vira ErrPackageNotFound", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), 999)
		assert.ErrorIs(t, err, service.ErrPackageNotFound)
	})

	t.Run("List respeita limit", func(t *testing.T) {
		rows, err := svc.List(context.Background(), 1, 0)
		require.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}
