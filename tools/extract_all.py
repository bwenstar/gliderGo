#!/usr/bin/env python3
"""extract_all.py -- the one command that turns GliderPRO/ into assets/extracted/.

The probe_*.py scripts are *inspection* CLIs: they exist to answer questions
about the 1994 resources and they print prose.  This driver is the build step.
It calls them in dependency order, writes one tree the Go side can embed, and
emits a top-level manifest.json recording what came out, what was deliberately
skipped and which input bytes it all came from.

    python3 tools/extract_all.py                    # -> assets/extracted/
    python3 tools/extract_all.py --out /tmp/x       # somewhere else
    python3 tools/extract_all.py --only art,sound   # one bucket, while iterating

Staging
-------
Nothing is written into the output tree while the extraction runs.  Each run
stages into a dotted sibling -- `assets/.extracted.tmp-<pid>` -- and renames it
into place once every count checks out, holding `assets/.extracted.lock` for
the duration so that two runs refuse rather than interleave.  See the comments
above `only_one_run` for why, which is a golden PNG that was read while it was
half-written (docs/IMPROVEMENTS.md 5.2).

Output tree
-----------
    res/<TYPE>/<id>.bin   all 538 resources of Glider PRO.r, raw, untyped
    art/                  sheet/ object/ strip/ bg/ ui/ misc/ + manifest.json
    sound/                snd_<id>.pcm  + manifest.tsv        (application)
    sound/houses/         <house>_<id>.pcm + manifest.tsv     (custom triggers)
    houses/<name>.house   the 22 house data forks, BinHex removed
    houses/<name>.rsrc    their resource forks (where the custom sounds live)
    houseart/<name>/      each house's own PICTs and 'bnds', decoded
    movie/<name>.idx8     the 15 TV movies as flat 8-bit index buffers
    manifest.json         counts, skips, provenance, per-bucket tree hashes

Reproducibility
---------------
Nothing here records a wall-clock time, so two runs over the same GliderPRO/
produce byte-identical output.  That is a property worth having, not an
accident: `manifest.json` carries a `tree_sha256` per bucket, so

    python3 tools/extract_all.py --out /tmp/a && python3 tools/extract_all.py --out /tmp/b
    diff <(python3 -c '...print hashes...') ...

or simply diffing two manifests proves the pipeline is deterministic.  The
`inputs` section hashes every file read out of GliderPRO/, so a manifest also
proves *which* vendored source the assets came from.

assets/extracted/ is committed, unlike most derived data, because a fresh clone
has neither the 1994 CD nor a Python interpreter's worth of patience: the game
plays without ever running this script.  So a run of it shows up as a diff over
binary files, and a *deterministic* run shows up as no diff at all -- which is
what `make assets-check` turns into a test.  The staging above is part of that
bargain: an extraction that is cancelled or that miscounts leaves the committed
tree exactly as it was.
"""

import argparse
import contextlib
import hashlib
import json
import os
import shutil
import sys

# Advisory locking, by whichever of the two the platform has. Both are in the standard
# library and both are released by the kernel when the holder dies, which is the property
# that matters here: CI cancels a job with a signal the process never sees, and a lock
# that the holder had to clean up would be a lock that is stale half the time.
#
# Both imports are guarded because both platforms really run this file. fcntl is absent on
# Windows, and .github/workflows/ci.yml extracts to a temp tree on the Windows runner as a
# best-effort step; msvcrt is absent everywhere else.
try:
    import fcntl
except ImportError:
    fcntl = None
try:
    import msvcrt
except ImportError:
    msvcrt = None

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, HERE)

import extract_art                                            # noqa: E402
import extract_house_art                                      # noqa: E402
import probe_house                                            # noqa: E402
import probe_mov                                              # noqa: E402
import probe_rez                                              # noqa: E402
import probe_snd                                              # noqa: E402

REZ = os.path.join(ROOT, "GliderPRO", "Glider PRO.r")
HOUSES = os.path.join(ROOT, "GliderPRO", "Houses")
DEFAULT_OUT = os.path.join(ROOT, "assets", "extracted")

BUCKETS = ("res", "art", "sound", "houses", "houseart", "movie")

