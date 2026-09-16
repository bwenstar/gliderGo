#!/usr/bin/env bash
# Bootstrap the gliderGo development environment.
#
# Two networks, one script. gliderGo was written on an airgapped host whose only
# software source is an internal package mirror, and it is meant to end up on
# GitHub where contributors have the open internet and, usually, a Go toolchain
# already. Those are different worlds and neither is the "real" one, so the script
# does not privilege either: it works out where dependencies can actually come
# from and says which it picked.
#
#   ./scripts/bootstrap-dev-env.sh                    # detect a source, install Go, write scripts/env.sh
#   ./scripts/bootstrap-dev-env.sh --sysroot          # also fetch the optional C dev libraries
#   ./scripts/bootstrap-dev-env.sh --check            # verify an existing environment, install nothing
#   ./scripts/bootstrap-dev-env.sh --source public    # force a source: system | internal | public
#   ./scripts/bootstrap-dev-env.sh --dry-run          # print the plan and every URL, touch nothing
#
# THE THREE SOURCES
#
#   system    A Go already on your PATH that satisfies go.mod's `go` directive.
#             Installs nothing and downloads nothing. This is the right answer for
#             almost every contributor and is why `auto` tries it first.
#   internal  The internal package mirror: the golang container image for the
#             toolchain, the mirrored Ubuntu archive for C libraries. The airgapped
#             host's only option, and unchanged from before this script grew a
#             second path.
#   public    go.dev for the toolchain, a public Ubuntu mirror for C libraries.
#             What a GitHub contributor without a Go install gets.
#
# `--source auto` (the default) tries them in that order. System first because it
# is free and correct; internal before public because a host that can see
# The mirror is usually a host that cannot see anything else.
#
# CREDENTIALS
#
# The internal source may need them: export MIRROR_USER / MIRROR_PASS,
# or put a machine entry for the mirror host in ~/.netrc. They are never
# written into the repository and never echoed. The public source needs none.
#
# The internal source also needs to be told where it is: there is no built-in
# hostname, so `internal` is skipped entirely unless GLIDERGO_MIRROR_HOST is
# exported. Nothing about anybody's private network is committed here.
#
# WHAT THIS SCRIPT IS NOT
#
# It is not required. `make check` finds a Go on PATH by itself (see the Makefile's
# GO variable), so a contributor who already has Go 1.23 can clone and build with
# no bootstrap at all. This script exists for the two cases where that is not
# true: no Go, or no internet.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLCHAIN_DIR="${GLIDERGO_TOOLCHAIN_DIR:-$HOME/.local/opt}"
GO_DIR="$TOOLCHAIN_DIR/go"
SYSROOT="$REPO_ROOT/.toolchain/sysroot"
CACHE="$REPO_ROOT/.toolchain/cache"

# --- an internal package mirror, if there is one -------------------------
# Opt-in, and deliberately so. This is empty unless GLIDERGO_MIRROR_HOST is
# exported, because a hostname on a private network is worth nothing to anybody
# else: the earlier default meant every stranger's `make doctor` resolved and
# probed a company host they have no business knowing about, waited three seconds
# for it, and then printed a row that reads like a missing prerequisite. The
# airgapped host this port was written on exports the variable once and behaves
# exactly as before.
MIRROR_HOST="${GLIDERGO_MIRROR_HOST:-}"
MIRROR="https://$MIRROR_HOST/repo"
INTERNAL_UBUNTU="$MIRROR/archive.ubuntu.com-ubuntu"
GO_IMAGE="$MIRROR_HOST/registry-1.docker.io/library/golang:1.23-bookworm"

# have_internal is the guard every probe of that mirror goes through, so that an
# unset host is "not configured" and never a failed connection to `https:///`.
have_internal() { [[ -n "$MIRROR_HOST" ]]; }
internal_reachable() { have_internal && reachable "$MIRROR/api/repositories"; }

# --- the open internet -------------------------------------------------------
# Both are overridable so that a third kind of network -- a company mirror that is
# neither of the above -- needs no edit to this file.
GO_DL_HOST="${GLIDERGO_GO_DL_HOST:-https://go.dev}"
PUBLIC_UBUNTU="${GLIDERGO_UBUNTU_MIRROR:-http://archive.ubuntu.com/ubuntu}"

