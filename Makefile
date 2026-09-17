# gliderGo -- a Go port of Glider PRO (1994, John Calhoun / Casady & Greene).
#
# If you have Go 1.23 or newer, `make run` works on a fresh clone with no setup: the
# assets are committed, and GO below finds a toolchain on PATH by itself.
# `scripts/bootstrap-dev-env.sh` is only needed when you have no Go, or no internet --
# see docs/DEV_ENVIRONMENT.md.

# A toolchain this repo's bootstrap installed, else whatever `go` is on PATH.
# Override with `make GO=/path/to/go` to build against a specific one.
#
# GLIDERGO_TOOLCHAIN_DIR is read rather than hard-coded so that this agrees with
# scripts/bootstrap-dev-env.sh, which honours the same variable and defaults to the same
# place. Set it there and not here and `make` would quietly build with a different Go
# than the one you just installed.
GLIDERGO_TOOLCHAIN_DIR ?= $(HOME)/.local/opt
GO      ?= $(shell test -x $(GLIDERGO_TOOLCHAIN_DIR)/go/bin/go && echo $(GLIDERGO_TOOLCHAIN_DIR)/go/bin/go || echo go)
BIN     := bin
PKG     := ./...
ASSETS  := assets/extracted
# The archive of that tree which every executable embeds, so that a downloaded binary needs no
# files beside it. It is committed like the tree; `make assets-zip` regenerates it.
ASSETS_ZIP := assets/extracted.zip

