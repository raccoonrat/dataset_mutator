.PHONY: test vet bench-obfuscate run-echo run-gateway redis-up

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
