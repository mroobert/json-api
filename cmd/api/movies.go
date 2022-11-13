package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mroobert/json-api/internal/data"
	"github.com/mroobert/json-api/internal/database"
	"github.com/mroobert/json-api/internal/validator"
	"github.com/mroobert/json-api/internal/web"
)

// createMovieHandler for the "POST /v1/movies" endpoint.
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input data.NewMovie

	err := web.ReadJSON(w, r, &input)
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

	err = web.WriteJSON(w, http.StatusCreated, web.Envelope{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// showMovieHandler for the "GET /v1/movies/:id" endpoint.
func (app *application) readMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := web.ReadIDParam(r)
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

	err = web.WriteJSON(w, http.StatusOK, web.Envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// updateMovieHandler for the "PUT /v1/movies/:id" endpoint.
func (app *application) updateMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := web.ReadIDParam(r)
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
	err = web.ReadJSON(w, r, &input)
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

	err = web.WriteJSON(w, http.StatusOK, web.Envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// deleteMovieHandler for the "DELETE" /v1/movies/:id" endpoint.
func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := web.ReadIDParam(r)
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

	err = web.WriteJSON(w, http.StatusOK, web.Envelope{"message": "movie succesfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

}

// readAllMoviesHandler for the "GET /v1/movies?..." endpoint.
func (app *application) readAllMoviesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Genres []string
		database.Filters
	}

	vld := validator.New()
	qs := r.URL.Query()

	input.Title = web.ReadString(qs, "title", "")
	input.Genres = web.ReadCSV(qs, "genres", []string{})
	input.Page = web.ReadInt(qs, "page", 1, vld)
	input.PageSize = web.ReadInt(qs, "page_size", 20, vld)
	input.Sort = web.ReadString(qs, "sort", "id")
	input.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	if input.ValidateFilters(vld); !vld.Valid() {
		app.failedValidationResponse(w, r, vld.Errors)
		return
	}

	movies, metadata, err := app.models.Movies.ReadAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = web.WriteJSON(w, http.StatusOK, web.Envelope{"movies": movies, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
