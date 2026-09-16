# Glider PRO 1.0.4 — Input Handling and Controls

## Scope

This document reverse-specifies **every** input path in Glider PRO 1.0.4 (John Calhoun,
1994; GPLv2 source release) to the level of detail required to re-implement the game in Go
with byte-for-byte compatible preferences, frame-for-frame compatible demo playback, and
identical feel.

Covered:

* The two completely separate input mechanisms the game uses (polled `GetKeys` bitmap during
  play; Event Manager queue everywhere else), and where the boundary is.
* The Mac `KeyMap` bit-numbering convention, the `KeyMapOffsetFromRawKey` transform, and a
  fully verified offset ↔ raw-virtual-key-code table for every key constant in the source.
* The four remappable play actions (left / right / battery-helium / rubber bands), their
  defaults, their storage in the glider struct and in the prefs file.
* The non-remappable play keys: pause (Tab or Esc), Command-Q, Command-S, Delete (two-player
  suicide), and the arcade-build "any key aborts the demo" behaviour.
* Key repeat / hold semantics: what auto-repeats, what latches (`fireHeld`), what is
  edge-triggered, and what is level-triggered.
* Player 2's hard-coded, non-remappable key set.
* The control-remapping dialog (`DoControlPrefs` / `ControlFilter`), including the verified
  `DITL`/`DLOG` resource layout, the keys it refuses to accept, and the key-naming function.
* Preferences file format: verified 226-byte `prefsInfo` layout with every field's offset,
  the file's type/creator, version gate, and the delete-on-mismatch path.
* The demo (attract-mode) input recording/playback system, including a byte-level
  verification of the `'demo'` resource.
* Every editor keyboard command and every mouse interaction in the editor, map window, tools
  palette, link window and modal dialogs.
* Proof that there is **no** joystick support and **no** mouse input during play.

Not covered (owned by sibling documents): glider physics beyond what is needed to explain
what a thrust value *means*, object semantics, rendering, sound, house file format.

Line numbers in citations refer to the source files **after CR→LF conversion**
(`tr '\r' '\n'`), which is the form a modern editor shows. Paths are relative to the repo
root.

## Sources read

Primary (read in full):

| File | Lines (LF) | Why |
|---|---|---|
| `GliderPRO/Sources/Input.c` | 398 | The entire in-game input subsystem |
| `GliderPRO/Sources/Events.c` | 573 | Main event loop, key/mouse/OS event dispatch |
| `GliderPRO/Sources/Prefs.c` | 281 | Preferences file read/write/delete |
| `GliderPRO/Sources/Utilities.c` | 788 | `GetKeys` helpers, key-code transform, key naming |
| `GliderPRO/Sources/Settings.c` | 1482 | Control-remapping dialog and defaults |
| `GliderPRO/Headers/Externs.h` | 393 | **All** key constants and the `prefsInfo` struct |
| `GliderPRO/Headers/GliderDefines.h` | 625 | Build flags, mode/sound/tool constants |
| `GliderPRO/Headers/GliderStructs.h` | 347 | `gliderType`, `demoType` |

Secondary (read in the relevant parts):

`GliderPRO/Sources/Main.c`, `Play.c`, `Player.c`, `Modes.c`, `Render.c`, `RubberBands.c`,
`Interactions.c`, `Transit.c`, `Dynamics.c`, `InterfaceInit.c`, `StructuresInit2.c`,
`MainWindow.c`, `ObjectEdit.c`, `ObjectAdd.c`, `Room.c`, `RoomInfo.c`, `Marquee.c`, `Map.c`,
`Tools.c`, `Link.c`, `Menu.c`, `About.c`, `SelectHouse.c`, `GameOver.c`, `Banner.c`,
`HighScores.c`, `AnimCursor.c`, `InterfaceInit.c`, `DebugUtilities.c`, `RectUtils.c`,
`GliderPRO/Prefix.h`, `GliderPRO/Headers/GliderProtos.h`.

Binary / resource evidence (parsed with `python3`, observed values quoted inline):

`GliderPRO/Glider PRO.r` (15 MB, 199 843 lines) — `DITL` 1012/1023/1041, `DLOG` 1012/1023,
`ALRT` 1041/1042, `PICT` 1012/1013/1015/1016, `MENU` 128/129/130/131/140/141, `STR#`
129/160, `ICON` 1040–1043, and `data 'demo' (128)`.

Struct layouts were verified by compiling equivalent declarations with `#pragma pack(2)` and
`int` substituted for the classic-Mac 4-byte `long`, then printing `sizeof`/`offsetof`.

---

## 1. The control scheme in one page

Glider PRO is a single-screen-at-a-time flight game. The player's glider is a paper aeroplane
that constantly falls (`kGravity 3`, `GliderPRO/Sources/Player.c:13`) and is pushed around by
air currents. The player has exactly **four** in-play actions, all of which are remappable,
plus a handful of fixed system keys.

| # | Action | Effect | Remappable |
|---|---|---|---|
| 1 | Left | Bank/tilt left: set horizontal target velocity to `-kNormalThrust` | yes |
| 2 | Right | Bank/tilt right: set horizontal target velocity to `+kNormalThrust` | yes |
| 3 | Battery / Helium | Spend one unit of battery for a `kHyperThrust` horizontal kick, **or** one unit of helium for `kHeliumLift` upward lift | yes |
| 4 | Rubber bands | Fire one rubber band (max 2 airborne) | yes |
| — | Pause | Freeze the game and show a "paused" placard | no (choice of two keys) |
| — | Command-Q | Quit the game (offers to save first) | no |
| — | Command-S | Save the game in place | no |
| — | Delete | Two-player only: kill the straggling player | no |

Pressing **Left and Right simultaneously** is a fifth, emergent action: it performs an
about-face (turn around in place) rather than applying thrust
(`GliderPRO/Sources/Input.c:306-310`).

Everything else — menus, the room editor, dialogs, the house picker — is driven by the
classic Mac Event Manager and is completely separate code.

### 1.1 Default key map (player 1)

| Action | Default key | KeyMap constant | Offset (dec) | Raw virtual key code | Struct field | Prefs field |
|---|---|---|---|---|---|---|
| Left | ← Left arrow | `kLeftArrowKeyMap` | 124 | `0x7B` | `gliderType.leftKey` | `prefsInfo.wasLeftMap` @ +146 |
| Right | → Right arrow | `kRightArrowKeyMap` | 123 | `0x7C` | `gliderType.rightKey` | `prefsInfo.wasRightMap` @ +150 |
| Battery / Helium | ↓ Down arrow | `kDownArrowKeyMap` | 122 | `0x7D` | `gliderType.battKey` | `prefsInfo.wasBattMap` @ +154 |
| Rubber bands | ↑ Up arrow | `kUpArrowKeyMap` | 121 | `0x7E` | `gliderType.bandKey` | `prefsInfo.wasBandMap` @ +158 |
| Pause | Tab (default) | `kTabKeyMap` | 55 | `0x30` | — | `prefsInfo.wasEscPauseKey == false` @ +218 |
| Pause (alternative) | Esc | `kEscKeyMap` | 50 | `0x35` | — | `prefsInfo.wasEscPauseKey == true` @ +218 |
| Quit | Command + Q | `kCommandKeyMap` + `kQKeyMap` | 48 + 11 | `0x37` + `0x0C` | — | — |
| Save | Command + S | `kCommandKeyMap` + `kSKeyMap` | 48 + 6 | `0x37` + `0x01` | — | — |
| Suicide (2P) | Delete/Backspace | `kDeleteKeyMap` | 52 | `0x33` | — | — |

Defaults are established in two places that must agree:

* First-launch / no-prefs path: `GliderPRO/Sources/Main.c:135-138` and
  `GliderPRO/Sources/Main.c:184`.
* "Defaults" buttons in the prefs dialog: `GliderPRO/Sources/Settings.c:335-343` and
  `GliderPRO/Sources/Settings.c:1233-1241`.

### 1.2 Player 2 key map — hard-coded, not remappable

```c
theGlider2.leftKey  = kControlKeyMap;    // 60, raw 0x3B, physical Control
theGlider2.rightKey = kCommandKeyMap;    // 48, raw 0x37, physical Command (⌘)
theGlider2.battKey  = kOptionKeyMap;     // 61, raw 0x3A, physical Option
theGlider2.bandKey  = kShiftKeyMap;      // 63, raw 0x38, physical Shift
theGlider2.which    = kPlayer2;
```

`GliderPRO/Sources/InterfaceInit.c:148-152` (`theGlider.which = kPlayer1` is the preceding
line, `:147`). These four assignments happen once in
`VariableInit` and are never overwritten by prefs, by `DoControlPrefs`, or by anything else
(verified: `theGlider2.leftKey` and friends appear nowhere else in `Sources/`). There is no
UI to change them and they are not stored in the prefs file.

This is the reason player 2 uses the four modifier keys: they are the only keys that (a) are
readable from the `KeyMap` bitmap, (b) do not generate characters, and (c) are physically
clustered on the left of a Mac keyboard, opposite the arrow cluster on the right.

**Consequence for a port:** in two-player mode, player 2 holding Command triggers
"right"; player 1's `DoCommandKey()` check (`GliderPRO/Sources/Input.c:286`) fires on the
*same* bit. `DoCommandKey` then requires Q or S to actually do anything, so player 2 pressing
Command alone is harmless. The two branches are gated differently, and the difference matters:
Command-S is gated whole on `!twoPlayerGame` (`GliderPRO/Sources/Input.c:65`) and so does
nothing at all in a two-player game, but **Command-Q is ungated** — it sets `playing = false`
and `paused = false` unconditionally (`:57-58`), and only the *save prompt* inside it is
gated, on `(!twoPlayerGame) && (!demoGoing)` (`:59`). So a two-player game can still be quit
from the keyboard; it just cannot be saved.

---

## 2. Two input mechanisms, strictly separated

Glider PRO has two entirely different input systems, chosen by which mode the application is
in (`theMode`, values from `GliderPRO/Headers/GliderDefines.h:191-193`):

| `theMode` | Value | Loop | Input mechanism |
|---|---|---|---|
| `kSplashMode` | 0 | `HandleEvent()` | Event Manager (`WaitNextEvent`) |
| `kEditMode` | 1 | `HandleEvent()` | Event Manager (+ a `GetKeys` pre-poll for modifiers) |
| `kPlayMode` | 2 | `PlayGame()` | Polled `GetKeys()` bitmap only |

The application's outer loop is:

```c
while (!quitting)
    HandleEvent();
```

`GliderPRO/Sources/Main.c:363-364`. When a game starts, `NewGame()` →
`PlayGame()` (`GliderPRO/Sources/Play.c:213-214`) takes over and spins its own loop until
`playing` goes false; the Event Manager is then almost entirely bypassed.

### 2.1 Why polling, not events

During play the game needs to know *which keys are currently held down*, at a fixed 30 Hz,
with no dependence on the OS key-repeat rate and no queue latency. The classic Mac
`GetKeys(KeyMap)` call copies a 16-byte (128-bit) hardware key-state bitmap; one `GetKeys`
call gives the complete instantaneous state of the keyboard.

Consequences that a Go port must reproduce:

1. **There is no key-repeat concept in play.** Holding Left applies thrust on *every* frame,
   because the bit is set on every frame. There is no "initial delay then repeat" behaviour.
2. **Key-down and key-up edges are not visible.** Only level. Any edge-triggered behaviour
   (firing exactly one band per press) has to be synthesised in software — which is exactly
   what `fireHeld` does (§10).
3. **Modifier keys are ordinary keys.** Shift/Control/Option/Command each have their own bit
   and are usable as game buttons; this is what makes player 2's mapping possible.
4. **Dead keys, keyboard layout, and character generation are irrelevant.** The bitmap is
   indexed by physical key position (ADB scan code), so the mapping is layout-independent.
5. **No events are pumped during play** unless the `doBackground` preference is on
   (`GliderPRO/Sources/Play.c:437-443`). By default `doBackground = false`
   (`GliderPRO/Sources/Main.c:186`), so during a game the application services **zero**
   events: no window updates, no menu, no Apple events, no cursor tracking.

### 2.2 The single shared KeyMap snapshot

`Input.c` declares one file-scope snapshot:

```c
KeyMap theKeys;
```

`GliderPRO/Sources/Input.c:31`. It is a global, not a local. Both `GetInput` and
`GetDemoInput` refresh it **only when the glider being processed is player 1**:

```c
if (thisGlider->which == kPlayer1)
{
    GetKeys(theKeys);
    ...
}
```

`GliderPRO/Sources/Input.c:283-288` (and `:188-209` for the demo variant).

`kPlayer1` is `TRUE` and `kPlayer2` is `FALSE`
(`GliderPRO/Headers/GliderDefines.h:556-557`). In two-player mode `PlayGame` calls
`GetInput(&theGlider)` then `GetInput(&theGlider2)`
(`GliderPRO/Sources/Play.c:452-453`), so **both players read the identical 16-byte snapshot
taken at the start of player 1's processing**. This is not merely an optimisation: it
guarantees the two players see a consistent keyboard state within a frame. A Go port should
sample the keyboard once per frame into one snapshot and pass it to both players.

---

## 3. The Mac KeyMap: bit numbering and the offset transform

### 3.1 `BitTst` is MSB-first

`KeyMap` is `typedef UInt32 KeyMap[4]` — 16 bytes, 128 bits. The Toolbox `BitTst(ptr, long
bitNum)` numbers bits **starting from the most-significant bit of the first byte**. So for a
bit offset `n`:

```
byteIndex = n / 8
bitMask   = 0x80 >> (n % 8)
isDown    = (bytes[byteIndex] & bitMask) != 0
```

This convention is confirmed independently inside this codebase: `GliderPRO/Sources/Prefs.c:51`
tests a Gestalt response bit with `BitTst(&theFeature, 31 - gestaltFindFolderPresent)` — the
`31 -` is the classic idiom for converting an LSB-numbered bit position in a 32-bit word into
a `BitTst` MSB-numbered offset. If `BitTst` were LSB-first no such adjustment would be
needed.

**Getting this backwards is the single easiest way to break a port.** If you use
`1 << (n % 8)` you will read the wrong key for every offset whose low three bits are not 3
or 4, and the failure is silent — the arrow keys will simply do nothing.

### 3.2 `KeyMapOffsetFromRawKey`

The constants in `Externs.h` are `BitTst` offsets, not raw key codes. When the remapping
dialog receives a `keyDown` event it must convert the event's virtual key code into an
offset. That is done by:

```c
char KeyMapOffsetFromRawKey (char rawKeyCode)
{
    char hiByte, loByte, theOffset;

    hiByte = rawKeyCode & 0xF0;
    loByte = rawKeyCode & 0x0F;

    if (loByte <= 0x07)  theOffset = hiByte + (0x07 - loByte);
    else                 theOffset = hiByte + (0x17 - loByte);

    return (theOffset);
}
```

`GliderPRO/Sources/Utilities.c:504-517`.

This looks arbitrary; it is not. It reverses the bit order **within each byte** while leaving
the byte index alone, i.e. it converts between "MSB-first bit numbering" and "LSB-first bit
numbering". A closed form is:

```
offset = (v & ~7) + (7 - (v & 7))
```

Verified with python3 over the whole domain 0..127:

```
self-inverse failures: 0   closed-form mismatches: 0   bijective: True
```

So the function is **its own inverse** (`f(f(v)) == v` for all `v` in 0..127) and a bijection.
A Go port can use either the original branchy form or `offset = (v &^ 7) | (7 - v&7)`; they
are provably identical over the full byte range that matters. (For inputs ≥ 0x80 the C
version's `char` may be signed and the arithmetic is not meaningful; Mac virtual key codes
are always 0x00..0x7F, and `GetKeyMapFromMessage` masks with `keyCodeMask` before the call,
so this never arises.)

### 3.3 Event message → offset

```c
char GetKeyMapFromMessage (long message)
{
    long theVirtual = (message & keyCodeMask) >> 8;
    return KeyMapOffsetFromRawKey((char)theVirtual);
}
```

`GliderPRO/Sources/Utilities.c:522-530`. `keyCodeMask` is `0x0000FF00`; `charCodeMask` is
`0x000000FF`. So the low byte of an `EventRecord.message` is the generated character
(MacRoman) and the next byte up is the raw virtual key code.

### 3.4 Verified offset ↔ virtual-key-code table

Every KeyMap-offset constant declared in `GliderPRO/Headers/Externs.h:115-163`, with the
byte/mask decomposition (MSB-first) and the raw ADB virtual key code obtained by applying the
self-inverse transform. The right-hand column is an independent cross-check: the recovered
codes are exactly the documented Mac ADB virtual key codes for those physical keys, which
confirms both the constants and the bit-numbering convention.

| Constant | `Externs.h` line | Offset | Byte | Mask | Raw vkey | Physical key |
|---|---|---|---|---|---|---|
| `kXKeyMap` | 152 | 0 | 0 | `0x80` | `0x07` | X |
| `kZKeyMap` | 153 | 1 | 0 | `0x40` | `0x06` | Z |
| `kGKeyMap` | 140 | 2 | 0 | `0x20` | `0x05` | G |
| `kHKeyMap` | 141 | 3 | 0 | `0x10` | `0x04` | H |
| `kFKeyMap` | 139 | 4 | 0 | `0x08` | `0x03` | F |
| `kDKeyMap` | 137 | 5 | 0 | `0x04` | `0x02` | D |
| `kSKeyMap` | 148 | 6 | 0 | `0x02` | `0x01` | S |
| `kAKeyMap` | 134 | 7 | 0 | `0x01` | `0x00` | A |
| `kRKeyMap` | 147 | 8 | 1 | `0x80` | `0x0F` | R |
| `kEKeyMap` | 138 | 9 | 1 | `0x40` | `0x0E` | E |
| `kWKeyMap` | 151 | 10 | 1 | `0x20` | `0x0D` | W |
| `kQKeyMap` | 146 | 11 | 1 | `0x10` | `0x0C` | Q |
| `kBKeyMap` | 135 | 12 | 1 | `0x08` | `0x0B` | B |
| `kVKeyMap` | 150 | 14 | 1 | `0x02` | `0x09` | V |
| `kCKeyMap` | 136 | 15 | 1 | `0x01` | `0x08` | C |
| `kTKeyMap` | 149 | 22 | 2 | `0x02` | `0x11` | T |
| `kOKeyMap` | 144 | 24 | 3 | `0x80` | `0x1F` | O |
| `kPKeyMap` | 145 | 36 | 4 | `0x08` | `0x23` | P |
| `kPeriodKeyMap` | 154 | 40 | 5 | `0x80` | `0x2F` | `.` (period) |
| `kMKeyMap` | 142 | 41 | 5 | `0x40` | `0x2E` | M |
| `kNKeyMap` | 143 | 42 | 5 | `0x20` | `0x2D` | N |
| `kCommandKeyMap` | 155 | 48 | 6 | `0x80` | `0x37` | Command (⌘) |
| `kEscKeyMap` | 156 | 50 | 6 | `0x20` | `0x35` | Escape |
| `kDeleteKeyMap` | 157 | 52 | 6 | `0x08` | `0x33` | Delete / Backspace |
| `kSpaceBarMap` | 158 | 54 | 6 | `0x02` | `0x31` | Space |
| `kTabKeyMap` | 159 | 55 | 6 | `0x01` | `0x30` | Tab |
| `kControlKeyMap` | 160 | 60 | 7 | `0x08` | `0x3B` | Control |
| `kOptionKeyMap` | 161 | 61 | 7 | `0x04` | `0x3A` | Option |
| `kCapsLockKeyMap` | 162 | 62 | 7 | `0x02` | `0x39` | Caps Lock |
| `kShiftKeyMap` | 163 | 63 | 7 | `0x01` | `0x38` | Shift |
| `kPlusKeypadMap` | 115 | 66 | 8 | `0x20` | `0x45` | keypad `+` |
| `kTimesKeypadMap` | 117 | 68 | 8 | `0x08` | `0x43` | keypad `*` |
| `kMinusKeypadMap` | 116 | 73 | 9 | `0x40` | `0x4E` | keypad `-` |
| `k5KeypadMap` | 123 | 80 | 10 | `0x80` | `0x57` | keypad 5 |
| `k4KeypadMap` | 122 | 81 | 10 | `0x40` | `0x56` | keypad 4 |
| `k3KeypadMap` | 121 | 82 | 10 | `0x20` | `0x55` | keypad 3 |
| `k2KeypadMap` | 120 | 83 | 10 | `0x10` | `0x54` | keypad 2 |
| `k1KeypadMap` | 119 | 84 | 10 | `0x08` | `0x53` | keypad 1 |
| `k0KeypadMap` | 118 | 85 | 10 | `0x04` | `0x52` | keypad 0 |
| `k9KeypadMap` | 127 | 91 | 11 | `0x10` | `0x5C` | keypad 9 |
| `k8KeypadMap` | 126 | 92 | 11 | `0x08` | `0x5B` | keypad 8 |
| `k7KeypadMap` | 125 | 94 | 11 | `0x02` | `0x59` | keypad 7 |
| `k6KeypadMap` | 124 | 95 | 11 | `0x01` | `0x58` | keypad 6 |
| `kUpArrowKeyMap` | 129 | 121 | 15 | `0x40` | `0x7E` | ↑ |
| `kDownArrowKeyMap` | 130 | 122 | 15 | `0x20` | `0x7D` | ↓ |
| `kRightArrowKeyMap` | 131 | 123 | 15 | `0x10` | `0x7C` | → |
| `kLeftArrowKeyMap` | 132 | 124 | 15 | `0x08` | `0x7B` | ← |

Observations worth recording:

* The keypad offsets run **downwards** as the digit rises within each group (0→85, 1→84,
  … 5→80, then 6→95, 7→94, 8→92, 9→91). That is a direct consequence of the within-byte bit
  reversal, and confirms the recovered ADB codes `0x52`…`0x5C` are contiguous in the natural
  order. Offset 93 (raw `0x5A`, an unassigned code) has no constant, which is why 8 and 9 are
  not adjacent.
* Offset 13 (raw `0x0A`, the ISO `§`/`~` key) has no constant.
* There is **no** offset constant for Return, Enter, Page Up/Down, Home, End, Help, or the
  function keys. Those keys are only reachable through the Event Manager path (§4.2/§4.3), so
  they cannot be used as play controls unless the remapping dialog is given one — and it can
  be: `ControlFilter` accepts *any* key whose offset it can compute (§19.4), including Return,
  even though no named constant exists for it.

### 3.5 Go representation

A faithful representation is a `[16]byte` with the accessor:

```go
type KeyMap [16]byte

func (k *KeyMap) BitTst(off int) bool {
    return k[off>>3] & (0x80 >> uint(off&7)) != 0
}

func KeyMapOffsetFromRawKey(v int) int { return (v &^ 7) | (7 - v&7) }
```

Because the stored preference values are *offsets*, and because those offsets are what the
game compares against, a port that uses a different keyboard abstraction (SDL scancodes,
`ebiten.Key`, browser `KeyboardEvent.code`) must build a bidirectional table
`offset ↔ hostKey` covering at minimum every offset in §3.4, so that (a) old prefs files
keep working and (b) newly written prefs stay readable by the original game.

---

## 4. Complete constant inventory

### 4.1 Build flags (`GliderPRO/Headers/GliderDefines.h:11-16`)

| Line | Directive | State | Effect on input |
|---|---|---|---|
| 11 | `//#define CREATEDEMODATA` | **off** | Demo *recording* (`LogDemoKey`) compiled out |
| 12 | `//#define COMPILEDEMO` | **off** | Crippled-demo build; `DoLoadHouse`, `LoadFilter` etc. would be `#ifdef`-ed out |
| 13 | `//#define CAREFULDEBUG` | **off** | — |
| 14 | `#define COMPILENOCP` | **on** | "no copy protection": the `encrypted`/`fakeLong` prefs fields are dead |
| 15 | `#define COMPILEQT` | **on** | QuickTime movie support in play loop |
| 16 | `#define BUILD_ARCADE_VERSION 1` | **on** | Changes editor arrow keys into menu shortcuts, and makes *any* game key abort a demo |

