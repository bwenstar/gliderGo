#!/usr/bin/env python3
"""probe_pict.py -- from-scratch QuickDraw PICT (v1 + v2) reader/decoder.

Written for the Glider PRO 1.0.4 reverse-documentation effort.  No third-party
dependencies: only the Python standard library (struct + zlib).  This is
deliberately a *probe*, not a general-purpose PICT library -- it implements
exactly the opcode set that Glider PRO's 152 'PICT' resources actually use,
plus enough of the rest of the opcode table to walk (skip) anything else
without desynchronising the stream.

Input is a raw resource-fork payload: the bytes of one 'PICT' resource, i.e.
    +0   int16  picSize      (16-bit, wraps for pictures > 32767 bytes)
    +2   Rect   picFrame     (top, left, bottom, right -- big-endian int16 x4)
    +10  ...    opcode stream

Subcommands
-----------
  header   FILE...              print picSize / picFrame / version of each
  opcodes  FILE                 dump the decoded opcode stream
  histogram DIR                 opcode histogram across every *.bin in DIR
  pixmaps  DIR                  PixMap/BitMap field table across every *.bin
  decode   FILE OUT.png         decode to PNG
  decodeall DIR OUTDIR          decode every *.bin in DIR to OUTDIR
  clut     FILE                 parse a 'clut' resource and print the table
  ctabs    DIR                  compare every PICT's embedded ColorTable

Mac Rect byte order on disk is (top, left, bottom, right).  Everything in this
file is big-endian; that is a load-bearing fact, not an accident.
"""

import os
import struct
import sys
import zlib

# --------------------------------------------------------------------------
# Basic readers
# --------------------------------------------------------------------------


class Reader:
    def __init__(self, data, pos=0):
        self.d = data
        self.p = pos

    def u8(self):
        v = self.d[self.p]
        self.p += 1
        return v

    def s8(self):
        v = struct.unpack_from(">b", self.d, self.p)[0]
        self.p += 1
        return v

    def u16(self):
        v = struct.unpack_from(">H", self.d, self.p)[0]
        self.p += 2
        return v

    def s16(self):
        v = struct.unpack_from(">h", self.d, self.p)[0]
        self.p += 2
        return v

    def u32(self):
        v = struct.unpack_from(">I", self.d, self.p)[0]
        self.p += 4
        return v

    def s32(self):
        v = struct.unpack_from(">i", self.d, self.p)[0]
        self.p += 4
        return v

    def fixed(self):
        return self.s32() / 65536.0

    def rect(self):
        t = self.s16()
        l = self.s16()
        b = self.s16()
        r = self.s16()
        return (t, l, b, r)

    def raw(self, n):
        v = self.d[self.p:self.p + n]
        if len(v) != n:
            raise EOFError("short read: wanted %d got %d at %d" % (n, len(v), self.p))
        self.p += n
        return v

    def skip(self, n):
        self.p += n

    def eof(self):
        return self.p >= len(self.d)

    def align(self):
        """Version 2 pictures word-align every opcode."""
        if self.p & 1:
            self.p += 1


def rect_wh(r):
    return (r[3] - r[1], r[2] - r[0])


# --------------------------------------------------------------------------
# Opcode table.  Value is either an int (fixed data length in bytes) or a
# string naming a variable-length handler.
# --------------------------------------------------------------------------

VAR_REGION = "rgn"        # int16 size (inclusive) + size-2 bytes
VAR_POLY = "poly"         # int16 size (inclusive) + size-2 bytes
VAR_PIXPAT = "pixpat"     # PatType(2) + Pattern(8) [+ pixmap + ctab + data]
VAR_TEXT = "text"         # point(4) + count(1) + count bytes
VAR_DHTEXT = "dhtext"     # dh(1) + count(1) + count bytes
VAR_DVTEXT = "dvtext"     # dv(1) + count(1) + count bytes
VAR_DHDVTEXT = "dhdvtext"  # dh(1) + dv(1) + count(1) + count bytes
VAR_LONGCOMMENT = "longcomment"  # kind(2) + size(2) + size bytes
VAR_LEN16 = "len16"       # int16 length + length bytes
VAR_LEN32 = "len32"       # int32 length + length bytes
VAR_BITS = "bits"         # BitsRect / PackBitsRect / DirectBitsRect family