# The asset-tree tests every target below shares, so that they cannot drift apart.
#
# assets/extracted/ is committed, so in a clone all three of these are true and nothing below
# skips. They are still here because the tree is output and can legitimately be absent: after
# `make clean-assets`, in a `git archive` that excluded it, or in a half-restored CI cache.
#
# Each one tests for *contents*, not for a directory. `[ -d $(ASSETS)/houses ]` was the old
# test and it says yes to an empty directory -- which is what a half-restored CI cache leaves
# behind. The glob then expands to the literal string `*.house`, and the step fails with
# "no such file or directory" instead of skipping. A cache is either warm or cold; a Makefile
# should not be the thing that discovers it was neither.
HAVE_HOUSES := ls $(ASSETS)/houses/*.house >/dev/null 2>&1
HAVE_SOUND  := [ -s $(ASSETS)/sound/manifest.tsv ]
HAVE_ART    := [ -s $(ASSETS)/art/manifest.json ]
HAVE_ZIP    := [ -s $(ASSETS_ZIP) ]
NO_ASSETS   := echo "   they are committed, so this means they were removed: \`make assets\` puts them back (about a minute)"

# Where the scratch PNGs and WAVs go. Overridable because these paths are fixed, not
# temporary: two CI jobs sharing one self-hosted runner would write to the same files and
# interleave. Use `make check OUT=$$(mktemp -d)` there. The default stays /tmp because the
# point of `make headless` is that you can go and look at the pictures afterwards, and a
# random directory name defeats that.
OUT ?= /tmp

# The version the title screen shows and a bug report quotes. `git describe` in a checkout
# with no tags yet answers with the short hash; outside a checkout it answers nothing, and
# "dev" -- the default in cmd/glidergo -- is then the honest string.
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w
GAMEFLAGS := $(LDFLAGS) -X main.version=$(VERSION)
# Both of these are `?=` so that a machine with an open internet can override them
# from its environment, and `:=` would have silently ignored the attempt.
#
# GOPROXY=off: the airgapped build network has no module proxy, and nothing here
# needs one -- gliderGo is standard library only, which internal/module asserts.
# Keeping it off everywhere turns "quietly grew a dependency" into an error you
# see at once. To evaluate a dependency deliberately:
#     GOPROXY=https://proxy.golang.org,direct make test
#
# GOTOOLCHAIN=local: never download a second toolchain to satisfy go.mod's `go`
# directive. On an airgapped host that download fails with an error about the
# proxy rather than about the version, which sends you looking in the wrong place.
# If your Go is genuinely too old, upgrade it, or let it fetch one:
#     GOTOOLCHAIN=auto GOPROXY=https://proxy.golang.org,direct make check
GOPROXY      ?= off
GOTOOLCHAIN  ?= local
export GOPROXY
export GOTOOLCHAIN

.PHONY: all build glidertool houses run bench smoke headless audio fidelity test vet fmt \
	fmt-check check check-caveats clean clean-assets cross cross-windows assets assets-zip \
	assets-check embedded tools doctor help

all: build glidertool

# embedded guards the one asset every build needs, as opposed to the ones only a run needs.
#
# assets/extracted.zip is a //go:embed input, so a checkout without it does not produce a binary
# that draws nothing -- it does not compile. That is the right failure, but the compiler reports
# it as `pattern extracted.zip: no matching files found`, which does not say what the file is or
# how to get it back. Every target that compiles depends on this instead.
embedded:
	@$(HAVE_ZIP) || { \
		echo "gliderGo: $(ASSETS_ZIP) is missing, and it is built into every executable."; \
		echo "          It is committed: \`git checkout -- $(ASSETS_ZIP)\` restores it,"; \
		echo "          or \`make assets-zip\` rebuilds it from $(ASSETS)."; \
		exit 1; \
	}

## build: compile the game for this host (x11 backend)
build: embedded
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(GAMEFLAGS)' -o $(BIN)/glidergo ./cmd/glidergo

## glidertool: compile the house inspector (`bin/glidertool help`)
glidertool: embedded
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BIN)/glidertool ./cmd/glidertool

## houses: round-trip and sanity-check every extracted house
houses: glidertool
	@if $(HAVE_HOUSES); then \
		$(BIN)/glidertool house check $(ASSETS)/houses/*.house && \
		$(BIN)/glidertool house info $(ASSETS)/houses/*.house | tail -1; \
	else \
		echo "houses: no extracted houses -- skipped"; $(NO_ASSETS); \
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
#
# There is no asset guard here any more, and that is the embed's doing: `-bench` flies
# Slumberland, and Slumberland is inside the executable. A checkout with `make clean-assets`
# run over it still benches.
smoke: build
	@if [ -z "$$DISPLAY" ]; then \
		echo "smoke: DISPLAY is unset -- skipped the on-screen bench;"; \
		echo "       the blit path is still covered by \`make headless\`."; \
		echo "       Run \`make bench\` from a desktop session to check X11."; \
	else \
		$(BIN)/glidergo -frames 300 -bench; \
	fi

## headless: dump 3 game frames and every shell screen as PNGs, with no display
#
# Two halves, because the program has two halves. -frames dumps the game's own frames
# through the null backend; -shot draws the shell, which needs no backend at all -- it
# composes one surface and writes a PNG, so the screens a player meets first are covered on
# a machine with no X server. That is the whole reason -shot exists.
#
# Neither half is guarded on the asset tree, because neither half reads it: both binaries carry
# the houses, the art and the sounds inside them. The guards that used to be here are what the
# embed removed, and the last two lines are what replaced them -- the no-assets layouts are now
# reachable only by naming a directory that is not there, which is what those two do.
headless: embedded
	@mkdir -p $(BIN)
	$(GO) build -tags nullbackend -ldflags '$(GAMEFLAGS)' -o $(BIN)/glidergo-null ./cmd/glidergo
	@# Both output directories are cleared first, because both are listed afterwards and $(OUT)
	@# is a fixed path: without this, a failed run still lists the frames an earlier run left
	@# behind, and the listing reads as work just done.
	@rm -rf $(OUT)/glidergo-frames $(OUT)/glidergo-shell
	$(BIN)/glidergo-null -frames 3 -dump $(OUT)/glidergo-frames
	@ls -1 $(OUT)/glidergo-frames
	@for s in splash houses settings about credits scores; do \
		$(BIN)/glidergo-null -shot $(OUT)/glidergo-shell/$$s.png -shot-screen $$s || exit 1; \
	done
	@# And the layouts a build with no assets shows: an empty picker, and an About box with no
	@# plate behind it. Nobody sees these by accident now -- a binary always has its own copy --
	@# so `-art /nonexistent -houses /nonexistent` is the only way they get drawn at all, and
	@# they are worth drawing because that is also what `-art` pointed at a typo looks like.
	@$(BIN)/glidergo-null -shot $(OUT)/glidergo-shell/first-run.png \
		-art /nonexistent -houses /nonexistent -quiet 2>/dev/null
	@$(BIN)/glidergo-null -shot $(OUT)/glidergo-shell/about-no-art.png -shot-screen about \
		-art /nonexistent -quiet
	@ls -1 $(OUT)/glidergo-shell

## audio: replay 600 frames and write the mix to /tmp/glidergo-audio.wav
#
# The build host has no sound card, so this is the only end-to-end check the audio path can
# get here: it runs the whole chain -- the bank inside the executable, house trigger sounds,
# channel policy, mixer, RIFF writer -- and leaves a file to carry to a machine that does have
# one. It prints the two digests, which is what a bug report quotes.
audio: glidertool
	$(BIN)/glidertool replay -house "CD Demo House" -room 4 -where 423,20 -frames 600 \
		-wav $(OUT)/glidergo-audio.wav | grep -E 'sound|mix|digest'
	@ls -l $(OUT)/glidergo-audio.wav

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
	@if $(HAVE_ART) && $(HAVE_HOUSES) && $(HAVE_SOUND); then \
		out=$$($(GO) test ./internal/fidelity/ -count=1 -v 2>&1) || { echo "$$out"; exit 1; }; \
		echo "$$out" | grep -E '^(--- |ok|FAIL)'; \
		if echo "$$out" | grep -q -- '--- SKIP'; then \
			echo "fidelity: a test skipped on a machine that has the assets -- that is a failure"; \
			exit 1; \
		fi; \
	else \
		echo "fidelity: no extracted assets -- skipped; the pixels were NOT checked"; $(NO_ASSETS); \
	fi

## cross-windows: build the Windows executable (win32 backend, no cgo needed)
cross-windows: embedded
	@mkdir -p $(BIN)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(BIN)/glidergo.exe ./cmd/glidergo
	@# `file` is not installed everywhere -- it is a nicety here, not a check.
	@command -v file >/dev/null && file $(BIN)/glidergo.exe || ls -l $(BIN)/glidergo.exe

## cross: compile every target a release would ship, and say what each one can actually do
#
# What this proves and what it does not, because a green cross-build is easy to over-read.
#
# The three backend selectors are exact logical complements, so every row below resolves to
# exactly one of them (internal/platform/backend/doc.go has them side by side):
#
#   backend_x11.go     linux && cgo && !nullbackend                              x11
#   backend_win32.go   windows && !nullbackend                                   win32
#   backend_null.go    nullbackend || (!linux && !windows) || (!cgo && !windows)  null
#
# The windows rows carry a real backend even at CGO_ENABLED=0, because win32 is pure syscall.
# The darwin rows and cross-compiled linux/arm64 resolve to null and compile happily, which
# makes those rows real evidence that the game logic, the house loader, the asset pipeline and
# the shell are portable -- the whole port except the window -- and NO evidence that the target
# can draw a pixel. Those binaries still do useful work: they run, they take -shot screenshots
# and they dump frames as PNGs with -dump. They just show nothing on screen. macOS is stage 6;
# see docs/PLAN.md.
#
# A green windows row is a weaker claim than a green linux/amd64 +cgo one, and worth saying
# plainly: it means the win32 backend compiles and vets for that architecture, not that it has
# ever opened a window. Nothing on this airgapped Linux host can run it. The first execution
# is ci.yml's `native` job on windows-latest; see internal/platform/win32/win32.go's own note.
#
# linux/arm64 is built with CGO_ENABLED=0 for a different reason: cgo there needs an aarch64
# cross-compiler, and without one the build dies inside runtime/cgo with
# `gcc_arm64.S: Error: no such instruction`. A native arm64 host builds the x11 backend fine,
# so this is a limitation of cross-compiling, not of the port.
#
# The last row -- the host's own cgo build -- used to run inside the `$(...)` that fed printf
# its size, so the `|| fail=1` after it was dead: printf succeeded whatever the compiler did,
# and a broken x11 build left `make cross` exiting 0. It is now an if/elif/else, and a host
# with no x11 pkg-config metadata says `skipped` instead of `FAILED`, because compiling the
# release targets is what this target is for and the host backend is `make build`'s job.
CROSS_TARGETS := windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/arm64 linux/amd64
cross: embedded
	@mkdir -p $(BIN)/cross
	@fail=0; for t in $(CROSS_TARGETS); do \
		os=$${t%%/*}; arch=$${t##*/}; ext=""; be="null backend"; \
		[ "$$os" = windows ] && { ext=".exe"; be="win32 backend (never run here)"; }; \
		out=$(BIN)/cross/glidergo-$$os-$$arch$$ext; \
		if GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -ldflags '$(GAMEFLAGS)' -o $$out ./cmd/glidergo 2>&1; then \
			printf '  %-22s %8s KiB  %s\n' "$$os/$$arch" "$$(( $$(wc -c < $$out) / 1024 ))" "$$be"; \
		else \
			printf '  %-22s FAILED\n' "$$os/$$arch"; fail=1; \
		fi; \
		if GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -o $(BIN)/cross/glidertool-$$os-$$arch$$ext ./cmd/glidertool 2>&1; then \
			:; else printf '  %-22s glidertool FAILED\n' "$$os/$$arch"; fail=1; fi; \
	done; \
	host=$(BIN)/cross/glidergo-linux-amd64-x11; \
	if ! pkg-config --exists x11 2>/dev/null; then \
		printf '  %-22s %8s       x11 backend skipped: no libx11 pkg-config metadata\n' \
			"linux/amd64 +cgo" "--"; \
	elif CGO_ENABLED=1 $(GO) build -ldflags '$(GAMEFLAGS)' -o $$host ./cmd/glidergo; then \
		printf '  %-22s %8s KiB  x11 backend (this host)\n' "linux/amd64 +cgo" \
			"$$(( $$(wc -c < $$host) / 1024 ))"; \
	else \
		printf '  %-22s FAILED\n' "linux/amd64 +cgo"; fail=1; \
	fi; \
	exit $$fail

