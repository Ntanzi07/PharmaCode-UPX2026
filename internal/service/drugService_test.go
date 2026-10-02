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
		ActiveIngredients:  []string{"Dipirona monoidratada"},
		Manufacturer:       "Laboratorio Exemplo",
	}
}

// newDrugService wires the service with the fake querier and a transaction
// that just runs the function (the fakes have nothing to roll back).
func newDrugService(fake *testutil.FakeDrugQuerier) *service.DrugService {
	return service.NewDrugService(fake, testutil.NewFakeDrugTx(fake))
}

// seedIngredients links ingredients to a drug through the fake, the same way
// the service does.
func seedIngredients(fake *testutil.FakeDrugQuerier, drugID int64, names ...string) {
	for _, name := range names {
		id, _ := fake.UpsertIngredient(context.Background(), name)
		_ = fake.LinkDrugIngredient(context.Background(), db.LinkDrugIngredientParams{DrugID: drugID, IngredientID: id})
	}
}

func TestDrugService_Create(t *testing.T) {
	t.Run("cria e devolve o id", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		id, err := svc.CreateDrugService(context.Background(), validDrugInput())

		require.NoError(t, err)
		assert.Equal(t, int64(1), id)
	})

	// brand_name is the only optional column of the drugs table.
	t.Run("brand_name vazio vira NULL", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		in := validDrugInput()
		in.BrandName = ""

		_, err := svc.CreateDrugService(context.Background(), in)
		require.NoError(t, err)

		assert.False(t, fake.LastCreateParams.BrandName.Valid,
			"brand_name vazio deveria virar NULL, nao string vazia")
	})

	t.Run("registro duplicado vira ErrDuplicateRegistration", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		_, err := svc.CreateDrugService(context.Background(), validDrugInput())
		require.NoError(t, err)

		_, err = svc.CreateDrugService(context.Background(), validDrugInput())

		assert.ErrorIs(t, err, service.ErrDuplicateRegistration)
	})

	t.Run("erro desconhecido passa direto", func(t *testing.T) {
		boom := errors.New("connection refused")
		fake := testutil.NewFakeDrugQuerier()
		fake.CreateErr = boom
		svc := newDrugService(fake)

		_, err := svc.CreateDrugService(context.Background(), validDrugInput())

		assert.Equal(t, boom, err)
	})
}

func TestDrugService_Update(t *testing.T) {
	t.Run("atualiza o remédio existente", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		seeded := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "111"})
		svc := newDrugService(fake)

		in := validDrugInput()
		in.ActiveIngredients = []string{"novo"}

		require.NoError(t, svc.UpdateDrugService(context.Background(), seeded.ID, in))
		assert.Equal(t, []string{"novo"}, fake.Ingredients(seeded.ID))
	})

	// This case only exists since the query changed from :exec to
	// :execrows. Before, a PUT on a missing id returned success.
	t.Run("id inexistente vira ErrDrugNotFound", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		err := svc.UpdateDrugService(context.Background(), 999, validDrugInput())

		assert.ErrorIs(t, err, service.ErrDrugNotFound)
	})

	t.Run("registro de outro remédio vira ErrDuplicateRegistration", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "ja-existe"})
		alvo := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "meu-numero"})
		svc := newDrugService(fake)

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
		svc := newDrugService(fake)

		require.NoError(t, svc.DeleteDrugService(context.Background(), seeded.ID))
	})

	t.Run("id inexistente vira ErrDrugNotFound", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		assert.ErrorIs(t, svc.DeleteDrugService(context.Background(), 999), service.ErrDrugNotFound)
	})
}

func TestDrugService_List(t *testing.T) {
	fake := testutil.NewFakeDrugQuerier()
	for i := 0; i < 5; i++ {
		fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: string(rune('a' + i))})
	}
	svc := newDrugService(fake)

	rows, err := svc.ListDrugsService(context.Background(), 2, 1)

	require.NoError(t, err)
	assert.Len(t, rows, 2)
}

