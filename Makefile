APP_NAME=vps-tools

.PHONY: build run tui test test-single fmt lint clean deps

build:
	go build -o bin/$(APP_NAME)

run:
	go run .

tui:
	go run . tui

test:
	go test ./...

test-single:
	@if [ -z "$(TEST)" ]; then echo "Usage: make test-single TEST=path/to/test.go"; exit 1; fi
	go test $(TEST)

fmt:
	gofmt -w .

lint:
	golangci-lint run

clean:
	rm -rf bin

deps:
	go mod download
	go mod tidy