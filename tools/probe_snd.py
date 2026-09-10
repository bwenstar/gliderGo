#!/usr/bin/env python3
"""probe_snd.py -- decode Macintosh 'snd ' resources out of Glider PRO.

Companion to docs/analysis/audio.md.  Everything this script prints was used to
write that document; if the two ever disagree, re-run the script -- it reads the
shipped resources, the document does not.

Two sources of 'snd ' resources exist in this tree:

  1. GliderPRO/Glider PRO.r      -- the DeRez text dump of the application's
                                    resource fork.  70 'snd ' resources:
                                    ids 1000..1062 (sound effects, 63 of them)
                                    and 2000..2006 (music, 7 of them).
  2. GliderPRO/Houses/*.binhex   -- BinHex 4.0 wrapped house files.  Their
                                    resource forks carry the optional custom
                                    "sound trigger" sounds, ids >= 3000.

Resource layout that is decoded here (all fields big-endian):

    'snd ' format 1 header
      +0   int16   format            (always 1 in this game)
      +2   int16   numSynths
      then numSynths x { int16 synthID; uint32 initOption }
      then int16   numCmds
      then numCmds x { uint16 cmd; int16 param1; int32 param2 }   (8 bytes each)

    SoundHeader (encode 0x00, "stdSH"), 22 bytes + samples
      +0   uint32  samplePtr   (0 => samples follow this header)
      +4   uint32  length      (sample frames, 1 byte each)
      +8   uint32  sampleRate  (16.16 unsigned fixed point, Hz)
      +12  uint32  loopStart
      +16  uint32  loopEnd
      +20  uint8   encode      0x00 stdSH / 0xFE cmpSH / 0xFF extSH
      +21  uint8   baseFrequency  (MIDI note; 60 = middle C)
      +22  uint8   sampleArea[length]      unsigned 8-bit PCM, 0x80 = silence

    CmpSoundHeader (encode 0xFE, "cmpSH"), 64 bytes + packets
      +0..+21 as above, then
      +22  uint32  numFrames        (number of *packets* when compressed)
      +26  ext80   AIFFSampleRate   (80-bit IEEE extended)
      +36  uint32  markerChunk
      +40  OSType  format
      +44  uint32  futureUse2
      +48  uint32  stateVars
      +52  uint32  leftOverSamples
      +56  int16   compressionID    3 = MACE 3:1, 4 = MACE 6:1
      +58  uint16  packetSize       bits per packet (16 for MACE3, 8 for MACE6)
      +60  uint16  snthID
      +62  uint16  sampleSize
      +64  uint8   sampleArea[]

Usage:
    probe_snd.py show [ID]            full decode of one app resource (default 1013)
    probe_snd.py list                 table of all 70 app 'snd ' resources
    probe_snd.py houses               all 'snd ' resources in GliderPRO/Houses/*
    probe_snd.py extract <outdir>     raw PCM + manifest.tsv for the Go port
    probe_snd.py wav <ID> <out.wav>   optional: WAV, only for listening by ear
"""

import os
import struct
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
REZ = os.path.join(ROOT, "GliderPRO", "Glider PRO.r")
HOUSES = os.path.join(ROOT, "GliderPRO", "Houses")

sys.path.insert(0, HERE)
import probe_rez                                            # noqa: E402
import probe_house                                          # noqa: E402

# ---------------------------------------------------------------- constants
# Sound.h command numbers and flags (Inside Macintosh: Sound).
NULL_CMD = 0
QUIET_CMD = 3
FLUSH_CMD = 4
SOUND_CMD = 80
BUFFER_CMD = 81
CALLBACK_CMD = 13
DATA_OFFSET_FLAG = 0x8000

SYNTHS = {1: "squareWaveSynth", 3: "waveTableSynth", 5: "sampledSynth"}
ENCODES = {0x00: "stdSH", 0xFE: "cmpSH", 0xFF: "extSH"}
COMPRESSIONS = {0: "notCompressed", -1: "fixedCompression",
                -2: "variableCompression", 1: "twoToOne", 2: "eightToThree",
                3: "threeToOne (MACE 3:1)", 4: "sixToOne (MACE 6:1)"}