func TestDrugService_GetByEAN(t *testing.T) {
	setup := func() *testutil.FakeDrugQuerier {
		fake := testutil.NewFakeDrugQuerier()
		d := fake.SeedDrug(db.ListDrugsRow{RegistrationNumber: "111"})
		seedIngredients(fake, d.ID, "Dipirona")
		fake.SeedEAN("7891234567890", d.ID)
		return fake
	}

	t.Run("GetDrugByEAN encontra", func(t *testing.T) {
		svc := newDrugService(setup())

		row, err := svc.GetDrugByEANService(context.Background(), "7891234567890")

		require.NoError(t, err)
		assert.Equal(t, []string{"Dipirona"}, row.ActiveIngredients)
	})

	t.Run("EAN inexistente vira ErrDrugNotFound", func(t *testing.T) {
		svc := newDrugService(setup())

		_, err := svc.GetDrugByEANService(context.Background(), "0000000000000")

		assert.ErrorIs(t, err, service.ErrDrugNotFound)
	})

	t.Run("sem resumo revisado devolve linha com campos NULL", func(t *testing.T) {
		svc := newDrugService(setup())

		row, err := svc.GetSummaryByEANService(context.Background(), "7891234567890")

		require.NoError(t, err)
		assert.Equal(t, []string{"Dipirona"}, row.ActiveIngredients)
		assert.False(t, row.WhatIsItFor.Valid, "sem resumo revisado, what_is_it_for deve ser NULL")
	})

	t.Run("com resumo revisado devolve os campos preenchidos", func(t *testing.T) {
		fake := setup()
		fake.SeedReviewedSummary(1, "Dor e febre")
		svc := newDrugService(fake)

		row, err := svc.GetSummaryByEANService(context.Background(), "7891234567890")

		require.NoError(t, err)
		require.True(t, row.WhatIsItFor.Valid)
		assert.Equal(t, "Dor e febre", row.WhatIsItFor.String)
	})
}

func TestDrugService_Ingredients(t *testing.T) {
	ctx := context.Background()

	t.Run("associação vira um vínculo por princípio ativo", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		in := validDrugInput()
		in.ActiveIngredients = []string{"dipirona sódica", "mucato de isometepteno", "cafeína"}
		id, err := svc.CreateDrugService(ctx, in)

		require.NoError(t, err)
		assert.ElementsMatch(t, in.ActiveIngredients, fake.Ingredients(id))
	})

	t.Run("mesmo princípio ativo com acento ou caixa diferente é um só", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		in := validDrugInput()
		in.ActiveIngredients = []string{"dipirona sódica"}
		primeiro, err := svc.CreateDrugService(ctx, in)
		require.NoError(t, err)

		outro := validDrugInput()
		outro.RegistrationNumber = "222"
		outro.ActiveIngredients = []string{"DIPIRONA SODICA"}
		segundo, err := svc.CreateDrugService(ctx, outro)
		require.NoError(t, err)

		rowsA, _ := fake.ListIngredientsByDrugID(ctx, primeiro)
		rowsB, _ := fake.ListIngredientsByDrugID(ctx, segundo)
		require.Len(t, rowsA, 1)
		require.Len(t, rowsB, 1)
		assert.Equal(t, rowsA[0].ID, rowsB[0].ID, "os dois remédios apontam para a mesma linha de princípio ativo")
	})

	t.Run("editar troca a lista inteira", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		in := validDrugInput()
		in.ActiveIngredients = []string{"dipirona sódica", "cafeína"}
		id, err := svc.CreateDrugService(ctx, in)
		require.NoError(t, err)

		in.ActiveIngredients = []string{"dipirona sódica", "mucato de isometepteno"}
		require.NoError(t, svc.UpdateDrugService(ctx, id, in))

		assert.ElementsMatch(t, []string{"dipirona sódica", "mucato de isometepteno"}, fake.Ingredients(id),
			"a cafeína saiu da lista, então o vínculo dela foi removido")
	})
}

