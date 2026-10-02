package testutil

import (
	"context"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
)

// FakeDrugTx satisfies service.DrugTx without a real transaction: the fakes
// keep everything in memory, so there is nothing to commit or roll back.
// A failure inside the function is returned as is, and whatever the fake had
// already written stays written - keep that in mind when reading the tests.
type FakeDrugTx struct {
	Q service.DrugQuerier
}

func NewFakeDrugTx(q service.DrugQuerier) FakeDrugTx {
	return FakeDrugTx{Q: q}
}

func (f FakeDrugTx) Run(ctx context.Context, fn func(q service.DrugQuerier) error) error {
	return fn(f.Q)
}
