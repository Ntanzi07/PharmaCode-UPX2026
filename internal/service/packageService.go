package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type PackageService struct {
	queries PackageQuerier
}

func NewPackageService(q PackageQuerier) *PackageService {
	return &PackageService{queries: q}
}

type CreatePackageInput struct {
	RegistrationNumber string
	Ean                string
	Description        string
}

type UpdatePackageInput struct {
	DrugID      int64
	Ean         string
	Description string
}

func (s *PackageService) Create(ctx context.Context, in CreatePackageInput) (int64, error) {
	id, err := s.queries.CreatePackage(ctx, db.CreatePackageParams{
		RegistrationNumber: in.RegistrationNumber,
		Ean:                in.Ean,
		Description:        in.Description,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrDrugNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrDuplicateEAN
		}
		return 0, err
	}
	return id, nil
}

func (s *PackageService) Update(ctx context.Context, id int64, in UpdatePackageInput) error {
	rows, err := s.queries.UpdatePackage(ctx, db.UpdatePackageParams{
		ID:          id,
		DrugID:      in.DrugID,
		Ean:         in.Ean,
		Description: in.Description,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEAN
		}
		return err
	}
	if rows == 0 {
		return ErrPackageNotFound
	}
	return nil
}

func (s *PackageService) Delete(ctx context.Context, id int64) error {
	rows, err := s.queries.DeletePackage(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPackageNotFound
	}
	return nil
}

func (s *PackageService) List(ctx context.Context, limit, offset int32) ([]db.ListPackagesRow, error) {
	return s.queries.ListPackages(ctx, db.ListPackagesParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *PackageService) GetByID(ctx context.Context, id int64) (db.GetPackageByIdRow, error) {
	row, err := s.queries.GetPackageById(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetPackageByIdRow{}, ErrPackageNotFound
	}
	return row, err
}