`BUILD_ARCADE_VERSION` being on in the shipped source is important and easy to miss: it
alters both `HandleKeyEvent` (`GliderPRO/Sources/Events.c:191-207`) and `GetDemoInput`
(`GliderPRO/Sources/Input.c:192-201`). See §17.

`GliderPRO/Prefix.h` (CR-only, 7 lines of `#define`) sets `TARGET_CARBON 1`,
`ACCESSOR_CALLS_ARE_FUNCTIONS 1`, `OPAQUE_TOOLBOX_STRUCTS 1`, `OPAQUE_UPP_TYPES 1`,
`forCarbon 1`, `BUILDING_RUN_LINKED_IN 0`, `DEBUG 1`. It contains **no** alignment pragma.

### 4.2 ASCII / character-code constants (`GliderPRO/Headers/Externs.h:29-113`)

These are compared against `theEvent->message & charCodeMask` in the Event Manager paths.
They are MacRoman/ASCII control codes as produced by the classic Mac keyboard driver.

| Constant | Value | Line | Key |
|---|---|---|---|
| `kHomeKeyASCII` | `0x01` | 29 | Home |
| `kEnterKeyASCII` | `0x03` | 30 | Enter (keypad) |
| `kEndKeyASCII` | `0x04` | 31 | End |
| `kHelpKeyASCII` | `0x05` | 32 | Help |
| `kDeleteKeyASCII` | `0x08` | 33 | Delete/Backspace |
| `kTabKeyASCII` | `0x09` | 34 | Tab |
| `kPageUpKeyASCII` | `0x0B` | 35 | Page Up |
| `kPageDownKeyASCII` | `0x0C` | 36 | Page Down |
| `kReturnKeyASCII` | `0x0D` | 37 | Return |
| `kFunctionKeyASCII` | `0x10` | 38 | any F-key (disambiguate by vkey) |
| `kClearKeyASCII` | `0x1A` | 39 | Clear |
| `kEscapeKeyASCII` | `0x1B` | 40 | Escape (**also** Clear on some keyboards) |
| `kLeftArrowKeyASCII` | `0x1C` | 41 | ← |
| `kRightArrowKeyASCII` | `0x1D` | 42 | → |
| `kUpArrowKeyASCII` | `0x1E` | 43 | ↑ |
| `kDownArrowKeyASCII` | `0x1F` | 44 | ↓ |
| `kSpaceBarASCII` | `0x20` | 45 | Space |
| `kExclamationASCII` | `0x21` | 46 | `!` (used as a range bound) |
| `kPlusKeyASCII` | `0x2B` | 47 | `+` |
| `kMinusKeyASCII` | `0x2D` | 48 | `-` |
| `k0KeyASCII` … `k9KeyASCII` | `0x30`…`0x39` | 49–58 | digits |
| `kCapAKeyASCII` … `kCapZKeyASCII` | `0x41`…`0x5A` | 60–85 | A–Z |
| `kAKeyASCII` … `kZKeyASCII` | `0x61`…`0x7A` | 87–112 | a–z |
| `kForwardDeleteASCII` | `0x7F` | 113 | forward Delete (extended keyboard) |

### 4.3 Raw virtual key codes (`GliderPRO/Headers/Externs.h:165-181`)

| Constant | Value | Line |
|---|---|---|
| `kTabRawKey` | `0x30` | 165 |
| `kClearRawKey` | `0x47` | 166 |
| `kF5RawKey` | `0x60` | 167 |
| `kF6RawKey` | `0x61` | 168 |
| `kF7RawKey` | `0x62` | 169 |
| `kF3RawKey` | `0x63` | 170 |
| `kF8RawKey` | `0x64` | 171 |
| `kF9RawKey` | `0x65` | 172 |
| `kF11RawKey` | `0x67` | 173 |
| `kF13RawKey` | `0x69` | 174 |
| `kF14RawKey` | `0x6B` | 175 |
| `kF10RawKey` | `0x6D` | 176 |
| `kF12RawKey` | `0x6F` | 177 |
| `kF15RawKey` | `0x71` | 178 |
| `kF4RawKey` | `0x76` | 179 |
| `kF2RawKey` | `0x78` | 180 |
| `kF1RawKey` | `0x7A` | 181 |

`kTabRawKey 0x30` is consistent with `kTabKeyMap 55` under the §3.2 transform
(`f(0x30) = 55`), a further self-check of the whole scheme. Note that the F-key codes are
*not* in ascending numeric order relative to their labels — that is genuinely how Apple's
extended keyboard numbered them, and `GetKeyName` hard-codes the mapping
(`GliderPRO/Sources/Utilities.c:598-648`).

### 4.4 Input-relevant constants in `Input.c`

| Constant | Value | Line | Meaning |
|---|---|---|---|
| `kNormalThrust` | 5 | `GliderPRO/Sources/Input.c:14` | Target horizontal velocity magnitude for left/right |
| `kHyperThrust` | 8 | `GliderPRO/Sources/Input.c:15` | Direct `hVel` impulse added per frame of battery use |
| `kHeliumLift` | 4 | `GliderPRO/Sources/Input.c:16` | Target vertical velocity (negative = up) for helium |
| `kEscPausePictID` | 1015 | `GliderPRO/Sources/Input.c:17` | "Esc to resume" placard PICT |
| `kTabPausePictID` | 1016 | `GliderPRO/Sources/Input.c:18` | "Tab to resume" placard PICT |
| `kSavingGameDial` | 1042 | `GliderPRO/Sources/Input.c:19` | **Declared but never used** in this file |
| `kSaveGameAlert` | 1041 | `GliderPRO/Sources/Input.c:385` | Quit-confirmation ALRT |
| `kYesSaveGameButton` | 1 | `GliderPRO/Sources/Input.c:386` | Item number of "Save First" |

### 4.5 Physics constants needed to interpret the thrust values

| Constant | Value | Citation |
|---|---|---|
| `kGravity` | 3 | `GliderPRO/Sources/Player.c:13` |
| `kHImpulse` | 2 | `GliderPRO/Sources/Player.c:14` |
| `kVImpulse` | 2 | `GliderPRO/Sources/Player.c:15` |
| `kMaxHVel` | 16 | `GliderPRO/Sources/Player.c:16` |
| `kRubberBandVelocity` | 20 | `GliderPRO/Sources/RubberBands.c:12` |
| `kMaxRubberBands` | 2 | `GliderPRO/Headers/GliderDefines.h:261` |
| `kTicksPerFrame` | 2 | `GliderPRO/Headers/GliderDefines.h:533` |

`MoveGlider` (`GliderPRO/Sources/Player.c:64-147`; the velocity ramp is `:66-92`) ramps the actual velocity towards the
desired velocity by `kHImpulse`/`kVImpulse` per frame, then **resets** `hDesiredVel = 0` and
`vDesiredVel = kGravity`:

```c
if (hVel > hDesiredVel) { hVel -= kHImpulse; if (hVel < hDesiredVel) hVel = hDesiredVel; }
else if (hVel < hDesiredVel) { hVel += kHImpulse; if (hVel > hDesiredVel) hVel = hDesiredVel; }
hDesiredVel = 0;
...same for vertical...
vDesiredVel = kGravity;
```

So:

* **Left/Right (`kNormalThrust 5`)** sets a *target*. From rest, holding Right ramps
  `hVel` 0 → 2 → 4 → 5 over three frames (100 ms) and then holds at 5. Releasing ramps back
  to 0 at 2/frame.
* **Battery (`kHyperThrust 8`)** modifies `hVel` **directly**, bypassing the ramp, so it can
  push `hVel` far past 5 (clamped to `±kMaxHVel = ±16`). It is applied every frame the key is
  held and costs one battery unit per frame.
* **Helium (`kHeliumLift 4`)** sets `vDesiredVel = -4`, which is ramped at 2/frame, i.e. it
  fights gravity (`+3`) rather than teleporting the glider upwards.

Because `hDesiredVel` is zeroed every frame by `MoveGlider`, the `+=` in
`GliderPRO/Sources/Input.c:313` behaves as `=` for player input; the `+=` form exists so that
input can compose with other sources that also write `hDesiredVel` in the same frame
(e.g. `GliderPRO/Sources/Dynamics.c:54-58` shoving from dynamic objects).

### 4.6 Sound constants triggered from input

| Constant | Value | Citation | Fired by |
|---|---|---|---|
| `kThrustSound` | 18 | `GliderPRO/Headers/GliderDefines.h:73` | battery, every 4th frame |
| `kFizzleSound` | 19 | `GliderPRO/Headers/GliderDefines.h:74` | battery/helium exhausted |
| `kFireBandSound` | 20 | `GliderPRO/Headers/GliderDefines.h:75` | band launched |
| `kHissSound` | 62 | `GliderPRO/Headers/GliderDefines.h:117` | helium, every 4th frame |
| `kFadeOutSound` | 2 | `GliderPRO/Headers/GliderDefines.h:57` | Delete-suicide |
| `kThrustPriority` | 300 | `GliderPRO/Headers/GliderDefines.h:129` | |
| `kFireBandPriority` | 301 | `GliderPRO/Headers/GliderDefines.h:130` | |
| `kHissPriority` | 311 | `GliderPRO/Headers/GliderDefines.h:140` | |
| `kFizzlePriority` | 703 | `GliderPRO/Headers/GliderDefines.h:159` | |

### 4.7 Glider mode values relevant to input gating

`GliderPRO/Headers/GliderDefines.h:571-594` (`kGliderNormal 0` … `kGliderTransportingIn 23`,
24 contiguous values).

| Constant | Value | Input consequence |
|---|---|---|
| `kGliderNormal` | 0 | All four actions live |
| `kGliderFaceLeft` | 7 | About-face in progress; battery & bands blocked |
| `kGliderFaceRight` | 8 | About-face in progress; battery & bands blocked |
| `kGliderBurning` | 9 | Input **ignored**; auto-thrust in facing direction |
| `kGliderInLimbo` | 21 | Player is waiting for the other player (2P) |
| `kGliderTransportingIn` | 23 | Battery & bands blocked (mode ≠ normal) |

Also `kFaceRight = TRUE`, `kFaceLeft = FALSE`
(`GliderPRO/Headers/GliderDefines.h:554-555`); `kPlayer1 = TRUE`, `kPlayer2 = FALSE`
(`:556-557`); `kNoOneEscaped = -1` (`:608`).

---

## 5. Where the four mappings live: `gliderType`

### 5.1 Declaration

```c
typedef struct
{
    Rect        src, mask, dest, whole;
    Rect        destShadow, wholeShadow;
    Rect        clip, enteredRect;
    long        leftKey, rightKey;
    long        battKey, bandKey;
    short       hVel, vVel;
    short       wasHVel, wasVVel;
    short       vDesiredVel, hDesiredVel;
    short       mode, frame, wasMode;
    Boolean     facing, tipped;
    Boolean     sliding, ignoreLeft, ignoreRight;
    Boolean     fireHeld, which;
    Boolean     heldLeft, heldRight;
    Boolean     dontDraw, ignoreGround;
} gliderType, *gliderPtr;
```

`GliderPRO/Headers/GliderStructs.h:200-216`.

### 5.2 Verified layout

`GliderPRO/Headers/GliderStructs.h` contains **no** `#pragma` at all (verified: zero matches
for `pragma` in the file), so the struct is laid out under whatever alignment the compiler is
using at the point of inclusion. Classic Mac 68k/PPC compilers default to 2-byte alignment for
this code base, and `Externs.h` explicitly forces `align=mac68k` around `prefsInfo`. Compiling
an equivalent declaration with `#pragma pack(2)` and `int` (4 bytes) substituted for
classic-Mac `long` gives:

```
sizeof(Rect)       =   8
sizeof(gliderType) = 110
```

Offsets (bytes, from the start of the struct):

| Offset | Size | Field | Type | Input relevance |
|---|---|---|---|---|
| 0 | 8 | `src` | Rect | source rect in the glider sprite GWorld |
| 8 | 8 | `mask` | Rect | |
| 16 | 8 | `dest` | Rect | **read by band firing** (`dest.left+24`, `dest.top+10`) |
| 24 | 8 | `whole` | Rect | |
| 32 | 8 | `destShadow` | Rect | |
| 40 | 8 | `wholeShadow` | Rect | |
| 48 | 8 | `clip` | Rect | |
| 56 | 8 | `enteredRect` | Rect | |
| **64** | 4 | **`leftKey`** | long | KeyMap offset of the Left action |
| **68** | 4 | **`rightKey`** | long | KeyMap offset of the Right action |
| **72** | 4 | **`battKey`** | long | KeyMap offset of the Battery/Helium action |
| **76** | 4 | **`bandKey`** | long | KeyMap offset of the Bands action |
| 80 | 2 | `hVel` | short | written directly by battery (`kHyperThrust`) |
| 82 | 2 | `vVel` | short | |
| 84 | 2 | `wasHVel` | short | |
| 86 | 2 | `wasVVel` | short | |
| 88 | 2 | `vDesiredVel` | short | written by helium (`-kHeliumLift`) |
| 90 | 2 | `hDesiredVel` | short | written by left/right (`±kNormalThrust`) |
| 92 | 2 | `mode` | short | gates battery/bands on `== kGliderNormal` |
| 94 | 2 | `frame` | short | |
| 96 | 2 | `wasMode` | short | |
| 98 | 1 | `facing` | Boolean | `kFaceRight TRUE` / `kFaceLeft FALSE` |
| 99 | 1 | `tipped` | Boolean | **set by input**, consumed by sprite pick + battery |
| 100 | 1 | `sliding` | Boolean | sprite pick |
| 101 | 1 | `ignoreLeft` | Boolean | collision flag, **not** input |
| 102 | 1 | `ignoreRight` | Boolean | collision flag, **not** input |
| **103** | 1 | **`fireHeld`** | Boolean | one-shot latch for the band key |
| **104** | 1 | **`which`** | Boolean | `kPlayer1 TRUE` / `kPlayer2 FALSE`; gates `GetKeys` |
| **105** | 1 | **`heldLeft`** | Boolean | set by input, read by `Interactions.c` |
| **106** | 1 | **`heldRight`** | Boolean | set by input, read by `Interactions.c` |
| 107 | 1 | `dontDraw` | Boolean | |
| 108 | 1 | `ignoreGround` | Boolean | |
| 109 | 1 | *(tail padding)* | — | struct rounded to even size |

Note that the key fields are `long` (4 bytes) even though every value they ever hold is in
0..127. `BitTst`'s second argument is a `long`, so the widths match and no cast is written at
the call sites (`GliderPRO/Sources/Input.c:301`, `:306`, `:318`, `:330`, `:344`).

`ignoreLeft` / `ignoreRight` are **not** input state despite the suggestive names; they are set
by collision resolution (`GliderPRO/Sources/Interactions.c:1417` and `:1421`), consumed at
`Interactions.c:515`, `:578`, `:605`, `:668`, and cleared at the end of every
`HandleGlider` (`GliderPRO/Sources/Player.c:1436-1438`) and by `FlagGliderNormal`
(`GliderPRO/Sources/Modes.c:356-357`).

### 5.3 Initialisation of the input-related fields

| Field | Reset where | Value |
|---|---|---|
| `leftKey`/`rightKey`/`battKey`/`bandKey` (P1) | `ReadInPrefs`, `GliderPRO/Sources/Main.c:69-72` (from prefs) or `:135-138` (defaults) | offsets |
| `leftKey`/`rightKey`/`battKey`/`bandKey` (P1) | `DoControlPrefs` commit, `GliderPRO/Sources/Settings.c:553-556` | offsets |
| `leftKey`/`rightKey`/`battKey`/`bandKey` (P2) | `VariableInit`, `GliderPRO/Sources/InterfaceInit.c:148-151` | fixed |
| `which` (P1) | `GliderPRO/Sources/InterfaceInit.c:147` | `kPlayer1` |
| `which` (P2) | `GliderPRO/Sources/InterfaceInit.c:152` | `kPlayer2` |
| `tipped`, `sliding`, `dontDraw` | `InitGlider`, `GliderPRO/Sources/Play.c:363-365` | `false` |
| `hVel`,`vVel`,`hDesiredVel`,`vDesiredVel` | `InitGlider`, `GliderPRO/Sources/Play.c:358-361` | 0 |
| `tipped`, `ignoreLeft`, `ignoreRight`, `ignoreGround`, `dontDraw`, `frame` | `FlagGliderNormal`, `GliderPRO/Sources/Modes.c:355-360` | `false`/0 |

**`fireHeld`, `heldLeft` and `heldRight` are never explicitly initialised.** They are set
every frame by `GetInput`/`GetDemoInput` before use, and `theGlider`/`theGlider2` are
file-scope globals (zero-initialised in C), so the effective initial value is `false`. A Go
port gets the same behaviour from zero values, but should not rely on it silently — set them
explicitly.

---

## 6. Frame timing: input is sampled exactly 30 times a second

`PlayGame` runs an unthrottled `while` loop; the throttle is inside `RenderFrame`:

```c
while (TickCount() < nextFrame)
{
}
nextFrame = TickCount() + kTicksPerFrame;
```

`GliderPRO/Sources/Render.c:662-665`, with `long nextFrame;` declared at
`GliderPRO/Sources/Render.c:40` and re-primed in `InitGarbageRects`
(`GliderPRO/Sources/Render.c:690`).

`TickCount()` counts 1/60 s ticks. `kTicksPerFrame` is 2
(`GliderPRO/Headers/GliderDefines.h:533`). So:

* Nominal frame rate: **30 fps**, one input sample per frame.
* The wait is a **busy spin**, not a sleep. On a Mac of the era that was acceptable; a Go port
  must not do this (see Porting notes).
* If a frame overruns 2 ticks, the loop does not catch up — `nextFrame` is recomputed from the
  *current* `TickCount`, so the game simply runs slower and the input sampling rate drops with
  it. There is no frame-skipping and no accumulator. **Demo playback keys off `gameFrame`, not
  wall clock**, so demos stay in sync even on a slow machine (§15).

Per-frame order within `PlayGame` (`GliderPRO/Sources/Play.c:430-495`):

1. `gameFrame++`; `evenFrame = !evenFrame` (`:434-435`)
2. If `doBackground`: pump events until not switched out (`:437-443`)
3. `HandleTelephone()` (`:445`)
4. `HandleDynamics()` (`:449` / `:475`)
5. **`GetInput(&theGlider)` [+ `GetInput(&theGlider2)`] or `GetDemoInput(&theGlider)`**
   (`:452-453` two-player; `:478-481` one-player)
6. `HandleInteraction()` (`:454` / `:482`)
7. `HandleTriggers()` (`:456` / `:484`)
8. `HandleBands()` (`:457` / `:485`)
9. `HandleGlider(...)` → `MoveGlider` applies the velocity ramp (`:460-461` / `:487`)
10. `RenderFrame()` — **busy-waits here** (`:469` / `:494`)
11. `HandleDynamicScoreboard()` (`:470` / `:495`)

So input is read *before* interactions and *before* physics integration in the same frame:
a key press affects the same frame it is sampled in.

---

## 7. `GetInput` — the live input routine

Source: `GliderPRO/Sources/Input.c:281-379`. This is the single most important routine in the
subsystem. Numbered pseudocode, original identifiers in parentheses:

```
GetInput(thisGlider):

 1. if thisGlider.which == kPlayer1:                       # Input.c:283
 2.     GetKeys(theKeys)                                   # Input.c:285  (the ONLY sample)
 3.     if BitTst(theKeys, kCommandKeyMap):                # Input.c:286
 4.         DoCommandKey()                                 # Input.c:287  (see §11)
 5.
 6. if thisGlider.mode == kGliderBurning:                  # Input.c:290
 7.     # burning glider is uncontrollable, thrusts forward on its own
 8.     if thisGlider.facing == kFaceLeft:
 9.         thisGlider.hDesiredVel -= kNormalThrust         # Input.c:293  (5)
10.     else:
11.         thisGlider.hDesiredVel += kNormalThrust         # Input.c:295
12.     return                                             # (implicit; rest is in else)
13.
14. # ---- normal (non-burning) path ----
15. thisGlider.heldLeft  = false                           # Input.c:299
16. thisGlider.heldRight = false                           # Input.c:300
17.
18. if BitTst(theKeys, thisGlider.rightKey):               # Input.c:301
19.     [LogDemoKey(0)]                                    # Input.c:304  (CREATEDEMODATA only)
20.     if BitTst(theKeys, thisGlider.leftKey):            # Input.c:306  BOTH keys held
21.         ToggleGliderFacing(thisGlider)                 # Input.c:308  about-face
22.         thisGlider.heldLeft = true                     # Input.c:309
23.         # NOTE: no thrust, and `tipped` is NOT touched — it keeps last frame's value
24.     else:
25.         thisGlider.hDesiredVel += kNormalThrust         # Input.c:313  (+5)
26.         thisGlider.tipped = (thisGlider.facing == kFaceLeft)   # Input.c:314
27.         thisGlider.heldRight = true                    # Input.c:315
28. elif BitTst(theKeys, thisGlider.leftKey):              # Input.c:318
29.     [LogDemoKey(1)]                                    # Input.c:321
30.     thisGlider.hDesiredVel -= kNormalThrust            # Input.c:323  (-5)
31.     thisGlider.tipped = (thisGlider.facing == kFaceRight)      # Input.c:324
32.     thisGlider.heldLeft = true                         # Input.c:325
33. else:
34.     thisGlider.tipped = false                          # Input.c:328
35.
36. # ---- battery / helium ----
37. if BitTst(theKeys, thisGlider.battKey)                 # Input.c:330
38.        and batteryTotal != 0
39.        and thisGlider.mode == kGliderNormal:           # Input.c:331
40.     [LogDemoKey(2)]                                    # Input.c:334
41.     if batteryTotal > 0:  DoBatteryEngaged(thisGlider) # Input.c:337
42.     else:                 DoHeliumEngaged(thisGlider)  # Input.c:339
43. else:
44.     batteryWasEngaged = false                          # Input.c:342
45.
46. # ---- rubber bands ----
47. if BitTst(theKeys, thisGlider.bandKey)                 # Input.c:344
48.        and bandsTotal > 0
49.        and thisGlider.mode == kGliderNormal:           # Input.c:345
50.     [LogDemoKey(3)]                                    # Input.c:348
51.     if not thisGlider.fireHeld:                        # Input.c:350
52.         if AddBand(thisGlider,
53.                    thisGlider.dest.left + 24,
54.                    thisGlider.dest.top  + 10,
55.                    thisGlider.facing):                 # Input.c:352-353
56.             bandsTotal--                               # Input.c:355
57.             if bandsTotal <= 0: QuickBandsRefresh(false)   # Input.c:356-357
58.             thisGlider.fireHeld = true                 # Input.c:359
59. else:
60.     thisGlider.fireHeld = false                        # Input.c:364
61.
62. # ---- two-player suicide ----
63. if otherPlayerEscaped != kNoOneEscaped                 # Input.c:366
64.        and BitTst(theKeys, kDeleteKeyMap)              # Input.c:367
65.        and thisGlider.which                            # Input.c:368  (== kPlayer1)
66.        and not onePlayerLeft:
67.     ForceKillGlider()                                  # Input.c:370
68.
69. # ---- pause ----
70. if (isEscPauseKey and BitTst(theKeys, kEscKeyMap))     # Input.c:373
71.        or (not isEscPauseKey and BitTst(theKeys, kTabKeyMap)):   # Input.c:374
72.     DoPause()                                          # Input.c:376
```

### 7.1 Key semantics extracted from the above

