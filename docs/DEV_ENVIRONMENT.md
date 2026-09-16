# gliderGo — development environment

Everything in this document was **probed on the actual machine**, not assumed. Dates of
probing: 2026-09-10 (the airgapped host: `Linux 6.17.0-1010-aws`, Ubuntu 24.04.4 LTS noble,
x86_64) and 2026-09-16 (the public build path).

This is the file to read first if you are a new session picking this project up.

---

## 1. Two ways in

gliderGo is developed on an airgapped host whose only software source is an internal
package mirror, and it is meant to end up on GitHub where contributors have the open
internet. Both paths are supported and neither is the "real" one. §4 documents the airgapped
network in detail because that is the one you cannot look up.

### If you have Go 1.23+ and an internet connection

Nothing to bootstrap. `make` finds a `go` on your PATH by itself.

```bash
git clone <the repo> && cd gliderGo
sudo apt-get install -y build-essential pkg-config libx11-dev   # or see §2 for your distro
make check                          # fmt, vet, tests, build, cross-compile, smoke  (~15 s)
make assets                         # extract the 1994 data (~70 s, gitignored output)
make check                          # again: now the houses, audio and pixel corpus run too
make run                            # play it
```

`make check` passes on a clone with **no** extracted assets and **no** display — verified. The
asset-dependent steps skip with a message and the closing summary lists what it could not
verify, so a green run never overclaims. `make doctor` reports what your machine has and is
missing.

### If you have no Go, or no internet

```bash
./scripts/bootstrap-dev-env.sh      # picks a dependency source and says which; installs Go
. scripts/env.sh                    # PATH/GOROOT/GOPROXY=off/GOTOOLCHAIN=local
make check
make assets
```

The script resolves dependencies from one of three sources and prints the one it chose:

| `--source` | Toolchain from | C libraries from | For |
|---|---|---|---|
| `system` | a `go` already on PATH satisfying `go.mod` | your package manager (§2) | almost every contributor |
| `internal` | the mirror's `golang:1.23-bookworm` image | the package mirror's Ubuntu tree | the airgapped host |
| `public` | `go.dev/dl`, sha256-verified | `archive.ubuntu.com` | a contributor with no Go |

`auto` (the default) tries them in that order: `system` first because it costs nothing,
`internal` before `public` because a host that can see the mirror usually cannot see anything
else. The toolchain source and the C-library source are resolved **separately** — on the
airgapped host `auto` picks `system` for Go (it is already installed) while the `.deb` archive
still has to be the package mirror.

Useful flags: `--dry-run` prints every URL and file it would touch and changes nothing, which
is the only way to review the public path from an airgapped box; `--check` (also `make doctor`)
reports the environment; `--sysroot` unpacks the optional C libraries into
`.toolchain/sysroot` without root. `GLIDERGO_GO_TARBALL=/path/to/go1.23.x.linux-amd64.tar.gz`
skips the network entirely, which is how you carry a toolchain across an airgap by hand.

Credentials for the internal source come from `MIRROR_USER` / `MIRROR_PASS` or
`~/.netrc`. **They are intentionally never written into this repository.**

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
| **gcc** + **pkg-config** | any | cgo, i.e. the x11 backend | see below |
| **libX11 headers** (`x11.pc`) | 1.8 | the x11 backend — the only one that opens a window | `make build` **silently produces the null backend**; `make check` now says so |
| **python3** | 3.6 (f-strings) | `make assets` | no art, sound, houses or movies; every asset-dependent step skips |
| **git** | any | the version string only | the binary reports `version=dev` |
| **make** | any | convenience | use `go build ./cmd/glidergo` directly |
| `libXext` headers | — | *nothing* — MIT-SHM is deliberately unused | nothing |
| `libasound2` headers | — | a future native ALSA sink | nothing today; audio mixes to WAV |
| `libsdl2` headers | — | the future macOS/iOS/Android backend (stage 6) | nothing today |
| `xvfb` | — | running the on-screen bench in CI | use `make headless` instead |

Per distro, the required set:

```bash
# Debian / Ubuntu
sudo apt-get install -y build-essential pkg-config libx11-dev python3
# Fedora / RHEL
sudo dnf install -y gcc pkgconf-pkg-config libX11-devel python3
# Arch
sudo pacman -S --needed base-devel pkgconf libx11 python
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

### Go

```
go version go1.23.12 linux/amd64      GOROOT=$HOME/.local/opt/go
```

Obtained with (no root, ~250 MB tar):

```bash
podman pull mirror.internal.example.com/registry-1.docker.io/library/golang:1.23-bookworm
podman run --rm -v /tmp:/out:z <image> sh -c 'tar -C /usr/local -cf /out/go-toolchain.tar go'
tar -C ~/.local/opt -xf /tmp/go-toolchain.tar
```

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

## 4. The package mirror: exactly what this network can and cannot reach

Base URL `https://mirror.internal.example.com/repo` — 49 repositories.
Credentials: supply via `MIRROR_USER` / `MIRROR_PASS` env vars or `~/.netrc`.
**They are intentionally not written down in this repo.**

### Works ✅

| What | Endpoint | Proven by |
|---|---|---|
| Ubuntu noble archive (main/restricted/universe/multiverse, all arches) | `/archive.ubuntu.com-ubuntu/…` | fetched `dists/noble/Release` + 15 MB `universe/binary-amd64/Packages.xz`, downloaded and extracted a real `.deb` |
| Docker Hub mirror | `/registry-1.docker.io/library/golang:1.23-bookworm` (podman pull) | pulled the golang image |
| PyPI | `/api/pypi/pypi.org/simple/…` | HTTP 200 |
| npm | `/api/npm/registry.npmjs.org/…` | HTTP 200 |
| crates.io | `/api/cargo/index.crates.io/…` | HTTP 200 |
| RHEL 9 yum, Fedora, Oracle yum, Maven, NuGet, Conan, HuggingFace | see repo list | not needed here |

### Does **not** work ❌

| What | Evidence |
|---|---|
| **Go module proxy** — there is no `proxy.golang.org` / package-mirror Go repo at all | `/api/go/...` → 404; no repo of `packageType: Go` in the 49-repo listing |
| **Arbitrary GitHub** — the `github.com` generic remote serves only an allowlist | `madler/zlib` tag tarball → **200**, but `hajimehoshi/ebiten` → **404**, `golang/go` → **404**. Cached orgs are: HandBrake, SonarSource, anchore, anomalyco, aquasecurity, aspect-build, astral-sh, bazel-contrib, bazelbuild, electron, facebook, gabime, git-for-windows, google, jgm, k3s-io, leethomason, madler, neovim, oras-project, protocolbuffers, sigstore, stedolan |
| GitLab.com generic remote | `gitlab-org/gitlab` archive → 404 |

**Therefore: no Ebitengine, no go-sdl2, no raylib-go, no ebitengine/oto, no
golang.org/x/mobile.** Ubuntu does package ~1920 `golang-github-*-dev` source
libraries and `golang-golang-x-{image,sync,sys,text,…}-dev`, which are reachable — but
none of the game/windowing/audio libraries are among them (checked: ebiten, go-sdl2,
go-gl/glfw, oto, purego → all absent).

This single fact drives the whole architecture in §5.

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
├── GliderPRO/              # the original 1994 source, unmodified, vendored as reference
│   ├── Sources/  *.c       # CR-only line endings! see note below
│   ├── Headers/  *.h
│   ├── Glider PRO.r        # 15 MB derez'ed resource fork: all art, sound, dialogs
│   └── Houses/  *.binhex   # the original levels (BinHex 4.0 of resource forks)
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
├── assets/extracted/       # gitignored: 1,899 generated files, 46 MB, rebuilt in ~70 s
├── scripts/                # bootstrap-dev-env.sh, env.sh (generated, gitignored)
├── .github/workflows/      # public CI -- see the caveat at the top of ci.yml
└── .toolchain/             # gitignored: sysroot + deb cache
```

### Working with the original source — read this before grepping

`GliderPRO/Sources/*.c` and `GliderPRO/Headers/*.h` are **classic Mac text files with
CR-only line endings**. `wc -l` reports `0`, and editors/tools see one enormous line.
Always convert before reading:

```bash
tr '\r' '\n' < "GliderPRO/Sources/Player.c" > /tmp/Player.c
```

`GliderPRO/Glider PRO.r` is the exception — it uses LF (199,843 lines).

The vendored `GliderPRO/.git` directory is gitignored so the original history stays
available locally without becoming a submodule.

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