# The rate the whole game was sampled at: Fixed 0x56EE8BA3.
RATE_22K = 0x56EE8BA3


def fixed_to_hz(v):
    """UnsignedFixed (16.16) -> float Hz."""
    return v / 65536.0


def ext80_to_float(b):
    """80-bit IEEE extended (as used by AIFF) -> float."""
    e = struct.unpack_from(">H", b, 0)[0]
    m = struct.unpack_from(">Q", b, 2)[0]
    sign = -1.0 if e & 0x8000 else 1.0
    e &= 0x7FFF
    if e == 0 and m == 0:
        return 0.0
    return sign * m * 2.0 ** (e - 16383 - 63)


# ------------------------------------------------------------------ parsing
class SndHeader(object):
    """The parsed 'snd ' format-1 wrapper."""

    def __init__(self, blob):
        self.blob = blob
        if len(blob) < 6:
            raise ValueError("too short to be a 'snd ' resource")
        self.format, self.num_synths = struct.unpack_from(">hh", blob, 0)
        if self.format != 1:
            # format 2 is { int16 format; int16 refCount; int16 numCmds; cmds }
            self.ref_count, self.num_cmds = struct.unpack_from(">hh", blob, 2)
            self.synths = []
            off = 6
        else:
            off = 4
            self.synths = []
            for _ in range(self.num_synths):
                sid, init = struct.unpack_from(">hI", blob, off)
                self.synths.append((sid, init))
                off += 6
            self.num_cmds = struct.unpack_from(">h", blob, off)[0]
            off += 2
        self.cmds = []
        for _ in range(self.num_cmds):
            cmd, p1, p2 = struct.unpack_from(">Hhi", blob, off)
            self.cmds.append((cmd, p1, p2))
            off += 8
        self.cmd_end = off

    def sound_header_offset(self):
        """Byte offset of the sampled-sound header the commands point at."""
        for cmd, _p1, p2 in self.cmds:
            if cmd & ~DATA_OFFSET_FLAG in (BUFFER_CMD, SOUND_CMD):
                if cmd & DATA_OFFSET_FLAG:
                    return p2                # offset from start of resource
                raise ValueError("bufferCmd without dataOffsetFlag: pointer, "
                                 "not offset -- unusable from a file")
        raise ValueError("no bufferCmd/soundCmd found")


class SampledSound(object):
    """The parsed SoundHeader / CmpSoundHeader plus its sample bytes."""

    def __init__(self, blob, off):
        self.off = off
        (self.sample_ptr, self.length, self.rate_fixed,
         self.loop_start, self.loop_end) = struct.unpack_from(">IIIII", blob, off)
        self.encode = blob[off + 20]
        self.base_frequency = blob[off + 21]
        self.rate_hz = fixed_to_hz(self.rate_fixed)
        self.cmp = None
        if self.encode == 0x00:                       # stdSH
            self.header_size = 22
            self.frames = self.length
            self.bits = 8
            self.channels = 1
            self.compression_id = 0
            self.packet_size = 8
            self.samples_per_packet = 1
        elif self.encode == 0xFE:                     # cmpSH
            self.header_size = 64
            self.channels = self.length               # +4 is numChannels here
            self.frames = struct.unpack_from(">I", blob, off + 22)[0]
            self.aiff_rate = ext80_to_float(blob[off + 26:off + 36])
            (self.marker_chunk, self.fmt, self.future_use2,
             self.state_vars, self.left_over) = struct.unpack_from(">IIIII", blob, off + 36)
            (self.compression_id, self.packet_size,
             self.snth_id, self.bits) = struct.unpack_from(">hHHH", blob, off + 56)
            # MACE always emits 6 samples per packet; 3:1 uses 2-byte packets,
            # 6:1 uses 1-byte packets.
            self.samples_per_packet = 6
        elif self.encode == 0xFF:                     # extSH
            self.header_size = 64
            self.channels = self.length
            self.frames = struct.unpack_from(">I", blob, off + 22)[0]
            self.aiff_rate = ext80_to_float(blob[off + 26:off + 36])
            self.bits = struct.unpack_from(">H", blob, off + 48)[0]
            self.compression_id = 0
            self.packet_size = self.bits
            self.samples_per_packet = 1
        else:
            raise ValueError("unknown encode byte 0x%02X" % self.encode)
        self.data_offset = off + self.header_size
        self.data = blob[self.data_offset:]

    @property
    def is_compressed(self):
        return self.encode == 0xFE

    def duration_s(self):
        n = self.frames * (self.samples_per_packet if self.is_compressed else 1)
        return n / self.rate_hz if self.rate_hz else 0.0

    def pcm_u8(self):
        """Unsigned 8-bit PCM exactly as stored.  None if compressed."""
        if self.is_compressed:
            return None
        return self.data[:self.length]


