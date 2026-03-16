package main

import (
	"fmt"
	"net/http"
	"flag"
	"os"
	"log"

	"github.com/msaher/browsh/shell"

)

type Job struct {
	Id int
	Src string
	Result *shell.Result
}

type App struct {
	Inter *shell.Interpreter
	NextJobId int
	Jobs map[int]*Job
	infoLog *log.Logger
	errorLog *log.Logger
	debug bool
}

func NewApp() *App {
	dir, err := os.Getwd()
	// TODO: dont panic
	if err != nil {
		panic(err)
	}

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime|log.LUTC)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)

	return &App {
		Inter: shell.NewInterpreter(dir),
		NextJobId: 1,
		Jobs: make(map[int]*Job),
		infoLog: infoLog,
		errorLog: errorLog,
	}
}

// TODO: lock
func RegisterJob(app *App, src string) int {
	j := &Job {
		Id: app.NextJobId,
		Src: src,
	}
	app.Jobs[app.NextJobId] = j
	app.NextJobId++
	return j.Id
}

type Envelope map[string]any

func entryPoint() int {
	app := NewApp()

	addr := flag.String("addr", "", "address")
	port := flag.Int("port", 4981, "port")
	debug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	app.debug = *debug

	fullAddr := fmt.Sprintf("%s:%d", *addr, *port)
	server := http.Server {
		Addr: fullAddr,
		Handler: makeHandler(app),
	}


	if *addr == "" {
		*addr = "http://localhost"
	}
	log.Printf("Listening on %s:%d", *addr, *port)

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
