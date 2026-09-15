# gliderGo -- a Go port of Glider PRO (1994, John Calhoun / Casady & Greene).
#
# Run `. scripts/env.sh` first, or use `make` which finds the bootstrapped
# toolchain itself.

GO      ?= $(shell test -x $(HOME)/.local/opt/go/bin/go && echo $(HOME)/.local/opt/go/bin/go || echo go)
BIN     := bin
PKG     := ./...
ASSETS  := assets/extracted

# The version the title screen shows and a bug report quotes. `git describe` in a checkout
# with no tags yet answers with the short hash; outside a checkout it answers nothing, and
# "dev" -- the default in cmd/glidergo -- is then the honest string.
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w
GAMEFLAGS := $(LDFLAGS) -X main.version=$(VERSION)
# The build network has no Go module proxy, and nothing here needs one.
export GOPROXY      := off
export GOTOOLCHAIN  := local

.PHONY: all build glidertool houses run bench smoke headless audio fidelity test vet fmt check \
	clean clean-assets cross-windows assets assets-check tools help

all: build glidertool

## build: compile the game for this host (x11 backend)
build:
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(GAMEFLAGS)' -o $(BIN)/glidergo ./cmd/glidergo

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

## bench: 300 frames flat out on screen, report frame rate (proves the blit path)
bench: build
	$(BIN)/glidergo -frames 300 -bench

# smoke is bench for `check`: the on-screen run is the only part of this Makefile
# that needs a display, and `check` has to pass over SSH and in CI, so a missing
# DISPLAY is reported and skipped rather than failing the build. `make bench`
# still fails without one, because there it is what was asked for.
smoke: build
	@if [ -n "$$DISPLAY" ]; then \
		$(BIN)/glidergo -frames 300 -bench; \
	else \
		echo "smoke: DISPLAY is unset -- skipped the on-screen bench;"; \
		echo "       the blit path is still covered by \`make headless\`."; \
		echo "       Run \`make bench\` from a desktop session to check X11."; \
	fi

## headless: dump 3 game frames and every shell screen as PNGs, with no display
#
# Two halves, because the program has two halves. -frames dumps the game's own frames
# through the null backend; -shot draws the shell, which needs no backend at all -- it
# composes one surface and writes a PNG, so the screens a player meets first are covered on
# a machine with no X server. That is the whole reason -shot exists.
headless:
	@mkdir -p $(BIN)
	$(GO) build -tags nullbackend -ldflags '$(GAMEFLAGS)' -o $(BIN)/glidergo-null ./cmd/glidergo
	$(BIN)/glidergo-null -frames 3 -dump /tmp/glidergo-frames
	@ls -1 /tmp/glidergo-frames
	@for s in splash houses settings about credits scores; do \
		$(BIN)/glidergo-null -shot /tmp/glidergo-shell/$$s.png -shot-screen $$s || exit 1; \
	done
	@# And the first-run screens: no art and no houses is what a fresh clone has, and it is
	@# the one layout nobody developing here ever sees by accident.
	@$(BIN)/glidergo-null -shot /tmp/glidergo-shell/first-run.png \
		-art /nonexistent -houses /nonexistent -quiet 2>/dev/null
	@$(BIN)/glidergo-null -shot /tmp/glidergo-shell/about-no-art.png -shot-screen about \
		-art /nonexistent -quiet
	@ls -1 /tmp/glidergo-shell

## audio: replay 600 frames and write the mix to /tmp/glidergo-audio.wav
#
# The build host has no sound card, so this is the only end-to-end check the audio path can
# get here: it runs the whole chain -- extracted bank, house trigger sounds, channel policy,
# mixer, RIFF writer -- and leaves a file to carry to a machine that does have one. It prints
# the two digests, which is what a bug report quotes, and skipped rather than failed without
# assets, for `make houses`' reason: a fresh clone has none and `check` must still pass.
audio: glidertool
	@if [ -d $(ASSETS)/sound ]; then \
		$(BIN)/glidertool replay -house "CD Demo House" -room 4 -where 423,20 -frames 600 \
			-wav /tmp/glidergo-audio.wav | grep -E 'sound|mix|digest'; \
		ls -l /tmp/glidergo-audio.wav; \
	else \
		echo "no extracted sounds -- run \`make assets\` first"; \
	fi

## fidelity: compare this build's pixels against the checked-in reference corpus
#
# `make test` already runs these, and this step exists for the one thing it cannot do: refuse
# to skip. internal/fidelity's tests skip without an extracted asset tree, because a fresh
# clone has to be able to run its own suite -- so on a machine that *has* the assets, a skip
# is silence where a pixel comparison was supposed to be, and silence is the failure mode this
# whole package exists to prevent. Hence the guard: assets present, and any SKIP is an error.
#
# A failure here is not necessarily a bug. It is a pixel that moved, which is either the
# change you just made -- `go test ./internal/fidelity -update`, read the diff, and say so in
# the commit message -- or a change you did not know you had made.
fidelity:
	@if [ -d $(ASSETS)/art ] && [ -d $(ASSETS)/houses ] && [ -d $(ASSETS)/sound ]; then \
		out=$$($(GO) test ./internal/fidelity/ -count=1 -v 2>&1) || { echo "$$out"; exit 1; }; \
		echo "$$out" | grep -E '^(--- |ok|FAIL)'; \
		if echo "$$out" | grep -q -- '--- SKIP'; then \
			echo "fidelity: a test skipped on a machine that has the assets -- that is a failure"; \
			exit 1; \
		fi; \
	else \
		echo "fidelity: no extracted assets -- run \`make assets\` to check the pixels"; \
	fi

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

## check: everything CI would do; adds an on-screen bench when there is a display
check: fmt vet test build glidertool houses headless audio fidelity cross-windows smoke
	@echo
	@echo "gliderGo: check passed -- toolchain, cgo, tests, houses, headless, audio, pixels and cross-build"

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
