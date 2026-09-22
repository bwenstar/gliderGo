// Command docscheck runs the command lines this project's documents tell a reader to run.
//
//	go run ./tools/docscheck        # what `make docs-check` is the name for
//
// # Why
//
// docs/IMPROVEMENTS.md 4.13 is a sweep over the instructions this repository gives a *person* --
// the commands in the READMEs, the ones the binaries' own `-h` offers -- and every defect it
// found was found by hand, by somebody typing a line into a shell and watching it fail. The
// worst of them had survived six stages: replay's own usage text offered `glidertool replay
// -script -` as the way to see the script format, and that line answered `no house to replay`.
// A documented command line that does not work is the most expensive wrong sentence in a
// repository, because it is the one a newcomer meets in their first five minutes and the only
// evidence they have about whether the rest is worth reading.
//
// So this exists to make `make check` read the documents the way a newcomer does.
//
// # Every line in a bash fence, rather than the three prefixes 4.13 proposed
//
// The bullet asked for "the ones in fenced blocks that begin with `bin/glidergo`,
// `bin/glidertool` or `make`". This takes every non-empty, non-comment line inside a ```bash
// fence instead, because a prefix list is a list that can be silently incomplete: `tr '\r' '\n' <
// GliderPRO/Sources/Player.c` and `go test ./internal/fidelity -update` are documented
// instructions too, a reader who runs them is following the document exactly as written, and a
// recogniser built from three prefixes would neither check them nor say that it had not. The
// fence tag is the marker 4.13 wanted, and it already exists: every ```bash block in these
// documents holds commands and nothing else, and every block of sample output is in an untagged
// fence. That convention was already being kept by hand; this makes it load-bearing.
//
// The price is that every line must be classified, including the ones that cannot run here, and
// the check fails if one is not. That is the point. A new command line in the README is a new
// claim, and the author is the person who knows whether it can be run unattended.
//
// # How a line is classified
//
// The table is keyed by the command text with its trailing `#` comment removed and its internal
// whitespace collapsed, so rewording the comment beside a line -- which these documents do often,
// since the comments are what make the blocks readable -- does not silently unclassify the
// command; editing the command does. What gets executed is the line exactly as written, comment
// and all, because a comment is a thing a shell already knows how to ignore and the check should
// not be running a line the reader cannot see.
//
// Both directions are checked. A line no rule names is a failure, and a rule no line names is
// also a failure: a stale rule is how a check quietly stops covering the thing it was written
// for. Between them they mean the table cannot drift from the documents in either direction
// without somebody being told.
//
// # Where they run
//
// Each line runs in its own `bash -o pipefail -c`, in a scratch directory holding a symlink to
// every top-level entry of the repository except `.git`. Three things follow from that, and all
// three were wanted:
//
//   - Relative paths resolve, so `bin/glidertool house info assets/extracted/houses/*.house`
//     means in the check what it means in the document.
//   - Files a documented line writes land in the scratch directory, so `house build -o
//     my-house.house` cannot leave a stray file in a working tree, and `make check` stays
//     something that can be run on a dirty branch.
//   - `.git` is left out deliberately. A documented `git` line is skipped below for other
//     reasons, but a check with a writable path to the repository's own history is a check that
//     can lose work, and the cost of the exclusion is that `make` computes its version as `dev`.
//
// `pipefail` because the fences say `bash` and `house dump ... | less` is one of the lines: under
// plain `sh` a failure on the left of that pipe is a success. Each line is its own shell, which
// is a claim about the documents -- that no documented line depends on a variable or a `cd` from
// the line before it. The two `cd` lines in the documents are skipped for other reasons; a third
// would need this paragraph revisited rather than quietly working by accident.
//
// A line gets two minutes. That is not a performance limit -- the nineteen that run here take
// about three seconds between them -- it is because one of 4.13's four defects *was* a hang, and
// a hang is the single failure a test suite cannot report on its own. A check that inherits it
// has inherited the worst version of it: `make check` that never ends.
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// docs are the documents whose command lines are checked, in the order a reader meets them.
//
// Two, and hand-kept rather than swept for. These are the documents that make a promise to
// somebody who has not built the project yet, which is 4.13's whole subject. `docs/` holds
// analysis, a plan and an improvements ledger, whose fenced blocks quote the original's C and the
// output of commands as evidence; running those is a different job with a different notion of
// what a pass would mean. If a third front-door document ever appears, adding it here is one
// line -- and docscheck_test.go checks that both of these still exist and still have a ```bash
// fence, so a rename cannot leave this list pointing at nothing.
var docs = []string{"README.md", "CONTRIBUTING.md"}

