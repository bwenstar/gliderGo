package main

// The `demo` subcommand: read, describe and check attract-mode input streams.
//
// A `'demo'` resource is six-byte records with no header, no magic and no length field
// (internal/demo documents the format and docs/analysis/input.md §14 derives it). That is
// exactly the kind of file that cannot be checked by looking at it: every question about one --
// is the stride really six, is it really big-endian, does it stall halfway through -- is a
// question about the *sequence* the bytes decode to. So this command answers those questions
// from the file, and `demo info` deliberately reprints the numbers the analysis doc arrived at
// by reading the C, so that the doc and the resource can be compared without trusting either.
//
//	glidertool demo info  assets/extracted/res/demo/128.bin
//	glidertool demo info  -stats assets/extracted/res/demo/*.bin
//	glidertool demo dump  assets/extracted/res/demo/128.bin | head
//	glidertool demo check assets/extracted/res/demo/128.bin
//
// There is no `demo build`. A hand-authored input stream is what `replay`'s `at` lines already
// are, and they are better at it -- two players, all seven keys, and no six-byte encoding to get
// wrong -- so the only streams worth writing are ones a game recorded.

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bwenstar/gliderGo/internal/demo"
)

func demoCmd(args []string) error {
	if len(args) == 0 {
		usage(os.Stderr)
		return fmt.Errorf("demo needs a subcommand: info, dump or check")
	}
	switch args[0] {
	case "info":
		return demoInfo(args[1:])
	case "dump":
		return demoDump(args[1:])
	case "check":
		return demoCheck(args[1:])
	}
	usage(os.Stderr)
	return fmt.Errorf("unknown demo subcommand %q", args[0])
}

// ------------------------------------------------------------------- info

func demoInfo(args []string) error {
	fs := flag.NewFlagSet(prog+" demo info", flag.ContinueOnError)
	stats := fs.Bool("stats", false, "add the timing analysis: gaps, held runs, frame parity")
	out := fs.String("o", "-", "write here instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("demo info takes one or more demo files")
	}
	dst, closeOut, err := openOut(*out)
	if err != nil {
		return err
	}

	// A load failure is a row rather than a return, so that a wildcard over a resource
	// directory reports every file instead of stopping at the first one that is not a demo.
	// `demo check` is the command that fails; this one describes.
	loaded := make([]demo.Stream, fs.NArg())
	w := tabwriter.NewWriter(dst, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "stream\tbytes\trecords\tframes\tright\tleft\tbatt\tband\tother\tpads\tstatus")
	for i, path := range fs.Args() {
		s, err := demo.Load(path)
		if err != nil {
			fmt.Fprintf(w, "%s\t--\t--\t--\t--\t--\t--\t--\t--\t--\t%v\n", demoStem(path), err)
			continue
		}
		loaded[i] = s
		keys := s.Keys()
		other := 0
		for k, n := range keys {
			if !k.Valid() {
				other += n
			}
		}
		span := "--"
		if len(s) > 0 {
			span = fmt.Sprintf("%d..%d", s[0].Frame, s.LastFrame())
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
			demoStem(path), len(s)*demo.RecordSize, len(s), span,
			keys[demo.KeyRight], keys[demo.KeyLeft], keys[demo.KeyBatt], keys[demo.KeyBand],
			other, distinctPads(s), demoStatus(s))
	}
	if err := w.Flush(); err != nil {
		closeOut()
		return err
	}

	if *stats {
		for i, path := range fs.Args() {
			if loaded[i] == nil {
				continue
			}
			fmt.Fprintln(dst)
			demoStats(dst, demoStem(path), loaded[i])
		}
	}
	return closeOut()
}

// demoStatus is the info table's last column: what playback would make of the stream.
//
// Validate's messages are sentences and this is a table cell, but it is the *last* cell, so the
// sentence is printed as it stands minus the package prefix. A single word would be worse:
// "invalid" in a column would send the reader to another command to find out which of the three
// rules was broken, and answering that is the whole point of the row.
func demoStatus(s demo.Stream) string {
	if err := s.Validate(); err != nil {
		return strings.TrimPrefix(err.Error(), "demo: ")
	}
	return "ok"
}

// distinctPads counts the values seen in the padding byte, which is the cheap evidence that it
// is heap residue rather than data: the shipped resource has 109 of them (internal/demo's
// Record.Pad), and anything this port records has exactly one, zero.
func distinctPads(s demo.Stream) int {
	seen := map[byte]bool{}
	for _, r := range s {
		seen[r.Pad] = true
	}
	return len(seen)
}

