# Glider PRO Audio Subsystem: Sound Effects and Music

## Scope

This document specifies, exhaustively, how audio works in Glider PRO 1.0.4 (John
Calhoun, 1994; GPLv2 source release), so that the Go port can reproduce it
without reference to the original binary or to Inside Macintosh.

It covers:

- the Mac Sound Manager surface the game actually touches (four channels total,
  three for sound effects and one for music) and what each call must be replaced
  with in Go;
- the on-disk `'snd '` resource format, verified byte-for-byte against the 70
  resources shipped in the application and the 63 shipped inside the bundled
  houses;
- the sound-effect engine in `GliderPRO/Sources/Sound.c`: 64 sound slots, the
  three-voice priority mixer, the reserved "trigger" slot, and the load/dump
  lifecycle;
- the complete table of sound constants -> `'snd '` resource ID -> resource name
  -> sample statistics -> priority, plus a cross-reference of every one of the
  167 `PlayPrioritySound()` call sites in the tree;
- the per-room custom sound facility (`kSoundTrigger`, object type `0x49`) and
  the house resource-fork search order that makes it work;
- the music engine in `GliderPRO/Sources/Music.c`: what "music" is (7 sampled
  pieces, ids 2000..2006), the two scores, the four musical modes, the gapless
  double-buffer queue, and every place the mode changes;
- volume: the 0..7 user scale, the exact integer mapping to and from the system
  volume, and the fact that Glider PRO changes the *machine's* volume rather
  than attenuating its own output;
- the memory budget that can silently disable music and/or sounds at launch;
- an extraction plan that turns the shipped resources into raw PCM plus a
  manifest for the Go port, implemented in `tools/probe_snd.py`;
- a Go implementation sketch that translates the C control flow line for line,
  including the concurrency the original did not need;
- the 14 defects in the shipped audio code, each marked reproduce-or-fix.

It does **not** cover QuickTime movie playback (`Sources/Play.c`,
`#ifdef COMPILEQT`), which has its own audio path through the Movie Toolbox and
is unrelated to `Sound.c`/`Music.c`.

Every line-number citation is of the form `GliderPRO/Sources/Sound.c:123` and is
relative to the repository root. The shipped `.c`/`.h` files use
classic-Mac CR-only line endings; all line numbers were taken from copies
converted with `tr '\r' '\n'`, which is a one-for-one byte substitution and
therefore preserves line numbering exactly. `GliderPRO/Glider PRO.r` is already
LF-terminated and is cited directly.

## Sources read

Read in full:

| File | Lines | Why |
|---|---:|---|
| `GliderPRO/Sources/Sound.c` | 535 | the sound-effect engine |
| `GliderPRO/Sources/Music.c` | 419 | the music engine |
| `GliderPRO/Headers/GliderDefines.h` | 625 | sound ids, priorities, music modes, object types |
| `GliderPRO/Headers/Externs.h` | 394 | `prefsInfo`, volume prototypes |
| `GliderPRO/Headers/GliderStructs.h` | 347 | `switchType`, `objectType`, `roomType`, `houseType` |
| `GliderPRO/Headers/GliderVars.h` | 59 | shared externs |
| `GliderPRO/Headers/GliderProtos.h` | 530 | `Sound.c` / `Music.c` prototypes |

Read in the regions that touch audio (all call sites were located mechanically,
see section 6):

`Sources/Utilities.c` (volume), `Sources/Main.c` (init/kill order, prefs),
`Sources/Environ.c` (memory budget, `hasSM3`), `Sources/Settings.c` (Sound
prefs dialog), `Sources/Play.c` (mode changes, telephone and chimes),
`Sources/Input.c` (thrust/helium retrigger), `Sources/Render.c` (frame clock,
shreds, pendulums), `Sources/Player.c`, `Sources/Modes.c`,
`Sources/Interactions.c`, `Sources/Dynamics.c`, `Sources/Dynamics2.c`,
`Sources/Dynamics3.c`, `Sources/Objects.c`, `Sources/ObjectRects.c`,
`Sources/ObjectAdd.c`, `Sources/ObjectInfo.c`, `Sources/Triggers.c`,
`Sources/Trip.c`, `Sources/Transit.c`, `Sources/RoomGraphics.c`,
`Sources/GameOver.c`, `Sources/Events.c`, `Sources/Menu.c`,
`Sources/HighScores.c`, `Sources/HouseIO.c`, `Sources/RubberBands.c`,
`Sources/Grease.c`, `Sources/Scoreboard.c`, `Sources/DynamicMaps.c`.

Binary data parsed with python3 (all numbers in this document that describe
bytes were produced by these, not by reading the C):

- `GliderPRO/Glider PRO.r` - the DeRez dump of the application resource fork;
  70 `'snd '` resources at lines 127657..198152.
- `GliderPRO/Houses/*.binhex` - 24 BinHex 4.0 house files; 15 of them carry
  `'snd '` resources (63 in total).
- `tools/probe_snd.py` - the deliverable prototype decoder (section 11).
- `tools/probe_rez.py`, `tools/probe_house.py` - pre-existing helpers in this
  repo for the `.r` dump and for BinHex/resource-fork parsing.

## 1. Architecture at a glance

```
                 application resource fork ('snd ' 1000..1062, 2000..2006)
                          |                                  |
      InitSound()         |                InitMusic()       |
  LoadBufferSounds() -----+            LoadMusicSounds() ----+
      |  strips 20 bytes off the front of each resource, keeps the rest
      v                                       v
 theSoundData[0..62]  (63 x Ptr)        theMusicData[0..6]  (7 x Ptr)
 theSoundData[63]     <- per-room custom sound, loaded from the HOUSE fork
      |                                       |
      |  PlayPrioritySound(which, priority)   |  StartMusic() / MusicCallBack()
      v                                       v
 +-------------+ +-------------+ +-------------+      +---------------+
 |  channel0   | |  channel1   | |  channel2   |      | musicChannel  |
 | priority0   | | priority1   | | priority2   |      | queue depth 2 |
 | soundPlaying0 | soundPlaying1 | soundPlaying2       | + callBackCmd |
 +-------------+ +-------------+ +-------------+      +---------------+
        \              |              /                       |
         \             |             /                        |
          +----- Sound Manager mixer (sampledSynth, mono, no interpolation) ----+
                                     |
                          system output volume (0..0x100)
```

Four `SndChannelPtr`s exist at once: `channel0`, `channel1`, `channel2`
(`GliderPRO/Sources/Sound.c:29`) and `musicChannel`
(`GliderPRO/Sources/Music.c:29`). All four are created with the same parameters:
`sampledSynth`, `initNoInterp + initMono`
(`GliderPRO/Sources/Sound.c:378-396`, `GliderPRO/Sources/Music.c:286-288`).

Sound effects are fire-and-forget: one `bufferCmd` issued *immediately*
(flushing whatever that voice was doing), followed by a *queued* `callBackCmd`
whose only job is to mark the voice idle again. Music is a continuous stream:
two `bufferCmd`s are queued ahead, and each `callBackCmd` queues the next piece,
so the channel never runs dry.

There is no mixing, panning, pitch shifting, envelope, or per-sound volume
anywhere in the game. The only volume control is the machine's system output
volume, which Glider PRO writes directly (section 8).

## 2. The Sound Manager surface the game touches

### 2.1 Every Toolbox audio call in the tree

| Call | Where | Purpose |
|---|---|---|
| `SndNewChannel(&chan, sampledSynth, initNoInterp+initMono, upp)` | `Sound.c:378`, `Sound.c:386`, `Sound.c:394`, `Music.c:286` | create the 4 channels |
| `SndDisposeChannel(chan, true)` | `Sound.c:415`, `Sound.c:419`, `Sound.c:423`, `Music.c:302` | destroy them; `true` = quiet now, do not drain |
| `SndDoImmediate(chan, &cmd)` | `Sound.c:99,103,111,115,123,127,150,178,203`, `Music.c:112,117` | bypass the queue: `bufferCmd`, `quietCmd`, `flushCmd` |
| `SndDoCommand(chan, &cmd, true)` | `Sound.c:155,183,208` | queue `callBackCmd` after the buffer; `true` = "noWait" |
| `SndDoCommand(chan, &cmd, false)` | `Music.c:62,69,81,88,208,213` | queue music buffers/callbacks, blocking if the queue is full |
| `NewSndCallBackProc(fn)` | `Sound.c:369-371`, `Music.c:278` | build the UPP for the interrupt-time callback |
| `DisposeSndCallBackUPP(upp)` | `Sound.c:429-431`, `Music.c:305` | release it |
| `GetDefaultOutputVolume(&long)` | `Utilities.c:750` | read the *system* volume |
| `SetDefaultOutputVolume(long)` | `Utilities.c:783` | write the *system* volume |
| `GetResource('snd ', id)` | `Sound.c:279`, `Sound.c:327`, `Sound.c:501`, `Music.c:234`, `Music.c:396` | fetch a sound |
| `GetHandleSize` / `GetMaxResourceSize` | `Sound.c:286,332,507`, `Music.c:239,402` | size it |
| `SetResLoad(false/true)` | `Sound.c:498,504,510`, `Music.c:393,399,405` | size resources without loading them |
| `HLock` / `HUnlock` / `ReleaseResource` | `Sound.c:288,331,334,339,341,291,297`, `Music.c:238,240,246,248` | lock while copying, then let go |
| `BlockMove(src, dst, n)` | `Sound.c:296,340`, `Music.c:247` | copy header+samples out of the resource |
| `NewPtr` / `DisposePtr` | `Sound.c:287,335,310,358`, `Music.c:242,265` | the permanent sample buffers |
| `SetCurrentA5()` / `SetA5()` | `Sound.c:154,182,207,223,228,239,244,255,260`, `Music.c:68,87,215` | 68k global-world switch (see 2.5) |

`SetSoundVol` / `GetSoundVol` (the pre-Sound-Manager-3 API) appear only inside
`#ifdef powerc` comments at `GliderPRO/Headers/Externs.h:385-389` and in dead
comments at `GliderPRO/Sources/Utilities.c:754,786`. The shipped build always
uses `GetDefaultOutputVolume` / `SetDefaultOutputVolume`.

### 2.2 `SndCommand`

```c
struct SndCommand {
    unsigned short  cmd;      // 2 bytes
    short           param1;   // 2 bytes
    long            param2;   // 4 bytes
};                            // 8 bytes, big-endian on disk
```

This is exactly the 8-byte record that appears inside a `'snd '` format-1
resource (section 3.1), which is why the game can hand the Sound Manager a
`bufferCmd` whose `param2` is a pointer while the resource on disk stores an
offset in the same field.

### 2.3 Command opcodes used

The numeric values come from Apple's `Sound.h`, which is **not** part of this
source tree - only `#include <Sound.h>` appears
(`GliderPRO/Sources/Sound.c:10`, `GliderPRO/Sources/Music.c:10`). Two of the
values are nevertheless independently confirmed by the shipped data: every
`'snd '` resource in the game stores the command word `0x8051`, which is
`bufferCmd (81 = 0x51) | dataOffsetFlag (0x8000)` (section 3.6).

| Constant | Value | Hex | Used where | Meaning as used here |
|---|---:|---|---|---|
| `nullCmd` | 0 | 0x0000 | `Music.c:66` (written as a literal `0`) | do nothing; used as a queue spacer |
| `quietCmd` | 3 | 0x0003 | `Sound.c:96,108,120`, `Music.c:114` | stop the sound now |
| `flushCmd` | 4 | 0x0004 | `Sound.c:100,112,124`, `Music.c:109` | discard everything queued |
| `callBackCmd` | 13 | 0x000D | `Sound.c:152,180,205`, `Music.c:85,210` | invoke the channel's callback when reached |
| `soundCmd` | 80 | 0x0050 | not used by the game; recognised by `tools/probe_snd.py` | install a sound header |
| `bufferCmd` | 81 | 0x0051 | `Sound.c:147,175,200`, `Music.c:59,78,205` | play the sampled sound at `param2` |
| `dataOffsetFlag` | 32768 | 0x8000 | present in all 133 shipped resources | `param2` is an offset from the start of the resource, not a pointer |

`param1` is always 0 in every command the game issues, except for the spacer at
`GliderPRO/Sources/Music.c:67`, which sets `param1 = 1964` (a signature value;
`nullCmd` ignores both parameters).

### 2.4 Channel creation flags

`SndNewChannel(&chan, sampledSynth, initNoInterp + initMono, upp)`:

These values are from Apple's `Sound.h`, which is **not** in this tree (only
`#include <Sound.h>` at `GliderPRO/Sources/Sound.c:10` and
`GliderPRO/Sources/Music.c:10`); they cannot be verified from the source in front
of us. `sampledSynth = 5` *is* independently confirmed by the shipped data:
every resource stores `synthID = 5` (section 3.8).

| Constant | Value | Hex | Effect |
|---|---:|---|---|
| `squareWaveSynth` | 1 | 0x0001 | not used |
| `waveTableSynth` | 3 | 0x0003 | not used |
| `sampledSynth` | 5 | 0x0005 | the sampled-sound synthesiser - confirmed by `synthID = 5` in all 133 resources |
| `initNoInterp` | 4 | 0x0004 | do **not** interpolate when resampling |
| `initSRate22k` | 32 | 0x0020 | sample rate hint, 22 kHz |
| `initMono` | 128 | 0x0080 | monophonic output |
| `initStereo` | 192 | 0x00C0 | not used |
| `initMACE3` | 768 | 0x0300 | not used by the app |
| `initMACE6` | 1024 | 0x0400 | not used by the app; **is** set in 5 house resources (section 6.5) |

`initNoInterp + initMono` = 0x0004 + 0x0080 = **0x0084 = 132**, which is the
`init` argument passed to all four `SndNewChannel` calls.

The `initOption` stored *inside* each resource is a different, unrelated value:
`0x000000A0` for every application resource and every uncompressed house
resource, `0x000004A0` for the five MACE-compressed house resources (section
3.8, section 6.5). 0xA0 = 0x80 | 0x20 = `initMono | initSRate22k`, and
0x4A0 = 0x400 | 0xA0 adds `initMACE6`. **The game never reads this field** - it
would only be used by `SndPlay`, which Glider PRO does not call; the game passes
its own hard-coded `initNoInterp + initMono` to `SndNewChannel` and then feeds
the channel raw `bufferCmd`s. The port can ignore `initOption` entirely except as
a cross-check that a resource is MACE-compressed.

The observable behaviour to reproduce is therefore: **mono, no interpolation**,
i.e. nearest-neighbour (sample-and-hold) resampling from 22254.5455 Hz to the
hardware rate.

For a faithful Go port this means: when resampling the shipped 22254.5455 Hz
samples up to 44100 or 48000 Hz, the original produced *sample-and-hold*
(zero-order-hold) output, which is audibly brighter/grittier than linear
interpolation. A port that wants bit-accurate character should use
nearest-neighbour; a port that wants "nicer" audio can use linear, and should
document the deviation.

### 2.5 Interrupt-time callbacks and the 68k A5 world

`callBackCmd` invokes the channel's callback **at interrupt time**, not on the
main thread. On 68k Macs, application globals are addressed off register A5,
which is not guaranteed to be set correctly inside an interrupt, so the idiom
is: pass `SetCurrentA5()` in `param2` when queueing the command, then
`SetA5(param2)` on entry to the callback and `SetA5(saved)` on exit.

`Sound.c` does this correctly for all three effect channels:

```c
gameA5 = theCommand->param2;      // Sound.c:222 / 238 / 254
thisA5 = SetA5(gameA5);           // Sound.c:223 / 239 / 255
priority0 = 0;                    // Sound.c:225 / 241 / 257
soundPlaying0 = kNoSoundPlaying;  // Sound.c:226 / 242 / 258
thisA5 = SetA5(thisA5);           // Sound.c:228 / 244 / 260
```

`Music.c` does **not**: the two lines that would set up the world are commented
out at `GliderPRO/Sources/Music.c:176-177`, so `gameA5` is read uninitialised at
`GliderPRO/Sources/Music.c:212` (it is stored into the next `callBackCmd`'s
`param2`) and `thisA5` is read uninitialised at
`GliderPRO/Sources/Music.c:215`. On a PowerPC build (which is what shipped -
`SetA5`/`SetCurrentA5` are no-ops returning 0 under CFM) this is harmless. A Go
port should simply drop the whole concept.

The important consequence for the port is *concurrency*, not registers: the
callback mutates `priority0/1/2` and `soundPlaying0/1/2`
(`GliderPRO/Sources/Sound.c:225-226,241-242,257-258`) and `musicCursor` /
`musicSoundID` (`GliderPRO/Sources/Music.c:182-202`) asynchronously with respect
to the game loop that reads them in `PlayPrioritySound`
(`GliderPRO/Sources/Sound.c:54-66`). In Go these are shared mutable state
between the audio goroutine and the game goroutine and need a mutex or a channel
hand-off. The original had no synchronisation at all; the resulting races are
benign because the values are single 16-bit words and the worst case is one
mis-chosen voice.

### 2.6 What a Go port replaces each concept with

| Mac concept | Go replacement |
|---|---|
| `SndChannelPtr` x 4 | 4 logical voices in your own mixer; the OS gets one stereo stream |
| `sampledSynth` | nothing; you only ever play PCM |
| `initMono` | mix all 4 voices to mono, then duplicate to L/R (the original produced identical L/R) |
| `initNoInterp` | nearest-neighbour resampler |
| `bufferCmd` via `SndDoImmediate` | "voice N: stop whatever you were doing, start buffer X at offset 0" |
| `flushCmd` + `quietCmd` | "voice N: clear pending, silence now" |
| `callBackCmd` queued behind a buffer | "when voice N reaches end-of-buffer, run f()" - i.e. an end-of-stream event from the mixer |
| Resource Manager `'snd '` | embedded PCM assets (section 11) |
| `GetDefaultOutputVolume` / `SetDefaultOutputVolume` | a process-local gain 0..1; do **not** change the user's system volume (section 8) |
| 68k A5 world switching | delete |
| Purgeable resource handles, `NewPtr`, `HLock` | Go slices; no manual memory management |
| Big-endian on-disk data | `encoding/binary.BigEndian` when parsing original resources |

## 3. The `'snd '` resource format, verified byte-for-byte

### 3.1 Format 1 wrapper

All big-endian. Offsets are from the first byte of the resource.

| Offset | Type | Field | Notes |
|---:|---|---|---|
| +0 | `int16` | `format` | 1 for "format 1"; 2 for the obsolete HyperCard format |
| +2 | `int16` | `numSynths` | number of synthesiser records that follow |
| +4 | `int16` | `synthID` | repeated `numSynths` times, 6 bytes each |
| +6 | `uint32` | `initOption` | ... |
| +4+6n | `int16` | `numCmds` | number of `SndCommand` records that follow |
| +6+6n | `uint16` | `cmd` | repeated `numCmds` times, 8 bytes each |
| +8+6n | `int16` | `param1` | ... |
| +10+6n | `int32` | `param2` | ... |

For a format 2 resource the header is instead
`{ int16 format; int16 refCount; int16 numCmds; SndCommand cmds[numCmds] }`.
`tools/probe_snd.py:117-137` handles both; **no format 2 resource exists in this
game** (all 133 are format 1).

With `numSynths = 1` and `numCmds = 1` - which is what every Glider PRO resource
has - the command block ends at byte 20, and the sampled-sound header starts
there. That is the origin of the magic `20L` in the C.

### 3.2 `SoundHeader` ("stdSH", `encode = 0x00`), 22 bytes

| Offset | Size | Type | Field | Meaning |
|---:|---:|---|---|---|
| +0 | 4 | `uint32` | `samplePtr` | 0 = the samples immediately follow this header |
| +4 | 4 | `uint32` | `length` | number of sample **frames**; 1 byte per frame |
| +8 | 4 | `uint32` | `sampleRate` | unsigned 16.16 fixed point, Hz |
| +12 | 4 | `uint32` | `loopStart` | frame index |
| +16 | 4 | `uint32` | `loopEnd` | frame index |
| +20 | 1 | `uint8` | `encode` | 0x00 stdSH / 0xFE cmpSH / 0xFF extSH |
| +21 | 1 | `uint8` | `baseFrequency` | MIDI note number of the recorded pitch; 60 = middle C |
| +22 | `length` | `uint8[]` | `sampleArea` | **unsigned** 8-bit PCM, offset binary, 0x80 = silence |

### 3.3 `CmpSoundHeader` ("cmpSH", `encode = 0xFE`), 64 bytes

Note that the first 22 bytes are laid out the same but two fields change
meaning: `+4` becomes `numChannels`, and the frame count moves to `+22`.

| Offset | Size | Type | Field | Meaning |
|---:|---:|---|---|---|
| +0 | 4 | `uint32` | `samplePtr` | 0 = packets follow the header |
| +4 | 4 | `uint32` | `numChannels` | 1 for every resource in this game |
| +8 | 4 | `uint32` | `sampleRate` | 16.16 fixed |
| +12 | 4 | `uint32` | `loopStart` | |
| +16 | 4 | `uint32` | `loopEnd` | |
| +20 | 1 | `uint8` | `encode` | 0xFE |
| +21 | 1 | `uint8` | `baseFrequency` | |
| +22 | 4 | `uint32` | `numFrames` | number of **packets** per channel |
| +26 | 10 | `extended80` | `AIFFSampleRate` | 80-bit IEEE extended, same rate again |
| +36 | 4 | `uint32` | `markerChunk` | 0 |
| +40 | 4 | `OSType` | `format` | 0 in this game (would be `'MAC6'`) |
| +44 | 4 | `uint32` | `futureUse2` | 0 |
| +48 | 4 | `uint32` | `stateVars` | 0 |
| +52 | 4 | `uint32` | `leftOverSamples` | 0 |
| +56 | 2 | `int16` | `compressionID` | 3 = MACE 3:1, 4 = MACE 6:1 |
| +58 | 2 | `uint16` | `packetSize` | bits per packet: 16 for MACE 3:1, 8 for MACE 6:1 |
| +60 | 2 | `uint16` | `snthID` | 13 observed |
| +62 | 2 | `uint16` | `sampleSize` | 8 |
| +64 | | `uint8[]` | `sampleArea` | `numFrames * numChannels * packetSize/8` bytes |

MACE always produces **6 samples per packet**; the compression ratio comes from
the packet size (1 byte -> 6 samples = 6:1; 2 bytes -> 6 samples = 3:1). This is
implemented at `tools/probe_snd.py:178-180`.

`compressionID` values, for completeness:

| Value | Constant | Meaning |
|---:|---|---|
| -2 | `variableCompression` | |
| -1 | `fixedCompression` | |
| 0 | `notCompressed` | |
| 1 | `twoToOne` | |
| 2 | `eightToThree` | |
| 3 | `threeToOne` | MACE 3:1 |
| 4 | `sixToOne` | MACE 6:1 - **the only compression present in this game** |

### 3.4 `ExtSoundHeader` ("extSH", `encode = 0xFF`), 64 bytes

Same first 22 bytes, then `numFrames` at +22, `AIFFSampleRate` at +26,
`markerChunk` at +36, `instrumentChunks` at +40, `AESRecording` at +44,
`sampleSize` (bits) at +48, `futureUse1..4`. This is the 16-bit / stereo
uncompressed form. `tools/probe_snd.py:181-189` parses it. **No extSH resource
exists in this game** - it is supported only so the tool does not silently
mis-parse a third-party house.

### 3.5 Fixed-point sample rates

`sampleRate` is `UnsignedFixed`: a 32-bit value interpreted as Hz with 16
fractional bits, so `Hz = value / 65536.0` (`tools/probe_snd.py:93-95`).

Rates that actually occur in this game:

