package web

import (
	"encoding/json"
	"net/http"
)

// Used as an envelope type in the http response.
type Envelope map[string]any

// WriteJSON takes the destination http.ResponseWriter, the HTTP status code to send,
//
//	the data to encode to JSON, and a  header map containing any additional HTTP headers
//
// we want to include in the response.
func WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	// Encode the data to JSON, returning the error if there was one.
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Append a newline to make it easier to view in terminal applications.
	js = append(js, '\n')

	// At this point, we know that we won't encounter any more errors before writing the
	// response, so it's safe to add any headers that we want to include.
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Add the "Content-Type: application/json" header, then write the status code and
	// JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