OPNAMES = {
    0x0000: "NOP", 0x0001: "Clip", 0x0002: "BkPat", 0x0003: "TxFont",
    0x0004: "TxFace", 0x0005: "TxMode", 0x0006: "SpExtra", 0x0007: "PnSize",
    0x0008: "PnMode", 0x0009: "PnPat", 0x000A: "FillPat", 0x000B: "OvSize",
    0x000C: "Origin", 0x000D: "TxSize", 0x000E: "FgColor", 0x000F: "BkColor",
    0x0010: "TxRatio", 0x0011: "VersionOp", 0x0012: "BkPixPat",
    0x0013: "PnPixPat", 0x0014: "FillPixPat", 0x0015: "PnLocHFrac",
    0x0016: "ChExtra", 0x001A: "RGBFgCol", 0x001B: "RGBBkCol",
    0x001C: "HiliteMode", 0x001D: "HiliteColor", 0x001E: "DefHilite",
    0x001F: "OpColor", 0x0020: "Line", 0x0021: "LineFrom",
    0x0022: "ShortLine", 0x0023: "ShortLineFrom", 0x0028: "LongText",
    0x0029: "DHText", 0x002A: "DVText", 0x002B: "DHDVText",
    0x002C: "fontName", 0x002D: "lineJustify", 0x002E: "glyphState",
    0x0030: "frameRect", 0x0031: "paintRect", 0x0032: "eraseRect",
    0x0033: "invertRect", 0x0034: "fillRect",
    0x0038: "frameSameRect", 0x0039: "paintSameRect", 0x003A: "eraseSameRect",
    0x003B: "invertSameRect", 0x003C: "fillSameRect",
    0x0040: "frameRRect", 0x0041: "paintRRect", 0x0042: "eraseRRect",
    0x0043: "invertRRect", 0x0044: "fillRRect",
    0x0048: "frameSameRRect", 0x0049: "paintSameRRect", 0x004A: "eraseSameRRect",
    0x004B: "invertSameRRect", 0x004C: "fillSameRRect",
    0x0050: "frameOval", 0x0051: "paintOval", 0x0052: "eraseOval",
    0x0053: "invertOval", 0x0054: "fillOval",
    0x0058: "frameSameOval", 0x0059: "paintSameOval", 0x005A: "eraseSameOval",
    0x005B: "invertSameOval", 0x005C: "fillSameOval",
    0x0060: "frameArc", 0x0061: "paintArc", 0x0062: "eraseArc",
    0x0063: "invertArc", 0x0064: "fillArc",
    0x0068: "frameSameArc", 0x0069: "paintSameArc", 0x006A: "eraseSameArc",
    0x006B: "invertSameArc", 0x006C: "fillSameArc",
    0x0070: "framePoly", 0x0071: "paintPoly", 0x0072: "erasePoly",
    0x0073: "invertPoly", 0x0074: "fillPoly",
    0x0078: "frameSamePoly", 0x0079: "paintSamePoly", 0x007A: "eraseSamePoly",
    0x007B: "invertSamePoly", 0x007C: "fillSamePoly",
    0x0080: "frameRgn", 0x0081: "paintRgn", 0x0082: "eraseRgn",
    0x0083: "invertRgn", 0x0084: "fillRgn",
    0x0088: "frameSameRgn", 0x0089: "paintSameRgn", 0x008A: "eraseSameRgn",
    0x008B: "invertSameRgn", 0x008C: "fillSameRgn",
    0x0090: "BitsRect", 0x0091: "BitsRgn",
    0x0098: "PackBitsRect", 0x0099: "PackBitsRgn",
    0x009A: "DirectBitsRect", 0x009B: "DirectBitsRgn",
    0x00A0: "ShortComment", 0x00A1: "LongComment",
    0x00FF: "OpEndPic",
    0x02FF: "Version", 0x0C00: "HeaderOp",
}


def op_datalen(op):
    """Return fixed byte length, or one of the VAR_* markers."""
    if op == 0x0000:
        return 0
    if op == 0x0001:
        return VAR_REGION
    if op in (0x0002, 0x0009, 0x000A):
        return 8
    if op == 0x0003:
        return 2
    if op == 0x0004:
        return 1
    if op == 0x0005:
        return 2
    if op == 0x0006:
        return 4
    if op == 0x0007:
        return 4
    if op == 0x0008:
        return 2
    if op == 0x000B:
        return 4
    if op == 0x000C:
        return 4
    if op == 0x000D:
        return 2
    if op in (0x000E, 0x000F):
        return 4
    if op == 0x0010:
        return 8
    if op == 0x0011:
        return 2                      # v2 header: 0x0011 0x02FF
    if op in (0x0012, 0x0013, 0x0014):
        return VAR_PIXPAT
    if op == 0x0015:
        return 2
    if op == 0x0016:
        return 2
    if 0x0017 <= op <= 0x0019:
        return 0
    if op in (0x001A, 0x001B):
        return 6
    if op == 0x001C:
        return 0
    if op == 0x001D:
        return 6
    if op == 0x001E:
        return 0
    if op == 0x001F:
        return 6
    if op == 0x0020:
        return 8
    if op == 0x0021:
        return 4
    if op == 0x0022:
        return 6
    if op == 0x0023:
        return 2
    if 0x0024 <= op <= 0x0027:
        return VAR_LEN16
    if op == 0x0028:
        return VAR_TEXT
    if op == 0x0029:
        return VAR_DHTEXT
    if op == 0x002A:
        return VAR_DVTEXT
    if op == 0x002B:
        return VAR_DHDVTEXT
    if 0x002C <= op <= 0x002F:
        return VAR_LEN16
    if 0x0030 <= op <= 0x0037:
        return 8
    if 0x0038 <= op <= 0x003F:
        return 0
    if 0x0040 <= op <= 0x0047:
        return 8
    if 0x0048 <= op <= 0x004F:
        return 0
    if 0x0050 <= op <= 0x0057:
        return 8
    if 0x0058 <= op <= 0x005F:
        return 0
    if 0x0060 <= op <= 0x0067:
        return 12
    if 0x0068 <= op <= 0x006F:
        return 4
    if 0x0070 <= op <= 0x0077:
        return VAR_POLY
    if 0x0078 <= op <= 0x007F:
        return 0
    if 0x0080 <= op <= 0x0087:
        return VAR_REGION
    if 0x0088 <= op <= 0x008F:
        return 0
    if op in (0x0090, 0x0091, 0x0098, 0x0099, 0x009A, 0x009B):
        return VAR_BITS
    if 0x0092 <= op <= 0x0097:
        return VAR_LEN16
    if 0x009C <= op <= 0x009F:
        return VAR_LEN16
    if op == 0x00A0:
        return 2
    if op == 0x00A1:
        return VAR_LONGCOMMENT
    if 0x00A2 <= op <= 0x00AF:
        return VAR_LEN16
    if 0x00B0 <= op <= 0x00CF:
        return 0
    if 0x00D0 <= op <= 0x00FE:
        return VAR_LEN32
    if op == 0x00FF:
        return 2                      # OpEndPic ("2 bytes" per Apple's table)
    if 0x0100 <= op <= 0x01FF:
        return 2
    if 0x0200 <= op <= 0x02FE:
        return 4
    if op == 0x02FF:
        return 2
    if 0x0300 <= op <= 0x0BFF:
        return 6
    if op == 0x0C00:
        return 24
    if 0x0C01 <= op <= 0x7EFF:
        return 24
    if 0x7F00 <= op <= 0x7FFF:
        return 254
    if 0x8000 <= op <= 0x80FF:
        return 0
    return VAR_LEN32                  # 0x8100..0xFFFF


