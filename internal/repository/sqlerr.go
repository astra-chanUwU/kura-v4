package repository

import (
	"database/sql"
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// NotFound maps a missing SQL row to model.ErrNotFound. Other errors pass through.
func NotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrNotFound) {
		return model.ErrNotFound
	}
	return err
}

// Empty maps a missing SQL row to model.ErrEmpty. Other errors pass through.
func Empty(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, model.ErrEmpty) {
		return model.ErrEmpty
	}
	return err
}