| Behaviour | Detail | Citation |
|---|---|---|
| Right beats Left | The test order is `if right … elif left`, so with both held the **right** branch runs and does the about-face | `Input.c:301`, `:318` |
| Both keys = about-face, no thrust | The both-held branch applies **no** `hDesiredVel` change | `Input.c:306-310` |
| Both keys does not reset `tipped` | `tipped` is only written in the three other branches; in the both-held branch it retains the previous frame's value | `Input.c:306-310` |
| About-face is not spammable | `ToggleGliderFacing` returns immediately unless `mode == kGliderNormal`, and it *sets* `mode` to `kGliderFaceLeft`/`kGliderFaceRight`, so repeated calls during the 3-frame animation are no-ops | `GliderPRO/Sources/Modes.c:484-493` |
| Burning glider ignores all input | Nothing after the burning branch runs — not even pause or Command-Q | `Input.c:290-296` |
| Battery/bands need `kGliderNormal` | During about-face, fade-in, transport-in etc. they are dead | `Input.c:331`, `:345` |
| Battery/helium auto-repeat freely | Held key spends one unit **per frame** (30/second) | `Input.c:330` |
| Bands are one-shot per press | `fireHeld` latch | `Input.c:350-364` |
| Left/Right auto-repeat freely | Level-triggered thrust every frame | `Input.c:301-326` |
| Only player 1 can press Command-Q/S | Wrapped in the `which == kPlayer1` block | `Input.c:283-288` |
| Only player 1 can press Delete | `thisGlider->which` is part of the condition | `Input.c:368` |
| Both players can pause | The pause test is outside the `kPlayer1` block, but re-entry is prevented because `DoPause` exits only after the key is released | `Input.c:373-377` |

### 7.2 The about-face animation, since it is an input-triggered mode change

```
ToggleGliderFacing(thisGlider):                       # Modes.c:484
    if thisGlider.mode != kGliderNormal: return       # Modes.c:486-487
    if thisGlider.facing == kFaceLeft:
        FlagGliderFaceRight(thisGlider)               # Modes.c:490
    else:
        FlagGliderFaceLeft(thisGlider)                # Modes.c:492

FlagGliderFaceLeft:   mode = kGliderFaceLeft  (7); frame = kLastAboutFaceFrame  (20)   # Modes.c:438-444
FlagGliderFaceRight:  mode = kGliderFaceRight (8); frame = kFirstAboutFaceFrame (18)   # Modes.c:448-454
```

`kFirstAboutFaceFrame 18`, `kLastAboutFaceFrame 20`
(`GliderPRO/Headers/GliderDefines.h:560-561`). The animation is driven from `HandleGlider`:

```
MoveGliderFaceLeft:  draw gliderSrc[frame]; MoveGlider(); frame--;
                     if frame < 18 → mode = kGliderNormal, facing = kFaceLeft    # Player.c:560-573
MoveGliderFaceRight: draw gliderSrc[frame]; MoveGlider(); frame++;
                     if frame > 20 → mode = kGliderNormal, facing = kFaceRight   # Player.c:577-590
```

So the about-face is exactly **3 frames** (100 ms) long, drawing sprite indices 20→19→18 (or
18→19→20). During those frames:

* Battery/helium and rubber bands **are** locked out, because both tests in `GetInput` require
  `thisGlider->mode == kGliderNormal` (`GliderPRO/Sources/Input.c:331`, `:345`).
* Direction-key thrust is **not** locked out. `GetInput` is called unconditionally every frame
  (`GliderPRO/Sources/Play.c:452-453`, `:481`) and its steering branch is gated only on
  `mode == kGliderBurning`, so `hDesiredVel` is still written; `MoveGliderFaceLeft` /
  `MoveGliderFaceRight` both call `MoveGlider` (`Player.c:565`, `:582`), which consumes it.
  A port must keep applying horizontal thrust through the animation.
* A second `ToggleGliderFacing` is a no-op while the animation runs, because of the
  `mode != kGliderNormal` early return (`Modes.c:486-487`) — so holding both direction keys
  does not spin the glider; it turns once per 3-frame animation.

---

## 8. Battery and helium: `DoBatteryEngaged` / `DoHeliumEngaged`

The same key (`battKey`, default ↓) drives two opposite effects selected by the **sign** of the
global `batteryTotal`:

* `batteryTotal > 0` → **battery**: horizontal boost, counter decrements towards 0.
* `batteryTotal < 0` → **helium**: upward lift, counter increments towards 0.
* `batteryTotal == 0` → the key does nothing (`Input.c:330` requires `batteryTotal != 0`).

### 8.1 `DoBatteryEngaged` (`GliderPRO/Sources/Input.c:121-156`)

```
 1. if thisGlider.facing == kFaceLeft:                      # Input.c:123
 2.     if thisGlider.tipped: thisGlider.hVel += kHyperThrust   # Input.c:126  (+8)
 3.     else:                 thisGlider.hVel -= kHyperThrust   # Input.c:128  (-8)
 4. else:                                                   # facing right
 5.     if thisGlider.tipped: thisGlider.hVel -= kHyperThrust   # Input.c:133  (-8)
 6.     else:                 thisGlider.hVel += kHyperThrust   # Input.c:135  (+8)
 7.
 8. batteryTotal--                                          # Input.c:138
 9.
10. if batteryTotal == 0:                                   # Input.c:140
11.     QuickBatteryRefresh(false)                          # Input.c:142  (scoreboard)
12.     PlayPrioritySound(kFizzleSound, kFizzlePriority)    # Input.c:143  (19, 703)
13. else:
14.     if not batteryWasEngaged: batteryFrame = 0          # Input.c:147-148
15.     if batteryFrame == 0:
16.         PlayPrioritySound(kThrustSound, kThrustPriority)   # Input.c:150  (18, 300)
17.     batteryFrame++                                      # Input.c:151
18.     if batteryFrame >= 4: batteryFrame = 0               # Input.c:152-153
19.     batteryWasEngaged = true                            # Input.c:154
```

Key points:

* The thrust direction depends on **both** `facing` and `tipped`. `tipped` is true when the
  player is holding the direction key *opposite* to the way the glider faces (see
  `Input.c:314` and `:324`), i.e. the glider is flying backwards/banked. In that state the
  battery pushes the *other* way. Net effect: the battery always pushes in the direction the
  player is steering, not the direction the nose points.
* `hVel` is written **directly**, bypassing the `kHImpulse` ramp, so the boost is instant and
  can exceed `kNormalThrust`; `MoveGlider` clamps to `±kMaxHVel = ±16`
  (`GliderPRO/Sources/Player.c:96-97`, `:113`).
* The sound plays on frames 0, 4, 8, … of a continuous hold — i.e. every 4th frame
  (every 8 ticks ≈ 7.5 Hz). `batteryFrame` is reset to 0 whenever the key was *not* held on
  the previous frame (`batteryWasEngaged == false`), so every fresh press starts with an
  immediate sound.
* The fizzle at exactly `batteryTotal == 0` fires **once**, on the frame the last unit is
  consumed, because the next frame `batteryTotal != 0` is false and the key is ignored.

### 8.2 `DoHeliumEngaged` (`GliderPRO/Sources/Input.c:160-182`)

```
 1. thisGlider.vDesiredVel = -kHeliumLift                   # Input.c:162  (-4)
 2. batteryTotal++                                          # Input.c:163
 3. if batteryTotal == 0:                                   # Input.c:165
 4.     QuickBatteryRefresh(false)                          # Input.c:167
 5.     PlayPrioritySound(kFizzleSound, kFizzlePriority)    # Input.c:168
 6.     batteryWasEngaged = false                           # Input.c:169  ← asymmetry!
 7. else:
 8.     if not batteryWasEngaged: batteryFrame = 0          # Input.c:173-174
 9.     if batteryFrame == 0:
10.         PlayPrioritySound(kHissSound, kHissPriority)    # Input.c:176  (62, 311)
11.     batteryFrame++                                      # Input.c:177
12.     if batteryFrame >= 4: batteryFrame = 0               # Input.c:178-179
13.     batteryWasEngaged = true                            # Input.c:180
```

Differences from the battery path that must be preserved:

* `vDesiredVel` is **assigned** (`=`), not accumulated, and it is a *target* velocity, so the
  glider accelerates upward at `kVImpulse = 2` per frame until `vVel == -4`. `MoveGlider`
  then resets `vDesiredVel = kGravity (3)` at the end of the frame
  (`GliderPRO/Sources/Player.c:92`), so the lift must be re-asserted every frame.
* `-kHeliumLift = -4` versus gravity `+3`: net upward acceleration exists but is small; the
  glider rises slowly.
* On exhaustion the helium path additionally sets `batteryWasEngaged = false`
  (`Input.c:169`) while the battery path does **not** (`Input.c:140-144` has no such line).
  This is almost certainly an oversight in the original, but it is observable: after a battery
  runs dry, `batteryWasEngaged` stays `true` until the next frame's `else` branch at
  `Input.c:342` clears it. Since the very next frame the key test fails (`batteryTotal == 0`),
  `Input.c:342` runs and clears it anyway, so there is no visible difference. Reproduce it
  verbatim rather than "fixing" it.
* Helium is *not* gated on `tipped` or `facing`.

### 8.3 Battery/helium state variables

| Variable | Type | Declared | Scope | Purpose |
|---|---|---|---|---|
| `batteryTotal` | short | `GliderPRO/Sources/Interactions.c` (global) | game | signed fuel counter; `>0` battery, `<0` helium |
| `batteryFrame` | short | `GliderPRO/Sources/Input.c:33` | file | 0..3 sound cadence counter |
| `batteryWasEngaged` | Boolean | `GliderPRO/Sources/Input.c:34` | file | was the key held last frame |
| `bandsTotal` | short | global | game | rubber bands in inventory |

---

## 9. Rubber bands: the only edge-triggered action

`AddBand` (`GliderPRO/Sources/RubberBands.c:256-290`):

```
AddBand(thisGlider, h, v, direction):
 1. if numBands >= kMaxRubberBands (2): return false        # RubberBands.c:258-259
 2. bands[numBands].mode  = 0
 3. bands[numBands].count = 0
 4. bands[numBands].vVel  = thisGlider.tipped ? -2 : 0
 5. bands[numBands].dest  = (h-8, v-3, h+8, v+3)
 6. if direction == kFaceRight:
 7.     offset dest by +32 horizontally; hVel = +kRubberBandVelocity (20)
 8. else:
 9.     offset dest by -32 horizontally; hVel = -kRubberBandVelocity (20)
10. thisGlider.hVel -= bands[numBands].hVel / 2             # recoil: ∓10
11. numBands++
12. PlayPrioritySound(kFireBandSound, kFireBandPriority)    # 20, 301
13. return true
```

The band spawn point passed by the input layer is `dest.left + 24, dest.top + 10`
(`GliderPRO/Sources/Input.c:352-353`), i.e. the horizontal centre of the 48-pixel-wide glider
(`kGliderWide 48`, `GliderPRO/Headers/GliderDefines.h:548`; `kHalfGliderWide 24`, `:550`) and 10 pixels
down from its top.

### 9.1 The `fireHeld` latch — exact semantics

```
if bandKeyDown and bandsTotal > 0 and mode == kGliderNormal:
    if not fireHeld:
        if AddBand(...):        # may FAIL when 2 bands are already airborne
            bandsTotal--
            fireHeld = true     # latch ONLY on success
else:
    fireHeld = false            # unlatch as soon as the gate fails
```

Consequences, all of which are player-visible:

1. **One band per keypress.** Holding the key does not stream bands.
2. **A failed shot is retried.** If `numBands >= 2`, `AddBand` returns false, `fireHeld` stays
   false, and the *next* frame tries again — while the key is still held. So holding the band
   key fires a third band the instant one of the first two expires. This is the mechanism by
   which a held band key behaves like a rate-limited autofire capped at 2 in flight.
3. **The latch is cleared by the whole `else`, not just by key release.** Running out of bands
   (`bandsTotal <= 0`) or leaving `kGliderNormal` clears `fireHeld` even with the key still
   down. So: fire your last band, pick up more bands while still holding the key → you
   immediately fire again without releasing.
4. `bandsTotal--` happens only on a successful `AddBand`, so the counter can never leak.

### 9.2 `numBands`

`numBands` is reset to 0 in `NewGame` (`GliderPRO/Sources/Play.c:113`) and decremented by
`HandleBands`. `kMaxRubberBands` is 2 (`GliderPRO/Headers/GliderDefines.h:261`) and the
`bands` array is allocated as exactly `sizeof(bandType) * kMaxRubberBands`
(`GliderPRO/Sources/StructuresInit2.c:241`) — so the `numBands >= kMaxRubberBands` guard is a
buffer-overrun guard as well as a game rule.

---

## 10. Pause

### 10.1 Which key

There is no separate pause key constant; the choice is a Boolean:

```c
Boolean isEscPauseKey;        // Input.c:34
```

* `isEscPauseKey == false` → **Tab** pauses (`kTabKeyMap` 55). This is the default
  (`GliderPRO/Sources/Main.c:184`, `GliderPRO/Sources/Settings.c:343`, `GliderPRO/Sources/Settings.c:1241`).
* `isEscPauseKey == true` → **Esc** pauses (`kEscKeyMap` 50).

Persisted as `prefsInfo.wasEscPauseKey` at byte offset **218**
(`GliderPRO/Sources/Main.c:269` writes, `GliderPRO/Sources/Main.c:114` reads).

The test appears verbatim in three places and is always written the same way:

```c
if ((isEscPauseKey && BitTst(&theKeys, kEscKeyMap)) ||
        (!isEscPauseKey && BitTst(&theKeys, kTabKeyMap)))
```

`GliderPRO/Sources/Input.c:93-94`, `:100-101`, `:115-116`, `:271-272`, `:373-374`.

**The non-chosen key is not tested at all.** If Tab pauses, Esc does nothing during play;
if Esc pauses, Tab does nothing.

### 10.2 `DoPause` (`GliderPRO/Sources/Input.c:77-117`)

```
 1. SetPort((GrafPtr)mainWindow)                            # Input.c:81
 2. QSetRect(&bounds, 0, 0, 214, 54)                        # Input.c:82
 3. CenterRectInRect(&bounds, &houseRect)                   # Input.c:83
 4. if isEscPauseKey: LoadScaledGraphic(kEscPausePictID /*1015*/, &bounds)   # Input.c:85
 5. else:             LoadScaledGraphic(kTabPausePictID /*1016*/, &bounds)   # Input.c:87
 6.
 7. # (A) debounce: wait for the pause key to be RELEASED
 8. do { GetKeys(theKeys) }                                 # Input.c:89-92
 9.    while pauseKeyDown(theKeys)                           # Input.c:93-94
10.
11. paused = true                                           # Input.c:96
12. while paused:                                           # Input.c:97
13.     GetKeys(theKeys)                                    # Input.c:99
14.     if pauseKeyDown(theKeys): paused = false            # Input.c:100-102
15.     elif BitTst(theKeys, kCommandKeyMap): DoCommandKey()    # Input.c:103-104
16.
17. CopyBits(workSrcMap → mainWindow, bounds, bounds, srcCopy, nil)   # Input.c:107-109
18.
19. # (B) debounce: wait for the pause key to be RELEASED again
20. do { GetKeys(theKeys) }                                 # Input.c:111-114
21.    while pauseKeyDown(theKeys)                           # Input.c:115-116
```

Facts a port must reproduce:

* The placard rect is **exactly 214 × 54** and is centred in `houseRect` by
  `CenterRectInRect` (`GliderPRO/Sources/RectUtils.c:142-154`, integer-divide centring —
  odd leftovers bias up/left).
* **Verified against the resources:** `PICT` 1015 (`GliderPRO/Glider PRO.r:111013`, 6092
  bytes) and `PICT` 1016 (`GliderPRO/Glider PRO.r:111397`, 6106 bytes) both have
  `picFrame = (top 0, left 0, bottom 54, right 214)`, i.e. 214×54 — identical to the
  hard-coded rect, so `LoadScaledGraphic` performs no scaling in practice.
* `DoPause` is a **blocking busy loop**. It calls `GetKeys` as fast as the CPU allows, with no
  `TickCount` throttle, no `WaitNextEvent`, and no `Delay`. On a modern machine this is a
  100 %-CPU spin.
* **No events are serviced while paused.** The screen is not redrawn if obscured, the menu bar
  is dead, and the application cannot be switched out cleanly. Music, being driven by the
  Sound Manager, keeps playing.
* Command-Q and Command-S **do** work while paused (line 15) because `DoCommandKey` is called
  from the pause loop. Quitting from the pause screen sets `playing = false` **and**
  `paused = false` (`GliderPRO/Sources/Input.c:57-58`), which is what lets the pause loop exit.
* The two debounce loops (A) and (B) mean: you cannot pause and unpause on the same physical
  press, and after unpausing the key must be released before the game resumes. If you hold the
  pause key down, the game freezes at step 21 with the placard *already erased*.
* On resume, the placard area is restored from the work GWorld (`workSrcMap`), not redrawn
  from scratch — the port needs the equivalent "last composited frame" buffer.

### 10.3 Pause and two players

The pause check in `GetInput` sits outside the player-1-only block
(`GliderPRO/Sources/Input.c:373`), so it runs for `theGlider2` as well. But because
`theKeys` is a *shared global* that `DoPause` itself overwrites — and the last thing `DoPause`
does is loop until the pause key is released — by the time control returns to `GetInput` and
then to `GetInput(&theGlider2)`, `theKeys` no longer has the pause bit set. So `DoPause` is
never entered twice for one press. A port that gives each player its own snapshot would break
this and pause twice.

---

## 11. `DoCommandKey` — quit and save during play

`GliderPRO/Sources/Input.c:53-73`:

```
 1. if BitTst(theKeys, kQKeyMap):                           # Input.c:55   Command-Q
 2.     playing = false                                     # Input.c:57
 3.     paused  = false                                     # Input.c:58
 4.     if (not twoPlayerGame) and (not demoGoing):          # Input.c:59
 5.         if QuerySaveGame(): SaveGame2()                 # Input.c:61-62
 6. elif BitTst(theKeys, kSKeyMap) and (not twoPlayerGame):  # Input.c:65   Command-S
 7.     RefreshScoreboard(kSavingTitleMode)                 # Input.c:67   (mode 2)
 8.     SaveGame2()                                         # Input.c:68
 9.     HideCursor()                                        # Input.c:69
10.     CopyRectWorkToMain(&workSrcRect)                    # Input.c:70
11.     RefreshScoreboard(kNormalTitleMode)                 # Input.c:71   (mode 0)
```

* `DoCommandKey` is only reached when the Command bit is already set
  (`GliderPRO/Sources/Input.c:286`, `:103`, `:205`). Command alone does nothing.
* Command-Q **always** ends the game, even in two-player mode or a demo; only the
  save-offer is suppressed.
* Command-S is silently ignored in two-player games.
* `kSavingTitleMode 2` / `kNormalTitleMode 0`
  (`GliderPRO/Headers/GliderDefines.h:619-621`).
* No other Command-letter combination is checked during play — the classic Mac menu-key
  mechanism (`MenuKey`) is not running.
* `HideCursor()` at step 9 is needed because `QuerySaveGame`/dialogs call `InitCursor()`.

### 11.1 `QuerySaveGame` (`GliderPRO/Sources/Input.c:383-397`)

```
InitCursor()                                # Input.c:389
FlushEvents(everyEvent, 0)                  # Input.c:390  ← drops the queued Command-Q keyDown
hitWhat = Alert(kSaveGameAlert /*1041*/, nil)   # Input.c:392
return (hitWhat == kYesSaveGameButton /*1*/)    # Input.c:393-396
```

`FlushEvents` is essential: the physical Command-Q press also queued a `keyDown` event that
would otherwise be delivered to the alert.

**Verified resource layout.** `ALRT` 1041 (`GliderPRO/Glider PRO.r:5275`, 12 bytes):
bounds `(40, 40, 116, 302)` = 262 × 76, `itemsID` = `0x0411` = 1041, stages word `0xCCCC`.
`DITL` 1041 (`GliderPRO/Glider PRO.r:5087`, 138 bytes, 4 items):

| Item | Type | Enabled | Rect (T,L,B,R) | Content |
|---|---|---|---|---|
| 1 | Button | yes | (48, 174, 68, 254) | `Save First` |
| 2 | Button | yes | (48, 82, 68, 162) | `Don't Save` |
| 3 | StaticText | no | (8, 8, 40, 209) | `Do you want to save the state of the game before quitting?` |
| 4 | Icon | no | (8, 222, 40, 254) | ICON `0x0430` = 1072 |

(In the `DITL` format, bit 7 of the type byte **set** means *disabled*; the table above has
that corrected.) So `kYesSaveGameButton == 1` is the **"Save First"** button, and the
function returns true for "save".

---

## 12. Delete: the two-player suicide key

`GliderPRO/Sources/Input.c:366-371`. Four conditions must all hold:

| Condition | Meaning | Citation |
|---|---|---|
| `otherPlayerEscaped != kNoOneEscaped` | one player has already left the room and is waiting | `Input.c:366`, `kNoOneEscaped -1` at `GliderDefines.h:608` |
| `BitTst(&theKeys, kDeleteKeyMap)` | Delete/Backspace is down (offset 52, raw `0x33`) | `Input.c:367` |
| `thisGlider->which` | this is player 1 (`kPlayer1 == TRUE`) | `Input.c:368` |
| `!onePlayerLeft` | both players are still alive | `Input.c:368` |

`ForceKillGlider` (`GliderPRO/Sources/Transit.c:449-469`) then kills whichever glider is
**not** in limbo:

```
if theGlider.mode == kGliderInLimbo:                       # Transit.c:451
    if theGlider2.mode != kGliderFadingOut:
        StartGliderFadingOut(&theGlider2)                  # Transit.c:455
        PlayPrioritySound(kFadeOutSound, kFadeOutPriority) # Transit.c:456  (2)
        playerSuicide = true                               # Transit.c:457
elif theGlider2.mode == kGliderInLimbo:                    # Transit.c:460
    if theGlider.mode != kGliderFadingOut:
        StartGliderFadingOut(&theGlider)                   # Transit.c:464
        PlayPrioritySound(kFadeOutSound, kFadeOutPriority) # Transit.c:465
        playerSuicide = true                               # Transit.c:466
```

Semantics: in a two-player game the "leader" who has already flown out of the room sits in
`kGliderInLimbo` while the other player catches up. The straggler is holding up the game, so
the leader presses Delete and the *straggler* dies. The `otherPlayerEscaped` values are
`kPlayerEscapedDownStairs -7`, `kPlayerEscapedUpStairs -6`, `kPlayerEscapedDown -5`,
`kPlayerEscapedUp -4`, `kPlayerEscapedLeft -3`, `kPlayerEscapedRight -2`, `kNoOneEscaped -1`
(`GliderPRO/Headers/GliderDefines.h:602-608`), set in `Interactions.c` when a glider leaves
through an exit (e.g. `GliderPRO/Sources/Interactions.c:181`, `:293`).

Because the check requires `thisGlider->which` (player 1) but `ForceKillGlider` inspects both
gliders, **only the physical player-1 keyboard position can trigger it** — but it will kill
either player depending on who is in limbo. Delete is level-triggered and unlatched, so it
fires on the first frame it is seen and then `ForceKillGlider`'s
`mode != kGliderFadingOut` test suppresses repeats.

---

## 13. Downstream consumers of input-set glider flags

The input layer's only outputs are `hDesiredVel`, `vDesiredVel`, `hVel`, `tipped`,
`heldLeft`, `heldRight`, `fireHeld`, plus side effects (bands, sounds, mode changes). Two of
those flags are read by systems outside the physics ramp, and a port that drops them will have
subtly wrong behaviour:

### 13.1 `tipped` → sprite selection

`GliderPRO/Sources/Player.c:151-199` (`MoveGliderNormal`), a nested `if` on `facing` first and
then `sliding` / `tipped` — `sliding` wins over `tipped`:

| State | Facing left (`kFaceLeft`) → sprite | Facing right → sprite |
|---|---|---|
| `sliding` | `gliderSrc[30]` (`:157-158`) | `gliderSrc[29]` (`:179-180`) |
| `tipped` (not sliding) | `gliderSrc[3]` (`:165-166`) | `gliderSrc[1]` (`:187-188`) |
| neither | `gliderSrc[2]` (`:170-171`) | `gliderSrc[0]` (`:192-193`) |

