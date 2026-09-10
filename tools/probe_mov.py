#!/usr/bin/env python3
"""probe_mov.py -- from-scratch QuickTime .mov reader for Glider PRO's TV movies.

Companion to docs/analysis/quicktime-movies.md.  Everything that document claims
about the shipped movie files was produced by this script; if the two disagree,
re-run the script -- it reads the bytes, the document does not.

No third-party dependencies: standard library only (struct + zlib, the latter
via probe_pict's PNG writer).  Deliberately a *probe*, not a general-purpose
QuickTime library: it implements exactly the atoms and exactly the three codecs
that GliderPRO/Houses/*.mov actually use.

Input files
-----------
    GliderPRO/Houses/*.mov      15 files, the TV-set movies OpenHouseMovie()
                                opens (GliderPRO/Sources/HouseIO.c:67).

14 of the 15 are *data forks only*: their 'moov' (movie header) atom lived in
the file's resource fork, which the GPL source release did not preserve.  Those
files consist of a single 'mdat' atom whose declared size field is 0 and whose
payload runs to EOF.  This script reconstructs their sample tables by walking
the compressed video data, which is self-delimiting for all three codecs in
use.  Only Demo House.mov still has a 'moov'.

On-disk layout (EVERYTHING here is big-endian; that is load-bearing)
-------------------------------------------------------------------

  Atom (a.k.a. QT Atom / ISO box)
    +0   uint32  size      total bytes including this header.
                           0 => "extends to end of file" (legal only for the
                                last atom; used by all 14 headerless movies)
                           1 => 64-bit size follows at +8
    +4   char[4] type      e.g. 'mdat', 'moov'
    +8   ...     payload   (container atoms hold child atoms here)

  Atom nesting actually present in Demo House.mov
    moov
      mvhd
      trak
        tkhd
        edts / elst
        mdia
          mdhd
          hdlr           (component type 'mhlr', subtype 'vide')
          minf
            vmhd
            hdlr         (component type 'dhlr', subtype 'alis')
            dinf / dref  (containing one 'alis' data reference)
            stbl
              stsd stts stsc stsz stco
    mdat                 (precedes moov in the file)

  'mvhd' MovieHeader, 100 bytes of payload
    +0   uint8   version                 (0)
    +1   uint24  flags                   (0)
    +4   uint32  creationTime            seconds since 1904-01-01 00:00 local
    +8   uint32  modificationTime
    +12  uint32  timeScale               time units per second
    +16  uint32  duration                in timeScale units
    +20  Fixed   preferredRate           16.16
    +24  ufix16  preferredVolume         8.8   (0x0100 = full)
    +26  byte[10] reserved
    +36  Fixed[9] matrix                 a b u / c d v / x y w
    +72  uint32  previewTime
    +76  uint32  previewDuration
    +80  uint32  posterTime
    +84  uint32  selectionTime
    +88  uint32  selectionDuration
    +92  uint32  currentTime
    +96  uint32  nextTrackID

  'tkhd' TrackHeader, 84 bytes of payload
    +0   uint8   version
    +1   uint24  flags       0x1 enabled, 0x2 inMovie, 0x4 inPreview,
                             0x8 inPoster
    +4   uint32  creationTime
    +8   uint32  modificationTime
    +12  uint32  trackID
    +16  byte[4] reserved
    +20  uint32  duration           in the MOVIE's timeScale
    +24  byte[8] reserved
    +32  uint16  layer
    +34  uint16  alternateGroup
    +36  ufix16  volume
    +38  uint16  reserved
    +40  Fixed[9] matrix
    +76  Fixed   trackWidth          <- GetMovieBox() width comes from here
    +80  Fixed   trackHeight

  'mdhd' MediaHeader, 24 bytes
    +0 version/flags, +4 creation, +8 modification,
    +12 uint32 timeScale, +16 uint32 duration,
    +20 uint16 language (0 = English), +22 uint16 quality

  'stsd' SampleDescription table
    +0 version/flags, +4 uint32 numEntries, then numEntries x ImageDescription

  ImageDescription (Quickdraw component; 86 bytes as written here)
    +0   uint32  idSize            86
    +4   char[4] cType             'rle ' / 'smc ' / 'raw '
    +8   uint32  resvd1            0
    +12  uint16  resvd2            0
    +14  uint16  dataRefIndex      1
    +16  uint16  version
    +18  uint16  revisionLevel
    +20  char[4] vendor            'appl'
    +24  uint32  temporalQuality   0..1023 (codecLosslessQuality = 0x400)
    +28  uint32  spatialQuality
    +32  uint16  width             pixels
    +34  uint16  height
    +36  Fixed   hRes              0x00480000 = 72 dpi
    +40  Fixed   vRes
    +44  uint32  dataSize          0 = "look at the sample table"
    +48  uint16  frameCount        frames per sample (1)
    +50  Str31   name              1 length byte + 31 bytes
    +82  uint16  depth             1,2,4,8,16,24,32 = colour;
                                   33,34,36,40 = 1/2/4/8-bit GRAYSCALE
    +84  int16   clutID            -1 = clut follows inline,
                                    0 = use the default table for this depth

  'stts' TimeToSample:  version/flags, uint32 count, count x {u32 n, u32 dur}
  'stsc' SampleToChunk: version/flags, uint32 count,
                        count x {u32 firstChunk, u32 samplesPerChunk, u32 descID}
  'stsz' SampleSize:    version/flags, uint32 sampleSize (0 = table follows),
                        uint32 count, count x u32
  'stco' ChunkOffset:   version/flags, uint32 count, count x u32 (file offsets)
  'stss' SyncSample:    absent in Demo House.mov => every frame is a key frame
  'elst' EditList:      version/flags, u32 count,
                        count x {u32 duration, s32 mediaTime, Fixed mediaRate}

Codecs
------
Apple Animation, 'rle ' (per-line RLE over 4-byte "pixel groups")
    +0   uint32  chunkSize   low 24 bits = sample size; high byte = flags
                             (0x40 observed on full-frame samples)
    +4   uint16  header      bit 3 (0x0008) set => an explicit line range
                             follows; clear => start at line 0, all lines
    if header & 8:
      +6 uint16 startLine, +8 uint16 (0), +10 uint16 numLines, +12 uint16 (0)
    then numLines x:
      uint8   skip           advance (skip-1) pixel groups from x=0
      then repeatedly int8 rleCode:
         -1  end of this line
          0  another skip byte follows: advance (skip-1) groups
         >0  literal: rleCode groups follow verbatim
         <0  run: one group follows, repeat it (-rleCode) times
    A "pixel group" is always 4 BYTES.  At depth 8 that is 4 pixels; at depth
    4/36 it is 8 pixels (two per byte, high nibble first).
    A 7- or 8-byte sample with header 0x0000 and no line data is a no-op frame
    ("nothing changed"); Rainbow's End has 3 and Teddy World has 2.

Apple Graphics, 'smc ' (4x4 blocks, 8-bit indexed only)
    +0   uint32  chunkSize (low 24 bits), then a stream of opcodes.
    Blocks are emitted left to right in 4-pixel columns, then down 4 rows;
    total blocks = ceil(w/4) * ceil(h/4).
    opcode & 0xF0:
      0x00/0x10  skip N blocks
      0x20/0x30  repeat the previous block N times
      0x40/0x50  repeat the previous PAIR of blocks N times (2N blocks)
      0x60/0x70  N one-colour blocks; 1 colour byte follows
      0x80       2-colour: 2 new colours -> pair cache; then N x uint16 of
                 1-bit-per-pixel flags (MSB first)
      0x90       2-colour reusing cache entry: 1 index byte, then N x uint16
      0xA0       4-colour: 4 new colours -> quad cache; then N x uint32,
                 2 bits per pixel, from bit 30 downwards
      0xB0       4-colour reusing cache: 1 index byte, then N x uint32
      0xC0       8-colour: 8 new colours -> octet cache; then N x 6 bytes,
                 3 bits per pixel; rows 0-1 use the first 3 bytes, rows 2-3
                 the last 3, each starting at bit 21
      0xD0       8-colour reusing cache: 1 index byte, then N x 6 bytes
      0xE0       N x 16 raw colour bytes (one block each)
      0xF0       undefined
    N = (opcode & 0x0F) + 1, except for opcodes 0x00..0x70 where bit 4 of the
    opcode means "N-1 is in the next byte instead".
    Cache indices are multiplied by 2/4/8 respectively.

Uncompressed, 'raw ' at depth 8
    width*height bytes of palette indices, top row first, no row padding and
    no per-sample header at all.

Palettes
--------
depth 8  -> the standard Macintosh 256-entry table, which this tree ships as
            'clut' 128 in GliderPRO/Glider PRO.r (see docs/analysis/
            graphics-assets.md).  Read via probe_rez + probe_pict.parse_clut.
depth 36 -> the standard 4-bit grayscale table: index i => gray (15-i)*17,
            so 0 = white and 15 = black.

Usage
-----
    probe_mov.py survey                 one line per Houses/*.mov (the headline)
    probe_mov.py atoms  [FILE...]       atom tree of each file (default: all)
    probe_mov.py moov   [FILE]          full field dump of every moov atom
    probe_mov.py samples FILE|all       per-sample table
    probe_mov.py rle    FILE [N]        opcode-level trace of sample N
    probe_mov.py opcodes [FILE...]      opcode-usage totals per movie
    probe_mov.py decode FILE OUTDIR     frame_000.png ... for every frame
    probe_mov.py sheet  FILE OUT.png    all frames as one contact sheet
    probe_mov.py extract OUTDIR         raw 8-bit index frames + manifest.tsv
    probe_mov.py all                    survey + samples for all 15 files

FILE may be a path, a bare house name ("Titanic"), a file name
("Titanic.mov"), or the literal word "all" for every Houses/*.mov.
"""

