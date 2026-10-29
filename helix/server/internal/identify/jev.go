package identify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/helixreader/helix/core/pkg/parse/parseutil"
)

// JevClient implements a typed System One decision adapter. Endpoint and
// model are configurable because hosted decision APIs evolve independently.
type JevClient struct {
	Endpoint string
	APIKey   string
	Model    string
	Client   *http.Client
}

func (j *JevClient) Decide(ctx context.Context, s Snapshot, c Candidate) (Decision, error) {
	if j.Endpoint == "" {
		return Decision{}, fmt.Errorf("Jev endpoint is required")
	}
	if err := requireSecureEndpoint(j.Endpoint, j.APIKey != ""); err != nil {
		return Decision{}, err
	}
	model := j.Model
	if model == "" {
		model = "jev-latest"
	}
	state := map[string]any{"file_name": s.FileName, "content_sha256": s.ContentSHA256, "candidate": c}
	// Untrusted excerpts are deliberately omitted from this routing decision;
	// this limits prompt-injection surface and protects reading content privacy.
	payload, _ := json.Marshal(map[string]any{"model": model, "state": state, "questions": map[string]any{"route": map[string]any{"type": "choice", "choices": []string{"accept", "review"}, "description": "Whether bibliographic evidence is sufficiently coherent for deterministic policy review; this signal never overrides local thresholds."}}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return Decision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if j.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+j.APIKey)
	}
	client := j.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := redirectSafeClient(client, []string{"Authorization"}).Do(req)
	if err != nil {
		return Decision{}, err
	}
	defer resp.Body.Close()
	body, err := parseutil.ReadAll(resp.Body, 1<<20)
	if err != nil {
		return Decision{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Decision{}, fmt.Errorf("Jev HTTP %d", resp.StatusCode)
	}
	var decoded struct {
		Answers map[string]struct {
			Choice        string             `json:"choice"`
			Probability   float64            `json:"probability"`
			Probabilities map[string]float64 `json:"probabilities"`
		} `json:"answers"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Decision{}, err
	}
	a, ok := decoded.Answers["route"]
	if !ok {
		return Decision{}, fmt.Errorf("Jev response lacks route answer")
	}
	d := Decision{Choice: strings.ToLower(a.Choice), Probability: a.Probability}
	if d.Probability == 0 && a.Probabilities != nil {
		d.Probability = a.Probabilities[a.Choice]
	}
	if d.Choice != "accept" && d.Choice != "review" {
		return Decision{}, fmt.Errorf("Jev returned unexpected choice %q", d.Choice)
	}
	if d.Probability < 0 || d.Probability > 1 {
		return Decision{}, fmt.Errorf("Jev probability outside [0,1]")
	}
	return d, nil
}
