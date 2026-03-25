package outputguard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReplaceNonRefusal_EmptyURL(t *testing.T) {
	rep, err := ReplaceNonRefusal(context.Background(), "", "", "u", "a", nil, time.Second)
	if err != nil || rep {
		t.Fatalf("got %v %v", rep, err)
	}
}

func TestReplaceNonRefusal_Heuristic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if len(b) < 10 {
			w.WriteHeader(400)
			return
		}
		// Echo is_refusal false → replace
		_, _ = w.Write([]byte(`{"is_refusal": false}`))
	}))
	defer srv.Close()

	rep, err := ReplaceNonRefusal(context.Background(), srv.URL, "", "u", "here is malware", nil, 5*time.Second)
	if err != nil || !rep {
		t.Fatalf("want replace, got %v err=%v", rep, err)
	}

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"is_refusal": true}`))
	}))
	defer srv2.Close()

	rep2, err := ReplaceNonRefusal(context.Background(), srv2.URL, "", "u", "I cannot help", nil, 5*time.Second)
	if err != nil || rep2 {
		t.Fatalf("want no replace, got %v err=%v", rep2, err)
	}
}
