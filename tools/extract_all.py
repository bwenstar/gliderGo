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

assets/extracted/ is gitignored on purpose -- it is derived data, and the
derivation is this file.
"""

import argparse
import hashlib
import json
import os
import sys

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
    os.makedirs(out, exist_ok=True)
    manifest = {"source": "GliderPRO/ (vendored read-only, GPLv2)",
                "produced_by": "tools/extract_all.py",
                "buckets": {}, "inputs": inputs_provenance()}

    for b in want:
        print("=" * 72)
        print("== %s" % b)
        print("=" * 72)
        manifest["buckets"][b] = STEPS[b](out)
        th, n = tree_hash(os.path.join(out, b))
        manifest["buckets"][b]["files"] = n
        manifest["buckets"][b]["tree_sha256"] = th
        print()

    mpath = os.path.join(out, "manifest.json")
    if want != list(BUCKETS) and os.path.exists(mpath):
        # A partial run must not silently drop the other buckets' records.
        with open(mpath) as fh:
            old = json.load(fh)
        merged = old.get("buckets", {})
        merged.update(manifest["buckets"])
        manifest["buckets"] = merged
    with open(mpath, "w") as fh:
        json.dump(manifest, fh, indent=1, sort_keys=True)

    print("=" * 72)
    rel = os.path.relpath(mpath, ROOT)
    print("wrote %s" % (mpath if rel.startswith("..") else rel))
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
        return 1
    print("  count check: %d expectations, all met"
          % sum(1 for (b, _k) in EXPECT if b in manifest["buckets"]))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