Both `src` and `mask` are set to the same `gliderSrc[]` rect. Note that `sliding` is
**consumed**: the sliding branch sets `sliding = false` immediately (`:159`, `:181`), so it is
a one-frame latch, unlike `tipped` which is recomputed every frame by `GetInput`. The
facing-left sprites (2/3/30) are the odd-looking ones because index 0 is the right-facing
neutral glider; `FlagGliderNormal` picks the same pairing (`gliderSrc[2]` when
`facing == kFaceLeft`, else `gliderSrc[0]`, `GliderPRO/Sources/Modes.c:341-350`).

So `tipped` is what makes the glider visibly bank when you steer against its nose.

### 13.2 `heldLeft` / `heldRight` → stair and duct transit suppression

`GliderPRO/Sources/Interactions.c`:

```c
case kMoveItUp:                       // Interactions.c:1251
    ... requires (!thisGlider->heldRight) ...
case kMoveItDown:                     // Interactions.c:1286
    ... requires (!thisGlider->heldLeft) ...
```

Holding the Right key prevents the glider from being taken *up* (stairs / floor transport);
holding the Left key prevents it being taken *down*. This is the "hold the direction key to
fly past the stairs instead of taking them" mechanic. Note the counter-intuitive pairing
(right↔up, left↔down) — it is correct as written and follows from the physical layout of the
up/down stair graphics (`kUpStairs`/`kDownStairs` are 160 × 267,
`GliderPRO/Sources/StructuresInit2.c:385-386`).

`heldLeft` is also set (with no thrust) by the both-keys-held about-face branch
(`GliderPRO/Sources/Input.c:309`), which means pressing both direction keys blocks downward
transit.

### 13.3 `facing`/`tipped` in interaction tests

`GliderPRO/Sources/Interactions.c:1435-1436` and `:1478-1479` also test the facing/tipped
combination; and `Dynamics.c:45-46` and `Interactions.c:692-693` exclude the two about-face
modes (`kGliderFaceLeft 7`, `kGliderFaceRight 8`) from certain interactions.

---

## 14. Demo (attract mode) playback: recorded input

Glider PRO's attract mode is not a scripted animation — it is a **recorded input stream**
replayed through the same physics. This makes the demo format part of the input subsystem.

### 14.1 `demoType`

```c
typedef struct
{
    long        frame;
    char        key;
    char        padding;
} demoType, *demoPtr;
```

`GliderPRO/Headers/GliderStructs.h:334-339`. Verified layout under `pack(2)` with 4-byte
`long`:

```
sizeof(demoType) = 6
offsetof(frame)  = 0   (4 bytes, big-endian on disk)
offsetof(key)    = 4   (1 byte)
offsetof(padding)= 5   (1 byte, never written)
```

### 14.2 Key codes in the stream

| `key` | Meaning per `GetDemoInput` | Meaning per `LogDemoKey` call site |
|---|---|---|
| 0 | `hDesiredVel += kNormalThrust`, `heldRight = true` → **right** | logged from the **right**-key branch (`Input.c:301`→`:304`) |
| 1 | `hDesiredVel -= kNormalThrust`, `heldLeft = true` → **left** | logged from the **left**-key branch (`Input.c:318`→`:321`) |
| 2 | battery or helium | logged from the batt-key branch (`Input.c:330`→`:334`) |
| 3 | rubber band | logged from the band-key branch (`Input.c:344`→`:348`) |

**The comments in `GetDemoInput` are wrong.** `GliderPRO/Sources/Input.c:228` says
`case 0: // left key` but the body applies *rightward* thrust; `:235` says
`case 1: // right key` but applies *leftward* thrust. Since `LogDemoKey(0)` is emitted from
the right-key branch and `LogDemoKey(1)` from the left-key branch, the recorded data and the
playback code agree — **only the comments are misleading**. Do not "fix" the comments into
code.

### 14.3 `GetDemoInput` (`GliderPRO/Sources/Input.c:186-277`)

```
GetDemoInput(thisGlider):

 1. if thisGlider.which == kPlayer1:                        # Input.c:188
 2.     GetKeys(theKeys)                                    # Input.c:190
 3. #if BUILD_ARCADE_VERSION                                # Input.c:192
 4.     if BitTst(theKeys, thisGlider.leftKey)              # Input.c:194
 5.        or BitTst(theKeys, thisGlider.rightKey)          # Input.c:195
 6.        or BitTst(theKeys, thisGlider.battKey)           # Input.c:196
 7.        or BitTst(theKeys, thisGlider.bandKey):          # Input.c:197
 8.         playing = false                                 # Input.c:199
 9.         paused  = false                                 # Input.c:200
10. #else                                                   # Input.c:203
11.     if BitTst(theKeys, kCommandKeyMap): DoCommandKey()  # Input.c:205-206
12. #endif
13.
14. if thisGlider.mode == kGliderBurning:                   # Input.c:211
15.     hDesiredVel ∓= kNormalThrust by facing               # Input.c:213-216
16. else:
17.     thisGlider.heldLeft  = false                        # Input.c:220
18.     thisGlider.heldRight = false                        # Input.c:221
19.     thisGlider.tipped    = false                        # Input.c:222   ← always cleared
20.
21.     if gameFrame == (long)demoData[demoIndex].frame:     # Input.c:224
22.         switch demoData[demoIndex].key:                  # Input.c:226
23.             case 0:                                     # Input.c:228  (RIGHT)
24.                 hDesiredVel += kNormalThrust            # Input.c:229
25.                 tipped   = (facing == kFaceLeft)        # Input.c:230
26.                 heldRight = true                        # Input.c:231
27.                 fireHeld = false                        # Input.c:232
28.             case 1:                                     # Input.c:235  (LEFT)
29.                 hDesiredVel -= kNormalThrust            # Input.c:236
30.                 tipped   = (facing == kFaceRight)       # Input.c:237
31.                 heldLeft = true                         # Input.c:238
32.                 fireHeld = false                        # Input.c:239
33.             case 2:                                     # Input.c:242  (BATT/HELIUM)
34.                 if batteryTotal > 0: DoBatteryEngaged() # Input.c:243-244
35.                 else:                DoHeliumEngaged()  # Input.c:246
36.                 fireHeld = false                        # Input.c:247
37.             case 3:                                     # Input.c:250  (BANDS)
38.                 if not fireHeld:                        # Input.c:251
39.                     if AddBand(thisGlider,
40.                                dest.left+24, dest.top+10, facing):   # Input.c:253-254
41.                         bandsTotal--                    # Input.c:256
42.                         if bandsTotal <= 0: QuickBandsRefresh(false) # Input.c:257-258
43.                         fireHeld = true                 # Input.c:260
44.         demoIndex++                                     # Input.c:266
45.     else:
46.         fireHeld = false                                # Input.c:269
47.
48.     if pauseKeyDown(theKeys): DoPause()                 # Input.c:271-275
```

Differences from `GetInput` that a port must NOT unify away:

| # | Difference | Citation |
|---|---|---|
| 1 | Exactly **one** action per frame. The stream stores one `key` byte per record and only one record can match a given `gameFrame`, so a demo can never hold left *and* fire simultaneously. | `Input.c:224-267` |
| 2 | `tipped` is unconditionally cleared *before* the switch, so unlike `GetInput` there is no both-keys carry-over case. | `Input.c:222` |
| 3 | There is no both-keys/about-face case at all — `ToggleGliderFacing` is unreachable in demo playback. | `Input.c:226-264` |
| 4 | The battery case does **not** check `batteryTotal != 0` nor `mode == kGliderNormal`. With `batteryTotal == 0` it calls `DoHeliumEngaged`, which increments to 1 and plays a hiss. | `Input.c:242-248` vs `Input.c:330-331` |
| 5 | The band case does **not** check `bandsTotal > 0` nor `mode == kGliderNormal`, so `bandsTotal` can go negative. | `Input.c:250-263` vs `Input.c:344-345` |
| 6 | `fireHeld` is cleared by *every* non-band record and by every frame with no record — so band records on consecutive frames still fire once each. | `Input.c:232`, `:239`, `:247`, `:269` |
| 7 | Delete-suicide is not checked. | absent |
| 8 | Command-Q/S work only in the non-arcade build. | `Input.c:203-208` |

### 14.4 `demoIndex` monotonicity and the end of the stream

`demoIndex` is a plain `short` (`GliderPRO/Sources/Input.c:33`), reset to 0 in `NewGame`
(`GliderPRO/Sources/Play.c:114`). There is **no bounds check** against the number of records.
When the stream is exhausted, `demoData[demoIndex].frame` reads past the allocated block
(`kDemoLength` bytes) and the comparison `gameFrame == thatGarbage` will almost always fail, so
playback simply stops producing input. This is a latent out-of-bounds read in the original.

The demo is terminated in practice by other means:

* In the arcade build, any of the four game keys aborts it (`Input.c:194-201`).
* `GameOver.c` / `Banner.c` end the game when the glider runs out of lives.
* The last record's frame is 3414 (§14.6), i.e. 3414 / 30 ≈ 114 seconds of demo.

A Go port should add an explicit `demoIndex < len(demoData)` guard; it changes no observable
behaviour before the end of the stream.

### 14.5 Loading the stream

```c
#ifdef CREATEDEMODATA
    demoData = (demoPtr)NewPtr(sizeof(demoType) * 2000);
#else
    demoData = (demoPtr)NewPtr(kDemoLength);
    if (demoData == nil) RedAlert(kErrNoMemory);
    tempHandle = GetResource('demo', 128);
    if (tempHandle == nil) RedAlert(kErrNoMemory);
    else { BlockMove(*tempHandle, demoData, kDemoLength); ReleaseResource(tempHandle); }
#endif
```

`GliderPRO/Sources/StructuresInit2.c:280-298`. `kDemoLength` is **6702**
(`GliderPRO/Headers/GliderDefines.h:625`). The `BlockMove` is a raw copy of the resource into
the pointer — **no byte swapping**, so the on-disk `long frame` is big-endian and was read
natively by the 68k/PPC.

### 14.6 Verified `'demo'` resource contents

Parsed `data 'demo' (128)` from `GliderPRO/Glider PRO.r:199389`. Observed:

```
bytes: 6702 = 1117.0 recs of 6
monotonic strictly increasing: True
min 46 max 3414 n 1117
key histogram: {0: 910, 1: 198, 3: 9}
distinct padding values: 109
first 8 recs: [(46,0,114),(47,0,114),(48,0,114),(49,0,69),(56,0,6),(57,0,114),(58,0,111),(59,0,114)]
last 4 recs:  [(3411,0,254),(3412,0,248),(3413,0,7),(3414,0,1)]
first24hex: 00 00 00 2e 00 72 00 00 00 2f 00 72 00 00 00 30 00 72 00 00 00 31 00 45
```

* **6702 bytes = exactly 1117 × 6.** `kDemoLength 6702` is therefore the exact resource size,
  not a rounded buffer, confirming stride 6.
* **Stride/endianness proven by elimination.** Only 6-byte stride with big-endian `frame`
  yields a strictly increasing frame sequence:
  * stride 6, big-endian → `46, 47, 48, 49, 56, 57, 58, 59, 60, 74, … 3414` — monotonic ✔
  * stride 8, big-endian → `46, 3080306, 7471104, 56, 3735666, …` — **not** monotonic ✘
  * stride 6, little-endian → `771751936, 788529152, 805306368, …` — **not** monotonic ✘
* **No frame number is repeated**, consistent with one action per frame.
* Key distribution: 910 × right (0), 198 × left (1), **0 × battery/helium (2)**, 9 × bands (3).
  The recorded demo never uses the battery key, which is why the missing guards in §14.3
  item 4 are never exercised by the shipped data.
* The `padding` byte holds **109 distinct values** including many ASCII letters
  (`0x72` = `'r'`, `0x6F` = `'o'`, `0x45` = `'E'` in the first records). This is uninitialised
  heap garbage: `LogDemoKey` writes only `frame` and `key`
  (`GliderPRO/Sources/Input.c:46-47`). A port must **ignore** the padding byte, and when
  writing a compatible resource may write anything (writing zero is the sane choice, but it
  will not be byte-identical to the shipped resource).
* Frame gaps: 1009 gaps of exactly 1 frame, then 8 × 7, 6 × 4, 6 × 8, 6 × 3, 6 × 10, 4 × 14,
  4 × 13, … — i.e. the demo is mostly long continuous holds of the right key, punctuated by
  pauses.
* Playback length: last frame 3414 at 30 fps ≈ 113.8 s.

### 14.7 How the demo is started and which house it uses

```
DoDemoGame():                                     # Play.c:282-306
    wasHouseIndex  = thisHouseIndex               # Play.c:287
    CloseHouse()                                  # Play.c:288
    thisHouseIndex = demoHouseIndex               # Play.c:289
    if OpenHouse():                               # Play.c:291
        ReadHouse()                               # Play.c:293
        demoGoing = true                          # Play.c:294
        NewGame(kNewGameMode)                     # Play.c:295
    # then the previous house is restored         # Play.c:297-301
```

`demoHouseIndex` is found by name during the house scan: it is `-1` unless a file called
`"Demo House"` exists (`GliderPRO/Sources/SelectHouse.c:636-644`). The demo input stream is
therefore only meaningful against that specific house — replaying it against any other house
produces garbage.

Two entry points call `DoDemoGame`:

1. Idle timeout in the splash screen (`GliderPRO/Sources/Events.c:536-540`).
2. The **Options ▸ Demo…** menu item, which is `iHelp` = 5
   (`GliderPRO/Headers/Externs.h`), handled in `DoOptionsMenu` as `case iHelp: DoDemoGame();`
   (`GliderPRO/Sources/Menu.c:427-428`). The `MENU` 130 resource's fifth item is literally
   `Demo…` with Command-key `D` (§25).

After the demo, `NewGame`'s tail runs `WaitCommandQReleased()`
(`GliderPRO/Sources/Play.c:275`), `demoGoing = false` (`:276`), and re-arms the idle timer
`incrementModeTime = TickCount() + kIdleSplashTicks` (`:277`).

### 14.8 Demo recording (compiled out, but documented)

`LogDemoKey` (`GliderPRO/Sources/Input.c:44-49`):

```c
demoData[demoIndex].frame = gameFrame;
demoData[demoIndex].key   = keyIs;
demoIndex++;
```

Called only under `#ifdef CREATEDEMODATA` at `Input.c:304` (key 0), `:321` (1), `:334` (2),
`:348` (3). Because the calls sit at the *top* of each branch, a key is logged even in the
both-keys-held case (`LogDemoKey(0)` runs before the both-keys test at `Input.c:306`), which
means a recording of a both-keys about-face replays as a plain right-key press. Another
reason demo playback is not bit-identical to the original session.

The buffer for recording is `sizeof(demoType) * 2000` = 12000 bytes
(`GliderPRO/Sources/StructuresInit2.c:282`) with no overflow check.

The dump path is `GliderPRO/Sources/Play.c:217`:

```c
#ifdef CREATEDEMODATA
    DumpToResEditFile((Ptr)demoData, sizeof(demoType) * (long)demoIndex);
#endif
```

and `DumpToResEditFile` (`GliderPRO/Sources/DebugUtilities.c:316-354`) builds a Pascal
filename `"Terrain " + hour + "-" + minute`, `Create(filesName, 0, 'RSED', 'rsrc')`,
`CreateResFile`, `OpenResFile`, `PtrToHand`, then:

```c
AddResource(newResource, 'demo', 128, "\p");     // DebugUtilities.c:352
ChangedResource(newResource);
```

That is exactly how the shipped `'demo'` 128 resource was produced, and it explains both
`kDemoLength 6702` (the byte count of one particular recording, hard-coded afterwards) and the
garbage padding bytes.

---

## 15. `BUILD_ARCADE_VERSION` input differences

`#define BUILD_ARCADE_VERSION 1` (`GliderPRO/Headers/GliderDefines.h:16`) is **on**. It
changes input behaviour in exactly two places:

### 15.1 Demo abort (`GliderPRO/Sources/Input.c:192-208`)

| Build | Behaviour during a demo |
|---|---|
| Arcade (`1`) | Any of player 1's four game keys (default ← → ↓ ↑) sets `playing = false; paused = false` — i.e. **any control key exits the demo**. Command-Q/S do nothing. |
| Non-arcade | Only Command-Q/Command-S are checked; the game keys do nothing and the demo runs to completion. |

Because this test uses `thisGlider->leftKey` etc. rather than the arrow-key constants, it
follows the player's *remapped* keys.

### 15.2 Arrow keys in the splash/editor event loop (`GliderPRO/Sources/Events.c:191-251`)

The arcade cases are **ungated**; every non-arcade case is wrapped in `if (houseUnlocked)` and
then branches on `objActive == kNoObjectSelected`.

| Key | Arcade build (`Events.c:193-207`) — no gate | Non-arcade build (`Events.c:211-249`) — all gated on `houseUnlocked` |
|---|---|---|
| ← | `DoOptionsMenu(iHighScores)` → show high scores | `SelectNeighborRoom(kRoomToLeft)` or `MoveObject(kBumpLeft, shiftDown)` |
| → | `DoOptionsMenu(iHelp)` → **start the demo** | `SelectNeighborRoom(kRoomToRight)` or `MoveObject(kBumpRight, shiftDown)` |
| ↑ | `DoGameMenu(iNewGame)` → start a new game | `SelectNeighborRoom(kRoomAbove)` or `MoveObject(kBumpUp, shiftDown)` |
| ↓ | `DoGameMenu(iNewGame)` → start a new game | `SelectNeighborRoom(kRoomBelow)` or `MoveObject(kBumpDown, shiftDown)` |

Menu item numbers: `iHighScores 3`, `iHelp 5`, `iNewGame 1`
(`GliderPRO/Headers/Externs.h:197-222`). `iHelp` maps to the **Demo…** menu item, not to a
help screen (`GliderPRO/Sources/Menu.c:427-428`, `case iHelp: DoDemoGame();`).

**Consequence: in the shipped build the room editor cannot be navigated with the arrow keys
and objects cannot be nudged with them.** The arrow keys are attract-mode shortcuts for a
kiosk/arcade cabinet. A Go port should almost certainly make this a runtime option rather
than a compile-time one, but must default to the arcade behaviour to be faithful.

Note that ↑ and ↓ both call `DoGameMenu(iNewGame)` — identical handlers, not a typo'd
`iTwoPlayer`.

---

## 16. The Event Manager path: `HandleEvent`

`GliderPRO/Sources/Events.c:479-541`. This is the loop that runs in splash and edit mode.

```
HandleEvent():

 1. KeyMap eventKeys; EventRecord theEvent; long sleep = 2;      # Events.c:481-483
 2.
 3. # ---- pre-poll for chorded modifiers, BEFORE any event is fetched ----
 4. GetKeys(eventKeys)                                            # Events.c:486
 5. if BitTst(eventKeys, kCommandKeyMap) and BitTst(eventKeys, kOptionKeyMap):   # 487-488
 6.     HiliteAllObjects()                                        # Events.c:490
 7. elif BitTst(eventKeys, kOptionKeyMap)
 8.        and theMode == kEditMode and houseUnlocked:            # Events.c:492-493
 9.     EraseSelectedTool()                                       # Events.c:495
10.     SelectTool(kSelectTool)      # kSelectTool == 0            # Events.c:496
11.
12. if thisMac.hasWNE:
13.     itHappened = WaitNextEvent(everyEvent, &theEvent, sleep /*2 ticks*/, nil)  # 500
14. else:
15.     itHappened = GetNextEvent(everyEvent, &theEvent)           # Events.c:504
16.
17. if itHappened:                                                # Events.c:507
18.     switch theEvent.what:                                     # Events.c:509
19.         mouseDown        → HandleMouseEvent(&theEvent)          # Events.c:511-512
20.         keyDown, autoKey → HandleKeyEvent(&theEvent)            # Events.c:515-517
21.         updateEvt        → HandleUpdateEvent(&theEvent)         # Events.c:520-521
22.         osEvt            → HandleOSEvent(&theEvent)             # Events.c:524-525
23.         kHighLevelEvent  → HandleHighLevelEvent(&theEvent)      # Events.c:528-529
24. else:
25.     HandleIdleTask()                                           # Events.c:533-534
26.
27. if theMode == kSplashMode and doAutoDemo and not switchedOut:  # Events.c:536
28.     if TickCount() >= incrementModeTime: DoDemoGame()           # Events.c:538-539
```

Facts:

* **`sleep = 2` ticks.** The application yields for at most 1/30 s, matching the play frame
  rate.
* **`keyDown` and `autoKey` are handled identically** (`Events.c:515-517`), so *every* editor
  keyboard command auto-repeats at the OS key-repeat rate. That includes destructive ones:
  holding Delete repeatedly deletes rooms/objects. This is a genuine behaviour, not a bug to
  fix.
* **The Command+Option pre-poll happens on every pass of the loop**, before the event fetch,
  so `HiliteAllObjects()` (a debug/authoring aid that XOR-frames all 24 object rects) fires
  continuously while the chord is held. `HiliteAllObjects`
  (`GliderPRO/Sources/ObjectEdit.c:2734-2765`) itself contains a nested spin
  `do { GetKeys(theseKeys); } while (Cmd && Option);` — so once entered it blocks until the
  chord is released, then erases its own XOR frames.
* **Holding Option alone in edit mode force-selects the arrow/select tool** (`kSelectTool 0`,
  `GliderPRO/Headers/GliderDefines.h:270`). This is the "temporarily get the arrow back"
  gesture; it is implemented by re-selecting on every loop pass rather than by remembering the
  previous tool, so releasing Option leaves the select tool active.
* `thisMac.hasWNE` chooses `WaitNextEvent` (System 6.0.4+/MultiFinder) versus a bare
  `GetNextEvent`; the commented-out `SystemTask()` at `Events.c:502` is the pre-MultiFinder
  requirement. A Go port needs neither.

### 16.1 `HandleIdleTask` (`GliderPRO/Sources/Events.c:459-473`)

```
if theMode == kEditMode:
    SetPort(mainWindow)
    DoMarquee()                                    # animate the marching-ants selection
    if autoRoomEdit and newRoomNow:
        DoRoomInfo()
        newRoomNow = false
```

Runs only when no event was available. `DoMarquee` advances the marquee pattern phase, so the
marching-ants animation rate depends on how idle the app is.

### 16.2 `HandleOSEvent` (`GliderPRO/Sources/Events.c:390-442`)

```
if (theEvent->message & 0x01000000):          # suspend/resume event class
    if (theEvent->message & 0x00000001):      # RESUMING
        (re-check colour depth; if wrong → BitchAboutColorDepth(); button 1 → quit)
        switchedOut = false
        InitCursor()
        (restart music)
        incrementModeTime = TickCount() + kIdleSplashTicks
    else:                                     # SUSPENDING
        switchedOut = true
        InitCursor()
        StopTheMusic()
```

`kIdleSplashTicks` is `7200L` = 120 seconds (`GliderPRO/Headers/GliderDefines.h:197`, comment
"2 minutes"). `kColorSwitchedAlert` is 1042 (`GliderPRO/Sources/Events.c:46-55`).

The same suspend/resume decoding appears in `HandlePlayEvent`
(`GliderPRO/Sources/Play.c:412-421`) and in `WaitForInputEvent`
(`GliderPRO/Sources/Utilities.c:461-472`). In a Go port these correspond to window
focus-lost/focus-gained.

### 16.3 `HandlePlayEvent` — the only event handling during play

`GliderPRO/Sources/Play.c:387-426`, called **only** when `doBackground` is true:

```
long sleep = 2;
if (WaitNextEvent(everyEvent, &theEvent, sleep, nil))
    switch (theEvent.what):
        updateEvt: if window == mainWindow:
                       BeginUpdate; CopyBits(justRoomsRect); RefreshScoreboard; EndUpdate
        osEvt:     resume  → switchedOut = false; ToggleMusicWhilePlaying(); HideCursor()
                   suspend → InitCursor(); switchedOut = true; ToggleMusicWhilePlaying()
```

