package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Minimal OpenAI-shaped echo for Milestone 1 (no inference).
// ECHO_LEAK_SYSTEM=1 appends system message contents to the reply (Milestone 3 demo: simulated decoy exfiltration).
// Paper-eval: ECHO_EVAL_SECRET / ECHO_REFUSE_SUBSTR, or per-request X-Echo-* headers (forwarded by gateway).
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", completions)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	addr := ":9090"
	if v := os.Getenv("ECHO_LISTEN"); v != "" {
		addr = v
	}
	log.Printf("echo-llm listening %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func completions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<22))
	if err != nil {
		http.Error(w, "read", http.StatusBadRequest)
		return
	}
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	_ = json.Unmarshal(body, &req)
	lastUser := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUser = req.Messages[i].Content
			break
		}
	}
	if lastUser == "" && len(req.Messages) > 0 {
		lastUser = req.Messages[len(req.Messages)-1].Content
	}

	evalSecret := strings.TrimSpace(os.Getenv("ECHO_EVAL_SECRET"))
	if v := strings.TrimSpace(r.Header.Get("X-Echo-Eval-Secret")); v != "" {
		evalSecret = v
	}
	refuseSub := strings.TrimSpace(os.Getenv("ECHO_REFUSE_SUBSTR"))
	if v := strings.TrimSpace(r.Header.Get("X-Echo-Refuse-Substr")); v != "" {
		refuseSub = v
	}
	leak := os.Getenv("ECHO_LEAK_SYSTEM") == "1"
	switch strings.TrimSpace(r.Header.Get("X-Echo-Leak-System")) {
	case "1":
		leak = true
	case "0":
		leak = false
	}

	model := req.Model
	if model == "" {
		model = "echo-mock"
	}

	var content string
	if refuseSub != "" && strings.Contains(strings.ToLower(lastUser), strings.ToLower(refuseSub)) {
		content = "REFUSAL_SINGLE_GUARD: cannot comply with safety policy."
	} else {
		content = "[echo] " + lastUser
		if leak {
			var sys strings.Builder
			for _, m := range req.Messages {
				if m.Role == "system" {
					sys.WriteString(m.Content)
				}
			}
			if evalSecret != "" {
				if sys.Len() > 0 {
					sys.WriteByte('\n')
				}
				sys.WriteString("EVAL_SECRET=")
				sys.WriteString(evalSecret)
			}
			if sys.Len() > 0 {
				content += "\n---\n" + sys.String()
			}
		}
	}
	out := map[string]any{
		"id":      "chatcmpl-echo",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
