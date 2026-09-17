package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type SummaryService struct {
	queries *db.Queries
}

func NewSummaryService(q *db.Queries) *SummaryService {
	return &SummaryService{queries: q}
}

type CreateSummaryInput struct {
	DrugID            int64
	WhatIsItFor       string
	Posology          string
	AdverseEffects    string
	DrugInteractions  string
	Contraindications string
	SideEffects       string
	WhenToSeekHelp    string
	MechanismOfAction string
	Storage           string
	SourceURL         string
}

type UpdateSummaryInput struct {
	WhatIsItFor       string
	Posology          string
	AdverseEffects    string
	DrugInteractions  string
	Contraindications string
	SideEffects       string
	WhenToSeekHelp    string
	MechanismOfAction string
	Storage           string
	SourceURL         string
}

// optionalText converts an empty string into a NULL column value.
func optionalText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func (s *SummaryService) Create(ctx context.Context, in CreateSummaryInput) (int64, error) {
	id, err := s.queries.CreateSummary(ctx, db.CreateSummaryParams{
		DrugID:            in.DrugID,
		WhatIsItFor:       in.WhatIsItFor,
		Posology:          in.Posology,
		AdverseEffects:    optionalText(in.AdverseEffects),
		DrugInteractions:  optionalText(in.DrugInteractions),
		Contraindications: optionalText(in.Contraindications),
		SideEffects:       optionalText(in.SideEffects),
		WhenToSeekHelp:    optionalText(in.WhenToSeekHelp),
		MechanismOfAction: optionalText(in.MechanismOfAction),
		Storage:           optionalText(in.Storage),
		SourceUrl:         in.SourceURL,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation on drug_id
				return 0, ErrSummaryAlreadyExists
			case "23503": // foreign_key_violation on drug_id
				return 0, ErrDrugNotFound
			}
		}
		return 0, err
	}
	return id, nil
}

func (s *SummaryService) Update(ctx context.Context, id int64, in UpdateSummaryInput) error {
	rows, err := s.queries.UpdateSummary(ctx, db.UpdateSummaryParams{
		ID:                id,
		WhatIsItFor:       in.WhatIsItFor,
		Posology:          in.Posology,
		AdverseEffects:    optionalText(in.AdverseEffects),
		DrugInteractions:  optionalText(in.DrugInteractions),
		Contraindications: optionalText(in.Contraindications),
		SideEffects:       optionalText(in.SideEffects),
		WhenToSeekHelp:    optionalText(in.WhenToSeekHelp),
		MechanismOfAction: optionalText(in.MechanismOfAction),
		Storage:           optionalText(in.Storage),
		SourceUrl:         in.SourceURL,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSummaryNotFound
	}
	return nil
}

func (s *SummaryService) Review(ctx context.Context, id int64, reviewedBy string) error {
	rows, err := s.queries.ReviewSummary(ctx, db.ReviewSummaryParams{
		ID:         id,
		ReviewedBy: optionalText(reviewedBy),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSummaryNotFound
	}
	return nil
}

func (s *SummaryService) Delete(ctx context.Context, id int64) error {
	rows, err := s.queries.DeleteSummary(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSummaryNotFound
	}
	return nil
}

func (s *SummaryService) List(ctx context.Context, limit, offset int32) ([]db.ListSummariesRow, error) {
	return s.queries.ListSummaries(ctx, db.ListSummariesParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *SummaryService) GetByID(ctx context.Context, id int64) (db.GetSummaryByIDRow, error) {
	row, err := s.queries.GetSummaryByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetSummaryByIDRow{}, ErrSummaryNotFound
	}
	return row, err
}

func (s *SummaryService) GetByDrugID(ctx context.Context, drugID int64) (db.GetSummaryByDrugIDRow, error) {
	row, err := s.queries.GetSummaryByDrugID(ctx, drugID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetSummaryByDrugIDRow{}, ErrSummaryNotFound
	}
	return row, err
}
