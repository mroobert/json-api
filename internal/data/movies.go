package data

import (
	"time"

	"github.com/mroobert/json-api/internal/validator"
)

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

	NewMovie struct {
		Title   string   `json:"title"`
		Year    int32    `json:"year"`
		Runtime Runtime  `json:"runtime"`
		Genres  []string `json:"genres"`
	}
)

func (movie Movie) ValidateMovie(vld *validator.Validator) {
	vld.Check(movie.Title != "", "title", "must be provided")
	vld.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	vld.Check(movie.Year != 0, "year", "must be provided")
	vld.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	vld.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	vld.Check(movie.Runtime != 0, "runtime", "must be provided")
	vld.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	vld.Check(movie.Genres != nil, "genres", "must be provided")
	vld.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	vld.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	vld.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

func (movie *Movie) FromDto(newMovie NewMovie) {
	movie.Title = newMovie.Title
	movie.Year = newMovie.Year
	movie.Runtime = newMovie.Runtime
	movie.Genres = newMovie.Genres
}
