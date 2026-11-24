package liveverify

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Process is a started server that can be stopped.
type Process interface{ Stop() error }

// Exiter is implemented by processes that can report an early exit.
type Exiter interface{ Exited() bool }

// ProcessStarter starts a server without waiting for readiness.
type ProcessStarter interface{ StartNoWait() (Process, error) }

// Launcher starts one server instance and returns its base URL once it is ready.
// The live verifiers restart the server through it, so a launcher must be reusable.
type Launcher interface {
	Start() (base string, p Process, err error)
}

// ExecLauncher runs a real helix-server binary, exactly like the Python harness:
// readiness polls GET /readyz (200) for up to ~90 s, and an exit before readiness
// is an error.
type ExecLauncher struct {
	Binary string
	Env    []string // full environment
	Base   string   // e.g. http://127.0.0.1:28080
	Log    io.Writer
	// ReadyAttempts overrides the default 900 polls (100 ms apart).
	ReadyAttempts int
	// ReadyPath is the readiness endpoint (default /readyz).
	ReadyPath string
	// Ready, when set, replaces the plain-HTTP readiness probe (used for TLS).
	Ready func(base string) bool
	// KillGrace is how long a server that missed readiness gets after SIGTERM before
	// SIGKILL (default 5 s).
	KillGrace time.Duration
}

type execProcess struct {
	cmd  *exec.Cmd
	done chan struct{}
	err  error
}

// StartNoWait starts the binary and returns without a readiness wait.
func (l *ExecLauncher) StartNoWait() (Process, error) {
	cmd := exec.Command(l.Binary)
	cmd.Env = l.Env
	cmd.Stdout, cmd.Stderr = l.Log, l.Log
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", l.Binary, err)
	}
	p := &execProcess{cmd: cmd, done: make(chan struct{})}
	go func() { p.err = cmd.Wait(); close(p.done) }()
	return p, nil
}

func (l *ExecLauncher) Start() (string, Process, error) {
	cmd := exec.Command(l.Binary)
	cmd.Env = l.Env
	cmd.Stdout, cmd.Stderr = l.Log, l.Log
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("start %s: %w", l.Binary, err)
	}
	p := &execProcess{cmd: cmd, done: make(chan struct{})}
	go func() { p.err = cmd.Wait(); close(p.done) }()
	attempts := l.ReadyAttempts
	if attempts == 0 {
		attempts = 900
	}
	ready := l.Ready
	if ready == nil {
		hc := NewClient()
		path := l.ReadyPath
		if path == "" {
			path = "/readyz"
		}
		ready = func(base string) bool {
			r, err := hc.TryDo("GET", base+path, Opts{Timeout: 250 * time.Millisecond})
			return err == nil && r.Status == 200
		}
		defer hc.Close()
	}
	for i := 0; i < attempts; i++ {
		select {
		case <-p.done:
			return "", nil, fmt.Errorf("server exited before readiness")
		default:
		}
		if ready(l.Base) {
			return l.Base, p, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	grace := l.KillGrace
	if grace == 0 {
		grace = 5 * time.Second
	}
	_ = p.StopWithin(grace) // reap the wedged server; its error is secondary to the timeout
	return "", nil, fmt.Errorf("server readiness timeout")
}

// Exited reports whether the process has already ended.
func (p *execProcess) Exited() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// Stop sends SIGTERM when still running and waits up to 20 s (Python parity).
func (p *execProcess) Stop() error { return p.StopWithin(20 * time.Second) }

func (p *execProcess) StopWithin(d time.Duration) error {
	select {
	case <-p.done:
		return nil
	default:
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.done:
		return nil
	case <-time.After(d):
	}
	_ = p.cmd.Process.Signal(syscall.SIGKILL)
	select {
	case <-p.done:
		return fmt.Errorf("server did not exit within %s of SIGTERM; killed with SIGKILL", d)
	case <-time.After(5 * time.Second):
		return fmt.Errorf("server did not exit within %s of SIGTERM and survived SIGKILL", d)
	}
}

// Environ builds a child environment: the current one with overrides applied.
func Environ(overrides map[string]string) []string {
	out := []string{}
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if _, replaced := overrides[key]; !replaced {
			out = append(out, kv)
		}
	}
	for k, v := range overrides {
		out = append(out, k+"="+v)
	}
	return out
}