# --------------------------------------------------------------------------
# PackBits
# --------------------------------------------------------------------------


def unpackbits_bytes(d, p, out_len):
    """Classic PackBits over bytes.  Returns (bytes, newpos)."""
    out = bytearray()
    while len(out) < out_len:
        n = d[p]
        p += 1
        if n < 128:
            cnt = n + 1
            out += d[p:p + cnt]
            p += cnt
        elif n > 128:
            cnt = 257 - n
            out += bytes([d[p]]) * cnt
            p += 1
        # n == 128 is a documented no-op
    return bytes(out[:out_len]), p


def unpackbits_words(d, p, out_len):
    """PackBits where the repeated/literal unit is a 16-bit word (packType 3)."""
    out = bytearray()
    while len(out) < out_len:
        n = d[p]
        p += 1
        if n < 128:
            cnt = (n + 1) * 2
            out += d[p:p + cnt]
            p += cnt
        elif n > 128:
            cnt = 257 - n
            out += d[p:p + 2] * cnt
            p += 2
    return bytes(out[:out_len]), p


# --------------------------------------------------------------------------
# PixMap / BitMap header
# --------------------------------------------------------------------------


class PixMapHdr:
    __slots__ = ("baseAddr", "rowBytes", "isPixMap", "bounds", "pmVersion",
                 "packType", "packSize", "hRes", "vRes", "pixelType",
                 "pixelSize", "cmpCount", "cmpSize", "planeBytes", "pmTable",
                 "pmReserved")

    def __repr__(self):
        if not self.isPixMap:
            return "BitMap(rowBytes=%d bounds=%s)" % (self.rowBytes, self.bounds)
        return ("PixMap(rowBytes=%d bounds=%s ver=%d packType=%d packSize=%d "
                "res=%gx%g pixelType=%d pixelSize=%d cmpCount=%d cmpSize=%d "
                "planeBytes=%d pmTable=0x%08X)"
                % (self.rowBytes, self.bounds, self.pmVersion, self.packType,
                   self.packSize, self.hRes, self.vRes, self.pixelType,
                   self.pixelSize, self.cmpCount, self.cmpSize,
                   self.planeBytes, self.pmTable))


def read_pixmap(r, has_baseaddr):
    h = PixMapHdr()
    h.baseAddr = r.u32() if has_baseaddr else None
    raw_rb = r.u16()
    h.isPixMap = bool(raw_rb & 0x8000)
    h.rowBytes = raw_rb & 0x7FFF
    h.bounds = r.rect()
    if h.isPixMap:
        h.pmVersion = r.s16()
        h.packType = r.s16()
        h.packSize = r.s32()
        h.hRes = r.fixed()
        h.vRes = r.fixed()
        h.pixelType = r.s16()
        h.pixelSize = r.s16()
        h.cmpCount = r.s16()
        h.cmpSize = r.s16()
        h.planeBytes = r.s32()
        h.pmTable = r.u32()
        h.pmReserved = r.u32()
    else:
        h.pmVersion = 0
        h.packType = 0
        h.packSize = 0
        h.hRes = h.vRes = 72.0
        h.pixelType = 0
        h.pixelSize = 1
        h.cmpCount = 1
        h.cmpSize = 1
        h.planeBytes = 0
        h.pmTable = 0
        h.pmReserved = 0
    return h


