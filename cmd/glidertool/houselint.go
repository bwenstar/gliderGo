package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/bwenstar/gliderGo/internal/audio"
	"github.com/bwenstar/gliderGo/internal/cliargs"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/render"
)

// houseLint is `glidertool house lint`: the authoring check.
//
// `house check` answers "did this file survive both codecs" -- a question about the
// port. This answers "will this house play the way its author meant" -- a question
// about the house. They are separate commands because they fail for separate
// reasons and a new house needs both: check proves the tool did not mangle it, lint
// proves the author did not.
//
// Everything it reports comes out of internal/house.Lint. What this file adds is the
// two things a house file cannot answer about itself -- whether a sound loads and how
// wide a background is -- and a report that can be read by a person, diffed between
// two versions of a house, or turned into an exit code by a CI step.
func houseLint(args []string) error {
	fs := flag.NewFlagSet("house lint", flag.ContinueOnError)
	min := fs.String("min", "note", "lowest severity to print: note, warn or error")
	failOn := fs.String("fail", "error",
		"exit non-zero if any finding reaches this severity; `never` to always exit 0")
	only := fs.String("check", "", "print only these checks (comma-separated ids)")
	summary := fs.Bool("summary", false, "print only the per-file tally")
	noAssets := fs.Bool("no-assets", false,
		"skip the checks that need art and sound, and say so")
	artDir := fs.String("art", "", "extracted application art to use instead of the built-in copy")
	houseDir := fs.String("houseart", "", "extracted per-house resource forks instead of the built-in ones")
	soundDir := fs.String("sounds", "", "extracted sounds instead of the built-in ones")
	if err := fs.Parse(cliargs.FlagsFirst(fs, args)); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("house lint takes one or more house files")
	}

	minSev, ok := house.ParseSeverity(*min)
	if !ok {
		return fmt.Errorf("-min: %q is not note, warn or error", *min)
	}
	failSev, failAny := house.ParseSeverity(*failOn)
	if !failAny && !strings.EqualFold(*failOn, "never") {
		return fmt.Errorf("-fail: %q is not note, warn, error or never", *failOn)
	}

	// A misspelled -check would otherwise filter everything out and report a clean
	// house, which is the one failure mode a linter must not have.
	var wanted map[string]bool
	if *only != "" {
		known := map[string]bool{}
		for _, c := range house.LintChecks() {
			known[c.ID] = true
		}
		wanted = map[string]bool{}
		for _, c := range strings.Split(*only, ",") {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if !known[c] {
				return fmt.Errorf("-check: %q is not a check; `%s house checks` lists them all", c, prog)
			}
			wanted[c] = true
		}
	}

	// One asset cache for the whole run. render.Assets memoises decoded pictures and
	// the house fork is swapped per file, exactly as HouseIO.c swaps it, so linting
	// twenty-two houses decodes the application art once.
	var (
		assets  *render.Assets
		bank    *audio.Bank
		artName string
		sndName string
	)
	if !*noAssets {
		artFS, an := assetRoot(*artDir, "art")
		assets, artName = render.NewAssets(artFS), an
		soundFS, sn := assetRoot(*soundDir, "sound")
		sndName = sn
		b, err := audio.LoadBank(soundFS)
		if err != nil {
			// Not fatal. A bank that will not load means the two sound checks cannot
			// run, and Lint says which checks did not run -- so the report stays
			// honest instead of quietly passing.
			fmt.Fprintf(os.Stderr,
				"%s: %s will not load, so the sound checks are skipped: %v\n",
				prog, sndName, err)
		} else {
			bank = b
		}
	}
	// Three roots searched rather than one, so that a house of this port's own can be linted
	// against art of its own without taking the 1994 houses' art away (houseArtRoots).
	artRoots := houseArtRoots(*houseDir)

	hit := false
	var totalNotes, totalWarns, totalErrs int

	for _, path := range fs.Args() {
		h, err := house.LoadFile(path)
		if err != nil {
			return err
		}

		// The house's own resources go in front of the application's for as long as
		// the house is open (HouseIO.c's OpenHouseResFork), and the fork is found by
		// the file's base name -- which is what the house is called on disk and in
		// the built-in tree alike.
		//
		// No Fork.Complaint here, and that is not an omission. The other two callers say "this
		// house wanted art and there was none" because that is all they can say; lint has
		// opt.PictSize and so reports the same fact per picture, naming the id that is missing
		// and the room it is in. A line above the report saying less than the report does
		// would be the second copy of a message that 4.17 was about there being two of.
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		var opt house.LintOptions
		if assets != nil {
			assets.CloseHouseResFork()
			if fork := artRoots.Fork(name); fork.Found() {
				assets.OpenHouseResFork(fork.Label, fork.FS)
			}
			opt.PictSize = func(id int16) (int, int, bool) {
				s := assets.Pict(id)
				if s == nil {
					return 0, 0, false
				}
				return s.W, s.H, true
			}
		}
		if bank != nil {
			if err := bank.LoadHouse(name); err != nil {
				return fmt.Errorf("%s: loading %s sounds: %w", path, name, err)
			}
			opt.SoundStatus = func(id int16) house.SoundStatus {
				switch {
				case bank.Trigger(id) != nil:
					return house.SoundOK
				case bank.WhyUnreadable(id) != "":
					return house.SoundUnreadable
				}
				return house.SoundMissing
			}
		}

		findings := h.Lint(opt)
		notes, warns, errs := house.Counts(findings)
		totalNotes, totalWarns, totalErrs = totalNotes+notes, totalWarns+warns, totalErrs+errs
		if failAny {
			for _, f := range findings {
				if f.Severity >= failSev {
					hit = true
				}
			}
		}

		fmt.Printf("%s: %d rooms, %d notes, %d warnings, %d errors\n",
			path, len(h.Rooms), notes, warns, errs)
		if *summary {
			continue
		}
		for _, f := range house.AtLeast(findings, minSev) {
			if wanted != nil && !wanted[f.Check] {
				continue
			}
			fmt.Printf("    %s\n", f)
		}
	}

	if fs.NArg() > 1 {
		fmt.Printf("total: %d files, %d notes, %d warnings, %d errors\n",
			fs.NArg(), totalNotes, totalWarns, totalErrs)
	}
	if *noAssets {
		fmt.Printf("note: -no-assets was given, so the art and sound checks did not run\n")
	} else if assets != nil {
		fmt.Printf("checked against %s and %s\n", artName, sndName)
	}

	if hit {
		return fmt.Errorf("a finding reached %s", failSev)
	}
	return nil
}

// houseLintChecks is `glidertool house checks`: the catalogue.
//
// A linter is only usable if its findings can be looked up, and a check id that
// appears in a report with no explanation anywhere is worse than no check. The table
// lives in internal/house beside the code that emits the findings, so that a test
// there can hold the two together; this only prints it.
//
// The severity column is the *worst* a check reports. Three of them report at two
// levels, because the same defect can be residue in one house and something a player
// walks into in another.
func houseLintChecks(args []string) error {
	fs := flag.NewFlagSet("house checks", flag.ContinueOnError)
	if err := fs.Parse(cliargs.FlagsFirst(fs, args)); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("house checks takes no arguments")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "CHECK\tUP TO\tWHAT IT MEANS\n")
	for _, c := range house.LintChecks() {
		fmt.Fprintf(w, "%s\t%s\t%s\n", c.ID, c.Severity, c.What)
	}
	return w.Flush()
}
