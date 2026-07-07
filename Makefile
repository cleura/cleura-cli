VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  = -X main.version=$(VERSION)

.PHONY: build install test vet

build:
	go build -ldflags "$(LDFLAGS)" -o cleura ./cmd/cleura

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/cleura

test:
	go test ./...

vet:
	go vet ./...