# What the analysis pass counted, independently of this script.  Checked after
# every run so that a decoder change, a re-vendored source or a silently
# dropped resource fails the build instead of quietly shrinking the asset set.
# Sources: docs/analysis/resource-fork.md (538/35/152), audio.md (70 app,
# 63 house = 58 stdSH + 5 cmpSH), houses-inventory.md (22/4070),
# quicktime-movies.md (15), graphics-assets.md 7.8 (18 backgrounds).  The
# houseart counts are this pipeline's own measurement of the 22 house forks.
EXPECT = {
    ("res", "resources"): 538,
    ("res", "types"): 35,
    ("art", "pict_total"): 152,
    ("art", "backgrounds"): 18,
    ("sound", "app_pcm"): 70,
    ("sound", "house_pcm"): 58,
    ("houses", "houses"): 22,
    ("houses", "rooms"): 4070,
    ("houseart", "pict_total"): 919,
    ("houseart", "bnds_total"): 70,
    ("movie", "movies"): 15,
}


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as fh:
        for chunk in iter(lambda: fh.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def tree_hash(root):
    """One hash over (relpath, content) for every file under root, sorted.

    Order-independent of the filesystem, so it is comparable across machines.
    """
    h = hashlib.sha256()
    n = 0
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames.sort()
        for fn in sorted(filenames):
            p = os.path.join(dirpath, fn)
            rel = os.path.relpath(p, root)
            h.update(rel.encode("utf-8") + b"\0")
            h.update(sha256_file(p).encode("ascii") + b"\0")
            n += 1
    return h.hexdigest(), n


def count_files(root, ext=None):
    n = 0
    for _dp, _dn, fns in os.walk(root):
        n += sum(1 for f in fns if ext is None or f.endswith(ext))
    return n


def read_skips(tsv):
    """Skip rows written by probe_snd's extractors, read back off disk."""
    out = []
    if not os.path.exists(tsv):
        return out
    with open(tsv) as fh:
        head = fh.readline().rstrip("\n").split("\t")
        for line in fh:
            row = line.rstrip("\n").split("\t")
            rec = dict(zip(head, row))
            status = rec.get("status", "ok")
            if status != "ok":
                out.append({"house": rec.get("house"), "id": rec.get("id"),
                            "name": rec.get("name"), "reason": status})
    return out


def house_files():
    return [os.path.join(HOUSES, f) for f in sorted(os.listdir(HOUSES))
            if f.endswith(".binhex")]


# ------------------------------------------------------------------ buckets

def do_res(out):
    """Every resource in the DeRez dump, raw.  Feeds art extraction."""
    d = os.path.join(out, "res")
    os.makedirs(d, exist_ok=True)
    probe_rez.cmd_extract(REZ, d)
    types = sorted(x for x in os.listdir(d) if os.path.isdir(os.path.join(d, x)))
    return {"dir": "res", "resources": count_files(d, ".bin"),
            "types": len(types), "type_names": types}


def do_art(out):
    """PICT -> PNG.  extract_art self-checks that all 152 PICTs are accounted
    for and raises if one is unclassified, so a new resource cannot slip past."""
    resdir = os.path.join(out, "res")
    if not os.path.isdir(os.path.join(resdir, "PICT")):
        sys.exit("extract_all: art needs res/ first (drop --only, or run res)")
    d = os.path.join(out, "art")
    art = extract_art.run(resdir, d)
    return {"dir": "art",
            "png": count_files(d, ".png"),
            "sheets": len(art["sheets"]), "strips": len(art["strips"]),
            "objects": len(art["objects"]),
            "backgrounds": len(art["backgrounds"]),
            "ui": len(art["ui"]), "misc": len(art["misc"]),
            "pict_buckets": {k: len(v) for k, v in art["buckets"].items()},
            "pict_total": sum(len(v) for v in art["buckets"].values())}


def do_sound(out):
    """'snd ' -> raw unsigned 8-bit PCM, both the application's and the houses'."""
    d = os.path.join(out, "sound")
    hd = os.path.join(d, "houses")
    probe_snd.cmd_extract([d])
    print()
    probe_snd.cmd_extract_houses([hd])
    skips = read_skips(os.path.join(hd, "manifest.tsv"))
    return {"dir": "sound",
            "app_pcm": len([f for f in os.listdir(d) if f.endswith(".pcm")]),
            "house_pcm": len([f for f in os.listdir(hd) if f.endswith(".pcm")]),
            "app_rate_hz": round(probe_snd.fixed_to_hz(probe_snd.RATE_22K), 4),
            "skipped": skips}


def do_houses(out):
    """BinHex 4.0 off, both forks to disk, so the Go loader never sees BinHex."""
    d = os.path.join(out, "houses")
    os.makedirs(d, exist_ok=True)
    rows = []
    for path in house_files():
        info = probe_house.binhex_decode(path)
        stem = os.path.basename(path)[:-7]
        dfp = os.path.join(d, stem + ".house")
        rfp = os.path.join(d, stem + ".rsrc")
        with open(dfp, "wb") as fh:
            fh.write(info["data"])
        with open(rfp, "wb") as fh:
            fh.write(info["rsrc"])
        h = probe_house.parse_house(info["data"])
        # A 2-byte tail is expected, not a corruption: a PowerPC build sized the
        # handle with sizeof(houseType) = 868 instead of offsetof(rooms) = 866,
        # so LopOffExtraRooms left two slack bytes (docs/analysis/house-format.md
        # 12.4).  Only `Sampler` shows it.  Anything else is a real surprise.
        tail = h["fileSize"] - h["expectedSize"]
        if tail not in (0, 2):
            raise ValueError("%s: %d bytes past %d rooms -- not the documented "
                             "PowerPC 2-byte slack; see house-format.md 12.4"
                             % (stem, tail, h["nRooms"]))
        rows.append({
            "house": stem, "file": "houses/%s.house" % stem,
            "type": info["type"].decode("mac-roman"),
            "creator": info["creator"].decode("mac-roman"),
            "data_bytes": len(info["data"]), "rsrc_bytes": len(info["rsrc"]),
            "version": "0x%04X" % (h["version"] & 0xFFFF),
            "rooms": h["nRooms"], "first_room": h["firstRoom"],
            "tail_bytes": tail, "banner": h["banner"],
        })
        print("house %-24s %7d data %7d rsrc  v%s  %4d rooms%s"
              % (stem, len(info["data"]), len(info["rsrc"]),
                 "0x%04X" % (h["version"] & 0xFFFF), h["nRooms"],
                 "" if not tail else "  (+%d byte PowerPC slack, expected)" % tail))
    return {"dir": "houses", "houses": len(rows),
            "rooms": sum(r["rooms"] for r in rows),
            "with_tail_slack": [r["house"] for r in rows if r["tail_bytes"]],
            "detail": rows}


def do_houseart(out):
    """Each house's own PICTs.  Custom backgrounds and every kCustomPict object
    name resources that live in the house fork, not in the application, so a
    room using either cannot be drawn from the `art` bucket alone."""
    housedir = os.path.join(out, "houses")
    resdir = os.path.join(out, "res")
    if not os.path.isdir(housedir):
        sys.exit("extract_all: houseart needs houses/ first")
    if not os.path.isdir(os.path.join(resdir, "clut")):
        sys.exit("extract_all: houseart needs res/ first (for clut 128)")
    d = os.path.join(out, "houseart")
    m = extract_house_art.run(resdir, housedir, d)
    return {"dir": "houseart", "houses": len(m["houses"]),
            "pict_total": m["pict_total"], "bnds_total": m["bnds_total"],
            "off_palette": m["off_palette"],
            "unknown_types": m["unknown_types"]}


def do_movie(out):
    """QuickTime raw/rle/smc -> one flat 8-bit index buffer per frame."""
    d = os.path.join(out, "movie")
    probe_mov.cmd_extract([d])
    return {"dir": "movie",
            "movies": len([f for f in os.listdir(d) if f.endswith(".idx8")])}


STEPS = {"res": do_res, "art": do_art, "sound": do_sound,
         "houses": do_houses, "houseart": do_houseart, "movie": do_movie}


# ----------------------------------------------------------------- publishing

# This section exists because this script used to write straight into
# assets/extracted/: for the ~70 seconds it ran, that tree was a mixture of the old
# extraction and the new one, and anything reading it saw half-written files. That is not
# hypothetical -- a `go test ./...` running beside a `make assets` failed decoding a golden
# PNG with "unexpected EOF", and it took a while to recognise as a build-system problem
# rather than a renderer one (docs/IMPROVEMENTS.md 5.2).
#
# Three properties, and the third is the one worth arguing for:
#
#   - A reader sees the old tree or the new tree. The extraction goes into a staging
#     directory and one rename publishes it.
#   - A cancelled run publishes nothing. It used to leave a tree that *looked* complete,
#     which the build's guards -- `[ -s art/manifest.json ]`, `ls houses/*.house` -- could
#     not tell from a finished one, because the manifest happens to be written last.
#   - A run whose counts are wrong publishes nothing either, and keeps its staging tree so
#     that the numbers can be looked at. EXPECT is checked before the rename rather than
#     after the write, so a regressed extractor no longer replaces 46 MB of good assets
#     with 46 MB of wrong ones and then reports a failure.
#
# It is not literally atomic. The old tree is renamed aside before the new one is renamed
# in, so there is a two-syscall window in which assets/extracted does not exist. That is
# chosen rather than overlooked: there is no portable directory swap (Linux's
# RENAME_EXCHANGE is not in the standard library), and a reader that finds no tree fails
# saying which path is missing, where a reader that finds a truncated PNG does not.


def sibling(out, suffix):
    """A dotted sibling of the output tree: assets/extracted -> assets/.extracted<suffix>.

    Beside it rather than inside it, because anything inside would be published along with
    the tree; dotted because an undotted one would show up in `git status` next to a
    committed tree, and a build artefact nobody can explain is its own small bug.
    .gitignore carries the pattern.
    """
    d, base = os.path.split(out)
    return os.path.join(d, "." + base + suffix)


def take_lock(fd):
    """Non-blocking exclusive lock, or False if somebody else holds it."""
    try:
        if fcntl is not None:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        elif msvcrt is not None:
            msvcrt.locking(fd, msvcrt.LK_NBLCK, 1)
        else:
            # Every CPython has one of the two. If a build turns up with neither, say so
            # rather than letting the caller believe the run is protected.
            print("extract_all: no file locking on this platform -- two runs at once "
                  "will interleave")
    except OSError:
        return False
    return True


@contextlib.contextmanager
def only_one_run(out):
    """Refuse, rather than interleave, when another extraction of `out` is running.

    Two `make assets` at once -- a developer in one terminal, an editor's build task in
    another -- used to interleave in silence. A release pipeline is precisely a place
    where two jobs share a checkout, so this wanted doing before one existed.

    Refuse and not wait, deliberately: the second run would produce the same bytes as the
    first (nothing here reads a clock), so waiting for it buys a caller nothing that
    reading the message and re-running does not.
    """
    path = sibling(out, ".lock")
    fd = os.open(path, os.O_CREAT | os.O_RDWR, 0o644)
    try:
        if not take_lock(fd):
            sys.exit("extract_all: %s is already being extracted by %s\n"
                     "             lock: %s"
                     % (out, holder(path) or "another process", path))
        os.ftruncate(fd, 0)
        os.write(fd, b"pid %d\n" % os.getpid())
        yield
    finally:
        os.close(fd)
        # The lock file stays. Unlinking it would race with the next run's open(), which
        # would then hold a lock on an unlinked inode and exclude nobody; one empty dotted
        # file beside the output tree is the cheaper end of that trade.


def holder(path):
    """Whatever the lock's holder recorded about itself, for the refusal message."""
    try:
        with open(path) as fh:
            return fh.readline().strip()
    except OSError:
        return ""


def seed_unbuilt(out, stage, rebuilding):
    """Hardlink the buckets this run is not rebuilding from the published tree.

    Two reasons, and the second is the one that is easy to miss. A partial run has to
    publish a *whole* tree, so the buckets it is not touching have to be in the staging
    tree before the rename. And two steps read a bucket they do not write -- art needs
    res/, houseart needs res/ and houses/ -- so `--only art` against an empty staging tree
    would exit saying so.

    Hardlinks, not copies: 46 MB is mostly PNGs, and nothing in a run writes to a bucket it
    is not rebuilding. Publishing then unlinks the old tree, which drops a reference and
    leaves the bytes owned by the new one.
    """
    if not os.path.isdir(out):
        return []
    kept = []
    for b in BUCKETS:
        src = os.path.join(out, b)
        if b in rebuilding or not os.path.isdir(src):
            continue
        dst = os.path.join(stage, b)
        try:
            shutil.copytree(src, dst, copy_function=os.link)
        except (OSError, shutil.Error):
            # A filesystem without hardlinks, or an output tree on a different one. Slower
            # and the same result.
            shutil.rmtree(dst, ignore_errors=True)
            shutil.copytree(src, dst)
        kept.append(b)
    return kept


def publish(stage, out):
    """Rename the staged tree into place, and put the old one back if that fails."""
    old = None
    if os.path.exists(out):
        old = sibling(out, ".old-%d" % os.getpid())
        shutil.rmtree(old, ignore_errors=True)
        os.replace(out, old)
    try:
        os.replace(stage, out)
    except OSError:
        if old is not None and not os.path.exists(out):
            os.replace(old, out)
        raise
    if old is not None:
        shutil.rmtree(old, ignore_errors=True)


# --------------------------------------------------------------------- main

def inputs_provenance():
    """Hash every byte of GliderPRO/ this pipeline reads."""
    files = [REZ] + house_files() + probe_mov.house_movies()
    return {os.path.relpath(p, ROOT): {"bytes": os.path.getsize(p),
                                       "sha256": sha256_file(p)}
            for p in sorted(files)}


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--out", default=DEFAULT_OUT,
                    help="output directory (default assets/extracted)")
    ap.add_argument("--only", default="",
                    help="comma-separated subset of: " + ",".join(BUCKETS))
    args = ap.parse_args(argv)

    want = [b.strip() for b in args.only.split(",") if b.strip()] or list(BUCKETS)
    for b in want:
        if b not in STEPS:
            ap.error("unknown bucket %r; pick from %s" % (b, ",".join(BUCKETS)))

    out = os.path.abspath(args.out)
    parent = os.path.dirname(out)
    if parent:
        os.makedirs(parent, exist_ok=True)
    with only_one_run(out):
        return run(out, want)


