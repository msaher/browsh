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
	Inter *Interpreter
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

func entryPoint() int {
	dir, err := os.GetCwd()
	if err != nil {
		panic(err)
	}
	app := &App{Inter := NewInterpreter(dir)}


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