## test: run the test suite
test:
	$(GO) test $(PKG)

## vet: static checks
vet:
	$(GO) vet $(PKG)

## fmt: gofmt the tree in place -- what to run when fmt-check complains
fmt:
	$(GO) fmt $(PKG)

## fmt-check: report unformatted files without touching them
#
# `check` uses this instead of `fmt`, because a step that rewrites the tree is not a
# verification step -- and the cost was concrete rather than aesthetic. `make check` on a CI
# runner left the worktree dirty, `git describe --tags --always --dirty` (VERSION, above) then
# answered `<hash>-dirty`, and that string is linked into the binary via `-X main.version`. A
# clean build would have shipped announcing itself as dirty.
#
# gofmt is taken from GOROOT rather than PATH: the bootstrapped toolchain is not necessarily on
# PATH (that is what scripts/env.sh is for), but `go env GOROOT` always finds its own.
GOFMT ?= $(shell $(GO) env GOROOT)/bin/gofmt
fmt-check:
	@out=$$($(GOFMT) -l cmd internal 2>&1); \
	if [ -n "$$out" ]; then \
		echo "gliderGo: these files are not gofmt'd:"; \
		echo "$$out" | sed 's/^/  /'; \
		echo "         run \`make fmt\`"; \
		exit 1; \
	else \
		echo "fmt-check: clean"; \
	fi

