package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/chat"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/config"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/decoy"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/logsink"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/obfuscate"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/policy"
)

const (
	headerGatewayExperimentMode = "X-Gateway-Experiment-Mode"
	headerExperimentRunID       = "X-Experiment-Run-Id"
	headerDefenseBaseline       = "X-Defense-Baseline"
)

// Handler is the sync hot path: policy → obfuscate → decoy → upstream.
type Handler struct {
	UpstreamBase string
	// UpstreamBearerToken, if non-empty, sets Authorization: Bearer on upstream requests.
	UpstreamBearerToken string
	UpstreamClient      *http.Client
	MaxBody             int64
	Policy              policy.Store
	Log                 logsink.Sink
	// Obfuscate rewrites user-visible message text; nil uses package default obfuscate.Prompt.
	Obfuscate func(string) string
	// DefaultExperimentMode from GATEWAY_EXPERIMENT_MODE; per-request header overrides.
	DefaultExperimentMode string
	// LogAfterResponse: if true, Log.Emit runs in a goroutine after the response body is written (avoids blocking on Redis).
	LogAfterResponse bool
	// LogMaxPromptRunes / LogMaxLLMRunes: 0 = do not truncate logged text (UTF-8 rune counts).
	LogMaxPromptRunes int
	LogMaxLLMRunes    int
}

func resolveExperimentMode(r *http.Request, fallback string) string {
	if v := strings.TrimSpace(r.Header.Get(headerGatewayExperimentMode)); v != "" {
		return strings.ToLower(v)
	}
	if fallback == "" {
		return "default"
	}
	return strings.ToLower(fallback)
}

func experimentFlags(mode string) (noObfuscate, noDecoy bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "intent_only":
		return true, true
	case "no_obfuscate":
		return true, false
	case "no_decoy":
		return false, true
	case "structured_wrap":
		// StruQ-style: delimit untrusted user spans; keep obfuscation + decoy like default.
		return false, false
	default:
		return false, false
	}
}

func experimentStructuredWrap(mode string) bool {
	return strings.ToLower(strings.TrimSpace(mode)) == "structured_wrap"
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
		http.NotFound(w, r)
		return
	}

	start := time.Now()
	traceID := r.Header.Get("X-Trace-ID")
	if traceID == "" {
		traceID = "req-" + randomHex(6)
	}

	body, err := chat.DecodeRequest(r.Body, h.MaxBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	expMode := resolveExperimentMode(r, h.DefaultExperimentMode)
	noObf, noDecoy := experimentFlags(expMode)
	obfuscateFn := h.obfuscateForRequest(noObf)

	var decoyID string
	if !noDecoy {
		decoyID, err = decoy.NewID()
		if err != nil {
			http.Error(w, "decoy id", http.StatusInternalServerError)
			return
		}
	}

	rawSnapshot, obfuscatedSnapshot, outBody, err := chat.PrepareRequestBody(body, obfuscateFn, decoyID, experimentStructuredWrap(expMode))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	buildEvent := func(llm string, degraded bool) contracts.GatewayLogEvent {
		return contracts.GatewayLogEvent{
			TraceID:               traceID,
			Timestamp:             time.Now().Unix(),
			InjectedDecoyID:       decoyID,
			RawUserPrompt:         rawSnapshot,
			ObfuscatedPrompt:      obfuscatedSnapshot,
			LLMResponse:           llm,
			DegradationTriggered:  degraded,
			ExperimentRunID:       strings.TrimSpace(r.Header.Get(headerExperimentRunID)),
			DefenseBaseline:       strings.TrimSpace(r.Header.Get(headerDefenseBaseline)),
			GatewayExperimentMode: expMode,
			SyncProcessingMS:      time.Since(start).Milliseconds(),
		}
	}

	if rule := h.Policy.MatchPrompt(obfuscatedSnapshot); rule != nil && rule.Action == policy.ActionDegradeToTemplate {
		respText := openAICompletionJSON(rule.TemplateResponse)
		event := buildEvent(respText, true)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respText))
		h.emitLogEvent(event)
		return
	}

	upURL := h.UpstreamBase + "/v1/chat/completions"
	ctx := r.Context()
	client := h.UpstreamClient
	if client == nil {
		client = http.DefaultClient
	}
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}

	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upURL, bytes.NewReader(outBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	upReq.Header.Set("Content-Type", "application/json")
	if h.UpstreamBearerToken != "" {
		upReq.Header.Set("Authorization", "Bearer "+h.UpstreamBearerToken)
	}
	if rid := r.Header.Get("X-Request-ID"); rid != "" {
		upReq.Header.Set("X-Request-ID", rid)
	}
	// Propagate eval headers to upstream echo-llm (paper benchmarks).
	for _, k := range []string{
		"X-Echo-Eval-Secret",
		"X-Echo-Refuse-Substr",
		"X-Echo-Leak-System",
	} {
		if v := r.Header.Get(k); v != "" {
			upReq.Header.Set(k, v)
		}
	}

	resp, err := client.Do(upReq)
	if err != nil {
		http.Error(w, "upstream: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	upBody, err := io.ReadAll(io.LimitReader(resp.Body, h.MaxBody+1))
	if err != nil {
		http.Error(w, "upstream read", http.StatusBadGateway)
		return
	}
	if int64(len(upBody)) > h.MaxBody {
		http.Error(w, "upstream body too large", http.StatusBadGateway)
		return
	}

	llmSnippet := extractAssistantContent(upBody)
	event := buildEvent(llmSnippet, false)

	for k, vv := range resp.Header {
		if k == "Content-Length" {
			continue
		}
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(upBody)
	h.emitLogEvent(event)
}

func (h *Handler) emitLogEvent(e contracts.GatewayLogEvent) {
	h.truncateLogEvent(&e)
	if h.LogAfterResponse {
		go h.Log.Emit(e)
		return
	}
	h.Log.Emit(e)
}

func (h *Handler) truncateLogEvent(e *contracts.GatewayLogEvent) {
	e.RawUserPrompt = truncateRunes(e.RawUserPrompt, h.LogMaxPromptRunes)
	e.ObfuscatedPrompt = truncateRunes(e.ObfuscatedPrompt, h.LogMaxPromptRunes)
	e.LLMResponse = truncateRunes(e.LLMResponse, h.LogMaxLLMRunes)
}

// truncateRunes returns s unchanged if max <= 0; otherwise cuts after max UTF-8 runes and appends "...".
func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i] + "..."
		}
		n++
	}
	return s
}