def shown(path):
    """A path as a reader of the build log wants to see it: relative, unless it is not."""
    rel = os.path.relpath(path, ROOT)
    return path if rel.startswith("..") else rel


def run(out, want):
    stage = sibling(out, ".tmp-%d" % os.getpid())
    shutil.rmtree(stage, ignore_errors=True)
    os.makedirs(stage)
    keep = False
    try:
        code, keep = extract(out, stage, want)
        return code
    finally:
        # Anything that did not get to the end is removed. An interrupted extraction is
        # incomplete by definition, and leaving 46 MB of it on disk under a name nobody
        # recognises is a smaller version of the trap this section exists to close. The one
        # thing worth keeping is a *complete* run whose counts came out wrong, which is
        # something to look at rather than something to clear up.
        if not keep:
            shutil.rmtree(stage, ignore_errors=True)


def extract(out, stage, want):
    seeded = seed_unbuilt(out, stage, want)
    print("staging in %s" % shown(stage))
    if seeded:
        print("  carried over from the published tree: %s" % ", ".join(seeded))
    manifest = {"source": "GliderPRO/ (vendored read-only, GPLv2)",
                "produced_by": "tools/extract_all.py",
                "buckets": {}, "inputs": inputs_provenance()}

    for b in want:
        print("=" * 72)
        print("== %s" % b)
        print("=" * 72)
        manifest["buckets"][b] = STEPS[b](stage)
        th, n = tree_hash(os.path.join(stage, b))
        manifest["buckets"][b]["files"] = n
        manifest["buckets"][b]["tree_sha256"] = th
        print()

    published = os.path.join(out, "manifest.json")
    if want != list(BUCKETS) and os.path.exists(published):
        # A partial run must not silently drop the other buckets' records. Read from the
        # published tree and not from the staging one, which has no manifest of its own --
        # seed_unbuilt carries buckets over, not the record of them.
        with open(published) as fh:
            old = json.load(fh)
        merged = old.get("buckets", {})
        merged.update(manifest["buckets"])
        manifest["buckets"] = merged
    with open(os.path.join(stage, "manifest.json"), "w") as fh:
        json.dump(manifest, fh, indent=1, sort_keys=True)

    print("=" * 72)
    for b in sorted(manifest["buckets"]):
        rec = manifest["buckets"][b]
        print("  %-7s %5d files  %s" % (b, rec.get("files", 0),
                                        rec.get("tree_sha256", "?")[:16]))
    total = sum(rec.get("files", 0) for rec in manifest["buckets"].values())
    print("  %-7s %5d files" % ("total", total))
    skipped = manifest["buckets"].get("sound", {}).get("skipped", [])
    if skipped:
        print("  deliberate skips (%d):" % len(skipped))
        for s in skipped:
            print("    %s %s %r: %s" % (s["house"], s["id"], s["name"],
                                        s["reason"]))

    bad = []
    for (bucket, key), want_n in sorted(EXPECT.items()):
        if bucket not in manifest["buckets"]:
            continue
        got = manifest["buckets"][bucket].get(key)
        if got != want_n:
            bad.append("%s.%s = %r, expected %d" % (bucket, key, got, want_n))
    if bad:
        print("  COUNT CHECK FAILED:")
        for line in bad:
            print("    " + line)
        print("  Either an extractor regressed or GliderPRO/ changed. Fix the")
        print("  cause, or update EXPECT *and* the doc it cites, not just one.")
        print("  NOT PUBLISHED. %s is as it was; the extraction that produced"
              % shown(out))
        print("  these numbers is kept at %s -- delete it when done." % shown(stage))
        return 1, True
    print("  count check: %d expectations, all met"
          % sum(1 for (b, _k) in EXPECT if b in manifest["buckets"]))

    publish(stage, out)
    print("wrote %s" % shown(os.path.join(out, "manifest.json")))
    return 0, False


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
