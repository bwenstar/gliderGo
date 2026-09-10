#!/usr/bin/env python3
"""probe_house.py -- BinHex 4.0 decoder + Glider PRO "house" file parser.

Written while reverse-documenting Glider PRO 1.0.4 for the Go port.  Nothing in
here is meant to be pretty; it exists so that claims in
docs/analysis/house-format.md can be checked against real bytes.

Usage:
    python3 tools/probe_house.py "GliderPRO/Houses/Demo House.binhex" [--rooms N]
    python3 tools/probe_house.py --forks "GliderPRO/Houses/Demo House.binhex"
    python3 tools/probe_house.py --resmap "GliderPRO/Houses/Demo House.binhex"
    python3 tools/probe_house.py --raw houseDataFork.bin           (already-decoded data fork)

The house level data lives in the *data fork* (a flat dump of the in-memory
`houseType` struct, big-endian, no padding).  The *resource fork* holds the
custom PICT/snd /bnds art, plus 'Date' pseudo-PICTs.
"""

import argparse
import struct
import sys

# ---------------------------------------------------------------- BinHex 4.0

BINHEX_ALPHABET = (
    "!\"#$%&'()*+,-012345689@ABCDEFGHIJKLMN"
    "PQRSTUVXYZ[`abcdefhijklmpqr"
)
assert len(BINHEX_ALPHABET) == 64, len(BINHEX_ALPHABET)
_DEC = {c: i for i, c in enumerate(BINHEX_ALPHABET)}

MAGIC_LINE = "(This file must be converted with BinHex 4.0)"


def _sixbit_decode(text):
    """Turn the 6-bit-per-char BinHex body into raw (still RLE'd) bytes."""
    acc = 0
    nbits = 0
    out = bytearray()
    for ch in text:
        v = _DEC.get(ch)
        if v is None:
            continue  # newlines, spaces, stray CR
        acc = ((acc << 6) | v) & 0xFFFFFF
        nbits += 6
        if nbits >= 8:
            nbits -= 8
            out.append((acc >> nbits) & 0xFF)
    return bytes(out)


def _rle_expand(data):
    """BinHex RLE: 0x90 <n> repeats the previous byte n times; n==0 -> literal 0x90."""
    out = bytearray()
    i = 0
    n = len(data)
    while i < n:
        b = data[i]
        i += 1
        if b != 0x90:
            out.append(b)
            continue
        if i >= n:
            out.append(0x90)
            break
        count = data[i]
        i += 1
        if count == 0:
            out.append(0x90)
        else:
            if not out:
                raise ValueError("RLE run with no preceding byte")
            prev = out[-1]
            out.extend(bytes([prev]) * (count - 1))
    return bytes(out)


def binhex_crc(data, crc=0):
    """Classic BinHex 4.0 CRC-16.

    Poly 0x1021, seed 0, MSB-first, and the *data* bits are shifted INTO the
    register (not XOR'd against the top byte).  Verified against all 22 shipped
    houses: append two 0x00 bytes to the fork and the result equals the stored
    CRC exactly.
    """
    for b in data:
        for i in range(8):
            hibit = crc & 0x8000
            crc = ((crc << 1) | ((b >> (7 - i)) & 1)) & 0xFFFF
            if hibit:
                crc ^= 0x1021
    return crc


