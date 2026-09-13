#!/usr/bin/env python3
"""extract_house_art.py -- the 919 PICTs that live in the *houses*, not the app.

Stage 3 of the asset pipeline.  extract_art.py handles the 152 PICTs in
"Glider PRO.r"; this handles the ones each house file carries in its own
resource fork, which the application never sees until that house is opened.

Why this exists
---------------
A room's `background` field is a PICT id, and ids from kUserBackground (3000)
up are the house's own (GliderDefines.h:522).  Separately, every kCustomPict
object names its art through data.g.height, and those ids are the house's too.
Both are common, not exotic: Demo House has 8 kCustomPict objects and 10 custom
backgrounds, and SpacePods has 2,844 kCustomPict objects.  Without these
resources a renderer cannot draw most of the shipped houses at all.

Output
------
    houseart/<house>/pict/<id>.png     RGB, no alpha
    houseart/<house>/bnds/<id>.bin     raw 'bnds' record, 8 bytes
    houseart/manifest.json

Why RGB and not RGBA
--------------------
The same PICT can be reached two ways -- srcCopy for a background, `transparent`
white-key for a kCustomPict -- and which one applies is decided by the room
data, not by the resource.  Baking an alpha channel here would have to guess.
The Go loader converts RGB back to palette indices exactly (the palette is
injective) and applies the transfer mode at the call site, which is where the
original decides it too.

The one wrinkle: 24 of the 919 carry their own ColorTable with colours outside
'clut' 128.  The original hands those to QuickDraw, which colour-matches them
into the 8-bit destination through a Color Manager inverse table.  This tool
records which pictures they are; the Go loader falls back to nearest-in-RGB for
their off-palette pixels and counts them.  Only Teddy World (23) and The Asylum
Pro (1) are affected.
"""

import json
import os
import struct
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
sys.path.insert(0, HERE)

import probe_house as PH                                        # noqa: E402
import probe_pict as PP                                         # noqa: E402

# Resource types a house fork can hold.  'snd ' is already handled by
# probe_snd.cmd_extract_houses; 'vers' is a Finder version record and the icon
# family is the Finder's custom-icon set -- neither has any bearing on
# rendering.  Anything outside this set is reported, not ignored, so a house
# type nobody has looked at cannot pass unnoticed.
#
# Note what is *absent*: no house carries a 'Date' resource, so
# LoadGraphicSpecial's second fallback (RoomGraphics.c:142) is dead code in
# every shipped house.
KNOWN_TYPES = frozenset(("PICT", "bnds", "snd ", "vers",
                         "ICN#", "icl8", "icl4", "ics#", "ics8", "ics4"))

# From GliderDefines.h:522.  Ids at or above this are the house's own
# backgrounds; the 18 built-ins are 2000..2017.
kUserBackground = 3000


def load_palette(resdir):
    return PP.ctab_to_palette(PP.parse_clut(
        open(os.path.join(resdir, "clut", "128.bin"), "rb").read()))


def house_forks(housedir):
    return [(f[:-5], os.path.join(housedir, f))
            for f in sorted(os.listdir(housedir)) if f.endswith(".rsrc")]


def run(resdir, housedir, outdir):
    pal = load_palette(resdir)
    palset = set(pal)

    houses = []
    unknown_types = {}
    total_pict = total_bnds = 0
    offpalette = []

    for stem, path in house_forks(housedir):
        with open(path, "rb") as fh:
            res = fh.read()
        fork = PH.parse_resource_fork(res)

        extra = sorted(set(fork) - KNOWN_TYPES)
        if extra:
            unknown_types[stem] = extra

        pdir = os.path.join(outdir, stem, "pict")
        bdir = os.path.join(outdir, stem, "bnds")
        picts = []

        for (rid, _name, off, ln) in sorted(fork.get("PICT", [])):
            blob = res[off:off + ln]
            pic = PP.Picture(blob).parse()
            w, h, rgb, _info = PP.rasterise(pic, pal)

            cols = set()
            for i in range(w * h):
                cols.add((rgb[i * 3], rgb[i * 3 + 1], rgb[i * 3 + 2]))
            off_pal = len(cols - palset)

            os.makedirs(pdir, exist_ok=True)
            PP.write_png_rgb(os.path.join(pdir, "%d.png" % rid), w, h, rgb)
            total_pict += 1

            rec = {"id": rid, "size": [w, h], "bytes": ln,
                   "file": "%s/pict/%d.png" % (stem, rid),
                   "role": "background" if rid >= kUserBackground and rid < 10000
                           else "custom_pict" if rid >= 10000 else "app_override"}
            if off_pal:
                rec["off_palette_colours"] = off_pal
                offpalette.append({"house": stem, "id": rid,
                                   "colours": off_pal})
            picts.append(rec)

        bnds = []
        for (rid, _name, off, ln) in sorted(fork.get("bnds", [])):
            os.makedirs(bdir, exist_ok=True)
            with open(os.path.join(bdir, "%d.bin" % rid), "wb") as fh:
                fh.write(res[off:off + ln])
            total_bnds += 1
            t, l, b, r = struct.unpack_from(">4h", res, off)
            bnds.append({"id": rid, "rect": [t, l, b, r], "bytes": ln})

        houses.append({"house": stem, "picts": picts, "bnds": bnds,
                       "types": sorted(fork)})
        print("houseart %-24s %4d PICT %3d bnds%s"
              % (stem, len(picts), len(bnds),
                 "" if not any("off_palette_colours" in p for p in picts)
                 else "  (%d off-palette)" % sum(1 for p in picts
                                                 if "off_palette_colours" in p)))

    manifest = {"houses": houses, "pict_total": total_pict,
                "bnds_total": total_bnds,
                "off_palette": offpalette,
                "unknown_types": unknown_types,
                "note": "RGB PNGs; the transfer mode is chosen by the caller, "
                        "see the module docstring"}
    os.makedirs(outdir, exist_ok=True)
    with open(os.path.join(outdir, "manifest.json"), "w") as fh:
        json.dump(manifest, fh, indent=1, sort_keys=True)
    return manifest


def main(argv):
    if len(argv) not in (0, 3):
        sys.exit("usage: extract_house_art.py [<resdir> <housedir> <outdir>]")
    if argv:
        resdir, housedir, outdir = argv
    else:
        base = os.path.join(ROOT, "assets", "extracted")
        resdir = os.path.join(base, "res")
        housedir = os.path.join(base, "houses")
        outdir = os.path.join(base, "houseart")
    m = run(resdir, housedir, outdir)
    print("%d PICT, %d bnds across %d houses"
          % (m["pict_total"], m["bnds_total"], len(m["houses"])))
    if m["off_palette"]:
        print("%d picture(s) carry colours outside clut 128:"
              % len(m["off_palette"]))
        for r in m["off_palette"]:
            print("  %-20s %5d  %d colour(s)"
                  % (r["house"], r["id"], r["colours"]))
    if m["unknown_types"]:
        print("unclassified resource types: %r" % m["unknown_types"])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
