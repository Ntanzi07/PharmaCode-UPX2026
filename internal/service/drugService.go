package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type DrugService struct {
	queries *db.Queries
}

func NewDrugService(q *db.Queries) *DrugService {
	return &DrugService{queries: q}
}

type CreateDrugInput struct {
	RegistrationNumber string
	BrandName          string
	ActiveIngredient   string
	Manufacturer       string
}

func (s *DrugService) Create(ctx context.Context, in CreateDrugInput) (int64, error) {
	id, err := s.queries.CreateDrug(ctx, db.CreateDrugParams{
		RegistrationNumber: in.RegistrationNumber,
		BrandName: pgtype.Text{
			String: in.BrandName,
			Valid:  in.BrandName != "",
		},
		ActiveIngredient: in.ActiveIngredient,
		Manufacturer:     in.Manufacturer,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return 0, ErrDuplicateRegistration
		}
		return 0, err
	}

	return id, nil
}

func (s *DrugService) Update(ctx context.Context, id int64, in CreateDrugInput) error {
	err := s.queries.UpdateDrug(ctx, db.UpdateDrugParams{
		ID:                 id,
		RegistrationNumber: in.RegistrationNumber,
		BrandName: pgtype.Text{
			String: in.BrandName,
			Valid:  in.BrandName != "",
		},
		ActiveIngredient: in.ActiveIngredient,
		Manufacturer:     in.Manufacturer,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return ErrDuplicateRegistration
		}
	}
	return err
}

func (s *DrugService) Delete(ctx context.Context, id int64) error {
	rows, err := s.queries.DeleteDrug(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDrugNotFound
	}
	return nil
}

func (s *DrugService) GetSummaryByEAN(ctx context.Context, ean string) (db.GetSummaryByEANRow, error) {
	row, err := s.queries.GetSummaryByEAN(ctx, ean)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetSummaryByEANRow{}, ErrDrugNotFound
	}
	return row, err
}

func (s *DrugService) ListDrugs(ctx context.Context, limit, offset int32) ([]db.ListDrugsRow, error) {
	return s.queries.ListDrugs(ctx, db.ListDrugsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *DrugService) GetDrugByEAN(ctx context.Context, ean string) (db.GetDrugByEANRow, error) {
	row, err := s.queries.GetDrugByEAN(ctx, ean)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetDrugByEANRow{}, ErrDrugNotFound
	}
	return row, err
}