import os
import struct
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
REZ = os.path.join(ROOT, "GliderPRO", "Glider PRO.r")
HOUSES = os.path.join(ROOT, "GliderPRO", "Houses")

sys.path.insert(0, HERE)
import probe_pict                                            # noqa: E402
import probe_rez                                             # noqa: E402

# ---------------------------------------------------------------- constants
# Seconds between the Mac epoch (1904-01-01) and the Unix epoch (1970-01-01).
MAC_EPOCH_DELTA = 2082844800

# Atoms whose payload is a list of child atoms.
CONTAINERS = frozenset([b"moov", b"trak", b"mdia", b"minf", b"stbl", b"dinf",
                        b"edts", b"udta", b"clip", b"matt", b"tref", b"gmhd"])

# StructuresInit.c:565 -- tvScreen1, the rect a TV movie is displayed in.
TV_SCREEN_W = 64
TV_SCREEN_H = 49

# ObjectDrawAll.c:684 / ObjectDraw2.c:681 -- the movie box is offset this far
# inside the TV object's own rect before being centred.
TV_SCREEN_DX = 17
TV_SCREEN_DY = 10

# tkhd flags (Movies.h).
TKHD_FLAGS = [(0x0001, "enabled"), (0x0002, "inMovie"),
              (0x0004, "inPreview"), (0x0008, "inPoster")]

# ImageDescription.depth values that mean grayscale rather than indexed colour.
GRAY_DEPTHS = {33: 1, 34: 2, 36: 4, 40: 8}

# 'rle ' sample header bit 3: an explicit (startLine, numLines) pair follows.
RLE_HDR_LINE_RANGE = 0x0008

# High byte of a 'rle '/'smc ' chunkSize long.  0x40 is observed on samples
# that repaint every line; 0x00 on partial updates.
RLE_FLAG_FULL = 0x40

# Trailing byte on every 'rle ' sample in this tree: an end-of-stream marker.
RLE_END = 0x00

# QuickDraw transfer modes (Quickdraw.h) that can appear in vmhd.graphicsMode.
GRAPHICS_MODES = {0: "srcCopy", 1: "srcOr", 2: "srcXor", 3: "srcBic",
                  4: "notSrcCopy", 32: "blend", 36: "transparent",
                  64: "ditherCopy"}

CODECS = {b"rle ": "Apple Animation", b"smc ": "Apple Graphics",
          b"raw ": "Uncompressed", b"rpza": "Apple Video (Road Pizza)",
          b"cvid": "Cinepak", b"jpeg": "Photo-JPEG", b"SVQ1": "Sorenson"}


def fixed(v):
    """Mac Fixed (16.16, signed) -> float."""
    if v & 0x80000000:
        v -= 1 << 32
    return v / 65536.0


def ufix8(v):
    """Mac 8.8 unsigned -> float."""
    return v / 256.0


def mac_time(v):
    """Mac seconds-since-1904 -> 'YYYY-MM-DD HH:MM:SS' (as a naive local time,
    which is how the Toolbox stored it), or '(0)' if unset."""
    import datetime
    if v == 0:
        return "(0)"
    return (datetime.datetime(1970, 1, 1)
            + datetime.timedelta(seconds=v - MAC_EPOCH_DELTA)
            ).strftime("%Y-%m-%d %H:%M:%S")


def fourcc(b):
    return b.decode("mac-roman").replace("\xa9", "(c)")


# ------------------------------------------------------------- atom walking
class Atom(object):
    __slots__ = ("off", "size", "type", "hdr", "depth", "parent")

    def __init__(self, off, size, typ, hdr, depth, parent):
        self.off, self.size, self.type = off, size, typ
        self.hdr, self.depth, self.parent = hdr, depth, parent

    @property
    def body(self):
        return self.off + self.hdr

    @property
    def end(self):
        return self.off + self.size

    def path(self):
        names = []
        a = self
        while a is not None:
            names.append(fourcc(a.type))
            a = a.parent
        return "/".join(reversed(names))

    def __repr__(self):
        return "<%s @%d +%d>" % (fourcc(self.type), self.off, self.size)


def walk(data, start=0, end=None, depth=0, parent=None):
    """Yield every Atom in [start, end), recursing into CONTAINERS.

    Handles the two special size encodings: 0 = to end of file (used by the 14
    headerless movies) and 1 = 64-bit size at +8 (not used by this game)."""
    if end is None:
        end = len(data)
    p = start
    while p + 8 <= end:
        size, typ = struct.unpack_from(">I4s", data, p)
        hdr = 8
        if size == 1:
            size = struct.unpack_from(">Q", data, p + 8)[0]
            hdr = 16
        elif size == 0:
            size = end - p                       # "rest of file"
        if size < hdr or p + size > end:
            # Truncated / bogus: report it and stop, do not desynchronise.
            yield Atom(p, end - p, typ, hdr, depth, parent)
            return
        a = Atom(p, size, typ, hdr, depth, parent)
        yield a
        if typ in CONTAINERS:
            for c in walk(data, p + hdr, p + size, depth + 1, a):
                yield c
        p += size


def raw_size_field(data, off):
    """The size long exactly as stored, without the size==0 fix-up."""
    return struct.unpack_from(">I", data, off)[0]


def find(data, typ, start=0, end=None):
    for a in walk(data, start, end):
        if a.type == typ:
            return a
    return None


# --------------------------------------------------------- moov field dumps
def parse_mvhd(d, o):
    f = {}
    f["version"] = d[o]
    f["flags"] = struct.unpack_from(">I", d, o)[0] & 0xFFFFFF
    (f["creationTime"], f["modificationTime"], f["timeScale"], f["duration"],
     f["preferredRate"]) = struct.unpack_from(">5I", d, o + 4)
    f["preferredVolume"] = struct.unpack_from(">H", d, o + 24)[0]
    f["matrix"] = struct.unpack_from(">9I", d, o + 36)
    (f["previewTime"], f["previewDuration"], f["posterTime"],
     f["selectionTime"], f["selectionDuration"], f["currentTime"],
     f["nextTrackID"]) = struct.unpack_from(">7I", d, o + 72)
    return f


