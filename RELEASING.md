# Releasing gliderGo

`.github/workflows/release.yml` runs on any `v*` tag. It does the tests, the builds, the archives,
the checksums and the GitHub Release. This file covers the rest: the steps a tag needs that the
workflow cannot do. Most of them happen on another machine, because the one gliderGo is written on
cannot reach GitHub. Each step says where it runs.

No test runs the commands in this file: `make docs-check` reads only README.md and CONTRIBUTING.md.

## Before the tag

1. **Fetch the tags.** In a clone that never fetched them, `git tag -l` is empty. A local build
   is then stamped with a bare hash, and the CHANGELOG cannot be split by tag.

   ```
   git fetch --tags origin
   ```

2. **The number, and a CHANGELOG section for it.** A patch release, 0.2.1 after 0.2.0, carries
   bug fixes only, and every value pinned in `cmd/glidergo/promises_test.go` must be the one the
   last tag had. A new feature, or any pinned value that has changed, makes the tag a minor
   release, 0.3.0 after 0.2.x, and its section says what it no longer races or reads
   (`docs/PLAN.md`, "what a version number promises"). A diff to that file that touches only
   comments does not count. The file starts at `v0.2.0`, so the first tag this applies to is the
   one after it.

   ```
   git diff "$(git describe --tags --abbrev=0)" -- cmd/glidergo/promises_test.go
   ```

   Then give the tag its section in `CHANGELOG.md`. After a tag, the next entry opens a section at
   the top, `## Unreleased`, whose first line is "Not tagged yet." on its own, and this step
   renames that heading to the tag's number. `v0.2.0`'s is already named. The first line stays
   through the rc, whose archives carry the file, and "Then tag" below dates it.

   If the tag carries a Go security fix, say so. SECURITY.md's "Versions" makes that reason enough
   for a release on its own.

3. **The release gate in `docs/PLAN.md` §4.** Each **Gate** line is either done or moved out of
   the gate on purpose. That includes the first run on a Windows desktop of the code written
   since the first one, all of which `docs/windows-first-run.md` lists under "What has changed
   since", and which step 8 runs:
   - auto scale;
   - the centred window;
   - the changed-rows present, read back at 2×;
   - the three bench rows that decide 2.76's cap;
   - the crash file and the console hold.

4. **A rehearsal build.** Do one of these:
   - Push an rc tag. It publishes as a prerelease, and it can be deleted.
   - Run the workflow by hand (Actions → Release → Run workflow). That uploads the archives as a
     run artifact and cannot publish anything.

   ```
   git tag vX.Y.Z-rc1 && git push origin vX.Y.Z-rc1
   ```

5. **govulncheck is clean on `GO_RELEASE`**, under `GOOS=linux`, `GOOS=windows` and
   `GOOS=darwin`. The workflow's `verify` job runs it and stops the tag on a finding, so this step
   is reading that log (`docs/IMPROVEMENTS.md` 5.11).

6. **Mark of the Web, written by hand, on Windows.** Any Windows machine will do, connected or not.
   Write the stream a browser writes onto the rc's `glidergo.exe`, then double-click the file in
   Explorer. Starting it from PowerShell is not the same test. Record two things under "Record"
   below: whether "Windows protected your PC" appears, and whether **Run anyway** is behind **More
   info**, as the release notes say. `docs/windows-first-run.md`, "Rehearsing what a download
   adds, with no download", has the detail.

   ```
   Set-Content -Path .\glidergo.exe -Stream Zone.Identifier -Value "[ZoneTransfer]`nZoneId=3"
   Get-Item -Path .\glidergo.exe -Stream Zone.Identifier
   ```

7. **Defender, on a connected machine.** A machine with no network cannot see Defender's cloud
   verdict, and that is the verdict a download meets (`docs/IMPROVEMENTS.md` 5.12).
   - Look up all four `.exe` hashes on VirusTotal, `glidergo.exe` and `glidertool.exe` from each
     Windows zip, and upload any that nobody has. Microsoft's line is the one that matters.
     Record a hit from one of the small engines that flag most new Go binaries, but do not chase
     it.
   - Turn on real-time and cloud protection on a Windows machine. `Get-MpComputerStatus` shows
     `RealTimeProtectionEnabled`, and `Get-MpPreference` shows a non-zero `MAPSReporting`. Download
     the rc zip in Edge, extract it in Explorer and double-click the exe. Expect the SmartScreen
     dialog, and nothing from Defender.
   - If Microsoft flags any of them, submit it at
     `https://www.microsoft.com/en-us/wdsi/filesubmission` as a software developer, and mark it
     incorrectly detected. Give the release URL, the tag, the zip's `SHA256SUMS` line and the
     flagged file's own SHA-256 (`Get-FileHash`). `SHA256SUMS` lists the archives, not the `.exe`
     inside each one. Record the submission ID and the date it cleared under "Record" below. A
     clearance covers one file, and every tag builds new ones, so this can be needed at every tag.

