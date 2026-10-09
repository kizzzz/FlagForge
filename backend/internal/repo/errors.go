package repo

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	DuplicateFlagKey  = "DUPLICATE_FLAG_KEY"
	DuplicatePriority = "DUPLICATE_PRIORITY"
	InternalError     = "INTERNAL_ERROR"
)

// Error carries a stable business code while retaining the database cause.
type Error struct {
	Code  string
	Cause error
}

func (e *Error) Error() string { return e.Code }
func (e *Error) Unwrap() error { return e.Cause }

// MapError is the single mapping point for PostgreSQL constraint errors.
// Other database errors are left intact for the caller's internal-error handling.
func MapError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	code := InternalError
	switch pgErr.ConstraintName {
	case "uq_flags_env_key":
		code = DuplicateFlagKey
	case "uq_rules_flag_priority":
		code = DuplicatePriority
	}
	return &Error{Code: code, Cause: err}
}