## check: everything CI would do; adds an on-screen bench when there is a display
check: fmt-check vet test build glidertool houses headless audio fidelity cross smoke
	@echo
	@echo "gliderGo: check passed"
	@$(MAKE) --no-print-directory check-caveats

# check-caveats says what `check` did NOT verify on this machine.
#
# The old summary line read "check passed -- toolchain, cgo, tests, houses, headless, audio,
# pixels and cross-build" unconditionally, and three separate configurations make parts of that
# false while everything still goes green: CGO_ENABLED=0 compiles the null backend instead of
# x11 and nothing complains; an unset DISPLAY skips the on-screen bench by design; and a
# checkout with no extracted assets skips houses, audio and pixels. A hosted CI runner hits all
# three at once. Claiming the pixels were checked when no pixel was compared is the one way a
# fidelity suite can actively mislead, so the summary now enumerates the gaps instead.
#
# One caveat about the caveats: the libx11 branch cannot fire during `make check`, because `vet`
# and `test` compile internal/platform/backend first and die on the missing pkg-config metadata
# with an error of pkg-config's own. It is reachable by running `make check-caveats` directly,
# which is what `make doctor`-adjacent use looks like, and it stays for that. There is no silent
# fallback to the null backend: that is `make headless`, asked for by name.
check-caveats:
	@n=0; \
	if [ "$$($(GO) env CGO_ENABLED)" != "1" ]; then \
		echo "  - cgo is off, so the x11 backend was NOT compiled; \`build\` produced the null backend"; n=1; \
	elif ! pkg-config --exists x11 2>/dev/null; then \
		echo "  - libx11 dev metadata is missing, so the x11 backend was NOT compiled"; n=1; \
	fi; \
	if ! $(HAVE_HOUSES) || ! $(HAVE_ART) || ! $(HAVE_SOUND); then \
		echo "  - no extracted asset tree: the house round-trip and the pixel corpus were NOT"; \
		echo "    checked (the runs above used the copy inside the binaries and are unaffected)"; n=1; \
	fi; \
	if [ -z "$$DISPLAY" ]; then \
		echo "  - DISPLAY is unset, so the on-screen blit was NOT exercised"; n=1; \
	fi; \
	if [ $$n -eq 0 ]; then \
		echo "         toolchain, cgo, tests, houses, headless, audio, pixels, cross-build and the blit path"; \
	else \
		echo "         everything above ran, but note the gaps -- this was not a full check"; \
	fi

