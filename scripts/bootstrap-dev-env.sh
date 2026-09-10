#!/usr/bin/env bash
# Bootstrap the gliderGo development environment on an airgapped host.
#
# Everything is fetched from the internal package mirror and installed
# WITHOUT root: the Go toolchain is extracted from a container image, and any
# C dev libraries are unpacked from .deb files into a local sysroot.
#
#   ./scripts/bootstrap-dev-env.sh              # Go toolchain only (enough to build+run on Linux/X11)
#   ./scripts/bootstrap-dev-env.sh --sysroot    # also fetch optional C dev libs (SHM, SDL2, ALSA)
#   ./scripts/bootstrap-dev-env.sh --check      # verify an existing environment, install nothing
#
# Credentials: export MIRROR_USER / MIRROR_PASS, or put a
# machine entry for the mirror host in ~/.netrc. They are never written
# into the repository.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLCHAIN_DIR="${GLIDERGO_TOOLCHAIN_DIR:-$HOME/.local/opt}"
GO_DIR="$TOOLCHAIN_DIR/go"
SYSROOT="$REPO_ROOT/.toolchain/sysroot"
CACHE="$REPO_ROOT/.toolchain/cache"

MIRROR_HOST="mirror.internal.example.com"
MIRROR="https://$MIRROR_HOST/repo"
UBUNTU_REPO="$MIRROR/archive.ubuntu.com-ubuntu"
UBUNTU_SUITE="noble"
GO_IMAGE="$MIRROR_HOST/registry-1.docker.io/library/golang:1.23-bookworm"

# Optional C dev libraries, only needed for the backends noted in brackets.
#   libxext-dev     -> MIT-SHM zero-copy blits          [x11 backend, optional speedup]
#   libasound2-dev  -> ALSA headers                     [linux audio backend]
#   libsdl2-dev     -> SDL2 headers + static/shared lib [portable backend: win/mac/ios/android]
SYSROOT_PACKAGES=(libxext-dev libx11-dev libasound2-dev libsdl2-dev libsdl2-2.0-0)

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m warn:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

curl_auth() {
  local args=(--fail --silent --show-error --location --max-time 300)
  if [[ -n "${MIRROR_USER:-}" && -n "${MIRROR_PASS:-}" ]]; then
    args+=(--user "$MIRROR_USER:$MIRROR_PASS")
  else
    args+=(--netrc-optional)
  fi
  curl "${args[@]}" "$@"
}

install_go() {
  if [[ -x "$GO_DIR/bin/go" ]]; then
    log "Go already present: $("$GO_DIR/bin/go" version)"
    return
  fi
  mkdir -p "$TOOLCHAIN_DIR" "$CACHE"

  local runtime=""
  for c in podman docker; do command -v "$c" >/dev/null 2>&1 && runtime="$c" && break; done

  if [[ -n "$runtime" ]]; then
    log "Extracting the Go toolchain from $GO_IMAGE via $runtime (no root required)"
    "$runtime" pull "$GO_IMAGE"
    "$runtime" run --rm -v "$CACHE:/out:z" "$GO_IMAGE" \
      sh -c 'tar -C /usr/local -cf /out/go-toolchain.tar go'
    tar -C "$TOOLCHAIN_DIR" -xf "$CACHE/go-toolchain.tar"
    rm -f "$CACHE/go-toolchain.tar"
  else
    log "No container runtime; falling back to the Ubuntu golang .deb"
    install_debs_into "$CACHE/godeb" golang-1.23-go golang-1.23-src || \
      install_debs_into "$CACHE/godeb" golang-1.22-go golang-1.22-src
    local found
    found="$(find "$CACHE/godeb/usr/lib" -maxdepth 1 -name 'go-1.*' | head -1)"
    [[ -n "$found" ]] || die "could not locate an extracted Go toolchain"
    cp -a "$found" "$GO_DIR"
  fi
  [[ -x "$GO_DIR/bin/go" ]] || die "Go install failed"
  log "Installed $("$GO_DIR/bin/go" version)"
}

# Resolve a binary package name to its pool path using the suite's Packages index.
apt_filename() {
  local pkg="$1"
  python3 - "$pkg" "$CACHE" <<'PY'
import re, sys, os
pkg, cache = sys.argv[1], sys.argv[2]
best = None
for comp in ("main", "universe", "restricted", "multiverse"):
    path = os.path.join(cache, f"Packages-{comp}")
    if not os.path.exists(path):
        continue
    txt = open(path, encoding="utf-8", errors="replace").read()
    m = re.search(r"^Package: %s\n(.*?)(?=\n\n)" % re.escape(pkg), txt, re.S | re.M)
    if m:
        fn = re.search(r"^Filename: (.*)$", m.group(1), re.M)
        if fn:
            best = fn.group(1)
            break
print(best or "")
PY
}

fetch_apt_indexes() {
  mkdir -p "$CACHE"
  for comp in main universe; do
    if [[ ! -s "$CACHE/Packages-$comp" ]]; then
      log "Fetching Ubuntu $UBUNTU_SUITE/$comp package index"
      curl_auth -o "$CACHE/Packages-$comp.xz" \
        "$UBUNTU_REPO/dists/$UBUNTU_SUITE/$comp/binary-amd64/Packages.xz"
      xz -dc "$CACHE/Packages-$comp.xz" > "$CACHE/Packages-$comp"
      rm -f "$CACHE/Packages-$comp.xz"
    fi
  done
}