def binhex_decode(path, check_crc=False):
    """Return (filename, type, creator, flags, datafork, resfork)."""
    with open(path, "rb") as fh:
        raw = fh.read().decode("mac-roman", errors="replace")
    idx = raw.find(MAGIC_LINE)
    if idx < 0:
        raise ValueError("not a BinHex 4.0 file (magic line missing)")
    body = raw[idx + len(MAGIC_LINE):]
    start = body.find(":")
    if start < 0:
        raise ValueError("no ':' start-of-data marker")
    end = body.find(":", start + 1)
    if end < 0:
        end = len(body)
    payload = _rle_expand(_sixbit_decode(body[start + 1:end]))

    p = 0
    namelen = payload[p]
    p += 1
    name = payload[p:p + namelen].decode("mac-roman")
    p += namelen
    version = payload[p]
    p += 1
    ftype = payload[p:p + 4]
    p += 4
    creator = payload[p:p + 4]
    p += 4
    flags = struct.unpack_from(">H", payload, p)[0]
    p += 2
    dlen, rlen = struct.unpack_from(">II", payload, p)
    p += 8
    hdr_crc = struct.unpack_from(">H", payload, p)[0]
    p += 2
    data = payload[p:p + dlen]
    p += dlen
    data_crc = struct.unpack_from(">H", payload, p)[0]
    p += 2
    res = payload[p:p + rlen]
    p += rlen
    res_crc = struct.unpack_from(">H", payload, p)[0] if p + 2 <= len(payload) else None

    if check_crc:
        got = binhex_crc(data + b"\0\0")
        if got != data_crc:
            raise ValueError(f"data fork CRC {got:#06x} != stored {data_crc:#06x}")
        got = binhex_crc(res + b"\0\0")
        if res_crc is not None and got != res_crc:
            raise ValueError(f"rsrc fork CRC {got:#06x} != stored {res_crc:#06x}")

    return {
        "name": name,
        "version": version,
        "type": ftype,
        "creator": creator,
        "flags": flags,
        "data": data,
        "rsrc": res,
        "hdr_crc": hdr_crc,
        "data_crc": data_crc,
        "rsrc_crc": res_crc,
    }


# ---------------------------------------------------- classic resource fork

def parse_resource_fork(res):
    """Minimal Resource Manager map walker -> {type: [(id, name, offset, len)]}."""
    if len(res) < 16:
        return {}
    data_off, map_off, data_len, map_len = struct.unpack_from(">IIII", res, 0)
    tl_off = struct.unpack_from(">H", res, map_off + 24)[0]
    # name list offset is at map_off+26 (unused here beyond names)
    name_off = struct.unpack_from(">H", res, map_off + 26)[0]
    tl = map_off + tl_off
    ntypes = (struct.unpack_from(">h", res, tl)[0] + 1) & 0xFFFF
    out = {}
    for i in range(ntypes):
        base = tl + 2 + i * 8
        rtype = res[base:base + 4].decode("mac-roman")
        count = struct.unpack_from(">H", res, base + 4)[0] + 1
        ref_off = struct.unpack_from(">H", res, base + 6)[0]
        entries = []
        for j in range(count):
            rb = tl + ref_off + j * 12
            rid = struct.unpack_from(">h", res, rb)[0]
            nameoff = struct.unpack_from(">h", res, rb + 2)[0]
            attr_and_off = struct.unpack_from(">I", res, rb + 4)[0]
            doff = attr_and_off & 0x00FFFFFF
            rname = ""
            if nameoff != -1:
                nb = map_off + name_off + nameoff
                rname = res[nb + 1:nb + 1 + res[nb]].decode("mac-roman")
            rlen = struct.unpack_from(">I", res, data_off + doff)[0]
            entries.append((rid, rname, data_off + doff + 4, rlen))
        out[rtype] = entries
    return out


# ------------------------------------------------------------ house parsing

kMaxScores = 10
kMaxRoomObs = 24
kNumTiles = 8
SIZEOF_HOUSE_HEADER = 866
SIZEOF_ROOM = 348
SIZEOF_OBJECT = 12
kRoomIsEmpty = -1
kObjectIsEmpty = -1
kNumUndergroundFloors = 8


def pstr(buf, off, cap):
    """Pascal string occupying `cap` bytes total (1 length byte + cap-1 chars)."""
    n = buf[off]
    n = min(n, cap - 1)
    return buf[off + 1:off + 1 + n].decode("mac-roman"), buf[off]


OBJ_CLASS = [
    # (lo, hi, class letter, name)
    (0x01, 0x10, "a", "blower"),
    (0x11, 0x1F, "b", "furniture"),
    (0x21, 0x2F, "c", "bonus"),
    (0x31, 0x40, "d", "transport"),
    (0x41, 0x49, "e", "switch"),
    (0x51, 0x58, "f", "light"),
    (0x61, 0x6E, "g", "appliance"),
    (0x71, 0x79, "h", "enemy"),
    (0x81, 0x8F, "i", "clutter"),
]