`mouseDown`, `keyDown`, `autoKey`, and `kHighLevelEvent` are **not** cases here. Therefore:

* **There is no mouse input during play, ever.**
* **There is no menu access during play.**
* Keyboard events that pile up in the queue during a game are simply left there; `FlushEvents`
  calls at game-over (`GliderPRO/Sources/GameOver.c:152`, `:454`) and in
  `QuerySaveGame`/`WaitForInputEvent` discard them.

### 16.4 `HandleUpdateEvent` / `HandleHighLevelEvent`

`HandleUpdateEvent` (`GliderPRO/Sources/Events.c:341-385`) dispatches on
`(WindowPtr)theEvent->message` to per-window update routines for `mainWindow`, `mapWindow`,
`toolsWindow`, `linkWindow`, `coordWindow`, `menuWindow`.
`HandleHighLevelEvent` (`GliderPRO/Sources/Events.c:447-454`) calls
`AEProcessAppleEvent(theEvent)` — this is how "Open Document" / "Quit" Apple events arrive
(relevant to a port only as "handle a file passed on the command line").

---

## 17. Editor keyboard commands (complete)

All from `HandleKeyEvent` (`GliderPRO/Sources/Events.c:162-336`). The dispatch is on
`theChar = theEvent->message & charCodeMask` — i.e. on the **generated character**, not the
physical key, so these commands follow the user's keyboard layout.

Command-key handling comes first and short-circuits everything else:

```c
if ((commandDown) && (!optionDown))
    DoMenuChoice(MenuKey(theChar));
```

`GliderPRO/Sources/Events.c:172-173`. Note `!optionDown`: Command+Option chords are
deliberately *not* routed to `MenuKey`, leaving them free for the `HiliteAllObjects` gesture
(§16).

| Key | ASCII | Guard | Action | Citation |
|---|---|---|---|---|
| Help | `0x05` | none | nothing (explicit empty case) | `Events.c:178-179` |
| Page Up | `0x0B` | `houseUnlocked` | `PrevToolMode()` | `Events.c:181-184` |
| Page Down | `0x0C` | `houseUnlocked` | `NextToolMode()` | `Events.c:186-189` |
| ← → ↑ ↓ | `0x1C`–`0x1F` | see §15.2 | arcade: menu shortcuts; else: room nav / object nudge | `Events.c:191-251` |
| Delete | `0x08` | `houseUnlocked` **only** | `objActive == kNoObjectSelected` ? `DeleteRoom(true)` : `DeleteObject()` | `Events.c:253-261` |
| Tab | `0x09` | `theMode == kEditMode && houseUnlocked` | `shiftDown ? SelectPrevObject() : SelectNextObject()` | `Events.c:263-271` |
| Esc | `0x1B` | `theMode == kEditMode && houseUnlocked` | `DeselectObject()` | `Events.c:273-276` |
| `a`/`A` | `0x61`/`0x41` | edit + unlocked | `SetSpecificToolMode(kApplianceMode)` (7) | `Events.c:278-282` |
| `b`/`B` | `0x62`/`0x42` | edit + unlocked | `SetSpecificToolMode(kBlowerMode)` (1) | `Events.c:284-288` |
| `c`/`C` | `0x63`/`0x43` | edit + unlocked | `SetSpecificToolMode(kClutterMode)` (9) | `Events.c:290-294` |
| `e`/`E` | `0x65`/`0x45` | edit + unlocked | `SetSpecificToolMode(kEnemyMode)` (8) | `Events.c:296-300` |
| `f`/`F` | `0x66`/`0x46` | edit + unlocked | `SetSpecificToolMode(kFurnitureMode)` (2) | `Events.c:302-306` |
| `l`/`L` | `0x6C`/`0x4C` | edit + unlocked | `SetSpecificToolMode(kLightMode)` (6) | `Events.c:308-312` |
| `p`/`P` | `0x70`/`0x50` | edit + unlocked | `SetSpecificToolMode(kBonusMode)` (3) | `Events.c:314-318` |
| `s`/`S` | `0x73`/`0x53` | edit + unlocked | `SetSpecificToolMode(kSwitchMode)` (5) | `Events.c:320-324` |
| `t`/`T` | `0x74`/`0x54` | edit + unlocked | `SetSpecificToolMode(kTransportMode)` (4) | `Events.c:326-330` |
| anything else | — | — | ignored | `Events.c:332-333` |

Tool-mode constants: `kSelectTool 0`, `kBlowerMode 1`, `kFurnitureMode 2`, `kBonusMode 3`,
`kTransportMode 4`, `kSwitchMode 5`, `kLightMode 6`, `kApplianceMode 7`, `kEnemyMode 8`,
`kClutterMode 9` (`GliderPRO/Headers/GliderDefines.h:270-280`).

There is deliberately **no** `d`, `g`–`k`, `m`–`o`, `q`, `r`, `u`–`z` shortcut, and no
shortcut for the "select" tool itself (Option-hold serves that purpose, §16).

Two quirks worth flagging because they are easy to "fix" by accident:

1. **Page Up/Page Down and Delete are gated on `houseUnlocked` but NOT on
   `theMode == kEditMode`.** So in splash mode with an unlocked house, Delete deletes the
   current room and Page Up/Down cycles tool groups. Every letter shortcut, Tab and Esc *do*
   check `theMode == kEditMode`.
2. **Delete's `DeleteRoom(true)` auto-repeats** because `autoKey` shares the handler.

### 17.1 `MoveObject` nudge increments (reachable from the keyboard only in the non-arcade build)

`GliderPRO/Sources/ObjectEdit.c:1374-…`. `MoveObject` itself is not `#if`-gated; only its
arrow-key callers in `HandleKeyEvent` are (§15.2). It early-returns unless
`theMode == kEditMode` (`:1382-1383`), then:

```
if shiftDown:                                  increment = 10   # :1387-1388
elif objActive == kInitialGliderSelected:      increment = 1    # :1391-1393
elif whichWay == kBumpRight or kBumpLeft:                       # :1397
    switch thisRoom->objects[objActive].what:                   # :1399
        44 small-object cases, kTaper…kMirror:  increment = 2    # :1401-1446
        case kManhole:                         increment = 64   # :1448-1450
        default:                               increment = 1    # :1452-1454
else:                                          increment = 1    # :1458
```

So the horizontal grid is 2 px for small objects and 1 px for everything else, and **only the
horizontal axis is quantised** — `kBumpUp`/`kBumpDown` always take the `increment = 1` path.

Direction constants: `kBumpUp 1`, `kBumpDown 2`, `kBumpRight 3`, `kBumpLeft 4`
(`GliderPRO/Headers/GliderDefines.h:205-208`). Room-neighbour constants: `kRoomAbove 1`,
`kRoomBelow 2`, `kRoomToRight 3`, `kRoomToLeft 4` (`:200-203`). Selection sentinels:
`kNoObjectSelected -1`, `kInitialGliderSelected -2`, `kLeftGliderSelected -3`,
`kRightGliderSelected -4` (`:527-530`).

### 17.2 Editor modifier-key gestures polled with `GetKeys` (not events)

Five places poll the keyboard directly from inside editor mouse/creation code. These are
*modifier gestures*, invisible to the event queue:

| Gesture | Effect | Citation |
|---|---|---|
| **Shift** held when clicking in the main window | Do **not** revert to the select tool after placing an object (sticky tool) | `GliderPRO/Sources/MainWindow.c:354-356` |
| **Shift** held when placing a `kFlower` | Reuse the previous random flower variant instead of re-rolling `wasFlower = RandomInt(kNumFlowers)` | `GliderPRO/Sources/ObjectAdd.c:740-742` |
| **Shift** held when a new room is created | Suppress the automatic Room Info dialog: `newRoomNow = false` instead of `= autoRoomEdit` | `GliderPRO/Sources/Room.c:232-236` |
| **Command+Option** held anywhere | XOR-frame all 24 object rects until released | `GliderPRO/Sources/Events.c:486-490` → `ObjectEdit.c:2734-2765` |
| **Option** held in edit mode | Force the select tool | `GliderPRO/Sources/Events.c:492-496` |

`MainWindow.c:338-360` in full, because it is the join between mouse and keyboard:

```c
if ((theMode != kEditMode) || (mainWindow == nil) || (!houseUnlocked))
    return;
SetPortWindowPort(mainWindow);
GlobalToLocal(&wherePt);
if (toolSelected == kSelectTool) DoSelectionClick(wherePt, isDoubleClick);
else                             DoNewObjectClick(wherePt);
GetKeys(theseKeys);                                          // MainWindow.c:354
if (!BitTst(&theseKeys, kShiftKeyMap))
    { EraseSelectedTool(); SelectTool(kSelectTool); }
```

---

## 18. Mouse handling

### 18.1 There is no mouse input during play

Established three independent ways:

1. `HandlePlayEvent` has no `mouseDown` case (`GliderPRO/Sources/Play.c:387-426`).
2. The cursor is hidden for the duration of a game: `HideCursor()` at
   `GliderPRO/Sources/Play.c:104`, again after Command-S at `GliderPRO/Sources/Input.c:69`,
   and again on resume at `GliderPRO/Sources/Play.c:414`.
3. With the default `doBackground = false` (`GliderPRO/Sources/Main.c:186`),
   `HandlePlayEvent` is never called at all, so no events of any kind are dequeued during a
   game.

Every `Button()`, `GetMouse()`, `StillDown()`, `WaitMouseUp()` call site in the code base is in
the editor, a dialog filter, the About box, a colour-fade loop, or `DebugUtilities.c`.

### 18.2 Window-part dispatch: `HandleMouseEvent`

`GliderPRO/Sources/Events.c:60-157`. `thePart = FindWindow(theEvent->where, &whichWindow)`
(`:67`), then:

| `FindWindow` part | Handling | Citation |
|---|---|---|
| `inSysWindow` | nothing (`SystemClick` is commented out — Carbon) | `Events.c:71-73` |
| `inMenuBar` | `DoMenuChoice(MenuSelect(theEvent->where))` | `Events.c:75-78` |
| `inDrag` | `DragWindow(whichWindow, where, &thisMac.screen)`; then save the window's new left/top into the matching `isEditH/V`, `isMapH/V`, `isToolsH/V`, `isLinkH/V`, `isCoordH/V` globals; `mainWindow` also does `SendBehind(mainWindow, 0L)`; finally `HiliteAllWindows()` | `Events.c:80-96` |
| `inGoAway` | `TrackGoAway` → `ToggleMapWindow()` / `ToggleToolsWindow()` / `CloseLinkWindow()` / `ToggleCoordinateWindow()` | `Events.c:98-110` |
| `inGrow` | `mapWindow` only: `GrowWindow(mapWindow, where, &thisMac.gray)` → `ResizeMapWindow(LoWord(newSize), HiWord(newSize))` | `Events.c:112-118` |
| `inZoomIn` / `inZoomOut` | `TrackBox` → `ZoomWindow(whichWindow, thePart, true)` | `Events.c:120-124` |
| `inContent` | see §18.3 | `Events.c:126-152` |

The window-position globals are what get persisted to prefs (`wasEditH/V`, `wasMapH/V`,
`wasToolsH/V`, `wasLinkH/V`, `wasCoordH/V`, §24).

### 18.3 `inContent`: double-click synthesis

Only the main window gets double-click detection, and it is hand-rolled:

```
hDelta = |theEvent->where.h - lastWhere.h|                 # Events.c:129-131
vDelta = |theEvent->where.v - lastWhere.v|                 # Events.c:132-134
if ((theEvent->when - lastUp) < doubleTime) and hDelta < 5 and vDelta < 5:   # Events.c:135-136
    isDoubleClick = true                                   # Events.c:137
else:
    isDoubleClick = false                                  # Events.c:140
    lastUp    = theEvent->when                             # Events.c:141
    lastWhere = theEvent->where                            # Events.c:142
HandleMainClick(theEvent->where, isDoubleClick)            # Events.c:144
```

then `mapWindow → HandleMapClick(theEvent)`, `toolsWindow → HandleToolsClick(where)`,
`linkWindow → HandleLinkClick(where)` (`Events.c:146-151`).

Constants and state:

| Name | Type | Declared | Meaning |
|---|---|---|---|
| `lastUp` | long | `GliderPRO/Sources/Events.c:27` | timestamp (ticks) of the previous non-double click |
| `lastWhere` | Point | `GliderPRO/Sources/Events.c:29` | global position of that click |
| `doubleTime` | UInt32 | `GliderPRO/Sources/Events.c:28` | `GetDblTime()` — the user's double-click interval in ticks |
| slop | — | `Events.c:135-136` | **5 pixels** in each axis, exclusive (`< 5`) |

`doubleTime = GetDblTime();` is read exactly once, at startup
(`GliderPRO/Sources/InterfaceInit.c:184`). A Go port must expose an equivalent
configurable interval; the classic Mac default was 32 ticks (≈533 ms) but it is user-settable,
so hard-coding it is a (small) behaviour change.

Note the asymmetry: `lastUp`/`lastWhere` are updated **only on a non-double click**. So three
rapid clicks in the same place register as click, double-click, double-click — the second and
third both see the same `lastUp`. This is a real (if benign) deviation from the Toolbox's own
double-click convention.

`IgnoreThisClick()` (`GliderPRO/Sources/Events.c:567-572`) exists to defeat the detector:

```c
lastUp     -= doubleTime;
lastWhere.h = -100;
lastWhere.v = -100;
```

It is called after placing a new object (`GliderPRO/Sources/ObjectEdit.c:773` region) so that
placing two objects quickly is not misread as a double-click on the second.

### 18.4 Main-window content clicks: `DoSelectionClick`

`GliderPRO/Sources/ObjectEdit.c:73-138`:

```
 1. if the click is on a marquee resize handle:
 2.     if StillDown(): DragHandle(where)                  # ObjectEdit.c:82
 3. else:
 4.     objActive = FindObjectSelected(where)              # ObjectEdit.c:94
 5.     if nothing selected and isDoubleClick: DoRoomInfo()
 6.     elif object selected and isDoubleClick: DoObjectInfo()
 7.     elif object selected: if StillDown(): DragObject(where)     # ObjectEdit.c:115
```

`FindObjectSelected` (`GliderPRO/Sources/ObjectEdit.c:46-68`) tests the two glider start
rects first, then scans objects **in reverse** (`for (i = kMaxRoomObs - 1; i >= 0; i--)`,
`kMaxRoomObs 24`) so the topmost/most-recently-added object wins.

`DoNewObjectClick` (`GliderPRO/Sources/ObjectEdit.c:773`) computes the object type from the
palette state:

```c
whatObject = toolSelected + ((toolMode - 1) * 0x0010);
if (AddNewObject(where, whatObject, true))
    IgnoreThisClick();
```

i.e. **object type = tool index + 16 × (toolGroup − 1)** — a 4-bit group, 4-bit index packing.

### 18.5 Drag loops: `StillDown` / `WaitMouseUp` / `GetMouse`

All four marquee drag routines share one idiom (`GliderPRO/Sources/Marquee.c`):

```c
while (WaitMouseUp())
{
    GetMouse(&newPt);
    if (DeltaPoint(wasPt, newPt))
    {
        ... erase old XOR outline, draw new one ...
        wasPt = newPt;
    }
}
```

| Routine | Cursor set | Loop | Purpose |
|---|---|---|---|
| `DragOutMarqueeRect` (`Marquee.c:188-214`) | `InitCursor()` (arrow) | `Marquee.c:200` | rubber-band a new selection rect |
| `DragMarqueeRect` | `SetCursor(&handCursor)` at `Marquee.c:223` | `Marquee.c:231` | move the selection; `lockH`/`lockV` constrain to one axis |
| `DragMarqueeHandle` | `SetCursor(&vertCursor)` `:265` / `&horiCursor` `:267` | `Marquee.c:275` | resize on one axis |
| `DragMarqueeCorner` | `SetCursor(&diagCursor)` at `Marquee.c:350` | `Marquee.c:358` | resize on both axes |

Constants: `kMarqueePatListID 128` (`GliderPRO/Sources/Marquee.c:15`),
`kHandleSideLong 9` (`:16`), and the pattern list is read with
`GetIndPattern(&theMarquee.pats[i], kMarqueePatListID, i + 1)` (`Marquee.c:504`).

Cursor resources: `kHandCursorID 128`, `kVertCursorID 129`, `kHoriCursorID 130`,
`kDiagCursorID 131` (`GliderPRO/Sources/InterfaceInit.c:16-19`), loaded in
`GetExtraCursors` (`GliderPRO/Sources/InterfaceInit.c:80-111`) along with the system
`iBeamCursor`. The animated wait cursor is `acur` 128 (`GliderPRO/Sources/AnimCursor.c:14`),
installed by `LoadCursors()` (`:140`) and stepped by `IncrementCursor` (`:181`) /
`SpinCursor`.

The XOR drawing uses `PenMode(patXor)` with a grey `PenPat` — a classic QuickDraw
rubber-band technique that a Go port replaces with plain overlay drawing.

### 18.6 Map window: `HandleMapClick`

`GliderPRO/Sources/Map.c:570-712`:

```
SetPortWindowPort(mapWindow)
globalWhere = wherePt; GlobalToLocal(&wherePt)
wherePt.h -= 1; wherePt.v -= 1                     # 1-pixel border compensation
whichPart = FindControl(wherePt, mapWindow, &theControl)
if whichPart == 0:                                  # not on a scroll bar
    localH = wherePt.h / kMapRoomWidth   (32)
    localV = wherePt.v / kMapRoomHeight  (20)
    if localH >= mapRoomsWide or localV >= mapRoomsHigh: return
    roomH = localH + mapLeftRoom
    roomV = kMapGroundValue - (localV + mapTopRoom)
    if RoomExists(roomH, roomV):
        CopyRoomToThisRoom(...); DeselectObject(); ReflectCurrentRoom(false); ...
    else:
        (if doBitchDialogs) QueryNewRoom(); then CreateNewRoom(roomH, roomV)
else:                                               # on a scroll bar
    GetControlReference(theControl) == kHScrollRef → TrackControl with LiveHScrollAction
                                     kVScrollRef → TrackControl with LiveVScrollAction
    for kControlIndicatorPart: TrackControl(..., nil) then
        mapLeftRoom / mapTopRoom = GetControlValue(theControl); RedrawMapContents()
```

`kMapRoomHeight 20` (`GliderPRO/Headers/GliderDefines.h:246`) and `kMapRoomWidth 32` (`:247`).
Note the deliberate `-1` on both
axes and that the vertical room number is **inverted** through `kMapGroundValue`.

### 18.7 Tools palette: `HandleToolsClick`

`GliderPRO/Sources/Tools.c:440-489`:

```
SetPortWindowPort(toolsWindow); GlobalToLocal(&wherePt);
part = FindControl(wherePt, toolsWindow, &theControl);
if theControl != nil and part != 0:
    part = TrackControl(theControl, wherePt, (ControlActionUPP)-1L);   # live-tracking popup
    if part != 0:
        newMode = GetControlValue(theControl)
        if newMode != toolMode: EraseSelectedTool(); SwitchToolModes(newMode)
else:
    for i in 0 .. kTotalTools-1:
        if PtInRect(wherePt, &toolRects[i]) and i <= lastTool:
            EraseSelectedTool()
            toolIcon = i
            if toolMode == kBlowerMode and toolIcon >= 7: toolIcon++
            if toolMode == kTransportMode and toolIcon >= 7:
                if toolIcon >= 11: toolIcon += 4
                else:              toolIcon = ((toolIcon - 7) * 2) + 7
            SelectTool(toolIcon)
            break
```

The `(ControlActionUPP)-1L` magic value means "use the control's own default action proc" —
the classic Toolbox convention for live-tracking a control.

The two `toolIcon` fix-ups compensate for gaps in the icon strips for the blower and
transport groups; they are pure palette-index arithmetic and must be replicated exactly or the
wrong object type gets placed.

### 18.8 Link window: `HandleLinkClick`

`GliderPRO/Sources/Link.c:366-395`:

```
if linkWindow == nil: return                       # Link.c:372-373
SetPortWindowPort(linkWindow); GlobalToLocal(&wherePt)
part = FindControl(wherePt, linkWindow, &theControl)
if theControl != nil and part != 0:                # Link.c:379
    part = TrackControl(theControl, wherePt, nil)   # Link.c:381
    if part != 0:                                   # Link.c:382
        if theControl == linkControl:   DoLink()
        elif theControl == unlinkControl: DoUnlink()
        if thisRoomNumber == linkRoom: CopyThisRoomToRoom()   # Link.c:389-390
        GenerateRetroLinks()                                  # Link.c:391
```

Note that `CopyThisRoomToRoom` / `GenerateRetroLinks` run **inside** the `part != 0` branch —
a click that misses both buttons, or a press-and-drag-off release, does nothing at all.

### 18.9 Room Info dialog: mouse-over and tile dragging

`GliderPRO/Sources/RoomInfo.c`:

* `DragMiniTile(Point mouseIs, short *newTileOver)` (`RoomInfo.c:119`) computes
  `tileOver = (mouseIs.h - tileSrc.left) / kMiniTileWide`, then runs the standard
  `while (WaitMouseUp()) { GetMouse(&mouseIs); if (DeltaPoint(...)) … }` XOR drag, updating
  `*newTileOver` whenever the mouse is inside `tileDest`.
* `RoomFilter` (`RoomInfo.c:299-371`) has a `default:` case at `:365-368`:
  ```c
  default:
      GetMouse(&mouseIs);
      HiliteTileOver(mouseIs);
      return(false);
  ```
  **Mouse-over highlighting is driven off null events.** Because `ModalDialog` delivers
  `nullEvent` continuously, this polls the mouse position every idle pass. A Go port that only
  reacts to real mouse-move events gets the same visual result; a port that never delivers
  idle events gets no highlight at all.

### 18.10 About box: manual button tracking

`AboutFilter` (`GliderPRO/Sources/About.c:168-256`) tracks the OK button by hand instead of
using `TrackControl`:

```
pre-switch: if (Button() && clickedDownInOkay) {
                GetMouse(&mousePt);
                PtInRgn(mousePt, okayButtRgn) ? HiLiteOkayButton() : UnHiLiteOkayButton();
            }
keyDown Return/Enter → HiLiteOkayButton(); Delay(8,&dummy); UnHiLiteOkayButton(); *hit = kOkayButton
mouseDown  → if in region: clickedDownInOkay = true; return false
mouseUp    → if in region and clickedDownInOkay: *hit = kOkayButton; handledIt = true
```

This is the only place in the code base that implements press-drag-release button feedback
manually.

---

## 19. Control remapping: the Controls Prefs dialog

Reached via **Options ▸ Preferences… ▸ Controls** (or by typing `c`/`C` in the main prefs
dialog, `GliderPRO/Sources/Settings.c:1329-1333`).

### 19.1 Dialog and item constants

| Constant | Value | Meaning | Citation |
|---|---|---|---|
| `kMainPrefsDialID` | 1012 | main prefs DLOG | `GliderPRO/Sources/Settings.c:17` |
| `kDisplayPrefsDialID` | 1017 | | `Settings.c:18` |
| `kSoundPrefsDialID` | 1018 | | `Settings.c:19` |
| `kControlPrefsDialID` | **1023** | Controls DLOG/DITL | `Settings.c:20` |
| `kBrainsPrefsDialID` | 1024 | | `Settings.c:21` |
| `kDisplayButton` | 3 | main-prefs item | `Settings.c:22` |
| `kSoundButton` | 4 | main-prefs item | `Settings.c:23` |
| `kControlsButton` | 5 | main-prefs item | `Settings.c:24` |
| `kBrainsButton` | 6 | main-prefs item | `Settings.c:25` |
| `kAllDefaultsButton` | 11 | main-prefs "All Defaults" | `Settings.c:1397` |
| `kRightControl` | **5** | Controls DITL item: right-key icon | `Settings.c:42` |
| `kLeftControl` | **6** | left-key icon | `Settings.c:43` |
| `kBattControl` | **7** | battery-key icon | `Settings.c:44` |
| `kBandControl` | **8** | band-key icon | `Settings.c:45` |
| `kControlDefaults` | **13** | "Defaults" button | `Settings.c:46` |
| `kESCPausesRadio` | **14** | radio | `Settings.c:47` |
| `kTABPausesRadio` | **15** | radio | `Settings.c:48` |
| `kOkayButton` | 1 | | `GliderPRO/Headers/Externs.h:22` |
| `kCancelButton` | 2 | | `GliderPRO/Headers/Externs.h:23` |

