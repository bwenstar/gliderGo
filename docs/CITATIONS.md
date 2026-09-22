# Citations

gliderGo is a transcription, and a citation is how it says so. There are about **17,800** of them,
each naming a file and a line of John Calhoun's 1994 C:

```
GliderPRO/Sources/Player.c:1234
```

This document is about those strings: what they mean, how to check one, what the automated check
does and does not promise, and what it found the first time it ran.

---

## 1. Why there are so many

The port's central claim is that the game behaves like the 1994 build. That claim is not testable in
general — nobody has the original running beside it, frame for frame, on every input — so it is made
in pieces, function by function, and each piece is backed by a pointer at the C it was read from.

Which means a citation is doing real work. It is the difference between

> The glider's vertical velocity moves one impulse per tick toward a target, and that target is
> reset to gravity on every tick.

and

> The glider's vertical velocity moves one impulse per tick toward a target, and that target is
> reset to gravity on every tick (`GliderPRO/Sources/Player.c:80-92`).

The first is a thing the port does. The second is a thing the port does *because the original did*,
and a reader can go and disagree. Without the pointer they would have to take our word for it, and
taking our word for it is precisely the position the citations exist to get them out of.

A consequence worth stating plainly: **a citation that does not resolve is worse than no citation at
all.** It looks like evidence, costs nothing to write, and spends somebody else's afternoon.

## 2. The forms

| Form | Example | When |
|---|---|---|
| Full path | `GliderPRO/Sources/Player.c:1234` | First mention in a document, and in any passage somebody might paste out of context. |
| Short | `Player.c:1234` | Once the file is established in the surrounding prose. |
| Range | `Room.c:1103-1206` | A whole function or block. |
| List | `Play.c:735, 736, 739` | Scattered lines that make one point. |
| Bare name | `Interactions.c` | A claim about a file rather than about a line. |

Two conventions are load-bearing:

- **The colon is required.** `Banner.c:205` is a citation; "`Banner.c` 300 times" is prose about a
  file. Without that rule the checker in §4 would read the second as a claim about line 300 of a
  237-line file and fail on writing that was correct.
- **Line numbers are into the LF-normalised copy.** See §3 — this is not a detail.

## 3. Making them resolve

The 1994 C is **not in this repository.** It is Calhoun's GPLv2 source and this project ships only
the data its extractors decode out of the resource forks, not his code. So the citations are pinned
to a commit rather than to a copy:

```bash
git clone https://github.com/softdorothy/glider_pro /tmp/glider_pro
git -C /tmp/glider_pro checkout 94fed96e0b4c810a6ac861e5d4b14d625a5a1c31
cp -r /tmp/glider_pro/Sources /tmp/glider_pro/Headers /tmp/glider_pro/Prefix.h GliderPRO/
```

All three paths are gitignored, so doing this leaves your tree clean. `Prefix.h` is named separately
because it sits at the top of upstream's tree rather than inside `Headers/` — which is how it came
to be left out of this recipe for four stages while four citations pointed at it.

**The trap.** `Sources/*.c` and `Headers/*.h` are classic Mac text with **CR-only** line endings.
`wc -l` answers `0`, most editors show one enormous line, and `grep` decides the file is binary and
prints nothing at all — so a pattern that is definitely there appears to be absent, silently. Every
line number in this repository is into the converted copy:

```bash
mkdir -p /tmp/gp
for f in GliderPRO/Sources/*.c GliderPRO/Headers/*.h; do
  tr '\r' '\n' < "$f" > "/tmp/gp/$(basename "$f")"
done
sed -n '1103,1206p' /tmp/gp/Room.c        # and the citation resolves
```

For reading the originals in place, `grep -a` defeats the binary-file guess.

## 4. The check

```bash
go test ./internal/citations/
```

`internal/citations/citations_test.go` sweeps every `.go`, `.md`, `.py`, `.sh`, `.txt` and `.yml`
file in the repository and resolves what it finds. Three classes of claim, and every one of them is
a promise this project made in prose and never checked until the file existed:

