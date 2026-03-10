package main

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type StreamType string

const (
	StreamStdout StreamType = "stdout"
	StreamStderr StreamType = "stderr"
	StreamControl StreamType = "control"
)

type WsMessage struct {
	Stream StreamType `json:"stream"`
	Data   string     `json:"data"`
}

type WsWriter struct {
	Conn   *websocket.Conn
	Stream StreamType
	Mu     *sync.Mutex
}

func (w *WsWriter) Write(p []byte) (int, error) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	writer, err := w.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	msg, _ := json.Marshal(WsMessage{Stream: w.Stream, Data: string(p)})
	writer.Write(msg)
	writer.Close()
	return len(p), nil
}

func (w *WsWriter) Close() error {
	return w.Conn.Close()
}

func NewWsStdio(conn *websocket.Conn) (stdin *os.File, stdout, stderr *WsWriter, err error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}

	mu := &sync.Mutex{}
	stdout = &WsWriter{Conn: conn, Stream: StreamStdout, Mu: mu}
	stderr = &WsWriter{Conn: conn, Stream: StreamStderr, Mu: mu}

	// when cmd.Stdin is not an *os.File, it spawns a goroutine that conteniously
	// reads from the reader till its exhausted.. websockets don't work that way
	// because they're not really readers. Instead, we spawn our own goroutine
	// and request the next reader from the websocket, copying it over to a pipe
	// (an os *os.File).
	go func() {
		defer pw.Close()
		for {
			_, r, err := conn.NextReader()
			if err != nil {
				return
			}
			data, err := io.ReadAll(r)
			if err != nil {
				return
			}
			if len(data) == 1 && data[0] == 0x04 {
				return
			}
			if _, err := pw.Write(data); err != nil {
				return
			}
		}
	}()

	return pr, stdout, stderr, nil
}
