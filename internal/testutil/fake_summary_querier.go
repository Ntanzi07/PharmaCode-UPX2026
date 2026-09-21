package testutil

import (
	"context"
	"sort"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

const (
	CodeUniqueViolation     = "23505"
	CodeForeignKeyViolation = "23503"
)

func PgError(code string) error {
	return &pgconn.PgError{Code: code, Message: "fake postgres error " + code}
}

// PgConstraintError é um PgError que também informa qual constraint falhou.
func PgConstraintError(code, constraint string) error {
	return &pgconn.PgError{Code: code, ConstraintName: constraint, Message: "fake postgres error " + code}
}

type FakeSummaryQuerier struct {
	mu     sync.Mutex
	nextID int64
	rows   map[int64]db.GetSummaryByIDRow

	CreateErr error
	UpdateErr error
	ReviewErr error
	DeleteErr error
	ListErr   error
	GetErr    error

	LastCreateParams db.CreateSummaryParams
	LastUpdateParams db.UpdateSummaryParams
	LastReviewParams db.ReviewSummaryParams

	CreateCalls int
	UpdateCalls int
	ReviewCalls int
	DeleteCalls int
}

func NewFakeSummaryQuerier() *FakeSummaryQuerier {
	return &FakeSummaryQuerier{
		nextID: 1,
		rows:   make(map[int64]db.GetSummaryByIDRow),
	}
}

func (f *FakeSummaryQuerier) Seed(row db.GetSummaryByIDRow) db.GetSummaryByIDRow {
	f.mu.Lock()
	defer f.mu.Unlock()

	if row.ID == 0 {
		row.ID = f.nextID
		f.nextID++
	} else if row.ID >= f.nextID {
		f.nextID = row.ID + 1
	}
	f.rows[row.ID] = row
	return row
}

func (f *FakeSummaryQuerier) CreateSummary(ctx context.Context, arg db.CreateSummaryParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.CreateCalls++
	f.LastCreateParams = arg

	if f.CreateErr != nil {
		return 0, f.CreateErr
	}

	for _, row := range f.rows {
		if row.DrugID == arg.DrugID {
			return 0, PgError(CodeUniqueViolation)
		}
	}

	id := f.nextID
	f.nextID++
	f.rows[id] = db.GetSummaryByIDRow{
		ID:                 id,
		DrugID:             arg.DrugID,
		WhatIsItFor:        arg.WhatIsItFor,
		Posology:           arg.Posology,
		MissedDose:         arg.MissedDose,
		Warnings:           arg.Warnings,
		AdverseEffects:     arg.AdverseEffects,
		DrugInteractions:   arg.DrugInteractions,
		Contraindications:  arg.Contraindications,
		SideEffects:        arg.SideEffects,
		WhenToSeekHelp:     arg.WhenToSeekHelp,
		MechanismOfAction:  arg.MechanismOfAction,
		Storage:            arg.Storage,
		SourceUrl:          arg.SourceUrl,
		LeafletExpedient:   arg.LeafletExpedient,
		LeafletPublishedAt: arg.LeafletPublishedAt,
	}
	return id, nil
}

func (f *FakeSummaryQuerier) UpdateSummary(ctx context.Context, arg db.UpdateSummaryParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.UpdateCalls++
	f.LastUpdateParams = arg

	if f.UpdateErr != nil {
		return 0, f.UpdateErr
	}

	row, ok := f.rows[arg.ID]
	if !ok {
		// Zero linhas afetadas: e assim que :execrows sinaliza "nao existe".
		return 0, nil
	}

	row.WhatIsItFor = arg.WhatIsItFor
	row.Posology = arg.Posology
	row.MissedDose = arg.MissedDose
	row.Warnings = arg.Warnings
	row.AdverseEffects = arg.AdverseEffects
	row.DrugInteractions = arg.DrugInteractions
	row.Contraindications = arg.Contraindications
	row.SideEffects = arg.SideEffects
	row.WhenToSeekHelp = arg.WhenToSeekHelp
	row.MechanismOfAction = arg.MechanismOfAction
	row.Storage = arg.Storage
	row.SourceUrl = arg.SourceUrl
	row.LeafletExpedient = arg.LeafletExpedient
	row.LeafletPublishedAt = arg.LeafletPublishedAt
	f.rows[arg.ID] = row

	return 1, nil
}

func (f *FakeSummaryQuerier) ReviewSummary(ctx context.Context, arg db.ReviewSummaryParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.ReviewCalls++
	f.LastReviewParams = arg

	if f.ReviewErr != nil {
		return 0, f.ReviewErr
	}

	row, ok := f.rows[arg.ID]
	if !ok {
		return 0, nil
	}

	row.ReviewedBy = arg.ReviewedBy
	row.ReviewedAt = pgtype.Timestamptz{Valid: true}
	f.rows[arg.ID] = row

	return 1, nil
}

func (f *FakeSummaryQuerier) DeleteSummary(ctx context.Context, id int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.DeleteCalls++

	if f.DeleteErr != nil {
		return 0, f.DeleteErr
	}
	if _, ok := f.rows[id]; !ok {
		return 0, nil
	}
	delete(f.rows, id)
	return 1, nil
}

func (f *FakeSummaryQuerier) ListSummaries(ctx context.Context, arg db.ListSummariesParams) ([]db.ListSummariesRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ListErr != nil {
		return nil, f.ListErr
	}

	ids := make([]int64, 0, len(f.rows))
	for id := range f.rows {
		ids = append(ids, id)
	}
	// Map em Go nao tem ordem garantida; sem ordenar, o teste ficaria flaky.
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var out []db.ListSummariesRow
	for _, id := range ids {
		row := f.rows[id]
		out = append(out, db.ListSummariesRow{
			ID:                 row.ID,
			DrugID:             row.DrugID,
			WhatIsItFor:        row.WhatIsItFor,
			SourceUrl:          row.SourceUrl,
			LeafletExpedient:   row.LeafletExpedient,
			LeafletPublishedAt: row.LeafletPublishedAt,
			ReviewedBy:         row.ReviewedBy,
			ReviewedAt:         row.ReviewedAt,
			UpdatedAt:          row.UpdatedAt,
		})
	}

	offset := int(arg.Offset)
	if offset >= len(out) {
		return nil, nil
	}
	out = out[offset:]
	if limit := int(arg.Limit); limit >= 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeSummaryQuerier) GetSummaryByID(ctx context.Context, id int64) (db.GetSummaryByIDRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetSummaryByIDRow{}, f.GetErr
	}
	row, ok := f.rows[id]
	if !ok {
		return db.GetSummaryByIDRow{}, pgx.ErrNoRows
	}
	return row, nil
}

func (f *FakeSummaryQuerier) GetSummaryByDrugID(ctx context.Context, drugID int64) (db.GetSummaryByDrugIDRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetSummaryByDrugIDRow{}, f.GetErr
	}
	for _, row := range f.rows {
		if row.DrugID == drugID {
			return db.GetSummaryByDrugIDRow(row), nil
		}
	}
	return db.GetSummaryByDrugIDRow{}, pgx.ErrNoRows
}