UBUNTU_SUITE="${GLIDERGO_UBUNTU_SUITE:-noble}"

# The architecture words Debian and gcc use for this machine. Hardcoding
# x86_64-linux-gnu was fine on one build host and wrong the first time anyone
# builds on an arm64 laptop, which is a normal thing to own now.
#
# dpkg is not the only way to ask, and it was the only way this asked: on Fedora,
# Arch or macOS there is no dpkg, so the fallback silently answered `amd64` and the
# public path would have downloaded an x86-64 Go onto an arm64 machine. `uname -m`
# exists everywhere, and its answers only need translating into Debian's spelling.
deb_arch() {
  local a
  a="$(dpkg --print-architecture 2>/dev/null)" && [[ -n "$a" ]] && { printf '%s\n' "$a"; return; }
  case "$(uname -m 2>/dev/null)" in
    x86_64|amd64)   printf 'amd64\n' ;;
    aarch64|arm64)  printf 'arm64\n' ;;
    armv7l|armv7*)  printf 'armhf\n' ;;
    i386|i686)      printf 'i386\n' ;;
    riscv64)        printf 'riscv64\n' ;;
    ppc64le)        printf 'ppc64el\n' ;;
    s390x)          printf 's390x\n' ;;
    *)              printf 'amd64\n' ;;
  esac
}
DEB_ARCH="$(deb_arch)"
MULTIARCH="$(gcc -dumpmachine 2>/dev/null || echo x86_64-linux-gnu)"

# Optional C dev libraries, only needed for the backends noted in brackets.
#   libxext-dev     -> MIT-SHM zero-copy blits          [x11 backend, optional speedup]
#   libasound2-dev  -> ALSA headers                     [linux audio backend]
#   libsdl2-dev     -> SDL2 headers + static/shared lib [portable backend: win/mac/ios/android]
SYSROOT_PACKAGES=(libxext-dev libx11-dev libasound2-dev libsdl2-dev libsdl2-2.0-0)

SOURCE=auto      # system | internal | public | auto
MODE=go-only     # go-only | sysroot | check
DRY_RUN=0

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m warn:\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }
plan() { printf '\033[1;36m  would:\033[0m %s\n' "$*"; }

# ---------------------------------------------------------------------------
# Versions
# ---------------------------------------------------------------------------

# go_floor reads the requirement out of go.mod rather than repeating it here.
# There was a "1.23" in this script and a "go 1.23" in go.mod, and the failure
# mode of two copies is that bumping one installs a toolchain that cannot build
# the module -- with an error from the go command, not from this script.
go_floor() { awk '/^go[ \t]/ { print $2; exit }' "$REPO_ROOT/go.mod"; }

# version_ge: is $1 at least $2, comparing dotted versions properly? `sort -V`
# knows that 1.9 < 1.23, which every string comparison gets backwards.
version_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" = "$2" ]; }

# go_version prints the bare version of a go binary: "1.23.12", not "go1.23.12".
go_version() { "$1" version 2>/dev/null | awk '{ print $3 }' | sed 's/^go//'; }

# ---------------------------------------------------------------------------
# Reachability, and choosing a source
# ---------------------------------------------------------------------------

curl_auth() {
  local args=(--fail --silent --show-error --location --max-time 300)
  if [[ -n "${MIRROR_USER:-}" && -n "${MIRROR_PASS:-}" ]]; then
    args+=(--user "$MIRROR_USER:$MIRROR_PASS")
  else
    args+=(--netrc-optional)
  fi
  curl "${args[@]}" "$@"
}

# reachable answers in a few seconds or not at all. The airgapped host does not
# refuse connections to the outside world, it black-holes them, so a probe with
# no timeout hangs for minutes and the script looks broken rather than offline.
reachable() {
  curl --silent --show-error --output /dev/null \
    --connect-timeout 3 --max-time 8 --location "$1" 2>/dev/null
}

