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

func TestExperimentModeStructuredWrapUpstreamHasDelimiters(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !bytes.Contains(b, []byte("[BEGIN_UNTRUSTED_USER]")) {
			t.Errorf("expected structured wrap in upstream body: %s", string(b))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer up.Close()

	h := &Handler{
		UpstreamBase:          strings.TrimSuffix(up.URL, ""),
		UpstreamClient:        &http.Client{Timeout: 5 * time.Second},
		MaxBody:               1 << 20,
		Policy:                policy.NewMemoryStore(),
		Log:                   logsink.Discard{},
		DefaultExperimentMode: "structured_wrap",
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	reqBody := `{"model":"m","messages":[{"role":"user","content":"ping"}]}`
	res, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestExperimentModeIntentOnlySkipsDecoyAndObfuscation(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), "user-uuid-550e8400-e29b-41d4-a716-446655440000") {
			t.Errorf("expected raw uuid in upstream body when obfuscation off, got %s", string(b))
		}
		if strings.Contains(string(b), "decoy_session") {
			t.Errorf("did not expect decoy in upstream body: %s", string(b))
		}
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
		MaxBody:               1 << 20,
		Policy:                policy.NewMemoryStore(),
		Log:                   logsink.NewJSONLines(&buf),
		Obfuscate:             obfuscate.Prompt,
		DefaultExperimentMode: "default",
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	uuid := "user-uuid-550e8400-e29b-41d4-a716-446655440000"
	reqBody := `{"model":"m","messages":[{"role":"user","content":"ping ` + uuid + ` done"}]}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("X-Gateway-Experiment-Mode", "intent_only")
	res, err := http.DefaultClient.Do(req)
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
	if ev.InjectedDecoyID != "" {
		t.Fatalf("expected empty decoy id, got %q", ev.InjectedDecoyID)
	}
	if !strings.Contains(ev.RawUserPrompt, uuid) && !strings.Contains(ev.ObfuscatedPrompt, uuid) {
		t.Fatalf("expected uuid in snapshot")
	}
	if !strings.Contains(ev.ObfuscatedPrompt, uuid) {
		t.Fatalf("intent_only should not strip uuid, got obf=%q", ev.ObfuscatedPrompt)
	}
	if ev.GatewayExperimentMode != "intent_only" {
		t.Fatalf("mode %q", ev.GatewayExperimentMode)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("hello世界", 0); got != "hello世界" {
		t.Fatalf("0 cap: %q", got)
	}
	if got := truncateRunes("hello世界", 5); got != "hello..." {
		t.Fatalf("5 runes: %q", got)
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

func TestOutputGuardReplacesWhenJudgeSaysNonRefusal(t *testing.T) {
	guard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_refusal": false}`))
	}))
	defer guard.Close()

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"unsafe completion"}}]}`))
	}))
	defer up.Close()

	var buf bytes.Buffer
	h := &Handler{
		UpstreamBase: strings.TrimSuffix(up.URL, ""),
		UpstreamClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		MaxBody:                  1 << 20,
		Policy:                   policy.NewMemoryStore(),
		Log:                      logsink.NewJSONLines(&buf),
		OutputGuardURL:           guard.URL,
		OutputGuardRequireHeader: true,
		OutputGuardFailOpen:      true,
		OutputGuardTimeout:       5 * time.Second,
		OutputGuardTemplate:      "BLOCKED",
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	reqBody := `{"model":"m","messages":[{"role":"user","content":"harmful ask"}]}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("X-Gateway-Output-Guard", "1")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var root map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&root); err != nil {
		t.Fatal(err)
	}
	var choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	_ = json.Unmarshal(root["choices"], &choices)
	if len(choices) == 0 || choices[0].Message.Content != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %+v", choices)
	}
}

func TestOutputGuardSkippedWithoutHeader(t *testing.T) {
	guardCalls := 0
	guard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		guardCalls++
		_, _ = w.Write([]byte(`{"is_refusal": false}`))
	}))
	defer guard.Close()

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"keep me"}}]}`))
	}))
	defer up.Close()

	h := &Handler{
		UpstreamBase: strings.TrimSuffix(up.URL, ""),
		UpstreamClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		MaxBody:                  1 << 20,
		Policy:                   policy.NewMemoryStore(),
		Log:                      logsink.NewJSONLines(&bytes.Buffer{}),
		OutputGuardURL:           guard.URL,
		OutputGuardRequireHeader: true,
		OutputGuardTimeout:       5 * time.Second,
		OutputGuardTemplate:      "BLOCKED",
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", strings.NewReader(`{"model":"m","messages":[{"role":"user","content":"x"}]}`))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "keep me") {
		t.Fatalf("expected original body, got %s", body)
	}
	if guardCalls != 0 {
		t.Fatalf("guard should not run without header, calls=%d", guardCalls)
	}
}
