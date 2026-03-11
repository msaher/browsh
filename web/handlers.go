package main

import (
	"io/fs"
	"embed"
	"net/http"
	"syscall"

	"github.com/msaher/browsh/shell"

	"github.com/gorilla/websocket"
)

// func (app *App) cmdWebsocket(w http.ResponseWriter, r *http.Request) {
// 	id := int64PathValue(r, "id")
//
// 	if id == 0 {
// 		msg := "invalid cmd id"
// 		app.errorResponse(w, r, http.StatusBadRequest, msg)
// 		return
// 	}
// 	cmd := app.Cmds[id]
// 	if cmd == nil {
// 		app.notFound(w, r, "not found")
// 		return
// 	}
//
// 	var upgrader = websocket.Upgrader{
// 		ReadBufferSize:  1024,
// 		WriteBufferSize: 1024,
// 		// TODO: temp
//     CheckOrigin: func(r *http.Request) bool {
//         return true
//     },
// 	}
//
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	// TODO: real error
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	app.runCmdSocket(cmd, conn)
// }

// func (app *App) registerCmd(w http.ResponseWriter, r *http.Request) {
// 	var payload struct {
// 		Argv []string `json:"argv"`
// 	}
//
// 	err := readJson(w, r, &payload)
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	if len(payload.Argv) == 0 {
// 		msg := "empty argv"
// 		app.errorResponse(w, r, http.StatusBadRequest, msg)
// 		return
// 	}
//
// 	path, err := exec.LookPath(payload.Argv[0])
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	c := &exec.Cmd{Path: path, Args: payload.Argv[0:]}
// 	cmdId := addCmd(app, c)
//
// 	writeJson(w, http.StatusOK, Envelope{"id": cmdId}, nil)
// }

// func (app *App) cmdMetadata(w http.ResponseWriter, r *http.Request) {
// 	cmdId := int64PathValue(r, "id")
// 	if cmdId == 0 || app.Cmds[cmdId] == nil {
// 		app.notFound(w, r, "not found")
// 		return
// 	}
//
// 	cmd := app.Cmds[cmdId]
// 	metadata := newCmdMetadata(cmd)
//
// 	writeJson(w, http.StatusOK, Envelope{"metadata": metadata}, nil)
// }

// func (app *App) runCmdSocket(cmd *Cmd, conn *websocket.Conn) error {
// 	stdin, err := cmd.StdinPipe()
// 	if err != nil {
// 		return err
// 	}
//
// 	stdout, err := cmd.StdoutPipe()
// 	if err != nil {
// 		return err
// 	}
//
// 	// TODO: temporary
// 	cmd.Stderr = cmd.Stdout
//
// 	go func() {
// 		defer stdin.Close()
// 		for {
// 			_, r, err := conn.NextReader()
// 			if err != nil {
// 				return
// 			}
//
// 			data, err := io.ReadAll(r)
// 			if err != nil {
// 				return
// 			}
//
// 			// check for eof marker (ctrl+d / \x04)
// 			if len(data) == 1 && data[0] == 0x04 {
// 				return
// 			}
//
// 			// otherwise write to stdin
// 			if _, err := stdin.Write(data); err != nil {
// 				return
// 			}
// 		}
// 	}()
//
// 	go func() {
// 	    buf := make([]byte, 4096)
// 	    for {
// 	        n, err := stdout.Read(buf)
// 	        if n > 0 {
// 	            conn.WriteMessage(websocket.BinaryMessage, buf[:n])
// 	        }
// 	        if err != nil {
// 	            return
// 	        }
// 	    }
// 	}()
//
// 	cmd.StartedAt = time.Now()
// 	if err := cmd.Start(); err != nil {
// 		return err
// 	}
//
// 	go func() {
// 		// run .wait() to populate .ProcessState
// 		_ = cmd.Wait()
// 		cmd.ExitedAt = time.Now()
// 		conn.Close()
// 	}()
//
//
// 	return nil
// }

//go:embed ui
var uiFiles embed.FS

// func makeHandler(app *App) http.Handler {
// 	mux := http.NewServeMux()
//
// 	staticFiles, err := fs.Sub(uiFiles, "ui/static")
// 	if err != nil {
// 		panic(err)
// 	}
// 	fileServer := http.FileServer(http.FS(staticFiles))
// 	fileServer = http.StripPrefix("/static/", fileServer)
// 	//  TODO: enable cache in production
// 	mux.Handle("/static/", noCache(fileServer))
//
// 	mux.HandleFunc("POST /run", app.registerCmd)
// 	mux.HandleFunc("GET /ws/{id}", app.cmdWebsocket)
// 	mux.HandleFunc("GET /cmd/{id}/metadata", app.cmdMetadata)
// 	mux.HandleFunc("POST /signal/{pid}", app.cmdSignal)
// 	mux.HandleFunc("POST /complete", app.complete)
// 	mux.HandleFunc("/", app.unkownPath)
//
// 	var handler http.Handler
// 	handler = mux
// 	handler = enableCors(handler)
// 	handler = logHandler(handler)
// 	return handler
// }

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

	stdin, stdout, stderr, err := NewWsStdio(conn)
	if err != nil {
		app.errorResponse(w, r, 500, err.Error())
	}
	stdio := shell.Stdio{Stdin: stdin, Stdout: stdout, Stderr: stderr}
	app.infoLog.Printf("about to execute %s\n", job.Src)

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
	mux.HandleFunc("/", app.unkownPath)

	var handler http.Handler
	handler = mux
	handler = enableCors(handler)
	handler = logHandler(handler)
	return handler
}
