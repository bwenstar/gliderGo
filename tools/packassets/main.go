// Command packassets builds the archive of assets/extracted that the game and glidertool carry
// inside them, or checks a built one against the tree.
//
//	go run ./tools/packassets                 # rebuild assets/extracted.zip
//	go run ./tools/packassets -check          # is the committed archive the committed tree?
//
// The flags make it general, and there are two archives now:
//
//	go run ./tools/packassets -tree assets/levels -out assets/levels.zip
//
// packs the port's own houses, which `make levels-zip` is the name for.
//
// It is a separate program and not a glidertool subcommand for one reason: glidertool embeds the
// archive, so building glidertool needs the archive to exist. The thing that makes the archive
// cannot be the thing that needs it. That is doubly true of assets/levels.zip, whose tree
// glidertool is what writes.
//
// internal/assetpack has the whole argument for why the assets are in an archive at all.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwenstar/gliderGo/internal/assetpack"
	"github.com/bwenstar/gliderGo/internal/cliargs"
)

func main() {
	tree := flag.String("tree", filepath.Join("assets", "extracted"), "the extracted asset tree to pack")
	out := flag.String("out", filepath.Join("assets", assetpack.Name), "the archive to write, or to check")
	check := flag.Bool("check", false, "compare the archive against the tree instead of writing it")

	// Every argument this program takes is a flag, so there is no ordering here to get wrong
	// and cliargs.FlagsFirst hands these three back untouched. It is called all the same,
	// because the rule that every FlagSet in the tree is reordered has no exceptions and so
	// needs no list of them (internal/cliargs' own sweep is what checks that).
	flag.CommandLine.Parse(cliargs.FlagsFirst(flag.CommandLine, os.Args[1:]))

	// And a positional argument is refused rather than ignored, which is this program's share
	// of docs/IMPROVEMENTS.md 4.13: `packassets assets/levels` looks exactly like a request to
	// pack that tree, and used to pack assets/extracted instead and say it had succeeded.
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "packassets: %s names a tree; use -tree, and -out for the "+
			"archive\n", flag.Arg(0))
		os.Exit(2)
	}

	if err := run(*tree, *out, *check); err != nil {
		fmt.Fprintf(os.Stderr, "packassets: %v\n", err)
		os.Exit(1)
	}
}

func run(tree, out string, check bool) error {
	if check {
		b, err := os.ReadFile(out)
		if err != nil {
			return err
		}
		fsys, err := assetpack.Open(b)
		if err != nil {
			return fmt.Errorf("%s: %w", out, err)
		}
		diffs, err := assetpack.Compare(fsys, tree, 20)
		if err != nil {
			return err
		}
		if len(diffs) > 0 {
			for _, d := range diffs {
				fmt.Fprintf(os.Stderr, "  %s\n", d)
			}
			// The remedy names this program and not a make target, because there are two
			// archives and the target for each is a different word. Dropping -check from
			// the line you just typed is the rebuild, whichever archive it was.
			return fmt.Errorf("%s is not %s: `go run ./tools/packassets -tree %s -out %s` rebuilds it",
				out, tree, tree, out)
		}
		names, err := assetpack.Files(tree)
		if err != nil {
			return err
		}
		fmt.Printf("packassets: %s holds exactly the %d files in %s\n", out, len(names), tree)
		return nil
	}

	if err := assetpack.CreateFile(out, tree); err != nil {
		return err
	}
	st, err := os.Stat(out)
	if err != nil {
		return err
	}
	names, err := assetpack.Files(tree)
	if err != nil {
		return err
	}
	fmt.Printf("packassets: wrote %s -- %d files, %d KiB\n", out, len(names), st.Size()/1024)
	return nil
}
