package service

import (
	"context"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type DrugQuerier interface {
	CreateDrug(ctx context.Context, arg db.CreateDrugParams) (int64, error)
	UpdateDrug(ctx context.Context, arg db.UpdateDrugParams) (int64, error)
	DeleteDrug(ctx context.Context, id int64) (int64, error)
	ListDrugs(ctx context.Context, arg db.ListDrugsParams) ([]db.ListDrugsRow, error)
	GetDrugByID(ctx context.Context, id int64) (db.GetDrugByIDRow, error)
	SearchIngredients(ctx context.Context, arg db.SearchIngredientsParams) ([]db.SearchIngredientsRow, error)
	GetDrugByEAN(ctx context.Context, ean string) (db.GetDrugByEANRow, error)
	GetSummaryByEAN(ctx context.Context, ean string) (db.GetSummaryByEANRow, error)

	// Active ingredients live in their own tables (see migration 000008).
	UpsertIngredient(ctx context.Context, name string) (int64, error)
	LinkDrugIngredient(ctx context.Context, arg db.LinkDrugIngredientParams) error
	DeleteDrugIngredientsNotIn(ctx context.Context, arg db.DeleteDrugIngredientsNotInParams) error
	ListIngredientsByDrugID(ctx context.Context, drugID int64) ([]db.ListIngredientsByDrugIDRow, error)
	ListDrugsByEANs(ctx context.Context, eans []string) ([]db.ListDrugsByEANsRow, error)

	// Interaction rules between two active ingredients
	UpsertIngredientInteraction(ctx context.Context, arg db.UpsertIngredientInteractionParams) (db.UpsertIngredientInteractionRow, error)
	DeleteIngredientInteraction(ctx context.Context, arg db.DeleteIngredientInteractionParams) (int64, error)
	ListIngredientInteractions(ctx context.Context, arg db.ListIngredientInteractionsParams) ([]db.ListIngredientInteractionsRow, error)
	CountIngredientInteractions(ctx context.Context) (int64, error)
	FindInteractionsBetweenEANs(ctx context.Context, eans []string) ([]db.FindInteractionsBetweenEANsRow, error)
}

type PackageQuerier interface {
	CreatePackage(ctx context.Context, arg db.CreatePackageParams) (int64, error)
	UpdatePackage(ctx context.Context, arg db.UpdatePackageParams) (int64, error)
	DeletePackage(ctx context.Context, id int64) (int64, error)
	ListPackages(ctx context.Context, arg db.ListPackagesParams) ([]db.ListPackagesRow, error)
	GetPackageById(ctx context.Context, id int64) (db.GetPackageByIdRow, error)
}

type SummaryQuerier interface {
	CreateSummary(ctx context.Context, arg db.CreateSummaryParams) (int64, error)
	UpdateSummary(ctx context.Context, arg db.UpdateSummaryParams) (int64, error)
	ReviewSummary(ctx context.Context, arg db.ReviewSummaryParams) (int64, error)
	DeleteSummary(ctx context.Context, id int64) (int64, error)
	ListSummaries(ctx context.Context, arg db.ListSummariesParams) ([]db.ListSummariesRow, error)
	GetSummaryByID(ctx context.Context, id int64) (db.GetSummaryByIDRow, error)
	GetSummaryByDrugID(ctx context.Context, drugID int64) (db.GetSummaryByDrugIDRow, error)
}
