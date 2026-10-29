package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

// Rclone delegates the long-tail transports to the Go rclone executable.
// No shell is involved; credentials stay in the configured rclone store.
type Rclone struct{ protocol, remote, binary, config string }

func NewRclone(cfg Config) (Connector, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("rclone remote is required (for example remote:container/prefix)")
	}
	binary := cfg.RclonePath
	if binary == "" {
		binary = "rclone"
	}
	return &Rclone{protocol: cfg.Protocol, remote: cfg.Endpoint, binary: binary, config: cfg.RcloneConfig}, nil
}
func (r *Rclone) Capabilities() Capabilities {
	return Capabilities{RecursiveList: true, RangeRead: true}
}
func (r *Rclone) Close() error { return nil }
func (r *Rclone) command(ctx context.Context, args ...string) *exec.Cmd {
	if r.config != "" {
		args = append([]string{"--config", r.config}, args...)
	}
	args = append([]string{"--contimeout", "15s", "--timeout", "2m", "--low-level-retries", "2", "--retries", "2"}, args...)
	return exec.CommandContext(ctx, r.binary, args...) //nolint:gosec -- fixed executable and argument vector, no shell
}
func (r *Rclone) List(ctx context.Context, prefix string) ([]Entry, error) {
	target := strings.TrimSuffix(r.remote, "/")
	if prefix != "" {
		target += "/" + strings.TrimPrefix(prefix, "/")
	}
	cmd := r.command(ctx, "lsjson", "--recursive", "--files-only", target)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, remaining: 64 << 10}
	cmd.Stdout = &limitedWriter{w: &stdout, remaining: 64 << 20}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rclone %s: %w: %s", r.protocol, err, strings.TrimSpace(stderr.String()))
	}
	out := stdout.Bytes()
	var rows []struct {
		Path, Name, ModTime string
		Size                int64
		IsDir               bool
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		if row.IsDir {
			continue
		}
		mod, _ := time.Parse(time.RFC3339Nano, row.ModTime)
		p := row.Path
		if prefix != "" {
			p = path.Join(prefix, p)
		}
		entries = append(entries, Entry{Path: p, Name: row.Name, Size: row.Size, Modified: mod})
	}
	return entries, nil
}

type readCloser struct {
	io.Reader
	cancel context.CancelFunc
	cmd    *exec.Cmd
}

func (r *readCloser) Close() error {
	r.cancel()
	err := r.cmd.Wait()
	// Cancelling the stream context is the normal close path; Go reports it
	// as context.Canceled (or "killed" if the signal landed first).
	if err != nil && (errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "killed")) {
		return nil
	}
	return err
}
func (r *Rclone) OpenRange(ctx context.Context, name string, offset, length int64) (io.ReadCloser, error) {
	if offset < 0 || length < -1 {
		return nil, fmt.Errorf("rclone: invalid range")
	}
	if strings.HasPrefix(name, "..") || strings.Contains(name, "/../") {
		return nil, ErrTraversal
	}
	args := []string{"cat", "--offset", strconv.FormatInt(offset, 10)}
	if length >= 0 {
		args = append(args, "--count", strconv.FormatInt(length, 10))
	}
	target := strings.TrimSuffix(r.remote, "/") + "/" + strings.TrimPrefix(name, "/")
	args = append(args, target)
	streamCtx, cancel := context.WithCancel(ctx)
	cmd := r.command(streamCtx, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, remaining: 64 << 10}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	return &readCloser{Reader: stdout, cancel: cancel, cmd: cmd}, nil
}
func (r *Rclone) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	if strings.HasPrefix(name, "..") || strings.Contains(name, "/../") {
		return nil, ErrTraversal
	}
	streamCtx, cancel := context.WithCancel(ctx)
	target := strings.TrimSuffix(r.remote, "/") + "/" + strings.TrimPrefix(name, "/")
	cmd := r.command(streamCtx, "cat", target)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, remaining: 64 << 10}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("rclone cat: %w", err)
	}
	return &readCloser{Reader: stdout, cancel: cancel, cmd: cmd}, nil
}
