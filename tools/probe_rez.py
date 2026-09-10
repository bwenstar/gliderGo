#!/usr/bin/env python3
"""probe_rez.py -- parse a DeRez'ed Rez text dump and inventory / extract resources.

Target file: GliderPRO/Glider PRO.r  (15,475,666 bytes, 199,843 LF-terminated lines)

The dump uses exactly one Rez construct, the raw-data declaration:

    data 'TYPE' (ID [, "name"] [, attr[, attr...]]) {
        $"48656C 6C6F"                       /* Hello */
        $"..."
    };

Grammar actually observed in "Glider PRO.r" (see docs/analysis/resource-fork.md):
  * the keyword is always `data` in column 0 (no `resource` / `#include` / `read`
    statements anywhere in the file),
  * TYPE is a 4-character resource type between single quotes; the space in
    'snd ' is significant,
  * ID is a non-negative decimal integer (no negative IDs, no hex IDs),
  * the optional name is a double-quoted MacRoman string.  It MAY contain
    parentheses -- e.g. 'WDEF' (129, "Infinity Windoid 2.6 (grow)", purgeable) --
    so you cannot parse the header with a  \\([^)]*\\)  regex,
  * the only attribute keyword that appears is `purgeable`,
  * every body line is  TAB $" hexpairs " [ optional  /* comment */ ],
    hex digits upper-case, grouped in fours (2 bytes) separated by single
    spaces; the last group of a line may be 1 byte (2 nibbles).  Nibble count
    per line is always even.
  * the block terminator is  `};`  in column 0.
  * no resource in this file has an empty body.

Comments (/* ... */) are pure DeRez annotation: the MacRoman rendering of the
bytes on that line, with unprintables replaced by '.'.  They carry no data and
must be discarded (they can themselves contain `$"`, quotes and braces).

Usage:
    probe_rez.py inventory  [<file>]        # type -> count, id ranges
    probe_rez.py ids <TYPE> [<file>]        # every id of one type, with size+name
    probe_rez.py sizes [<file>]             # per-resource sizes, sorted
    probe_rez.py dump <TYPE> <ID> [<file>]  # hexdump one resource to stdout
    probe_rez.py extract <outdir> [<file>]  # write every resource as a raw file
    probe_rez.py verify [<file>]            # self-checks on the grammar
"""

import os
import re
import sys
from collections import OrderedDict

DEFAULT_REZ = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    "GliderPRO", "Glider PRO.r")

# ---------------------------------------------------------------------------
# lexing
# ---------------------------------------------------------------------------

# `data 'TYPE' (` prefix.  Anchored at start of line.
_HDR = re.compile(rb"^data '(....)' \(")
# a body line: leading TAB, $"...", optional trailing comment
_HEX = re.compile(rb'^\t\$"([0-9A-Fa-f ]*)"')


class Resource(object):
    __slots__ = ("type", "id", "name", "attrs", "data", "line")

    def __init__(self, rtype, rid, name, attrs, data, line):
        self.type = rtype        # bytes, exactly 4 chars, e.g. b'snd '
        self.id = rid            # int
        self.name = name         # bytes or None (MacRoman, undecoded)
        self.attrs = attrs       # list of bytes, e.g. [b'purgeable']
        self.data = data         # bytes, the resource body
        self.line = line         # 1-based line number of the `data` header

    def __repr__(self):
        return "<%s %d %r %d bytes>" % (
            self.type.decode("latin-1"), self.id, self.name, len(self.data))


