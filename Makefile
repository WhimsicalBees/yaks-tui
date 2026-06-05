.PHONY: build test lint run
build:
	go build -o bin/yaks-tui .
test:
	go test ./...
lint:
	go vet ./...
	staticcheck ./... || true
run:
	go run .
