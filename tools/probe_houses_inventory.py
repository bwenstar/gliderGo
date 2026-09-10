#!/usr/bin/env python3
"""probe_houses_inventory.py -- corpus-wide inventory and design metrics for the
22 "house" (level) files shipped with the Glider PRO 1.0.4 GPL source release.

This is the tool behind docs/analysis/original-houses.md and
docs/analysis/houses-inventory.md.  It reuses the verified BinHex 4.0 decoder,
resource-fork walker and houseType/roomType/objectType parsers from
tools/probe_house.py (which in turn implement the byte layout documented in
docs/analysis/house-format.md) and layers *design* measurements on top:

  * per-house inventory row (size, version, rooms, floors/suites, object count,
    object vocabulary, background PICT IDs, embedded art/sound, start room,
    author)
  * object-class and object-type histograms
  * room-opening derivation (a faithful re-implementation of
    GliderPRO/Sources/Room.c:816-933 DetermineRoomOpenings, plus
    DoesRoomHaveFloor / DoesRoomHaveCeiling / IsRoomAStructure)
  * room-graph reachability from firstRoom
  * prize / hazard / light / link economy
  * room-name and room-archetype statistics

Usage:
    python3 tools/probe_houses_inventory.py                     # markdown to stdout
    python3 tools/probe_houses_inventory.py --md docs/analysis/houses-inventory.md
    python3 tools/probe_houses_inventory.py --json /tmp/houses.json
    python3 tools/probe_houses_inventory.py --house "Demo House" --rooms

Everything printed is measured, never assumed.  Constants are quoted from
GliderPRO/Headers/GliderDefines.h.
"""

import argparse
import collections
import importlib.util
import json
import os
import re
import statistics
import struct
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)
DEFAULT_HOUSES = os.path.join(REPO, "GliderPRO", "Houses")


def _load_probe_house():
    spec = importlib.util.spec_from_file_location(
        "probe_house", os.path.join(HERE, "probe_house.py"))
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


ph = _load_probe_house()

# ---------------------------------------------------------------- constants
# GliderPRO/Headers/GliderDefines.h line numbers are for the CR->LF copy.

kMaxRoomObs = 24                # GliderDefines.h:250
kNumTiles = 8                   # GliderDefines.h:496
kRoomWide = 512                 # GliderDefines.h:499
kTileHigh = 322                 # GliderDefines.h:498
kHouseVersion = 0x0200          # GliderDefines.h:517
kNewHouseVersion = 0x0300       # GliderDefines.h:518
kBaseBackgroundID = 2000        # GliderDefines.h:519
kFirstOutdoorBack = 2009        # GliderDefines.h:520
kNumBackgrounds = 18            # GliderDefines.h:521
kUserBackground = 3000          # GliderDefines.h:522
kUserStructureRange = 3300      # GliderDefines.h:523
kUserBackgroundTop = 3800       # RoomInfo.c:762 (exclusive)
kRoomIsEmpty = -1               # GliderDefines.h:525
kObjectIsEmpty = -1             # GliderDefines.h:526
kNumUndergroundFloors = 8       # GliderDefines.h:535
kMaxNumRoomsH = 128             # GliderDefines.h:543
kMaxNumRoomsV = 64              # GliderDefines.h:544
kMaxStars = 4                   # GliderDefines.h:263 (per room, ObjectAdd.c:225)
kRoomVisitScore = 100           # GliderDefines.h:536
kStarPoints = 5000              # GliderDefines.h:541
kRedClockPoints = 100           # GliderDefines.h:537
kBlueClockPoints = 300          # GliderDefines.h:538
kYellowClockPoints = 500        # GliderDefines.h:539
kCuckooClockPoints = 1000       # GliderDefines.h:540
kInitialGliders = 2             # Play.c:19

SIZEOF_HOUSE_HEADER = 866
SIZEOF_ROOM = 348
SIZEOF_OBJECT = 12

# Built-in backgrounds, GliderDefines.h:227-244.
BUILTIN_BACKS = {
    2000: "kSimpleRoom", 2001: "kPaneledRoom", 2002: "kBasement",
    2003: "kChildsRoom", 2004: "kAsianRoom", 2005: "kUnfinishedRoom",
    2006: "kSwingersRoom", 2007: "kBathroom", 2008: "kLibrary",
    2009: "kGarden", 2010: "kSkywalk", 2011: "kDirt", 2012: "kMeadow",
    2013: "kField", 2014: "kRoof", 2015: "kSky", 2016: "kStratosphere",
    2017: "kStars",
}
# IsRoomAStructure explicit list, Room.c:790-801.
STRUCTURE_BACKS = {2001, 2000, 2003, 2004, 2005, 2006, 2007, 2008, 2010, 2014}
# DoesRoomHaveFloor, Room.c:1138-1167.
NO_FLOOR_BACKS = {2015, 2016, 2017}
# DoesRoomHaveCeiling, Room.c:1172-1201.
NO_CEILING_BACKS = {2009, 2012, 2013, 2014, 2015, 2016, 2017}
# GetNumberOfLights "outdoor is self-lit" list, Room.c:1038-1049.
SELF_LIT_BACKS = {2009, 2010, 2012, 2013, 2014, 2015, 2016, 2017}

# Object id -> name, from GliderDefines.h:311-434 (reused from probe_house).
OBJ_NAMES = ph.OBJ_NAMES
CLASS_OF = ph.OBJ_CLASS