def parse_tkhd(d, o):
    f = {}
    f["version"] = d[o]
    f["flags"] = struct.unpack_from(">I", d, o)[0] & 0xFFFFFF
    (f["creationTime"], f["modificationTime"], f["trackID"]) = \
        struct.unpack_from(">3I", d, o + 4)
    f["duration"] = struct.unpack_from(">I", d, o + 20)[0]
    (f["layer"], f["alternateGroup"], f["volume"]) = \
        struct.unpack_from(">3H", d, o + 32)
    f["matrix"] = struct.unpack_from(">9I", d, o + 40)
    f["trackWidth"], f["trackHeight"] = struct.unpack_from(">2I", d, o + 76)
    return f


def parse_mdhd(d, o):
    f = {}
    f["version"] = d[o]
    f["flags"] = struct.unpack_from(">I", d, o)[0] & 0xFFFFFF
    (f["creationTime"], f["modificationTime"], f["timeScale"],
     f["duration"]) = struct.unpack_from(">4I", d, o + 4)
    f["language"], f["quality"] = struct.unpack_from(">2H", d, o + 20)
    return f


def parse_hdlr(d, o):
    return {"version": d[o],
            "flags": struct.unpack_from(">I", d, o)[0] & 0xFFFFFF,
            "componentType": d[o + 4:o + 8],
            "componentSubType": d[o + 8:o + 12],
            "componentManufacturer": d[o + 12:o + 16],
            "componentFlags": struct.unpack_from(">I", d, o + 16)[0],
            "componentFlagsMask": struct.unpack_from(">I", d, o + 20)[0],
            # Str31: one length byte then that many characters.
            "name": d[o + 25:o + 25 + d[o + 24]]}


def parse_imagedesc(d, o):
    f = {}
    f["idSize"] = struct.unpack_from(">I", d, o)[0]
    f["cType"] = d[o + 4:o + 8]
    f["resvd1"] = struct.unpack_from(">I", d, o + 8)[0]
    f["resvd2"], f["dataRefIndex"] = struct.unpack_from(">2H", d, o + 12)
    f["version"], f["revisionLevel"] = struct.unpack_from(">2H", d, o + 16)
    f["vendor"] = d[o + 20:o + 24]
    f["temporalQuality"], f["spatialQuality"] = \
        struct.unpack_from(">2I", d, o + 24)
    f["width"], f["height"] = struct.unpack_from(">2H", d, o + 32)
    f["hRes"], f["vRes"], f["dataSize"] = struct.unpack_from(">3I", d, o + 36)
    f["frameCount"] = struct.unpack_from(">H", d, o + 48)[0]
    nlen = d[o + 50]
    f["name"] = d[o + 51:o + 51 + nlen]
    f["depth"] = struct.unpack_from(">H", d, o + 82)[0]
    f["clutID"] = struct.unpack_from(">h", d, o + 84)[0]
    f["extra"] = d[o + 86:o + f["idSize"]]
    return f


def parse_stsd(d, a):
    o = a.body
    n = struct.unpack_from(">I", d, o + 4)[0]
    out, p = [], o + 8
    for _ in range(n):
        idesc = parse_imagedesc(d, p)
        out.append(idesc)
        p += max(idesc["idSize"], 8)
    return out


def parse_stts(d, a):
    o = a.body
    n = struct.unpack_from(">I", d, o + 4)[0]
    return [struct.unpack_from(">2I", d, o + 8 + i * 8) for i in range(n)]


def parse_stsc(d, a):
    o = a.body
    n = struct.unpack_from(">I", d, o + 4)[0]
    return [struct.unpack_from(">3I", d, o + 8 + i * 12) for i in range(n)]


def parse_stsz(d, a):
    o = a.body
    fixedsz, n = struct.unpack_from(">2I", d, o + 4)
    if fixedsz:
        return fixedsz, [fixedsz] * n
    return 0, list(struct.unpack_from(">%dI" % n, d, o + 12))


def parse_stco(d, a):
    o = a.body
    n = struct.unpack_from(">I", d, o + 4)[0]
    return list(struct.unpack_from(">%dI" % n, d, o + 8))


def parse_elst(d, a):
    o = a.body
    n = struct.unpack_from(">I", d, o + 4)[0]
    return [struct.unpack_from(">IiI", d, o + 8 + i * 12) for i in range(n)]


# ---------------------------------------------------------------- palettes
_PAL8 = None


def palette8():
    """The standard Mac 256-colour table, read from 'clut' 128."""
    global _PAL8
    if _PAL8 is None:
        pal = None
        for r in probe_rez.parse(REZ):
            if r.type == b"clut" and r.id == 128:
                ct = probe_pict.parse_clut(r.data)
                pal = [(e[1] >> 8, e[2] >> 8, e[3] >> 8) for e in ct["entries"]]
        if pal is None:                                  # pragma: no cover
            pal = [(i, i, i) for i in range(256)]
        _PAL8 = pal
    return _PAL8


def palette_gray(bits):
    """Standard Mac grayscale table: index 0 = white, max index = black."""
    n = 1 << bits
    step = 255 // (n - 1)
    return [(255 - i * step,) * 3 for i in range(n)] + \
           [(0, 0, 0)] * (256 - n)


def palette_for(depth):
    if depth in GRAY_DEPTHS:
        return palette_gray(GRAY_DEPTHS[depth]), GRAY_DEPTHS[depth]
    return palette8(), depth


# ---------------------------------------------------------------- decoders
class Frame(object):
    """A width*height bytearray of palette indices, reused across samples so
    that delta frames composite exactly the way QuickTime's screen buffer
    does."""

    def __init__(self, w, h, fill=0):
        self.w, self.h = w, h
        self.px = bytearray([fill]) * (w * h)

    def copy(self):
        f = Frame(self.w, self.h)
        f.px[:] = self.px
        return f

    def rows(self):
        return [self.px[y * self.w:(y + 1) * self.w] for y in range(self.h)]


def rle_parse_header(sample):
    """-> (chunkSize, flagByte, header, startLine, numLines, dataOffset)."""
    if len(sample) < 6:
        return (len(sample), 0, 0, 0, 0, len(sample))
    v = struct.unpack_from(">I", sample, 0)[0]
    chunk = v & 0x00FFFFFF
    flag = v >> 24
    header = struct.unpack_from(">H", sample, 4)[0]
    if header & RLE_HDR_LINE_RANGE:
        start, _z0, lines, _z1 = struct.unpack_from(">4H", sample, 6)
        return (chunk, flag, header, start, lines, 14)
    return (chunk, flag, header, 0, None, 6)


