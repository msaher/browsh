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
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// TODO: multi-user support. Use locks?

// rename to TrackedCmd?
type Cmd struct {
	Id int64
	StartedAt time.Time
	ExitedAt time.Time
	*exec.Cmd
}

type App struct {
	Cmds map[int64]*Cmd
	LastId int64
}

type Envelope map[string]any

func addCmd(app *App, cmd *exec.Cmd) int64 {
	app.LastId++
	app.Cmds[app.LastId] = &Cmd{Id: app.LastId, Cmd: cmd}
	return app.LastId
}

type CmdMetadata struct {
	CmdId int64 `json:"cmdId"`
	Pid	int `json:"pid,omitempty"`
	Status string `json:"status,omitempty"`
	StartedAt time.Time `json:"startedAt,omitempty"`
	ExitCode int  `json:"exitCode,omitempty"`
	ExitedAt time.Time `json:"exitedAt,omitempty"`
}

func newCmdMetadata(cmd *Cmd) *CmdMetadata {
	metadata := &CmdMetadata{
		CmdId: cmd.Id,
		StartedAt: cmd.StartedAt,
		ExitedAt: cmd.ExitedAt,
	}
	if cmd.Process == nil {
		if cmd.Err != nil {
			// TODO: be more elaborate
			metadata.Status = "lookup error"
		} else {
			metadata.Status = "waiting"
		}
		return metadata
	}

	metadata.Pid = cmd.Process.Pid
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		metadata.Status = "exited"
		metadata.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		// started but not finished
		metadata.Pid = cmd.Process.Pid
		metadata.Status = "running"
	}

	return metadata
}

func (app *App) runCmdSocket(cmd *Cmd, conn *websocket.Conn) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	// TODO: temporary
	cmd.Stderr = cmd.Stdout

	go func() {
		defer stdin.Close()

		for {
			_, r, err := conn.NextReader()
			if err != nil {
				return
			}
			if _, err := io.Copy(stdin, r); err != nil {
				return
			}
		}
	}()

	go func() {
	    buf := make([]byte, 4096)
	    for {
	        n, err := stdout.Read(buf)
	        if n > 0 {
	            conn.WriteMessage(websocket.BinaryMessage, buf[:n])
	        }
	        if err != nil {
	            return
	        }
	    }
	}()

	cmd.StartedAt = time.Now()
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		// run .wait() to populate .ProcessState
		_ = cmd.Wait()
		cmd.ExitedAt = time.Now()
		conn.Close()
	}()


	return nil
}

func (app *App) cmdWebsocket(w http.ResponseWriter, r *http.Request) {
	id := int64PathValue(r, "id")
	// TODO:
	if id == 0 {
		panic("got id 0")
	}
	cmd := app.Cmds[id]
	// TODO: not found
	if cmd == nil {
		panic("cant find cmd in app.cmds")
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// TODO: temp
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	// TODO: real error
	if err != nil {
		panic(err)
	}

	app.runCmdSocket(cmd, conn)
}

func (app *App) registerCmd(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Argv []string `json:"argv"`
	}

	err := readJson(w, r, &payload)
	// TODO: show error
	if err != nil {
		panic(err)
	}

	c := exec.Command(payload.Argv[0], payload.Argv[1:]...)
	cmdId := addCmd(app, c)

	writeJson(w, http.StatusOK, Envelope{"id": cmdId}, nil)
}

func (app *App) cmdMetadata(w http.ResponseWriter, r *http.Request) {
	cmdId := int64PathValue(r, "id")
	if cmdId == 0 || app.Cmds[cmdId] == nil {
		app.notFound(w, r, "not found")
		return
	}

	cmd := app.Cmds[cmdId]
	metadata := newCmdMetadata(cmd)

	writeJson(w, http.StatusOK, Envelope{"metadata": metadata}, nil)
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

func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: make this stricter
		// allow any origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// handle preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func int64PathValue(r *http.Request, name string) int64 {
	valueStr := r.PathValue(name)
	valueInt, _ := strconv.Atoi(valueStr)
	return int64(valueInt)
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

func (app *App) notFound(w http.ResponseWriter, r *http.Request, message any) {
	app.errorResponse(w, r, http.StatusNotFound, message)
}

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

func makeHandler(app *App) http.Handler {
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
	mux.HandleFunc("POST /run", app.registerCmd)
	mux.HandleFunc("GET /ws/{id}", app.cmdWebsocket)
	mux.HandleFunc("GET /cmd/{id}/metadata", app.cmdMetadata)

	var handler http.Handler
	handler = mux
	handler = enableCors(handler)
	handler = logHandler(handler)
	return handler
}

func newApp() *App {
	return &App{Cmds: make(map[int64]*Cmd)}
}

func entryPoint() int {
	app := newApp()
	port := flag.Int("port", 8000, "port")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	server := http.Server {
		Addr: addr,
		Handler: makeHandler(app),
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