OBJ_NAMES = {
    0x01: "kFloorVent", 0x02: "kCeilingVent", 0x03: "kFloorBlower",
    0x04: "kCeilingBlower", 0x05: "kSewerGrate", 0x06: "kLeftFan",
    0x07: "kRightFan", 0x08: "kTaper", 0x09: "kCandle", 0x0A: "kStubby",
    0x0B: "kTiki", 0x0C: "kBBQ", 0x0D: "kInvisBlower", 0x0E: "kGrecoVent",
    0x0F: "kSewerBlower", 0x10: "kLiftArea",
    0x11: "kTable", 0x12: "kShelf", 0x13: "kCabinet", 0x14: "kFilingCabinet",
    0x15: "kWasteBasket", 0x16: "kMilkCrate", 0x17: "kCounter",
    0x18: "kDresser", 0x19: "kDeckTable", 0x1A: "kStool", 0x1B: "kTrunk",
    0x1C: "kInvisObstacle", 0x1D: "kManhole", 0x1E: "kBooks",
    0x1F: "kInvisBounce",
    0x21: "kRedClock", 0x22: "kBlueClock", 0x23: "kYellowClock",
    0x24: "kCuckoo", 0x25: "kPaper", 0x26: "kBattery", 0x27: "kBands",
    0x28: "kGreaseRt", 0x29: "kGreaseLf", 0x2A: "kFoil", 0x2B: "kInvisBonus",
    0x2C: "kStar", 0x2D: "kSparkle", 0x2E: "kHelium", 0x2F: "kSlider",
    0x31: "kUpStairs", 0x32: "kDownStairs", 0x33: "kMailboxLf",
    0x34: "kMailboxRt", 0x35: "kFloorTrans", 0x36: "kCeilingTrans",
    0x37: "kDoorInLf", 0x38: "kDoorInRt", 0x39: "kDoorExRt", 0x3A: "kDoorExLf",
    0x3B: "kWindowInLf", 0x3C: "kWindowInRt", 0x3D: "kWindowExRt",
    0x3E: "kWindowExLf", 0x3F: "kInvisTrans", 0x40: "kDeluxeTrans",
    0x41: "kLightSwitch", 0x42: "kMachineSwitch", 0x43: "kThermostat",
    0x44: "kPowerSwitch", 0x45: "kKnifeSwitch", 0x46: "kInvisSwitch",
    0x47: "kTrigger", 0x48: "kLgTrigger", 0x49: "kSoundTrigger",
    0x51: "kCeilingLight", 0x52: "kLightBulb", 0x53: "kTableLamp",
    0x54: "kHipLamp", 0x55: "kDecoLamp", 0x56: "kFlourescent",
    0x57: "kTrackLight", 0x58: "kInvisLight",
    0x61: "kShredder", 0x62: "kToaster", 0x63: "kMacPlus", 0x64: "kGuitar",
    0x65: "kTV", 0x66: "kCoffee", 0x67: "kOutlet", 0x68: "kVCR",
    0x69: "kStereo", 0x6A: "kMicrowave", 0x6B: "kCinderBlock",
    0x6C: "kFlowerBox", 0x6D: "kCDs", 0x6E: "kCustomPict",
    0x71: "kBalloon", 0x72: "kCopterLf", 0x73: "kCopterRt", 0x74: "kDartLf",
    0x75: "kDartRt", 0x76: "kBall", 0x77: "kDrip", 0x78: "kFish",
    0x79: "kCobweb",
    0x81: "kOzma", 0x82: "kMirror", 0x83: "kMousehole", 0x84: "kFireplace",
    0x85: "kFlower", 0x86: "kWallWindow", 0x87: "kBear", 0x88: "kCalendar",
    0x89: "kVase1", 0x8A: "kVase2", 0x8B: "kBulletin", 0x8C: "kCloud",
    0x8D: "kFaucet", 0x8E: "kRug", 0x8F: "kChimes",
}


def obj_class(what):
    for lo, hi, letter, cname in OBJ_CLASS:
        if lo <= what <= hi:
            return letter, cname
    return "?", "unknown"


