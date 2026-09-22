// Command glidertool is the developer-facing half of the port. The game reads
// houses; this reads, writes, checks and explains them, so that the 1994 data can
// be inspected and hand-authored without a running game and diffed in git as
// text.
//
// Flags come before file names, as the Go flag package requires:
//
//	glidertool house dump Demo.house               # binary -> text, on stdout
//	glidertool house dump -residue -o d.txt d.house # byte-exact text (forensics)
//	glidertool house build -o new.house new.txt    # text -> binary
//	glidertool house check assets/extracted/houses/*.house
//	glidertool house lint  -min warn new.house     # will it play as authored?
//	glidertool house checks                        # what every lint check means
//	glidertool house info  assets/extracted/houses/*.house
//	glidertool house rooms Demo.house
//	glidertool render -scale 2 -o room.png Demo.house
//	glidertool render -all -o /tmp/demo Demo.house  # every room, for eyeballing
//	glidertool replay -house Demo -frames 600      # headless run, summary + digest
//	glidertool replay -trace bug.txt               # per-frame trace from a script
//	glidertool demo info -stats res/demo/128.bin   # the attract-mode input stream
//	glidertool types vent                          # object type names, filtered
//	glidertool version                             # this build, for pasting into a report
//
// Nothing here is needed to play. It exists because every later stage of the port
// is a claim about what the original data means, and a claim is only worth
// something if it can be checked cheaply.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/bwenstar/gliderGo/internal/project"
)

const prog = "glidertool"

func main() {
	log.SetFlags(0)
	log.SetPrefix(prog + ": ")
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage(os.Stderr)
		return errors.New("no command given")
	}
	switch args[0] {
	case "house":
		return houseCmd(args[1:])
	case "render":
		return renderCmd(args[1:])
	case "replay":
		return replayCmd(args[1:])
	case "demo":
		return demoCmd(args[1:])
	case "types":
		return typesCmd(args[1:])
	case "version", "-version", "--version":
		versionCmd(os.Stdout)
		return nil
	case "help", "-h", "-help", "--help":
		usage(os.Stdout)
		return nil
	}
	usage(os.Stderr)
	return fmt.Errorf("unknown command %q", args[0])
}

func usage(w io.Writer) {
	fmt.Fprintf(w, `%s -- read, write and check `+project.Original+` house files.

usage: %s <command> [flags] [file...]

  house dump  [-residue] [-o out] <house>   print a house in the text format
  house build [-o out] <text|->             assemble a text house into binary
  house check <house>...                    load, round-trip and sanity-check
  house lint  [flags] <house>...            report what will not play as authored
  house checks                              every lint check, with what it means
  house info  <house>...                    one summary line per house
  house rooms [-objects] <house>            per-room table
  render      [flags] <house>               compose a room to PNG, as the game does
  replay      [flags] [script]              play headlessly from a script; trace it
  demo info   [-stats] <demo>...            describe a recorded input stream
  demo dump   [-bare] [-o out] <demo>       print one record per line
  demo check  <demo>...                     decode, round-trip and check playability
  types       [substring]                   the object type names, by code
  version                                   this build, and what it is a port of

Flags precede file names. The text format is documented by the header comment
that `+"`house dump`"+` writes; the binary formats are documented in
docs/analysis/house-format.md and docs/analysis/input.md §14.

`+project.Home+` -- bugs to `+project.Issues+`
`, prog, prog)
}

// version is what a report quotes, set by the Makefile's -X exactly as cmd/glidergo's is. A
// plain `go build` leaves it as it stands here and the stamps below fill the gap.
var version = "dev"

// versionCmd answers `glidertool version`.
//
// This tool had no way to identify itself for four stages, which is a worse gap here than it
// would be in the game: every output `glidertool` produces is an *assertion about somebody
// else's data* -- a house dumped to text, a lint report, a replay digest, a per-frame trace --
// and those get pasted into bug reports and compared between machines. A digest that differs
// between two people is either a real divergence or two different builds, and until now there
// was no way to tell which. `house dump`'s output is committed to git in places, too.
//
// It is shorter than the game's, because a developer tool has no assets to account for, no
// window to fail to open and no preferences to lose. What it does carry is the upstream pin:
// the whole point of `house dump` and `demo info` is that they explain a 1994 format, and the
// document they are explaining is a particular revision of a particular repository.
func versionCmd(w io.Writer) {
	fmt.Fprintf(w, "%s %s\n", prog, version)
	fmt.Fprintf(w, "  built by  %s\n", runtime.Version())
	fmt.Fprintf(w, "  platform  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	if info, ok := debug.ReadBuildInfo(); ok {
		// Same three stamps cmd/glidergo prints and for the same reason; see vcsRows there.
		// Not shared between the two commands, because `internal` is where shared code goes
		// and a package holding one nine-line loop that formats output for two callers with
		// different label widths would be a worse trade than the repetition.
		for _, key := range []struct{ setting, label string }{
			{"vcs.revision", "commit"},
			{"vcs.time", "committed"},
			{"vcs.modified", "modified"},
		} {
			for _, s := range info.Settings {
				if s.Key == key.setting {
					fmt.Fprintf(w, "  %-9s %s\n", key.label, s.Value)
					break
				}
			}
		}
	}
	fmt.Fprintf(w, "  home      %s\n", project.Home)
	fmt.Fprintf(w, "  bugs      %s\n", project.Issues)
	fmt.Fprintf(w, "  licence   %s\n", project.Licence)
	fmt.Fprintf(w, "  reads     %s data -- %s / %s, %s\n", project.Original,
		project.OriginalAuthor, project.OriginalPublisher, project.OriginalYear)
	fmt.Fprintf(w, "  from      %s @ %s\n", project.Upstream, project.UpstreamCommit)
}

// openOut resolves an -o flag: empty or "-" means stdout, which is not closed.
func openOut(path string) (io.Writer, func() error, error) {
	if path == "" || path == "-" {
		return os.Stdout, func() error { return nil }, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, f.Close, nil
}
