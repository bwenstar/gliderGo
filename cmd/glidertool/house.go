package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"glidergo/internal/house"
)

func houseCmd(args []string) error {
	if len(args) == 0 {
		usage(os.Stderr)
		return fmt.Errorf("house needs a subcommand: dump, build, check, info or rooms")
	}
	switch args[0] {
	case "dump":
		return houseDump(args[1:])
	case "build":
		return houseBuild(args[1:])
	case "check":
		return houseCheck(args[1:])
	case "info":
		return houseInfo(args[1:])
	case "rooms":
		return houseRooms(args[1:])
	}
	usage(os.Stderr)
	return fmt.Errorf("unknown house subcommand %q", args[0])
}

// ------------------------------------------------------------------- dump

func houseDump(args []string) error {
	fs := flag.NewFlagSet("house dump", flag.ContinueOnError)
	residue := fs.Bool("residue", false,
		"keep the bytes the text format normally drops, making the text byte-exact")
	bare := fs.Bool("bare", false, "omit the explanatory header comment")
	out := fs.String("o", "-", "write here instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("house dump takes exactly one house file")
	}
	h, err := house.LoadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	w, closeOut, err := openOut(*out)
	if err != nil {
		return err
	}
	opt := house.TextOptions{Residue: *residue, NoHeader: *bare}
	if err := h.WriteText(w, opt); err != nil {
		closeOut()
		return err
	}
	return closeOut()
}

// ------------------------------------------------------------------- build

func houseBuild(args []string) error {
	fs := flag.NewFlagSet("house build", flag.ContinueOnError)
	out := fs.String("o", "", "write the binary house here; `-o -` means stdout")
	quiet := fs.Bool("q", false, "do not report what was written")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("house build takes exactly one text file (`-` for stdin)")
	}
	if *out == "" {
		return fmt.Errorf("house build needs -o <file>; pass `-o -` to write binary to stdout")
	}

	var (
		h   *house.House
		err error
	)
	if fs.Arg(0) == "-" {
		h, err = house.ParseText(os.Stdin)
	} else {
		h, err = house.ParseTextFile(fs.Arg(0))
	}
	if err != nil {
		return err
	}

	if *out == "-" {
		_, err := h.WriteTo(os.Stdout)
		return err
	}
	if err := h.SaveFile(*out); err != nil {
		return err
	}
	if !*quiet {
		fmt.Fprintf(os.Stderr, "%s: %d rooms, %d objects, %d bytes\n",
			*out, len(h.Rooms), liveObjects(h), h.Size())
	}
	return nil
}

// ------------------------------------------------------------------- check

// houseCheck is the tool the rest of the port leans on: it proves that a file
// survives both codecs unchanged, then reports the structural oddities that are
// legal but worth knowing about. Round-trip failures are errors and set the exit
// status; the oddities are notes, because six shipped rooms have holes in their
// object slots and a tool that called that an error would be wrong about the
// format rather than about the file.
func houseCheck(args []string) error {
	fs := flag.NewFlagSet("house check", flag.ContinueOnError)
	quiet := fs.Bool("q", false, "print only files that fail")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("house check takes one or more house files")
	}

	bad := 0
	for _, path := range fs.Args() {
		problems, notes := checkFile(path)
		if len(problems) > 0 {
			bad++
			fmt.Printf("%s: FAILED\n", path)
			for _, p := range problems {
				fmt.Printf("    error: %s\n", p)
			}
			for _, n := range notes {
				fmt.Printf("    note:  %s\n", n)
			}
			continue
		}
		if *quiet {
			continue
		}
		fmt.Printf("%s: ok\n", path)
		for _, n := range notes {
			fmt.Printf("    note:  %s\n", n)
		}
	}
	if bad > 0 {
		return fmt.Errorf("%d of %d files failed", bad, fs.NArg())
	}
	return nil
}

