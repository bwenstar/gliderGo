# gliderGo — development environment

Everything in this document was **probed on the actual machine**, not assumed. Dates of
probing: 2026-09-10 (the airgapped host: `Linux 6.17.0-1010-aws`, Ubuntu 24.04.4 LTS noble,
x86_64) and 2026-09-16 (the public build path).

This is the file to read first if you are a new session picking this project up.

---

## 1. Two ways in

gliderGo was written on an airgapped host — no internet, one private package mirror — and it is
meant to be built by contributors who have the open internet. Both paths are supported and
neither is the "real" one. §4 says what that airgapped network could and could not supply,
which is the fact that shaped the whole port.

### If you have Go 1.23+ and an internet connection

Nothing to bootstrap. `make` finds a `go` on your PATH by itself.

```bash
git clone https://github.com/bwenstar/gliderGo && cd gliderGo
sudo apt-get install -y build-essential pkg-config libx11-dev   # or §2 for your distro
make check                          # fmt, vet, tests, build, cross-compile, smoke  (~15 s)
make run                            # play it
```

Nothing else is needed and nothing else is fetched. The game's data is committed:
`assets/extracted/` holds the 1994 art, sounds, music, 22 houses and movies, already decoded,
so there is no extraction step before playing and no copy of Glider PRO to find.
`tools/extract_all.py` produced that tree from the vendored `GliderPRO/` and `make assets`
re-runs it (~70 s, needs python3); `make assets-check` re-extracts to a temp tree and proves
the committed copy is byte-for-byte identical.

`make check` also passes on a tree with the assets **removed** and with **no** display —
verified. The asset-dependent steps skip with a message and the closing summary lists what it
could not verify, so a green run never overclaims. `make doctor` reports what your machine has
and is missing. `libx11-dev` is the one requirement that is not optional: `vet` and `test`
compile the X11 backend whenever cgo is on, so without its pkg-config metadata `make check`
fails with an error from pkg-config. `make headless` is the build that needs neither it nor a
display.

### If you have no Go, or no internet

```bash
./scripts/bootstrap-dev-env.sh      # picks a dependency source and says which; installs Go
. scripts/env.sh                    # PATH/GOROOT/GOPROXY=off/GOTOOLCHAIN=local
make check
```

The script resolves dependencies from one of three sources and prints the one it chose:

| `--source` | Toolchain from | C libraries from | For |
|---|---|---|---|
| `system` | a `go` already on PATH satisfying `go.mod` | your package manager (§2) | almost every contributor |
| `local` | whatever `scripts/local-source.sh` defines | ditto | a machine that can reach neither |
| `public` | `go.dev/dl`, sha256-verified | `archive.ubuntu.com` | a contributor with no Go |

`auto` (the default) tries them in that order: `system` first because it costs nothing, `local`
before `public` because a machine with a private mirror usually has one precisely because it
cannot see the open internet. `local` is skipped, silently, unless `scripts/local-source.sh`
exists — it does not exist in a clone, it is gitignored, and nothing about anyone's private
network is built into this repository, so nobody else's `make doctor` probes a host they have
never heard of. The toolchain source and the C-library source are resolved **separately**,
which is what the airgapped host needs: `auto` picks `system` for Go, because Go is already
installed there, while a `.deb` still has to come from that host's own mirror.

Useful flags: `--dry-run` prints every URL and file it would touch and changes nothing, which
is the only way to review the public path from an airgapped box; `--check` (also `make doctor`)
reports the environment; `--sysroot` unpacks the optional C libraries into
`.toolchain/sysroot` without root.

If your machine cannot reach `go.dev` there are three ways out, and none of them needs an edit
to the script:

| | |
|---|---|
| `GLIDERGO_GO_TARBALL=/path/to/go1.23.x.linux-amd64.tar.gz` | uses a tarball you carried across by hand; no network at all |
| `GLIDERGO_GO_DL_HOST=…`, `GLIDERGO_UBUNTU_MIRROR=…` | points the `public` source at a mirror that speaks the same protocols |
| `scripts/local-source.sh` | for a source that needs real logic rather than a different URL |

That last one is the `local` source: a gitignored file defining up to five shell functions —
`local_source_label`, `_reachable`, `_install_go`, `_ubuntu_url`, `_curl_args` — documented
where the script sources it. It is where a hostname, a mirror's layout and a path to a
credential belong, because none of those are a public repository's to carry. **No credential
and no private hostname is written into this repository**, and the script behaves exactly as
though the file did not exist when it does not.

---

## 2. Dependencies: the complete list

gliderGo has **no Go dependencies at all** — standard library only, asserted offline by
`internal/module` (which parses `go.mod` and every import in the tree, so it catches an import
hidden behind a build tag that `go build` on this host would never compile). That is why there
is no `go.sum` and no `vendor/`, and why `GOPROXY=off` is set everywhere.

