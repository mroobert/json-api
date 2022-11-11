package data

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mroobert/json-api/internal/validator"
)

//go:embed queries/insert.sql
var insertSQL string

//go:embed queries/read.sql
var readSQL string

//go:embed queries/update.sql
var updateSQL string

//go:embed queries/delete.sql
var deleteSQL string

type (
	Movie struct {
		ID        int64     `json:"id"`                // Unique integer ID for the movie
		CreatedAt time.Time `json:"-"`                 // Timestamp for when the movie is added to our database
		Title     string    `json:"title"`             // Movie title
		Year      int32     `json:"year,omitempty"`    // Movie release year
		Runtime   Runtime   `json:"runtime,omitempty"` // Movie runtime (in minutes)
		Genres    []string  `json:"genres,omitempty"`  // Slice of genres for the movie (romance, comedy, etc.)
		Version   int32     `json:"version"`           // The version number starts at 1 and will be incremented each time the movie information is updated
	}

	MovieModel struct {
		DB *pgxpool.Pool
	}

	NewMovie struct {
		Title   string   `json:"title"`
		Year    int32    `json:"year"`
		Runtime Runtime  `json:"runtime"`
		Genres  []string `json:"genres"`
	}

	UpdateMovie struct {
		Title   *string  `json:"title"`
		Year    *int32   `json:"year"`
		Runtime *Runtime `json:"runtime"`
		Genres  []string `json:"genres"`
	}
)

func (m Movie) ValidateMovie(vld *validator.Validator) {
	vld.Check(m.Title != "", "title", "must be provided")
	vld.Check(len(m.Title) <= 500, "title", "must not be more than 500 bytes long")

	vld.Check(m.Year != 0, "year", "must be provided")
	vld.Check(m.Year >= 1888, "year", "must be greater than 1888")
	vld.Check(m.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	vld.Check(m.Runtime != 0, "runtime", "must be provided")
	vld.Check(m.Runtime > 0, "runtime", "must be a positive integer")

	vld.Check(m.Genres != nil, "genres", "must be provided")
	vld.Check(len(m.Genres) >= 1, "genres", "must contain at least 1 genre")
	vld.Check(len(m.Genres) <= 5, "genres", "must not contain more than 5 genres")
	vld.Check(validator.Unique(m.Genres), "genres", "must not contain duplicate values")
}

func (m *Movie) FromNewMovie(input NewMovie) {
	m.Title = input.Title
	m.Year = input.Year
	m.Runtime = input.Runtime
	m.Genres = input.Genres
}

func (m *Movie) FromUpdateMovie(input UpdateMovie) {
	if input.Title != nil {
		m.Title = *input.Title
	}

	if input.Year != nil {
		m.Year = *input.Year
	}
	if input.Runtime != nil {
		m.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		m.Genres = input.Genres
	}
}

// Insert will create a new movie in the database.
func (m MovieModel) Insert(movie *Movie) error {
	args := []any{movie.Title, movie.Year, movie.Runtime, movie.Genres}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRow(ctx, insertSQL, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

// Read will fetch a movie from the database.
func (m MovieModel) Read(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	var movie Movie
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRow(ctx, readSQL, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		&movie.Genres,
		&movie.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

// Update will update a movie from the database.
// This operation is implementing optimistic locking.
func (m MovieModel) Update(movie *Movie) error {
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		movie.Genres,
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRow(ctx, updateSQL, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

// Delete will delete a movie from the database.
func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := m.DB.Exec(ctx, deleteSQL, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrRecordNotFound
	}

	return nil
}