func checkFile(path string) (problems, notes []string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{err.Error()}, nil
	}
	h, err := house.Load(raw)
	if err != nil {
		return []string{err.Error()}, nil
	}
	problem := func(format string, a ...any) {
		problems = append(problems, fmt.Sprintf(format, a...))
	}
	note := func(format string, a ...any) {
		notes = append(notes, fmt.Sprintf(format, a...))
	}

	// 1. Binary round trip: the loader kept every byte it read.
	out, err := h.Save()
	switch {
	case err != nil:
		problem("save: %v", err)
	case len(out) != len(raw):
		problem("re-saving produced %d bytes, the file is %d", len(out), len(raw))
	default:
		if d := firstDiff(raw, out); d >= 0 {
			problem("byte %d changed by a binary round trip: 0x%02X -> 0x%02X (%s)",
				d, raw[d], out[d], locate(d, len(h.Rooms)))
		}
	}

	// 2. Text round trip, twice. The default mode is field-exact, so it is
	// compared against the canonical form; -residue mode must be byte-exact
	// against the file itself.
	if canon, err := h.Canonical().Save(); err != nil {
		problem("save canonical: %v", err)
	} else if got, err := textRoundTrip(h, house.TextOptions{}); err != nil {
		problem("text round trip: %v", err)
	} else if d := firstDiff(canon, got); d >= 0 {
		problem("text round trip lost a field at byte %d (%s)", d, locate(d, len(h.Rooms)))
	}
	if got, err := textRoundTrip(h, house.TextOptions{Residue: true}); err != nil {
		problem("exact text round trip: %v", err)
	} else if len(got) != len(raw) {
		problem("exact text round trip produced %d bytes, the file is %d", len(got), len(raw))
	} else if d := firstDiff(raw, got); d >= 0 {
		problem("byte %d changed by an exact text round trip: 0x%02X -> 0x%02X (%s)",
			d, raw[d], got[d], locate(d, len(h.Rooms)))
	}

	// 3. Structure. None of this is fatal; all of it changes how a room reads.
	if h.Version != house.HouseVersion {
		note("version is 0x%04X, not the 0x%04X every shipped house uses",
			uint16(h.Version), house.HouseVersion)
	}
	if int(h.FirstRoom) < 0 || int(h.FirstRoom) >= len(h.Rooms) {
		problem("firstRoom is %d, outside the %d rooms", h.FirstRoom, len(h.Rooms))
	}
	if n := len(h.Slack); n > 0 {
		note("%d trailing byte(s) past the last room (%s) -- a PowerPC save, see house-format.md 12.4",
			n, hex(h.Slack))
	}
	if h.HasGame != 0 {
		note("carries a saved game: room %d, score %d, %d gliders",
			h.SavedGame.RoomNumber, h.SavedGame.Score, h.SavedGame.NumGliders)
	}
	if !h.Unlocked() {
		note("locked (timeStamp bit 0 set): the original's editor refuses to save it")
	}

	holes := 0
	for i := range h.Rooms {
		r := &h.Rooms[i]
		if !r.Compacted() {
			holes++
		}
		if int(r.NumObjects) != r.LiveObjects() {
			note("room %d (%q): numObjects says %d, %d slots are live -- the original would rewrite it",
				i, r.Name.Text(), r.NumObjects, r.LiveObjects())
		}
		if n := int(r.Name[0]); n < 1 || n > len(r.Name)-1 {
			note("room %d: name length byte is %d, outside 1..%d", i, n, len(r.Name)-1)
		}
		for slot, o := range r.Objects {
			if !o.IsEmpty() && o.Group() == house.GroupNone {
				note("room %d slot %d: what 0x%02X is in none of the nine ranges",
					i, slot, uint16(o.What))
			}
		}
	}
	if holes > 0 {
		note("%d room(s) have holes in objects[]; slots are never renumbered", holes)
	}
	return problems, notes
}

// textRoundTrip writes the house as text, parses it back and returns the binary
// encoding of the result, which is the only comparable form of "every field".
func textRoundTrip(h *house.House, opt house.TextOptions) ([]byte, error) {
	var buf bytes.Buffer
	if err := h.WriteText(&buf, opt); err != nil {
		return nil, err
	}
	back, err := house.ParseText(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}
	return back.Save()
}

// ------------------------------------------------------------------- info

func houseInfo(args []string) error {
	fs := flag.NewFlagSet("house info", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("house info takes one or more house files")
	}
	star, _ := house.ObjectCode("kStar")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "house\trooms\tobjects\tstars\tbytes\tstart room\tsaved\tflags\tbanner")
	totRooms, totObjects, totStars := 0, 0, 0
	for _, path := range fs.Args() {
		h, err := house.LoadFile(path)
		if err != nil {
			fmt.Fprintf(w, "%s\t--\t--\t--\t--\t--\t--\t--\t%v\n", stem(path), err)
			continue
		}
		objects, stars := 0, 0
		for i := range h.Rooms {
			for _, o := range h.Rooms[i].Objects {
				if o.IsEmpty() {
					continue
				}
				objects++
				if o.What == star {
					stars++
				}
			}
		}
		start := fmt.Sprintf("%d", h.FirstRoom)
		if int(h.FirstRoom) >= 0 && int(h.FirstRoom) < len(h.Rooms) {
			start = fmt.Sprintf("%d %q", h.FirstRoom, h.Rooms[h.FirstRoom].Name.Text())
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%q\n",
			stem(path), len(h.Rooms), objects, stars, h.Size(), start,
			h.Saved().Format("2006-01-02"), flagString(h), h.Banner.Text())
		totRooms += len(h.Rooms)
		totObjects += objects
		totStars += stars
	}
	if fs.NArg() > 1 {
		// Four cells and stop: a shorter final line keeps tabwriter from padding
		// the columns it does not fill.
		fmt.Fprintf(w, "%d houses\t%d\t%d\t%d\n", fs.NArg(), totRooms, totObjects, totStars)
	}
	return w.Flush()
}

