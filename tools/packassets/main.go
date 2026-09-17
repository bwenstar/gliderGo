// Command packassets builds the archive of assets/extracted that the game and glidertool carry
// inside them, or checks a built one against the tree.
//
//	go run ./tools/packassets                 # rebuild assets/extracted.zip
//	go run ./tools/packassets -check          # is the committed archive the committed tree?
//
// It is a separate program and not a glidertool subcommand for one reason: glidertool embeds the
// archive, so building glidertool needs the archive to exist. The thing that makes the archive
// cannot be the thing that needs it.
//
// internal/assetpack has the whole argument for why the assets are in an archive at all.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwenstar/gliderGo/internal/assetpack"
)

func main() {
	tree := flag.String("tree", filepath.Join("assets", "extracted"), "the extracted asset tree to pack")
	out := flag.String("out", filepath.Join("assets", assetpack.Name), "the archive to write, or to check")
	check := flag.Bool("check", false, "compare the archive against the tree instead of writing it")
	flag.Parse()

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
			return fmt.Errorf("%s is not %s: `make assets-zip` rebuilds it", out, tree)
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