CLASS_LABEL = {
    "a": "blower", "b": "furniture", "c": "prize", "d": "transport",
    "e": "switch", "f": "light", "g": "appliance", "h": "enemy",
    "i": "clutter",
}

# Objects that grant kIgnoreLeftWall / kIgnoreRightWall (ObjectRects.c:799-869),
# i.e. let the glider leave a room through an otherwise solid side wall.
IGNORE_LEFT_WALL = {0x37, 0x3A, 0x3B, 0x3E}   # DoorInLf DoorExLf WindowInLf WindowExLf
IGNORE_RIGHT_WALL = {0x38, 0x39, 0x3C, 0x3D}  # DoorInRt DoorExRt WindowInRt WindowExRt
# Objects counted as a light source by GetNumberOfLights, Room.c:1071-1091.
WINDOW_LIGHTS = {0x37, 0x38, 0x3B, 0x3C, 0x86}  # DoorInLf/Rt WindowInLf/Rt WallWindow
LAMP_OBJECTS = {0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58}
# Objects whose data.d/data.e carries a (where, who) link.  house-format.md 7.3.
LINK_TRANSPORTS = {0x33, 0x34, 0x35, 0x36, 0x3F, 0x40}   # mailboxes, ducts, invis, deluxe
LINK_SWITCHES = {0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48}
SOUND_TRIGGER = 0x49
STAIRS_UP, STAIRS_DOWN = 0x31, 0x32
kStar = 0x2C
kCustomPict = 0x6E

# Prize economy (data.c) -- GliderDefines.h:345-359.
PRIZES = {
    0x21: "kRedClock", 0x22: "kBlueClock", 0x23: "kYellowClock",
    0x24: "kCuckoo", 0x25: "kPaper", 0x26: "kBattery", 0x27: "kBands",
    0x28: "kGreaseRt", 0x29: "kGreaseLf", 0x2A: "kFoil", 0x2B: "kInvisBonus",
    0x2C: "kStar", 0x2D: "kSparkle", 0x2E: "kHelium", 0x2F: "kSlider",
}
ENEMIES = {
    0x71: "kBalloon", 0x72: "kCopterLf", 0x73: "kCopterRt", 0x74: "kDartLf",
    0x75: "kDartRt", 0x76: "kBall", 0x77: "kDrip", 0x78: "kFish",
    0x79: "kCobweb",
}
# Appliances that can kill or shove the glider (Interactions.c dispatch).
HAZARD_APPLIANCES = {0x61: "kShredder", 0x62: "kToaster", 0x6A: "kMicrowave"}
# Open-flame blowers that set the glider on fire (kBurnIt).
FLAME_BLOWERS = {0x08: "kTaper", 0x09: "kCandle", 0x0A: "kStubby",
                 0x0B: "kTiki", 0x0C: "kBBQ"}

# GliderPRO/README.md:7-13 credits.
AUTHORS = {
    "Art Museum": "unattributed (not in README credits)",
    "CD Demo House": "John Calhoun & Kim Money",
    "California or Bust!": "unattributed (not in README credits)",
    "Castle o' the Air": "unattributed (not in README credits)",
    "Davis Station": "Jonathan Chin (alias Paul Finn) & John Calhoun",
    "Demo House": "John Calhoun & Kim Money",
    "Empty House": "unattributed (not in README credits)",
    "Fun House": "unattributed (not in README credits)",
    "Grand Prix": "Jonathan Chin (alias Paul Finn)",
    "ImagineHouse PRO II": "Jonathan Chin (alias Paul Finn)",
    "In The Mirror": "Jonathan Chin (alias Paul Finn)",
    "Land of Illusion": "Ward Hartenstein",
    "Leviathan": "Jonathan Chin (alias Paul Finn)",
    "Metropolis": "Jonathan Chin (alias Paul Finn) & John Calhoun",
    "Nemo's Market": "Ward Hartenstein",
    "Rainbow's End": "Ward Hartenstein",
    "Sampler": "unattributed (not in README credits)",
    "Slumberland": "John Calhoun, Jonathan Chin, Steve Sullivan, Ward Hartenstein",
    "SpacePods": "Ward Hartenstein",
    "Teddy World": "Shawn Brenneman",
    "The Asylum Pro": "Steve Sullivan",
    "Titanic": "Jonathan Chin (alias Paul Finn) & John Calhoun",
}


# ------------------------------------------------------------ derived helpers

def class_of(what):
    for lo, hi, letter, _cname in CLASS_OF:
        if lo <= what <= hi:
            return letter
    return "?"


def is_structure(room):
    """Room.c:763-811 IsRoomAStructure."""
    bg = room["background"]
    if bg >= kUserBackground:
        if room["bounds"] != 0:
            return (room["bounds"] & 32) == 32
        return bg < kUserStructureRange
    return bg in STRUCTURE_BACKS


def bounds_code(room, bnds):
    """boundsCode for user art: bounds>>1, else the 'bnds' resource."""
    if room["bounds"] != 0:
        return room["bounds"] >> 1
    return bnds.get(room["background"], 0)