def decode_rle(sample, frame, bpp, trace=None, stats=None):
    """Apple Animation. Composites in place onto `frame`. Returns bytes used.

    `stats`, if given, is a dict the opcode tallies are accumulated into:
    lineSkip / midSkip / litOps / litGroups / runOps / runGroups / endLine /
    noop, plus a set `ranges` of the (startLine, numLines) windows seen.
    """
    def bump(k, n=1):
        if stats is not None:
            stats[k] = stats.get(k, 0) + n

    chunk, flag, header, start, lines, p = rle_parse_header(sample)
    if lines is None:
        lines = frame.h
    if stats is not None:
        stats.setdefault("ranges", set())
    if len(sample) - p < 2:                  # no-op frame ("nothing changed")
        bump("noop")
        if trace is not None:
            trace.append("no-op sample: chunk=%d flag=0x%02X header=0x%04X"
                         % (chunk, flag, header))
        return len(sample)
    if stats is not None:
        stats["ranges"].add((start, lines))
    per_unit = 8 if bpp == 4 else 4          # PIXELS per 4-byte pixel group
    w, h = frame.w, frame.h
    y = start
    for _ln in range(lines):
        if p >= len(sample):
            break
        x = (sample[p] - 1) * per_unit
        p += 1
        bump("lineSkip")
        maxx = x
        while True:
            code = sample[p]
            p += 1
            if code == 0xFF:                 # -1: end of line
                bump("endLine")
                break
            if code == 0x00:                 # another skip byte
                x += (sample[p] - 1) * per_unit
                p += 1
                bump("midSkip")
                continue
            if code < 0x80:                  # literal: `code` pixel groups
                bump("litOps")
                bump("litGroups", code)
                nbytes = code * 4
                grp = sample[p:p + nbytes]
                p += nbytes
                if bpp == 4:
                    for b in grp:
                        _put(frame, x, y, b >> 4)
                        _put(frame, x + 1, y, b & 0x0F)
                        x += 2
                else:
                    for b in grp:
                        _put(frame, x, y, b)
                        x += 1
            else:                            # run: repeat one group N times
                n = 256 - code
                bump("runOps")
                bump("runGroups", n)
                grp = sample[p:p + 4]
                p += 4
                if bpp == 4:
                    nib = []
                    for b in grp:
                        nib.append(b >> 4)
                        nib.append(b & 0x0F)
                else:
                    nib = list(grp)
                for _ in range(n):
                    for v in nib:
                        _put(frame, x, y, v)
                        x += 1
            maxx = max(maxx, x)
        if trace is not None:
            trace.append("line %3d -> x ends at %d" % (y, maxx))
        y += 1
        if y >= h:
            break
    # Every 'rle ' sample in this tree ends with one extra 0x00 byte: the
    # end-of-stream marker (a skip count of 0, which cannot start a real line).
    if p < len(sample) and sample[p] == RLE_END:
        p += 1
    return p


def _put(frame, x, y, v):
    if 0 <= x < frame.w and 0 <= y < frame.h:
        frame.px[y * frame.w + x] = v


def decode_raw8(sample, frame):
    n = min(len(sample), frame.w * frame.h)
    frame.px[:n] = sample[:n]
    return n


CPAIR, CQUAD, COCTET = 2, 4, 8
COLORS_PER_TABLE = 256


class SmcState(object):
    def __init__(self):
        self.pairs = bytearray(CPAIR * COLORS_PER_TABLE)
        self.quads = bytearray(CQUAD * COLORS_PER_TABLE)
        self.octets = bytearray(COCTET * COLORS_PER_TABLE)
        self.pi = self.qi = self.oi = 0


def decode_smc(sample, frame, st, stats=None):
    """Apple Graphics ('smc '). Returns bytes consumed.

    `stats`, if given, is a dict tallying "0xN0" opcode-class counts plus
    "blocks", the number of 4x4 blocks emitted.
    """
    w, h = frame.w, frame.h
    bw, bh = (w + 3) // 4, (h + 3) // 4
    total = bw * bh
    p = 4                                    # skip the chunkSize long
    bx = by = 0                              # block coordinates
    prev = []                                # (bx,by) of blocks already drawn

    def blk_put(bx, by, vals):
        for j in range(4):
            for i in range(4):
                _put(frame, bx * 4 + i, by * 4 + j, vals[j * 4 + i])

    def blk_get(bx, by):
        out = []
        for j in range(4):
            for i in range(4):
                x, y = bx * 4 + i, by * 4 + j
                out.append(frame.px[y * frame.w + x]
                           if 0 <= x < w and 0 <= y < h else 0)
        return out

    def advance():
        nonlocal bx, by, total
        prev.append((bx, by))
        bx += 1
        if bx >= bw:
            bx = 0
            by += 1
        total -= 1
        if stats is not None:
            stats["blocks"] = stats.get("blocks", 0) + 1

    while p < len(sample) and total > 0:
        op = sample[p]
        p += 1
        hi = op & 0xF0
        if stats is not None:
            k = "0x%02X" % hi
            stats[k] = stats.get(k, 0) + 1
        if hi <= 0x70:
            if op & 0x10:
                n = sample[p] + 1
                p += 1
            else:
                n = (op & 0x0F) + 1
        else:
            n = (op & 0x0F) + 1
        if hi in (0x00, 0x10):                       # skip N blocks
            for _ in range(n):
                advance()
        elif hi in (0x20, 0x30):                     # repeat previous block
            src = prev[-1]
            vals = blk_get(*src)
            for _ in range(n):
                blk_put(bx, by, vals)
                advance()
        elif hi in (0x40, 0x50):                     # repeat previous pair
            a, b = prev[-2], prev[-1]
            va, vb = blk_get(*a), blk_get(*b)
            for _ in range(n * 2):
                blk_put(bx, by, va if _ % 2 == 0 else vb)
                advance()
        elif hi in (0x60, 0x70):                     # N one-colour blocks
            c = sample[p]
            p += 1
            for _ in range(n):
                blk_put(bx, by, [c] * 16)
                advance()
        elif hi in (0x80, 0x90):                     # 2-colour blocks
            if hi == 0x80:
                base = CPAIR * st.pi
                for i in range(CPAIR):
                    st.pairs[base + i] = sample[p]
                    p += 1
                st.pi = (st.pi + 1) % COLORS_PER_TABLE
            else:
                base = CPAIR * sample[p]
                p += 1
            for _ in range(n):
                flags = struct.unpack_from(">H", sample, p)[0]
                p += 2
                vals, m = [], 0x8000
                for _j in range(16):
                    vals.append(st.pairs[base + (1 if flags & m else 0)])
                    m >>= 1
                blk_put(bx, by, vals)
                advance()
        elif hi in (0xA0, 0xB0):                     # 4-colour blocks
            if hi == 0xA0:
                base = CQUAD * st.qi
                for i in range(CQUAD):
                    st.quads[base + i] = sample[p]
                    p += 1
                st.qi = (st.qi + 1) % COLORS_PER_TABLE
            else:
                base = CQUAD * sample[p]
                p += 1
            for _ in range(n):
                flags = struct.unpack_from(">I", sample, p)[0]
                p += 4
                vals, sh = [], 30
                for _j in range(16):
                    vals.append(st.quads[base + ((flags >> sh) & 3)])
                    sh -= 2
                blk_put(bx, by, vals)
                advance()
        elif hi in (0xC0, 0xD0):                     # 8-colour blocks
            if hi == 0xC0:
                base = COCTET * st.oi
                for i in range(COCTET):
                    st.octets[base + i] = sample[p]
                    p += 1
                st.oi = (st.oi + 1) % COLORS_PER_TABLE
            else:
                base = COCTET * sample[p]
                p += 1
            for _ in range(n):
                v1, v2, v3 = struct.unpack_from(">3H", sample, p)
                p += 6
                fa = (v1 << 8) | (v2 >> 8)
                fb = ((v2 & 0xFF) << 16) | v3
                vals = []
                for j in range(4):
                    f = fa if j < 2 else fb
                    sh = 21 - (j % 2) * 12
                    for i in range(4):
                        vals.append(st.octets[base + ((f >> (sh - i * 3)) & 7)])
                blk_put(bx, by, vals)
                advance()
        elif hi == 0xE0:                             # 16 raw colour bytes
            for _ in range(n):
                vals = list(sample[p:p + 16])
                p += 16
                blk_put(bx, by, vals)
                advance()
        else:                                        # 0xF0: undefined
            raise ValueError("smc: undefined opcode 0x%02X at +%d" % (op, p - 1))
    return p, total