def read_colortable(r):
    ctSeed = r.u32()
    ctFlags = r.u16()
    ctSize = r.s16()
    n = ctSize + 1
    entries = []
    for _ in range(n):
        value = r.u16()
        red = r.u16()
        green = r.u16()
        blue = r.u16()
        entries.append((value, red, green, blue))
    return {"ctSeed": ctSeed, "ctFlags": ctFlags, "ctSize": ctSize,
            "entries": entries}


def ctab_to_palette(ct):
    """Return a 256-entry [(r8,g8,b8)] list indexed by pixel value.

    ctFlags bit 15 (0x8000) set  => "device" table, index == position.
    ctFlags bit 15 clear         => the 'value' field of each ColorSpec is the
                                    pixel value.  Glider PRO's PICTs use
                                    sequential values so both agree, but honour
                                    the field anyway.
    """
    pal = [(0, 0, 0)] * 256
    device = bool(ct["ctFlags"] & 0x8000)
    for i, (value, red, green, blue) in enumerate(ct["entries"]):
        idx = i if device else value
        if 0 <= idx < 256:
            pal[idx] = (red >> 8, green >> 8, blue >> 8)
    return pal


# --------------------------------------------------------------------------
# The picture walker
# --------------------------------------------------------------------------


class Bits:
    """One decoded raster op."""

    def __init__(self, op, hdr, ctab, srcRect, dstRect, mode, maskRgn, pixels):
        self.op = op
        self.hdr = hdr
        self.ctab = ctab
        self.srcRect = srcRect
        self.dstRect = dstRect
        self.mode = mode
        self.maskRgn = maskRgn
        self.pixels = pixels          # list of rows, each `rowBytes` bytes