The four key-name text fields are `item + 4`, i.e. items **9, 10, 11, 12**
(`GliderPRO/Sources/Settings.c:352-355`).

### 19.2 Verified DITL 1023 ("Controls Prefs")

Parsed from `data 'DITL' (1023, "Controls Prefs")` at `GliderPRO/Glider PRO.r:4724`
(272 bytes, item count field = 14 → **15 items**). In the DITL format **bit 7 of the type byte
SET means the item is DISABLED**, so `0x80` = disabled userItem, `0xC0` = disabled picItem,
`0x04` = enabled Button, `0x06` = enabled radio button, `0x20` = enabled icon item.

| Item | Type | Enabled | Rect (T,L,B,R) | Content |
|---|---|---|---|---|
| 1 | Button | yes | 148, 250, 168, 308 | "Okay" |
| 2 | Button | yes | 148, 184, 168, 242 | "Cancel" |
| 3 | userItem | no | 138, 8, 139, 308 | 1-px divider line |
| 4 | Picture | no | 0, 0, 32, 316 | PICT `0x03F4` = **1012** (banner) |
| 5 | Icon | **yes** | 43, **80**, 75, 112 | ICON `0x0410` = **1040** → `kRightControl` |
| 6 | Icon | **yes** | 43, **20**, 75, 52 | ICON `0x0411` = **1041** → `kLeftControl` |
| 7 | Icon | **yes** | 43, **140**, 75, 172 | ICON `0x0412` = **1042** → `kBattControl` |
| 8 | Icon | **yes** | 43, **200**, 75, 232 | ICON `0x0413` = **1043** → `kBandControl` |
| 9 | userItem | no | 80, 68, 92, 124 | right-key name text |
| 10 | userItem | no | 80, 8, 92, 64 | left-key name text |
| 11 | userItem | no | 80, 128, 92, 184 | batt-key name text |
| 12 | userItem | no | 80, 188, 92, 244 | band-key name text |
| 13 | Button | yes | 148, 8, 168, 72 | "Defaults" |
| 14 | RadioButton | yes | 99, 8, 115, 140 | "Esc Pauses Game" |
| 15 | RadioButton | yes | 115, 8, 131, 140 | "Tab Pauses Game" |

**The visual left-to-right order is Left (l=20), Right (l=80), Batt (l=140), Band (l=200), but
the item numbers run Right, Left, Batt, Band.** `whichCtrl` is `itemHit - kRightControl`
(`GliderPRO/Sources/Settings.c:572`), so:

| `whichCtrl` | Action | DITL item | Screen position |
|---|---|---|---|
| 0 | right | 5 | 2nd from left |
| 1 | left | 6 | leftmost |
| 2 | battery | 7 | 3rd |
| 3 | bands | 8 | rightmost |

Verified DLOG 1023 (`GliderPRO/Glider PRO.r:6233`, 21 bytes): bounds (40, 40, 216, 356) =
**316 × 176**, `procID` 1 (`dBoxProc` — modal, no title bar), **`visible` = 0**, `goAway` = 0,
`refCon` 0, `itemsID` `0x03FF` = 1023, empty title. Because `visible` is 0 the code must call
`ShowWindow` explicitly (`GliderPRO/Sources/Settings.c:535`) — that is deliberate, so the
dialog is fully drawn before it appears.

`ICON` 1040, 1041 and 1043 are **byte-identical** 128-byte 1-bit 32×32 glider silhouettes;
`ICON` 1042 additionally carries battery and keyboard art. Colour `cicn` variants exist at the
same IDs 1040-1043.

### 19.3 `DoControlPrefs` (`GliderPRO/Sources/Settings.c:502-595`)

```
 1. controlFilterUPP = NewModalFilterUPP(ControlFilter)          # Settings.c:509
 2. prefDlg = GetNewDialog(kControlPrefsDialID /*1023*/, nil, kPutInFront)   # :512
 3. if prefDlg == nil: RedAlert(kErrDialogDidntLoad /*3*/)       # Settings.c:513-514
 4. SetPort((GrafPtr)prefDlg)                                    # Settings.c:515
 5. for i in 0..3:                                               # Settings.c:516
 6.     GetDialogItemRect(prefDlg, i + kRightControl, &controlRects[i])   # :518
 7.     InsetRect(&controlRects[i], -3, -3)                       # Settings.c:519
 8. whichCtrl = 1                     # left is pre-selected      # Settings.c:521
 9. tempLeftStr  = leftName ; tempRightStr = rightName            # Settings.c:523-524
10. tempBattStr  = batteryName ; tempBandStr = bandName           # Settings.c:525-526
11. tempLeftMap  = theGlider.leftKey                              # Settings.c:527
12. tempRightMap = theGlider.rightKey                             # Settings.c:528
13. tempBattMap  = theGlider.battKey                              # Settings.c:529
14. tempBandMap  = theGlider.bandKey                              # Settings.c:530
15. wasEscPauseKey = isEscPauseKey                                # Settings.c:531
16. leaving = false                                               # Settings.c:533
17. ShowWindow(GetDialogWindow(prefDlg))                          # Settings.c:535
18. select the Esc/Tab radio from isEscPauseKey                    # Settings.c:536-541
19. while not leaving:                                            # Settings.c:543
20.     ModalDialog(controlFilterUPP, &itemHit)                    # Settings.c:545
21.     switch itemHit:
22.         kOkayButton (1):                                       # Settings.c:548
23.             leftName = tempLeftStr … bandName = tempBandStr    # Settings.c:549-552
24.             theGlider.leftKey  = tempLeftMap                   # Settings.c:553
25.             theGlider.rightKey = tempRightMap                  # Settings.c:554
26.             theGlider.battKey  = tempBattMap                   # Settings.c:555
27.             theGlider.bandKey  = tempBandMap                   # Settings.c:556
28.             isEscPauseKey      = wasEscPauseKey                 # Settings.c:557
29.             leaving = true                                     # Settings.c:558
30.         kCancelButton (2):  leaving = true   # discard all      # Settings.c:561-562
31.         kRightControl..kBandControl (5..8):                    # Settings.c:565-568
32.             PenSize(2,2); ForeColor(white); FrameRect(controlRects[whichCtrl])   # :569-571
33.             whichCtrl = itemHit - kRightControl                # Settings.c:572
34.             ForeColor(red); FrameRect(controlRects[whichCtrl]) # Settings.c:573-574
35.             ForeColor(black); PenNormal()                       # Settings.c:575-576
36.             UpdateControlKeyName(prefDlg)                       # Settings.c:577
37.         kESCPausesRadio | kTABPausesRadio (14,15):              # Settings.c:580-581
38.             SelectFromRadioGroup(prefDlg, itemHit,
39.                                  kESCPausesRadio, kTABPausesRadio)   # Settings.c:582
40.             wasEscPauseKey = !wasEscPauseKey                    # Settings.c:583
41.         kControlDefaults (13):                                  # Settings.c:586
42.             SetControlsToDefaults(prefDlg)                      # Settings.c:587
43.             UpdateControlKeyName(prefDlg)                       # Settings.c:588
44. DisposeDialog(prefDlg)                                        # Settings.c:593
45. DisposeModalFilterUPP(controlFilterUPP)                       # Settings.c:594
```

Two behaviours to note:

* **`wasEscPauseKey = !wasEscPauseKey` is a toggle, not an assignment** (`Settings.c:583`).
  Clicking the *already-selected* radio flips the flag while the radio graphics stay put,
  desynchronising them. Clicking either radio twice restores agreement. This is a genuine bug
  in the original; a faithful port reproduces it, a sensible port sets
  `wasEscPauseKey = (itemHit == kESCPausesRadio)`.
* Nothing is written to `theGlider` until Okay, and **nothing is written to `theGlider2`
  ever** — player 2's keys are not editable (§1.2).

### 19.4 `ControlFilter` (`GliderPRO/Sources/Settings.c:380-498`)

The remap capture is entirely inside the modal filter. Four near-identical `case whichCtrl`
blocks; `case 0` shown, the others differ only in which three maps they compare against and
which `temp*Str`/`temp*Map` they write.

```
case keyDown:                                             # Settings.c:386
  switch whichCtrl:                                       # Settings.c:387
    case 0:                                               # Settings.c:389
      wasKeyMap = (long)GetKeyMapFromMessage(event->message)     # Settings.c:390
      if wasKeyMap in { tempLeftMap, tempBattMap, tempBandMap,
                        kTabKeyMap (55), kEscKeyMap (50),
                        kDeleteKeyMap (52) }:                    # Settings.c:391-393
          if wasKeyMap == kEscKeyMap:                            # Settings.c:395
              FlashDialogButton(dial, kCancelButton)             # Settings.c:397
              *item = kCancelButton ; return true                # Settings.c:398-399
          else:
              SysBeep(1)                                         # Settings.c:402
      else:
          GetKeyName(event->message, tempRightStr)                # Settings.c:406
          tempRightMap = wasKeyMap                                # Settings.c:407
      break
    case 1: … writes tempLeftStr / tempLeftMap                     # Settings.c:411-431
    case 2: … writes tempBattStr / tempBattMap                     # Settings.c:433-453
    case 3: … writes tempBandStr / tempBandMap                     # Settings.c:455-475
  UpdateControlKeyName(dial)                                       # Settings.c:477
  return false                                                     # Settings.c:478

case mouseDown: return false                                       # Settings.c:481-482
case updateEvt: SetPort(dial); BeginUpdate; UpdateSettingsControl(dial);
                EndUpdate; event->what = nullEvent; return false    # Settings.c:485-491
default:        return false                                       # Settings.c:494-495
```

**Reserved keys that can never be bound** — the same set for all four slots:

| Key | Offset | Behaviour when pressed | Citation |
|---|---|---|---|
| Tab | `kTabKeyMap` 55 | `SysBeep(1)`, rejected | `Settings.c:392`, `:402` |
| Esc | `kEscKeyMap` 50 | **cancels the dialog** (no beep) | `Settings.c:395-399` |
| Delete | `kDeleteKeyMap` 52 | `SysBeep(1)`, rejected | `Settings.c:393`, `:402` |
| any key already bound to one of the other three actions | — | `SysBeep(1)`, rejected | `Settings.c:391-393` |

Tab and Esc are reserved because they are the two candidate pause keys; Delete is reserved
because it is the two-player suicide key (§12).

Notice what is **not** filtered: the Command, Option, Control and Shift keys are perfectly
legal player-1 bindings, and so are letters, digits, keypad keys and function keys. Binding
Command breaks Command-Q quitting (`GliderPRO/Sources/Input.c:286`) because the same bit is
then also the thrust key; binding Control/Option/Command/Shift collides with player 2's
hard-coded map (§1.2). The original never guards against either.

Also: **`ControlFilter` never checks `event->modifiers`.** The map offset comes only from the
virtual key code, so Shift-A and A produce the same binding, and the *displayed name* comes
from the character code, so binding Shift-A displays `"A key"` while binding A displays
`"a key"` — the same physical key with two different labels.

### 19.5 `SetControlsToDefaults` (`GliderPRO/Sources/Settings.c:333-346`)

```c
PasStringCopy("\plf arrow", tempLeftStr);
PasStringCopy("\prt arrow", tempRightStr);
PasStringCopy("\pdn arrow", tempBattStr);
PasStringCopy("\pup arrow", tempBandStr);
tempLeftMap  = kLeftArrowKeyMap;    /* 124 */          // Settings.c:339
tempRightMap = kRightArrowKeyMap;   /* 123 */          // Settings.c:340
tempBattMap  = kDownArrowKeyMap;    /* 122 */          // Settings.c:341
tempBandMap  = kUpArrowKeyMap;      /* 121 */          // Settings.c:342
wasEscPauseKey = false;                                 // Settings.c:343
SelectFromRadioGroup(theDialog, kTABPausesRadio,
                     kESCPausesRadio, kTABPausesRadio); // Settings.c:344-345
```

The exact same four assignments plus `isEscPauseKey = false` appear in two other places, and
all three must agree:

| Site | Purpose | Citation |
|---|---|---|
| `SetControlsToDefaults` | the Controls-panel "Defaults" button | `GliderPRO/Sources/Settings.c:335-343` |
| `SetAllDefaults` | the main-panel "All Defaults" button | `GliderPRO/Sources/Settings.c:1233-1241` |
| `ReadInPrefs` else-branch | first launch / prefs deleted | `GliderPRO/Sources/Main.c:129-138`, `:184` |

### 19.6 Drawing the panel

`UpdateControlKeyName` (`GliderPRO/Sources/Settings.c:350-356`):

```c
DrawDialogUserText(theDialog, kRightControl + 4, tempRightStr, whichCtrl == 0);  // item 9
DrawDialogUserText(theDialog, kLeftControl  + 4, tempLeftStr,  whichCtrl == 1);  // item 10
DrawDialogUserText(theDialog, kBattControl  + 4, tempBattStr,  whichCtrl == 2);  // item 11
DrawDialogUserText(theDialog, kBandControl  + 4, tempBandStr,  whichCtrl == 3);  // item 12
```

The 4th argument is an "inverted/highlighted" flag, so the selected slot's name is drawn
inverted.

`UpdateSettingsControl` (`GliderPRO/Sources/Settings.c:360-376`): `DrawDialog`, then
`PenSize(2,2)`, white-frame all four `controlRects`, red-frame `controlRects[whichCtrl]`,
`PenNormal()`, `UpdateControlKeyName`, and finally
`FrameDialogItemC(theDialog, 3, kRedOrangeColor8)` where `kRedOrangeColor8` is **23**
(`GliderPRO/Headers/GliderDefines.h:542`) — an index into the 8-bit system palette, not an RGB
value.

---

## 20. `GetKeyName`: the display-name table

`GliderPRO/Sources/Utilities.c:536-691`. Produces the Pascal string shown in the Controls
panel. Inputs: `theASCII = message & charCodeMask`, `theVirtual = (message & keyCodeMask) >> 8`.

### 20.1 Printable range

```c
if ((theASCII >= kExclamationASCII /*0x21*/) && (theASCII <= kZKeyASCII /*0x7A*/))
{
    if ((theVirtual >= 0x0041) && (theVirtual <= 0x005C))   // numeric keypad
        { PasStringCopy("\p( )", theName); theName[2] = (char)theASCII; }
    else
        { PasStringCopy("\p  key", theName); theName[1] = (char)theASCII; }
}
```

`Utilities.c:543-556`. So:

* Any character in `0x21`–`0x7A` produced by a **keypad** key (virtual code 0x41–0x5C) is
  displayed as `( x )` → literally `(`, the character, `)`. Note `theName[2]` is the *second*
  data byte of the Pascal string (`theName[0]` is the length), so `"( )"` becomes `"(x)"`.
* Any other printable character is displayed as `"x key"` — `PasStringCopy("\p  key", …)` then
  `theName[1] = theASCII` overwrites the first space, giving e.g. `"a key"` (length still 5,
  so the second space remains: `"a key"`).

Characters **outside** `0x21`–`0x7A` fall through to the switch, which means `{`, `|`, `}`,
`~` (0x7B–0x7E) and every non-ASCII MacRoman character land in `default:` and display
`"????"`.

### 20.2 Complete special-key name table

| ASCII | Constant | Displayed name | Citation |
|---|---|---|---|
| 0x01 | `kHomeKeyASCII` | `home` | `Utilities.c:561-562` |
| 0x03 | `kEnterKeyASCII` | `enter` | `Utilities.c:565-566` |
| 0x04 | `kEndKeyASCII` | `end` | `Utilities.c:569-570` |
| 0x05 | `kHelpKeyASCII` | `help` | `Utilities.c:573-574` |
| 0x08 | `kDeleteKeyASCII` | `delete` | `Utilities.c:577-578` |
| 0x09 | `kTabKeyASCII` | `tab` | `Utilities.c:581-582` |
| 0x0B | `kPageUpKeyASCII` | `pg up` | `Utilities.c:585-586` |
| 0x0C | `kPageDownKeyASCII` | `pg dn` | `Utilities.c:589-590` |
| 0x0D | `kReturnKeyASCII` | `return` | `Utilities.c:593-594` |
| 0x10 | `kFunctionKeyASCII` | see §20.3 | `Utilities.c:597-649` |
| 0x1A | `kClearKeyASCII` | `clear` | `Utilities.c:651-652` |
| 0x1B | `kEscapeKeyASCII` | `clear` if `theVirtual == 0x0047`, else `esc` | `Utilities.c:655-659` |
| 0x1C | `kLeftArrowKeyASCII` | `lf arrow` | `Utilities.c:662-663` |
| 0x1D | `kRightArrowKeyASCII` | `rt arrow` | `Utilities.c:666-667` |
| 0x1E | `kUpArrowKeyASCII` | `up arrow` | `Utilities.c:670-671` |
| 0x1F | `kDownArrowKeyASCII` | `dn arrow` | `Utilities.c:674-675` |
| 0x20 | `kSpaceBarASCII` | `space` | `Utilities.c:678-679` |
| 0x7F | `kForwardDeleteASCII` | `frwd del` | `Utilities.c:682-683` |
| anything else | — | `????` | `Utilities.c:686-687` |

The Esc special case exists because on Mac keyboards the numeric-keypad **Clear** key
(virtual 0x47 = `kClearRawKey`, `GliderPRO/Headers/Externs.h:166`) also generates ASCII 0x1B.

### 20.3 Function-key names (inner switch on `theVirtual`)

| virtual | name | virtual | name | virtual | name |
|---|---|---|---|---|---|
| 0x60 | `F5` | 0x65 | `F9` | 0x6F | `F12` |
| 0x61 | `F6` | 0x67 | `F11` | 0x71 | `F15` |
| 0x62 | `F7` | 0x69 | `F13` | 0x76 | `F4` |
| 0x63 | `F3` | 0x6B | `F14` | 0x78 | `F2` |
| 0x64 | `F8` | 0x6D | `F10` | 0x7A | `F1` |

`Utilities.c:600-644`. `default: NumToString(theVirtual, theName)` (`Utilities.c:645-646`) —
an unrecognised function key displays its decimal virtual code. These are exactly the
`k*RawKey` constants at `GliderPRO/Headers/Externs.h:165-181` (note that `Externs.h` also
defines `kF13RawKey 0x69` etc. but `GetKeyName` hard-codes the literals rather than using
them).

**Missing: 0x66, 0x68, 0x6A, 0x6C, 0x6E, 0x70, 0x72-0x75, 0x77, 0x79.** There is no name for
F16-F20 or for the ISO/JIS keys, so those show a bare number.

### 20.4 Names persisted, not recomputed

The four display names are stored in prefs as `Str15` (`wasLeftName`, `wasRightName`,
`wasBattName`, `wasBandName`; `GliderPRO/Headers/Externs.h:236-237`) alongside the four
offsets. **The name and the offset can therefore disagree** if a prefs file is hand-edited —
the game trusts the offset for play and the name for display. A Go port that recomputes names
from offsets is *more* correct but not identical (in particular, an offset alone cannot
distinguish keypad `5` from main-keyboard `5` for naming purposes without a reverse table).

---

## 21. Prefs storage

### 21.1 File identity

| Constant | Value | Citation |
|---|---|---|
| `kPrefCreatorType` | `'ozm5'` (0x6F7A6D35) | `GliderPRO/Sources/Prefs.c:18` |
| `kPrefFileType` | `'gliP'` (0x676C6950) | `GliderPRO/Sources/Prefs.c:19` |
| `kPrefFileName` | `"\pGlider Prefs"` | `GliderPRO/Sources/Prefs.c:20` |
| `kDefaultPrefFName` | `"\pPreferences"` | `GliderPRO/Sources/Prefs.c:21` |
| `kPrefsStringsID` | 160 (`STR#`) | `GliderPRO/Sources/Prefs.c:22` |
| `kNewPrefsAlertID` | 160 (`ALRT`) | `GliderPRO/Sources/Prefs.c:23` |
| `kPrefsFNameIndex` | 1 | `GliderPRO/Sources/Prefs.c:24` |
| `kPrefsVersion` | **0x0034** (= 52) | `GliderPRO/Sources/Main.c:16` |

`STR#` 160 was verified to contain exactly one string, `"Preferences"`
(`GliderPRO/Glider PRO.r:1`).

House files use the same creator with type `'gliH'`
(`GliderPRO/Sources/SelectHouse.c:592-593`).

### 21.2 Verified `prefsInfo` layout — 226 bytes

Declared at `GliderPRO/Headers/Externs.h:233-267`, wrapped in
`#pragma options align=mac68k` (`:231`) … `#pragma options align=reset` (`:269`). That pragma
means **maximum alignment 2 bytes**, and classic-Mac `long` is **4 bytes**. Compiled with
`#pragma pack(2)` and `int` substituted for `long`, `sizeof(prefsInfo)` is **226** and the
offsets are:

| Offset | Size | Type | Field | Input-relevant? |
|---:|---:|---|---|---|
| 0 | 33 | `Str32` | `wasDefaultName` | |
| 33 | 16 | `Str15` | `wasLeftName` | **yes** |
| 49 | 16 | `Str15` | `wasRightName` | **yes** |
| 65 | 16 | `Str15` | `wasBattName` | **yes** |
| 81 | 16 | `Str15` | `wasBandName` | **yes** |
| 97 | 16 | `Str15` | `wasHighName` | |
| 113 | 32 | `Str31` | `wasHighBanner` | |
| **145** | **1** | — | **implicit pad byte** (to align the `long`s) | |
| 146 | 4 | `long` | `wasLeftMap` | **yes** |
| 150 | 4 | `long` | `wasRightMap` | **yes** |
| 154 | 4 | `long` | `wasBattMap` | **yes** |
| 158 | 4 | `long` | `wasBandMap` | **yes** |
| 162 | 2 | `short` | `wasVolume` | |
| 164 | 2 | `short` | `prefVersion` | |
| 166 | 2 | `short` | `wasMaxFiles` | |
| 168 | 2 | `short` | `wasEditH` | |
| 170 | 2 | `short` | `wasEditV` | |
| 172 | 2 | `short` | `wasMapH` | |
| 174 | 2 | `short` | `wasMapV` | |
| 176 | 2 | `short` | `wasMapWide` | |
| 178 | 2 | `short` | `wasMapHigh` | |
| 180 | 2 | `short` | `wasToolsH` | |
| 182 | 2 | `short` | `wasToolsV` | |
| 184 | 2 | `short` | `wasLinkH` | |
| 186 | 2 | `short` | `wasLinkV` | |
| 188 | 2 | `short` | `wasCoordH` | |
| 190 | 2 | `short` | `wasCoordV` | |
| 192 | 2 | `short` | `isMapLeft` | |
| 194 | 2 | `short` | `isMapTop` | |
| 196 | 2 | `short` | `wasNumNeighbors` | |
| 198 | 2 | `short` | `wasDepthPref` | |
| 200 | 2 | `short` | `wasToolGroup` | |
| 202 | 2 | `short` | `smWarnings` | |
| 204 | 2 | `short` | `wasFloor` | |
| 206 | 2 | `short` | `wasSuite` | |
| 208 | 1 | `Boolean` | `wasZooms` | |
| 209 | 1 | `Boolean` | `wasMusicOn` | |
| 210 | 1 | `Boolean` | `wasAutoEdit` | |
| 211 | 1 | `Boolean` | `wasDoColorFade` | |
| 212 | 1 | `Boolean` | `wasMapOpen` | |
| 213 | 1 | `Boolean` | `wasToolsOpen` | |
| 214 | 1 | `Boolean` | `wasCoordOpen` | |
| 215 | 1 | `Boolean` | `wasQuickTrans` | |
| 216 | 1 | `Boolean` | `wasIdleMusic` | |
| 217 | 1 | `Boolean` | `wasGameMusic` | |
| **218** | **1** | `Boolean` | **`wasEscPauseKey`** | **yes** |
| 219 | 1 | `Boolean` | `wasDoAutoDemo` | |
| 220 | 1 | `Boolean` | `wasScreen2` | |
| 221 | 1 | `Boolean` | `wasDoBackground` | |
| 222 | 1 | `Boolean` | `wasHouseChecks` | |
| 223 | 1 | `Boolean` | `wasPrettyMap` | |
| 224 | 1 | `Boolean` | `wasBitchDialogs` | |
| **225** | **1** | — | **implicit trailing pad byte** | |

