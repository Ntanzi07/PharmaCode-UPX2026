package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

// DrugTx runs a function inside a database transaction. Creating a drug writes
// to two tables (drugs and drug_ingredients), so it must be all or nothing.
//
// It is an interface so the tests can run the same code against the fakes.
type DrugTx interface {
	Run(ctx context.Context, fn func(q DrugQuerier) error) error
}

// PoolDrugTx is the real implementation, on top of the pgx pool.
type PoolDrugTx struct {
	pool *pgxpool.Pool
}

func NewPoolDrugTx(pool *pgxpool.Pool) PoolDrugTx {
	return PoolDrugTx{pool: pool}
}

func (p PoolDrugTx) Run(ctx context.Context, fn func(q DrugQuerier) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a commit

	if err := fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
