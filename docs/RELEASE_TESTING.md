# Checked by a person

`make check` and the release workflow test everything a program can test: houses finish, frames
hash, two games race over loopback, and the Windows build compiles and renders. This is the list
of what they cannot test, because each step needs a person: someone to play a house, press keys,
move a window or answer a dialog from Windows. `docs/PLAN.md` §5 names this list as its own row,
and §4's step 6 makes it part of the release gate.

`RELEASING.md` runs it as step 8 of "Before the tag", on each rc. Use the rc's own archives,
downloaded from its prerelease page or its workflow run, and not a local build.
A local build is the wrong binary and carries no Mark of the Web. Anything a step finds goes into
the release notes' or the README's wording before the tag (`docs/IMPROVEMENTS.md` 4.33), or into
`docs/IMPROVEMENTS.md` as an item.

Every step has a **Do**, an **Expect** and a row in "Last run" at the end. "not run" is a result,
and so is "no machine for it". A blank is not a result. When a step is run, replace its row. The
history is in `git log -p` on this file.

**Before you start.** Move your own settings, scores and saved games aside, because several steps
need a first start. `glidergo -version` prints where they are. Rename the directories it names for
the run, and rename them back afterwards.

Two checks by hand are not here, because `RELEASING.md` records them itself: Mark of the Web
(step 6) and Defender (step 7).

## On any machine

1. **Open House, played through.**
   *Do:* on the title screen, `L`, Tab to the New set, and play Open House from its first room.
   Do not read `levels/Open House.house.txt` first, because that is the map.
   *Expect:* the star in the Belfry is taken and the game ends with the win screen. Record how many
   gliders it cost, and any room that felt unfair or that you could not work out.
   `TestOpenHouseCanBeFinished` proves only that a route exists. This step asks whether a person
   finds it, and it is the clause `docs/PLAN.md` Stage 2 records as "nobody has played either".

2. **Boarding House, played through.**
   *Do:* the same, with Boarding House.
   *Expect:* the bell in the turret, with all three stars taken. Record what step 1 records.

## On Windows, from the rc's zip

Extract the `windows-amd64` zip in Explorer and start `glidergo.exe` by double-clicking it, unless
the step says otherwise. The console window that opens with it is where the game prints.

3. **The first start.**
   *Do:* check 1 of `docs/windows-first-run.md`, "What has changed since", with the settings moved
   aside. Start a game, because the banner is printed when a game starts.
   *Expect:* what that check says. The banner reads `scale=N (auto)`, with N at most 3, and the
   window is centred and clear of the taskbar. With a second monitor, the window opens on the one
   under the pointer.

4. **The pixels at 2×.**
   *Do:* check 2 of the same list.
   *Expect:* what it says. The frames match Linux, and a minimised window comes back whole.

5. **The bench rows.**
   *Do:* check 3 of the same list.
   *Expect:* three results to paste into `docs/IMPROVEMENTS.md` 2.76. Whether they lift the 3× cap
   is decided there, not here.

6. **The crash file and the console hold.**
   *Do:* check 4 of the same list.
   *Expect:* what it says. A shortcut that fails waits for Enter. The same line typed in PowerShell
   does not wait. The next start keeps the test binary's panic as `crash-last.log`, not an error.

7. **The keys.**
   *Do:* `N` for a one-player game, in any house. Steer with `←` and `→`. If you pick up rubber
   bands or a battery, use them with `↑` or `↓`, and hold the key down. Press Tab, then Tab again.
   Press Esc, then `Q`, and read the question before answering `N`. Then press `2` on the title
   screen for a two-player game, and steer player two with `A` and `D`.
   *Expect:* the glider answers `←` and `→` at once. A held `↑` throws one band, not a stream. Tab
   pauses and resumes. Esc pauses as well, and `Q` from the pause asks before it ends the game: Y
   saves first, N does not, and Tab goes on playing. Player two answers to `A`, `D`, `S` and `W`,
   and neither player's keys move the other glider.

8. **Away and back.**
   *Do:* in a game, hold `→` and press Alt+Tab to another window while you are still holding it.
   Let go of both keys, wait a few seconds, and click back into the game.
   *Expect:* the game stood still while it was in the background, because pausing then is the
   default. It carries on by itself when it has the focus again, and the glider is not still
   steering right. The window forgets a held key when it loses the focus, so a key released in
   another window cannot stay down.