Total **226**. Both pad bytes are never initialised, so a byte-for-byte-identical prefs file
cannot be reproduced; anything reading the file must skip them.

`Str15` is a 16-byte Pascal string (1 length byte + 15 data), `Str31` is 32, `Str32` is 33.

Note the commented-out `// long encrypted, fakeLong;` at `GliderPRO/Headers/Externs.h:240`,
right where the pad byte now sits — evidence that the pad is an accident of removing two
fields, not a deliberate reserve.

**The four key maps are stored as 4-byte big-endian `long`s even though every legal value is
0-127.** A Go port reading a real prefs file must read `binary.BigEndian.Uint32`.

### 21.3 Save/load path

```
ReadInPrefs()                        # Main.c:52
  if LoadPrefs(&thePrefs, kPrefsVersion /*0x0034*/):     # Main.c:56
      theGlider.leftKey  = thePrefs.wasLeftMap           # Main.c:69
      theGlider.rightKey = thePrefs.wasRightMap          # Main.c:70
      theGlider.battKey  = thePrefs.wasBattMap           # Main.c:71
      theGlider.bandKey  = thePrefs.wasBandMap           # Main.c:72
      ...
      isEscPauseKey = thePrefs.wasEscPauseKey            # Main.c:114
  else:                                                  # Main.c:122-189
      hard-coded defaults, including
      theGlider.leftKey  = kLeftArrowKeyMap  (124)       # Main.c:135
      theGlider.rightKey = kRightArrowKeyMap (123)       # Main.c:136
      theGlider.battKey  = kDownArrowKeyMap  (122)       # Main.c:137
      theGlider.bandKey  = kUpArrowKeyMap    (121)       # Main.c:138
      isEscPauseKey      = false                          # Main.c:184
      doBackground       = false                          # Main.c:186
```

```
WriteOutPrefs()                      # Main.c:208
  thePrefs.wasLeftMap  = theGlider.leftKey               # Main.c:225
  thePrefs.wasRightMap = theGlider.rightKey              # Main.c:226
  thePrefs.wasBattMap  = theGlider.battKey               # Main.c:227
  thePrefs.wasBandMap  = theGlider.bandKey               # Main.c:228
  ...
  thePrefs.wasEscPauseKey = isEscPauseKey                # Main.c:269
  if (!SavePrefs(&thePrefs, kPrefsVersion)) SysBeep(1)   # Main.c:275
```

`ReadInPrefs` is called from `main()` at `GliderPRO/Sources/Main.c:302`;
`WriteOutPrefs()` at `GliderPRO/Sources/Main.c:381`, immediately before
`RestoreColorDepth()` and `FlushEvents(everyEvent, 0)` (`:382-383`). **Prefs are written only
on a clean quit** — a crash loses remapped keys.

### 21.4 `Prefs.c` mechanics

| Function | Lines | Behaviour |
|---|---|---|
| `CanUseFindFolder` | `GliderPRO/Sources/Prefs.c:39-55` | **dead code, never called.** Uses `Gestalt(gestaltFindFolderAttr, …)` then `BitTst(&theFeature, 31 - gestaltFindFolderPresent)` at `Prefs.c:51` — a third independent confirmation that `BitTst` is MSB-first. |
| `GetPrefsFPath` | `Prefs.c:59-69` | `FindFolder(kOnSystemDisk, kPreferencesFolderType, kCreateFolder, &systemVolRef, &prefDirID)` |
| `CreatePrefsFolder` | `Prefs.c:73-93` | **dead code, never called.** |
| `WritePrefs` | `Prefs.c:97-144` | `FSMakeFSSpec`; on `fnfErr` → `FSpCreate(&theSpecs, kPrefCreatorType, kPrefFileType, smSystemScript)`; `FSpOpenDF(…, fsRdWrPerm, …)`; `byteCount = sizeof(*thePrefs)` (= 226); `FSWrite`; `FSClose` |
| `SavePrefs` | `Prefs.c:148-162` | sets `thePrefs->prefVersion = versionNow` (`Prefs.c:153`) **before** writing |
| `ReadPrefs` | `Prefs.c:166-216` | `byteCount = sizeof(*thePrefs)`; `FSRead`; special-cases `eofErr` (short file) |
| `DeletePrefs` | `Prefs.c:220-236` | `FSpDelete` |
| `LoadPrefs` | `Prefs.c:240-269` | on `eofErr` → `BringUpDeletePrefsAlert(); DeletePrefs(); return false;`. On `thePrefs->prefVersion != versionNeed` (`Prefs.c:261`) → same. |
| `BringUpDeletePrefsAlert` | `Prefs.c:273-280` | `InitCursor(); Alert(kNewPrefsAlertID /*160*/, nil)` |

So a version bump or a truncated file silently discards the user's key mapping after one
alert. There is **no migration path**.

---

## 22. Menus and Command-key equivalents

`DoMenuChoice` (`GliderPRO/Sources/Menu.c:591-621`):

```c
if (menuChoice == 0) return;
theMenu = HiWord(menuChoice);
theItem = LoWord(menuChoice);
switch (theMenu) { kAppleMenuID: … kGameMenuID: … kOptionsMenuID: … kHouseMenuID: … }
HiliteMenu(0);                                          // Menu.c:620
```

Menu IDs: `kAppleMenuID 128`, `kGameMenuID 129`, `kOptionsMenuID 130`, `kHouseMenuID 131`
(`GliderPRO/Headers/GliderDefines.h:186-189`).

Reached two ways, both in `Events.c`:

* `inMenuBar` mouse click → `DoMenuChoice(MenuSelect(theEvent->where))`
  (`GliderPRO/Sources/Events.c:75-77`).
* `keyDown`/`autoKey` with Command and not Option → `DoMenuChoice(MenuKey(theChar))`
  (`GliderPRO/Sources/Events.c:172-173`).

### 22.1 Verified MENU resources

Parsed the six `MENU` resources from `GliderPRO/Glider PRO.r`. Format per item: length byte,
text, icon byte, **cmd-key byte**, mark byte, style byte; `0x00` terminates the item list.
`0xC9` is MacRoman `…`; `0x14` is the Apple logo.

**MENU 128 @ `Glider PRO.r:5291`** — menuID 128, title = `0x14` (Apple), enableFlags
`0xFFFFFFFB` (item 2 disabled):

| # | Item | Cmd key |
|---|---|---|
| 1 | `About Glider PRO…` | (none) |
| 2 | `-` | (none) |

**MENU 129 @ `Glider PRO.r:5297`** — menuID 129, title `Game`, enableFlags `0xFFFFFFAF`
(items 4 and 6 disabled):

| # | Item | Cmd key | Handler |
|---|---|---|---|
| 1 | `New Game` (text is 9 bytes: `"New Game"` + a stray trailing NUL) | **N** | `iNewGame 1` |
| 2 | `Two Player Game` | **2** | `iTwoPlayer 2` |
| 3 | `Open Saved Game…` | **O** | `iOpenSavedGame 3` |
| 4 | `-` | — | |
| 5 | `Load House…` | **L** | `iLoadHouse 5` |
| 6 | `-` | — | |
| 7 | `Quit` | **Q** | `iQuit 7` |

**MENU 130 @ `Glider PRO.r:5307`** — menuID 130, title `Options`, enableFlags `0xFFFFFFFB`:

| # | Item | Cmd key | Handler |
|---|---|---|---|
| 1 | `Room Editor` | **E** | `iEditor 1` — toggles `kEditMode`/`kSplashMode` |
| 2 | `-` | — | |
| 3 | `High Scores…` | **H** | `iHighScores 3` |
| 4 | `Preferences…` | **P** | `iPrefs 4` → `DoSettingsMain()` |
| 5 | `Demo…` | **D** | `iHelp 5` → **`DoDemoGame()`** |

**MENU 131 @ `Glider PRO.r:5316`** — menuID 131, title `House`, 21 items, enableFlags
`0xFFFADF77` (items 3, 7, 13, 16, 18 disabled — exactly the separators):

| # | Item | Cmd key | Constant |
|---|---|---|---|
| 1 | `New House…` | **N** | `iNewHouse 1` |
| 2 | `Save House` | **S** | `iSave 2` |
| 3 | `-` | — | |
| 4 | `House Info…` | (none) | `iHouse 4` |
| 5 | `Room Info…` | **R** | `iRoom 5` |
| 6 | `Object Info…` | **I** | `iObject 6` |
| 7 | `-` | — | |
| 8 | `Cut Room` | **X** | `iCut 8` |
| 9 | `Copy Room` | **C** | `iCopy 9` |
| 10 | `Paste Room` | **V** | `iPaste 10` |
| 11 | `Delete Room` | (none) | `iClear 11` |
| 12 | `Duplicate Object` | **D** | `iDuplicate 12` |
| 13 | `-` | — | |
| 14 | `Bring To Front` | **=** | `iBringForward 14` |
| 15 | `Send To Back` | **-** | `iSendBack 15` |
| 16 | `-` | — | |
| 17 | `Go To Room…` | **G** | `iGoToRoom 17` |
| 18 | `-` | — | |
| 19 | `Map Window` | **M** | `iMapWindow 19` |
| 20 | `Tools Window` | **T** | `iObjectWindow 20` |
| 21 | `Coordinate Window` | **K** | `iCoordinateWindow 21` |

Menu-item constants are at `GliderPRO/Headers/Externs.h:197-222`.

**Command-key collisions across menus:** `N` (Game ▸ New Game *and* House ▸ New House),
`D` (Options ▸ Demo *and* House ▸ Duplicate Object). `MenuKey` resolves these by scanning
menus in menu-bar order and returning the first match, so the House-menu `N` and `D` are
**unreachable by keyboard** whenever the Game and Options menus are installed. This is
original behaviour.

**MENU 140 @ `Glider PRO.r:5339`** — resource ID 140 but **`menuID` field = 133**; 19
background names, no command keys. **MENU 141 @ `Glider PRO.r:5360`** — resource ID 141,
`menuID` also **133**; 9 tool-group names. These are popup-menu contents for the tools window
and room-info dialog, not menu-bar menus; the duplicate `menuID` is harmless because they are
never both inserted into the menu list.

### 22.2 `DoGameMenu` / `DoOptionsMenu` input-relevant actions

`DoGameMenu` (`GliderPRO/Sources/Menu.c:301-361`):

| Item | Action | Lines |
|---|---|---|
| `iNewGame` 1 | `twoPlayerGame = false; resumedSavedGame = false; NewGame(kNewGameMode)` | `:305-309` |
| `iTwoPlayer` 2 | `twoPlayerGame = true; resumedSavedGame = false; NewGame(kNewGameMode)` | `:311-315` |
| `iOpenSavedGame` 3 | `resumedSavedGame = true;` **first, unconditionally**, then `HeyYourPissingAHighScore(); if (OpenSavedGame()) { twoPlayerGame = false; NewGame(kResumeGameMode); }` | `:317-325` |
| `iLoadHouse` 5 | gated on `if (splashDrawn)`, then `DoLoadHouse(); OpenCloseEditWindows(); UpdateMenus(false); incrementModeTime = TickCount() + kIdleSplashTicks;` plus an `InvalWindowRect` of the house-name strip when `theMode` is `kSplashMode` or `kPlayMode` | `:327-346` |
| `iQuit` 7 | `quitting = true; if (!QuerySaveChanges()) quitting = false;` (the `QuerySaveChanges` veto is `#ifndef COMPILEDEMO`; the demo build just sets `quitting = true`) | `:348-356` |

`kResumeGameMode 0`, `kNewGameMode 1` (`GliderPRO/Headers/GliderDefines.h:616-617`).

`DoOptionsMenu` (`GliderPRO/Sources/Menu.c:366-431`): `iEditor` toggles between `kEditMode 1`
and `kSplashMode 0` (`:374-415`); `iHighScores` → `DoHighScores` (`:417-420`); `iPrefs` →
`DoSettingsMain` (`:422-425`); `iHelp` → `DoDemoGame` (`Menu.c:427-428`). The edit→splash
branch can be **vetoed**: `if (!QuerySaveChanges()) break;` (`:382-383`) aborts before
`theMode` is touched. Only the edit→splash branch re-arms the idle timer (`:402`, §24.1).

### 22.3 The Brains-panel easter egg

`GliderPRO/Sources/Settings.c:1448`, inside `DoSettingsMain`'s `kBrainsButton` case:

```c
if ((OptionKeyDown()) && (!houseUnlocked))
{
    houseUnlocked = true;
    changeLockStateOfHouse = true;
    saveHouseLocked = false;
}
```

**Option-clicking the Brains button in the Preferences dialog unlocks the current house for
editing.** `OptionKeyDown()` is a `GetKeys` poll (`GliderPRO/Sources/Utilities.c:696-705`),
so this is an input feature, and `houseUnlocked` is the gate on nearly every editor key
command (§17).

---

## 23. Modal-dialog keyboard filters

Every dialog in the game supplies a `ModalFilterUPP` to `ModalDialog`. The filter sees the raw
`EventRecord` before the Dialog Manager does; returning `true` means "I have set `*item`,
treat it as an item hit", `false` means "carry on".

Standard idiom (all filters): `theChar = (event->message) & charCodeMask`, then a switch.

| Filter | Lines | Return/Enter | Escape | Other keys |
|---|---|---|---|---|
| `ControlFilter` | `GliderPRO/Sources/Settings.c:380-498` | *not handled* (falls through to Dialog Manager, which activates Okay) | **captured as Cancel** — but only when it is a *rejected* binding, which it always is | every other key is captured as a **key binding** (§19.4) |
| `PrefsFilter` | `GliderPRO/Sources/Settings.c:1305-1391` | `FlashDialogButton(dial, kOkayButton); *item = kOkayButton; return true` (`:1316-1320`) | — | `b`/`B` → `kBrainsButton` (`:1323-1326`); `c`/`C` → `kControlsButton` (`:1329-1332`); `d`/`D` → `kDisplayButton` (`:1335-1338`); `s`/`S` → `kSoundButton` (`:1341-1344`); all `return true` |
| `LoadFilter` | `GliderPRO/Sources/SelectHouse.c:198-349` | `FlashDialogButton(kOkayButton)` (`:209-213`) | `FlashDialogButton(kCancelButton)` (`:216-219`) | Page Up → item 29; Page Down → item 30; ↑ `-= 4`; ↓ `+= 4`; ← `--`; **Tab and → both `++`** (`:277-278`); A-Z/a-z → type-select |
| `RoomFilter` | `GliderPRO/Sources/RoomInfo.c:299-371` | Okay | Cancel | **Tab → `SelectDialogItemText(dial, kRoomNameItem, 0, 1024)`**; `default:` polls the mouse for tile highlighting (`:365-368`) |
| `ResumeFilter` | `GliderPRO/Sources/Menu.c:668-703` | Okay only | — | — |
| `AboutFilter` | `GliderPRO/Sources/About.c:168-256` | manual button flash then `*hit = kOkayButton` | — | manual press-drag-release tracking (§18.10) |

### 23.1 `LoadFilter` navigation arithmetic (`GliderPRO/Sources/SelectHouse.c:198-349`)

The house-picker grid is **12 cells, 4 columns × 3 rows** (`kDispFiles 12`,
`GliderPRO/Sources/SelectHouse.c:21`). `thisHouseIndex` is the on-page index 0..11;
`housePage` is the page base.

```
screenCount = housesFound - housePage; if (screenCount > 12) screenCount = 12

↑  (SelectHouse.c:232-249)
    thisHouseIndex -= 4
    if thisHouseIndex < 0:
        thisHouseIndex += 4
        thisHouseIndex = (((screenCount - 1) / 4) * 4) + (thisHouseIndex % 4)
        if thisHouseIndex >= screenCount: thisHouseIndex -= 4

↓  (SelectHouse.c:251-261)
    thisHouseIndex += 4
    if thisHouseIndex >= screenCount: thisHouseIndex %= 4

←  (SelectHouse.c:263-275)
    thisHouseIndex--
    if thisHouseIndex < 0: thisHouseIndex = screenCount - 1

→ or Tab  (SelectHouse.c:277-288)
    thisHouseIndex++
    if thisHouseIndex >= screenCount: thisHouseIndex = 0
```

Type-select (`SelectHouse.c:290-321`): accepts `theChar` in `(0x40, 0x5A]` or `(0x60, 0x7A]`
(note: **strictly greater than 0x40**, so `@` is excluded and `A` is included), up-cases
lowercase by `-= 0x20`, then scans `fileFirstChar[0..11]` for the first entry
`>= theChar` and `!= 0x7F`; if none, selects `screenCount - 1`. `fileFirstChar[]` is filled
with `0x7F` sentinels and the up-cased first letter of each visible file name in
`UpdateLoadDialog` (`SelectHouse.c:90-91`, `:120-123`).

The house picker has its **own** double-click detector, independent of `Events.c`'s, built
from `mouseDown`/`mouseUp` in the filter:

```c
case mouseDown:  lastWhenClick = event->when - lastWhenClick;
                 SubPt(event->where, &lastWhereClick);  return false;   // SelectHouse.c:325-328
case mouseUp:    lastWhenClick = event->when;
                 lastWhereClick = event->where;         return false;   // SelectHouse.c:331-334
```

and the test in `DoLoadHouse` (`GliderPRO/Sources/SelectHouse.c:443-448` for name items,
`:478-483` for icon items):

```c
if (lastWhereClick.h < 0) lastWhereClick.h = -lastWhereClick.h;
if (lastWhereClick.v < 0) lastWhereClick.v = -lastWhereClick.v;
if ((lastWhenClick < doubleTime) && (lastWhereClick.h < 5) && (lastWhereClick.v < 5))
    → open the house immediately
```

Same **5-pixel** slop and same `doubleTime` as `Events.c`, but computed by mutating the
"last" variables in place (`SubPt` turns `lastWhereClick` into a delta) rather than by
comparison. Note it measures **mouseUp → next mouseDown**, i.e. the gap *between* clicks,
whereas `Events.c` measures mouseDown → mouseDown.

---

## 24. Idle / attract mode and the blocking input waits

### 24.1 Idle timeout

| Name | Value / type | Citation |
|---|---|---|
| `kIdleSplashTicks` | **7200L** ticks = 120 s (source comment "2 minutes") | `GliderPRO/Headers/GliderDefines.h:197` |
| `kIdleSplashMode` | 0 | `GliderPRO/Headers/GliderDefines.h:197` block |
| `kIdleDemoMode` | 1 | same block |
| `kIdleLastMode` | 1 | same block |
| `incrementModeTime` | `long`, absolute deadline in ticks | `GliderPRO/Sources/Events.c:27` |
| `idleMode` | `short` | `GliderPRO/Sources/Events.c:30` |
| `doAutoDemo` | `Boolean`, from prefs `wasDoAutoDemo`; default **true** | `GliderPRO/Sources/Events.c:31`, `Main.c` defaults |

The deadline is armed/re-armed at **eight** sites, always with the identical statement
`incrementModeTime = TickCount() + kIdleSplashTicks;` (this is the complete list — verified by
grepping every `.c` file):

1. `VariableInit` — startup (`GliderPRO/Sources/InterfaceInit.c:163`).
2. On app resume: `HandleOSEvent` (`GliderPRO/Sources/Events.c:427`, inside `:390-442`).
3. After a game ends: `NewGame` tail (`GliderPRO/Sources/Play.c:277`).
4. After the attract-mode demo finishes: `DoDemoGame` tail (`GliderPRO/Sources/Play.c:302`).
5. Game menu → Load House (`GliderPRO/Sources/Menu.c:336`).
6. Options menu → Editor, but **only** on the edit→splash transition
   (`GliderPRO/Sources/Menu.c:402`); the splash→edit transition does *not* re-arm.
7. Options menu → High Scores (`GliderPRO/Sources/Menu.c:419`) and → Preferences (`:424`).
8. Apple Event "open document" handler, after loading a dropped house
   (`GliderPRO/Sources/AppleEvents.c:111`).

Note that Options → Help (`iHelp` → `DoDemoGame`) does not need its own re-arm because
`DoDemoGame` re-arms at its tail (site 4).

and fired in `HandleEvent` (`GliderPRO/Sources/Events.c:536-540`):

```c
if ((theMode == kSplashMode) && (doAutoDemo) && (!switchedOut))
    if (TickCount() >= incrementModeTime)
        DoDemoGame();
```

**Only in `kSplashMode`** — the timer does not fire in the editor.

### 24.2 `WaitForInputEvent` (`GliderPRO/Sources/Utilities.c:439-479`)

The universal "press anything to continue, or time out" primitive. It is the one place that
uses **both** input mechanisms at once.

```
 1. timeToBail = TickCount() + 60L * (long)seconds                # Utilities.c:446
 2. FlushEvents(everyEvent, 0)                                    # Utilities.c:447
 3. waiting = true ; didResume = false                            # Utilities.c:448-449
 4. while waiting:                                                # Utilities.c:451
 5.     GetKeys(theKeys)                                          # Utilities.c:453
 6.     if BitTst(kCommandKeyMap) or BitTst(kOptionKeyMap)
 7.        or BitTst(kShiftKeyMap) or BitTst(kControlKeyMap):      # Utilities.c:454-455
 8.         waiting = false                                       # Utilities.c:456
 9.     if GetNextEvent(everyEvent, &theEvent):                    # Utilities.c:457
10.         if theEvent.what == mouseDown or keyDown:              # Utilities.c:459
11.             waiting = false                                   # Utilities.c:460
12.         elif theEvent.what == osEvt and (message & 0x01000000):# Utilities.c:461
13.             if message & 0x00000001:      # resuming            # Utilities.c:463
14.                 didResume = true ; waiting = false             # Utilities.c:465-466
15.             else: InitCursor()            # suspending          # Utilities.c:470
16.     if seconds != -1 and TickCount() >= timeToBail:            # Utilities.c:474
17.         waiting = false                                       # Utilities.c:475
18. FlushEvents(everyEvent, 0)                                    # Utilities.c:477
19. return didResume                                              # Utilities.c:478
```

Key points:

* **The four modifier keys are polled**, because modifier presses do not generate `keyDown`
  events on classic Mac OS. Every other key arrives as an event.
* `seconds == -1` means **wait forever**.
* `FlushEvents` at both ends discards queued input so a keystroke from before the wait cannot
  satisfy it and a keystroke that satisfied it cannot leak into the next screen.
* Returns `true` **only** if the app was resumed, telling the caller to redraw.
* This is a **tight busy-wait** — no yielding, no sleeping. It pegs the CPU for up to
  `seconds`. A Go port must replace it with a channel select on {input event, timer}.

Call sites: `GliderPRO/Sources/Banner.c:190-192` (`demoGoing ? WaitForInputEvent(4) :
WaitForInputEvent(15)`), `Banner.c:232-233` (`DelayTicks(60); if (WaitForInputEvent(30))
RestoreEntireGameScreen();`), `GliderPRO/Sources/HighScores.c:81-82` (`DelayTicks(60);
WaitForInputEvent(30);`), `GliderPRO/Sources/GameOver.c:226` (`WaitForInputEvent(5)`),
`GameOver.c:497` (`(1)`), `GameOver.c:502` (`(10)`).

