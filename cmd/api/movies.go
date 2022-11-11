package main

import (
	"errors"
	"fmt"
	"net/http"

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
	movie.FromNewMovie(input)

	if movie.ValidateMovie(vld); !vld.Valid() {
		app.failedValidationResponse(w, r, vld.Errors)
		return
	}

	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// showMovieHandler for the "GET /v1/movies/:id" endpoint.
func (app *application) readMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movies.Read(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// updateMovieHandler for the "PUT /v1/movies/:id" endpoint.
func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movies.Read(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	var input data.UpdateMovie
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	vld := validator.New()
	movie.FromUpdateMovie(input)
	if movie.ValidateMovie(vld); !vld.Valid() {
		app.failedValidationResponse(w, r, vld.Errors)
		return
	}

	err = app.models.Movies.Update(movie)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// deleteMovieHandler for the "DELETE" /v1/movies/:id" endpoint.
func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie succesfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}