Everything it does need is a host tool:

| What | Floor | Needed for | Without it |
|---|---|---|---|
| **Go** | 1.23 (`go.mod`) | everything | nothing builds |
| **gcc** + **pkg-config** | any | cgo, i.e. the x11 backend | as the next row: nothing that opens a window builds |
| **libX11 headers** (`x11.pc`) | 1.8 | the x11 backend — the only one that opens a window | `make build`, `vet` and `test` **fail**, with pkg-config's own error. There is no silent fallback: the null backend is `make headless`, or `-tags nullbackend`, asked for by name |
| **python3** | 3.6 (f-strings) | `make assets` / `assets-check` only | nothing — the assets those produce are committed. Needed to change the extractor or verify the tree |
| **git** | any | the version string only | the binary reports `version=dev` |
| **make** | any | convenience | use `go build ./cmd/glidergo` directly |
| `libXext` headers | — | *nothing* — MIT-SHM is deliberately unused | nothing |
| `libasound2` headers | — | a future native ALSA sink | nothing today; audio mixes to WAV |
| `libsdl2` headers | — | the future macOS/iOS/Android backend (stage 6) | nothing today |
| `xvfb` | — | running the on-screen bench in CI | use `make headless` instead |

Per distro, the required set:

```bash
# Debian / Ubuntu
sudo apt-get install -y build-essential pkg-config libx11-dev   # + python3 to re-extract
# Fedora / RHEL
sudo dnf install -y gcc pkgconf-pkg-config libX11-devel
# Arch
sudo pacman -S --needed base-devel pkgconf libx11
# macOS -- builds and tests, but has no backend yet, so it cannot draw
xcode-select --install
```

`./scripts/bootstrap-dev-env.sh --check` prints this table filled in for your machine.

**If you run the bench under Xvfb, set the depth explicitly.** `internal/platform/x11`
rejects anything but 24 or 32 bits, and Xvfb's historical default is 8 — which fails the build
step rather than skipping it:

```bash
xvfb-run -a -s '-screen 0 640x480x24' make bench
```

---

## 3. Toolchain: what is installed and how

This section describes **one host** — the airgapped machine the port was written on — and is
here as a worked example rather than as instructions. Anything Go 1.23 or newer works; the
supported ways to get one are `scripts/bootstrap-dev-env.sh` (§1) or your distribution's own
package, and `make` finds a `go` on `PATH` without either.

### Go

```
go version go1.23.12 linux/amd64      GOROOT=$HOME/.local/opt/go
```

Obtained without root, and without a Go download being reachable, by taking the toolchain out
of the official `golang` container image — that host had a mirror of Docker Hub but not of
`go.dev`. Worth knowing as a trick in its own right, since it needs no root and no package
manager (~250 MB tar):

```bash
podman pull golang:1.23-bookworm          # or "$YOUR_REGISTRY/library/golang:1.23-bookworm"
podman run --rm -v /tmp:/out:z golang:1.23-bookworm \
        sh -c 'tar -C /usr/local -cf /out/go-toolchain.tar go'
tar -C ~/.local/opt -xf /tmp/go-toolchain.tar
```

On a machine with the open internet, `./scripts/bootstrap-dev-env.sh --source public` does the
equivalent from `go.dev` and verifies the tarball's sha256 against the release index.

`GOTOOLCHAIN=local` is mandatory in `scripts/env.sh`: without it, a `go` directive in
`go.mod` newer than 1.23.12 makes the toolchain try to *download* a newer one, which
fails on an airgapped host with a confusing error.

`GOPROXY=off` is also set deliberately — it turns "silently tries the network for 30s"
into an immediate, legible error if anyone adds a dependency by accident.

### C toolchain (for cgo)

`gcc` 13, `make`, `pkg-config` are present in the base image. `cgo` works: verified by
building and running a real X11 program (below).

### Already present on the host, no install needed

| Library | Version | Notes |
|---|---|---|
| `libX11` + headers (`x11.pc`) | 1.8.7 | enough for window + `XPutImage` + keyboard |
| `libGL`, `libGLX`, `libEGL` + `GL/gl.h`, `GL/glx.h` | GL 1.2 / EGL 1.5 | GLX path available if we ever want texture-blit or scaling on the GPU |
| `libasound.so.2` (runtime only, **no headers**) | 1.2.x | ALSA can be reached by declaring the handful of `snd_pcm_*` prototypes ourselves, or `dlopen`, without `libasound2-dev` |
| `libpulse-simple.so.0`, `libpipewire-0.3.so.0` | — | alternative audio sinks |
| `aplay`, `pw-play` | — | useful for verifying extracted sounds by ear on a machine that has audio |