# found_go prints the path of a usable Go, preferring one this script installed
# earlier so that re-running it is idempotent, then whatever is on PATH.
found_go() {
  local candidate v floor
  floor="$(go_floor)"
  for candidate in "$GO_DIR/bin/go" "$(command -v go 2>/dev/null || true)"; do
    [[ -n "$candidate" && -x "$candidate" ]] || continue
    v="$(go_version "$candidate")"
    [[ -n "$v" ]] || continue
    if version_ge "$v" "$floor"; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done
  return 1
}

# choose_source decides where dependencies come from, and says why. The "why" is
# printed rather than kept, because the commonest confusion this script can cause
# is installing from a network the user did not expect.
choose_source() {
  if [[ "$SOURCE" != auto ]]; then
    # The one forced source that can be impossible: there is no built-in internal
    # mirror to fall back on, on purpose (see GLIDERGO_MIRROR_HOST above), so
    # say which variable is missing rather than fetching from `https:///`.
    if [[ "$SOURCE" == internal ]] && ! have_internal; then
      die "--source internal needs a mirror to talk to, and none is configured.
       Export GLIDERGO_MIRROR_HOST=<your mirror host> and re-run, or
       use --source system (a Go already on PATH) or --source public (go.dev)."
    fi
    log "Source: $SOURCE (forced with --source)"
    return
  fi

  local go floor
  floor="$(go_floor)"
  if go="$(found_go)"; then
    SOURCE=system
    log "Source: system -- $go is go$(go_version "$go"), which satisfies go.mod's $floor"
    return
  fi

  # Say what was rejected. A machine with go1.22 installed and a script that
  # silently downloads go1.23 is a puzzle; a line saying which one was too old
  # is not.
  local onpath
  onpath="$(command -v go 2>/dev/null || true)"
  if [[ -n "$onpath" ]]; then
    warn "$onpath is go$(go_version "$onpath"), older than go.mod's $floor -- looking for a newer one"
  fi

  if internal_reachable; then
    SOURCE=internal
    log "Source: internal -- $MIRROR_HOST answered"
    return
  fi
  if reachable "$GO_DL_HOST/dl/"; then
    SOURCE=public
    log "Source: public -- $GO_DL_HOST answered"
    return
  fi

  die "no source for a Go toolchain: no local go >= $floor, and neither
       ${MIRROR_HOST:-no internal mirror configured} nor $GO_DL_HOST is reachable.

  If you have a Go tarball already, point at it and re-run:
      GLIDERGO_GO_TARBALL=/path/to/go$floor.linux-$DEB_ARCH.tar.gz $0
  Or install Go yourself and re-run; anything >= $floor on PATH is enough.
  Or force a source with --source system|internal|public to see its own error."
}

# ---------------------------------------------------------------------------
# Installing Go
# ---------------------------------------------------------------------------

install_go_from_tarball() {
  local tarball="$1"
  [[ -s "$tarball" ]] || die "not a readable tarball: $tarball"
  if (( DRY_RUN )); then
    plan "extract $tarball into $TOOLCHAIN_DIR (creating $GO_DIR)"
    return
  fi
  mkdir -p "$TOOLCHAIN_DIR"
  # The tarball's top-level directory is `go`, so this creates $TOOLCHAIN_DIR/go.
  tar -C "$TOOLCHAIN_DIR" -xf "$tarball"
}