def parse_object(buf, off):
    what = struct.unpack_from(">h", buf, off)[0]
    raw = buf[off:off + SIZEOF_OBJECT]
    o = {"what": what, "name": OBJ_NAMES.get(what, "?"), "raw": raw}
    if what == kObjectIsEmpty:
        o["class"] = ("-", "empty")
        return o
    letter, cname = obj_class(what)
    o["class"] = (letter, cname)
    d = off + 2
    if letter in ("a",):  # blowerType
        v, h, dist, initial, state, vector, tall = struct.unpack_from(">hhhBBBB", buf, d)
        o.update(topLeft=(h, v), distance=dist, initial=initial,
                 state=state, vector=vector, tall=tall)
    elif letter in ("b", "i"):  # furnitureType / clutterType (Rect + pict)
        top, left, bottom, right, pict = struct.unpack_from(">hhhhh", buf, d)
        o.update(bounds=(left, top, right, bottom), pict=pict)
    elif letter == "c":  # bonusType
        v, h, length, points, state, initial = struct.unpack_from(">hhhhBB", buf, d)
        o.update(topLeft=(h, v), length=length, points=points,
                 state=state, initial=initial)
    elif letter == "d":  # transportType
        v, h, tall, where, who, wide = struct.unpack_from(">hhhhBB", buf, d)
        o.update(topLeft=(h, v), tall=tall, where=where, who=who, wide=wide)
    elif letter == "e":  # switchType
        v, h, delay, where, who, typ = struct.unpack_from(">hhhhBB", buf, d)
        o.update(topLeft=(h, v), delay=delay, where=where, who=who, type=typ)
    elif letter == "f":  # lightType
        v, h, length, b0, b1, initial, state = struct.unpack_from(">hhhBBBB", buf, d)
        o.update(topLeft=(h, v), length=length, byte0=b0, byte1=b1,
                 initial=initial, state=state)
    elif letter == "g":  # applianceType
        v, h, height, b0, delay, initial, state = struct.unpack_from(">hhhBBBB", buf, d)
        o.update(topLeft=(h, v), height=height, byte0=b0, delay=delay,
                 initial=initial, state=state)
    elif letter == "h":  # enemyType
        v, h, length, delay, b0, initial, state = struct.unpack_from(">hhhBBBB", buf, d)
        o.update(topLeft=(h, v), length=length, delay=delay, byte0=b0,
                 initial=initial, state=state)
    return o