9. **The window's size.**
   *Do:* try to drag an edge or a corner of the window. Look for a maximise button. Press Win+Up.
   Minimise the window and restore it.
   *Expect:* the size cannot change. There is no maximise button, and Win+Up does nothing. The
   window has one size per scale, and `-scale` is how to change it. After a restore it comes back
   whole, with no strip of an old frame.

10. **Closing the window.**
    *Do:* close it with its × three times: on the title screen, in a game, and in the high-score
    name box with a few letters typed. Open House's board ships empty, so any game there that
    scores anything and then ends, with the last glider lost or the house finished, asks for a
    name.
    *Expect:* each time the game exits at once, and the console window closes with it, because the
    game waits for Enter only after an error. After the third, the next start shows the letters
    you had typed on Open House's board.

11. **A name typed on another keyboard layout.**
    *Do:* add French (AZERTY) in Windows' language settings and switch to it with Win+Space. Get
    to Open House's name box as in step 10. Type `é`, `ç` and `à`, then `ê` with the `^` dead key
    and `e`. If a Russian layout is installed, type one Cyrillic letter as well.
    *Expect:* the four letters appear as typed, and the name keeps them on the board. A Cyrillic
    letter adds nothing, because the 1994 board holds only characters Mac Roman has, and the game
    refuses it without saying so, which is known. On AZERTY, player two's `A`, `D`, `S` and `W`
    follow what the keys type, so they are no longer a cluster. That is known too
    (`docs/IMPROVEMENTS.md` 2.74).

## Two machines, with Windows hosting

Windows has to be the host for these, because its firewall asks only when a program listens.
Hosting on Linux and joining from Windows never shows the dialog. The other machine can be anything
this release runs on, on a network that carries TCP port 1994 between the two.

12. **Allowed.**
    *Do:* on Windows, **Race...** on the title screen and host. When Windows asks, compare its
    dialog with the release notes' "Racing another machine" and the README's "When the other
    machine cannot be reached". Tick the kind of network you are on and click **Allow access**.
    Join from the other machine with the address the waiting screen reads out, and race until both
    runs end.
    *Expect:* the dialog is Windows Defender Firewall's, with a tick box for each kind of network
    and the two buttons the notes name. Record its exact wording, because the notes, the README
    and the Windows `HOW-TO-RUN.txt` describe it. The race starts, the panel in the top-left
    corner shows where the other player is, and both machines show the same result. If no dialog
    appears at all, check that the firewall is on for this network before recording anything,
    because a firewall that is off never asks.

13. **Refused, then undone.**
    *Do:* on Windows, open `wf.msc` and delete the inbound rules for glidergo, so that Windows asks
    again. Host, and click **Cancel** this time. Join from the other machine a few times. Then undo
    the no the way the README says: Windows Security, *Firewall & network protection*, *Allow an
    app through firewall*. Join once more.
    *Expect:* the joining screen reports no answer, and after a few tries it names a firewall.
    After the undo, the join connects. Record whether the README's path to the setting is still
    what Windows calls it.

## Not in this list yet

- **`windows-arm64`.** No ARM64 Windows machine has been found to run it. Its row says so until one
  is.
- **A house-feedback issue template.** There is none: `.github/ISSUE_TEMPLATE/` holds a bug report
  and a fidelity-difference form. When one is added, a step here files a test report through it.
- **macOS**, which has no windowed backend until stage 6.

## Last run

`v0.2.0` went without this list. No Windows desktop was to hand, and on 2026-09-29 the owner moved
it out of that tag's gate (`docs/PLAN.md` §4, step 6), so nobody has run a step below on any build.

| Step | Build | Date | Result |
|---|---|---|---|
| 1 Open House | `v0.2.0-rc1` | 2026-09-29 | not run |
| 2 Boarding House | `v0.2.0-rc1` | 2026-09-29 | not run |
| 3 The first start | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 4 The pixels at 2× | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 5 The bench rows | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 6 The crash file and the console hold | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 7 The keys | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 8 Away and back | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 9 The window's size | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 10 Closing the window | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 11 A name on another layout | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 12 A race, allowed | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| 13 A race, refused and undone | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
| `windows-arm64` | `v0.2.0-rc1` | 2026-09-29 | no machine for it |
