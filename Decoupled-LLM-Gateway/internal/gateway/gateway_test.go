package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/config"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/logsink"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/obfuscate"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/policy"
)

func TestEndToEndEcho(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var root map[string]any
		_ = json.Unmarshal(b, &root)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer up.Close()

	var buf bytes.Buffer
	sink := logsink.NewJSONLines(&buf)
	h := &Handler{
		UpstreamBase: strings.TrimSuffix(up.URL, ""),
		UpstreamClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		MaxBody: 1 << 20,
		Policy:  policy.NewMemoryStore(),
		Log:     sink,
	}

	srv := httptest.NewServer(h)
	defer srv.Close()

	reqBody := `{"model":"m","messages":[{"role":"user","content":"hello"}]}`
	res, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}

	line := bytes.TrimSpace(bytes.Split(buf.Bytes(), []byte{'\n'})[0])
	var ev contracts.GatewayLogEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		t.Fatalf("log line: %v — %q", err, line)
	}
	if ev.RawUserPrompt != "hello" {
		t.Fatalf("raw %q", ev.RawUserPrompt)
	}
	if ev.DegradationTriggered {
		t.Fatal("unexpected degrade")
	}
}

func TestPolicyDegrade(t *testing.T) {
	store := policy.NewMemoryStore()
	_ = store.Upsert(contracts.PolicyRule{
		RuleID:           "r1",
		Action:           policy.ActionDegradeToTemplate,
		TriggerPattern:   `(?i)forbidden`,
		TemplateResponse: "nope",
	})

	var buf bytes.Buffer
	h := &Handler{
		UpstreamBase: "http://127.0.0.1:9",
		UpstreamClient: &http.Client{
			Timeout: time.Millisecond,
		},
		MaxBody: 1 << 20,
		Policy:  store,
		Log:     logsink.NewJSONLines(&buf),
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	reqBody := `{"messages":[{"role":"user","content":"FORBIDDEN phrase"}]}`
	res, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", res.StatusCode, b)
	}
	if !bytes.Contains(b, []byte("nope")) {
		t.Fatalf("body %s", b)
	}
}

// Mirrors examples/policy_seed.json so CI does not depend on filesystem layout.
func TestNew_InvalidObfuscateProfile(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		UpstreamURL:      u,
		ObfuscateProfile: "not-a-valid-profile",
	}
	_, err = New(cfg, policy.NewMemoryStore(), logsink.Discard{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandlerObfuscateMinimalPreservesIP(t *testing.T) {
	eng, err := obfuscate.NewConfiguredEngine("minimal", "")
	if err != nil {
		t.Fatal(err)
	}
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer up.Close()

	var buf bytes.Buffer
	h := &Handler{
		UpstreamBase: strings.TrimSuffix(up.URL, ""),
		UpstreamClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		MaxBody:   1 << 20,
		Policy:    policy.NewMemoryStore(),
		Log:       logsink.NewJSONLines(&buf),
		Obfuscate: eng.Apply,
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	reqBody := `{"model":"m","messages":[{"role":"user","content":"ping 192.168.1.1 done"}]}`
	res, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	line := bytes.TrimSpace(bytes.Split(buf.Bytes(), []byte{'\n'})[0])
	var ev contracts.GatewayLogEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		t.Fatalf("log: %v", err)
	}
	if !strings.Contains(ev.ObfuscatedPrompt, "192.168.1.1") {
		t.Fatalf("expected IP preserved in obfuscated snapshot, got %q", ev.ObfuscatedPrompt)
	}
}

func TestPolicySeedEquivalent(t *testing.T) {
	store := policy.NewMemoryStore()
	if err := store.Upsert(contracts.PolicyRule{
		RuleID:           "demo-forbidden-keyword",
		Action:           policy.ActionDegradeToTemplate,
		TriggerPattern:   `(?i)\bFORBIDDEN\b`,
		TemplateResponse: "该请求已被网关策略拒绝（演示规则）。",
	}); err != nil {
		t.Fatal(err)
	}
	if store.MatchPrompt("prefix FORBIDDEN suffix") == nil {
		t.Fatal("expected match")
	}
}
