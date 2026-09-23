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

# The port's own houses, in the same three parts: levels/*.house.txt is source, $(LEVELS) is
# what `make levels` builds from it, and $(LEVELS_ZIP) is that directory packed for //go:embed.
#
# Here the archive is committed and the directory is not, which is the reverse of the pair
# above and follows from the build order rather than from taste: `make levels` runs
# bin/glidertool, glidertool embeds $(LEVELS_ZIP), so the archive has to be in hand before the
# tool that fills $(LEVELS) will compile. .gitignore makes the same argument at more length.
LEVELS     := assets/levels
LEVELS_ZIP := assets/levels.zip

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
# CAN_RACE answers whether `go test -race` can run on this machine. Both halves are asked because
# either alone lies: the detector is cgo-only, and CGO_ENABLED is 1 by default on a box with no C
# compiler at all, where the failure is a linker error that says nothing about the detector.
CAN_RACE    := [ "$$($(GO) env CGO_ENABLED)" = "1" ] && command -v "$$($(GO) env CC)" >/dev/null 2>&1
HAVE_LEVELS_ZIP := [ -s $(LEVELS_ZIP) ]
NO_ASSETS   := echo "   they are committed, so this means they were removed: \`make assets\` puts them back (about a minute)"

# Where the scratch PNGs and WAVs go. Overridable because these paths are fixed, not
# temporary: two CI jobs sharing one self-hosted runner would write to the same files and
# interleave. Use `make check OUT=$$(mktemp -d)` there. The default stays /tmp because the
# point of `make headless` is that you can go and look at the pictures afterwards, and a
# random directory name defeats that.
OUT ?= /tmp

# The version the title screen shows and a bug report quotes. `git describe` in a checkout
# with no tags fetched answers with the short hash (`git fetch --tags` fixes that; RELEASING.md,
# step 1); outside a checkout it answers nothing, and
# "dev" -- the default in cmd/glidergo -- is then the honest string.
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w
# Both commands carry the version now, and both have a `var version = "dev"` in package main
# for it to land in. glidertool went four stages without one, which mattered more than it
# sounds: every line that tool prints is an assertion about somebody else's data, and a lint
# report or a replay digest pasted into a bug report is only comparable between two machines
# if both can say which build produced it.
STAMPED := $(LDFLAGS) -X main.version=$(VERSION)
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

.PHONY: all build glidertool houses levels levels-zip run bench smoke headless audio fidelity \
	test race fuzz vet fmt fmt-check check check-caveats docs-check clean clean-assets cross cross-windows assets \
	assets-zip assets-check embedded tools doctor help

## all: compile both binaries -- the default target, so `make` on its own does this
all: build glidertool

# embedded guards the assets every build needs, as opposed to the ones only a run needs.
#
# Both archives are //go:embed inputs, so a checkout without one does not produce a binary that
# draws nothing -- it does not compile. That is the right failure, but the compiler reports it as
# `pattern extracted.zip: no matching files found`, which does not say what the file is or how to
# get it back. Every target that compiles depends on this instead.
embedded:
	@$(HAVE_ZIP) || { \
		echo "gliderGo: $(ASSETS_ZIP) is missing, and it is built into every executable."; \
		echo "          It is committed: \`git checkout -- $(ASSETS_ZIP)\` restores it,"; \
		echo "          or \`make assets-zip\` rebuilds it from $(ASSETS)."; \
		exit 1; \
	}
	@# The second clause has only the one remedy, and that is not an oversight. $(LEVELS_ZIP)
	@# is rebuilt from $(LEVELS), $(LEVELS) is written by bin/glidertool, and glidertool is
	@# one of the binaries that will not compile while the archive is missing. Telling anyone
	@# to run `make levels-zip` out of this hole would be telling them to go round it.
	@$(HAVE_LEVELS_ZIP) || { \
		echo "gliderGo: $(LEVELS_ZIP) is missing, and it is built into every executable."; \
		echo "          It is committed: \`git checkout -- $(LEVELS_ZIP)\` restores it."; \
		echo "          Rebuilding it needs bin/glidertool, which needs this file, so the"; \
		echo "          checkout is the way out and not \`make levels\`."; \
		exit 1; \
	}

