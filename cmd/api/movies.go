package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mroobert/json-api/internal/data"
	"github.com/mroobert/json-api/internal/validator"
)

// createMovieHandler for the "POST /v1/movies" endpoint.
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input data.NewMovie

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	vld := validator.New()
	movie := new(data.Movie)
	movie.FromDto(input)

	if movie.ValidateMovie(vld); !vld.Valid() {
		app.failedValidationResponse(w, r, vld.Errors)
		return
	}

	// Dump the contents of the input struct in a HTTP response.
	fmt.Fprintf(w, "%+v\n", input)
}

// showMovieHandler for the "GET /v1/movies/:id" endpoint.
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "title-placeholder",
		Runtime:   120,
		Genres:    []string{"romance", "comedy"},
		Version:   1,
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