def _parse_header(rest, lineno):
    """Parse the text after `data 'TYPE' (` up to the matching `) {`.

    Returns (id, name_or_None, [attrs]).  Hand-rolled because a quoted name may
    contain ')' and ','.
    """
    i = 0
    n = len(rest)
    fields = []
    cur = bytearray()
    depth = 1
    while i < n:
        c = rest[i:i + 1]
        if c == b'"':
            # consume a quoted string, honouring \" and \\ escapes
            cur += c
            i += 1
            while i < n:
                d = rest[i:i + 1]
                cur += d
                i += 1
                if d == b"\\":
                    cur += rest[i:i + 1]
                    i += 1
                elif d == b'"':
                    break
            continue
        if c == b")":
            depth -= 1
            if depth == 0:
                fields.append(bytes(cur))
                break
            cur += c
        elif c == b"(":
            depth += 1
            cur += c
        elif c == b",":
            fields.append(bytes(cur))
            cur = bytearray()
        else:
            cur += c
        i += 1
    else:
        raise ValueError("line %d: unterminated resource header" % lineno)

    fields = [f.strip() for f in fields]
    if not fields:
        raise ValueError("line %d: empty resource header" % lineno)
    rid = int(fields[0], 0)
    name = None
    attrs = []
    for f in fields[1:]:
        if f.startswith(b'"'):
            name = f[1:-1]
            # Rez string escapes that DeRez can emit
            name = name.replace(b'\\"', b'"').replace(b"\\\\", b"\\")
        else:
            attrs.append(f)
    return rid, name, attrs


def parse(path=DEFAULT_REZ):
    """Yield Resource objects in file order.

    Read in binary: the file is MacRoman, and it contains 0x85 bytes (MacRoman
    'a-grave' / Unicode NEL) that some tools mistake for line terminators.
    """
    with open(path, "rb") as fp:
        lineno = 0
        cur = None
        body = None
        for raw in fp:
            lineno += 1
            line = raw.rstrip(b"\r\n")
            if not line:
                continue
            if line.startswith(b"data "):
                if cur is not None:
                    raise ValueError("line %d: nested data block" % lineno)
                m = _HDR.match(line)
                if not m:
                    raise ValueError("line %d: bad data header %r" % (lineno, line))
                rtype = m.group(1)
                rid, name, attrs = _parse_header(line[m.end():], lineno)
                cur = (rtype, rid, name, attrs, lineno)
                body = bytearray()
                continue
            if line.startswith(b"};"):
                if cur is None:
                    raise ValueError("line %d: stray `};`" % lineno)
                yield Resource(cur[0], cur[1], cur[2], cur[3], bytes(body), cur[4])
                cur = None
                body = None
                continue
            m = _HEX.match(line)
            if m:
                if cur is None:
                    raise ValueError("line %d: hex outside data block" % lineno)
                hx = m.group(1).replace(b" ", b"")
                if len(hx) % 2:
                    raise ValueError("line %d: odd nibble count" % lineno)
                body += bytes.fromhex(hx.decode("ascii"))
                continue
            raise ValueError("line %d: unrecognised syntax %r" % (lineno, line[:60]))
        if cur is not None:
            raise ValueError("EOF inside data block started at line %d" % cur[4])


# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------

def ranges(ids):
    """Compress a sorted id list into 'a-b, c, d-e' form."""
    out = []
    ids = sorted(ids)
    i = 0
    while i < len(ids):
        j = i
        while j + 1 < len(ids) and ids[j + 1] == ids[j] + 1:
            j += 1
        out.append(str(ids[i]) if i == j else "%d-%d" % (ids[i], ids[j]))
        i = j + 1
    return ", ".join(out)


def load(path):
    by_type = OrderedDict()
    for r in parse(path):
        by_type.setdefault(r.type, []).append(r)
    return by_type


# ---------------------------------------------------------------------------
# commands
# ---------------------------------------------------------------------------

def cmd_inventory(path):
    by_type = load(path)
    total = 0
    tbytes = 0
    print("%-6s %5s %10s  %s" % ("type", "count", "bytes", "ids"))
    for t in sorted(by_type, key=lambda k: (-len(by_type[k]), k)):
        rs = by_type[t]
        nb = sum(len(r.data) for r in rs)
        total += len(rs)
        tbytes += nb
        print("%-6s %5d %10d  %s" % ("'%s'" % t.decode("latin-1"), len(rs), nb,
                                     ranges([r.id for r in rs])))
    print("-" * 60)
    print("%-6s %5d %10d  (%d types)" % ("TOTAL", total, tbytes, len(by_type)))


