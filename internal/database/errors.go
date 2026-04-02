package database

import "strings"

func IsDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint") ||
		strings.Contains(err.Error(), "Violates unique constraint") ||
		strings.Contains(err.Error(), "SQLSTATE 23505")
}

func IsCheckViolationError(err error) bool {
	return strings.Contains(err.Error(), "violates check constraint") ||
		strings.Contains(err.Error(), "SQLSTATE 23514")
}