func TestDrugService_CheckInteractions(t *testing.T) {
	ctx := context.Background()

	// Advil (ibuprofeno) e um anticoagulante: o par tem regra cadastrada.
	// A Neosaldina entra como terceiro remédio, sem regra com os outros.
	setup := func(t *testing.T) (*testutil.FakeDrugQuerier, *service.DrugService) {
		t.Helper()
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		novo := func(registro, marca string, ingredientes []string, ean string) {
			in := validDrugInput()
			in.RegistrationNumber = registro
			in.BrandName = marca
			in.ActiveIngredients = ingredientes
			id, err := svc.CreateDrugService(ctx, in)
			require.NoError(t, err)
			fake.SeedEAN(ean, id)
		}
		novo("192900007", "Advil 12h", []string{"ibuprofeno"}, "7890000000011")
		novo("200000001", "Marevan", []string{"varfarina sódica"}, "7890000000028")
		novo("100000001", "Neosaldina", []string{"dipirona sódica", "cafeína"}, "7890000000035")

		_, err := svc.SaveInteractionRule(ctx, service.InteractionRuleInput{
			IngredientA: "ibuprofeno", IngredientB: "varfarina sódica",
			Severity:    service.SeverityHigh,
			Description: "Juntos aumentam o risco de sangramento.",
			SourceURL:   "https://consultas.anvisa.gov.br/#/bulario/",
		})
		require.NoError(t, err)
		return fake, svc
	}

	t.Run("par com regra cadastrada vira um achado", func(t *testing.T) {
		_, svc := setup(t)

		report, err := svc.CheckInteractions(ctx, []string{"7890000000011", "7890000000028"})

		require.NoError(t, err)
		require.Len(t, report.Findings, 1)
		f := report.Findings[0]
		assert.Equal(t, service.SeverityHigh, f.Severity)
		assert.Equal(t, service.SeverityHigh, report.WorstSeverity)
		assert.ElementsMatch(t, []string{"ibuprofeno", "varfarina sódica"}, []string{f.IngredientA, f.IngredientB})
		assert.ElementsMatch(t, []string{"Advil 12h", "Marevan"}, []string{f.DrugA.BrandName, f.DrugB.BrandName})
		assert.Len(t, report.Drugs, 2)
	})

	t.Run("sem regra cadastrada não acusa nada", func(t *testing.T) {
		_, svc := setup(t)

		report, err := svc.CheckInteractions(ctx, []string{"7890000000011", "7890000000035"})

		require.NoError(t, err)
		assert.Empty(t, report.Findings)
		assert.Empty(t, report.WorstSeverity, "sem achado, não há gravidade")
	})

	t.Run("o mesmo princípio ativo nos dois remédios não é uma interação", func(t *testing.T) {
		fake, svc := setup(t)
		in := validDrugInput()
		in.RegistrationNumber = "300000001"
		in.BrandName = "Ibuprofeno genérico"
		in.ActiveIngredients = []string{"ibuprofeno"}
		id, err := svc.CreateDrugService(ctx, in)
		require.NoError(t, err)
		fake.SeedEAN("7890000000042", id)

		report, err := svc.CheckInteractions(ctx, []string{"7890000000011", "7890000000042"})

		require.NoError(t, err)
		assert.Empty(t, report.Findings)
	})

	t.Run("EAN desconhecido vai para not_found, sem derrubar a consulta", func(t *testing.T) {
		_, svc := setup(t)

		report, err := svc.CheckInteractions(ctx, []string{"7890000000011", "0000000000000"})

		require.NoError(t, err)
		assert.Equal(t, []string{"0000000000000"}, report.NotFound)
		assert.Len(t, report.Drugs, 1)
	})

	t.Run("a pior gravidade entre os achados é a que vale", func(t *testing.T) {
		fake, svc := setup(t)
		_, err := svc.SaveInteractionRule(ctx, service.InteractionRuleInput{
			IngredientA: "ibuprofeno", IngredientB: "cafeína",
			Severity:    service.SeverityLow,
			Description: "Efeito somado leve.",
			SourceURL:   "https://x",
		})
		require.NoError(t, err)
		_ = fake

		report, err := svc.CheckInteractions(ctx, []string{"7890000000011", "7890000000028", "7890000000035"})

		require.NoError(t, err)
		assert.Len(t, report.Findings, 2)
		assert.Equal(t, service.SeverityHigh, report.WorstSeverity)
	})
}