def extract_floor_suite(combo, version):
    if combo == -1:
        return None
    if version < 0x0200:
        return (combo // 100) - kNumUndergroundFloors, combo % 100
    return (combo % 100) - kNumUndergroundFloors, combo // 100


def parse_room(buf, off, version):
    name, namelen = pstr(buf, off + 0, 28)
    bounds, = struct.unpack_from(">h", buf, off + 28)
    leftStart = buf[off + 30]
    rightStart = buf[off + 31]
    unusedByte = buf[off + 32]
    visited = buf[off + 33]
    background, = struct.unpack_from(">h", buf, off + 34)
    tiles = list(struct.unpack_from(">8h", buf, off + 36))
    floor, suite, openings, numObjects = struct.unpack_from(">4h", buf, off + 52)
    objects = [parse_object(buf, off + 60 + i * SIZEOF_OBJECT)
               for i in range(kMaxRoomObs)]
    return {
        "name": name, "nameLenByte": namelen, "bounds": bounds,
        "leftStart": leftStart, "rightStart": rightStart,
        "unusedByte": unusedByte, "visited": visited,
        "background": background, "tiles": tiles,
        "floor": floor, "suite": suite, "openings": openings,
        "numObjects": numObjects, "objects": objects,
        "isEmpty": suite == kRoomIsEmpty,
    }


def parse_house(buf):
    version, unusedShort = struct.unpack_from(">hh", buf, 0)
    timeStamp, flags = struct.unpack_from(">ii", buf, 4)
    iv, ih = struct.unpack_from(">hh", buf, 12)   # Point == {v, h}
    banner, bannerLen = pstr(buf, 16, 256)
    trailer, trailerLen = pstr(buf, 272, 256)

    sb, _ = pstr(buf, 528, 32)
    names = [pstr(buf, 528 + 32 + i * 16, 16)[0] for i in range(kMaxScores)]
    scores = list(struct.unpack_from(">10i", buf, 528 + 192))
    stamps = list(struct.unpack_from(">10I", buf, 528 + 232))
    levels = list(struct.unpack_from(">10h", buf, 528 + 272))

    g = {}
    gb = 820
    (g["version"], g["wasStarsLeft"]) = struct.unpack_from(">hh", buf, gb)
    (g["timeStamp"],) = struct.unpack_from(">i", buf, gb + 4)
    (g["where_v"], g["where_h"]) = struct.unpack_from(">hh", buf, gb + 8)
    (g["score"], g["unusedLong"], g["unusedLong2"]) = struct.unpack_from(">3i", buf, gb + 12)
    (g["energy"], g["bands"], g["roomNumber"], g["gliderState"],
     g["numGliders"], g["foil"], g["unusedShort"]) = struct.unpack_from(">7h", buf, gb + 24)
    g["facing"] = buf[gb + 38]
    g["showFoil"] = buf[gb + 39]

    hasGame = buf[860]
    unusedBoolean = buf[861]
    firstRoom, nRooms = struct.unpack_from(">hh", buf, 862)

    rooms = []
    for i in range(nRooms):
        off = SIZEOF_HOUSE_HEADER + i * SIZEOF_ROOM
        if off + SIZEOF_ROOM > len(buf):
            break
        rooms.append(parse_room(buf, off, version))

    return {
        "version": version, "unusedShort": unusedShort,
        "timeStamp": timeStamp, "flags": flags,
        "initial": (ih, iv), "banner": banner, "bannerLen": bannerLen,
        "trailer": trailer, "trailerLen": trailerLen,
        "scoreBanner": sb, "scoreNames": names, "scores": scores,
        "scoreStamps": stamps, "scoreLevels": levels,
        "savedGame": g, "hasGame": hasGame, "unusedBoolean": unusedBoolean,
        "firstRoom": firstRoom, "nRooms": nRooms, "rooms": rooms,
        "fileSize": len(buf),
        "expectedSize": SIZEOF_HOUSE_HEADER + nRooms * SIZEOF_ROOM,
        "wardBit": (flags & 1) == 1,
        "phoneBit": (flags & 2) == 2,
        "bannerStarCountOn": (flags & 4) == 0,
        "houseUnlocked": (timeStamp & 1) == 0,
    }


# ------------------------------------------------------------------ report

def hexdump(b, base=0, width=16, limit=None):
    out = []
    n = len(b) if limit is None else min(limit, len(b))
    for i in range(0, n, width):
        chunk = b[i:i + width]
        hexs = " ".join(f"{c:02X}" for c in chunk)
        txt = "".join(chr(c) if 32 <= c < 127 else "." for c in chunk)
        out.append(f"{base + i:08X}  {hexs:<{width * 3}} |{txt}|")
    return "\n".join(out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("path")
    ap.add_argument("--rooms", type=int, default=3)
    ap.add_argument("--room", type=int, action="append", default=[])
    ap.add_argument("--raw", action="store_true",
                    help="path is an already-decoded data fork")
    ap.add_argument("--forks", action="store_true", help="just show fork sizes")
    ap.add_argument("--resmap", action="store_true", help="list resource fork")
    ap.add_argument("--dump-data", metavar="OUT", help="write data fork to file")
    ap.add_argument("--dump-rsrc", metavar="OUT", help="write rsrc fork to file")
    ap.add_argument("--hexhdr", type=int, default=0, help="hexdump N header bytes")
    ap.add_argument("--allrooms", action="store_true")
    args = ap.parse_args()

    if args.raw:
        with open(args.path, "rb") as fh:
            data = fh.read()
        info = {"name": args.path, "type": b"????", "creator": b"????",
                "data": data, "rsrc": b"", "flags": 0}
    else:
        info = binhex_decode(args.path)

    print(f"file            : {args.path}")
    print(f"binhex name     : {info['name']!r}")
    print(f"type/creator    : {info['type'].decode('mac-roman')!r} /"
          f" {info['creator'].decode('mac-roman')!r}")
    print(f"finder flags    : 0x{info['flags']:04X}")
    print(f"data fork len   : {len(info['data'])}")
    print(f"rsrc fork len   : {len(info['rsrc'])}")

    if args.dump_data:
        open(args.dump_data, "wb").write(info["data"])
        print(f"wrote {args.dump_data}")
    if args.dump_rsrc:
        open(args.dump_rsrc, "wb").write(info["rsrc"])
        print(f"wrote {args.dump_rsrc}")

    if args.resmap:
        try:
            m = parse_resource_fork(info["rsrc"])
        except Exception as e:  # noqa
            print("resource fork parse failed:", e)
            return
        for t in sorted(m):
            ids = sorted(e[0] for e in m[t])
            print(f"  '{t}'  n={len(m[t]):4d}  ids={ids[:14]}"
                  f"{' ...' if len(ids) > 14 else ''}")
        return

    if args.forks:
        return

    h = parse_house(info["data"])
    if args.hexhdr:
        print(hexdump(info["data"][:args.hexhdr]))

    print("--- houseType header ---")
    print(f"  version        = 0x{h['version'] & 0xFFFF:04X} ({h['version']})")
    print(f"  unusedShort    = {h['unusedShort']}")
    print(f"  timeStamp      = {h['timeStamp']} (0x{h['timeStamp'] & 0xFFFFFFFF:08X})"
          f"  houseUnlocked={h['houseUnlocked']}")
    print(f"  flags          = 0x{h['flags'] & 0xFFFFFFFF:08X}  ward={h['wardBit']}"
          f" phone={h['phoneBit']} bannerStarCount={h['bannerStarCountOn']}")
    print(f"  initial (h,v)  = {h['initial']}")
    print(f"  banner[{h['bannerLen']:3d}]   = {h['banner']!r}")
    print(f"  trailer[{h['trailerLen']:3d}]  = {h['trailer']!r}")
    print(f"  hs.banner      = {h['scoreBanner']!r}")
    for i in range(kMaxScores):
        print(f"    hs[{i}] {h['scoreNames'][i]!r:20s} score={h['scores'][i]:>9d}"
              f" stamp={h['scoreStamps'][i]:>10d} level={h['scoreLevels'][i]}")
    print(f"  savedGame      = {h['savedGame']}")
    print(f"  hasGame        = {h['hasGame']}   unusedBoolean = {h['unusedBoolean']}")
    print(f"  firstRoom      = {h['firstRoom']}")
    print(f"  nRooms         = {h['nRooms']}")
    print(f"  file size      = {h['fileSize']}   expected 866+348*nRooms ="
          f" {h['expectedSize']}   match={h['fileSize'] == h['expectedSize']}")

    empties = sum(1 for r in h["rooms"] if r["isEmpty"])
    print(f"  real rooms     = {h['nRooms'] - empties} ({empties} placeholder)")

    want = args.room if args.room else list(range(min(args.rooms, h["nRooms"])))
    if args.allrooms:
        want = list(range(h["nRooms"]))
    for ri in want:
        r = h["rooms"][ri]
        print(f"--- room[{ri}] @0x{SIZEOF_HOUSE_HEADER + ri * SIZEOF_ROOM:X} ---")
        print(f"  name[{r['nameLenByte']}]     = {r['name']!r}")
        print(f"  bounds       = {r['bounds']} (0x{r['bounds'] & 0xFFFF:04X})")
        print(f"  leftStart    = {r['leftStart']}   rightStart = {r['rightStart']}"
              f"   unusedByte = {r['unusedByte']}   visited = {r['visited']}")
        print(f"  background   = {r['background']}")
        print(f"  tiles        = {r['tiles']}")
        print(f"  floor        = {r['floor']}   suite = {r['suite']}"
              f"   openings = {r['openings']}   numObjects = {r['numObjects']}")
        live = 0
        for oi, o in enumerate(r["objects"]):
            if o["what"] == kObjectIsEmpty:
                continue
            live += 1
            extra = {k: v for k, v in o.items()
                     if k not in ("what", "name", "raw", "class")}
            print(f"    obj[{oi:2d}] what=0x{o['what'] & 0xFFFF:02X}"
                  f" {o['name']:16s} cls={o['class'][1]:10s} {extra}")
            w = o.get("where")
            if w is not None and w != -1:
                fs = extract_floor_suite(w, h["version"])
                print(f"           link where={w} -> floor={fs[0]} suite={fs[1]}"
                      f" who={o.get('who')}")
        print(f"  live objects = {live}  (numObjects field says {r['numObjects']})")


if __name__ == "__main__":
    sys.exit(main())