## build: compile the game for this host (x11 backend)
build: embedded
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(STAMPED)' -o $(BIN)/glidergo ./cmd/glidergo

## glidertool: compile the house inspector (`bin/glidertool help`)
glidertool: embedded
	@mkdir -p $(BIN)
	$(GO) build -ldflags '$(STAMPED)' -o $(BIN)/glidertool ./cmd/glidertool

## houses: round-trip and sanity-check every extracted house
houses: glidertool
	@if $(HAVE_HOUSES); then \
		$(BIN)/glidertool house check $(ASSETS)/houses/*.house && \
		$(BIN)/glidertool house info $(ASSETS)/houses/*.house | tail -1; \
	else \
		echo "houses: no extracted houses -- skipped"; $(NO_ASSETS); \
	fi

## levels: build the port's own houses from levels/*.house.txt and validate them
#
# levels/ is source and $(LEVELS) is output, which is the one thing to know about this
# target. A house is a binary file with 348 bytes per room in it; authoring one means
# editing the text and running this, the same way the extracted art is a build of the
# 1994 resource forks. The output is *not* committed, unlike assets/extracted/ -- see the
# note in .gitignore for why those two differ.
#
# The name mapping is nothing, on purpose: `levels/X.house.txt` becomes `$(LEVELS)/X.house`
# and the picker lists it as "X", because a house is called whatever its file is called and
# houseType has no name field to disagree with. That is why the source file has a space in
# it.
#
# `-fail warn` and not the default `-fail error`: a shipped house may carry the odd warning
# for reasons that are now history (27 of the originals set `bounds` on a built-in
# background), but a house written this week has no such excuse, so anything the linter will
# say out loud has to be fixed or argued with in the text. docs/analysis/original-houses.md
# 10.5 is the list of things that get you one.
#
# The houses built here are also inside the executable, via $(LEVELS_ZIP), and that is what a
# player gets. This directory is for looking at the build product and for `-levels`:
#
#     bin/glidergo -levels $(LEVELS)
#
# replaces the embedded set with whatever is in it, which is how you play a house you are in
# the middle of writing without repacking anything.
#
# A house here may carry pictures of its own, in `levels/houseart/<House Name>/pict/<id>.png`,
# copied through to the same path under $(LEVELS) and so into the archive and every executable.
# That is the whole of docs/IMPROVEMENTS.md 4.15: before it, the only way to give a house art was
# -houseart, and -houseart worked by *replacing* the root that half the 1994 houses read from, so
# a new house could have pictures or the originals could, never both. Nothing is committed under
# levels/houseart yet -- the port's two houses are drawn entirely with built-in backgrounds, and
# painting one is an art task rather than an engineering one -- so today this copies nothing.
levels: glidertool
	@mkdir -p $(LEVELS)
	@# Clear the previous build's houses first, because this directory is compared against the
	@# archive below and a leftover is indistinguishable from a house. Rename or delete
	@# levels/X.house.txt and, without this, assets/levels/X.house survives as an orphan: the
	@# comparison then fails, the remedy it prints (`make levels-zip`) packs the orphan *into*
	@# the archive, and `go test ./assets` is left to catch a house nobody wrote. The glob is
	@# narrow on purpose -- only this target's own output, not the directory.
	@rm -f $(LEVELS)/*.house
	@# The art goes the same way and for the same reason, except that the whole directory is the
	@# unit: a picture deleted from levels/houseart/ has to disappear from the archive too, and
	@# `cp -r` over the top of the old tree would leave it. Removed before the test for the
	@# source, not after, so that deleting levels/houseart entirely is a complete removal rather
	@# than a state where the archive keeps the last copy for ever.
	@rm -rf $(LEVELS)/houseart
	@if [ -d levels/houseart ]; then cp -r levels/houseart $(LEVELS)/houseart; fi
	@set -e; for src in levels/*.house.txt; do \
		out="$(LEVELS)/$$(basename "$$src" .txt)"; \
		$(BIN)/glidertool house build -o "$$out" "$$src"; \
		$(BIN)/glidertool house lint -fail warn -houseart $(LEVELS)/houseart "$$out"; \
	done
	@# Read-only, and the reason it is here rather than in `levels-zip` is that this is the
	@# moment the two can disagree: the text just changed, the directory has caught up, and the
	@# archive -- the copy a player actually loads -- has not. Nothing else in `make check`
	@# notices. `go test ./assets` says the same thing from the other direction and does it
	@# without the tree, which is why both exist.
	@$(GO) run ./tools/packassets -tree $(LEVELS) -out $(LEVELS_ZIP) -check || { \
		echo "levels: $(LEVELS_ZIP) no longer matches $(LEVELS)/, so the houses inside every"; \
		echo "        executable are the old ones. \`make levels-zip\` repacks it, and the"; \
		echo "        result is committed."; \
		exit 1; \
	}
	@echo "levels: built into $(LEVELS)/ and matching $(LEVELS_ZIP) -- play the build product"
	@echo "        directly with \`$(BIN)/glidergo -levels $(LEVELS)\`"

## run: build and run windowed at 1:1; pass flags with ARGS='-scale 2'
run: build
	$(BIN)/glidergo $(ARGS)

## bench: frame rate and CPU on screen, at 1x and 4x (proves the blit path)
#
# Three rows, because docs/IMPROVEMENTS.md 2.76's budget is in two currencies: flat out at 1x
# and at 4x for the rate the machine sustains, and paced at 4x for what a player's machine is
# charged for the game it is actually playing, which is the CPU line of the last row. -scale is
# explicit on all three: a timed run never asks the display (2.53), so without it this would be
# 1x three times, and an explicit scale is kept even on a screen too small for it, with a
# warning, rather than quietly measuring a smaller window. 4x is a 2560x1920 window; CI's Xvfb
# is sized for it.
bench: build
	$(BIN)/glidergo -frames 300 -bench -scale 1
	$(BIN)/glidergo -frames 300 -bench -scale 4
	$(BIN)/glidergo -frames 150 -scale 4

## smoke: the on-screen bench `check` runs, skipped with a note where it cannot
#
# smoke is bench for `check`: the on-screen run is the only part of this Makefile
# that needs a display, and `check` has to pass over SSH and in CI, so a missing
# DISPLAY is reported and skipped rather than failing the build. `make bench`
# still fails without one, because there it is what was asked for.
#
# `DISPLAY` is an X11 question, so off Linux the note has to be a different note
# (docs/IMPROVEMENTS.md 4.13). Windows sets no DISPLAY and needs none -- the win32 backend
# asks the OS for a window -- so the old message skipped for a reason that was not the reason,
# and told a Windows reader to go and check X11. The behaviour is unchanged, because nothing
# in a Makefile can decide from outside whether a window would open on Windows; what changes
# is that it now says that, and names the target that would find out.
# `GOOS=windows make smoke BIN=/tmp/x` prints it, which is how it was read from here.
#
# There is no asset guard here any more, and that is the embed's doing: `-bench` flies
# Slumberland, and Slumberland is inside the executable. A checkout with `make clean-assets`
# run over it still benches.
smoke: build
	@if [ "$$($(GO) env GOOS)" = windows ]; then \
		echo "smoke: not attempted on windows -- DISPLAY is not what decides a window here, and"; \
		echo "       nothing in this Makefile can tell whether one would open. \`make bench\` is"; \
		echo "       the on-screen run, and it is the one that would say so."; \
	elif [ -z "$$DISPLAY" ]; then \
		echo "smoke: DISPLAY is unset -- skipped the on-screen bench;"; \
		echo "       what it draws is still covered by \`make headless\`, which renders the same"; \
		echo "       frames through the null backend. The blit itself is not: run \`make bench\`"; \
		echo "       from a desktop session for that."; \
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
	$(GO) build -tags nullbackend -ldflags '$(STAMPED)' -o $(BIN)/glidergo-null ./cmd/glidergo
	@# Both output directories are cleared first, because both are listed afterwards and $(OUT)
	@# is a fixed path: without this, a failed run still lists the frames an earlier run left
	@# behind, and the listing reads as work just done.
	@rm -rf $(OUT)/glidergo-frames $(OUT)/glidergo-shell
	$(BIN)/glidergo-null -frames 3 -dump $(OUT)/glidergo-frames
	@ls -1 $(OUT)/glidergo-frames
	@for s in splash houses settings race about credits scores; do \
		$(BIN)/glidergo-null -shot $(OUT)/glidergo-shell/$$s.png -shot-screen $$s || exit 1; \
	done
	@# And the layouts a build with no assets shows: an empty picker, and an About box with no
	@# plate behind it. Nobody sees these by accident now -- a binary always has its own copy --
	@# so naming three directories that are not there is the only way they get drawn at all, and
	@# they are worth drawing because that is also what `-art` pointed at a typo looks like.
	@#
	@# Three, because there are two house roots now. Without `-levels /nonexistent` this shot
	@# lists the port's own houses and stops being the empty picker it is here for -- which is
	@# the useful half of the news that every binary carries houses of its own.
	@$(BIN)/glidergo-null -shot $(OUT)/glidergo-shell/first-run.png \
		-art /nonexistent -houses /nonexistent -levels /nonexistent -quiet 2>/dev/null
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
# ever opened a window. Nothing on this airgapped Linux host can run it. amd64 has been run
# elsewhere -- on a Windows Server 2025 desktop, see docs/windows-first-run.md -- and arm64 has
# not been run at all, which is why only the amd64 row says so below.
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
#
# That row is also the only thing here that is about the machine you are standing on rather than
# about a release target, so it asks `go env GOOS` before it says anything. A macOS reader used to
# be told that `linux/amd64 +cgo` had skipped the x11 backend for want of libX11 pkg-config
# metadata -- a caveat about an operating system they are not using, with a remedy that would do
# them no good (docs/IMPROVEMENTS.md 4.13). Off Linux the cgo build is not attempted at all,
# because there is nothing left for it to prove: darwin resolves to the null backend with cgo on
# or off, and windows to win32 either way, so the row above it already carries that claim. The
# binary keeps its `-x11` suffix, because .github/workflows/release.yml packages
# `bin/cross/glidergo-linux-amd64-x11` under that exact name. `GOOS=darwin make cross` prints the
# row a macOS reader gets without building anything for it, which is how the three non-Linux
# wordings were read from here; see check-caveats below for why that is the only test they have.
CROSS_TARGETS := windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/arm64 linux/amd64
cross: embedded
	@mkdir -p $(BIN)/cross
	@fail=0; for t in $(CROSS_TARGETS); do \
		os=$${t%%/*}; arch=$${t##*/}; ext=""; be="null backend"; \
		[ "$$os" = windows ] && { ext=".exe"; be="win32 backend (run on Server 2025)"; }; \
		[ "$$t" = windows/arm64 ] && be="win32 backend (never run on arm64)"; \
		out=$(BIN)/cross/glidergo-$$os-$$arch$$ext; \
		if GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -ldflags '$(STAMPED)' -o $$out ./cmd/glidergo 2>&1; then \
			printf '  %-22s %8s KiB  %s\n' "$$os/$$arch" "$$(( $$(wc -c < $$out) / 1024 ))" "$$be"; \
		else \
			printf '  %-22s FAILED\n' "$$os/$$arch"; fail=1; \
		fi; \
		if GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -ldflags '$(STAMPED)' -o $(BIN)/cross/glidertool-$$os-$$arch$$ext ./cmd/glidertool 2>&1; then \
			:; else printf '  %-22s glidertool FAILED\n' "$$os/$$arch"; fail=1; fi; \
	done; \
	hostos=$$($(GO) env GOOS); hostarch=$$($(GO) env GOARCH); \
	host=$(BIN)/cross/glidergo-$$hostos-$$hostarch-x11; \
	if [ "$$hostos" != linux ]; then \
		case $$hostos in \
			windows) why="not attempted: win32 needs no cgo, so the row above is it";; \
			darwin)  why="not attempted: cgo buys macOS nothing until stage 6";; \
			*)       why="not attempted: cgo buys this OS nothing -- no backend";; \
		esac; \
		printf '  %-22s %8s       %s\n' "$$hostos/$$hostarch +cgo" "--" "$$why"; \
	elif ! pkg-config --exists x11 2>/dev/null; then \
		printf '  %-22s %8s       x11 backend skipped: no libx11 pkg-config metadata\n' \
			"$$hostos/$$hostarch +cgo" "--"; \
	elif CGO_ENABLED=1 $(GO) build -ldflags '$(STAMPED)' -o $$host ./cmd/glidergo; then \
		printf '  %-22s %8s KiB  x11 backend (this host)\n' "$$hostos/$$hostarch +cgo" \
			"$$(( $$(wc -c < $$host) / 1024 ))"; \
	else \
		printf '  %-22s FAILED\n' "$$hostos/$$hostarch +cgo"; fail=1; \
	fi; \
	exit $$fail