def parse(blob):
    hdr = SndHeader(blob)
    return hdr, SampledSound(blob, hdr.sound_header_offset())


# ------------------------------------------------------------ resource I/O
_APP_CACHE = None


def app_snds():
    """{id: (name, bytes)} for every 'snd ' in Glider PRO.r."""
    global _APP_CACHE
    if _APP_CACHE is None:
        out = {}
        for r in probe_rez.parse(REZ):
            if r.type == b"snd ":
                name = (r.name or b"").decode("mac-roman")
                out[r.id] = (name, r.data)
        _APP_CACHE = out
    return _APP_CACHE


def house_snds():
    """[(house, id, name, bytes)] for every 'snd ' in every house file."""
    out = []
    if not os.path.isdir(HOUSES):
        return out
    for fn in sorted(os.listdir(HOUSES)):
        if not fn.endswith(".binhex"):
            continue
        path = os.path.join(HOUSES, fn)
        try:
            bh = probe_house.binhex_decode(path)
            fork = probe_house.parse_resource_fork(bh["rsrc"])
        except Exception as exc:                       # noqa: BLE001
            print("!! %s: %s" % (fn, exc), file=sys.stderr)
            continue
        for rid, rname, off, ln in fork.get("snd ", []):
            out.append((fn[:-7], rid, rname, bh["rsrc"][off:off + ln]))
    return out


# -------------------------------------------------------------- formatting
def hexdump(b, base=0, limit=None):
    if limit is not None:
        b = b[:limit]
    for i in range(0, len(b), 16):
        chunk = b[i:i + 16]
        hx = " ".join("%02X" % c for c in chunk)
        asc = "".join(chr(c) if 32 <= c < 127 else "." for c in chunk)
        print("  %04X  %-47s  %s" % (base + i, hx, asc))


