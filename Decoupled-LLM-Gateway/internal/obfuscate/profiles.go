package obfuscate

import (
	"fmt"
	"strings"
)

// ProfileStrict and ProfileFull apply the full M2 pipeline (default).
const (
	ProfileStrict   = "strict"
	ProfileFull     = "full"
	ProfileMinimal  = "minimal"
	ProfileBalanced = "balanced"
)

// RulesForProfile returns the built-in rule set. Empty, "strict", and "full" are equivalent.
//   - strict / full: all DefaultRules (maximum redaction for logs / policy snapshots).
//   - minimal: UUID variants + id_kv only (faster, fewer false positives on docs/snippets).
//   - balanced: full minus ipv4 and email (common SOTA compromise when prompts mix code, IPs, and prose).
func RulesForProfile(profile string) ([]Rule, error) {
	p := strings.ToLower(strings.TrimSpace(profile))
	switch p {
	case "", ProfileStrict, ProfileFull:
		return DefaultRules(), nil
	case ProfileMinimal:
		return minimalRules()
	case ProfileBalanced:
		return balancedRules(), nil
	default:
		return nil, fmt.Errorf("obfuscate: unknown GATEWAY_OBFUSCATE_PROFILE %q (want strict, full, minimal, balanced)", profile)
	}
}

func minimalRules() ([]Rule, error) {
	byName := ruleIndex(DefaultRules())
	order := []string{"uuid_braced", "uuid_std", "uuid_compact", "id_kv"}
	out := make([]Rule, 0, len(order))
	for _, n := range order {
		r, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("obfuscate: minimal profile missing built-in rule %q", n)
		}
		out = append(out, r)
	}
	return out, nil
}

func balancedRules() []Rule {
	var out []Rule
	for _, r := range DefaultRules() {
		if r.Name == "ipv4" || r.Name == "email" {
			continue
		}
		out = append(out, r)
	}
	return out
}

func ruleIndex(rules []Rule) map[string]Rule {
	m := make(map[string]Rule, len(rules))
	for _, r := range rules {
		m[r.Name] = r
	}
	return m
}