def openings(room, bnds):
    """Faithful port of Room.c:816-933 DetermineRoomOpenings (+1138-1201).

    Returns
      left/right   : the leftOpen / rightOpen *flags* (Room.c:831-919)
      leftPass/rightPass : whether leftThresh/rightThresh got the "no wall"
                     value, which is what CheckEscapeLeft/Right actually test
                     (Interactions.c:572 CheckEscapeLeft, :662 CheckEscapeRight).  Differ for
                     exactly one background, kDirt -- see the doc.
      top/bottom   : topOpen / bottomOpen (Room.c:924-932), the only vertical
                     gate CheckEscapeUp/Down consults.
    """
    bg = room["background"]
    lt = room["tiles"][0]
    rt = room["tiles"][kNumTiles - 1]
    if bg >= kUserBackground:
        bc = bounds_code(room, bnds)
        left = (bc & 0x0001) == 0x0001
        right = (bc & 0x0004) == 0x0004
        bottom = (bc & 0x0008) == 0x0008          # no floor  -> bottom open
        top = (bc & 0x0002) == 0x0002             # no ceiling-> top open
        return {"left": left, "right": right, "leftPass": left,
                "rightPass": right, "top": top, "bottom": bottom,
                "src": "bounds" if room["bounds"] else "bnds"}
    if bg in (2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2015):
        left, right = (lt != 0), (rt != kNumTiles - 1)
        lpass, rpass = (lt != 0), (rt != kNumTiles - 1)
    elif bg == 2011:                              # kDirt -- Room.c:857-870
        left, right = (lt != 0), (rt != kNumTiles - 1)
        lpass, rpass = (lt != 1), (rt != kNumTiles - 1)   # note: != 1, not != 0
    elif bg == 2012:                              # kMeadow -- Room.c:872-885
        left, right = (lt != 6), (rt != 7)
        lpass, rpass = (lt != 6), (rt != 7)
    elif bg in (2009, 2010, 2013, 2016, 2017):    # Garden Skywalk Field Strat Stars
        left = right = lpass = rpass = True
    else:                                         # default incl. kRoof 2014
        left, right = (lt != 0), (rt != kNumTiles - 1)
        lpass, rpass = (lt != 0), (rt != kNumTiles - 1)
    return {"left": left, "right": right, "leftPass": lpass, "rightPass": rpass,
            "top": bg in NO_CEILING_BACKS,
            "bottom": bg in NO_FLOOR_BACKS,
            "src": "tiles"}


def vertical_passable(room, op):
    """topPass / bottomPass including the kDirt tile rules
    (Interactions.c:244-277 CheckEscapeUp, :378-416 CheckEscapeDown) and kManhole's
    kIgnoreGround (ObjectRects.c:640-647, Interactions.c:404-441)."""
    top, bottom = op["top"], op["bottom"]
    if room["background"] == 2011:                # kDirt
        if any(t in (5, 6) for t in room["tiles"]):
            top = True
        if any(t in (2, 3) for t in room["tiles"]):
            bottom = True
    if any(o["what"] == 0x1D for o in room["objects"]):   # kManhole
        bottom = True
    return top, bottom


def number_of_lights(room):
    """Room.c:970-1097 GetNumberOfLights, play-mode branch, using `initial`
    (Play.c:663-672 copies initial -> state before play).

    The background switch only *pre-seeds* count; the object loop runs whenever
    that seed is 0, so a kDirt room whose tiles are not all 0 still counts its
    window/lamp objects.  Do not early-return here."""
    bg = room["background"]
    if bg in SELF_LIT_BACKS:
        return 1
    count = 0
    if bg == 2011 and all(t == 0 for t in room["tiles"]):   # kDirt
        return 1
    for o in room["objects"]:
        w = o["what"]
        if w in WINDOW_LIGHTS:
            count += 1
        elif w in LAMP_OBJECTS:
            if o.get("initial"):
                count += 1
    return count


def merge_floor_suite(floor, suite):
    """Link.c:34 MergeFloorSuite, with the +kNumUndergroundFloors bias."""
    return suite * 100 + (floor + kNumUndergroundFloors)


