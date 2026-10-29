// Package identify implements evidence-led publication identification.
package identify

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/helixreader/helix/core/isbn"
	"github.com/helixreader/helix/server/internal/catalog"
)

var ErrNoUsableInput = errors.New("identify: no usable metadata, identifiers, or content")

// Snapshot is a bounded, privacy-aware extraction from a publication.
type Snapshot struct {
	FileName      string
	Embedded      catalog.Edition
	TextSample    string
	ContentSHA256 string
}

// Candidate is a provider result, never automatically catalog truth.
type Candidate struct {
	Edition    catalog.Edition `json:"edition"`
	Provider   string          `json:"provider"`
	Confidence float64         `json:"confidence"`
}

// Resolver obtains candidates from an external or local source.
type Resolver interface {
	Name() string
	Resolve(context.Context, Snapshot) ([]Candidate, error)
}

// Decision is a typed routing signal. Models cannot raise confidence.
type Decision struct {
	Choice      string  `json:"choice"`
	Probability float64 `json:"probability"`
}

// DecisionModel can recommend review; only deterministic policy can accept.
type DecisionModel interface {
	Decide(context.Context, Snapshot, Candidate) (Decision, error)
}

// Pipeline runs local extraction, resolvers, validation and deterministic
// arbitration. Resolver failures are retained as evidence, not hidden.
type Pipeline struct {
	Resolvers     []Resolver
	Decision      DecisionModel
	AutoAcceptMin float64
	Now           func() time.Time
}

// Result records the selected candidate and every attempted provider.
type Result struct {
	Edition  catalog.Edition `json:"edition"`
	Accepted bool            `json:"accepted"`
	Errors   []string        `json:"errors,omitempty"`
}

func cleanText(s string, max int) string {
	s = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s))
	if max > 0 && len(s) > max {
		return s[:max]
	}
	return s
}

func validateCandidate(c Candidate) (Candidate, error) {
	if c.Provider == "" {
		return c, errors.New("empty provider")
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return c, fmt.Errorf("%s: confidence outside [0,1]", c.Provider)
	}
	c.Edition.Title = cleanText(c.Edition.Title, 512)
	if c.Edition.Title == "" {
		return c, fmt.Errorf("%s: empty title", c.Provider)
	}
	for i := range c.Edition.Authors {
		c.Edition.Authors[i] = cleanText(c.Edition.Authors[i], 256)
	}
	ids := make([]catalog.Identifier, 0, len(c.Edition.Identifiers))
	for _, id := range c.Edition.Identifiers {
		id.Scheme = strings.ToLower(cleanText(id.Scheme, 32))
		id.Value = cleanText(id.Value, 128)
		if id.Scheme == "isbn" || id.Scheme == "isbn13" || id.Scheme == "isbn10" {
			v, ok := isbn.Canonical(id.Value)
			if !ok {
				continue
			}
			id.Value = v
		}
		if id.Scheme != "" && id.Value != "" {
			ids = append(ids, id)
		}
	}
	c.Edition.Identifiers = ids
	if len(ids) == 0 && len(c.Edition.Authors) == 0 {
		return c, fmt.Errorf("%s: candidate lacks both validated identifier and author", c.Provider)
	}
	return c, nil
}

func candidateLess(a, b Candidate) bool {
	if a.Confidence != b.Confidence {
		return a.Confidence > b.Confidence
	}
	if a.Provider != b.Provider {
		return a.Provider < b.Provider
	}
	return a.Edition.Title < b.Edition.Title
}

func localCandidate(s Snapshot) (Candidate, bool) {
	e := s.Embedded
	e.Title = cleanText(e.Title, 512)
	best := 0.0
	for i := range e.Identifiers {
		id := &e.Identifiers[i]
		id.Scheme = strings.ToLower(cleanText(id.Scheme, 32))
		if id.Scheme != "isbn" && id.Scheme != "isbn13" && id.Scheme != "isbn10" {
			continue
		}
		v, ok := isbn.Canonical(id.Value)
		if !ok {
			continue
		}
		id.Value = v
		if best < 0.98 {
			best = 0.98
		}
	}
	if e.Title == "" {
		return Candidate{}, false
	}
	if best > 0 {
		return Candidate{Edition: e, Provider: "embedded-metadata", Confidence: best}, true
	}
	if len(e.Authors) > 0 {
		return Candidate{Edition: e, Provider: "embedded-metadata", Confidence: 0.72}, true
	}
	return Candidate{}, false
}

// Resolve executes the complete pipeline. Low-confidence and model-only
// outcomes remain NeedsReview; no arbitrary title is invented as a fallback.
func (p *Pipeline) Resolve(ctx context.Context, s Snapshot) (Result, error) {
	if strings.TrimSpace(s.Embedded.Title) == "" && len(s.Embedded.Identifiers) == 0 && strings.TrimSpace(s.TextSample) == "" {
		return Result{}, ErrNoUsableInput
	}
	if len(s.TextSample) > 32768 {
		s.TextSample = s.TextSample[:32768]
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	min := p.AutoAcceptMin
	if min == 0 {
		min = 0.86
	}
	var candidates []Candidate
	var errs []string
	if c, ok := localCandidate(s); ok {
		candidates = append(candidates, c)
	}
	for _, r := range p.Resolvers {
		cs, err := r.Resolve(ctx, s)
		if err != nil {
			errs = append(errs, r.Name()+": "+sanitizeError(err).Error())
			continue
		}
		for _, c := range cs {
			c.Provider = r.Name()
			valid, err := validateCandidate(c)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			// Content-model proposals are capped until externally corroborated.
			if strings.HasPrefix(r.Name(), "llm:") && valid.Confidence > 0.79 {
				valid.Confidence = 0.79
			}
			candidates = append(candidates, valid)
		}
	}
	if len(candidates) == 0 {
		e := s.Embedded
		e.State = catalog.NeedsReview
		e.Confidence = 0
		e.UpdatedAt = now().UTC()
		if e.Title == "" {
			base := filepath.Base(s.FileName)
			e.Title = cleanText(strings.TrimSuffix(base, filepath.Ext(base)), 512)
		}
		return Result{Edition: e, Errors: errs}, nil
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidateLess(candidates[i], candidates[j]) })
	best := candidates[0]
	accepted := best.Confidence >= min && (len(best.Edition.Identifiers) > 0 || len(best.Edition.Authors) > 0)
	if p.Decision != nil {
		d, err := p.Decision.Decide(ctx, s, best)
		if err != nil {
			errs = append(errs, "decision-model: "+err.Error())
			accepted = false
		} else if d.Choice != "accept" || d.Probability < min {
			accepted = false
		}
	}
	best.Edition.UpdatedAt = now().UTC()
	best.Edition.Confidence = best.Confidence
	best.Edition.State = catalog.NeedsReview
	if accepted {
		best.Edition.State = catalog.Identified
	} else if best.Edition.Title != "" {
		best.Edition.State = catalog.Partial
	}
	best.Edition.Evidence = append(best.Edition.Evidence, catalog.MetadataEvidence{Field: "edition", Provider: best.Provider, Confidence: best.Confidence, ObservedAt: now().UTC()})
	return Result{Edition: best.Edition, Accepted: accepted, Errors: errs}, nil
}
