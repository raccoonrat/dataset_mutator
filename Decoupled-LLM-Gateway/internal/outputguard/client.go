// Package outputguard calls an HTTP service compatible with experiments/judge_service (task refusal_binary).
// When the service reports is_refusal==false, the gateway may replace the assistant message (non-refusal → block).
// This is unsafe for general benign traffic unless used with X-Gateway-Output-Guard or only harmful eval prompts.
package outputguard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReplaceNonRefusal POSTs to url (full URL, any path) with judge_service contract; returns true if assistant should be replaced.
// Semantics: replace when the judge says is_refusal is false (model did not refuse — often unsafe on harmful prompts).
func ReplaceNonRefusal(ctx context.Context, endpoint, bearer, user, assistant string, httpClient *http.Client, timeout time.Duration) (bool, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return false, nil
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload := map[string]string{
		"task":        "refusal_binary",
		"user":        user,
		"assistant":   assistant,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if b := strings.TrimSpace(bearer); b != "" {
		req.Header.Set("Authorization", "Bearer "+b)
	}
	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("output guard HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out struct {
		IsRefusal *bool `json:"is_refusal"`
		Refusal   *bool `json:"refusal"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return false, fmt.Errorf("output guard json: %w", err)
	}
	var ref bool
	if out.IsRefusal != nil {
		ref = *out.IsRefusal
	} else if out.Refusal != nil {
		ref = *out.Refusal
	} else {
		return false, fmt.Errorf("output guard: missing is_refusal/refusal in %s", truncate(string(raw), 200))
	}
	// Replace upstream text when model did NOT produce a refusal-shaped response.
	return !ref, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