// demoStats is the timing half of `demo info`, and it is a flag rather than always on because it
// answers a different question. The table says what is in the file; this says how the recording
// was *paced*, which is what a fidelity investigation needs:
//
//	gaps    mostly-1 gaps mean a key held down and logged once per frame, which is what
//	        LogDemoKey does. Mostly-2 would mean the original logged on alternate frames,
//	        and would make the port's frame counter the wrong clock to replay against.
//	runs    the same fact grouped: how long the recorded player held each control, which is
//	        the readable form -- "84 presses, the longest 37 frames" -- of the gap histogram.
//	parity  even against odd frame numbers. The port alternates World.EvenFrame, and a
//	        recording confined to one parity would say some input pass ran on one of them.
func demoStats(out io.Writer, name string, s demo.Stream) {
	fmt.Fprintf(out, "%s: %d records", name, len(s))
	if len(s) == 0 {
		fmt.Fprintln(out)
		return
	}
	fmt.Fprintf(out, " over %d frames (%d..%d), %d idle before the first\n",
		s.LastFrame()-s[0].Frame+1, s[0].Frame, s.LastFrame(), s[0].Frame)

	// Gaps, most common first. Capped at eight columns because the tail of a long recording
	// is a scatter of one-offs -- the shipped stream has gaps up to 175 -- and the shape is in
	// the head.
	gaps := map[int64]int{}
	for i := 1; i < len(s); i++ {
		gaps[s[i].Frame-s[i-1].Frame]++
	}
	type gapCount struct {
		gap, n int64
	}
	byCount := make([]gapCount, 0, len(gaps))
	for g, n := range gaps {
		byCount = append(byCount, gapCount{g, int64(n)})
	}
	sort.Slice(byCount, func(i, j int) bool {
		if byCount[i].n != byCount[j].n {
			return byCount[i].n > byCount[j].n
		}
		return byCount[i].gap < byCount[j].gap
	})
	var b strings.Builder
	for i, gc := range byCount {
		if i == 8 {
			fmt.Fprintf(&b, " and %d more", len(byCount)-i)
			break
		}
		fmt.Fprintf(&b, " %d:%d", gc.gap, gc.n)
	}
	fmt.Fprintf(out, "  gaps     %d distinct;%s\n", len(byCount), b.String())

	// Runs: maximal stretches of consecutive frames holding one key.
	type run struct {
		key    demo.Key
		start  int64
		length int
	}
	var runs []run
	cur := run{key: s[0].Key, start: s[0].Frame, length: 1}
	for i := 1; i < len(s); i++ {
		if s[i].Key == cur.key && s[i].Frame == s[i-1].Frame+1 {
			cur.length++
			continue
		}
		runs = append(runs, cur)
		cur = run{key: s[i].Key, start: s[i].Frame, length: 1}
	}
	runs = append(runs, cur)
	perKey := map[demo.Key]int{}
	longest := runs[0]
	for _, r := range runs {
		perKey[r.key]++
		if r.length > longest.length {
			longest = r
		}
	}
	order := []demo.Key{demo.KeyRight, demo.KeyLeft, demo.KeyBatt, demo.KeyBand}
	var parts []string
	for _, k := range order {
		if perKey[k] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", k, perKey[k]))
		}
	}
	fmt.Fprintf(out, "  runs     %d held stretches (%s); longest %d frames of %s from frame %d\n",
		len(runs), strings.Join(parts, ", "), longest.length, longest.key, longest.start)

	// Parity.
	even := 0
	for _, r := range s {
		if r.Frame%2 == 0 {
			even++
		}
	}
	fmt.Fprintf(out, "  parity   %d even, %d odd\n", even, len(s)-even)
}

// ------------------------------------------------------------------- dump

// demoDumpHeader is written above the records unless -bare. It is the same bargain `house dump`
// makes: the text is meant to be read by somebody who has never seen the format, and the file
// they are looking at is the only place they are certain to look.
const demoDumpHeader = `# gliderGo demo dump -- Glider PRO's 'demo' resource, one record per line.
#
# Six bytes per record, big-endian: a 4-byte frame, a key byte, a padding byte.
# The frame is the value of the game's frame counter while the key was down, so a
# held key is a run of consecutive frames, one record each. Playback compares the
# frame for *equality* and advances one record, which is why two records on one
# frame would strand the cursor and silently kill the rest of the stream.
#
# key 0 is right and key 1 is left. GetDemoInput's own case comments say the
# opposite and are wrong; the recorder's call sites produced these bytes. See
# docs/analysis/input.md §14.2.
#
# pad is uninitialised heap from a 1994 Macintosh, not data. gap is this record's
# frame minus the previous one's: 1 means the key was still down.
#
`