# public_go_pick asks go.dev what it has and prints "version url sha256".
#
# The index is JSON, and this repository already requires python3 for asset
# extraction, so parsing it with python3 is cheaper than a jq dependency or a
# regex over JSON. It picks the newest *stable* release that satisfies go.mod's
# floor -- newest rather than exactly-the-floor because a patch release is where
# security fixes live, and the floor is a floor.
public_go_pick() {
  local floor="$1" goos="$2" goarch="$3"
  curl --fail --silent --show-error --location --max-time 120 \
    "$GO_DL_HOST/dl/?mode=json&include=all" |
    python3 - "$floor" "$goos" "$goarch" <<'PY'
import json, sys

floor, goos, goarch = sys.argv[1], sys.argv[2], sys.argv[3]

def key(v):
    # "go1.23.12" -> (1, 23, 12). Release candidates sort below their release,
    # which is what we want: never pick an rc for a floor of the same version.
    v = v.removeprefix("go")
    for sep in ("rc", "beta"):
        if sep in v:
            v = v.split(sep)[0].rstrip(".")
    return tuple(int(p) for p in v.split(".") if p.isdigit())

floor_key = tuple(int(p) for p in floor.split("."))
best = None
for rel in json.load(sys.stdin):
    if not rel.get("stable"):
        continue
    if "rc" in rel["version"] or "beta" in rel["version"]:
        continue
    if key(rel["version"]) < floor_key:
        continue
    for f in rel.get("files", ()):
        if (f.get("os"), f.get("arch"), f.get("kind")) == (goos, goarch, "archive"):
            cand = (key(rel["version"]), rel["version"], f["filename"], f.get("sha256", ""))
            if best is None or cand[0] > best[0]:
                best = cand
if best is None:
    sys.exit(f"go.dev has no stable {goos}/{goarch} archive at or above {floor}")
_, version, filename, sha = best
print(version, filename, sha)
PY
}

install_go_public() {
  local floor goos goarch picked version filename sha url tarball
  floor="$(go_floor)"
  goos=linux
  case "$(uname -s)" in
    Linux) goos=linux ;;
    Darwin) goos=darwin ;;
    *) warn "unrecognised kernel $(uname -s); asking go.dev for a linux build" ;;
  esac
  case "$DEB_ARCH" in
    amd64|arm64|armhf|ppc64el|s390x) goarch="${DEB_ARCH/armhf/armv6l}"; goarch="${goarch/ppc64el/ppc64le}" ;;
    *) goarch="$DEB_ARCH" ;;
  esac

  if (( DRY_RUN )); then
    plan "GET $GO_DL_HOST/dl/?mode=json&include=all  -- find the newest stable $goos/$goarch >= $floor"
    plan "GET $GO_DL_HOST/dl/<that file>             -- ~70 MB tarball"
    plan "verify its sha256 against the digest in that JSON index"
    plan "extract it into $TOOLCHAIN_DIR (creating $GO_DIR)"
    return
  fi

  mkdir -p "$CACHE"
  log "Asking $GO_DL_HOST which stable Go >= $floor it has for $goos/$goarch"
  picked="$(public_go_pick "$floor" "$goos" "$goarch")" || die "could not read go.dev's download index"
  read -r version filename sha <<<"$picked"
  url="$GO_DL_HOST/dl/$filename"
  tarball="$CACHE/$filename"

  if [[ ! -s "$tarball" ]]; then
    log "Downloading $version ($filename)"
    curl --fail --silent --show-error --location --max-time 900 -o "$tarball.part" "$url"
    mv "$tarball.part" "$tarball"
  else
    log "Using the cached $filename"
  fi

  # Integrity, not authenticity, and the difference matters enough to write down.
  # The digest came from the same TLS host as the tarball, so this catches a
  # truncated or corrupted download and a mismatched CDN copy -- it does not
  # catch a compromised go.dev, because both halves would move together. Pin one
  # out of band with GLIDERGO_GO_SHA256 if that distinction matters to you.
  local want="${GLIDERGO_GO_SHA256:-$sha}"
  if [[ -n "$want" ]]; then
    local got
    got="$(sha256sum "$tarball" | awk '{ print $1 }')"
    [[ "$got" == "$want" ]] || die "sha256 mismatch for $filename
  expected $want
  got      $got
  The download is corrupt or the index disagrees with the file. Delete
  $tarball and re-run."
    log "sha256 verified"
  else
    warn "go.dev's index carried no sha256 for $filename; the download is unverified"
  fi

  install_go_from_tarball "$tarball"
}