### Fetchable on demand (rootless), verified working

`dpkg-deb -x` into `.toolchain/sysroot` — proven end-to-end by downloading
`libsdl2-dev_2.30.0+dfsg-1build3_amd64.deb` (1.14 MB) and confirming
`usr/include/SDL2/SDL.h` landed. `./scripts/bootstrap-dev-env.sh --sysroot` automates it.

| Package | Why we might want it |
|---|---|
| `libxext-dev` 1.3.4 | MIT-SHM headers → zero-copy blits (optional; XPutImage is already fast enough) |
| `libasound2-dev` 1.2.11 | proper ALSA headers for the Linux audio backend |
| `libsdl2-dev` 2.30.0 (+ `libsdl2-image/mixer/ttf-dev`) | the portability escape hatch — see §5 |
| `gcc-mingw-w64-x86-64`, `binutils-mingw-w64-x86-64` 13.2 | cgo cross-compilation to Windows |

---

## 4. The constraint that produced this architecture: no obtainable Go game engine

gliderGo was written on an airgapped host whose only software source was one private package
mirror. The details of that network are its owner's business and are not written down here;
§1 says how a machine like it plugs its own source in without this repository knowing anything
about it. What matters here is only what such a mirror could and could not supply, because that
is what shaped the port.

Reachable, and used: a mirrored Ubuntu archive (so C dev libraries could be fetched rootless as
`.deb`s and unpacked into a sysroot) and a Docker Hub mirror (so the official `golang` image
could supply a toolchain — §3). Between them that is a Go compiler and `libX11`.

Not reachable, and this is the load-bearing half:

| What | Consequence |
|---|---|
| **No Go module proxy of any kind** — no `proxy.golang.org`, no Go-typed repository | `GOPROXY=off` is the repo default, and `internal/module` asserts as a *test* that nothing outside the standard library is imported |
| **Only an allowlisted subset of GitHub**, by organisation | Nothing could be fetched by module path even when the module existed |

**Therefore: no Ebitengine, no go-sdl2, no raylib-go, no oto, no `golang.org/x/mobile`.**
Ubuntu does package ~1,920 `golang-github-*-dev` source libraries and
`golang-golang-x-{image,sync,sys,text,…}-dev`, all reachable through the mirror — but none of
the game, windowing or audio libraries are among them (checked: ebiten, go-sdl2, go-gl/glfw,
oto, purego; all absent).

That single fact drives the whole architecture in §5, and it is worth saying that it has
turned out to be a feature rather than a wound: gliderGo has no dependencies, no lockfile, no
supply chain and no `go.sum`, and `git clone && make check` needs nothing from any network.

---

## 5. Consequence: the platform layer is ours

Because no Go game engine is obtainable, gliderGo carries its own thin platform
abstraction (a *port layer*, not an engine) with the standard library plus small,
hand-written cgo shims. This is a good fit for Glider PRO specifically: the original is
a fixed-resolution 2D **blitter** with no scaling, no rotation, no alpha — one 640×480
software framebuffer per frame is a faithful and complete rendering model.

```
internal/platform          # interface: Window, Framebuffer, Events, AudioSink, Clock
  ├── x11/                 # cgo + Xlib + XPutImage         [Linux, works today]
  ├── win32/               # pure Go syscall → user32/gdi32 StretchDIBits [Windows, no cgo, no deps]
  ├── sdl2/                # hand-written minimal cgo binding [macOS/iOS/Android escape hatch]
  └── null/                # headless: renders to PNG/WAV for tests and CI
```

Rationale for each backend:

- **x11** — zero downloads, already proven at 533 fps. Ships first.
- **win32** — Windows can be reached with *pure Go* (`syscall` to `user32.dll`/`gdi32.dll`,
  `StretchDIBits` for the blit, `waveOutWrite` for audio). No cgo, no mingw, and it
  cross-compiles from this box with `GOOS=windows go build`. This is the cheapest possible
  route to the user's "Windows later" goal given we cannot fetch SDL.
- **sdl2** — the one dependency we *can* obtain (as a C library via `.deb`, plus source for
  cross-builds). SDL2 covers macOS, iOS and Android, so it is the long-term answer for those
  targets. Writing our own ~300-line cgo binding for the ~30 SDL functions we need avoids
  needing the unobtainable `go-sdl2` module.
- **null** — required, not optional: with no audio device and a remote X server, automated
  fidelity tests must be able to run the game headless and diff framebuffers against
  reference PNGs.

Audio: the original uses Mac Sound Manager `'snd '` resources (see
`docs/analysis/audio.md`). Since this box has no `/dev/snd`, the plan is to decode `'snd '`
to raw PCM once, at asset-extraction time, and have the null sink write mixed output to WAV
so correctness is verifiable offline; ALSA/`waveOut`/SDL sinks are then thin.

