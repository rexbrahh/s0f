GO ?= go
PROFILE ?= ./_dev_profile

.PHONY: all build test run-daemon s0f bridge clean fmt

all: build

build:
	$(GO) build ./...

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

run-daemon:
	$(GO) run ./cmd/bmd --profile $(PROFILE)

s0f:
	$(GO) build -o bin/s0f ./cmd/s0f

bridge:
	$(GO) build -o bin/bmd-bridge ./cmd/bmd-bridge

clean:
	rm -rf bin