install_go_internal() {
  local runtime="" c
  for c in podman docker; do command -v "$c" >/dev/null 2>&1 && runtime="$c" && break; done

  if [[ -n "$runtime" ]]; then
    if (( DRY_RUN )); then
      plan "$runtime pull $GO_IMAGE"
      plan "$runtime run --rm -v $CACHE:/out:z $GO_IMAGE  -- tar /usr/local/go out of the image"
      plan "extract that tar into $TOOLCHAIN_DIR (creating $GO_DIR)"
      return
    fi
    mkdir -p "$TOOLCHAIN_DIR" "$CACHE"
    log "Extracting the Go toolchain from $GO_IMAGE via $runtime (no root required)"
    "$runtime" pull "$GO_IMAGE"
    "$runtime" run --rm -v "$CACHE:/out:z" "$GO_IMAGE" \
      sh -c 'tar -C /usr/local -cf /out/go-toolchain.tar go'
    tar -C "$TOOLCHAIN_DIR" -xf "$CACHE/go-toolchain.tar"
    rm -f "$CACHE/go-toolchain.tar"
    return
  fi

  if (( DRY_RUN )); then
    plan "no container runtime found; fall back to Ubuntu golang-1.2x-go debs from $INTERNAL_UBUNTU"
    return
  fi
  log "No container runtime; falling back to the Ubuntu golang .deb"
  install_debs_into "$CACHE/godeb" golang-1.23-go golang-1.23-src ||
    install_debs_into "$CACHE/godeb" golang-1.22-go golang-1.22-src
  local found
  found="$(find "$CACHE/godeb/usr/lib" -maxdepth 1 -name 'go-1.*' | head -1)"
  [[ -n "$found" ]] || die "could not locate an extracted Go toolchain"
  cp -a "$found" "$GO_DIR"
}

install_go() {
  # An explicit tarball beats every network. It is the bridge between the two
  # worlds: someone with internet downloads go1.23.x.linux-amd64.tar.gz, carries
  # it across on a disk, and this script does the rest.
  if [[ -n "${GLIDERGO_GO_TARBALL:-}" ]]; then
    log "Using the tarball named by GLIDERGO_GO_TARBALL"
    install_go_from_tarball "$GLIDERGO_GO_TARBALL"
  else
    case "$SOURCE" in
      system)
        local go
        go="$(found_go)" || die "--source system, but no go >= $(go_floor) is on PATH"
        log "Using $go (go$(go_version "$go")); nothing to install"
        return
        ;;
      internal) install_go_internal ;;
      public)   install_go_public ;;
      *) die "unknown source: $SOURCE" ;;
    esac
  fi

  (( DRY_RUN )) && return
  [[ -x "$GO_DIR/bin/go" ]] || die "Go install failed: no $GO_DIR/bin/go"
  local v floor
  v="$(go_version "$GO_DIR/bin/go")"
  floor="$(go_floor)"
  version_ge "$v" "$floor" ||
    die "installed go$v, which is older than go.mod's $floor"
  log "Installed $("$GO_DIR/bin/go" version)"
}

# ---------------------------------------------------------------------------
# C libraries: the rootless sysroot, and the one-liner for people with root
# ---------------------------------------------------------------------------

# Where .deb files come from, which is NOT always where the Go toolchain came
# from -- and conflating the two is a mistake worth naming, because it is the one
# this script made first.
#
# On the airgapped build host, `auto` resolves the toolchain to `system`: Go is
# already installed at ~/.local/opt/go, so nothing needs downloading. But the
# Ubuntu archive still has to be the package mirror, because archive.ubuntu.com
# does not exist on that network. Deriving the mirror from SOURCE meant
# `--sysroot` reached for the public archive on the only machine the flag has
# ever run on. So the two are resolved separately.
DEB_SOURCE=""

ubuntu_mirror() {
  case "$DEB_SOURCE" in
    internal) printf '%s\n' "$INTERNAL_UBUNTU" ;;
    *)        printf '%s\n' "$PUBLIC_UBUNTU" ;;
  esac
}

resolve_deb_source() {
  [[ -n "$DEB_SOURCE" ]] && return
  case "$SOURCE" in
    internal|public) DEB_SOURCE="$SOURCE" ;;
    *)
      # An explicit mirror wins; then a reachable mirror; then the public
      # archive, which is also the honest default when nothing is reachable --
      # the fetch fails with a connection error naming a URL, which is a better
      # diagnostic than this script guessing.
      if [[ -n "${GLIDERGO_UBUNTU_MIRROR:-}" ]]; then
        DEB_SOURCE=public
      elif internal_reachable; then
        DEB_SOURCE=internal
      else
        DEB_SOURCE=public
      fi
      ;;
  esac
  log "C libraries: the $DEB_SOURCE Ubuntu mirror ($(ubuntu_mirror))"
}

