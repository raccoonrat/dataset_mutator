package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Minimal OpenAI-shaped echo for Milestone 1 (no inference).
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
	model := req.Model
	if model == "" {
		model = "echo-mock"
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
				"content": "[echo] " + lastUser,
			},
			"finish_reason": "stop",
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
