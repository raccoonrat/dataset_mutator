package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/config"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/gateway"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/logsink"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/policy"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal(err)
	}

	store := policy.NewMemoryStore()
	if err := store.LoadFromFile(cfg.PolicySeedPath); err != nil {
		log.Fatalf("policy seed: %v", err)
	}

	var sink logsink.Sink = logsink.Stdout()
	if !cfg.AsyncLogging {
		sink = logsink.Discard{}
	}

	base := strings.TrimRight(cfg.UpstreamURL.String(), "/")
	h := &gateway.Handler{
		UpstreamBase: base,
		UpstreamClient: &http.Client{
			Timeout: cfg.UpstreamTimeout,
		},
		MaxBody: cfg.MaxBodyBytes,
		Policy:  store,
		Log:     sink,
	}

	log.Printf("gateway listening %s → upstream %s", cfg.ListenAddr, base)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, h))
}