**Class 1 — the 1994 C.** Four tiers, in order of how convincing a wrong one looks:

1. **The file exists.** `Environs.c` is not `Environ.c`; `Sounds.c` is not `Sound.c`. Near-misses
   of real names are the errors a reader forgives and a `grep` does not survive.
2. **The line exists.** Every number, every range endpoint, against the real line count.
3. **A full path names the directory its file is actually in,** and every range runs forwards. A
   path that says `Sources/` for a header that lives in `Headers/` reads perfectly, resolves to
   nothing, and passes both tiers above, because the file name is real and the line is in range.
4. **Coverage, both ways.** Every one of upstream's 67 `.c` files is cited somewhere — a source file
   nothing cites is a subsystem nobody has accounted for. And every entry on the exemption list is
   still earned, because an exemption nothing needs is a name the next person is free to get wrong.

**Class 2 — this repository's own paths.** `internal/render/locale.go:97` is the same kind of
promise, about a tree where files actually do get renamed. With a coverage half, for the same reason
tier 4 above has one: every `internal/` package must appear on the map `README.md`'s "What is in
here" draws. A package missing from that table breaks no link and the table still reads as complete,
so the only way a reader finds out is by concluding the thing they were looking for is not in this
project. It found two, and they were the worst available — `internal/shell`, which is everything a
player sees before a room is composed, and `internal/audio`, which the README discusses at length
three sections higher up without ever saying where it is.

**Class 3 — the test names the prose quotes.** "`TestFooBar` holds this true" is a citation, and the
one most likely to go stale: renaming a test is a thing people do without reading the comments that
named it.

And one invariant the rest lean on: **every place the upstream commit is written names the same
commit.** The revision appears in a dozen-odd places — `internal/project`, the documents, two GitHub
templates, `.gitignore`, `.gitattributes` — about half of them as a pasteable `git checkout`.
Nothing held them together, and the failure that matters is not a typo but a *bump*: repointing at a
newer upstream means re-reading 18,000 line numbers, so it is exactly the change that gets made to
the README and the constant and then stopped for the evening.

### What it measures

Roughly 17,800 citations naming roughly 18,300 lines and ranges, against 93 files of the pinned C;
roughly 1,050 references to about 160 of our own files; every `internal/` package against the
README's map; and every written mention of the pin, all in agreement.

Those are rounded on purpose. The exact figures move every time anybody writes a paragraph —
including this one — so the test reports them rather than asserting them, and the only number it
holds is a deliberately loose floor. What the floor is for is the failure it could not otherwise
see: an edit that breaks the matching turns 18,000 assertions into a vacuous pass over an empty
slice, and the suite stays green. That failure does not arrive at 17,000. It arrives at nought.

For the current numbers:

```bash
go test -v ./internal/citations/ | grep -E 'checked|in agreement'
```

### When the C is absent

Classes 1's tiers skip, with the three commands from §3 printed in the skip message. Classes 2 and 3
and the pin check need nothing and always run.

A skip is a weak thing to rely on, so `.github/workflows/ci.yml` has a `citations` job that clones
upstream at the pin on every push — and that job **fails if any check skipped**, because `go test`
prints `ok` for a package whose every test skipped, and a mistyped path would otherwise leave it
green while proving nothing.

## 5. What it found

Twenty-two defects, on a body of prose that had been reviewed repeatedly. The wrong values below are
written without their colons and without their directories on purpose: in the forms §2 defines they
would be citations, and the checker sweeps this file like any other.

| Kind | Count | Examples |
|---|---|---|
| Line past the end of the file | 9 | `RectUtils.c` line 322 in a 318-line file; `Room.c` lines 1172–1215 in a 1206-line one; `Banner.c` lines 205–243 in a 237-line one |
| A file that does not exist | 6 | `Environs.c` → `Environ.c`; `Sounds.c` → `Sound.c`; `StructuresInit1.c` → `StructuresInit.c`; `MenuBar.c` → `Menu.c:549` |
| The wrong directory | 1 | `Marquee.h` cited under `Sources/`, where only `Marquee.c` is |
| A document of ours that is not there | 2 | `sound.md`, which is `audio.md` |
| A test name that no longer exists | 4 | `TestEveryKeyHasAName`, renamed to `TestEveryKeyHasANameAndRoundTrips` |

