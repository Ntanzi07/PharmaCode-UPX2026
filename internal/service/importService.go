package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
)

// maxImportErrors keeps a badly filled spreadsheet from returning thousands of lines.
const maxImportErrors = 200

// ImportCounts is how many rows a step created and updated.
type ImportCounts struct {
	Created int `json:"created" example:"12"`
	Updated int `json:"updated" example:"3"`
}

// ImportResult is what the panel shows after a preview or an import.
type ImportResult struct {
	// Applied is false on a preview and on anything that failed: nothing was written.
	Applied   bool                `json:"applied"`
	Drugs     ImportCounts        `json:"drugs"`
	Packages  ImportCounts        `json:"packages"`
	Eans      ImportCounts        `json:"eans"`
	Summaries ImportCounts        `json:"summaries"`
	Errors    []importer.RowError `json:"errors"`
}

// ImportService writes a parsed spreadsheet to the database. It needs the pool
// (not only the queries) because everything runs inside one transaction: either
// the whole file is imported or nothing is.
type ImportService struct {
	pool *pgxpool.Pool
}

func NewImportService(pool *pgxpool.Pool) *ImportService {
	return &ImportService{pool: pool}
}

// Run writes the file. With dryRun the transaction is rolled back at the end,
// so the counts are real but nothing is kept. Errors found here (a drug that
// doesn't exist, an EAN that belongs to another package) also roll everything back.
func (s *ImportService) Run(ctx context.Context, file *importer.File, dryRun bool) (ImportResult, error) {
	result := ImportResult{Errors: []importer.RowError{}}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a commit

	q := db.New(tx)
	drugIDs := make(map[string]int64, len(file.Drugs))

	for _, d := range file.Drugs {
		row, err := q.UpsertDrug(ctx, db.UpsertDrugParams{
			RegistrationNumber: d.RegistrationNumber,
			BrandName:          optionalText(d.BrandName),
			ActiveIngredient:   d.ActiveIngredient,
			Manufacturer:       d.Manufacturer,
		})
		if err != nil {
			return result, fmt.Errorf("drug %s (line %d): %w", d.RegistrationNumber, d.Line, err)
		}
		drugIDs[d.RegistrationNumber] = row.ID
		count(&result.Drugs, row.Created)

		if !d.HasSummary {
			continue
		}
		publishedAt, err := optionalDate(d.LeafletPublishedAt)
		if err != nil {
			result.Errors = append(result.Errors, importer.RowError{
				Sheet: importer.SheetDrugs, Line: d.Line, Column: "bula_publicada_em", Message: err.Error(),
			})
			continue
		}
		summary, err := q.UpsertSummary(ctx, db.UpsertSummaryParams{
			DrugID:             row.ID,
			WhatIsItFor:        d.WhatIsItFor,
			Posology:           d.Posology,
			MissedDose:         optionalText(d.MissedDose),
			Warnings:           optionalText(d.Warnings),
			AdverseEffects:     optionalText(d.AdverseEffects),
			DrugInteractions:   optionalText(d.DrugInteractions),
			Contraindications:  optionalText(d.Contraindications),
			SideEffects:        optionalText(d.SideEffects),
			WhenToSeekHelp:     optionalText(d.WhenToSeekHelp),
			MechanismOfAction:  optionalText(d.MechanismOfAction),
			Storage:            optionalText(d.Storage),
			SourceUrl:          d.SourceURL,
			LeafletExpedient:   optionalText(d.LeafletExpedient),
			LeafletPublishedAt: publishedAt,
		})
		if err != nil {
			return result, fmt.Errorf("leaflet of %s (line %d): %w", d.RegistrationNumber, d.Line, err)
		}
		count(&result.Summaries, summary.Created)
	}

	for _, p := range file.Packages {
		drugID, ok := drugIDs[p.DrugRegistration]
		if !ok {
			drugID, err = q.GetDrugIDByRegistration(ctx, p.DrugRegistration)
			if errors.Is(err, pgx.ErrNoRows) {
				result.Errors = append(result.Errors, importer.RowError{
					Sheet: importer.SheetPackages, Line: p.Line, Column: "registro_anvisa",
					Message: "drug not registered: add it to the remedios sheet or register it in the panel first",
				})
				continue
			}
			if err != nil {
				return result, err
			}
			drugIDs[p.DrugRegistration] = drugID
		}

		pkg, err := q.UpsertPackage(ctx, db.UpsertPackageParams{
			DrugID:                   drugID,
			Description:              p.Description,
			PresentationRegistration: optionalText(p.PresentationRegistration),
		})
		if err != nil {
			return result, fmt.Errorf("package %s (line %d): %w", p.PresentationRegistration, p.Line, err)
		}
		count(&result.Packages, pkg.Created)

		for _, ean := range p.Eans {
			ce, err := q.UpsertPackageEAN(ctx, db.UpsertPackageEANParams{PackageID: pkg.ID, Ean: ean})
			if err != nil {
				return result, fmt.Errorf("ean %s (line %d): %w", ean, p.Line, err)
			}
			if ce.PackageID != pkg.ID {
				result.Errors = append(result.Errors, importer.RowError{
					Sheet: importer.SheetPackages, Line: p.Line, Column: "ean",
					Message: "EAN " + ean + " already belongs to another package",
				})
				continue
			}
			count(&result.Eans, ce.Created)
		}
	}

	if len(result.Errors) > maxImportErrors {
		result.Errors = result.Errors[:maxImportErrors]
	}
	if dryRun || len(result.Errors) > 0 {
		return result, nil // the deferred Rollback throws the work away
	}
	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	result.Applied = true
	return result, nil
}

func count(c *ImportCounts, created bool) {
	if created {
		c.Created++
		return
	}
	c.Updated++
}
