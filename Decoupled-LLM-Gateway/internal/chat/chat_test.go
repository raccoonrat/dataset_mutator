package chat

import (
	"strings"
	"testing"
)

func id(s string) string { return s }

func TestTransformPreservesExtraKeys(t *testing.T) {
	body := []byte(`{"model":"x","temperature":0.2,"messages":[{"role":"user","content":"hi"}]}`)
	out, err := TransformRequestBody(body, id, "decoy-abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"temperature":0.2`) {
		t.Fatalf("lost field: %s", out)
	}
	if !strings.Contains(string(out), "decoy-abc") {
		t.Fatalf("missing decoy: %s", out)
	}
}

func TestPrepareRequestBodyStructuredWrap(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	raw, obf, out, err := PrepareRequestBody(body, id, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if raw != "hi" || obf != "hi" {
		t.Fatalf("snapshots should exclude delimiters: raw=%q obf=%q", raw, obf)
	}
	// json.Marshal escapes newlines as \ in output bytes — check stable substrings only.
	if !strings.Contains(string(out), "[BEGIN_UNTRUSTED_USER]") || !strings.Contains(string(out), "[END_UNTRUSTED_USER]") {
		t.Fatalf("upstream body missing delimiters: %s", out)
	}
}

func TestPrepareRequestBodyMatchesPiecewise(t *testing.T) {
	body := []byte(`{"model":"x","temperature":0.2,"messages":[{"role":"user","content":"id 550e8400-e29b-41d4-a716-446655440000"}]}`)
	ob := func(s string) string {
		return strings.ReplaceAll(s, "550e8400-e29b-41d4-a716-446655440000", "[ID_REMOVED]")
	}
	rawWant, err := UserPromptSnapshotFromBody(body)
	if err != nil {
		t.Fatal(err)
	}
	obfWant, err := ObfuscatedSnapshotFromBody(body, ob)
	if err != nil {
		t.Fatal(err)
	}
	outWant, err := TransformRequestBody(body, ob, "decoy-z")
	if err != nil {
		t.Fatal(err)
	}
	raw, obf, out, err := PrepareRequestBody(body, ob, "decoy-z", false)
	if err != nil {
		t.Fatal(err)
	}
	if raw != rawWant || obf != obfWant || string(out) != string(outWant) {
		t.Fatalf("raw %q vs %q; obf %q vs %q; out %s vs %s", raw, rawWant, obf, obfWant, out, outWant)
	}
}

func TestObfuscatedSnapshot(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"id 550e8400-e29b-41d4-a716-446655440000"}]}`)
	snap, err := ObfuscatedSnapshotFromBody(body, func(s string) string {
		return strings.ReplaceAll(s, "550e8400-e29b-41d4-a716-446655440000", "[ID_REMOVED]")
	})
	if err != nil {
		t.Fatal(err)
	}
	if snap != "id [ID_REMOVED]" {
		t.Fatalf("got %q", snap)
	}
}

func TestSystemDecoyPrependsWhenNoSystem(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "hi"}}
	ensureSystemDecoy(&msgs, "d1")
	if msgs[0].Role != "system" || !strings.Contains(msgs[0].Content, "d1") {
		t.Fatalf("%+v", msgs)
	}
}