func demoDump(args []string) error {
	fs := flag.NewFlagSet(prog+" demo dump", flag.ContinueOnError)
	bare := fs.Bool("bare", false, "omit the explanatory header comment")
	out := fs.String("o", "-", "write here instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("demo dump takes exactly one demo file")
	}
	s, err := demo.Load(fs.Arg(0))
	if err != nil {
		return err
	}
	dst, closeOut, err := openOut(*out)
	if err != nil {
		return err
	}
	if !*bare {
		fmt.Fprint(dst, demoDumpHeader)
	}
	w := tabwriter.NewWriter(dst, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "index\tframe\tgap\tkey\tcode\tpad")
	for i, r := range s {
		gap := "-"
		if i > 0 {
			gap = fmt.Sprintf("%d", r.Frame-s[i-1].Frame)
		}
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%d\t%d\n", i, r.Frame, gap, r.Key, byte(r.Key), r.Pad)
	}
	if err := w.Flush(); err != nil {
		closeOut()
		return err
	}
	return closeOut()
}

// ------------------------------------------------------------------- check

// demoCheck is the one demo command with an exit status, and it checks the two things that
// cannot be seen by reading the file: that the bytes survive a decode/encode round trip, and
// that playback would reach the last record.
//
// The second is the one that matters. A stream that stalls plays a shorter demo than it
// contains and reports nothing at all -- no error, no truncation, just a glider that stops
// responding -- so this is the only place the failure is ever visible.
func demoCheck(args []string) error {
	fs := flag.NewFlagSet(prog+" demo check", flag.ContinueOnError)
	quiet := fs.Bool("q", false, "print only files that fail")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("demo check takes one or more demo files")
	}

	bad := 0
	for _, path := range fs.Args() {
		raw, err := os.ReadFile(path)
		if err != nil {
			bad++
			fmt.Printf("%s: FAILED\n    error: %v\n", path, err)
			continue
		}
		s, err := demo.Decode(raw)
		if err != nil {
			bad++
			fmt.Printf("%s: FAILED\n    error: %v\n", path, err)
			continue
		}

		var problems, notes []string
		if err := s.Validate(); err != nil {
			problems = append(problems, err.Error())
		}
		if got := s.Encode(); len(got) != len(raw) {
			problems = append(problems, fmt.Sprintf(
				"re-encoding produced %d bytes, the file is %d", len(got), len(raw)))
		} else if d := firstDiff(raw, got); d >= 0 {
			problems = append(problems, fmt.Sprintf(
				"byte %d changed by a round trip: 0x%02X -> 0x%02X (record %d, %s)",
				d, raw[d], got[d], d/demo.RecordSize, demoField(d%demo.RecordSize)))
		}

		// The notes are all "legal, and worth knowing". None of them stops playback.
		if len(s) == 0 {
			notes = append(notes, "no records: playback is a game with no input")
		}
		if n := distinctPads(s); n > 1 {
			notes = append(notes, fmt.Sprintf(
				"%d distinct padding bytes -- uninitialised heap, as in the 1994 resource", n))
		}
		if len(raw) == demo.ShippedLength {
			notes = append(notes, fmt.Sprintf(
				"%d bytes: the length the original hard-codes as kDemoLength", demo.ShippedLength))
		}
		for k, n := range s.Keys() {
			if !k.Valid() {
				// Validate has already reported this as an error; the note says what
				// playback would actually do with it, which the error does not.
				notes = append(notes, fmt.Sprintf(
					"the %d record(s) with key %d are consumed without being acted on and "+
						"without clearing fireHeld, so they play as if absent", n, byte(k)))
			}
		}

		if len(problems) > 0 {
			bad++
			fmt.Printf("%s: FAILED\n", path)
		} else if !*quiet {
			fmt.Printf("%s: ok, %d records, %d..%d\n", path, len(s), first(s), s.LastFrame())
		} else {
			continue
		}
		for _, p := range problems {
			fmt.Printf("    error: %s\n", p)
		}
		for _, n := range notes {
			fmt.Printf("    note:  %s\n", n)
		}
	}
	if bad > 0 {
		return fmt.Errorf("%d of %d files failed", bad, fs.NArg())
	}
	return nil
}

// ------------------------------------------------------------------- helpers

// demoField names a byte offset within a record, for a round-trip failure message.
func demoField(off int) string {
	switch {
	case off < 4:
		return fmt.Sprintf("frame byte %d", off)
	case off == 4:
		return "key"
	}
	return "padding"
}

func first(s demo.Stream) int64 {
	if len(s) == 0 {
		return -1
	}
	return s[0].Frame
}

// demoStem is stem's counterpart for demo files. The extractor names resources <type>/<id>.bin,
// so the interesting part of the path is the id and the directory above it: "res/demo/128.bin"
// prints as "demo/128", which is what the resource is called in the analysis docs.
func demoStem(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".bin")
	if dir := filepath.Base(filepath.Dir(path)); dir != "." && dir != string(filepath.Separator) {
		return dir + "/" + base
	}
	return base
}