## test: run the test suite
test:
	$(GO) test $(PKG)

## race: run the code that starts goroutines under the race detector
#
# Three places start one, and this runs the tests that reach each of them:
#
#   - `internal/netplay` reads on one goroutine and sends from another for the whole of a match;
#   - `internal/audio`'s Pipe writes to the player on its own, so that the game never waits on a
#     sound server (pipe_test.go; WaveOut, the Windows twin, cannot be raced from here);
#   - `cmd/glidergo` shakes hands on one behind the race's waiting screen, and drives netplay
#     from the frame loop -- the loopback tests, and only those, because the rest of that
#     package is single-threaded and takes twenty seconds without the detector.
#
# Nothing else is concurrent. That is why `test` above does not pass -race and should not -- the
# detector needs cgo and a C compiler, it costs an order of magnitude, and it would buy nothing on
# the other packages. Keeping the concurrency in a few small places is what makes this
# affordable, and is the reason it was designed that way rather than the consequence. A patch that
# starts a goroutine anywhere else adds its package here in the same patch.
#
# -count=2 because the detector only reports races it observes, and a scheduler that happened to
# interleave two goroutines safely once will not say so. Two runs is not a proof; it is the
# cheapest thing better than one.
#
# Skipped rather than failed where it cannot run, for the same reason `cross` skips its host row:
# a check a machine cannot perform is not a defect in the code, and `check-caveats` says so at the
# end so that a green run does not claim it.
race:
	@if $(CAN_RACE); then \
		$(GO) test -race -count=2 ./internal/netplay/ ./internal/audio/ && \
		$(GO) test -race -count=2 -run 'Race|Loopback' ./cmd/glidergo/; \
	else \
		echo "race: skipped -- the detector needs cgo and a C compiler, and this machine has"; \
		echo "      CGO_ENABLED=$$($(GO) env CGO_ENABLED) with CC=$$($(GO) env CC)"; \
	fi

