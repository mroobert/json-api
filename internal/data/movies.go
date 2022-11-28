package data

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mroobert/json-api/internal/database"
	"github.com/mroobert/json-api/internal/validator"
)

//go:embed queries/movies/create.sql
var createMovieSQL string

//go:embed queries/movies/read.sql
var readMovieSQL string

//go:embed queries/movies/update.sql
var updateMovieSQL string

//go:embed queries/movies/delete.sql
var deleteMovieSQL string

type (
	// Movie represents an individual movie.
	Movie struct {
		ID        int64     `json:"id"`                // Unique integer ID for the movie
		CreatedAt time.Time `json:"-"`                 // Timestamp for when the movie is added to our database
		Title     string    `json:"title"`             // Movie title
		Year      int32     `json:"year,omitempty"`    // Movie release year
		Runtime   Runtime   `json:"runtime,omitempty"` // Movie runtime (in minutes)
		Genres    []string  `json:"genres,omitempty"`  // Slice of genres for the movie (romance, comedy, etc.)
		Version   int32     `json:"version"`           // The version number starts at 1 and will be incremented each time the movie information is updated
	}

	// MovieRepository manages the set of APIs for movie database access.
	MovieRepository struct {
		DB *pgxpool.Pool
	}

	// NewMovie contains information needed to create a new movie.
	NewMovie struct {
		Title   string   `json:"title"`
		Year    int32    `json:"year"`
		Runtime Runtime  `json:"runtime"`
		Genres  []string `json:"genres"`
	}

	// UpdateMovie contains information needed to update a Movie.
	// All fields are optional so clients can send just the fields they want
	// changed. It uses pointer fields so we can differentiate between a field that
	// was not provided and a field that was provided as explicitly blank.
	UpdateMovie struct {
		Title   *string  `json:"title"`
		Year    *int32   `json:"year"`
		Runtime *Runtime `json:"runtime"`
		Genres  []string `json:"genres"`
	}
)

func (m Movie) Validate(vld *validator.Validator) {
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

// Create will insert a new movie in the database.
func (r MovieRepository) Create(movie *Movie) error {
	args := []any{movie.Title, movie.Year, movie.Runtime, movie.Genres}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return r.DB.QueryRow(ctx, createMovieSQL, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

// Read will fetch a movie from the database.
func (r MovieRepository) Read(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	var movie Movie
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.DB.QueryRow(ctx, readMovieSQL, id).Scan(
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
func (r MovieRepository) Update(movie *Movie) error {
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
	err := r.DB.QueryRow(ctx, updateMovieSQL, args...).Scan(&movie.Version)
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
func (r MovieRepository) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.DB.Exec(ctx, deleteMovieSQL, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// ReadAll will fetch all movies based on the provided parameters.
// It uses a full-text search for the title.
func (r MovieRepository) ReadAll(title string, genres []string, filters database.Filters) ([]*Movie, database.Metadata, error) {
	query := fmt.Sprintf(`
        SELECT  count(*) OVER(), id, created_at, title, year, runtime, genres, version
        FROM movies
        WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '') 
        AND (genres @> $2 OR $2 = '{}')     
        ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`, filters.SortColumn(), filters.SortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{title, genres, filters.Limit(), filters.Offset()}
	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, database.Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	movies := []*Movie{}
	for rows.Next() {
		var movie Movie

		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			&movie.Genres,
			&movie.Version,
		)
		if err != nil {
			return nil, database.Metadata{}, err
		}

		movies = append(movies, &movie)
	}

	if err := rows.Err(); err != nil {
		return nil, database.Metadata{}, err
	}

	metadata := database.NewMetadata(totalRecords, filters.Page, filters.PageSize)

	return movies, metadata, err
}
