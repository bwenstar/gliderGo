# gliderGo -- a Go port of Glider PRO (1994, John Calhoun / Casady & Greene).
#
# Run `. scripts/env.sh` first, or use `make` which finds the bootstrapped
# toolchain itself.

GO      ?= $(shell test -x $(HOME)/.local/opt/go/bin/go && echo $(HOME)/.local/opt/go/bin/go || echo go)
BIN     := bin
PKG     := ./...
ASSETS  := assets/extracted
LDFLAGS := -s -w
# The build network has no Go module proxy, and nothing here needs one.
export GOPROXY      := off
export GOTOOLCHAIN  := local

.PHONY: all build glidertool houses run bench headless test vet fmt check \
	clean clean-assets cross-windows assets assets-check tools help

all: build glidertool

## build: compile the game for this host (x11 backend)
build:
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN)/glidergo ./cmd/glidergo

## glidertool: compile the house inspector (`bin/glidertool help`)
glidertool:
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN)/glidertool ./cmd/glidertool

## houses: round-trip and sanity-check every extracted house
houses: glidertool
	@if [ -d $(ASSETS)/houses ]; then \
		$(BIN)/glidertool house check $(ASSETS)/houses/*.house && \
		$(BIN)/glidertool house info $(ASSETS)/houses/*.house | tail -1; \
	else \
		echo "no extracted houses -- run \`make assets\` first"; \
	fi

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
check: fmt vet test build glidertool houses headless cross-windows bench
	@echo
	@echo "gliderGo: environment OK -- toolchain, cgo, X11, headless and cross-build all work"

## assets: extract the 1994 art, sound, houses and movies into assets/extracted/
assets:
	python3 tools/extract_all.py

## assets-check: re-extract to a temp tree and prove the pipeline is deterministic
assets-check:
	@rm -rf /tmp/glidergo-assets-check
	@python3 tools/extract_all.py --out /tmp/glidergo-assets-check >/dev/null 2>&1
	@diff $(ASSETS)/manifest.json /tmp/glidergo-assets-check/manifest.json \
		&& echo "assets: reproducible -- manifests identical"

## tools: list the extraction/inspection scripts (each is a standalone CLI)
tools:
	@ls tools/*.py 2>/dev/null || echo "no extraction tools yet"

## clean: remove build output (not assets/extracted -- use clean-assets)
clean:
	rm -rf $(BIN)

## clean-assets: remove extracted assets; `make assets` regenerates them
clean-assets:
	rm -rf $(ASSETS)

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
