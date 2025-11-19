GO ?= go
PROFILE ?= ./_dev_profile

.PHONY: all build test run-daemon build-daemon build-cli build-bridge clean fmt smoke

all: build

build:
	$(GO) build ./...

build-daemon:
	$(GO) build -o bin/bmd ./cmd/bmd

build-cli:
	$(GO) build -o bin/s0f ./cmd/s0f

build-bridge:
	$(GO) build -o bin/bmd-bridge ./cmd/bmd-bridge

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

run-daemon:
	$(GO) run ./cmd/bmd --profile $(PROFILE)

smoke:
	./scripts/smoke.sh

clean:
	rm -rf bin