8. **The list in `docs/RELEASE_TESTING.md`**, on the rc's archives: two houses played through,
   the Windows window, keys and firewall, and a race with Windows hosting. Fill in its "Last run"
   table. Anything it finds is fixed in the release notes' or the README's wording before the
   tag, or filed. Then make the notes' "What a person checked" agree with the table. That section
   ends by saying `v0.2.0` went without the list, and the next tag says what the table records
   instead. Steps 6 to 8 are left out only when the owner moves them out of the gate in
   `docs/PLAN.md` §4, as for `v0.2.0`.

Then date the section and tag, on the same day. The date is one command, for GNU `sed` as on
Linux, and `git diff --stat` shows it changed one line of `CHANGELOG.md`:

```
sed -i "s/^Not tagged yet\.$/Tagged on $(date +%F)./" CHANGELOG.md
git diff --stat
git commit -am "docs: CHANGELOG.md dates vX.Y.Z"
git push origin main
```

Wait for CI on that commit to go green. If it goes red, do not tag. Then:

```
git tag vX.Y.Z && git push origin vX.Y.Z
```

## After the tag

Each of these **needs a connected machine**.

1. **Download from the Releases page and check the sums.** Run this in the directory the archives
   are in:

   ```
   sha256sum -c --ignore-missing SHA256SUMS
   ```

2. **Run it on a clean system.** Unpack the `linux-amd64` archive on a fresh install of the oldest
   system README's table names (Ubuntu 22.04 or Debian 12), with nothing added.

3. **Read the glibc line in the build log.** The build job's "Hold the Linux archive to the glibc
   floor it states" step prints the glibc the shipped binary needs, and fails the tag above 2.34.
   Nothing to do unless it failed.

4. **A real download on Windows, and step 7 again.** The tag builds new files, with its version
   stamped into them, so the rc's verdicts do not carry over. Repeat step 7 above on the tag's
   own archives: the four VirusTotal lookups, and the `windows-amd64` zip downloaded in Edge,
   which writes Mark of the Web itself. Double-clicking the exe should give what step 6 recorded,
   "Windows protected your PC" with **Run anyway** behind **More info**, and nothing from
   Defender. A file copied over `scp` carries no stream, which is why step 6 writes one by hand.
   For `v0.2.0`, which left out steps 6 to 8, there is nothing recorded to compare with: expect
   what the release notes say, and record this as the first run of steps 6 and 7. Step 8's list
   follows on the same archives, as the first work after that tag (`docs/PLAN.md` §4, step 6).

5. **Fill in the tag's row** under "Record": the Release run's URL, whether the tag needed a
   correction, the Go the release notes name, and what steps 6, 7 and 4 found.

## Still open, and each needs a connected machine once

- `scripts/bootstrap-dev-env.sh --source public` has never run against go.dev, because CI gets
  its Go from `setup-go` (`docs/IMPROVEMENTS.md` 5.1, check 2). `--dry-run` does not close this,
  because it never touches the network. Its first run end to end, against a stand-in for go.dev,
  found that it could not read the index at all, which is fixed. This real run installs into a
  directory of its own, and leaves the Go you have alone:

  ```
  GLIDERGO_TOOLCHAIN_DIR=$(mktemp -d) scripts/bootstrap-dev-env.sh --source public
  ```

- Paste the bench line from a CI run's "On-screen bench under Xvfb" step into 5.1, check 3.
- README has no word yet about Aerofoil, the other Glider PRO port. Every fact that paragraph
  would state is to be checked against Aerofoil's own pages first (`docs/IMPROVEMENTS.md` 5.13).

## Record

| Tag | Run | Needed a correction? | Built with | Mark of the Web / Defender / VirusTotal |
|---|---|---|---|---|
| `v0.1.0` | not recorded here | not recorded here | Go 1.23; `docs/IMPROVEMENTS.md` 5.11 has the vulnerabilities that carries | not checked |
| `v0.1.1` | not recorded here | not recorded here | Go 1.23, as above | not checked |
| `v0.1.2` | not recorded here | not recorded here | Go 1.23, as above | not checked |
