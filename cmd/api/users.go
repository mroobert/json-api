package main

import (
	"errors"
	"net/http"

	"github.com/mroobert/json-api/internal/data"
	"github.com/mroobert/json-api/internal/validator"
	"github.com/mroobert/json-api/internal/web"
)

// registerUserHandler for the "POST /v1/users" endpoint.
func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var (
		input data.NewUser
		user  data.User
	)

	err := web.ReadJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user.FromNewUser(input)
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	vld := validator.New()
	if user.Validate(vld); !vld.Valid() {
		app.failedValidationResponse(w, r, vld.Errors)
		return
	}

	err = app.repositories.Users.Create(&user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			vld.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, vld.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = web.WriteJSON(w, http.StatusCreated, web.Envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
