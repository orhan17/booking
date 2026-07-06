.PHONY: run test lint

# Run the API locally.
run:
	go run ./cmd/api

# Run the full test suite with the race detector.
test:
	go test ./... -race

# Lint the codebase.
lint:
	golangci-lint run
