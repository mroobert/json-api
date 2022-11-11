package data

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Models will represent a convenient single 'container' which
// can hold and represent all database models.
type Models struct {
	Movies MovieModel
}

func NewModels(db *pgxpool.Pool) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}
