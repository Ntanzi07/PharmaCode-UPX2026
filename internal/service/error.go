package service

import "errors"

var (
	ErrDuplicateRegistration = errors.New("registration number already exists")
	ErrDrugNotFound          = errors.New("drug not found")
	ErrPackageNotFound       = errors.New("package not found")
	ErrDuplicateEAN          = errors.New("ean already exists")
	ErrDuplicatePresentation = errors.New("presentation registration already exists")
	ErrInvalidLeafletDate    = errors.New("leaflet_published_at must be YYYY-MM-DD")
	ErrSummaryNotFound       = errors.New("summary not found")
	ErrSummaryAlreadyExists  = errors.New("summary already exists for this drug")

	ErrUserNotFound    = errors.New("user not found")
	ErrDuplicateEmail  = errors.New("email already registered")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidUserName = errors.New("name is required")
	ErrInvalidRole     = errors.New("role must be editor, reviewer or admin")
	ErrLastAdmin       = errors.New("cannot remove the last active admin")
	ErrWrongPassword   = errors.New("current password is wrong")
)