## fuzz: run every fuzz target for FUZZTIME (default 30s), then SOAK damaged-house plays
#
# Opt-in, and not part of `check`: it takes about six minutes, and what it finds is new rather
# than a regression (docs/IMPROVEMENTS.md 4.30). Plain `go test` already replays every target's
# seeds, and every input committed under a package's testdata/fuzz/.
#
# Go's -fuzz takes one target a run, so this finds them all by name and runs each in turn. A new
# `func Fuzz...` joins without an edit here. A target that fails has written its input to
# testdata/fuzz/<target>/ in its package: commit that file with the fix, and from then on plain
# `go test` replays it. The loop carries on past a failure so that one run reports all of them.
#
# -fuzzminimizetime is short because the seeds include shipped houses: minimising a 17 KB input
# the default way spends the whole budget on a handful of executions.
#
# SOAK is TestADamagedHouseStillPlays's -soak: that many shipped houses damaged and played, about
# 40 ms each.
FUZZTIME ?= 30s
SOAK     ?= 3000
fuzz:
	@fail=0; \
	for file in $$(grep -rl --include='*_test.go' '^func Fuzz' cmd internal tools | sort); do \
		dir=./$$(dirname $$file); \
		for target in $$(sed -n 's/^func \(Fuzz[A-Za-z0-9_]*\)(.*/\1/p' $$file); do \
			echo "fuzz: $$dir $$target, $(FUZZTIME)"; \
			$(GO) test $$dir -run '^$$' -fuzz "^$$target\$$" -fuzztime $(FUZZTIME) \
				-fuzzminimizetime 5s || { fail=1; echo "fuzz: $$dir $$target FAILED"; }; \
		done; \
	done; \
	echo "fuzz: $(SOAK) damaged-house plays"; \
	$(GO) test ./internal/replay -run '^TestADamagedHouseStillPlays$$' -soak $(SOAK) || fail=1; \
	if [ $$fail = 0 ]; then echo "fuzz: every target and the soak passed"; fi; \
	exit $$fail

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
#
# The four directories are every directory in here that holds Go, which is the point: `fmt` above
# uses $(PKG) and so reformats all four, and for a while this read `cmd internal`, so an
# unformatted assets/assets.go or tools/packassets/main.go passed `make check` and was then
# rewritten by the fix for a complaint about something else. gofmt walks assets/extracted/'s 1,878
# files to find no Go in them, and takes 0.1 s doing it.
GOFMT ?= $(shell $(GO) env GOROOT)/bin/gofmt
fmt-check:
	@out=$$($(GOFMT) -l cmd internal assets tools 2>&1); \
	if [ -n "$$out" ]; then \
		echo "gliderGo: these files are not gofmt'd:"; \
		echo "$$out" | sed 's/^/  /'; \
		echo "         run \`make fmt\`"; \
		exit 1; \
	else \
		echo "fmt-check: clean"; \
	fi