# deps_hint prints the native way to get the C headers, for the many machines
# where the rootless sysroot is a worse answer than one apt command. It does not
# run anything: installing packages system-wide is the user's decision, and a
# script that sudo's without being asked is a script people stop trusting.
deps_hint() {
  cat <<EOF

C development headers, if you want the x11 backend and the optional extras.
gliderGo needs libX11 to build its default backend; everything else is optional.

  Debian / Ubuntu   sudo apt-get install -y build-essential pkg-config libx11-dev \\
                        libxext-dev libasound2-dev libsdl2-dev
  Fedora / RHEL     sudo dnf install -y gcc pkgconf-pkg-config libX11-devel \\
                        libXext-devel alsa-lib-devel SDL2-devel
  Arch              sudo pacman -S --needed base-devel pkgconf libx11 libxext alsa-lib sdl2
  macOS             xcode-select --install    # then the sdl2 backend via: brew install sdl2
                                              # (there is no X11 backend on macOS; see docs)
  No root at all    ./scripts/bootstrap-dev-env.sh --sysroot
                    unpacks the .deb files into .toolchain/sysroot instead.
EOF
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
  resolve_deb_source
  mkdir -p "$CACHE"
  local mirror comp
  mirror="$(ubuntu_mirror)"
  for comp in main universe; do
    if [[ ! -s "$CACHE/Packages-$comp" ]]; then
      log "Fetching Ubuntu $UBUNTU_SUITE/$comp package index for $DEB_ARCH"
      curl_auth -o "$CACHE/Packages-$comp.xz" \
        "$mirror/dists/$UBUNTU_SUITE/$comp/binary-$DEB_ARCH/Packages.xz"
      xz -dc "$CACHE/Packages-$comp.xz" > "$CACHE/Packages-$comp"
      rm -f "$CACHE/Packages-$comp.xz"
    fi
  done
}

install_debs_into() {
  local dest="$1"; shift
  fetch_apt_indexes
  mkdir -p "$dest" "$CACHE/debs"
  local pkg fn deb mirror
  mirror="$(ubuntu_mirror)"
  for pkg in "$@"; do
    fn="$(apt_filename "$pkg")"
    if [[ -z "$fn" ]]; then warn "package not found in index: $pkg"; return 1; fi
    deb="$CACHE/debs/$(basename "$fn")"
    if [[ ! -s "$deb" ]]; then
      log "Fetching $pkg"
      curl_auth -o "$deb" "$mirror/$fn"
    fi
    dpkg-deb -x "$deb" "$dest"
  done
}

