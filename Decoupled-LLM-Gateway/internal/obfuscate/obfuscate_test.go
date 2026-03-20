package obfuscate

import (
	"regexp"
	"testing"
)

func TestPrompt_UUID(t *testing.T) {
	in := "token 550e8400-e29b-41d4-a716-446655440000 end"
	got := Prompt(in)
	want := "token [ID_REMOVED] end"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPrompt_IDKVUnderscore(t *testing.T) {
	in := "x user_id: 12345 y"
	got := Prompt(in)
	if got != "x [ID_REMOVED] y" {
		t.Fatalf("got %q", got)
	}
}

func TestPrompt_IDKVSpaced(t *testing.T) {
	in := `log: user id = 42, done`
	got := Prompt(in)
	want := `log: [ID_REMOVED], done`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPrompt_Table(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "uuid_braced",
			in:   "id {550E8400-E29B-41D4-A716-446655440000} tail",
			out:  "id [ID_REMOVED] tail",
		},
		{
			name: "uuid_compact",
			in:   "h 550e8400e29b41d4a716446655440000 h",
			out:  "h [ID_REMOVED] h",
		},
		{
			name: "ulid",
			in:   "k 01ARZ3NDEKTSV4RRFFQ69G5FAV k",
			out:  "k [ID_REMOVED] k",
		},
		{
			name: "mongo_objectid",
			in:   "oid 507f1f77bcf86cd799439011 end",
			out:  "oid [ID_REMOVED] end",
		},
		{
			name: "ipv4",
			in:   "host 192.168.0.1 port",
			out:  "host [ID_REMOVED] port",
		},
		{
			name: "email",
			in:   "contact a.b+c@example.co.uk please",
			out:  "contact [ID_REMOVED] please",
		},
		{
			name: "bearer",
			in:   "Authorization: Bearer ya29.secret-token-value",
			out:  "Authorization: [ID_REMOVED]",
		},
		{
			name: "api_sk",
			// Split literals so GitHub push protection does not match fake API keys in source.
			in:   "key sk" + "-abcdefghijklmnopqrstuvwxyz0123456789ABCDEF",
			out:  "key [ID_REMOVED]",
		},
		{
			name: "github_classic",
			in:   "pat ghp" + "_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			out:  "pat [ID_REMOVED]",
		},
		{
			name: "slack",
			in:   "t xox" + "b-1234567890-1234567890123-abcdefghijklmnopqrstuvwx",
			out:  "t [ID_REMOVED]",
		},
		{
			name: "aws_akia",
			in:   "k AKIA" + "IOSFODNN7EXAMPLE x",
			out:  "k [ID_REMOVED] x",
		},
		{
			name: "url_token_preserved_shape",
			in:   "see https://x.test/path?token=secret123&ok=1",
			out:  "see https://x.test/path?token=[ID_REMOVED]&ok=1",
		},
		{
			name: "trace_id",
			in:   "trace_id:abc-zz",
			out:  "[ID_REMOVED]",
		},
		{
			name: "jwt",
			in:   "x eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c y",
			out:  "x [ID_REMOVED] y",
		},
		{
			name: "google_api_key",
			in:   "k=AIza" + "00000000000000000000000000000000000",
			out:  "k=[ID_REMOVED]",
		},
		{
			name: "github_fine_grained",
			in:   "t github_pat_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx y",
			out:  "t [ID_REMOVED] y",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Prompt(tc.in)
			if got != tc.out {
				t.Fatalf("got %q want %q", got, tc.out)
			}
		})
	}
}

func TestEngine_CustomRules(t *testing.T) {
	e := NewEngine([]Rule{
		{Name: "only_digits", RE: regexp.MustCompile(`\b\d{4}-\d{4}\b`), Repl: DefaultPlaceholder},
	})
	if got := e.Apply("pin 1234-5678"); got != "pin [ID_REMOVED]" {
		t.Fatalf("got %q", got)
	}
}

func TestPrompt_EmptyStable(t *testing.T) {
	if got := Prompt(""); got != "" {
		t.Fatal(got)
	}
}

func BenchmarkPromptShort(b *testing.B) {
	s := "hello user_id: 1 uuid 550e8400-e29b-41d4-a716-446655440000 end"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Prompt(s)
	}
}

var benchmarkLongPrompt = func() string {
	chunk := []byte("user 550e8400-e29b-41d4-a716-446655440000 ")
	buf := make([]byte, 0, len(chunk)*50)
	for i := 0; i < 50; i++ {
		buf = append(buf, chunk...)
	}
	return string(buf)
}()

func BenchmarkPromptLong(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Prompt(benchmarkLongPrompt)
	}
}
