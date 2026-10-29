// Package kosync implements the KOReader progress-sync HTTP contract.
// It contains an original implementation of the public protocol, not copied
// upstream server code.
package kosync

import (
	"context"
	"errors"
)

var (
	ErrUserExists       = errors.New("kosync: user already exists")
	ErrUserNotFound     = errors.New("kosync: user not found")
	ErrProgressNotFound = errors.New("kosync: progress not found")
)

// Progress is the wire representation used by KOReader. Timestamp is assigned
// by the server; client timestamps are never trusted for conflict decisions.
type Progress struct {
	Document   string  `json:"document"`
	Progress   string  `json:"progress"`
	Percentage float64 `json:"percentage"`
	Device     string  `json:"device"`
	DeviceID   string  `json:"device_id"`
	Timestamp  int64   `json:"timestamp"`
}

// Store persists account verifiers and per-user document progress.
type Store interface {
	CreateUser(context.Context, string, string) error
	UserVerifier(context.Context, string) (string, error)
	PutProgress(context.Context, string, Progress) error
	GetProgress(context.Context, string, string) (Progress, error)
	Close() error
}
