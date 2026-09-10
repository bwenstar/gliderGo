# gliderGo — development environment

Everything in this document was **probed on the actual machine**, not assumed. Dates of
probing: 2026-09-10, host `Linux 6.17.0-1010-aws`, Ubuntu 24.04.4 LTS (noble), x86_64.

This is the file to read first if you are a new session picking this project up.

---

## 1. TL;DR for a fresh session

```bash
cd gliderGo
./scripts/bootstrap-dev-env.sh      # rootless; installs Go into ~/.local/opt/go
. scripts/env.sh                    # PATH/GOROOT/GOPROXY=off/GOTOOLCHAIN=local
make check                          # builds and smoke-tests the platform layer
```

Hard constraints you must design around (each proven below):

| Constraint | Consequence |
|---|---|
| No internet; **no Go module proxy exists on this network** | gliderGo is **standard-library-only**. Any third-party Go code must be vendored by hand from a source we can actually reach. Ebitengine/SDL bindings/raylib-go are **not obtainable**. |
| No root (`sudo` unavailable to this session) | C libraries are unpacked from `.deb` files into `.toolchain/sysroot`, never installed system-wide. |
| Go itself is not in the base image | Extracted rootlessly from the `golang:1.23-bookworm` container image (works, verified) or from Ubuntu `golang-1.2x-go` debs (fallback). |
| No audio hardware (`/dev/snd` absent) | Audio cannot be heard on this box. Develop against a WAV-dumping sink; keep the real ALSA/PulseAudio path behind an interface. |
| Display is NICE DCV X11 (`DISPLAY=:1`), no MIT-SHM headers | `XPutImage` over the wire is the default blit path. Measured **533 fps** for full-screen 640×480×32bpp — 8.9× the 60 fps budget, so this is a non-issue. |

---

## 2. Toolchain: what is installed and how

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
| `libsdl2-dev` 2.30.0 (+ `libsdl2-image/mixer/ttf-dev`) | the portability escape hatch — see §4 |
| `gcc-mingw-w64-x86-64`, `binutils-mingw-w64-x86-64` 13.2 | cgo cross-compilation to Windows |

---

## 3. The package mirror: exactly what this network can and cannot reach

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

This single fact drives the whole architecture in §4.

---

## 4. Consequence: the platform layer is ours

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

## 5. Measured performance baseline

A cgo+Xlib probe (`/tmp/x11test`, reproduced by `make check`) on this host:

```
depth=24  bitmap_pad=32  bytes_per_line=2560  byte_order=LSBFirst (0)
blitted 120 frames of 640x480 in 225.1ms => 533.1 fps (655.0 MB/s)
```

The original game runs its world at a fixed tick (see `docs/analysis/architecture.md`),
so we have ~9× headroom on the display path even before MIT-SHM. Rendering is not a risk;
**fidelity** is.

---

## 6. Repository layout

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
├── tools/                  # asset-extraction and probe scripts (python3)
├── scripts/                # bootstrap-dev-env.sh, env.sh (generated)
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

## 7. Known environment gaps

| Gap | Workaround | Blocks |
|---|---|---|
| No audio device | WAV-dump sink; verify by ear elsewhere | hearing sound locally, not development |
| No `sudo` | rootless sysroot via `dpkg-deb -x` | nothing |
| No Go module proxy | stdlib-only + hand-written cgo shims | third-party engines (accepted; see §4) |
| No MIT-SHM headers | plain `XPutImage` (fast enough) | nothing |
| No macOS/iOS build host | SDL2 backend + CI later | Mac/iOS targets, by definition |
| X server is remote (DCV) | fine for dev; frame-diff tests use the null backend | precise vsync measurement |
