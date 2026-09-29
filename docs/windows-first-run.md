# The Windows backend's first run

Everything under `internal/platform/win32/` and `internal/audio/waveout_windows.go` was written
on an offline Linux machine with no Windows on it and no way to run one. Until 2026-09-21 the
release notes said so in as many words — *"the Windows code has never been run"* — because it was
true, and a caveat is cheaper than a bug report from a stranger.

It has now been run. This is what was run, what it proved, and the four things it did not prove.
It covers **windows/amd64 only**; `windows/arm64` is still unexecuted and the caveat stands there.

The two results worth the page are at the top. Everything after them is the supporting detail.

## The two results

**1. The pixels are identical to Linux's, including the ones on the actual desktop.** The six
title screens the game can render came out byte-for-byte equal to the same version's Linux
output, and so did a screenshot of the running window: the game's client area, captured by the
operating system off a real desktop, is an exact pixel-for-pixel match for a frame the Linux build
renders. Not "looks right" — no differing pixel.

That last part is the one that needed a camera rather than an exit code. `Present` in
`internal/platform/win32/win32.go` **deliberately does not check what `StretchDIBits` returns**,
so a clean exit and a plausible frame rate prove the blit was *called*, not that anything reached
a screen. A green bench is consistent with a window that never painted. The screenshot is not.

**2. The sound device took every sound the game gave it.** `-audio list` found `waveout`, the sink
opened, and across the four runs that used the device rather than a WAV file the game asked for 73
sounds and the driver played 73, refusing none.

## The machine

| | |
|---|---|
| OS | Windows Server 2025 Datacenter, build 26100 (`10.0.26100.32690`) |
| Architecture | amd64 |
| Session | an interactive remote-desktop session, logged in, with a real desktop and a real sound device |
| Binary | `glidergo-windows-amd64.exe` from `make cross`, tag `v0.1.1`, cross-compiled on Linux |
| Beside the binary | nothing — it was copied alone into a temporary directory and run from there |

Server 2025 rather than a desktop Windows is worth naming: it is the same GDI and the same
`waveOut`, but it is not the typical player's machine, and a second run on Windows 10 or 11 would
still be worth having.

## What was run