class Picture:
    def __init__(self, data):
        self.data = data
        self.picSize = struct.unpack_from(">h", data, 0)[0]
        self.picFrame = struct.unpack_from(">4h", data, 2)
        self.version = None           # 1 or 2
        self.extended = False
        self.hdrOp = None
        self.ops = []                 # (offset, opcode, name, detail)
        self.bits = []                # Bits objects, in stream order
        self.unhandled = []           # names of drawing ops we skipped
        self.trailing = 0

    # ---- header --------------------------------------------------------
    def parse(self, verbose=False):
        r = Reader(self.data, 10)
        b = self.data
        if b[10] == 0x00 and b[11] == 0x11 and b[12] == 0x02 and b[13] == 0xFF:
            self.version = 2
            r.p = 14
            # Optional HeaderOp
            if len(b) >= 18 and struct.unpack_from(">H", b, 14)[0] == 0x0C00:
                self.hdrOp = b[16:16 + 24]
                ver = struct.unpack_from(">i", self.hdrOp, 0)[0]
                self.extended = (ver == -2)
                r.p = 16 + 24
                self.ops.append((14, 0x0C00, "HeaderOp",
                                 "extended=%s" % self.extended))
            self.ops.insert(0, (10, 0x0011, "VersionOp", "version=2"))
        elif b[10] == 0x11 and b[11] == 0x01:
            self.version = 1
            r.p = 12
            self.ops.append((10, 0x11, "VersionOp", "version=1"))
        else:
            raise ValueError("unrecognised PICT version prefix %s"
                             % b[10:14].hex())
        self._walk(r, verbose)
        return self

    # ---- opcode loop ---------------------------------------------------
    def _walk(self, r, verbose):
        v2 = (self.version == 2)
        while not r.eof():
            if v2:
                r.align()
                if r.p + 2 > len(r.d):
                    break
                off = r.p
                op = r.u16()
            else:
                off = r.p
                op = r.u8()
            name = OPNAMES.get(op, "op_0x%04X" % op)
            if op == 0x00FF or (not v2 and op == 0xFF):
                self.ops.append((off, op, "OpEndPic", ""))
                self.trailing = len(r.d) - r.p
                return
            detail = ""
            ln = op_datalen(op if v2 else self._v1op(op))
            try:
                if ln == VAR_BITS:
                    bo = self._read_bits(r, op)
                    detail = "%s src=%s dst=%s mode=%d" % (
                        bo.hdr, bo.srcRect, bo.dstRect, bo.mode)
                    self.bits.append(bo)
                elif ln == VAR_REGION:
                    sz = struct.unpack_from(">H", r.d, r.p)[0]
                    bbox = struct.unpack_from(">4h", r.d, r.p + 2)
                    detail = "rgnSize=%d bbox=%s" % (sz, bbox)
                    r.skip(max(sz, 2))
                elif ln == VAR_POLY:
                    sz = struct.unpack_from(">H", r.d, r.p)[0]
                    detail = "polySize=%d" % sz
                    r.skip(max(sz, 2))
                elif ln == VAR_PIXPAT:
                    self._read_pixpat(r)
                elif ln == VAR_TEXT:
                    pt = (r.s16(), r.s16())
                    n = r.u8()
                    s = r.raw(n)
                    detail = "at=%s %r" % (pt, s)
                elif ln == VAR_DHTEXT:
                    dh = r.u8()
                    n = r.u8()
                    detail = "dh=%d %r" % (dh, r.raw(n))
                elif ln == VAR_DVTEXT:
                    dv = r.u8()
                    n = r.u8()
                    detail = "dv=%d %r" % (dv, r.raw(n))
                elif ln == VAR_DHDVTEXT:
                    dh = r.u8()
                    dv = r.u8()
                    n = r.u8()
                    detail = "dh=%d dv=%d %r" % (dh, dv, r.raw(n))
                elif ln == VAR_LONGCOMMENT:
                    kind = r.u16()
                    n = r.u16()
                    r.skip(n)
                    detail = "kind=%d size=%d" % (kind, n)
                elif ln == VAR_LEN16:
                    n = r.u16()
                    r.skip(n)
                    detail = "len=%d" % n
                elif ln == VAR_LEN32:
                    n = r.u32()
                    r.skip(n)
                    detail = "len=%d" % n
                else:
                    blob = r.raw(ln)
                    if ln in (8, 12) and 0x0030 <= op <= 0x006F:
                        detail = "rect=%s" % (struct.unpack_from(">4h", blob, 0),)
                    elif op == 0x0001:
                        pass
                    elif blob:
                        detail = blob.hex()
            except (EOFError, struct.error, IndexError) as e:
                self.ops.append((off, op, name, "TRUNCATED: %s" % e))
                return
            self.ops.append((off, op, name, detail))
            if verbose:
                print("  %6d  0x%04X  %-16s %s" % (off, op, name, detail))

    @staticmethod
    def _v1op(op):
        """Map a 1-byte v1 opcode onto the v2 table for length lookup."""
        return op

    def _read_pixpat(self, r):
        patType = r.u16()
        r.raw(8)                       # 1-bit pattern
        if patType == 1:               # full colour pixel pattern
            hdr = read_pixmap(r, has_baseaddr=False)
            ct = read_colortable(r)
            w, h = rect_wh(hdr.bounds)
            r.skip(hdr.rowBytes * h)
            del ct, w
        elif patType == 2:             # RGB pattern
            r.raw(6)

    # ---- raster ops ----------------------------------------------------
    def _read_bits(self, r, op):
        direct = op in (0x009A, 0x009B)
        has_rgn = op in (0x0091, 0x0099, 0x009B)
        packed = op in (0x0098, 0x0099, 0x009A, 0x009B)
        hdr = read_pixmap(r, has_baseaddr=direct)
        ctab = None
        if hdr.isPixMap and not direct:
            ctab = read_colortable(r)
        srcRect = r.rect()
        dstRect = r.rect()
        mode = r.u16()
        maskRgn = None
        if has_rgn:
            sz = struct.unpack_from(">H", r.d, r.p)[0]
            maskRgn = r.raw(max(sz, 2))
        rows = self._read_rows(r, hdr, packed)
        return Bits(op, hdr, ctab, srcRect, dstRect, mode, maskRgn, rows)

    def _read_rows(self, r, hdr, packed):
        w, h = rect_wh(hdr.bounds)
        rb = hdr.rowBytes
        rows = []
        if not packed or rb < 8 or hdr.packType == 1:
            # Unpacked: rowBytes * height raw bytes.
            for _ in range(h):
                rows.append(r.raw(rb))
            return rows
        if hdr.packType == 2:
            # 32-bit "drop pad byte": 3 bytes per pixel, no compression.
            n = w * 3
            for _ in range(h):
                rows.append(r.raw(n))
            return rows
        big = rb > 250
        d = r.d
        p = r.p
        for _ in range(h):
            if big:
                n = struct.unpack_from(">H", d, p)[0]
                p += 2
            else:
                n = d[p]
                p += 1
            end = p + n
            if hdr.packType == 3:
                row, _ = unpackbits_words(d, p, rb)
            else:
                row, _ = unpackbits_bytes(d, p, rb)
            rows.append(row)
            p = end
        r.p = p
        return rows


# --------------------------------------------------------------------------
# Rasteriser (only the raster ops; vector ops are reported, not drawn)
# --------------------------------------------------------------------------

VECTOR_DRAW_OPS = set(range(0x0030, 0x0090)) | {0x0028, 0x0029, 0x002A, 0x002B}


def expand_row(row, hdr, w):
    """Return a list of `w` pixel *values* for one row."""
    ps = hdr.pixelSize
    if ps == 8:
        return list(row[:w])
    if ps == 1:
        out = []
        for i in range(w):
            byte = row[i >> 3]
            out.append((byte >> (7 - (i & 7))) & 1)
        return out
    if ps == 2:
        out = []
        for i in range(w):
            byte = row[i >> 2]
            out.append((byte >> (6 - 2 * (i & 3))) & 3)
        return out
    if ps == 4:
        out = []
        for i in range(w):
            byte = row[i >> 1]
            out.append((byte >> 4) if (i & 1) == 0 else (byte & 0x0F))
        return out
    if ps == 16:
        out = []
        for i in range(w):
            v = (row[2 * i] << 8) | row[2 * i + 1]
            out.append(v)
        return out
    if ps == 32:
        out = []
        if hdr.packType == 4 and hdr.cmpCount == 3:
            for i in range(w):
                out.append((row[i] << 16) | (row[w + i] << 8) | row[2 * w + i])
        elif hdr.packType == 4:
            for i in range(w):
                out.append((row[w + i] << 16) | (row[2 * w + i] << 8) | row[3 * w + i])
        elif hdr.packType == 2:
            for i in range(w):
                out.append((row[3 * i] << 16) | (row[3 * i + 1] << 8) | row[3 * i + 2])
        else:
            for i in range(w):
                out.append((row[4 * i + 1] << 16) | (row[4 * i + 2] << 8) | row[4 * i + 3])
        return out
    raise ValueError("pixelSize %d unsupported" % ps)


