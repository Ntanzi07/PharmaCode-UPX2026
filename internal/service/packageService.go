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
	RegistrationNumber       string
	Eans                     []string
	Description              string
	PresentationRegistration string
}

type UpdatePackageInput struct {
	DrugID                   int64
	Eans                     []string
	Description              string
	PresentationRegistration string
}

// Name of the UNIQUE constraint Postgres generates for packages.presentation_registration.
const presentationRegistrationConstraint = "packages_presentation_registration_key"

// packageWriteError maps Postgres errors from package inserts/updates.
func packageWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation: duplicate EAN or presentation registration
			if pgErr.ConstraintName == presentationRegistrationConstraint {
				return ErrDuplicatePresentation
			}
			return ErrDuplicateEAN
		case "23503": // foreign_key_violation: drug_id doesn't exist
			return ErrDrugNotFound
		}
	}
	return err
}

func (s *PackageService) Create(ctx context.Context, in CreatePackageInput) (int64, error) {
	id, err := s.queries.CreatePackage(ctx, db.CreatePackageParams{
		RegistrationNumber:       in.RegistrationNumber,
		Eans:                     in.Eans,
		Description:              in.Description,
		PresentationRegistration: optionalText(in.PresentationRegistration),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrDrugNotFound
		}
		return 0, packageWriteError(err)
	}
	return id, nil
}

func (s *PackageService) Update(ctx context.Context, id int64, in UpdatePackageInput) error {
	rows, err := s.queries.UpdatePackage(ctx, db.UpdatePackageParams{
		ID:                       id,
		DrugID:                   in.DrugID,
		Eans:                     in.Eans,
		Description:              in.Description,
		PresentationRegistration: optionalText(in.PresentationRegistration),
	})
	if err != nil {
		return packageWriteError(err)
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