def cmd_show(argv):
    rid = int(argv[0]) if argv else 1013
    snds = app_snds()
    if rid not in snds:
        sys.exit("no 'snd ' %d in %s (have %s)" % (rid, REZ, sorted(snds)[:4]))
    name, blob = snds[rid]
    hdr, snd = parse(blob)
    print("resource      'snd ' %d %r" % (rid, name))
    print("resource size %d bytes" % len(blob))
    print("first 48 bytes:")
    hexdump(blob, 0, 48)
    print()
    print("format            %d" % hdr.format)
    print("numSynths         %d" % hdr.num_synths)
    for sid, init in hdr.synths:
        print("  synth           %d (%s)  initOption 0x%08X"
              % (sid, SYNTHS.get(sid, "?"), init))
    print("numCmds           %d" % hdr.num_cmds)
    for cmd, p1, p2 in hdr.cmds:
        base = cmd & ~DATA_OFFSET_FLAG
        nm = {BUFFER_CMD: "bufferCmd", SOUND_CMD: "soundCmd",
              CALLBACK_CMD: "callBackCmd", NULL_CMD: "nullCmd"}.get(base, "?")
        print("  cmd 0x%04X      %s%s param1=%d param2=%d"
              % (cmd, nm, " | dataOffsetFlag" if cmd & DATA_OFFSET_FLAG else "",
                 p1, p2))
    print()
    print("sound header at   +%d" % snd.off)
    print("  samplePtr       0x%08X %s"
          % (snd.sample_ptr, "(samples follow header)" if not snd.sample_ptr else ""))
    print("  encode          0x%02X (%s)"
          % (snd.encode, ENCODES.get(snd.encode, "?")))
    print("  sampleRate      0x%08X = %.4f Hz%s"
          % (snd.rate_fixed, snd.rate_hz,
             "  [rate22khz]" if snd.rate_fixed == RATE_22K else ""))
    print("  baseFrequency   %d" % snd.base_frequency)
    print("  loopStart       %d" % snd.loop_start)
    print("  loopEnd         %d" % snd.loop_end)
    if snd.is_compressed:
        print("  numChannels     %d" % snd.channels)
        print("  numFrames       %d packets" % snd.frames)
        print("  AIFFSampleRate  %.4f Hz" % snd.aiff_rate)
        print("  compressionID   %d (%s)"
              % (snd.compression_id,
                 COMPRESSIONS.get(snd.compression_id, "?")))
        print("  packetSize      %d bits" % snd.packet_size)
        print("  snthID          %d" % snd.snth_id)
        print("  sampleSize      %d bits" % snd.bits)
        print("  packet bytes    %d (resource size - %d)"
              % (len(snd.data), snd.data_offset))
        print("  ** compressed: MACE decode required, not implemented here **")
    else:
        print("  length          %d sample frames" % snd.length)
        print("  header+samples  %d + %d = %d (resource size %d)"
              % (snd.header_size, snd.length,
                 snd.off + snd.header_size + snd.length, len(blob)))
        pcm = snd.pcm_u8()
        print("  duration        %.4f s (%.1f ms)"
              % (snd.duration_s(), snd.duration_s() * 1000.0))
        print("  u8 PCM min/max/mean  %d / %d / %.2f  (0x80 = silence)"
              % (min(pcm), max(pcm), sum(pcm) / float(len(pcm))))
        print("  first 16 samples     %s" % list(pcm[:16]))
        print("  last 8 samples       %s" % list(pcm[-8:]))
        if snd.loop_start == 0 and snd.loop_end >= snd.length - 1:
            print("  LOOPED: loopStart 0, loopEnd %d == length-1" % snd.loop_end)
        elif snd.loop_start == snd.length - 2:
            print("  no loop (degenerate loopStart=length-2, loopEnd=length-1)")


def cmd_list(_argv):
    snds = app_snds()
    print("%-6s %-26s %8s %9s %10s %9s %6s %6s %s"
          % ("id", "name", "resbytes", "frames", "rateHz", "ms", "loopS", "loopE",
             "enc"))
    tot = 0
    for rid in sorted(snds):
        name, blob = snds[rid]
        hdr, snd = parse(blob)
        tot += len(blob)
        print("%-6d %-26s %8d %9d %10.4f %9.1f %6d %6d %s"
              % (rid, name, len(blob), snd.frames, snd.rate_hz,
                 snd.duration_s() * 1000.0, snd.loop_start, snd.loop_end,
                 ENCODES.get(snd.encode, "?")))
    print("-- %d resources, %d bytes total" % (len(snds), tot))
    sfx = [i for i in snds if i < 2000]
    mus = [i for i in snds if i >= 2000]
    print("-- sfx %d ids %d..%d  %d bytes (%d after the -20 strip)"
          % (len(sfx), min(sfx), max(sfx),
             sum(len(snds[i][1]) for i in sfx),
             sum(len(snds[i][1]) - 20 for i in sfx)))
    print("-- music %d ids %d..%d  %d bytes (%d after the -20 strip)"
          % (len(mus), min(mus), max(mus),
             sum(len(snds[i][1]) for i in mus),
             sum(len(snds[i][1]) - 20 for i in mus)))


