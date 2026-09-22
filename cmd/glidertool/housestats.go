package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/bwenstar/gliderGo/internal/cliargs"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/profile"
	"github.com/bwenstar/gliderGo/internal/render"
)

// houseStats is `glidertool house stats`: the numbers a house author wants before there
// is a test to fail.
//
// `house info` is one line per house and `house lint` is a list of defects. This is
// neither. It is the eighteen rows of docs/analysis/original-houses.md 10.2 measured on a
// real file, with the 1994 houses' own spread printed beside them when -tier says which
// tier to compare against. The question it answers is "is this house the size and shape of
// the ones it will be played next to", which nothing in the port could answer before:
// internal/replay's two house tests held a tier's column between them and a test only
// speaks when it fails (docs/IMPROVEMENTS.md 4.16).
//
// # Why -tier does not fail by default
//
// 10.2's bands are a measurement of 22 houses, not a rule those 22 obey. Each column is
// the extreme of whichever house was most extreme in it, so **21 of the 22 shipped houses
// are outside at least one band of their own tier** -- Slumberland, the largest there is,
// is outside epic's empty-rooms band, and the one house that is inside all eighteen
// (Leviathan) manages it by having set eight of epic's own bounds. A miss is therefore a
// reason to look and not a verdict: every row prints whether it passes or not, and turning
// misses into an exit status is something a caller has to ask for with -fail.
//
// That 21 is measured rather than assumed, because the first draft of this paragraph
// assumed it was 22 and the corpus says otherwise:
// profile.TestTheBandsExcludeTheHousesTheyWereMeasuredFrom holds the figure and names the
// three mechanisms behind it, and docs/IMPROVEMENTS.md 4.25 is the entry about the table.
// 4.18 is the separate one about where the tiers and 10.3's construction procedure
// disagree.
//
// # What -no-assets costs
//
// Two rows, and only for the eight houses that carry resources of their own. A room with a
// house's own background and no bounds field of its own reads its four openings from that
// house's 'bnds' resource, so without the art tree those rooms read as sealed and the
// reachable count and the eccentricity both come out low. 155 rooms across 7 of the
// shipped houses take that path. Nothing else here needs a file: the dark-room count is a
// table lookup and the other fifteen rows are arithmetic.
//
// So the caveat is counted and not assumed. Profile.ForkBounded is how many rooms this run
// held that take that path, and when it is zero -- which is every house of the port's own
// and fifteen of the 22 -- the closing line says the measurement is exact rather than
// warning about a cost nobody paid.
func houseStats(args []string) error {
	fs := flag.NewFlagSet("house stats", flag.ContinueOnError)
	tierName := fs.String("tier", "",
		"compare against one of 10.2's size tiers: tutorial, small, medium, large or epic")
	failOnMiss := fs.Bool("fail", false,
		"exit non-zero if a row is outside its tier's band; needs -tier")
	summary := fs.Bool("summary", false, "one line per house instead of the eighteen-row table")
	withNames := fs.Bool("rooms", false, "name the dark and unreachable rooms")
	noAssets := fs.Bool("no-assets", false,
		"measure with no art, and say which rows that costs")
	artDir := fs.String("art", "", "extracted application art to use instead of the built-in copy")
	houseDir := fs.String("houseart", "",
		"extracted per-house resource forks instead of the built-in ones")
	if err := fs.Parse(cliargs.FlagsFirst(fs, args)); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("house stats takes one or more house files")
	}

	var tier profile.Tier
	haveTier := *tierName != ""
	if haveTier {
		t, ok := profile.ParseTier(*tierName)
		if !ok {
			return fmt.Errorf("-tier: %q is not one of %s",
				*tierName, strings.Join(profile.TierNames[:], ", "))
		}
		tier = t
	}
	// -fail with no tier would silently never fire, which is the shape of a CI step that
	// looks like it is checking something and is not.
	if *failOnMiss && !haveTier {
		return fmt.Errorf("-fail needs -tier: there is nothing to miss without a tier")
	}

	// One asset cache for the whole run, with the house fork swapped per file exactly as
	// HouseIO.c swaps it, so measuring twenty-two houses decodes the application art once.
	var assets *render.Assets
	artName := "no art"
	if !*noAssets {
		artFS, an := assetRoot(*artDir, "art")
		assets, artName = render.NewAssets(artFS), an
	}
	artRoots := houseArtRoots(*houseDir)

	// Rooms across this run whose four openings would have come out of a 'bnds' resource.
	// Counted rather than assumed so that the -no-assets caveat below names a number: for
	// fifteen of the 22 shipped houses, and for every house of the port's own, it is zero
	// and the measurement with no art is exact.
	missed, forkBounded := false, 0
	for i, path := range fs.Args() {
		h, err := house.LoadFile(path)
		if err != nil {
			return err
		}

		// The fork is found by the file's base name, which is what the house is called
		// on disk and in the built-in tree alike, and it is looked for in each of the three
		// roots in turn (houseArtRoots). Silent either way: this subcommand's output is a
		// measurement, and forkBounded below already counts the rooms whose numbers depended
		// on a fork being there.
		name := stem(path)
		if assets != nil {
			assets.CloseHouseResFork()
			if fork := artRoots.Fork(name); fork.Found() {
				assets.OpenHouseResFork(fork.Label, fork.FS)
			}
		}
		p := profile.Measure(h, assets)
		forkBounded += p.ForkBounded

		if *summary {
			if i == 0 {
				statsSummaryHeader(os.Stdout)
			}
			statsSummaryLine(os.Stdout, name, p)
			continue
		}
		if i > 0 {
			fmt.Println()
		}
		if statsTable(h, name, p, haveTier, tier, *withNames) {
			missed = true
		}
	}

	switch {
	case !*noAssets:
		fmt.Printf("measured against %s\n", artName)
	case forkBounded == 0:
		// Said rather than left out. "No art was used" invites the reader to discount the
		// two graph rows, and here there is nothing to discount.
		fmt.Printf("measured with no art, which costs nothing here: no room reads its " +
			"openings from a 'bnds' resource\n")
	default:
		fmt.Printf("note: -no-assets was given and %d room(s) read their openings from a "+
			"'bnds' resource, so those read as sealed and the reachable and eccentricity "+
			"rows come out low\n", forkBounded)
	}
	if haveTier {
		// Said once per run rather than once per house, and said even when nothing
		// missed: the reader who needs it is the one about to treat a band as a rule.
		//
		// The number is 21 of 22 and it is measured, not estimated -- an earlier draft of
		// this line said "no shipped house is inside all eighteen", which was a guess and
		// was wrong: Leviathan is inside all eighteen of epic's, being the house that set
		// eight of that tier's bounds. TestTheBandsExcludeTheHousesTheyWereMeasuredFrom
		// holds the figure, and docs/IMPROVEMENTS.md 4.25 is why it is 21 and not 0.
		fmt.Printf("note: 10.2's bands are the spread of the 1994 houses of this size, " +
			"not rules -- 21 of the 22 shipped houses are themselves outside a band of " +
			"their own tier, most often at an edge rounded inward past their own " +
			"value (IMPROVEMENTS 4.25)\n")
	}
	if missed && *failOnMiss {
		return fmt.Errorf("a row is outside the %s band", tier)
	}
	return nil
}

