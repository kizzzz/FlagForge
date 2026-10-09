package repo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapErrorUsesConstraintAndPreservesCause(t *testing.T) {
	for _, tc := range []struct{ constraint, code string }{
		{"uq_flags_env_key", "DUPLICATE_FLAG_KEY"},
		{"uq_rules_flag_priority", "DUPLICATE_PRIORITY"},
		{"flags_pkey", "INTERNAL_ERROR"},
	} {
		t.Run(tc.constraint, func(t *testing.T) {
			cause := &pgconn.PgError{Code: "23505", ConstraintName: tc.constraint}
			got := MapError(fmt.Errorf("wrapped: %w", cause))
			var mapped *Error
			if !errors.As(got, &mapped) || mapped.Code != tc.code {
				t.Fatalf("expected code %s, got %v", tc.code, got)
			}
			if !errors.Is(got, cause) {
				t.Fatal("original PG cause lost")
			}
		})
	}
}

func TestMapErrorDoesNotParseStringsOrMapOtherErrors(t *testing.T) {
	for _, err := range []error{
		nil,
		errors.New("23505 uq_flags_env_key"),
		&pgconn.PgError{Code: "23514", ConstraintName: "uq_flags_env_key"},
	} {
		if got := MapError(err); got != err {
			t.Fatalf("expected passthrough, got %v", got)
		}
	}
}