install_sysroot() {
  resolve_deb_source
  if (( DRY_RUN )); then
    plan "GET $(ubuntu_mirror)/dists/$UBUNTU_SUITE/{main,universe}/binary-$DEB_ARCH/Packages.xz"
    plan "GET and dpkg-deb -x into $SYSROOT: ${SYSROOT_PACKAGES[*]}"
    plan "rewrite the .pc prefixes under $SYSROOT/usr/lib/$MULTIARCH/pkgconfig"
    return
  fi
  if ! command -v dpkg-deb >/dev/null 2>&1; then
    warn "dpkg-deb is not installed, so the rootless sysroot cannot be unpacked"
    deps_hint
    return
  fi
  log "Building rootless sysroot at $SYSROOT"
  install_debs_into "$SYSROOT" "${SYSROOT_PACKAGES[@]}" || warn "some sysroot packages were skipped"
  # .pc files carry absolute /usr prefixes; rewrite them to point at the sysroot.
  local pcdir="$SYSROOT/usr/lib/$MULTIARCH/pkgconfig"
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

# ---------------------------------------------------------------------------
# scripts/env.sh
# ---------------------------------------------------------------------------

# write_env_file emits the file you source before building. It is gitignored
# because it holds this machine's absolute paths; nothing in the build requires
# it, and `make` finds a Go on PATH without it.
#
# GOROOT is written only when this script installed the toolchain. For a system
# Go, pinning GOROOT is actively harmful -- it survives the day you upgrade or
# switch toolchains, and then the go binary and its library disagree.
write_env_file() {
  local goroot_line path_line
  if [[ "$SOURCE" == system && ! -x "$GO_DIR/bin/go" ]]; then
    goroot_line="# Using the Go already on your PATH ($(command -v go)); GOROOT is left alone
# on purpose, so that upgrading or switching your toolchain keeps working."
    path_line="export PATH=\"\$HOME/.local/gopath/bin:\$PATH\""
  else
    goroot_line="export GOROOT=\"$GO_DIR\""
    path_line="export PATH=\"$GO_DIR/bin:\$HOME/.local/gopath/bin:\$PATH\""
  fi

  if (( DRY_RUN )); then
    plan "write $REPO_ROOT/scripts/env.sh, recording source=$SOURCE"
    return
  fi

  cat > "$REPO_ROOT/scripts/env.sh" <<EOF
# Generated by scripts/bootstrap-dev-env.sh -- source this before building.
#   . scripts/env.sh
#
# Dependency source for this machine: $SOURCE
$goroot_line
$path_line
export GOPATH="\$HOME/.local/gopath"

# GOPROXY=off is a policy, not just a fact about a network.
#
# It began as a description of the airgapped build host, which has no Go module
# proxy at all. It is kept on machines that *do* have one because gliderGo
# depends on the standard library only -- see internal/module -- and \`off\` turns
# "quietly downloads something" into an immediate, legible error. If you are
# deliberately evaluating a dependency, override it for that command:
#     GOPROXY=https://proxy.golang.org,direct GOFLAGS= go get example.com/x
export GOPROXY=off
export GOFLAGS=-mod=mod

# GOTOOLCHAIN=local means "use the go binary I have, never download another".
# Without it, a \`go\` directive in go.mod newer than this toolchain makes the go
# command fetch a newer one -- which fails on an airgapped host with a confusing
# error, and on any host with GOPROXY=off. This script's job is to install a
# toolchain that satisfies go.mod in the first place, so the download is never
# wanted. If you hit the version error anyway, either re-run the bootstrap or:
#     GOPROXY=https://proxy.golang.org,direct GOTOOLCHAIN=auto make check
export GOTOOLCHAIN=local
export CGO_ENABLED=1

# Optional sysroot (created by --sysroot) for SHM / SDL2 / ALSA headers.
if [ -d "$SYSROOT" ]; then
  export GLIDERGO_SYSROOT="$SYSROOT"
  export PKG_CONFIG_PATH="$SYSROOT/usr/lib/$MULTIARCH/pkgconfig:$SYSROOT/usr/share/pkgconfig:\${PKG_CONFIG_PATH:-}"
  export CGO_CFLAGS="-I$SYSROOT/usr/include \${CGO_CFLAGS:-}"
  export CGO_LDFLAGS="-L$SYSROOT/usr/lib/$MULTIARCH -Wl,-rpath,$SYSROOT/usr/lib/$MULTIARCH \${CGO_LDFLAGS:-}"
fi
EOF
  log "Wrote scripts/env.sh (source=$SOURCE)"
}

# ---------------------------------------------------------------------------
# --check
# ---------------------------------------------------------------------------

check_env() {
  local ok=0 floor go v
  floor="$(go_floor)"
  printf '\n%-34s %s\n' "CHECK" "RESULT"
  probe() { printf '%-34s %s\n' "$1" "$2"; }

  if go="$(found_go)"; then
    v="$(go_version "$go")"
    probe "go toolchain" "go$v at $go"
  elif go="$(command -v go 2>/dev/null)"; then
    probe "go toolchain" "go$(go_version "$go") -- TOO OLD, go.mod needs $floor"
    ok=1
  else
    probe "go toolchain" "MISSING (go.mod needs $floor)"
    ok=1
  fi
  probe "go.mod requirement" "$floor"

  command -v gcc >/dev/null && probe "cgo C compiler" "$(gcc -dumpversion) ($MULTIARCH)" || { probe "cgo C compiler" "MISSING"; ok=1; }
  command -v python3 >/dev/null && probe "python3 (asset extraction)" "$(python3 -c 'import sys; print("%d.%d.%d" % sys.version_info[:3])')" || { probe "python3 (asset extraction)" "MISSING"; ok=1; }
  if pkg-config --exists x11 2>/dev/null; then probe "libX11 dev (x11.pc)" "$(pkg-config --modversion x11)"; else probe "libX11 dev (x11.pc)" "MISSING -- x11 backend will not build"; ok=1; fi
  pkg-config --exists xext 2>/dev/null && probe "libXext dev (MIT-SHM)" "$(pkg-config --modversion xext)" || probe "libXext dev (MIT-SHM)" "absent (optional)"
  pkg-config --exists sdl2 2>/dev/null && probe "SDL2 dev" "$(pkg-config --modversion sdl2)" || probe "SDL2 dev" "absent (optional)"
  [[ -n "${DISPLAY:-}" ]] && probe "X display" "$DISPLAY" || probe "X display" "unset (headless: null backend, or Xvfb)"
  [[ -e /dev/snd ]] && probe "audio device" "present" || probe "audio device" "absent (null audio sink)"
  probe "original source" "$([[ -d "$REPO_ROOT/GliderPRO/Sources" ]] && echo present || echo MISSING)"
  probe "extracted assets" "$([[ -d "$REPO_ROOT/assets/extracted/art" ]] && echo present || echo "absent (run: make assets)")"
  probe "scripts/env.sh" "$([[ -f "$REPO_ROOT/scripts/env.sh" ]] && echo present || echo "absent (optional: make finds go on PATH)")"

  # Which networks this machine can see. Reported rather than acted on: --check
  # installs nothing, and knowing the answer is most of diagnosing a bootstrap.
  if have_internal; then
    internal_reachable \
      && probe "internal mirror ($MIRROR_HOST)" "reachable" \
      || probe "internal mirror ($MIRROR_HOST)" "unreachable"
  else
    probe "internal mirror" "not configured (set GLIDERGO_MIRROR_HOST)"
  fi
  reachable "$GO_DL_HOST/dl/" && probe "public $GO_DL_HOST" "reachable" || probe "public $GO_DL_HOST" "unreachable"

  if (( ok )); then
    printf '\n'
    warn "something required is missing above"
    deps_hint
  fi
  return $ok
}

# ---------------------------------------------------------------------------
# Arguments
# ---------------------------------------------------------------------------

SOURCE="${GLIDERGO_DEP_SOURCE:-auto}"
while (( $# )); do
  case "$1" in
    --sysroot)  MODE=sysroot ;;
    --check)    MODE=check ;;
    --dry-run|-n) DRY_RUN=1 ;;
    --source)   shift; [[ $# -gt 0 ]] || die "--source needs a value: system, internal, public or auto"; SOURCE="$1" ;;
    --source=*) SOURCE="${1#--source=}" ;;
    # Print the header block and stop at the first line of code, so that help
    # cannot drift out of date the way a hardcoded line range does.
    -h|--help)  awk 'NR==1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "$0"; exit 0 ;;
    *) die "unknown flag: $1 (try --help)" ;;
  esac
  shift
done

case "$SOURCE" in
  system|internal|public|auto) ;;
  *) die "unknown --source: $SOURCE (want system, internal, public or auto)" ;;
esac

case "$MODE" in
  check)
    check_env
    exit $?
    ;;
  sysroot|go-only)
    (( DRY_RUN )) && log "Dry run: nothing will be downloaded, written or extracted"
    choose_source
    install_go
    [[ "$MODE" == sysroot ]] && install_sysroot
    write_env_file
    if (( DRY_RUN )); then
      printf '\n'
      log "Dry run complete. Re-run without --dry-run to do the above."
      exit 0
    fi
    check_env || true
    ;;
esac

cat <<'EOF'

Next:
  . scripts/env.sh          # put the toolchain on PATH (optional: make finds go itself)
  make check                # build + smoke-test the platform layer
  make assets               # extract the 1994 art, sound, houses and movies
EOF