func extractAssistantContent(body []byte) string {
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || len(payload.Choices) == 0 {
		return string(body)
	}
	return payload.Choices[0].Message.Content
}

func openAICompletionJSON(content string) string {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type choice struct {
		Index        int    `json:"index"`
		Message      msg    `json:"message"`
		FinishReason string `json:"finish_reason"`
	}
	out := struct {
		ID      string   `json:"id"`
		Object  string   `json:"object"`
		Created int64    `json:"created"`
		Model   string   `json:"model"`
		Choices []choice `json:"choices"`
	}{
		ID:      "chatcmpl-degraded",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gateway-degraded",
		Choices: []choice{{
			Index:        0,
			Message:      msg{Role: "assistant", Content: content},
			FinishReason: "stop",
		}},
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func (h *Handler) baseObfuscate() func(string) string {
	if h.Obfuscate != nil {
		return h.Obfuscate
	}
	return obfuscate.Prompt
}

func (h *Handler) obfuscateForRequest(disable bool) func(string) string {
	if disable {
		return func(s string) string { return s }
	}
	base := h.baseObfuscate()
	return base
}

// New builds a handler from config (upstream, obfuscation profile, optional extra rules file).
func New(cfg *config.Config, store policy.Store, sink logsink.Sink) (*Handler, error) {
	eng, err := obfuscate.NewConfiguredEngine(cfg.ObfuscateProfile, cfg.ObfuscateRulesFile)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(cfg.UpstreamURL.String(), "/")
	return &Handler{
		UpstreamBase:        base,
		UpstreamBearerToken: cfg.UpstreamAPIKey,
		UpstreamClient: &http.Client{
			Timeout: cfg.UpstreamTimeout,
		},
		MaxBody:               cfg.MaxBodyBytes,
		Policy:                store,
		Log:                   sink,
		Obfuscate:             eng.Apply,
		DefaultExperimentMode: cfg.ExperimentMode,
		LogAfterResponse:      cfg.LogAfterResponse,
		LogMaxPromptRunes:     cfg.LogMaxPromptRunes,
		LogMaxLLMRunes:        cfg.LogMaxLLMRunes,
	}, nil
}
