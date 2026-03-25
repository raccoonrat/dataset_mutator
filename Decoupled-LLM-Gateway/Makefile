.PHONY: test vet bench-obfuscate run-echo run-gateway redis-up paper-eval-check smoke-output-guard

# Go refuses to build when GOPATH and GOMODCACHE are both empty (some shells/IDEs export empty GOPATH).
ifeq ($(strip $(GOMODCACHE)),)
ifneq ($(strip $(GOPATH)),)
export GOMODCACHE := $(GOPATH)/pkg/mod
else
export GOMODCACHE := $(HOME)/go/pkg/mod
endif
endif

# When proxy.golang.org times out (e.g. restricted network), try public mirrors in order.
ifeq ($(strip $(GOPROXY)),)
export GOPROXY := https://goproxy.cn,https://goproxy.io,https://proxy.golang.org,direct
endif

test:
	go test ./... -race -count=1

vet:
	go vet ./...

bench-obfuscate:
	go test ./internal/obfuscate/ -bench=. -benchmem -count=5

run-echo:
	go run ./cmd/echo-llm

run-gateway:
	GATEWAY_UPSTREAM=http://127.0.0.1:9090 go run ./cmd/gateway

redis-up:
	docker compose up -d redis

# Paper eval harness sanity (no network; CI uses this)
paper-eval-check:
	python3 experiments/run_paper_benchmark.py --self-check

# Echo + heuristic judge + gateway output guard (local TCP; no external LLM API)
smoke-output-guard:
	bash experiments/scripts/smoke_output_guard.sh
