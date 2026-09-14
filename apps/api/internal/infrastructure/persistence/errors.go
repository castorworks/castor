package persistence

import (
	"errors"

	"github.com/castorworks/castor/internal/domain/shared"
	"gorm.io/gorm"
)

// translateError maps GORM storage errors to domain errors at the repository boundary.
func translateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return shared.ErrNotFound
	}
	return err
}
