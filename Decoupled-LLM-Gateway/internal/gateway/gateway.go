package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/chat"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/decoy"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/logsink"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/obfuscate"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/policy"
)

// Handler is the sync hot path: policy → obfuscate → decoy → upstream.
type Handler struct {
	UpstreamBase   string
	UpstreamClient *http.Client
	MaxBody        int64
	Policy         policy.Store
	Log            logsink.Sink
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
		http.NotFound(w, r)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	if traceID == "" {
		traceID = "req-" + randomHex(6)
	}

	body, err := chat.DecodeRequest(r.Body, h.MaxBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawSnapshot, err := chat.UserPromptSnapshotFromBody(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	obfuscatedSnapshot, err := chat.ObfuscatedSnapshotFromBody(body, obfuscate.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	decoyID, err := decoy.NewID()
	if err != nil {
		http.Error(w, "decoy id", http.StatusInternalServerError)
		return
	}

	if rule := h.Policy.MatchPrompt(obfuscatedSnapshot); rule != nil && rule.Action == policy.ActionDegradeToTemplate {
		respText := openAICompletionJSON(rule.TemplateResponse)
		event := contracts.GatewayLogEvent{
			TraceID:              traceID,
			Timestamp:            time.Now().Unix(),
			InjectedDecoyID:      decoyID,
			RawUserPrompt:        rawSnapshot,
			ObfuscatedPrompt:     obfuscatedSnapshot,
			LLMResponse:          respText,
			DegradationTriggered: true,
		}
		h.Log.Emit(event)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respText))
		return
	}

	outBody, err := chat.TransformRequestBody(body, obfuscate.Prompt, decoyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	if rid := r.Header.Get("X-Request-ID"); rid != "" {
		upReq.Header.Set("X-Request-ID", rid)
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
	event := contracts.GatewayLogEvent{
		TraceID:              traceID,
		Timestamp:            time.Now().Unix(),
		InjectedDecoyID:      decoyID,
		RawUserPrompt:        rawSnapshot,
		ObfuscatedPrompt:     obfuscatedSnapshot,
		LLMResponse:          llmSnippet,
		DegradationTriggered: false,
	}
	h.Log.Emit(event)

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
