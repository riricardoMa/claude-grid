BINARY_NAME=claude-grid
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test vet clean install check

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

test:
	go test ./... -count=1 -v

vet:
	go vet ./...

clean:
	rm -rf bin/

install:
	go install $(LDFLAGS) .

check: vet test build
