VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.date=$(DATE)"

.PHONY: build install test

build:
	go build $(LDFLAGS) -o sg .

install: build
	./sg install

test:
	go test ./...
