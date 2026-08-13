package store

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrValidation      = errors.New("validation")
	ErrIntegrity       = errors.New("note integrity check failed")
	ErrConflict        = errors.New("conflict")
	ErrSessionBusy     = errors.New("session busy")
	ErrSessionTerminal = errors.New("session terminal")
)
