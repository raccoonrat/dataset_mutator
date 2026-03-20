package obfuscate

import "testing"

func TestPrompt_UUID(t *testing.T) {
	in := "token 550e8400-e29b-41d4-a716-446655440000 end"
	got := Prompt(in)
	want := "token [ID_REMOVED] end"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPrompt_IDKV(t *testing.T) {
	in := "x user_id: 12345 y"
	got := Prompt(in)
	if got != "x [ID_REMOVED] y" {
		t.Fatalf("got %q", got)
	}
}
