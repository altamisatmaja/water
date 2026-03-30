.PHONY: help setup build test run clean fmt

BINARY ?= water
VERSION ?= dev
LDFLAGS ?= -X main.Version=$(VERSION)

help:
	@echo "Targets:"
	@echo "  setup   - download Go deps"
	@echo "  build   - build ./cmd/water -> bin/water"
	@echo "  test    - run Go tests"
	@echo "  run     - init + serve using .water-dev"
	@echo "  fmt     - gofmt all Go files"
	@echo "  clean   - remove build artifacts"

setup:
	go mod tidy
	go mod download

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/water

test:
	go test ./...

run: build
	./bin/$(BINARY) init --db-path .water-dev
	./bin/$(BINARY) serve --db-path .water-dev --open-browser=false

fmt:
	gofmt -w .

clean:
	rm -rf bin/ .water-dev/