# --------------------------------------------------- movie model (both kinds)
class Movie(object):
    """Either a real parsed movie (Demo House.mov) or a reconstruction of a
    headerless one."""

    def __init__(self, path):
        self.path = path
        self.name = os.path.basename(path)
        self.data = open(path, "rb").read()
        self.size = len(self.data)
        self.atoms = list(walk(self.data))
        self.mdat = find(self.data, b"mdat")
        self.moov = find(self.data, b"moov")
        self.mdat_decl = (raw_size_field(self.data, self.mdat.off)
                          if self.mdat else None)
        self.has_moov = self.moov is not None
        self.warnings = []
        if self.has_moov:
            self._from_moov()
        else:
            self._reconstruct()

    # -- real header ------------------------------------------------------
    def _from_moov(self):
        d = self.data
        self.mvhd = parse_mvhd(d, find(d, b"mvhd", self.moov.body,
                                       self.moov.end).body)
        self.traks = []
        for a in walk(d, self.moov.body, self.moov.end):
            if a.type == b"trak":
                self.traks.append(self._parse_trak(a))
        vid = [t for t in self.traks
               if t["hdlr_media"]["componentSubType"] == b"vide"]
        t = (vid or self.traks)[0]
        self.trak = t
        self.codec = t["stsd"][0]["cType"]
        self.depth = t["stsd"][0]["depth"]
        self.width = t["stsd"][0]["width"]
        self.height = t["stsd"][0]["height"]
        self.box_w = fixed(t["tkhd"]["trackWidth"])
        self.box_h = fixed(t["tkhd"]["trackHeight"])
        self.timescale = t["mdhd"]["timeScale"]
        self.durations = []
        for cnt, dur in t["stts"]:
            self.durations += [dur] * cnt
        self.sizes = t["sizes"]
        self.offsets = t["offsets"]
        self.nframes = len(self.sizes)
        self.duration = self.mvhd["duration"]
        self.mvhd_timescale = self.mvhd["timeScale"]
        self.keyframes = None
        self.stss = t["stss"]

    def _parse_trak(self, a):
        d = self.data
        t = {"atom": a}
        t["tkhd"] = parse_tkhd(d, find(d, b"tkhd", a.body, a.end).body)
        el = find(d, b"elst", a.body, a.end)
        t["elst"] = parse_elst(d, el) if el else None
        mdia = find(d, b"mdia", a.body, a.end)
        t["mdhd"] = parse_mdhd(d, find(d, b"mdhd", mdia.body, mdia.end).body)
        hds = [x for x in walk(d, mdia.body, mdia.end) if x.type == b"hdlr"]
        t["hdlr_media"] = parse_hdlr(d, hds[0].body)
        t["hdlr_data"] = parse_hdlr(d, hds[1].body) if len(hds) > 1 else None
        stbl = find(d, b"stbl", mdia.body, mdia.end)
        t["stsd"] = parse_stsd(d, find(d, b"stsd", stbl.body, stbl.end))
        t["stts"] = parse_stts(d, find(d, b"stts", stbl.body, stbl.end))
        t["stsc"] = parse_stsc(d, find(d, b"stsc", stbl.body, stbl.end))
        fixedsz, sizes = parse_stsz(d, find(d, b"stsz", stbl.body, stbl.end))
        t["stsz_fixed"], t["sizes"] = fixedsz, sizes
        t["offsets"] = parse_stco(d, find(d, b"stco", stbl.body, stbl.end))
        ss = find(d, b"stss", stbl.body, stbl.end)
        t["stss"] = parse_stco(d, ss) if ss else None
        vm = find(d, b"vmhd", stbl.parent.body, stbl.parent.end) \
            if stbl.parent else None
        # vmhd payload is 12 bytes: version/flags, graphicsMode, opColor RGB.
        t["vmhd"] = (struct.unpack_from(">IH3H", d, vm.body) if vm else None)
        dr = find(d, b"dref", a.body, a.end)
        t["dref"] = None
        if dr:
            cnt = struct.unpack_from(">I", d, dr.body + 4)[0]
            ents, q = [], dr.body + 8
            for _ in range(cnt):
                esz, etyp = struct.unpack_from(">I4s", d, q)
                ents.append((etyp, struct.unpack_from(">I", d, q + 8)[0] & 0xFFFFFF,
                             d[q + 12:q + esz]))
                q += esz
            t["dref"] = ents
        return t

    # -- reconstruction ---------------------------------------------------
    def _reconstruct(self):
        """No moov: walk the mdat payload as a self-delimiting sample stream."""
        d = self.data
        base = self.mdat.body if self.mdat else 0
        end = self.mdat.end if self.mdat else len(d)
        blob = d[base:end]
        self.mvhd = self.trak = self.traks = None
        self.timescale = None
        self.mvhd_timescale = None
        self.duration = None
        self.durations = None
        self.stss = None
        self.depth = 8
        cand = self._try_chunked(blob, base) or self._try_raw(blob, base)
        if cand is None:
            self.codec = None
            self.width = self.height = 0
            self.sizes, self.offsets = [], []
            self.nframes = 0
            self.warnings.append("could not reconstruct a sample table")
        else:
            (self.codec, self.width, self.height,
             self.sizes, self.offsets) = cand
            self.nframes = len(self.sizes)
        self.box_w, self.box_h = float(self.width), float(self.height)
        self.keyframes = None

    def _try_chunked(self, blob, base):
        """'rle ' and 'smc ' both start each sample with a 24-bit length."""
        sizes, offs, p = [], [], 0
        while p + 4 <= len(blob):
            n = struct.unpack_from(">I", blob, p)[0] & 0x00FFFFFF
            if n < 4 or p + n > len(blob):
                return None
            sizes.append(n)
            offs.append(base + p)
            p += n
        if p != len(blob) or not sizes:
            return None
        # 'rle ' samples carry a recognisable 16-bit header at +4.
        hdrs = [struct.unpack_from(">H", blob, o - base + 4)[0]
                for o, s in zip(offs, sizes) if s >= 6]
        if hdrs and all(h in (0x0000, 0x0008) for h in hdrs):
            w, h = self._infer_rle_geometry(blob, base, sizes, offs)
            return (b"rle ", w, h, sizes, offs)
        # otherwise try 'smc ' at the TV screen size
        st = SmcState()
        f = Frame(TV_SCREEN_W, TV_SCREEN_H)
        for o, s in zip(offs, sizes):
            used, left = decode_smc(blob[o - base:o - base + s], f, st)
            if used != s or left != 0:
                self.warnings.append(
                    "smc: sample at %d used %d/%d bytes, %d blocks left"
                    % (o, used, s, left))
                return None
        return (b"smc ", TV_SCREEN_W, TV_SCREEN_H, sizes, offs)

    def _try_raw(self, blob, base):
        """'raw ' at depth 8 has no per-sample header at all: every sample is
        exactly width*height bytes.  Prefer the TV screen size (64x49 =
        3136 bytes, StructuresInit.c:565); fall back to the divisor of the
        payload length closest to it."""
        n = len(blob)
        frame = TV_SCREEN_W * TV_SCREEN_H
        if n % frame != 0:
            cands = [h for h in range(8, 256)
                     if n % (TV_SCREEN_W * h) == 0]
            if not cands:
                return None
            h = min(cands, key=lambda v: abs(v - TV_SCREEN_H))
            frame = TV_SCREEN_W * h
        h = frame // TV_SCREEN_W
        cnt = n // frame
        sizes = [frame] * cnt
        offs = [base + i * frame for i in range(cnt)]
        return (b"raw ", TV_SCREEN_W, h, sizes, offs)

    def _infer_rle_geometry(self, blob, base, sizes, offs):
        """Height comes from the sample's own line range; width from the
        widest line actually painted.  Documented heuristic: assume depth 8
        (4 pixels per pixel group), which is what all 13 headerless 'rle '
        movies use -- Demo House.mov, the one file that kept its header, is
        the only depth-36 movie in the tree."""
        h = 0
        for o, s in zip(offs, sizes):
            ch, fl, hd, st, ln, _p = rle_parse_header(blob[o - base:o - base + s])
            if ln:
                h = max(h, st + ln)
        h = h or TV_SCREEN_H
        # Decode with a deliberately over-wide canvas and see where lines stop.
        probe = Frame(256, h)
        maxx = 0
        for o, s in zip(offs, sizes):
            tr = []
            decode_rle(blob[o - base:o - base + s], probe, 8, tr)
            for line in tr:
                if "x ends at" in line:
                    maxx = max(maxx, int(line.rsplit(" ", 1)[1]))
        return (maxx or TV_SCREEN_W), h

    # -- frames -----------------------------------------------------------
    def sample(self, i):
        return self.data[self.offsets[i]:self.offsets[i] + self.sizes[i]]

    def frames(self):
        """Yield a Frame per sample, composited in order."""
        bpp = GRAY_DEPTHS.get(self.depth, 8)
        f = Frame(self.width, self.height,
                  0 if bpp != 4 else 0)
        st = SmcState()
        for i in range(self.nframes):
            s = self.sample(i)
            if self.codec == b"rle ":
                used = decode_rle(s, f, bpp)
                if used != len(s):
                    self.warnings.append("frame %d: used %d of %d bytes"
                                         % (i, used, len(s)))
            elif self.codec == b"raw ":
                decode_raw8(s, f)
            elif self.codec == b"smc ":
                used, left = decode_smc(s, f, st)
                if used != len(s) or left:
                    self.warnings.append(
                        "frame %d: smc used %d/%d, %d blocks left"
                        % (i, used, len(s), left))
            else:
                raise ValueError("no decoder for %r" % self.codec)
            yield f.copy()

    def fps(self):
        if not self.durations or not self.timescale:
            return None
        n = len(self.durations)
        tot = sum(self.durations)
        return n * self.timescale / float(tot) if tot else None

    def classify(self):
        """Per-sample (size, chunkSize, flagByte, header, startLine, numLines,
        kind, allLines).

        `kind` is taken from the flag byte in the high 8 bits of the chunkSize
        long: 0x40 => key frame, 0x00 => delta.  `allLines` is independent: it
        says the sample's line range covers row 0..height-1, which a delta
        frame can also do (Nemo's Market has 6 such)."""
        out = []
        for i in range(self.nframes):
            s = self.sample(i)
            if self.codec == b"rle ":
                ch, fl, hd, st, ln, p = rle_parse_header(s)
                allln = (ln is not None and st == 0 and ln >= self.height)
                if len(s) - p < 2:
                    kind = "no-op"
                elif fl == RLE_FLAG_FULL:
                    kind = "key"
                else:
                    kind = "delta"
                out.append((self.sizes[i], ch, fl, hd, st, ln, kind, allln))
            elif self.codec == b"smc ":
                ch = struct.unpack_from(">I", s, 0)[0]
                out.append((self.sizes[i], ch & 0xFFFFFF, ch >> 24,
                            None, None, None, "smc", None))
            else:
                out.append((self.sizes[i], None, None, None, None, None,
                            "raw", True))
        return out


