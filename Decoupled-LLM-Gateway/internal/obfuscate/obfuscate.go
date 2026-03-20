package obfuscate

import "regexp"

// DefaultPlaceholder marks redacted material so policy rules stay predictable.
const DefaultPlaceholder = "[ID_REMOVED]"

// Rule is one transform in pipeline order. More specific patterns must run first.
type Rule struct {
	Name string
	RE   *regexp.Regexp
	Repl string
}

// Engine runs a fixed sequence of replacements. Safe for concurrent use.
type Engine struct {
	rules []Rule
}

// NewEngine copies the rule slice; use for tests or custom pipelines.
func NewEngine(rules []Rule) *Engine {
	cp := make([]Rule, len(rules))
	copy(cp, rules)
	return &Engine{rules: cp}
}

// Apply returns a new string; s is not modified.
func (e *Engine) Apply(s string) string {
	if s == "" || len(e.rules) == 0 {
		return s
	}
	out := s
	for i := range e.rules {
		out = e.rules[i].RE.ReplaceAllString(out, e.rules[i].Repl)
	}
	return out
}

// DefaultRules returns the Milestone 2 pipeline: token shapes first, then generic IDs.
func DefaultRules() []Rule {
	return []Rule{
		{Name: "jwt", RE: regexp.MustCompile(`eyJ[^.]+\.eyJ[^.]+\.[^.\s]+`), Repl: DefaultPlaceholder},
		{Name: "bearer", RE: regexp.MustCompile(`(?i)bearer\s+[a-z0-9._~+/=-]+`), Repl: DefaultPlaceholder},
		{Name: "api_sk", RE: regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{10,}\b`), Repl: DefaultPlaceholder},
		{Name: "github_pat", RE: regexp.MustCompile(`\b(?:ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]+)\b`), Repl: DefaultPlaceholder},
		{Name: "slack_token", RE: regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]+\b`), Repl: DefaultPlaceholder},
		{Name: "aws_access_key", RE: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), Repl: DefaultPlaceholder},
		{Name: "google_api_key", RE: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`), Repl: DefaultPlaceholder},
		{Name: "url_secrets", RE: regexp.MustCompile(`(?i)([?&])(access_token|refresh_token|api[_-]?key|apikey|token|secret|password|client_secret)=([^&\s#]+)`), Repl: "$1$2=" + DefaultPlaceholder},
		{Name: "uuid_braced", RE: regexp.MustCompile(`(?i)\{[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\}`), Repl: DefaultPlaceholder},
		{Name: "uuid_std", RE: regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`), Repl: DefaultPlaceholder},
		{Name: "uuid_compact", RE: regexp.MustCompile(`(?i)\b[0-9a-f]{32}\b`), Repl: DefaultPlaceholder},
		{Name: "ulid", RE: regexp.MustCompile(`\b[0-9A-HJKMNP-TV-Z]{26}\b`), Repl: DefaultPlaceholder},
		{Name: "mongo_objectid", RE: regexp.MustCompile(`(?i)\b[0-9a-f]{24}\b`), Repl: DefaultPlaceholder},
		{Name: "ipv4", RE: regexp.MustCompile(`(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`), Repl: DefaultPlaceholder},
		{Name: "email", RE: regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`), Repl: DefaultPlaceholder},
		{Name: "id_kv", RE: regexp.MustCompile(`(?i)\b(user|session|tenant|account|customer|org|organization|device|request|trace|correlation|order|transaction|subscription|invoice)(?:_id|\s+id)\s*[:=]\s*[^\s,;]+`), Repl: DefaultPlaceholder},
	}
}

var defaultEngine = NewEngine(DefaultRules())

// Prompt applies the default Milestone 2 pipeline. The input is never modified in place.
func Prompt(s string) string {
	return defaultEngine.Apply(s)
}
