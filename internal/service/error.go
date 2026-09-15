package service

import "errors"

var (
	ErrDuplicateRegistration = errors.New("registration number already exists")
	ErrDrugNotFound          = errors.New("drug not found")
	ErrPackageNotFound       = errors.New("package not found")
	ErrDuplicateEAN          = errors.New("ean already exists")
)
