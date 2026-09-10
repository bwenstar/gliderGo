# gliderGo -- a Go port of Glider PRO (1994, John Calhoun / Casady & Greene).
#
# Run `. scripts/env.sh` first, or use `make` which finds the bootstrapped
# toolchain itself.

GO      ?= $(shell test -x $(HOME)/.local/opt/go/bin/go && echo $(HOME)/.local/opt/go/bin/go || echo go)
BIN     := bin
PKG     := ./...
LDFLAGS := -s -w
# The build network has no Go module proxy, and nothing here needs one.
export GOPROXY      := off
export GOTOOLCHAIN  := local

.PHONY: all build run bench headless test vet fmt check clean cross-windows tools help

all: build

## build: compile the game for this host (x11 backend)
build:
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN)/glidergo ./cmd/glidergo

## run: build and run windowed at 1:1; pass flags with ARGS='-scale 2'
run: build
	$(BIN)/glidergo $(ARGS)

## bench: 300 frames flat out, report frame rate (proves the blit path)
bench: build
	$(BIN)/glidergo -frames 300 -bench

## headless: build the null backend and dump 3 frames as PNGs to /tmp/glidergo-frames
headless:
	@mkdir -p $(BIN)
	$(GO) build -tags nullbackend -o $(BIN)/glidergo-null ./cmd/glidergo
	$(BIN)/glidergo-null -frames 3 -dump /tmp/glidergo-frames
	@ls -1 /tmp/glidergo-frames

## cross-windows: prove the Windows target still compiles (null backend until win32 lands)
cross-windows:
	@mkdir -p $(BIN)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(BIN)/glidergo.exe ./cmd/glidergo
	@file $(BIN)/glidergo.exe

## test: run the test suite
test:
	$(GO) test $(PKG)

## vet: static checks
vet:
	$(GO) vet $(PKG)

## fmt: gofmt the tree
fmt:
	$(GO) fmt $(PKG)

## check: everything CI would do, plus an on-screen smoke test
check: fmt vet test build headless cross-windows bench
	@echo
	@echo "gliderGo: environment OK -- toolchain, cgo, X11, headless and cross-build all work"

## tools: run the asset extraction pipeline (python3, no third-party modules)
tools:
	@ls tools/*.py 2>/dev/null || echo "no extraction tools yet"

## clean: remove build output
clean:
	rm -rf $(BIN)

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