| Fixed | Decimal Hz | Classic constant | Where |
|---|---:|---|---|
| 0x56EE8BA3 | 22254.5455 | `rate22khz` | all 70 application resources; 47 of 63 house resources |
| 0x56EE8B9F | 22254.5454 | (0x56EE8BA3 - 4) | 2 house resources ("Blackbird") |
| 0x56EF0000 | 22255.0 | (hand-rounded) | 1 house resource (Grand Prix "V8 startup") |
| 0x56220000 | 22050.0 | `rate22050hz` | 1 house resource (Grand Prix "Goodbye!") |
| 0x2B7745D1 | 11127.2727 | `rate11khz` | 6 house resources |
| 0x2B778000 | 11127.5 | (hand-rounded) | 1 house resource (Rainbow's End) |
| 0x26330000 | 9779.0 | - | 1 house resource (SpacePods "Organ") |
| 0x1CFA2E8B | 7418.1818 | `rate22khz / 3` | 3 house resources |
| 0x15BBA2E8 | 5563.6364 | `rate22khz / 4` | 1 house resource (Leviathan "Ricochet") |

22254.5455 Hz is the classic Macintosh sound hardware rate (the original Mac
sound circuit ran at 22.25454... kHz = 783360/35.2 ; Apple's constant
`rate22khz` is exactly 0x56EE8BA3). Every sound John Calhoun recorded for the
application is at that rate; some third-party house sounds are at halves,
thirds and quarters of it.

### 3.6 Walkthrough of a real resource: `'snd '` 1013 "Tok"

Produced by `python3 tools/probe_snd.py show 1013`:

```
resource      'snd ' 1013 'Tok'
resource size 494 bytes
first 48 bytes:
  0000  00 01 00 01 00 05 00 00 00 A0 00 01 80 51 00 00  .............Q..
  0010  00 00 00 14 00 00 00 00 00 00 01 C4 56 EE 8B A3  ............V...
  0020  00 00 01 C2 00 00 01 C3 00 3C 50 50 84 9B A5 99  .........<PP....

format            1
numSynths         1
  synth           5 (sampledSynth)  initOption 0x000000A0
numCmds           1
  cmd 0x8051      bufferCmd | dataOffsetFlag param1=0 param2=20

sound header at   +20
  samplePtr       0x00000000 (samples follow header)
  encode          0x00 (stdSH)
  sampleRate      0x56EE8BA3 = 22254.5455 Hz  [rate22khz]
  baseFrequency   60
  loopStart       450
  loopEnd         451
  length          452 sample frames
  header+samples  22 + 452 = 494 (resource size 494)
  duration        0.0203 s (20.3 ms)
  u8 PCM min/max/mean  80 / 175 / 135.18  (0x80 = silence)
  first 16 samples     [80, 80, 132, 155, 165, 153, 130, 114, 114, 130, 152, 162, 159, 158, 158, 162]
  last 8 samples       [82, 87, 94, 101, 104, 103, 101, 100]
  no loop (degenerate loopStart=length-2, loopEnd=length-1)
```

Byte-by-byte:

| Bytes | Value | Field |
|---|---|---|
| `00 01` | 1 | `format` |
| `00 01` | 1 | `numSynths` |
| `00 05` | 5 | `synthID` = `sampledSynth` |
| `00 00 00 A0` | 0xA0 | `initOption` = `initMono | initSRate22k` |
| `00 01` | 1 | `numCmds` |
| `80 51` | 0x8051 | `cmd` = `bufferCmd | dataOffsetFlag` |
| `00 00` | 0 | `param1` |
| `00 00 00 14` | 20 | `param2` = offset of the sound header |
| `00 00 00 00` | 0 | `samplePtr` |
| `00 00 01 C4` | 452 | `length` |
| `56 EE 8B A3` | 0x56EE8BA3 | `sampleRate` = 22254.5455 Hz |
| `00 00 01 C2` | 450 | `loopStart` |
| `00 00 01 C3` | 451 | `loopEnd` |
| `00` | 0x00 | `encode` = stdSH |
| `3C` | 60 | `baseFrequency` = middle C |
| `50 50 84 9B ...` | 80, 80, 132, 155 | first PCM samples |

The `0x8051` word is the empirical proof that `bufferCmd = 81` and
`dataOffsetFlag = 0x8000`.

### 3.7 The `-20L` / `+20L` invariant

Both loaders do the same thing:

```c
soundDataSize = GetHandleSize(theSound) - 20L;              // Sound.c:286, 332; Music.c:239
theSoundData[i] = NewPtr(soundDataSize);                    // Sound.c:287, 335; Music.c:242
BlockMove((Ptr)(*theSound + 20L), theSoundData[i], soundDataSize);  // Sound.c:296, 340; Music.c:247
```

i.e. **skip the 20-byte format-1 wrapper and keep the sampled-sound header plus
all of the samples**. The resulting pointer is exactly what `bufferCmd` wants in
`param2` when the flag is *not* set: a pointer to a `SoundHeader`.

This is a hard-coded assumption that the wrapper is exactly 20 bytes, i.e.
`format = 1`, `numSynths = 1`, `numCmds = 1`. It is true of all 70 application
resources and all 63 house resources (verified below). A house authored with a
different tool - two commands, or format 2 - would make the game feed the Sound
Manager garbage. There is no validation anywhere.

### 3.8 Observed uniformity of the 70 application resources

Every single one of the 70 `'snd '` resources in `GliderPRO/Glider PRO.r` shares
*identical* values in every header field:

| Field | Value (all 70) |
|---|---|
| `format` | 1 |
| `numSynths` | 1 |
| `synthID` | 5 (`sampledSynth`) |
| `initOption` | 0x000000A0 |
| `numCmds` | 1 |
| `cmd` | 0x8051 (`bufferCmd | dataOffsetFlag`) |
| `param1` | 0 |
| `param2` | 20 |
| sound header offset | +20 |
| `samplePtr` | 0 |
| `sampleRate` | 0x56EE8BA3 = 22254.5455 Hz |
| `encode` | 0x00 (stdSH) |
| `baseFrequency` | 60 |
| `resource size - 42 - length` | 0 (no slack, no padding) |

The last row is the key invariant: **`resource size == 42 + length`** for all 70,
where 42 = 20-byte wrapper + 22-byte `SoundHeader`. The buffer the game keeps in
`theSoundData[i]` is therefore `22 + length` bytes.

Totals: 70 resources, 1,129,344 resource bytes, of which 1,126,404 bytes are
PCM. Split: 63 sound effects (ids 1000..1062) = 352,348 resource bytes; 7 music
pieces (ids 2000..2006) = 776,996 resource bytes. After the `-20` strip the game
holds 351,088 + 776,856 = **1,127,944 bytes** of `NewPtr` memory for audio.

### 3.9 PCM encoding

- 8 bits per sample, **unsigned offset binary**: 0x00 is full negative, 0x80 is
  silence, 0xFF is full positive.
- Mono, one channel.
- Over all 1,126,404 application PCM bytes: min 0, max 255, mean 126.21. So the
  full 8-bit range is used and the DC centre is (as expected) just under 128.
- `'snd '` 1018 "Thrust" begins with thirteen bytes of exactly 128 - digital
  silence - which is consistent with it being a hand-trimmed loop.

Conversion to signed 16-bit for a modern mixer:

```
s16 = (int16(u8) - 128) * 256          // or << 8
```

To signed 8-bit: `s8 = int8(u8 ^ 0x80)`, i.e. `int8(u8) - 128` with wraparound.

### 3.10 Loop points, and whether they matter

Only 11 of the 70 application resources carry meaningful loop points:

| `snd ` ID | Name | length | loopStart | loopEnd |
|---:|---|---:|---:|---:|
| 1018 | Thrust | 5759 | 0 | 5758 |
| 1026 | Shred | 1934 | 0 | 1933 |
| 1047 | Sizzle | 2240 | 0 | 2239 |
| 1062 | Hiss | 2960 | 0 | 2959 |
| 2000 | Refrain1.22 | 97034 | 0 | 97034 |
| 2001 | Refrain2.22 | 97102 | 0 | 97102 |
| 2002 | Refrain3.22 | 97032 | 0 | 97032 |
| 2003 | Refrain4.22 | 97284 | 0 | 97284 |
| 2004 | Chorus.22 | 194028 | 0 | 194028 |
| 2005 | RefrainSparse1.22 | 96888 | 0 | 96888 |
| 2006 | RefrainSparse2.22 | 97334 | 0 | 97334 |

(the four sound effects use `loopEnd = length - 1`, the
seven music pieces use `loopEnd = length`.)

The other 59 resources all carry the *degenerate* marker
`loopStart = length - 2`, `loopEnd = length - 1`, which is how the Mac sound
tools of the era wrote "no loop".

**These loop points are never honoured.** `bufferCmd` plays a buffer exactly
once and then moves on; only `soundCmd` + `freqDurationCmd` (which the game
never issues) consults `loopStart`/`loopEnd`. Two independent internal proofs:

1. If `bufferCmd` looped, the `callBackCmd` queued behind it
   (`GliderPRO/Sources/Sound.c:152-155`) would never fire, so `priority0` would
   never return to 0 and channel 0 would be permanently busy after the first
   sound. The game plainly works.
2. The music engine depends on each piece *ending*: `MusicCallBack` is what
   queues the next piece (`GliderPRO/Sources/Music.c:205-213`). If `Refrain1.22`
   looped, the score would never advance.

The looping *effect* is instead produced by the game retriggering the sound from
scratch every few frames - see section 4.14. A Go port must therefore treat all
`'snd '` buffers as one-shot and ignore `loopStart`/`loopEnd` entirely. (Keep
them in the manifest anyway; they are the only record of the author's intent, and
a "nicer" port might want to use them.)

## 4. The sound-effect engine (`GliderPRO/Sources/Sound.c`)

### 4.1 File-local constants

| Constant | Value | Line | Meaning |
|---|---:|---|---|
| `kBaseBufferSoundID` | 1000 | `GliderPRO/Sources/Sound.c:14` | resource id of sound 0; sound *n* is `'snd '` *n*+1000 |
| `kMaxSounds` | 64 | `GliderPRO/Sources/Sound.c:15` | size of `theSoundData[]`; slots 0..62 are the built-ins, slot 63 is the per-room trigger sound |
| `kNoSoundPlaying` | -1 | `GliderPRO/Sources/Sound.c:16` | sentinel for `soundPlayingN` |
| `kNoMemForSoundsAlert` | 1039 | `GliderPRO/Sources/Sound.c:518` | `ALRT` id, declared inside `TellHerNoSounds()` |
| `kNoSoundManager3Alert` | 1030 | `GliderPRO/Sources/Sound.c:529` | `ALRT` id, declared inside `BitchAboutSM3()` |

Note the off-by-design: `kMaxSounds` is 64 but only **63** built-in sounds are
loaded, because `LoadBufferSounds()` iterates `i < kMaxSounds - 1`
(`GliderPRO/Sources/Sound.c:325`). Resource ids 1000..1062 inclusive. There is
no `'snd '` 1063 in the application.

### 4.2 Globals (all module-level, all shared with interrupt-time callbacks)

```c
SndCallBackUPP  callBack0UPP, callBack1UPP, callBack2UPP;   // Sound.c:28
SndChannelPtr   channel0, channel1, channel2;               // Sound.c:29
Ptr             theSoundData[kMaxSounds];                   // Sound.c:30
short           numSoundsLoaded, priority0, priority1, priority2;   // Sound.c:31
short           soundPlaying0, soundPlaying1, soundPlaying2;        // Sound.c:32
Boolean         soundLoaded[kMaxSounds], dontLoadSounds;            // Sound.c:33
Boolean         channelOpen, isSoundOn, failedSound;                // Sound.c:34
```

| Global | Type | Meaning | Written by | Read by |
|---|---|---|---|---|
| `theSoundData[i]` | `Ptr` | 22-byte `SoundHeader` + PCM, malloc'd | `LoadBufferSounds`, `LoadTriggerSound`, `Dump*` | `PlaySound0/1/2` |
| `priority0/1/2` | `short` | priority of the sound currently in voice *n*; 0 = idle | `PlaySoundN` (main thread), `CallBackN` (interrupt) | `PlayPrioritySound`, `FlushAnyTriggerPlaying` |
| `soundPlaying0/1/2` | `short` | sound index currently in voice *n*, or -1 | same | **nobody** - write-only diagnostics |
| `channelOpen` | `Boolean` | at least one channel was created | `OpenSoundChannels`, `CloseSoundChannels` | both |
| `isSoundOn` | `Boolean` | user volume != 0 | `Main.c:197-200` (and `:147` on first run), `Settings.c:635`, `:790` | `PlaySound0/1/2`, `Music.c` (extern at `Music.c:38`, never read there) |
| `failedSound` | `Boolean` | channel creation or resource load failed; audio permanently off | `InitSound` | every entry point |
| `dontLoadSounds` | `Boolean` | not enough RAM at launch; audio never even attempted | `Environ.c:572`, `Environ.c:673` | every entry point |
| `numSoundsLoaded` | `short` | **dead** - declared, never read or written anywhere in the tree |
| `soundLoaded[64]` | `Boolean[]` | **dead** - declared, never read or written anywhere in the tree |

`soundLoaded[]` and `numSoundsLoaded` are vestigial: `grep -a -w` over the whole
`Sources/` + `Headers/` tree finds them only at `GliderPRO/Sources/Sound.c:31`
and `GliderPRO/Sources/Sound.c:33`. A Go port should not reproduce them.

### 4.3 How many channels, and why three

Exactly **three** sound-effect voices, created once in `OpenSoundChannels()`
(`GliderPRO/Sources/Sound.c:365-401`), plus one music voice. All are `SndChannelPtr`s
handed to the Sound Manager's own mixer; the game never mixes samples itself.

Every voice is identical - there is no "left channel", no dedicated
music-vs-effects split beyond the separate `musicChannel`, and no notion of a
voice being better suited to one sound than another. The three exist purely so
up to three effects can overlap.

### 4.4 `PlayPrioritySound(which, priority)` - the allocation policy

The single entry point used by the rest of the game: 167 call sites.
`GliderPRO/Sources/Sound.c:40-85`, prototyped
`void PlayPrioritySound (SInt16, SInt16);` at `GliderPRO/Headers/GliderProtos.h:447`.

```
 1  PlayPrioritySound(which, priority):
 2      if failedSound or dontLoadSounds:                 // Sound.c:44
 3          return                                        // audio disabled, silently
 4
 5      // the trigger sound is allowed at most one voice at a time
 6      if priority == kTriggerPriority (999)             // Sound.c:47-50
 7         and (priority0 == 999 or priority1 == 999 or priority2 == 999):
 8          return                                        // Sound.c:51
 9
10      whosLowest     = 0                                // Sound.c:53
11      lowestPriority = priority0                        // Sound.c:54
12
13      if priority1 < lowestPriority:                    // Sound.c:56
14          lowestPriority = priority1;  whosLowest = 1   // Sound.c:58-59
15
16      if priority2 < lowestPriority:                    // Sound.c:62
17          lowestPriority = priority2;  whosLowest = 2   // Sound.c:64-65
18
19      if priority >= lowestPriority:                    // Sound.c:68
20          switch whosLowest:                            // Sound.c:70
21              0: PlaySound0(which, priority)            // Sound.c:73
22              1: PlaySound1(which, priority)            // Sound.c:77
23              2: PlaySound2(which, priority)            // Sound.c:81
24      // else: the new sound is dropped entirely
```

Properties that must be reproduced exactly:

1. **Idle voices have priority 0** (set by `InitSound` at
   `GliderPRO/Sources/Sound.c:451-453` and by each callback at
   `GliderPRO/Sources/Sound.c:225`, `:241`, `:257`). Every game priority is
   >= 100 (the smallest is `kHitWallPriority` = 100, see section 5.2), so *any*
   idle voice always wins the `priority >= lowestPriority` test at
   `GliderPRO/Sources/Sound.c:68` and the sound plays.
2. **Ties go to the lowest-numbered voice.** The comparisons are strict `<`, so
   with all three idle (`0,0,0`) `whosLowest` stays 0 and channel 0 is used. With
   `priority0 == priority1 == priority2 == 300`, `whosLowest` is again 0 and
   channel 0 is stolen. Channel 0 therefore takes the overwhelming majority of
   sounds and channel 2 is only reached when the other two are both busy with
   strictly higher priorities.
3. **Stealing is unconditional and immediate.** `PlaySoundN` issues its
   `bufferCmd` with `SndDoImmediate` (`GliderPRO/Sources/Sound.c:150`, `:178`,
   `:203`), which in Sound Manager terms bypasses the queue and starts the new
   buffer at once, cutting off whatever was playing. There is no fade and no
   "let it finish" path.
4. **`priority >= lowestPriority` uses `>=`, not `>`.** A sound of priority *p*
   will interrupt an in-flight sound of the same priority *p*. In practice this
   is what makes rapid repeats of the same effect (e.g. `kShredSound`, retriggered
   every frame) work at all.
5. **The trigger sound is capped at one voice**
   (`GliderPRO/Sources/Sound.c:47-51`). `kTriggerPriority` = 999
   (`GliderPRO/Headers/GliderDefines.h:180`) is above every other priority (the
   next highest is `kTransOutPriority` = 906), so without the cap a player
   standing on a sound trigger would stomp all three voices. Note the guard
   tests only
   `priority == kTriggerPriority`; the *trigger sound already playing* can still
   be evicted by... nothing, because nothing else reaches 999. So once a trigger
   sound starts, its voice is locked for the sound's full duration.
6. **A dropped sound is dropped, not queued.** There is no retry, no waiting
   list. If all three voices hold priorities strictly above the requested one,
   the call is a no-op.

Worked examples (state written as `(priority0, priority1, priority2)`):

| State before | Call | `whosLowest` / `lowestPriority` | Result | State after |
|---|---|---|---|---|
| (0,0,0) | `kHitWallSound` @ 100 | 0 / 0 | plays on ch0 | (100,0,0) |
| (100,0,0) | `kHitWallSound` @ 100 | 1 / 0 | plays on ch1 | (100,100,0) |
| (100,100,0) | `kHitWallSound` @ 100 | 2 / 0 | plays on ch2 | (100,100,100) |
| (100,100,100) | `kHitWallSound` @ 100 | 0 / 100 | 100 >= 100, **steals ch0** | (100,100,100) |
| (300,300,300) | `kHitWallSound` @ 100 | 0 / 300 | 100 < 300, **dropped** | (300,300,300) |
| (300,906,413) | `kBonusSound` @ 812 | 0 / 300 | 812 >= 300, steals ch0 | (812,906,413) |
| (900,906,902) | `kTikSound` @ 200 | 0 / 900 | 200 < 900, **dropped** | (900,906,902) |
| (999,100,100) | `kTriggerSound` @ 999 | - | early return at `Sound.c:51` | (999,100,100) |
| (999,100,100) | `kHitWallSound` @ 100 | 1 / 100 | steals ch1 | (999,100,100) |

### 4.5 `PlaySound0` / `PlaySound1` / `PlaySound2`

`GliderPRO/Sources/Sound.c:133-157`, `:161-185`, `:189-213`. All three are the
same six statements against a different channel. Taking channel 0:

```
 1  PlaySound0(soundID, priority):
 2      if failedSound or dontLoadSounds: return           // Sound.c:138-139
 3      if not isSoundOn: return                           // Sound.c:142 (implicit)
 4
 5      priority0     = priority                           // Sound.c:144
 6      soundPlaying0 = soundID                            // Sound.c:145
 7
 8      cmd    = bufferCmd (81)                            // Sound.c:147
 9      param1 = 0                                         // Sound.c:148
10      param2 = (long)theSoundData[soundID]               // Sound.c:149  raw pointer
11      SndDoImmediate(channel0, &cmd)                     // Sound.c:150
12
13      cmd    = callBackCmd (13)                          // Sound.c:152
14      param1 = 0                                         // Sound.c:153
15      param2 = SetCurrentA5()                            // Sound.c:154  68k A5 world
16      SndDoCommand(channel0, &cmd, true)                 // Sound.c:155  queued
```

Notes:

- `param2` of the `bufferCmd` is a **raw pointer** and the `dataOffsetFlag` is
  *not* set. That is why the loaders strip the 20-byte wrapper: the Sound Manager
  is handed a bare `SoundHeader*`.
- `param1` = 0 for `bufferCmd` means "play the whole buffer at its natural rate".
  No rate override, no pitch bend, ever.
- The `callBackCmd` is queued (`SndDoCommand`) rather than immediate, so it fires
  when the buffer finishes. `noWait = true` means "return `queueFull` rather than
  block if the queue is full".
- `theErr` is assigned from every call and then discarded. Sound-command failures
  are silently ignored.
- **`isSoundOn` is checked here, not in `PlayPrioritySound`.** So when the volume
  is 0, `PlayPrioritySound` still runs the full allocation algorithm and (because
  `PlaySoundN` returns before writing `priorityN`) leaves the priorities
  untouched at 0. Harmless, but it means the priority state machine is inert
  while muted.

**The one real asymmetry**: `PlaySound0` and `PlaySound1` set
`priorityN`/`soundPlayingN` *before* issuing the commands
(`GliderPRO/Sources/Sound.c:144-145`, `:172-173`), but **`PlaySound2` sets them
*after*** (`GliderPRO/Sources/Sound.c:210-211`). This is a latent race: on a real
Mac the `bufferCmd` for a very short sound could complete and `CallBack2` could
fire (setting `priority2 = 0`, `soundPlaying2 = -1`) *before* line 210 runs,
after which `priority2` is left stuck at the finished sound's priority and
channel 2 looks permanently busy until something else steals it. The window is
tiny and the shortest sound is 20.3 ms (`snd ` 1013 "Tok", 452 frames), far
longer than the few instructions involved, so it almost certainly never bit
anyone. A Go port
should set the state before issuing the buffer in all three voices (i.e. fix it),
and must in any case guard these variables with a mutex - see section 12.

### 4.6 `CallBack0` / `CallBack1` / `CallBack2`

`GliderPRO/Sources/Sound.c:217-229`, `:233-245`, `:249-261`. Declared
`pascal void CallBackN (SndChannelPtr, SndCommand *)`
(`GliderPRO/Sources/Sound.c:19-21`) and wrapped with `NewSndCallBackProc`
(`GliderPRO/Sources/Sound.c:369-371`).

```
 1  CallBack0(theChannel, theCommand):     // theChannel #pragma unused (Sound.c:219)
 2      gameA5 = theCommand->param2        // Sound.c:222  the A5 stashed by PlaySound0
 3      thisA5 = SetA5(gameA5)             // Sound.c:223  install the app's globals
 4      priority0     = 0                  // Sound.c:225
 5      soundPlaying0 = kNoSoundPlaying    // Sound.c:226
 6      thisA5 = SetA5(thisA5)             // Sound.c:228  restore
```

That is the entire completion path: mark the voice idle. There is no chaining, no
"next sound", no bookkeeping. The `SetA5` dance exists because on 68k Macs a
Sound Manager callback runs at interrupt time with an arbitrary A5 register, so
the application's globals (`priority0` and friends, which are A5-relative) are
not addressable until A5 is restored. `SetCurrentA5()` in `PlaySound0` captures
it; `param2` ferries it into the interrupt. **On PowerPC this is a no-op** and in
Go it is meaningless - see section 2.6.

### 4.7 `FlushAnyTriggerPlaying()`

`GliderPRO/Sources/Sound.c:89-129`. For each voice *n* in 0,1,2, if
`priorityN == kTriggerPriority`, send two immediate commands to that channel:

```
 1  if priority0 == kTriggerPriority (999):     // Sound.c:94
 2      quietCmd (3),  param1=0, param2=0  -> SndDoImmediate(channel0)  // Sound.c:96-99
 3      flushCmd (4),  param1=0, param2=0  -> SndDoImmediate(channel0)  // Sound.c:100-103
 4  ... same for channel1 (Sound.c:106-116) and channel2 (Sound.c:118-128)
```

`quietCmd` (3) stops the currently playing sound; `flushCmd` (4) empties the
channel's command queue. Per the Sound Manager semantics this pair is the
standard "stop now and forget everything pending" idiom.

**Important consequence, and the one claim here I cannot verify from this tree:**
if `flushCmd` removes the queued `callBackCmd`, then `CallBackN` never fires and
`priorityN` is *not* reset to 0 by this path - the voice stays marked busy at
priority 999. Nothing else in the program ever clears it (`InitSound` runs once
per launch). So the first time the player leaves a room *while its custom trigger
sound is still audible*, one of the three voices is bricked at 999 for the rest of
the session: no other sound can ever be the "lowest priority" on it, and because
of the guard at `GliderPRO/Sources/Sound.c:47-51` no further trigger sound will
ever play either. If instead the Sound Manager delivers flushed callbacks (or
`quietCmd` itself completes the buffer and fires the callback before `flushCmd`
arrives), the voice recovers and there is no bug. Apple's `Sound.h` is not in this
source tree and the machine is airgapped, so this cannot be settled here - it is
listed in `## Open questions`. A Go port should simply reset the voice's priority
to 0 explicitly when it stops it, which is correct under either reading.

The only caller is `DrawLocale()` (`GliderPRO/Sources/RoomGraphics.c:57`),
immediately before `DumpTriggerSound()` (`GliderPRO/Sources/RoomGraphics.c:58`)
and eventually `ListAllLocalObjects()` (`GliderPRO/Sources/RoomGraphics.c:72`)
which reloads the new room's trigger sound. So the sequence on every room change
is: silence the old custom sound, free the buffer it was playing out of, load the
new one. The flush is *necessary* - `DumpTriggerSound` calls `DisposePtr` on
`theSoundData[63]` while the Sound Manager may still be reading samples out of
it, so without the `quietCmd` the mixer would read freed memory.

Also note the commented-out `FlushAnyTriggerPlaying()` at
`GliderPRO/Sources/Sound.c:275`, inside `LoadTriggerSound` - the author moved the
flush out to `DrawLocale`.

### 4.8 `LoadBufferSounds()` - loading the 63 built-in effects

`GliderPRO/Sources/Sound.c:316-347`.

```
 1  theErr = noErr
 2  for i = 0 to kMaxSounds-2 (i.e. 0..62):                  // Sound.c:325
 3      theSound = GetResource('snd ', i + 1000)              // Sound.c:327
 4      if theSound == nil: return MemError()                 // Sound.c:328-329
 5      HLock(theSound)                                       // Sound.c:331
 6      soundDataSize = GetHandleSize(theSound) - 20          // Sound.c:332
 7      HUnlock(theSound)                                     // Sound.c:333
 8      theSoundData[i] = NewPtr(soundDataSize)               // Sound.c:335
 9      if theSoundData[i] == nil: return MemError()           // Sound.c:336-337
10      HLock(theSound)                                       // Sound.c:339
11      BlockMove(*theSound + 20, theSoundData[i], soundDataSize)  // Sound.c:340
12      ReleaseResource(theSound)                             // Sound.c:341
13  theSoundData[63] = nil                                    // Sound.c:344
14  return noErr
```

Observations:

- `ReleaseResource` is called on a **locked** handle (`HLock` at :339 is never
  undone). Legal but sloppy; the handle is being disposed anyway.
- The `HLock`/`HUnlock` around `GetHandleSize` is pointless (`GetHandleSize`
  doesn't move memory).
- The gap between the `HUnlock` at :333 and the `HLock` at :339 spans a `NewPtr`,
  which *can* compact the heap and purge a purgeable handle. 54 of the 70
  application `'snd '` resources are purgeable (see section 5.1), so
  `*theSound` could in principle be nil at line 340. In practice
  `ReleaseResource` is called before the next iteration and the Mac heap wouldn't
  purge a just-loaded handle in the same allocation, so this never fired. A Go
  port has no purgeable handles and no such hazard.
- Early return on the *first* failure leaves `theSoundData[i..63]` as whatever
  they were (BSS zero on first call). `InitSound` then sets `failedSound = true`
  (`GliderPRO/Sources/Sound.c:462`) and every entry point bails out, so the
  partially-filled array is never indexed.

### 4.9 `LoadTriggerSound(soundID)` / `DumpTriggerSound()` - slot 63

`GliderPRO/Sources/Sound.c:265-303` and `:307-312`.

```
 1  LoadTriggerSound(soundID):
 2      if dontLoadSounds or theSoundData[63] != nil:      // Sound.c:271
 3          return -1                                      // Sound.c:272  already occupied
 4      theSound = GetResource('snd ', soundID)            // Sound.c:279
 5      if theSound == nil: return -1                      // Sound.c:280-282
 6      soundDataSize = GetHandleSize(theSound) - 20       // Sound.c:286
 7      theSoundData[63] = NewPtr(soundDataSize)           // Sound.c:287
 8      HLock(theSound)                                    // Sound.c:288
 9      if theSoundData[63] == nil:
10          ReleaseResource(theSound); return MemError()   // Sound.c:291-292
11      BlockMove(*theSound + 20, theSoundData[63], soundDataSize)  // Sound.c:296
12      ReleaseResource(theSound)                          // Sound.c:297
13      return noErr
14
15  DumpTriggerSound():
16      if theSoundData[63] != nil: DisposePtr(theSoundData[63])   // Sound.c:309-310
17      theSoundData[63] = nil                                     // Sound.c:311
```

- `soundID` here is a **raw resource id** (3000..32767), not an index. Everywhere
  else in the API `soundID`/`which` is an index 0..63.
- `GetResource` searches the resource chain, and `OpenHouseResFork()` has made
  the house file the current resource file
  (`GliderPRO/Sources/HouseIO.c:571`, `:575`), so house `'snd '` resources are
  found first. The application has nothing at 3000+, so a house sound is the only
  possible hit.
- `theSoundData[63] != nil` is the "only one custom sound per room" enforcement,
  matching `kMaxSoundTriggers` = 1 (`GliderPRO/Sources/ObjectAdd.c:17`).
- The return value matters: `ObjectRects.c:927` only creates a hot spot
  `if (LoadTriggerSound(theObject.data.e.where) == noErr)`, so a missing resource
  means the trigger object is silently inert (no hot spot at all, so not even the
  generic sound plays). 13 shipped trigger objects are in exactly that state -
  see section 6.4.

### 4.10 `OpenSoundChannels()` / `CloseSoundChannels()`

`GliderPRO/Sources/Sound.c:365-401`, `:405-434`.

```
 1  OpenSoundChannels():
 2      callBack0UPP = NewSndCallBackProc(CallBack0)      // Sound.c:369
 3      callBack1UPP = NewSndCallBackProc(CallBack1)      // Sound.c:370
 4      callBack2UPP = NewSndCallBackProc(CallBack2)      // Sound.c:371
 5      if channelOpen: return noErr                      // Sound.c:375-376  idempotent-ish
 6      err = SndNewChannel(&channel0, sampledSynth, initNoInterp+initMono, callBack0UPP)
 7      if err: return err  else channelOpen = true       // Sound.c:378-384
 8      err = SndNewChannel(&channel1, ...)               // Sound.c:386-388
 9      if err: return err  else channelOpen = true       // Sound.c:389-392
10      err = SndNewChannel(&channel2, ...)               // Sound.c:394-396
11      if err == noErr: channelOpen = true               // Sound.c:397-398
12      return err
```

`channelOpen` is set after the *first* success, so a failure on channel 1 or 2
returns an error with `channelOpen` already true and `channel1`/`channel2`
possibly nil. `InitSound` turns that into `failedSound = true`
(`GliderPRO/Sources/Sound.c:471`) so nothing dereferences the nil channel.

Note that `NewSndCallBackProc` runs *before* the `channelOpen` early-out, so
calling `OpenSoundChannels()` twice leaks two sets of three UPPs. It is only
called from `InitSound` (`GliderPRO/Sources/Sound.c:467`), once per launch.

`CloseSoundChannels()` disposes channels 0,1,2 with
`SndDisposeChannel(chan, true)` (`GliderPRO/Sources/Sound.c:415`, `:419`, `:423`;
the `true` = quiet the channel immediately rather than draining the queue), nils
each pointer, clears `channelOpen` if no error, and disposes the three UPPs
(`GliderPRO/Sources/Sound.c:429-431`).

### 4.11 `InitSound()` / `KillSound()`

`GliderPRO/Sources/Sound.c:438-474`, `:478-487`. Called from `main()` at
`GliderPRO/Sources/Main.c:337` and `GliderPRO/Sources/Main.c:371`.

```
 1  InitSound():
 2      if dontLoadSounds: return                                // Sound.c:442-443
 3      failedSound = false                                      // Sound.c:445
 4      channel0 = channel1 = channel2 = nil                     // Sound.c:447-449
 5      priority0 = priority1 = priority2 = 0                    // Sound.c:451-453
 6      soundPlaying0 = soundPlaying1 = soundPlaying2 = -1       // Sound.c:454-456
 7      if LoadBufferSounds() != noErr:                          // Sound.c:458
 8          YellowAlert(kYellowFailedSound, err); failedSound = true   // Sound.c:461-462
 9      if not failedSound:
10          if OpenSoundChannels() != noErr:                     // Sound.c:467
11              YellowAlert(kYellowFailedSound, err); failedSound = true  // Sound.c:470-471
12
13  KillSound():
14      if dontLoadSounds: return                                // Sound.c:482-483
15      DumpBufferSounds()                                       // Sound.c:485
16      CloseSoundChannels()                                     // Sound.c:486
```

Order matters on the way in (resources first, then channels) and on the way out
(`Main.c:370-371` calls `KillMusic()` *before* `KillSound()`).

`DumpBufferSounds()` (`GliderPRO/Sources/Sound.c:351-361`) walks all 64 slots -
including the trigger slot 63 - disposing and nilling each.

`kYellowFailedSound` = 14 (`GliderPRO/Headers/GliderDefines.h:31`) selects string
14 of `STR#` 1006, verbatim from the resource fork:

> Wow, there was a problem bringing sounds up.  You might try giving Glider PRO(tm) more memory - otherwise ... silence.

### 4.12 `SoundBytesNeeded()`

`GliderPRO/Sources/Sound.c:491-512`. Called from `CheckMemorySize()`
(`GliderPRO/Sources/Environ.c:575`) before `InitSound`.

```
 1  totalBytes = 0
 2  SetResLoad(false)                            // Sound.c:498  don't actually load
 3  for i = 0..62:                               // Sound.c:499
 4      theSound = GetResource('snd ', i+1000)   // Sound.c:501
 5      if nil: SetResLoad(true); return ResError()   // Sound.c:502-506
 6      totalBytes += GetMaxResourceSize(theSound)    // Sound.c:507
 7  SetResLoad(true)                             // Sound.c:510
 8  return totalBytes
```

`SetResLoad(false)` makes `GetResource` return an *empty* handle (the resource is
not read from disk); `GetMaxResourceSize` then reports the on-disk size from the
resource map. Measured from the shipped fork, this returns
**352,348** (the sum of the 63 resource sizes). Note it sums the *resource*
sizes, not the `-20` sizes the game will actually allocate (351,088), so it
over-estimates by 63 * 20 = 1,260 bytes. The commented-out `ReleaseResource` at
`GliderPRO/Sources/Sound.c:508` means the 63 empty handles are leaked (63 * ~12
bytes of master pointers) - deliberate or not, it is negligible.

### 4.13 `TellHerNoSounds()` / `BitchAboutSM3()`

- `TellHerNoSounds()` (`GliderPRO/Sources/Sound.c:516-523`) shows `ALRT` 1039.
  Called from `CheckMemorySize()` at `GliderPRO/Sources/Environ.c:672`. Its text,
  read out of the shipped `DITL` 1039: *"Okay, with your monitor & color depth,
  there isn't enough memory.  To run Glider PRO(tm), the music & sounds
  won't be loaded."* (note: "To run", where the music-only `DITL` 1038 says "In
  order to run"), single button "Whatever", `ICON` 1011.
- `BitchAboutSM3()` (`GliderPRO/Sources/Sound.c:527-534`) shows `ALRT` 1030:
  *"Where is Sound Manager 3.0?  I highly recommend you install it.  It's fast,
  it's unobtrusive, and it's the law!"*, `ICON` 1070. **Never called** - the only
  call site is commented out at `GliderPRO/Sources/Main.c:357-361` (the call
  itself at `:360`), and
  `thisMac.hasSM3` is hard-coded `true` at `GliderPRO/Sources/Environ.c:451` with
  the comment `// TEMP`. A Go port drops both this function and the whole
  `hasSM3` concept.

### 4.14 The "looping sound" idiom: retriggering on the frame clock

Because `bufferCmd` is one-shot (section 3.10), continuous sounds are produced by
calling `PlayPrioritySound` again every *N* frames. The frame period is
`kTicksPerFrame` = 2 ticks (`GliderPRO/Headers/GliderDefines.h:533`, used at
`GliderPRO/Sources/Render.c:665` and `:690`), and a tick is 1/60.15 s, so one
frame is 33.3 ms and the game runs at 30 fps.

The clearest case is the battery/helium thrust, `GliderPRO/Sources/Input.c:119-181`:

```
 1  DoBatteryEngaged(theGlider):                       // Input.c:121
 2      ...
 3      batteryTotal--                                 // Input.c:138
 4      if batteryTotal == 0:                          // Input.c:143
 5          QuickBatteryRefresh(false)
 6          PlayPrioritySound(kFizzleSound, kFizzlePriority)
 7      else:
 8          if not batteryWasEngaged: batteryFrame = 0 // Input.c:147-148
 9          if batteryFrame == 0:                      // Input.c:149
10              PlayPrioritySound(kThrustSound, kThrustPriority)   // Input.c:150
11          batteryFrame++                             // Input.c:151
12          if batteryFrame >= 4: batteryFrame = 0     // Input.c:152-153
13          batteryWasEngaged = true                   // Input.c:154
```

`DoHeliumEngaged` (`GliderPRO/Sources/Input.c:160-181`) is the same with
`batteryTotal++` (`:163`), `kFizzleSound` + `batteryWasEngaged = false` (`:168-169`),
and `kHissSound` (`:176`).

So the retrigger period is 4 frames = 8 ticks = **133.0 ms**. And the measured
duration of `'snd '` 1062 "Hiss" is 2960 frames / 22254.5455 Hz =
**133.0 ms**. The author tuned the sample length to the retrigger period exactly,
which is why the effect sounds seamless. `'snd '` 1018 "Thrust" is 5759 frames =
258.8 ms, i.e. about twice the period, so consecutive thrusts overlap-and-cut -
each new `bufferCmd` truncates the previous one at 133 ms. Both sounds carry
`loopStart = 0` in the resource, confirming the author's intent, but the loop
fields are unused.

The other two looped effects are retriggered every single frame:

| Sound | Site | Retrigger cadence |
|---|---|---|
| `kShredSound` (1026, 86.9 ms) | `GliderPRO/Sources/Player.c:1287` (`MoveGliderShredding`), `GliderPRO/Sources/Render.c:585` (`RenderShreds`) | every frame, 33.3 ms |
| `kSizzleSound` (1047, 100.7 ms) | `GliderPRO/Sources/Interactions.c:1365` (`case kBurnIt`) | every frame while burning, 33.3 ms |

Both are longer than one frame, so each retrigger truncates the previous
instance. A Go port that naively played these to completion would sound identical
only by accident; the correct behaviour is "restart the buffer from sample 0 on
every call", which is what `SndDoImmediate(bufferCmd)` does.

## 5. The complete sound table

### 5.1 Sound index -> constant -> `'snd '` id -> resource -> priority

Sound *index* (the `which` argument to `PlayPrioritySound`) is 0..62 for the
built-ins and 63 for the trigger slot; the resource id is `index + 1000`
(`GliderPRO/Sources/Sound.c:14`, `:327`). The constants are defined contiguously
at `GliderPRO/Headers/GliderDefines.h:55-118` and the priorities at
`GliderPRO/Headers/GliderDefines.h:120-180`.

Columns: **frames** = `SoundHeader.length` in samples; **ms** = frames /
22254.5455 Hz; **loop** = whether `loopStart == 0` (never actually honoured, see
section 3.10); **purge** = resource attribute bit 0x20 set in the resource map;
**priority** = the constant paired with this sound in the source and its value.
All frame counts, durations and purge flags were read out of
`GliderPRO/Glider PRO.r` with `tools/probe_snd.py`.

| ID | constant (GliderDefines.h line) | `snd ` ID | resource name | frames | ms | loop | purge | priority constant | value |
|---:|---|---:|---|---:|---:|:--:|:--:|---|---:|
| 0 | `kHitWallSound` (55) | 1000 | Wall Hit | 1840 | 82.7 | no | Y | `kHitWallPriority` (120) | 100 |
| 1 | `kFadeInSound` (56) | 1001 | Fade In | 8832 | 396.9 | no | Y | `kFadeInPriority` (173) | 900 |
| 2 | `kFadeOutSound` (57) | 1002 | Fade Out | 9952 | 447.2 | no | Y | `kFadeOutPriority` (174) | 901 |
| 3 | `kBeepsSound` (58) | 1003 | Beeps | 9680 | 435.0 | no | Y | `kBeepsPriority` (160) | 800 |
| 4 | `kBuzzerSound` (59) | 1004 | Buzzer | 8032 | 360.9 | no | Y | `kBuzzerPriority` (161) | 801 |
| 5 | `kDingSound` (60) | 1005 | Ding | 9728 | 437.1 | no | Y | `kDingPriority` (162) | 802 |
| 6 | `kEnergizeSound` (61) | 1006 | Energize | 11904 | 534.9 | no | Y | `kEnergizePriority` (163) | 803 |
| 7 | `kFollowSound` (62) | 1007 | Follow | 7296 | 327.8 | no | Y | `kFollowPriority` (177) | 904 |
| 8 | `kMicrowavedSound` (63) | 1008 | Miked | 10880 | 488.9 | no | Y | `kMicrowavedPriority` (171) | 811 |
| 9 | `kSwitchSound` (64) | 1009 | PowerSwitch | 1264 | 56.8 | no | Y | `kSwitchPriority` (156) | 700 |
| 10 | `kBirdSound` (65) | 1010 | Bird | 5312 | 238.7 | no | Y | `kBirdPriority` (164) | 804 |
| 11 | `kCuckooSound` (66) | 1011 | Cuckoo | 5136 | 230.8 | no | Y | `kCuckooPriority` (165) | 805 |
| 12 | `kTikSound` (67) | 1012 | Tik | 612 | 27.5 | no | Y | `kTikPriority` (124) | 200 |
| 13 | `kTokSound` (68) | 1013 | Tok | 452 | 20.3 | no | Y | `kTokPriority` (125) | 201 |
| 14 | `kBlowerOn` (69) | 1014 | Blowers On | 9376 | 421.3 | no | . | `kBlowerOnPriority` (157) | 701 |
| 15 | `kBlowerOff` (70) | 1015 | Blowers Off | 8128 | 365.2 | no | . | `kBlowerOffPriority` (158) | 702 |
| 16 | `kCaughtFireSound` (71) | 1016 | Yow! | 7664 | 344.4 | no | . | `kCaughtFirePriority` (175) | 902 |
| 17 | `kScoreTikSound` (72) | 1017 | Score Tick | 644 | 28.9 | no | . | `kScoreTikPriority` (121) | 101 |
| 18 | `kThrustSound` (73) | 1018 | Thrust | 5759 | 258.8 | yes | Y | `kThrustPriority` (129) | 300 |
| 19 | `kFizzleSound` (74) | 1019 | Fizzle | 2816 | 126.5 | no | Y | `kFizzlePriority` (159) | 703 |
| 20 | `kFireBandSound` (75) | 1020 | Fire Band | 590 | 26.5 | no | . | `kFireBandPriority` (130) | 301 |
| 21 | `kBandReboundSound` (76) | 1021 | Band Rebound | 1048 | 47.1 | no | Y | `kBandReboundPriority` (122) | 102 |
| 22 | `kGreaseSpillSound` (77) | 1022 | Grease Spill | 4544 | 204.2 | no | . | `kGreaseSpillPriority` (166) | 806 |
| 23 | `kChordSound` (78) | 1023 | A Chord | 19968 | 897.3 | no | Y | `kChordPriority` (131) | 302 |
| 24 | `kVCRSound` (79) | 1024 | VCR | 11520 | 517.6 | no | Y | `kVCRPriority` (132) | 303 |
| 25 | `kFoilHitSound` (80) | 1025 | Foil Hit | 3776 | 169.7 | no | Y | `kFoilHitPriority` (141) | 400 |
| 26 | `kShredSound` (81) | 1026 | Shred | 1934 | 86.9 | yes | . | `kShredPriority` (176) | 903 |
| 27 | `kToastLaunchSound` (82) | 1027 | Toast Launch | 1232 | 55.4 | no | . | `kToastLaunchPriority` (133) | 304 |
| 28 | `kToastLandSound` (83) | 1028 | Toast Land | 2096 | 94.2 | no | . | `kToastLandPriority` (134) | 305 |
| 29 | `kMacOnSound` (84) | 1029 | MacClickOn | 1332 | 59.9 | no | Y | `kMacOnPriority` (142) | 401 |
| 30 | `kMacBeepSound` (85) | 1030 | MacOn | 12160 | 546.4 | no | Y | `kMacBeepPriority` (144) | 403 |
| 31 | `kMacOffSound` (86) | 1031 | MacClickOff | 1184 | 53.2 | no | Y | `kMacOffPriority` (143) | 402 |
| 32 | `kTVOnSound` (87) | 1032 | TVOn | 9784 | 439.6 | no | Y | `kTVOnPriority` (145) | 404 |
| 33 | `kTVOffSound` (88) | 1033 | TVOff | 7648 | 343.7 | no | Y | `kTVOffPriority` (146) | 405 |
| 34 | `kCoffeeSound` (89) | 1034 | Coffee | 8832 | 396.9 | no | Y | `kCoffeePriority` (135) | 306 |
| 35 | `kMysticSound` (90) | 1035 | Mystic | 9696 | 435.7 | no | Y | `kMysticPriority` (126) | 202 |
| 36 | `kZapSound` (91) | 1036 | Zap | 4339 | 195.0 | no | . | `kZapPriority` (147) | 406 |
| 37 | `kPopSound` (92) | 1037 | Pop! | 480 | 21.6 | no | . | `kPopPriority` (148) | 407 |
| 38 | `kEnemyInSound` (93) | 1038 | Enemy In | 6480 | 291.2 | no | Y | `kEnemyInPriority` (149) | 408 |
| 39 | `kEnemyOutSound` (94) | 1039 | Enemy Out | 4880 | 219.3 | no | Y | `kEnemyOutPriority` (150) | 409 |
| 40 | `kPaperCrunchSound` (95) | 1040 | Paper Crunch | 6928 | 311.3 | no | . | `kPaperCrunchPriority` (151) | 410 |
| 41 | `kBounceSound` (96) | 1041 | Bounce | 2976 | 133.7 | no | . | `kBouncePriority` (136) | 307 |
| 42 | `kDripSound` (97) | 1042 | Drip | 1008 | 45.3 | no | . | `kDripPriority` (137) | 308 |
| 43 | `kDropSound` (98) | 1043 | Drop | 1376 | 61.8 | no | Y | `kDropPriority` (138) | 309 |
| 44 | `kFishOutSound` (99) | 1044 | Fish Leap | 6688 | 300.5 | no | Y | `kFishOutPriority` (152) | 411 |
| 45 | `kFishInSound` (100) | 1045 | Fish Land | 3760 | 169.0 | no | Y | `kFishInPriority` (153) | 412 |
| 46 | `kDontExitSound` (101) | 1046 | Dont Exit | 2368 | 106.4 | no | Y | `kDontExitPriority` (123) | 103 |
| 47 | `kSizzleSound` (102) | 1047 | Sizzle | 2240 | 100.7 | yes | Y | `kSizzlePriority` (154) | 413 |
| 48 | `kPaper1Sound` (103) | 1048 | Paper1 | 4432 | 199.2 | no | Y | `kPapersPriority` (167) | 807 |
| 49 | `kPaper2Sound` (104) | 1049 | Paper2 | 4096 | 184.1 | no | Y | `kPapersPriority` (167) | 807 |
| 50 | `kPaper3Sound` (105) | 1050 | Paper3 | 1648 | 74.1 | no | Y | `kPapersPriority` (167) | 807 |
| 51 | `kPaper4Sound` (106) | 1051 | Paper4 | 1584 | 71.2 | no | Y | `kPapersPriority` (167) | 807 |
| 52 | `kTypingSound` (107) | 1052 | Keystroke | 2524 | 113.4 | no | . | `kTypingPriority` (168) | 808 |
| 53 | `kCarriageSound` (108) | 1053 | Carriage Return | 5248 | 235.8 | no | . | `kCarriagePriority` (169) | 809 |
| 54 | `kChord2Sound` (109) | 1054 | Check | 5056 | 227.2 | no | Y | `kChord2Priority` (170) | 810 |
| 55 | `kPhoneRingSound` (110) | 1055 | Phone | 11008 | 494.6 | no | Y | `kPhoneRingPriority` (155) | 500 |
| 56 | `kChime1Sound` (111) | 1056 | Ding 1 | 7872 | 353.7 | no | Y | `kChime1Priority` (127) | 203 |
| 57 | `kChime2Sound` (112) | 1057 | Ding 2 | 9504 | 427.1 | no | Y | `kChime2Priority` (128) | 204 |
| 58 | `kWebTwangSound` (113) | 1058 | Twunk | 2272 | 102.1 | no | Y | `kWebTwangPriority` (139) | 310 |
| 59 | `kTransOutSound` (114) | 1059 | TransOut | 9952 | 447.2 | no | Y | `kTransOutPriority` (179) | 906 |
| 60 | `kTransInSound` (115) | 1060 | TransIn | 9984 | 448.6 | no | Y | `kTransInPriority` (178) | 905 |
| 61 | `kBonusSound` (116) | 1061 | Bonus | 5388 | 242.1 | no | Y | `kBonusPriority` (172) | 812 |
| 62 | `kHissSound` (117) | 1062 | Hiss | 2960 | 133.0 | yes | Y | `kHissPriority` (140) | 311 |
| 63 | `kTriggerSound` (118) | n/a - see below | (none - loaded at run time) | 0 | - | - | - | `kTriggerPriority` (180) | 999 |

Notes on the table:

- Index 63 (`kTriggerSound`) has **no application resource**. It is the slot
  filled by `LoadTriggerSound()` from the *house* resource fork with whatever id
  the room's `kSoundTrigger` object names (>= 3000). See section 6.3.
- Four sounds share one priority constant: `kPaper1Sound`..`kPaper4Sound` (indices
  48..51, `GliderPRO/Headers/GliderDefines.h:103-106`) all use
  `kPapersPriority` = 807
  (`GliderPRO/Headers/GliderDefines.h:167`). Every other sound has its own.
- Two constants are named without the `Sound` suffix: `kBlowerOn` (index 14) and
  `kBlowerOff` (index 15), at `GliderPRO/Headers/GliderDefines.h:69-70`. They are
  sound indices like any other; do not confuse them with the object-state
  booleans of the same idea in `Objects.c`.
- 16 of the 63 effects are **not** purgeable (attribute bit 0x20 clear):
  1014 Blowers On, 1015 Blowers Off, 1016 Yow!, 1017 Score Tick, 1020 Fire Band,
  1022 Grease Spill, 1026 Shred, 1027 Toast Launch, 1028 Toast Land, 1036 Zap,
  1037 Pop!, 1040 Paper Crunch, 1041 Bounce, 1042 Drip, 1052 Keystroke,
  1053 Carriage Return. The other 47 effects and all 7 music resources are
  purgeable. This is irrelevant to behaviour - `LoadBufferSounds` copies every
  resource into a `NewPtr` block and releases the handle immediately - but it is
  what the shipped fork says, and a Go port ignores it entirely.
- Total: 63 effects, 352,348 resource bytes, 351,088 bytes of PCM+header retained.
  Longest effect: 1023 "A Chord", 19,968 frames = 897.3 ms. Shortest:
  1013 "Tok", 452 frames = 20.3 ms.

### 5.2 Priorities, sorted by value

The priorities are hand-banded by category, in ascending order of "how much the
player needs to hear this". The bands are 100-103, 200-204, 300-311, 400-413,
500, 700-703, 800-812, 900-906, 999. **There is no 600 band** and there are no
gaps within a band. Idle is 0 and is not a constant.

| priority | constant | line | sound(s) that use it |
|---:|---|---:|---|
| 100 | `kHitWallPriority` | 120 | `kHitWallSound` |
| 101 | `kScoreTikPriority` | 121 | `kScoreTikSound` |
| 102 | `kBandReboundPriority` | 122 | `kBandReboundSound` |
| 103 | `kDontExitPriority` | 123 | `kDontExitSound` |
| 200 | `kTikPriority` | 124 | `kTikSound` |
| 201 | `kTokPriority` | 125 | `kTokSound` |
| 202 | `kMysticPriority` | 126 | `kMysticSound` |
| 203 | `kChime1Priority` | 127 | `kChime1Sound` |
| 204 | `kChime2Priority` | 128 | `kChime2Sound` |
| 300 | `kThrustPriority` | 129 | `kThrustSound` |
| 301 | `kFireBandPriority` | 130 | `kFireBandSound` |
| 302 | `kChordPriority` | 131 | `kChordSound` |
| 303 | `kVCRPriority` | 132 | `kVCRSound` |
| 304 | `kToastLaunchPriority` | 133 | `kToastLaunchSound` |
| 305 | `kToastLandPriority` | 134 | `kToastLandSound` |
| 306 | `kCoffeePriority` | 135 | `kCoffeeSound` |
| 307 | `kBouncePriority` | 136 | `kBounceSound` |
| 308 | `kDripPriority` | 137 | `kDripSound` |
| 309 | `kDropPriority` | 138 | `kDropSound` |
| 310 | `kWebTwangPriority` | 139 | `kWebTwangSound` |
| 311 | `kHissPriority` | 140 | `kHissSound` |
| 400 | `kFoilHitPriority` | 141 | `kFoilHitSound` |
| 401 | `kMacOnPriority` | 142 | `kMacOnSound` |
| 402 | `kMacOffPriority` | 143 | `kMacOffSound` |
| 403 | `kMacBeepPriority` | 144 | `kMacBeepSound` |
| 404 | `kTVOnPriority` | 145 | `kTVOnSound` |
| 405 | `kTVOffPriority` | 146 | `kTVOffSound` |
| 406 | `kZapPriority` | 147 | `kZapSound` |
| 407 | `kPopPriority` | 148 | `kPopSound` |
| 408 | `kEnemyInPriority` | 149 | `kEnemyInSound` |
| 409 | `kEnemyOutPriority` | 150 | `kEnemyOutSound` |
| 410 | `kPaperCrunchPriority` | 151 | `kPaperCrunchSound` |
| 411 | `kFishOutPriority` | 152 | `kFishOutSound` |
| 412 | `kFishInPriority` | 153 | `kFishInSound` |
| 413 | `kSizzlePriority` | 154 | `kSizzleSound` |
| 500 | `kPhoneRingPriority` | 155 | `kPhoneRingSound` |
| 700 | `kSwitchPriority` | 156 | `kSwitchSound` |
| 701 | `kBlowerOnPriority` | 157 | `kBlowerOn` |
| 702 | `kBlowerOffPriority` | 158 | `kBlowerOff` |
| 703 | `kFizzlePriority` | 159 | `kFizzleSound` |
| 800 | `kBeepsPriority` | 160 | `kBeepsSound` |
| 801 | `kBuzzerPriority` | 161 | `kBuzzerSound` |
| 802 | `kDingPriority` | 162 | `kDingSound` |
| 803 | `kEnergizePriority` | 163 | `kEnergizeSound` |
| 804 | `kBirdPriority` | 164 | `kBirdSound` |
| 805 | `kCuckooPriority` | 165 | `kCuckooSound` |
| 806 | `kGreaseSpillPriority` | 166 | `kGreaseSpillSound` |
| 807 | `kPapersPriority` | 167 | `kPaper1Sound`, `kPaper2Sound`, `kPaper3Sound`, `kPaper4Sound` |
| 808 | `kTypingPriority` | 168 | `kTypingSound` |
| 809 | `kCarriagePriority` | 169 | `kCarriageSound` |
| 810 | `kChord2Priority` | 170 | `kChord2Sound` |
| 811 | `kMicrowavedPriority` | 171 | `kMicrowavedSound` |
| 812 | `kBonusPriority` | 172 | `kBonusSound` |
| 900 | `kFadeInPriority` | 173 | `kFadeInSound` |
| 901 | `kFadeOutPriority` | 174 | `kFadeOutSound` |
| 902 | `kCaughtFirePriority` | 175 | `kCaughtFireSound` |
| 903 | `kShredPriority` | 176 | `kShredSound` |
| 904 | `kFollowPriority` | 177 | `kFollowSound` |
| 905 | `kTransInPriority` | 178 | `kTransInSound` |
| 906 | `kTransOutPriority` | 179 | `kTransOutSound` |
| 999 | `kTriggerPriority` | 180 | `kTriggerSound` |

Reading the bands as the author's intent:

| Band | Character | Examples |
|---|---|---|
| 100-103 | constant background collisions - interruptible by anything | wall hit, score tick, band rebound, "can't exit" |
| 200-204 | clock and ambience | tik, tok, mystic, the two chimes |
| 300-311 | player-driven mechanics | thrust, hiss, rubber band, chord, appliances, drips |
| 400-413 | object and enemy events | foil hit, Mac on/off, TV on/off, zap, pop, enemy in/out, sizzle |
| 500 | the telephone - one entry, deliberately above objects | phone ring |
| 700-703 | switches and blowers - things the player just clicked | switch, blowers on/off, fizzle |
| 800-812 | "attention" sounds and scoring | beeps, buzzer, ding, energize, bird, cuckoo, papers, typing, bonus |
| 900-906 | room transitions and death - must never be cut | fade in/out, caught fire, shred, follow, transporter in/out |
| 999 | the house author's custom sound | trigger |

Since all three voices start at 0 and there are only three, the practical effect
is: a 900-band sound will always play; a 100-band sound plays unless all three
voices are already busy with anything at all.

## 6. What triggers each sound

### 6.1 Summary: sound -> game event

Derived from all 167 call sites (section 6.2). "n" is the number of call sites.

| # | Constant | n | Game event |
|---:|---|---:|---|
| 0 | `kHitWallSound` | 7 | glider hits a wall or floor and bounces (`BounceGlider`); glider tries to leave the room through a wall (`CheckEscapeLeft/Right[Two]`); a rubber band hits a wall (`CheckBandCollision`) |
| 1 | `kFadeInSound` | 1 | a new glider fades into the room (`FadeGliderIn`) |
| 2 | `kFadeOutSound` | 32 | the glider dies, in every possible way: room-exit failure, roof collision, dynamic-object collision, burning, shredding, web, forced kill, map restore |
| 3 | `kBeepsSound` | 1 | picked up a reward object (`HandleRewards`) |
| 4 | `kBuzzerSound` | 1 | picked up a reward object (`HandleRewards`) |
| 5 | `kDingSound` | 1 | picked up a reward object (`HandleRewards`) |
| 6 | `kEnergizeSound` | 8 | picked up a reward; and each character typed check in the high-score name/banner entry |
| 7 | `kFollowSound` | 1 | the glider enters limbo, i.e. the room scrolls to follow it (`FlagGliderInLimbo`) |
| 8 | `kMicrowavedSound` | 1 | the microwave cooks something (`HandleMicrowaveAction`) |
| 9 | `kSwitchSound` | 5 | any of the five switch kinds is flipped (`HandleSwitches`) |
| 10 | `kBirdSound` | 1 | played once from `main()` - the startup/splash chirp |
| 11 | `kCuckooSound` | 1 | picked up a reward object (`HandleRewards`) |
| 12 | `kTikSound` | 1 | pendulum swings one way (`RenderPendulums`) |
| 13 | `kTokSound` | 1 | pendulum swings the other way (`RenderPendulums`) |
| 14 | `kBlowerOn` | 1 | a blower/duct/candle/fan is switched on (`SetObjectState`) |
| 15 | `kBlowerOff` | 1 | a blower/duct/candle/fan is switched off (`SetObjectState`) |
| 16 | `kCaughtFireSound` | 2 | the glider catches fire or begins shredding (`FlagGliderBurning`, `FlagGliderShredding`) |
| 17 | `kScoreTikSound` | 1 | the on-screen score counter ticks up (`RefreshScoreboard`) |
| 18 | `kThrustSound` | 1 | battery thrust held down; retriggered every 4 frames (`DoBatteryEngaged`) |
| 19 | `kFizzleSound` | 3 | battery or helium runs out (`DoBatteryEngaged`, `DoHeliumEngaged`); tin-foil protection is lost (`StartGliderFoilLosing`) |
| 20 | `kFireBandSound` | 1 | the player fires a rubber band (`AddBand`) |
| 21 | `kBandReboundSound` | 3 | a rubber band bounces off something (`CheckBandCollision`) |
| 22 | `kGreaseSpillSound` | 1 | a grease can tips over (`SpillGrease`) |
| 23 | `kChordSound` | 4 | strum a guitar / touch a "strum it" hot spot (`HandleHotSpotCollision`); a switch fires a linked guitar or sound trigger (`HandleSwitches`, `FireTrigger`) |
| 24 | `kVCRSound` | 1 | the VCR is switched on (`HandleVCR`) |
| 25 | `kFoilHitSound` | 8 | the glider is protected by tin foil and hits something (all the same collision sites as `kHitWallSound`, plus `GliderHitTop` and hot spots) |
| 26 | `kShredSound` | 2 | the glider is being shredded; retriggered every frame (`MoveGliderShredding`, `RenderShreds`) |
| 27 | `kToastLaunchSound` | 2 | toast pops out of the toaster (`HandleToast`, `TriggerToast`) |
| 28 | `kToastLandSound` | 1 | toast lands (`HandleToast`) |
| 29 | `kMacOnSound` | 5 | any appliance turns on: Mac Plus, coffee maker, VCR, stereo, microwave |
| 30 | `kMacBeepSound` | 1 | the Mac Plus boot chime, one frame after `kMacOnSound` (`HandleMacPlus`) |
| 31 | `kMacOffSound` | 5 | any appliance turns off: Mac Plus, coffee maker, VCR, stereo, microwave |
| 32 | `kTVOnSound` | 1 | the TV turns on (`HandleTV`) |
| 33 | `kTVOffSound` | 1 | the TV turns off (`HandleTV`) |
| 34 | `kCoffeeSound` | 2 | the coffee maker gurgles (`HandleCoffee`); a switch fires a linked coffee maker (`FireTrigger`) |
| 35 | `kMysticSound` | 2 | a sparkle/star object animates (`HandleSparkleObject`); the game-over star animation (`DoGameOverStarAnimation`) |
| 36 | `kZapSound` | 3 | an electrical outlet arcs (`HandleOutlet`); a switch fires a linked outlet (`TriggerOutlet`) |
| 37 | `kPopSound` | 1 | a balloon is popped (`HandleBalloon`) |
| 38 | `kEnemyInSound` | 6 | balloon, copter or dart enters the room |
| 39 | `kEnemyOutSound` | 3 | balloon, copter or dart leaves the room |
| 40 | `kPaperCrunchSound` | 2 | a copter or dart is destroyed (`HandleCopter`, `HandleDart`) |
| 41 | `kBounceSound` | 1 | the bouncing ball bounces (`HandleBall`) |
| 42 | `kDripSound` | 1 | a drip forms (`HandleDrip`) |
| 43 | `kDropSound` | 2 | a drip falls; a fish drops back (`HandleDrip`, `HandleFish`) |
| 44 | `kFishOutSound` | 2 | the fish jumps out of the tank (`HandleFish`, `TriggerFish`) |
| 45 | `kFishInSound` | 1 | the fish splashes back in (`HandleFish`) |
| 46 | `kDontExitSound` | 8 | the glider tries to leave through an opening that is closed or leads nowhere (all four `CheckEscape*Two` directions) |
| 47 | `kSizzleSound` | 1 | the glider is over a "burn it" hot spot; retriggered every frame (`HandleHotSpotCollision`, `case kBurnIt`) |
| 48 | `kPaper1Sound` | 1 | game-over page-turn animation, page 1 (`HandlePages`) |
| 49 | `kPaper2Sound` | 1 | game-over page-turn animation, page 2 (`HandlePages`) |
| 50 | `kPaper3Sound` | 1 | game-over page-turn animation, page 3 (`HandlePages`) |
| 51 | `kPaper4Sound` | 1 | game-over page-turn animation, page 4 (`HandlePages`) |
| 52 | `kTypingSound` | 2 | a printable key in high-score name / banner entry (`NameFilter`, `BannerFilter`) |
| 53 | `kCarriageSound` | 2 | Return/Enter in high-score name / banner entry |
| 54 | `kChord2Sound` | 1 | the Sound Prefs dialog changes the volume, as an audible preview (`HandleSoundMusicChange`) |
| 55 | `kPhoneRingSound` | 1 | the room telephone rings (`HandleTelephone`) |
| 56 | `kChime1Sound` | 1 | grandfather-clock chime, variant 1, chosen at random (`HandleTelephone`) |
| 57 | `kChime2Sound` | 1 | grandfather-clock chime, variant 2, chosen at random (`HandleTelephone`) |
| 58 | `kWebTwangSound` | 1 | the glider is caught in a spider web (`WebGlider`) |
| 59 | `kTransOutSound` | 4 | the glider leaves via mail, duct (up or down) or transporter (`StartGlider*`) |
| 60 | `kTransInSound` | 4 | the glider arrives via mail (left/right), duct or transporter (`Finish*`, `TransportGliderIn`) |
| 61 | `kBonusSound` | 1 | a bonus reward object is collected (`HandleRewards`) |
| 62 | `kHissSound` | 1 | helium thrust held down; retriggered every 4 frames (`DoHeliumEngaged`) |
| 63 | `kTriggerSound` | 2 | the glider steps on a `kSoundTrigger` hot spot (`case kSoundIt`), or a switch fires one (`case kSoundTrigger` in `HandleSwitches`) |

`kFadeOutSound` (32 sites) and `kFadeInSound` (1 site) between them bracket every
glider life. `kFadeOutSound` having 32 call sites and priority 901 - second only
to the trigger - reflects that dying must always be audible.

### 6.2 Every `PlayPrioritySound()` call site

Located mechanically:

```
grep -an 'PlayPrioritySound' /tmp/.../Sources_*.c
```

170 raw occurrences of the token across `Sources/*.c` + `Headers/*.h`. Three are
not calls: the banner comment at `GliderPRO/Sources/Sound.c:38`, the definition at
`GliderPRO/Sources/Sound.c:40`, and the prototype at
`GliderPRO/Headers/GliderProtos.h:447`. That leaves **167 real call sites** in
**20** source files (`DynamicMaps.c`, `Dynamics.c`, `Dynamics2.c`, `GameOver.c`,
`Grease.c`, `HighScores.c`, `Input.c`, `Interactions.c`, `Main.c`, `Modes.c`,
`Objects.c`, `Play.c`, `Player.c`, `Render.c`, `RubberBands.c`, `Scoreboard.c`,
`Settings.c`, `Transit.c`, `Triggers.c`, `Trip.c`). Note `Dynamics3.c` contains
*no* call sites. Grouped by sound:


**`kHitWallSound`** (0, `snd ` 1000) - 7 call sites:

- `GliderPRO/Sources/Interactions.c:166` in `BounceGlider()`
- `GliderPRO/Sources/Interactions.c:543` in `CheckEscapeLeftTwo()`
- `GliderPRO/Sources/Interactions.c:588` in `CheckEscapeLeft()`
- `GliderPRO/Sources/Interactions.c:633` in `CheckEscapeRightTwo()`
- `GliderPRO/Sources/Interactions.c:678` in `CheckEscapeRight()`
- `GliderPRO/Sources/RubberBands.c:167` in `CheckBandCollision()`
- `GliderPRO/Sources/RubberBands.c:190` in `CheckBandCollision()`

**`kFadeInSound`** (1, `snd ` 1001) - 1 call site:

- `GliderPRO/Sources/Player.c:234` in `FadeGliderIn()`

**`kFadeOutSound`** (2, `snd ` 1002) - 32 call sites:

- `GliderPRO/Sources/DynamicMaps.c:158` in `RestoreFromSavedMap()`
- `GliderPRO/Sources/Dynamics.c:70` in `CheckDynamicCollision()`
- `GliderPRO/Sources/Interactions.c:355` in `CheckEscapeDownTwo()`
- `GliderPRO/Sources/Interactions.c:371` in `CheckEscapeDownTwo()`
- `GliderPRO/Sources/Interactions.c:413` in `CheckEscapeDown()`
- `GliderPRO/Sources/Interactions.c:428` in `CheckEscapeDown()`
- `GliderPRO/Sources/Interactions.c:443` in `CheckEscapeDown()`
- `GliderPRO/Sources/Interactions.c:465` in `CheckRoofCollision()`
- `GliderPRO/Sources/Interactions.c:475` in `CheckRoofCollision()`
- `GliderPRO/Sources/Interactions.c:485` in `CheckRoofCollision()`
- `GliderPRO/Sources/Interactions.c:495` in `CheckRoofCollision()`
- `GliderPRO/Sources/Interactions.c:502` in `CheckRoofCollision()`
- `GliderPRO/Sources/Interactions.c:702` in `CheckGliderInRoom()`
- `GliderPRO/Sources/Interactions.c:715` in `CheckGliderInRoom()`
- `GliderPRO/Sources/Interactions.c:731` in `CheckGliderInRoom()`
- `GliderPRO/Sources/Interactions.c:744` in `CheckGliderInRoom()`
- `GliderPRO/Sources/Interactions.c:1226` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1241` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1257` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1292` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1386` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1429` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1472` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1515` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1548` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1611` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Interactions.c:1745` in `WebGlider()`
- `GliderPRO/Sources/Interactions.c:1774` in `WebGlider()`
- `GliderPRO/Sources/Player.c:224` in `MoveGliderBurning()`
- `GliderPRO/Sources/Render.c:604` in `RenderShreds()`
- `GliderPRO/Sources/Transit.c:456` in `ForceKillGlider()`
- `GliderPRO/Sources/Transit.c:465` in `ForceKillGlider()`

**`kBeepsSound`** (3, `snd ` 1003) - 1 call site:

- `GliderPRO/Sources/Interactions.c:770` in `HandleRewards()`

**`kBuzzerSound`** (4, `snd ` 1004) - 1 call site:

- `GliderPRO/Sources/Interactions.c:786` in `HandleRewards()`

**`kDingSound`** (5, `snd ` 1005) - 1 call site:

- `GliderPRO/Sources/Interactions.c:802` in `HandleRewards()`

**`kEnergizeSound`** (6, `snd ` 1006) - 8 call sites:

- `GliderPRO/Sources/HighScores.c:512` in `GetHighScoreName()`
- `GliderPRO/Sources/HighScores.c:618` in `GetHighScoreBanner()`
- `GliderPRO/Sources/Interactions.c:835` in `HandleRewards()`
- `GliderPRO/Sources/Interactions.c:854` in `HandleRewards()`
- `GliderPRO/Sources/Interactions.c:876` in `HandleRewards()`
- `GliderPRO/Sources/Interactions.c:904` in `HandleRewards()`
- `GliderPRO/Sources/Interactions.c:937` in `HandleRewards()`
- `GliderPRO/Sources/Interactions.c:960` in `HandleRewards()`

**`kFollowSound`** (7, `snd ` 1007) - 1 call site:

- `GliderPRO/Sources/Modes.c:464` in `FlagGliderInLimbo()`

**`kMicrowavedSound`** (8, `snd ` 1008) - 1 call site:

- `GliderPRO/Sources/Interactions.c:1193` in `HandleMicrowaveAction()`

**`kSwitchSound`** (9, `snd ` 1009) - 5 call sites:

- `GliderPRO/Sources/Interactions.c:1006` in `HandleSwitches()`
- `GliderPRO/Sources/Interactions.c:1011` in `HandleSwitches()`
- `GliderPRO/Sources/Interactions.c:1016` in `HandleSwitches()`
- `GliderPRO/Sources/Interactions.c:1021` in `HandleSwitches()`
- `GliderPRO/Sources/Interactions.c:1026` in `HandleSwitches()`

**`kBirdSound`** (10, `snd ` 1010) - 1 call site:

- `GliderPRO/Sources/Main.c:343` in `main()`

**`kCuckooSound`** (11, `snd ` 1011) - 1 call site:

- `GliderPRO/Sources/Interactions.c:818` in `HandleRewards()`

**`kTikSound`** (12, `snd ` 1012) - 1 call site:

- `GliderPRO/Sources/Render.c:291` in `RenderPendulums()`

**`kTokSound`** (13, `snd ` 1013) - 1 call site:

- `GliderPRO/Sources/Render.c:307` in `RenderPendulums()`

**`kBlowerOn`** (14, `snd ` 1014) - 1 call site:

- `GliderPRO/Sources/Objects.c:412` in `SetObjectState()`

**`kBlowerOff`** (15, `snd ` 1015) - 1 call site:

- `GliderPRO/Sources/Objects.c:414` in `SetObjectState()`

**`kCaughtFireSound`** (16, `snd ` 1016) - 2 call sites:

- `GliderPRO/Sources/Modes.c:368` in `FlagGliderShredding()`
- `GliderPRO/Sources/Modes.c:410` in `FlagGliderBurning()`

**`kScoreTikSound`** (17, `snd ` 1017) - 1 call site:

- `GliderPRO/Sources/Scoreboard.c:91` in `RefreshScoreboard()`

**`kThrustSound`** (18, `snd ` 1018) - 1 call site:

- `GliderPRO/Sources/Input.c:150` in `DoBatteryEngaged()`

**`kFizzleSound`** (19, `snd ` 1019) - 3 call sites:

- `GliderPRO/Sources/Input.c:143` in `DoBatteryEngaged()`
- `GliderPRO/Sources/Input.c:168` in `DoHeliumEngaged()`
- `GliderPRO/Sources/Modes.c:612` in `StartGliderFoilLosing()`

**`kFireBandSound`** (20, `snd ` 1020) - 1 call site:

- `GliderPRO/Sources/RubberBands.c:288` in `AddBand()`

**`kBandReboundSound`** (21, `snd ` 1021) - 3 call sites:

- `GliderPRO/Sources/RubberBands.c:50` in `CheckBandCollision()`
- `GliderPRO/Sources/RubberBands.c:59` in `CheckBandCollision()`
- `GliderPRO/Sources/RubberBands.c:115` in `CheckBandCollision()`

**`kGreaseSpillSound`** (22, `snd ` 1022) - 1 call site:

- `GliderPRO/Sources/Grease.c:263` in `SpillGrease()`

**`kChordSound`** (23, `snd ` 1023) - 4 call sites:

- `GliderPRO/Sources/Interactions.c:1098` in `HandleSwitches()`
- `GliderPRO/Sources/Interactions.c:1346` in `HandleHotSpotCollision()`
- `GliderPRO/Sources/Triggers.c:132` in `FireTrigger()`
- `GliderPRO/Sources/Triggers.c:140` in `FireTrigger()`

**`kVCRSound`** (24, `snd ` 1024) - 1 call site:

- `GliderPRO/Sources/Dynamics.c:619` in `HandleVCR()`

**`kFoilHitSound`** (25, `snd ` 1025) - 8 call sites:

- `GliderPRO/Sources/Dynamics.c:59` in `CheckDynamicCollision()`
- `GliderPRO/Sources/Interactions.c:81` in `GliderHitTop()`
- `GliderPRO/Sources/Interactions.c:164` in `BounceGlider()`
- `GliderPRO/Sources/Interactions.c:541` in `CheckEscapeLeftTwo()`
- `GliderPRO/Sources/Interactions.c:586` in `CheckEscapeLeft()`
- `GliderPRO/Sources/Interactions.c:631` in `CheckEscapeRightTwo()`
- `GliderPRO/Sources/Interactions.c:676` in `CheckEscapeRight()`
- `GliderPRO/Sources/Interactions.c:1330` in `HandleHotSpotCollision()`

**`kShredSound`** (26, `snd ` 1026) - 2 call sites:

- `GliderPRO/Sources/Player.c:1287` in `MoveGliderShredding()`
- `GliderPRO/Sources/Render.c:585` in `RenderShreds()`

**`kToastLaunchSound`** (27, `snd ` 1027) - 2 call sites:

- `GliderPRO/Sources/Dynamics.c:378` in `HandleToast()`
- `GliderPRO/Sources/Trip.c:162` in `TriggerToast()`

**`kToastLandSound`** (28, `snd ` 1028) - 1 call site:

- `GliderPRO/Sources/Dynamics.c:364` in `HandleToast()`

**`kMacOnSound`** (29, `snd ` 1029) - 5 call sites:

- `GliderPRO/Sources/Dynamics.c:407` in `HandleMacPlus()`
- `GliderPRO/Sources/Dynamics.c:496` in `HandleCoffee()`
- `GliderPRO/Sources/Dynamics.c:616` in `HandleVCR()`
- `GliderPRO/Sources/Dynamics.c:685` in `HandleStereo()`
- `GliderPRO/Sources/Dynamics.c:728` in `HandleMicrowave()`

**`kMacBeepSound`** (30, `snd ` 1030) - 1 call site:

- `GliderPRO/Sources/Dynamics.c:399` in `HandleMacPlus()`

**`kMacOffSound`** (31, `snd ` 1031) - 5 call sites:

- `GliderPRO/Sources/Dynamics.c:415` in `HandleMacPlus()`
- `GliderPRO/Sources/Dynamics.c:515` in `HandleCoffee()`
- `GliderPRO/Sources/Dynamics.c:658` in `HandleVCR()`
- `GliderPRO/Sources/Dynamics.c:702` in `HandleStereo()`
- `GliderPRO/Sources/Dynamics.c:754` in `HandleMicrowave()`

**`kTVOnSound`** (32, `snd ` 1032) - 1 call site:

- `GliderPRO/Sources/Dynamics.c:448` in `HandleTV()`

**`kTVOffSound`** (33, `snd ` 1033) - 1 call site:

- `GliderPRO/Sources/Dynamics.c:469` in `HandleTV()`

**`kCoffeeSound`** (34, `snd ` 1034) - 2 call sites:

- `GliderPRO/Sources/Dynamics.c:505` in `HandleCoffee()`
- `GliderPRO/Sources/Triggers.c:144` in `FireTrigger()`

**`kMysticSound`** (35, `snd ` 1035) - 2 call sites:

- `GliderPRO/Sources/Dynamics.c:308` in `HandleSparkleObject()`
- `GliderPRO/Sources/GameOver.c:158` in `DoGameOverStarAnimation()`

**`kZapSound`** (36, `snd ` 1036) - 3 call sites:

- `GliderPRO/Sources/Dynamics.c:561` in `HandleOutlet()`
- `GliderPRO/Sources/Dynamics.c:593` in `HandleOutlet()`
- `GliderPRO/Sources/Trip.c:179` in `TriggerOutlet()`

**`kPopSound`** (37, `snd ` 1037) - 1 call site:

- `GliderPRO/Sources/Dynamics2.c:66` in `HandleBalloon()`

**`kEnemyInSound`** (38, `snd ` 1038) - 6 call sites:

- `GliderPRO/Sources/Dynamics2.c:119` in `HandleBalloon()`
- `GliderPRO/Sources/Dynamics2.c:126` in `HandleBalloon()`
- `GliderPRO/Sources/Dynamics2.c:227` in `HandleCopter()`
- `GliderPRO/Sources/Dynamics2.c:234` in `HandleCopter()`
- `GliderPRO/Sources/Dynamics2.c:343` in `HandleDart()`
- `GliderPRO/Sources/Dynamics2.c:350` in `HandleDart()`

**`kEnemyOutSound`** (39, `snd ` 1039) - 3 call sites:

- `GliderPRO/Sources/Dynamics2.c:97` in `HandleBalloon()`
- `GliderPRO/Sources/Dynamics2.c:199` in `HandleCopter()`
- `GliderPRO/Sources/Dynamics2.c:305` in `HandleDart()`

**`kPaperCrunchSound`** (40, `snd ` 1040) - 2 call sites:

- `GliderPRO/Sources/Dynamics2.c:167` in `HandleCopter()`
- `GliderPRO/Sources/Dynamics2.c:275` in `HandleDart()`

**`kBounceSound`** (41, `snd ` 1041) - 1 call site:

- `GliderPRO/Sources/Dynamics2.c:398` in `HandleBall()`

**`kDripSound`** (42, `snd ` 1042) - 1 call site:

- `GliderPRO/Sources/Dynamics2.c:493` in `HandleDrip()`

**`kDropSound`** (43, `snd ` 1043) - 2 call sites:

- `GliderPRO/Sources/Dynamics2.c:461` in `HandleDrip()`
- `GliderPRO/Sources/Dynamics2.c:537` in `HandleFish()`

**`kFishOutSound`** (44, `snd ` 1044) - 2 call sites:

- `GliderPRO/Sources/Dynamics2.c:584` in `HandleFish()`
- `GliderPRO/Sources/Trip.c:203` in `TriggerFish()`

**`kFishInSound`** (45, `snd ` 1045) - 1 call site:

- `GliderPRO/Sources/Dynamics2.c:542` in `HandleFish()`

**`kDontExitSound`** (46, `snd ` 1046) - 8 call sites:

- `GliderPRO/Sources/Interactions.c:192` in `CheckEscapeUpTwo()`
- `GliderPRO/Sources/Interactions.c:226` in `CheckEscapeUpTwo()`
- `GliderPRO/Sources/Interactions.c:304` in `CheckEscapeDownTwo()`
- `GliderPRO/Sources/Interactions.c:338` in `CheckEscapeDownTwo()`
- `GliderPRO/Sources/Interactions.c:532` in `CheckEscapeLeftTwo()`
- `GliderPRO/Sources/Interactions.c:563` in `CheckEscapeLeftTwo()`
- `GliderPRO/Sources/Interactions.c:622` in `CheckEscapeRightTwo()`
- `GliderPRO/Sources/Interactions.c:653` in `CheckEscapeRightTwo()`

**`kSizzleSound`** (47, `snd ` 1047) - 1 call site:

- `GliderPRO/Sources/Interactions.c:1365` in `HandleHotSpotCollision()`

**`kPaper1Sound`** (48, `snd ` 1048) - 1 call site:

- `GliderPRO/Sources/GameOver.c:386` in `HandlePages()`

**`kPaper2Sound`** (49, `snd ` 1049) - 1 call site:

- `GliderPRO/Sources/GameOver.c:388` in `HandlePages()`

**`kPaper3Sound`** (50, `snd ` 1050) - 1 call site:

- `GliderPRO/Sources/GameOver.c:347` in `HandlePages()`

**`kPaper4Sound`** (51, `snd ` 1051) - 1 call site:

- `GliderPRO/Sources/GameOver.c:349` in `HandlePages()`

**`kTypingSound`** (52, `snd ` 1052) - 2 call sites:

- `GliderPRO/Sources/HighScores.c:476` in `NameFilter()`
- `GliderPRO/Sources/HighScores.c:584` in `BannerFilter()`

**`kCarriageSound`** (53, `snd ` 1053) - 2 call sites:

- `GliderPRO/Sources/HighScores.c:464` in `NameFilter()`
- `GliderPRO/Sources/HighScores.c:572` in `BannerFilter()`

**`kChord2Sound`** (54, `snd ` 1054) - 1 call site:

- `GliderPRO/Sources/Settings.c:656` in `HandleSoundMusicChange()`

**`kPhoneRingSound`** (55, `snd ` 1055) - 1 call site:

- `GliderPRO/Sources/Play.c:755` in `HandleTelephone()`

**`kChime1Sound`** (56, `snd ` 1056) - 1 call site:

- `GliderPRO/Sources/Play.c:776` in `HandleTelephone()`

**`kChime2Sound`** (57, `snd ` 1057) - 1 call site:

- `GliderPRO/Sources/Play.c:778` in `HandleTelephone()`

**`kWebTwangSound`** (58, `snd ` 1058) - 1 call site:

- `GliderPRO/Sources/Interactions.c:1760` in `WebGlider()`

**`kTransOutSound`** (59, `snd ` 1059) - 4 call sites:

- `GliderPRO/Sources/Modes.c:161` in `StartGliderMailingIn()`
- `GliderPRO/Sources/Modes.c:219` in `StartGliderDuctingDown()`
- `GliderPRO/Sources/Modes.c:252` in `StartGliderDuctingUp()`
- `GliderPRO/Sources/Modes.c:294` in `StartGliderTransporting()`

**`kTransInSound`** (60, `snd ` 1060) - 4 call sites:

- `GliderPRO/Sources/Player.c:264` in `TransportGliderIn()`
- `GliderPRO/Sources/Player.c:857` in `FinishGliderMailingLeft()`
- `GliderPRO/Sources/Player.c:895` in `FinishGliderMailingRight()`
- `GliderPRO/Sources/Player.c:933` in `FinishGliderDuctingIn()`

**`kBonusSound`** (61, `snd ` 1061) - 1 call site:

- `GliderPRO/Sources/Interactions.c:924` in `HandleRewards()`

**`kHissSound`** (62, `snd ` 1062) - 1 call site:

- `GliderPRO/Sources/Input.c:176` in `DoHeliumEngaged()`

**`kTriggerSound`** (63, no application resource) - 2 call sites:

- `GliderPRO/Sources/Interactions.c:1072` in `HandleSwitches()`
- `GliderPRO/Sources/Interactions.c:1618` in `HandleHotSpotCollision()`

### 6.3 The per-room custom sound: `kSoundTrigger`

Glider PRO lets a house author attach one arbitrary sound to one room. The
mechanism spans five files.

**The object.** `kSoundTrigger` = `0x49` = 73
(`GliderPRO/Headers/GliderDefines.h:385`). Its `objectType.data` uses the
`switchType` variant (`GliderPRO/Headers/GliderStructs.h:45-52`,
`GliderPRO/Headers/GliderStructs.h:99`):

| Offset in `objectType` | Size | Field | Meaning for `kSoundTrigger` |
|---:|---:|---|---|
| +0 | 2 | `short what` | 0x0049 |
| +2 | 4 | `Point topLeft` | top-left of the 48x48 trigger area, in room coordinates |
| +6 | 2 | `short delay` | unused |
| +8 | 2 | `short where` | **the `'snd '` resource id to play** (3000..32767) |
| +10 | 1 | `Byte who` | unused |
| +11 | 1 | `Byte type` | unused |

`objectType` is 12 bytes total (`GliderPRO/Headers/GliderStructs.h:90-105`), with
the 10-byte `switchType` in the union. Rooms hold `objects[kMaxRoomObs]` =
`objects[24]` at offset 60 within the 348-byte `roomType`
(`GliderPRO/Headers/GliderStructs.h:166-180`).

**Authoring limits.** `kMaxSoundTriggers` = 1
(`GliderPRO/Sources/ObjectAdd.c:17`); the editor refuses to add a second one to a
room:

```c
if ((what == kSoundTrigger) && (HowManySoundObjects() >= kMaxSoundTriggers))   // ObjectAdd.c:484
```

`HowManySoundObjects()` (`GliderPRO/Sources/ObjectAdd.c:989-999`) counts
`kSoundTrigger` objects in the current room. New objects default to
`data.e.where = 3000` (`GliderPRO/Sources/ObjectAdd.c:499`). The info dialog
validates the id with `if ((wasPict < 3000L) || (wasPict > 32767L))`
(`GliderPRO/Sources/ObjectInfo.c:1237`) and prompts with
`ParamText(numberStr, kindStr, "\pSound", "\p3000")`
(`GliderPRO/Sources/ObjectInfo.c:1188`). So **3000 is the floor and 32767 the
ceiling** for a custom sound id.

**Hot spot creation.** `CreateActiveRects()`
(`GliderPRO/Sources/ObjectRects.c:923-928`):

```c
case kSoundTrigger:
QSetRect(&bounds, 0, 0, 48, 48);                                      // ObjectRects.c:924
QOffsetRect(&bounds, theObject.data.e.topLeft.h,
                     theObject.data.e.topLeft.v);                     // ObjectRects.c:925
if (LoadTriggerSound(theObject.data.e.where) == noErr)                // ObjectRects.c:926
    hotSpotNumber = AddActiveRect(&bounds, kSoundIt, who, true, false);  // ObjectRects.c:927
break;
```

The trigger area is a fixed **48 x 48** rectangle. If `LoadTriggerSound` fails
(resource missing, or slot 63 already full) **no hot spot is created at all**, so
the object is completely inert - not even a fallback sound.

`kSoundIt` = 27 (`GliderPRO/Headers/GliderDefines.h:309`) is the hot-spot action
code. `AddActiveRect` refuses to add beyond `kMaxHotSpots` = 56
(`GliderPRO/Headers/GliderDefines.h:259`, enforced at
`GliderPRO/Sources/ObjectRects.c:279-280`).

**When the load happens.** `ListOneRoomsObjects()` only calls
`CreateActiveRects()` for the *central* room:

```c
if ((where == kCentralRoom) && (IsThisValid(roomNum, n)))             // Objects.c:283
    masterObjects[numMasterObjects].hotNum = CreateActiveRects(n);    // Objects.c:284
```

so `LoadTriggerSound` is invoked exactly once per room entry, for the room the
player is in. `ListAllLocalObjects()` (`GliderPRO/Sources/Objects.c:300-328`)
drives that, and is called from `DrawLocale()`
(`GliderPRO/Sources/RoomGraphics.c:72`).

**The room-change sequence** (`GliderPRO/Sources/RoomGraphics.c:44-72`), in order:

```
 1  ZeroFlamesAndTheLike()
 2  ZeroDinahs()
 3  KillAllBands()
 4  ZeroMirrorRegion()
 5  ZeroTriggers()
 6  numTempManholes = 0
 7  FlushAnyTriggerPlaying()        // RoomGraphics.c:57  quiet+flush any voice at 999
 8  DumpTriggerSound()              // RoomGraphics.c:58  DisposePtr(theSoundData[63])
 9  tvInRoom = false
    ... (background, tiles, objects drawn)
10  ListAllLocalObjects()           // RoomGraphics.c:72  -> LoadTriggerSound() for the new room
```

**Playback.** Two sites, both in `Interactions.c`:

```c
case kSoundIt:                                                        // Interactions.c:1615
if (!who->stillOver)                                                  // Interactions.c:1616
{
    PlayPrioritySound(kTriggerSound, kTriggerPriority);               // Interactions.c:1618
    who->stillOver = true;                                            // Interactions.c:1619
}
```

The `stillOver` latch means the sound fires **once on entry** into the 48x48
rectangle and does not retrigger while the glider stays inside it. The other site
is a switch wired to the sound trigger:

```c
case kSoundTrigger:                                                   // Interactions.c:1071
PlayPrioritySound(kTriggerSound, kTriggerPriority);                   // Interactions.c:1072
```

**The remote-trigger wart.** `FireTrigger()` in `Triggers.c` handles a *linked*
trigger firing an object elsewhere. For `kSoundTrigger` it plays the wrong thing:

```c
case kSoundTrigger:
PlayPrioritySound(kChordSound, kChordPriority);   // Change me     Triggers.c:131-133
```

The author's own `// Change me` comment. A remotely-fired sound trigger plays the
generic guitar chord (`snd ` 1023, priority 302) instead of the house's custom
sound. A faithful port must reproduce this.

### 6.4 House `'snd '` resources actually shipped

`GliderPRO/Houses/` holds 37 files: **22 `.binhex` house files** plus 15 `.mov`
QuickTime movies that are not houses. The 22 `.binhex` files are BinHex 4.0
wrappers; the data fork is the
house (rooms and objects) and the resource fork carries `PICT`s, icons and
`'snd '`s. `OpenHouseResFork()` makes the house the current resource file:

```c
houseResFork = FSpOpenResFile(&theHousesSpecs[thisHouseIndex], fsCurPerm);   // HouseIO.c:571
...
UseResFile(houseResFork);                                                    // HouseIO.c:575
```

so `GetResource('snd ', 3000+)` searches the house first, then the application.
Since the application has nothing at 3000+, only house sounds can match.

**13 of the 22 houses carry `'snd '` resources; 63 resources in total.** (The
other nine - Castle o' the Air, Empty House, Fun House, Land of Illusion,
Metropolis, Sampler, Slumberland, Teddy World, The Asylum Pro - have none.) All were
parsed with `tools/probe_snd.py houses`; the observed values:

| house | `snd ` ID | resource name | res bytes | encode | frames | rate (Hz) | ms | rate (Fixed) | baseFreq | trigger objs | notes |
|---|---:|---|---:|---|---:|---:|---:|---|---:|---:|---|
| Art Museum | 3000 | A-hem! | 13573 | stdSH | 13531 | 22254.5455 | 608.0 | 0x56EE8BA3 | 60 | 1 | - |
| Art Museum | 3001 | Elevator | 37546 | stdSH | 37504 | 22254.5455 | 1685.2 | 0x56EE8BA3 | 60 | 1 | - |
| Art Museum | 3002 | Security | 49706 | stdSH | 49664 | 22254.5455 | 2231.6 | 0x56EE8BA3 | 60 | 1 | - |
| Art Museum | 3003 | Shhhh! | 11103 | stdSH | 11061 | 22254.5455 | 497.0 | 0x56EE8BA3 | 60 | 0 | - |
| Art Museum | 3004 | Tired | 48942 | stdSH | 48900 | 22254.5455 | 2197.3 | 0x56EE8BA3 | 60 | 1 | - |
| Art Museum | 3042 | Camera (auto advance) | 10826 | stdSH | 10784 | 22254.5455 | 484.6 | 0x56EE8BA3 | 60 | 5 | - |
| Art Museum | 3043 | What? | 52883 | stdSH | 52841 | 22254.5455 | 2374.4 | 0x56EE8BA3 | 60 | 1 | - |
| Art Museum | 3044 | Warning (naval) | 50928 | stdSH | 50886 | 22254.5455 | 2286.5 | 0x56EE8BA3 | 72 | 3 | - |
| Art Museum | 3045 | Piece of Paper | 47683 | stdSH | 47641 | 22254.5455 | 2140.7 | 0x56EE8BA3 | 60 | 1 | - |
| Art Museum | 3046 | Nothin to See Here | 44080 | stdSH | 44038 | 22254.5455 | 1978.8 | 0x56EE8BA3 | 60 | 1 | - |
| CD Demo House | 3000 | Rattlesnake | 36394 | stdSH | 36352 | 22254.5455 | 1633.5 | 0x56EE8BA3 | 60 | 1 | - |
| CD Demo House | 3001 | Hawk | 38154 | stdSH | 38112 | 22254.5455 | 1712.5 | 0x56EE8BA3 | 60 | 1 | - |
| CD Demo House | 3002 | Bell 1 | 24490 | stdSH | 24448 | 22254.5455 | 1098.6 | 0x56EE8BA3 | 60 | 1 | - |
| CD Demo House | 3003 | Clang Clang Clang | 18465 | stdSH | 18423 | 22254.5455 | 827.8 | 0x56EE8BA3 | 72 | 1 | - |
| CD Demo House | 3004 | Blackbird | 4298 | stdSH | 4256 | 22254.5454 | 191.2 | 0x56EE8B9F | 72 | 1 | - |
| CD Demo House | 3005 | A-hem! | 13573 | stdSH | 13531 | 22254.5455 | 608.0 | 0x56EE8BA3 | 60 | 1 | - |
| CD Demo House | 3006 | Elevator | 37546 | stdSH | 37504 | 22254.5455 | 1685.2 | 0x56EE8BA3 | 60 | 1 | - |
| CD Demo House | 3007 | Door Chime | 3454 | cmpSH | 3370 | 22254.5455 | 908.6 | 0x56EE8BA3 | 60 | 1 | MACE 6:1 |
| CD Demo House | 3008 | Fly Buzz | 14033 | stdSH | 13991 | 7418.1818 | 1886.0 | 0x1CFA2E8B | 72 | 1 | - |
| CD Demo House | 3042 | Camera (auto advance) | 10826 | stdSH | 10784 | 22254.5455 | 484.6 | 0x56EE8BA3 | 60 | 1 | - |
| California or Bust! | 3001 | Glypha Bird | 7978 | stdSH | 7936 | 22254.5455 | 356.6 | 0x56EE8BA3 | 60 | 1 | - |
| California or Bust! | 3002 | Glider 4.0 Bonus | 9342 | stdSH | 9300 | 22254.5455 | 417.9 | 0x56EE8BA3 | 60 | 1 | loop 0..4650 |
| California or Bust! | 3003 | Clang Clang Clang | 18465 | stdSH | 18423 | 22254.5455 | 827.8 | 0x56EE8BA3 | 72 | 1 | - |
| Davis Station | 3000 | Rattlesnake | 36394 | stdSH | 36352 | 22254.5455 | 1633.5 | 0x56EE8BA3 | 60 | 1 | - |
| Davis Station | 3001 | Hawk | 38154 | stdSH | 38112 | 22254.5455 | 1712.5 | 0x56EE8BA3 | 60 | 1 | - |
| Davis Station | 3002 | Bell 1 | 24490 | stdSH | 24448 | 22254.5455 | 1098.6 | 0x56EE8BA3 | 60 | 1 | - |
| Davis Station | 3003 | Clang Clang Clang | 18465 | stdSH | 18423 | 22254.5455 | 827.8 | 0x56EE8BA3 | 72 | 1 | - |
| Davis Station | 3004 | Blackbird | 4298 | stdSH | 4256 | 22254.5454 | 191.2 | 0x56EE8B9F | 72 | 1 | - |
| Demo House | 3011 | Meow | 2387 | cmpSH | 2303 | 22254.5455 | 620.9 | 0x56EE8BA3 | 60 | 1 | MACE 6:1 |
| Grand Prix | 3000 | V8 startup | 198341 | stdSH | 198239 | 22255.0000 | 8907.6 | 0x56EF0000 | 60 | 1 | **loopEnd 198280 > length 198239**, 60 slack bytes |
| Grand Prix | 3001 | Goodbye! | 83815 | stdSH | 83773 | 22050.0000 | 3799.2 | 0x56220000 | 60 | 1 | - |
| Grand Prix | 3002 | Ahooga | 14634 | stdSH | 14592 | 22254.5455 | 655.7 | 0x56EE8BA3 | 60 | 1 | - |
| ImagineHouse PRO II | 3001 | Sawing Steel | 63491 | stdSH | 63449 | 22254.5455 | 2851.1 | 0x56EE8BA3 | 0 | 1 | - |
| ImagineHouse PRO II | 3003 | Touchdown | 15530 | stdSH | 15488 | 22254.5455 | 695.9 | 0x56EE8BA3 | 60 | 1 | - |
| In The Mirror | 3001 | Krusty Laugh | 11706 | stdSH | 11664 | 11127.2727 | 1048.2 | 0x2B7745D1 | 60 | 3 | - |
| In The Mirror | 3002 | Glass breaking | 13558 | stdSH | 13516 | 11127.2727 | 1214.7 | 0x2B7745D1 | 60 | 2 | - |
| Leviathan | 3000 | Yeah! | 17962 | stdSH | 17920 | 22254.5455 | 805.2 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3001 | Flush! | 195836 | stdSH | 195794 | 22254.5455 | 8797.9 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3003 | Hey Lucy, I'm home | 53290 | stdSH | 53248 | 22254.5455 | 2392.7 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3004 | Zzzt! | 14890 | stdSH | 14848 | 22254.5455 | 667.2 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3005 | Photon Torpedoes | 27756 | stdSH | 27714 | 22254.5455 | 1245.3 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3006 | Transporter | 124458 | stdSH | 124416 | 22254.5455 | 5590.6 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3007 | Applause | 37710 | stdSH | 37668 | 11127.2727 | 3385.2 | 0x2B7745D1 | 60 | 1 | - |
| Leviathan | 3008 | Exit, Stage Right | 38340 | stdSH | 38298 | 11127.2727 | 3441.8 | 0x2B7745D1 | 60 | 1 | - |
| Leviathan | 3009 | Ho ho ho... | 20010 | stdSH | 19968 | 22254.5455 | 897.3 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3010 | Ricochet | 1460 | stdSH | 1418 | 5563.6364 | 254.9 | 0x15BBA2E8 | 60 | 1 | - |
| Leviathan | 3011 | Jet Take-off | 159620 | stdSH | 159578 | 22254.5455 | 7170.6 | 0x56EE8BA3 | 60 | 1 | - |
| Leviathan | 3012 | Now Availible on video cassette | 68650 | stdSH | 68608 | 22254.5455 | 3082.9 | 0x56EE8BA3 | 60 | 1 | - |
| Nemo's Market | 3001 | Door Chime | 3454 | cmpSH | 3370 | 22254.5455 | 908.6 | 0x56EE8BA3 | 60 | 2 | MACE 6:1 |
| Nemo's Market | 3002 | Fly Buzz | 14033 | stdSH | 13991 | 7418.1818 | 1886.0 | 0x1CFA2E8B | 72 | 3 | - |
| Nemo's Market | 3003 | Cash Register | 4350 | cmpSH | 4266 | 22254.5455 | 1150.1 | 0x56EE8BA3 | 60 | 1 | MACE 6:1 |
| Nemo's Market | 3004 | Cat Meow | 2387 | cmpSH | 2303 | 22254.5455 | 620.9 | 0x56EE8BA3 | 60 | 2 | MACE 6:1 |
| Nemo's Market | 3005 | Bird Chirp | 1488 | stdSH | 1446 | 7418.1818 | 194.9 | 0x1CFA2E8B | 60 | 3 | - |
| Rainbow's End | 3000 | Curly Woob-Woob | 23573 | stdSH | 23531 | 11127.5000 | 2114.7 | 0x2B778000 | 60 | 2 | - |
| SpacePods | 3000 | Squawk | 6532 | stdSH | 6490 | 22254.5455 | 291.6 | 0x56EE8BA3 | 60 | 14 | - |
| SpacePods | 3001 | Bubbles | 5416 | stdSH | 5374 | 11127.2727 | 483.0 | 0x2B7745D1 | 60 | 6 | - |
| SpacePods | 3002 | Organ | 6760 | stdSH | 6716 | 9779.0000 | 686.8 | 0x26330000 | 60 | 4 | **loopEnd 6759 > length 6716**, 2 slack bytes |
| Titanic | 3000 | Splash | 36645 | stdSH | 36603 | 22254.5455 | 1644.7 | 0x56EE8BA3 | 60 | 3 | - |
| Titanic | 3006 | Bubbles | 5416 | stdSH | 5374 | 11127.2727 | 483.0 | 0x2B7745D1 | 60 | 11 | - |
| Titanic | 3032 | Flourish | 16189 | stdSH | 16147 | 22254.5455 | 725.6 | 0x56EE8BA3 | 60 | 0 | - |
| Titanic | 3037 | Horn (fog) | 68538 | stdSH | 68496 | 22254.5455 | 3077.8 | 0x56EE8BA3 | 60 | 1 | - |
| Titanic | 3058 | Metal Clang | 23018 | stdSH | 22976 | 22254.5455 | 1032.4 | 0x56EE8BA3 | 60 | 1 | - |
| Titanic | 3061 | Whistle (boat) | 44810 | stdSH | 44768 | 22254.5455 | 2011.6 | 0x56EE8BA3 | 60 | 0 | - |

House sound observations:

- **58 of 63 are `stdSH`** (uncompressed 8-bit) and **5 are `cmpSH`** MACE 6:1
  (`compressionID` 4, `packetSize` 8 bits, `initOption` 0x000004A0 which adds
  `initMACE6` = 0x400 to the usual 0xA0): CD Demo House 3007 "Door Chime"
  (3370 packets), Demo House 3011 "Meow" (2303 packets), Nemo's Market 3001
  "Door Chime" (3370), 3003 "Cash Register" (4266), 3004 "Cat Meow" (2303).
  For a `cmpSH` the frame count in the table is *packets*; the sample count is
  `packets * 6`, and the duration column already accounts for that.
- Nine distinct sample rates occur (section 3.5); the majority are the usual
  22254.5455 Hz but there are 11 kHz, 9779 Hz, 7418 Hz and 5563 Hz sounds.
- Exactly **one** genuine loop: California or Bust! 3002 "Glider 4.0 Bonus",
  `loopStart` 0, `loopEnd` 4650. Never honoured (section 3.10).
- Two resources have a `loopEnd` past the end of their own sample data:
  Grand Prix 3000 "V8 startup" (`loopEnd` 198280, `length` 198239, plus 60 bytes
  of slack after the samples) and SpacePods 3002 "Organ" (`loopEnd` 6759,
  `length` 6716, 2 bytes of slack). The `length` field is authoritative; a port
  must clamp to it, not trust `loopEnd`.
- `baseFrequency` is 60 for 54 of them, 72 for 8, and 0 for one.
- Exactly **three** of the 63 are present but never referenced by any
  `kSoundTrigger` object (the "trigger objs" column above is 0): Art Museum 3003
  "Shhhh!", Titanic 3032 "Flourish" and Titanic 3061 "Whistle (boat)". They are
  leftovers from authoring. (Every other shipped resource has at least one
  referencing object. Leviathan has no `'snd '` 3002 at all - its ids run
  3000, 3001, 3003..3012.)
- The same sound is duplicated across houses under the same id: "Rattlesnake"
  3000, "Hawk" 3001, "Bell 1" 3002, "Clang Clang Clang" 3003 and "Blackbird" 3004
  appear byte-identically in both CD Demo House and Davis Station; "A-hem!",
  "Elevator" and "Camera (auto advance)" are shared between Art Museum and CD
  Demo House; "Fly Buzz" between CD Demo House and Nemo's Market; "Bubbles"
  between SpacePods and Titanic.

### 6.5 `kSoundTrigger` object census, and the 13 broken ones

Every house data fork was walked with python3: `houseType` header is 866 bytes,
then `rooms[]` of 348 bytes each (`GliderPRO/Headers/GliderStructs.h:166-198`);
each room's `numObjects` is at room offset +58 and `objects[24]` of 12 bytes each
start at room offset +60. Scanning for `what == 0x0049` and reading
`data.e.where` at object offset +8:

| house | rooms | `kSoundTrigger` objects | `data.e.where` values (count) | resolves? |
|---|---:|---:|---|---|
| Art Museum | 109 | 15 | 3000 (x1), 3001 (x1), 3002 (x1), 3004 (x1), 3042 (x5), 3043 (x1), 3044 (x3), 3045 (x1), 3046 (x1) | all present |
| CD Demo House | 206 | 10 | 3000 (x1), 3001 (x1), 3002 (x1), 3003 (x1), 3004 (x1), 3005 (x1), 3006 (x1), 3007 (x1), 3008 (x1), 3042 (x1) | all present |
| California or Bust! | 16 | 3 | 3001 (x1), 3002 (x1), 3003 (x1) | all present |
| Davis Station | 65 | 5 | 3000 (x1), 3001 (x1), 3002 (x1), 3003 (x1), 3004 (x1) | all present |
| Demo House | 45 | 1 | 3011 (x1) | all present |
| Grand Prix | 175 | 3 | 3000 (x1), 3001 (x1), 3002 (x1) | all present |
| ImagineHouse PRO II | 279 | 2 | 3001 (x1), 3003 (x1) | all present |
| In The Mirror | 97 | 7 | 3001 (x3), 3002 (x2), 10000 (x2) | **10000 absent from fork** |
| Leviathan | 472 | 12 | 3000 (x1), 3001 (x1), 3003 (x1), 3004 (x1), 3005 (x1), 3006 (x1), 3007 (x1), 3008 (x1), 3009 (x1), 3010 (x1), 3011 (x1), 3012 (x1) | all present |
| Nemo's Market | 124 | 11 | 3001 (x2), 3002 (x3), 3003 (x1), 3004 (x2), 3005 (x3) | all present |
| Rainbow's End | 223 | 2 | 3000 (x2) | all present |
| SpacePods | 402 | 24 | 3000 (x14), 3001 (x6), 3002 (x4) | all present |
| Teddy World | 531 | 11 | 3000 (x11) | **3000 absent from fork** |
| Titanic | 208 | 16 | 3000 (x3), 3006 (x11), 3037 (x1), 3058 (x1) | all present |
| **total** | | **122** | | **13 objects point at an absent resource** |

**13 of the 122 shipped sound triggers are broken:**

- **In The Mirror**, 2 objects with `data.e.where = 10000`. The house fork has
  only `'snd '` 3001 "Krusty Laugh" and 3002 "Glass breaking". 10000 is outside
  the range the editor's own validator accepts
  (`GliderPRO/Sources/ObjectInfo.c:1237` requires 3000..32767, so 10000 *is*
  legal there) but there is no such resource; note 10000..32767 is also the range
  used for **custom PICTs** (`GliderPRO/Sources/ObjectInfo.c:1213`), so this
  looks like an author who pasted a picture id into a sound field.
- **Teddy World**, 11 objects with `data.e.where = 3000`. Teddy World's resource
  fork contains **zero** `'snd '` resources of any kind (only `PICT`, `ICN#`,
  `icl8`, `icl4`, `ics#`, `ics8`, `ics4`, `vers`). All 11 triggers are dead.

In every case `GetResource` returns nil, `LoadTriggerSound` returns -1
(`GliderPRO/Sources/Sound.c:280-282`), `CreateActiveRects` skips `AddActiveRect`
(`GliderPRO/Sources/ObjectRects.c:926-927`), and the object silently does nothing.
A Go port must reproduce that silence rather than substituting a default sound.

## 7. The music engine (`GliderPRO/Sources/Music.c`)

### 7.1 What "music" is

Seven sampled pieces, `'snd '` 2000..2006, in exactly the same format as the
sound effects (8-bit unsigned mono at 22254.5455 Hz, `stdSH`). There is no MIDI,
no sequencer, no synthesis: the "score" is a list of indices into seven
pre-rendered wave files, played back to back.

| Music index | `'snd '` id | Resource name | frames | ms | loopStart | loopEnd | Constant that names it |
|---:|---:|---|---:|---:|---:|---:|---|
| 0 | 2000 | Refrain1.22 | 97034 | 4360.2 | 0 | 97034 | - |
| 1 | 2001 | Refrain2.22 | 97102 | 4363.2 | 0 | 97102 | - |
| 2 | 2002 | Refrain3.22 | 97032 | 4360.1 | 0 | 97032 | - |
| 3 | 2003 | Refrain4.22 | 97284 | 4371.4 | 0 | 97284 | - |
| 4 | 2004 | Chorus.22 | 194028 | 8718.6 | 0 | 194028 | `kPlayChorus` = 4 |
| 5 | 2005 | RefrainSparse1.22 | 96888 | 4353.6 | 0 | 96888 | `kPlayRefrainSparse1` = 5 |
| 6 | 2006 | RefrainSparse2.22 | 97334 | 4373.7 | 0 | 97334 | `kPlayRefrainSparse2` = 6 |

Total 776,996 resource bytes; 776,856 bytes retained after the `-20` strip.
Every piece is about 4.36 s except the Chorus, which is exactly twice as long
(194028 is not exactly 2 x 97034, but 8.72 s vs 4.36 s musically means two bars
against one). All seven carry `loopStart = 0`, `loopEnd = length` - i.e. "loop the
whole thing" - which, as established in section 3.10, the engine never honours;
looping is done by re-queueing.

The three named constants are at `GliderPRO/Headers/GliderDefines.h:51-53`. Note
that they double as *music modes* (see 7.4) because the mode value and the music
index share one `short`.

### 7.2 Constants and globals

| Constant | Value | Line |
|---|---:|---|
| `kBaseBufferMusicID` | 2000 | `GliderPRO/Sources/Music.c:15` |
| `kMaxMusic` | 7 | `GliderPRO/Sources/Music.c:16` |
| `kLastMusicPiece` | 16 | `GliderPRO/Sources/Music.c:17` |
| `kLastGamePiece` | 6 | `GliderPRO/Sources/Music.c:18` |
| `kNoMemForMusicAlert` | 1038 | `GliderPRO/Sources/Music.c:413` |
| `kProdGameScoreMode` | -4 | `GliderPRO/Headers/GliderDefines.h:47` |
| `kKickGameScoreMode` | -3 | `GliderPRO/Headers/GliderDefines.h:48` |
| `kPlayGameScoreMode` | -2 | `GliderPRO/Headers/GliderDefines.h:49` |
| `kPlayWholeScoreMode` | -1 | `GliderPRO/Headers/GliderDefines.h:50` |
| `kPlayChorus` | 4 | `GliderPRO/Headers/GliderDefines.h:51` |
| `kPlayRefrainSparse1` | 5 | `GliderPRO/Headers/GliderDefines.h:52` |
| `kPlayRefrainSparse2` | 6 | `GliderPRO/Headers/GliderDefines.h:53` |
| `kYellowNoMusic` | 13 | `GliderPRO/Headers/GliderDefines.h:30` |

```c
SndCallBackUPP  musicCallBackUPP;                   // Music.c:28
SndChannelPtr   musicChannel;                       // Music.c:29
Ptr             theMusicData[kMaxMusic];            // Music.c:30   7 slots
short           musicSoundID, musicCursor;          // Music.c:31
short           musicScore[kLastMusicPiece];        // Music.c:32   16 entries
short           gameScore[kLastGamePiece];          // Music.c:33   6 entries
short           musicMode;                          // Music.c:34
Boolean         isMusicOn, isPlayMusicIdle, isPlayMusicGame;  // Music.c:35
Boolean         failedMusic, dontLoadMusic;         // Music.c:36
extern Boolean  isSoundOn;                          // Music.c:38  declared, never used here
```

| Global | Meaning |
|---|---|
| `musicSoundID` | index 0..6 of the piece to queue **next** |
| `musicCursor` | position in whichever score is active |
| `musicScore[16]` | the "whole score" - the idle/menu music |
| `gameScore[6]` | the in-game score, with a back-jump sentinel |
| `musicMode` | one of `kPlayWholeScoreMode` (-1) or `kPlayGameScoreMode` (-2); see 7.4 |
| `isMusicOn` | music is currently streaming |
| `isPlayMusicIdle` | user preference: play music outside a game |
| `isPlayMusicGame` | user preference: play music during a game (also toggled by the in-room stereo object) |
| `failedMusic` | load or channel failure; music permanently off |
| `dontLoadMusic` | not enough RAM at launch |

`extern Boolean isSoundOn;` at `GliderPRO/Sources/Music.c:38` is never referenced
in the file - music volume is checked via `UnivGetSoundVolume` instead
(`GliderPRO/Sources/Music.c:55`). So **music plays even when `isSoundOn` is false**,
as long as the system volume is non-zero... except `isSoundOn` is defined as
`(newVolume != 0)` (`GliderPRO/Sources/Settings.c:635`), so in practice the two
agree and the distinction never shows.

### 7.3 The two scores

`InitMusic()` fills them element-by-element at
`GliderPRO/Sources/Music.c:332-354`:

```
musicScore[16] = { 0, 1, 2, 3, 4, 4, 0, 1, 2, 3, 4, 4, 5, 6, 4, 4 }
                                                    ^  ^     Chorus, Chorus
                   |  |  |  |  |  |                       (indices 10,11)
                   Refrain1..4, Chorus, Chorus
gameScore[6]   = { 6, 5, -1, 6, 4, 4 }
                        ^^ back-jump sentinel
```

Spelled out with the constants used in the source:

| i | `musicScore[i]` | piece | source line |
|---:|---:|---|---|
| 0 | 0 | Refrain1 | `GliderPRO/Sources/Music.c:332` |
| 1 | 1 | Refrain2 | `:333` |
| 2 | 2 | Refrain3 | `:334` |
| 3 | 3 | Refrain4 | `:335` |
| 4 | 4 | Chorus | `:336` |
| 5 | 4 | Chorus | `:337` |
| 6 | 0 | Refrain1 | `:338` |
| 7 | 1 | Refrain2 | `:339` |
| 8 | 2 | Refrain3 | `:340` |
| 9 | 3 | Refrain4 | `:341` |
| 10 | `kPlayChorus` = 4 | Chorus | `:342` |
| 11 | `kPlayChorus` = 4 | Chorus | `:343` |
| 12 | `kPlayRefrainSparse1` = 5 | RefrainSparse1 | `:344` |
| 13 | `kPlayRefrainSparse2` = 6 | RefrainSparse2 | `:345` |
| 14 | `kPlayChorus` = 4 | Chorus | `:346` |
| 15 | `kPlayChorus` = 4 | Chorus | `:347` |

| i | `gameScore[i]` | piece | source line |
|---:|---:|---|---|
| 0 | `kPlayRefrainSparse2` = 6 | RefrainSparse2 | `GliderPRO/Sources/Music.c:349` |
| 1 | `kPlayRefrainSparse1` = 5 | RefrainSparse1 | `:350` |
| 2 | -1 | **sentinel: jump back 1** | `:351` |
| 3 | `kPlayRefrainSparse2` = 6 | RefrainSparse2 | `:352` |
| 4 | `kPlayChorus` = 4 | Chorus | `:353` |
| 5 | `kPlayChorus` = 4 | Chorus | `:354` |

Duration of one full pass of `musicScore` (indices 0..14, because the whole-score
callback wraps at `kLastMusicPiece - 1` = 15, see 7.6):
4360.2 + 4363.2 + 4360.1 + 4371.4 + 8718.6 + 8718.6 + 4360.2 + 4363.2 + 4360.1 +
4371.4 + 8718.6 + 8718.6 + 4353.6 + 4373.7 + 8718.6 = **87,230.1 ms**, about
1 minute 27 seconds. (Index 15, another Chorus, is only ever reached by
`StartMusic`, never by the whole-score callback - see 7.6.)

### 7.4 The four "musical modes"

`SetMusicalMode(short newMode)` (`GliderPRO/Sources/Music.c:146-166`) is the only
way the rest of the game influences the music. It has **three** branches for
**four** constants:

```
 1  SetMusicalMode(newMode):
 2      if dontLoadMusic: return                     // Music.c:148-149
 3      switch newMode:
 4          case kKickGameScoreMode (-3):            // Music.c:153
 5              musicCursor = 2                      // Music.c:154   musicMode UNCHANGED
 6          case kProdGameScoreMode (-4):            // Music.c:157
 7              musicCursor = -1                     // Music.c:158   musicMode UNCHANGED
 8          default:                                 // Music.c:161
 9              musicMode = newMode                  // Music.c:162
10              musicCursor = 0                      // Music.c:163
```

So:

| Constant | Value | Effect |
|---|---:|---|
| `kPlayWholeScoreMode` | -1 | `musicMode = -1`, `musicCursor = 0`: play `musicScore` from the top |
| `kPlayGameScoreMode` | -2 | `musicMode = -2`, `musicCursor = 0`: play `gameScore` |
| `kKickGameScoreMode` | -3 | *seek only*: `musicCursor = 2`. Mode unchanged. |
| `kProdGameScoreMode` | -4 | *seek only*: `musicCursor = -1`. Mode unchanged. |

"Kick" and "Prod" are therefore not modes at all - they are cursor seeks inside
the currently-selected score, used to make the music respond to the player moving
between rooms. Because they never assign `musicMode`, **`musicMode` only ever
holds -1 or -2** (initial value `kPlayWholeScoreMode` at
`GliderPRO/Sources/Music.c:358`; the only other assignment is the `default` branch
at `:162`, reached only from the two call sites that pass -1 and -2). That makes
the `default:` case of `MusicCallBack` (`GliderPRO/Sources/Music.c:200-202`,
`musicSoundID = musicMode`, i.e. "loop one fixed piece forever") **dead code** in
the shipped game. It would only come alive if someone called
`SetMusicalMode(kPlayChorus)`, which nothing does.

Every `SetMusicalMode` call site in the tree - there are exactly 10:

| Site | Argument | When |
|---|---|---|
| `GliderPRO/Sources/Play.c:95` | `kPlayGameScoreMode` | a game starts, if `isPlayMusicGame` |
| `GliderPRO/Sources/Play.c:246` | `kPlayWholeScoreMode` | a game ends and we return to splash, if `isPlayMusicIdle` |
| `GliderPRO/Sources/GameOver.c:241` | `kPlayWholeScoreMode` | `FlagGameOver()` - the moment the last glider dies |
| `GliderPRO/Sources/Transit.c:159` | `kProdGameScoreMode` | glider leaves the room to the right (`kToRight`) |
| `GliderPRO/Sources/Transit.c:191` | `kProdGameScoreMode` | glider leaves the room to the left (`kToLeft`) |
| `GliderPRO/Sources/Transit.c:223` | `kKickGameScoreMode` | glider leaves upward (`kAbove`) |
| `GliderPRO/Sources/Transit.c:255` | `kKickGameScoreMode` | glider leaves downward (`kBelow`) |
| `GliderPRO/Sources/Transit.c:318` | `kKickGameScoreMode` | `TransportRoomToRoom()` |
| `GliderPRO/Sources/Transit.c:356` | `kKickGameScoreMode` | `MoveDuctToDuct()` |
| `GliderPRO/Sources/Transit.c:395` | `kKickGameScoreMode` | `MoveMailToMail()` |

So: walking sideways "prods" the score; going up, down, or teleporting "kicks" it.

### 7.5 `StartMusic()` - priming the stream

`GliderPRO/Sources/Music.c:44-94`. Returns `OSErr`.

```
 1  StartMusic():
 2      theErr = noErr
 3      if dontLoadMusic: return noErr                           // Music.c:52-53
 4      UnivGetSoundVolume(&soundVolume, thisMac.hasSM3)          // Music.c:55
 5      if soundVolume == 0 or failedMusic:                       // Music.c:57
 6          return noErr        // NB: isMusicOn is left false
 7
 8      // --- piece A: whatever musicSoundID currently names
 9      cmd = bufferCmd; param1 = 0
10      param2 = (long)theMusicData[musicSoundID]                 // Music.c:61
11      err = SndDoCommand(musicChannel, &cmd, false)             // Music.c:62  QUEUED
12      if err: return err                                       // Music.c:63-64
13
14      // --- a nullCmd used as a queue marker / no-op
15      cmd = 0 (nullCmd); param1 = 1964                          // Music.c:66-67
16      param2 = SetCurrentA5()                                   // Music.c:68
17      err = SndDoCommand(musicChannel, &cmd, false)             // Music.c:69
18      if err: return err                                       // Music.c:70-71
19
20      // --- advance the cursor through musicScore (ALWAYS musicScore!)
21      musicCursor++                                             // Music.c:73
22      if musicCursor >= kLastMusicPiece (16): musicCursor = 0    // Music.c:74-75
23      musicSoundID = musicScore[musicCursor]                    // Music.c:76
24
25      // --- piece B
26      cmd = bufferCmd; param1 = 0
27      param2 = (long)theMusicData[musicSoundID]                 // Music.c:80
28      err = SndDoCommand(musicChannel, &cmd, false)             // Music.c:81  QUEUED
29      if err: return err                                       // Music.c:82-83
30
31      // --- the callback that will keep the stream alive
32      cmd = callBackCmd; param1 = 0
33      param2 = SetCurrentA5()                                   // Music.c:87
34      err = SndDoCommand(musicChannel, &cmd, false)             // Music.c:88  QUEUED
35
36      isMusicOn = true                                          // Music.c:90
37      return err
```

Things to note:

- Everything is **queued** (`SndDoCommand`, `noWait = false`), unlike the sound
  effects which use `SndDoImmediate`. The channel plays A, executes the nullCmd,
  plays B, then fires the callback. There is no gap between A and B because both
  are already in the queue when A starts.
- `param1 = 1964` on the `nullCmd` is a magic constant with no functional effect;
  `nullCmd` (0) does nothing. It is almost certainly a debugging marker (1964 is
  a year). A Go port can drop it, but it does consume one command-queue slot,
  which matters if a port models a fixed-size queue.
- **`StartMusic` always walks `musicScore`, never `gameScore`**
  (`GliderPRO/Sources/Music.c:76`), whatever `musicMode` says. So starting music
  while in game mode plays one piece of the idle score before `MusicCallBack`
  takes over with the game score. This is why `Play.c` calls `StartMusic()`
  *first* and `SetMusicalMode(kPlayGameScoreMode)` *second*
  (`GliderPRO/Sources/Play.c:88`, `:95`) - the `SetMusicalMode` resets
  `musicCursor` to 0 immediately afterwards, discarding the advance that
  `StartMusic` just did, while the two already-queued buffers keep playing.
- The wrap at `kLastMusicPiece` (16) here differs from the callback's wrap at
  `kLastMusicPiece - 1` (15). So index 15 of `musicScore` (a Chorus) is reachable
  only through `StartMusic`.
- If `soundVolume == 0`, `isMusicOn` stays false and nothing is queued. Turning
  the volume up later does not itself restart the music; something has to call
  `StartMusic()` again (`Settings.c:645` does exactly that).
- `StartMusic` does not check `isMusicOn`. Calling it twice would queue four
  buffers and two callbacks, and from then on there would be two independent
  callback chains re-queueing on the same channel - i.e. the music would play
  twice as fast, overlapping. Every caller guards with `if (!isMusicOn)`:
  `GliderPRO/Sources/Play.c:86`, `GliderPRO/Sources/Play.c:237`,
  `GliderPRO/Sources/Settings.c:643`, `GliderPRO/Sources/Settings.c:1247`,
  `GliderPRO/Sources/Music.c:134` (in `ToggleMusicWhilePlaying`). Two call sites do
  **not**: `GliderPRO/Sources/Events.c:420` (resume event) and
  `GliderPRO/Sources/Menu.c:393` (entering splash mode). Both are reached only
  when the music was previously stopped, so the guard is implicit - but it is
  fragile, and a Go port should make `StartMusic` idempotent.

### 7.6 `MusicCallBack()` - the streaming engine

`GliderPRO/Sources/Music.c:170-216`. This runs at interrupt time on a real Mac.

```
 1  MusicCallBack(theChannel, theCommand):        // theChannel unused (Music.c:172)
 2      // gameA5 = theCommand.param2             <-- COMMENTED OUT at Music.c:176
 3      // thisA5 = SetA5(gameA5)                 <-- COMMENTED OUT at Music.c:177
 4
 5      switch musicMode:
 6          case kPlayGameScoreMode (-2):                        // Music.c:181
 7              musicCursor++                                    // Music.c:182
 8              if musicCursor >= kLastGamePiece (6):
 9                  musicCursor = 1                              // Music.c:183-184
10              musicSoundID = gameScore[musicCursor]            // Music.c:185
11              if musicSoundID < 0:                             // Music.c:186
12                  musicCursor += musicSoundID   // back-jump    // Music.c:188
13                  musicSoundID = gameScore[musicCursor]        // Music.c:189
14              break
15          case kPlayWholeScoreMode (-1):                        // Music.c:193
16              musicCursor++                                    // Music.c:194
17              if musicCursor >= kLastMusicPiece - 1 (15):
18                  musicCursor = 0                              // Music.c:195-196
19              musicSoundID = musicScore[musicCursor]           // Music.c:197
20              break
21          default:                                             // Music.c:200  DEAD
22              musicSoundID = musicMode                         // Music.c:201
23              break
24
25      theCommand->cmd = bufferCmd; param1 = 0
26      theCommand->param2 = (long)theMusicData[musicSoundID]    // Music.c:207
27      SndDoCommand(musicChannel, theCommand, false)            // Music.c:208
28
29      theCommand->cmd = callBackCmd; param1 = 0
30      theCommand->param2 = gameA5      // UNINITIALISED         // Music.c:212
31      SndDoCommand(musicChannel, theCommand, false)            // Music.c:213
32
33      thisA5 = SetA5(thisA5)           // UNINITIALISED         // Music.c:215
```

**The uninitialised-variable bug.** Lines `GliderPRO/Sources/Music.c:176-177` are
commented out, so `gameA5` and `thisA5` are never assigned. Line
`GliderPRO/Sources/Music.c:212` writes garbage into the callback's `param2`
(harmless - the next invocation ignores `param2` because line 176 is commented
out) and line `GliderPRO/Sources/Music.c:215` calls `SetA5(thisA5)` with garbage.
On a 68k Mac `SetA5` writes the A5 register, so this sets A5 to whatever junk was
on the stack, at interrupt time, and does *not* restore it - a genuine memory-
corruption hazard on 68k. On PowerPC `SetA5`/`SetCurrentA5` are no-ops, which is
presumably why it shipped. A Go port has no A5 and simply omits all four lines.

**Whole-score walk** (`musicMode == -1`), starting from `musicCursor = 0`:
the callback pre-increments, so the *first* callback plays `musicScore[1]`, and
the cursor wraps at 15 rather than 16:

```
cursor: 0 -> 1 -> 2 -> ... -> 14 -> (15 >= 15) -> 0 -> 1 -> ...
pieces:  Refrain2, Refrain3, Refrain4, Chorus, Chorus, Refrain1, Refrain2,
         Refrain3, Refrain4, Chorus, Chorus, RefrainSparse1, RefrainSparse2,
         Chorus, [wrap] Refrain1, Refrain2, ...
```

so `musicScore[15]` (the 16th entry, a Chorus) is never played by the callback.

**Game-score walk** (`musicMode == -2`). The `-1` at `gameScore[2]` is a
"back up one" sentinel: on reading it the cursor is decremented by 1 and the
entry there is used instead. Traced from each possible entry point:

| Start | Sequence of `(cursor -> piece)` | Steady state |
|---|---|---|
| `SetMusicalMode(kPlayGameScoreMode)` -> cursor 0 | 1 -> RefrainSparse1; then 2 -> sentinel -> cursor 1 -> RefrainSparse1; ... | **RefrainSparse1 forever** |
| `kProdGameScoreMode` -> cursor -1 | 0 -> RefrainSparse2; 1 -> RefrainSparse1; 2 -> sentinel -> 1 -> RefrainSparse1; ... | RefrainSparse1 forever |
| `kKickGameScoreMode` -> cursor 2 | 3 -> RefrainSparse2; 4 -> Chorus; 5 -> Chorus; (6 >= 6) -> 1 -> RefrainSparse1; 2 -> sentinel -> 1 -> RefrainSparse1; ... | RefrainSparse1 forever |

So in-game the music is a sparse drone (`RefrainSparse1`, 4.35 s, looping) that
the player's movement periodically interrupts:

- moving **left or right** ("prod", cursor = -1) inserts one `RefrainSparse2`
  before returning to the drone;
- moving **up or down**, or using a duct / transporter / mailbox ("kick",
  cursor = 2) inserts `RefrainSparse2, Chorus, Chorus` - a 21.8 s musical
  flourish - before returning to the drone.

Crucially the effect is **not immediate**: `SetMusicalMode` only moves the cursor,
and the cursor is read by the *next* callback, which fires only when the currently
playing piece ends. So a "kick" can take up to 8.7 s (a Chorus) to be heard. A
port must reproduce this latency; snapping to the new piece immediately would
sound quite different.

**Queue depth.** After the initial `StartMusic`, the queue holds
`[bufferA, nullCmd, bufferB, callBack]`. When `bufferB` completes the callback
fires and appends `[bufferC, callBack]`. From then on the steady state is one
buffer in flight plus one callback, and the callback appends the next pair. So
after the first two pieces there is exactly one queued command boundary per piece,
and the seam relies on the interrupt being serviced promptly. In Go, the natural
equivalent is a ring buffer or a channel feeding the mixer, filled by a goroutine
that appends the next piece when the previous one is consumed.

### 7.7 `StopTheMusic()`

`GliderPRO/Sources/Music.c:98-121`.

```
 1  if dontLoadMusic: return                                 // Music.c:103-104
 2  if isMusicOn and not failedMusic:                        // Music.c:107
 3      flushCmd (4), 0, 0 -> SndDoImmediate(musicChannel)   // Music.c:109-112
 4      quietCmd (3), 0, 0 -> SndDoImmediate(musicChannel)   // Music.c:114-117
 5      isMusicOn = false                                    // Music.c:119
```

Note the order is **flush then quiet**, the opposite of
`FlushAnyTriggerPlaying()` (`GliderPRO/Sources/Sound.c:96-103`, quiet then
flush). Flushing first removes the pending `bufferCmd`/`callBackCmd` so nothing
new starts, then `quietCmd` cuts off the piece that is currently sounding. This is
the correct order and it means the callback chain is broken - the music genuinely
stops rather than resuming after the current piece.

`musicCursor`, `musicSoundID` and `musicMode` are **not** reset, so a later
`StartMusic()` resumes from wherever the score had got to (plus one).

### 7.8 `ToggleMusicWhilePlaying()`

`GliderPRO/Sources/Music.c:125-142`. Despite the name it is not a toggle - it
*synchronises* the music to `isPlayMusicGame`:

```
 1  if dontLoadMusic: return                    // Music.c:129-130
 2  if isPlayMusicGame:                         // Music.c:132
 3      if not isMusicOn: StartMusic()          // Music.c:134-135
 4  else:
 5      if isMusicOn: StopTheMusic()            // Music.c:139-140
```

Call sites, all four:

| Site | Context |
|---|---|
| `GliderPRO/Sources/Dynamics.c:681` | the in-room stereo just turned **on** (`HandleStereo`, active branch, `timer == 0`) |
| `GliderPRO/Sources/Dynamics.c:698` | the in-room stereo just turned **off** (`HandleStereo`, inactive branch, `timer == 0`) |
| `GliderPRO/Sources/Play.c:413` | application **resume** event during play |
| `GliderPRO/Sources/Play.c:421` | application **suspend** event during play |

The two `Play.c` sites are a bug: both the resume and the suspend handler call the
same function, and since the function only looks at `isPlayMusicGame` (which the
suspend does not change), **suspending the application while game music is on
leaves the music playing in the background**. Compare
`GliderPRO/Sources/Events.c:418-439`, which handles the same pair correctly for
non-play modes (`StartMusic()` on resume if `isPlayMusicIdle`, `StopTheMusic()` on
suspend if `isMusicOn`).

### 7.9 The stereo object - music as a game object

`kStereo` = `0x69` = 105 (`GliderPRO/Headers/GliderDefines.h:404`). The in-room
stereo is wired directly to `isPlayMusicGame`:

- toggling it: `newState = !isPlayMusicGame; isPlayMusicGame = newState;`
  (`GliderPRO/Sources/Objects.c:584-587`);
- reading its state: `theState = isPlayMusicGame;`
  (`GliderPRO/Sources/Objects.c:814-815`);
- initialising the room's copy from the preference:
  `thisHousePtr->rooms[r].objects[i].data.g.state = isPlayMusicGame;`
  (`GliderPRO/Sources/Play.c:676`);
- drawing it lit/unlit: `DrawStereo(&itsRect, isPlayMusicGame, isLit)`
  (`GliderPRO/Sources/ObjectDrawAll.c:769`,
  `GliderPRO/Sources/ObjectEdit.c:2636`).

Animation and sound (`HandleStereo`, `GliderPRO/Sources/Dynamics.c:671-711`):

| Branch | `timer` | Action |
|---|---:|---|
| turning on | 0 | `AddRectToWorkRects(...)`; `ToggleMusicWhilePlaying()` (`Dynamics.c:681`) |
| turning on | 1 | `PlayPrioritySound(kMacOnSound, kMacOnPriority)` (`Dynamics.c:685`); draw `stereoLight2` |
| turning off | 0 | `ToggleMusicWhilePlaying()` (`Dynamics.c:698`) |
| turning off | 1 | `PlayPrioritySound(kMacOffSound, kMacOffPriority)` (`Dynamics.c:702`); draw `stereoLight1` |

So the music change happens one frame *before* the click sound. Because
`isPlayMusicGame` is a global preference rather than per-room state, flipping a
stereo in one room changes the setting for the whole session -
`GliderPRO/Sources/Play.c:193` saves it (`wasPlayMusicPref = isPlayMusicGame`) and
`GliderPRO/Sources/Play.c:220` restores it when the game ends, so the user's
preference survives the game.

### 7.10 `LoadMusicSounds()` / `DumpMusicSounds()`

`GliderPRO/Sources/Music.c:220-251`, `:255-270`. Identical in shape to
`LoadBufferSounds()`, with two differences: all 7 slots are pre-nilled first
(`GliderPRO/Sources/Music.c:229-230`), and the loop bound is `i < kMaxMusic`
(all 7, no reserved slot).

```
 1  for i = 0..6: theMusicData[i] = nil                       // Music.c:229-230
 2  for i = 0..6:                                             // Music.c:232
 3      theSound = GetResource('snd ', i + 2000)               // Music.c:234
 4      if nil: return MemError()                             // Music.c:235-236
 5      HLock; soundDataSize = GetHandleSize(theSound) - 20; HUnlock   // Music.c:238-240
 6      theMusicData[i] = NewPtr(soundDataSize)                // Music.c:242
 7      if nil: return MemError()                             // Music.c:243-244
 8      HLock(theSound)                                       // Music.c:246
 9      BlockMove(*theSound + 20, theMusicData[i], soundDataSize)   // Music.c:247
10      ReleaseResource(theSound)                             // Music.c:248
```

### 7.11 `OpenMusicChannel()` / `CloseMusicChannel()`

`GliderPRO/Sources/Music.c:274-291`, `:295-308`.

```
 1  OpenMusicChannel():
 2      musicCallBackUPP = NewSndCallBackProc(MusicCallBack)      // Music.c:278
 3      if musicChannel != nil: return noErr                      // Music.c:282-283
 4      musicChannel = nil                                        // Music.c:285
 5      return SndNewChannel(&musicChannel, sampledSynth,
 6                           initNoInterp + initMono, musicCallBackUPP)   // Music.c:286-288
```

Same flags as the three effect channels. `CloseMusicChannel()` disposes with
`SndDisposeChannel(musicChannel, true)` (`GliderPRO/Sources/Music.c:302`), nils
the pointer, and disposes the UPP (`GliderPRO/Sources/Music.c:305`).

### 7.12 `InitMusic()` / `KillMusic()` / `MusicBytesNeeded()`

`GliderPRO/Sources/Music.c:312-369`, `:373-382`, `:386-407`. Called from
`GliderPRO/Sources/Main.c:338` and `GliderPRO/Sources/Main.c:370`.

```
 1  InitMusic():
 2      if dontLoadMusic: return                                 // Music.c:316-317
 3      musicChannel = nil                                       // Music.c:319
 4      failedMusic = false; isMusicOn = false                   // Music.c:321-322
 5      if LoadMusicSounds() != noErr:                           // Music.c:323
 6          YellowAlert(kYellowNoMusic, err); failedMusic = true; return   // Music.c:326-328
 7      OpenMusicChannel()      // error IGNORED                 // Music.c:330
 8      fill musicScore[0..15]                                   // Music.c:332-347
 9      fill gameScore[0..5]                                     // Music.c:349-354
10      musicCursor  = 0                                         // Music.c:356
11      musicSoundID = musicScore[0] = 0                          // Music.c:357
12      musicMode    = kPlayWholeScoreMode                       // Music.c:358
13      if isPlayMusicIdle:                                      // Music.c:360
14          if StartMusic() != noErr:                            // Music.c:362-363
15              YellowAlert(kYellowNoMusic, err); failedMusic = true   // Music.c:365-366
```

Note line 7: the `OSErr` from `OpenMusicChannel()` is assigned to `theErr` and
never tested (`GliderPRO/Sources/Music.c:330`). If the channel cannot be created,
`failedMusic` stays false and `StartMusic` will happily `SndDoCommand` on a nil
`musicChannel`. In practice a fourth channel is always available.

`kYellowNoMusic` = 13 (`GliderPRO/Headers/GliderDefines.h:30`) selects string 13
of `STR#` 1006, verbatim from the resource fork:

> Well, the music didn't load.  Glider PRO(tm) will still run, you'll just be musically challenged.

`MusicBytesNeeded()` (`GliderPRO/Sources/Music.c:386-407`) mirrors
`SoundBytesNeeded()`: `SetResLoad(false)`, sum `GetMaxResourceSize` over
`'snd '` 2000..2006, `SetResLoad(true)`. Measured against the shipped fork it
returns **776,996**.

`TellHerNoMusic()` (`GliderPRO/Sources/Music.c:411-418`) shows `ALRT` 1038:
*"Okay, with your monitor & color depth, there isn't enough memory.  In order to
run Glider PRO(tm), the music won't be loaded."*, single button "Whatever", `ICON`
1011.

### 7.13 Every place music starts or stops

| Site | Action | Condition |
|---|---|---|
| `GliderPRO/Sources/Music.c:362` | `StartMusic()` | end of `InitMusic()`, if `isPlayMusicIdle` |
| `GliderPRO/Sources/Play.c:88` | `StartMusic()` | game begins, if `isPlayMusicGame && !isMusicOn` |
| `GliderPRO/Sources/Play.c:100` | `StopTheMusic()` | game begins, if `!isPlayMusicGame && isMusicOn` |
| `GliderPRO/Sources/Play.c:239` | `StartMusic()` | game ends, if `isPlayMusicIdle && !isMusicOn` |
| `GliderPRO/Sources/Play.c:251` | `StopTheMusic()` | game ends, if `!isPlayMusicIdle && isMusicOn` |
| `GliderPRO/Sources/Play.c:413` | `ToggleMusicWhilePlaying()` | app resume during play |
| `GliderPRO/Sources/Play.c:421` | `ToggleMusicWhilePlaying()` | app suspend during play (bug, see 7.8) |
| `GliderPRO/Sources/Events.c:420` | `StartMusic()` | app resume, if `isPlayMusicIdle && theMode != kEditMode` |
| `GliderPRO/Sources/Events.c:439` | `StopTheMusic()` | app suspend, if `isMusicOn && theMode != kEditMode` |
| `GliderPRO/Sources/Menu.c:393` | `StartMusic()` | entering splash mode, if `isPlayMusicIdle` |
| `GliderPRO/Sources/Menu.c:407` | `StopTheMusic()` | entering edit mode, unconditionally |
| `GliderPRO/Sources/Settings.c:640` | `StopTheMusic()` | Sound Prefs volume set to 0, if `wasIdle` |
| `GliderPRO/Sources/Settings.c:645` | `StartMusic()` | Sound Prefs volume set non-zero, if `wasIdle && !isMusicOn` |
| `GliderPRO/Sources/Settings.c:802` | `StartMusic()` | Sound Prefs "idle music" checkbox turned on |
| `GliderPRO/Sources/Settings.c:811` | `StopTheMusic()` | Sound Prefs "idle music" checkbox turned off |
| `GliderPRO/Sources/Settings.c:857` | `StartMusic()` | Sound Prefs Cancel restoring "idle music" on |
| `GliderPRO/Sources/Settings.c:866` | `StopTheMusic()` | Sound Prefs Cancel restoring "idle music" off |
| `GliderPRO/Sources/Settings.c:1249` | `StartMusic()` | `SetAllDefaults()`, if `!isMusicOn` |
| `GliderPRO/Sources/Dynamics.c:681` | `ToggleMusicWhilePlaying()` | stereo turned on |
| `GliderPRO/Sources/Dynamics.c:698` | `ToggleMusicWhilePlaying()` | stereo turned off |
| `GliderPRO/Sources/Music.c:381` | `CloseMusicChannel()` | `KillMusic()` at quit |

## 8. Volume - and the absence of any other DSP

### 8.1 There is no per-sound or per-channel volume

Glider PRO never issues `volumeCmd`, `ampCmd`, `rateCmd`, `freqCmd`,
`freqDurationCmd` or `rateMultiplierCmd`. The only commands it ever sends are
`bufferCmd` (81), `callBackCmd` (13), `quietCmd` (3), `flushCmd` (4) and
`nullCmd` (0). Consequently:

- no pitch shifting (`bufferCmd` with `param1 = 0` plays at the header's rate);
- no per-sound attenuation;
- no panning (channels are created `initMono`);
- no interpolation when resampling (`initNoInterp`), i.e. nearest-neighbour;
- no envelopes, no fades. The "fade in"/"fade out" sounds are *recordings* of
  fades, not applied gain.

Volume is the machine's global output volume, which the game overwrites.

One audible thing does bypass the whole engine: `SysBeep()`. It is called 26 times
across the tree (`GliderPRO/Sources/HighScores.c:753`, `:760`, `:812`, `:819`,
`House.c:737`, `ObjectInfo.c:1215`, `:1239`, `:1448`, `:1468`, `:1487`, `:1506`,
`:1681`, `:1712`, `:2231`, `:2259`, `:2561`, `ObjectEdit.c:1166`,
`SelectHouse.c:144`, `:174`, `Main.c:276`, `Play.c:198`, `RoomInfo.c:785`,
`Settings.c:402`, `:424`, `:446`, `:468`) - all of them editor / dialog / error
paths, none in the gameplay loop. It plays the *system* alert sound at the
*system* volume, so it is unaffected by everything in this section, and a port
should map it to its own UI-error sound rather than to a `'snd '` resource.

### 8.2 `UnivGetSoundVolume` / `UnivSetSoundVolume`

`GliderPRO/Sources/Utilities.c:742-760` and `:766-787`. Prototypes at
`GliderPRO/Headers/Externs.h:370-371`. Both take a `Boolean hasSM3` that is
`#pragma unused` (`GliderPRO/Sources/Utilities.c:744`, `:768`) - the pre-Sound
Manager 3 path is commented out, leaving the Sound Manager 3 calls
unconditional. In the getter the dead lines are `748`, `749`, `752`, `753` and the
`GetSoundVol(volume)` call at `754`, wrapped around the live `750-751`; in the
setter they are `777`, `778`, `784`, `785` and the `SetSoundVol(volume)` call at
`786`, wrapped around the live `779-783`.

```c
void UnivGetSoundVolume (short *volume, Boolean hasSM3)      // Utilities.c:742
{
    theErr = GetDefaultOutputVolume(&longVol);               // Utilities.c:750
    *volume = LoWord(longVol) / 0x0024;                      // Utilities.c:751  /36
    if (*volume > 7)      *volume = 7;                       // Utilities.c:756-757
    else if (*volume < 0) *volume = 0;                       // Utilities.c:758-759
}

void UnivSetSoundVolume (short volume, Boolean hasSM3)       // Utilities.c:766
{
    if (volume > 7)      volume = 7;                         // Utilities.c:772-773
    else if (volume < 0) volume = 0;                          // Utilities.c:774-775
    longVol = (long)volume * 0x0025;                         // Utilities.c:779  *37
    if (longVol > 0x00000100) longVol = 0x00000100;          // Utilities.c:780-781
    longVol = longVol + (longVol << 16);                     // Utilities.c:782
    theErr = SetDefaultOutputVolume(longVol);                // Utilities.c:783
}
```

`SetDefaultOutputVolume` takes a 32-bit value whose **low word is the left channel
and high word the right channel**, each 0..0x100 where 0x100 = 1.0 (unity). Line
`GliderPRO/Sources/Utilities.c:782` duplicates the low word into the high word, so
both speakers get the same level.

Note the asymmetric constants: **set multiplies by 0x25 (37), get divides by 0x24
(36)**. That is deliberate. Verified by computation:

| user value | `volume * 0x25` | after clamp to 0x100 | 32-bit written | read back `LoWord / 0x24` |
|---:|---:|---:|---|---:|
| 0 | 0x000 (0) | 0x000 | 0x00000000 | 0 |
| 1 | 0x025 (37) | 0x025 | 0x00250025 | 1 |
| 2 | 0x04A (74) | 0x04A | 0x004A004A | 2 |
| 3 | 0x06F (111) | 0x06F | 0x006F006F | 3 |
| 4 | 0x094 (148) | 0x094 | 0x00940094 | 4 |
| 5 | 0x0B9 (185) | 0x0B9 | 0x00B900B9 | 5 |
| 6 | 0x0DE (222) | 0x0DE | 0x00DE00DE | 6 |
| 7 | 0x103 (259) | **0x100 (256)** | 0x01000100 | 7 |

The round trip is exact for all eight values. Had the getter also divided by 37,
value 7 would read back as 6 (256/37 = 6) - so the 36 is there purely to make the
clamped 7 survive the round trip.

Reading an *arbitrary* system volume back into the 0..7 scale
(`LoWord / 36`, clamped at 7):

| system low word | reported |
|---|---:|
| 0x000..0x023 (0..35) | 0 |
| 0x024..0x047 (36..71) | 1 |
| 0x048..0x06B (72..107) | 2 |
| 0x06C..0x08F (108..143) | 3 |
| 0x090..0x0B3 (144..179) | 4 |
| 0x0B4..0x0D7 (180..215) | 5 |
| 0x0D8..0x0FB (216..251) | 6 |
| >= 0x0FC (252) | 7 |

Because `LoWord()` takes the *low* 16 bits, a machine whose left and right
volumes differ has only its left channel considered.

### 8.3 Glider PRO changes the machine's volume, and puts it back

This is the single most surprising design decision in the audio subsystem. At
startup (`GliderPRO/Sources/Main.c:194-200`):

```c
UnivGetSoundVolume(&wasVolume, thisMac.hasSM3);   // Main.c:194  remember the user's setting
UnivSetSoundVolume(isVolume, thisMac.hasSM3);     // Main.c:195  impose Glider's setting

if (isVolume == 0)                                // Main.c:197
    isSoundOn = false;                            // Main.c:198
else
    isSoundOn = true;                             // Main.c:200
```

then in `WriteOutPrefs()`, called just before quitting
(`GliderPRO/Sources/Main.c:208-279`):

```c
UnivGetSoundVolume(&isVolume, thisMac.hasSM3);    // Main.c:212  save Glider's setting
...
thePrefs.wasVolume    = isVolume;                 // Main.c:235
thePrefs.wasMusicOn   = isMusicOn;                // Main.c:237
thePrefs.wasIdleMusic = isPlayMusicIdle;          // Main.c:241
thePrefs.wasGameMusic = isPlayMusicGame;          // Main.c:242
...
UnivSetSoundVolume(wasVolume, thisMac.hasSM3);    // Main.c:278  restore the USER's setting
```

So the game's volume preference is stored in its own prefs file and the machine's
volume is borrowed for the duration of the run. A Go port on a modern OS **must
not** do this: it should keep an internal 0..7 gain and apply it in its own mixer.
The 0..7 -> 0..0x100 table above is the gain curve to reproduce (it is
essentially linear: `gain = min(v * 37, 256) / 256`).

The relevant `prefsInfo` fields, from `GliderPRO/Headers/Externs.h:233-267` under
`#pragma options align=mac68k` (`GliderPRO/Headers/Externs.h:231`):

| Field | Type | Line |
|---|---|---|
| `short wasVolume;` | `short` (2 bytes, big-endian) | `GliderPRO/Headers/Externs.h:243` |
| `Boolean wasZooms, wasMusicOn;` | 1 byte each | `GliderPRO/Headers/Externs.h:258` |
| `Boolean wasIdleMusic, wasGameMusic;` | 1 byte each | `GliderPRO/Headers/Externs.h:262` |

`#pragma options align=mac68k` means 2-byte alignment, so `Boolean` fields pack
without padding to 4. A Go port reading an original prefs file must honour that.

First-run defaults (`GliderPRO/Sources/Main.c:140-150`): the *current machine*
volume is read (`UnivGetSoundVolume(&isVolume, ...)` at
`GliderPRO/Sources/Main.c:140`) and then **clamped into 1..3** before being
adopted as Glider's own -

```c
UnivGetSoundVolume(&isVolume, thisMac.hasSM3);   // Main.c:140
if (isVolume < 1)        isVolume = 1;           // Main.c:141-142
else if (isVolume > 3)   isVolume = 3;           // Main.c:143-144
```

so a fresh install is never silent and never louder than 3 (of 7), whatever the
machine was set to. `isSoundOn`, `isMusicOn`, `isPlayMusicIdle`,
`isPlayMusicGame` are all set true (`GliderPRO/Sources/Main.c:147-150`).
`SetAllDefaults()` in `Settings.c` instead forces volume 3:
`isPlayMusicIdle = true` (`GliderPRO/Sources/Settings.c:1243`),
`isPlayMusicGame = true` (`:1244`), `UnivSetSoundVolume(3, ...)` (`:1245`),
`isSoundOn = true` (`:1246`), `if (!isMusicOn) StartMusic()` (`:1247-1249`).

### 8.4 The Sound Prefs dialog

It is a modal **dialog**, not an alert: `kSoundPrefsDialID` = 1018
(`GliderPRO/Sources/Settings.c:19`) names `DLOG` 1018 + `DITL` 1018 + `dctb` 1018
(there is no `ALRT` 1018), opened by `BringUpDialog(&prefDlg, kSoundPrefsDialID)`
(`GliderPRO/Sources/Settings.c:770`).
Item numbers (`GliderPRO/Sources/Settings.c:36-41`):

| Item | Constant | Line | Kind, from the shipped `DITL` 1018 |
|---:|---|---|---|
| 1 | `kOkayButton` | - | button "Okay" |
| 2 | `kCancelButton` | - | button "Cancel" |
| 3 | - | - | `ICON` 1030 (speaker) at (52,244,84,276) |
| 4 | `kSofterItem` | `GliderPRO/Sources/Settings.c:36` | `ICON` 1032 (softer) at (69,276,101,308) |
| 5 | `kLouderItem` | `:37` | `ICON` 1031 (louder) at (35,276,67,308) |
| 6 | - | - | static text "Volume:" |
| 7 | `kVolNumberItem` | `:38` | static text - the number |
| 8 | `kIdleMusicItem` | `:39` | checkbox "Play music when idle" |
| 9 | `kPlayMusicItem` | `:40` | checkbox "Play music during game" |
| 10 | - | - | static text "Set the game volume and background music options." |
| 11 | - | - | a disabled `userItem` at (138,8,139,308) - a 1-pixel-high divider rule - stroked by `FrameDialogItemC(theDialog, 11, kRedOrangeColor8)` (`GliderPRO/Sources/Settings.c:626`) |
| 12 | - | - | `PICT` 1008 |
| 13 | `kSoundDefault` | `:41` | button "Defaults" |

`cicn` 1033 (louder, pressed) and `cicn` 1034 (softer, pressed) are drawn over
the icons for feedback with `DrawCIcon` (`GliderPRO/Sources/Settings.c:821`,
`:836`) followed by `DelayTicks(8)` (`:827`, `:845`) - an 8-tick (133 ms) button
flash.

**"11".** When the volume is 7 the dialog displays the number **11**, not 7:
`SetDialogNumToStr(theDialog, kVolNumberItem, 11L)` at
`GliderPRO/Sources/Settings.c:622`, `:703`, `:839`. This is the Spinal Tap joke.
The softer path never displays 11 (`GliderPRO/Sources/Settings.c:823` prints the
plain value, which after decrementing from 7 is 6).

Keyboard handling in `SoundFilter` (`GliderPRO/Sources/Settings.c:661-754`):

| Key | Effect |
|---|---|
| Return / Enter | Okay |
| Escape | Cancel |
| Up arrow | `kLouderItem` |
| Down arrow | `kSofterItem` |
| `0`..`7` | set the volume directly (`newVolume = char - '0'`), display 11 if 7, `UnivSetSoundVolume`, `HandleSoundMusicChange(newVolume, true)` (`GliderPRO/Sources/Settings.c:701-709`) |
| `D` / `d` | Defaults |
| `G` / `g` | toggle "play music during game" |
| `I` / `i` | toggle "play music when idle" |

`HandleSoundMusicChange(newVolume, sayIt)` (`GliderPRO/Sources/Settings.c:631-657`)
is the one function that ties volume to the music engine:

```
 1  isSoundOn = (newVolume != 0)                      // Settings.c:635
 2  if wasIdle:                                       // Settings.c:637  the dialog's live copy
 3      if newVolume == 0: StopTheMusic()             // Settings.c:639-640
 4      else if not isMusicOn: StartMusic()           // Settings.c:643-645
 5  if newVolume != 0 and sayIt:
 6      PlayPrioritySound(kChord2Sound, kChord2Priority)   // Settings.c:655-656
```

so every volume change previews itself with `kChord2Sound` = index 54 =
`'snd '` 1054, whose resource name is actually **"Check"** (5056 frames,
227.2 ms, priority `kChord2Priority` = 810). Dropping to 0 stops the music;
raising from 0 restarts it.

Cancel semantics (`GliderPRO/Sources/Settings.c:793-814`): the system volume is
restored to `wasLoudness` captured at `GliderPRO/Sources/Settings.c:772`,
`HandleSoundMusicChange(wasLoudness, false)` is called with `sayIt = false` (no
preview chord), and the "idle music" toggle is un-done. Okay
(`GliderPRO/Sources/Settings.c:785-791`) commits `wasIdle`/`wasPlay` into
`isPlayMusicIdle`/`isPlayMusicGame` and recomputes `isSoundOn`. Note that the
*volume* is not committed on Okay - it was already applied live.

## 9. The memory budget that can silently disable audio

`CheckMemorySize()` (`GliderPRO/Sources/Environ.c:562-692`) runs before
`InitSound()`/`InitMusic()` and can turn audio off entirely.

| Constant | Value | Line |
|---|---:|---|
| `kBaseBytesNeeded` | 614400 (600 KB) | `GliderPRO/Sources/Environ.c:564` |
| `kPaddingBytes` | 204800 (200 KB) | `GliderPRO/Sources/Environ.c:565` |

```
 1  dontLoadMusic  = false                                  // Environ.c:571
 2  dontLoadSounds = false                                  // Environ.c:572
 3  bytesNeeded = 614400                                    // Environ.c:574
 4  soundBytes = SoundBytesNeeded()                         // Environ.c:575  = 352348
 5  if soundBytes <= 0: RedAlert(kErrNoMemory)              // Environ.c:576-577
 6  else bytesNeeded += soundBytes                          // Environ.c:579
 7  musicBytes = MusicBytesNeeded()                         // Environ.c:580  = 776996
 8  if musicBytes <= 0: RedAlert(kErrNoMemory)              // Environ.c:581-582
 9  else bytesNeeded += musicBytes                          // Environ.c:584
10  ... 40-odd further terms for GWorlds, sprite maps and object arrays
11      (Environ.c:585-657), ending with kDemoLength = 6702
12  bytesAvail = FreeMem()                                  // Environ.c:659
13  if bytesAvail < bytesNeeded:                            // Environ.c:661
14      if bytesAvail >= bytesNeeded - musicBytes:           // Environ.c:664
15          TellHerNoMusic();  dontLoadMusic = true;  return    // Environ.c:666-668
16      else if bytesAvail >= bytesNeeded - (musicBytes + soundBytes):   // Environ.c:670
17          TellHerNoSounds(); dontLoadMusic = true;
18                             dontLoadSounds = true; return   // Environ.c:672-675
19      Alert(kSetMemoryAlert) with (bytesNeeded + 204800)/1024 KB   // Environ.c:685-687
20      ExitToShell()                                       // Environ.c:690
```

So the degradation ladder is: **full audio -> no music (776,996 bytes reclaimed)
-> no music and no sounds (1,129,344 bytes reclaimed) -> refuse to launch**. Music
is always the first thing sacrificed because it is more than twice the size of all
63 sound effects combined.

Note that `dontLoadSounds = true` always comes with `dontLoadMusic = true` - there
is no "sounds but no music" state other than as a *user preference*.

The three flags interact like this at run time:

| `dontLoadSounds` | `dontLoadMusic` | `failedSound` | `failedMusic` | Behaviour |
|:--:|:--:|:--:|:--:|---|
| false | false | false | false | normal |
| false | true | false | - | effects work; every music entry point returns immediately at its `if (dontLoadMusic) return` guard (`Music.c:52`, `:103`, `:129`, `:148`, `:316`, `:377`) |
| true | true | - | - | `PlayPrioritySound` returns at `Sound.c:44`; `InitSound` returns at `Sound.c:442`; nothing is allocated or opened |
| false | false | true | false | resource load or channel creation failed at launch; effects permanently silent, `YellowAlert(kYellowFailedSound)` shown once |
| false | false | false | true | `LoadMusicSounds` or `StartMusic` failed; music permanently silent, `YellowAlert(kYellowNoMusic)` shown once |

`YellowAlert(short whichAlert, short identifier)`
(`GliderPRO/Sources/HouseIO.c:640-655`) uses `#define kYellowAlert 1006`
(`GliderPRO/Sources/HouseIO.c:642`), fetches string `whichAlert` from `STR#` 1006
with `GetIndString(errStr, kYellowAlert, whichAlert)`
(`GliderPRO/Sources/HouseIO.c:648`) and shows `Alert(kYellowAlert, nil)`
(`GliderPRO/Sources/HouseIO.c:654`). The `identifier` is the `OSErr`, displayed as
a number. `STR#` 1006 has 24 strings; items 13 and 14 are the two audio ones,
quoted verbatim in sections 7.12 and 4.11.

## 10. The audio-related resources, verbatim from the fork

Recovered by parsing `GliderPRO/Glider PRO.r` with python3.

| Resource | Id | Content |
|---|---:|---|
| `STR#` 1006 item 13 | - | "Well, the music didn't load.  Glider PRO(tm) will still run, you'll just be musically challenged." |
| `STR#` 1006 item 14 | - | "Wow, there was a problem bringing sounds up.  You might try giving Glider PRO(tm) more memory - otherwise ... silence." |
| `ALRT` / `DITL` | 1030 | "Where is Sound Manager 3.0?  I highly recommend you install it.  It's fast, it's unobtrusive, and it's the law!" - icon 1070, rect (40,40,148,314), stages 0x4444. **Unused** (`BitchAboutSM3` is never called). |
| `ALRT` / `DITL` | 1038 | "Okay, with your monitor & color depth, there isn't enough memory.  In order to run Glider PRO(tm), the music won't be loaded." - button "Whatever", `ICON` 1011, rect (54,96,162,370), stages 0x4444 |
| `ALRT` / `DITL` | 1039 | "Okay, with your monitor & color depth, there isn't enough memory.  To run Glider PRO(tm), the music & sounds won't be loaded." - same button/`ICON` 1011/rect/stages as 1038; note the wording differs from 1038 in more than the last clause ("To run" vs "In order to run") |
| `DLOG` + `DITL` + `dctb` | 1018 | the Sound Prefs dialog, 13 items - enumerated in section 8.4 |
| `ICON` | 1030 | speaker glyph, Sound Prefs item 3 |
| `ICON` | 1031 | louder arrow, Sound Prefs item 5 |
| `ICON` | 1032 | softer arrow, Sound Prefs item 4 |
| `cicn` | 1033 | louder arrow, pressed state (`DrawCIcon`, `Settings.c:836`) |
| `cicn` | 1034 | softer arrow, pressed state (`DrawCIcon`, `Settings.c:821`) |
| `PICT` | 1008 | decorative art in the Sound Prefs dialog, item 12 |
| `'snd '` | 1000..1062 | the 63 sound effects (section 5.1) |
| `'snd '` | 2000..2006 | the 7 music pieces (section 7.1) |

The "(tm)" above is a literal MacRoman 0xAA byte in the resource; a Go port
reading these strings must decode MacRoman, not Latin-1 or UTF-8.

## 11. Extraction plan for the Go port

### 11.1 The prototype decoder

`tools/probe_snd.py`, 442 lines. It depends on two
pre-existing helpers in the same directory, `probe_rez.py` (parses the DeRez text
dump `GliderPRO/Glider PRO.r` into typed resources) and `probe_house.py` (BinHex
4.0 decode plus Macintosh resource-fork map parsing).

Sub-commands:

| Command | Effect |
|---|---|
| `probe_snd.py show [ID]` | full annotated decode of one application resource (default 1013) |
| `probe_snd.py list` | one-line-per-resource table of all 70, with totals |
| `probe_snd.py houses` | every `'snd '` in `GliderPRO/Houses/*.binhex`, with anomaly flags |
| `probe_snd.py extract <outdir>` | writes `snd_<id>.pcm` for every resource plus `manifest.tsv` |
| `probe_snd.py wav <ID> <out.wav>` | 8-bit mono WAV, for listening by ear only |

Key functions: `fixed_to_hz` (`tools/probe_snd.py:93-95`), `ext80_to_float`
(`:98-106`), `class SndHeader` (`:110-147`, handles format 1 and format 2;
`sound_header_offset()` returns the `param2` of the first
`bufferCmd`/`soundCmd` and *raises* if `dataOffsetFlag` is absent, because a bare
pointer is meaningless in a file), `class SampledSound` (`:150-207`, all three
`encode` values), `app_snds()` (`:219-229`), `house_snds()` (`:232-249`).

### 11.2 Observed output for one real resource

`python3 tools/probe_snd.py show 1013`, verbatim (the required prototype
demonstration):

```
resource      'snd ' 1013 'Tok'
resource size 494 bytes
first 48 bytes:
  0000  00 01 00 01 00 05 00 00 00 A0 00 01 80 51 00 00  .............Q..
  0010  00 00 00 14 00 00 00 00 00 00 01 C4 56 EE 8B A3  ............V...
  0020  00 00 01 C2 00 00 01 C3 00 3C 50 50 84 9B A5 99  .........<PP....

format            1
numSynths         1
  synth           5 (sampledSynth)  initOption 0x000000A0
numCmds           1
  cmd 0x8051      bufferCmd | dataOffsetFlag param1=0 param2=20

sound header at   +20
  samplePtr       0x00000000 (samples follow header)
  encode          0x00 (stdSH)
  sampleRate      0x56EE8BA3 = 22254.5455 Hz  [rate22khz]
  baseFrequency   60
  loopStart       450
  loopEnd         451
  length          452 sample frames
  header+samples  22 + 452 = 494 (resource size 494)
  duration        0.0203 s (20.3 ms)
  u8 PCM min/max/mean  80 / 175 / 135.18  (0x80 = silence)
  first 16 samples     [80, 80, 132, 155, 165, 153, 130, 114, 114, 130, 152, 162, 159, 158, 158, 162]
  last 8 samples       [82, 87, 94, 101, 104, 103, 101, 100]
  no loop (degenerate loopStart=length-2, loopEnd=length-1)
```

**Reported sample rate: 22254.5455 Hz (Fixed 0x56EE8BA3). Reported length: 452
sample frames = 20.3 ms.**

### 11.3 Observed output for the extraction

`python3 tools/probe_snd.py extract <dir>`:

```
wrote 70 .pcm files + manifest.tsv to <dir>
format: unsigned 8-bit mono, offset binary (0x80 = silence),
        22254.5455 Hz for every application resource.
```

70 `.pcm` files totalling **1,126,404 bytes**, plus a 5,067-byte
`manifest.tsv` whose first lines are:

```
file            id    name       frames  rate_fixed  rate_hz     loop_start loop_end base_freq encode
snd_1000.pcm    1000  Wall Hit   1840    0x56EE8BA3  22254.5455  1838       1839     60        0x00
snd_1001.pcm    1001  Fade In    8832    0x56EE8BA3  22254.5455  8830       8831     60        0x00
snd_1002.pcm    1002  Fade Out   9952    0x56EE8BA3  22254.5455  9950       9951     60        0x00
snd_1003.pcm    1003  Beeps      9680    0x56EE8BA3  22254.5455  9678       9679     60        0x00
```

### 11.4 Recommended asset pipeline

The port should **not** parse `'snd '` resources at run time. Convert once,
offline, and embed the results with `go:embed`.

1. **Application effects and music.** Run
   `probe_snd.py extract assets/audio/app/`. That yields 70 raw files plus the
   manifest. Convert each to the port's internal mixer format at build time:

   ```
   for each u8 sample b:  s16 = (int16(b) - 128) << 8
   ```

   Keep the native rate 22254.5455 Hz in the manifest and resample once at load
   time to the output device rate (typically 44100 or 48000). Because the original
   used `initNoInterp` (nearest-neighbour), a *faithful* port would resample with
   point sampling; a *nicer* port uses linear or better. Point sampling from
   22254.5455 Hz to 44100 Hz is audibly grainy in exactly the way the original was,
   so this is a deliberate fidelity choice, not a bug to fix silently.

2. **House sounds.** Run `probe_snd.py houses` to enumerate, then extract
   per-house into `assets/audio/houses/<house>/snd_<id>.pcm`. **58 of the 63 are
   plain `stdSH` and convert exactly as above. The remaining 5 are MACE 6:1 and
   need a MACE decoder, which is not implemented** - see `## Open questions`. As
   an interim measure the port can ship those 5 as silence and log a warning; the
   affected triggers are one in Demo House, one in CD Demo House and three in
   Nemo's Market.

3. **Manifest fields the runtime actually needs**: resource id, frame count,
   sample rate, and the PCM blob. `loopStart`/`loopEnd`/`baseFrequency` are
   *unused by the engine* and need only be carried for documentation.

4. **No WAV, no OGG.** The `wav` sub-command exists purely so a human can audition
   a sound; the shipping pipeline emits raw PCM with a side-table, which is what
   `go:embed` + a `[]int16` slice wants.

5. **Deterministic ordering.** Sound *index* is what game code uses
   (`PlayPrioritySound(kHitWallSound, ...)` where `kHitWallSound == 0`), so the
   port should build `sounds [64][]int16` indexed 0..63, filling 0..62 from ids
   1000..1062 and leaving 63 for the per-room trigger. Music should be
   `music [7][]int16` from ids 2000..2006. Reproducing the index-to-id mapping
   exactly means the priority table and all 167 call sites port over unchanged.

## 12. A Go implementation sketch

This is not a proposal for an architecture; it is a translation of the C so a
porter can diff. The names in parentheses are the originals.

### 12.1 The three-voice mixer

```go
const (
    NumVoices     = 3
    MaxSounds     = 64    // kMaxSounds
    TriggerSlot   = 63    // kMaxSounds - 1
    NoSoundPlaying = -1   // kNoSoundPlaying
    TriggerPriority = 999 // kTriggerPriority
    BaseBufferSoundID = 1000 // kBaseBufferSoundID
)

type voice struct {
    priority int   // priority0 / priority1 / priority2
    sound    int   // soundPlaying0 / soundPlaying1 / soundPlaying2
    pcm      []int16
    pos      int   // read cursor, in output frames (fixed point if resampling)
}

type SoundEngine struct {
    mu       sync.Mutex
    v        [NumVoices]voice
    data     [MaxSounds][]int16   // theSoundData
    on       bool                 // isSoundOn
    dontLoad bool                 // dontLoadSounds
    failed   bool                 // failedSound
}
```

`PlayPrioritySound(soundID, priority)`, faithful to
`GliderPRO/Sources/Sound.c:40-85`:

```go
func (e *SoundEngine) PlayPrioritySound(soundID, priority int) {
    if e.failed || e.dontLoad { return }            // Sound.c:44
    e.mu.Lock(); defer e.mu.Unlock()
    // Sound.c:47-51: if this IS a trigger sound and some voice is already
    // playing one, drop it. No soundID rewriting happens here -- callers pass
    // kTriggerSound (63) explicitly.
    if priority == TriggerPriority &&
        (e.v[0].priority == TriggerPriority ||
            e.v[1].priority == TriggerPriority ||
            e.v[2].priority == TriggerPriority) {
        return
    }
    lowest, which := e.v[0].priority, 0             // Sound.c:53-54
    if e.v[1].priority < lowest { lowest, which = e.v[1].priority, 1 }
    if e.v[2].priority < lowest { lowest, which = e.v[2].priority, 2 }
    if priority < lowest { return }                 // Sound.c:68, note >= 
    e.start(which, soundID, priority)
}

func (e *SoundEngine) start(i, soundID, priority int) {
    if e.failed || e.dontLoad { return }            // Sound.c:138-139
    if e.data[soundID] == nil { return }            // not in the C: it indexes
                                                    // theSoundData[] unchecked
    if !e.on { return }                             // Sound.c:142
    e.v[i] = voice{priority: priority, sound: soundID,
                   pcm: e.data[soundID], pos: 0}
}
```

Three details a porter will get wrong if they paraphrase:

- The comparison is `priority >= lowestPriority` (`Sound.c:68`), i.e. **ties
  win**. A second `kHitWallSound` (priority 100) does displace a currently
  playing `kHitWallSound`. That is the retrigger behaviour the game depends on.
- Search order is 0, then 1 only if *strictly* lower, then 2 only if *strictly*
  lower (`Sound.c:56`, `:62`). With all three idle at priority 0 the winner is
  always channel 0, so channel 0 does the most work and channel 2 is only reached
  when 0 and 1 are both busy.
- The `kTriggerPriority` block at `Sound.c:47-51` is **only** a "don't overlap two
  trigger sounds" early return. It does not rewrite `which`/`soundID`, and it does
  not check that slot 63 is loaded. Callers name sound 63 themselves
  (`kTriggerSound`, `Interactions.c:1072` and `:1618`).

Where the C called `SndDoImmediate(bufferCmd)` followed by
`SndDoCommand(callBackCmd)`, Go just assigns the voice. The mixer callback
replaces the Sound Manager's interrupt:

```go
// Called from the audio device callback. Replaces CallBack0/1/2
// (Sound.c:217-261) plus the Sound Manager's own mixing.
func (e *SoundEngine) Mix(out []int16) {
    e.mu.Lock(); defer e.mu.Unlock()
    for i := range out { out[i] = 0 }
    for i := range e.v {
        v := &e.v[i]
        if v.pcm == nil { continue }
        n := copy32(out, v.pcm[v.pos:])
        v.pos += n
        if v.pos >= len(v.pcm) {                    // sound finished
            v.pcm = nil
            v.priority = 0                          // Sound.c:225
            v.sound = NoSoundPlaying                // Sound.c:226
        }
    }
    clip(out)
}
```

`FlushAnyTriggerPlaying` (`Sound.c:89-129`) becomes trivially correct in Go, and
the port should fix the original's bug while it is there:

```go
func (e *SoundEngine) FlushAnyTriggerPlaying() {
    e.mu.Lock(); defer e.mu.Unlock()
    for i := range e.v {
        if e.v[i].sound == TriggerSlot {
            e.v[i] = voice{priority: 0, sound: NoSoundPlaying}  // note: 0, not 999
        }
    }
}
```

The original left `priorityN` at 999 (see 4.7 and `## Open questions`); resetting
it to 0 here is the intended behaviour and is what the callback would have done.

Mixing: the Sound Manager summed the three channels and clipped. The port should
sum into `int32` and clip to `[-32768, 32767]`. Because the source material is
8-bit, three voices summed cannot exceed 3 * 32512 = 97536, so clipping does
occur and *is* part of the original's sound.

### 12.2 Concurrency

The C got away with plain `short` globals because the callbacks ran at interrupt
time on a single CPU with the main thread stopped. In Go, `priority0..2`,
`soundPlaying0..2`, `musicCursor` and `musicSoundID` are touched from both the
game goroutine and the audio-device goroutine. Options, in order of preference:

1. One `sync.Mutex` around the whole engine, held for microseconds. The audio
   callback must never block on anything else while holding it (no allocation, no
   channel send, no logging).
2. A lock-free command queue: the game goroutine pushes `{soundID, priority}`
   onto a ring buffer, the audio goroutine owns all voice state. This is closest
   in spirit to `SndDoImmediate` and removes the priority-decision race entirely,
   but the priority decision then happens up to one buffer late.

Do not use `atomic` on the individual fields: the priority scan reads three
priorities and then writes one voice, and that whole sequence has to be atomic or
two simultaneous `PlayPrioritySound` calls can both pick the same voice.

### 12.3 Music

```go
const (
    MaxMusic       = 7   // kMaxMusic
    LastMusicPiece = 16  // kLastMusicPiece
    LastGamePiece  = 6   // kLastGamePiece
    BaseBufferMusicID = 2000

    PlayWholeScoreMode = -1  // kPlayWholeScoreMode
    PlayGameScoreMode  = -2  // kPlayGameScoreMode
    KickGameScoreMode  = -3  // kKickGameScoreMode
    ProdGameScoreMode  = -4  // kProdGameScoreMode
)

var musicScore = [LastMusicPiece]int{0,1,2,3,4,4,0,1,2,3,4,4,5,6,4,4}
var gameScore  = [LastGamePiece]int{6,5,-1,6,4,4}
```

The Sound Manager's double-queued `bufferCmd` (`Music.c:59-62` and `:78-81`) gave
gapless playback. In Go the equivalent is: the music mixer keeps `cur []int16`
and `pos`, and when `pos` reaches `len(cur)` it calls `advance()` *in the same
callback iteration* and keeps filling from the new buffer. Do not wait for the
next callback or you insert a gap of one buffer period.

```go
// Replaces MusicCallBack (Music.c:170-216).
func (m *MusicEngine) advance() {
    switch m.mode {
    case PlayGameScoreMode:                          // Music.c:181-191
        m.cursor++
        if m.cursor >= LastGamePiece { m.cursor = 1 }   // note: 1, not 0
        m.soundID = gameScore[m.cursor]
        if m.soundID < 0 {                              // back-up sentinel
            m.cursor += m.soundID
            m.soundID = gameScore[m.cursor]
        }
    case PlayWholeScoreMode:                         // Music.c:193-198
        m.cursor++
        if m.cursor >= LastMusicPiece-1 { m.cursor = 0 } // note: -1
        m.soundID = musicScore[m.cursor]
    default:                                          // Music.c:200-202
        m.soundID = m.mode      // dead code: mode is only ever -1 or -2
    }
    m.cur, m.pos = m.data[m.soundID], 0
}
```

Two off-by-one details that are load-bearing and look like typos but are not:

- Game-score wrap is to **1**, not 0 (`Music.c:184`), so `gameScore[0]`
  (RefrainSparse2) is only ever heard if `SetMusicalMode(kProdGameScoreMode)` set
  `musicCursor = -1` (`Music.c:157-158`).
- Whole-score wrap is at `kLastMusicPiece - 1` = 15 (`Music.c:195`), so
  `musicScore[15]` is unreachable from the callback and only plays if
  `StartMusic` happens to advance into it.

`StartMusic` (`Music.c:44-94`) queues piece `musicCursor`, then a `nullCmd` with
`param1 = 1964`, then advances the cursor and queues the *next* piece, then the
`callBackCmd`. The `nullCmd` is a no-op; nothing in this tree ever reads its
`param1`, and the literal `1964` occurs exactly once in the whole tree
(`GliderPRO/Sources/Music.c:67`; `grep -arn 1964 GliderPRO/Sources
GliderPRO/Headers` matches no other file), so its meaning is unrecoverable from
the source - see `## Open questions`. A Go port drops the `nullCmd` entirely and just primes
`cur` from `musicScore[musicCursor]`, then pre-computes the following piece if it
wants an explicit two-deep queue.

`StartMusic` is **not idempotent** (`Music.c:44-94` has no `if (isMusicOn) return`
guard): calling it twice queues four buffers and advances the cursor twice. Only
**four of the nine** external call sites guard with `if (!isMusicOn)`
(`GliderPRO/Sources/Play.c:86-88`, `Play.c:237-239`, `Settings.c:643-645`,
`Settings.c:1247-1249`); the other five guard on something else entirely and can
re-enter `StartMusic` while it is already running - `Music.c:360-362`
(`isPlayMusicIdle`), `Events.c:418-420` (`isPlayMusicIdle && theMode !=
kEditMode`), `Menu.c:391-393` (`isPlayMusicIdle`), `Settings.c:798-802`
(`isPlayMusicIdle && wasLoudness != 0`) and `Settings.c:852-857` (`wasIdle &&
tempVolume != 0`). A Go port should put the `if isMusicOn { return }` guard inside
`StartMusic` and drop the four external copies.

### 12.4 Volume

Replace `GetDefaultOutputVolume`/`SetDefaultOutputVolume`
(`GliderPRO/Sources/Utilities.c:750`, `:783`) with an internal gain. Keep the
0..7 integer scale, because it is what the prefs file stores
(`prefsInfo.wasVolume`, `GliderPRO/Headers/Externs.h:243`) and what the Sound
Prefs dialog draws. Map it with the same arithmetic so old prefs files round-trip:

```go
// UnivSetSoundVolume (Utilities.c:766-784)
func volumeToGain(v int) float64 {
    if v > 7 { v = 7 }; if v < 0 { v = 0 }
    lo := v * 0x25                  // 37
    if lo > 0x100 { lo = 0x100 }    // 7*37 = 259 -> 256
    return float64(lo) / 256.0
}
```

`isSoundOn` is derived, not independent: it is exactly `volume != 0`
(`GliderPRO/Sources/Settings.c:635`, `GliderPRO/Sources/Main.c:197-200`). Keep
that invariant or the mute button and the volume slider will disagree.

Crucially, do **not** touch the host system volume. The original's
save-and-restore dance (`Main.c:194-195` on launch, `Main.c:278` on quit) exists
only because it was hijacking a global; it is a bug to reproduce.

### 12.5 Resampling

Every application resource is 22254.5455 Hz (verified for all 70, section 3.8).
House resources are not: nine distinct rates were observed (section 6.4). So the
port needs a resampler regardless. Since `initNoInterp` was set
(`GliderPRO/Sources/Sound.c:379`, `:387`, `:395`), the original did
nearest-neighbour. Store the increment as 16.16 fixed point to mirror the Sound
Manager exactly:

```go
inc := uint32((srcRateHz / dstRateHz) * 65536.0)
```

Computed increments for the game's one rate (0x56EE8BA3 = 22254.5455 Hz):

| Output rate | ratio | increment (16.16) |
|---|---|---|
| 44100 Hz | 0.5046382190 | 33072 = 0x8130 |
| 48000 Hz | 0.4636363637 | 30385 = 0x76B1 |
| 22050 Hz | 1.0092764379 | 66144 = 0x10260 |

Note the last one is greater than 1.0: 22254.5455 Hz material played out at
22050 Hz is *slightly* downsampled, so a port that assumes "22 kHz means 1:1" is
0.93% sharp. The error is inaudible in isolation but the music loop drifts by
about 40 ms per 4.36-second piece, which is audible against a fixed 30 fps frame
clock if anything is synchronised to it (nothing is, in this game).

## 13. Bugs and warts in the shipped audio code

Fourteen defects, in rough order of how much they affect audible behaviour. Each
one is a decision point for a port: reproduce it (fidelity) or fix it (quality).
The recommendation column says which the author of this document would pick.

| # | Where | Defect | Audible? | Recommend |
|---|---|---|---|---|
| 1 | `Sound.c:200-211` | `PlaySound2` writes `priority2`/`soundPlaying2` *after* issuing `bufferCmd`+`callBackCmd` | rarely | fix |
| 2 | `Sound.c:89-129` | `FlushAnyTriggerPlaying` leaves `priorityN` at 999 | yes | fix |
| 3 | `Music.c:176-177`, `:210-215` | `MusicCallBack` uses uninitialised `gameA5`/`thisA5` | no (68k only) | delete |
| 4 | `Play.c:408-424` | suspend and resume both call `ToggleMusicWhilePlaying()` | yes | fix |
| 5 | `Triggers.c:131-133` | `kSoundTrigger` in the editor plays `kChordSound`, marked `// Change me` | yes (editor) | fix |
| 6 | `Music.c:330` | `InitMusic` ignores `OpenMusicChannel`'s `OSErr` | yes, if it fails | fix |
| 7 | `Music.c:44-94` | `StartMusic` is not idempotent | yes | fix |
| 8 | `Sound.c:491-512` | `SoundBytesNeeded` over-counts and leaks 63 handles | no | fix |
| 9 | `Sound.c:331-339`, `Music.c:238-246` | `HUnlock` -> `NewPtr` -> `HLock` purge window | no (load time) | n/a in Go |
| 10 | `Sound.c:33`, `:31` | `soundLoaded[64]` and `numSoundsLoaded` are never read or written | no | delete |
| 11 | `Sound.c:527-534` | `BitchAboutSM3` is unreachable | no | delete |
| 12 | `Music.c:200-202` | `MusicCallBack`'s `default:` case is dead | no | delete |
| 13 | `Music.c:38` | `extern Boolean isSoundOn;` declared, never used in the file | no | delete |
| 14 | `Sound.c:15` | `kMaxSounds` is 64 but only 63 built-in sounds exist | no | keep, document |

### 13.1 A wedged voice: `FlushAnyTriggerPlaying`

`FlushAnyTriggerPlaying` (`GliderPRO/Sources/Sound.c:89-129`) sends `quietCmd`
then `flushCmd` to whichever channel is playing the trigger sound, but it never
resets `priority0/1/2` back to 0. The reset lived in `CallBack0/1/2`
(`Sound.c:225`, `:241`, `:257`), which only runs if the queued `callBackCmd`
survives the flush. If it does not, that channel is stuck advertising priority
999 and `PlayPrioritySound` will never choose it again (`Sound.c:56`, `:62`,
`:68`), permanently reducing the game to two voices until the next trigger sound
is loaded into that channel.

`FlushAnyTriggerPlaying` runs on every room change
(`GliderPRO/Sources/RoomGraphics.c:57`), so if the pessimistic reading is right
the game degrades within seconds of play in a house with sound triggers. I could
not settle this from the tree (see `## Open questions`); either way, a port should
reset the priority explicitly, which makes the question moot.

### 13.2 `PlaySound2`'s reordered writes

```c
// PlaySound0, Sound.c:144-155 -- state written FIRST
priority0 = priority;
soundPlaying0 = soundID;
theCommand.cmd = bufferCmd;   ... SndDoImmediate(channel0, &theCommand);
theCommand.cmd = callBackCmd; ... SndDoCommand(channel0, &theCommand, true);

// PlaySound2, Sound.c:200-211 -- state written LAST
theCommand.cmd = bufferCmd;   ... SndDoImmediate(channel2, &theCommand);
theCommand.cmd = callBackCmd; ... SndDoCommand(channel2, &theCommand, true);
priority2 = priority;
soundPlaying2 = soundID;
```

For a sound short enough to complete inside the two `SndDoCommand` calls,
`CallBack2` fires first and sets `priority2 = 0`, `soundPlaying2 = -1`
(`Sound.c:257-258`), and then `PlaySound2` overwrites them with the values for
the sound that has already finished. Channel 2 then reports itself busy at that
priority forever, until something with an equal or higher priority displaces it.
The shortest sound in the game is 20.3 ms (`'snd ' 1013 "Tok"`, 452 frames), which
is far longer than two Sound Manager calls, so this is very unlikely to trigger in
practice - but it is a genuine ordering bug, and it is only in channel 2.

### 13.3 Music keeps playing when the app is suspended

`GliderPRO/Sources/Play.c:408-424` handles `osEvt` during gameplay:

```c
else if ((theEvent.what == osEvt) &&
         (theEvent.message & 0x01000000))     // :408  an if/else chain, not a switch
{
    if (theEvent.message & 0x00000001)        // :410  resume event
    {
        switchedOut = false;                  // :412
        ToggleMusicWhilePlaying();            // :413
    }
    else                                      // :417  suspend event
    {
        InitCursor();                         // :419
        switchedOut = true;                   // :420
        ToggleMusicWhilePlaying();            // :421
    }
}
```

`ToggleMusicWhilePlaying` (`GliderPRO/Sources/Music.c:125-142`) is not a toggle
at all - it is "make the music state match `isPlayMusicGame`". So on suspend it
*starts* the music if `isPlayMusicGame` is true, exactly the opposite of what is
wanted, and on resume it does the same thing again (a no-op the second time,
because of the `if (!isMusicOn)` guard at `Music.c:134`). Glider PRO therefore
keeps playing music while switched out during a game.

Compare the non-gameplay path, which gets it right by calling the two functions
explicitly: `GliderPRO/Sources/Events.c:418-420` (resume -> `StartMusic`) and
`:438-439` (suspend -> `StopTheMusic`).

The same misnaming makes `HandleStereo` (`GliderPRO/Sources/Dynamics.c:671-711`)
work: the stereo object flips `isPlayMusicGame` and then calls
`ToggleMusicWhilePlaying()` to reconcile, at `Dynamics.c:681` (turning on) and
`:698` (turning off). A port that "fixes" the function into a real toggle breaks
the stereo.

### 13.4 The editor plays the wrong sound

```c
case kSoundTrigger:                                          // Triggers.c:131
PlayPrioritySound(kChordSound, kChordPriority);	// Change me  // :132
break;                                                        // :133
```

In the editor, clicking a sound trigger plays a hard-coded chord instead of the
sound the trigger actually references. The author's own comment says so. The fix
is to call `LoadTriggerSound(theObject.data.e.where)` and then
`PlayPrioritySound(kTriggerSound, kTriggerPriority)`, matching what gameplay does
at `GliderPRO/Sources/Interactions.c:1071-1072`.

### 13.5 `SoundBytesNeeded` over-counts by 1260 bytes and leaks

```c
SetResLoad(false);                                    // Sound.c:498
for (i = 0; i < kMaxSounds - 1; i++)                  // :499
{
    theSound = GetResource('snd ', i + kBaseBufferSoundID);   // :501
    if (theSound == nil) { ... }                      // :502-506
    totalBytes += GetMaxResourceSize(theSound);       // :507
//  ReleaseResource(theSound);                        // :508  commented out
}
SetResLoad(true);                                     // :510
```

Two problems. First, `GetMaxResourceSize` returns the whole resource, but
`LoadBufferSounds` allocates `GetHandleSize(theSound) - 20L` (`Sound.c:332`), so
the estimate is 20 bytes per sound too large: 63 * 20 = **1260 bytes** for the
effects, plus 7 * 20 = **140** for the music in `MusicBytesNeeded`
(`GliderPRO/Sources/Music.c:386-407`, same shape). Harmless - it errs on the safe
side - but it means the memory check in `CheckMemorySize`
(`GliderPRO/Sources/Environ.c:562-692`) is looking for 1400 bytes more than the
game will use.

Second, the commented-out `ReleaseResource` leaks 63 (then 7) empty resource
handles, since `SetResLoad(false)` means each `GetResource` returns an unloaded
handle. 70 * 8ish bytes of Master Pointer, permanently. Irrelevant, but it is
why the sound-memory estimate is called exactly once at startup.

### 13.6 Details that look like bugs but are not

- `gameScore[2] = -1` (`Music.c:351`). Not a terminator: `MusicCallBack:186-190`
  treats a negative entry as "add me to the cursor and re-read", i.e. back up one
  slot. It is what makes the in-game music settle into RefrainSparse1 looping
  forever.
- `musicCursor = -1` for `kProdGameScoreMode` (`Music.c:157-158`). The callback
  increments before it reads (`Music.c:182`), so -1 becomes 0 and the *next* piece
  is `gameScore[0]`. Setting it to 0 would have skipped that piece.
- `musicCursor = 2` for `kKickGameScoreMode` (`Music.c:153-154`). Increments to 3,
  so the next pieces are `gameScore[3]`, `[4]`, `[5]` = RefrainSparse2, Chorus,
  Chorus - the 21.8-second flourish (4373.7 + 8718.6 + 8718.6 ms) that plays on a
  *vertical* or *teleporting* room change: `case kAbove`
  (`GliderPRO/Sources/Transit.c:223`), `case kBelow` (`:255`),
  `TransportRoomToRoom` (`:318`), `MoveDuctToDuct` (`:356`), `MoveMailToMail`
  (`:395`). Plain horizontal walking through a doorway gets the milder
  `kProdGameScoreMode` instead: `case kToRight` (`:159`), `case kToLeft` (`:191`).
- `thisMac.hasSM3 = true;	// TEMP` (`GliderPRO/Sources/Environ.c:451`). Not a
  bug in 1994; it hard-wires "Sound Manager 3 present", which is why
  `BitchAboutSM3` is unreachable and why every `UnivGetSoundVolume`/
  `UnivSetSoundVolume` call ignores its `hasSM3` argument
  (`GliderPRO/Sources/Utilities.c:744`, `:768`).
- The degenerate `loopStart = length - 2, loopEnd = length - 1` on 59 of the 70
  resources (section 3.10). It is what SoundEdit-era tools wrote for
  "non-looping"; `bufferCmd` ignores loop points anyway.

## Open questions

Things I could not settle from this tree. Each one names what evidence would
settle it.

1. **Does `flushCmd` discard an already-queued `callBackCmd`?**
   This decides whether `FlushAnyTriggerPlaying`
   (`GliderPRO/Sources/Sound.c:89-129`) permanently wedges a voice at priority 999
   (section 4.7, section 13.1). Apple's `Sound.h` and *Inside Macintosh: Sound* are
   not in this tree and the machine is airgapped. **Resolution:** read
   `Sound.h`'s comment on `flushCmd`, or run the original under an emulator with a
   house that has sound triggers and watch whether the third voice stops
   responding. A port should sidestep it by resetting the priority in the flush.

2. **What does `nullCmd`'s `param1 = 1964` mean?**
   `GliderPRO/Sources/Music.c:66-69` sends `cmd = 0, param1 = 1964,
   param2 = SetCurrentA5()`. Nothing reads it; the literal occurs nowhere else in
   the tree. Best guess is a debugging marker or a placeholder that once held a
   `callBackCmd`. **Resolution:** none available from source. Harmless to drop.

3. **The MACE 6:1 quantization tables.**
   Five house resources are `cmpSH` with `compressionID = 4` (`sixToOne`),
   `packetSize = 8` (section 6.4). MACE decompresses 1 byte to 6 samples using
   fixed tables that live in the Sound Manager, not in any shipped data here, and
   cannot be derived. **Resolution:** obtain the MACE tables (they are published in
   several open-source Mac audio decoders), or re-record those five sounds. Until
   then the port must ship them as silence. Affected: `CD Demo House` 3007
   "Door Chime", `Demo House` 3011 "Meow", `Nemo's Market` 3001 "Door Chime",
   3003 "Cash Register", 3004 "Cat Meow".

4. **Was the intended stereo image mono or centred?**
   `SndNewChannel` is called with `initMono` (0x0080) for all four channels
   (`GliderPRO/Sources/Sound.c:379`, `:387`, `:395`,
   `GliderPRO/Sources/Music.c:286-288`), and `UnivSetSoundVolume` writes the same
   value into both halves of the long (`GliderPRO/Sources/Utilities.c:782`). So
   the game is unambiguously mono. But the `kStereo` object exists
   (`GliderPRO/Headers/GliderDefines.h:404`, `= 0x69`) and toggles music, which
   invites a port to pan. **Resolution:** design decision, not a fact. Faithful =
   mono.

5. **Is `soundLoaded[kMaxSounds]` a vestige of an on-demand loader?**
   Declared at `GliderPRO/Sources/Sound.c:33` alongside `numSoundsLoaded`
   (`:31`); neither is ever read or written anywhere in the tree. Almost certainly
   left over from a design where sounds loaded lazily. **Resolution:** compare
   against Glider 4.0's sources if they are ever available. Delete in the port.

6. **What clips, and where?**
   The Sound Manager summed three 8-bit channels into its output buffer; I have
   asserted (section 12.1) that it clipped rather than wrapped, on the grounds that
   wrapping would be grossly audible and nobody reported it. I did not verify this
   against the mixer. **Resolution:** measure on real hardware or in an emulator
   with a known-loud triple hit. Practical impact is small: the priority system
   makes three simultaneous loud sounds rare.

7. **Do any houses in the wild use `'snd '` ids outside 3000..32767?**
   `GliderPRO/Sources/ObjectInfo.c:1237` constrains the editor's sound-trigger
   field to 3000..32767, and all 63 shipped house resources are 3000..3061
   (section 6.4). A hand-edited house could hold anything. **Resolution:** the port
   should treat "resource not found" as "no sound" rather than an error, which is
   what `LoadTriggerSound` already does (`GliderPRO/Sources/Sound.c:280-282`
   returns -1).

8. **What sound did the author intend for the 13 broken triggers?**
   `In The Mirror` has two objects pointing at `'snd '` 10000, which does not
   exist; `Teddy World` has eleven pointing at the editor default 3000
   (`GliderPRO/Sources/ObjectAdd.c:499`) and contains no `'snd '` resources at all
   (section 6.5). These are authoring mistakes in the shipped houses.
   **Resolution:** none. Reproduce the silence; do not substitute a sound.

9. **Precise behaviour when the trigger sound is disposed mid-playback.**
   `DumpTriggerSound` (`GliderPRO/Sources/Sound.c:307-312`) `DisposePtr`s the
   buffer that `FlushAnyTriggerPlaying` just tried to stop the mixer from reading
   (`GliderPRO/Sources/RoomGraphics.c:57-58` calls them in that order). If the
   flush does not take effect synchronously, the Sound Manager reads freed memory.
   Whether it does is the same unknown as question 1. In Go this is a non-issue -
   the slice stays alive as long as the voice references it.

## Porting notes

Ordered by how likely each is to bite.

1. **The engine is 3 voices plus 1 music voice, hard-coded.** Not a pool, not
   dynamic. `channel0/1/2` (`GliderPRO/Sources/Sound.c:29`) and `musicChannel`
   (`GliderPRO/Sources/Music.c:29`). Do not "improve" this to N voices: the
   priority table (section 5.2) is tuned to exactly three, and adding voices makes
   the game noticeably noisier than the original.

2. **The priority comparison is `>=`, not `>`.** `GliderPRO/Sources/Sound.c:68`.
   Ties displace. Get this wrong and rapid-fire sounds (thrust, tik/tok, band
   fire) stop retriggering, which changes the feel of the controls.

3. **Voice search order is 0, 1, 2 with strict-less-than.**
   `GliderPRO/Sources/Sound.c:53-66`. Channel 0 is always preferred. This is
   observable: it determines which sound gets cut when the third one arrives.

4. **Priority 999 is a sentinel, not a magnitude.** `kTriggerPriority` = 999
   (`GliderPRO/Headers/GliderDefines.h:180`). `PlayPrioritySound` does **not**
   rewrite the sound id: the only thing `Sound.c:47-51` does is *drop* the request
   outright if any of the three voices is already at 999, so two trigger sounds
   can never overlap. The trigger slot id is supplied by the caller -
   `PlayPrioritySound(kTriggerSound, kTriggerPriority)` with `kTriggerSound` = 63
   (`GliderPRO/Sources/Interactions.c:1072` and `:1618`) - and nothing checks that
   `theSoundData[63]` is non-nil before `PlaySoundN` dereferences it. In the
   original that is safe only because the hot spot is not created at all when
   `LoadTriggerSound` fails (section 6.3); a port must keep that coupling or add
   its own nil check.

5. **All 70 application resources are `stdSH`, 8-bit unsigned, 22254.5455 Hz.**
   Verified byte-for-byte (section 3.8). Convert with
   `s16 = (int16(u8) - 128) << 8`. Forgetting the -128 gives you a DC offset of
   half full scale and a very loud click at every sound start.

6. **`resource size == 42 + length` for all 70.** Which is why
   `LoadBufferSounds` can hard-code `- 20L` (`Sound.c:332`) and `bufferCmd` can
   hard-code `param2 = 20`. Verified. A port that instead parses the header
   properly (as `probe_snd.py` does) is strictly more robust and handles the house
   sounds too, which are *not* uniform.

7. **Loop points are present but never honoured.** `bufferCmd` plays a buffer
   once (section 3.10, proven two ways from the game's own control flow). The
   thrust/hiss/shred/sizzle sustained effects are re-triggered from the frame loop
   every 4 frames instead (`GliderPRO/Sources/Input.c:149-153`; 4 frames x 2 ticks
   x 16.6254 ms = 133.0 ms at `kTicksPerFrame` = 2, see section 4.14). Implement
   them as one-shots plus a retrigger timer, not
   as looping voices, or the "engaged" sounds will not stop when the key is
   released.

8. **Music is 7 whole PCM songs, not a sequencer.** `'snd '` 2000..2006
   (section 7.1). Playback is a hard-coded playlist walked by two tables,
   `musicScore[16]` and `gameScore[6]` (`GliderPRO/Sources/Music.c:332-354`).
   Gapless. Total whole-score cycle 87,230.1 ms.

9. **`ToggleMusicWhilePlaying` is not a toggle.** It reconciles music state with
   `isPlayMusicGame` (`GliderPRO/Sources/Music.c:125-142`). The stereo object
   depends on that (`GliderPRO/Sources/Dynamics.c:681`, `:698`). Renaming it to
   `SyncMusicToPreference` in the port is worth doing; making it an actual toggle
   is not.

10. **Volume was a *system* setting, hijacked.** `GetDefaultOutputVolume` /
    `SetDefaultOutputVolume` (`GliderPRO/Sources/Utilities.c:750`, `:783`), saved
    on launch (`GliderPRO/Sources/Main.c:194`) and restored on quit (`:278`).
    Replace with an internal gain. Keep the 0..7 integer scale for the prefs file
    (`prefsInfo.wasVolume`, `GliderPRO/Headers/Externs.h:243`) and reuse the
    original `*0x25` / `/0x24` arithmetic so old prefs round-trip (section 8.2).

11. **`isSoundOn == (volume != 0)` is an invariant, not two settings.**
    `GliderPRO/Sources/Settings.c:635`, `GliderPRO/Sources/Main.c:197-200`.

12. **Sounds can be silently disabled by the memory check.**
    `CheckMemorySize` (`GliderPRO/Sources/Environ.c:562-692`) sets
    `dontLoadMusic` and then `dontLoadSounds` when `FreeMem()` is short, and every
    audio entry point early-returns on those flags. A Go port has no equivalent
    constraint; keep the flags as a debug switch (`-nosound`) but never set them
    from a memory measurement.

13. **Big-endian everywhere on disk.** Every field in the `'snd '` resource, the
    resource-fork map, and the house data fork. Use `encoding/binary.BigEndian`.
    This matters at the extraction tool, not at run time, if you pre-convert.

14. **Interrupt-time callbacks become goroutines.** `CallBack0/1/2`
    (`Sound.c:217-261`) and `MusicCallBack` (`Music.c:170-216`) ran at interrupt
    level with the main thread halted, so unsynchronised `short` globals were safe.
    They are not safe in Go. Guard `priority0/1/2`, `soundPlaying0/1/2`,
    `musicCursor` and `musicSoundID` (section 12.2). Delete every `SetA5`/
    `SetCurrentA5` - they are 68k global-context switches with no Go analogue.

15. **The game does no DSP at all.** No pitch shift, no pan, no filter, no
    envelope, no fade. `rateCmd`, `rateMultiplierCmd`, `ampCmd` and `freqCmd` are
    never used; the only volume control is the system one (section 8.1). The
    fade-in/fade-out sounds (`'snd '` 1001, 1002) are *pre-rendered samples*, not
    a gain ramp.

16. **House sounds are per-room, singular, and hot-swapped.** One trigger sound
    is resident at a time (`theSoundData[63]`), loaded on room entry by
    `CreateActiveRects` -> `LoadTriggerSound`
    (`GliderPRO/Sources/ObjectRects.c:923-928`) and disposed on room exit
    (`GliderPRO/Sources/RoomGraphics.c:57-58`). `kMaxSoundTriggers` = 1
    (`GliderPRO/Sources/ObjectAdd.c:17`) enforces one per room in the editor. If a
    house did violate that, the ***first*** one loaded wins: `LoadTriggerSound`
    returns -1 as soon as `theSoundData[63] != nil`
    (`GliderPRO/Sources/Sound.c:271-272`), so the second object gets no hot spot
    and is inert. No shipped house has two `kSoundTrigger` objects in one room
    (verified over all 22 house files). Preloading
    all house sounds in the port is fine and simpler, but keep the "one active
    trigger sound per room" semantics or the wrong sound will play.

17. **The frame clock is 30 fps, 2 ticks per frame.** `kTicksPerFrame` = 2
    (`GliderPRO/Headers/GliderDefines.h:533`, used at
    `GliderPRO/Sources/Render.c:665`, `:690`). Audio retrigger intervals are
    expressed in frames, so the audio and the frame rate are coupled. If the port
    decouples them, scale the retrigger counters by the real elapsed time or the
    sustained sounds will change pitch character.
