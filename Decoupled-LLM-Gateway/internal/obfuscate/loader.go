package obfuscate

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

type fileRule struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Repl    string `json:"repl"`
}

// LoadRulesFromFile reads a JSON array of {name, pattern, repl}. Rules are appended after profile rules.
// Empty or missing repl defaults to DefaultPlaceholder. Pattern must be valid RE2 (Go regexp).
func LoadRulesFromFile(path string) ([]Rule, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []fileRule
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("obfuscate rules file: %w", err)
	}
	out := make([]Rule, 0, len(raw))
	for i, fr := range raw {
		if fr.Pattern == "" {
			return nil, fmt.Errorf("obfuscate rules file: entry %d empty pattern", i)
		}
		re, err := regexp.Compile(fr.Pattern)
		if err != nil {
			return nil, fmt.Errorf("obfuscate rules file entry %d (%s): %w", i, fr.Name, err)
		}
		name := fr.Name
		if name == "" {
			name = fmt.Sprintf("custom_%d", i)
		}
		repl := fr.Repl
		if repl == "" {
			repl = DefaultPlaceholder
		}
		out = append(out, Rule{Name: name, RE: re, Repl: repl})
	}
	return out, nil
}

// NewConfiguredEngine builds an engine from profile name plus optional JSON rule file (appended).
func NewConfiguredEngine(profile, extraRulesFile string) (*Engine, error) {
	base, err := RulesForProfile(profile)
	if err != nil {
		return nil, err
	}
	extra, err := LoadRulesFromFile(extraRulesFile)
	if err != nil {
		return nil, err
	}
	if len(extra) == 0 {
		return NewEngine(base), nil
	}
	rules := make([]Rule, 0, len(base)+len(extra))
	rules = append(rules, base...)
	rules = append(rules, extra...)
	return NewEngine(rules), nil
}