## docs-check: run the command lines README.md and CONTRIBUTING.md tell people to run
#
# The documents make claims like any other part of this repository, and until this target they
# were the only claims nothing checked. docs/IMPROVEMENTS.md 4.13 is a list of defects in them --
# a command in replay's own usage text that could not run, a `glidergo <house>` that was silently
# ignored, two sentences about a build that were wrong in opposite directions -- and every one was
# found by a person typing a line into a shell.
#
# tools/docscheck reads every ```bash fence in both documents, runs what can be run unattended in
# a scratch directory of symlinks, and prints what it did not run and why: the same bargain
# `check-caveats` makes for the rest of this file. A documented line that no rule classifies is a
# failure, which is how the target keeps up with the documents rather than falling behind them.
#
# Both binaries, because the documented lines name `bin/glidergo` and `bin/glidertool` by path.
# In `check` it sits after `fidelity` rather than beside `glidertool`, and that is deliberate: if
# the house round-trip is broken, `houses` should be the target that says so -- not a README line
# failing for a reason the README has nothing to do with.
docs-check: build glidertool
	$(GO) run ./tools/docscheck

## check: everything CI would do; adds an on-screen bench when there is a display
#
# `embedded` leads, ahead of even fmt-check, and only for the error message. Both archives are
# //go:embed inputs, so a checkout missing one fails at `vet` -- the third step -- with the
# compiler's `pattern levels.zip: no matching files found`, which names neither what the file is
# nor how to get it back. Asking the guard first means the first thing printed is the sentence with
# the `git checkout --` in it. Prerequisites are made left to right, which this list already
# depends on elsewhere (build before the targets that run the binary).
check: embedded fmt-check vet test race build glidertool houses levels headless audio fidelity docs-check cross smoke
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
#
# Every sentence here is about a backend, and which backend you have is decided by GOOS, so both
# halves ask before they speak (docs/IMPROVEMENTS.md 4.13). The old text was three Linux
# assumptions worn as universals. On Windows it said cgo was off and so the x11 backend had not
# been compiled -- of a build whose win32 backend needs no cgo and *was* compiled -- and then
# blamed an unset `DISPLAY`, a variable Windows has no reason to set, for a window that did not
# open. On macOS it said the same two things about a platform that has no backend to compile at
# all. A caveat list is the one place in this file that must be right about what did not happen,
# because it is the only thing standing between a green run and a claim the run did not make.
#
# The other branches are reachable from here, which is why they can be trusted at all: they ask
# `go env GOOS`, and that answers the environment, so `GOOS=darwin make check-caveats` and
# `GOOS=windows make check-caveats` print exactly what a reader on those machines would see. CI
# does not do it -- windows-latest has no `make`, and the macOS half of the same job calls `go`
# directly for the same reason (the `native` job in .github/workflows/ci.yml) -- so those two
# command lines are the only test these branches have, and a change here should run all four.
check-caveats:
	@n=0; os=$$($(GO) env GOOS); \
	case $$os in \
	linux) \
		if [ "$$($(GO) env CGO_ENABLED)" != "1" ]; then \
			echo "  - cgo is off, so the x11 backend was NOT compiled; \`build\` produced the null backend"; n=1; \
		elif ! pkg-config --exists x11 2>/dev/null; then \
			echo "  - libx11 dev metadata is missing, so the x11 backend was NOT compiled"; n=1; \
		fi;; \
	windows) ;; \
	*) \
		echo "  - $$os has no backend of its own yet (stage 6), so \`build\` produced the null"; \
		echo "    backend and cgo makes no difference to that"; n=1;; \
	esac; \
	if ! $(CAN_RACE); then \
		echo "  - the race detector did NOT run, so the code with goroutines (internal/netplay,"; \
		echo "    internal/audio's Pipe and the race in cmd/glidergo) was not checked for races"; n=1; \
	fi; \
	if ! $(HAVE_HOUSES) || ! $(HAVE_ART) || ! $(HAVE_SOUND); then \
		echo "  - no extracted asset tree: the house round-trip and the pixel corpus were NOT"; \
		echo "    checked (the runs above used the copy inside the binaries and are unaffected)"; n=1; \
	fi; \
	case $$os in \
	linux) \
		if [ -z "$$DISPLAY" ]; then \
			echo "  - DISPLAY is unset, so the on-screen blit was NOT exercised"; n=1; \
		fi;; \
	windows) \
		echo "  - nothing opened a window: \`smoke\` is the X11 bench, and the on-screen run"; \
		echo "    here is \`make bench\` by hand"; n=1;; \
	esac; \
	if [ $$n -eq 0 ]; then \
		echo "         toolchain, cgo, tests, races, houses, levels, headless, audio, pixels, cross-build and the blit path"; \
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
#
# Interrupting it is safe, and that is the extractor's doing rather than this rule's: it stages
# into assets/.extracted.tmp-<pid> and renames the finished tree into place, so ^C leaves the
# committed tree exactly as it was (docs/IMPROVEMENTS.md 5.2). Two of these at once refuse rather
# than interleave, for the same reason.
assets:
	python3 tools/extract_all.py
	@$(MAKE) --no-print-directory assets-zip

