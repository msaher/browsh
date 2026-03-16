package main

import (
	"fmt"
	"io/fs"
	"embed"
	"net/http"
	"syscall"

	"github.com/msaher/browsh/shell"

	"github.com/gorilla/websocket"
)

func (app *App) registerCmd(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Src string `json:"src"`
	}

	err := readJson(w, r, &payload)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	id := RegisterJob(app, payload.Src)
	writeJson(w, http.StatusOK, Envelope{"id": id}, nil)
}

func (app *App) startJob(w http.ResponseWriter, r *http.Request) {
	id := int(int64PathValue(r, "id"))
	job, exists := app.Jobs[id]
	if !exists {
		app.notFound(w, r, "not found")
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	// TODO: real error
	if err != nil {
		panic(err)
	}

	stdin, stdout, stderr, err := NewWsStdio(conn)
	if err != nil {
		app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
	}
	stdio := shell.Stdio{Stdin: stdin, Stdout: stdout, Stderr: stderr}

	job.Result = shell.NewResult()
	go func() {
		app.Inter.ExecStrRes(job.Src, stdio, job.Result)
		if job.Result.IsErr() {
			app.errorLog.Println(job.Result.Err())
		}

		msg := map[string]any {
			"kind": "exit",
			"stream": StreamControl,
			"exitCode": job.Result.ExitCode(),
		}
		conn.WriteJSON(msg)

		conn.Close()

	}()

}

func (app *App) signal(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Signal string `json:"signal"`
	}

	err := readJson(w, r, &payload)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	id := int(int64PathValue(r, "id"))
	job, exists := app.Jobs[id]
	if !exists {
		app.notFound(w, r, "not found")
		return
	}
	result := job.Result

	sigMap := map[string]syscall.Signal{
		"SIGTERM": syscall.SIGTERM,
		"SIGKILL": syscall.SIGKILL,
		"SIGSTOP": syscall.SIGSTOP,
		"SIGCONT": syscall.SIGCONT,
		"SIGINT":  syscall.SIGINT,
	}
	sig, exists := sigMap[payload.Signal]
	if !exists {
		app.errorResponse(w, r, http.StatusBadRequest, "unknown signal")
		return
	}

	if result.CurrentCmd().IsBuiltin {
		app.badRequestResponse(w, r, fmt.Errorf("signals are not yet implemented for builtin cmds"))
	}

	err = app.Inter.Signal(result, sig)
	if err != nil {
		app.errorLog.Printf("%v", err)
	}
}

func (app *App) complete(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Src    string `json:"src"`
		Cursor int    `json:"cursor"`
	}
	if err := readJson(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	completions := app.Inter.Complete(payload.Src, payload.Cursor)
	writeJson(w, http.StatusOK, Envelope{"completions": completions}, nil)
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

	mux.HandleFunc("POST /job/register", app.registerCmd)
	mux.HandleFunc("GET /job/{id}/ws", app.startJob)
	mux.HandleFunc("POST /job/{id}/signal", app.signal)
	mux.HandleFunc("POST /complete", app.complete)
	mux.HandleFunc("/", app.unkownPath)

	var handler http.Handler
	handler = mux
	handler = enableCors(handler)
	handler = app.logHandler(handler)
	return handler
}
