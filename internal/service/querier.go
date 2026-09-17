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
	GetDrugByEAN(ctx context.Context, ean string) (db.GetDrugByEANRow, error)
	GetSummaryByEAN(ctx context.Context, ean string) (db.GetSummaryByEANRow, error)
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