// perLine is how long a documented command line gets before it is a failure. See the package
// comment: the number is arbitrary, the existence of the deadline is not.
const perLine = 2 * time.Minute

// rule says what to do with one documented command line. The zero value runs it.
type rule struct {
	// skip is why the line is not run, in words, printed under it. A check that quietly does
	// less than it claims is worse than one that does less and says so, which is the argument
	// `make check-caveats` already makes for the rest of the suite.
	skip string
	// needs are paths, relative to the repository root, that the line reads. A missing one skips
	// the line and says which path was missing, rather than failing: `GliderPRO/Sources` is not
	// redistributable and is absent on most machines, and a check that fails there would teach
	// people to ignore it.
	needs []string
	// setup are command lines run first, in the same directory and shell as the line itself,
	// for the documents' habit of naming a file the reader is assumed to have made earlier
	// (`my-house.txt`). Setup is not part of the claim; if it fails, that is this program's bug
	// and it says so.
	setup []string
}

// rules classifies every command line in docs. Keyed by the command text (see command), so one
// entry covers a line that both documents give, which is deliberate: the reason `make run` cannot
// run in a check does not depend on which document said it.
var rules = map[string]rule{
	// Getting started: install, clone, build, play. Nothing here can run in a check, and the
	// reasons are worth reading, because each is a thing this project would be wrong to do
	// during `make check`.
	"sudo apt-get install -y build-essential pkg-config libx11-dev": {
		skip: "installs packages as root; a check that does that to the machine it is checking " +
			"is not a check",
	},
	"git clone https://github.com/bwenstar/gliderGo && cd gliderGo": {
		skip: "clones over the network, which this project does not have (GOPROXY=off, and the " +
			"host it is written on is airgapped)",
	},
	"git clone https://github.com/bwenstar/gliderGo": {
		skip: "clones over the network; the same line the README gives, split across two",
	},
	"cd gliderGo": {
		skip: "names the directory the clone above would have made",
	},
	"make": {
		skip: "builds both binaries, which is what `make docs-check` already depends on, so " +
			"running it here would be checking that make is idempotent",
	},
	"make run": {
		skip: "opens a window and plays until the player quits; -shot and -frames are the " +
			"headless forms, and CONTRIBUTING's -shot line below is checked",
	},
	"make check": {
		skip: "is the target this check runs inside",
	},
	"make doctor": {},
	"make help":   {},

	// README's glidertool block -- the first commands anybody runs against the 1994 data. The
	// house the block invents is `my-house.txt`, which the prose says came from a `house dump`;
	// setup makes it from a house this repository actually carries.
	//
	// The four lines that name a path under assets/extracted/ declare it, and that is not
	// defensive padding: `make clean-assets` is a documented thing to do, the Makefile's guards
	// are careful to let `make check` still pass afterwards, and a docs-check that turned four of
	// those skips into failures would have quietly taken that promise away. The houses named by
	// *title* need nothing -- `replay -house "CD Demo House"` reads the copy inside the binary.
	"bin/glidertool house info assets/extracted/houses/*.house": {
		needs: []string{filepath.Join("assets", "extracted", "houses")},
	},
	`bin/glidertool house dump "assets/extracted/houses/Demo House.house" | less`: {
		needs: []string{filepath.Join("assets", "extracted", "houses", "Demo House.house")},
	},
	"bin/glidertool house build -o my-house.house my-house.txt": {
		setup: []string{`cp "levels/Open House.house.txt" my-house.txt`},
	},
	"bin/glidertool house lint my-house.house":              {},
	"bin/glidertool house checks":                           {},
	"bin/glidertool house stats -tier small my-house.house": {},
	`bin/glidertool render -all -o /tmp/demo "assets/extracted/houses/Demo House.house"`: {
		needs: []string{filepath.Join("assets", "extracted", "houses", "Demo House.house")},
	},
	`bin/glidertool replay -house "CD Demo House" -frames 600 -wav /tmp/run.wav`: {},
	"bin/glidertool types": {},

	// The original source. The three lines that fetch it cannot run; the one that reads it can,
	// on a machine that has it.
	"git clone https://github.com/softdorothy/glider_pro /tmp/glider_pro": {
		skip: "clones over the network",
	},
	"git -C /tmp/glider_pro checkout 94fed96e0b4c810a6ac861e5d4b14d625a5a1c31": {
		skip: "needs the clone above",
	},
	"cp -r /tmp/glider_pro/Sources /tmp/glider_pro/Headers /tmp/glider_pro/Prefix.h GliderPRO/": {
		skip: "needs the clone above, and writes into the repository",
	},
	`tr '\r' '\n' < "GliderPRO/Sources/Player.c" > /tmp/Player.c`: {
		needs: []string{filepath.Join("GliderPRO", "Sources", "Player.c")},
	},

	// CONTRIBUTING: the screenshot form, the fidelity corpus, the authoring walkthrough, the
	// port's own houses.
	"bin/glidergo -shot /tmp/screen.png -shot-screen about": {},
	"go test ./internal/fidelity -update": {
		skip: "rewrites the committed pixel corpus, which is the thing the corpus exists to " +
			"stop happening by accident",
	},
	`bin/glidertool house dump "assets/extracted/houses/Slumberland.house" > slumberland.txt`: {
		needs: []string{filepath.Join("assets", "extracted", "houses", "Slumberland.house")},
	},
	// And the three that read what the dump wrote inherit its condition: without the house there
	// is no slumberland.txt, so they would fail for a reason that is four lines up the page and
	// nothing to do with them. Naming the same file is what makes the skip say so.
	"bin/glidertool house build -o out.house slumberland.txt": {
		needs: []string{filepath.Join("assets", "extracted", "houses", "Slumberland.house")},
	},
	"bin/glidertool house lint out.house": {
		needs: []string{filepath.Join("assets", "extracted", "houses", "Slumberland.house")},
	},
	"bin/glidertool house stats -tier small out.house": {
		needs: []string{filepath.Join("assets", "extracted", "houses", "Slumberland.house")},
	},
	"make levels": {
		skip: "is a target `make check` runs itself, and it writes assets/levels/",
	},
	"make levels-zip": {
		skip: "rewrites assets/levels.zip, which is committed and embedded",
	},
	"bin/glidergo -levels assets/levels": {
		skip: "opens a window and plays until the player quits",
	},

	// README's race block. Neither line can run here, and for once the reason is not the window:
	// each of these waits for the *other* one, on a second machine, and a check that ran one of
	// them would be a check that waits thirty seconds to discover that it is alone. The pair is
	// covered instead by internal/netplay's own tests, which race two peers over a loopback
	// socket, and by hand on two processes -- see docs/PLAN.md Stage 3.
	"bin/glidergo -host Slumberland": {
		skip: "waits for a second machine to join, then opens a window; the race is tested over " +
			"a loopback socket in internal/netplay instead",
	},
	"bin/glidergo -join 192.168.1.20 Slumberland": {
		skip: "names an address on somebody else's LAN, which is the one thing in these documents " +
			"that cannot be true on the machine reading them",
	},
}