### 24.3 `WaitCommandQReleased` (`GliderPRO/Sources/Utilities.c:485-499`)

```c
waiting = true;
while (waiting)
{
    GetKeys(theKeys);
    if ((!BitTst(&theKeys, kCommandKeyMap)) || (!BitTst(&theKeys, kQKeyMap)))
        waiting = false;
}
FlushEvents(everyEvent, 0);
```

Called once, at the end of `NewGame` (`GliderPRO/Sources/Play.c:275`). Its purpose: Command-Q
during play is detected by polling (`GliderPRO/Sources/Input.c:53-56`), and after the game
loop exits the Event Manager would *also* deliver the queued Command-Q `keyDown` to
`HandleEvent`, which would route it to `MenuKey('q')` → `iQuit` → quit the whole application.
The spin waits for the physical release, then `FlushEvents` drops the queued events. **A port
that does not replicate this will quit the app whenever the player quits a game with
Command-Q.**

### 24.4 The `GameOver.c` abort idiom

`GliderPRO/Sources/GameOver.c:209-220` and `:461-478` contain the same construct twice — a
frame-paced animation loop that any input aborts:

```c
do
{
    GetKeys(theKeys);
    if ((BitTst(&theKeys, kCommandKeyMap)) || (BitTst(&theKeys, kOptionKeyMap)) ||
        (BitTst(&theKeys, kShiftKeyMap))   || (BitTst(&theKeys, kControlKeyMap)))
        <abort>;
    if (GetNextEvent(everyEvent, &theEvent))
        if ((theEvent.what == mouseDown) || (theEvent.what == keyDown))
            <abort>;
}
while (TickCount() < nextLoop);
nextLoop = TickCount() + 2;
```

`FlushEvents(everyEvent, 0)` at `GameOver.c:152` and `:454` bracket these.

---

## 25. There is no joystick, gamepad or other device support

Verified by exhaustive search of `GliderPRO/Sources/*.c`, `GliderPRO/Headers/*.h`,
`GliderPRO/Prefix.h` and the 199843-line `GliderPRO/Glider PRO.r` for every plausible
identifier: `joystick`, `joypad`, `gamepad`, `ADBOp`, `InputSprocket`, `ISp`, `HID`. **Zero
matches** in every case (case-sensitive; a case-*insensitive* search hits unrelated
identifiers such as `isPlayMusicIdle` and `HideCursor`, so re-run it case-sensitively).

The complete set of input APIs the program uses is:

| Toolbox call | Purpose | Where |
|---|---|---|
| `GetKeys(KeyMap)` | poll all keys | 15 sites, §26 |
| `BitTst(ptr, offset)` | test one bit of the KeyMap | throughout |
| `WaitNextEvent` / `GetNextEvent` | fetch keyboard/mouse/OS events | all 7 sites: `Events.c:500`, `:504`, `Play.c:393`, `Utilities.c:457`, `GameOver.c:215`, `:470`, plus `Validate.c:283` (a `diskMask`-only `GetNextEvent`, not user input) |
| `FlushEvents(everyEvent, 0)` | discard queued input | all 12 sites: `Utilities.c:48`, `:447`, `:477`, `:498`, `Input.c:390`, `Settings.c:1438`, `GameOver.c:152`, `:454`, `Main.c:383`, `HighScores.c:402`, `:514`, `SavedGames.c:46` |
| `MenuKey` / `MenuSelect` / `HiliteMenu` | menu command keys / menu bar | `Events.c:76`, `:173`, `Menu.c:620` |
| `Button()` | is the mouse button down | all 6 sites: `About.c:174`, `MainWindow.c:587`, `DebugUtilities.c:111`, `:114`, `:143`, `:146`. (There is **no** `Button()` call in `Events.c`.) |
| `GetMouse` / `LocalToGlobal` / `GlobalToLocal` | cursor position | `Utilities.c:33-34`, `Marquee.c`, `RoomInfo.c` |
| `StillDown` / `WaitMouseUp` | drag loops | `ObjectEdit.c:82`, `:115`, `Marquee.c`, `RoomInfo.c` |
| `GetDblTime` | double-click interval | `InterfaceInit.c:184` |
| `FindWindow` / `FindControl` / `TrackControl` / `TrackGoAway` / `TrackBox` / `DragWindow` / `GrowWindow` | window and control hit-testing | `Events.c:60-157`, `Tools.c`, `Map.c`, `Link.c` |
| `ModalDialog` + `ModalFilterUPP` | dialog input | §23 |
| `HideCursor` / `InitCursor` / `SetCursor` | cursor visibility and shape | `Play.c:104`, `Input.c:69`, `:389`, `Utilities.c:65`, `Marquee.c` |
| `Delay` | busy sleep | `Utilities.c:735` |
| `TickCount` | 60 Hz clock | throughout |
| `SysBeep(1)` | reject feedback | `Settings.c:402`, `:424`, `:446`, `:468` (the four `ControlFilter` slots), `Main.c:276`, `SelectHouse.c:144`, `:174` |

---

## 26. Every `GetKeys` call site

All 15, exhaustively:

| # | File:line | Context | Bits tested |
|---|---|---|---|
| 1 | `GliderPRO/Sources/Input.c:91` | `DoPause` — wait for pause key release before pausing | pause key (`kEscKeyMap` or `kTabKeyMap`) |
| 2 | `GliderPRO/Sources/Input.c:99` | `DoPause` — the paused spin loop | pause key, `kCommandKeyMap` |
| 3 | `GliderPRO/Sources/Input.c:113` | `DoPause` — wait for release before resuming | pause key |
| 4 | `GliderPRO/Sources/Input.c:190` | `GetDemoInput`, player 1 only | 4 game keys (arcade) or `kCommandKeyMap` |
| 5 | `GliderPRO/Sources/Input.c:285` | **`GetInput`, player 1 only — the main game poll** | all four glider keys for *both* players, `kCommandKeyMap`, `kDeleteKeyMap`, pause key |
| 6 | `GliderPRO/Sources/Events.c:486` | `HandleEvent` pre-poll | `kCommandKeyMap`, `kOptionKeyMap` |
| 7 | `GliderPRO/Sources/Utilities.c:453` | `WaitForInputEvent` | Command, Option, Shift, Control |
| 8 | `GliderPRO/Sources/Utilities.c:494` | `WaitCommandQReleased` | `kCommandKeyMap`, `kQKeyMap` |
| 9 | `GliderPRO/Sources/Utilities.c:700` | `OptionKeyDown` | `kOptionKeyMap` |
| 10 | `GliderPRO/Sources/MainWindow.c:354` | after an editor click — sticky tool | `kShiftKeyMap` |
| 11 | `GliderPRO/Sources/ObjectAdd.c:740` | placing a flower — reuse variant | `kShiftKeyMap` |
| 12 | `GliderPRO/Sources/Room.c:232` | after creating a room — suppress Room Info | `kShiftKeyMap` |
| 13 | `GliderPRO/Sources/ObjectEdit.c:2754` | `HiliteAllObjects` release spin | `kCommandKeyMap`, `kOptionKeyMap` |
| 14 | `GliderPRO/Sources/GameOver.c:211` | animation abort | Command, Option, Shift, Control |
| 15 | `GliderPRO/Sources/GameOver.c:463` | animation abort | Command, Option, Shift, Control |

`theKeys` in `Input.c` is a **file-scope global** (`GliderPRO/Sources/Input.c:31`) shared by
`DoPause`, `DoCommandKey`, `GetInput` and `GetDemoInput`; everywhere else `theKeys` /
`theseKeys` / `eventKeys` is a local. See §2.2 for why the sharing matters (player 2 reads the
snapshot player 1 took).

---

## 27. Mac Toolbox → Go replacement table

| Mac Toolbox / 68k-PPC facility | What it does here | Go replacement |
|---|---|---|
| `GetKeys(KeyMap)` — 16-byte bitmap of all 128 keys | the entire in-game input mechanism | maintain a `[128]bool` (or `[2]uint64`) updated from key-down/key-up callbacks; snapshot it once per frame |
| `BitTst(ptr, offset)` — **MSB-first** bit test | reads one key from the KeyMap | `keys[offset]` after mapping via §3; do **not** implement MSB-first bit maths unless you also load real prefs files |
| `KeyMap` offset numbering | how key identities are stored in prefs and structs | keep the 0-127 offset as the canonical key ID so prefs files stay readable; provide `offsetToScancode` / `scancodeToOffset` tables |
| Mac virtual key codes (ADB) | what the hardware reports | map your platform's scancodes to Mac virtual codes, then apply `KeyMapOffsetFromRawKey` |
| `WaitNextEvent` / `GetNextEvent` + `EventRecord` | menus, editor keys, dialogs, window management | your GUI toolkit's event loop; `EventRecord.message` splits into `charCodeMask` (0xFF) and `keyCodeMask` (0xFF00) which correspond to *rune* and *scancode* |
| `charCodeMask` dispatch (`Events.c:167`) | editor shortcuts follow keyboard layout | dispatch on the produced rune, not the physical key |
| `autoKey` events | editor commands auto-repeat, including Delete | enable key repeat for editor commands; **never** for game keys (they are polled) |
| `FlushEvents(everyEvent, 0)` | drop stale input at mode boundaries | drain your input channel |
| `MenuKey` / `MenuSelect` / `HiliteMenu` | Command-key menu equivalents (§22) | your menu system; replicate the first-match collision behaviour or fix it deliberately |
| `ModalDialog` + `ModalFilterUPP` | all dialog input, including key capture for remapping | a modal UI state that gets raw key events before the widgets |
| `GetDblTime()` | double-click interval | a configurable duration (classic default 32 ticks ≈ 533 ms) |
| `TickCount()` — 60 Hz since boot | all timing; `kTicksPerFrame 2` → 30 fps | a monotonic clock; 1 tick = 16.667 ms exactly |
| busy-wait `while (TickCount() < nextFrame) {}` (`Render.c:662-665`) | frame pacing | `time.Ticker` at 30 Hz, or vsync; do not busy-wait |
| `Delay(ticks, &dummy)` | short blocking sleeps | `time.Sleep` |
| `SysBeep(1)` | rejected-key feedback | a short error sound or a visual shake |
| `HideCursor` / `InitCursor` / `SetCursor(&handCursor)` + `CURS` 128-131, `acur` 128 | cursor hidden during play; four editor drag cursors | hide the OS cursor during play; ship the four cursor images |
| `PenMode(patXor)` + `PenPat(gray)` XOR rubber-banding (`Marquee.c`, `RoomInfo.c`, `ObjectEdit.c:2734`) | selection outlines and drag previews | draw an overlay each frame; XOR erase-by-redraw has no analogue |
| `StillDown` / `WaitMouseUp` / `GetMouse` drag loops | modal drag interaction | a drag state machine driven by mouse-move events |
| `FindWindow` / `FindControl` / `TrackControl` / `GrowWindow` / `ZoomWindow` | multi-window editor chrome | your GUI toolkit's windows and widgets |
| `GetNewDialog` / `DITL` / `DLOG` / `ALRT` resources | dialog layout as big-endian resource data | hand-written layout code or a data file; the verified DITL tables in §19.2 and §11.1 are the layout |
| `PICT` 1015/1016 (214×54) | the pause overlay graphic | pre-decoded PNGs |
| `GetResource('demo', 128)` + `BlockMove` | attract-mode input stream, 6702 bytes | embed the bytes; parse as §14.6 |
| `AddResource(..., 'demo', 128, ...)` (`DebugUtilities.c:352`) | how the demo was recorded | a `--record-demo` flag writing the same 6-byte records |
| big-endian on-disk `long` | prefs key maps, `demoType.frame` | `encoding/binary.BigEndian` |
| `#pragma options align=mac68k`, 4-byte `long` | 226-byte `prefsInfo`, 110-byte `gliderType` | explicit offset-based (un)marshalling; never rely on Go struct layout |
| `Str15`/`Str31`/`Str32` Pascal strings | key display names in prefs | length-prefixed byte slices; convert MacRoman → UTF-8 |
| MacRoman text encoding (`0xC9` = `…`, `0x14` = Apple logo) | menu/dialog strings | decode as MacRoman |
| `Gestalt` / `FindFolder` / `FSSpec` / `FSpCreate` / `FSWrite` | prefs file location and I/O | `os.UserConfigDir()` + `os.WriteFile` |
| 8-bit indexed colour, `kPreferredDepth 8`, `kRedOrangeColor8 23` | palette indices used as colours | resolve index 23 through the game's `clut` to an RGB triple |
| `AEProcessAppleEvent` | Open-Document / Quit Apple events | command-line arguments and window-close |
| suspend/resume `osEvt` (`message & 0x01000000`, then `& 0x00000001`) | pause music, show cursor, re-arm idle timer | window focus lost/gained callbacks |

---

## Open questions

1. **Is the swapped `// left key` / `// right key` comment pair in `GetDemoInput`
   (`GliderPRO/Sources/Input.c:228`, `:235`) the only place the demo key encoding is
   documented?** The recorded resource, `LogDemoKey`'s call sites and the playback code all
   agree that 0 = right and 1 = left, so the comments are simply wrong — but I cannot rule out
   that an earlier revision had the opposite encoding and that some third-party recorded demo
   exists with the other convention. Nothing in the shipped data suggests it.

2. **What is the intended semantics of `wasEscPauseKey = !wasEscPauseKey`
   (`GliderPRO/Sources/Settings.c:583`)?** As written, clicking the already-selected pause
   radio desynchronises the flag from the radio graphics. I have documented it as a bug; a port
   must decide whether to reproduce it. There is no comment or changelog to settle intent.

3. **Was player 2's control scheme ever meant to be remappable?** `theGlider2`'s four keys are
   hard-coded in `VariableInit` (`GliderPRO/Sources/InterfaceInit.c:148-151`), `prefsInfo` has
   exactly one set of four maps, and DITL 1023 has exactly four icons. So the answer is
   presumably "no", but the fact that the keys live in the *per-glider* struct rather than in a
   global suggests the plumbing was built for two sets.

4. **Why is `kSavingGameDial 1042` defined in `Input.c:19` but never used?** DLOG/ALRT 1042 is
   `kColorSwitchedAlert` (`GliderPRO/Sources/Events.c:46-55`), so the ID is reused. The
   Command-S path uses `RefreshScoreboard(kSavingTitleMode)` instead of a dialog. Presumably a
   removed feature.

5. **What exactly does `HandleTelephone()` do with input?** It is called once per frame from
   `PlayGame` (`GliderPRO/Sources/Play.c:445`) before `GetInput`. I did not read
 `Sounds.c`/`Utilities` deeply enough to confirm it never consumes events; from its name and
   position it appears to be pure audio/animation, but a port should verify.

6. **Does any house rely on the out-of-bounds `demoData[demoIndex]` read past 1117 records?**
   The shipped Demo House ends the game before the stream is exhausted in normal play, but I
   could not confirm this by running the game (no Mac emulator available here). If a port adds
   a bounds check and the demo behaves differently, this is the reason.

7. **What are the exact `kNumFlowers`, `kMaxHotSpots` etc. values relevant to the Shift-flower
   gesture?** I confirmed the gesture (`GliderPRO/Sources/ObjectAdd.c:740-742`) but did not
   trace `kNumFlowers`; the six `flowerSrc` rects in
   `GliderPRO/Sources/StructuresInit2.c:72-88` imply 6.

8. **Is `theKeys`'s file-scope sharing between `GetInput(&theGlider)` and
   `GetInput(&theGlider2)` deliberate or accidental?** It is *load-bearing* (player 2 has no
   `GetKeys` call of its own), so it must be reproduced, but there is no comment saying so.

## Porting notes

1. **Model input as a 128-entry key-down bitmap sampled once per frame, not as an event
   stream.** The game's semantics depend on it: holding a key thrusts every frame; the
   both-keys-held about-face; `fireHeld` edge detection; `heldLeft`/`heldRight` gating stair
   transit. An event-driven port will feel wrong in ways that are hard to diagnose.

2. **Keep the KeyMap offset (0-127) as the canonical key identifier.** It is the value stored
   in `gliderType.leftKey` and in the prefs file, and it is what `GetKeyMapFromMessage`
   produces. Implement `offset = (v & ~7) + (7 - (v & 7))` (verified self-inverse and bijective
   over 0..127, §3.2) to convert to and from Mac virtual key codes, and a separate table from
   your platform's scancodes to Mac virtual codes. Do not invent a new key ID space unless you
   also abandon prefs-file compatibility.

3. **Sample the keyboard exactly once per frame, at the top of the frame, into one shared
   snapshot, and let both players read it.** `GetInput` calls `GetKeys` only when
   `thisGlider->which == kPlayer1` (`GliderPRO/Sources/Input.c:283-285`) and `theKeys` is a
   file-scope global (`Input.c:31`), so player 2's `GetInput` reuses player 1's snapshot. Two
   independent polls would introduce a one-call-latency skew.

4. **Player 1 is polled first, and interactions run after both.** `PlayGame` order is
   `HandleDynamics` → `GetInput(&theGlider)` → `GetInput(&theGlider2)` → `HandleInteraction`
   → `HandleTriggers` → `HandleBands` → `HandleGlider` ×2 → `RenderFrame`
   (`GliderPRO/Sources/Play.c:447-471`). Preserve it: `AddBand` mutates the shared `bands[]`
   array and `numBands`, so which player fires first when both fire on the same frame with
   `numBands == 1` matters.

5. **`hDesiredVel` and `vDesiredVel` are accumulators that `MoveGlider` zeroes every frame.**
   `hDesiredVel = 0` at `GliderPRO/Sources/Player.c:78` and `vDesiredVel = kGravity` (3) at
   `:92`. Input adds to them; physics consumes and resets them. If you make them persistent
   state the glider accelerates without bound.

6. **The both-keys-held branch does not write `tipped`.** `GliderPRO/Sources/Input.c:306-311`
   sets `heldLeft = true` and calls `ToggleGliderFacing` but leaves `tipped` at its previous
   frame's value. Zeroing it "for cleanliness" changes the sprite and the rubber-band launch
   angle (`RubberBands.c:256` region: `vVel = tipped ? -2 : 0`).

7. **`fireHeld` is set only when `AddBand` succeeds.** `AddBand` returns `false` when
   `numBands >= kMaxRubberBands` (2) (`GliderPRO/Sources/RubberBands.c:258-259`), so a held
   fire key retries every frame until a slot frees. And the `else` at `Input.c:363-364` clears
   `fireHeld` whenever `bandsTotal` hits 0 or the glider leaves `kGliderNormal`, so picking up
   bands while still holding the key fires immediately. Both are observable.

8. **`DoHeliumEngaged` clears `batteryWasEngaged` on exhaustion; `DoBatteryEngaged` does
   not.** `GliderPRO/Sources/Input.c:169` versus the corresponding gap at `:140-145`. The
   effect is on the thrust-sound retrigger phase. Reproduce verbatim.

9. **Reproduce the exact pause state machine, including both release-waits.** `DoPause`
   (`GliderPRO/Sources/Input.c:77-117`) waits for the pause key to be *released* before
   entering the pause (`:89-94`) and again before leaving (`:111-116`). Without them a single
   keypress pauses and unpauses in the same instant. Note also that the entire game loop is
   suspended inside `DoPause`'s `while (paused)` spin — no rendering, no physics, no event
   dispatch except `DoCommandKey`.

10. **`WaitCommandQReleased` is not optional.** Without it (`GliderPRO/Sources/Play.c:275`) the
    Command-Q that ended the game is still in the event queue and `HandleEvent` will route it
    to `MenuKey('q')` → quit the application.

11. **`DoCommandKey`'s two branches are gated *differently*.** Command-S is gated whole on
    `!twoPlayerGame` (`GliderPRO/Sources/Input.c:65`); Command-Q is **not** gated — only the
    save prompt inside it is, on `(!twoPlayerGame) && (!demoGoing)`
    (`GliderPRO/Sources/Input.c:59`). So in a two-player game Command-Q *does* set
    `playing = false` (ending the game) but never offers to save; Command-S does nothing at
    all. Note that player 2's right-thrust key **is** `kCommandKeyMap`, so player 2 thrusting
    right triggers the Command-Q check every frame — harmless only because Q is not also held.

12. **Delete-suicide requires four conditions**: `otherPlayerEscaped != kNoOneEscaped` (−1),
    Delete held, `thisGlider->which` (i.e. player 1, since `kPlayer1 == TRUE`), and
    `!onePlayerLeft` (`GliderPRO/Sources/Input.c:366-368`). Because of the `which` test only
    player 1's `GetInput` can trigger it, but `ForceKillGlider`
    (`GliderPRO/Sources/Transit.c:449-469`) kills whichever glider is *not* in limbo.

13. **The pause key is a boolean choice between two fixed keys, not a binding.**
    `isEscPauseKey ? kEscKeyMap (50) : kTabKeyMap (55)`. It is stored as a `Boolean` at prefs
    offset 218, and both keys are permanently excluded from remapping
    (`GliderPRO/Sources/Settings.c:392-393`).

14. **The arcade build changes what the arrow keys do outside a game** (§15.2), disabling
    keyboard room navigation and object nudging entirely. `BUILD_ARCADE_VERSION` is 1 in the
    shipped source (`GliderPRO/Headers/GliderDefines.h:16`). Ship the arcade behaviour as the
    default; expose the editor navigation as an option.

15. **`keyDown` and `autoKey` share one handler**, so every editor command auto-repeats,
    including `DeleteRoom` (`GliderPRO/Sources/Events.c:515-517`, `:253-261`). And Delete /
    Page Up / Page Down are gated only on `houseUnlocked`, not on `theMode == kEditMode`.

16. **Prefs are 226 bytes with two uninitialised pad bytes and four big-endian 4-byte key
    maps** (§21.2). If you want to read real `Glider Prefs` files, marshal by explicit offset.
    A version mismatch against `kPrefsVersion 0x0034` silently deletes the file
    (`GliderPRO/Sources/Prefs.c:261-266`).

17. **The demo stream is 1117 six-byte big-endian records and is only valid against the
    house named `"Demo House"`** (`GliderPRO/Sources/SelectHouse.c:639`). Ignore the sixth
    byte. Add the bounds check the original lacks.

18. **Replace both busy-wait loops.** Frame pacing
    (`GliderPRO/Sources/Render.c:662-665`, `kTicksPerFrame 2` → 30 fps) and
    `WaitForInputEvent` (`GliderPRO/Sources/Utilities.c:451-476`) both spin. Use a ticker and
    a select respectively — but keep the *timing* identical: 2 ticks per frame is exactly
    33.333 ms, and `WaitForInputEvent(n)` times out after exactly `60 × n` ticks.

19. **The remap UI's reserved-key set and its Esc-cancels behaviour are user-visible.** Tab,
    Esc and Delete cannot be bound; Esc cancels the dialog rather than beeping; any key already
    bound to another action beeps (`GliderPRO/Sources/Settings.c:391-402`). But Command,
    Option, Control and Shift *can* be bound, which breaks Command-Q and collides with player
    2 — consider warning, but note that doing so is a deviation.

20. **`whichCtrl` indices do not match screen order**: 0 = right, 1 = left, 2 = battery,
    3 = bands, while the icons run left, right, battery, bands from left to right (DITL 1023,
    §19.2). And `whichCtrl` starts at **1** (left), not 0
    (`GliderPRO/Sources/Settings.c:521`).

21. **Store display names alongside offsets, or accept a cosmetic difference.** The original
    persists the four `Str15` names produced by `GetKeyName` (§20) rather than recomputing
    them, so e.g. a keypad `5` shows as `(5)` and a main-keyboard `5` as `5 key` — a
    distinction an offset alone cannot recover without a reverse table.

22. **Nothing in play mode reads the mouse, and the cursor is hidden.** Do not add mouse
    controls, and do remember to hide the cursor (`GliderPRO/Sources/Play.c:104`) and restore
    it on suspend (`Play.c:419`).