def house_movies():
    return [os.path.join(HOUSES, fn) for fn in sorted(os.listdir(HOUSES))
            if fn.endswith(".mov")]


def resolve(args):
    if not args or args == ["all"]:
        return house_movies()
    out = []
    for a in args:
        if a == "all":
            out.extend(house_movies())
        elif os.path.exists(a):
            out.append(a)
        elif os.path.exists(os.path.join(HOUSES, a)):
            out.append(os.path.join(HOUSES, a))
        elif os.path.exists(os.path.join(HOUSES, a + ".mov")):
            out.append(os.path.join(HOUSES, a + ".mov"))
        else:
            sys.exit("no such movie: %s" % a)
    return out


# ------------------------------------------------------------------ commands
def cmd_survey(argv):
    paths = resolve(argv)
    print("%-22s %8s %10s %6s %8s %7s %-5s %6s %5s %-9s"
          % ("movie", "bytes", "mdat_decl", "moov", "codec", "frames", "depth",
             "WxH", "fps", "kinds"))
    tot = 0
    for p in paths:
        m = Movie(p)
        tot += m.size
        kinds = {}
        for row in m.classify():
            kinds[row[6]] = kinds.get(row[6], 0) + 1
        ks = " ".join("%s=%d" % (k, kinds[k]) for k in sorted(kinds))
        print("%-22s %8d %10s %6s %8s %7d %-5s %6s %5s %-9s"
              % (m.name[:-4], m.size,
                 "%d" % m.mdat_decl if m.mdat_decl is not None else "-",
                 "yes @%d" % m.moov.off if m.has_moov else "NO",
                 fourcc(m.codec) if m.codec else "?", m.nframes,
                 m.depth, "%dx%d" % (m.width, m.height),
                 ("%.4f" % m.fps()) if m.fps() else "-", ks))
        for w in m.warnings:
            print("    !! %s" % w)
    print("-- %d movies, %d bytes total" % (len(paths), tot))


def cmd_atoms(argv):
    for p in resolve(argv):
        d = open(p, "rb").read()
        print("== %s (%d bytes)" % (os.path.basename(p), len(d)))
        print("   %-10s %-10s %-10s %s" % ("offset", "size", "declared", "type"))
        for a in walk(d):
            decl = raw_size_field(d, a.off)
            note = ""
            if decl == 0:
                note = "   <- size field is 0: 'extends to EOF'"
            print("   %-10d %-10d %-10d %s%s%s"
                  % (a.off, a.size, decl, "  " * a.depth, fourcc(a.type), note))
        if find(d, b"moov") is None:
            print("   ** no 'moov' atom: the movie header is missing **")
        print()


def _pr(k, v, extra=""):
    print("  %-20s %s%s" % (k, v, extra))


