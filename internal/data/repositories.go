package data

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Repositories will represent a convenient single 'container' which
// can hold and represent the set of APIs for database access.
type Repositories struct {
	Movies MovieRepository
	Users  UserRepository
}

func NewRepositories(db *pgxpool.Pool) Repositories {
	return Repositories{
		Movies: MovieRepository{DB: db},
		Users:  UserRepository{DB: db},
	}
}
