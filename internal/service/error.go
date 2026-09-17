package service

import "errors"

var (
	ErrDuplicateRegistration = errors.New("registration number already exists")
	ErrDrugNotFound          = errors.New("drug not found")
	ErrPackageNotFound       = errors.New("package not found")
	ErrDuplicateEAN          = errors.New("ean already exists")
	ErrSummaryNotFound       = errors.New("summary not found")
	ErrSummaryAlreadyExists  = errors.New("summary already exists for this drug")
)