## assets-zip: repack assets/extracted.zip from assets/extracted/ without re-extracting
#
# The second half of `make assets` on its own, for a tree that changed by some other means: a
# hand-authored house dropped in, or a file restored from git.
assets-zip:
	$(GO) run ./tools/packassets

## levels-zip: repack assets/levels.zip from assets/levels/ after `make levels`
#
# The one target in here whose output is meant to be committed in the same commit as the source
# change that caused it. `make levels` builds the text into $(LEVELS) and refuses to pass while
# the archive disagrees; this is what makes it agree again. Two files move in git status, the
# text and the archive, and that is the shape of a level change.
#
# It does not depend on `levels`, so that a directory built by hand or by an editor can be packed
# without a rebuild -- the same licence `assets-zip` has over `assets`.
levels-zip:
	$(GO) run ./tools/packassets -tree $(LEVELS) -out $(LEVELS_ZIP)

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
# It leaves both archives alone, deliberately, and that is worth knowing: they are build inputs,
# so removing one stops the build, and a binary built without the tree present still plays every
# house -- the twenty-two originals and the port's own. What a cleaned tree costs is the
# extractor's own checks -- `make houses`, `make assets-check` and the fidelity corpus, which read
# files rather than the archives.
clean-assets:
	rm -rf $(ASSETS)

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
