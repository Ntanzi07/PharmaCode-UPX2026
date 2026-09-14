package service

import "errors"

var (
	ErrDuplicateRegistration = errors.New("registration number already exists")
	ErrDrugNotFound          = errors.New("drug not found")
)