func TestDrugService_InteractionRules(t *testing.T) {
	ctx := context.Background()

	valid := service.InteractionRuleInput{
		IngredientA: "ibuprofeno", IngredientB: "varfarina sódica",
		Severity:    service.SeverityHigh,
		Description: "Juntos aumentam o risco de sangramento.",
		SourceURL:   "https://consultas.anvisa.gov.br/#/bulario/",
	}

	t.Run("cria, atualiza e remove", func(t *testing.T) {
		fake := testutil.NewFakeDrugQuerier()
		svc := newDrugService(fake)

		created, err := svc.SaveInteractionRule(ctx, valid)
		require.NoError(t, err)
		assert.True(t, created)

		// a ordem dos princípios ativos não importa: é o mesmo par
		invertido := valid
		invertido.IngredientA, invertido.IngredientB = valid.IngredientB, valid.IngredientA
		invertido.Severity = service.SeverityModerate
		created, err = svc.SaveInteractionRule(ctx, invertido)
		require.NoError(t, err)
		assert.False(t, created, "mesmo par, só que ao contrário: atualiza em vez de criar")

		rules, err := svc.ListInteractionRules(ctx, 20, 0)
		require.NoError(t, err)
		require.Len(t, rules, 1)
		assert.Equal(t, service.SeverityModerate, rules[0].Severity)

		require.NoError(t, svc.DeleteInteractionRule(ctx, rules[0].IngredientAID, rules[0].IngredientBID))
		n, err := svc.CountInteractionRules(ctx)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("entradas inválidas", func(t *testing.T) {
		tests := []struct {
			name  string
			build func(in service.InteractionRuleInput) service.InteractionRuleInput
			want  error
		}{
			{"gravidade desconhecida", func(in service.InteractionRuleInput) service.InteractionRuleInput {
				in.Severity = "altíssima"
				return in
			}, service.ErrInvalidSeverity},
			{"sem descrição", func(in service.InteractionRuleInput) service.InteractionRuleInput {
				in.Description = "  "
				return in
			}, service.ErrInvalidInteractionText},
			{"sem fonte", func(in service.InteractionRuleInput) service.InteractionRuleInput {
				in.SourceURL = ""
				return in
			}, service.ErrInvalidInteractionSource},
			{"mesmo princípio ativo dos dois lados", func(in service.InteractionRuleInput) service.InteractionRuleInput {
				in.IngredientB = "IBUPROFENO"
				return in
			}, service.ErrInvalidIngredientPair},
			{"princípio ativo vazio", func(in service.InteractionRuleInput) service.InteractionRuleInput {
				in.IngredientB = " "
				return in
			}, service.ErrInvalidIngredientPair},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc := newDrugService(testutil.NewFakeDrugQuerier())
				_, err := svc.SaveInteractionRule(ctx, tt.build(valid))
				assert.ErrorIs(t, err, tt.want)
			})
		}
	})

	t.Run("remover par sem regra vira ErrInteractionRuleNotFound", func(t *testing.T) {
		svc := newDrugService(testutil.NewFakeDrugQuerier())
		assert.ErrorIs(t, svc.DeleteInteractionRule(ctx, 1, 2), service.ErrInteractionRuleNotFound)
	})
}