// statsTable prints one house's eighteen rows and returns whether any of them missed.
func statsTable(h *house.House, name string, p profile.Profile,
	haveTier bool, tier profile.Tier, withNames bool) bool {

	fmt.Printf("%s: %d rooms in %d slots, %d floors x %d suites, %d objects\n",
		name, p.Rooms, p.Slots, p.Floors, p.Suites, p.Objects)

	// The verdict column asks profile.Check rather than testing the band again here, so
	// that the tool and the package cannot come to disagree about what a miss is -- and
	// so that the n/a rule (a tier with no target for a row has nothing to miss) is
	// stated in one place.
	miss := map[string]bool{}
	if haveTier {
		for _, m := range profile.Check(p, tier) {
			miss[m.Row.Target] = true
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	if haveTier {
		fmt.Fprintf(w, "target\tvalue\t%s\tverdict\n", tier)
	} else {
		fmt.Fprintf(w, "target\tvalue\n")
	}
	for _, r := range profile.Rows() {
		// A row that cannot be measured is not a row that measured zero. prize:enemy on
		// a house with no enemies has no denominator, and printing 0 would read as
		// "this house has no prizes".
		value := "n/a"
		if v, ok := r.Measure(p); ok {
			value = r.Unit.Format(v)
		}
		if !haveTier {
			fmt.Fprintf(w, "%s\t%s\n", r.Target, value)
			continue
		}
		verdict := "ok"
		switch {
		case r.Bands[tier].NA:
			verdict = "-"
		case miss[r.Target]:
			verdict = "miss"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Target, value, r.Bands[tier].String(r.Unit), verdict)
	}
	if err := w.Flush(); err != nil {
		// A write to stdout that fails is a closed pipe, and there is nothing useful to
		// do about it here that the next write will not do again.
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
	}

	fmt.Printf("reachable: %d of %d rooms from %s\n", p.Reachable, p.Rooms, startRoom(h, p))
	if withNames {
		statsNames("dark", p.DarkRooms)
		statsNames("unreachable", p.Unreachable)
	} else if len(p.DarkRooms) > 0 || len(p.Unreachable) > 0 {
		fmt.Printf("    %d dark, %d unreachable; -rooms names them\n",
			len(p.DarkRooms), len(p.Unreachable))
	}
	return len(miss) > 0
}

// startRoom names the room the walk began at.
//
// It is Profile's answer rather than the house's firstRoom field, because the two differ
// for a house that names a room it does not contain -- the original starts such a house in
// room 0 -- and a reachable count read against the wrong room is worse than no count.
func startRoom(h *house.House, p profile.Profile) string {
	if p.Start < 0 || int(p.Start) >= len(h.Rooms) {
		return "nowhere: this house has no room to start in"
	}
	s := fmt.Sprintf("%d %q", p.Start, h.Rooms[p.Start].Name.Text())
	if p.Start != h.FirstRoom {
		s += fmt.Sprintf(" (firstRoom says %d, which is not a room)", h.FirstRoom)
	}
	return s
}

// statsNames prints one list of rooms, wrapped, because a 531-room house can have
// fifty-seven dark rooms and a single line of those is unreadable.
//
// Index and name both, which is RoomRef's own doing: Titanic's 43 unreachable rooms are 24
// "Murmur", 9 "Dirt" and 8 "Pitch", and a list of those names is a list an author cannot
// act on.
func statsNames(what string, rooms []profile.RoomRef) {
	if len(rooms) == 0 {
		return
	}
	fmt.Printf("%s (%d):\n", what, len(rooms))
	line := "   "
	for _, r := range rooms {
		item := " " + r.String()
		if len(line)+len(item) > 78 {
			fmt.Println(line)
			line = "   "
		}
		line += item
	}
	fmt.Println(line)
}

// statsSummaryHeader and statsSummaryLine are the -summary shape: one line per house,
// which is what makes `house stats -summary assets/extracted/houses/*.house` a readable
// answer to "how do these 22 compare". The columns are the ones
// docs/analysis/original-houses.md 3.6 publishes, so the output can be read against it.
func statsSummaryHeader(w *os.File) {
	fmt.Fprintf(w, "%-22s %6s %7s %8s %7s %6s %11s %5s %8s\n",
		"house", "rooms", "objects", "obj/room", "empty", "dark", "reach/real", "ecc", "points")
}

func statsSummaryLine(w *os.File, name string, p profile.Profile) {
	fmt.Fprintf(w, "%-22s %6d %7d %8.1f %6.1f%% %6d %11s %5d %8d\n",
		name, p.Rooms, p.Objects, p.PerRoom(p.Objects), 100*p.PerRoom(p.Empty), p.Dark,
		fmt.Sprintf("%d/%d", p.Reachable, p.Rooms), p.Eccentricity, p.Points)
}