// flagString renders the three house flags plus the lock bit, in the sense the
// original tests them rather than the sense their names suggest -- the star count
// shows when its bit is *clear*.
func flagString(h *house.House) string {
	var s []string
	if h.Ward() {
		s = append(s, "ward")
	}
	if h.Phone() {
		s = append(s, "phone")
	}
	if h.BannerStarCount() {
		s = append(s, "starcount")
	}
	if !h.Unlocked() {
		s = append(s, "locked")
	}
	if h.HasGame != 0 {
		s = append(s, "savedgame")
	}
	if len(s) == 0 {
		return "-"
	}
	return strings.Join(s, ",")
}

// ------------------------------------------------------------------- rooms

func houseRooms(args []string) error {
	fs := flag.NewFlagSet("house rooms", flag.ContinueOnError)
	objects := fs.Bool("objects", false, "list every object under its room")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("house rooms takes exactly one house file")
	}
	h, err := house.LoadFile(fs.Arg(0))
	if err != nil {
		return err
	}

	// Two shapes, because a table and an outline do not mix: without -objects the
	// rooms line up in columns, and with it each room heads its own block.
	if *objects {
		for i := range h.Rooms {
			r := &h.Rooms[i]
			fmt.Printf("room %d %q  floor %d  suite %d  bg %d  bounds 0x%04X  visited %d%s\n",
				i, r.Name.Text(), r.Floor, r.Suite, r.Background,
				uint16(r.Bounds), r.Visited, startMark(h, i))
			for slot, o := range r.Objects {
				if o.IsEmpty() {
					continue
				}
				at := o.TopLeft()
				fmt.Printf("    %2d  %-16s %-10s v=%-4d h=%d\n",
					slot, o, o.Group(), at.V, at.H)
			}
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "room\tname\tfloor\tsuite\tbg\tobjs\tbounds\tvisited")
	for i := range h.Rooms {
		r := &h.Rooms[i]
		fmt.Fprintf(w, "%d\t%q\t%d\t%d\t%d\t%d\t0x%04X\t%d%s\n",
			i, r.Name.Text(), r.Floor, r.Suite, r.Background,
			r.LiveObjects(), uint16(r.Bounds), r.Visited, startMark(h, i))
	}
	return w.Flush()
}

// startMark tags the room the house starts in, which is the one fact about a room
// list that a reader is most likely to be looking for.
func startMark(h *house.House, room int) string {
	if int(h.FirstRoom) == room {
		return "  <- start"
	}
	return ""
}

// ------------------------------------------------------------------- helpers

func liveObjects(h *house.House) int {
	n := 0
	for i := range h.Rooms {
		n += h.Rooms[i].LiveObjects()
	}
	return n
}

func stem(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".house")
}

func firstDiff(a, b []byte) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

// locate names a file offset, so that a round-trip failure points at a field
// instead of a number. The object base is derived from the exported sizes rather
// than copied: roomType ends with its 24 twelve-byte slots.
func locate(off, nRooms int) string {
	if off < house.SizeofHouseHeader {
		return fmt.Sprintf("header byte %d", off)
	}
	rel := off - house.SizeofHouseHeader
	room, within := rel/house.SizeofRoom, rel%house.SizeofRoom
	if room >= nRooms {
		return fmt.Sprintf("trailing byte %d past room %d", rel-nRooms*house.SizeofRoom, nRooms-1)
	}
	const objectBase = house.SizeofRoom - house.MaxRoomObs*house.SizeofObject
	if within >= objectBase {
		o := within - objectBase
		return fmt.Sprintf("room %d object %d byte %d",
			room, o/house.SizeofObject, o%house.SizeofObject)
	}
	return fmt.Sprintf("room %d byte %d", room, within)
}

func hex(b []byte) string {
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = fmt.Sprintf("%02x", v)
	}
	return strings.Join(parts, " ")
}