func main() {
	if len(os.Args) > 1 {
		// Refused rather than ignored, which is 4.13's own principle applied to this program:
		// `docscheck README.md` looks exactly like a request to check one document.
		fmt.Fprintf(os.Stderr, "docscheck: %s: no arguments; it checks %s\n",
			os.Args[1], strings.Join(docs, " and "))
		os.Exit(2)
	}
	if err := run(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "docscheck: %v\n", err)
		os.Exit(1)
	}
}

func run(w io.Writer) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	if _, err := exec.LookPath("bash"); err != nil {
		// One message rather than the same failure nineteen times. The fences say `bash`, so a
		// machine without one cannot run what the documents say, and pretending otherwise by
		// falling back to `sh` would check a different command line than the one on the page.
		return fmt.Errorf("the documents' fences say bash, and there is none in PATH: %w", err)
	}

	lines, err := scrapeAll(root)
	if err != nil {
		return err
	}
	if err := classified(lines); err != nil {
		return err
	}

	var ran, skipped, failed int
	var notRun []outcome
	for _, doc := range docs {
		// One scratch directory per document rather than one for the run. Both documents build a
		// house out of thin air and neither names the other's file today, but a shared directory
		// would mean a file `README.md` wrote could be what makes a `CONTRIBUTING.md` line pass --
		// which is the same defect as a test that depends on the test before it.
		dir, err := shadow(root)
		if err != nil {
			return err
		}
		keep := false
		fmt.Fprintf(w, "\n%s\n", doc)
		for _, l := range lines[doc] {
			o := do(dir, l, rules[l.key])
			switch {
			case o.why != "":
				skipped++
				notRun = append(notRun, o)
			case o.err != nil:
				failed, keep = failed+1, true
				fmt.Fprintf(w, "  %4d  FAIL  %5.2fs  %s\n", l.num, o.dur.Seconds(), l.text)
				fmt.Fprintf(w, "              %s at %s:%d\n", o.err, doc, l.num)
				for _, t := range tail(o.output, 12) {
					fmt.Fprintf(w, "              | %s\n", t)
				}
			default:
				ran++
				fmt.Fprintf(w, "  %4d  ok    %5.2fs  %s\n", l.num, o.dur.Seconds(), l.text)
			}
		}
		// Kept only when something failed, because then what the lines wrote is evidence. A clean
		// run leaves nothing behind: `make check` is run often enough that a directory per run
		// would be a slow leak nobody would connect to this.
		if keep {
			fmt.Fprintf(w, "  what those lines wrote is in %s\n", dir)
			continue
		}
		os.RemoveAll(dir)
	}

	if len(notRun) > 0 {
		fmt.Fprintf(w, "\nnot run -- %d lines, and why\n", len(notRun))
		for _, o := range notRun {
			fmt.Fprintf(w, "  %s:%d  %s\n", o.line.doc, o.line.num, o.line.text)
			fmt.Fprintf(w, "      %s\n", o.why)
		}
	}

	total := ran + skipped + failed
	if failed > 0 {
		return fmt.Errorf("%d of the %d documented command lines this machine can run did not work",
			failed, ran+failed)
	}
	fmt.Fprintf(w, "\ndocscheck: %d of %d documented command lines ran, and all of them worked\n",
		ran, total)
	return nil
}

