package main

import (
	"fmt"
	"net/http"
	"flag"
	"log"
	"embed"
	"io"
	"strings"
	"io/fs"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"syscall"
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
	ExitCode int  `json:"exitCode"`
	ExitedAt time.Time `json:"exitedAt,omitempty"`
}

func newCmdMetadata(cmd *Cmd) *CmdMetadata {
	m := &CmdMetadata{
		CmdId:     cmd.Id,
		StartedAt: cmd.StartedAt,
		ExitedAt:  cmd.ExitedAt,
	}

	// binary lookup failed (exec.Command stage)
	if cmd.Err != nil {
		m.Status = "error"
		return m
	}

	// never started or Start() failed
	if cmd.Process == nil {
		m.Status = "waiting"
		return m
	}

	m.Pid = cmd.Process.Pid

	// finished
	if cmd.ProcessState != nil {
		m.Status = "exited"
		m.ExitCode = cmd.ProcessState.ExitCode()
		return m
	}

	// still running
	m.Status = "running"
	return m
}

func findAliveProcess(pid int) (*os.Process, error) {
	// always succeeds in unix
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, err
	}

	// TODO: do this on unix only
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		return nil, err
	}

	return proc, nil // alive
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

			data, err := io.ReadAll(r)
			if err != nil {
				return
			}

			// check for eof marker (ctrl+d / \x04)
			if len(data) == 1 && data[0] == 0x04 {
				return
			}

			// otherwise write to stdin
			if _, err := stdin.Write(data); err != nil {
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

	if id == 0 {
		msg := "invalid cmd id"
		app.errorResponse(w, r, http.StatusBadRequest, msg)
		return
	}
	cmd := app.Cmds[id]
	if cmd == nil {
		app.notFound(w, r, "not found")
		return
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
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if len(payload.Argv) == 0 {
		msg := "empty argv"
		app.errorResponse(w, r, http.StatusBadRequest, msg)
		return
	}

	path, err := exec.LookPath(payload.Argv[0])
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	c := &exec.Cmd{Path: path, Args: payload.Argv[0:]}
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

func (app *App) cmdSignal(w http.ResponseWriter, r *http.Request) {
	pid := intPathValue(r, "pid")
	if pid <= 0 {
		msg := "Invalid pid"
		app.errorResponse(w, r, http.StatusBadRequest, msg)
		return
	}
	var payload struct {
		Signal string `json:"signal"`
	}

	err := readJson(w, r, &payload)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	sigMap := map[string]syscall.Signal{
		"SIGTERM": syscall.SIGTERM,
		"SIGKILL": syscall.SIGKILL,
		"SIGSTOP": syscall.SIGSTOP,
		"SIGCONT": syscall.SIGCONT,
		"SIGINT":  syscall.SIGINT,
	}

	signal, ok := sigMap[payload.Signal]
	if !ok {
		app.badRequestResponse(w, r, err)
		return
	}

	process, err := findAliveProcess(pid)
	if err != nil {
		msg := "can't find alive process"
		app.notFound(w, r, msg)
		return
	}

	if err := process.Signal(signal); err != nil {
			app.errorResponse(w, r, http.StatusInternalServerError, err.Error())
	    return
	}


	writeJson(w, http.StatusOK, Envelope{"msg": "all good"}, nil)
}

func (app *App) complete(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Text string `json:"text"`
	}
	err := readJson(w, r, &payload)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	result, err := complete(payload.Text)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	writeJson(w, http.StatusOK, Envelope{"result": result}, nil)
}

//go:embed ui
var uiFiles embed.FS

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

	mux.HandleFunc("POST /run", app.registerCmd)
	mux.HandleFunc("GET /ws/{id}", app.cmdWebsocket)
	mux.HandleFunc("GET /cmd/{id}/metadata", app.cmdMetadata)
	mux.HandleFunc("POST /signal/{pid}", app.cmdSignal)
	mux.HandleFunc("POST /complete", app.complete)
	mux.HandleFunc("/", app.unkownPath)

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
