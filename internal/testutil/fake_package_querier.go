package testutil

import (
	"context"
	"sort"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type FakePackageQuerier struct {
	mu     sync.Mutex
	nextID int64
	pkgs   map[int64]db.ListPackagesRow

	registrations map[string]int64 // registration_number -> drug id

	CreateErr error
	UpdateErr error
	DeleteErr error
	ListErr   error
	GetErr    error

	LastCreateParams db.CreatePackageParams
	LastUpdateParams db.UpdatePackageParams

	CreateCalls int
	UpdateCalls int
	DeleteCalls int
}

func NewFakePackageQuerier() *FakePackageQuerier {
	return &FakePackageQuerier{
		nextID:        1,
		pkgs:          make(map[int64]db.ListPackagesRow),
		registrations: make(map[string]int64),
	}
}

func (f *FakePackageQuerier) SeedDrug(registrationNumber string, drugID int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registrations[registrationNumber] = drugID
}

func (f *FakePackageQuerier) SeedPackage(row db.ListPackagesRow) db.ListPackagesRow {
	f.mu.Lock()
	defer f.mu.Unlock()

	if row.ID == 0 {
		row.ID = f.nextID
		f.nextID++
	} else if row.ID >= f.nextID {
		f.nextID = row.ID + 1
	}
	f.pkgs[row.ID] = row
	return row
}

func (f *FakePackageQuerier) CreatePackage(ctx context.Context, arg db.CreatePackageParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.CreateCalls++
	f.LastCreateParams = arg

	if f.CreateErr != nil {
		return 0, f.CreateErr
	}

	drugID, ok := f.registrations[arg.RegistrationNumber]
	if !ok {
		return 0, pgx.ErrNoRows
	}
	if err := f.checkUnique(0, arg.Eans, arg.PresentationRegistration); err != nil {
		return 0, err
	}

	id := f.nextID
	f.nextID++
	f.pkgs[id] = db.ListPackagesRow{
		ID:                       id,
		DrugID:                   drugID,
		Eans:                     append([]string(nil), arg.Eans...),
		Description:              arg.Description,
		PresentationRegistration: arg.PresentationRegistration,
	}
	return id, nil
}

func (f *FakePackageQuerier) UpdatePackage(ctx context.Context, arg db.UpdatePackageParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.UpdateCalls++
	f.LastUpdateParams = arg

	if f.UpdateErr != nil {
		return 0, f.UpdateErr
	}

	row, ok := f.pkgs[arg.ID]
	if !ok {
		return 0, nil
	}
	if err := f.checkUnique(arg.ID, arg.Eans, arg.PresentationRegistration); err != nil {
		return 0, err
	}

	row.DrugID = arg.DrugID
	row.Eans = append([]string(nil), arg.Eans...)
	row.Description = arg.Description
	row.PresentationRegistration = arg.PresentationRegistration
	f.pkgs[arg.ID] = row

	return 1, nil
}

// checkUnique mimics the database UNIQUE constraints: an EAN can belong to only one
// package, and the presentation registration is unique too. Ignores the "self" package.
func (f *FakePackageQuerier) checkUnique(self int64, eans []string, presentation pgtype.Text) error {
	for id, p := range f.pkgs {
		if id == self {
			continue
		}
		for _, existing := range p.Eans {
			for _, e := range eans {
				if existing == e {
					return PgConstraintError(CodeUniqueViolation, "package_eans_ean_key")
				}
			}
		}
		if presentation.Valid && p.PresentationRegistration.Valid &&
			p.PresentationRegistration.String == presentation.String {
			return PgConstraintError(CodeUniqueViolation, "packages_presentation_registration_key")
		}
	}
	return nil
}

func (f *FakePackageQuerier) DeletePackage(ctx context.Context, id int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.DeleteCalls++

	if f.DeleteErr != nil {
		return 0, f.DeleteErr
	}
	if _, ok := f.pkgs[id]; !ok {
		return 0, nil
	}
	delete(f.pkgs, id)
	return 1, nil
}

func (f *FakePackageQuerier) ListPackages(ctx context.Context, arg db.ListPackagesParams) ([]db.ListPackagesRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ListErr != nil {
		return nil, f.ListErr
	}

	ids := make([]int64, 0, len(f.pkgs))
	for id := range f.pkgs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var out []db.ListPackagesRow
	for _, id := range ids {
		out = append(out, f.pkgs[id])
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

func (f *FakePackageQuerier) GetPackageById(ctx context.Context, id int64) (db.GetPackageByIdRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.GetErr != nil {
		return db.GetPackageByIdRow{}, f.GetErr
	}
	row, ok := f.pkgs[id]
	if !ok {
		return db.GetPackageByIdRow{}, pgx.ErrNoRows
	}
	return db.GetPackageByIdRow(row), nil
}