def cmd_moov(argv):
    for p in resolve(argv):
        m = Movie(p)
        print("== %s" % m.name)
        if not m.has_moov:
            print("  no 'moov' atom -- nothing to dump (see `samples`)")
            print()
            continue
        print("moov @%d size %d" % (m.moov.off, m.moov.size))
        f = m.mvhd
        print(" mvhd")
        _pr("version/flags", "%d / 0x%06X" % (f["version"], f["flags"]))
        _pr("creationTime", "%d" % f["creationTime"],
            "  = %s" % mac_time(f["creationTime"]))
        _pr("modificationTime", "%d" % f["modificationTime"],
            "  = %s" % mac_time(f["modificationTime"]))
        _pr("timeScale", f["timeScale"], "  units/second")
        _pr("duration", f["duration"],
            "  = %.4f s" % (f["duration"] / float(f["timeScale"])))
        _pr("preferredRate", "0x%08X" % f["preferredRate"],
            "  = %.4f" % fixed(f["preferredRate"]))
        _pr("preferredVolume", "0x%04X" % f["preferredVolume"],
            "  = %.4f" % ufix8(f["preferredVolume"]))
        _pr("matrix", " ".join("0x%08X" % v for v in f["matrix"][:3]))
        _pr("", " ".join("0x%08X" % v for v in f["matrix"][3:6]))
        _pr("", " ".join("0x%08X" % v for v in f["matrix"][6:]))
        for k in ("previewTime", "previewDuration", "posterTime",
                  "selectionTime", "selectionDuration", "currentTime",
                  "nextTrackID"):
            _pr(k, f[k])
        for ti, t in enumerate(m.traks):
            print(" trak[%d] @%d size %d" % (ti, t["atom"].off, t["atom"].size))
            g = t["tkhd"]
            print("  tkhd")
            _pr("version/flags", "%d / 0x%06X" % (g["version"], g["flags"]),
                "  (%s)" % "|".join(n for b, n in TKHD_FLAGS if g["flags"] & b))
            _pr("creationTime", g["creationTime"],
                "  = %s" % mac_time(g["creationTime"]))
            _pr("modificationTime", g["modificationTime"],
                "  = %s" % mac_time(g["modificationTime"]))
            _pr("trackID", g["trackID"])
            _pr("duration", g["duration"], "  (movie timeScale)")
            _pr("layer", g["layer"])
            _pr("alternateGroup", g["alternateGroup"])
            _pr("volume", "0x%04X" % g["volume"], "  = %.4f" % ufix8(g["volume"]))
            _pr("matrix", " ".join("0x%08X" % v for v in g["matrix"][:3]))
            _pr("", " ".join("0x%08X" % v for v in g["matrix"][3:6]))
            _pr("", " ".join("0x%08X" % v for v in g["matrix"][6:]))
            _pr("trackWidth", "0x%08X" % g["trackWidth"],
                "  = %.4f px" % fixed(g["trackWidth"]))
            _pr("trackHeight", "0x%08X" % g["trackHeight"],
                "  = %.4f px" % fixed(g["trackHeight"]))
            if t["elst"]:
                print("  elst  %d entry(s)" % len(t["elst"]))
                for dur, mt, rate in t["elst"]:
                    _pr("entry", "duration=%d mediaTime=%d rate=0x%08X (%.4f)"
                        % (dur, mt, rate, fixed(rate)))
            g = t["mdhd"]
            print("  mdhd")
            _pr("timeScale", g["timeScale"])
            _pr("duration", g["duration"],
                "  = %.4f s" % (g["duration"] / float(g["timeScale"])))
            _pr("language", g["language"])
            _pr("quality", g["quality"])
            for lbl, hd in (("hdlr (media)", t["hdlr_media"]),
                            ("hdlr (data)", t["hdlr_data"])):
                if not hd:
                    continue
                print("  %s" % lbl)
                _pr("componentType", "'%s'" % fourcc(hd["componentType"]))
                _pr("componentSubType", "'%s'" % fourcc(hd["componentSubType"]))
                _pr("manufacturer", "'%s'"
                    % fourcc(hd["componentManufacturer"]))
                _pr("name", repr(bytes(hd["name"])))
            if t["vmhd"]:
                vf, gmode, r, gg, b = t["vmhd"]
                print("  vmhd")
                _pr("version/flags", "0x%08X" % vf,
                    "  (flag 0x000001 = noLeanAhead)")
                _pr("graphicsMode", "%d (0x%04X)" % (gmode, gmode),
                    "  (%s)" % GRAPHICS_MODES.get(gmode, "?"))
                _pr("opColor", "(0x%04X,0x%04X,0x%04X)" % (r, gg, b))
            if t["dref"]:
                print("  dref  %d entry(s)" % len(t["dref"]))
                for etyp, eflags, edata in t["dref"]:
                    _pr("entry", "'%s' flags=0x%06X %d payload bytes"
                        % (fourcc(etyp), eflags, len(edata)),
                        "  (0x000001 = self reference)")
            print("  stsd  %d entry(s)" % len(t["stsd"]))
            for idesc in t["stsd"]:
                _pr("idSize", idesc["idSize"])
                _pr("cType", "'%s'" % fourcc(idesc["cType"]),
                    "  (%s)" % CODECS.get(idesc["cType"], "?"))
                _pr("dataRefIndex", idesc["dataRefIndex"])
                _pr("version/revision", "%d / %d"
                    % (idesc["version"], idesc["revisionLevel"]))
                _pr("vendor", "'%s'" % fourcc(idesc["vendor"]))
                _pr("temporalQuality", idesc["temporalQuality"])
                _pr("spatialQuality", idesc["spatialQuality"],
                    "  (0x400 = codecLosslessQuality)")
                _pr("width x height", "%d x %d"
                    % (idesc["width"], idesc["height"]))
                _pr("hRes / vRes", "0x%08X / 0x%08X = %.1f / %.1f dpi"
                    % (idesc["hRes"], idesc["vRes"],
                       fixed(idesc["hRes"]), fixed(idesc["vRes"])))
                _pr("dataSize", idesc["dataSize"],
                    "  (0 = use the sample table)")
                _pr("frameCount", idesc["frameCount"])
                _pr("name", repr(bytes(idesc["name"])))
                _pr("depth", idesc["depth"],
                    "  (%s)" % ("%d-bit grayscale" % GRAY_DEPTHS[idesc["depth"]]
                                if idesc["depth"] in GRAY_DEPTHS
                                else "%d-bit indexed/direct" % idesc["depth"]))
                _pr("clutID", idesc["clutID"],
                    "  (0 = a ColorTable follows this ImageDescription, "
                    "-1 = use the destination's, >=8 = a 'clut' resource ID)")
                if idesc["extra"]:
                    _pr("trailing bytes", idesc["extra"].hex())
            print("  stts  %d entry(s)" % len(t["stts"]))
            for cnt, dur in t["stts"]:
                _pr("entry", "sampleCount=%d sampleDuration=%d" % (cnt, dur))
            tot = sum(c * d2 for c, d2 in t["stts"])
            n = sum(c for c, _ in t["stts"])
            _pr("total", "%d units, %d samples" % (tot, n),
                "  = %.4f fps" % (n * t["mdhd"]["timeScale"] / float(tot)))
            print("  stsc  %d entry(s)" % len(t["stsc"]))
            for fc, spc, sid in t["stsc"]:
                _pr("entry", "firstChunk=%d samplesPerChunk=%d descID=%d"
                    % (fc, spc, sid))
            print("  stsz  fixed=%d count=%d" % (t["stsz_fixed"],
                                                 len(t["sizes"])))
            _pr("sum of sizes", sum(t["sizes"]))
            _pr("min/max", "%d / %d" % (min(t["sizes"]), max(t["sizes"])))
            print("  stco  count=%d  first=%d last=%d"
                  % (len(t["offsets"]), t["offsets"][0], t["offsets"][-1]))
            print("  stss  %s"
                  % ("absent: EVERY sample is a key frame" if not t["stss"]
                     else t["stss"]))
        ud = find(m.data, b"udta", m.moov.body, m.moov.end)
        print(" udta  %s" % ("absent (no 'LOOP' user data on disk)"
                             if ud is None else "present @%d" % ud.off))
        print()


def cmd_samples(argv):
    if not argv:
        sys.exit("usage: probe_mov.py samples FILE")
    for p in resolve(argv):
        m = Movie(p)
        print("== %s  codec '%s'  %dx%d  depth %d  %d samples%s"
              % (m.name, fourcc(m.codec) if m.codec else "?",
                 m.width, m.height, m.depth, m.nframes,
                 "" if m.has_moov else "   (RECONSTRUCTED: no moov)"))
        print("  %-4s %-9s %-7s %-8s %-7s %-7s %-6s %-6s %-7s %s"
              % ("#", "offset", "size", "chunkSz", "flag", "header",
                 "line0", "lines", "kind", "allLines"))
        cls = m.classify()
        for i, (sz, ch, fl, hd, st, ln, kind, allln) in enumerate(cls):
            print("  %-4d %-9d %-7d %-8s %-7s %-7s %-6s %-6s %-7s %s"
                  % (i, m.offsets[i], sz,
                     "-" if ch is None else ch,
                     "-" if fl is None else "0x%02X" % fl,
                     "-" if hd is None else "0x%04X" % hd,
                     "-" if st is None else st,
                     "-" if ln is None else ln, kind,
                     "-" if allln is None else ("yes" if allln else "no")))
        print("  sum of sample sizes %d; mdat payload %d; %s"
              % (sum(m.sizes),
                 (m.mdat.size - m.mdat.hdr) if m.mdat else 0,
                 "EXACT" if m.mdat and sum(m.sizes) == m.mdat.size - m.mdat.hdr
                 else "difference %d" % ((m.mdat.size - m.mdat.hdr)
                                         - sum(m.sizes))))
        # Force a decode so composition warnings surface.
        for _ in m.frames():
            pass
        for w in m.warnings:
            print("  !! %s" % w)
        print()