def extract_floor_suite(combo, version):
    """Link.c:41-53 ExtractFloorSuite."""
    if version < 0x0200:
        return (combo // 100) - kNumUndergroundFloors, combo % 100
    return (combo % 100) - kNumUndergroundFloors, combo // 100


# ------------------------------------------------------------------ analysis

def analyze(path, name):
    info = ph.binhex_decode(path)
    house = ph.parse_house(info["data"])
    version = house["version"]

    # -------- resource fork
    try:
        resmap = ph.parse_resource_fork(info["rsrc"])
    except Exception:
        resmap = {}
    res_counts = {t: len(v) for t, v in resmap.items()}
    pict_ids = sorted(e[0] for e in resmap.get("PICT", []))
    snd_ids = sorted(e[0] for e in resmap.get("snd ", []))
    bnds_ids = sorted(e[0] for e in resmap.get("bnds", []))
    pict_names = {e[0]: e[1] for e in resmap.get("PICT", []) if e[1]}
    bnds = {}
    for rid, _nm, off, ln in resmap.get("bnds", []):
        if ln >= 4:
            b = info["rsrc"][off:off + 4]
            code = 0
            if b[0]:
                code += 1
            if b[1]:
                code += 2
            if b[2]:
                code += 4
            if b[3]:
                code += 8
            bnds[rid] = code

    # -------- rooms
    rooms = house["rooms"]
    real = [r for r in rooms if r["suite"] != kRoomIsEmpty]
    placeholders = len(rooms) - len(real)

    grid = {}
    for i, r in enumerate(rooms):
        if r["suite"] != kRoomIsEmpty:
            grid.setdefault((r["floor"], r["suite"]), i)

    floors = sorted({r["floor"] for r in real})
    suites = sorted({r["suite"] for r in real})

    what_counts = collections.Counter()
    class_counts = collections.Counter()
    bg_counts = collections.Counter()
    per_room_obj = []
    tiles_counter = collections.Counter()
    name_counter = collections.Counter()
    custom_pict_ids = collections.Counter()
    sound_trigger_ids = collections.Counter()
    links = []            # (srcRoom, srcSlot, what, where, who)
    dark_rooms = 0
    no_floor_rooms = 0
    no_ceiling_rooms = 0
    structure_rooms = 0
    full_rooms = 0
    numobj_mismatch = 0
    star_rooms = []
    open_dirs = collections.Counter()

    for i, r in enumerate(rooms):
        if r["suite"] == kRoomIsEmpty:
            continue
        live = [o for o in r["objects"] if o["what"] != kObjectIsEmpty]
        per_room_obj.append(len(live))
        if len(live) == kMaxRoomObs:
            full_rooms += 1
        if r["numObjects"] != len(live):
            numobj_mismatch += 1
        bg_counts[r["background"]] += 1
        name_counter[r["name"]] += 1
        for t in r["tiles"]:
            tiles_counter[t] += 1
        op = openings(r, bnds)
        for k in ("left", "right", "top", "bottom"):
            if op[k]:
                open_dirs[k] += 1
        if op["bottom"]:
            no_floor_rooms += 1
        if op["top"]:
            no_ceiling_rooms += 1
        if is_structure(r):
            structure_rooms += 1
        if number_of_lights(r) == 0:
            dark_rooms += 1
        nstars = 0
        for si, o in enumerate(r["objects"]):
            w = o["what"]
            if w == kObjectIsEmpty:
                continue
            what_counts[w] += 1
            class_counts[class_of(w)] += 1
            if w == kStar:
                nstars += 1
            if w == kCustomPict:
                custom_pict_ids[o.get("height")] += 1
            if w == SOUND_TRIGGER:
                sound_trigger_ids[o.get("where")] += 1
            if w in LINK_TRANSPORTS or w in LINK_SWITCHES:
                where = o.get("where", -1)
                who = o.get("who", 255)
                links.append((i, si, w, where, who))
        if nstars:
            star_rooms.append((i, r["name"], nstars))

    total_objects = sum(what_counts.values())

    # -------- CountTotalHousePoints, HouseInfo.c:51-105
    #   rooms*100 + 100/300/500/1000 per clock + 5000 per star
    #   + data.c.points per kInvisBonus
    total_points = len(real) * kRoomVisitScore
    invis_bonus_points = 0
    for r in real:
        for o in r["objects"]:
            w = o["what"]
            if w == 0x21:
                total_points += kRedClockPoints
            elif w == 0x22:
                total_points += kBlueClockPoints
            elif w == 0x23:
                total_points += kYellowClockPoints
            elif w == 0x24:
                total_points += kCuckooClockPoints
            elif w == kStar:
                total_points += kStarPoints
            elif w == 0x2B:                       # kInvisBonus
                total_points += o.get("points", 0)
                invis_bonus_points += o.get("points", 0)

    # -------- link integrity
    # A link field is (where, who).  where == -1 (kRoomIsEmpty) means "no link
    # at all"; who == 255 means "room set, object slot not yet chosen" -- the
    # editor writes that intermediate state (ObjectEdit.c) and the runtime
    # treats kInvisTrans with who == 255 as inert (ObjectRects.c:668).
    links_with_where = 0      # where != -1
    links_full = 0            # where != -1 and who != 255
    dangling = 0              # where != -1 but (floor,suite) cell is empty
    resolved = 0              # where != -1 and cell exists
    bad_who = 0               # cell exists, who != 255, who >= kMaxRoomObs
    who_unset = 0             # where != -1 and who == 255
    no_room = 0               # where == -100 == MergeFloorSuite(0, kRoomIsEmpty)
    dangling_hard = 0         # dangling AND who != 255 AND where != -100
    for (src, slot, w, where, who) in links:
        if where == -1:
            continue
        links_with_where += 1
        if where == -100:
            no_room += 1
        if who != 255:
            links_full += 1
        else:
            who_unset += 1
        fl, su = extract_floor_suite(where, version)
        tgt = grid.get((fl, su))
        if tgt is None:
            dangling += 1
            if who != 255 and where != -100:
                dangling_hard += 1
        else:
            resolved += 1
            if who != 255 and who >= kMaxRoomObs:
                bad_who += 1

    # -------- staircase pairing (HouseLegal.c:961-1042)
    up_unpaired = down_unpaired = up_total = down_total = 0
    for i, r in enumerate(rooms):
        if r["suite"] == kRoomIsEmpty:
            continue
        f, s = r["floor"], r["suite"]
        ups = sum(1 for o in r["objects"] if o["what"] == STAIRS_UP)
        downs = sum(1 for o in r["objects"] if o["what"] == STAIRS_DOWN)
        up_total += ups
        down_total += downs
        if ups:
            n = grid.get((f + 1, s))
            if n is None or not any(o["what"] == STAIRS_DOWN
                                    for o in rooms[n]["objects"]):
                up_unpaired += ups
        if downs:
            n = grid.get((f - 1, s))
            if n is None or not any(o["what"] == STAIRS_UP
                                    for o in rooms[n]["objects"]):
                down_unpaired += downs

    # -------- reachability BFS from firstRoom
    adj = collections.defaultdict(set)
    dir_open = {}             # room index -> (l, r, u, d) after object overrides
    for i, r in enumerate(rooms):
        if r["suite"] == kRoomIsEmpty:
            continue
        f, s = r["floor"], r["suite"]
        op = openings(r, bnds)
        objs = {o["what"] for o in r["objects"] if o["what"] != kObjectIsEmpty}
        top_ok, bottom_ok = vertical_passable(r, op)
        left_ok = op["leftPass"] or bool(objs & IGNORE_LEFT_WALL)
        right_ok = op["rightPass"] or bool(objs & IGNORE_RIGHT_WALL)
        if STAIRS_UP in objs:
            top_ok = True
        if STAIRS_DOWN in objs:
            bottom_ok = True
        dir_open[i] = (left_ok, right_ok, top_ok, bottom_ok)
        if left_ok and (f, s - 1) in grid:
            adj[i].add(grid[(f, s - 1)])
        if right_ok and (f, s + 1) in grid:
            adj[i].add(grid[(f, s + 1)])
        if top_ok and (f + 1, s) in grid:
            adj[i].add(grid[(f + 1, s)])
        if bottom_ok and (f - 1, s) in grid:
            adj[i].add(grid[(f - 1, s)])
    for (src, slot, w, where, who) in links:
        if w in LINK_TRANSPORTS and where != -1 and who != 255:
            fl, su = extract_floor_suite(where, version)
            tgt = grid.get((fl, su))
            if tgt is not None:
                adj[src].add(tgt)

    # -------- one-way horizontal / vertical passage census
    one_way_h = two_way_h = wall_h = 0
    one_way_v = two_way_v = wall_v = 0
    for i, r in enumerate(rooms):
        if r["suite"] == kRoomIsEmpty:
            continue
        f, s = r["floor"], r["suite"]
        e = grid.get((f, s + 1))
        if e is not None:
            a = dir_open[i][1]
            b = dir_open[e][0]
            if a and b:
                two_way_h += 1
            elif a or b:
                one_way_h += 1
            else:
                wall_h += 1
        n = grid.get((f + 1, s))
        if n is not None:
            a = dir_open[i][2]              # this room's ceiling is open
            b = dir_open[n][3]              # neighbour above has no floor
            if a and b:
                two_way_v += 1
            elif a or b:
                one_way_v += 1
            else:
                wall_v += 1

    # -------- undirected connected components of the same graph
    und = collections.defaultdict(set)
    for a, nbrs in adj.items():
        for b in nbrs:
            und[a].add(b)
            und[b].add(a)
    seen = set()
    comps = []
    for i in range(len(rooms)):
        if rooms[i]["suite"] == kRoomIsEmpty or i in seen:
            continue
        stack = [i]
        seen.add(i)
        size = 0
        members = set()
        while stack:
            c = stack.pop()
            size += 1
            members.add(c)
            for nx in und[c]:
                if nx not in seen:
                    seen.add(nx)
                    stack.append(nx)
        comps.append((size, members))
    comps.sort(key=lambda t: -t[0])

    first = house["firstRoom"]
    comp_first = next((sz for sz, m in comps if first in m), 0)
    dist = {}
    if 0 <= first < len(rooms) and rooms[first]["suite"] != kRoomIsEmpty:
        dist[first] = 0
        q = collections.deque([first])
        while q:
            cur = q.popleft()
            for nx in adj[cur]:
                if nx not in dist:
                    dist[nx] = dist[cur] + 1
                    q.append(nx)
    reachable = len(dist)
    unreachable = len(real) - reachable

    # -------- per-hop difficulty ramp
    ramp = collections.defaultdict(lambda: {"rooms": 0, "objs": 0, "enemy": 0,
                                            "prize": 0, "star": 0})
    for i, d in dist.items():
        r = rooms[i]
        b = ramp[d]
        b["rooms"] += 1
        for o in r["objects"]:
            w = o["what"]
            if w == kObjectIsEmpty:
                continue
            b["objs"] += 1
            if w in ENEMIES:
                b["enemy"] += 1
            if w in PRIZES:
                b["prize"] += 1
            if w == kStar:
                b["star"] += 1

    return {
        "name": name,
        "file": os.path.basename(path),
        "fileBytes": os.path.getsize(path),
        "dataBytes": len(info["data"]),
        "rsrcBytes": len(info["rsrc"]),
        "type": info["type"].decode("mac-roman"),
        "creator": info["creator"].decode("mac-roman"),
        "version": version,
        "flags": house["flags"] & 0xFFFFFFFF,
        "locked": (house["timeStamp"] & 1) == 1,
        "banner": house["banner"],
        "trailer": house["trailer"],
        "initial": house["initial"],
        "firstRoom": first,
        "firstRoomName": rooms[first]["name"] if 0 <= first < len(rooms) else "",
        "firstRoomFloorSuite": ((rooms[first]["floor"], rooms[first]["suite"])
                                if 0 <= first < len(rooms) else None),
        "nRooms": house["nRooms"],
        "realRooms": len(real),
        "placeholders": placeholders,
        "floors": floors,
        "suites": suites,
        "nFloors": len(floors),
        "nSuites": len(suites),
        "floorRange": (floors[0], floors[-1]) if floors else None,
        "suiteRange": (suites[0], suites[-1]) if suites else None,
        "gridArea": ((floors[-1] - floors[0] + 1) * (suites[-1] - suites[0] + 1)
                     if floors else 0),
        "totalObjects": total_objects,
        "totalPoints": total_points,
        "invisBonusPoints": invis_bonus_points,
        "whatCounts": dict(what_counts),
        "classCounts": dict(class_counts),
        "distinctTypes": len(what_counts),
        "bgCounts": dict(bg_counts),
        "builtinBgs": sorted(b for b in bg_counts if b < kUserBackground),
        "customBgs": sorted(b for b in bg_counts if b >= kUserBackground),
        "resCounts": res_counts,
        "pictIds": pict_ids,
        "sndIds": snd_ids,
        "bndsIds": bnds_ids,
        "bndsCodes": bnds,
        "pictNames": pict_names,
        "perRoomObj": per_room_obj,
        "objMean": (statistics.mean(per_room_obj) if per_room_obj else 0),
        "objMedian": (statistics.median(per_room_obj) if per_room_obj else 0),
        "objMax": max(per_room_obj) if per_room_obj else 0,
        "objMin": min(per_room_obj) if per_room_obj else 0,
        "fullRooms": full_rooms,
        "numObjMismatch": numobj_mismatch,
        "darkRooms": dark_rooms,
        "noFloorRooms": no_floor_rooms,
        "noCeilingRooms": no_ceiling_rooms,
        "structureRooms": structure_rooms,
        "openDirs": dict(open_dirs),
        "stars": what_counts.get(kStar, 0),
        "starRooms": star_rooms,
        "links": len(links),
        "linkSlots": len(links),
        "linksWithWhere": links_with_where,
        "linksFull": links_full,
        "linksWhoUnset": who_unset,
        "linksResolved": resolved,
        "linksDangling": dangling,
        "linksNoRoom": no_room,
        "linksDanglingHard": dangling_hard,
        "linksBadWho": bad_who,
        "oneWayH": one_way_h, "twoWayH": two_way_h, "wallH": wall_h,
        "oneWayV": one_way_v, "twoWayV": two_way_v, "wallV": wall_v,
        "starHops": sorted(dist[i] for (i, _n, _c) in star_rooms if i in dist),
        "stairsUp": up_total,
        "stairsDown": down_total,
        "stairsUpUnpaired": up_unpaired,
        "stairsDownUnpaired": down_unpaired,
        "reachable": reachable,
        "unreachable": unreachable,
        "components": len(comps),
        "largestComponent": comps[0][0] if comps else 0,
        "componentWithFirst": comp_first,
        "roomsPerFloor": {str(f): sum(1 for r in real if r["floor"] == f)
                          for f in floors},
        "roomsPerSuite": {str(s): sum(1 for r in real if r["suite"] == s)
                          for s in suites},
        "maxHop": max(dist.values()) if dist else -1,
        "ramp": {str(k): v for k, v in sorted(ramp.items())},
        "roomNames": dict(name_counter),
        "tiles": dict(tiles_counter),
        "customPictIds": {str(k): v for k, v in sorted(custom_pict_ids.items(),
                                                       key=lambda kv: -kv[1])},
        "soundTriggerIds": {str(k): v for k, v in sorted(sound_trigger_ids.items())},
        "author": AUTHORS.get(name, "?"),
        "sizeOk": len(info["data"]) == SIZEOF_HOUSE_HEADER
                  + SIZEOF_ROOM * house["nRooms"],
        "hasGame": house["hasGame"],
    }


# ------------------------------------------------------------------ reporting

def compact_ids(ids):
    """[3000,3001,3002,3005] -> '3000-3002, 3005'"""
    if not ids:
        return "-"
    out = []
    start = prev = ids[0]
    for v in ids[1:]:
        if v == prev + 1:
            prev = v
            continue
        out.append(str(start) if start == prev else f"{start}-{prev}")
        start = prev = v
    out.append(str(start) if start == prev else f"{start}-{prev}")
    return ", ".join(out)


def topn(counter, n, namemap=OBJ_NAMES):
    items = sorted(counter.items(), key=lambda kv: (-kv[1], kv[0]))[:n]
    return ", ".join(f"{namemap.get(k, hex(k))} {v}" for k, v in items)


def emit_markdown(rows, out):
    w = out.write
    w("<!-- generated by tools/probe_houses_inventory.py -- do not hand-edit -->\n\n")
    w("# Glider PRO 1.0.4 — shipped-house inventory (generated)\n\n")
    w(f"{len(rows)} house files in `GliderPRO/Houses/`. "
      "Every number below is measured from the decoded BinHex payload.\n\n")

    w("## Main inventory table\n\n")
    w("| # | House file | .binhex B | data fork B | rsrc fork B | ver | rooms "
      "| floors | suites | objects | distinct types | start room | author |\n")
    w("|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
    for n, r in enumerate(rows, 1):
        w(f"| {n} | `{r['file']}` | {r['fileBytes']} | {r['dataBytes']} | "
          f"{r['rsrcBytes']} | 0x{r['version']:04X} | {r['nRooms']} | "
          f"{r['nFloors']} ({r['floorRange'][0]}..{r['floorRange'][1]}) | "
          f"{r['nSuites']} ({r['suiteRange'][0]}..{r['suiteRange'][1]}) | "
          f"{r['totalObjects']} | {r['distinctTypes']} | "
          f"{r['firstRoom']} \"{r['firstRoomName']}\" | {r['author']} |\n")

    w("\n## Object vocabulary — top 15 types per house\n\n")
    w("| House | top 15 `what` codes by count |\n|---|---|\n")
    for r in rows:
        w(f"| {r['name']} | {topn(r['whatCounts'], 15)} |\n")

    w("\n## Backgrounds and embedded resources\n\n")
    w("| House | built-in bg IDs | custom bg IDs | PICT | snd | bnds | "
      "custom art? | custom sound? |\n|---|---|---|---|---|---|---|---|\n")
    for r in rows:
        w(f"| {r['name']} | {compact_ids(r['builtinBgs'])} | "
          f"{compact_ids(r['customBgs'])} | {r['resCounts'].get('PICT', 0)} | "
          f"{r['resCounts'].get('snd ', 0)} | {r['resCounts'].get('bnds', 0)} | "
          f"{'yes' if r['resCounts'].get('PICT', 0) else 'no'} | "
          f"{'yes' if r['resCounts'].get('snd ', 0) else 'no'} |\n")

    w("\n## Design metrics\n\n")
    w("| House | rooms | obj/room mean | median | max | full(24) | dark | "
      "no-floor | no-ceiling | structure | stars | links | dangling | "
      "reach/real | max hop |\n")
    w("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
    for r in rows:
        w(f"| {r['name']} | {r['realRooms']} | {r['objMean']:.1f} | "
          f"{r['objMedian']:.0f} | {r['objMax']} | {r['fullRooms']} | "
          f"{r['darkRooms']} | {r['noFloorRooms']} | {r['noCeilingRooms']} | "
          f"{r['structureRooms']} | {r['stars']} | {r['links']} | "
          f"{r['linksDangling']} | {r['reachable']}/{r['realRooms']} | "
          f"{r['maxHop']} |\n")

    w("\n## Object classes per house (9 unions of `objectType.data`)\n\n")
    w("| House | blower | furniture | prize | transport | switch | light | "
      "appliance | enemy | clutter |\n|---|---|---|---|---|---|---|---|---|---|\n")
    for r in rows:
        c = r["classCounts"]
        w(f"| {r['name']} | " + " | ".join(str(c.get(k, 0)) for k in "abcdefghi")
          + " |\n")

    w("\n## Link integrity\n\n")
    w("| House | link slots | `where`!=-1 | `who`==255 | fully linked | "
      "resolved | dangling | of which `where`==-100 | hard dangling | "
      "`who`>=24 |\n|---|---|---|---|---|---|---|---|---|---|\n")
    for r in rows:
        w(f"| {r['name']} | {r['linkSlots']} | {r['linksWithWhere']} | "
          f"{r['linksWhoUnset']} | {r['linksFull']} | {r['linksResolved']} | "
          f"{r['linksDangling']} | {r['linksNoRoom']} | "
          f"{r['linksDanglingHard']} | {r['linksBadWho']} |\n")

    w("\n## Connectivity census\n\n")
    w("Counts of *adjacent room pairs* (grid neighbours that both exist).\n\n")
    w("| House | E/W pairs 2-way | E/W 1-way | E/W walled | N/S 2-way | "
      "N/S 1-way | N/S walled | up-stairs | unpaired | down-stairs | "
      "unpaired |\n|---|---|---|---|---|---|---|---|---|---|---|\n")
    for r in rows:
        w(f"| {r['name']} | {r['twoWayH']} | {r['oneWayH']} | {r['wallH']} | "
          f"{r['twoWayV']} | {r['oneWayV']} | {r['wallV']} | {r['stairsUp']} | "
          f"{r['stairsUpUnpaired']} | {r['stairsDown']} | "
          f"{r['stairsDownUnpaired']} |\n")

    w("\n## Static reachability\n\n")
    w("| House | real rooms | components | largest | component holding "
      "`firstRoom` | BFS-reachable from `firstRoom` | max hop |\n"
      "|---|---|---|---|---|---|---|\n")
    for r in rows:
        w(f"| {r['name']} | {r['realRooms']} | {r['components']} | "
          f"{r['largestComponent']} | {r['componentWithFirst']} | "
          f"{r['reachable']} | {r['maxHop']} |\n")

    w("\n## Star placement (`kStar` 0x2C) vs. BFS depth from `firstRoom`\n\n")
    w("| House | stars | BFS hops of star rooms | max hop | star rooms |\n"
      "|---|---|---|---|---|\n")
    for r in rows:
        sr = "; ".join(f"{i} \"{n}\"" + (f" x{c}" if c > 1 else "")
                       for i, n, c in r["starRooms"])
        w(f"| {r['name']} | {r['stars']} | "
          f"{r['starHops'] if r['starHops'] else '-'} | {r['maxHop']} | "
          f"{sr if sr else '-'} |\n")

    w("\n## Banner / trailer text\n\n")
    w("| House | `banner` | `trailer` |\n|---|---|---|\n")
    for r in rows:
        b = r["banner"].replace("|", "\\|")
        t = r["trailer"].replace("|", "\\|")
        w(f"| {r['name']} | {b} | {t} |\n")

    w("\n## Per-hop difficulty ramp (rooms / objects / enemies / prizes / "
      "stars by BFS depth)\n\n")
    for r in rows:
        if not r["ramp"]:
            continue
        w(f"\n### {r['name']}\n\n")
        w("| hop | rooms | objects | obj/room | enemies | prizes | stars |\n"
          "|---|---|---|---|---|---|---|\n")
        for k in sorted(r["ramp"], key=int):
            b = r["ramp"][k]
            w(f"| {k} | {b['rooms']} | {b['objs']} | "
              f"{b['objs'] / b['rooms']:.1f} | {b['enemy']} | {b['prize']} | "
              f"{b['star']} |\n")

    # corpus aggregate
    agg_what = collections.Counter()
    agg_class = collections.Counter()
    agg_bg = collections.Counter()
    for r in rows:
        agg_what.update({int(k): v for k, v in r["whatCounts"].items()})
        agg_class.update(r["classCounts"])
        agg_bg.update({int(k): v for k, v in r["bgCounts"].items()})
    w("\n## Corpus totals\n\n")
    w(f"- houses: {len(rows)}\n")
    w(f"- rooms: {sum(r['nRooms'] for r in rows)} "
      f"(real {sum(r['realRooms'] for r in rows)})\n")
    w(f"- live objects: {sum(agg_what.values())}\n")
    w(f"- distinct `what` codes used: {len(agg_what)}\n")
    w(f"- stars: {sum(r['stars'] for r in rows)}\n")
    w("\n### Full corpus `what` histogram\n\n")
    w("| code | name | count | houses using |\n|---|---|---|---|\n")
    for code, cnt in sorted(agg_what.items()):
        houses = sum(1 for r in rows if str(code) in r["whatCounts"]
                     or code in r["whatCounts"])
        w(f"| 0x{code:02X} | {OBJ_NAMES.get(code, '?')} | {cnt} | {houses} |\n")
    w("\n### Full corpus background histogram\n\n")
    w("| PICT ID | kind | rooms | houses |\n|---|---|---|---|\n")
    for bg, cnt in sorted(agg_bg.items()):
        kind = BUILTIN_BACKS.get(bg, "user art")
        houses = sum(1 for r in rows if bg in [int(k) for k in r["bgCounts"]])
        w(f"| {bg} | {kind} | {cnt} | {houses} |\n")

    # ---- room-name motifs
    agg_names = collections.Counter()
    agg_words = collections.Counter()
    blank = 0
    for r in rows:
        for nm, c in r["roomNames"].items():
            agg_names[nm] += c
            if not nm.strip():
                blank += c
            for tok in re.findall(r"[A-Za-z']+", nm):
                agg_words[tok.lower()] += c
    w("\n### Room-name reuse across the whole corpus\n\n")
    w(f"- distinct room names: {len(agg_names)} over "
      f"{sum(agg_names.values())} rooms\n")
    w(f"- rooms with an empty name: {blank}\n")
    w("\n| room name | uses | \n|---|---|\n")
    for nm, c in agg_names.most_common(40):
        w(f"| `{nm}` | {c} |\n")
    w("\n| word in room name | uses |\n|---|---|\n")
    for tok, c in agg_words.most_common(50):
        w(f"| {tok} | {c} |\n")

    # ---- tile histogram
    agg_tiles = collections.Counter()
    for r in rows:
        agg_tiles.update({int(k): v for k, v in r["tiles"].items()})
    w("\n### Corpus `tiles[8]` value histogram\n\n")
    w("| tile value | count |\n|---|---|\n")
    for t, c in sorted(agg_tiles.items()):
        w(f"| {t} | {c} |\n")

    # ---- kCustomPict / kSoundTrigger IDs
    w("\n### `kCustomPict` (0x6E) PICT IDs, top 20 per corpus\n\n")
    agg_cp = collections.Counter()
    for r in rows:
        agg_cp.update({int(k): v for k, v in r["customPictIds"].items()})
    w("| PICT ID | placements |\n|---|---|\n")
    for pid, c in agg_cp.most_common(20):
        w(f"| {pid} | {c} |\n")
    w("\n### `kSoundTrigger` (0x49) `where` (= `snd ` resource ID) values\n\n")
    w("| House | ids |\n|---|---|\n")
    for r in rows:
        if r["soundTriggerIds"]:
            w(f"| {r['name']} | " +
              ", ".join(f"{k}x{v}" for k, v in r["soundTriggerIds"].items())
              + " |\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--houses", default=DEFAULT_HOUSES)
    ap.add_argument("--md", help="write markdown here (default stdout)")
    ap.add_argument("--json", help="also dump raw metrics as JSON")
    ap.add_argument("--house", action="append", default=[],
                    help="restrict to these house names")
    args = ap.parse_args()

    files = sorted(f for f in os.listdir(args.houses) if f.endswith(".binhex"))
    rows = []
    for f in files:
        name = f[:-len(".binhex")]
        if args.house and name not in args.house:
            continue
        rows.append(analyze(os.path.join(args.houses, f), name))

    if args.json:
        with open(args.json, "w") as fh:
            json.dump(rows, fh, indent=1, default=str)
        sys.stderr.write(f"wrote {args.json}\n")

    if args.md:
        with open(args.md, "w") as fh:
            emit_markdown(rows, fh)
        sys.stderr.write(f"wrote {args.md}\n")
    else:
        emit_markdown(rows, sys.stdout)


if __name__ == "__main__":
    sys.exit(main())