## assets: re-extract assets/extracted/ and repack the archive the binaries embed
#
# Nobody needs to run this to play: both the tree and the archive it writes are in the
# repository. It is here for three cases -- changing the extractor, restoring the tree after
# `make clean-assets`, and regenerating the houses/*.rsrc intermediates, which are the one part
# not committed.
#
# It packs as well as extracts, because the executables read the archive and not the tree: an
# extraction that stopped at the tree would leave a developer looking at a new PNG in git status
# and the old one on screen. `go test ./assets` is the backstop that says the two agree.
assets:
	python3 tools/extract_all.py
	@$(MAKE) --no-print-directory assets-zip

## assets-zip: repack assets/extracted.zip from assets/extracted/ without re-extracting
#
# The second half of `make assets` on its own, for a tree that changed by some other means: a
# hand-authored house dropped in, or a file restored from git.
assets-zip:
	$(GO) run ./tools/packassets

## assets-check: prove the committed asset tree is exactly what the extractor produces
#
# The strong version of what used to be a manifest diff. It re-extracts to a temp tree and
# compares every file, which is the check that matters now that the output is committed: it
# catches a hand-edited asset, a partial commit, and a checkout that mangled a byte (see
# .gitattributes on why that was a real risk on Windows).
#
# houses/*.rsrc is excluded because it is deliberately not committed -- see .gitignore.
#
# The archive is not compared here. `go test ./assets` does that, file by file, in the ordinary
# test run -- so the two halves of the claim divide as "the tree is what the extractor makes"
# (this target, which needs python3 and a minute) and "the binaries carry the tree" (a test,
# which needs neither).
assets-check:
	@rm -rf $(OUT)/glidergo-assets-check
	@python3 tools/extract_all.py --out $(OUT)/glidergo-assets-check >/dev/null \
		|| { echo "assets-check: the extractor failed (its error is above)"; exit 1; }
	@diff -r -x '*.rsrc' $(ASSETS) $(OUT)/glidergo-assets-check \
		&& echo "assets: the committed tree is byte-for-byte what tools/extract_all.py produces"

## doctor: report what this machine has, what it is missing, and which networks it can see
#
# The first thing to run when a build fails on a machine nobody has built on
# before. It installs nothing.
doctor:
	@./scripts/bootstrap-dev-env.sh --check

## tools: list the extraction and packing tools (each is a standalone CLI)
tools:
	@ls tools/*.py 2>/dev/null || echo "no extraction scripts yet"
	@# The Go tools, by the file that makes one a command rather than by directory, so that a
	@# stray __pycache__ is not listed as something to `go run`.
	@ls tools/*/main.go 2>/dev/null | sed 's|/main.go$$||;s|^|go run ./|' || true

## clean: remove build output (not assets/extracted -- use clean-assets)
clean:
	rm -rf $(BIN)

## clean-assets: remove assets/extracted/; `make assets` regenerates it
#
# This deletes committed files, so `git status` will have plenty to say afterwards.
# `git checkout -- assets/extracted` restores them without re-running the extractor.
#
# It leaves assets/extracted.zip alone, deliberately, and that is worth knowing: the archive is
# a build input, so removing it stops the build, and a binary built without the tree present
# still plays the 22 houses. What a cleaned tree costs is the extractor's own checks -- `make
# houses`, `make assets-check` and the fidelity corpus, which read files rather than the archive.
clean-assets:
	rm -rf $(ASSETS)

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
