package main

import (
	"encoding/json"
	"sync"
	// "github.com/msaher/browsh/shell"

	"github.com/gorilla/websocket"
)

type StreamType string

const (
	StreamStdout StreamType = "stdout"
	StreamStderr StreamType = "stderr"
	StreamStdin  StreamType = "stdin"
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
	msg, _ := json.Marshal(WsMessage{Stream: w.Stream, Data: string(p)})
	w.Mu.Lock()
	err := w.Conn.WriteMessage(websocket.TextMessage, msg)
	w.Mu.Unlock()
	return len(p), err
}

type WsReader struct {
	Conn *websocket.Conn
}

func (r *WsReader) Read(p []byte) (int, error) {
	_, msg, err := r.Conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	n := copy(p, msg)
	return n, nil
}

func NewWsStdio(conn *websocket.Conn) (stdin *WsReader, stdout, stderr *WsWriter) {
	mu := &sync.Mutex{}
	stdin = &WsReader{Conn: conn}
	stdout = &WsWriter{Conn: conn, Stream: StreamStdout, Mu: mu}
	stderr = &WsWriter{Conn: conn, Stream: StreamStderr, Mu: mu}
	return
}
