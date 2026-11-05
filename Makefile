.PHONY: run build test
run:
	go run ./cmd/server
build:
	go build -o bin/ip-region-service ./cmd/server
test:
	go test -race ./...