def cmd_rle(argv):
    if not argv:
        sys.exit("usage: probe_mov.py rle FILE [N]")
    m = Movie(resolve(argv[:1])[0])
    n = int(argv[1]) if len(argv) > 1 else 0
    s = m.sample(n)
    ch, fl, hd, st, ln, p = rle_parse_header(s)
    print("%s sample %d: %d bytes at offset %d"
          % (m.name, n, len(s), m.offsets[n]))
    print("  chunkSize      %d (0x%06X)   %s"
          % (ch, ch, "== sample size" if ch == len(s) else "!= sample size!"))
    print("  flag byte      0x%02X%s"
          % (fl, "  (RLE_FLAG_FULL)" if fl == RLE_FLAG_FULL else ""))
    print("  header         0x%04X%s"
          % (hd, "  (line range follows)" if hd & RLE_HDR_LINE_RANGE else ""))
    print("  startLine      %s" % st)
    print("  numLines       %s" % ln)
    print("  data starts    +%d" % p)
    for i in range(0, min(len(s), 64), 16):
        chunk = s[i:i + 16]
        print("  %04X  %-47s  %s"
              % (i, " ".join("%02X" % c for c in chunk),
                 "".join(chr(c) if 32 <= c < 127 else "." for c in chunk)))
    f = Frame(m.width, m.height)
    tr = []
    used = decode_rle(s, f, GRAY_DEPTHS.get(m.depth, 8), tr)
    for line in tr:
        print("  %s" % line)
    print("  consumed %d of %d bytes" % (used, len(s)))


def cmd_opcodes(argv):
    """Opcode-usage totals per movie -- the tables in the analysis document."""
    paths = resolve(argv)
    rle_rows, smc_rows = [], []
    for p in paths:
        m = Movie(p)
        if m.codec not in (b"rle ", b"smc "):
            continue
        stats = {}
        st = SmcState() if m.codec == b"smc " else None
        f = Frame(m.width, m.height)
        for i in range(m.nframes):
            s = m.sample(i)
            if m.codec == b"rle ":
                decode_rle(s, f, GRAY_DEPTHS.get(m.depth, 8), None, stats)
            else:
                decode_smc(s, f, st, stats)
        if m.codec == b"rle ":
            rle_rows.append((m.name[:-4], stats))
        else:
            smc_rows.append((m.name[:-4], stats))

    if rle_rows:
        cols = ["lineSkip", "midSkip", "litOps", "litGroups", "runOps",
                "runGroups", "endLine", "noop"]
        print("'rle ' opcode usage")
        print("  %-22s %s %7s" % ("movie",
                                  " ".join("%9s" % c for c in cols), "ranges"))
        for name, s in rle_rows:
            print("  %-22s %s %7d"
                  % (name, " ".join("%9d" % s.get(c, 0) for c in cols),
                     len(s.get("ranges", ()))))
        print()
    if smc_rows:
        print("'smc ' opcode usage")
        for name, s in smc_rows:
            print("  %s" % name)
            for k in sorted(k for k in s if k.startswith("0x")):
                print("    %-6s %6d" % (k, s[k]))
            print("    %-6s %6d" % ("blocks", s.get("blocks", 0)))


def cmd_decode(argv):
    if len(argv) < 2:
        sys.exit("usage: probe_mov.py decode FILE OUTDIR")
    m = Movie(resolve(argv[:1])[0])
    outdir = argv[1]
    os.makedirs(outdir, exist_ok=True)
    pal, _ = palette_for(m.depth)
    n = 0
    for i, f in enumerate(m.frames()):
        rows = f.rows()
        probe_pict.write_png_indexed(
            os.path.join(outdir, "frame_%03d.png" % i), f.w, f.h,
            [[rows[y][x] for x in range(f.w)] for y in range(f.h)], pal)
        n += 1
    print("wrote %d PNGs (%dx%d, depth %d) to %s" % (n, m.width, m.height,
                                                     m.depth, outdir))
    for w in m.warnings:
        print("  !! %s" % w)


def cmd_sheet(argv):
    if len(argv) < 2:
        sys.exit("usage: probe_mov.py sheet FILE OUT.png [COLS]")
    m = Movie(resolve(argv[:1])[0])
    out = argv[1]
    cols = int(argv[2]) if len(argv) > 2 else 8
    frames = list(m.frames())
    rows_n = (len(frames) + cols - 1) // cols
    gap = 2
    W = cols * (m.width + gap) - gap
    H = rows_n * (m.height + gap) - gap
    pal, _ = palette_for(m.depth)
    canvas = [[None] * W for _ in range(H)]
    for i, f in enumerate(frames):
        ox = (i % cols) * (m.width + gap)
        oy = (i // cols) * (m.height + gap)
        rows = f.rows()
        for y in range(m.height):
            for x in range(m.width):
                canvas[oy + y][ox + x] = rows[y][x]
    pal = list(pal)
    while len(pal) < 256:
        pal.append((255, 0, 255))
    canvas = [[(255 if v is None else v) for v in row] for row in canvas]
    probe_pict.write_png_indexed(out, W, H, canvas, pal)
    print("%s: %d frames, %d cols, %dx%d" % (out, len(frames), cols, W, H))


def cmd_extract(argv):
    """The form a Go port should embed: one flat 8-bit index buffer per frame
    plus a manifest.  No codec needed at runtime."""
    if not argv:
        sys.exit("usage: probe_mov.py extract <outdir>")
    outdir = argv[0]
    os.makedirs(outdir, exist_ok=True)
    man = open(os.path.join(outdir, "manifest.tsv"), "w")
    man.write("house\tfile\tcodec\tdepth\twidth\theight\tframes\ttimescale\t"
              "frame_dur\tfps\thas_moov\n")
    for p in house_movies():
        m = Movie(p)
        stem = os.path.basename(p)[:-4].replace(" ", "_").replace("'", "")
        raw = os.path.join(outdir, stem + ".idx8")
        pal, bits = palette_for(m.depth)
        with open(raw, "wb") as fh:
            for f in m.frames():
                if bits == 4:                 # widen 4-bit gray to 8-bit gray
                    fh.write(bytes(pal[v][0] for v in f.px))
                else:
                    fh.write(bytes(f.px))
        dur = (m.durations[0] if m.durations else 0)
        man.write("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\n"
                  % (os.path.basename(p)[:-4], os.path.basename(raw),
                     fourcc(m.codec) if m.codec else "?", m.depth,
                     m.width, m.height, m.nframes,
                     m.timescale if m.timescale else "-", dur or "-",
                     ("%.4f" % m.fps()) if m.fps() else "-",
                     "yes" if m.has_moov else "no"))
    man.close()
    print("wrote %d .idx8 files + manifest.tsv to %s"
          % (len(house_movies()), outdir))
    print("format: one frame after another, top row first, no padding.")
    print("        depth-8 movies store 'clut' 128 palette indices;")
    print("        Demo House (depth 36) is widened to 8-bit gray levels.")


def cmd_all(argv):
    cmd_survey([])
    print()
    cmd_samples(house_movies())


COMMANDS = {"survey": cmd_survey, "atoms": cmd_atoms, "moov": cmd_moov,
            "samples": cmd_samples, "rle": cmd_rle, "opcodes": cmd_opcodes,
            "decode": cmd_decode, "sheet": cmd_sheet, "extract": cmd_extract,
            "all": cmd_all}


def main(argv):
    if not argv or argv[0] not in COMMANDS:
        print(__doc__)
        return 2
    COMMANDS[argv[0]](argv[1:])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
