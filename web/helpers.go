package main

import (
	"io"
	"strings"
	"strconv"
	"log"
	"encoding/json"
	"net/http"
	"errors"
	"fmt"
)

func readJson(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarsahlTypeError *json.UnmarshalTypeError
		var invalidUnmarsahlError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarsahlTypeError):

			if unmarsahlTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarsahlTypeError.Field)
			}

			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarsahlTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

			// have to unga bunga extract the error
			// https://github.com/golang/go/issues/29035
		case strings.HasPrefix(err.Error(), "json: unkown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unkown field ")
			return fmt.Errorf("body contains unkown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

			// if a nil pointer was passed
		case errors.As(err, &invalidUnmarsahlError):
			panic(err)

		default:
			return err
		}
	}

	// use pointer to a dummy empty struct
	// if calling decode again doesn't fail then with io.EOF then the body
	// contains two json objects
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func writeJson(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")

	if err != nil {
		return err
	}

	// make it easier to see in curl
	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *App) notFound(w http.ResponseWriter, r *http.Request, message any) {
	app.errorResponse(w, r, http.StatusNotFound, message)
}

func (app *App) unkownPath(w http.ResponseWriter, r *http.Request) {
	app.notFound(w, r, "unknown path")
}


func (app *App) errorResponse(w http.ResponseWriter, r *http.Request, status int, mesage any) {
	env := Envelope{"error": mesage}

	err := writeJson(w, status, env, nil)
	if err != nil {
		// TODO: make a real error logger
		log.Println(r, err)
		w.WriteHeader(500)
	}
}

func (app *App) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func int64PathValue(r *http.Request, name string) int64 {
	valueStr := r.PathValue(name)
	valueInt, _ := strconv.Atoi(valueStr)
	return int64(valueInt)
}

func intPathValue(r *http.Request, name string) int {
	valueStr := r.PathValue(name)
	valueInt, _ := strconv.Atoi(valueStr)
	return valueInt
}

