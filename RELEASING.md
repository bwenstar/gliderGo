# Releasing gliderGo

`.github/workflows/release.yml` runs on any `v*` tag. It does the tests, the builds, the archives,
the checksums and the GitHub Release. This file covers the rest: the steps a tag needs that the
workflow cannot do. Most of them happen on another machine, because the one gliderGo is written on
cannot reach GitHub. Each step says where it runs.

The commands here are fenced without a language, so `make docs-check` does not try to run them.

## Before the tag

1. **Fetch the tags.** In a clone that never fetched them, `git tag -l` is empty. A local build
   is then stamped with a bare hash, and the CHANGELOG cannot be split by tag.

   ```
   git fetch --tags origin
   ```

2. **A CHANGELOG section for the tag.** Move what the tag carries out of `Unreleased` into its own
   section. If the tag carries a Go security fix, say so. SECURITY.md's "Versions" makes that
   reason enough for a release on its own.

3. **The release gate in `docs/PLAN.md` §4.** Each **Gate** line is either done or moved out of
   the gate on purpose. That includes the first Windows run of the code written since the first
   one, all of which `docs/windows-first-run.md` lists under "What has changed since":
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
   git tag v0.1.2-rc1 && git push origin v0.1.2-rc1
   ```

5. **govulncheck is clean on `GO_RELEASE`**, under `GOOS=linux`, `GOOS=windows` and
   `GOOS=darwin`. The workflow's `verify` job runs it and stops the tag on a finding, so this step
   is reading that log (`docs/IMPROVEMENTS.md` 5.11).

6. **Mark of the Web, written by hand, on Windows.** Any Windows machine will do, connected or not.
   Write the stream a browser writes onto the rc's `glidergo.exe`, then double-click the file in
   Explorer. Starting it from PowerShell is not the same test. Record two things: whether "Windows
   protected your PC" appears, and whether **Run anyway** is behind **More info**, as the release
   notes say. `docs/windows-first-run.md`, "Rehearsing what a download adds, with no download", has
   the detail.

   ```
   Set-Content -Path .\glidergo.exe -Stream Zone.Identifier -Value "[ZoneTransfer]`nZoneId=3"
   Get-Item -Path .\glidergo.exe -Stream Zone.Identifier
   ```

7. **Defender, on a connected machine.** A machine with no network cannot see Defender's cloud
   verdict, and that is the verdict a download meets (`docs/IMPROVEMENTS.md` 5.12).
   - Look up both `.exe` hashes on VirusTotal, and upload them if nobody has. Microsoft's line is
     the one that matters. Record a hit from one of the small engines that flag most new Go
     binaries, but do not chase it.
   - Turn on real-time and cloud protection on a Windows machine. `Get-MpComputerStatus` shows
     `RealTimeProtectionEnabled`, and `Get-MpPreference` shows a non-zero `MAPSReporting`. Download
     the rc zip in Edge, extract it in Explorer and double-click the exe. Expect the SmartScreen
     dialog, and nothing from Defender.
   - If Microsoft flags either file, submit it at
     `https://www.microsoft.com/en-us/wdsi/filesubmission` as a software developer, and mark it
     incorrectly detected. Give the release URL, the tag, the zip's `SHA256SUMS` line and the
     flagged file's own SHA-256 (`Get-FileHash`). `SHA256SUMS` lists the archives, not the `.exe`
     inside each one. Record the submission ID and the date it cleared under "Record" below. A clearance covers one file,
     and every tag builds new ones, so this can be needed at every tag.

Then tag:

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

4. **A real download on Windows.** Download the zip through a browser on a connected machine, so
   that the browser writes Mark of the Web itself. A file copied over `scp` carries no stream,
   which is why step 6 above writes one by hand.

5. **Record the run URL** under "Record".

## Still open, and each needs a connected machine once

- `scripts/bootstrap-dev-env.sh --source public` (or `--dry-run`) has never run, because CI gets
  its Go from `setup-go` (`docs/IMPROVEMENTS.md` 5.1, check 2).
- Paste the bench line from a CI run's "On-screen bench under Xvfb" step into 5.1, check 3.
- README has no word yet about Aerofoil, the other Glider PRO port. Every fact that paragraph
  would state is to be checked against Aerofoil's own pages first (`docs/IMPROVEMENTS.md` 5.13).

## Record

| Tag | Run | Needed a correction? | Built with | Defender / VirusTotal |
|---|---|---|---|---|
| `v0.1.0` | not recorded here | not recorded here | Go 1.23; `docs/IMPROVEMENTS.md` 5.11 has the vulnerabilities that carries | not checked |
| `v0.1.1` | not recorded here | not recorded here | Go 1.23, as above | not checked |