---

## 6. Measured performance baseline

A cgo+Xlib probe (`/tmp/x11test`, reproduced by `make bench`, or by `make check` when
`DISPLAY` is set) on this host:

```
depth=24  bitmap_pad=32  bytes_per_line=2560  byte_order=LSBFirst (0)
blitted 120 frames of 640x480 in 225.1ms => 533.1 fps (655.0 MB/s)
```

The original game runs its world at a fixed tick (see `docs/analysis/architecture.md`),
so we have ~9× headroom on the display path even before MIT-SHM. Rendering is not a risk;
**fidelity** is.

---

## 7. Repository layout

```
gliderGo/
├── GliderPRO/              # the original's DATA, unmodified, vendored as the extractors' input
│   ├── Glider PRO.r        # 15 MB derez'ed resource fork: all art, sound, dialogs
│   ├── Houses/  *.binhex   # the original levels (BinHex 4.0 of resource forks)
│   ├── Sources/  *.c       # NOT in the repo: gitignored, drop upstream's in -- see below
│   └── Headers/  *.h       # NOT in the repo: likewise
├── docs/
│   ├── DEV_ENVIRONMENT.md  # this file
│   ├── ORIGINAL_GAME.md    # consolidated source of truth for the original's behaviour
│   ├── PLAN.md             # staged implementation plan
│   └── analysis/*.md       # per-subsystem byte-level specs (the detailed authority)
├── cmd/
│   ├── glidergo/           # the game
│   └── glidertool/         # house CLI: dump/build/check/info/rooms + `types`
├── internal/
│   ├── house/              # house model, binary codec (byte-exact), text codec
│   ├── module/             # the stdlib-only invariant, checked offline (tests only)
│   └── platform/           # 640x480 framebuffer, x11 (cgo) and null backends
├── tools/                  # asset-extraction and probe scripts (python3)
│   └── extract_all.py      #   the driver: `make assets` -> assets/extracted/
├── assets/extracted/       # committed game data: 1,877 files, 15.5 MB, rebuilt in ~70 s
├── scripts/                # bootstrap-dev-env.sh, env.sh (generated, gitignored)
├── .github/workflows/      # public CI -- see the caveat at the top of ci.yml
└── .toolchain/             # gitignored: sysroot + deb cache
```

### Working with the original source — read this before grepping

**The 1994 C is not in this repository**; only the data the extractors read is. `docs/` cites
that C about 9,500 times anyway, because the citations are the receipts for the transcription,
so they are pinned to an upstream commit. To make them resolve:

```bash
git clone https://github.com/softdorothy/glider_pro /tmp/glider_pro
git -C /tmp/glider_pro checkout 94fed96e0b4c810a6ac861e5d4b14d625a5a1c31
cp -r /tmp/glider_pro/Sources /tmp/glider_pro/Headers GliderPRO/
```

`GliderPRO/Sources/` and `GliderPRO/Headers/` are gitignored, so this leaves the tree clean.
`94fed96` is the commit every line number in `docs/` was taken against. Nothing in the build
needs it: `make check` and `make assets-check` both pass without it.

Once you have it: `Sources/*.c` and `Headers/*.h` are **classic Mac text files with CR-only
line endings**. `wc -l` reports `0`, and editors/tools see one enormous line. Always convert
before reading:

```bash
tr '\r' '\n' < "GliderPRO/Sources/Player.c" > /tmp/Player.c
```

`GliderPRO/Glider PRO.r` is the exception — it uses LF (199,843 lines) — and it *is* vendored.

`GliderPRO/upstream.git` is gitignored too, so the original history can stay available locally
without becoming a submodule.

---

## 8. Known environment gaps

| Gap | Workaround | Blocks |
|---|---|---|
| No audio device | WAV-dump sink; verify by ear elsewhere | hearing sound locally, not development |
| No `sudo` | rootless sysroot via `dpkg-deb -x` | nothing |
| No Go module proxy | stdlib-only + hand-written cgo shims | third-party engines (accepted; see §5) |
| No MIT-SHM headers | plain `XPutImage` (fast enough) | nothing |
| No macOS/iOS build host | SDL2 backend + CI later | Mac/iOS targets, by definition |
| X server is remote (DCV) | fine for dev; frame-diff tests use the null backend | precise vsync measurement |
| **The public path cannot be tested from here** | `--dry-run` reviews it offline; every *command* it runs was verified | confidence in `go.dev` parsing, `archive.ubuntu.com`, and `.github/workflows/ci.yml` until someone runs them on a connected host |
| No `xvfb` installed | verified against `Xephyr :77 -screen 640x480x24` instead, which the x11 backend accepted at 686 fps | certainty about `xvfb-run` specifically |