def rasterise(pic, default_palette=None):
    """Render pic's raster ops into an RGB canvas the size of picFrame.

    Returns (width, height, bytearray RGB, info dict).
    """
    t, l, b, rgt = pic.picFrame
    W, H = rgt - l, b - t
    if W <= 0 or H <= 0:
        raise ValueError("degenerate picFrame %s" % (pic.picFrame,))
    canvas = bytearray(b"\xff" * (W * H * 3))   # QuickDraw ports start white
    idxmap = [[None] * W for _ in range(H)]     # index plane, when available
    info = {"nbits": len(pic.bits), "pixelSizes": [], "packTypes": [],
            "modes": [], "ctabs": []}
    for bo in pic.bits:
        hdr = bo.hdr
        info["pixelSizes"].append(hdr.pixelSize)
        info["packTypes"].append(hdr.packType)
        info["modes"].append(bo.mode)
        pal = None
        if hdr.pixelSize <= 8:
            if bo.ctab is not None:
                pal = ctab_to_palette(bo.ctab)
                info["ctabs"].append((bo.ctab["ctSeed"], bo.ctab["ctFlags"],
                                      bo.ctab["ctSize"]))
            elif default_palette is not None:
                pal = default_palette
            else:
                n = 1 << hdr.pixelSize
                pal = [(255 - (255 * i // max(1, n - 1)),) * 3 for i in range(n)]
                pal += [(0, 0, 0)] * (256 - len(pal))
        bw, bh = rect_wh(hdr.bounds)
        sT, sL, sB, sR = bo.srcRect
        dT, dL, dB, dR = bo.dstRect
        sw, sh = sR - sL, sB - sT
        dw, dh = dR - dL, dB - dT
        if sw <= 0 or sh <= 0 or dw <= 0 or dh <= 0:
            continue
        for dy in range(dh):
            sy = sT + (dy * sh) // dh
            ry = sy - hdr.bounds[0]
            if ry < 0 or ry >= bh or ry >= len(bo.pixels):
                continue
            vals = expand_row(bo.pixels[ry], hdr, bw)
            cy = dT + dy - t
            if cy < 0 or cy >= H:
                continue
            for dx in range(dw):
                sx = sL + (dx * sw) // dw
                rx = sx - hdr.bounds[1]
                if rx < 0 or rx >= bw:
                    continue
                cx = dL + dx - l
                if cx < 0 or cx >= W:
                    continue
                v = vals[rx]
                if hdr.pixelSize <= 8:
                    rgb = pal[v & 0xFF]
                    idxmap[cy][cx] = v
                elif hdr.pixelSize == 16:
                    r5 = (v >> 10) & 0x1F
                    g5 = (v >> 5) & 0x1F
                    b5 = v & 0x1F
                    rgb = (r5 * 255 // 31, g5 * 255 // 31, b5 * 255 // 31)
                else:
                    rgb = ((v >> 16) & 0xFF, (v >> 8) & 0xFF, v & 0xFF)
                o = (cy * W + cx) * 3
                canvas[o] = rgb[0]
                canvas[o + 1] = rgb[1]
                canvas[o + 2] = rgb[2]
    for off, op, name, detail in pic.ops:
        if op in VECTOR_DRAW_OPS:
            pic.unhandled.append(name)
    info["unhandled"] = sorted(set(pic.unhandled))
    info["idxmap"] = idxmap
    return W, H, canvas, info


# --------------------------------------------------------------------------
# Minimal PNG writer (stdlib only)
# --------------------------------------------------------------------------


def _chunk(tag, payload):
    return (struct.pack(">I", len(payload)) + tag + payload
            + struct.pack(">I", zlib.crc32(tag + payload) & 0xFFFFFFFF))


def write_png_rgb(path, w, h, rgb):
    raw = bytearray()
    for y in range(h):
        raw.append(0)
        raw += rgb[y * w * 3:(y + 1) * w * 3]
    with open(path, "wb") as f:
        f.write(b"\x89PNG\r\n\x1a\n")
        f.write(_chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)))
        f.write(_chunk(b"IDAT", zlib.compress(bytes(raw), 9)))
        f.write(_chunk(b"IEND", b""))


def write_png_indexed(path, w, h, idxmap, palette):
    raw = bytearray()
    for y in range(h):
        raw.append(0)
        for x in range(w):
            v = idxmap[y][x]
            raw.append(0 if v is None else (v & 0xFF))
    plte = bytearray()
    for (r, g, b) in palette:
        plte += bytes((r, g, b))
    with open(path, "wb") as f:
        f.write(b"\x89PNG\r\n\x1a\n")
        f.write(_chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 3, 0, 0, 0)))
        f.write(_chunk(b"PLTE", bytes(plte)))
        f.write(_chunk(b"IDAT", zlib.compress(bytes(raw), 9)))
        f.write(_chunk(b"IEND", b""))