Plus one that was not a citation defect but caused four: `Prefix.h` was missing from the copy recipe
in every document, so the four citations to it could not resolve on any machine that had followed
the instructions.

**Every out-of-range number was fixed by reading the C and finding the line the claim was about, not
by lowering the number until it fit.** `Banner.c` 205–243 became `Banner.c:205-236` because that is
where `DisplayStarsRemaining` ends; `RectUtils.c` 322 became `RectUtils.c:210-216` because that is
where `QSetRect` is. A citation that resolves to the wrong place is worse than one that resolves to
nothing, because nothing announces itself and the wrong place does not.

Two of the twenty-two are worth singling out, because of what they imply rather than what they cost.
`Room.c` cited to 1215 is a 1206-line file cited nine lines past its end, which means that reading
was done against a differently converted copy — so the *other* numbers from it were suspect too. And
`PlayerControl.c` was a file an early draft invented, which has never existed in any version of
Glider PRO. It is now named in exactly one place, in `docs/analysis/determinism.md`, to say so.

Four of the names in the table above are on the exemption lists in §4's tier 4 for exactly this
reason — they are quoted here and in `docs/IMPROVEMENTS.md` as errors that *were* found, and
correcting the quotations would delete the findings. That is what an exemption with a reason is for.

## 6. What it does not promise

Worth being blunt about, because a green check invites more confidence than it has earned:

- **That a citation is about the right thing.** `Player.c:1234` resolves, and line 1234 is
  `FlagGliderNormal(thisGlider);` — whether that is what the sentence beside it claims, no checker
  can say. It verifies that a pointer lands inside the file; only a reader can verify that it lands
  on the argument. This is the big one.
- **Line numbers on bare names.** `Interactions.c` is checked for existence and nothing else.
- **The space-separated form.** Deliberately not matched — see §2.
- **`docs/analysis/stage15-raw/`.** Three Go files and four documents reference this directory,
  which is gitignored because those drafts still carry errors the review pass corrected. The
  references are dead on every clone. This is a known defect, recorded in `docs/IMPROVEMENTS.md`,
  deferred because the "releasePolish N" numbering they quote exists only there and so they cannot
  simply be repointed — and now the exemption is machine-tracked, so the day that numbering lands in
  a committed file the checker asks for the exemption back.
- **Upstream headers that are cited but not shipped.** `Sound.h`, `QuickDraw.h`, `WinUser.h`,
  `Xlib.h` and fifteen others are Apple, Microsoft and X11 SDK headers, named where a document
  explains which system call the C is making. They are on an exemption list with a reason each; the
  reasons are checked for being *needed*, not for being true.
- **Two upstream headers nothing cites.** `About.h` (10 lines) and `Play.h` (13 lines) are
  prototype-only companions to two of the most heavily cited `.c` files in the tree, with no
  `typedef`, `#define` or `struct` between them. Coverage is asserted for the 67 sources and not for
  the 25 headers, for this reason.

## 7. Adding one

Write the file, a colon and the line — in the LF-normalised numbering. Then:

```bash
go test ./internal/citations/
```

If it fails, read the C. The message names the file, the line your citation is on, the number you
wrote and the number of lines the file has; that is usually enough to see whether the reading was
off by a conversion or the number was off by a digit.

If the name is genuinely not one of upstream's — an SDK header, a placeholder in an explanation of
the convention — add it to `notInTheOriginal` **with the reason**. An exemption without a reason is
indistinguishable from an error somebody silenced, and that list is the one part of the checker that
cannot fail on its own.

---

**See also:** [ORIGINAL_GAME.md](ORIGINAL_GAME.md) for what the original does,
[DEV_ENVIRONMENT.md](DEV_ENVIRONMENT.md) for the rest of the setup,
[../CONTRIBUTING.md](../CONTRIBUTING.md) for when a change needs a citation in the first place.
