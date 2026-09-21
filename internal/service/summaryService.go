package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type SummaryService struct {
	queries SummaryQuerier
}

func NewSummaryService(q SummaryQuerier) *SummaryService {
	return &SummaryService{queries: q}
}

type CreateSummaryInput struct {
	DrugID            int64
	WhatIsItFor       string
	Posology          string
	MissedDose        string
	Warnings          string
	AdverseEffects    string
	DrugInteractions  string
	Contraindications string
	SideEffects       string
	WhenToSeekHelp    string
	MechanismOfAction string
	Storage           string
	SourceURL         string
	// Versão da bula resumida: número do expediente e data de publicação (YYYY-MM-DD)
	LeafletExpedient   string
	LeafletPublishedAt string
}

type UpdateSummaryInput struct {
	WhatIsItFor       string
	Posology          string
	MissedDose        string
	Warnings          string
	AdverseEffects    string
	DrugInteractions  string
	Contraindications string
	SideEffects       string
	WhenToSeekHelp    string
	MechanismOfAction string
	Storage           string
	SourceURL         string
	// Versão da bula resumida: número do expediente e data de publicação (YYYY-MM-DD)
	LeafletExpedient   string
	LeafletPublishedAt string
}

func optionalText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// optionalDate converte "YYYY-MM-DD" em pgtype.Date; string vazia vira NULL.
func optionalDate(s string) (pgtype.Date, error) {
	if s == "" {
		return pgtype.Date{}, nil
	}
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return pgtype.Date{}, ErrInvalidLeafletDate
	}
	return pgtype.Date{Time: d, Valid: true}, nil
}

func (s *SummaryService) Create(ctx context.Context, in CreateSummaryInput) (int64, error) {
	publishedAt, err := optionalDate(in.LeafletPublishedAt)
	if err != nil {
		return 0, err
	}
	id, err := s.queries.CreateSummary(ctx, db.CreateSummaryParams{
		DrugID:             in.DrugID,
		WhatIsItFor:        in.WhatIsItFor,
		Posology:           in.Posology,
		MissedDose:         optionalText(in.MissedDose),
		Warnings:           optionalText(in.Warnings),
		AdverseEffects:     optionalText(in.AdverseEffects),
		DrugInteractions:   optionalText(in.DrugInteractions),
		Contraindications:  optionalText(in.Contraindications),
		SideEffects:        optionalText(in.SideEffects),
		WhenToSeekHelp:     optionalText(in.WhenToSeekHelp),
		MechanismOfAction:  optionalText(in.MechanismOfAction),
		Storage:            optionalText(in.Storage),
		SourceUrl:          in.SourceURL,
		LeafletExpedient:   optionalText(in.LeafletExpedient),
		LeafletPublishedAt: publishedAt,
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
	publishedAt, err := optionalDate(in.LeafletPublishedAt)
	if err != nil {
		return err
	}
	rows, err := s.queries.UpdateSummary(ctx, db.UpdateSummaryParams{
		ID:                 id,
		WhatIsItFor:        in.WhatIsItFor,
		Posology:           in.Posology,
		MissedDose:         optionalText(in.MissedDose),
		Warnings:           optionalText(in.Warnings),
		AdverseEffects:     optionalText(in.AdverseEffects),
		DrugInteractions:   optionalText(in.DrugInteractions),
		Contraindications:  optionalText(in.Contraindications),
		SideEffects:        optionalText(in.SideEffects),
		WhenToSeekHelp:     optionalText(in.WhenToSeekHelp),
		MechanismOfAction:  optionalText(in.MechanismOfAction),
		Storage:            optionalText(in.Storage),
		SourceUrl:          in.SourceURL,
		LeafletExpedient:   optionalText(in.LeafletExpedient),
		LeafletPublishedAt: publishedAt,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSummaryNotFound
	}
	return nil
}

// Review marca a bula como revisada pelo usuário logado (reviewer ou admin).
// Guarda o id dele e o nome no momento da revisão.
func (s *SummaryService) Review(ctx context.Context, id, reviewerID int64, reviewerName string) error {
	rows, err := s.queries.ReviewSummary(ctx, db.ReviewSummaryParams{
		ID:               id,
		ReviewedBy:       optionalText(reviewerName),
		ReviewedByUserID: pgtype.Int8{Int64: reviewerID, Valid: reviewerID != 0},
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
