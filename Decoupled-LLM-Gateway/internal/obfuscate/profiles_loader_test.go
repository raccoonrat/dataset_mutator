package obfuscate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRulesForProfile_MinimalSkipsIPv4(t *testing.T) {
	rules, err := RulesForProfile("minimal")
	if err != nil {
		t.Fatal(err)
	}
	e := NewEngine(rules)
	s := "ip 192.168.1.1 uuid 550e8400-e29b-41d4-a716-446655440000"
	got := e.Apply(s)
	if got != "ip 192.168.1.1 uuid [ID_REMOVED]" {
		t.Fatalf("got %q", got)
	}
}

func TestRulesForProfile_StrictStripsIPv4(t *testing.T) {
	rules, err := RulesForProfile("strict")
	if err != nil {
		t.Fatal(err)
	}
	e := NewEngine(rules)
	s := "host 192.168.1.1 ok"
	got := e.Apply(s)
	if got != "host [ID_REMOVED] ok" {
		t.Fatalf("got %q", got)
	}
}

func TestRulesForProfile_BalancedKeepsIPv4(t *testing.T) {
	rules, err := RulesForProfile("balanced")
	if err != nil {
		t.Fatal(err)
	}
	e := NewEngine(rules)
	s := "host 192.168.1.1 email a@b.co"
	got := e.Apply(s)
	if got != "host 192.168.1.1 email a@b.co" {
		t.Fatalf("got %q", got)
	}
}

func TestRulesForProfile_BalancedStillStripsJWT(t *testing.T) {
	rules, err := RulesForProfile("balanced")
	if err != nil {
		t.Fatal(err)
	}
	e := NewEngine(rules)
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.sig"
	got := e.Apply("t " + jwt + " z")
	if got != "t [ID_REMOVED] z" {
		t.Fatalf("got %q", got)
	}
}

func TestRulesForProfile_Unknown(t *testing.T) {
	if _, err := RulesForProfile("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRulesFromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rules.json")
	content := `[{"name":"x","pattern":"(?i)SECRET:\\S+","repl":"[REDACTED]"}]`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	rules, err := LoadRulesFromFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Name != "x" {
		t.Fatalf("%+v", rules)
	}
	got := NewEngine(rules).Apply("prefix SECRET:abc suffix")
	if got != "prefix [REDACTED] suffix" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadRulesFromFile_DefaultRepl(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(p, []byte(`[{"name":"n","pattern":"foo\\d+"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	rules, err := LoadRulesFromFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if rules[0].Repl != DefaultPlaceholder {
		t.Fatalf("repl %q", rules[0].Repl)
	}
}

func TestLoadRulesFromFile_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(p, []byte(`[{"name":"bad","pattern":"("}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRulesFromFile(p); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewConfiguredEngine_AppendsCustom(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(p, []byte(`[{"pattern":"ZZZ\\d+","repl":"[Z]"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err := NewConfiguredEngine("minimal", p)
	if err != nil {
		t.Fatal(err)
	}
	got := e.Apply("ZZZ99 uuid 550e8400-e29b-41d4-a716-446655440000")
	want := "[Z] uuid [ID_REMOVED]"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
