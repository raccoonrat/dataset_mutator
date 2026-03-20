package obfuscate

import "regexp"

// MVP: strip common machine IDs from free text. Hot path only does regexp.ReplaceAllString.
var (
	uuidRE = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	// Loose "user/session id=123" style leaks.
	idKVRE = regexp.MustCompile(`(?i)\b(user|session|tenant|account)_?id\s*[:=]\s*[[:alnum:]_-]+\b`)
)

// Prompt returns a normalized copy; original is unchanged.
func Prompt(s string) string {
	if s == "" {
		return s
	}
	out := uuidRE.ReplaceAllString(s, "[ID_REMOVED]")
	out = idKVRE.ReplaceAllString(out, "[ID_REMOVED]")
	return out
}
