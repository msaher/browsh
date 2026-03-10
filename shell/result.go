package shell

import (
	"sync"
	"time"
)

type Result struct {
	mu sync.Mutex
	exitCode int
	startedAt time.Time
	exitedAt time.Time
	err error
	currentCmd *Cmd // last cmd we were waiting for
}

func NewResult() *Result {
	return &Result{
		exitCode:  -1,
	}
}
func (r *Result) StartedAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startedAt
}

func (r *Result) ExitedAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.exitedAt
}

func (r *Result) Duration() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.startedAt.IsZero() || r.exitedAt.IsZero() {
		return 0
	}
	return r.exitedAt.Sub(r.startedAt)
}

func (r *Result) ExitCode() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.exitCode
}

func (r *Result) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (r *Result) IsErr() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err != nil
}

func (r *Result) CurrentCmd() *Cmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentCmd
}

func (r *Result) SetCurrentCmd(c *Cmd) {
	r.mu.Lock()
	r.currentCmd = c
	r.mu.Unlock()
}

func (r *Result) SetExitCode(code int) {
    r.mu.Lock()
    r.exitCode = code
    r.mu.Unlock()
}

func (r *Result) SetErr(err error) {
    r.mu.Lock()
    r.err = err
    r.mu.Unlock()
}

func (r *Result) SetExitedAt(t time.Time) {
    r.mu.Lock()
    r.exitedAt = t
    r.mu.Unlock()
}

func (r *Result) SetStartedAt(t time.Time) {
    r.mu.Lock()
    r.startedAt = t
    r.mu.Unlock()
}

