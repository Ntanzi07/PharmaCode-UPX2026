package testutil

import (
	"context"
	"sort"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type FakeDrugQuerier struct {
	mu     sync.Mutex
	nextID int64
	drugs  map[int64]db.ListDrugsRow

	eans              map[string]int64
	reviewedSummaries map[int64]string

	CreateErr error
	UpdateErr error
	DeleteErr error
	ListErr   error
	GetErr    error

	LastCreateParams db.CreateDrugParams
	LastUpdateParams db.UpdateDrugParams

	CreateCalls int
	UpdateCalls int
	DeleteCalls int
}

func NewFakeDrugQuerier() *FakeDrugQuerier {
	return &FakeDrugQuerier{
		nextID:            1,
		drugs:             make(map[int64]db.ListDrugsRow),
		eans:              make(map[string]int64),
		reviewedSummaries: make(map[int64]string),
	}
}

func (f *FakeDrugQuerier) SeedDrug(row db.ListDrugsRow) db.ListDrugsRow {
	f.mu.Lock()
	defer f.mu.Unlock()

	if row.ID == 0 {
		row.ID = f.nextID
		f.nextID++
	} else if row.ID >= f.nextID {
		f.nextID = row.ID + 1
	}
	f.drugs[row.ID] = row
	return row
}

func (f *FakeDrugQuerier) SeedEAN(ean string, drugID int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eans[ean] = drugID
}

func (f *FakeDrugQuerier) SeedReviewedSummary(drugID int64, whatIsItFor string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviewedSummaries[drugID] = whatIsItFor
}

func (f *FakeDrugQuerier) CreateDrug(ctx context.Context, arg db.CreateDrugParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.CreateCalls++
	f.LastCreateParams = arg

	if f.CreateErr != nil {
		return 0, f.CreateErr
	}
	for _, d := range f.drugs {
		if d.RegistrationNumber == arg.RegistrationNumber {
			return 0, PgError(CodeUniqueViolation)
		}
	}

	id := f.nextID
	f.nextID++
	f.drugs[id] = db.ListDrugsRow{
		ID:                 id,
		RegistrationNumber: arg.RegistrationNumber,
		BrandName:          arg.BrandName,
		ActiveIngredient:   arg.ActiveIngredient,
		Manufacturer:       arg.Manufacturer,
	}
	return id, nil
}

func (f *FakeDrugQuerier) UpdateDrug(ctx context.Context, arg db.UpdateDrugParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.UpdateCalls++
	f.LastUpdateParams = arg

	if f.UpdateErr != nil {
		return 0, f.UpdateErr
	}

	row, ok := f.drugs[arg.ID]
	if !ok {
		return 0, nil
	}
	for id, d := range f.drugs {
		if id != arg.ID && d.RegistrationNumber == arg.RegistrationNumber {
			return 0, PgError(CodeUniqueViolation)
		}
	}

	row.RegistrationNumber = arg.RegistrationNumber
	row.BrandName = arg.BrandName
	row.ActiveIngredient = arg.ActiveIngredient
	row.Manufacturer = arg.Manufacturer
	f.drugs[arg.ID] = row

	return 1, nil
}

func (f *FakeDrugQuerier) DeleteDrug(ctx context.Context, id int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.DeleteCalls++

	if f.DeleteErr != nil {
		return 0, f.DeleteErr
	}
	if _, ok := f.drugs[id]; !ok {
		return 0, nil
	}
	delete(f.drugs, id)
	return 1, nil
}

func (f *FakeDrugQuerier) ListDrugs(ctx context.Context, arg db.ListDrugsParams) ([]db.ListDrugsRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ListErr != nil {
		return nil, f.ListErr
	}

	ids := make([]int64, 0, len(f.drugs))
	for id := range f.drugs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var out []db.ListDrugsRow
	for _, id := range ids {
		out = append(out, f.drugs[id])
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

func (f *FakeDrugQuerier) GetDrugByEAN(ctx context.Context, ean string) (db.GetDrugByEANRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetDrugByEANRow{}, f.GetErr
	}

	drugID, ok := f.eans[ean]
	if !ok {
		return db.GetDrugByEANRow{}, pgx.ErrNoRows
	}
	d, ok := f.drugs[drugID]
	if !ok {
		return db.GetDrugByEANRow{}, pgx.ErrNoRows
	}

	return db.GetDrugByEANRow{
		Ean:                ean,
		ID:                 d.ID,
		RegistrationNumber: d.RegistrationNumber,
		BrandName:          d.BrandName,
		ActiveIngredient:   d.ActiveIngredient,
		Manufacturer:       d.Manufacturer,
	}, nil
}

func (f *FakeDrugQuerier) GetSummaryByEAN(ctx context.Context, ean string) (db.GetSummaryByEANRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetSummaryByEANRow{}, f.GetErr
	}

	drugID, ok := f.eans[ean]
	if !ok {
		return db.GetSummaryByEANRow{}, pgx.ErrNoRows
	}
	d, ok := f.drugs[drugID]
	if !ok {
		return db.GetSummaryByEANRow{}, pgx.ErrNoRows
	}

	row := db.GetSummaryByEANRow{
		Ean:                ean,
		Description:        "Caixa com 20 comprimidos",
		RegistrationNumber: d.RegistrationNumber,
		BrandName:          d.BrandName,
		ActiveIngredient:   d.ActiveIngredient,
		Manufacturer:       d.Manufacturer,
	}
	if what, ok := f.reviewedSummaries[drugID]; ok {
		row.WhatIsItFor = pgtype.Text{String: what, Valid: true}
		row.Posology = pgtype.Text{String: "1 comprimido a cada 8 horas", Valid: true}
		row.SourceUrl = pgtype.Text{String: "https://consultas.anvisa.gov.br/bula/1", Valid: true}
	}
	return row, nil
}
