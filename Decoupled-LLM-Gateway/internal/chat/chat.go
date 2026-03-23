package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/decoy"
)

// Message is a minimal OpenAI-compatible chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DecodeRequest reads the body with a hard cap (caller passes limit reader).
func DecodeRequest(r io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(r, maxBytes+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("body too large")
	}
	return b, nil
}

// TransformRequestBody parses the JSON envelope, transforms only `messages`, and re-marshals.
// Unknown top-level fields are preserved (OpenAI clients often send many optional keys).
func TransformRequestBody(body []byte, obfuscate func(string) string, decoyID string) ([]byte, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	rawMsgs, ok := root["messages"]
	if !ok {
		return nil, fmt.Errorf("messages required")
	}
	var msgs []Message
	if err := json.Unmarshal(rawMsgs, &msgs); err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("messages required")
	}
	applyObfuscation(&msgs, obfuscate)
	ensureSystemDecoy(&msgs, decoyID)
	outMsgs, err := json.Marshal(msgs)
	if err != nil {
		return nil, err
	}
	root["messages"] = json.RawMessage(outMsgs)
	return json.Marshal(root)
}

// UserPromptSnapshotFromBody extracts user-role text before any transform (for logging / policy input).
func UserPromptSnapshotFromBody(body []byte) (string, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return "", err
	}
	rawMsgs, ok := root["messages"]
	if !ok {
		return "", fmt.Errorf("messages required")
	}
	var msgs []Message
	if err := json.Unmarshal(rawMsgs, &msgs); err != nil {
		return "", err
	}
	return userPromptSnapshot(msgs), nil
}

// ObfuscatedSnapshotFromBody returns the concatenated user text after obfuscation only (no decoy).
func ObfuscatedSnapshotFromBody(body []byte, obfuscate func(string) string) (string, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return "", err
	}
	rawMsgs, ok := root["messages"]
	if !ok {
		return "", fmt.Errorf("messages required")
	}
	var msgs []Message
	if err := json.Unmarshal(rawMsgs, &msgs); err != nil {
		return "", err
	}
	cp := append([]Message(nil), msgs...)
	applyObfuscation(&cp, obfuscate)
	return userPromptSnapshot(cp), nil
}

func userPromptSnapshot(msgs []Message) string {
	var parts []byte
	for _, m := range msgs {
		if m.Role == "user" {
			if len(parts) > 0 {
				parts = append(parts, '\n')
			}
			parts = append(parts, m.Content...)
		}
	}
	if len(parts) == 0 {
		for _, m := range msgs {
			if m.Role != "system" {
				if len(parts) > 0 {
					parts = append(parts, '\n')
				}
				parts = append(parts, m.Content...)
			}
		}
	}
	return string(parts)
}

func applyObfuscation(msgs *[]Message, fn func(string) string) {
	for i := range *msgs {
		switch (*msgs)[i].Role {
		case "user", "assistant", "tool":
			(*msgs)[i].Content = fn((*msgs)[i].Content)
		}
	}
}

func ensureSystemDecoy(msgs *[]Message, decoyID string) {
	if decoyID == "" {
		return
	}
	if len(*msgs) == 0 {
		return
	}
	if (*msgs)[0].Role == "system" {
		(*msgs)[0].Content = decoy.InjectIntoSystem((*msgs)[0].Content, decoyID)
		return
	}
	prefix := strings.TrimSpace(decoy.InjectIntoSystem("", decoyID))
	*msgs = append([]Message{{Role: "system", Content: prefix}}, *msgs...)
}