def cmd_houses(_argv):
    print("%-22s %-6s %-32s %8s %9s %11s %8s %s"
          % ("house", "id", "name", "resbytes", "frames", "rateHz", "ms", "enc"))
    for house, rid, rname, blob in house_snds():
        try:
            hdr, snd = parse(blob)
        except Exception as exc:                       # noqa: BLE001
            print("%-22s %-6d %-32s  !! %s" % (house, rid, rname, exc))
            continue
        extra = ""
        if snd.is_compressed:
            extra = " compressionID=%d packetSize=%d" % (snd.compression_id,
                                                         snd.packet_size)
        if snd.loop_start == 0 and snd.loop_end > 1:
            extra += " LOOP 0..%d" % snd.loop_end
        elif snd.loop_end > snd.frames:
            extra += " !!loopEnd %d > frames %d" % (snd.loop_end, snd.frames)
        print("%-22s %-6d %-32s %8d %9d %11.4f %8.1f %s%s"
              % (house, rid, rname, len(blob), snd.frames, snd.rate_hz,
                 snd.duration_s() * 1000.0, ENCODES.get(snd.encode, "?"), extra))


def cmd_extract(argv):
    """Write raw PCM + a manifest -- the form the Go port should embed."""
    if not argv:
        sys.exit("usage: probe_snd.py extract <outdir>")
    outdir = argv[0]
    os.makedirs(outdir, exist_ok=True)
    man = open(os.path.join(outdir, "manifest.tsv"), "w")
    man.write("file\tid\tname\tframes\trate_fixed\trate_hz\tloop_start\t"
              "loop_end\tbase_freq\tencode\n")
    n = 0
    for rid, (name, blob) in sorted(app_snds().items()):
        hdr, snd = parse(blob)
        pcm = snd.pcm_u8()
        if pcm is None:
            print("skip %d (%s): compressed" % (rid, name), file=sys.stderr)
            continue
        fn = "snd_%d.pcm" % rid
        with open(os.path.join(outdir, fn), "wb") as fh:
            fh.write(pcm)
        man.write("%s\t%d\t%s\t%d\t0x%08X\t%.4f\t%d\t%d\t%d\t0x%02X\n"
                  % (fn, rid, name, snd.length, snd.rate_fixed, snd.rate_hz,
                     snd.loop_start, snd.loop_end, snd.base_frequency,
                     snd.encode))
        n += 1
    man.close()
    print("wrote %d .pcm files + manifest.tsv to %s" % (n, outdir))
    print("format: unsigned 8-bit mono, offset binary (0x80 = silence),")
    print("        %.4f Hz for every application resource." % fixed_to_hz(RATE_22K))


def cmd_wav(argv):
    """Only for verifying by ear; the Go port needs no WAV parser."""
    if len(argv) < 2:
        sys.exit("usage: probe_snd.py wav <ID> <out.wav>")
    rid, out = int(argv[0]), argv[1]
    name, blob = app_snds()[rid]
    hdr, snd = parse(blob)
    pcm = snd.pcm_u8()
    if pcm is None:
        sys.exit("resource %d is MACE-compressed; cannot write WAV" % rid)
    rate = int(round(snd.rate_hz))
    with open(out, "wb") as fh:
        fh.write(b"RIFF" + struct.pack("<I", 36 + len(pcm)) + b"WAVEfmt ")
        fh.write(struct.pack("<IHHIIHH", 16, 1, 1, rate, rate, 1, 8))
        fh.write(b"data" + struct.pack("<I", len(pcm)) + pcm)
    print("%s: %r, %d frames, %d Hz, %.3f s"
          % (out, name, snd.length, rate, snd.duration_s()))


COMMANDS = {"show": cmd_show, "list": cmd_list, "houses": cmd_houses,
            "extract": cmd_extract, "wav": cmd_wav}


def main(argv):
    if not argv or argv[0] not in COMMANDS:
        print(__doc__)
        return 2
    COMMANDS[argv[0]](argv[1:])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
