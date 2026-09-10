# Glider PRO 1.0.4 — the 22 shipped houses: inventory and design analysis

## Scope

This document inventories and characterises the 22 house files that ship with the
Glider PRO 1.0.4 GPL source release, and derives from them a concrete, quantitative
brief for authoring *new* houses that feel like the originals.

It covers:

1. How a house file physically reaches the running game — the BinHex 4.0 container used
   in this repository, the Mac file-system search that finds houses, `ReadHouse`, the
   Resource Manager search chain that lets a house override application art, and the
   QuickTime `.mov` sidecar (§1).
2. The exact on-disk byte layout of `houseType`, `roomType` and `objectType`, verified
   against real bytes (§2).
3. A full per-file inventory: size, version, room/floor/suite counts, object counts and
   vocabulary, backgrounds, embedded art and sound, start room, and author (§3, §4).
4. A quantitative design characterisation of the shipped corpus: room-count
   distributions, difficulty ramps measured against BFS depth from the start room,
   the object vocabulary the official houses actually stay inside, the recurring room
   motifs, the prize/hazard economy, and the connectivity idioms (§5–§8).
5. What the editor's own validator demands of a legal house, and a recipe for a new
   house in the same spirit (§9, §10).
6. Every anomaly and authoring bug found in the shipped data (§11).

Out of scope: the physics of gliders, blowers and enemies (that is the runtime spec,
not the level spec); the rendering pipeline; and the byte layout of the QuickTime movies.
Object *semantics* appear here only insofar as they explain why a level designer places
a given object.

Everything numeric below was measured, not recalled. The measurement tool is
`tools/probe_houses_inventory.py` (945 lines) and its generated table dump is
`docs/analysis/houses-inventory.md`; the per-house detail in §4 is machine-generated
from the same parse. Re-run with:

```sh
python3 tools/probe_houses_inventory.py \
    --md docs/analysis/houses-inventory.md \
    --json /tmp/houses.json
```

Line numbers in citations are for the CR→LF converted copies of the classic-Mac text
files (the `GliderPRO/Sources/*.c` and `GliderPRO/Headers/*.h` in this repository use
bare `\r` line endings; convert before grepping, and always pass `grep -a` because the
MacRoman high-bit bytes otherwise make GNU grep treat them as binary).

## Sources read

Binary/data inputs, read in full and parsed byte-for-byte:

| path | what it is |
|---|---|
| `GliderPRO/Houses/*.binhex` | the 22 shipped houses, BinHex 4.0-encoded (data fork + resource fork + Finder info) |
| `GliderPRO/Houses/*.mov` | 15 QuickTime sidecar movies (see §1.6) |
| `GliderPRO/Glider PRO.r` | 15,475,666-byte Rez text dump of the application resource fork; mined for the application's `PICT`/`snd ` ID space so house-vs-app ID collisions can be detected |
| `GliderPRO/README.md` | authorship credits (`README.md:6-15`) |
| `tools/probe_house.py` | pre-existing decoder in this repository (BinHex, resource fork, `houseType`/`roomType`/`objectType`); used as the reference parser and cross-checked against an independent re-parse |

Source files read in full (converted to LF), listed with the reason:

| path | why |
|---|---|
| `GliderPRO/Headers/GliderStructs.h` | authoritative struct layouts and per-field byte comments |
| `GliderPRO/Headers/GliderDefines.h` | every constant quoted in this document |
| `GliderPRO/Sources/HouseIO.c` | `OpenHouse`, `ReadHouse`, `WriteHouse`, `CloseHouse`, resource-fork open/close, `OpenHouseMovie` |
| `GliderPRO/Sources/House.c` | `RealRoomNumberCount`, `GetFirstRoomNumber`, `WhereDoesGliderBegin`, `HouseHasOriginalPicts`, `CountHouseLinks`, `CountRoomsVisited`, `ConvertHouseVer1To2` |
| `GliderPRO/Sources/HouseLegal.c` | the whole validator: `CheckHouseForProblems` and its twelve sub-checks |
| `GliderPRO/Sources/HouseInfo.c` | `CountTotalHousePoints` — the scoring model used for the per-house point totals here |
| `GliderPRO/Sources/Room.c` | `IsRoomAStructure`, `DetermineRoomOpenings`, `GetOriginalBounding`, `GetNumberOfLights`, `DoesRoomHaveFloor`, `DoesRoomHaveCeiling`, `IsShadowVisible` — i.e. how `background`, `bounds` and `tiles` become walls, and where `openings` is (not) used |
| `GliderPRO/Sources/RoomInfo.c` | the room editor: how `bounds` is *written*, and the 3000..3799 user-background ID rule |
| `GliderPRO/Sources/Link.c` | `MergeFloorSuite` / `ExtractFloorSuite` / `DoLink` / `DoUnlink` — the link encoding used for every transport, mailbox and remote switch |
| `GliderPRO/Sources/SelectHouse.c` | `DoDirSearch`, `BuildHouseList` — how the game finds houses on disk |
| `GliderPRO/Sources/Play.c` | `InitGlider`, `SetHouseToFirstRoom`, `SetHouseToSavedRoom` — what the house header feeds into a new game |
| `GliderPRO/Sources/Banner.c` | `CountStarsInHouse`, banner PICT IDs, the stars-remaining display |
| `GliderPRO/Sources/Main.c` | `maxFiles` default and clamp |
| `GliderPRO/Sources/ObjectInfo.c` | the editor clamps on `kCustomPict` PICT IDs and `kSoundTrigger` sound IDs |
| `GliderPRO/Sources/ObjectDraw2.c`, `ObjectDrawAll.c`, `RoomGraphics.c`, `StructuresInit2.c`, `Input.c`, `Sound.c`, `Music.c` | the application-owned resource ID ranges a house must avoid (or deliberately shadows) |

---

## 1. How a house reaches the game

### 1.1 A house is three things

A Glider PRO house is a classic Mac OS document with type `'gliH'` and creator `'ozm5'`
(`GliderPRO/Sources/House.c:85`, `FSpCreate(&theSpec, 'ozm5', 'gliH', ...)`):

| part | contents | required? |
|---|---|---|
| **data fork** | one `houseType` header (866 bytes) immediately followed by `nRooms` × `roomType` (348 bytes each), big-endian, unpadded | yes |
| **resource fork** | `PICT` (custom backgrounds ≥ 3000 and `kCustomPict` art ≥ 10000), `snd ` (sound-trigger audio ≥ 3000), `bnds` (4-byte wall descriptors for custom backgrounds), plus a Finder icon family | optional — `Empty House` and `Sampler` ship an essentially empty fork |
| **`<name>.mov` sidecar** | a QuickTime movie played inside `kTV` objects | optional — 15 of 22 have one |

There is no magic number, no length field, and no checksum in the data fork. The only
integrity check the game performs is the implicit one in `ValidateNumberOfRooms`
(`GliderPRO/Sources/HouseLegal.c:621`), which recomputes `nRooms` from the handle size.

Observed: all 22 houses report `version` = `0x0200` = `kHouseVersion`
(`GliderPRO/Headers/GliderDefines.h:517`). No shipped house is version 1, so the
`ConvertHouseVer1To2` path (`GliderPRO/Sources/House.c:746-818`) is never exercised by
the shipped corpus.

### 1.2 The BinHex 4.0 container used in this repository

The GPL release cannot store Mac resource forks in a Git tree, so each house is stored
as a single BinHex 4.0 text file. Decoding is a prerequisite for everything else in this
document, so the container is documented and verified here.

Layout of a BinHex 4.0 stream (as implemented in `tools/probe_house.py`):

1. Skip everything up to and including the literal line
   `(This file must be converted with BinHex 4.0)`.
2. Find the first `:`; the payload runs to the next `:`.
3. Strip newlines; decode the remaining characters with the 64-symbol alphabet
   `!"#$%&'()*+,-012345689@ABCDEFGHIJKLMNPQRSTUVXYZ[`abcdefhijklmpqr` (6 bits each,
   MSB-first) into a byte stream.
4. Run-length expand: byte `0x90` followed by count `n` repeats the previous byte
   `n` times; `0x90 0x00` is a literal `0x90`.
5. The expanded stream is
   `nameLen:1, name:nameLen, 0x00, type:4, creator:4, flags:2, dataLen:4, rsrcLen:4,
    hdrCRC:2, data:dataLen, dataCRC:2, rsrc:rsrcLen, rsrcCRC:2`.

The CRC is CRC-16 with polynomial `0x1021`, seed `0`, MSB-first, no final XOR and no
input/output reflection — i.e. **exactly CRC-16/XMODEM**, which is what Python exposes as
`binascii.crc_hqx(data, 0)` (its check value over `"123456789"` is `0x31C3`). The stream
is *not* augmented: the stored CRC equals the CRC of the covered bytes alone, with **no
two zero bytes appended**. (Appending them yields a different, wrong value — for
`Sampler`'s header, `0x826F` instead of the stored `0x03B7`.) Verified empirically:
**all 66 CRCs (3 per file × 22 files) match** under that definition, so the decoded forks
are byte-exact.

| # | house | Finder flags | hdr CRC | data CRC | rsrc CRC |
|---|---|---|---|---|---|
| 1 | Art Museum | `0x0500` | `0xB785` | `0xED0D` | `0xC6B0` |
| 2 | CD Demo House | `0x0500` | `0x4FC0` | `0x4C2C` | `0xD506` |
| 3 | California or Bust! | `0x0500` | `0xB043` | `0x4ED8` | `0x4D19` |
| 4 | Castle o' the Air | `0x0500` | `0x30EA` | `0x88D9` | `0x6A44` |
| 5 | Davis Station | `0x0500` | `0x5D6D` | `0x31C3` | `0x9680` |
| 6 | Demo House | `0x0500` | `0x9CF6` | `0xFF60` | `0xE588` |
| 7 | Empty House | `0x0500` | `0x4FB8` | `0x65C4` | `0x67FE` |
| 8 | Fun House | `0x0500` | `0xB08F` | `0x975D` | `0x70D2` |
| 9 | Grand Prix | `0x0500` | `0xE9B7` | `0xB4E8` | `0x90D2` |
| 10 | ImagineHouse PRO II | `0x0500` | `0xF69C` | `0x01FA` | `0xF7BD` |
| 11 | In The Mirror | **`0x0504`** | `0xFEA0` | `0xDF37` | `0xC95C` |
| 12 | Land of Illusion | `0x0500` | `0xAA82` | `0xC07D` | `0xD070` |
| 13 | Leviathan | `0x0500` | `0x1116` | `0xBE15` | `0x5634` |
| 14 | Metropolis | `0x0500` | `0x66C7` | `0x7FE8` | `0xC4D6` |
| 15 | Nemo's Market | `0x0500` | `0x6A98` | `0x29AC` | `0x6877` |
| 16 | Rainbow's End | `0x0500` | `0x5B46` | `0x53B2` | `0x46BF` |
| 17 | Sampler | **`0x0100`** | `0x03B7` | `0xA523` | `0x4B45` |
| 18 | Slumberland | `0x0500` | `0x3D27` | `0xEB5E` | `0x196C` |
| 19 | SpacePods | `0x0500` | `0x8CF8` | `0x6806` | `0x8758` |
| 20 | Teddy World | `0x0500` | `0xF273` | `0x0625` | `0xC59A` |
| 21 | The Asylum Pro | `0x0500` | `0xA6B6` | `0x6BD3` | `0x1089` |
| 22 | Titanic | `0x0500` | `0x5390` | `0x6D22` | `0x72A6` |

Every file: BinHex version byte `0x00`, type `gliH`, creator `ozm5`. The Finder flag
word is `0x0500` for 20 of 22 (`hasBundle` + `inited`); `In The Mirror` additionally has
`0x0004` set (`hasBeenInited`/custom-icon bookkeeping) and `Sampler` has only `0x0100`.
None of this matters to a port: the flags are Finder desktop-database state, not game
data.

**Port note.** A Go port does not need BinHex at all at runtime — it needs it once, to
extract the 22 houses into whatever container the port chooses (e.g. a directory holding
`house.dat` + extracted PICTs/sounds). Keep the BinHex reader in a tool, not in the game.

### 1.3 How the game finds houses

`DoDirSearch` (`GliderPRO/Sources/SelectHouse.c:558-620`) does a breadth-first walk of
the folder containing the application:

```
 1. dirStack[0] = thisMac.dirID; nDirs = 1; housesFound = 0.
 2. #define kMaxDirectories 32                                (SelectHouse.c:559)
 3. For each directory in the stack (up to kMaxDirectories):
 4.     For index = 1, 2, 3, ... call PBGetCatInfo:
 5.         if (ioFlAttrib & 0x10) == 0x10       -> it is a folder; push ioDrDirID
 6.         else if ioFlFndrInfo.fdType    == 'gliH'
 7.              and ioFlFndrInfo.fdCreator == 'ozm5'
 8.              and housesFound < maxFiles      -> record the FSSpec  (:592-594)
 9. Stop when the stack is exhausted or 32 directories have been scanned.
```

`BuildHouseList` (`GliderPRO/Sources/SelectHouse.c:650-664`) requires
`thisMac.hasSystem7`, then inserts up to `kMaxExtraHouses` = 8
(`GliderPRO/Sources/SelectHouse.c:35`) manually-added specs ahead of the scan results.
`maxFiles` comes from preferences. When a prefs file exists, an out-of-range value is
replaced by 12 (*not* clamped to the nearest bound):

```c
maxFiles = thePrefs.wasMaxFiles;                 /* Main.c:87  */
if ((maxFiles < 12) || (maxFiles > 500))         /* Main.c:88  */
    maxFiles = 12;                               /* Main.c:89  */
```

When there is no prefs file at all, the shipped default is **48**:
`maxFiles = 48; willMaxFiles = 48;` (`GliderPRO/Sources/Main.c:156-157`), which
`SetAllDefaults` also restores (`Settings.c:1225`). The `[12, 500]` clamp proper lives in
the Brains-preferences dialog, at `Settings.c:264-267`. So 48 is the effective default and
12 is only the fallback for a corrupt or out-of-range stored preference.

**Port note.** Nothing here is worth reproducing literally. A Go port should scan a
`houses/` directory for its own container extension. The only behaviour worth keeping is
that the house list is *discovered*, not hard-coded, and that a user-supplied house is a
first-class citizen: the shipped 22 are just the ones in the box.

### 1.4 `ReadHouse` — the load path, in order

`ReadHouse` (`GliderPRO/Sources/HouseIO.c:315-443`). Numbered to match the control flow;
original variable names in parentheses.

```
 1. If !houseOpen                       -> YellowAlert(kYellowUnaccounted, 2), fail.   (:321-325)
 2. If gameDirty || fileDirty:
 2a.    if houseIsReadOnly              -> WriteScoresToDisk(), else fail.             (:329-336)
 2b.    else                            -> WriteHouse(false), fail on error.           (:337-338)
 3. byteCount = GetEOF(houseRefNum).                                                   (:341)
 4. [demo build only] if byteCount != 16526 -> fail.        /* = Demo House exactly */ (:349)
 5. Dispose the old thisHouse handle; thisHouse = NewHandle(byteCount).                (:353-361)
 6. MoveHHi, SetFPos(0), HLock, FSRead the whole file into the handle.                  (:362-378)
 7. numberRooms = (*thisHouse)->nRooms.                                                (:380)
 8. [demo] if numberRooms != 45 -> fail.                                               (:382)
 9. If numberRooms < 1 || byteCount == 0:
        numberRooms = 0; noRoomAtAll = true; YellowAlert(kYellowNoRooms, 0); fail.      (:385-392)
10. wasHouseVersion = (*thisHouse)->version.                                           (:394)
11. If wasHouseVersion >= kNewHouseVersion (0x0300):
        YellowAlert(kYellowNewerVersion, 0); fail.        /* forward-compat refusal */ (:395-400)
12. houseUnlocked = (((*thisHouse)->timeStamp & 0x00000001) == 0).                     (:402)
13. [demo] if houseUnlocked -> fail.                       /* demo needs a locked house */
14. changeLockStateOfHouse = false; saveHouseLocked = false.                           (:407-408)
15. whichRoom = (*thisHouse)->firstRoom.                                               (:410)
16. [demo] if whichRoom != 0 -> fail.                                                  (:412)
17. wardBitSet        = (flags & 0x00000001) == 0x00000001.                             (:416)
18. phoneBitSet       = (flags & 0x00000002) == 0x00000002.                             (:417)
19. bannerStarCountOn = (flags & 0x00000004) == 0x00000000.   /* note: inverted */      (:418)
20. HUnlock.
21. noRoomAtAll = (RealRoomNumberCount() == 0).                                        (:422)
22. thisRoomNumber = -1; previousRoom = -1.                                            (:423-424)
23. If !noRoomAtAll -> CopyRoomToThisRoom(whichRoom).                                  (:425-426)
24. If houseIsReadOnly -> houseUnlocked = false; ReadScoresFromDisk().                 (:428-434)
25. objActive = kNoObjectSelected; ReflectCurrentRoom(true).                           (:436-437)
26. gameDirty = false; fileDirty = false; UpdateMenus(false).                          (:438-440)
```

Three properties matter for a port:

* **The whole file is slurped into one handle and used in place.** Rooms are not
  deserialised into a friendlier representation; `(*thisHouse)->rooms[n]` is direct
  pointer arithmetic over the mmap-like buffer. That is why the on-disk layout is
  exactly the in-memory C layout, big-endian and unpadded.
* **The size field is `nRooms`, and it is trusted at load time.** Nothing at load
  verifies `866 + 348*nRooms == byteCount`; only the editor's
  `ValidateNumberOfRooms` does, and only when the user asks to check the house.
* **`firstRoom` is *not* range-checked here.** The clamp lives in
  `GetFirstRoomNumber` (`GliderPRO/Sources/House.c:196-217`), which returns 0 if
  `firstRoom` is out of `[0, nRooms)` and sets `noRoomAtAll` when `nRooms <= 0`.

`WriteHouse` (`GliderPRO/Sources/HouseIO.c:448-519`) is the mirror image, and is the
origin of the two header quirks documented in §1.7:

```
1. SetFPos(0); CopyThisRoomToRoom(); if (checkIt) CheckHouseForProblems().             (:460-470)
2. byteCount = GetHandleSize(thisHouse).        /* the handle IS the file */           (:473)
3. If fileDirty:
3a.    GetDateTime(&timeStamp); timeStamp &= 0x7FFFFFFF;      /* clears bit 31 */      (:477-478)
3b.    if changeLockStateOfHouse -> houseUnlocked = !saveHouseLocked;                  (:480-481)
3c.    if houseUnlocked -> timeStamp &= 0x7FFFFFFE; else timeStamp |= 0x00000001;      (:483-486)
3d.    (*thisHouse)->timeStamp = timeStamp; (*thisHouse)->version = wasHouseVersion;   (:487-488)
4. FSWrite(byteCount); SetEOF(byteCount).                                              (:491-499)
```

### 1.5 The resource fork and the Resource Manager search chain

`OpenHouseResFork` (`GliderPRO/Sources/HouseIO.c:567-577`):

```c
if (houseResFork == -1) {
    houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm);
    if (houseResFork == -1) YellowAlert(kYellowFailedResOpen, ResError());
    else                    UseResFile(houseResFork);      /* HouseIO.c:575 */
}
```

`UseResFile` pushes the house fork to the **front** of the Resource Manager search
chain. Consequences, all of which the shipped houses exploit:

* `GetResource('PICT', id)` finds the *house's* PICT if the house defines that ID,
  otherwise the application's. So a house can silently replace any application graphic.
* `Count1Resources` counts only the current (house) fork — which is exactly how
  `HouseHasOriginalPicts` (`GliderPRO/Sources/House.c:246-252`) decides whether the
  editor's "custom background" menu item should be enabled:
  `return (Count1Resources('PICT') > 0);`, used at `GliderPRO/Sources/RoomInfo.c:399`.
* `CloseHouseResFork` (`GliderPRO/Sources/HouseIO.c:582-589`) pops it again on close.

Resource ID ranges, and who owns them:

| range | type | owner | citation |
|---|---|---|---|
| 150, 151, 153 | `PICT` | app (splash, about) | `GliderPRO/Glider PRO.r` |
| 1000–1023 | `PICT` | app UI (1000 = `kSplash8BitPICT`, 1015 = `kEscPausePictID`, 1016 = `kTabPausePictID`, 1017 = `kStarsRemainingPICT`, 1018 = `kStarRemainingPICT`, 1019 = `kAngelPictID`) | `GliderDefines.h:524`, `Input.c:17-18`, `Banner.c:20-21`, `StructuresInit2.c:20` |
| 1988–1999 | `PICT` | app (1991 = `kBannerPageBottomMask`, 1992 = `kBannerPageBottomPICT`, 1993 = `kBannerPageTopPICT`, 1995 = `kStarPictID`, 1999 = `kSupportPictID`) | `Banner.c:17-19`, `GliderDefines.h:534`, `StructuresInit2.c:21` |
| **2000–2017** | `PICT` | app — the 18 built-in room backgrounds | `GliderDefines.h:227-244`, `:519-521` |
| 3900–3927 | `PICT` | app object masks | `ObjectDraw2.c:39-66` |
| 3957–3999 | `PICT` | app object art (3957 `kManholeThruFloor`, 3964 `kBooksPictID`, 3974 `kGlider2PictID`, 3975 `kOzmaPictID`, 3976 `kGliderFoilPictID`, 3981 `kDoorExLeftPictID`, 3982 `kDoorExRightPictID`, 3983 `kDoorInRightPictID`, 3984 `kDoorInLeftPictID`, 3985 `kMailboxRightPictID`, 3986 `kMailboxLeftPictID`) | `RoomGraphics.c:16`, `ObjectDraw2.c:67-100`, `GliderDefines.h:566-567` |
| 4000–5018 | `PICT` | app | `GliderPRO/Glider PRO.r` |
| **3000–3299** | `PICT` | **house** — user backgrounds, "structure" sub-range | `GliderDefines.h:522-523`, `RoomInfo.c:762` |
| **3300–3799** | `PICT` | **house** — user backgrounds, "open/outdoor" sub-range | ditto |
| 10000 | `PICT` | app — the 72×34 `kCustomPict` placeholder | `GliderPRO/Glider PRO.r`; observed header `$"1056 0000 0000 0022 0048"`, i.e. `picSize` 0x1056, `picFrame` {0,0,34,72} |
| **10001+** | `PICT` | **house** — `kCustomPict` art | `ObjectInfo.c:1213` clamps the editor field to 10000..32767 |
| 1000–1062 | `snd ` | app sound effects (`kBaseBufferSoundID` 1000, `kMaxSounds` 64) | `Sound.c:14-15` |
| 2000–2006 | `snd ` | app music (`kBaseBufferMusicID` 2000, `kMaxMusic` 7) | `Music.c:15-16` |
| **3000+** | `snd ` | **house** — `kSoundTrigger` audio | `ObjectInfo.c:1237` clamps to 3000..32767 |
| **any** | `bnds` | **house only** — 4-byte wall descriptor per user background | `Room.c:937-966` |

Application resource-type census of `GliderPRO/Glider PRO.r` (the Rez file uses
`data 'TYPE' (id, "name") { ... }` syntax, *not* `resource 'TYPE' (id)`):
`PICT` 152, `snd ` 70, `DITL` 54, `cicn` 44, `ICON` 35, `DLOG` 28, `ALRT` 26, `dctb` 20,
`CURS` 16, `crsr` 12, `STR#` 10, `MENU` 6, `FREF` 6, `ICN#` 6, `icl8` 6, `icl4` 6,
`mctb` 5, `CNTL` 5, `ics8` 4, `ics4` 4, `ics#` 4, `WIND` 3, `vers` 2, `WDEF` 2, `clut` 2,
`acur` 1, `BNDL` 1, `ozm5` 1, `CDEF` 1, `PAT#` 1, `demo` 1, `wctb` 1, `ictb` 1, `cctb` 1,
`DLGX` 1.

**Observed shadowing.** 16 of the 22 houses define at least one `PICT` ID that the
application also defines, and therefore override application art while that house is
open:

| house | shadowed app `PICT` IDs |
|---|---|
| Art Museum | 1991, 1992, 1993 |
| CD Demo House | 3904, 3986, 10000 |
| California or Bust! | 10000 |
| Castle o' the Air | 10000 |
| Davis Station | 1991, 1992, 1993, 10000 |
| Fun House | 1991, 1992, 1993, **2014, 2015**, 3976, 10000 |
| Grand Prix | 1991, 1992, 1993, 10000 |
| ImagineHouse PRO II | 1991, 1992, 1993 |
| In The Mirror | 1991, 1992, 1993 |
| Land of Illusion | 1017, 1018, 1991, 1992, 1993, 10000 |
| Leviathan | 1991, 1992, 1993 |
| Metropolis | 1991, 1992, 1993, 1999, 10000 |
| Nemo's Market | 1017, 1018, 1991, 1992, 1993, 3981, 3982, 3983, 3984, 10000 |
| Rainbow's End | 1017, 1018, 1991, 1992, 1993, 3964, 10000 |
| SpacePods | 1018, 1991, 1992, 1993, 10000 |
| Teddy World | 1015, 1016, 1991, 1992, 1993, 3975, 10000 |
| Titanic | 10000 |

Counts: 1991/1992/1993 (the banner page frame) shadowed by 13 houses each; 10000 (the
`kCustomPict` placeholder) by 13; 1018 by 4; 1017 by 3; and 1015, 1016, 1999, 2014, 2015,
3904, 3964, 3975, 3976, 3981, 3982, 3983, 3984, 3986 by exactly one house each.

Two of these are load-bearing rather than cosmetic:

* **Fun House shadows `PICT` 2014 (`kRoof`) and 2015 (`kSky`).** Those are *built-in
  background* IDs. Fun House therefore reskins two of the eighteen stock backgrounds for
  its own 4 roof rooms and 9 sky rooms. A port that hard-codes the 18 backgrounds as
  compiled-in assets and only consults the house fork for IDs ≥ 3000 will render Fun
  House wrong.
* **No house shadows any application `snd ` ID.** Every house sound ID observed is
  ≥ 3000, so the sound chain is safe.

**Port note.** Go has no Resource Manager. The correct model is an explicit two-level
lookup: `func (h *House) Pict(id int) image.Image` tries the house's own PICT table
first, then falls back to the built-in application table. Do *not* special-case ID
ranges — Fun House proves that houses override IDs inside the "application" ranges on
purpose.

### 1.6 The QuickTime `.mov` sidecar

`OpenHouseMovie` (`GliderPRO/Sources/HouseIO.c:67-142`):

```
1. Guard on thisMac.hasQT and the whole function on #ifdef COMPILEQT.                 (:69, :78)
2. theSpec = theHousesSpecs[thisHouseIndex];
   PasStringConcat(theSpec.name, "\p.mov");   /* "Titanic" -> "Titanic.mov" */        (:80-81)
3. FSpGetFInfo -> if it fails, return silently. No movie is not an error.              (:83-85)
4. OpenMovieFile, NewMovieFromFile(newMovieActive).                                    (:87-101)
5. spaceSaver = NewHandle(307200L);           /* 300 KB reserve, freed immediately */ (:104)
6. GoToBeginningOfMovie; LoadMovieIntoRam(whole duration).                             (:112-121)
7. PrerollMovie(theMovie, 0, 0x000F0000);     /* rate 15.0 in 16.16 fixed */          (:124)
8. theTime = GetMovieTimeBase(theMovie);
   SetTimeBaseFlags(theTime, loopTimeBase);   /* hardware loop */                     (:132-133)
   SetMovieMasterTimeBase(theMovie, theTime, nil);
   LoopMovie();                               /* also clears+adds a 'LOOP' UserData */(:135, :48-63)
9. GetMovieBox(theMovie, &movieRect); hasMovie = true.                                 (:137-139)
```

The movie is mounted into a `kTV` object at draw time
(`GliderPRO/Sources/ObjectDrawAll.c:674-712`): the TV screen rect is the object rect
offset by `(left + 17, top + 10)` (`:685`), and the first TV encountered in a room claims
the movie — `tvWithMovieNumber = dynamicNum; tvInRoom = true;` (`:707-708`).

Observed sidecars — **15 of 22 houses ship one**:

| house | `.mov` bytes | `kTV` objects |
|---|---|---|
| Demo House | **119789** (largest) | 1 |
| Leviathan | 65920 | 4 |
| SpacePods | 51291 | 8 |
| Slumberland | 37640 | 8 |
| Davis Station | 34504 | 1 |
| CD Demo House | 33042 | 1 |
| ImagineHouse PRO II | 30707 | 7 |
| Teddy World | 28587 | 7 |
| Nemo's Market | 28541 | 1 |
| Art Museum | 28232 | 2 |
| Grand Prix | 26407 | 2 |
| Castle o' the Air | 21960 | 2 |
| Land of Illusion | 16016 | 10 |
| Rainbow's End | 7021 | 5 |
| Titanic | **6534** (smallest) | 13 |
| *(no movie)* California or Bust! | — | 1 |
| *(no movie)* In The Mirror | — | 3 |
| *(no movie)* Metropolis | — | 2 |
| *(no movie)* The Asylum Pro | — | 2 |
| *(no movie)* Empty House | — | 0 |
| *(no movie)* Fun House | — | 0 |
| *(no movie)* Sampler | — | 0 |

Note the asymmetry: **four houses place `kTV` objects but ship no movie**
(California or Bust!, In The Mirror, Metropolis, The Asylum Pro). A `kTV` with no movie
is legal and draws a static screen — the movie lookup fails silently at
`HouseIO.c:83-85`. Conversely there is no house with a movie but no TV.

**Port note.** QuickTime is unreplaceable as-is. Practical options: (a) decode the movies
offline into a frame sequence or GIF/APNG and play that in the TV rect; (b) ignore movies
and draw the static TV art. Either way, keep the "movie is optional, silently" semantics,
and keep the `(+17, +10)` screen inset.

### 1.7 `timeStamp`, the lock bit, and `flags`

`houseType.timeStamp` is a 4-byte Mac `GetDateTime` value (seconds since
1904-01-01 00:00:00 local) with two bits stolen:

* **Bit 0 is the lock flag.** `houseUnlocked = ((timeStamp & 0x00000001) == 0)`
  (`GliderPRO/Sources/HouseIO.c:402`); `WriteHouse` clears it for an unlocked house and
  sets it for a locked one (`HouseIO.c:483-486`). A *locked* house is the shipped,
  read-only state: the editor refuses to modify it. Timestamp resolution is therefore
  2 seconds.
* **Bit 31 is destroyed.** `WriteHouse` does `timeStamp &= 0x7FFFFFFF`
  (`GliderPRO/Sources/HouseIO.c:478`) before storing. Since every real 1990s date has
  bit 31 set (2^31 seconds after 1904-01-01 is 1972-01-19 03:14:08), the stored value reads as a date
  ~68.05 years too early unless bit 31 is restored.

Verified: adding back `0x80000000` turns every shipped house's nonsense 1927 date into a
coherent 1995 development timeline, cross-checked against the high-score `timeStamps`
field (which is *not* masked):

| house | raw `timeStamp` | date with bit 31 restored | lock | top high-score date |
|---|---|---|---|---|
| The Asylum Pro | `0x2BFD3EA1` | 1995-06-08 23:56:17 | LOCKED | 1995-06-09 00:00 |
| ImagineHouse PRO II | `0x2C1DC93D` | 1995-07-03 16:20:13 | LOCKED | 1995-07-03 16:18 |
| Nemo's Market | `0x2C2C9CD1` | 1995-07-14 22:14:41 | LOCKED | 1995-07-14 22:13 |
| Rainbow's End | `0x2C2C9DD1` | 1995-07-14 22:18:57 | LOCKED | 1995-07-14 22:18 |
| SpacePods | `0x2C2CA019` | 1995-07-14 22:28:41 | LOCKED | 1995-07-14 22:28 |
| Empty House | `0x2C3130EC` | 1995-07-18 09:35:40 | unlocked | — |
| Grand Prix | `0x2C318A3D` | 1995-07-18 15:56:45 | LOCKED | 1995-07-18 15:54 |
| In The Mirror | `0x2C329401` | 1995-07-19 10:50:41 | LOCKED | 1995-07-19 10:49 |
| Leviathan | `0x2C329E33` | 1995-07-19 11:34:11 | LOCKED | 1995-07-19 11:31 |
| Castle o' the Air | `0x2C32F464` | 1995-07-19 17:41:56 | unlocked | 1995-07-17 15:05 |
| Metropolis | `0x2C34904B` | 1995-07-20 22:59:23 | LOCKED | 1995-07-25 13:15 |
| CD Demo House | `0x2C3E8559` | 1995-07-28 12:15:21 | LOCKED | 1995-07-28 19:30 |
| Fun House | `0x2C446C10` | 1995-08-01 23:41:04 | unlocked | 1995-07-23 12:31 |
| Davis Station | `0x2C53AEFD` | 1995-08-13 13:30:37 | LOCKED | 1996-10-17 23:22 |
| Demo House | `0x2C53B041` | 1995-08-13 13:36:01 | LOCKED | 1995-07-17 11:04 |
| Slumberland | `0x2C53B149` | 1995-08-13 13:40:25 | LOCKED | 2000-05-11 11:50 |
| Teddy World | `0x2C5EAB21` | 1995-08-21 21:29:05 | LOCKED | 1995-07-20 17:33 |
| Land of Illusion | `0x2C6BE354` | 1995-08-31 22:08:20 | unlocked | — |
| California or Bust! | `0x2C7542B8` | 1995-09-08 00:45:44 | unlocked | 1995-09-08 16:53 |
| Art Museum | `0x2C808C35` | 1995-09-16 14:14:13 | LOCKED | 1996-01-13 00:31 |
| Titanic | `0x2D0246FE` | 1995-12-23 23:53:34 | unlocked | 1995-07-20 15:38 |
| Sampler | `0x3540BFE8` | 2000-05-11 19:52:08 | unlocked | 2000-05-11 19:52 |

Reading order in that table is the shipped houses' authoring order: Asylum Pro first
(June 1995), the Hartenstein trio (Nemo's Market / Rainbow's End / SpacePods) saved within
14 minutes of each other on 1995-07-14, and `Sampler` re-saved in 2000 — the same day as
Slumberland's newest high score. **15 of 22 houses are LOCKED; 7 are unlocked**
(Empty House, Castle o' the Air, Fun House, Land of Illusion, California or Bust!,
Titanic, Sampler).

`houseType.flags` is a `long` documented as "bit 0 = wardBit" (`GliderStructs.h:187`) and
decoded at `HouseIO.c:416-418`:

| bit | mask | variable | sense | observed |
|---|---|---|---|---|
| 0 | `0x00000001` | `wardBitSet` | set = on | **false in all 22 houses** |
| 1 | `0x00000002` | `phoneBitSet` | set = on | true in 8 houses |
| 2 | `0x00000004` | `bannerStarCountOn` | **clear = on** (inverted) | off in exactly one house |

Observed `flags` values: `0` in 14 houses, `2` in 7 (CD Demo House, California or Bust!,
Davis Station, Land of Illusion, Nemo's Market, Rainbow's End, SpacePods), and `6` in
exactly one — **Art Museum**, which is therefore the only shipped house that suppresses
the "stars remaining" count on the banner page.

---

## 2. On-disk layout, verified byte by byte

All multi-byte fields are **big-endian**. Structs are **unpadded**: `Boolean` and `Byte`
are 1 byte, `short` 2, `long` 4, and the compiler's 2-byte alignment happens to coincide
with the natural layout everywhere, so `sizeof` matches the sum of the fields exactly
(`GliderStructs.h` annotates each field with its size and each struct with its total).

Mac scalar types used:

| type | bytes | note |
|---|---|---|
| `Point` | 4 | `{short v; short h}` — **vertical first**. Constant source of port bugs. |
| `Rect` | 8 | `{short top, left, bottom, right}` |
| `Str15` | 16 | Pascal string: length byte + 15 chars |
| `Str27` | 28 | length byte + 27 chars |
| `Str31` | 32 | length byte + 31 chars |
| `Str255` | 256 | length byte + 255 chars |

Text is **MacRoman**, not ASCII and not UTF-8. Room names and banners in the shipped
corpus do contain high-bit MacRoman bytes (e.g. the `…` in `Welcome…`, and `Uhhh… Hi,
Officer…` in Grand Prix's high-score banner).

### 2.1 `houseType` — 866 bytes + rooms

`GliderPRO/Headers/GliderStructs.h:182-198`.

| offset | size | field | type | notes |
|---|---|---|---|---|
| 0 | 2 | `version` | `short` | `0x0200` = `kHouseVersion` (`GliderDefines.h:517`); `>= 0x0300` = `kNewHouseVersion` is refused (`HouseIO.c:395`) |
| 2 | 2 | `unusedShort` | `short` | **not zero in 10 of 22 houses** — see §11.1 |
| 4 | 4 | `timeStamp` | `long` | Mac seconds since 1904, bit 31 cleared, bit 0 = locked (§1.7) |
| 8 | 4 | `flags` | `long` | bit 0 `wardBit`, bit 1 `phoneBit`, bit 2 inverted `bannerStarCount` |
| 12 | 4 | `initial` | `Point` | glider spawn point, **v at 12, h at 14** |
| 16 | 256 | `banner` | `Str255` | shown before play; wrapped to 40 columns (`HouseLegal.c:611`) |
| 272 | 256 | `trailer` | `Str255` | shown on winning; wrapped to 64 columns (`HouseLegal.c:612`) |
| 528 | 292 | `highScores` | `scoresType` | §2.2 |
| 820 | 40 | `savedGame` | `gameType` | §2.2; meaningful only when `hasGame` != 0 |
| 860 | 1 | `hasGame` | `Boolean` | **1 in exactly 2 houses** (ImagineHouse PRO II, Titanic) |
| 861 | 1 | `unusedBoolean` | `Boolean` | **not zero in 7 of 22 houses** — see §11.1 |
| 862 | 2 | `firstRoom` | `short` | index into `rooms[]`, not a (floor, suite) |
| 864 | 2 | `nRooms` | `short` | array length |
| 866 | 348·`nRooms` | `rooms[]` | `roomType[]` | §2.3 |

Verified against `Demo House` (`GliderPRO/Houses/Demo House.binhex`, data fork
16526 bytes):

```
offset 0x0000  02 00 00 00 2c 53 b0 41 00 00 00 00 00 6b 00 31
               ^ver  ^unus ^timeStamp  ^flags=0    ^initial v=107 h=49
offset 0x0010  6f 57 65 6c 63 6f 6d 65 20 74 6f 20 74 68 65 20
               ^banner len=0x6f=111, "Welcome to the "
offset 0x0110  6f 45 78 63 65 6c 6c 65 6e 74 21 0d 54 68 61 74
               ^trailer len=111, "Excellent!\rThat..."   (note the embedded CR)
offset 0x0210  13 54 68 65 20 52 65 74 75 72 6e 20 6f 66 20 4f
               ^highScores.banner len=0x13=19, "The Return of O..."
offset 0x035c  00 00 00 00 00 2d
               ^hasGame=0 ^unusedBoolean=0 ^firstRoom=0 ^nRooms=0x2d=45
```

`16526 = 866 + 348 × 45`. ✔

`banner`/`trailer` may contain embedded `\r` (`0x0D`) as a hard line break, as Demo
House's trailer does. A port must treat `\r` inside these Pascal strings as a newline,
not as text.

`initial` feeds `WhereDoesGliderBegin` (`GliderPRO/Sources/House.c:224-240`):

```
1. QSetRect(theRect, 0, 0, kGliderWide, kGliderHigh);   /* 48 x 20 */
2. QOffsetRect(theRect, initialPt.h, initialPt.v);
```

with `kGliderWide` = 48 and `kGliderHigh` = 20 (`GliderDefines.h:548-549`). So the spawn
rect is `{top: initial.v, left: initial.h, bottom: initial.v+20, right: initial.h+48}`.
All 22 shipped `initial` Points are distinct — no two houses spawn the glider at the
same pixel.

### 2.2 `scoresType` (292 B) and `gameType` (40 B)

`scoresType`, `GliderStructs.h:107-114`, at house offset 528:

| offset (abs) | size | field | notes |
|---|---|---|---|
| 528 | 32 | `banner` (`Str31`) | the high-score screen caption |
| 560 | 160 | `names[10]` (`Str15[]`) | `kMaxScores` = 10 (`GliderDefines.h:249`) |
| 720 | 40 | `scores[10]` (`long[]`) | points |
| 760 | 40 | `timeStamps[10]` (`unsigned long[]`) | Mac seconds — **not** masked, so directly usable |
| 800 | 20 | `levels[10]` (`short[]`) | rooms visited, *not* a difficulty level |

Observed high-score state (top entry only):

| house | `highScores.banner` | non-zero scores | top score | top name | rooms visited |
|---|---|---|---|---|---|
| Art Museum | `Your Message Here` | 2 | 18500 | `Your Name` | 22 |
| CD Demo House | `The Return of Ozma!` | 3 | 25300 | `Ozma` | 28 |
| California or Bust! | `The Return of Ozma!` | 2 | 7500 | `Ozma` | 13 |
| Castle o' the Air | `The Return of Ozma!` | 3 | 7500 | `Ozma` | 15 |
| Davis Station | `Johnner beat the Kimmer.` | 3 | 17800 | `Johnner` | 38 |
| Demo House | `The Return of Ozma!` | 1 | 7400 | `Ozma` | 13 |
| Empty House | `Empty House` | **0** | 0 | `--------------` | 0 |
| Fun House | `The Return of Ozma!` | **10 (full board)** | 1600 | `Ozma` | 8 |
| Grand Prix | `Uhhh… Hi, Officer…` | 3 | 34400 | `Paul` | 88 |
| ImagineHouse PRO II | `Thankyuh… Thankyuh verra much…` | 1 | **47000 (highest)** | `Paul` | **108** |
| In The Mirror | `The Return of Ozma!` | 1 | 4100 | `Ozma` | 11 |
| Land of Illusion | `Land of Illusion` | **0** | 0 | `--------------` | 0 |
| Leviathan | `The Return of Ozma!` | 1 | 8400 | `Ozma` | 40 |
| Metropolis | `The Return of Ozma!` | 3 | 2300 | `Ozma` | 17 |
| Nemo's Market | `The Return of Ozma!` | 1 | 20700 | `Ozma` | 23 |
| Rainbow's End | `The Return of Ozma!` | 1 | 2000 | `Ozma` | 8 |
| Sampler | `Your Message Here` | 2 | 5200 | `Your Name` | 2 |
| Slumberland | `Your Message Here` | 3 | 10800 | `Your Name` | 25 |
| SpacePods | `The Return of Ozma!` | 1 | 6700 | `Ozma` | 43 |
| Teddy World | `The Return of Ozma!` | 1 | 3400 | `Ozma` | 16 |
| The Asylum Pro | `Spam Is Good For You.` | 1 | 4100 | `Albert` | 16 |
| Titanic | `The Return of Ozma!` | 2 | 15300 | `Ozma` | 33 |

The empty-name filler `--------------` (14 hyphens) is the initialised-but-unused state;
`Your Message Here` / `Your Name` is the untouched template. `Paul` is Jonathan Chin's
alias (`README.md:9`), and the two houses with `Paul` at the top of the board
(Grand Prix, ImagineHouse PRO II) are both his.

`gameType`, `GliderStructs.h:116-134`, at house offset 820:

| offset (rel) | size | field | notes |
|---|---|---|---|
| 0 | 2 | `version` | 256 = `0x0100` in 17 houses |
| 2 | 2 | `wasStarsLeft` | seeds `numStarsRemaining` on resume (`Play.c:312`) |
| 4 | 4 | `timeStamp` | *not* masked here |
| 8 | 4 | `where` (`Point`) | glider position |
| 12 | 4 | `score` | |
| 16 | 4 | `unusedLong` | |
| 20 | 4 | `unusedLong2` | |
| 24 | 2 | `energy` | |
| 26 | 2 | `bands` | rubber bands held |
| 28 | 2 | `roomNumber` | index into `rooms[]`; `SetHouseToSavedRoom` = `ForceThisRoom(smallGame.roomNumber)` (`Play.c:380`) |
| 30 | 2 | `gliderState` | |
| 32 | 2 | `numGliders` | `kInitialGliders` = 2 for a new game (`Play.c:18`) |
| 34 | 2 | `foil` | |
| 36 | 2 | `unusedShort` | |
| 38 | 1 | `facing` (`Boolean`) | |
| 39 | 1 | `showFoil` (`Boolean`) | |

Only two houses have `hasGame` = 1, and only for those two is this block meaningful:

| house | roomNumber | score | gliders | energy | bands | foil | starsLeft |
|---|---|---|---|---|---|---|---|
| ImagineHouse PRO II | 45 (of 279) | 5900 | 5 | **−150** | 23 | 8 | 3 |
| Titanic | 104 (of 208) | 4700 | 2 | 0 | 0 | 0 | 1 |

The other 20 houses carry stale bytes here. See §11.1 — this is a real trap.

### 2.3 `roomType` — 348 bytes

`GliderPRO/Headers/GliderStructs.h:166-180`.

| offset | size | field | type | notes |
|---|---|---|---|---|
| 0 | 28 | `name` | `Str27` | length clamped to 27 by `CheckRoomNameLength` (`HouseLegal.c:857-879`; the clamp itself is at `:870-875`) |
| 28 | 2 | `bounds` | `short` | **only meaningful when `background` ≥ 3000**; bit 0 = "original bounds used" marker, `bounds >> 1` is the wall code (§2.3.1) |
| 30 | 1 | `leftStart` | `Byte` | glider entry **y** (vertical pixel offset) when arriving from the west: `enterRect` is offset by `kGliderStartsDown + leftStart - 2` = `30 + leftStart` (`Transit.c:176`, `:185`); mode 32 |
| 31 | 1 | `rightStart` | `Byte` | same, for arrival from the east (`Transit.c:208`, `:217`); mode 32 |
| 32 | 1 | `unusedByte` | `Byte` | zeroed by `CheckRoomNameLength` (`HouseLegal.c:868`); **0 in all 4070 shipped rooms** |
| 33 | 1 | `visited` | `Boolean` | authoring residue; see §11.2 |
| 34 | 2 | `background` | `short` | PICT ID: 2000–2017 built-in, 3000–3799 user |
| 36 | 16 | `tiles[8]` | `short[8]` | `kNumTiles` = 8 (`GliderDefines.h:496`); each is a 0..7 column index into the 512-px-wide background strip |
| 52 | 2 | `floor` | `short` | signed; legal range −7..56 (`HouseLegal.c:805-816`); **+1 is north/up** |
| 54 | 2 | `suite` | `short` | 0..127; **+1 is east/right**; `kRoomIsEmpty` (−1) marks a placeholder |
| 56 | 2 | `openings` | `short` | **dead field** — the only assignment anywhere is `thisRoom->openings = 0;` (`Room.c:179`), and it is 0 in all 4070 shipped rooms |
| 58 | 2 | `numObjects` | `short` | live prefix length of `objects[]` |
| 60 | 288 | `objects[24]` | `objectType[24]` | `kMaxRoomObs` = 24 (`GliderDefines.h:250`) |

Verified against `Demo House` room 0 ("Air Vents", which is also its `firstRoom`), at
data-fork offset 866. The raw 60 bytes preceding the objects are:

```
866  09 41 69 72 20 56 65 6e 74 73 6d 6f 6f 6d 55 70 00 00 00 00 07 e0 1f f8 3f fc 7f fe
     ^len=9 ^"Air Vents"                 ^^^^^^^^^^^^^^^^^ STALE PADDING ^^^^^^^^^^^^^^^^
894  00 00      bounds     = 0            -> fall back to the 'bnds' resource
896  00         leftStart  = 0
897  00         rightStart = 0
898  00         unusedByte = 0
899  01         visited    = 1  (TRUE)
900  0b b8      background = 0x0bb8 = 3000 = kUserBackground
902  00 00 00 01 00 02 00 03 00 04 00 05 00 06 00 07
                tiles[8]   = 0, 1, 2, 3, 4, 5, 6, 7   (the identity strip)
918  00 01      floor      = 1
920  00 3f      suite      = 0x3f = 63
922  00 00      openings   = 0
924  00 0a      numObjects = 10
926  <objects[24]>
```

This parse is self-consistent: `firstRoom` = 0 and the §3.5 table's "(floor 1, suite 63)"
agree, `background` 3000 with `bounds` 0 is exactly the case that needs Demo House's
`bnds` 3000 resource (code 4 = right side open only), and `numObjects` = 10 matches the live
object count.

**The `Str27` padding is not zeroed.** Bytes `name[len+1 .. 27]` hold whatever was there
before, and the editor never clears them. Measured: **4008 of 4070 rooms (98.5 %) have
non-zero bytes past the name length**; 9 of the 22 houses have it in *every* room
(Art Museum, California or Bust!, Castle o' the Air, Davis Station, Demo House,
Empty House, Fun House, In The Mirror, Sampler). The residue is legible as the tails of
previous room names — Art Museum room 0 is named `Narcissus` and its padding reads
`rcissus\0\0\0\x07\xe0\x1f\xf8?\xfc\x7f\xfe`; room 9 is named `Van Gogh` and its padding
reads `'s Starry Night?\xfc\x7f\xfe`, i.e. the room was once called
"Van Gogh's Starry Night". The recurring trailing octet
`07 e0 1f f8 3f fc 7f fe` is the never-overwritten tail of the editor's blank-room
template.

Implications for a port: read the name as `name[1 : 1+name[0]]` and never as a
NUL-terminated string; and if the port ever wants to re-emit a byte-identical house file,
it must preserve the padding verbatim rather than zero-filling it.

Corpus-wide field census over all 4070 real rooms:

| field | observation |
|---|---|
| `leftStart` | mode 32 → 3208 rooms (78.8 %); 0 → 642; the remaining 220 rooms spread over 110 distinct values in 2..255 |
| `rightStart` | mode 32 → 3211 rooms; 0 → 639 |
| `unusedByte` | **0 in all 4070** |
| `openings` | **0 in all 4070** |
| `visited` | 1 in 436 rooms (see §11.2) |
| `numObjects` | matches the live-object count in **all 4070** |
| `tiles[]` histogram | 0 → 5114, 1 → 7036, 2 → 6395, 3 → 3229, 4 → 3925, 5 → 2263, 6 → 2161, 7 → 2437 (32560 slots = 4070 × 8; the eight counts sum to exactly 32560, so no tile value in the corpus is outside 0..7) |
| `name` | 2140 distinct names over 4070 rooms; **zero empty names** |

Room geometry constants (`GliderPRO/Headers/GliderDefines.h`):

| constant | value | line | meaning |
|---|---|---|---|
| `kRoomWide` | 512 | :499 | room width in pixels |
| `kTileWide` | 64 | :497 | 512 / 8 tiles |
| `kTileHigh` | 322 | :498 | room height in pixels |
| `kVertLocalOffset` | 322 | :501 | vertical offset between adjacent floors |
| `kFloorSupportTall` | 44 | :500 | |
| `kCeilingLimit` | 8 | :503 | |
| `kFloorLimit` | 312 | :504 | |
| `kRoofLimit` | 122 | :505 | |
| `kLeftWallLimit` | 12 | :506 | glider stops here when the left wall is closed |
| `kNoLeftWallLimit` | −24 | :507 | glider may leave the room to the west |
| `kRightWallLimit` | 500 | :508 | |
| `kNoRightWallLimit` | 536 | :509 | |
| `kNoCeilingLimit` | −10 | :510 | |
| `kNoFloorLimit` | 332 | :511 | |
| `kMaxNumRoomsH` | 128 | :543 | suite range 0..127 |
| `kMaxNumRoomsV` | 64 | :544 | floor range −7..56 |
| `kNumUndergroundFloors` | 8 | :535 | added to `floor` before link encoding |

#### 2.3.1 `bounds` — what it actually means

`bounds` is only consulted for **user backgrounds** (`background` ≥ `kUserBackground`
3000). The editor decodes it at `RoomInfo.c:744-749` and re-encodes it at
`RoomInfo.c:767-780`. **A set bit means that side is OPEN, not that it has a wall** — the
editor's own variables are named `originalLeftOpen`, `originalTopOpen`,
`originalRightOpen`, `originalBottomOpen`:

```
1. Accept the typed PICT ID only if longID >= 3000 && longID < 3800 && PictIDExists(longID). (:762)
2. tempShort = 0;
3. if (originalLeftOpen)   tempShort += 1;                                       (:768-769)
4. if (originalTopOpen)    tempShort += 2;                                       (:770-771)
5. if (originalRightOpen)  tempShort += 4;                                       (:772-773)
6. if (originalBottomOpen) tempShort += 8;                                       (:774-775)
7. if (originalFloor)      tempShort += 16;   /* floor support */                (:776-777)
8. tempShort = tempShort << 1;                                                   (:778)
9. tempShort += 1;   /* "flag that says orginal bounds used" [sic] */            (:779)
10. thisRoom->bounds = tempShort;                                                (:780)
```

and the matching decode is `tempShort = thisRoom->bounds >> 1;` followed by
`originalLeftOpen = ((tempShort & 1) == 1)` and so on (`RoomInfo.c:744-749`).

So the encoding is:

| `bounds` bit | mask | meaning |
|---|---|---|
| 0 | `0x0001` | **marker**: "this room has an explicit bounds code" — always 1 when non-zero |
| 1 | `0x0002` | left side **open** (no left wall) |
| 2 | `0x0004` | top **open** (no ceiling) |
| 3 | `0x0008` | right side **open** (no right wall) |
| 4 | `0x0010` | bottom **open** (no floor) |
| 5 | `0x0020` | floor support present |

and the code the readers use is `boundsCode = bounds >> 1`, with bit 0 = left open,
bit 1 = top open, bit 2 = right open, bit 3 = bottom open, bit 4 = floor support.
`bounds == 0` means "no explicit code — fall back to the `bnds` resource".

The polarity is confirmed independently by every reader: `DetermineRoomOpenings` does
`leftOpen = ((boundsCode & 0x0001) == 0x0001)` (`Room.c:831`), `DoesRoomHaveFloor` does
`hasFloor = ((boundsCode & 0x0008) != 0x0008)` (`Room.c:1149`), and `DoesRoomHaveCeiling`
does `hasCeiling = ((boundsCode & 0x0002) != 0x0002)` (`Room.c:1183`).

Verified across the corpus: the `bounds` histogram has 32 distinct values (31 of them
non-zero), and **every non-zero value has bit 0 set**, exactly as the encoder guarantees.
In the table below `L`/`T`/`R`/`B` mark the sides that are **open**.

| `bounds` | code | open sides | rooms |
|---|---|---|---|
| 0 | — | *(use `bnds` resource)* | 2401 |
| 1 | 0 | *(nothing open — fully enclosed)* | 167 |
| 3 | 1 | L | 30 |
| 5 | 2 | T | 9 |
| 7 | 3 | LT | 18 |
| 9 | 4 | R | 29 |
| 11 | 5 | LR | 63 |
| 13 | 6 | TR | 25 |
| 15 | 7 | LTR | 93 |
| 17 | 8 | B | 13 |
| 19 | 9 | LB | 29 |
| 21 | 10 | TB | 115 |
| 23 | 11 | LTB | 22 |
| 25 | 12 | RB | 110 |
| 27 | 13 | LRB | 101 |
| 29 | 14 | TRB | 10 |
| **31** | **15** | **LTRB (all four sides open)** | **542** |
| 33 | 16 | floorSupport | 47 |
| 35 | 17 | L + support | 36 |
| 37 | 18 | T + support | 8 |
| 39 | 19 | LT + support | 14 |
| 41 | 20 | R + support | 26 |
| 43 | 21 | LR + support | 39 |
| 45 | 22 | TR + support | 16 |
| 47 | 23 | LTR + support | 51 |
| 49 | 24 | B + support | 1 |
| 53 | 26 | TB + support | 1 |
| 55 | 27 | LTB + support | 5 |
| 57 | 28 | RB + support | 2 |
| 59 | 29 | LRB + support | 23 |
| 61 | 30 | TRB + support | 4 |
| 63 | 31 | LTRB + support | 20 |
| | | **total non-zero** | **1669** |

`IsRoomAStructure` (`GliderPRO/Sources/Room.c:763-812`) is the odd one out: it tests the
**raw** `bounds`, not the shifted code:

```
1. if (whichBack >= kUserBackground)               /* 3000 */
2.     if (bounds != 0) return ((bounds & 32) == 32);      /* bit 5 = floor support */
3.     else             return (background < kUserStructureRange);   /* 3300 */
4. else switch (whichBack): return true for kPaneledRoom, kSimpleRoom, kChildsRoom,
        kAsianRoom, kUnfinishedRoom, kSwingersRoom, kBathroom, kLibrary, kSkywalk, kRoof;
        false otherwise.
```

That is: for a user background with no explicit `bounds`, the *ID range itself* decides —
3000..3299 is a "structure" (has a floor support), 3300..3799 is not. `kUserBackground`
= 3000 and `kUserStructureRange` = 3300 (`GliderDefines.h:522-523`).

#### 2.3.2 `bnds` resources — the fallback wall descriptor

`GetOriginalBounding` (`GliderPRO/Sources/Room.c:937-966`):

```
1. theRes = GetResource('bnds', theID);
2. if (theRes == nil):
3.     boundCode = 0;                       /* nothing open — fully enclosed */
4.     if (PictIDExists(theID)) YellowAlert(kYellowNoBoundsRes, 0);
5. else:
6.     boundCode = 0;
7.     if ((*theRes)->left)   boundCode += 1;     /* left open   */
8.     if ((*theRes)->top)    boundCode += 2;     /* top open    */
9.     if ((*theRes)->right)  boundCode += 4;     /* right open  */
10.    if ((*theRes)->bottom) boundCode += 8;     /* bottom open */
```

The resource is 4 bytes, one `Boolean` per side, in the order left, top, right, bottom —
i.e. `boundsType {Boolean left, top, right, bottom}` (`GliderStructs.h:266-272`).
Note that `bnds` cannot express the floor-support bit; that is what
`IsRoomAStructure`'s 3300 threshold is for.

**Observed: 8 houses ship `bnds`, 70 resources in total.** Every byte observed is `0x00`
or `0x01`.

| house | count | ID = code |
|---|---|---|
| Castle o' the Air | 9 | 3000=5, 3001=7, 3002=5, 3003=15, 3300=7, 3301=15, 3302=7, 3303=15, 3304=5 |
| Demo House | 10 | 3000=4, 3001=5, 3002=5, 3003=5, 3004=1, 3005=0, 3300=14, 3301=6, 3302=7, 3303=7 |
| ImagineHouse PRO II | 14 | 3000=15, 3001=1, 3002=4, 3003=1, 3004=14, 3005=6, 3006=12, 3007=5, 3008=15, 3300=15, 3301=2, 3302=12, 3303=15, 3304=7 |
| Land of Illusion | 5 | 3001=13, 3002=15, 3003=5, 3301=15, 3302=5 |
| Leviathan | 4 | 3000=2, 3001=5, 3300=15, 3301=15 |
| Rainbow's End | 6 | 3001=15, 3301=7, 3302=3, 3303=15, 3304=15, 3305=15 |
| Slumberland | 20 | 3000=5, 3001=0, 3002=0, 3003=1, 3004=1, 3005=1, 3006=7, 3007=7, 3008=1, 3009=5, 3010=5, 3300=3, 3301=10, 3302=12, 3303=9, 3304=7, 3305=7, 3306=7, 3307=15, 3308=15 |
| The Asylum Pro | 2 | 3000=0, 3001=0 |

**155 rooms across 7 houses actually need the fallback** (user background *and*
`bounds == 0`): Castle o' the Air 55, Rainbow's End 30, Slumberland 24, Land of Illusion
21, Demo House 10, Leviathan 9, The Asylum Pro 6. **Zero of those 155 hit a missing
`bnds`** — the `YellowAlert(kYellowNoBoundsRes)` path is never taken by shipped data.
ImagineHouse PRO II ships 14 `bnds` resources but has no room that needs them
(every one of its user-background rooms carries an explicit `bounds`); they are vestigial.

#### 2.3.3 `DetermineRoomOpenings` — the actual wall model

`GliderPRO/Sources/Room.c:816-933`. This is what a port must reproduce exactly, because
it decides whether the glider can leave a room sideways.

```
 1. leftTile = tiles[0];  rightTile = tiles[kNumTiles-1];      /* kNumTiles = 8 */
 2. if (whichBack >= kUserBackground) {                          /* 3000 */
 3.     boundsCode = (bounds != 0) ? (bounds >> 1)
 4.                                : GetOriginalBounding(whichBack);
 5.     leftOpen  = ((boundsCode & 0x0001) == 0x0001);   /* bit 0 = left open  */
 6.     rightOpen = ((boundsCode & 0x0004) == 0x0004);   /* bit 2 = right open */
 7.     leftThresh  = leftOpen  ? kNoLeftWallLimit  : kLeftWallLimit;   /* Room.c:834-837 */
 8.     rightThresh = rightOpen ? kNoRightWallLimit : kRightWallLimit;  /* Room.c:839-842 */
 9. } else switch (whichBack) {
10.     case kSimpleRoom, kPaneledRoom, kBasement, kChildsRoom, kAsianRoom,
11.          kUnfinishedRoom, kSwingersRoom, kBathroom, kLibrary, kSky:
12.         leftThresh  = (leftTile  == 0) ? kLeftWallLimit  : kNoLeftWallLimit;
13.         rightThresh = (rightTile == 7) ? kRightWallLimit : kNoRightWallLimit;
14.         leftOpen  = (leftTile  != 0);
15.         rightOpen = (rightTile != 7);
16.         break;
17.     case kDirt:                                        /* 2011 */
18.         leftThresh  = (leftTile  == 1) ? kLeftWallLimit  : kNoLeftWallLimit;
19.         rightThresh = (rightTile == 7) ? kRightWallLimit : kNoRightWallLimit;
20.         leftOpen    = (leftTile != 0);                  /* <-- disagrees with line 18 */
21.         rightOpen   = (rightTile != 7);
22.         break;
23.     case kMeadow:                                      /* 2012 */
24.         leftThresh  = (leftTile  == 6) ? kLeftWallLimit  : kNoLeftWallLimit;
25.         rightThresh = (rightTile == 7) ? kRightWallLimit : kNoRightWallLimit;
26.         leftOpen  = (leftTile  != 6);
27.         rightOpen = (rightTile != 7);
28.         break;
29.     case kGarden, kSkywalk, kField, kStratosphere, kStars:
30.         leftThresh = kNoLeftWallLimit; rightThresh = kNoRightWallLimit;
31.         leftOpen = true; rightOpen = true;              /* always passable */
32.         break;
33.     default:                                           /* includes kRoof 2014 */
34.         leftThresh  = (leftTile  == 0) ? kLeftWallLimit  : kNoLeftWallLimit;
35.         rightThresh = (rightTile == 7) ? kRightWallLimit : kNoRightWallLimit;
36.         leftOpen  = (leftTile  != 0);
37.         rightOpen = (rightTile != 7);
38. }
39. bottomOpen = !DoesRoomHaveFloor();                     /* Room.c:924-927 */
40. topOpen    = !DoesRoomHaveCeiling();                   /* Room.c:929-932 */
```

Note the polarity of lines 7–8: an *open* side gets the `kNoWallLimit` threshold (the
glider may walk past the room edge), a *closed* side gets the `kWallLimit` threshold.
Note also that `kSky` (2015), not `kSkywalk`, belongs to the tile-driven group on line 11;
`kSkywalk` (2010) appears only in the always-passable group on line 29.

The `kDirt` case (lines 17–22) is the single place in the source where the *threshold* and
the *open flag* are computed from different predicates. For `tiles[0] == 0` the room is
"not left-open" **and** gets the `kNoLeftWallLimit` (−24) threshold, so the glider walks
out to the west while `leftOpen` is false. `kDirt` is the most common background in the
corpus (421 rooms in 16 houses), so this is not an edge case.

Companion predicates:

| function | user background (≥3000) | built-in |
|---|---|---|
| `DoesRoomHaveFloor` (`Room.c:1138-1168`) | `(boundsCode & 0x0008) != 0x0008` | false for kSky 2015, kStratosphere 2016, kStars 2017; true otherwise |
| `DoesRoomHaveCeiling` (`Room.c:1172-1205`) | `(boundsCode & 0x0002) != 0x0002` | false for kGarden 2009, kMeadow 2012, kField 2013, kRoof 2014, kSky, kStratosphere, kStars; true otherwise |
| `IsShadowVisible` (`Room.c:1103-1134`) | same shape | adds kRoof to the "no floor to cast a shadow on" set |

Corpus openings census, computed with the faithful port of the above over all 4070 real
rooms: **left open 2759 (67.8 %), right open 3015 (74.1 %), top open 2263 (55.6 %),
bottom open 1972 (48.5 %)**. Rooms are more often open sideways than vertically — the
corpus is built out of horizontal runs, punctuated by vertical shafts.

#### 2.3.4 `GetNumberOfLights` — is the room dark?

`GliderPRO/Sources/Room.c:970-1099`. A room with zero lights renders black, which is a
deliberate design device (Slumberland has 29 dark rooms in play mode, the third-largest
count in the corpus). The function is written twice in the source — once for
`theMode == kEditMode` operating on `thisRoom`, once for play mode operating on
`(*thisHouse)->rooms[where]` — and the two copies differ only in which boolean the lamp
cases test:

```
 1. count = 0;
 2. switch (background) {
 3.     case kGarden, kSkywalk, kMeadow, kField, kRoof, kSky, kStratosphere, kStars:
 4.         count = 1; break;                         /* self-lit outdoor backgrounds */
 5.     case kDirt:
 6.         count = 0;
 7.         if (all 8 tiles == 0) count = 1;          /* open sky above the dirt */
 8.         break;
 9.     default:                                      /* incl. every user background */
10.         count = 0; break;
11. }
12. if (count == 0)                     /* <-- guard: a self-lit room never scans objects */
13.     for (i = 0; i < kMaxRoomObs; i++)             /* all 24 slots, not numObjects */
14.         switch (objects[i].what) {
15.             case kDoorInLf, kDoorInRt, kWindowInLf, kWindowInRt, kWallWindow:
16.                 count++;  break;                  /* always counts, no state */
17.             case kCeilingLight, kLightBulb, kTableLamp, kHipLamp, kDecoLamp,
18.                  kFlourescent, kTrackLight, kInvisLight:
19.                 if (playing) { if (objects[i].data.f.state)   count++; }
20.                 else         { if (objects[i].data.f.initial) count++; }
21.                 break;
22.         }
23. return count;
```

Two details a port must not paraphrase away: the object scan is **skipped entirely** when
the background switch already produced a light (`if (count == 0)`, `Room.c:1004` and
`Room.c:1068`), and it runs over **all `kMaxRoomObs` = 24 slots**, not `numObjects`
(`Room.c:1006` and `Room.c:1070`).

Note that doors and windows are unconditional light sources, and that the eight lamp
types are gated on `data.f.state` while *playing* (`Room.c:1090`) but `data.f.initial` in
the *editor* (`Room.c:1026`) — so a light that starts off makes the room look dark in the
editor and lit after a switch is thrown.

### 2.4 `objectType` — 12 bytes, 2 + a 10-byte union

`GliderPRO/Headers/GliderStructs.h:90-105`.

| offset | size | field |
|---|---|---|
| 0 | 2 | `what` (`short`) — object code, `kObjectIsEmpty` = −1 for an empty slot (`GliderDefines.h:526`) |
| 2 | 10 | `data` — one of nine variants, selected by the *class* of `what` (its high nibble) |

The nine variants, each exactly 10 bytes:

**a — `blowerType` (`GliderStructs.h:11-19`), `what` 0x01–0x10**

| off | size | field | note |
|---|---|---|---|
| 2 | 4 | `topLeft` (`Point`) | v then h |
| 6 | 2 | `distance` | |
| 8 | 1 | `initial` (`Boolean`) | |
| 9 | 1 | `state` (`Boolean`) | |
| 10 | 1 | `vector` (`Byte`) | bit flags: 1 = up, 2 = right, 4 = down, 8 = left (`GliderStructs.h:16-17`) |
| 11 | 1 | `tall` (`Byte`) | |

**b — `furnitureType` (`:21-25`), `what` 0x11–0x1F**

| off | size | field |
|---|---|---|
| 2 | 8 | `bounds` (`Rect`) — top, left, bottom, right |
| 10 | 2 | `pict` |

**c — `bonusType` (`:27-34`), `what` 0x21–0x2F (prizes)**

| off | size | field | note |
|---|---|---|---|
| 2 | 4 | `topLeft` (`Point`) | |
| 6 | 2 | `length` | grease-spill length |
| 8 | 2 | `points` | **`kInvisBonus` point value** — read by `CountTotalHousePoints` |
| 10 | 1 | `state` (`Boolean`) | |
| 11 | 1 | `initial` (`Boolean`) | |

**d — `transportType` (`:36-43`), `what` 0x31–0x40**

| off | size | field | note |
|---|---|---|---|
| 2 | 4 | `topLeft` (`Point`) | |
| 6 | 2 | `tall` | invisible-transport height |
| 8 | 2 | `where` | **link destination**, `MergeFloorSuite`-encoded; −1 = unlinked |
| 10 | 1 | `who` (`Byte`) | destination object index 0..23; **255 = unlinked** |
| 11 | 1 | `wide` (`Byte`) | |

**e — `switchType` (`:45-52`), `what` 0x41–0x49**

| off | size | field | note |
|---|---|---|---|
| 2 | 4 | `topLeft` (`Point`) | |
| 6 | 2 | `delay` | |
| 8 | 2 | `where` | link destination, **or the `snd ` resource ID for `kSoundTrigger`** |
| 10 | 1 | `who` (`Byte`) | |
| 11 | 1 | `type` (`Byte`) | |

**f — `lightType` (`:54-62`), `what` 0x51–0x58**

| off | size | field |
|---|---|---|
| 2 | 4 | `topLeft` (`Point`) |
| 6 | 2 | `length` |
| 8 | 1 | `byte0` |
| 9 | 1 | `byte1` |
| 10 | 1 | `initial` (`Boolean`) |
| 11 | 1 | `state` (`Boolean`) |

**g — `applianceType` (`:64-72`), `what` 0x61–0x6E**

| off | size | field | note |
|---|---|---|---|
| 2 | 4 | `topLeft` (`Point`) | |
| 6 | 2 | `height` | toaster launch height — **and the PICT ID for `kCustomPict`** |
| 8 | 1 | `byte0` | |
| 9 | 1 | `delay` (`Byte`) | |
| 10 | 1 | `initial` (`Boolean`) | |
| 11 | 1 | `state` (`Boolean`) | |

**h — `enemyType` (`:74-82`), `what` 0x71–0x79**

| off | size | field |
|---|---|---|
| 2 | 4 | `topLeft` (`Point`) |
| 6 | 2 | `length` |
| 8 | 1 | `delay` (`Byte`) |
| 9 | 1 | `byte0` |
| 10 | 1 | `initial` (`Boolean`) |
| 11 | 1 | `state` (`Boolean`) |

**i — `clutterType` (`:84-88`), `what` 0x81–0x8F**

| off | size | field |
|---|---|---|
| 2 | 8 | `bounds` (`Rect`) |
| 10 | 2 | `pict` |

Two classes (b furniture and i clutter) are **rect-based**; the other seven are
**point-based**. Empirically over the whole corpus:

* 22,606 point-based objects: `topLeft.h` ∈ [−1, 509], `topLeft.v` ∈ [0, 319].
* 8,834 rect-based objects: `left`/`top` minima 0/0, `right`/`bottom` maxima 512/322.
* Nothing exceeds `kRoomWide` 512 or `kTileHigh` 322.
* **Exactly one negative coordinate in the entire corpus**: Titanic room 161
  "I Want to be a Narrator", slot 9, a `kTV` at `topLeft` = (h = −1, v = 44). Raw 12
  bytes: `0065 002c ffff 0000 0000 0101` — `what` = 0x0065 = `kTV`, `topLeft.v` = 0x002c
  = 44, `topLeft.h` = 0xffff = −1.
* Objects are **not tile-aligned**: only 1703 of 22606 have `topLeft.h % 64 == 0`. The
  next most common residues mod 64 are 52 (677), 38 (616), 56 (566), 48 (514), 20 (508),
  40 (500), 50 (493), 16 (474), 58 (462). Level authors dragged objects freehand.

Verified against `Demo House` room 0 slot 0, at offset 926:

```
926  00 01 01 31 00 ab 01 0d 01 01 01 01
     ^what=0x0001 kFloorVent
           ^topLeft.v=0x0131=305  ^topLeft.h=0x00ab=171
                       ^distance=0x010d=269
                             ^initial=1 ^state=1 ^vector=1 (up) ^tall=1
```

A `kFloorVent` blowing **up** (`vector` bit 0), initially on, at (171, 305) — near the
floor (`kFloorLimit` = 312). Consistent with a room named "Air Vents".

### 2.5 Object codes — the complete vocabulary

The class of an object is its high nibble; the nine unions map onto nine code blocks
(`GliderPRO/Headers/GliderDefines.h:311-435`). All 117 codes below appear in the shipped
corpus; the `houses` column is how many of the 22 houses use the code at least once.

| code | name | class | corpus count | houses | header line |
|---|---|---|---|---|---|
| 0x01 | `kFloorVent` | a blower | 1458 | 19 | :311 |
| 0x02 | `kCeilingVent` | a | 28 | 7 | :312 |
| 0x03 | `kFloorBlower` | a | 83 | 8 | :313 |
| 0x04 | `kCeilingBlower` | a | 12 | 8 | :314 |
| 0x05 | `kSewerGrate` | a | 507 | 16 | :315 |
| 0x06 | `kLeftFan` | a | 45 | 12 | :316 |
| 0x07 | `kRightFan` | a | 54 | 14 | :317 |
| 0x08 | `kTaper` | a | 90 | 17 | :318 |
| 0x09 | `kCandle` | a | 180 | 18 | :319 |
| 0x0A | `kStubby` | a | 127 | 17 | :320 |
| 0x0B | `kTiki` | a | 58 | 12 | :321 |
| 0x0C | `kBBQ` | a | 43 | 11 | :322 |
| 0x0D | `kInvisBlower` | a | **2336** | 21 | :323 |
| 0x0E | `kGrecoVent` | a | 116 | **7** | :324 |
| 0x0F | `kSewerBlower` | a | 191 | 11 | :325 |
| 0x10 | `kLiftArea` | a | 716 | 16 | :326 |
| 0x11 | `kTable` | b furniture | 170 | 13 | :328 |
| 0x12 | `kShelf` | b | 389 | 19 | :329 |
| 0x13 | `kCabinet` | b | 457 | 16 | :330 |
| 0x14 | `kFilingCabinet` | b | 107 | 14 | :331 |
| 0x15 | `kWasteBasket` | b | 103 | 14 | :332 |
| 0x16 | `kMilkCrate` | b | 252 | 18 | :333 |
| 0x17 | `kCounter` | b | 287 | 16 | :334 |
| 0x18 | `kDresser` | b | 122 | 17 | :335 |
| 0x19 | `kDeckTable` | b | 30 | 10 | :336 |
| 0x1A | `kStool` | b | 91 | 13 | :337 |
| 0x1B | `kTrunk` | b | 101 | 16 | :338 |
| 0x1C | `kInvisObstacle` | b | 666 | 18 | :339 |
| 0x1D | `kManhole` | b | 40 | 13 | :340 |
| 0x1E | `kBooks` | b | 210 | 19 | :341 |
| 0x1F | `kInvisBounce` | b | **1670** | 15 | :342 |
| 0x21 | `kRedClock` | c prize | 140 | 16 | :344 |
| 0x22 | `kBlueClock` | c | 163 | 18 | :345 |
| 0x23 | `kYellowClock` | c | 330 | 19 | :346 |
| 0x24 | `kCuckoo` | c | 100 | 19 | :347 |
| 0x25 | `kPaper` | c | 283 | 18 | :348 |
| 0x26 | `kBattery` | c | 119 | 19 | :349 |
| 0x27 | `kBands` | c | 150 | 19 | :350 |
| 0x28 | `kGreaseRt` | c | 195 | 17 | :351 |
| 0x29 | `kGreaseLf` | c | 143 | 17 | :352 |
| 0x2A | `kFoil` | c | 83 | 14 | :353 |
| 0x2B | `kInvisBonus` | c | 327 | 18 | :354 |
| 0x2C | `kStar` | c | **69** | 21 | :355 |
| 0x2D | `kSparkle` | c | 486 | 17 | :356 |
| 0x2E | `kHelium` | c | 50 | 13 | :357 |
| 0x2F | `kSlider` | c | 354 | **7** | :358 |
| 0x31 | `kUpStairs` | d transport | 163 | 16 | :360 |
| 0x32 | `kDownStairs` | d | 163 | 16 | :361 |
| 0x33 | `kMailboxLf` | d | 44 | 10 | :362 |
| 0x34 | `kMailboxRt` | d | 34 | 10 | :363 |
| 0x35 | `kFloorTrans` | d | 244 | 15 | :364 |
| 0x36 | `kCeilingTrans` | d | 465 | 18 | :365 |
| 0x37 | `kDoorInLf` | d | 23 | 14 | :366 |
| 0x38 | `kDoorInRt` | d | 11 | 8 | :367 |
| 0x39 | `kDoorExRt` | d | 21 | 13 | :368 |
| 0x3A | `kDoorExLf` | d | 11 | 8 | :369 |
| 0x3B | `kWindowInLf` | d | 21 | 12 | :370 |
| 0x3C | `kWindowInRt` | d | 32 | 12 | :371 |
| 0x3D | `kWindowExRt` | d | 19 | 12 | :372 |
| 0x3E | `kWindowExLf` | d | 29 | 12 | :373 |
| 0x3F | `kInvisTrans` | d | 385 | 17 | :374 |
| 0x40 | `kDeluxeTrans` | d | 61 | 14 | :375 |
| 0x41 | `kLightSwitch` | e switch | 113 | 14 | :377 |
| 0x42 | `kMachineSwitch` | e | 79 | 13 | :378 |
| 0x43 | `kThermostat` | e | 108 | 15 | :379 |
| 0x44 | `kPowerSwitch` | e | 78 | **7** | :380 |
| 0x45 | `kKnifeSwitch` | e | 230 | 15 | :381 |
| 0x46 | `kInvisSwitch` | e | 635 | 18 | :382 |
| 0x47 | `kTrigger` | e | 239 | 13 | :383 |
| 0x48 | `kLgTrigger` | e | 81 | 11 | :384 |
| 0x49 | `kSoundTrigger` | e | 122 | 14 | :385 |
| 0x51 | `kCeilingLight` | f light | 160 | 16 | :387 |
| 0x52 | `kLightBulb` | f | 252 | 15 | :388 |
| 0x53 | `kTableLamp` | f | 70 | 15 | :389 |
| 0x54 | `kHipLamp` | f | 26 | 9 | :390 |
| 0x55 | `kDecoLamp` | f | 62 | 14 | :391 |
| 0x56 | `kFlourescent` | f | 115 | 12 | :392 |
| 0x57 | `kTrackLight` | f | 81 | 12 | :393 |
| 0x58 | `kInvisLight` | f | **1764** | 20 | :394 |
| 0x61 | `kShredder` | g appliance | 50 | 13 | :396 |
| 0x62 | `kToaster` | g | 140 | 17 | :397 |
| 0x63 | `kMacPlus` | g | 74 | 14 | :398 |
| 0x64 | `kGuitar` | g | 29 | 8 | :399 |
| 0x65 | `kTV` | g | 80 | 19 | :400 |
| 0x66 | `kCoffee` | g | 38 | 12 | :401 |
| 0x67 | `kOutlet` | g | 100 | 14 | :402 |
| 0x68 | `kVCR` | g | 26 | 10 | :403 |
| 0x69 | `kStereo` | g | 36 | 10 | :404 |
| 0x6A | `kMicrowave` | g | 56 | 17 | :405 |
| 0x6B | `kCinderBlock` | g | 43 | 9 | :406 |
| 0x6C | `kFlowerBox` | g | 35 | 11 | :407 |
| 0x6D | `kCDs` | g | 63 | 13 | :408 |
| 0x6E | `kCustomPict` | g | **4782** | 19 | :409 |
| 0x71 | `kBalloon` | h enemy | 500 | 18 | :411 |
| 0x72 | `kCopterLf` | h | 259 | 15 | :412 |
| 0x73 | `kCopterRt` | h | 155 | 16 | :413 |
| 0x74 | `kDartLf` | h | 151 | 15 | :414 |
| 0x75 | `kDartRt` | h | 117 | 12 | :415 |
| 0x76 | `kBall` | h | 212 | 18 | :416 |
| 0x77 | `kDrip` | h | 477 | 18 | :417 |
| 0x78 | `kFish` | h | 120 | 16 | :418 |
| 0x79 | `kCobweb` | h | 86 | 17 | :419 |
| 0x81 | `kOzma` | i clutter | 77 | 13 | :421 |
| 0x82 | `kMirror` | i | 667 | 20 | :422 |
| 0x83 | `kMousehole` | i | 174 | 12 | :423 |
| 0x84 | `kFireplace` | i | 33 | 11 | :424 |
| 0x85 | `kFlower` | i | 547 | 17 | :425 |
| 0x86 | `kWallWindow` | i | 195 | 15 | :426 |
| 0x87 | `kBear` | i | 169 | 19 | :427 |
| 0x88 | `kCalendar` | i | 58 | 15 | :428 |
| 0x89 | `kVase1` | i | 72 | 16 | :429 |
| 0x8A | `kVase2` | i | 86 | 16 | :430 |
| 0x8B | `kBulletin` | i | 38 | 13 | :431 |
| 0x8C | `kCloud` | i | **1860** | 19 | :432 |
| 0x8D | `kFaucet` | i | 28 | 11 | :433 |
| 0x8E | `kRug` | i | 81 | 18 | :434 |
| 0x8F | `kChimes` | i | 54 | 17 | :435 |

Codes `0x00`, `0x20`, `0x30`, `0x50`, `0x60`, `0x70`, `0x80` and `0x90+` are unused;
`0x1F`→`0x21` and `0x2F`→`0x31` skip the class boundary. Class totals:

| class | union | codes | corpus objects | share |
|---|---|---|---|---|
| a | `blowerType` | 0x01–0x10 (16) | 6044 | 19.2 % |
| b | `furnitureType` | 0x11–0x1F (15) | 4695 | 14.9 % |
| c | `bonusType` | 0x21–0x2F (15) | 2992 | 9.5 % |
| d | `transportType` | 0x31–0x40 (16) | 1726 | 5.5 % |
| e | `switchType` | 0x41–0x49 (9) | 1685 | 5.4 % |
| f | `lightType` | 0x51–0x58 (8) | 2530 | 8.0 % |
| g | `applianceType` | 0x61–0x6E (14) | 5552 | 17.7 % |
| h | `enemyType` | 0x71–0x79 (9) | 2077 | 6.6 % |
| i | `clutterType` | 0x81–0x8F (15) | 4139 | 13.2 % |
| | | **117** | **31440** | 100 % |

`kMaxMasterObjects` = 216 (`GliderDefines.h:266`) caps the number of *live* dynamic
objects the engine tracks at once, and `kMaxStars` = 4 (`:263`) caps simultaneous star
sprites — neither is a per-house limit.

### 2.6 Links: `where` / `who`

Any object in classes d (transport) and e (switch) can carry a link. The encoding is in
`GliderPRO/Sources/Link.c`:

```c
short MergeFloorSuite (short floor, short suite)   /* Link.c:34-37 */
{
    return ((suite * 100) + floor);
}

void ExtractFloorSuite (short combo, short *floor, short *suite)  /* Link.c:41-53 */
{
    if ((*thisHouse)->version < 0x0200) {      /* version 1 layout */
        *floor = (combo / 100) - kNumUndergroundFloors;
        *suite =  combo % 100;
    } else {                                    /* version 2 layout */
        *floor = (combo % 100) - kNumUndergroundFloors;
        *suite =  combo / 100;
    }
}
```

**The `+8` is not in `MergeFloorSuite`.** Callers add `kNumUndergroundFloors` themselves;
`DoLink` (`Link.c:270-319`) does `floor += kNumUndergroundFloors;` at `Link.c:277` before
merging. So for a version-2 house:

```
where = suite * 100 + (floor + 8)      encode
floor = (where % 100) - 8              decode
suite =  where / 100
```

and because legal floors are −7..56 (`HouseLegal.c:804-805`, suites 0..127 at `HouseLegal.c:814-815`), the low two digits of a
legal `where` are 01..64 and the high digits are the suite 0..127. The version-1 layout
swapped the roles — which is why `ExtractFloorSuite` re-reads `(*thisHouse)->version`. No shipped
house is version 1, so only the version-2 branch is exercised.

`DoUnlink` (`Link.c:325-361`) writes the "no link" state, which is
`linkRoom = -1; linkObject = 255;` (`Link.c:246-247`):

* `where == -1` → unlinked.
* `who == 255` → no target object (the link is inert even if `where` is set).

The observed "points at nothing" sentinel is `where == -100`, which is
`MergeFloorSuite(0, kRoomIsEmpty)` = `(-1)*100 + 0`: a link whose destination room was
deleted. 165 of the 189 dangling links in the corpus are exactly this, all with
`who == 255`, i.e. harmless (§11.3).

### 2.7 The size identity, and the one house that breaks it

```
dataForkBytes == sizeof(houseType) + sizeof(roomType) * nRooms
              == 866 + 348 * nRooms
```

`ValidateNumberOfRooms` (`GliderPRO/Sources/HouseLegal.c:621-644`) enforces the inverse:
`nRooms = (GetHandleSize(thisHouse) - sizeof(houseType)) / sizeof(roomType)`.

Verified: **21 of 22 houses satisfy the identity exactly.** The exception is `Sampler`:

| house | data fork | 866 + 348·`nRooms` | slack |
|---|---|---|---|
| Sampler | 1564 | 866 + 348×2 = 1562 | **+2** |

Two trailing bytes past the last room. `ReadHouse` does not care (it allocates
`byteCount` and only reads `nRooms` from the header), and `ValidateNumberOfRooms` would
compute `(1564 - 866) / 348 = 2` by integer truncation and agree. A port must therefore
**truncate, not reject**: read `nRooms` rooms and ignore any trailing bytes.

Room-array capacity: `kMaxNumRoomsH` (128 suites) × `kMaxNumRoomsV` (64 floors) = 8192
grid cells, but `nRooms` is a `short`, and `CheckDuplicateFloorSuite`
(`HouseLegal.c:646-686`) uses a `char[8192]` pigeonhole indexed by
`bitPlace = ((floor + 7) * 128) + suite`. Note the **`+7` here versus
`kNumUndergroundFloors` = 8 in the link encoding** — the validator's legal floor range is
−7..56 (64 values), while the link encoding reserves floor + 8 ∈ 1..64. The two agree on
"64 floors" but disagree on the origin by one; that is a latent inconsistency in the
original, not a transcription error here.

---

## 3. Corpus inventory

### 3.1 Main inventory table

Every column measured from the decoded BinHex payload. `floors` and `suites` count
*occupied* grid lines, with the observed range in parentheses. `objects` counts live
object slots (`what != kObjectIsEmpty`). `distinct types` counts distinct `what` codes.
`start room` is `firstRoom` with the room's name. Authors are from `README.md:6-15`.

| # | House file | .binhex B | data fork B | rsrc fork B | ver | rooms | floors | suites | objects | distinct types | start room | author |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | `Art Museum.binhex` | 10056249 | 38798 | 7476159 | 0x0200 | 109 | 11 (-1..9) | 16 (54..71) | 569 | 59 | 91 "Art Museum" | unattributed (not in README credits) |
| 2 | `CD Demo House.binhex` | 2205334 | 72554 | 1612342 | 0x0200 | 206 | 16 (-2..14) | 45 (2..53) | 1533 | 100 | 70 "Welcome…" | John Calhoun & Kim Money |
| 3 | `California or Bust!.binhex` | 344163 | 6434 | 259542 | 0x0200 | 16 | 1 (1..1) | 16 (55..70) | 308 | 55 | 14 "Leaving the Heartland" | unattributed (not in README credits) |
| 4 | `Castle o' the Air.binhex` | 437515 | 30446 | 306364 | 0x0200 | 85 | 10 (-1..8) | 13 (63..75) | 620 | 50 | 33 "West Courtyard" | unattributed (not in README credits) |
| 5 | `Davis Station.binhex` | 2499538 | 23486 | 1871320 | 0x0200 | 65 | 14 (-6..8) | 20 (58..77) | 591 | 69 | 4 "Let's Roll" | Jonathan Chin (alias Paul Finn) & John Calhoun |
| 6 | `Demo House.binhex` | 664651 | 16526 | 491757 | 0x0200 | 45 | 7 (-1..5) | 10 (62..71) | 138 | 48 | 0 "Air Vents" | John Calhoun & Kim Money |
| 7 | `Empty House.binhex` | 19343 | 13046 | 2670 | 0x0200 | 35 | 6 (-1..4) | 9 (62..70) | 78 | 15 | 0 "Untitled Room" | unattributed (not in README credits) |
| 8 | `Fun House.binhex` | 895562 | 15830 | 662443 | 0x0200 | 43 | 5 (1..5) | 14 (56..71) | 406 | 54 | 29 "Mission Impossible?" | unattributed (not in README credits) |
| 9 | `Grand Prix.binhex` | 1882081 | 61766 | 1347764 | 0x0200 | 175 | 12 (-2..9) | 27 (53..79) | 1281 | 100 | 127 "Fast!" | Jonathan Chin (alias Paul Finn) |
| 10 | `ImagineHouse PRO II.binhex` | 1029380 | 97958 | 677770 | 0x0200 | 279 | 15 (-3..11) | 39 (6..95) | 1814 | 114 | 1 "Falling from the Sky" | Jonathan Chin (alias Paul Finn) |
| 11 | `In The Mirror.binhex` | 243295 | 34622 | 151870 | 0x0200 | 97 | 12 (-2..9) | 17 (55..71) | 795 | 97 | 6 "Let's Begin" | Jonathan Chin (alias Paul Finn) |
| 12 | `Land of Illusion.binhex` | 659819 | 106310 | 401793 | 0x0200 | 303 | 20 (-2..17) | 29 (49..77) | 1816 | 85 | 43 "Magic Mirror" | Ward Hartenstein |
| 13 | `Leviathan.binhex` | 2733762 | 165122 | 1901282 | 0x0200 | 472 | 26 (-6..19) | 58 (0..94) | 3335 | 114 | 39 "Black" | Jonathan Chin (alias Paul Finn) |
| 14 | `Metropolis.binhex` | 1324749 | 45062 | 946907 | 0x0200 | 127 | 14 (-1..12) | 15 (54..68) | 809 | 84 | 8 "Courtesy Desk" | Jonathan Chin (alias Paul Finn) & John Calhoun |
| 15 | `Nemo's Market.binhex` | 815640 | 44018 | 590602 | 0x0200 | 124 | 3 (0..2) | 42 (54..95) | 836 | 68 | 0 "Welcome to Nemo's Market!" | Ward Hartenstein |
| 16 | `Rainbow's End.binhex` | 513683 | 78470 | 316065 | 0x0200 | 223 | 14 (-7..6) | 36 (52..87) | 1673 | 110 | 30 "Out Of Thin Air" | Ward Hartenstein |
| 17 | `Sampler.binhex` | 2169 | 1564 | 286 | 0x0200 | 2 | 1 (1..1) | 2 (66..67) | 11 | 9 | 1 "Welcome" | unattributed (not in README credits) |
| 18 | `Slumberland.binhex` | 1552921 | 134150 | 1041112 | 0x0200 | 383 | 18 (-7..10) | 48 (0..86) | 2996 | 105 | 126 "Welcome…" | John Calhoun, Jonathan Chin, Steve Sullivan, Ward Hartenstein |
| 19 | `SpacePods.binhex` | 1005729 | 140762 | 636844 | 0x0200 | 402 | 35 (4..39) | 18 (110..127) | 5840 | 60 | 259 "Mission Statement" | Ward Hartenstein |
| 20 | `Teddy World.binhex` | 4434970 | 185654 | 3107348 | 0x0200 | 531 | 24 (-5..26) | 101 (1..113) | 3521 | 112 | 0 "Wanna come in and play?" | Shawn Brenneman |
| 21 | `The Asylum Pro.binhex` | 727114 | 49586 | 494107 | 0x0200 | 140 | 14 (0..13) | 10 (63..72) | 1096 | 103 | 20 "Greetings!" | Steve Sullivan |
| 22 | `Titanic.binhex` | 1196597 | 73250 | 845727 | 0x0200 | 208 | 13 (-7..5) | 28 (46..73) | 1374 | 89 | 92 "Chute!" | Jonathan Chin (alias Paul Finn) & John Calhoun |

Corpus totals: **4070 rooms** (0 placeholders), **31440 live objects** of 97680 slots
(32.2 % fill), 117 distinct object codes, 1,435,414 data-fork bytes, 25,142,074
resource-fork bytes, **69 stars**.

### 3.2 Authorship

From `GliderPRO/README.md:6-15`:

| author | houses |
|---|---|
| John Calhoun & Kim Money | Demo House, CD Demo House (`README.md:7`) |
| Jonathan Chin (alias Paul Finn) & John Calhoun | Davis Station, Metropolis, Titanic (`:8`) |
| Jonathan Chin (alias Paul Finn) | Grand Prix, Leviathan, ImagineHouse PRO II, In The Mirror (`:9`) |
| Ward Hartenstein | Land of Illusion, Nemo's Market, Rainbow's End, SpacePods (`:10`) |
| John Calhoun (house 1 + top of house 4), Jonathan Chin (house 2), Steve Sullivan (house 3), Ward Hartenstein (bottom of house 4) | Slumberland (`:11`) |
| Shawn Brenneman | Teddy World (`:12`) |
| Steve Sullivan | The Asylum Pro (`:13`) |
| **unattributed** | **Art Museum, California or Bust!, Castle o' the Air, Empty House, Fun House, Sampler** |

Also credited: `PICT` 3975 (Ozma) after an illustration by John R. Neill (`:14`), and
`PICT` 153 (About box) after a Winsor McCay *Little Nemo* comic (`:15`). Both are
*application* resources, not house resources — but note that **Teddy World shadows PICT
3975** (§1.5), i.e. it replaces the Ozma sprite with its own.

Signature per author, measured (mean over that author's houses):

| author | houses | mean rooms | mean obj/room | mean distinct types | notable habit |
|---|---|---|---|---|---|
| Ward Hartenstein | 4 (+ part of Slumberland) | 263 | 9.3 | 81 | invisible everything: `kInvisBlower`, `kInvisBounce`, `kInvisLight` dominate his top-5 in all four |
| Jonathan Chin | 4 (+2 co-authored, + part of Slumberland) | 256 | 7.4 | 105 | widest vocabularies in the corpus (114, 114, 100, 97); `kCloud` is his #1 in Grand Prix, ImagineHouse, In The Mirror and Leviathan |
| John Calhoun (solo/co) | 2 + 3 co | 126 | 6.9 | 74 | small, tightly-linked, tutorialised; the only houses with non-empty `banner` text in every case |
| Shawn Brenneman | 1 (Teddy World) | 531 | 6.6 | 112 | largest house; owns 333 of the corpus's 354 `kSlider` and 85 of its 116 `kGrecoVent` |
| Steve Sullivan | 1 (+ part of Slumberland) | 140 | 7.8 | 103 | densest switch/trigger usage per room outside Land of Illusion |

### 3.3 Object vocabulary — top 15 types per house

| House | top 15 `what` codes by count |
|---|---|
| Art Museum | kInvisLight 93, kCustomPict 74, kFloorVent 65, kCabinet 20, kInvisSwitch 19, kInvisBonus 18, kShelf 16, kSoundTrigger 15, kFlower 15, kLiftArea 13, kYellowClock 11, kCeilingTrans 11, kDeluxeTrans 11, kCandle 10, kCopterLf 10 |
| CD Demo House | kCustomPict 647, kFloorVent 70, kInvisLight 68, kInvisBlower 51, kMirror 46, kFlower 34, kCloud 33, kBalloon 26, kLiftArea 25, kInvisBounce 24, kInvisObstacle 23, kWallWindow 20, kCounter 18, kDrip 17, kCeilingTrans 16 |
| California or Bust! | kCustomPict 83, kLiftArea 32, kInvisBlower 25, kFlower 25, kInvisLight 14, kFish 12, kCloud 12, kBear 10, kCinderBlock 6, kBall 5, kMirror 5, kWallWindow 5, kRightFan 4, kTaper 4, kCandle 4 |
| Castle o' the Air | kCloud 115, kSewerGrate 65, kSparkle 65, kInvisLight 52, kInvisBounce 37, kInvisBlower 28, kDrip 28, kInvisBonus 24, kCabinet 20, kCeilingTrans 18, kBalloon 17, kCopterLf 13, kPaper 10, kFloorTrans 9, kDartLf 8 |
| Davis Station | kInvisBounce 79, kCustomPict 66, kCloud 62, kFlower 55, kInvisLight 41, kDrip 22, kSewerGrate 19, kSewerBlower 18, kInvisBlower 16, kLiftArea 16, kCopterLf 13, kLightBulb 12, kBalloon 12, kMilkCrate 11, kInvisObstacle 10 |
| Demo House | kFlower 16, kFloorVent 15, kInvisBlower 13, kCustomPict 8, kWallWindow 7, kInvisObstacle 6, kInvisLight 5, kCloud 5, kShelf 4, kMilkCrate 4, kStool 3, kGreaseRt 3, kBalloon 3, kSewerGrate 2, kTaper 2 |
| Empty House | kFloorVent 28, kSewerGrate 11, kCeilingLight 10, kLightBulb 6, kInvisBlower 4, kUpStairs 4, kDownStairs 4, kManhole 3, kFlourescent 2, kStar 1, kDoorInLf 1, kDoorExRt 1, kWindowInRt 1, kWindowExLf 1, kTableLamp 1 |
| Fun House | kFloorVent 39, kCabinet 37, kCustomPict 34, kShelf 31, kInvisSwitch 29, kBall 23, kInvisTrans 19, kToaster 15, kCloud 15, kLiftArea 14, kMachineSwitch 14, kCeilingTrans 9, kCeilingLight 9, kInvisLight 9, kBooks 8 |
| Grand Prix | kCloud 176, kInvisBounce 111, kLiftArea 104, kFloorVent 69, kInvisLight 58, kInvisSwitch 47, kBalloon 44, kCustomPict 41, kInvisBlower 34, kLgTrigger 33, kSewerGrate 32, kDrip 31, kCabinet 27, kCeilingTrans 25, kCandle 18 |
| ImagineHouse PRO II | kCloud 304, kFloorVent 112, kInvisBlower 105, kInvisBounce 103, kInvisLight 63, kSewerGrate 46, kShelf 39, kBalloon 39, kCeilingTrans 38, kCabinet 34, kSparkle 34, kCustomPict 34, kCopterLf 32, kCounter 31, kBooks 30 |
| In The Mirror | kCloud 175, kInvisSwitch 55, kMirror 50, kFoil 37, kFloorVent 29, kBalloon 23, kSewerGrate 22, kInvisBlower 20, kInvisBounce 20, kInvisLight 18, kCustomPict 17, kCopterLf 15, kMilkCrate 14, kBlueClock 14, kInvisBonus 10 |
| Land of Illusion | kInvisBlower 368, kInvisLight 210, kInvisObstacle 124, kInvisSwitch 118, kTrigger 100, kInvisBounce 82, kLiftArea 49, kMirror 47, kFloorVent 43, kSparkle 42, kCustomPict 41, kCeilingTrans 38, kInvisTrans 38, kCloud 35, kFloorTrans 33 |
| Leviathan | kCloud 381, kFloorVent 218, kInvisBlower 176, kCustomPict 131, kInvisBounce 113, kInvisLight 103, kBalloon 102, kSparkle 82, kShelf 77, kCeilingTrans 70, kSewerGrate 67, kCounter 64, kCabinet 58, kCopterLf 58, kTable 54 |
| Metropolis | kCustomPict 68, kFloorVent 62, kInvisLight 54, kInvisBlower 39, kBalloon 32, kDrip 29, kCloud 27, kCeilingTrans 26, kSewerGrate 23, kLiftArea 22, kTrackLight 21, kShelf 19, kCabinet 19, kInvisSwitch 18, kCopterRt 16 |
| Nemo's Market | kCustomPict 358, kInvisLight 94, kFloorVent 55, kInvisObstacle 49, kFlourescent 19, kCobweb 17, kRedClock 14, kYellowClock 14, kInvisBonus 13, kInvisSwitch 13, kCeilingTrans 12, kBlueClock 11, kSoundTrigger 11, kFish 11, kStool 6 |
| Rainbow's End | kInvisBounce 241, kInvisBlower 219, kFlower 117, kInvisTrans 55, kCloud 50, kInvisObstacle 49, kInvisLight 48, kCustomPict 46, kLightBulb 41, kInvisSwitch 40, kLiftArea 39, kFloorVent 32, kYellowClock 26, kInvisBonus 25, kBlueClock 24 |
| Sampler | kMailboxLf 2, kDoorInLf 2, kFloorVent 1, kBBQ 1, kStar 1, kFloorTrans 1, kCeilingTrans 1, kDoorExRt 1, kCeilingLight 1 |
| Slumberland | kInvisBlower 265, kFloorVent 254, kCloud 156, kSewerGrate 131, kDrip 97, kCeilingTrans 90, kFlower 86, kMousehole 84, kInvisLight 78, kYellowClock 73, kCabinet 71, kShelf 69, kLightBulb 66, kCounter 56, kInvisBonus 48 |
| SpacePods | kCustomPict 2844, kInvisBlower 612, kInvisBounce 593, kInvisLight 384, kMirror 266, kInvisSwitch 120, kLiftArea 97, kInvisObstacle 89, kWallWindow 74, kCabinet 71, kSparkle 58, kSewerBlower 56, kPowerSwitch 53, kDrip 47, kLightBulb 36 |
| Teddy World | kSlider 333, kInvisObstacle 195, kLiftArea 186, kInvisLight 174, kInvisBlower 168, kInvisBounce 166, kCustomPict 160, kMirror 134, kInvisTrans 120, kFloorVent 117, kKnifeSwitch 98, kCloud 94, kGrecoVent 85, kBalloon 85, kFlower 75 |
| The Asylum Pro | kInvisBlower 160, kFloorVent 123, kCloud 87, kInvisSwitch 38, kCeilingLight 32, kCustomPict 32, kInvisLight 31, kInvisTrans 28, kMirror 28, kInvisBounce 25, kTrigger 24, kFlower 24, kShelf 22, kSparkle 22, kCeilingTrans 22 |
| Titanic | kInvisLight 167, kFloorVent 119, kCloud 103, kCustomPict 94, kInvisObstacle 76, kLiftArea 71, kYellowClock 51, kCeilingTrans 40, kBalloon 40, kInvisBounce 34, kMilkCrate 33, kInvisBonus 31, kDrip 28, kSparkle 23, kInvisBlower 21 |

### 3.4 Backgrounds and embedded resources

"custom art?" is true iff the house's resource fork contains at least one `PICT`;
"custom sound?" iff at least one `snd `.

| House | built-in bg IDs | custom bg IDs | `PICT` | `snd ` | `bnds` | custom art? | custom sound? |
|---|---|---|---|---|---|---|---|
| Art Museum | 2002, 2017 | 3000-3053 | 76 | 10 | 0 | yes | yes |
| CD Demo House | 2000-2001, 2004-2005, 2007-2008, 2011-2015, 2017 | 3009, 3011, 3100, 3102-3104, 3211, 3301-3302, 3306-3308, 3314 | 116 | 10 | 0 | yes | yes |
| California or Bust! | 2013 | 3000 | 30 | 3 | 0 | yes | yes |
| Castle o' the Air | 2011, 2015 | 3000-3003, 3300-3304 | 12 | 0 | 9 | yes | no |
| Davis Station | 2000-2001, 2003, 2011, 2015 | 3207-3236 | 54 | 5 | 0 | yes | yes |
| Demo House | 2011-2017 | 3000-3005, 3300-3303 | 19 | 1 | 10 | yes | yes |
| Empty House | 2000-2004, 2011-2015 | — | 0 | 0 | 0 | **no** | **no** |
| Fun House | 2000-2001, 2003-2004, 2012, 2014-2015 | 3000-3004, 3352-3353 | 26 | 0 | 0 | yes | no |
| Grand Prix | 2000-2008, 2010-2012, 2014-2015 | 3001-3016, 3301-3303, 3305-3312 | 70 | 3 | 0 | yes | yes |
| ImagineHouse PRO II | **2000-2017 (all 18)** | 3001-3005, 3300-3303, 3305-3311 | 45 | 2 | 14 | yes | yes |
| In The Mirror | 2000-2003, 2005-2007, 2011-2012, 2014-2017 | 3300-3301 | 15 | 2 | 0 | yes | yes |
| Land of Illusion | 2011-2012, 2014-2015, 2017 | 3001-3005, 3301-3302 | 20 | 0 | 5 | yes | no |
| Leviathan | **2000-2017 (all 18)** | 3000-3004, 3300-3309, 3311-3312, 3314-3319 | 53 | 12 | 4 | yes | yes |
| Metropolis | 2000-2001, 2011-2012, 2014-2015, 2017 | 3000, 3002-3003, 3007-3011, 3300-3306 | 39 | 0 | 0 | yes | no |
| Nemo's Market | **—** | 3301-3304 | 73 | 5 | 0 | yes | yes |
| Rainbow's End | 2000-2013, 2015-2017 | 3001, 3301-3304 | 17 | 1 | 6 | yes | yes |
| Sampler | 2003, 2012 | — | 0 | 0 | 0 | **no** | **no** |
| Slumberland | 2000-2008, 2010-2017 | 3000-3010, 3300-3308 | 20 | 0 | 20 | yes | no |
| SpacePods | **—** | 3001-3010 | 49 | 3 | 0 | yes | yes |
| Teddy World | 2000-2001, 2003-2006, 2008-2009, 2011-2017 | 3000-3028 | 120 | 0 | 0 | yes | no |
| The Asylum Pro | 2000-2001, 2003-2007, 2009-2012, 2014-2015 | 3000-3003 | 17 | 0 | 2 | yes | no |
| Titanic | 2002, 2011, 2015 | 3300-3312, 3314-3327 | 48 | 6 | 0 | yes | yes |

Extremes worth naming:

* **Only 2 houses embed no custom art and no custom sound at all**: `Empty House`
  (resource fork 2670 bytes, entirely a Finder icon family) and `Sampler` (286 bytes).
  Both are templates, not levels.
* **Only 2 houses use *no* built-in background**: `Nemo's Market` (all 124 rooms on
  user PICTs 3301-3304) and `SpacePods` (all 402 rooms on 3001-3010). These are the
  fully-reskinned houses.
* **Only 2 houses use all 18 built-in backgrounds**: `ImagineHouse PRO II` and
  `Leviathan` — both by Jonathan Chin.
* Largest resource fork: `Art Museum`, 7,476,159 bytes for 76 `PICT`s — 91 of its 109
  rooms use a unique painting as the background (IDs 3000-3053).
* Largest `.binhex`: `Art Museum` at 10,056,249 bytes; largest data fork:
  `Teddy World` at 185,654 bytes (531 rooms).

### 3.5 Header/flags summary

| House | `flags` | ward b0 | phone b1 | starCount (!b2) | locked | `hasGame` | `initial` (v, h) | `firstRoom` | (floor, suite) | size ok |
|---|---|---|---|---|---|---|---|---|---|---|
| Art Museum | **6** | F | T | **F** | T | 0 | (56, 231) | 91 | (4, 55) | ✔ |
| CD Demo House | 2 | F | T | T | T | 0 | (105, 223) | 70 | (2, 2) | ✔ |
| California or Bust! | 2 | F | T | T | F | 0 | (62, 308) | 14 | (1, 69) | ✔ |
| Castle o' the Air | 0 | F | F | T | F | 0 | (124, 239) | 33 | (1, 68) | ✔ |
| Davis Station | 2 | F | T | T | T | 0 | (95, 189) | 4 | (1, 59) | ✔ |
| Demo House | 0 | F | F | T | T | 0 | (107, 49) | 0 | (1, 63) | ✔ |
| Empty House | 0 | F | F | T | F | 0 | (64, 83) | 0 | (1, 64) | ✔ |
| Fun House | 0 | F | F | T | F | 0 | (78, 42) | 29 | (4, 58) | ✔ |
| Grand Prix | 0 | F | F | T | T | 0 | (58, 268) | 127 | (1, 54) | ✔ |
| ImagineHouse PRO II | 0 | F | F | T | T | **1** | (35, 211) | 1 | (6, 64) | ✔ |
| In The Mirror | 0 | F | F | T | T | 0 | (50, 424) | 6 | (2, 66) | ✔ |
| Land of Illusion | 2 | F | T | T | F | 0 | (163, 245) | 43 | (15, 54) | ✔ |
| Leviathan | 0 | F | F | T | T | 0 | (44, 229) | 39 | (18, 55) | ✔ |
| Metropolis | 0 | F | F | T | T | 0 | (24, 245) | 8 | (1, 63) | ✔ |
| Nemo's Market | 2 | F | T | T | T | 0 | (51, 342) | 0 | (1, 55) | ✔ |
| Rainbow's End | 2 | F | T | T | T | 0 | (200, 237) | 30 | (5, 62) | ✔ |
| Sampler | 0 | F | F | T | F | 0 | (85, 384) | 1 | (1, 66) | **+2 B** |
| Slumberland | 0 | F | F | T | T | 0 | (7, 361) | 126 | (1, 56) | ✔ |
| SpacePods | 2 | F | T | T | T | 0 | (72, 30) | 259 | (8, 112) | ✔ |
| Teddy World | 0 | F | F | T | T | 0 | (41, 362) | 0 | (1, 7) | ✔ |
| The Asylum Pro | 0 | F | F | T | T | 0 | (27, 78) | 20 | (1, 63) | ✔ |
| Titanic | 0 | F | F | T | F | **1** | (97, 39) | 92 | (−1, 61) | ✔ |

Note **Titanic's start room is on floor −1** — the only house that begins the player
below ground level (fitting: the room is named "Chute!").

### 3.6 Reachability, stars and links at a glance

`reach/real` is BFS-forward-reachable rooms from `firstRoom` under a static model
(walk through open left/right/top/bottom boundaries, plus staircases and resolved links);
`max hop` is the BFS eccentricity in that graph.

| House | rooms | obj/room mean | median | max | full(24) | dark | no-floor | no-ceiling | structure | stars | link slots | dangling | reach/real | max hop |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Art Museum | 109 | 5.2 | 3 | 23 | 0 | 9 | 54 | 67 | 33 | 6 | 61 | 0 | 76/109 | 36 |
| CD Demo House | 206 | 7.4 | 6 | 24 | 7 | 37 | 96 | 116 | 65 | 9 | 101 | 0 | 129/206 | 11 |
| California or Bust! | 16 | **19.2** | 21 | 24 | 6 | 0 | 0 | 2 | 0 | 1 | 0 | 0 | 16/16 | 14 |
| Castle o' the Air | 85 | 7.3 | 4 | 24 | 1 | 3 | 31 | 46 | 24 | 4 | 35 | 12 | 77/85 | 10 |
| Davis Station | 65 | 9.1 | 7 | 24 | 6 | 0 | 23 | 41 | 6 | 4 | 24 | 0 | 57/65 | 23 |
| Demo House | 45 | 3.1 | 0 | 23 | 0 | 0 | 16 | 26 | 11 | 1 | 2 | 0 | 33/45 | 15 |
| Empty House | 35 | **2.2** | 3 | 7 | 0 | 0 | 7 | 16 | 16 | 1 | 0 | 0 | 35/35 | 11 |
| Fun House | 43 | 9.4 | 7 | 24 | 4 | 0 | 13 | 23 | 25 | **0** | 91 | 0 | **5/43** | **2** |
| Grand Prix | 175 | 7.3 | 6 | 24 | 9 | 1 | 50 | 78 | 70 | 3 | 152 | 0 | 149/175 | 35 |
| ImagineHouse PRO II | 279 | 6.5 | 6 | 24 | 5 | 1 | 87 | 119 | 81 | 3 | 137 | 1 | 218/279 | 46 |
| In The Mirror | 97 | 8.2 | 7 | 24 | 2 | 0 | 45 | 62 | 26 | 1 | 75 | 0 | 96/97 | 35 |
| Land of Illusion | 303 | 6.0 | 1 | 24 | 22 | 11 | 218 | 202 | 40 | 5 | 359 | 48 | 278/303 | 26 |
| Leviathan | 472 | 7.1 | 6 | 24 | 5 | 10 | 217 | 286 | 170 | 6 | 254 | 23 | 458/472 | **59** |
| Metropolis | 127 | 6.4 | 5 | 20 | 0 | 4 | 47 | 58 | 56 | 4 | 92 | 0 | 110/127 | 39 |
| Nemo's Market | 124 | 6.7 | 1 | 24 | 18 | 0 | 82 | 12 | 0 | 5 | 36 | 0 | 81/124 | 39 |
| Rainbow's End | 223 | 7.5 | 3 | 24 | 17 | 1 | 95 | 118 | 29 | 5 | 174 | 36 | 187/223 | 24 |
| Sampler | 2 | 5.5 | 6 | 7 | 0 | 0 | 0 | 1 | 1 | 1 | 4 | 0 | 2/2 | 1 |
| Slumberland | 383 | 7.8 | 7 | 24 | 4 | **29** | 102 | 149 | 141 | 6 | 302 | **69** | 357/383 | 57 |
| SpacePods | 402 | 14.5 | 14 | 24 | **77** | 0 | 402 | 401 | 0 | 1 | 282 | 0 | **402/402** | 24 |
| Teddy World | 531 | 6.6 | 1 | 24 | 35 | **57** | 269 | 285 | 182 | 1 | 377 | 0 | 401/531 | 44 |
| The Asylum Pro | 140 | 7.8 | 6 | 24 | 3 | 5 | 40 | 73 | 79 | 1 | 141 | 0 | **140/140** | 21 |
| Titanic | 208 | 6.6 | 5 | 24 | 1 | 19 | 78 | 82 | 0 | 1 | 97 | 0 | 165/208 | 30 |

---

## 4. Per-house detail

Each subsection below is machine-generated from the same parse. "BFS depth" always means
hop count in the static reachability graph rooted at `firstRoom`. `CountTotalHousePoints`
is a faithful port of `GliderPRO/Sources/HouseInfo.c:51-104`:

```
1. pointTotal = RealRoomNumberCount() * kRoomVisitScore;     /* 100 per room */   (:57)
2. for each non-empty room, for each of the 24 object slots:
3.     kRedClock    -> += 100     (kRedClockPoints,    GliderDefines.h:537)
4.     kBlueClock   -> += 300     (kBlueClockPoints,   :538)
5.     kYellowClock -> += 500     (kYellowClockPoints, :539)
6.     kCuckoo      -> += 1000    (kCuckooClockPoints, :540)
7.     kStar        -> += 5000    (kStarPoints,        :541)
8.     kInvisBonus  -> += objects[h].data.c.points     /* author-chosen */
```

Note it iterates **all 24 slots**, not `numObjects` — but since `numObjects` matches the
live count in all 4070 shipped rooms, the two agree here.

### 4.1 Art Museum

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Art Museum.binhex`, 10056249 bytes; BinHex name `Art Museum`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 38798 bytes (866 + 348 x 109 = 38798) |
| resource fork | 7476159 bytes |
| `.mov` sidecar | `Art Museum.mov`, 28232 bytes |
| author (README.md:6-13) | unattributed (not in README credits) |
| `version` | 0x0200 |
| `unusedShort` | -30082 |
| `timeStamp` | 0x2C808C35 -> `& 1` = 1, house LOCKED |
| `flags` | 6 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=False |
| `initial` | v=56 h=231 -> glider spawn rect {top 56, left 231, bottom 76, right 279} |
| `firstRoom` | 91 = "Art Museum" at (floor 4, suite 55) |
| `nRooms` | 109 (real 109, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 255 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `Thank you for visiting the museum. Donations are welcome. Good day.` |
| `highScores.banner` | `Your Message Here` |
| top score | 18500 by `Your Name`, rooms visited 22 |

**Geometry.** 11 occupied floors (-1..9), 16 occupied suites (54..71); grid occupancy 109 / (11 x 16) = 0.62.

- rooms per floor (floor:count, north-up): 9:3, 8:3, 7:13, 6:10, 5:13, 4:13, 3:16, 2:13, 1:13, 0:6, -1:6
- rooms per suite (suite:count, west-first): 54:6, 55:6, 56:6, 59:7, 60:7, 61:7, 62:9, 63:9, 64:9, 65:9, 66:9, 67:9, 68:7, 69:3, 70:3, 71:3

**Backgrounds.** 56 distinct IDs over 109 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2002 | kBasement | 9 |
| 2017 | kStars | 9 |

- user-art backgrounds (`>= kUserBackground` 3000): 54 distinct IDs over 91 rooms - 3000-3053
  - structure sub-range 3000-3299: 54 IDs / 91 rooms; open sub-range 3300-3799: 0 IDs / 0 rooms
  - most-used: 3049 (10 rooms), 3050 (9 rooms), 3001 (4 rooms), 3031 (4 rooms), 3019 (3 rooms), 3020 (3 rooms), 3032 (3 rooms), 3051 (3 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 22 rooms; `bounds`==0 in 18, non-zero in 91; `leftStart` mode 32 (109 rooms), `rightStart` mode 32 (107 rooms); rooms named exactly "Untitled Room": 0; distinct room names 76 over 109 rooms (reuse ratio 1.43).

- most reused names: `Column` x11, `Enlightenment` x9, `Up on the Roof` x8, `Capstone` x3, `Ground Floor` x2

**Objects.** 569 live of 2616 slots (21.8% fill); per room mean 5.22, median 3, max 23; 0 rooms at the 24-object ceiling, 16 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 103 | 50 | 75 | 26 | 50 | 99 | 85 | 46 | 35 |

Top 15 `what` codes: `kInvisLight` 0x58 93, `kCustomPict` 0x6E 74, `kFloorVent` 0x01 65, `kCabinet` 0x13 20, `kInvisSwitch` 0x46 19, `kInvisBonus` 0x2B 18, `kShelf` 0x12 16, `kSoundTrigger` 0x49 15, `kFlower` 0x85 15, `kLiftArea` 0x10 13, `kYellowClock` 0x23 11, `kCeilingTrans` 0x36 11, `kDeluxeTrans` 0x40 11, `kCandle` 0x09 10, `kCopterLf` 0x72 10.

Distinct `what` codes used: 59 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 75, enemies 46 (ratio 1.63), batteries 1, rubber bands 4, aluminum foil 4, helium 1, clocks 18, stars 6. `CountTotalHousePoints` = **58900** points, of which 9000 come from `kInvisBonus.data.c.points`.

**Links.** 61 link-bearing slots; 53 have `where` != -1; 53 fully linked (`who` != 255); 53 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 74 two-way, 1 one-way, 19 walled. North/South adjacent pairs: 51 two-way, 0 one-way, 39 walled. Staircases: 0 up / 0 down (unpaired 0 / 0). Undirected components 9 (largest 101, the one holding `firstRoom` 101). Forward-reachable from `firstRoom` 76 of 109 rooms; BFS eccentricity 36.

**Stars.** 6, at BFS depths [3, 14, 18, 22, 32, 34] of 36: room 6 "Starry Night" x1; room 27 "Overdrive" x1; room 35 "Water Lillies" x1; room 48 "Bathing" x1; room 56 "This is a Pipe" x1; room 104 "Enlightenment" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x76 (1991-1993, 3000-3053, 10076-10094); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x10 (3000-3004, 3042-3046).

Shadows application `PICT` IDs: 1991-1993.

`kSoundTrigger` `data.e.where` values: 3000 x1, 3001 x1, 3002 x1, 3004 x1, 3042 x5, 3043 x1, 3044 x3, 3045 x1, 3046 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 74 placements over 19 distinct IDs, top: 10082 x18, 10081 x15, 10077 x9, 10076 x6, 10078 x5, 10080 x4, 10079 x4, 10085 x2. All resolve inside this fork.

**Notes.**

- 33 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.2 CD Demo House

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/CD Demo House.binhex`, 2205334 bytes; BinHex name `CD Demo House`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 72554 bytes (866 + 348 x 206 = 72554) |
| resource fork | 1612342 bytes |
| `.mov` sidecar | `CD Demo House.mov`, 33042 bytes |
| author (README.md:6-13) | John Calhoun & Kim Money |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C3E8559 -> `& 1` = 1, house LOCKED |
| `flags` | 2 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=True |
| `initial` | v=105 h=223 -> glider spawn rect {top 105, left 223, bottom 125, right 271} |
| `firstRoom` | 70 = "Welcome…" at (floor 2, suite 2) |
| `nRooms` | 206 (real 206, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | `Sample Rooms from Art Museum, SpacePods, Metropolis, Leviathan, Nemo's Market, Land of Illusion, Davis Station, Titanic, and Teddy World. ` |
| `trailer` (Str255) | `If you liked this demo, you'll love the CD.  Order it now!` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 25300 by `Ozma`, rooms visited 28 |

**Geometry.** 16 occupied floors (-2..14), 45 occupied suites (2..53); grid occupancy 206 / (16 x 45) = 0.29.

- rooms per floor (floor:count, north-up): 14:3, 13:6, 12:6, 11:6, 9:5, 8:14, 7:14, 6:18, 5:9, 4:8, 3:15, 2:36, 1:42, 0:22, -1:1, -2:1
- rooms per suite (suite:count, west-first): 2:2, 6:8, 7:13, 8:14, 9:14, 10:12, 11:10, 12:9, 14:6, 15:6, 16:6, 17:3, 18:3, 19:3, 20:4, 21:1, 22:1, 23:1, 24:1, 25:2, 26:2, 27:2, 28:2, 29:2, 31:4, 32:4, 33:4, 34:4, 35:4, 36:4, 37:4, 38:4, 40:3, 41:3, 42:3, 43:3, 44:3, 45:3, 46:3, 48:2, 49:5, 50:5, 51:7, 52:5, 53:2

**Backgrounds.** 25 distinct IDs over 206 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 13 |
| 2001 | kPaneledRoom | 1 |
| 2004 | kAsianRoom | 1 |
| 2005 | kUnfinishedRoom | 6 |
| 2007 | kBathroom | 5 |
| 2008 | kLibrary | 1 |
| 2011 | kDirt | 9 |
| 2012 | kMeadow | 4 |
| 2013 | kField | 7 |
| 2014 | kRoof | 12 |
| 2015 | kSky | 32 |
| 2017 | kStars | 3 |

- user-art backgrounds (`>= kUserBackground` 3000): 13 distinct IDs over 112 rooms - 3009, 3011, 3100, 3102-3104, 3211, 3301-3302, 3306-3308, 3314
  - structure sub-range 3000-3299: 7 IDs / 94 rooms; open sub-range 3300-3799: 6 IDs / 18 rooms
  - most-used: 3104 (36 rooms), 3011 (24 rooms), 3009 (18 rooms), 3301 (8 rooms), 3100 (5 rooms), 3102 (4 rooms), 3211 (4 rooms), 3103 (3 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 28 rooms; `bounds`==0 in 71, non-zero in 135; `leftStart` mode 32 (183 rooms), `rightStart` mode 32 (189 rooms); rooms named exactly "Untitled Room": 15; distinct room names 90 over 206 rooms (reuse ratio 2.29).

- most reused names: `Black Space` x32, `Untitled Room` x15, `Column` x8, `N` x8, `Sub Nemo's` x7

**Objects.** 1533 live of 4944 slots (31.0% fill); per room mean 7.44, median 6.0, max 24; 7 rooms at the 24-object ceiling, 40 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 196 | 135 | 77 | 60 | 57 | 105 | 660 | 75 | 168 |

Top 15 `what` codes: `kCustomPict` 0x6E 647, `kFloorVent` 0x01 70, `kInvisLight` 0x58 68, `kInvisBlower` 0x0D 51, `kMirror` 0x82 46, `kFlower` 0x85 34, `kCloud` 0x8C 33, `kBalloon` 0x71 26, `kLiftArea` 0x10 25, `kInvisBounce` 0x1F 24, `kInvisObstacle` 0x1C 23, `kWallWindow` 0x86 20, `kCounter` 0x17 18, `kDrip` 0x77 17, `kCeilingTrans` 0x36 16.

Distinct `what` codes used: 100 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 77, enemies 75 (ratio 1.03), batteries 1, rubber bands 4, aluminum foil 0, helium 2, clocks 18, stars 9. `CountTotalHousePoints` = **77600** points, of which 4000 come from `kInvisBonus.data.c.points`.

**Links.** 101 link-bearing slots; 94 have `where` != -1; 82 fully linked (`who` != 255); 94 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 1 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 123 two-way, 18 one-way, 27 walled. North/South adjacent pairs: 67 two-way, 14 one-way, 61 walled. Staircases: 1 up / 1 down (unpaired 0 / 0). Undirected components 21 (largest 159, the one holding `firstRoom` 159). Forward-reachable from `firstRoom` 129 of 206 rooms; BFS eccentricity 11.

**Stars.** 9, at BFS depths [3, 4, 6, 6, 6, 6, 6, 7, 8] of 11: room 5 "7 Second Shopping Spree" x1; room 30 "Ball Illusion" x1; room 52 "Teddy On Top" x1; room 78 "Outdoor Heating" x1; room 96 "Ground Floor" x1; room 119 "Down We Descend!" x1; room 143 "Every Teddy's a Transport!" x1; room 176 "BalloonGate" x1; room 195 "Claustrophobia" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x116 (3009, 3011, 3100, 3102-3104, 3211, 3301-3302, 3306-3308, 3314, 3904, 3986, 10000-10007, 10010-10031, 10050-10051, 10053-10067, 10076-10091, 10100-10105, 10107-10110, 11003-11019, 11021-11031); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x10 (3000-3008, 3042).

Shadows application `PICT` IDs: 3904, 3986, 10000.

`kSoundTrigger` `data.e.where` values: 3000 x1, 3001 x1, 3002 x1, 3003 x1, 3004 x1, 3005 x1, 3006 x1, 3007 x1, 3008 x1, 3042 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 647 placements over 100 distinct IDs, top: 11003 x126, 11011 x56, 10020 x48, 10012 x32, 10082 x28, 11004 x24, 10088 x24, 11010 x21. All resolve inside this fork.

**Notes.**

- 23 room(s) carry a non-zero `bounds` while using a built-in background, where `DetermineRoomOpenings` ignores it: room 33 "The Other Side" (bg 2015, bounds 3), room 71 "So Far" (bg 2013, bounds 15), room 72 "Let's Roll" (bg 2013, bounds 15), room 73 "Flatland" (bg 2013, bounds 15), room 74 "Caboose" (bg 2013, bounds 15), room 75 "Cruising the "Fe"" (bg 2013, bounds 15), room 76 "Step on up" (bg 2013, bounds 15), room 79 "Darling" (bg 2015, bounds 15)
- 1 link(s) have `who` >= kMaxRoomObs (24)
- 77 room(s) are not forward-reachable from `firstRoom` under the static model
- 15 room(s) still named "Untitled Room" (`CountUntitledRooms`, HouseLegal.c:834)

### 4.3 California or Bust!

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/California or Bust!.binhex`, 344163 bytes; BinHex name `California or Bust!`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 6434 bytes (866 + 348 x 16 = 6434) |
| resource fork | 259542 bytes |
| `.mov` sidecar | none |
| author (README.md:6-13) | unattributed (not in README credits) |
| `version` | 0x0200 |
| `unusedShort` | 13107 |
| `timeStamp` | 0x2C7542B8 -> `& 1` = 0, house unlocked |
| `flags` | 2 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=True |
| `initial` | v=62 h=308 -> glider spawn rect {top 62, left 308, bottom 82, right 356} |
| `firstRoom` | 14 = "Leaving the Heartland" at (floor 1, suite 69) |
| `nRooms` | 16 (real 16, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | `What will it take to move all John's stuff?  A U-haul?  A pick-up truck?  Nope . . . it'll have to be the one and only "Glider Express"                    All Aboard!` |
| `trailer` (Str255) | `  Toto, we're not in Kansas anymore . . .` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 7500 by `Ozma`, rooms visited 13 |

**Geometry.** 1 occupied floors (1..1), 16 occupied suites (55..70); grid occupancy 16 / (1 x 16) = 1.00.

- rooms per floor (floor:count, north-up): 1:16
- rooms per suite (suite:count, west-first): 55:1, 56:1, 57:1, 58:1, 59:1, 60:1, 61:1, 62:1, 63:1, 64:1, 65:1, 66:1, 67:1, 68:1, 69:1, 70:1

**Backgrounds.** 2 distinct IDs over 16 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2013 | kField | 2 |

- user-art backgrounds (`>= kUserBackground` 3000): 1 distinct IDs over 14 rooms - 3000
  - structure sub-range 3000-3299: 1 IDs / 14 rooms; open sub-range 3300-3799: 0 IDs / 0 rooms
  - most-used: 3000 (14 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 13 rooms; `bounds`==0 in 2, non-zero in 14; `leftStart` mode 32 (16 rooms), `rightStart` mode 32 (16 rooms); rooms named exactly "Untitled Room": 2; distinct room names 15 over 16 rooms (reuse ratio 1.07).

- most reused names: `Untitled Room` x2

**Objects.** 308 live of 384 slots (80.2% fill); per room mean 19.25, median 21.0, max 24; 6 rooms at the 24-object ceiling, 0 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 75 | 8 | 15 | 0 | 3 | 15 | 104 | 20 | 68 |

Top 15 `what` codes: `kCustomPict` 0x6E 83, `kLiftArea` 0x10 32, `kInvisBlower` 0x0D 25, `kFlower` 0x85 25, `kInvisLight` 0x58 14, `kFish` 0x78 12, `kCloud` 0x8C 12, `kBear` 0x87 10, `kCinderBlock` 0x6B 6, `kBall` 0x76 5, `kMirror` 0x82 5, `kWallWindow` 0x86 5, `kRightFan` 0x07 4, `kTaper` 0x08 4, `kCandle` 0x09 4.

Distinct `what` codes used: 55 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 15, enemies 20 (ratio 0.75), batteries 1, rubber bands 0, aluminum foil 1, helium 0, clocks 3, stars 1. `CountTotalHousePoints` = **7900** points, of which 400 come from `kInvisBonus.data.c.points`.

**Links.** 0 link-bearing slots; 0 have `where` != -1; 0 fully linked (`who` != 255); 0 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 15 two-way, 0 one-way, 0 walled. North/South adjacent pairs: 0 two-way, 0 one-way, 0 walled. Staircases: 0 up / 0 down (unpaired 0 / 0). Undirected components 1 (largest 16, the one holding `firstRoom` 16). Forward-reachable from `firstRoom` 16 of 16 rooms; BFS eccentricity 14.

**Stars.** 1, at BFS depths [13] of 14: room 0 "Welcome to Silicon Valley!" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x30 (3000, 10000-10028); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x3 (3001-3003).

Shadows application `PICT` IDs: 10000.

`kSoundTrigger` `data.e.where` values: 3001 x1, 3002 x1, 3003 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 83 placements over 29 distinct IDs, top: 10000 x14, 10022 x9, 10021 x9, 10020 x7, 10024 x5, 10002 x4, 10003 x3, 10001 x3. All resolve inside this fork.

**Notes.**

- 2 room(s) still named "Untitled Room" (`CountUntitledRooms`, HouseLegal.c:834)

### 4.4 Castle o' the Air

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Castle o' the Air.binhex`, 437515 bytes; BinHex name `Castle o' the Air`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 30446 bytes (866 + 348 x 85 = 30446) |
| resource fork | 306364 bytes |
| `.mov` sidecar | `Castle o' the Air.mov`, 21960 bytes |
| author (README.md:6-13) | unattributed (not in README credits) |
| `version` | 0x0200 |
| `unusedShort` | 259 |
| `timeStamp` | 0x2C32F464 -> `& 1` = 0, house unlocked |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=124 h=239 -> glider spawn rect {top 124, left 239, bottom 144, right 287} |
| `firstRoom` | 33 = "West Courtyard" at (floor 1, suite 68) |
| `nRooms` | 85 (real 85, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 30 |
| `banner` (Str255) | `Knowest That Thou Hast Entered The Haunted Castle o' the Air  (by john calhoun)` |
| `trailer` (Str255) | `Thou Hast Broken The Curse of the Castle of Air.  Ye Are Knighted!` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 7500 by `Ozma`, rooms visited 15 |

**Geometry.** 10 occupied floors (-1..8), 13 occupied suites (63..75); grid occupancy 85 / (10 x 13) = 0.65.

- rooms per floor (floor:count, north-up): 8:3, 7:3, 6:3, 5:3, 4:13, 3:13, 2:13, 1:13, 0:13, -1:8
- rooms per suite (suite:count, west-first): 63:6, 64:6, 65:6, 66:6, 67:6, 68:10, 69:10, 70:10, 71:5, 72:5, 73:5, 74:5, 75:5

**Backgrounds.** 11 distinct IDs over 85 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2011 | kDirt | 16 |
| 2015 | kSky | 14 |

- user-art backgrounds (`>= kUserBackground` 3000): 9 distinct IDs over 55 rooms - 3000-3003, 3300-3304
  - structure sub-range 3000-3299: 4 IDs / 24 rooms; open sub-range 3300-3799: 5 IDs / 31 rooms
  - most-used: 3301 (14 rooms), 3000 (12 rooms), 3300 (8 rooms), 3002 (6 rooms), 3304 (5 rooms), 3001 (4 rooms), 3302 (3 rooms), 3003 (2 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 1 rooms; `bounds`==0 in 85, non-zero in 0; `leftStart` mode 32 (77 rooms), `rightStart` mode 32 (78 rooms); rooms named exactly "Untitled Room": 0; distinct room names 39 over 85 rooms (reuse ratio 2.18).

- most reused names: `Kitten` x44, `Falling` x4

**Objects.** 620 live of 2040 slots (30.4% fill); per room mean 7.29, median 4, max 24; 1 rooms at the 24-object ceiling, 16 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 111 | 74 | 120 | 32 | 7 | 52 | 7 | 78 | 139 |

Top 15 `what` codes: `kCloud` 0x8C 115, `kSewerGrate` 0x05 65, `kSparkle` 0x2D 65, `kInvisLight` 0x58 52, `kInvisBounce` 0x1F 37, `kInvisBlower` 0x0D 28, `kDrip` 0x77 28, `kInvisBonus` 0x2B 24, `kCabinet` 0x13 20, `kCeilingTrans` 0x36 18, `kBalloon` 0x71 17, `kCopterLf` 0x72 13, `kPaper` 0x25 10, `kFloorTrans` 0x35 9, `kDartLf` 0x74 8.

Distinct `what` codes used: 50 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 120, enemies 78 (ratio 1.54), batteries 1, rubber bands 3, aluminum foil 1, helium 0, clocks 5, stars 4. `CountTotalHousePoints` = **45500** points, of which 12000 come from `kInvisBonus.data.c.points`.

**Links.** 35 link-bearing slots; 35 have `where` != -1; 23 fully linked (`who` != 255); 23 resolved to a live room; 12 dangling (of which 12 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 75 two-way, 0 one-way, 0 walled. North/South adjacent pairs: 28 two-way, 8 one-way, 36 walled. Staircases: 0 up / 0 down (unpaired 0 / 0). Undirected components 2 (largest 77, the one holding `firstRoom` 77). Forward-reachable from `firstRoom` 77 of 85 rooms; BFS eccentricity 10.

**Stars.** 4, at BFS depths [1, 3, 4, 6] of 10: room 3 "Heraldic" x1; room 11 "Strange Window" x1; room 20 "Early To Bed!" x1; room 72 "Magic" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x12 (3000-3003, 3300-3304, 10000-10002); `bnds` x9 (3000-3003, 3300-3304); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455).

Shadows application `PICT` IDs: 10000.

`kCustomPict` `data.g.height` (PICT ID) values: 4 placements over 3 distinct IDs, top: 10002 x2, 10001 x1, 10000 x1. All resolve inside this fork.

**Notes.**

- 8 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.5 Davis Station

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Davis Station.binhex`, 2499538 bytes; BinHex name `Davis Station`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 23486 bytes (866 + 348 x 65 = 23486) |
| resource fork | 1871320 bytes |
| `.mov` sidecar | `Davis Station.mov`, 34504 bytes |
| author (README.md:6-13) | Jonathan Chin (alias Paul Finn) & John Calhoun |
| `version` | 0x0200 |
| `unusedShort` | 222 |
| `timeStamp` | 0x2C53AEFD -> `& 1` = 1, house LOCKED |
| `flags` | 2 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=True |
| `initial` | v=95 h=189 -> glider spawn rect {top 95, left 189, bottom 115, right 237} |
| `firstRoom` | 4 = "Let's Roll" at (floor 1, suite 59) |
| `nRooms` | 65 (real 65, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 37 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `Come back real soon — if you're ever out this way!` |
| `highScores.banner` | `Johnner beat the Kimmer.` |
| top score | 17800 by `Johnner`, rooms visited 38 |

**Geometry.** 14 occupied floors (-6..8), 20 occupied suites (58..77); grid occupancy 65 / (14 x 20) = 0.23.

- rooms per floor (floor:count, north-up): 8:3, 7:3, 6:3, 5:3, 3:5, 2:11, 1:20, 0:5, -1:7, -2:1, -3:1, -4:1, -5:1, -6:1
- rooms per suite (suite:count, west-first): 58:1, 59:1, 60:1, 61:1, 62:1, 63:9, 64:7, 65:7, 66:7, 67:3, 68:4, 69:3, 70:2, 71:2, 72:3, 73:3, 74:3, 75:3, 76:3, 77:1

**Backgrounds.** 35 distinct IDs over 65 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 1 |
| 2001 | kPaneledRoom | 2 |
| 2003 | kChildsRoom | 1 |
| 2011 | kDirt | 6 |
| 2015 | kSky | 8 |

- user-art backgrounds (`>= kUserBackground` 3000): 30 distinct IDs over 47 rooms - 3207-3236
  - structure sub-range 3000-3299: 30 IDs / 47 rooms; open sub-range 3300-3799: 0 IDs / 0 rooms
  - most-used: 3212 (5 rooms), 3216 (4 rooms), 3207 (3 rooms), 3211 (3 rooms), 3234 (3 rooms), 3214 (2 rooms), 3219 (2 rooms), 3220 (2 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 38 rooms; `bounds`==0 in 18, non-zero in 47; `leftStart` mode 32 (53 rooms), `rightStart` mode 32 (56 rooms); rooms named exactly "Untitled Room": 0; distinct room names 60 over 65 rooms (reuse ratio 1.08).

- most reused names: `Darling` x4, `So Far` x2, `Bookends` x2

**Objects.** 591 live of 1560 slots (37.9% fill); per room mean 9.09, median 7, max 24; 6 rooms at the 24-object ceiling, 0 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 80 | 120 | 29 | 24 | 15 | 56 | 67 | 64 | 136 |

Top 15 `what` codes: `kInvisBounce` 0x1F 79, `kCustomPict` 0x6E 66, `kCloud` 0x8C 62, `kFlower` 0x85 55, `kInvisLight` 0x58 41, `kDrip` 0x77 22, `kSewerGrate` 0x05 19, `kSewerBlower` 0x0F 18, `kInvisBlower` 0x0D 16, `kLiftArea` 0x10 16, `kCopterLf` 0x72 13, `kLightBulb` 0x52 12, `kBalloon` 0x71 12, `kMilkCrate` 0x16 11, `kInvisObstacle` 0x1C 10.

Distinct `what` codes used: 69 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 29, enemies 64 (ratio 0.45), batteries 1, rubber bands 2, aluminum foil 0, helium 1, clocks 9, stars 4. `CountTotalHousePoints` = **31800** points, of which 0 come from `kInvisBonus.data.c.points`.

**Links.** 24 link-bearing slots; 17 have `where` != -1; 17 fully linked (`who` != 255); 17 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 39 two-way, 5 one-way, 5 walled. North/South adjacent pairs: 24 two-way, 5 one-way, 9 walled. Staircases: 2 up / 2 down (unpaired 0 / 0). Undirected components 1 (largest 65, the one holding `firstRoom` 65). Forward-reachable from `firstRoom` 57 of 65 rooms; BFS eccentricity 23.

**Stars.** 4, at BFS depths [13, 17, 19, 23] of 23: room 26 "The Star!" x1; room 33 "Grain Elevator" x1; room 49 "John's Loft" x1; room 54 "Final Departure" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x54 (1991-1993, 3207-3236, 10000-10020); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x5 (3000-3004).

Shadows application `PICT` IDs: 1991-1993, 10000.

`kSoundTrigger` `data.e.where` values: 3000 x1, 3001 x1, 3002 x1, 3003 x1, 3004 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 66 placements over 21 distinct IDs, top: 10006 x12, 10000 x11, 10018 x7, 10008 x5, 10009 x4, 10010 x4, 10007 x4, 10001 x2. All resolve inside this fork.

**Notes.**

- 8 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.6 Demo House

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Demo House.binhex`, 664651 bytes; BinHex name `Demo House`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 16526 bytes (866 + 348 x 45 = 16526) |
| resource fork | 491757 bytes |
| `.mov` sidecar | `Demo House.mov`, 119789 bytes |
| author (README.md:6-13) | John Calhoun & Kim Money |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C53B041 -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=107 h=49 -> glider spawn rect {top 107, left 49, bottom 127, right 97} |
| `firstRoom` | 0 = "Air Vents" at (floor 1, suite 63) |
| `nRooms` | 45 (real 45, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | `Welcome to the Demo House! This is a small beginner house that acts as a sort of tutorial. (house by Kim Money)` |
| `trailer` (Str255) | `Excellent! That's the extent of the Demo House though.  The house "Slumberland" has over 400 rooms!  Good luck.` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 7400 by `Ozma`, rooms visited 13 |

**Geometry.** 7 occupied floors (-1..5), 10 occupied suites (62..71); grid occupancy 45 / (7 x 10) = 0.64.

- rooms per floor (floor:count, north-up): 5:3, 4:3, 3:6, 2:10, 1:10, 0:10, -1:3
- rooms per suite (suite:count, west-first): 62:3, 63:3, 64:3, 65:3, 66:4, 67:5, 68:5, 69:7, 70:6, 71:6

**Backgrounds.** 17 distinct IDs over 45 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2011 | kDirt | 13 |
| 2012 | kMeadow | 1 |
| 2013 | kField | 1 |
| 2014 | kRoof | 5 |
| 2015 | kSky | 9 |
| 2016 | kStratosphere | 3 |
| 2017 | kStars | 3 |

- user-art backgrounds (`>= kUserBackground` 3000): 10 distinct IDs over 10 rooms - 3000-3005, 3300-3303
  - structure sub-range 3000-3299: 6 IDs / 6 rooms; open sub-range 3300-3799: 4 IDs / 4 rooms
  - most-used: 3000 (1 rooms), 3001 (1 rooms), 3002 (1 rooms), 3003 (1 rooms), 3004 (1 rooms), 3005 (1 rooms), 3300 (1 rooms), 3301 (1 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 12 rooms; `bounds`==0 in 45, non-zero in 0; `leftStart` mode 0 (45 rooms), `rightStart` mode 0 (45 rooms); rooms named exactly "Untitled Room": 0; distinct room names 44 over 45 rooms (reuse ratio 1.02).

- most reused names: `Out Here` x2

**Objects.** 138 live of 1080 slots (12.8% fill); per room mean 3.07, median 0, max 23; 0 rooms at the 24-object ceiling, 24 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 35 | 24 | 13 | 4 | 3 | 9 | 10 | 4 | 36 |

Top 15 `what` codes: `kFlower` 0x85 16, `kFloorVent` 0x01 15, `kInvisBlower` 0x0D 13, `kCustomPict` 0x6E 8, `kWallWindow` 0x86 7, `kInvisObstacle` 0x1C 6, `kInvisLight` 0x58 5, `kCloud` 0x8C 5, `kShelf` 0x12 4, `kMilkCrate` 0x16 4, `kStool` 0x1A 3, `kGreaseRt` 0x28 3, `kBalloon` 0x71 3, `kSewerGrate` 0x05 2, `kTaper` 0x08 2.

Distinct `what` codes used: 48 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 13, enemies 4 (ratio 3.25), batteries 1, rubber bands 1, aluminum foil 0, helium 0, clocks 5, stars 1. `CountTotalHousePoints` = **12500** points, of which 800 come from `kInvisBonus.data.c.points`.

**Links.** 2 link-bearing slots; 2 have `where` != -1; 2 fully linked (`who` != 255); 2 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 33 two-way, 2 one-way, 3 walled. North/South adjacent pairs: 18 two-way, 0 one-way, 17 walled. Staircases: 1 up / 1 down (unpaired 0 / 0). Undirected components 2 (largest 42, the one holding `firstRoom` 42). Forward-reachable from `firstRoom` 33 of 45 rooms; BFS eccentricity 15.

**Stars.** 1, at BFS depths [10] of 15: room 23 "Grande Recompense" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x19 (3000-3005, 3300-3303, 10008, 10011-10012, 10014, 10017-10018, 10036, 10049, 10065); `bnds` x10 (3000-3005, 3300-3303); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x1 (3011).

`kSoundTrigger` `data.e.where` values: 3011 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 8 placements over 7 distinct IDs, top: 10065 x2, 10014 x1, 10017 x1, 10049 x1, 10036 x1, 10012 x1, 10018 x1. All resolve inside this fork.

**Notes.**

- 12 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.7 Empty House

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Empty House.binhex`, 19343 bytes; BinHex name `Empty House`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 13046 bytes (866 + 348 x 35 = 13046) |
| resource fork | 2670 bytes |
| `.mov` sidecar | none |
| author (README.md:6-13) | unattributed (not in README credits) |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C3130EC -> `& 1` = 0, house unlocked |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=64 h=83 -> glider spawn rect {top 64, left 83, bottom 84, right 131} |
| `firstRoom` | 0 = "Untitled Room" at (floor 1, suite 64) |
| `nRooms` | 35 (real 35, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | `This is an empty house.  You can fly through it but there are no obstacles or prizes.  Make a copy of this house and feel free to fill it up.` |
| `trailer` (Str255) | `Enter Finished-House Message here (max. 255 characters)` |
| `highScores.banner` | `Empty House` |
| top score | 0 (no scores recorded) |

**Geometry.** 6 occupied floors (-1..4), 9 occupied suites (62..70); grid occupancy 35 / (6 x 9) = 0.65.

- rooms per floor (floor:count, north-up): 4:6, 3:6, 2:7, 1:9, 0:6, -1:1
- rooms per suite (suite:count, west-first): 62:1, 63:5, 64:5, 65:5, 66:5, 67:5, 68:4, 69:4, 70:1

**Backgrounds.** 10 distinct IDs over 35 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 4 |
| 2001 | kPaneledRoom | 4 |
| 2002 | kBasement | 2 |
| 2003 | kChildsRoom | 2 |
| 2004 | kAsianRoom | 2 |
| 2011 | kDirt | 5 |
| 2012 | kMeadow | 4 |
| 2013 | kField | 1 |
| 2014 | kRoof | 4 |
| 2015 | kSky | 7 |

**Room fields.** `visited`=1 in 16 rooms; `bounds`==0 in 35, non-zero in 0; `leftStart` mode 0 (35 rooms), `rightStart` mode 0 (35 rooms); rooms named exactly "Untitled Room": 34; distinct room names 2 over 35 rooms (reuse ratio 17.50).

- most reused names: `Untitled Room` x34

**Objects.** 78 live of 840 slots (9.3% fill); per room mean 2.23, median 3, max 7; 0 rooms at the 24-object ceiling, 12 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 43 | 3 | 1 | 12 | 0 | 19 | 0 | 0 | 0 |

Top 15 `what` codes: `kFloorVent` 0x01 28, `kSewerGrate` 0x05 11, `kCeilingLight` 0x51 10, `kLightBulb` 0x52 6, `kInvisBlower` 0x0D 4, `kUpStairs` 0x31 4, `kDownStairs` 0x32 4, `kManhole` 0x1D 3, `kFlourescent` 0x56 2, `kStar` 0x2C 1, `kDoorInLf` 0x37 1, `kDoorExRt` 0x39 1, `kWindowInRt` 0x3C 1, `kWindowExLf` 0x3E 1, `kTableLamp` 0x53 1.

Distinct `what` codes used: 15 of the 117 that appear anywhere in the corpus.

Full vocabulary: `kFloorVent` 0x01 x28, `kSewerGrate` 0x05 x11, `kCeilingLight` 0x51 x10, `kLightBulb` 0x52 x6, `kInvisBlower` 0x0D x4, `kUpStairs` 0x31 x4, `kDownStairs` 0x32 x4, `kManhole` 0x1D x3, `kFlourescent` 0x56 x2, `kStar` 0x2C x1, `kDoorInLf` 0x37 x1, `kDoorExRt` 0x39 x1, `kWindowInRt` 0x3C x1, `kWindowExLf` 0x3E x1, `kTableLamp` 0x53 x1.

**Economy.** prizes 1, enemies 0 (ratio n/a), batteries 0, rubber bands 0, aluminum foil 0, helium 0, clocks 0, stars 1. `CountTotalHousePoints` = **8500** points, of which 0 come from `kInvisBonus.data.c.points`.

**Links.** 0 link-bearing slots; 0 have `where` != -1; 0 fully linked (`who` != 255); 0 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 22 two-way, 0 one-way, 6 walled. North/South adjacent pairs: 14 two-way, 0 one-way, 12 walled. Staircases: 4 up / 4 down (unpaired 0 / 0). Undirected components 1 (largest 35, the one holding `firstRoom` 35). Forward-reachable from `firstRoom` 35 of 35 rooms; BFS eccentricity 11.

**Stars.** 1, at BFS depths [11] of 11: room 28 "Finish" x1.

**Resource fork.** `ICN#` x1 (-16455); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455).

**Notes.**

- 34 room(s) still named "Untitled Room" (`CountUntitledRooms`, HouseLegal.c:834)

### 4.8 Fun House

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Fun House.binhex`, 895562 bytes; BinHex name `Fun House`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 15830 bytes (866 + 348 x 43 = 15830) |
| resource fork | 662443 bytes |
| `.mov` sidecar | none |
| author (README.md:6-13) | unattributed (not in README credits) |
| `version` | 0x0200 |
| `unusedShort` | 26228 |
| `timeStamp` | 0x2C446C10 -> `& 1` = 0, house unlocked |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=78 h=42 -> glider spawn rect {top 78, left 42, bottom 98, right 90} |
| `firstRoom` | 29 = "Mission Impossible?" at (floor 4, suite 58) |
| `nRooms` | 43 (real 43, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `h: 107 v: 55` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 1600 by `Ozma`, rooms visited 8 |

**Geometry.** 5 occupied floors (1..5), 14 occupied suites (56..71); grid occupancy 43 / (5 x 14) = 0.61.

- rooms per floor (floor:count, north-up): 5:6, 4:6, 3:6, 2:11, 1:14
- rooms per suite (suite:count, west-first): 56:5, 57:5, 58:5, 59:5, 60:5, 61:5, 63:2, 64:2, 65:2, 66:2, 67:2, 69:1, 70:1, 71:1

**Backgrounds.** 14 distinct IDs over 43 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 7 |
| 2001 | kPaneledRoom | 3 |
| 2003 | kChildsRoom | 1 |
| 2004 | kAsianRoom | 1 |
| 2012 | kMeadow | 5 |
| 2014 | kRoof | 4 |
| 2015 | kSky | 9 |

- user-art backgrounds (`>= kUserBackground` 3000): 7 distinct IDs over 13 rooms - 3000-3004, 3352-3353
  - structure sub-range 3000-3299: 5 IDs / 7 rooms; open sub-range 3300-3799: 2 IDs / 6 rooms
  - most-used: 3352 (3 rooms), 3353 (3 rooms), 3001 (2 rooms), 3002 (2 rooms), 3000 (1 rooms), 3003 (1 rooms), 3004 (1 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 2 rooms; `bounds`==0 in 30, non-zero in 13; `leftStart` mode 32 (39 rooms), `rightStart` mode 32 (33 rooms); rooms named exactly "Untitled Room": 2; distinct room names 15 over 43 rooms (reuse ratio 2.87).

- most reused names: `Out` x17, `Balancing Act` x6, `Welcome` x2, `Gray Room` x2, `Ghosts` x2

**Objects.** 406 live of 1032 slots (39.3% fill); per room mean 9.44, median 7, max 24; 4 rooms at the 24-object ceiling, 12 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 63 | 96 | 17 | 43 | 52 | 25 | 57 | 23 | 30 |

Top 15 `what` codes: `kFloorVent` 0x01 39, `kCabinet` 0x13 37, `kCustomPict` 0x6E 34, `kShelf` 0x12 31, `kInvisSwitch` 0x46 29, `kBall` 0x76 23, `kInvisTrans` 0x3F 19, `kToaster` 0x62 15, `kCloud` 0x8C 15, `kLiftArea` 0x10 14, `kMachineSwitch` 0x42 14, `kCeilingTrans` 0x36 9, `kCeilingLight` 0x51 9, `kInvisLight` 0x58 9, `kBooks` 0x1E 8.

Distinct `what` codes used: 54 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 17, enemies 23 (ratio 0.74), batteries 1, rubber bands 1, aluminum foil 2, helium 0, clocks 7, stars 0. `CountTotalHousePoints` = **7900** points, of which 0 come from `kInvisBonus.data.c.points`.

**Links.** 91 link-bearing slots; 83 have `where` != -1; 83 fully linked (`who` != 255); 83 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 18 two-way, 0 one-way, 17 walled. North/South adjacent pairs: 14 two-way, 0 one-way, 15 walled. Staircases: 1 up / 1 down (unpaired 0 / 0). Undirected components 4 (largest 25, the one holding `firstRoom` 25). Forward-reachable from `firstRoom` 5 of 43 rooms; BFS eccentricity 2.

**Stars.** none - `CheckHouseForProblems` (HouseLegal.c:1203) would flag this house in red.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x26 (1991-1993, 2014-2015, 3000-3004, 3352-3353, 3976, 10000-10012); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455).

Shadows application `PICT` IDs: 1991-1993, 2014-2015, 3976, 10000.

`kCustomPict` `data.g.height` (PICT ID) values: 34 placements over 10 distinct IDs, top: 10007 x10, 10005 x8, 10006 x5, 10008 x5, 10000 x1, 10001 x1, 10002 x1, 10003 x1. All resolve inside this fork.

**Notes.**

- 38 room(s) are not forward-reachable from `firstRoom` under the static model
- 2 room(s) still named "Untitled Room" (`CountUntitledRooms`, HouseLegal.c:834)
- no `kStar` anywhere: the house is unwinnable by the normal exit condition

### 4.9 Grand Prix

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Grand Prix.binhex`, 1882081 bytes; BinHex name `Grand Prix`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 61766 bytes (866 + 348 x 175 = 61766) |
| resource fork | 1347764 bytes |
| `.mov` sidecar | `Grand Prix.mov`, 26407 bytes |
| author (README.md:6-13) | Jonathan Chin (alias Paul Finn) |
| `version` | 0x0200 |
| `unusedShort` | 60 |
| `timeStamp` | 0x2C318A3D -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=58 h=268 -> glider spawn rect {top 58, left 268, bottom 78, right 316} |
| `firstRoom` | 127 = "Fast!" at (floor 1, suite 54) |
| `nRooms` | 175 (real 175, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 185 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `You know, I really hope you didn't go over the speed limit. Speeding is illegal in Glider PRO, too!` |
| `highScores.banner` | `Uhhh… Hi, Officer…` |
| top score | 34400 by `Paul`, rooms visited 88 |

**Geometry.** 12 occupied floors (-2..9), 27 occupied suites (53..79); grid occupancy 175 / (12 x 27) = 0.54.

- rooms per floor (floor:count, north-up): 9:5, 8:5, 7:10, 6:10, 5:10, 4:15, 3:21, 2:27, 1:27, 0:27, -1:13, -2:5
- rooms per suite (suite:count, west-first): 53:3, 54:3, 55:4, 56:5, 57:5, 58:5, 59:11, 60:12, 61:12, 62:12, 63:12, 64:10, 65:9, 66:9, 67:9, 68:9, 69:3, 70:3, 71:3, 72:3, 73:5, 74:5, 75:5, 76:5, 77:5, 78:4, 79:4

**Backgrounds.** 41 distinct IDs over 175 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 7 |
| 2001 | kPaneledRoom | 8 |
| 2002 | kBasement | 10 |
| 2003 | kChildsRoom | 3 |
| 2004 | kAsianRoom | 1 |
| 2005 | kUnfinishedRoom | 4 |
| 2006 | kSwingersRoom | 3 |
| 2007 | kBathroom | 1 |
| 2008 | kLibrary | 2 |
| 2010 | kSkywalk | 2 |
| 2011 | kDirt | 35 |
| 2012 | kMeadow | 8 |
| 2014 | kRoof | 19 |
| 2015 | kSky | 36 |

- user-art backgrounds (`>= kUserBackground` 3000): 27 distinct IDs over 36 rooms - 3001-3016, 3301-3303, 3305-3312
  - structure sub-range 3000-3299: 16 IDs / 20 rooms; open sub-range 3300-3799: 11 IDs / 16 rooms
  - most-used: 3002 (3 rooms), 3003 (3 rooms), 3301 (3 rooms), 3305 (3 rooms), 3302 (2 rooms), 3001 (1 rooms), 3004 (1 rooms), 3005 (1 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 33 rooms; `bounds`==0 in 137, non-zero in 38; `leftStart` mode 32 (169 rooms), `rightStart` mode 32 (169 rooms); rooms named exactly "Untitled Room": 0; distinct room names 97 over 175 rooms (reuse ratio 1.80).

- most reused names: `Nitrogen` x33, `Stuff` x31, `Roof!` x16, `Grassy` x2

**Objects.** 1281 live of 4200 slots (30.5% fill); per room mean 7.32, median 6, max 24; 9 rooms at the 24-object ceiling, 60 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 284 | 210 | 87 | 83 | 104 | 77 | 83 | 132 | 221 |

Top 15 `what` codes: `kCloud` 0x8C 176, `kInvisBounce` 0x1F 111, `kLiftArea` 0x10 104, `kFloorVent` 0x01 69, `kInvisLight` 0x58 58, `kInvisSwitch` 0x46 47, `kBalloon` 0x71 44, `kCustomPict` 0x6E 41, `kInvisBlower` 0x0D 34, `kLgTrigger` 0x48 33, `kSewerGrate` 0x05 32, `kDrip` 0x77 31, `kCabinet` 0x13 27, `kCeilingTrans` 0x36 25, `kCandle` 0x09 18.

Distinct `what` codes used: 100 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 87, enemies 132 (ratio 0.66), batteries 4, rubber bands 6, aluminum foil 0, helium 3, clocks 19, stars 3. `CountTotalHousePoints` = **44600** points, of which 4800 come from `kInvisBonus.data.c.points`.

**Links.** 152 link-bearing slots; 127 have `where` != -1; 127 fully linked (`who` != 255); 127 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 112 two-way, 6 one-way, 43 walled. North/South adjacent pairs: 62 two-way, 3 one-way, 83 walled. Staircases: 10 up / 10 down (unpaired 0 / 0). Undirected components 2 (largest 170, the one holding `firstRoom` 170). Forward-reachable from `firstRoom` 149 of 175 rooms; BFS eccentricity 35.

**Stars.** 3, at BFS depths [17, 22, 33] of 35: room 2 "One more left!" x1; room 24 "Gotcha!" x1; room 172 "The Winner's Circle" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x70 (1991-1993, 3001-3016, 3301-3303, 3305-3312, 10000-10019, 10100-10119); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x3 (3000-3002); `vers` x2 (1-2).

Shadows application `PICT` IDs: 1991-1993, 10000.

`kSoundTrigger` `data.e.where` values: 3000 x1, 3001 x1, 3002 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 41 placements over 39 distinct IDs, top: 10001 x2, 10005 x2, 10116 x1, 10117 x1, 10118 x1, 10119 x1, 10112 x1, 10113 x1. All resolve inside this fork.

**Notes.**

- 2 room(s) carry a non-zero `bounds` while using a built-in background, where `DetermineRoomOpenings` ignores it: room 52 "First Signs" (bg 2000, bounds 35), room 127 "Fast!" (bg 2012, bounds 9)
- 26 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.10 ImagineHouse PRO II

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/ImagineHouse PRO II.binhex`, 1029380 bytes; BinHex name `ImagineHouse PRO II`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 97958 bytes (866 + 348 x 279 = 97958) |
| resource fork | 677770 bytes |
| `.mov` sidecar | `ImagineHouse PRO II.mov`, 30707 bytes |
| author (README.md:6-13) | Jonathan Chin (alias Paul Finn) |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C1DC93D -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=35 h=211 -> glider spawn rect {top 35, left 211, bottom 55, right 259} |
| `firstRoom` | 1 = "Falling from the Sky" at (floor 6, suite 64) |
| `nRooms` | 279 (real 279, placeholders 0) |
| `hasGame` / `unusedBoolean` | 1 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `Congratulations on beating ImagineHouse PRO II! Hope you enjoyed your stay!  ` |
| `highScores.banner` | `Thankyuh… Thankyuh verra much…` |
| top score | 47000 by `Paul`, rooms visited 108 |

**Geometry.** 15 occupied floors (-3..11), 39 occupied suites (6..95); grid occupancy 279 / (15 x 39) = 0.48.

- rooms per floor (floor:count, north-up): 11:3, 10:3, 9:3, 8:3, 7:11, 6:14, 5:21, 4:27, 3:34, 2:39, 1:39, 0:39, -1:26, -2:11, -3:6
- rooms per suite (suite:count, west-first): 6:3, 7:3, 8:5, 9:5, 10:5, 11:5, 63:9, 64:9, 65:9, 66:7, 67:7, 68:7, 69:5, 70:5, 71:5, 72:3, 73:6, 74:10, 75:10, 76:10, 77:10, 78:11, 79:10, 80:10, 81:15, 82:15, 83:15, 84:7, 85:4, 86:4, 87:3, 88:6, 89:7, 90:7, 91:7, 92:7, 93:6, 94:4, 95:3

**Backgrounds.** 34 distinct IDs over 279 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 8 |
| 2001 | kPaneledRoom | 12 |
| 2002 | kBasement | 18 |
| 2003 | kChildsRoom | 6 |
| 2004 | kAsianRoom | 4 |
| 2005 | kUnfinishedRoom | 6 |
| 2006 | kSwingersRoom | 11 |
| 2007 | kBathroom | 4 |
| 2008 | kLibrary | 2 |
| 2009 | kGarden | 1 |
| 2010 | kSkywalk | 3 |
| 2011 | kDirt | 64 |
| 2012 | kMeadow | 12 |
| 2013 | kField | 2 |
| 2014 | kRoof | 17 |
| 2015 | kSky | 67 |
| 2016 | kStratosphere | 3 |
| 2017 | kStars | 8 |

- user-art backgrounds (`>= kUserBackground` 3000): 16 distinct IDs over 31 rooms - 3001-3005, 3300-3303, 3305-3311
  - structure sub-range 3000-3299: 5 IDs / 9 rooms; open sub-range 3300-3799: 11 IDs / 22 rooms
  - most-used: 3301 (7 rooms), 3005 (4 rooms), 3311 (3 rooms), 3004 (2 rooms), 3302 (2 rooms), 3306 (2 rooms), 3310 (2 rooms), 3001 (1 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 108 rooms; `bounds`==0 in 248, non-zero in 31; `leftStart` mode 32 (265 rooms), `rightStart` mode 32 (266 rooms); rooms named exactly "Untitled Room": 0; distinct room names 151 over 279 rooms (reuse ratio 1.85).

- most reused names: `Dirty` x52, `Skyish` x50, `Spaced Out` x14, `Rooftop` x12, `Grassy` x3

**Objects.** 1814 live of 6696 slots (27.1% fill); per room mean 6.50, median 6, max 24; 5 rooms at the 24-object ceiling, 100 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 345 | 327 | 179 | 122 | 55 | 109 | 91 | 163 | 423 |

Top 15 `what` codes: `kCloud` 0x8C 304, `kFloorVent` 0x01 112, `kInvisBlower` 0x0D 105, `kInvisBounce` 0x1F 103, `kInvisLight` 0x58 63, `kSewerGrate` 0x05 46, `kShelf` 0x12 39, `kBalloon` 0x71 39, `kCeilingTrans` 0x36 38, `kCabinet` 0x13 34, `kSparkle` 0x2D 34, `kCustomPict` 0x6E 34, `kCopterLf` 0x72 32, `kCounter` 0x17 31, `kBooks` 0x1E 30.

Distinct `what` codes used: 114 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 179, enemies 163 (ratio 1.10), batteries 15, rubber bands 10, aluminum foil 6, helium 5, clocks 43, stars 3. `CountTotalHousePoints` = **68500** points, of which 6000 come from `kInvisBonus.data.c.points`.

**Links.** 137 link-bearing slots; 100 have `where` != -1; 100 fully linked (`who` != 255); 99 resolved to a live room; 1 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 186 two-way, 14 one-way, 48 walled. North/South adjacent pairs: 96 two-way, 13 one-way, 131 walled. Staircases: 12 up / 12 down (unpaired 0 / 0). Undirected components 13 (largest 244, the one holding `firstRoom` 244). Forward-reachable from `firstRoom` 218 of 279 rooms; BFS eccentricity 46.

**Stars.** 3, at BFS depths [21, 36, 38] of 46: room 26 "Special Delivery" x1; room 145 "Finally Home." x1; room 157 "Emergence" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x45 (1991-1993, 3001-3005, 3300-3303, 3305-3311, 10001-10017, 10026, 10049, 10065-10066, 10100-10104); `bnds` x14 (3000-3008, 3300-3304); `icl4` x1 (-16455); `icl8` x2 (-16455, 128); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x2 (3001, 3003); `vers` x2 (1-2).

Shadows application `PICT` IDs: 1991-1993.

`kSoundTrigger` `data.e.where` values: 3001 x1, 3003 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 34 placements over 25 distinct IDs, top: 10012 x5, 10011 x2, 10017 x2, 10100 x2, 10010 x2, 10007 x2, 10066 x1, 10065 x1. All resolve inside this fork.

**Notes.**

- 1 link(s) point at a (floor, suite) cell that holds no room *and* have `who` != 255
- 61 room(s) are not forward-reachable from `firstRoom` under the static model
- ships a saved game (`hasGame` = 1): `savedGame.roomNumber` = 45, score 5900, 5 gliders

### 4.11 In The Mirror

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/In The Mirror.binhex`, 243295 bytes; BinHex name `In The Mirror`, type `gliH`, creator `ozm5`, Finder flags `0x0504` |
| data fork | 34622 bytes (866 + 348 x 97 = 34622) |
| resource fork | 151870 bytes |
| `.mov` sidecar | none |
| author (README.md:6-13) | Jonathan Chin (alias Paul Finn) |
| `version` | 0x0200 |
| `unusedShort` | 147 |
| `timeStamp` | 0x2C329401 -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=50 h=424 -> glider spawn rect {top 50, left 424, bottom 70, right 472} |
| `firstRoom` | 6 = "Let's Begin" at (floor 2, suite 66) |
| `nRooms` | 97 (real 97, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `Congratulations! Good thing none of the mirrors broke!` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 4100 by `Ozma`, rooms visited 11 |

**Geometry.** 12 occupied floors (-2..9), 17 occupied suites (55..71); grid occupancy 97 / (12 x 17) = 0.48.

- rooms per floor (floor:count, north-up): 9:3, 8:3, 7:3, 6:3, 5:6, 4:14, 3:15, 2:17, 1:17, 0:10, -1:3, -2:3
- rooms per suite (suite:count, west-first): 55:3, 56:10, 57:10, 58:10, 59:4, 60:2, 61:5, 62:8, 63:8, 64:7, 65:5, 66:5, 67:5, 68:5, 69:4, 70:4, 71:2

**Backgrounds.** 15 distinct IDs over 97 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 2 |
| 2001 | kPaneledRoom | 4 |
| 2002 | kBasement | 7 |
| 2003 | kChildsRoom | 4 |
| 2005 | kUnfinishedRoom | 4 |
| 2006 | kSwingersRoom | 2 |
| 2007 | kBathroom | 1 |
| 2011 | kDirt | 9 |
| 2012 | kMeadow | 10 |
| 2014 | kRoof | 7 |
| 2015 | kSky | 34 |
| 2016 | kStratosphere | 3 |
| 2017 | kStars | 6 |

- user-art backgrounds (`>= kUserBackground` 3000): 2 distinct IDs over 4 rooms - 3300-3301
  - structure sub-range 3000-3299: 0 IDs / 0 rooms; open sub-range 3300-3799: 2 IDs / 4 rooms
  - most-used: 3300 (3 rooms), 3301 (1 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 3 rooms; `bounds`==0 in 93, non-zero in 4; `leftStart` mode 32 (93 rooms), `rightStart` mode 32 (92 rooms); rooms named exactly "Untitled Room": 0; distinct room names 58 over 97 rooms (reuse ratio 1.67).

- most reused names: `Sky` x28, `Roof` x6, `Starry` x5, `Meadow` x3, `Strato` x2

**Objects.** 795 live of 2328 slots (34.1% fill); per room mean 8.20, median 7, max 24; 2 rooms at the 24-object ceiling, 18 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 101 | 87 | 115 | 28 | 68 | 34 | 34 | 74 | 254 |

Top 15 `what` codes: `kCloud` 0x8C 175, `kInvisSwitch` 0x46 55, `kMirror` 0x82 50, `kFoil` 0x2A 37, `kFloorVent` 0x01 29, `kBalloon` 0x71 23, `kSewerGrate` 0x05 22, `kInvisBlower` 0x0D 20, `kInvisBounce` 0x1F 20, `kInvisLight` 0x58 18, `kCustomPict` 0x6E 17, `kCopterLf` 0x72 15, `kMilkCrate` 0x16 14, `kBlueClock` 0x22 14, `kInvisBonus` 0x2B 10.

Distinct `what` codes used: 97 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 115, enemies 74 (ratio 1.55), batteries 3, rubber bands 6, aluminum foil 37, helium 5, clocks 31, stars 1. `CountTotalHousePoints` = **30700** points, of which 3600 come from `kInvisBonus.data.c.points`.

**Links.** 75 link-bearing slots; 68 have `where` != -1; 68 fully linked (`who` != 255); 68 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 64 two-way, 3 one-way, 14 walled. North/South adjacent pairs: 56 two-way, 0 one-way, 24 walled. Staircases: 5 up / 5 down (unpaired 0 / 0). Undirected components 1 (largest 97, the one holding `firstRoom` 97). Forward-reachable from `firstRoom` 96 of 97 rooms; BFS eccentricity 35.

**Stars.** 1, at BFS depths [32] of 35: room 41 "Gift of Light" x1.

**Resource fork.** `ICN#` x2 (-16455, 128); `PICT` x15 (1991-1993, 3300-3301, 10002, 10006, 10008, 10018, 10025-10026, 10034, 10036, 10038, 10074); `icl4` x2 (-16455, 128); `icl8` x2 (-16455, 128); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x2 (3001-3002); `vers` x2 (1-2).

Shadows application `PICT` IDs: 1991-1993.

`kSoundTrigger` `data.e.where` values: 3001 x3, 3002 x2, 10000 x2. **Silent: 10000 not present in this fork.**

`kCustomPict` `data.g.height` (PICT ID) values: 17 placements over 9 distinct IDs, top: 10026 x4, 10038 x3, 10008 x3, 10025 x2, 10002 x1, 10018 x1, 10036 x1, 10074 x1. All resolve inside this fork.

**Notes.**

- 1 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.12 Land of Illusion

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Land of Illusion.binhex`, 659819 bytes; BinHex name `Land of Illusion`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 106310 bytes (866 + 348 x 303 = 106310) |
| resource fork | 401793 bytes |
| `.mov` sidecar | `Land of Illusion.mov`, 16016 bytes |
| author (README.md:6-13) | Ward Hartenstein |
| `version` | 0x0200 |
| `unusedShort` | 259 |
| `timeStamp` | 0x2C6BE354 -> `& 1` = 0, house unlocked |
| `flags` | 2 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=True |
| `initial` | v=163 h=245 -> glider spawn rect {top 163, left 245, bottom 183, right 293} |
| `firstRoom` | 43 = "Magic Mirror" at (floor 15, suite 54) |
| `nRooms` | 303 (real 303, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 30 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `        The real world may look a bit different    now that you've been to the Land of Illusion!` |
| `highScores.banner` | `Land of Illusion` |
| top score | 0 (no scores recorded) |

**Geometry.** 20 occupied floors (-2..17), 29 occupied suites (49..77); grid occupancy 303 / (20 x 29) = 0.52.

- rooms per floor (floor:count, north-up): 17:6, 16:9, 15:9, 14:6, 13:3, 12:9, 11:11, 10:12, 9:12, 8:12, 7:21, 6:21, 5:19, 4:24, 3:24, 2:24, 1:28, 0:28, -1:21, -2:4
- rooms per suite (suite:count, west-first): 49:3, 50:4, 51:4, 52:4, 53:16, 54:15, 55:15, 56:9, 57:16, 58:16, 59:13, 60:5, 61:15, 62:16, 63:18, 64:10, 65:10, 66:10, 67:9, 68:13, 69:13, 70:12, 71:9, 72:9, 73:9, 74:9, 75:9, 76:9, 77:3

**Backgrounds.** 12 distinct IDs over 303 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2011 | kDirt | 17 |
| 2012 | kMeadow | 2 |
| 2014 | kRoof | 5 |
| 2015 | kSky | 23 |
| 2017 | kStars | 9 |

- user-art backgrounds (`>= kUserBackground` 3000): 7 distinct IDs over 247 rooms - 3001-3005, 3301-3302
  - structure sub-range 3000-3299: 5 IDs / 83 rooms; open sub-range 3300-3799: 2 IDs / 164 rooms
  - most-used: 3302 (115 rooms), 3301 (49 rooms), 3003 (35 rooms), 3005 (19 rooms), 3004 (15 rooms), 3001 (8 rooms), 3002 (6 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 9 rooms; `bounds`==0 in 76, non-zero in 227; `leftStart` mode 32 (274 rooms), `rightStart` mode 32 (281 rooms); rooms named exactly "Untitled Room": 0; distinct room names 111 over 303 rooms (reuse ratio 2.73).

- most reused names: `Black On Black` x61, `Edge Of Light` x27, `Vortex Side` x22, `Skyslide` x21, `Mudslide` x17

**Objects.** 1816 live of 7272 slots (25.0% fill); per room mean 5.99, median 1, max 24; 22 rooms at the 24-object ceiling, 53 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 508 | 254 | 150 | 140 | 249 | 262 | 72 | 62 | 119 |

Top 15 `what` codes: `kInvisBlower` 0x0D 368, `kInvisLight` 0x58 210, `kInvisObstacle` 0x1C 124, `kInvisSwitch` 0x46 118, `kTrigger` 0x47 100, `kInvisBounce` 0x1F 82, `kLiftArea` 0x10 49, `kMirror` 0x82 47, `kFloorVent` 0x01 43, `kSparkle` 0x2D 42, `kCustomPict` 0x6E 41, `kCeilingTrans` 0x36 38, `kInvisTrans` 0x3F 38, `kCloud` 0x8C 35, `kFloorTrans` 0x35 33.

Distinct `what` codes used: 85 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 150, enemies 62 (ratio 2.42), batteries 0, rubber bands 1, aluminum foil 0, helium 14, clocks 42, stars 5. `CountTotalHousePoints` = **77700** points, of which 3200 come from `kInvisBonus.data.c.points`.

**Links.** 359 link-bearing slots; 358 have `where` != -1; 310 fully linked (`who` != 255); 310 resolved to a live room; 48 dangling (of which 48 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 144 two-way, 25 one-way, 92 walled. North/South adjacent pairs: 157 two-way, 64 one-way, 39 walled. Staircases: 14 up / 14 down (unpaired 0 / 0). Undirected components 6 (largest 289, the one holding `firstRoom` 289). Forward-reachable from `firstRoom` 278 of 303 rooms; BFS eccentricity 26.

**Stars.** 5, at BFS depths [10, 12, 15, 16, 25] of 26: room 1 "Steamy Star" x1; room 59 ". . . Lightness Of Being" x1; room 115 "The Dollhouse" x1; room 158 "Final Reward" x1; room 240 "The Star Of The Show" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x20 (1017-1018, 1991-1993, 3001-3005, 3301-3302, 10000-10005, 10007-10008); `bnds` x5 (3001-3003, 3301-3302); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455).

Shadows application `PICT` IDs: 1017-1018, 1991-1993, 10000.

`kCustomPict` `data.g.height` (PICT ID) values: 41 placements over 8 distinct IDs, top: 10003 x14, 10000 x9, 10004 x5, 10008 x3, 10001 x3, 10002 x3, 10005 x3, 10007 x1. All resolve inside this fork.

**Notes.**

- 1 room(s) carry a non-zero `bounds` while using a built-in background, where `DetermineRoomOpenings` ignores it: room 9 "The Other Side" (bg 2015, bounds 3)
- 25 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.13 Leviathan

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Leviathan.binhex`, 2733762 bytes; BinHex name `Leviathan`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 165122 bytes (866 + 348 x 472 = 165122) |
| resource fork | 1901282 bytes |
| `.mov` sidecar | `Leviathan.mov`, 65920 bytes |
| author (README.md:6-13) | Jonathan Chin (alias Paul Finn) |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C329E33 -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=44 h=229 -> glider spawn rect {top 44, left 229, bottom 64, right 277} |
| `firstRoom` | 39 = "Black" at (floor 18, suite 55) |
| `nRooms` | 472 (real 472, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `Congratulations on beating Leviathan! You're obviously a great Glider player! Good luck in other houses! ` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 8400 by `Ozma`, rooms visited 40 |

**Geometry.** 26 occupied floors (-6..19), 58 occupied suites (0..94); grid occupancy 472 / (26 x 58) = 0.31.

- rooms per floor (floor:count, north-up): 19:3, 18:3, 17:3, 16:3, 15:6, 14:6, 13:6, 12:6, 11:8, 10:8, 9:13, 8:19, 7:27, 6:36, 5:45, 4:42, 3:49, 2:58, 1:58, 0:31, -1:11, -2:5, -3:10, -4:6, -5:7, -6:3
- rooms per suite (suite:count, west-first): 0:3, 1:3, 2:3, 3:3, 4:3, 5:3, 6:3, 7:3, 45:3, 46:5, 47:7, 48:7, 49:10, 50:19, 51:21, 52:19, 53:13, 54:19, 55:19, 56:19, 57:12, 58:11, 59:10, 60:11, 61:5, 62:9, 63:9, 64:9, 65:9, 66:10, 67:10, 68:10, 69:9, 70:9, 71:6, 72:5, 73:5, 74:6, 75:7, 76:7, 77:7, 78:8, 79:8, 80:8, 81:7, 82:5, 83:5, 84:5, 85:9, 86:9, 87:9, 88:9, 89:9, 90:7, 91:7, 92:2, 93:2, 94:2

**Backgrounds.** 41 distinct IDs over 472 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 26 |
| 2001 | kPaneledRoom | 25 |
| 2002 | kBasement | 24 |
| 2003 | kChildsRoom | 6 |
| 2004 | kAsianRoom | 5 |
| 2005 | kUnfinishedRoom | 13 |
| 2006 | kSwingersRoom | 8 |
| 2007 | kBathroom | 7 |
| 2008 | kLibrary | 2 |
| 2009 | kGarden | 3 |
| 2010 | kSkywalk | 16 |
| 2011 | kDirt | 24 |
| 2012 | kMeadow | 17 |
| 2013 | kField | 2 |
| 2014 | kRoof | 48 |
| 2015 | kSky | 149 |
| 2016 | kStratosphere | 6 |
| 2017 | kStars | 27 |

- user-art backgrounds (`>= kUserBackground` 3000): 23 distinct IDs over 64 rooms - 3000-3004, 3300-3309, 3311-3312, 3314-3319
  - structure sub-range 3000-3299: 5 IDs / 10 rooms; open sub-range 3300-3799: 18 IDs / 54 rooms
  - most-used: 3301 (11 rooms), 3308 (10 rooms), 3302 (6 rooms), 3311 (6 rooms), 3001 (5 rooms), 3309 (4 rooms), 3000 (2 rooms), 3305 (2 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 0 rooms; `bounds`==0 in 417, non-zero in 55; `leftStart` mode 0 (362 rooms), `rightStart` mode 0 (358 rooms); rooms named exactly "Untitled Room": 0; distinct room names 280 over 472 rooms (reuse ratio 1.69).

- most reused names: `Open` x116, `Rooftop` x33, `Starry` x13, `Soil` x12, `Space` x8

**Objects.** 3335 live of 11328 slots (29.4% fill); per room mean 7.07, median 6.0, max 24; 5 rooms at the 24-object ceiling, 129 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 648 | 599 | 387 | 234 | 114 | 191 | 225 | 344 | 593 |

Top 15 `what` codes: `kCloud` 0x8C 381, `kFloorVent` 0x01 218, `kInvisBlower` 0x0D 176, `kCustomPict` 0x6E 131, `kInvisBounce` 0x1F 113, `kInvisLight` 0x58 103, `kBalloon` 0x71 102, `kSparkle` 0x2D 82, `kShelf` 0x12 77, `kCeilingTrans` 0x36 70, `kSewerGrate` 0x05 67, `kCounter` 0x17 64, `kCabinet` 0x13 58, `kCopterLf` 0x72 58, `kTable` 0x11 54.

Distinct `what` codes used: 114 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 387, enemies 344 (ratio 1.12), batteries 27, rubber bands 32, aluminum foil 8, helium 6, clocks 70, stars 6. `CountTotalHousePoints` = **118100** points, of which 10700 come from `kInvisBonus.data.c.points`.

**Links.** 254 link-bearing slots; 191 have `where` != -1; 189 fully linked (`who` != 255); 168 resolved to a live room; 23 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 304 two-way, 10 one-way, 105 walled. North/South adjacent pairs: 245 two-way, 18 one-way, 133 walled. Staircases: 28 up / 28 down (unpaired 0 / 0). Undirected components 11 (largest 462, the one holding `firstRoom` 462). Forward-reachable from `firstRoom` 458 of 472 rooms; BFS eccentricity 59.

**Stars.** 6, at BFS depths [14, 18, 44, 45, 48, 58] of 59: room 120 "Blessed Light!" x1; room 176 "The Eagle has Landed" x1; room 305 "Center of the Spiral" x1; room 339 "Starwalk" x1; room 376 ""You got another star!"" x1; room 387 "Stellar Sentries" x1.

**Resource fork.** `ICN#` x2 (-16455, 128); `PICT` x53 (1991-1993, 3000-3004, 3300-3309, 3311-3312, 3314-3319, 10001-10004, 10006, 10009-10012, 10020, 10025, 10031-10033, 10036, 10065-10066, 10079-10088); `bnds` x4 (3000-3001, 3300-3301); `icl4` x1 (-16455); `icl8` x2 (-16455, 128); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x12 (3000-3001, 3003-3012); `vers` x2 (1-2).

Shadows application `PICT` IDs: 1991-1993.

`kSoundTrigger` `data.e.where` values: 3000 x1, 3001 x1, 3003 x1, 3004 x1, 3005 x1, 3006 x1, 3007 x1, 3008 x1, 3009 x1, 3010 x1, 3011 x1, 3012 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 131 placements over 27 distinct IDs, top: 10020 x36, 10088 x7, 10036 x6, 10010 x6, 10001 x6, 10084 x5, 10025 x5, 10083 x5. All resolve inside this fork.

**Notes.**

- 21 link(s) point at a (floor, suite) cell that holds no room *and* have `who` != 255
- 14 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.14 Metropolis

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Metropolis.binhex`, 1324749 bytes; BinHex name `Metropolis`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 45062 bytes (866 + 348 x 127 = 45062) |
| resource fork | 946907 bytes |
| `.mov` sidecar | none |
| author (README.md:6-13) | Jonathan Chin (alias Paul Finn) & John Calhoun |
| `version` | 0x0200 |
| `unusedShort` | 196 |
| `timeStamp` | 0x2C34904B -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=24 h=245 -> glider spawn rect {top 24, left 245, bottom 44, right 293} |
| `firstRoom` | 8 = "Courtesy Desk" at (floor 1, suite 63) |
| `nRooms` | 127 (real 127, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 14 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `Thank you for visiting the Metropolis building -  Home of a thousand Drips, Darts, and Ducts!` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 2300 by `Ozma`, rooms visited 17 |

**Geometry.** 14 occupied floors (-1..12), 15 occupied suites (54..68); grid occupancy 127 / (14 x 15) = 0.60.

- rooms per floor (floor:count, north-up): 12:8, 11:8, 10:8, 9:8, 8:8, 7:8, 6:8, 5:8, 4:8, 3:8, 2:15, 1:15, 0:12, -1:5
- rooms per suite (suite:count, west-first): 54:3, 55:3, 56:3, 57:3, 58:3, 59:3, 60:3, 61:12, 62:12, 63:14, 64:14, 65:14, 66:14, 67:14, 68:12

**Backgrounds.** 22 distinct IDs over 127 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 2 |
| 2001 | kPaneledRoom | 1 |
| 2011 | kDirt | 7 |
| 2012 | kMeadow | 4 |
| 2014 | kRoof | 3 |
| 2015 | kSky | 4 |
| 2017 | kStars | 18 |

- user-art backgrounds (`>= kUserBackground` 3000): 15 distinct IDs over 88 rooms - 3000, 3002-3003, 3007-3011, 3300-3306
  - structure sub-range 3000-3299: 8 IDs / 48 rooms; open sub-range 3300-3799: 7 IDs / 40 rooms
  - most-used: 3305 (18 rooms), 3002 (16 rooms), 3011 (7 rooms), 3306 (7 rooms), 3000 (5 rooms), 3007 (5 rooms), 3008 (5 rooms), 3303 (5 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 17 rooms; `bounds`==0 in 39, non-zero in 88; `leftStart` mode 32 (117 rooms), `rightStart` mode 32 (122 rooms); rooms named exactly "Untitled Room": 0; distinct room names 87 over 127 rooms (reuse ratio 1.46).

- most reused names: `Nightwind` x17, `Star` x11, `Dirt` x7, `Sky` x4, `Español` x3

**Objects.** 809 live of 3048 slots (26.5% fill); per room mean 6.37, median 5, max 20; 0 rooms at the 24-object ceiling, 29 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 160 | 123 | 63 | 48 | 54 | 91 | 88 | 129 | 53 |

Top 15 `what` codes: `kCustomPict` 0x6E 68, `kFloorVent` 0x01 62, `kInvisLight` 0x58 54, `kInvisBlower` 0x0D 39, `kBalloon` 0x71 32, `kDrip` 0x77 29, `kCloud` 0x8C 27, `kCeilingTrans` 0x36 26, `kSewerGrate` 0x05 23, `kLiftArea` 0x10 22, `kTrackLight` 0x57 21, `kShelf` 0x12 19, `kCabinet` 0x13 19, `kInvisSwitch` 0x46 18, `kCopterRt` 0x73 16.

Distinct `what` codes used: 84 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 63, enemies 129 (ratio 0.49), batteries 8, rubber bands 6, aluminum foil 1, helium 4, clocks 13, stars 4. `CountTotalHousePoints` = **39300** points, of which 2000 come from `kInvisBonus.data.c.points`.

**Links.** 92 link-bearing slots; 74 have `where` != -1; 74 fully linked (`who` != 255); 74 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 76 two-way, 14 one-way, 22 walled. North/South adjacent pairs: 45 two-way, 6 one-way, 61 walled. Staircases: 3 up / 3 down (unpaired 0 / 0). Undirected components 11 (largest 110, the one holding `firstRoom` 110). Forward-reachable from `firstRoom` 110 of 127 rooms; BFS eccentricity 39.

**Stars.** 4, at BFS depths [10, 22, 32, 36] of 39: room 34 "The stars are out tonight" x1; room 64 "Reach for a Star" x1; room 88 "Takeoff" x1; room 114 "Star in the Heavens" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x39 (1991-1993, 1999, 3000, 3002-3003, 3007-3011, 3300-3306, 10000-10019); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `vers` x1 (1).

Shadows application `PICT` IDs: 1991-1993, 1999, 10000.

`kCustomPict` `data.g.height` (PICT ID) values: 68 placements over 20 distinct IDs, top: 10003 x20, 10000 x8, 10019 x8, 10002 x7, 10008 x3, 10009 x3, 10016 x3, 10004 x2. All resolve inside this fork.

**Notes.**

- 17 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.15 Nemo's Market

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Nemo's Market.binhex`, 815640 bytes; BinHex name `Nemo's Market`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 44018 bytes (866 + 348 x 124 = 44018) |
| resource fork | 590602 bytes |
| `.mov` sidecar | `Nemo's Market.mov`, 28541 bytes |
| author (README.md:6-13) | Ward Hartenstein |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C2C9CD1 -> `& 1` = 1, house LOCKED |
| `flags` | 2 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=True |
| `initial` | v=51 h=342 -> glider spawn rect {top 51, left 342, bottom 71, right 390} |
| `firstRoom` | 0 = "Welcome to Nemo's Market!" at (floor 1, suite 55) |
| `nRooms` | 124 (real 124, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `  Now, about those bananas you shoplifted . . .` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 20700 by `Ozma`, rooms visited 23 |

**Geometry.** 3 occupied floors (0..2), 42 occupied suites (54..95); grid occupancy 124 / (3 x 42) = 0.98.

- rooms per floor (floor:count, north-up): 2:40, 1:42, 0:42
- rooms per suite (suite:count, west-first): 54:2, 55:3, 56:3, 57:3, 58:3, 59:3, 60:3, 61:3, 62:3, 63:3, 64:3, 65:3, 66:3, 67:3, 68:3, 69:3, 70:3, 71:3, 72:3, 73:3, 74:3, 75:3, 76:3, 77:3, 78:3, 79:3, 80:3, 81:3, 82:3, 83:3, 84:3, 85:3, 86:3, 87:3, 88:3, 89:3, 90:3, 91:3, 92:3, 93:3, 94:3, 95:2

**Backgrounds.** 4 distinct IDs over 124 rooms.

- user-art backgrounds (`>= kUserBackground` 3000): 4 distinct IDs over 124 rooms - 3301-3304
  - structure sub-range 3000-3299: 0 IDs / 0 rooms; open sub-range 3300-3799: 4 IDs / 124 rooms
  - most-used: 3302 (82 rooms), 3301 (26 rooms), 3303 (12 rooms), 3304 (4 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 23 rooms; `bounds`==0 in 0, non-zero in 124; `leftStart` mode 32 (118 rooms), `rightStart` mode 32 (119 rooms); rooms named exactly "Untitled Room": 0; distinct room names 35 over 124 rooms (reuse ratio 3.54).

- most reused names: `Sub Nemo` x42, `Nemo's Roof` x40, `Parking Lot` x10

**Objects.** 836 live of 2976 slots (28.1% fill); per room mean 6.74, median 1.0, max 24; 18 rooms at the 24-object ceiling, 0 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 72 | 75 | 75 | 24 | 29 | 125 | 385 | 41 | 10 |

Top 15 `what` codes: `kCustomPict` 0x6E 358, `kInvisLight` 0x58 94, `kFloorVent` 0x01 55, `kInvisObstacle` 0x1C 49, `kFlourescent` 0x56 19, `kCobweb` 0x79 17, `kRedClock` 0x21 14, `kYellowClock` 0x23 14, `kInvisBonus` 0x2B 13, `kInvisSwitch` 0x46 13, `kCeilingTrans` 0x36 12, `kBlueClock` 0x22 11, `kSoundTrigger` 0x49 11, `kFish` 0x78 11, `kStool` 0x1A 6.

Distinct `what` codes used: 68 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 75, enemies 41 (ratio 1.83), batteries 1, rubber bands 1, aluminum foil 2, helium 1, clocks 40, stars 5. `CountTotalHousePoints` = **52200** points, of which 2100 come from `kInvisBonus.data.c.points`.

**Links.** 36 link-bearing slots; 27 have `where` != -1; 27 fully linked (`who` != 255); 27 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 26 two-way, 85 one-way, 10 walled. North/South adjacent pairs: 10 two-way, 30 one-way, 42 walled. Staircases: 0 up / 0 down (unpaired 0 / 0). Undirected components 2 (largest 82, the one holding `firstRoom` 82). Forward-reachable from `firstRoom` 81 of 124 rooms; BFS eccentricity 39.

**Stars.** 5, at BFS depths [6, 11, 17, 23, 30] of 39: room 6 "8 Second Shopping Spree" x1; room 16 "7 Second Shopping Spree" x1; room 58 "12 Second Shopping Spree" x1; room 74 "9 Second Shopping Spree" x1; room 122 "Into The Recycling Bin" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x73 (1017-1018, 1991-1993, 3301-3304, 3981-3984, 10000-10059); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x5 (3001-3005).

Shadows application `PICT` IDs: 1017-1018, 1991-1993, 3981-3984, 10000.

`kSoundTrigger` `data.e.where` values: 3001 x2, 3002 x3, 3003 x1, 3004 x2, 3005 x3.

`kCustomPict` `data.g.height` (PICT ID) values: 358 placements over 60 distinct IDs, top: 10008 x20, 10036 x16, 10002 x15, 10009 x15, 10037 x14, 10000 x13, 10014 x12, 10001 x12. All resolve inside this fork.

**Notes.**

- 43 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.16 Rainbow's End

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Rainbow's End.binhex`, 513683 bytes; BinHex name `Rainbow's End`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 78470 bytes (866 + 348 x 223 = 78470) |
| resource fork | 316065 bytes |
| `.mov` sidecar | `Rainbow's End.mov`, 7021 bytes |
| author (README.md:6-13) | Ward Hartenstein |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C2C9DD1 -> `& 1` = 1, house LOCKED |
| `flags` | 2 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=True |
| `initial` | v=200 h=237 -> glider spawn rect {top 200, left 237, bottom 220, right 285} |
| `firstRoom` | 30 = "Out Of Thin Air" at (floor 5, suite 62) |
| `nRooms` | 223 (real 223, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `  May the colors of the Rainbow be with you always!` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 2000 by `Ozma`, rooms visited 8 |

**Geometry.** 14 occupied floors (-7..6), 36 occupied suites (52..87); grid occupancy 223 / (14 x 36) = 0.44.

- rooms per floor (floor:count, north-up): 6:6, 5:18, 4:20, 3:27, 2:36, 1:36, 0:36, -1:15, -2:11, -3:6, -4:3, -5:3, -6:3, -7:3
- rooms per suite (suite:count, west-first): 52:4, 53:4, 54:4, 55:4, 56:4, 57:6, 58:7, 59:9, 60:10, 61:10, 62:14, 63:14, 64:14, 65:9, 66:8, 67:8, 68:7, 69:7, 70:7, 71:7, 72:6, 73:6, 74:6, 75:6, 76:6, 77:5, 78:4, 79:3, 80:3, 81:3, 82:3, 83:3, 84:3, 85:3, 86:3, 87:3

**Backgrounds.** 22 distinct IDs over 223 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 4 |
| 2001 | kPaneledRoom | 2 |
| 2002 | kBasement | 4 |
| 2003 | kChildsRoom | 1 |
| 2004 | kAsianRoom | 3 |
| 2005 | kUnfinishedRoom | 3 |
| 2006 | kSwingersRoom | 3 |
| 2007 | kBathroom | 1 |
| 2008 | kLibrary | 1 |
| 2009 | kGarden | 7 |
| 2010 | kSkywalk | 5 |
| 2011 | kDirt | 76 |
| 2012 | kMeadow | 5 |
| 2013 | kField | 1 |
| 2015 | kSky | 40 |
| 2016 | kStratosphere | 18 |
| 2017 | kStars | 6 |

- user-art backgrounds (`>= kUserBackground` 3000): 5 distinct IDs over 43 rooms - 3001, 3301-3304
  - structure sub-range 3000-3299: 1 IDs / 11 rooms; open sub-range 3300-3799: 4 IDs / 32 rooms
  - most-used: 3303 (12 rooms), 3001 (11 rooms), 3304 (8 rooms), 3301 (7 rooms), 3302 (5 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 8 rooms; `bounds`==0 in 210, non-zero in 13; `leftStart` mode 32 (180 rooms), `rightStart` mode 32 (181 rooms); rooms named exactly "Untitled Room": 0; distinct room names 126 over 223 rooms (reuse ratio 1.77).

- most reused names: `Underdog` x53, `Some Air` x24, `No Air` x14, `Night Sky` x6, `Beyond` x5

**Objects.** 1673 live of 5352 slots (31.3% fill); per room mean 7.50, median 3, max 24; 17 rooms at the 24-object ceiling, 56 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 394 | 376 | 162 | 142 | 70 | 105 | 121 | 61 | 242 |

Top 15 `what` codes: `kInvisBounce` 0x1F 241, `kInvisBlower` 0x0D 219, `kFlower` 0x85 117, `kInvisTrans` 0x3F 55, `kCloud` 0x8C 50, `kInvisObstacle` 0x1C 49, `kInvisLight` 0x58 48, `kCustomPict` 0x6E 46, `kLightBulb` 0x52 41, `kInvisSwitch` 0x46 40, `kLiftArea` 0x10 39, `kFloorVent` 0x01 32, `kYellowClock` 0x23 26, `kInvisBonus` 0x2B 25, `kBlueClock` 0x22 24.

Distinct `what` codes used: 110 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 162, enemies 61 (ratio 2.66), batteries 2, rubber bands 6, aluminum foil 2, helium 2, clocks 71, stars 5. `CountTotalHousePoints` = **77500** points, of which 2500 come from `kInvisBonus.data.c.points`.

**Links.** 174 link-bearing slots; 174 have `where` != -1; 138 fully linked (`who` != 255); 138 resolved to a live room; 36 dangling (of which 36 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 178 two-way, 22 one-way, 9 walled. North/South adjacent pairs: 105 two-way, 13 one-way, 69 walled. Staircases: 9 up / 9 down (unpaired 0 / 0). Undirected components 2 (largest 220, the one holding `firstRoom` 220). Forward-reachable from `firstRoom` 187 of 223 rooms; BFS eccentricity 24.

**Stars.** 5, at BFS depths [6, 9, 11, 12, 19] of 24: room 91 "Underground Cafe" x1; room 105 "Aztec Roof" x1; room 106 "Rainbow's End" x1; room 114 "Gotta Get That Switch!" x1; room 207 "Ride The Slide!" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x17 (1017-1018, 1991-1993, 3001, 3301-3304, 3964, 10000-10005); `bnds` x6 (3001, 3301-3305); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x1 (3000).

Shadows application `PICT` IDs: 1017-1018, 1991-1993, 3964, 10000.

`kSoundTrigger` `data.e.where` values: 3000 x2.

`kCustomPict` `data.g.height` (PICT ID) values: 46 placements over 6 distinct IDs, top: 10002 x20, 10004 x11, 10001 x6, 10000 x4, 10003 x3, 10005 x2. All resolve inside this fork.

**Notes.**

- 36 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.17 Sampler

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Sampler.binhex`, 2169 bytes; BinHex name `Sampler`, type `gliH`, creator `ozm5`, Finder flags `0x0100` |
| data fork | 1564 bytes (866 + 348 x 2 = 1562) |
| resource fork | 286 bytes |
| `.mov` sidecar | none |
| author (README.md:6-13) | unattributed (not in README credits) |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x3540BFE8 -> `& 1` = 0, house unlocked |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=85 h=384 -> glider spawn rect {top 85, left 384, bottom 105, right 432} |
| `firstRoom` | 1 = "Welcome" at (floor 1, suite 66) |
| `nRooms` | 2 (real 2, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 2 |
| `banner` (Str255) | `Welcome to Omid's Happy Home.` |
| `trailer` (Str255) | `Congratulations.` |
| `highScores.banner` | `Your Message Here` |
| top score | 5200 by `Your Name`, rooms visited 2 |

**Geometry.** 1 occupied floors (1..1), 2 occupied suites (66..67); grid occupancy 2 / (1 x 2) = 1.00.

- rooms per floor (floor:count, north-up): 1:2
- rooms per suite (suite:count, west-first): 66:1, 67:1

**Backgrounds.** 2 distinct IDs over 2 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2003 | kChildsRoom | 1 |
| 2012 | kMeadow | 1 |

**Room fields.** `visited`=1 in 2 rooms; `bounds`==0 in 2, non-zero in 0; `leftStart` mode 32 (2 rooms), `rightStart` mode 27 (1 rooms); rooms named exactly "Untitled Room": 0; distinct room names 2 over 2 rooms (reuse ratio 1.00).

**Objects.** 11 live of 48 slots (22.9% fill); per room mean 5.50, median 5.5, max 7; 0 rooms at the 24-object ceiling, 0 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 2 | 0 | 1 | 7 | 0 | 1 | 0 | 0 | 0 |

Top 15 `what` codes: `kMailboxLf` 0x33 2, `kDoorInLf` 0x37 2, `kFloorVent` 0x01 1, `kBBQ` 0x0C 1, `kStar` 0x2C 1, `kFloorTrans` 0x35 1, `kCeilingTrans` 0x36 1, `kDoorExRt` 0x39 1, `kCeilingLight` 0x51 1.

Distinct `what` codes used: 9 of the 117 that appear anywhere in the corpus.

Full vocabulary: `kMailboxLf` 0x33 x2, `kDoorInLf` 0x37 x2, `kFloorVent` 0x01 x1, `kBBQ` 0x0C x1, `kStar` 0x2C x1, `kFloorTrans` 0x35 x1, `kCeilingTrans` 0x36 x1, `kDoorExRt` 0x39 x1, `kCeilingLight` 0x51 x1.

**Economy.** prizes 1, enemies 0 (ratio n/a), batteries 0, rubber bands 0, aluminum foil 0, helium 0, clocks 0, stars 1. `CountTotalHousePoints` = **5200** points, of which 0 come from `kInvisBonus.data.c.points`.

**Links.** 4 link-bearing slots; 2 have `where` != -1; 2 fully linked (`who` != 255); 2 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 1 two-way, 0 one-way, 0 walled. North/South adjacent pairs: 0 two-way, 0 one-way, 0 walled. Staircases: 0 up / 0 down (unpaired 0 / 0). Undirected components 1 (largest 2, the one holding `firstRoom` 2). Forward-reachable from `firstRoom` 2 of 2 rooms; BFS eccentricity 1.

**Stars.** 1, at BFS depths [0] of 1: room 1 "Welcome" x1.

**Resource fork.** *(empty)*.

**Notes.**

- data fork is 1564 bytes but 866 + 348 x 2 = 1562 - **+2 slack bytes past the last room**

### 4.18 Slumberland

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Slumberland.binhex`, 1552921 bytes; BinHex name `Slumberland`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 134150 bytes (866 + 348 x 383 = 134150) |
| resource fork | 1041112 bytes |
| `.mov` sidecar | `Slumberland.mov`, 37640 bytes |
| author (README.md:6-13) | John Calhoun, Jonathan Chin, Steve Sullivan, Ward Hartenstein |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C53B149 -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=7 h=361 -> glider spawn rect {top 7, left 361, bottom 27, right 409} |
| `firstRoom` | 126 = "Welcome…" at (floor 1, suite 56) |
| `nRooms` | 383 (real 383, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | `Welcome to Slumberland!  House by john calhoun, Paul Finn,  Ward Hartenstein & Steve Sullivan.` |
| `trailer` (Str255) | `Congratulations!  You made it through the largest Glider house to date.  Try to improve your score.` |
| `highScores.banner` | `Your Message Here` |
| top score | 10800 by `Your Name`, rooms visited 25 |

**Geometry.** 18 occupied floors (-7..10), 48 occupied suites (0..86); grid occupancy 383 / (18 x 48) = 0.44.

- rooms per floor (floor:count, north-up): 10:5, 9:6, 8:14, 7:25, 6:28, 5:28, 4:28, 3:37, 2:46, 1:48, 0:37, -1:13, -2:12, -3:11, -4:12, -5:12, -6:11, -7:10
- rooms per suite (suite:count, west-first): 0:3, 1:3, 2:3, 5:3, 6:3, 7:3, 8:3, 9:3, 12:3, 13:4, 14:4, 15:4, 16:4, 17:4, 18:4, 19:3, 55:1, 56:6, 57:8, 58:8, 59:8, 60:12, 61:16, 62:16, 63:16, 64:16, 65:16, 66:16, 67:16, 68:15, 69:15, 70:15, 71:11, 72:9, 73:12, 74:8, 75:6, 76:2, 77:2, 78:6, 79:8, 80:8, 81:10, 82:11, 83:11, 84:11, 85:10, 86:4

**Backgrounds.** 37 distinct IDs over 383 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 37 |
| 2001 | kPaneledRoom | 20 |
| 2002 | kBasement | 20 |
| 2003 | kChildsRoom | 13 |
| 2004 | kAsianRoom | 13 |
| 2005 | kUnfinishedRoom | 11 |
| 2006 | kSwingersRoom | 9 |
| 2007 | kBathroom | 2 |
| 2008 | kLibrary | 1 |
| 2010 | kSkywalk | 1 |
| 2011 | kDirt | 96 |
| 2012 | kMeadow | 11 |
| 2013 | kField | 5 |
| 2014 | kRoof | 23 |
| 2015 | kSky | 59 |
| 2016 | kStratosphere | 14 |
| 2017 | kStars | 24 |

- user-art backgrounds (`>= kUserBackground` 3000): 20 distinct IDs over 24 rooms - 3000-3010, 3300-3308
  - structure sub-range 3000-3299: 11 IDs / 11 rooms; open sub-range 3300-3799: 9 IDs / 13 rooms
  - most-used: 3304 (5 rooms), 3000 (1 rooms), 3001 (1 rooms), 3002 (1 rooms), 3003 (1 rooms), 3004 (1 rooms), 3005 (1 rooms), 3006 (1 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 25 rooms; `bounds`==0 in 383, non-zero in 0; `leftStart` mode 32 (361 rooms), `rightStart` mode 32 (347 rooms); rooms named exactly "Untitled Room": 0; distinct room names 302 over 383 rooms (reuse ratio 1.27).

- most reused names: `Darling` x48, `We Drown` x13, `me and my sisters` x8, `Tease` x5, `Ballroom` x4

**Objects.** 2996 live of 9192 slots (32.6% fill); per room mean 7.82, median 7, max 24; 4 rooms at the 24-object ceiling, 52 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 769 | 418 | 386 | 254 | 146 | 231 | 97 | 224 | 471 |

Top 15 `what` codes: `kInvisBlower` 0x0D 265, `kFloorVent` 0x01 254, `kCloud` 0x8C 156, `kSewerGrate` 0x05 131, `kDrip` 0x77 97, `kCeilingTrans` 0x36 90, `kFlower` 0x85 86, `kMousehole` 0x83 84, `kInvisLight` 0x58 78, `kYellowClock` 0x23 73, `kCabinet` 0x13 71, `kShelf` 0x12 69, `kLightBulb` 0x52 66, `kCounter` 0x17 56, `kInvisBonus` 0x2B 48.

Distinct `what` codes used: 105 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 386, enemies 224 (ratio 1.72), batteries 25, rubber bands 21, aluminum foil 8, helium 0, clocks 134, stars 6. `CountTotalHousePoints` = **141400** points, of which 15000 come from `kInvisBonus.data.c.points`.

**Links.** 302 link-bearing slots; 302 have `where` != -1; 233 fully linked (`who` != 255); 233 resolved to a live room; 69 dangling (of which 69 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 226 two-way, 2 one-way, 117 walled. North/South adjacent pairs: 168 two-way, 4 one-way, 159 walled. Staircases: 33 up / 33 down (unpaired 0 / 0). Undirected components 8 (largest 365, the one holding `firstRoom` 365). Forward-reachable from `firstRoom` 357 of 383 rooms; BFS eccentricity 57.

**Stars.** 6, at BFS depths [12, 15, 35, 45, 48, 49] of 57: room 61 "There it is!" x1; room 104 "Steve's Room" x1; room 211 "Nirvana" x1; room 220 "Treasures" x1; room 273 "Spirit Dampening" x1; room 296 "Dark Star" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x20 (3000-3010, 3300-3308); `bnds` x20 (3000-3010, 3300-3308); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455).

**Notes.**

- 26 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.19 SpacePods

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/SpacePods.binhex`, 1005729 bytes; BinHex name `SpacePods`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 140762 bytes (866 + 348 x 402 = 140762) |
| resource fork | 636844 bytes |
| `.mov` sidecar | `SpacePods.mov`, 51291 bytes |
| author (README.md:6-13) | Ward Hartenstein |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C2CA019 -> `& 1` = 1, house LOCKED |
| `flags` | 2 -> wardBitSet=False phoneBitSet=True bannerStarCountOn=True |
| `initial` | v=72 h=30 -> glider spawn rect {top 72, left 30, bottom 92, right 78} |
| `firstRoom` | 259 = "Mission Statement" at (floor 8, suite 112) |
| `nRooms` | 402 (real 402, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | ` Sorry, it's illegal for individuals to own quintar crystals.  You'll have to turn it over to the authorities!  (Just kidding . . . . . Congratulations!)` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 6700 by `Ozma`, rooms visited 43 |

**Geometry.** 35 occupied floors (4..39), 18 occupied suites (110..127); grid occupancy 402 / (35 x 18) = 0.64.

- rooms per floor (floor:count, north-up): 39:9, 38:9, 37:9, 36:9, 35:9, 34:13, 33:13, 32:13, 31:6, 30:5, 29:5, 28:3, 26:9, 25:9, 24:14, 23:14, 22:14, 21:14, 20:14, 19:17, 18:17, 17:17, 16:15, 15:15, 14:15, 13:14, 12:14, 11:14, 10:10, 9:13, 8:13, 7:12, 6:9, 5:8, 4:8
- rooms per suite (suite:count, west-first): 110:3, 111:9, 112:9, 113:9, 114:14, 115:22, 116:28, 117:30, 118:35, 119:35, 120:35, 121:32, 122:31, 123:31, 124:26, 125:26, 126:16, 127:11

**Backgrounds.** 10 distinct IDs over 402 rooms.

- user-art backgrounds (`>= kUserBackground` 3000): 10 distinct IDs over 402 rooms - 3001-3010
  - structure sub-range 3000-3299: 10 IDs / 402 rooms; open sub-range 3300-3799: 0 IDs / 0 rooms
  - most-used: 3004 (172 rooms), 3009 (45 rooms), 3002 (35 rooms), 3003 (35 rooms), 3005 (33 rooms), 3007 (31 rooms), 3001 (24 rooms), 3006 (22 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 43 rooms; `bounds`==0 in 0, non-zero in 402; `leftStart` mode 32 (358 rooms), `rightStart` mode 32 (359 rooms); rooms named exactly "Untitled Room": 0; distinct room names 176 over 402 rooms (reuse ratio 2.28).

- most reused names: `Space` x146, `Pod Side` x35, `Pod Bottom` x21, `Pod Top` x20, `Force Field` x8

**Objects.** 5840 live of 9648 slots (60.5% fill); per room mean 14.53, median 13.5, max 24; 77 rooms at the 24-object ceiling, 0 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 778 | 809 | 171 | 17 | 289 | 420 | 2876 | 103 | 377 |

Top 15 `what` codes: `kCustomPict` 0x6E 2844, `kInvisBlower` 0x0D 612, `kInvisBounce` 0x1F 593, `kInvisLight` 0x58 384, `kMirror` 0x82 266, `kInvisSwitch` 0x46 120, `kLiftArea` 0x10 97, `kInvisObstacle` 0x1C 89, `kWallWindow` 0x86 74, `kCabinet` 0x13 71, `kSparkle` 0x2D 58, `kSewerBlower` 0x0F 56, `kPowerSwitch` 0x44 53, `kDrip` 0x77 47, `kLightBulb` 0x52 36.

Distinct `what` codes used: 60 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 171, enemies 103 (ratio 1.66), batteries 2, rubber bands 8, aluminum foil 0, helium 4, clocks 56, stars 1. `CountTotalHousePoints` = **73000** points, of which 3300 come from `kInvisBonus.data.c.points`.

**Links.** 282 link-bearing slots; 279 have `where` != -1; 279 fully linked (`who` != 255); 279 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 359 two-way, 2 one-way, 0 walled. North/South adjacent pairs: 365 two-way, 1 one-way, 0 walled. Staircases: 0 up / 0 down (unpaired 0 / 0). Undirected components 1 (largest 402, the one holding `firstRoom` 402). Forward-reachable from `firstRoom` 402 of 402 rooms; BFS eccentricity 24.

**Stars.** 1, at BFS depths [7] of 24: room 55 "Ion Generator" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x49 (1018, 1991-1993, 3001-3010, 10000-10034); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x3 (3000-3002).

Shadows application `PICT` IDs: 1018, 1991-1993, 10000.

`kSoundTrigger` `data.e.where` values: 3000 x14, 3001 x6, 3002 x4.

`kCustomPict` `data.g.height` (PICT ID) values: 2844 placements over 35 distinct IDs, top: 10000 x2104, 10001 x246, 10002 x241, 10006 x45, 10028 x24, 10005 x23, 10009 x21, 10033 x17. All resolve inside this fork.

### 4.20 Teddy World

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Teddy World.binhex`, 4434970 bytes; BinHex name `Teddy World`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 185654 bytes (866 + 348 x 531 = 185654) |
| resource fork | 3107348 bytes |
| `.mov` sidecar | `Teddy World.mov`, 28587 bytes |
| author (README.md:6-13) | Shawn Brenneman |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2C5EAB21 -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=41 h=362 -> glider spawn rect {top 41, left 362, bottom 61, right 410} |
| `firstRoom` | 0 = "Wanna come in and play?" at (floor 1, suite 7) |
| `nRooms` | 531 (real 531, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | *(empty)* |
| `trailer` (Str255) | `Hope you enjoyed it!  And remember: "No matter where you glide, there you are!" -Shawn Brenneman aol: A Marmoset` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 3400 by `Ozma`, rooms visited 16 |

**Geometry.** 24 occupied floors (-5..26), 101 occupied suites (1..113); grid occupancy 531 / (24 x 101) = 0.22.

- rooms per floor (floor:count, north-up): 26:6, 25:6, 24:6, 16:3, 15:3, 14:3, 13:3, 12:14, 11:17, 10:30, 9:38, 8:37, 7:19, 6:18, 5:13, 4:49, 3:49, 2:63, 1:63, -1:22, -2:21, -3:23, -4:15, -5:10
- rooms per suite (suite:count, west-first): 1:3, 2:3, 3:3, 4:7, 5:10, 6:15, 7:14, 8:9, 9:7, 10:4, 11:9, 12:9, 13:9, 14:9, 15:9, 16:14, 17:15, 18:14, 19:13, 20:10, 21:11, 22:11, 23:11, 24:5, 25:5, 26:9, 27:9, 28:10, 29:9, 30:8, 31:4, 32:4, 33:4, 34:8, 35:8, 36:4, 37:4, 38:4, 39:4, 40:5, 41:5, 42:5, 43:5, 44:4, 45:4, 46:4, 47:4, 48:4, 49:4, 56:4, 57:12, 58:4, 59:3, 60:3, 61:3, 62:3, 63:5, 64:5, 65:5, 66:5, 67:3, 68:3, 69:3, 70:3, 71:3, 72:3, 73:3, 74:3, 75:9, 76:9, 77:10, 78:4, 79:4, 80:4, 86:1, 87:1, 88:1, 89:1, 90:1, 91:1, 92:1, 93:1, 94:1, 95:1, 96:1, 97:1, 98:2, 99:2, 100:2, 101:2, 103:3, 104:3, 105:3, 106:3, 107:3, 108:3, 109:3, 110:3, 111:3, 112:3, 113:3

**Backgrounds.** 44 distinct IDs over 531 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 45 |
| 2001 | kPaneledRoom | 14 |
| 2003 | kChildsRoom | 6 |
| 2004 | kAsianRoom | 3 |
| 2005 | kUnfinishedRoom | 1 |
| 2006 | kSwingersRoom | 10 |
| 2008 | kLibrary | 1 |
| 2009 | kGarden | 1 |
| 2011 | kDirt | 25 |
| 2012 | kMeadow | 7 |
| 2013 | kField | 4 |
| 2014 | kRoof | 9 |
| 2015 | kSky | 47 |
| 2016 | kStratosphere | 15 |
| 2017 | kStars | 133 |

- user-art backgrounds (`>= kUserBackground` 3000): 29 distinct IDs over 210 rooms - 3000-3028
  - structure sub-range 3000-3299: 29 IDs / 210 rooms; open sub-range 3300-3799: 0 IDs / 0 rooms
  - most-used: 3000 (25 rooms), 3002 (22 rooms), 3014 (15 rooms), 3015 (15 rooms), 3011 (14 rooms), 3020 (14 rooms), 3021 (14 rooms), 3022 (10 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 16 rooms; `bounds`==0 in 320, non-zero in 211; `leftStart` mode 32 (521 rooms), `rightStart` mode 32 (521 rooms); rooms named exactly "Untitled Room": 0; distinct room names 238 over 531 rooms (reuse ratio 2.23).

- most reused names: `N` x247, `n` x34, `This is a long ride` x5, `Penthouse turns out tiny` x4, `a` x3

**Objects.** 3521 live of 12744 slots (27.6% fill); per room mean 6.63, median 1, max 24; 35 rooms at the 24-object ceiling, 199 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 699 | 571 | 586 | 230 | 196 | 258 | 287 | 262 | 432 |

Top 15 `what` codes: `kSlider` 0x2F 333, `kInvisObstacle` 0x1C 195, `kLiftArea` 0x10 186, `kInvisLight` 0x58 174, `kInvisBlower` 0x0D 168, `kInvisBounce` 0x1F 166, `kCustomPict` 0x6E 160, `kMirror` 0x82 134, `kInvisTrans` 0x3F 120, `kFloorVent` 0x01 117, `kKnifeSwitch` 0x45 98, `kCloud` 0x8C 94, `kGrecoVent` 0x0E 85, `kBalloon` 0x71 85, `kFlower` 0x85 75.

Distinct `what` codes used: 112 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 586, enemies 262 (ratio 2.24), batteries 15, rubber bands 27, aluminum foil 7, helium 0, clocks 42, stars 1. `CountTotalHousePoints` = **80500** points, of which 3900 come from `kInvisBonus.data.c.points`.

**Links.** 377 link-bearing slots; 297 have `where` != -1; 297 fully linked (`who` != 255); 297 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 289 two-way, 34 one-way, 134 walled. North/South adjacent pairs: 202 two-way, 59 one-way, 123 walled. Staircases: 15 up / 15 down (unpaired 0 / 0). Undirected components 81 (largest 445, the one holding `firstRoom` 445). Forward-reachable from `firstRoom` 401 of 531 rooms; BFS eccentricity 44.

**Stars.** 1, at BFS depths [6] of 44: room 506 "Nothing glorious..." x1.

**Resource fork.** `ICN#` x2 (-16455, 128); `PICT` x120 (1015-1016, 1991-1993, 3000-3028, 3975, 8253, 10000-10063, 10065-10079, 10081-10085); `icl4` x2 (-16455, 128); `icl8` x2 (-16455, 128); `ics#` x2 (-16455, 128); `ics4` x2 (-16455, 128); `ics8` x2 (-16455, 128); `vers` x2 (1-2).

Shadows application `PICT` IDs: 1015-1016, 1991-1993, 3975, 10000.

`kSoundTrigger` `data.e.where` values: 3000 x11. **Silent: 3000 not present in this fork.**

`kCustomPict` `data.g.height` (PICT ID) values: 160 placements over 80 distinct IDs, top: 10014 x10, 10040 x9, 10041 x8, 10027 x6, 10011 x6, 10001 x5, 10016 x4, 10008 x4. All resolve inside this fork.

**Notes.**

- 1 room(s) carry a non-zero `bounds` while using a built-in background, where `DetermineRoomOpenings` ignores it: room 0 "Wanna come in and play?" (bg 2012, bounds 1)
- 130 room(s) are not forward-reachable from `firstRoom` under the static model

### 4.21 The Asylum Pro

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/The Asylum Pro.binhex`, 727114 bytes; BinHex name `The Asylum Pro`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 49586 bytes (866 + 348 x 140 = 49586) |
| resource fork | 494107 bytes |
| `.mov` sidecar | none |
| author (README.md:6-13) | Steve Sullivan |
| `version` | 0x0200 |
| `unusedShort` | 0 |
| `timeStamp` | 0x2BFD3EA1 -> `& 1` = 1, house LOCKED |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=27 h=78 -> glider spawn rect {top 27, left 78, bottom 47, right 126} |
| `firstRoom` | 20 = "Greetings!" at (floor 1, suite 63) |
| `nRooms` | 140 (real 140, placeholders 0) |
| `hasGame` / `unusedBoolean` | 0 / 0 |
| `banner` (Str255) | `Welcome To The Asylum Pro!  If you want hints, have a comment,  flame, or anything else, send e-mail to: acmesteve@aol.com Thanks, I hope you enjoy this house.` |
| `trailer` (Str255) | `Congratulations, you made it all the way though The Asylum! You must be crazy!` |
| `highScores.banner` | `Spam Is Good For You.` |
| top score | 4100 by `Albert`, rooms visited 16 |

**Geometry.** 14 occupied floors (0..13), 10 occupied suites (63..72); grid occupancy 140 / (14 x 10) = 1.00.

- rooms per floor (floor:count, north-up): 13:10, 12:10, 11:10, 10:10, 9:10, 8:10, 7:10, 6:10, 5:10, 4:10, 3:10, 2:10, 1:10, 0:10
- rooms per suite (suite:count, west-first): 63:14, 64:14, 65:14, 66:14, 67:14, 68:14, 69:14, 70:14, 71:14, 72:14

**Backgrounds.** 17 distinct IDs over 140 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2000 | kSimpleRoom | 15 |
| 2001 | kPaneledRoom | 9 |
| 2003 | kChildsRoom | 5 |
| 2004 | kAsianRoom | 7 |
| 2005 | kUnfinishedRoom | 6 |
| 2006 | kSwingersRoom | 3 |
| 2007 | kBathroom | 2 |
| 2009 | kGarden | 2 |
| 2010 | kSkywalk | 1 |
| 2011 | kDirt | 10 |
| 2012 | kMeadow | 6 |
| 2014 | kRoof | 25 |
| 2015 | kSky | 40 |

- user-art backgrounds (`>= kUserBackground` 3000): 4 distinct IDs over 9 rooms - 3000-3003
  - structure sub-range 3000-3299: 4 IDs / 9 rooms; open sub-range 3300-3799: 0 IDs / 0 rooms
  - most-used: 3000 (4 rooms), 3001 (2 rooms), 3002 (2 rooms), 3003 (1 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 16 rooms; `bounds`==0 in 137, non-zero in 3; `leftStart` mode 0 (137 rooms), `rightStart` mode 0 (138 rooms); rooms named exactly "Untitled Room": 0; distinct room names 119 over 140 rooms (reuse ratio 1.18).

- most reused names: `Fixing A Hole` x10, `The Sky And I` x9, `Sky II` x4, `7th Floor Roof` x2

**Objects.** 1096 live of 3360 slots (32.6% fill); per room mean 7.83, median 6.0, max 24; 3 rooms at the 24-object ceiling, 0 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 326 | 86 | 113 | 112 | 81 | 72 | 68 | 48 | 190 |

Top 15 `what` codes: `kInvisBlower` 0x0D 160, `kFloorVent` 0x01 123, `kCloud` 0x8C 87, `kInvisSwitch` 0x46 38, `kCeilingLight` 0x51 32, `kCustomPict` 0x6E 32, `kInvisLight` 0x58 31, `kInvisTrans` 0x3F 28, `kMirror` 0x82 28, `kInvisBounce` 0x1F 25, `kTrigger` 0x47 24, `kFlower` 0x85 24, `kShelf` 0x12 22, `kSparkle` 0x2D 22, `kCeilingTrans` 0x36 22.

Distinct `what` codes used: 103 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 113, enemies 48 (ratio 2.35), batteries 6, rubber bands 4, aluminum foil 1, helium 0, clocks 30, stars 1. `CountTotalHousePoints` = **38000** points, of which 6100 come from `kInvisBonus.data.c.points`.

**Links.** 141 link-bearing slots; 120 have `where` != -1; 120 fully linked (`who` != 255); 120 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 89 two-way, 0 one-way, 37 walled. North/South adjacent pairs: 58 two-way, 24 one-way, 48 walled. Staircases: 18 up / 18 down (unpaired 0 / 0). Undirected components 1 (largest 140, the one holding `firstRoom` 140). Forward-reachable from `firstRoom` 140 of 140 rooms; BFS eccentricity 21.

**Stars.** 1, at BFS depths [1] of 21: room 62 "Finally!" x1.

**Resource fork.** `ICN#` x1 (-16455); `PICT` x17 (3000-3003, 10006, 10033, 10050, 10069-10070, 10074-10075, 12345, 12347-12351); `bnds` x2 (3000-3001); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455).

`kCustomPict` `data.g.height` (PICT ID) values: 32 placements over 13 distinct IDs, top: 10050 x13, 12345 x5, 12351 x2, 10074 x2, 10075 x2, 12350 x1, 10069 x1, 10070 x1. All resolve inside this fork.

### 4.22 Titanic

| header field | observed value |
|---|---|
| container | `GliderPRO/Houses/Titanic.binhex`, 1196597 bytes; BinHex name `Titanic`, type `gliH`, creator `ozm5`, Finder flags `0x0500` |
| data fork | 73250 bytes (866 + 348 x 208 = 73250) |
| resource fork | 845727 bytes |
| `.mov` sidecar | `Titanic.mov`, 6534 bytes |
| author (README.md:6-13) | Jonathan Chin (alias Paul Finn) & John Calhoun |
| `version` | 0x0200 |
| `unusedShort` | 2074 |
| `timeStamp` | 0x2D0246FE -> `& 1` = 0, house unlocked |
| `flags` | 0 -> wardBitSet=False phoneBitSet=False bannerStarCountOn=True |
| `initial` | v=97 h=39 -> glider spawn rect {top 97, left 39, bottom 117, right 87} |
| `firstRoom` | 92 = "Chute!" at (floor -1, suite 61) |
| `nRooms` | 208 (real 208, placeholders 0) |
| `hasGame` / `unusedBoolean` | 1 / 0 |
| `banner` (Str255) | `The Titanic is about to leave on her maiden voyage!  Welcome aboard! What?  You want off?  Find the front of the ship.  House by john calhoun & Paul Finn` |
| `trailer` (Str255) | `Excellent seamanship! You've escaped from the Titanic in the nick of time! Whew!` |
| `highScores.banner` | `The Return of Ozma!` |
| top score | 15300 by `Ozma`, rooms visited 33 |

**Geometry.** 13 occupied floors (-7..5), 28 occupied suites (46..73); grid occupancy 208 / (13 x 28) = 0.57.

- rooms per floor (floor:count, north-up): 5:6, 4:27, 3:27, 2:27, 1:25, 0:24, -1:23, -2:7, -3:6, -4:6, -5:12, -6:9, -7:9
- rooms per suite (suite:count, west-first): 46:3, 47:6, 48:6, 49:4, 50:5, 51:6, 52:6, 53:6, 54:9, 55:9, 56:9, 57:9, 58:9, 59:9, 60:12, 61:12, 62:12, 63:6, 64:6, 65:7, 66:7, 67:7, 68:7, 69:7, 70:8, 71:7, 72:7, 73:7

**Backgrounds.** 30 distinct IDs over 208 rooms.

| PICT ID | name | rooms |
|---|---|---|
| 2002 | kBasement | 9 |
| 2011 | kDirt | 9 |
| 2015 | kSky | 17 |

- user-art backgrounds (`>= kUserBackground` 3000): 27 distinct IDs over 173 rooms - 3300-3312, 3314-3327
  - structure sub-range 3000-3299: 0 IDs / 0 rooms; open sub-range 3300-3799: 27 IDs / 173 rooms
  - most-used: 3303 (46 rooms), 3300 (21 rooms), 3301 (21 rooms), 3302 (10 rooms), 3308 (9 rooms), 3325 (9 rooms), 3309 (8 rooms), 3314 (7 rooms)
  - all present in this house's own `PICT` resources: yes

**Room fields.** `visited`=1 in 1 rooms; `bounds`==0 in 35, non-zero in 173; `leftStart` mode 32 (174 rooms), `rightStart` mode 32 (178 rooms); rooms named exactly "Untitled Room": 0; distinct room names 114 over 208 rooms (reuse ratio 1.82).

- most reused names: `Murmur` x44, `Sky` x25, `Dirt` x9, `Deep` x8, `Pitch` x8

**Objects.** 1374 live of 4992 slots (27.5% fill); per room mean 6.61, median 5.0, max 24; 1 rooms at the 24-object ceiling, 24 rooms empty.

| class | blower | furniture | prize | transport | switch | light | appliance | enemy | clutter |
|---|---|---|---|---|---|---|---|---|---|
| count | 252 | 250 | 170 | 84 | 43 | 174 | 135 | 124 | 142 |

Top 15 `what` codes: `kInvisLight` 0x58 167, `kFloorVent` 0x01 119, `kCloud` 0x8C 103, `kCustomPict` 0x6E 94, `kInvisObstacle` 0x1C 76, `kLiftArea` 0x10 71, `kYellowClock` 0x23 51, `kCeilingTrans` 0x36 40, `kBalloon` 0x71 40, `kInvisBounce` 0x1F 34, `kMilkCrate` 0x16 33, `kInvisBonus` 0x2B 31, `kDrip` 0x77 28, `kSparkle` 0x2D 23, `kInvisBlower` 0x0D 21.

Distinct `what` codes used: 89 of the 117 that appear anywhere in the corpus.

**Economy.** prizes 170, enemies 124 (ratio 1.37), batteries 4, rubber bands 7, aluminum foil 3, helium 2, clocks 77, stars 1. `CountTotalHousePoints` = **86300** points, of which 14300 come from `kInvisBonus.data.c.points`.

**Links.** 97 link-bearing slots; 71 have `where` != -1; 71 fully linked (`who` != 255); 71 resolved to a live room; 0 dangling (of which 0 are `where` == -100 with `who` == 255, i.e. inert); 0 with `who` >= kMaxRoomObs.

**Connectivity.** East/West adjacent pairs: 107 two-way, 21 one-way, 63 walled. North/South adjacent pairs: 56 two-way, 29 one-way, 87 walled. Staircases: 7 up / 7 down (unpaired 0 / 0). Undirected components 20 (largest 181, the one holding `firstRoom` 181). Forward-reachable from `firstRoom` 165 of 208 rooms; BFS eccentricity 30.

**Stars.** 1, at BFS depths [25] of 30: room 139 "Sky" x1.

**Resource fork.** `ICN#` x2 (-16455, 128); `PICT` x48 (3300-3312, 3314-3327, 10000-10013, 10058-10064); `icl4` x1 (-16455); `icl8` x1 (-16455); `ics#` x1 (-16455); `ics4` x1 (-16455); `ics8` x1 (-16455); `snd ` x6 (3000, 3006, 3032, 3037, 3058, 3061); `vers` x1 (1).

Shadows application `PICT` IDs: 10000.

`kSoundTrigger` `data.e.where` values: 3000 x3, 3006 x11, 3037 x1, 3058 x1.

`kCustomPict` `data.g.height` (PICT ID) values: 94 placements over 21 distinct IDs, top: 10002 x11, 10004 x10, 10003 x9, 10009 x8, 10008 x6, 10000 x5, 10001 x5, 10061 x5. All resolve inside this fork.

**Notes.**

- 43 room(s) are not forward-reachable from `firstRoom` under the static model
- ships a saved game (`hasGame` = 1): `savedGame.roomNumber` = 104, score 4700, 2 gliders

## 5. Corpus statistics

All numbers in this section were produced by `tools/probe_houses_inventory.py` over
all 22 `.binhex` files and are reproduced in `docs/analysis/houses-inventory.md`.
Where a statistic is "per room" it is per *real* room (`suite != kRoomIsEmpty`,
`GliderPRO/Headers/GliderDefines.h:525`); there are **4070** real rooms and **zero**
placeholder rooms in the whole corpus, so `nRooms` and the real-room count coincide
for every shipped house.

### 5.1 Size distribution

Percentiles below use the order statistics `sorted[n/4]` and `sorted[3n/4]` with
`n = 22`, i.e. the 6th and 17th smallest value; medians are the mean of the 11th and
12th.

| statistic | min | p25 | median | p75 | max | mean | total |
|---|---|---|---|---|---|---|---|
| real rooms per house | 2 (Sampler) | 65 | 133.5 | 279 | 531 (Teddy World) | 185.0 | 4070 |
| occupied floors per house | 1 | 7 | 13.5 | 16 | 35 (SpacePods) | 13.2 | 291 |
| occupied suites per house | 2 | 14 | 19 | 39 | 101 (Teddy World) | 27.9 | 613 |
| live objects per house | 11 (Sampler) | 569 | 966 | 1814 | 5840 (SpacePods) | 1429.1 | 31440 |
| distinct `what` codes per house | 9 (Sampler) | 55 | 84.5 | 103 | 114 | 77.3 | 117 |
| data-fork bytes | 1564 | 23486 | 47324 | 97958 | 185654 | 65246.1 | 1435414 |
| resource-fork bytes | 286 | 316065 | 649643.5 | 1347764 | 7476159 | 1142821.5 | 25142074 |
| `.binhex` container bytes | 2169 | 513683 | 950645.5 | 1882081 | 10056249 | 1602012.0 | 35244264 |
| `kStar` per house | 0 (Fun House) | 1 | 3 | 5 | 9 (CD Demo House) | 3.1 | 69 |
| mean objects per room, per house | 2.23 | 6.37 | 7.18 | 7.83 | 19.25 | 7.62 | - |
| `CountTotalHousePoints` | 5200 | 30700 | 48850 | 77600 | 141400 | 53800 | 1183600 |

The data fork is a pure function of the room count: `dataBytes = 866 + 348 * nRooms`
holds exactly for 21 of 22 houses; Sampler is 2 bytes longer than the identity
predicts (see 2.7 and 11.5).

The resource fork dominates the file: the corpus is 1.4 MB of level data and
25.1 MB of art - a 17.5:1 ratio. Media weight per room ranges from
**76 bytes/room** (Empty House, 2670 bytes of resources over 35 rooms) and
143 bytes/room (Sampler) up to **68589 bytes/room** (Art Museum, 109 rooms and
7.5 MB of paintings), with Davis Station second at 28790 and California or Bust!
third at 16221.

### 5.2 Grid shape and density

Rooms live on a fixed lattice: suite (horizontal) `0..127` = `kMaxNumRoomsH` 128
(`GliderPRO/Headers/GliderDefines.h:543`), floor (vertical) `-7..56` = 64 cells =
`kMaxNumRoomsV` (`:544`). `ValidateRoomNumbers` (`GliderPRO/Sources/HouseLegal.c:805-816`)
enforces exactly those ranges. Density here means
`realRooms / (occupiedFloors x occupiedSuites)` - how solidly the author filled the
bounding box of their own layout.

| house | floors | suites | rooms | bounding cells | density |
|---|---|---|---|---|---|
| California or Bust! | 1 | 16 | 16 | 16 | 1.00 |
| The Asylum Pro | 14 | 10 | 140 | 140 | 1.00 |
| Sampler | 1 | 2 | 2 | 2 | 1.00 |
| Nemo's Market | 3 | 42 | 124 | 126 | 0.98 |
| Castle o' the Air | 10 | 13 | 85 | 130 | 0.65 |
| Empty House | 6 | 9 | 35 | 54 | 0.65 |
| Demo House | 7 | 10 | 45 | 70 | 0.64 |
| SpacePods | 35 | 18 | 402 | 630 | 0.64 |
| Art Museum | 11 | 16 | 109 | 176 | 0.62 |
| Fun House | 5 | 14 | 43 | 70 | 0.61 |
| Metropolis | 14 | 15 | 127 | 210 | 0.60 |
| Titanic | 13 | 28 | 208 | 364 | 0.57 |
| Grand Prix | 12 | 27 | 175 | 324 | 0.54 |
| Land of Illusion | 20 | 29 | 303 | 580 | 0.52 |
| ImagineHouse PRO II | 15 | 39 | 279 | 585 | 0.48 |
| In The Mirror | 12 | 17 | 97 | 204 | 0.48 |
| Rainbow's End | 14 | 36 | 223 | 504 | 0.44 |
| Slumberland | 18 | 48 | 383 | 864 | 0.44 |
| Leviathan | 26 | 58 | 472 | 1508 | 0.31 |
| CD Demo House | 16 | 45 | 206 | 720 | 0.29 |
| Davis Station | 14 | 20 | 65 | 280 | 0.23 |
| Teddy World | 24 | 101 | 531 | 2424 | 0.22 |

Two shapes dominate. **Slab houses** (density >= 0.55) are dense rectangular
blocks - The Asylum Pro is a perfect 14x10 tower with no holes at all, California
or Bust! is a single 16-room train of rooms on one floor. **Sprawl houses**
(density <= 0.35) are thin corridors wandering across a wide lattice: Teddy World
occupies 101 suites but averages only 5.3 rooms per suite, Davis Station spreads
65 rooms over 14 floors.

Suite/floor extents also encode where the author started. The editor's "new
house" default sits around suite 63-64, floor 1 (the majority of houses have
`firstRoom` at suite 54-72, floor 1), and authors then grew outward. Only five
houses escape the 46..95 suite band: SpacePods (110..127, hard against the right
edge of the lattice), Teddy World (1..113), Leviathan (0..94), Slumberland (0..86),
ImagineHouse PRO II (6..95). Floors run from `-7` (Rainbow's End, Slumberland,
Titanic all reach the legal floor floor) to `+56`? No house does. Three houses bottom out at the legal
minimum floor `-7` (Rainbow's End `-7..6`, Slumberland `-7..10`, Titanic `-7..5`),
but the highest floor used anywhere is **39** (SpacePods, `4..39`; Teddy World is
second at 26), so **floors 40..56 - 17 of the 64 legal rows - were never used by
anybody.**

### 5.3 Objects per room

`kMaxRoomObs` is 24 (`GliderPRO/Headers/GliderDefines.h:250`), so every room has
exactly 24 slots on disk and 97680 slots exist corpus-wide. 31440 are live
(**32.2 %** fill). Empty slots carry `what == kObjectIsEmpty` = `-1` (`:526`).

Full distribution over 4070 rooms (count of rooms with exactly N live objects):

| N | rooms | cum % | N | rooms | cum % |
|---|---|---|---|---|---|
| 0 | 840 | 20.6 | 13 | 116 | 76.9 |
| 1 | 588 | 35.1 | 14 | 114 | 79.7 |
| 2 | 132 | 38.3 | 15 | 127 | 82.8 |
| 3 | 150 | 42.0 | 16 | 85 | 84.9 |
| 4 | 124 | 45.1 | 17 | 81 | 86.9 |
| 5 | 114 | 47.9 | 18 | 59 | 88.3 |
| 6 | 140 | 51.3 | 19 | 50 | 89.6 |
| 7 | 155 | 55.1 | 20 | 39 | 90.6 |
| 8 | 164 | 59.1 | 21 | 49 | 91.8 |
| 9 | 154 | 62.9 | 22 | 53 | 93.1 |
| 10 | 147 | 66.5 | 23 | 60 | 94.5 |
| 11 | 146 | 70.1 | **24** | **222** | 100.0 |
| 12 | 161 | 74.1 | | | |

mean 7.72, median 6, p25 1, p75 13, p90 20.

Two spikes carry the shape. **840 rooms (20.6 %) hold nothing at all** - pure
transit space. **222 rooms (5.5 %) are jammed against the 24-slot ceiling**, which
is a strong signal that those authors wanted more and the format said no.
SpacePods alone contributes 77 of the 222 (19 % of its own rooms); Teddy World 35,
Land of Illusion 22, Rainbow's End 17, Nemo's Market 18.

**Single-object rooms are almost a genre of their own**: 588 rooms hold exactly
one object, and of those **434 hold nothing but a `kInvisLight` (0x58)**. That is
the mandatory-lighting idiom of 7.3.5. The rest of the single-object rooms:
`kInvisBounce` 57, `kCustomPict` 49, `kCloud` 32, `kInvisObstacle` 7, `kLiftArea` 3,
`kTrackLight` 2, `kWindowExLf` 2, `kDeluxeTrans` 1, `kDownStairs` 1.

### 5.4 Which object classes appear together

`what` codes are grouped into 9 classes by the 10-byte union arm they use
(`GliderPRO/Headers/GliderStructs.h:90-105`); the code ranges are contiguous
(`GliderDefines.h:311-435`). Presence = "at least one object of that class in the
room".

| class | union arm | code range | placements | share | rooms containing >=1 | share of rooms |
|---|---|---|---|---|---|---|
| blower | `a` `blowerType` | 0x01-0x10 | 6044 | 19.2 % | 1931 | 47.4 % |
| furniture | `b` `furnitureType` | 0x11-0x1F | 4695 | 14.9 % | 1644 | 40.4 % |
| prize | `c` `bonusType` | 0x21-0x2F | 2992 | 9.5 % | 1074 | 26.4 % |
| transport | `d` `transportType` | 0x31-0x40 | 1726 | 5.5 % | 998 | 24.5 % |
| switch | `e` `switchType` | 0x41-0x49 | 1685 | 5.4 % | 602 | 14.8 % |
| light | `f` `lightType` | 0x51-0x58 | 2530 | 8.0 % | 2410 | 59.2 % |
| appliance | `g` `applianceType` | 0x61-0x6E | 5552 | 17.7 % | 1225 | 30.1 % |
| enemy | `h` `enemyType` | 0x71-0x79 | 2077 | 6.6 % | 760 | 18.7 % |
| clutter | `i` `clutterType` | 0x81-0x8F | 4139 | 13.2 % | 1230 | 30.2 % |

287 distinct class signatures occur (out of 512 possible). Top 20:

| rank | signature | rooms | % |
|---|---|---|---|
| 1 | *(empty room)* | 840 | 20.6 |
| 2 | light | 436 | 10.7 |
| 3 | clutter | 204 | 5.0 |
| 4 | light + appliance | 194 | 4.8 |
| 5 | appliance | 87 | 2.1 |
| 6 | furniture + light + appliance | 78 | 1.9 |
| 7 | furniture | 66 | 1.6 |
| 8 | blower + light | 45 | 1.1 |
| 9 | blower + furniture + light + enemy | 43 | 1.1 |
| 10 | blower + furniture + prize + light | 42 | 1.0 |
| 11 | blower + furniture + prize + transport + light | 42 | 1.0 |
| 12 | blower + furniture + prize + switch + light + appliance | 41 | 1.0 |
| 13 | blower + furniture + prize + light + appliance | 40 | 1.0 |
| 14 | blower + furniture + light + appliance | 40 | 1.0 |
| 15 | all nine except enemy | 40 | 1.0 |
| 16 | blower + furniture + prize + transport + light + clutter | 39 | 1.0 |
| 17 | blower + light + enemy | 35 | 0.9 |
| 18 | blower + furniture + prize + transport + switch + light + appliance | 34 | 0.8 |
| 19 | blower + furniture + prize + light + appliance + clutter | 31 | 0.8 |
| 20 | blower + furniture + light | 30 | 0.7 |

The head of that list is not "designed rooms" - it is the four kinds of
*connective tissue*: empty room, lit-but-empty room, decorated room, and
lit + wallpapered room. Only from rank 9 onward do you get rooms with an actual
puzzle in them (blower + obstacle + prize + hazard).

### 5.5 Openings

`DetermineRoomOpenings` (`GliderPRO/Sources/Room.c:816-933`) resolves whether each
of the four walls is passable. Corpus totals over 4070 rooms:

| wall | open | share |
|---|---|---|
| left (west) | 2759 | 67.8 % |
| right (east) | 3015 | 74.1 % |
| top (north, no ceiling) | 2263 | 55.6 % |
| bottom (south, no floor) | 1972 | 48.5 % |

Horizontal passage is the norm, vertical passage is a coin flip. Right is open
more often than left (3015 vs 2759): the built-in default rule is
`leftOpen = (leftTile != 0)` and `rightOpen = (rightTile != 7)`
(`Room.c:918-919`), and tile 0 (a solid left wall) is far more commonly authored at
the left edge than tile 7 is at the right.

### 5.6 Object coordinate envelope

Point-based objects (union arms a, c, d, e, f, g, h - 22606 placements) store
`Point topLeft` = `{short v; short h}`, v first. Rect-based objects (arms b and i -
8834 placements) store `Rect bounds` = `{top, left, bottom, right}` plus a `short pict`.

| measure | observed |
|---|---|
| `topLeft.h` range | `-1 .. 509` |
| `topLeft.v` range | `0 .. 319` |
| `bounds.left` / `bounds.top` minimum | 0 / 0 |
| `bounds.right` / `bounds.bottom` maximum | 512 / 322 |

Those envelopes match the room canvas exactly: `kRoomWide` 512 =
`kNumTiles` 8 x `kTileWide` 64, `kTileHigh` 322
(`GliderPRO/Headers/GliderDefines.h:496-499`). **Exactly one negative coordinate
exists in the entire corpus**: Titanic room 161 "I Want to be a Narrator" slot 9,
a `kTV` (0x65) with `topLeft = (h = -1, v = 44)`, raw bytes
`0065 002c ffff 0000 0000 0101` (the leading `0065` is `what` = kTV, then
`002c` = v 44, `ffff` = h -1). A Go port must therefore keep object coordinates
**signed 16-bit** and must not assume non-negative placement.

Objects are *not* snapped to the 64-pixel tile grid: only 1703 of 22606 point
objects have `h % 64 == 0`. The next most common residues are 52 (677), 38 (616),
56 (566), 48 (514), 20 (508), 40 (500), 50 (493), 16 (474), 58 (462) - i.e. free
mouse placement, dragged by hand in the editor.

### 5.7 Room names

2140 distinct names over 4070 rooms; **zero** rooms have an empty name.
`CheckRoomNameLength` clamps `name[0]` to 27 (`GliderPRO/Sources/HouseLegal.c:870-875`),
matching the `Str27` field.

Top 20 reused names: `N` 255 (Teddy World 247 + CD Demo House 8), `Space` 156,
`Open` 116, `Black On Black` 61, `Sky` 60, `Darling` 57, `Untitled Room` 53,
`Underdog` 53, `Dirty` 52, `Skyish` 50, `Nemo's Roof` 46, `Rooftop` 45, `Kitten` 44,
`Murmur` 44, `Sub Nemo` 42, `Pod Side` 35, `n` 34, `Nitrogen` 33, `Black Space` 32,
`Stuff` 31.

Top 20 words (case-folded): the 310, n 289, space 202, black 156, open 120, on 117,
roof 112, a 110, pod 102, to 97, of 96, room 93, sky 93, out 76, up 68, side 62,
nemo's 61, darling 57, dirty 55, air 54.

Name-reuse ratio (`rooms / distinctNames`) separates the two workflows cleanly:

| house | rooms | distinct names | ratio |
|---|---|---|---|
| Empty House | 35 | 2 | **17.50** |
| Nemo's Market | 124 | 35 | 3.54 |
| Fun House | 43 | 15 | 2.87 |
| Land of Illusion | 303 | 111 | 2.73 |
| CD Demo House | 206 | 90 | 2.29 |
| SpacePods | 402 | 176 | 2.28 |
| Teddy World | 531 | 238 | 2.23 |
| Castle o' the Air | 85 | 39 | 2.18 |
| ImagineHouse PRO II | 279 | 151 | 1.85 |
| Titanic | 208 | 114 | 1.82 |
| Grand Prix | 175 | 97 | 1.80 |
| Rainbow's End | 223 | 126 | 1.77 |
| Leviathan | 472 | 280 | 1.69 |
| In The Mirror | 97 | 58 | 1.67 |
| Metropolis | 127 | 87 | 1.46 |
| Art Museum | 109 | 76 | 1.43 |
| Slumberland | 383 | 302 | 1.27 |
| The Asylum Pro | 140 | 119 | 1.18 |
| Davis Station | 65 | 60 | 1.08 |
| California or Bust! | 16 | 15 | 1.07 |
| Demo House | 45 | 44 | 1.02 |
| Sampler | 2 | 2 | 1.00 |

Empty House has exactly **two** distinct names across 35 rooms
(`Untitled Room` x34 and `Finish` x1), which is the format's degenerate case.

`Untitled Room` - the literal editor default that `CountUntitledRooms`
(`HouseLegal.c:834-846`) warns about - survives in 53 rooms: Empty House 34
(including its own `firstRoom`), CD Demo House 15, California or Bust! 2,
Fun House 2. So 4 of 22 shipped houses ship with the editor's blue warning
outstanding.

---

## 6. The object vocabulary the official houses stay within

### 6.1 There is no unused object

Every one of the **117** object codes defined in `GliderDefines.h:311-435` is used
by at least one shipped house, and **every code appears in at least 7 of the 22
houses**. There is no "editor-only" or "dead" object type in the format. Any port
that wants to load the shipped corpus must implement all 117.

### 6.2 Tiers by house-spread

| tier | definition | codes | placements | share of 31440 |
|---|---|---|---|---|
| core | used in >= 15 of 22 houses | 62 | 27519 | 87.5 % |
| common | used in 8-14 houses | 51 | 3345 | 10.6 % |
| rare | used in <= 7 houses | 4 | 576 | 1.8 % |

The rare tier is only four codes, and all four sit at exactly 7 houses:
`kCeilingVent` 0x02 (28 placements), `kGrecoVent` 0x0E (116), `kSlider` 0x2F (354),
`kPowerSwitch` 0x44 (78). `kSlider` is an outlier - 354 placements but concentrated
in 7 houses, i.e. a few authors leaned on it heavily.

The bottom of the common tier (8 houses) is `kFloorBlower` 0x03, `kCeilingBlower`
0x04, `kDoorInRt` 0x38, `kDoorExLf` 0x3A, `kGuitar` 0x64.

### 6.3 Coverage curve

| top N codes by placement | cumulative share |
|---|---|
| 1 (`kCustomPict`) | 15.2 % |
| 3 | 28.6 % |
| 5 | 39.5 % |
| 9 | 50.6 % |
| 15 | 60.7 % |
| 20 | 67.2 % |
| 25 | 71.9 % |
| 30 | 75.8 % |
| 37 | 80.1 % |
| 45 | 84.2 % |
| 50 | 86.4 % |
| 60 | 90.1 % |
| 70 | 93.1 % |
| 80 | 95.5 % |
| 90 | 97.4 % |
| 100 | 98.7 % |
| 117 | 100.0 % |

Half of every object ever placed in an official house is one of nine codes:
`kCustomPict` 0x6E (4782), `kInvisBlower` 0x0D (2336), `kCloud` 0x8C (1860),
`kInvisLight` 0x58 (1764), `kInvisBounce` 0x1F (1670), `kFloorVent` 0x01 (1458),
`kLiftArea` 0x10 (716), `kMirror` 0x82 (667), `kInvisObstacle` 0x1C (666).

**Seven of those nine are invisible.** The visible content of an official house is
`kCustomPict` (author art) plus `kCloud`; everything else in the top nine is
machinery the player feels but never sees. That is the single most important
design fact in this document: Glider PRO houses are built out of invisible
colliders, invisible updrafts, invisible lights and invisible bouncers laid over
hand-painted PICT backdrops.

### 6.4 Full corpus histogram, grouped by class

Format: `name` code count/houses.

**a - blower (`blowerType`, 16 codes, 6044 placements)**
`kFloorVent` 0x01 1458/19; `kCeilingVent` 0x02 28/7; `kFloorBlower` 0x03 83/8;
`kCeilingBlower` 0x04 12/8; `kSewerGrate` 0x05 507/16; `kLeftFan` 0x06 45/12;
`kRightFan` 0x07 54/14; `kTaper` 0x08 90/17; `kCandle` 0x09 180/18;
`kStubby` 0x0A 127/17; `kTiki` 0x0B 58/12; `kBBQ` 0x0C 43/11;
`kInvisBlower` 0x0D 2336/21; `kGrecoVent` 0x0E 116/7; `kSewerBlower` 0x0F 191/11;
`kLiftArea` 0x10 716/16.

**b - furniture / obstacle (`furnitureType`, 15 codes, 4695)**
`kTable` 0x11 170/13; `kShelf` 0x12 389/19; `kCabinet` 0x13 457/16;
`kFilingCabinet` 0x14 107/14; `kWasteBasket` 0x15 103/14; `kMilkCrate` 0x16 252/18;
`kCounter` 0x17 287/16; `kDresser` 0x18 122/17; `kDeckTable` 0x19 30/10;
`kStool` 0x1A 91/13; `kTrunk` 0x1B 101/16; `kInvisObstacle` 0x1C 666/18;
`kManhole` 0x1D 40/13; `kBooks` 0x1E 210/19; `kInvisBounce` 0x1F 1670/15.

**c - prize (`bonusType`, 15 codes, 2992)**
`kRedClock` 0x21 140/16; `kBlueClock` 0x22 163/18; `kYellowClock` 0x23 330/19;
`kCuckoo` 0x24 100/19; `kPaper` 0x25 283/18; `kBattery` 0x26 119/19;
`kBands` 0x27 150/19; `kGreaseRt` 0x28 195/17; `kGreaseLf` 0x29 143/17;
`kFoil` 0x2A 83/14; `kInvisBonus` 0x2B 327/18; `kStar` 0x2C 69/21;
`kSparkle` 0x2D 486/17; `kHelium` 0x2E 50/13; `kSlider` 0x2F 354/7.

**d - transport (`transportType`, 16 codes, 1726)**
`kUpStairs` 0x31 163/16; `kDownStairs` 0x32 163/16; `kMailboxLf` 0x33 44/10;
`kMailboxRt` 0x34 34/10; `kFloorTrans` 0x35 244/15; `kCeilingTrans` 0x36 465/18;
`kDoorInLf` 0x37 23/14; `kDoorInRt` 0x38 11/8; `kDoorExRt` 0x39 21/13;
`kDoorExLf` 0x3A 11/8; `kWindowInLf` 0x3B 21/12; `kWindowInRt` 0x3C 32/12;
`kWindowExRt` 0x3D 19/12; `kWindowExLf` 0x3E 29/12; `kInvisTrans` 0x3F 385/17;
`kDeluxeTrans` 0x40 61/14.

**e - switch (`switchType`, 9 codes, 1685)**
`kLightSwitch` 0x41 113/14; `kMachineSwitch` 0x42 79/13; `kThermostat` 0x43 108/15;
`kPowerSwitch` 0x44 78/7; `kKnifeSwitch` 0x45 230/15; `kInvisSwitch` 0x46 635/18;
`kTrigger` 0x47 239/13; `kLgTrigger` 0x48 81/11; `kSoundTrigger` 0x49 122/14.

**f - light (`lightType`, 8 codes, 2530)**
`kCeilingLight` 0x51 160/16; `kLightBulb` 0x52 252/15; `kTableLamp` 0x53 70/15;
`kHipLamp` 0x54 26/9; `kDecoLamp` 0x55 62/14; `kFlourescent` 0x56 115/12;
`kTrackLight` 0x57 81/12; `kInvisLight` 0x58 1764/20.

**g - appliance (`applianceType`, 14 codes, 5552)**
`kShredder` 0x61 50/13; `kToaster` 0x62 140/17; `kMacPlus` 0x63 74/14;
`kGuitar` 0x64 29/8; `kTV` 0x65 80/19; `kCoffee` 0x66 38/12; `kOutlet` 0x67 100/14;
`kVCR` 0x68 26/10; `kStereo` 0x69 36/10; `kMicrowave` 0x6A 56/17;
`kCinderBlock` 0x6B 43/9; `kFlowerBox` 0x6C 35/11; `kCDs` 0x6D 63/13;
`kCustomPict` 0x6E 4782/19.

**h - enemy (`enemyType`, 9 codes, 2077)**
`kBalloon` 0x71 500/18; `kCopterLf` 0x72 259/15; `kCopterRt` 0x73 155/16;
`kDartLf` 0x74 151/15; `kDartRt` 0x75 117/12; `kBall` 0x76 212/18;
`kDrip` 0x77 477/18; `kFish` 0x78 120/16; `kCobweb` 0x79 86/17.

**i - clutter (`clutterType`, 15 codes, 4139)**
`kOzma` 0x81 77/13; `kMirror` 0x82 667/20; `kMousehole` 0x83 174/12;
`kFireplace` 0x84 33/11; `kFlower` 0x85 547/17; `kWallWindow` 0x86 195/15;
`kBear` 0x87 169/19; `kCalendar` 0x88 58/15; `kVase1` 0x89 72/16;
`kVase2` 0x8A 86/16; `kBulletin` 0x8B 38/13; `kCloud` 0x8C 1860/19;
`kFaucet` 0x8D 28/11; `kRug` 0x8E 81/18; `kChimes` 0x8F 54/17.

### 6.5 Vocabulary breadth per house

Distinct `what` codes used, and what that says about the house:

| house | distinct codes | rooms | codes per 100 rooms |
|---|---|---|---|
| ImagineHouse PRO II | 114 | 279 | 40.9 |
| Leviathan | 114 | 472 | 24.2 |
| Teddy World | 112 | 531 | 21.1 |
| Rainbow's End | 110 | 223 | 49.3 |
| Slumberland | 105 | 383 | 27.4 |
| The Asylum Pro | 103 | 140 | 73.6 |
| CD Demo House | 100 | 206 | 48.5 |
| Grand Prix | 100 | 175 | 57.1 |
| In The Mirror | 97 | 97 | 100.0 |
| Titanic | 89 | 208 | 42.8 |
| Land of Illusion | 85 | 303 | 28.1 |
| Metropolis | 84 | 127 | 66.1 |
| Davis Station | 69 | 65 | 106.2 |
| Nemo's Market | 68 | 124 | 54.8 |
| SpacePods | 60 | 402 | 14.9 |
| Art Museum | 59 | 109 | 54.1 |
| California or Bust! | 55 | 16 | 343.8 |
| Fun House | 54 | 43 | 125.6 |
| Castle o' the Air | 50 | 85 | 58.8 |
| Demo House | 48 | 45 | 106.7 |
| Empty House | 15 | 35 | 42.9 |
| Sampler | 9 | 2 | 450.0 |

The big houses converge on ~110 of 117 codes. The interesting low outlier is
**SpacePods: 402 rooms with only 60 codes**. It is the most repetitive house in the
corpus by vocabulary and the densest by objects (14.5/room) - a very deliberate
"one mechanic, four hundred variations" design. Empty House at 15 codes is the
blank template: `kFloorVent`, `kCeilingLight`, `kDoorInLf`, `kUpStairs`,
`kDownStairs`, `kStar` and little else.

---

## 7. Room motifs

### 7.1 Background choice is the motif taxonomy

The `background` field (`short`, offset 34 in `roomType`) does more than pick
wallpaper: it drives lighting (`GetNumberOfLights`, `Room.c:970-1099`), whether the
room has a floor and ceiling (`DoesRoomHaveFloor` `Room.c:1138-1168`,
`DoesRoomHaveCeiling` `Room.c:1172-1205`), whether a shadow is drawn under the
glider (`IsShadowVisible` `Room.c:1103-1134`), and whether a 44-pixel floor-support
strip is painted between vertically adjacent rooms (`IsRoomAStructure`
`Room.c:763-812`, consumed only by `DrawFloorSupport` `RoomGraphics.c:257`). Picking
a background therefore *is* picking a room motif. I group the 18 built-ins plus the
two user ranges into 7 classes:

| class | background IDs | rooms | houses using | lit for free? | floor? | ceiling? |
|---|---|---|---|---|---|---|
| indoor | 2000-2008 (`kSimpleRoom`..`kLibrary`) | 604 | 16 | no | yes | yes |
| kDirt | 2011 | 421 | 16 | only if all 8 tiles == 0 | yes | yes |
| ground-outdoor | 2009 `kGarden`, 2012 `kMeadow`, 2013 `kField` | 136 | 16 | yes | yes | no |
| elevated | 2010 `kSkywalk`, 2014 `kRoof` | 209 | 14 | yes | 2010 yes / 2014 yes | 2010 yes / 2014 no |
| air/space | 2015 `kSky`, 2016 `kStratosphere`, 2017 `kStars` | 903 | 18 | yes | no | no |
| user-structure | 3000-3299 | 1096 | 17 | **no** | per `bounds` bit 3 | per `bounds` bit 1 |
| user-open | 3300-3799 | 701 | 14 | **no** | per `bounds` bit 3 | per `bounds` bit 1 |

(`kBasement` 2002 is indoor and is *not* in the self-lit list and *not* in the
`IsRoomAStructure` list - it is the one built-in interior that draws no floor
support.)

Per-house room mix:

| house | indoor | kDirt | ground | elevated | air/space | user-struct | user-open |
|---|---|---|---|---|---|---|---|
| Art Museum | 9 | 0 | 0 | 0 | 9 | 91 | 0 |
| CD Demo House | 27 | 9 | 11 | 12 | 35 | 94 | 18 |
| California or Bust! | 0 | 0 | 2 | 0 | 0 | 14 | 0 |
| Castle o' the Air | 0 | 16 | 0 | 0 | 14 | 24 | 31 |
| Davis Station | 4 | 6 | 0 | 0 | 8 | 47 | 0 |
| Demo House | 0 | 13 | 2 | 5 | 15 | 6 | 4 |
| Empty House | 14 | 5 | 5 | 4 | 7 | 0 | 0 |
| Fun House | 12 | 0 | 5 | 4 | 9 | 7 | 6 |
| Grand Prix | 39 | 35 | 8 | 21 | 36 | 20 | 16 |
| ImagineHouse PRO II | 71 | 64 | 15 | 20 | 78 | 9 | 22 |
| In The Mirror | 24 | 9 | 10 | 7 | 43 | 0 | 4 |
| Land of Illusion | 0 | 17 | 2 | 5 | 32 | 83 | 164 |
| Leviathan | 116 | 24 | 22 | 64 | 182 | 10 | 54 |
| Metropolis | 3 | 7 | 4 | 3 | 22 | 48 | 40 |
| Nemo's Market | 0 | 0 | 0 | 0 | 0 | 0 | 124 |
| Rainbow's End | 22 | 76 | 13 | 5 | 64 | 11 | 32 |
| Sampler | 1 | 0 | 1 | 0 | 0 | 0 | 0 |
| Slumberland | 126 | 96 | 16 | 24 | 97 | 11 | 13 |
| SpacePods | 0 | 0 | 0 | 0 | 0 | 402 | 0 |
| Teddy World | 80 | 25 | 12 | 9 | 195 | 210 | 0 |
| The Asylum Pro | 47 | 10 | 8 | 26 | 40 | 9 | 0 |
| Titanic | 9 | 9 | 0 | 0 | 17 | 0 | 173 |
| **corpus** | **604** | **421** | **136** | **209** | **903** | **1096** | **701** |

Built-in background usage, by ID (rooms / houses):

| ID | name | rooms | houses | ID | name | rooms | houses |
|---|---|---|---|---|---|---|---|
| 2000 | kSimpleRoom | 171 | 13 | 2009 | kGarden | 14 | 5 |
| 2001 | kPaneledRoom | 105 | 13 | 2010 | kSkywalk | 28 | 6 |
| 2002 | kBasement | 103 | 9 | 2011 | kDirt | 421 | 16 |
| 2003 | kChildsRoom | 49 | 12 | 2012 | kMeadow | 97 | 15 |
| 2004 | kAsianRoom | 40 | 10 | 2013 | kField | 25 | 9 |
| 2005 | kUnfinishedRoom | 54 | 9 | 2014 | kRoof | 181 | 13 |
| 2006 | kSwingersRoom | 49 | 8 | 2015 | kSky | **595** | 17 |
| 2007 | kBathroom | 23 | 8 | 2016 | kStratosphere | 62 | 7 |
| 2008 | kLibrary | 10 | 7 | 2017 | kStars | 246 | 11 |

`kSky` (2015) is the single most-used background in the corpus at 595 rooms. Most
used user-art IDs: 3302 (231 rooms / 12 houses), 3004 (202/10), 3301 (151/13),
3002 (97/13), 3003 (86/13), 3303 (81/10), 3009 (78/7), 3005 (67/8), 3000 (66/10),
3001 (65/13).

### 7.2 Object budget by motif

The single most transferable design number in this document: how many objects a
room of each background class actually gets.

| class | rooms | objects | mean | median | max | empty rooms | 24-object rooms | dominant classes | top 5 `what` |
|---|---|---|---|---|---|---|---|---|---|
| indoor | 604 | 7481 | **12.39** | 12 | 24 | 50 (8.3 %) | 26 | a 22 %, b 16 %, i 12 %, c 11 %, d 10 % | kFloorVent 858, kCeilingTrans 237, kCabinet 213, kCustomPict 200, kShelf 188 |
| kDirt | 421 | 1532 | **3.64** | 0 | 24 | 222 (52.7 %) | 4 | a 24 %, f 14 %, b 14 %, c 13 %, h 12 % | kInvisBlower 130, kSewerBlower 105, kInvisLight 105, kSewerGrate 92, kDrip 89 |
| ground-outdoor | 136 | 1052 | 7.74 | 7 | 24 | 20 (14.7 %) | 8 | i 35 %, a 17 %, g 12 %, d 9 %, c 8 % | kCloud 225, kFlower 124, kCustomPict 116, kSewerGrate 103, kInvisBounce 42 |
| elevated | 209 | 767 | 3.67 | 0 | 24 | 105 (50.2 %) | 2 | a 28 %, i 21 %, c 16 %, b 13 %, g 8 % | kInvisBlower 126, kCloud 82, kFlower 54, kFloorVent 45, kInvisBounce 31 |
| air/space | 903 | 2684 | **2.97** | 1 | 24 | 413 (45.7 %) | 19 | i 43 %, a 17 %, c 16 %, b 10 %, g 5 % | kCloud 1106, kInvisBlower 386, kSlider 264, kInvisBounce 243, kCustomPict 140 |
| user-structure | 1096 | 13077 | **11.93** | 11 | 24 | 30 (2.7 %) | **126** | g 30 %, a 17 %, b 15 %, f 8 %, i 8 % | kCustomPict 3648, kInvisBlower 1041, kInvisBounce 910, kInvisLight 872, kInvisObstacle 451 |
| user-open | 701 | 4847 | 6.91 | 3 | 24 | **0** | 37 | a 21 %, b 18 %, f 14 %, g 14 %, c 10 % | kInvisLight 644, kCustomPict 570, kInvisBlower 450, kInvisBounce 366, kCloud 233 |

And what fraction of each class's rooms contains something from a given class:

| class | with enemy | with prize | with transport | with switch | with light | empty |
|---|---|---|---|---|---|---|
| user-structure | 21.2 % | 33.1 % | 22.5 % | 23.4 % | **93.4 %** | 2.7 % |
| air/space | 4.0 % | 5.5 % | 5.6 % | 1.7 % | 0.9 % | 45.7 % |
| user-open | 16.5 % | 24.7 % | 20.8 % | 11.6 % | **98.9 %** | 0.0 % |
| indoor | **39.1 %** | **56.5 %** | **71.0 %** | 33.3 % | 80.0 % | 8.3 % |
| kDirt | 21.9 % | 15.4 % | 9.7 % | 6.7 % | 45.4 % | 52.7 % |
| elevated | 8.6 % | 26.3 % | 16.3 % | 3.8 % | 4.3 % | 50.2 % |
| ground-outdoor | 22.1 % | 19.9 % | 36.8 % | 8.8 % | 1.5 % | 14.7 % |

Read across: **indoor rooms are where the game happens** (71 % contain a
transport, 56 % a prize, 39 % an enemy), **air/space rooms are the negative space**
(46 % empty, 4 % enemies), and **user-art rooms are where the light objects go**
because the engine gives them nothing for free.

### 7.3 The recurring motifs, named

Nine motifs cover the great majority of the 4070 rooms. Each is defined by an
objectively checkable predicate so a generator can reproduce it.

#### 7.3.1 Empty transit room

Predicate: 0 live objects. **840 rooms (20.6 %).** Concentrated in air/space (413),
kDirt (222) and elevated (105). Function: let the glider cross distance and lose
or gain altitude. Almost always self-lit backgrounds, because an unlit empty room
would be a black screen.

#### 7.3.2 Pure air room

Predicate: >= 1 `kCloud` and no object outside {`kCloud`, `kInvisLight`,
`kInvisBounce`, `kInvisBlower`}. **293 rooms.** (557 rooms contain at least one
`kCloud` at all.) Function: sky filler with visual interest and an updraft or two.
This is the outdoor equivalent of 7.3.1.

#### 7.3.3 Furnished interior

Predicate: background in 2000-2008 or 3000-3299, and >= 1 furniture object.
Median 11-12 objects. The canonical recipe, from the observed top-5 lists:
one background + 1-3 `kFloorVent` updrafts + 2-5 furniture obstacles
(`kCabinet`, `kShelf`, `kCounter`, `kTable`, `kMilkCrate`) + 1 light + 1-2 clutter
decorations (`kRug`, `kVase1/2`, `kCalendar`, `kBulletin`, `kWallWindow`) +
0-2 prizes + 0-1 enemy.

#### 7.3.4 Vertical shaft

Predicate: room contains both `kFloorTrans` (0x35) and `kCeilingTrans` (0x36).
**107 rooms.** These are the elevator/duct rooms that move the glider between
floors without a staircase. Note the asymmetry in the corpus:
`kCeilingTrans` 465 vs `kFloorTrans` 244 - authors used ceiling ducts (fly up
through the ceiling) nearly twice as often as floor ducts.

#### 7.3.5 Mandatory-light room

Predicate: background >= 3000 and the room's only light source is `kInvisLight`.
This is why `kInvisLight` has 1764 placements and is present in 20 of 22 houses,
and why **434 rooms contain exactly one object and that object is a
`kInvisLight`**. `GetNumberOfLights` (`Room.c:1038-1067`) has *no case* for
`background >= kUserBackground`, so it falls into `default: count = 0`, and unless
the room contains a lamp, a door or a window, the room renders dark. Custom art
therefore *always* needs an explicit light object.

Measured consequence: 93.4 % of user-structure rooms and 98.9 % of user-open rooms
contain a light object, versus 0.9 % of air/space rooms.

#### 7.3.6 Staircase pair

Predicate: room contains `kUpStairs` (0x31) or `kDownStairs` (0x32). **284 rooms**,
163 up + 163 down placements. `CheckForStaircasePairs`
(`HouseLegal.c:961-1042`) requires that every up-stair has a matching down-stair
in the room above, and **all 22 shipped houses pass with zero unpaired
staircases**. The counts are exactly equal (163 = 163) corpus-wide, which is the
strongest single indicator that this check was actually run on every shipped
house. Staircase-heavy houses: Slumberland 33 up / 33 down, Leviathan 28/28,
The Asylum Pro 18/18, Teddy World 15/15, Land of Illusion 14/14.

#### 7.3.7 Reward room

Predicate: contains `kStar` (0x2C). 69 rooms across 21 houses. Star-room object
counts: mean 14.75, median 14, min 3, max 24 - noticeably richer than the corpus
median of 6. By background class: user-structure 29, user-open 15, indoor 8,
air/space 6, ground-outdoor 4, kDirt 4, elevated 3.

The recurring decoration is **`kSparkle` clusters**: Leviathan "The Eagle has
Landed" (bg 2017) has 16 `kSparkle`; Land of Illusion "Final Reward" (bg 3302) has
10 `kSparkle` + 6 `kLiftArea`; Slumberland's star rooms likewise. `kSparkle` has
486 placements over 17 houses and essentially exists to say "prize here".

The other recurring reward-room idiom is a **`kLiftArea` bowl**: 4-6 `kLiftArea`
objects arranged to hold the glider in place around the star (Art Museum
"Enlightenment" bg 2017: 4x `kLiftArea` + `kLgTrigger` + `kDeluxeTrans`;
Land of Illusion "Final Reward": 6x `kLiftArea`).

Extremes worth copying: Nemo's Market has four star rooms all named
"*N* Second Shopping Spree", all on background 3301, all at the 24-object ceiling
and all stuffed with clocks - a timed bonus room. CD Demo House "Ball Illusion"
(bg 3100, 24 objects) is 8x `kTrigger` + 4x `kWallWindow` + 4x `kBall` +
4x `kInvisSwitch` - a pure switch/hazard puzzle.

#### 7.3.8 Machine room

Predicate: >= 1 switch object and >= 1 appliance object. Switches appear in only
14.8 % of rooms, so this is the least common major motif, but it is the densest:
signature `blower + furniture + prize + switch + light + appliance` alone accounts
for 41 rooms. The switch vocabulary is dominated by `kInvisSwitch` 635 and
`kTrigger` 239 - again, invisible machinery.

#### 7.3.9 Hazard gauntlet

Predicate: >= 3 enemy objects. Enemy density corpus-wide is 0.51/room, and enemies
are present in only 18.7 % of rooms, so a 3-enemy room is deliberately hostile.
Enemy vocabulary is `kBalloon` 500, `kDrip` 477, `kCopterLf` 259, `kBall` 212,
`kCopterRt` 155, `kDartLf` 151, `kFish` 120, `kDartRt` 117, `kCobweb` 86.
`kDrip` and `kBalloon` are the workhorses; darts and copters are the "advanced"
threats.

### 7.4 The `firstRoom` motif

`firstRoom` (`houseType` offset 862) plus `initial` (offset 12, a `Point` used as
`{top, left}` of a 48x20 spawn rect - `kGliderWide` 48, `kGliderHigh` 20,
`GliderDefines.h:548-549`) is where every play session begins.

| house | `firstRoom` | room name | bg | class | live objs | contents |
|---|---|---|---|---|---|---|
| Art Museum | 91 | Art Museum | 2002 | indoor | 1 | kDeluxeTrans x1 |
| CD Demo House | 70 | Welcome... | 2015 | air/space | 10 | kCloud x6, kInvisLight, kTV, kCustomPict, kInvisBlower |
| California or Bust! | 14 | Leaving the Heartland | 3000 | user-struct | 18 | kFlower x5, kCustomPict x4, kInvisBlower x3, kCloud x2, kLiftArea x2, kInvisLight, kSoundTrigger |
| Castle o' the Air | 33 | West Courtyard | 3302 | user-open | 12 | kCloud x2, kCopterLf x2, kBalloon x2, kInvisLight, kSewerGrate, kInvisBounce, kFloorTrans, kCopterRt, kPaper |
| Davis Station | 4 | Let's Roll | 3207 | user-struct | 24 | kFlower x13, kCloud x5, kInvisLight, kSewerGrate, kInvisBounce, kSoundTrigger, kLiftArea, kMailboxRt |
| Demo House | 0 | Air Vents | 3000 | user-struct | 10 | kFloorVent x3, kDresser, kRedClock, kWallWindow, kVase2, kFlower, kCeilingLight, kCustomPict |
| Empty House | 0 | Untitled Room | 2000 | indoor | 4 | kFloorVent x2, kCeilingLight, kDoorInLf |
| Fun House | 29 | Mission Impossible? | 2004 | indoor | 24 | kInvisTrans x19, kCeilingLight, kFloorVent, kCeilingTrans, kDeluxeTrans, kCabinet |
| Grand Prix | 127 | Fast! | 2012 | ground | 3 | kInvisLight, kSewerGrate, kInvisBounce |
| ImagineHouse PRO II | 1 | Falling from the Sky | 2015 | air/space | 9 | kCloud x6, kInvisBounce x2, kInvisLight |
| In The Mirror | 6 | Let's Begin | 3301 | user-open | 21 | kFoil x8, kInvisSwitch x8, kInvisLight, kCabinet, kMirror, kSoundTrigger, kCustomPict |
| Land of Illusion | 43 | Magic Mirror | 2017 | air/space | 24 | kInvisBlower x13, kMirror x6, kLiftArea x3, kInvisTrans x2 |
| Leviathan | 39 | Black | 3300 | user-open | **1** | kInvisLight x1 |
| Metropolis | 8 | Courtesy Desk | 3000 | user-struct | 10 | kInvisBlower x2, kCounter, kDecoLamp, kWasteBasket, kBulletin, kMacPlus, kRightFan, kCustomPict, kBands |
| Nemo's Market | 0 | Welcome to Nemo's Market! | 3303 | user-open | 5 | kInvisLight, kWindowExRt, kSewerGrate, kDoorExRt, kLiftArea |
| Rainbow's End | 30 | Out Of Thin Air | 2016 | air/space | 17 | kCloud x9, kInvisBlower x6, kLiftArea, kInvisTrans |
| Sampler | 1 | Welcome | 2012 | ground | 4 | kDoorExRt, kBBQ, kMailboxLf, kStar |
| Slumberland | 126 | Welcome... | 3300 | user-open | 16 | kFlower x7, kSparkle x4, kDoorExRt, kSewerGrate, kInvisLight, kInvisBounce, kCloud |
| SpacePods | 259 | Mission Statement | 3008 | user-struct | 24 | kCustomPict x10, kInvisBlower x8, kInvisSwitch x2, kInvisLight, kInvisTrans, kDartLf, kSoundTrigger |
| Teddy World | 0 | Wanna come in and play? | 2012 | ground | 24 | kFlower x10, kBear x9, kInvisTrans x2, kDoorExRt, kBBQ, kChimes |
| The Asylum Pro | 20 | Greetings! | 2012 | ground | 12 | kPaper x2, kDoorExRt, kSewerGrate, kDeckTable, kTaper, kGreaseRt, kBattery, kInvisTrans, kDeluxeTrans, kCustomPict, kInvisBounce |
| Titanic | 92 | Chute! | 3315 | user-open | 16 | kLiftArea x4, kCustomPict x4, kInvisBounce x3, kInvisLight, kInvisBonus, kCopterLf, kInvisBlower, kSoundTrigger |

Statistics: mean 13.14 objects, median 12, min 1, max 24. Background classes:
user-open 6, user-structure 5, air/space 4, ground-outdoor 4, indoor 3 - notably
**zero** first rooms use kDirt or elevated.

Design conclusions:
1. `firstRoom` is *richer* than the median room (13.1 vs 7.7 objects). Authors put
   their best foot forward.
2. `firstRoom` is almost never a hazard room. Only Castle o' the Air (2 copters,
   2 balloons) and SpacePods (1 dart) place enemies in room 0.
3. `firstRoom` index is not 0 in 16 of 22 houses - `GetFirstRoomNumber`
   (`GliderPRO/Sources/House.c:196-217`) exists precisely because the author moved
   the start after building.
4. **Five houses open on a `kSoundTrigger`** - California or Bust!, Davis Station,
   In The Mirror, SpacePods, Titanic - so the first thing the player hears is a
   house-supplied `snd ` resource.
5. Where the author wanted the arrival to feel like a fall, the first room is
   air/space with clouds (CD Demo House, ImagineHouse PRO II "Falling from the
   Sky", Rainbow's End "Out Of Thin Air") and no floor.

### 7.5 Naming motifs

Three naming strategies, matching the reuse ratios of 5.7:

1. **Narrative / joke names, one per room** - Slumberland 1.27, The Asylum Pro
   1.18, Davis Station 1.08, California or Bust! 1.07, Demo House 1.02,
   Sampler 1.00. Names double as in-game guidance ("Let's Roll", "Greetings!",
   "Mission Impossible?"). Slumberland is the striking case: 302 distinct names
   for 383 rooms, hand-written by four different authors.
2. **Themed families** - Nemo's Market (`Nemo's Roof` x46, `Sub Nemo` x42),
   SpacePods (`Pod Side` x35, `Space` families), Castle o' the Air. Names identify
   which *region* a room is in rather than the room itself.
3. **Deliberately degenerate** - Teddy World named 247 of its 531 rooms just `N`.
   Combined with 320 rooms whose `bounds` is 0, this reads as a build where the
   author stopped naming rooms after the first hundred.

Room names are player-visible only in the editor's room-info dialog and in the
saved-game/high-score "rooms visited" bookkeeping, so degenerate naming has no
gameplay consequence.

---

## 8. Difficulty progression

### 8.1 What the format lets you measure

There is no difficulty field anywhere in `houseType` or `roomType`. Difficulty is
entirely emergent from (a) how far a room is from `firstRoom`, (b) how many
obstacles/enemies are in it, (c) how much energy/consumable supply is nearby, and
(d) whether the room is dark. All four are measurable, so this section quantifies
them.

The distance metric used throughout is **BFS hop count from `firstRoom` over the
static room graph**, where an edge exists if:
- the two rooms are horizontally adjacent (same floor, suite +-1) and the shared
  wall is open in the source room per `DetermineRoomOpenings`; or
- the two rooms are vertically adjacent (same suite, floor +-1) and the source's
  ceiling/floor is absent per `DoesRoomHaveCeiling` / `DoesRoomHaveFloor`; or
- an object in the source carries a resolved link (`where` decoding to a live
  room) - staircases, ducts, doors, mailboxes, `kInvisTrans`, `kDeluxeTrans`.

**Caveats a porter must keep in mind.** This static graph over-connects (a
`kCeilingTrans` you cannot physically reach still counts) and under-connects (it
ignores whether the glider has the energy or altitude to make a crossing, and it
ignores switch-gated transport whose `where` is set at runtime). Treat the hop
numbers as an ordering, not as a walkthrough.

### 8.2 The corpus difficulty ramp

Rooms bucketed by their BFS depth expressed as a fraction of their own house's
eccentricity, then aggregated across all 22 houses:

| depth decile | rooms | objects | obj/room | enemies | enemies/room | prizes | prizes/room | stars |
|---|---|---|---|---|---|---|---|---|
| 0-10 % | 185 | 1730 | 9.35 | 69 | 0.373 | 156 | 0.843 | 4 |
| 10-20 % | 331 | 3047 | 9.21 | 224 | 0.677 | 307 | 0.927 | 2 |
| 20-30 % | 454 | 4722 | 10.40 | 285 | 0.628 | 557 | 1.227 | 9 |
| 30-40 % | 402 | 3994 | 9.94 | 234 | 0.582 | 468 | 1.164 | 6 |
| 40-50 % | 446 | 4020 | 9.01 | 269 | 0.603 | 274 | 0.614 | 7 |
| 50-60 % | 445 | 3689 | 8.29 | 275 | 0.618 | 353 | 0.793 | 10 |
| 60-70 % | 395 | 3163 | 8.01 | 314 | **0.795** | 330 | 0.835 | 6 |
| 70-80 % | 358 | 2991 | 8.35 | 177 | 0.494 | 202 | 0.564 | 8 |
| 80-90 % | 293 | 2288 | 7.81 | 156 | 0.532 | 225 | 0.768 | 8 |
| 90-100 % | 163 | 816 | **5.01** | 42 | 0.258 | 98 | 0.601 | 9 |

Three real signals here, and one non-signal:

1. **Object density falls with depth**, from ~9.4/room near the start to 5.0/room
   at the far edge. The far reaches of a house are emptier, not busier. This is
   the opposite of the naive expectation and is the most useful single finding for
   new-level design: authors front-loaded detail.
2. **Enemy density peaks in the 60-70 % band** (0.795/room) - a late-mid-game
   hazard spike - then drops sharply in the last 30 %.
3. **Prize density peaks in the 20-40 % band** (1.16-1.23/room) - the early-mid
   game is where you stock up on clocks and bands - and roughly halves after the
   halfway point.
4. **Stars are essentially flat across depth** (4, 2, 9, 6, 7, 10, 6, 8, 8, 9).
   Glider PRO houses do *not* put the goal at the end of a corridor; they scatter
   stars so that the player is exploring in several directions at once.

The picture is: dense, prize-rich, low-threat opening; a hazard ridge at roughly
two-thirds depth; and a thin, sparse, long tail. Difficulty in these houses comes
from *distance and navigation*, not from escalating object counts.

### 8.3 Per-house difficulty profile

| house | rooms | obj/room | enemies/room | prizes/room | dark % | stars | eccentricity | total points |
|---|---|---|---|---|---|---|---|---|
| Teddy World | 531 | 6.63 | 0.493 | 1.104 | 10.7 | 1 | 44 | 80500 |
| Leviathan | 472 | 7.07 | 0.729 | 0.820 | 2.1 | 6 | **59** | 118100 |
| SpacePods | 402 | **14.53** | 0.256 | 0.425 | 0.0 | 1 | 24 | 73000 |
| Slumberland | 383 | 7.82 | 0.585 | 1.008 | 7.6 | 6 | 57 | **141400** |
| Land of Illusion | 303 | 5.99 | 0.205 | 0.495 | 3.6 | 5 | 26 | 77700 |
| ImagineHouse PRO II | 279 | 6.50 | 0.584 | 0.642 | 0.4 | 3 | 46 | 68500 |
| Rainbow's End | 223 | 7.50 | 0.274 | 0.726 | 0.4 | 5 | 24 | 77500 |
| Titanic | 208 | 6.61 | 0.596 | 0.817 | 9.1 | 1 | 30 | 86300 |
| CD Demo House | 206 | 7.44 | 0.364 | 0.374 | **18.0** | **9** | 11 | 77600 |
| Grand Prix | 175 | 7.32 | 0.754 | 0.497 | 0.6 | 3 | 35 | 44600 |
| The Asylum Pro | 140 | 7.83 | 0.343 | 0.807 | 3.6 | 1 | 21 | 38000 |
| Metropolis | 127 | 6.37 | **1.016** | 0.496 | 3.1 | 4 | 39 | 39300 |
| Nemo's Market | 124 | 6.74 | 0.331 | 0.605 | 0.0 | 5 | 39 | 52200 |
| Art Museum | 109 | 5.22 | 0.422 | 0.688 | 8.3 | 6 | 36 | 58900 |
| In The Mirror | 97 | 8.20 | 0.763 | **1.186** | 0.0 | 1 | 35 | 30700 |
| Castle o' the Air | 85 | 7.29 | 0.918 | **1.412** | 3.5 | 4 | 10 | 45500 |
| Davis Station | 65 | 9.09 | 0.985 | 0.446 | 0.0 | 4 | 23 | 31800 |
| Demo House | 45 | 3.07 | **0.089** | 0.289 | 0.0 | 1 | 15 | 12500 |
| Fun House | 43 | 9.44 | 0.535 | 0.395 | 0.0 | **0** | **2** | 7900 |
| Empty House | 35 | 2.23 | **0.000** | 0.029 | 0.0 | 1 | 11 | 8500 |
| California or Bust! | 16 | **19.25** | **1.250** | 0.938 | 0.0 | 1 | 14 | 7900 |
| Sampler | 2 | 5.50 | 0.000 | 0.500 | 0.0 | 1 | 1 | 5200 |

Grouping by intent:

| tier | rooms | houses | signature |
|---|---|---|---|
| template / tech demo | 2-45 | Sampler (2), Empty House (35), Demo House (45) | near-zero enemies (0.00-0.09/room), 1 star, low object density (2.2-3.1) |
| showcase / short | 16-65 | California or Bust! (16), Fun House (43), Davis Station (65) | very high object density (9.1-19.3), high enemy density (0.5-1.3), low eccentricity |
| medium campaign | 85-140 | Castle o' the Air, In The Mirror, Art Museum, Nemo's Market, Metropolis, The Asylum Pro | 6.4-8.2 obj/room, 1-6 stars, eccentricity 10-39 |
| large campaign | 175-303 | Grand Prix, CD Demo House, Titanic, Rainbow's End, ImagineHouse PRO II, Land of Illusion | 6.0-7.5 obj/room, 3-5 stars, eccentricity 24-46 |
| epic | 383-531 | Slumberland, SpacePods, Leviathan, Teddy World | 6.6-14.5 obj/room, eccentricity 24-59, 1-6 stars |

Note that room count and difficulty are only loosely coupled. **Demo House is the
tutorial by design**: 0.089 enemies/room (4 enemies in 45 rooms), 3.07 objects/room,
one star at BFS depth 10 of 15, and a banner that says so verbatim
(`"Welcome to the Demo House!\rThis is a small beginner house that acts as a sort
of tutorial.\r(house by Kim Money)"`). Its trailer then explicitly points at the
top of the difficulty ladder: `"...The house \"Slumberland\" has over 400 rooms!"`.

The hardest houses by measurable threat are not the biggest: **Metropolis** has
1.016 enemies/room over 127 rooms ("Home of a thousand Drips, Darts, and Ducts!"
per its trailer) and **California or Bust!** is a 16-room pressure cooker at
19.25 objects/room and 1.25 enemies/room.

### 8.4 Darkness as a difficulty axis

**187 of 4070 rooms (4.6 %)** evaluate to `GetNumberOfLights() == 0` in *play* mode
(`data.f.state`) and render dark. The editor's variant (`data.f.initial`) finds
**197 (4.8 %)** — 10 rooms ship with a lamp switched off at load that a `kLightSwitch`
turns on. Distribution is wildly uneven: **Teddy World 57 dark rooms (10.7 % of
itself)**, CD Demo House 37 (18.0 %), Slumberland 29 (7.6 %), Titanic 19 (9.1 %),
Land of Illusion 11, Leviathan 10, Art Museum 9, The Asylum Pro 5, Metropolis 4,
Castle o' the Air 3, Grand Prix 1, ImagineHouse PRO II 1, Rainbow's End 1; and
**zero** in California or Bust!, Davis Station, Demo House, Empty House, Fun House,
In The Mirror, Nemo's Market, Sampler, SpacePods.

> [verified by re-implementing `GetNumberOfLights` literally over all 4070 rooms:
> the self-lit background set of eight, the `kDirt` all-tiles-zero rule, the
> `if (count == 0)` guard, all 24 object slots, doors/windows unconditional, the
> eight lamp types gated on `data.f.state` (play) or `data.f.initial` (editor) read
> from `objectType` arm bytes 9 and 8. Lamp bytes in the corpus are clean booleans:
> 2431 lamps `(initial,state) = (1,1)`, 89 `(0,0)`, 10 `(0,1)`.]

Darkness is thus a signature choice, not a gradient: some authors used it as a
core mechanic (CD Demo House at 18 %, Teddy World at 11 %) and others banned it
outright — nine of the 22 houses contain no dark room at all.

### 8.5 The consumable economy

Points are computed by `CountTotalHousePoints` (`GliderPRO/Sources/HouseInfo.c:51-104`):
`RealRoomNumberCount() * kRoomVisitScore` (100) plus `kRedClock` 100,
`kBlueClock` 300, `kYellowClock` 500, `kCuckoo` 1000, `kStar` 5000, and each
`kInvisBonus`'s own `data.c.points`.

| house | prizes | clocks | battery | bands | foil | helium | enemies | prize:enemy | hazard appliances | open flames | total points | from kInvisBonus |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Art Museum | 75 | 18 | 1 | 4 | 4 | 1 | 46 | 1.6 | 9 | 19 | 58900 | 9000 |
| CD Demo House | 77 | 18 | 1 | 4 | 0 | 2 | 75 | 1.0 | 6 | 12 | 77600 | 4000 |
| California or Bust! | 15 | 3 | 1 | 0 | 1 | 0 | 20 | 0.8 | 5 | 13 | 7900 | 400 |
| Castle o' the Air | 120 | 5 | 1 | 3 | 1 | 0 | 78 | 1.5 | 0 | 18 | 45500 | 12000 |
| Davis Station | 29 | 9 | 1 | 2 | 0 | 1 | 64 | 0.5 | 0 | 4 | 31800 | 0 |
| Demo House | 13 | 5 | 1 | 1 | 0 | 0 | 4 | 3.2 | 0 | 3 | 12500 | 800 |
| Empty House | 1 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 0 | 8500 | 0 |
| Fun House | 17 | 7 | 1 | 1 | 2 | 0 | 23 | 0.7 | 17 | 5 | 7900 | 0 |
| Grand Prix | 87 | 19 | 4 | 6 | 0 | 3 | 132 | 0.7 | 22 | 21 | 44600 | 4800 |
| ImagineHouse PRO II | 179 | 43 | 15 | 10 | 6 | 5 | 163 | 1.1 | 19 | 23 | 68500 | 6000 |
| In The Mirror | 115 | 31 | 3 | 6 | **37** | 5 | 74 | 1.6 | 5 | 10 | 30700 | 3600 |
| Land of Illusion | 150 | 42 | 0 | 1 | 0 | 14 | 62 | 2.4 | 9 | 21 | 77700 | 3200 |
| Leviathan | 387 | 70 | 27 | 32 | 8 | 6 | 344 | 1.1 | 23 | 69 | 118100 | 10700 |
| Metropolis | 63 | 13 | 8 | 6 | 1 | 4 | 129 | 0.5 | 9 | 7 | 39300 | 2000 |
| Nemo's Market | 75 | 40 | 1 | 1 | 2 | 1 | 41 | 1.8 | 6 | 1 | 52200 | 2100 |
| Rainbow's End | 162 | 71 | 2 | 6 | 2 | 2 | 61 | 2.7 | 13 | 58 | 77500 | 2500 |
| Sampler | 1 | 0 | 0 | 0 | 0 | 0 | 0 | n/a | 0 | 1 | 5200 | 0 |
| Slumberland | 386 | 134 | 25 | 21 | 8 | 0 | 224 | 1.7 | 23 | 84 | 141400 | 15000 |
| SpacePods | 171 | 56 | 2 | 8 | 0 | 4 | 103 | 1.7 | 14 | 6 | 73000 | 3300 |
| Teddy World | 586 | 42 | 15 | 27 | 7 | 0 | 262 | 2.2 | 42 | 80 | 80500 | 3900 |
| The Asylum Pro | 113 | 30 | 6 | 4 | 1 | 0 | 48 | 2.4 | 12 | 24 | 38000 | 6100 |
| Titanic | 170 | 77 | 4 | 7 | 3 | 2 | 124 | 1.4 | 12 | 19 | 86300 | 14300 |
| **corpus** | **2992** | **733** | **119** | **150** | **83** | **50** | **2077** | **1.44** | **246** | **498** | **1183600** | **103700** |

("clocks" = `kRedClock` 140 + `kBlueClock` 163 + `kYellowClock` 330 + `kCuckoo` 100
= 733. "hazard appliances" = `kShredder` 0x61 50 + `kToaster` 0x62 140 +
`kMicrowave` 0x6A 56 = 246. "open flames" = `kTaper` 0x08 90 + `kCandle` 0x09 180 +
`kStubby` 0x0A 127 + `kTiki` 0x0B 58 + `kBBQ` 0x0C 43 = 498; these are
simultaneously blowers, i.e. updrafts, and hazards.)

Design rules of thumb visible here:
- **Prize:enemy ratio clusters at 1.0-1.8** (median 1.44). Houses below 0.8
  (Davis Station 0.5, Metropolis 0.5, Grand Prix 0.7, Fun House 0.7,
  California or Bust! 0.8) are the punishing ones.
- **Batteries and rubber bands are scarce.** 119 batteries and 150 bands over
  4070 rooms - one battery per 34 rooms. `Play.c:18` starts you with
  `kInitialGliders` 2 lives, so these are the whole safety margin. Land of
  Illusion ships **zero** batteries and one rubber band across 303 rooms.
- **Aluminum foil is a per-house gimmick, not a staple.** 83 placements corpus-wide,
  37 of them in In The Mirror alone (its `firstRoom` hands you 8 immediately).
- **Points scale sublinearly with rooms** because room-visit score is only 100 and
  stars are 5000: Slumberland's 141400 = 38300 from 383 room visits + 30000 from
  6 stars + 15000 from `kInvisBonus` + 58100 from clocks and other prizes.
- Corpus average is **290.8 points per room**.

### 8.6 Star placement vs depth

Stars per house and their BFS depth (as `[depths] / eccentricity`):

| house | stars | depths | ecc | mean depth fraction |
|---|---|---|---|---|
| Art Museum | 6 | 3,14,18,22,32,34 | 36 | 0.57 |
| CD Demo House | 9 | 3,4,6,6,6,6,6,7,8 | 11 | 0.53 |
| California or Bust! | 1 | 13 | 14 | 0.93 |
| Castle o' the Air | 4 | 1,3,4,6 | 10 | 0.35 |
| Davis Station | 4 | 13,17,19,23 | 23 | 0.78 |
| Demo House | 1 | 10 | 15 | 0.67 |
| Empty House | 1 | 11 | 11 | 1.00 |
| Fun House | 0 | - | 2 | - |
| Grand Prix | 3 | 17,22,33 | 35 | 0.69 |
| ImagineHouse PRO II | 3 | 21,36,38 | 46 | 0.69 |
| In The Mirror | 1 | 32 | 35 | 0.91 |
| Land of Illusion | 5 | 10,12,15,16,25 | 26 | 0.60 |
| Leviathan | 6 | 14,18,44,45,48,58 | 59 | 0.64 |
| Metropolis | 4 | 10,22,32,36 | 39 | 0.64 |
| Nemo's Market | 5 | 6,11,17,23,30 | 39 | 0.45 |
| Rainbow's End | 5 | 6,9,11,12,19 | 24 | 0.47 |
| Sampler | 1 | 0 | 1 | 0.00 |
| Slumberland | 6 | 12,15,35,45,48,49 | 57 | 0.60 |
| SpacePods | 1 | 7 | 24 | 0.29 |
| Teddy World | 1 | 6 | 44 | 0.14 |
| The Asylum Pro | 1 | 1 | 21 | 0.05 |
| Titanic | 1 | 25 | 30 | 0.83 |

`CountStarsInHouse` (`GliderPRO/Sources/Banner.c:89-111`) totals them and
`numStarsRemaining` is initialised from it (`Play.c:314`); collecting the last star
ends the house. So the star set is the win condition, and the placement policy
splits into two philosophies:

- **Single terminal star** (California or Bust! 0.93, In The Mirror 0.91,
  Titanic 0.83, Empty House 1.00, Demo House 0.67): the star is the finish line.
- **Star as an early reward** (The Asylum Pro 0.05, Teddy World 0.14,
  SpacePods 0.29, Castle o' the Air 0.35): with only one or a few stars placed
  near the start, the house is effectively an exploration sandbox that you can end
  whenever you like. The Asylum Pro's single star sits at BFS depth 1 of 21 - you
  can finish it in two rooms.
- **Multi-star campaigns** (Art Museum, CD Demo House, Land of Illusion, Leviathan,
  Slumberland, Rainbow's End, Nemo's Market, Metropolis, Davis Station): 4-9 stars
  spread over 35-78 % of the depth range, so the house is a collect-them-all.

For designing new levels, the corpus rate is **one star per 59 rooms** and
the mean star depth fraction is **0.57**.

---

## 9. What the editor enforces: legality rules a new house must satisfy

`CheckHouseForProblems` (`GliderPRO/Sources/HouseLegal.c:1052-1214`) is the gate a
house passes through on every editor save (`WriteHouse(true)`,
`GliderPRO/Sources/HouseIO.c:469-470`). A new house authored outside the original
editor should satisfy the same rules or the original app will silently rewrite it.
In call order:

| # | check | function | citation | measured compliance of the 22 |
|---|---|---|---|---|
| 1 | banner wrapped to 40 columns, trailer to 64 | `WrapBannerAndTrailer` | `HouseLegal.c:604-618` (`WrapText(banner, 40)` :610, `WrapText(trailer, 64)` :611) | all conform; embedded `\r` at wrap points |
| 2 | `nRooms == (handleSize - sizeof(houseType)) / sizeof(roomType)` | `ValidateNumberOfRooms` | `HouseLegal.c:621-644` (:629-631) | 22/22 pass; Sampler has 2 slack bytes that integer division absorbs |
| 3 | no two rooms share a (floor, suite) cell | `CheckDuplicateFloorSuite` | `HouseLegal.c:646-686`; `char[8192]` bitmap, `bitPlace = ((floor + 7) * 128) + suite` (:663-664) | 22/22 pass, no duplicates found |
| 4 | placeholder rooms squeezed out | `CompressHouse` | `HouseLegal.c:688-738` | 22/22 already compressed - **zero** placeholder rooms corpus-wide |
| 5 | handle truncated to `sizeof(houseType) + sizeof(roomType) * nRooms` | `LopOffExtraRooms` | `HouseLegal.c:740-778` | 21/22 exact; Sampler +2 |
| 6 | `floor` in `[-7, 56]`, `suite` in `[0, 127]` | `ValidateRoomNumbers` | `HouseLegal.c:785-816` | 22/22 pass (observed floors -7..39, suites 0..127) |
| 7 | count rooms still named "Untitled Room" -> blue warning | `CountUntitledRooms` | `HouseLegal.c:834-846` | 4 houses fail: Empty House 34, CD Demo House 15, California or Bust! 2, Fun House 2 |
| 8 | `name[0] <= 27`, `unusedByte = 0` | `CheckRoomNameLength` | `HouseLegal.c:857-877` | 22/22 pass; `unusedByte == 0` in all 4070 rooms |
| 9 | `numObjects` equals the live-slot count | `MakeSureNumObjectsJives` | `HouseLegal.c:885-917` | 22/22 pass, all 4070 rooms exact |
| 10 | every object's fields clamped to their legal ranges | `KeepAllObjectsLegal` -> `KeepObjectLegal` | `HouseLegal.c:919-959`, `:42` | see 5.6 - one `topLeft.h == -1` survives (Titanic), so the clamp does not forbid negative h |
| 11 | every up-stair has a down-stair above it and vice versa | `CheckForStaircasePairs` | `HouseLegal.c:961-1042` | 22/22 pass, 0 unpaired, and 163 up == 163 down corpus-wide |
| 12 | `CountStarsInHouse() >= 1` else red warning | inline | `HouseLegal.c:1203`, message `GetLocalizedString(35)` | **1 house fails: Fun House (0 stars)** |

Two subtleties worth transcribing exactly:

- The floor bias is **inconsistent between the two subsystems**.
  `CheckDuplicateFloorSuite` biases by `+7` (`HouseLegal.c:665-666`) and
  `ValidateRoomNumbers` allows `-7..56`, but the link encoding biases by
  `kNumUndergroundFloors` = **8** (`GliderDefines.h:535`, applied at
  `GliderPRO/Sources/Link.c:270`). So a legal room's encoded floor digit runs
  `1..64` and the observed "no room" sentinel `where == -100` decodes as
  `MergeFloorSuite(0, kRoomIsEmpty)`.
- `MergeFloorSuite` itself is just `((suite * 100) + floor)`
  (`Link.c:34-37`) - the `+8` is the caller's job. Getting this wrong shifts every
  link in a house by one floor.

Beyond `CheckHouseForProblems`, three additional constraints are enforced elsewhere
and matter for a new house:

| constraint | citation | value |
|---|---|---|
| the Finder type/creator the house scanner looks for | `SelectHouse.c:592-594`; created at `House.c:85` `FSpCreate(&theSpec, 'ozm5', 'gliH', ...)` | type `gliH`, creator `ozm5` |
| the directory scan depth | `SelectHouse.c:559` `#define kMaxDirectories 32` | 32 directories, breadth-first from the app's directory |
| how many houses the app will list | `SelectHouse.c:594` `housesFound < maxFiles`; fresh default at `Main.c:156-157`, bad-prefs fallback at `Main.c:87-89`, dialog clamp at `Settings.c:264-267` | default **48**; 12 only as the fallback for a stored value outside `[12, 500]` |
| manually added houses outside the scan | `SelectHouse.c:35` `#define kMaxExtraHouses 8` | 8 |
| the house version the reader accepts | `HouseIO.c:394-400` | `version < kNewHouseVersion` (0x0300); all 22 ship `0x0200` = `kHouseVersion` |
| `kUserBackground` PICT ID window the editor accepts | `GliderPRO/Sources/RoomInfo.c:762` | `longID >= 3000 && longID < 3800 && PictIDExists(longID)` |
| `kCustomPict` PICT ID clamp | `GliderPRO/Sources/ObjectInfo.c:1213` | `data.g.height` clamped to `[10000, 32767]` |
| `kSoundTrigger` sound ID clamp | `ObjectInfo.c:1237` | `data.e.where` clamped to `[3000, 32767]` |

`maxFiles` is a real, citable limitation, but the shipped default is **48**, not 12:
with all 22 shipped houses installed and default preferences the original app
enumerates all of them. The 12-house ceiling only bites when a preferences file
carries an out-of-range `wasMaxFiles`, in which case `Main.c:89` substitutes 12 and
the app then lists only the first 12 houses it finds.

---

## 10. Recipe: a new house in the same spirit

Everything in this section is derived from the measurements above. It is written as
a construction procedure so it can be automated.

### 10.1 Byte-level must-haves

1. Data fork is exactly `866 + 348 * nRooms` bytes, big-endian, no padding, no
   alignment holes (see 2.1-2.4).
2. `version = 0x0200` (`kHouseVersion`). `0x0300` or higher is rejected outright
   (`HouseIO.c:395-400`).
3. `timeStamp` = seconds since 1904-01-01 with bit 31 cleared
   (`timeStamp &= 0x7FFFFFFF`, `HouseIO.c:478`). Bit 0 is the lock flag:
   set it to lock (`|= 0x00000001`, :486), clear it to leave the house editable
   (`&= 0x7FFFFFFE`, :484). 15 of 22 shipped houses are locked.
4. `flags`: bit 0 = wardBit (**never set in any shipped house**), bit 1 = phoneBit
   (7 houses), bit 2 = *disable* the star count in the banner (only Art Museum).
   `flags = 0` is the safe default.
5. `nRooms >= 1` or `ReadHouse` bails with `kYellowNoRooms` (`HouseIO.c:385-392`).
6. `firstRoom` must index a real room; its `(floor, suite)` is where the glider
   spawns, at rect `{initial.v, initial.h, initial.v + 20, initial.h + 48}`.
7. Every room: `name[0] <= 27`, `unusedByte = 0`, `numObjects` == live slot count,
   all 24 slots present with `what = -1` for empties, unique `(floor, suite)`,
   `floor` in `[-7, 56]`, `suite` in `[0, 127]`.
8. `openings` can be left 0 - it is recomputed at load and **is 0 in all 4070
   shipped rooms** (`Room.c:179` is the only assignment).
9. `visited` should be 0 in a fresh house (436 shipped rooms have it set, which is
   just play residue).
10. For any room whose `background >= 3000`, either set `bounds` non-zero with bit 0
    set (the "version 2 bounds present" marker) or ship a 4-byte `'bnds'` resource
    with the same ID as the background PICT. 155 shipped rooms rely on the
    `'bnds'` fallback and **zero** of them are missing the resource. If neither
    exists, `GetOriginalBounding` returns 0 and raises
    `YellowAlert(kYellowNoBoundsRes, 0)` (`Room.c:937-966`).
11. `bounds` bit layout (raw field, after the `<< 1` and `+ 1` of
    `RoomInfo.c:778-779`): bit 0 = marker, bit 1 = left open, bit 2 = top open,
    bit 3 = right open, bit 4 = bottom open, bit 5 = floor support / is-structure.
    Observed values: every one of the 1669 non-zero `bounds` in the corpus has
    bit 0 set; the most common are 31 (all four walls open, no support) 542 times
    and 1 (nothing open) 167 times.
12. Resource fork: `PICT` IDs 3000-3299 for structure backgrounds, 3300-3799 for
    open backgrounds, 10000+ for `kCustomPict` art, `snd ` IDs 3000+ for
    `kSoundTrigger`. Remember that the house fork is pushed onto the Resource
    Manager search chain by `UseResFile` (`HouseIO.c:575`), so **any** ID you ship
    shadows the application's resource of the same type and ID.
13. Optional QuickTime sidecar named `<house name>.mov` in the same folder
    (`PasStringConcat(theSpec.name, "\p.mov")`, `HouseIO.c:81`), displayed by
    `kTV` objects. 15 of 22 houses ship one; four houses place `kTV`s without a
    movie and simply show a blank screen.

### 10.2 Numeric targets

Pick a tier, then hit these:

| target | tutorial | small | medium | large | epic |
|---|---|---|---|---|---|
| real rooms | 35-45 | 45-85 | 85-140 | 175-303 | 383-531 |
| occupied floors | 6-7 | 5-14 | 10-14 | 12-20 | 18-35 |
| occupied suites | 9-10 | 13-20 | 10-42 | 27-39 | 45-101 |
| grid density | 0.6-0.65 | 0.2-0.65 | 0.5-1.0 | 0.44-0.57 | 0.22-0.64 |
| total objects | 78-138 | 300-620 | 570-1100 | 1280-1820 | 2990-5840 |
| objects/room | 2.2-3.1 | 7-19 | 5.2-8.2 | 6.0-7.5 | 6.6-14.5 |
| empty rooms | ~20 % | 10-25 % | 15-25 % | 20-25 % | 20-30 % |
| rooms at the 24 ceiling | 0 | 0-6 | 0-3 | 5-22 | 4-77 |
| enemies/room | 0.0-0.1 | 0.5-1.3 | 0.3-1.0 | 0.2-0.8 | 0.25-0.73 |
| prizes/room | 0.03-0.29 | 0.4-0.95 | 0.5-1.4 | 0.37-0.73 | 0.42-1.10 |
| prize:enemy | n/a | 0.7-1.5 | 0.5-2.4 | 0.7-2.7 | 1.1-2.2 |
| stars | 1 | 1-4 | 1-6 | 3-5 | 1-6 |
| batteries | 0-1 | 1-6 | 1-8 | 0-15 | 2-27 |
| rubber bands | 0-1 | 0-4 | 1-6 | 1-10 | 8-32 |
| dark rooms | 0 % | 0-4 % | 0-8 % | 0-18 % | 0-11 % |
| BFS eccentricity | 11-15 | 2-23 | 10-39 | 24-46 | 24-59 |
| distinct `what` codes | 15-48 | 50-69 | 50-103 | 85-110 | 60-114 |
| total points | 8500-12500 | 7900-45500 | 30700-58900 | 44600-86300 | 73000-141400 |

Corpus-wide targets that hold at any size: **7.7 objects/room mean, median 6**;
**0.51 enemies/room**; **0.74 prizes/room**; **one star per 59 rooms**;
**291 points/room**; **0.69 link-bearing object slots/room** (2796 corpus-wide);
**4.6 % dark rooms** (play mode; 4.8 % in the editor); **left wall open 68 %, right 74 %, ceiling absent 56 %, floor
absent 49 %**.

### 10.3 Construction procedure

1. **Choose a spine.** Pick a start suite in the 54-69 band (**19 of 22 houses'
   `firstRoom` sits there**; the exceptions are CD Demo House at suite 2,
   Teddy World at 7 and SpacePods at 112) and floor 1 (**12 of 22**) and lay a
   horizontal run of 8-16 rooms. Corpus
   horizontal adjacency is much more common than vertical (3015 open right walls
   vs 2263 absent ceilings), and 2-way horizontal links outnumber 2-way vertical
   in 21 of 22 houses.
2. **Add vertical branches.** Every 3-6 rooms along the spine, branch up or down.
   Use one of three mechanisms, in the corpus proportions: absent floor/ceiling
   (free, no object needed), a `kUpStairs`/`kDownStairs` pair (163 pairs
   corpus-wide, always paired), or a `kFloorTrans`/`kCeilingTrans` duct (244/465
   placements; 107 rooms have both).
3. **Assign backgrounds by region**, following 7.1: interiors (2000-2008 or
   3000-3299) for the parts where the puzzles are, `kSky`/`kStars` (2015/2017) for
   vertical shafts and outdoor gaps, `kDirt` (2011) for basements/sewers,
   `kRoof`/`kSkywalk` (2014/2010) for the transition band between the two.
   Target mix, from the corpus: 15 % indoor, 10 % kDirt, 3 % ground-outdoor,
   5 % elevated, 22 % air/space, 27 % user-structure, 17 % user-open.
4. **Light every user-art room.** Add exactly one `kInvisLight` (0x58) with
   `initial = true` to any room whose background is >= 3000 and has no lamp, door
   or window. This single rule accounts for 434 of the corpus's rooms and is the
   most common mistake a new author can make.
5. **Fill by motif budget** (7.2): interiors get 11-13 objects, user-open rooms
   ~7, air/space and kDirt and elevated rooms 0-4. Do not fill uniformly; the
   corpus median room has 6 objects and 20 % have none.
6. **Place updrafts before obstacles.** `kFloorVent` (0x01, 1458 placements) is
   the primary vertical-mobility tool indoors; `kInvisBlower` (0x0D, 2336) is the
   universal one; `kLiftArea` (0x10, 716) holds the glider in place. Corpus ratio
   of blower-class objects to rooms is 1.49.
7. **Add obstacles with `kInvisObstacle` (0x1C, 666) and `kInvisBounce` (0x1F,
   1670)** rather than relying on visible furniture, which is what the shipped
   houses do: the invisible pair outnumbers every named piece of furniture.
8. **Sprinkle prizes at 0.74/room, front-loaded.** Concentrate clocks and bands in
   the 20-40 % depth band (corpus peak 1.16-1.23 prizes/room there) and thin out
   after halfway (0.56-0.79).
9. **Put the hazard ridge at ~two-thirds depth.** Corpus enemy density peaks at
   0.795/room in the 60-70 % depth decile; keep the first 10 % under 0.4/room.
   Start with `kDrip` (0x77) and `kBalloon` (0x71); reserve `kDartLf/Rt` and
   `kCopterLf/Rt` for the ridge.
10. **Place stars.** One per ~59 rooms; mean depth fraction 0.57. Decorate each
    star room to ~15 objects (corpus mean 14.75) with a `kSparkle` cluster
    (6-16) and/or a `kLiftArea` bowl (4-6).
11. **Write the banner and trailer.** Banner wrapped at 40 columns, trailer at 64,
    each stored as a `Str255` with embedded `\r` at the wrap points. 14 of 22
    houses ship an empty banner, so it is optional; every house has a trailer.
12. **Set `firstRoom`'s contents richer than average**: 13 objects, no enemies,
    and either a welcome sign in `kCustomPict` or a "falling from the sky"
    cloudscape.
13. **Run the equivalent of `CheckHouseForProblems`** (9) before shipping,
    especially the staircase pairing and the "at least one star" checks.

### 10.4 The smallest legal house

Sampler is the shipped existence proof and is worth copying byte-for-byte as a
template: 2 rooms, 1564-byte data fork, 286-byte resource fork (no `PICT`, no
`snd `, no `bnds`), no `.mov`, 11 objects, 9 distinct codes, backgrounds 2003 and
2012 only, 1 star, 4 link slots of which 2 are set, 5200 total points, unlocked.
A one-room house would be `866 + 348 = 1214` bytes.

### 10.5 What to avoid

- **Do not ship 0 stars.** Fun House does and is unwinnable by the normal exit
  condition (`HouseLegal.c:1203`).
- **Do not rely on `where == -100` links.** 165 of the corpus's 189 dangling links
  are exactly that value with `who == 255`; they are inert placeholders the editor
  wrote and nobody cleaned up.
- **Do not exceed 24 objects per room.** 222 shipped rooms are jammed at the
  ceiling, which is a design smell, not a target.
- **Do not leave `bounds` non-zero on a built-in background.** 27 shipped rooms do;
  `DetermineRoomOpenings` ignores `bounds` for `background < kUserBackground`
  (`Room.c:816-933`), so the value is dead data that misleads tools.
- **Do not place a `kSoundTrigger` whose `where` has no matching `snd ` resource.**
  13 shipped triggers are silent, 11 of them in Teddy World, which ships zero
  `snd ` resources at all.
- **Do not place `kTV` without a `.mov`.** Four houses do (California or Bust!,
  In The Mirror, Metropolis, The Asylum Pro).
- **Do not leave rooms named "Untitled Room"** (53 shipped rooms in 4 houses).
- **Do not create rooms that are unreachable from `firstRoom`.** 15 of 22 houses
  have some, and Fun House reaches only 5 of its 43 rooms under the static model.

---

## 11. Anomalies and bugs in the shipped data

Every item here was measured, not inferred. A Go port must tolerate all of them or
it will refuse to load houses the original app loads happily.

### 11.1 `openings` is dead data in every shipped room

`openings` (`roomType` offset 56, `short`) is 0 in **all 4070 rooms**. The only
assignment in the whole source is `thisRoom->openings = 0;`
(`GliderPRO/Sources/Room.c:179`); passability is always recomputed live from
`bounds`, `background` and `tiles`. A port should treat the field as reserved.

### 11.2 The `Str27` room-name padding is never zeroed

**4008 of 4070 rooms (98.5 %)** have non-zero bytes after `name[0]` bytes of text
inside the 28-byte `Str27` field. The residue is the tail of whatever name
previously occupied that memory. Concretely, in Art Museum: room 0 is named
`Narcissus` and its padding reads `rcissus\x00\x00\x00\x07\xe0\x1f\xf8?\xfc\x7f\xfe`;
room 9 is `Van Gogh` with padding `'s Starry Night?\xfc\x7f\xfe`; room 11 is
`Salvador Dali` with padding `sus\x00...`. In Demo House room 0 (`Air Vents`) the
padding is `6d 6f 6f 6d 55 70 00 00 00 00 07 e0 1f f8 3f fc 7f fe` - the tail of
"...moomUp".

The recurring trailing octet `07 e0 1f f8 3f fc 7f fe` appears at the end of the
name field in room after room; it is the untouched tail of the editor's blank-room
template struct.

All-rooms-dirty in **9 houses**: Art Museum 109/109, California or Bust! 16/16,
Castle o' the Air 85/85, Davis Station 65/65, Demo House 45/45, Empty House 35/35,
Fun House 43/43, In The Mirror 97/97, Sampler 2/2. Lowest ratio: Nemo's Market
118/124.

**Consequence for a port:** never hash, compare or round-trip the full 28 bytes.
Read `name[0]` and ignore the rest; write whatever you like into the padding (the
original certainly did).

### 11.3 The three "unused" header fields carry garbage

| field | offset | non-zero in | values |
|---|---|---|---|
| `unusedShort` | 2 | **10 of 22** | Art Museum -30082, California or Bust! 13107, Castle o' the Air 259, Davis Station 222, Fun House 26228, Grand Prix 60, In The Mirror 147, Land of Illusion 259, Metropolis 196, Titanic 2074 |
| `unusedBoolean` | 861 | **7 of 22** | Art Museum 255, Castle o' the Air 30, Davis Station 37, Grand Prix 185, Land of Illusion 30, Metropolis 14, Sampler 2 |
| `savedGame` (40 B `gameType`) with `hasGame == 0` | 820 | 20 of 22 | see 11.4 |

`unusedBoolean` is a `Boolean` (1 byte) holding 255, 30, 37, 185, 14, 2 - values
that are neither `true` (1) nor `false` (0). A Go port must read it as `uint8`, not
`bool`.

### 11.4 `savedGame` is stale memory in 20 of 22 houses

`hasGame` is 1 in exactly two houses: **ImagineHouse PRO II** (`roomNumber` 45 of
279, score 5900, 5 gliders, `energy` -150, `bands` 23, `foil` 8, `wasStarsLeft` 3)
and **Titanic** (`roomNumber` 104 of 208, score 4700, 2 gliders,
`wasStarsLeft` 1). Both are valid resumable games.

The other 20 houses still contain a `gameType` block, and it is not zeroed:

- Art Museum's is raw pixel garbage:
  `f6 f6 f6 ff f6 f6 f6 f6 f6 ff 00 00 00 ff f6 f6 f6 f6 f6 ff f6 f6 f9 f9 f9 ff 00 00 00 ff f6 f6 f6 f6 f6 ff f6 f9 2a 2a`,
  which parses as `version = -2314`, `energy = -1537`, `roomNumber = 255`.
- **Fun House's `savedGame` contains the Pascal string `\pDiskCopy`** (bytes
  `08 44 69 73 6b 43 6f 70 79`) - the house file was built from a Disk Copy image
  and 40 bytes of the image header leaked into the struct.
- Grand Prix's is a run of ascending `Point` pairs
  (`00a9 0191 | 00a9 01df | 00a9 01e0 | 00b7 0190 | ...`) - leftover object
  coordinates.
- Sampler's is `00 01 00 08 00...00 47 3e 04 00 00 00 ff 00 00 ff ff ff ff ff ff 00 01 ff ff ff ff cc cc`.

Two pairs of houses have **byte-identical** `savedGame` blocks, which pins down
the copy-paste lineage of the files: **CD Demo House == Slumberland** and
**Nemo's Market == SpacePods**.

**Consequence:** `savedGame` must only be interpreted when `hasGame != 0`, and a
port must not validate its contents otherwise.

### 11.5 Sampler's data fork has 2 slack bytes

`866 + 348 * 2 = 1562` but the fork is **1564** bytes. `ValidateNumberOfRooms`
(`HouseLegal.c:629-631`) uses integer division so it accepts the file; `ReadHouse`
allocates `GetEOF` bytes and never notices. Sampler is also the only house with
BinHex Finder flags `0x0100` (the other 21 are `0x0500`, except In The Mirror at
`0x0504`) and the only one time-stamped in the year 2000 (`0x3540BFE8` ->
2000-05-11 19:52:08) rather than 1995.

**Consequence:** a Go loader must size the room array from `nRooms`, not from
`(fileSize - 866) / 348`, and must not reject trailing bytes.

### 11.6 189 dangling links, 165 of them the same sentinel

| house | dangling | of which `where == -100` | hard (`who != 255`) |
|---|---|---|---|
| Slumberland | 69 | 69 | 0 |
| Land of Illusion | 48 | 48 | 0 |
| Rainbow's End | 36 | 36 | 0 |
| Leviathan | 23 | 0 | **21** |
| Castle o' the Air | 12 | 12 | 0 |
| ImagineHouse PRO II | 1 | 0 | **1** |

`where == -100` is `MergeFloorSuite(0, kRoomIsEmpty)` = `(-1 * 100) + 0`, i.e. the
editor's "room not chosen yet" placeholder, and all 165 also have `who == 255`, so
the runtime treats them as inert. By object type: `kCeilingTrans` 110,
`kInvisTrans` 38, `kFloorTrans` 12, `kMailboxRt` 2, `kThermostat` 1,
`kKnifeSwitch` 1, `kLightSwitch` 1.

The 22 "hard" dangling links (Leviathan 21, ImagineHouse PRO II 1) point at a
`(floor, suite)` cell that holds no room *and* name a specific object slot. Those
are genuine authoring errors.

Separately, **CD Demo House room 72 slot 22** is a `kMailboxRt` with `who = 35`,
which is `>= kMaxRoomObs` (24) - an out-of-range object index that would index past
the destination room's object array. A port must bounds-check `who`.

### 11.7 27 rooms carry stale `bounds` on a built-in background

`DetermineRoomOpenings`, `DoesRoomHaveFloor`, `DoesRoomHaveCeiling`,
`IsShadowVisible` and `IsRoomAStructure` all consult `bounds` **only** when
`background >= kUserBackground` (3000). These 27 rooms therefore carry a non-zero
`bounds` that the engine ignores:

| house | (bounds / background) x count |
|---|---|
| CD Demo House | (1/2017) x3, (3/2015) x1, (15/2013) x6, (15/2015) x5, (23/2015) x8 |
| Grand Prix | (9/2012) x1, (35/2000) x1 |
| Land of Illusion | (3/2015) x1 |
| Teddy World | (1/2012) x1 |

They are the fossil record of rooms whose background was changed from custom art
to a built-in after the openings were set.

### 11.8 Fun House is broken in four separate ways

1. **0 `kStar`** - unwinnable by the normal exit condition
   (`HouseLegal.c:1203` would flag it red).
2. **Only 5 of 43 rooms are forward-reachable** from `firstRoom` (29,
   "Mission Impossible?") and its BFS eccentricity is **2**. Room 29 holds 19
   `kInvisTrans` plus a `kDeluxeTrans` that all self-link to `where = 5812`, and
   its `kCeilingTrans` exits to room 39, the head of a dead-end 3-room meadow strip
   whose `kMailboxRt` has `where = -1`. This is authored data, not a parse error.
3. **The trailer is a debug string**: `h: 107v: 55`.
4. **Its high-score board is completely full** (10 non-zero scores, top 1600 by
   `Ozma`) while its `savedGame` block contains the leaked `\pDiskCopy` bytes.

It also shadows two *built-in* background PICTs (2014 `kRoof` and 2015 `kSky`), the
only house in the corpus to override built-in room art.

### 11.9 High-score and banner residue

- **Empty House** ships its trailer as the unedited editor template:
  `"Enter Finished-House Message here (max. 255 characters)"`.
- **Art Museum, Sampler and Slumberland** ship the default high-score banner
  `"Your Message Here"` with the default name `"Your Name"`.
- **Empty House and Land of Illusion** have completely blank score boards
  (`0` scores, names `"--------------"`).
- Demo House's trailer claims Slumberland "has over 400 rooms" - it has **383**.
- **`houseType.timeStamp` vs the high-score timestamps disagree in 6 houses**,
  and in 4 of them the score is *newer* than the house: Davis Station
  (house 1995-08-13, top score 1996-10-17), Slumberland (1995-08-13 vs
  2000-05-11), Metropolis (1995-07-20 vs 1995-07-25), Art Museum (1995-09-16 vs
  1996-01-13). The `timeStamp` field is only rewritten when `fileDirty`
  (`HouseIO.c:475-489`), whereas scores are written by `WriteScoresToDisk`, so the
  two clocks drift.

### 11.10 SpacePods's 402 rooms are all non-structures

All 402 SpacePods rooms use backgrounds 3001-3010, which are in the *structure*
sub-range (`< kUserStructureRange` 3300), yet `IsRoomAStructure` returns **false**
for all of them: 401 rooms have `bounds = 31` and one has `bounds = 17`, and
`31 & 32 == 0`. The `bounds != 0` branch (`Room.c:775-778`) wins over the
sub-range heuristic, so no floor-support strip is ever drawn and all 402 rooms
have neither floor nor ceiling. It is the most extreme use of the `bounds` bit-5
override in the corpus, and it is deliberate: SpacePods is a free-fall space
station.

The same mechanism gives Nemo's Market and Titanic zero structure rooms.
Conversely Castle o' the Air has `bounds == 0` in **all 85** rooms and relies
entirely on its 9 `'bnds'` resources.

### 11.11 Corpus-wide field census, for a port's test suite

| field | observation |
|---|---|
| `version` | `0x0200` in all 22 |
| `nRooms` vs real rooms | equal in all 22 (no placeholders) |
| `unusedByte` (roomType offset 32) | 0 in all 4070 rooms |
| `openings` | 0 in all 4070 rooms |
| `numObjects` | matches live-slot count in all 4070 rooms |
| `visited` | 1 in 436 rooms; **Leviathan has 0** despite 472 rooms |
| `leftStart` | 32 in 3208 rooms, 0 in 642, 220 rooms over 110 distinct values in 2..255 |
| `rightStart` | 32 in 3211 rooms, 0 in 639 |
| `tiles[8]` histogram | tile 0: 5114, 1: 7036, 2: 6395, 3: 3229, 4: 3925, 5: 2263, 6: 2161, 7: 2437 |
| `bounds` | 0 in 2401 rooms; 31 distinct non-zero values, **all with bit 0 set**; most common 31 (542, all four sides open) and 1 (167, fully enclosed) |
| `wardBitSet` (`flags` bit 0) | false in all 22 |
| BinHex CRCs | all 66 (22 header + 22 data + 22 resource) verify |
| `kCustomPict` resolution | all 4782 placements resolve to a `PICT` in the *same* house's fork; none fall through to the app's placeholder `PICT` 10000 |

---

## Open questions

1. **What exactly does `unusedShort` at `houseType` offset 2 hold?** Ten houses
   carry non-zero values ranging from -30082 to 26228 with no visible pattern; the
   source never reads or writes it. It may be a version-1 field abandoned by
   `ConvertHouseVer1To2` (`GliderPRO/Sources/House.c:746-818`), but the conversion
   routine does not clear it either.
2. **What was `roomType.unusedByte` (offset 32) for?** It is zeroed defensively by
   `CheckRoomNameLength` (`HouseLegal.c:868`) and is 0 in all 4070 shipped
   rooms, so its original meaning is unrecoverable from data. Its position -
   immediately after the 28-byte name and the 4 bytes of `bounds`/`leftStart`/
   `rightStart` - suggests it once paired with `visited` as a 2-byte field.
3. ~~**Are `leftStart` / `rightStart` in pixels or tiles?**~~ **Resolved: pixels,
   and the axis is vertical, not horizontal.** `Transit.c:176` and `:185` offset the
   48x20 `enterRect` by `(0, kGliderStartsDown + leftStart - 2)` when the glider
   enters at the room's west edge, and `:208`/`:217` offset it by
   `(kRoomWide - 48, kGliderStartsDown + rightStart - 2)` at the east edge. With
   `kGliderStartsDown` = 32 (`GliderDefines.h:569`), the effective top of the entry
   rect is `30 + leftStart` pixels. The editor writes the same fields with a purely
   vertical drag (`ObjectEdit.c:539-547` for `leftStart`, `:551-559` for
   `rightStart`), and `CreateNewRoom` initialises both to 32 (`Room.c:170-171`), which
   is why 32 dominates the corpus. 32 being half of `kTileWide` is a coincidence.
4. **What is `Fun House`'s room 29 `where = 5812` supposed to mean?**
   `ExtractFloorSuite(5812)` for version `0x0200` gives `floor = 12 - 8 = 4`,
   `suite = 58`, which *is* room 29 itself. Nineteen `kInvisTrans` and one
   `kDeluxeTrans` all self-link there. Was this a deliberate "teleport in place"
   trick, or the residue of a bulk edit?
5. **Why is `kSlider` (0x2F, 354 placements) confined to 7 houses?** It is a prize
   by union arm but has an unusually high placement-to-house ratio. Whether it
   behaves as a moving platform or a slippery patch is a question for the object
   behaviour document, not this one.
6. ~~**Do any shipped `.mov` files have more than one video track / a non-trivial
   `GetMovieBox`?**~~ **Resolved: no movie has more than one track, but the boxes
   are non-trivial - three distinct sizes, two of which get clipped.** The
   containers were decoded in `docs/analysis/quicktime-movies.md` (see
   "Appendix A" at the end of this document for the per-house table). 13 movies are
   64x49 (an exact fit for the TV screen rect), `Teddy World.mov` is 64x50 (one row
   clipped) and `Demo House.mov` is 82x62 (18 columns and 13 rows clipped by
   `SetMovieDisplayClipRgn`, `ObjectDrawAll.c:691`). Single-track is proved
   directly for `Demo House.mov`, the only file whose `moov` survived
   (`mvhd.nextTrackID` = 2, exactly one `trak`, one `vide` media handler,
   `tkhd.volume` = 0); for the other 14 the `moov` was lost with the resource fork
   and the size is recovered from the codec bitstream, whose samples tile the
   `mdat` payload exactly with no bytes left for a second track.
7. **Is `bounds` bit 5 the only meaning of "structure", or did the editor ever
   write bits above 5?** Observed `bounds` values top out at 63, i.e. bits 0-5 only.
   `RoomInfo.c:770-779` builds exactly those six bits. But
   `GetOriginalBounding` returns only bits 0-3 (`Room.c:955-963`), so a `'bnds'`
   room can never be a structure by the `bounds` path and falls back to the
   `background < 3300` test. Whether that asymmetry was intentional is unclear.
8. **Which of the 22 houses were shipped with the retail 1.0.4 disk versus added
   later?** The `timeStamp` ordering (1995-06-08 .. 1995-12-23, plus Sampler at
   2000-05-11) suggests Sampler and possibly Titanic post-date the rest, but the
   GPL release contains no manifest.
9. **`GliderPRO/README.md` leaves six houses unattributed**: Art Museum,
   California or Bust!, Castle o' the Air, Empty House, Fun House, Sampler. Castle
   o' the Air's own banner says "(by john calhoun)" and Sampler's says "Welcome to
   Omid's Happy Home", but the README credits neither.

---

## Porting notes

### Data representation

1. **Everything on disk is big-endian and unpadded.** `houseType` is 866 bytes,
   `roomType` is 348, `objectType` is 12, `scoresType` is 292, `gameType` is 40.
   Do not use Go struct layout to define the format - write explicit
   `binary.BigEndian` readers per field. A naive `encoding/binary.Read` on a Go
   struct will insert alignment padding and produce garbage.
2. **`Point` is `{v, h}`, vertical first.** `Rect` is `{top, left, bottom, right}`.
   Getting either order backwards silently mirrors or transposes every object in
   the game.
3. **The 10-byte object payload is a C union with nine arms.** Decode it by
   dispatching on the `what` code range (0x01-0x10 -> `blowerType`, 0x11-0x1F ->
   `furnitureType`, 0x21-0x2F -> `bonusType`, 0x31-0x40 -> `transportType`,
   0x41-0x49 -> `switchType`, 0x51-0x58 -> `lightType`, 0x61-0x6E ->
   `applianceType`, 0x71-0x79 -> `enemyType`, 0x81-0x8F -> `clutterType`). Note
   the gaps: 0x20, 0x30, 0x4A-0x50, 0x59-0x60, 0x6F-0x70, 0x7A-0x80, 0x90+ are
   unused. Preserve the raw 10 bytes alongside the decoded view so a round-trip is
   byte-exact.
4. **Pascal strings, not C strings.** `Str15` = 16 bytes, `Str27` = 28,
   `Str31` = 32, `Str255` = 256; byte 0 is the length. The padding is garbage in
   98.5 % of shipped rooms (11.2), so slice to `[1 : 1+n]` and discard the rest.
5. **MacRoman, not UTF-8.** Room names, banners and trailers use MacRoman;
   `GliderPRO/Glider PRO.r` and the house data both contain high-bit bytes.
   Decode explicitly. (This is also why GNU `grep` treats the source files as
   binary - use `grep -a`.)
6. **Classic Mac line endings.** Banners and trailers wrap with bare `\r`
   (`WrapText(banner, 40)`, `HouseLegal.c:611`). Rendering them with `\n`-only
   logic will show one long line.
7. **The Mac epoch is 1904-01-01, not 1970-01-01.** `timeStamp` is an unsigned
   seconds count with **bit 31 forcibly cleared** (`HouseIO.c:478`), so restoring a
   real date requires adding `0x80000000` back for stamps in the 1995 era. Bit 0 is
   the lock flag, so timestamps are always even-or-odd-by-design and have
   2-second granularity at best.
8. **`Boolean` fields are bytes that are not always 0 or 1.** `unusedBoolean`
   holds 255, 30, 37, 185, 14 and 2 in shipped houses. Model as `uint8` and test
   `!= 0`.
9. **`who` is a `Byte` with 255 as a sentinel**, and `where` is a `short` with
   `-1` (`kRoomIsEmpty`) and `-100` as sentinels. Both need explicit checks;
   `who >= 24` occurs in the wild (11.6).

### Container and resources

10. **BinHex 4.0 is the distribution wrapper, not the format.** Decode the 64-char
    alphabet, expand `0x90` RLE, then split into header / data fork / resource fork
    with a CRC-16 after each. The CRC is polynomial `0x1021`, seed 0, MSB-first, no
    reflection, no final XOR - i.e. **CRC-16/XMODEM** (`binascii.crc_hqx(buf, 0)`),
    computed over the covered bytes with **no zero-byte augmentation**. All 66 CRCs
    in the corpus verify with that definition, which is the check that proves the
    decoder is right.
11. **A house is up to three files**: data fork (level data), resource fork (art
    and sound), and an optional `<name>.mov` sidecar. A Go port on a modern
    filesystem must invent a container - either keep the BinHex, or split into
    `house.dat` + `house.rsrc` + `house.mov`, or repackage as a zip. Whatever you
    choose, the resource fork is 17x larger than the data fork on average, so
    lazy-loading it matters.
12. **The Resource Manager search chain is a feature, not an accident.**
    `OpenHouseResFork` calls `UseResFile(houseResFork)` (`HouseIO.c:575`), which
    pushes the house's fork in *front* of the application's. Thirteen shipped
    houses exploit this to replace the banner-page art (`PICT` 1991/1992/1993),
    three replace the star-counter art (1017/1018), Teddy World replaces the pause
    screens (1015/1016) and the Ozma image (3975), Nemo's Market replaces all four
    door PICTs (3981-3984), and Fun House replaces two *built-in room backgrounds*
    (2014, 2015). A Go port needs an explicit two-level resource lookup
    (house first, then app) or those houses will look wrong.
13. **`PICT` ID ranges are load-bearing**: 2000-2017 built-in backgrounds
    (`kBaseBackgroundID` 2000, `kNumBackgrounds` 18,
    `GliderPRO/Headers/GliderDefines.h:519-521`), 3000-3299 user structure
    backgrounds, 3300-3799 user open backgrounds (`kUserBackground` 3000,
    `kUserStructureRange` 3300, `:522-523`), 3900-3997 object art and masks,
    10000+ `kCustomPict` art. `snd ` 1000-1062 and 2000-2006 in the app,
    3000+ for house `kSoundTrigger`s.
14. **`'bnds'` is a 4-byte resource** whose ID equals a user background's PICT ID,
    holding `{left, top, right, bottom}` as four `Boolean` bytes and combined as
    `left*1 + top*2 + right*4 + bottom*8` (`Room.c:955-963`). 70 exist across 8
    houses; 155 rooms depend on them; ImagineHouse PRO II ships 14 it never uses.
15. **QuickTime is unavoidable for `kTV`.** `OpenHouseMovie` (`HouseIO.c:67-142`)
    pre-loads the whole movie into RAM (`LoadMovieIntoRam`), prerolls it
    (`PrerollMovie(theMovie, 0, 0x000F0000)`), and sets `loopTimeBase` so it loops
    forever. A Go port needs a QuickTime demuxer or a pre-conversion step; the 15
    shipped movies total ~566 KB and range 6534 B (Titanic) to 119789 B
    (Demo House).

### Semantics

16. **Reimplement `DetermineRoomOpenings` exactly, including the kDirt bug.**
    `Room.c:816-933`. For `background >= 3000` the four open flags come from
    `bounds >> 1` bits 0-3 (or from `'bnds'` if `bounds == 0`). For built-ins it
    is per-background logic over `tiles[0]` and `tiles[7]`. **`kDirt` (2011) is
    the one case where the wall-limit and the open-flag disagree**: the left limit
    becomes `kLeftWallLimit` only when `leftTile == 1`, but `leftOpen` is
    `(leftTile != 0)`. Reproduce the discrepancy; do not "fix" it.
17. **Reimplement `GetNumberOfLights` exactly** (`Room.c:970-1099`), including that
    it uses `data.f.initial` in edit mode and `data.f.state` in play mode, that
    `kDirt` is self-lit only when **all eight** tiles are 0, that
    `kDoorInLf/kDoorInRt/kWindowInLf/kWindowInRt/kWallWindow` count unconditionally,
    and that user backgrounds (>= 3000) get **no** free light. Two structural details
    are easy to lose in a rewrite: the object scan is skipped entirely when the
    background switch already yielded a light (`if (count == 0)`, `Room.c:1004`
    and `:1068`), and it iterates all **`kMaxRoomObs` = 24** slots rather than
    `numObjects` (`Room.c:1006` and `:1070`). 187 shipped rooms are dark in play
    mode (197 in the editor) and 434 rooms exist only to hold a single
    `kInvisLight`.
18. **`MergeFloorSuite(floor, suite) = (suite * 100) + floor`** (`Link.c:34-37`)
    and the caller adds `kNumUndergroundFloors` (8) first (`Link.c:277`). For
    `version >= 0x0200`, `ExtractFloorSuite(combo) = (combo % 100 - 8, combo / 100)`;
    for version 1 houses the two are swapped. Off-by-eight here silently relocates
    every linked destination by one floor.
19. **`IsRoomAStructure` is purely cosmetic** - its only consumer is
    `DrawFloorSupport` (`RoomGraphics.c:70`, `:257`), which paints a 44-pixel
    (`kFloorSupportTall` 44) strip of `PICT` 1999 between vertically adjacent
    rooms. Getting it wrong costs you a visual detail, not gameplay.
20. **Point/prize scoring lives in `CountTotalHousePoints`**
    (`GliderPRO/Sources/HouseInfo.c:51-104`): `RealRoomNumberCount() * 100`, then
    `kRedClock` 100, `kBlueClock` 300, `kYellowClock` 500, `kCuckoo` 1000,
    `kStar` 5000, plus each `kInvisBonus`'s own `data.c.points`. Note it iterates
    **all 24 slots** of every non-empty room, so it counts objects in slots past
    `numObjects` if any existed (none do in the shipped corpus).
21. **`SetObjectsToDefaults`** (`GliderPRO/Sources/Play.c:603-710`) copies
    `initial -> state` for prizes (`data.c`), lights (`data.f`, :671-672),
    appliances (`data.g`), enemies (`data.h`) and does a nibble dance on
    `kDeluxeTrans.data.d.wide` (`(wide & 0xF0) >> 4`, :658-660). A port must run
    the equivalent pass before play or every switchable object starts in the wrong
    state.
22. **`maxFiles` defaults to 48** (`Main.c:156-157`, used at `SelectHouse.c:594`); a
    preferences value outside `[12, 500]` is replaced by 12 (`Main.c:87-89`), and the
    Brains dialog clamps user input to that range (`Settings.c:264-267`). Only the
    12-house fallback is below the 22 shipped houses. A port has no reason to keep
    any of these numbers.
23. **`kMaxRoomObs` 24, `kMaxStars` 4, `kMaxMasterObjects` 216**
    (`GliderDefines.h:250, 263, 266`). `kMaxStars` 4 is *not* a per-house limit -
    CD Demo House ships 9 stars - it is a per-screen drawing limit. Do not confuse
    them.

### Graphics and platform

24. **8-bit indexed colour throughout.** The app ships `clut` 2 and `dctb` 20
    resources; `PICT`s are QuickDraw pictures, many of them 8-bit with a custom
    palette. A Go port needs a QuickDraw `PICT` decoder (opcode-level) or a
    pre-conversion pass to PNG. `PICT` 10000 in the app is the 72x34 default
    placeholder for `kCustomPict` (`picFrame {0,0,34,72}`, `picSize 0x1056`).
25. **GWorlds are offscreen pixmaps.** `backSrcMap`, `workSrcMap`, `suppSrcMap`
    etc. become plain RGBA buffers. `CopyBits(..., srcCopy, nil)` is a blit;
    `CopyMask`/`CopyBits` with a mask region becomes an alpha blit. The tile
    engine draws `kNumTiles` 8 tiles of `kTileWide` 64 x `kTileHigh` 322 from
    `workSrcMap` into `backSrcMap` (`RoomGraphics.c:241-252`), i.e. the background
    strip for a room is a horizontal 8-tile arrangement selected by `tiles[8]`.
26. **The room canvas is 512 x 322** with these limits
    (`GliderDefines.h:503-511`): `kCeilingLimit` 8, `kFloorLimit` 312,
    `kRoofLimit` 122, `kLeftWallLimit` 12, `kNoLeftWallLimit` -24,
    `kRightWallLimit` 500, `kNoRightWallLimit` 536, `kNoCeilingLimit` -10,
    `kNoFloorLimit` 332. `kVertLocalOffset` is 322. Objects may sit at h = -1
    (11.6 / 5.6), so clip rather than assert.
27. **`FSSpec` (70 bytes inside `game2Type`) has no modern equivalent.** Saved
    external games reference houses by `{vRefNum, parID, name}`. A port must
    substitute a path or a house ID and version the save format accordingly.
28. **The 22 shipped houses are the compatibility test suite.** Any loader change
    should be validated by re-deriving the numbers in
    `docs/analysis/houses-inventory.md`: 4070 rooms, 31440 live objects,
    117 distinct `what` codes, 69 stars, 2796 link-bearing slots, 187 dark rooms (play mode),
    1669 non-zero `bounds`, 1,183,600 total points, and all 66 BinHex CRCs
    verifying.

---

## Appendix A (appended): the `.mov` companion files

*Added when open question 6 was resolved. This section is a summary only; the full
treatment - container layout, codec bitstreams, palettes, recovery procedure and the
`kTV` drawing path - is in `docs/analysis/quicktime-movies.md`, and the numbers below
are reproduced by `python3 tools/probe_mov.py survey`.*

A house named `X` can carry a sibling movie `GliderPRO/Houses/X.mov`, opened by
`OpenHouseMovie` by concatenating `"\p.mov"` onto the house's own `FSSpec` name
(`GliderPRO/Sources/HouseIO.c:81`). **15 of the 22 houses have one.** The 7 without are
California or Bust!, Empty House, Fun House, In The Mirror, Metropolis, Sampler and
The Asylum Pro.

At most **one** movie plays at a time, in the **first `kTV` (0x65) object of the current
central room only** (`GliderPRO/Sources/ObjectDrawAll.c:680-681`). Of the 80 `kTV`
placements in the corpus, 72 are in movie-bearing houses and 8 are in houses with no
movie at all, so at least 65 of the 80 can never show anything but static.

| House | `.mov` bytes | `moov` present | Codec | Frames | Depth | `GetMovieBox` W x H | kTV objects |
|---|---:|---|---|---:|---|---|---:|
| Art Museum | 28,232 | no | `raw ` | 9 | 8 | 64 x 49 | 2 |
| CD Demo House | 33,042 | no | `rle ` | 32 | 8 | 64 x 49 | 1 |
| Castle o' the Air | 21,960 | no | `raw ` | 7 | 8 | 64 x 49 | 2 |
| Davis Station | 34,504 | no | `raw ` | 11 | 8 | 64 x 49 | 1 |
| Demo House | 119,789 | **yes**, at offset 118,792 | `rle ` | 41 | 36 (4-bit gray) | **82 x 62** | 1 |
| Grand Prix | 26,407 | no | `smc ` | 21 | 8 | 64 x 49 | 2 |
| ImagineHouse PRO II | 30,707 | no | `rle ` | 10 | 8 | 64 x 49 | 7 |
| Land of Illusion | 16,016 | no | `rle ` | 28 | 8 | 64 x 49 | 10 |
| Leviathan | 65,920 | no | `rle ` | 21 | 8 | 64 x 49 | 4 |
| Nemo's Market | 28,541 | no | `rle ` | 24 | 8 | 64 x 49 | 1 |
| Rainbow's End | 7,021 | no | `rle ` | 25 | 8 | 64 x 49 | 5 |
| Slumberland | 37,640 | no | `raw ` | 12 | 8 | 64 x 49 | 8 |
| SpacePods | 51,291 | no | `rle ` | 40 | 8 | 64 x 49 | 8 |
| Teddy World | 28,587 | no | `rle ` | 45 | 8 | **64 x 50** | 7 |
| Titanic | 6,534 | no | `rle ` | 11 | 8 | 64 x 49 | 13 |
| **total** | **536,191** | 1 of 15 | 3 codecs | **337** | | 3 sizes | **72** (of 80) |

Two facts matter for a port and were not known when this document was first written.

1. **14 of the 15 files are headerless.** Their single `mdat` atom declares
   `size == 0` ("extends to end of file") and there is no `moov` atom anywhere in the
   file - the movie header lived in the **resource fork**, which was destroyed when the
   GPL source was published as plain files. `git log --all -- 'Houses/*.mov'` in
   `GliderPRO/upstream.git` shows the `.mov` files were committed once, in the initial
   check-in, and never re-exported; the author later fixed the same problem for the house
   files (BinHex) and for the application resource fork (DeRez) but not for the movies.
   Only `Demo House.mov` is self-contained. All 15 were nevertheless recovered by walking
   the self-delimiting codec bitstreams, whose samples tile each `mdat` payload exactly.
2. **The box is not always 64 x 49.** The `kTV` case builds a 64 x 49 destination at
   `itsRect.left + 17, itsRect.top + 10` (`GliderPRO/Sources/ObjectDrawAll.c:685`, with
   `tvScreen1` = `(0,171)..(64,220)` from `StructuresInit.c:570-571`), centres the movie
   box in it *without scaling* (`CenterRectInRect`, `RectUtils.c:142-154`) and clips to
   it (`SetMovieDisplayClipRgn`, `ObjectDrawAll.c:691`). So Teddy World loses one pixel
   row and Demo House loses 18 columns and 13 rows: `dx = (64-82)/2 = -9`,
   `dy = (49-62)/2 = -6`.

Frame rate is known only for `Demo House.mov`: `mvhd.timeScale` = 600 with
`stts` `sampleDuration` = 121, i.e. **4.9587 fps** (41 frames, `mvhd.duration` = 4961
units = 8.2683 s). For the other
14 the timescale was in the lost header and is unrecoverable.