Fourteen steps, scripted, plus one paced capture run. Every step's exit code and stdout were
logged; all fifteen exited 0 except step 14, which is discussed under
[Two non-findings](#two-non-findings) below.

### The binary carries itself

```
glidergo v0.1.1
  backend   win32
  built by  go1.23.12
  platform  windows/amd64
  assets    built in (1877 files, 15921 KiB)
  art       found (built-in:art)
  sound     found (built-in:sound)
  houses    found (built-in:houses)
  houseart  found (built-in:houseart)
```

All four asset roots resolved to `built-in:`, from a directory containing only the `.exe`. That is
the claim the README makes about the release archives, tested rather than assumed.

The numbers above are v0.1.1's and are left as they were logged. A build since then carries a fifth
root and a twenty-third house — `levels`, from a second embedded archive — so a current `-version`
reports 1878 files and a current `-shot-screen houses` says `23 houses`. Four of the six render
hashes below have moved with it: **splash, houses, about and credits**. Three of those four move for
one shared reason, which is the status band along the bottom — it counts the houses, so "22 houses"
became "23 houses: 22 Original, 1 New" on every screen that draws it. `settings` and `scores` are
unchanged because those two replace the band with a message of their own. The picker also draws a
strip of set names now, and the About box says that houses of this port's own ship alongside the 1994
ones. So the table is v0.1.1's and reproduces against v0.1.1, which is what the instructions at the
end already ask for. Nothing else in this report is affected: every finding below is about the
backend, the sound and the pixels.

### The six renders, against Linux

`-shot` draws one title-screen frame to a PNG and needs no display. Each of the six reported
`22 houses, 0 skipped`, and each file is byte-identical to the same version's Linux render:

| Screen | SHA-256 (identical on both platforms) |
|---|---|
| splash | `1ff1cd5e5d7f0464c20fc77dc1140b791f49a1307786c87e02134da77f5c847f` |
| houses | `b533e671b4585f84a8c30d3ebb6e1362b3561b18804be28c216b01a3da8c0cb4` |
| settings | `fa8abcb3671c3b4b70b37ae283d4431b8b8aa87b680c0a963f38e16a1332b66c` |
| about | `479334299fc95c0d484793c715d9291aaeae4d64f1e98423c4ee71485042e1ed` |
| credits | `3d5ca6828a33ae76da9a20fe2133791a26baebe0ebab5f6a1e37af8ca2a7207f` |
| scores | `3354697a96decef0a85a83878e6a23fcbdf5debd9278ca1c23f5eb42d1aa218f` |

These are reproducible: `-shot` output has no timestamp, no path and no locale in it, so the same
tag on any amd64 host should produce the same six hashes. That makes them a usable regression
check and not just a story about one afternoon — see the reproduction commands at the end.

This is the payoff of the work in commit `66f6e6f`, which made the pixel and audio arithmetic
architecture-independent. Byte-equality across two operating systems and two compilers' worth of
code generation is a stronger statement about `internal/render` than any test on one platform.

### Five bench runs on screen

These are the first executions of `CreateWindowExW`, the message pump and `StretchDIBits` by
anybody. All five opened a window, ran to completion and exited 0.

All five played Slumberland, which is what `-frames` selects when no house is named.

| Run | Frames | Wall | Rate |
|---|---|---|---|
| plain | 300 | 387 ms | 775.1 fps |
| writing a `-wav` | 900 | 951 ms | 945.9 fps |
| `-house Slumberland` explicitly | 1800 | 2.012 s | 894.8 fps |
| `-scale 2` | 120 | 633 ms | 189.6 fps |
| no path overrides at all | 600 | 701 ms | 856.4 fps |

Unpaced, so these are headroom figures, not the game's speed. `-scale 2` at 189.6 fps is the
interesting one: against the 856.4 fps of the 600-frame run at `-scale 1`, magnification costs
about 4.5× per frame and is still six times the original's target. That answers whether the naive
`StretchDIBits` magnify needs replacing. It does not.

### One paced run, which is the real test

```
glidergo: 600 frames in 20.067s (29.9 fps), score 0, 6 stars left
glidergo: sound -- 16 requests, 16 played, 0 refused, 0 cut off, 5 music pieces
glidergo: mix -- 20.0s of audio, 197 samples clipped
```

29.9 fps against the original's 30.07 target, held over twenty seconds, with the frame pacer doing
the work rather than the machine. `0 cut off` is the line to read: in the unpaced runs the game
starts sounds faster than they can finish and cuts them off by design, and here nothing was cut
off because time ran at the speed the game expects.

### The screenshot

Three captures were taken off the desktop at about 4, 9 and 14 seconds into that paced run, by
`CopyFromScreen` — the operating system's own view of the screen, not the game's. The desktop was
592×440, so the window's client area was clipped to 584×409 of the 640×480 the game wants.

Cropping each capture's client area and comparing it to the Linux `-dump` frames of the identical
command gives an exact match, no differing pixel: captures 1 and 2 to frame 41, capture 3 to frame
40. Captures 1 and 2 match frames 59 and 61 just as exactly, for the reason in the next paragraph.

Captures 1 and 2 being identical to each other looked at first like a frozen window, which is
exactly what it would look like. It is not: by frame 40 the scene has settled into a two-frame
alternation, and frames 40 and 41 differ *only* inside a 46×19 box at (426, 154) — the glider
animating in place in an otherwise static room. Captures 1 and 2 landed on the same parity. A
frozen window would have failed the frame-40 match as well; capture 3 made it.

So: the window opened, the DIB reached the screen, and the pixels that got there were the right
ones.

### Sound on the device

| Run | Requested | Played | Refused |
|---|---|---|---|
| 300 frames | 9 | 9 | 0 |
| Slumberland 1800 | 42 | 42 | 0 |
| `-scale 2` 120 | 6 | 6 | 0 |
| 600 frames | 16 | 16 | 0 |
| **total** | **73** | **73** | **0** |

The `-wav` runs are excluded because they never touch the device. The file the 900-frame one wrote
is a valid 22255 Hz mono 16-bit WAV with 98.6% of its samples non-zero, so the mixer's output is
real audio and not silence with a header on it.

The unpaced runs also report samples `dropped by waveout` — 5,673, 30,809, 4,597 and 10,517. That
is not a fault. `-bench` mixes far faster than 22 kHz of audio can play, so the ring buffer fills
and the excess is discarded; the paced run dropped none. A *paced* run that drops anything is the
thing to look at.

### The `%AppData%` branch

`internal/datadir/datadir.go` falls through to `os.UserConfigDir()` off Linux, which on Windows is
`%AppData%`. That branch had never run either. It was exercised directly with `-import-prefs` on a
synthetic 226-byte 1994 `Glider Prefs` record: `…\AppData\Roaming\glidergo\prefs.json` was
created, and read back on the next start.

## Two non-findings

Both of these looked like platform bugs for a while and neither is one. They are written down
because the next person to run this will see them too.

**Step 14 checked for something the game is documented not to do.** The step ran `dir /s /b` over
`%AppData%\glidergo` after the bench runs and found nothing, exiting 1. The expectation was wrong,
not Windows: `-house` with `-frames` goes through `playDirect` in `cmd/glidergo/main.go`, whose own
comment says it saves nothing, and prefs are written only when the shell's *selected house*
changes. A Linux run of the same command with a fresh `HOME` also creates zero files. The real gap
— that nothing had ever written to `%AppData%` — was closed with `-import-prefs` instead, above.

**`-wav` output is not reproducible, and that is the mixer's clock, not the platform.** The Windows
and Linux paced WAVs differ in roughly 124,000 byte positions, which reads like a regression until
you run Linux against itself: two Linux runs of the *identical* command differ from each other in
roughly 122,000. Same magnitude — and neither count is repeatable, which is the point.

The mixer is clocked by wall time rather than by frames, so an unpaced run produces a capture as
long as the run took in real seconds: 900 frames gave 0.9 s of audio here and 0.5 s on the Linux
host, matching their 951 ms and 491 ms run times. A paced run *is* frame-locked in length — three
600-frame captures came to 887,420, 887,970 and 888,060 bytes — but not sample-aligned. The
game-logic layer above it is deterministic: 16 of 16 sounds and 5 music pieces, identical on both
platforms.

That is worth fixing, and it is filed as [IMPROVEMENTS.md](IMPROVEMENTS.md) 4.11: a frame-locked
mixer clock would make `-wav` a byte-reproducible artifact, and CI could then diff audio across
platforms the way it already diffs pixels.

## What this did not prove

Four gaps, stated plainly because the release notes point here for them.

1. **No key was ever pressed.** Every run was driven by `-frames`, so nobody has steered a glider
   through a house on Windows by hand. `internal/platform/win32/keys.go` — the virtual-key table
   and the `WM_CHAR` accumulator — remains the least-exercised file in the package on the platform
   it exists for. Its pure parts are unit-tested on any host; the wiring that feeds them is not.
2. **`windows/arm64` has never run at all.** Same source, same backend, a different compiler
   target and a different machine. Nothing here transfers to it.
3. **Nobody touched the window.** No resize, no focus change, no drag, no close-button quit, no
   alt-tab. The window was created, painted 4,320 frames across six runs, and exited on its own
   frame count every time.
4. **The `.exe` did not arrive the way a player's will.** It was copied over SSH from the machine
   that built it, so it carried no Mark of the Web and Defender SmartScreen never had an opinion
   about it. A downloaded zip gets both, and what a first-time player meets is therefore one screen
   *earlier* than anything on this page — see the section below, which is the one part of this
   document that can be rehearsed without a download.

Also untested: more than one sound device, a machine with no sound device at all (the code has a
path for it), any non-US keyboard layout, and a full game played through to a high score.

## What has changed since, and has not been seen working on Windows

Three things in `internal/platform/win32` were written after this run, for the release gate's
step 4 (`docs/IMPROVEMENTS.md` 2.1 and 2.76). They build and vet for amd64 and arm64, and no
Windows screen has shown them. CI's service-session bench asks for the centred window at every
push to `main`, and draws into it with the new present if the window opens. Neither is checked:
the step is best-effort, its log has not been read, and `Present` ignores what `StretchDIBits`
returns. A `-frames` run never calls `Room`.

- **`Room`**: the work area of the monitor under the pointer, less the frame.
- **The placement**: the window is centred in that work area instead of `CW_USEDEFAULT`.
- **The changed-rows present**: one `StretchDIBits` per changed block, from a DIB header whose bits
  pointer is the block's first row. It also includes the `IsIconic` skip and a whole frame after
  `WM_PAINT`.

Step 4 also added two things outside the backend (`docs/IMPROVEMENTS.md` 4.35), which have not run
on Windows either:

- **The crash file**: `%AppData%\glidergo\crash.log`, kept as `crash-last.log` by the start after
  a crash. `runtime/debug.SetCrashOutput` writes it, and the rename falls back to a copy when
  another copy of the game has the file open.
- **The console hold**: `holdConsole` in `cmd/glidergo/console_windows.go`. If
  `GetConsoleProcessList` finds only this process on the console, the game stopped in a window
  that it opened itself, so it waits for Enter before that window closes.

The next run on a Windows desktop checks them, in this order:

1. **Auto.** With `-prefs none`, or `-prefs` naming a file that does not exist yet, and no
   `-scale`, the console's banner says `scale=N (auto)`. N is the largest that fits the monitor,
   at most 3. The window is centred and clear of the taskbar. On a second monitor, launched with
   the pointer on it, the window opens there.
2. **The pixels, at 2×.** This is the paced run and screenshot comparison above, with `-scale 2`.
   Halve the crop by nearest neighbour and compare it with the Linux frames, as above. A block sent
   to the wrong row or column shows as a strip of the previous frame. Minimise and restore the
   window during the run: it must come back whole, which is the `IsIconic` skip and `WM_PAINT`
   together.
3. **The bench rows**, which are what lifts the 3× cap:

   ```
   glidergo.exe -frames 300 -bench -scale 1 -prefs none -scores none -saves none
   glidergo.exe -frames 300 -bench -scale 4 -prefs none -scores none -saves none
   glidergo.exe -frames 150 -scale 4 -prefs none -scores none -saves none
   ```

   2.76's budget is paced 4× under 15% of one core, which is the last row's CPU line, and flat-out
   4× at 120 fps or more. A run with `-frames` never asks the monitor its size, so a monitor too
   small for 4× gives no warning (`docs/IMPROVEMENTS.md` 4.47). It measures a window partly off
   the screen, which is not a result, so check by eye that the whole 4× window, title bar
   included, is on the screen. If both rows meet the budget, `autoMax` in `cmd/glidergo/scale.go`
   becomes 4, in a release after the rc that measured it, because that rc is the build being
   checked.
4. **The crash file and the console hold.** Make a shortcut to `glidergo.exe` whose target ends in
   `-house Nowhere`, and double-click it. The console must print the error and then "press Enter
   to close this window", and wait. Run the same line from PowerShell: it must print the error
   and exit, with no wait. `%AppData%\glidergo\crash.log` then holds the `-version` block and a
   `glidergo:` line under the rule, and the next start says nothing about a crash. The shipped
   game has no way to crash on purpose, so the crash itself comes from the test binary:

   ```
   GOOS=windows go test -c -o crashtest.exe ./cmd/glidergo      (on Linux)
   crashtest.exe -test.run "Crash|Band" -test.v                 (on Windows)
   ```

   That makes the runtime panic in a child process on Windows, and then checks its report was
   kept. The band test checks the `%AppData%` form of the path.

## Repeating it

The six render hashes are the cheap half and need no Windows desktop — a CI runner will do:

```
glidergo.exe -shot shot-splash.png   -shot-screen splash   -prefs none -scores none -saves none
glidergo.exe -shot shot-houses.png   -shot-screen houses   -prefs none -scores none -saves none
glidergo.exe -shot shot-settings.png -shot-screen settings -prefs none -scores none -saves none
glidergo.exe -shot shot-about.png    -shot-screen about    -prefs none -scores none -saves none
glidergo.exe -shot shot-credits.png  -shot-screen credits  -prefs none -scores none -saves none
glidergo.exe -shot shot-scores.png   -shot-screen scores   -prefs none -scores none -saves none
```

Compare the six SHA-256s against the table above, or against a Linux build of the same tag.
`-prefs none -scores none` is not optional for a hash comparison: two of those six screens draw
the current settings and the current high scores, so an existing configuration changes the pixels
legitimately. The hashes in the table are this build's defaults.

The expensive half needs a logged-in desktop, because that is the whole point of it — a Windows
*service* session, which is what CI gives you, has no visible desktop and a screenshot of it
proves nothing. On such a desktop:

```
glidergo.exe -house Slumberland -frames 600 -prefs none -scores none -saves none -wav paced.wav
```

Take a screenshot while it runs, crop the window's client area, and compare it against the frames
a Linux build of the same tag writes:

```
make headless      # -dump needs the nullbackend build, bin/glidergo-null
bin/glidergo-null -house Slumberland -frames 60 -dump frames/ -prefs none -scores none -saves none
```

Expect an exact match against *some* frame, not against a particular one: once the scene settles it
alternates between two frames, so a capture matches every frame of its own parity rather than
exactly one.

The two flags that matter there: `-prefs none -scores none -saves none` so an existing
configuration cannot change what is drawn, and **paced rather than `-bench`**, because the window
has to stay up long enough to photograph.

## Rehearsing what a download adds, with no download

Gap 4 above is the one gap on this page that an airgapped machine can close, because a browser
download is not a special kind of file — it is an ordinary file with one alternate data stream on
it. Write the stream by hand and Windows cannot tell the difference:

```
Set-Content -Path .\glidergo.exe -Stream Zone.Identifier -Value "[ZoneTransfer]`nZoneId=3"
Get-Item -Path .\glidergo.exe -Stream Zone.Identifier
```

`ZoneId=3` is the internet zone, which is what a browser writes. Then **double-click it in
Explorer**, which is the path that matters: SmartScreen's prompt comes from the shell's attachment
check and not from `CreateProcess`, so launching the same file from PowerShell is not the same test
and may well not prompt at all. What to record: whether the dialog appears, its exact wording, and
whether *Run anyway* is behind *More info* as
[`.github/workflows/release.yml`](../.github/workflows/release.yml)'s release notes tell a player it
is. `Unblock-File .\glidergo.exe` removes the stream again, so the test can be repeated.

Worth doing before every tag, because those notes describe this from documentation rather than
from having watched it (`docs/IMPROVEMENTS.md` 4.13). `v0.2.0` is tagged without it (`docs/PLAN.md`
§4, step 6), so for that tag it is the first work after it. It is also the only caveat in that file
which does not need github.com to check.
