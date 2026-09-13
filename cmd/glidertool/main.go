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
//	glidertool house info  assets/extracted/houses/*.house
//	glidertool house rooms Demo.house
//	glidertool render -scale 2 -o room.png Demo.house
//	glidertool render -all -o /tmp/demo Demo.house  # every room, for eyeballing
//	glidertool types vent                          # object type names, filtered
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
	case "types":
		return typesCmd(args[1:])
	case "help", "-h", "-help", "--help":
		usage(os.Stdout)
		return nil
	}
	usage(os.Stderr)
	return fmt.Errorf("unknown command %q", args[0])
}

func usage(w io.Writer) {
	fmt.Fprintf(w, `%s -- read, write and check Glider PRO house files.

usage: %s <command> [flags] [file...]

  house dump  [-residue] [-o out] <house>   print a house in the text format
  house build [-o out] <text|->             assemble a text house into binary
  house check <house>...                    load, round-trip and sanity-check
  house info  <house>...                    one summary line per house
  house rooms [-objects] <house>            per-room table
  render      [flags] <house>               compose a room to PNG, as the game does
  types       [substring]                   the object type names, by code

Flags precede file names. The text format is documented by the header comment
that `+"`house dump`"+` writes; the binary format is documented in
docs/analysis/house-format.md.
`, prog, prog)
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
