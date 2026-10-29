package kosync

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidAuth          = errors.New("kosync: invalid credentials")
	ErrRegistrationDisabled = errors.New("kosync: registration disabled")
	ErrInvalidInput         = errors.New("kosync: invalid input")
	validUsername           = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)
)

// Service owns validation and verifier handling. KOReader's password digest is
// treated as a password and is never stored directly; only bcrypt verifiers are
// persisted. TLS remains mandatory when exposing the protocol over a network.
type Service struct {
	store        Store
	registration bool
	dummy        string
	now          func() time.Time
}

func NewService(store Store, registration bool) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("not-a-user-password"), bcrypt.DefaultCost)
	return &Service{store: store, registration: registration, dummy: string(dummy), now: time.Now}
}

func (s *Service) Close() error { return s.store.Close() }

func (s *Service) Register(ctx context.Context, username, password string) error {
	if !s.registration {
		return ErrRegistrationDisabled
	}
	if !validUsername.MatchString(username) || len(password) < 8 || len(password) > 72 {
		return ErrInvalidInput
	}
	verifier, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash verifier: %w", err)
	}
	return s.store.CreateUser(ctx, username, string(verifier))
}

func (s *Service) Authenticate(ctx context.Context, username, password string) error {
	if username == "" || password == "" || len(username) > 255 || len(password) > 72 {
		return ErrInvalidAuth
	}
	verifier, err := s.store.UserVerifier(ctx, username)
	if errors.Is(err, ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword([]byte(s.dummy), []byte(password))
		return ErrInvalidAuth
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(verifier), []byte(password)) != nil {
		return ErrInvalidAuth
	}
	return nil
}

func (s *Service) PutProgress(ctx context.Context, username string, progress Progress) (Progress, error) {
	if progress.Document == "" || progress.Progress == "" || progress.Device == "" ||
		len(progress.Document) > 255 || len(progress.Progress) > 255 || len(progress.Device) > 255 || len(progress.DeviceID) > 255 ||
		math.IsNaN(progress.Percentage) || math.IsInf(progress.Percentage, 0) || progress.Percentage < 0 || progress.Percentage > 1 {
		return progress, ErrInvalidInput
	}
	progress.Timestamp = s.now().Unix()
	if err := s.store.PutProgress(ctx, username, progress); err != nil {
		return progress, err
	}
	// Stores resolve concurrent writes monotonically. Return the persisted
	// winner so a rejected stale write cannot acknowledge a newer timestamp.
	return s.store.GetProgress(ctx, username, progress.Document)
}

func (s *Service) GetProgress(ctx context.Context, username, document string) (Progress, error) {
	if document == "" || len(document) > 255 {
		return Progress{}, ErrInvalidInput
	}
	return s.store.GetProgress(ctx, username, document)
}
