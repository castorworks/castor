package persistence

import (
	"errors"

	"github.com/castorworks/castor/internal/domain/shared"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// pgUniqueViolation SQLSTATE 23505
const pgUniqueViolation = "23505"

// translateError maps GORM storage errors to domain errors at the repository boundary.
func translateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shared.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return shared.ErrDuplicate
	}
	return err
}
