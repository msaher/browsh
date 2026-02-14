package main

import (
	"fmt"
	"net/http"
	"flag"
	"log"
	"os"
	"embed"
	"io"
	"strings"
	"io/fs"
	"encoding/json"
	"errors"
	"os/exec"
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

func runCmd(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Argv []string `json:"argv"`
	}

	err := readJson(w, r, &payload)
	// TODO: show response
	if err != nil {
		panic(err)
	}
	cmd := exec.Command(payload.Argv[0], payload.Argv[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w
	w.Header().Set("Content-Type", "text/plain")
	cmd.Run()
}

//go:embed ui
var uiFiles embed.FS

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func logHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	    next.ServeHTTP(w, r)
	    log.Printf("%s %s\n", r.Method, r.URL.Path)
	})
}

func makeHandler() http.Handler {
	mux := http.NewServeMux()

	staticFiles, err := fs.Sub(uiFiles, "ui/static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(staticFiles))
	fileServer = http.StripPrefix("/static/", fileServer)
	//  TODO: enable cache in production
	mux.Handle("/static/", noCache(fileServer))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "src/ui/index.html")
	})
	mux.HandleFunc("POST /run", runCmd)

	handler := logHandler(mux)
	return handler
}

func entryPoint() int {
	port := flag.Int("port", 8000, "port")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	server := http.Server {
		Addr: addr,
		Handler: makeHandler(),
	}

	log.Printf("Listening on :%d", *port)

	err := server.ListenAndServe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %s\n", err)
		return 1
	}

	return 0
}

func main() {
	result := entryPoint()
	os.Exit(result)
}
