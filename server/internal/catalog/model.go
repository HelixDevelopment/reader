// Package catalog defines the work/edition/file model shared by ingestion,
// identification, OPDS and client APIs. A work is the conceptual creation;
// editions are distinguishable manifestations; files are concrete variants.
package catalog

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("catalog: publication not found")

// IdentificationState is a closed state machine. NeedsReview is a first-class
// catalog state, not an invitation to invent metadata.
type IdentificationState string

const (
	Pending     IdentificationState = "pending"
	Identified  IdentificationState = "identified"
	Partial     IdentificationState = "partial"
	NeedsReview IdentificationState = "needs_review"
	Failed      IdentificationState = "failed"
)

// Identifier retains provenance and confidence for every external identity.
type Identifier struct {
	Scheme     string  `json:"scheme"`
	Value      string  `json:"value"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
}

// MetadataEvidence records where a field came from; content hashes allow the
// exact input to be reproduced without exposing book text in logs.
type MetadataEvidence struct {
	Field      string    `json:"field"`
	Provider   string    `json:"provider"`
	Confidence float64   `json:"confidence"`
	ObservedAt time.Time `json:"observed_at"`
}

// File describes a publication resource, including encrypted variants.
type File struct {
	ID          string    `json:"id"`
	SourceID    string    `json:"source_id,omitempty"`
	Path        string    `json:"-"`
	Format      string    `json:"format"`
	MIMEType    string    `json:"mime_type"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	Protected   bool      `json:"protected"`
	DRMScheme   string    `json:"drm_scheme,omitempty"`
	LicenseHREF string    `json:"license_href,omitempty"`
	ModifiedAt  time.Time `json:"modified_at"`
}

// Edition contains edition-specific data; never merge ISBNs at work level.
type Edition struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Subtitle    string              `json:"subtitle,omitempty"`
	Authors     []string            `json:"authors"`
	Publisher   string              `json:"publisher,omitempty"`
	Published   string              `json:"published,omitempty"`
	Language    string              `json:"language,omitempty"`
	Description string              `json:"description,omitempty"`
	Identifiers []Identifier        `json:"identifiers"`
	Files       []File              `json:"files"`
	CoverHREF   string              `json:"cover_href,omitempty"`
	State       IdentificationState `json:"identification_state"`
	Confidence  float64             `json:"identification_confidence"`
	Evidence    []MetadataEvidence  `json:"evidence,omitempty"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Work groups editions without destroying edition-level identity.
type Work struct {
	ID             string    `json:"id"`
	CanonicalTitle string    `json:"canonical_title"`
	Series         string    `json:"series,omitempty"`
	SeriesIndex    float64   `json:"series_index,omitempty"`
	Subjects       []string  `json:"subjects,omitempty"`
	Editions       []Edition `json:"editions"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Query is a deterministic catalog query. Offset pagination is suitable for
// the in-memory development store; PostgreSQL implementations use keysets.
type Query struct {
	Text   string
	Author string
	State  IdentificationState
	Offset int
	Limit  int
	// After requests the page following a keyset position. When it is set,
	// Offset is ignored: the two are alternative ways of saying where a page
	// starts, and honoring both would let a client ask for a position and an
	// offset at once and get one of them silently.
	After *Cursor
}

// Page contains stable pagination metadata.
type Page struct {
	Works  []Work `json:"works"`
	Total  int    `json:"total"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
	// HasNext reports that at least one more work follows this page in the same
	// ordering. Offset pages derive it from Total, cursor pages from the row the
	// repository read past the end of the page, so a client following cursors
	// never needs Total to know when to stop.
	HasNext bool `json:"has_next"`
}

// Repository is the storage seam used by all protocol handlers.
type Repository interface {
	UpsertWork(context.Context, Work) error
	MergeWork(context.Context, Work) error
	// ReconcileSource removes files absent from a complete recursive full-source
	// listing and returns the number removed. Prefix and non-recursive scans must
	// not call it: an incomplete listing is never deletion evidence.
	ReconcileSource(context.Context, string, []string) (int, error)
	GetWork(context.Context, string) (Work, error)
	FindFile(context.Context, string) (Work, Edition, File, error)
	FindEdition(context.Context, string) (Work, Edition, error)
	// Search returns one page and whether any work follows it. Implementations
	// must populate Page.HasNext: the protocol layers page from it, and inferring
	// it from Total would require a count they are allowed to skip.
	Search(context.Context, Query) (Page, error)
	DeleteWork(context.Context, string) error
	// ReferencedLCPArtifacts returns the distinct artifact hashes of every
	// fulfilled-LCP file in one consistent read (cache GC reference source).
	ReferencedLCPArtifacts(context.Context) (map[string]bool, error)
	Close() error
}
