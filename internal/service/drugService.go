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
	queries DrugQuerier
	tx      DrugTx
}

func NewDrugService(q DrugQuerier, tx DrugTx) *DrugService {
	return &DrugService{queries: q, tx: tx}
}

type CreateDrugInput struct {
	RegistrationNumber string
	BrandName          string
	// ActiveIngredients: one entry per ingredient. A combination drug such as
	// Neosaldina has three.
	ActiveIngredients []string
	Manufacturer      string
}

func (s *DrugService) CreateDrugService(ctx context.Context, in CreateDrugInput) (int64, error) {
	var id int64
	err := s.tx.Run(ctx, func(q DrugQuerier) error {
		var err error
		id, err = q.CreateDrug(ctx, db.CreateDrugParams{
			RegistrationNumber: in.RegistrationNumber,
			BrandName: pgtype.Text{
				String: in.BrandName,
				Valid:  in.BrandName != "",
			},
			Manufacturer: in.Manufacturer,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrDuplicateRegistration
			}
			return err
		}
		return syncIngredients(ctx, q, id, in.ActiveIngredients)
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// syncIngredients makes the drug's links match the list that came in: every
// ingredient is created (or found) by its normalized name, linked, and the
// links that are no longer in the list are removed.
func syncIngredients(ctx context.Context, q DrugQuerier, drugID int64, names []string) error {
	ids := make([]int64, 0, len(names))
	for _, name := range names {
		ingredientID, err := q.UpsertIngredient(ctx, name)
		if err != nil {
			return err
		}
		if err := q.LinkDrugIngredient(ctx, db.LinkDrugIngredientParams{
			DrugID:       drugID,
			IngredientID: ingredientID,
		}); err != nil {
			return err
		}
		ids = append(ids, ingredientID)
	}
	return q.DeleteDrugIngredientsNotIn(ctx, db.DeleteDrugIngredientsNotInParams{
		DrugID:        drugID,
		IngredientIds: ids,
	})
}

func (s *DrugService) UpdateDrugService(ctx context.Context, id int64, in CreateDrugInput) error {
	return s.tx.Run(ctx, func(q DrugQuerier) error {
		rows, err := q.UpdateDrug(ctx, db.UpdateDrugParams{
			ID:                 id,
			RegistrationNumber: in.RegistrationNumber,
			BrandName: pgtype.Text{
				String: in.BrandName,
				Valid:  in.BrandName != "",
			},
			Manufacturer: in.Manufacturer,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrDuplicateRegistration
			}
			return err
		}
		if rows == 0 {
			return ErrDrugNotFound
		}
		return syncIngredients(ctx, q, id, in.ActiveIngredients)
	})
}

func (s *DrugService) DeleteDrugService(ctx context.Context, id int64) error {
	rows, err := s.queries.DeleteDrug(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDrugNotFound
	}
	return nil
}

func (s *DrugService) ListDrugsService(ctx context.Context, limit, offset int32) ([]db.ListDrugsRow, error) {
	return s.queries.ListDrugs(ctx, db.ListDrugsParams{
		Limit:  limit,
		Offset: offset,
	})
}

// ListIngredients returns the active ingredients of one drug.
func (s *DrugService) ListIngredients(ctx context.Context, drugID int64) ([]db.ListIngredientsByDrugIDRow, error) {
	return s.queries.ListIngredientsByDrugID(ctx, drugID)
}

func (s *DrugService) GetSummaryByEANService(ctx context.Context, ean string) (db.GetSummaryByEANRow, error) {
	row, err := s.queries.GetSummaryByEAN(ctx, ean)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetSummaryByEANRow{}, ErrDrugNotFound
	}
	return row, err
}

func (s *DrugService) GetDrugByEANService(ctx context.Context, ean string) (db.GetDrugByEANRow, error) {
	row, err := s.queries.GetDrugByEAN(ctx, ean)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetDrugByEANRow{}, ErrDrugNotFound
	}
	return row, err
}