# --------------------------------------------------------------------------
# clut resource
# --------------------------------------------------------------------------


def parse_clut(data):
    ctSeed, ctFlags, ctSize = struct.unpack_from(">IHh", data, 0)
    entries = []
    for i in range(ctSize + 1):
        value, r, g, b = struct.unpack_from(">4H", data, 8 + i * 8)
        entries.append((value, r, g, b))
    return {"ctSeed": ctSeed, "ctFlags": ctFlags, "ctSize": ctSize,
            "entries": entries, "consumed": 8 + (ctSize + 1) * 8,
            "len": len(data)}


# --------------------------------------------------------------------------
# CLI
# --------------------------------------------------------------------------


def cmd_header(paths):
    print("%-14s %8s %8s %-24s %-12s %s"
          % ("file", "reslen", "picSize", "picFrame(t,l,b,r)", "prefix", "ver"))
    for p in paths:
        d = open(p, "rb").read()
        ps = struct.unpack_from(">h", d, 0)[0]
        fr = struct.unpack_from(">4h", d, 2)
        pre = d[10:14].hex()
        if d[10:14] == b"\x00\x11\x02\xff":
            v = "2"
            if len(d) >= 18 and d[14:16] == b"\x0c\x00":
                ext = struct.unpack_from(">i", d, 16)[0]
                v = "2ext" if ext == -2 else "2hdr"
        elif d[10] == 0x11 and d[11] == 0x01:
            v = "1"
        else:
            v = "?"
        print("%-14s %8d %8d %-24s %-12s %s"
              % (os.path.basename(p), len(d), ps, str(fr), pre, v))


def cmd_opcodes(path):
    d = open(path, "rb").read()
    pic = Picture(d)
    print("reslen=%d picSize=%d picFrame=%s"
          % (len(d), pic.picSize, pic.picFrame))
    pic.parse(verbose=False)
    print("version=%d extended=%s trailing=%d"
          % (pic.version, pic.extended, pic.trailing))
    for off, op, name, detail in pic.ops:
        print("  %6d  0x%04X  %-16s %s" % (off, op, name, detail))


def cmd_histogram(dirpath):
    from collections import Counter
    per_op = Counter()
    per_op_files = {}
    vers = Counter()
    fails = []
    files = sorted(f for f in os.listdir(dirpath) if f.endswith(".bin"))
    for f in files:
        p = os.path.join(dirpath, f)
        d = open(p, "rb").read()
        try:
            pic = Picture(d).parse()
        except Exception as e:
            fails.append((f, repr(e)))
            continue
        vers["v%d%s" % (pic.version, "ext" if pic.extended else "")] += 1
        seen = set()
        for off, op, name, detail in pic.ops:
            per_op[(op, name)] += 1
            seen.add((op, name))
        for k in seen:
            per_op_files.setdefault(k, []).append(f[:-4])
    print("files=%d  versions=%s" % (len(files), dict(vers)))
    print()
    print("%-8s %-16s %6s %6s  %s" % ("opcode", "name", "count", "files", "example ids"))
    for (op, name), c in sorted(per_op.items()):
        ids = per_op_files.get((op, name), [])
        ex = ",".join(ids[:6]) + (" ..." if len(ids) > 6 else "")
        print("0x%04X   %-16s %6d %6d  %s" % (op, name, c, len(ids), ex))
    if fails:
        print("\nFAILURES:")
        for f, e in fails:
            print("  %s: %s" % (f, e))


def cmd_pixmaps(dirpath):
    files = sorted((int(f[:-4]), f) for f in os.listdir(dirpath) if f.endswith(".bin"))
    print("%-7s %-4s %-8s %-22s %-4s %-4s %-3s %-3s %-3s %-3s %-6s %-9s %-5s %s"
          % ("id", "ver", "opcode", "bounds(t,l,b,r)", "rB", "pack", "pxT",
             "pxS", "cC", "cS", "mode", "ctSeed", "ctSz", "ctFlags"))
    for _, f in files:
        p = os.path.join(dirpath, f)
        d = open(p, "rb").read()
        try:
            pic = Picture(d).parse()
        except Exception as e:
            print("%-7s PARSE FAIL %s" % (f[:-4], e))
            continue
        if not pic.bits:
            print("%-7s %-4d (no raster ops; ops=%s)"
                  % (f[:-4], pic.version,
                     ",".join(sorted({n for _, _, n, _ in pic.ops}))))
            continue
        for bo in pic.bits:
            h = bo.hdr
            ct = bo.ctab
            print("%-7s %-4d 0x%04X   %-22s %-4d %-4d %-3d %-3d %-3d %-3d %-6d %-9s %-5s %s"
                  % (f[:-4], pic.version, bo.op, str(h.bounds), h.rowBytes,
                     h.packType, h.pixelType, h.pixelSize, h.cmpCount,
                     h.cmpSize, bo.mode,
                     ("0x%08X" % ct["ctSeed"]) if ct else "-",
                     (str(ct["ctSize"]) if ct else "-"),
                     ("0x%04X" % ct["ctFlags"]) if ct else "-"))


