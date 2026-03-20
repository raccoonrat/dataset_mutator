.PHONY: test vet run-echo run-gateway

test:
	go test ./... -race -count=1

vet:
	go vet ./...

run-echo:
	go run ./cmd/echo-llm

run-gateway:
	GATEWAY_UPSTREAM=http://127.0.0.1:9090 go run ./cmd/gateway