// outcome is what happened to one documented line.
type outcome struct {
	line   line
	why    string // why it was not run; empty means it ran
	dur    time.Duration
	output string
	err    error
}

// do runs one line, or says why not.
func do(dir string, l line, r rule) outcome {
	if r.skip != "" {
		return outcome{line: l, why: r.skip}
	}
	for _, need := range r.needs {
		if _, err := os.Stat(filepath.Join(dir, need)); err != nil {
			return outcome{line: l, why: fmt.Sprintf("%s is not on this machine, and this line "+
				"needs it", need)}
		}
	}
	for _, s := range r.setup {
		if out, err := shell(dir, s); err != nil {
			// Not the document's fault, and said so, because a reader debugging a docs-check
			// failure should not go looking in the README for `cp`.
			return outcome{line: l, output: out, dur: 0,
				err: fmt.Errorf("docscheck's own setup line `%s` failed: %w", s, err)}
		}
	}
	start := time.Now()
	out, err := shell(dir, l.text)
	return outcome{line: l, dur: time.Since(start), output: out, err: err}
}

// shell runs one command line the way the fence says to run it.
func shell(dir, cmdline string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), perLine)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-o", "pipefail", "-c", cmdline)
	cmd.Dir = dir
	// WaitDelay so that a line which leaves a child holding the output pipe cannot keep this
	// program waiting after the deadline has killed the shell.
	cmd.WaitDelay = 5 * time.Second
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	if ctx.Err() != nil {
		err = fmt.Errorf("did not finish in %s, which is the failure a test suite cannot report "+
			"on its own", perLine)
	}
	return buf.String(), err
}

// shadow builds the directory the lines run in: a symlink to every top-level entry of the
// repository except .git. See the package comment for why each of those three properties matters.
func shadow(root string) (string, error) {
	dir, err := os.MkdirTemp("", "docscheck-")
	if err != nil {
		return "", err
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if e.Name() == ".git" {
			continue
		}
		if err := os.Symlink(filepath.Join(root, e.Name()), filepath.Join(dir, e.Name())); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// tail is the last n lines of some output, which is where a command puts the reason it failed.
func tail(out string, n int) []string {
	all := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(all) == 1 && all[0] == "" {
		return []string{"(no output)"}
	}
	if len(all) > n {
		all = append([]string{fmt.Sprintf("(%d earlier lines)", len(all)-n)}, all[len(all)-n:]...)
	}
	return all
}