def cmd_ids(path, want):
    want = want.encode("latin-1")
    want = (want + b"    ")[:4]
    for r in parse(path):
        if r.type == want:
            print("%-6s %6d %9d  line %-7d %s%s" % (
                r.type.decode("latin-1"), r.id, len(r.data), r.line,
                ("" if r.name is None else r.name.decode("latin-1")),
                (" [" + b",".join(r.attrs).decode("latin-1") + "]") if r.attrs else ""))


def cmd_sizes(path):
    rs = sorted(parse(path), key=lambda r: -len(r.data))
    for r in rs:
        print("%9d  '%s' %6d  %s" % (len(r.data), r.type.decode("latin-1"), r.id,
                                     "" if r.name is None else r.name.decode("latin-1")))


def cmd_dump(path, want, rid):
    want = (want.encode("latin-1") + b"    ")[:4]
    rid = int(rid, 0)
    for r in parse(path):
        if r.type == want and r.id == rid:
            print("'%s' %d  %d bytes  name=%r attrs=%r" % (
                r.type.decode("latin-1"), r.id, len(r.data), r.name, r.attrs))
            d = r.data
            for off in range(0, len(d), 16):
                chunk = d[off:off + 16]
                hexs = " ".join("%02X" % b for b in chunk)
                txt = "".join(chr(b) if 32 <= b < 127 else "." for b in chunk)
                print("%06X  %-47s  |%s|" % (off, hexs, txt))
            return
    sys.exit("not found")


def cmd_extract(path, outdir):
    n = 0
    for r in parse(path):
        t = r.type.decode("latin-1").replace(" ", "_").replace("/", "_")
        d = os.path.join(outdir, t)
        os.makedirs(d, exist_ok=True)
        with open(os.path.join(d, "%d.bin" % r.id), "wb") as fp:
            fp.write(r.data)
        n += 1
    print("wrote %d resources to %s" % (n, outdir))


def cmd_verify(path):
    """Grammar self-checks; every assertion here held on the shipped file."""
    seen = set()
    types = set()
    attrs = set()
    named = 0
    nres = 0
    minid = 1 << 30
    maxid = -(1 << 30)
    for r in parse(path):
        nres += 1
        key = (r.type, r.id)
        assert key not in seen, "duplicate %r" % (key,)
        seen.add(key)
        assert len(r.type) == 4
        assert len(r.data) > 0, "empty resource %r" % (key,)
        types.add(r.type)
        attrs.update(r.attrs)
        if r.name is not None:
            named += 1
        minid = min(minid, r.id)
        maxid = max(maxid, r.id)
    print("resources           : %d" % nres)
    print("distinct types      : %d" % len(types))
    print("named resources     : %d" % named)
    print("attribute keywords  : %s" % sorted(a.decode() for a in attrs))
    print("id range            : %d .. %d" % (minid, maxid))
    print("all (type,id) unique: yes")
    print("empty resources     : none")


def main(argv):
    if len(argv) < 2:
        sys.exit(__doc__)
    cmd = argv[1]
    if cmd == "inventory":
        cmd_inventory(argv[2] if len(argv) > 2 else DEFAULT_REZ)
    elif cmd == "ids":
        cmd_ids(argv[3] if len(argv) > 3 else DEFAULT_REZ, argv[2])
    elif cmd == "sizes":
        cmd_sizes(argv[2] if len(argv) > 2 else DEFAULT_REZ)
    elif cmd == "dump":
        cmd_dump(argv[4] if len(argv) > 4 else DEFAULT_REZ, argv[2], argv[3])
    elif cmd == "extract":
        cmd_extract(argv[3] if len(argv) > 3 else DEFAULT_REZ, argv[2])
    elif cmd == "verify":
        cmd_verify(argv[2] if len(argv) > 2 else DEFAULT_REZ)
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main(sys.argv)