install_debs_into() {
  local dest="$1"; shift
  fetch_apt_indexes
  mkdir -p "$dest" "$CACHE/debs"
  local pkg fn deb
  for pkg in "$@"; do
    fn="$(apt_filename "$pkg")"
    if [[ -z "$fn" ]]; then warn "package not found in index: $pkg"; return 1; fi
    deb="$CACHE/debs/$(basename "$fn")"
    if [[ ! -s "$deb" ]]; then
      log "Fetching $pkg"
      curl_auth -o "$deb" "$UBUNTU_REPO/$fn"
    fi
    dpkg-deb -x "$deb" "$dest"
  done
}

install_sysroot() {
  log "Building rootless sysroot at $SYSROOT"
  install_debs_into "$SYSROOT" "${SYSROOT_PACKAGES[@]}" || warn "some sysroot packages were skipped"
  # .pc files carry absolute /usr prefixes; rewrite them to point at the sysroot.
  local pcdir="$SYSROOT/usr/lib/x86_64-linux-gnu/pkgconfig"
  if [[ -d "$pcdir" ]]; then
    python3 - "$SYSROOT" "$pcdir" <<'PY'
import os, re, sys
sysroot, pcdir = sys.argv[1], sys.argv[2]
for name in os.listdir(pcdir):
    if not name.endswith(".pc"):
        continue
    p = os.path.join(pcdir, name)
    s = open(p).read()
    s = re.sub(r"^prefix=/usr\s*$", f"prefix={sysroot}/usr", s, flags=re.M)
    open(p, "w").write(s)
PY
  fi
}

write_env_file() {
  cat > "$REPO_ROOT/scripts/env.sh" <<EOF
# Generated by scripts/bootstrap-dev-env.sh -- source this before building.
#   . scripts/env.sh
export GOROOT="$GO_DIR"
export PATH="$GO_DIR/bin:\$HOME/.local/gopath/bin:\$PATH"
export GOPATH="\$HOME/.local/gopath"
# Airgapped: there is no Go module proxy on this network. gliderGo therefore
# depends on the standard library only; anything else must be vendored by hand.
export GOPROXY=off
export GOFLAGS=-mod=mod
export GOTOOLCHAIN=local
export CGO_ENABLED=1
# Optional sysroot (created by --sysroot) for SHM / SDL2 / ALSA headers.
if [ -d "$SYSROOT" ]; then
  export GLIDERGO_SYSROOT="$SYSROOT"
  export PKG_CONFIG_PATH="$SYSROOT/usr/lib/x86_64-linux-gnu/pkgconfig:$SYSROOT/usr/share/pkgconfig:\${PKG_CONFIG_PATH:-}"
  export CGO_CFLAGS="-I$SYSROOT/usr/include \${CGO_CFLAGS:-}"
  export CGO_LDFLAGS="-L$SYSROOT/usr/lib/x86_64-linux-gnu -Wl,-rpath,$SYSROOT/usr/lib/x86_64-linux-gnu \${CGO_LDFLAGS:-}"
fi
EOF
  log "Wrote scripts/env.sh"
}

check_env() {
  local ok=0
  printf '\n%-34s %s\n' "CHECK" "RESULT"
  probe() { printf '%-34s %s\n' "$1" "$2"; }

  if [[ -x "$GO_DIR/bin/go" ]]; then probe "go toolchain" "$("$GO_DIR/bin/go" version)"; else probe "go toolchain" "MISSING"; ok=1; fi
  command -v gcc >/dev/null && probe "cgo C compiler" "$(gcc -dumpversion)" || { probe "cgo C compiler" "MISSING"; ok=1; }
  pkg-config --exists x11 && probe "libX11 dev (x11.pc)" "$(pkg-config --modversion x11)" || { probe "libX11 dev (x11.pc)" "MISSING"; ok=1; }
  pkg-config --exists xext && probe "libXext dev (MIT-SHM)" "$(pkg-config --modversion xext)" || probe "libXext dev (MIT-SHM)" "absent (optional)"
  pkg-config --exists sdl2 && probe "SDL2 dev" "$(pkg-config --modversion sdl2)" || probe "SDL2 dev" "absent (optional)"
  [[ -n "${DISPLAY:-}" ]] && probe "X display" "$DISPLAY" || probe "X display" "unset (headless: use Xvfb)"
  [[ -e /dev/snd ]] && probe "audio device" "present" || probe "audio device" "absent (null audio sink)"
  probe "original source" "$([[ -d "$REPO_ROOT/GliderPRO/Sources" ]] && echo present || echo MISSING)"
  return $ok
}

MODE=go-only
for arg in "$@"; do
  case "$arg" in
    --sysroot) MODE=sysroot ;;
    --check)   MODE=check ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) die "unknown flag: $arg" ;;
  esac
done

case "$MODE" in
  check) check_env; exit $? ;;
  sysroot) install_go; install_sysroot; write_env_file; check_env || true ;;
  go-only) install_go; write_env_file; check_env || true ;;
esac

cat <<'EOF'

Next:
  . scripts/env.sh          # put the toolchain on PATH
  make check                # build + smoke-test the platform layer
EOF
