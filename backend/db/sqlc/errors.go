package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	ForeignKeyViolation = "23503"
	UniqueViolation     = "23505"
)

// ErrRecordNotFound is what every :one query returns for an empty result.
// Handlers compare against this to answer 404 rather than 500.
var ErrRecordNotFound = pgx.ErrNoRows

var ErrUniqueViolation = &pgconn.PgError{
	Code: UniqueViolation,
}

func ErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func ConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// IsNotFound reports whether err is the "no rows" case.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrRecordNotFound)
}

// IsUniqueViolation reports whether err is a duplicate-key rejection, which is
// how a re-registered email or a second application to the same task surfaces.
func IsUniqueViolation(err error) bool {
	return ErrorCode(err) == UniqueViolation
}