def cmd_ctabs(dirpath):
    """Group PICTs by the hash of their embedded ColorTable."""
    import hashlib
    groups = {}
    files = sorted((int(f[:-4]), f) for f in os.listdir(dirpath) if f.endswith(".bin"))
    for _, f in files:
        d = open(os.path.join(dirpath, f), "rb").read()
        try:
            pic = Picture(d).parse()
        except Exception:
            continue
        for bo in pic.bits:
            if bo.ctab is None:
                key = ("none", bo.hdr.pixelSize)
            else:
                pal = ctab_to_palette(bo.ctab)
                hh = hashlib.sha256(
                    b"".join(bytes(c) for c in pal)).hexdigest()[:16]
                key = (hh, bo.hdr.pixelSize, bo.ctab["ctSize"],
                       bo.ctab["ctFlags"])
            groups.setdefault(key, []).append(f[:-4])
    for k, v in sorted(groups.items(), key=lambda kv: -len(kv[1])):
        print("%s  n=%d  %s" % (k, len(v), ",".join(v[:20])
                                + (" ..." if len(v) > 20 else "")))


def cmd_decode(path, out, clutpath=None):
    d = open(path, "rb").read()
    pic = Picture(d).parse()
    default_pal = None
    if clutpath:
        ct = parse_clut(open(clutpath, "rb").read())
        default_pal = ctab_to_palette(ct)
    W, H, rgb, info = rasterise(pic, default_pal)
    write_png_rgb(out, W, H, rgb)
    print("%s -> %s  %dx%d  version=%d ext=%s bits=%d pixelSizes=%s "
          "packTypes=%s modes=%s ctabs=%s unhandled=%s trailing=%d"
          % (os.path.basename(path), out, W, H, pic.version, pic.extended,
             info["nbits"], info["pixelSizes"], info["packTypes"],
             info["modes"], info["ctabs"], info["unhandled"], pic.trailing))


def cmd_decodeall(dirpath, outdir, clutpath=None):
    os.makedirs(outdir, exist_ok=True)
    default_pal = None
    if clutpath:
        default_pal = ctab_to_palette(parse_clut(open(clutpath, "rb").read()))
    files = sorted((int(f[:-4]), f) for f in os.listdir(dirpath) if f.endswith(".bin"))
    ok = 0
    for _, f in files:
        p = os.path.join(dirpath, f)
        d = open(p, "rb").read()
        try:
            pic = Picture(d).parse()
            W, H, rgb, info = rasterise(pic, default_pal)
            write_png_rgb(os.path.join(outdir, f[:-4] + ".png"), W, H, rgb)
            ok += 1
            print("OK   %-7s %4dx%-4d v%d bits=%d px=%s pack=%s trailing=%d %s"
                  % (f[:-4], W, H, pic.version, info["nbits"],
                     info["pixelSizes"], info["packTypes"], pic.trailing,
                     ("UNHANDLED:" + ",".join(info["unhandled"]))
                     if info["unhandled"] else ""))
        except Exception as e:
            print("FAIL %-7s %r" % (f[:-4], e))
    print("decoded %d/%d" % (ok, len(files)))


def cmd_clut(path):
    ct = parse_clut(open(path, "rb").read())
    print("len=%d consumed=%d ctSeed=0x%08X ctFlags=0x%04X ctSize=%d (%d entries)"
          % (ct["len"], ct["consumed"], ct["ctSeed"], ct["ctFlags"],
             ct["ctSize"], ct["ctSize"] + 1))
    seq = all(e[0] == i for i, e in enumerate(ct["entries"]))
    hilo = all((e[1] >> 8) == (e[1] & 0xFF) and (e[2] >> 8) == (e[2] & 0xFF)
               and (e[3] >> 8) == (e[3] & 0xFF) for e in ct["entries"])
    print("value fields sequential 0..n: %s ; every component hi==lo: %s"
          % (seq, hilo))
    for i, (value, r, g, b) in enumerate(ct["entries"]):
        print("%3d  value=%3d  %04X %04X %04X  #%02X%02X%02X"
              % (i, value, r, g, b, r >> 8, g >> 8, b >> 8))


def main(argv):
    if len(argv) < 2:
        print(__doc__)
        return 2
    cmd = argv[1]
    if cmd == "header":
        cmd_header(argv[2:])
    elif cmd == "opcodes":
        cmd_opcodes(argv[2])
    elif cmd == "histogram":
        cmd_histogram(argv[2])
    elif cmd == "pixmaps":
        cmd_pixmaps(argv[2])
    elif cmd == "ctabs":
        cmd_ctabs(argv[2])
    elif cmd == "decode":
        cmd_decode(argv[2], argv[3], argv[4] if len(argv) > 4 else None)
    elif cmd == "decodeall":
        cmd_decodeall(argv[2], argv[3], argv[4] if len(argv) > 4 else None)
    elif cmd == "clut":
        cmd_clut(argv[2])
    else:
        print(__doc__)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
