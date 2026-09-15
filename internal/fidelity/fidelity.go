// Package fidelity is the pixel half of determinism, checked in.
//
// internal/replay answers "do two runs of this script agree?". That is the question a bug
// report needs and it is the cheaper one, because the answer is computed twice on the same
// machine in the same minute and nothing has to be stored. This package answers the other
// one: "does this build draw what the build that was looked at drew?" -- which cannot be
// answered without a file on disk, because the thing being compared against is the past.
//
// # Why hashes and not images
//
// The corpus is per-frame hashes of the three index planes, as text, one frame per line.
// Not PNGs, for three reasons:
//
//   - A 1,200-frame run is 1,200 hashes, which is 60KB of diffable text. The same run as
//     images is 300MB of binary blobs in git history, and a repository that a player is
//     meant to be able to clone.
//   - `git diff` on the corpus says *which frames changed*, which is the whole finding.
//     Frames 0-216 identical and 217 onwards different is a divergence at 217; every frame
//     different is a palette or a view change; three frames different in the middle of a
//     transition is a wipe that got faster.
//   - The indices are what the game composites, so hashing them and not the RGB catches a
//     palette-index change that happens to name the same colour -- internal/render's own
//     argument, and the reason replay.planeDigest works the same way.
//
// What a hash cannot do is show a human the difference, and that is what Snapshot is for: a
// failing test re-runs the script to the one frame that diverged and writes it as a PNG.
// Cheap, because it only ever happens on failure, and it is the frame you actually want
// rather than all 1,200.
//
// # What is in a reference
//
// Two shapes, one file format. A game reference is a replay script's frames, three columns
// (Main, Work, Back). A screens reference is the shell's own surfaces -- the splash, the
// house picker, the settings screen -- one column, one row per screen, which covers the
// pixels a player meets before the game starts and which no replay script can reach.
//
// The header is the run: the house, the seed, the frame count, the view size, whether sound
// was on. It is compared before the rows are, because a header mismatch explains every row
// mismatch under it, and a reader who does not know the seed changed will spend an afternoon
// looking for a physics bug.
package fidelity

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"glidergo/internal/render"
	"glidergo/internal/replay"
)

// Reference is a checked-in corpus: a description of the run, and one row per thing hashed.
type Reference struct {
	// Kind is what was recorded -- "frames" or "screens" -- and is the first line of the
	// file, so a reader knows what the rows mean before reading any.
	Kind string

	// Head is the run's settings as ordered key/value lines. Ordered because it is compared
	// line by line and printed as a diff, and a map would reorder it on every rewrite and
	// make every regeneration look like a change.
	Head []KV

	// Columns names the hash columns, left to right.
	Columns []string

	// Rows is the corpus. One per frame or per screen, in the order recorded.
	Rows []Row
}

// KV is one header line.
type KV struct{ Key, Value string }

// Row is one hashed thing: a frame number, or a screen's name.
type Row struct {
	Name   string
	Hashes []string
}

// Hash is the corpus's hash of one surface: the dimensions, then the index plane.
//
// Byte-for-byte the same function as replay.planeDigest, which is not duplication worth
// removing: that one is a run's own report and this one is a file format's contract. The
// day the trace wants a longer hash, this must not follow it, or every reference in
// testdata is invalidated by a change to a diagnostic.
//
// The dimensions are hashed first because a surface that came back the wrong size would
// otherwise collide with a correctly sized one whose extra rows happened to hold the white
// a fresh GWorld is filled with.
func Hash(s *render.Surface) string {
	if s == nil {
		return "-"
	}
	sum := sha256.New()
	fmt.Fprintf(sum, "%dx%d\n", s.W, s.H)
	sum.Write(s.Pix)
	return hex.EncodeToString(sum.Sum(nil))[:16]
}

// ---------------------------------------------------------------------------
// Recording
// ---------------------------------------------------------------------------

// RecordFrames runs a script and hashes the picture at the end of every frame.
//
// The run is a normal replay in every respect -- same seed, same clock, same absence of a
// tick source -- with a watcher on the side. Nothing about the recording can change what is
// recorded, which is the property that makes a reference worth having at all.
func RecordFrames(s *replay.Script) (*Reference, error) {
	ref := &Reference{
		Kind:    "frames",
		Columns: []string{"main", "work", "back"},
	}
	view := ""
	watch := func(sample replay.Sample, main, work, back *render.Surface) {
		if view == "" && main != nil {
			view = fmt.Sprintf("%dx%d", main.W, main.H)
		}
		ref.Rows = append(ref.Rows, Row{
			Name:   fmt.Sprintf("%d", sample.Frame),
			Hashes: []string{Hash(main), Hash(work), Hash(back)},
		})
	}

	res, err := replay.RunWatching(s, nil, watch)
	if err != nil {
		return nil, err
	}

	// The header. Everything here either changes the pixels or explains why they changed:
	// the house and the seed and the start point are the run, `sound` is in because a build
	// with no sound composes rooms differently (a trigger whose sample will not load gets no
	// hot spot -- replay.Script.Sound), and the digest ties the picture to the behaviour so
	// that a corpus failure can be read beside a trace failure and one of them ruled out.
	ref.Head = []KV{
		{"house", s.House},
		{"seed", fmt.Sprint(s.Seed)},
		{"frames", fmt.Sprint(s.Frames)},
		{"neighbors", fmt.Sprint(s.Neighbors)},
		{"players", map[bool]string{true: "2", false: "1"}[s.TwoPlayer]},
		{"clock", s.Clock.UTC().Format("2006-01-02T15:04:05Z")},
		{"sound", onOff(s.Sound)},
		{"music", onOff(s.Music)},
		{"view", view},
		{"trace-digest", res.Digest},
	}
	if s.Room >= 0 {
		ref.Head = append(ref.Head, KV{"start", fmt.Sprintf("room %d where %d,%d", s.Room, s.Where.H, s.Where.V)})
	}
	// The deviations are in the header and not a row, because they are the run's own list of
	// places it knows it is not the original (game.Diagnostics). A reference recorded while
	// one was in force is a reference of that behaviour, and the next person to regenerate
	// the file needs to see it appear or disappear.
	for _, d := range res.Diag.Seen {
		ref.Head = append(ref.Head, KV{"deviation", d.String()})
	}
	return ref, nil
}

// Shot is the three planes of one frame, kept.
type Shot struct {
	Frame            int64
	Main, Work, Back *render.Surface
}

// Snapshot re-runs a script and keeps the planes of one frame.
//
// This is the failure path and only the failure path: RecordFrames hashes and discards,
// because keeping 1,200 frames of three 512x322 planes to look at one of them is 600MB of
// nothing. A test that finds a divergence at frame 217 comes back here for 217 and writes
// three PNGs a human can open.
//
// The script is run from the start rather than resumed at the frame, deliberately: a resume
// is a different run (replay's one exception to "the script cannot express anything a player
// could not do"), and a picture of a different run is not evidence about this one.
func Snapshot(s *replay.Script, frame int64) (*Shot, error) {
	got := &Shot{Frame: frame}
	watch := func(sample replay.Sample, main, work, back *render.Surface) {
		if sample.Frame != frame || got.Main != nil {
			return
		}
		// Cloned, because Watch's surfaces are a scratch buffer the next frame overwrites.
		got.Main, got.Work, got.Back = main.Clone(), work.Clone(), back.Clone()
	}
	if _, err := replay.RunWatching(s, nil, watch); err != nil {
		return nil, err
	}
	if got.Main == nil {
		return nil, fmt.Errorf("fidelity: frame %d was never presented (the run is %d frames)", frame, s.Frames)
	}
	return got, nil
}

// WritePNGs writes a shot's three planes into dir and returns the paths written.
//
// Named for the frame and the plane, so that two shots of the same frame from two builds can
// sit in one directory and be flipped between.
func (sh *Shot) WritePNGs(dir, tag string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range []struct {
		name string
		surf *render.Surface
	}{
		{"main", sh.Main}, {"work", sh.Work}, {"back", sh.Back},
	} {
		if p.surf == nil {
			continue
		}
		path := filepath.Join(dir, fmt.Sprintf("%s-frame%06d-%s.png", tag, sh.Frame, p.name))
		if err := WritePNG(path, p.surf); err != nil {
			return paths, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// WritePNG saves one surface at 1:1.
//
// Unscaled, unlike cmd/glidergo's -shot: this image is evidence rather than a screenshot, and
// an integer upscale would put the viewer three pixels away from the pixel that changed.
func WritePNG(path string, s *render.Surface) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, s.ToPaletted()); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// ---------------------------------------------------------------------------
// The file format
// ---------------------------------------------------------------------------

// Write writes the reference in its checked-in form.
//
// Deliberately plain: `#` comments, `key value` header lines, then whitespace-separated rows.
// It is read by `git diff` far more often than by Parse.
func (r *Reference) Write(out io.Writer) error {
	bw := bufio.NewWriter(out)
	fmt.Fprintf(bw, "# gliderGo fidelity reference -- %s\n", r.Kind)
	fmt.Fprintf(bw, "# regenerate with `go test ./internal/fidelity -update` and read the diff:\n")
	fmt.Fprintf(bw, "# every changed line is a pixel that moved.\n")
	fmt.Fprintf(bw, "kind %s\n", r.Kind)
	for _, kv := range r.Head {
		fmt.Fprintf(bw, "%s %s\n", kv.Key, kv.Value)
	}
	fmt.Fprintf(bw, "columns %s\n", strings.Join(r.Columns, " "))
	for _, row := range r.Rows {
		fmt.Fprintf(bw, "%s %s\n", row.Name, strings.Join(row.Hashes, " "))
	}
	return bw.Flush()
}

// WriteFile writes the reference to a path.
func (r *Reference) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := r.Write(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Parse reads a reference back.
//
// A header key is anything before the `columns` line; everything after it is a row. That
// ordering is the file's only real rule, and it is what lets a new header key be added
// without a version number: an unknown key is a header line like any other, and the compare
// below reports it as one.
func Parse(in io.Reader) (*Reference, error) {
	r := &Reference{}
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	rows := false
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		switch {
		case rows:
			r.Rows = append(r.Rows, Row{Name: fields[0], Hashes: fields[1:]})
		case fields[0] == "columns":
			r.Columns, rows = fields[1:], true
		case fields[0] == "kind":
			if len(fields) != 2 {
				return nil, fmt.Errorf("fidelity: line %d: kind takes one word", line)
			}
			r.Kind = fields[1]
		default:
			r.Head = append(r.Head, KV{Key: fields[0], Value: strings.Join(fields[1:], " ")})
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !rows {
		return nil, fmt.Errorf("fidelity: no `columns` line: this is not a reference file")
	}
	return r, nil
}

// Load reads a reference from a path.
func Load(path string) (*Reference, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// ---------------------------------------------------------------------------
// Comparing
// ---------------------------------------------------------------------------

// Diff is what changed between a reference and a run, in the order a reader needs it.
type Diff struct {
	// Header lines that differ, appeared or vanished. Reported first and separately,
	// because a header change is a different run and explains everything below it.
	Header []string

	// First is the first row that differs, and the whole finding in most failures.
	First *RowDiff

	// Rows is how many rows differ in total, and Total how many were compared. One row of
	// 1,200 is a frame; 1,200 of 1,200 is the palette or the view.
	Rows, Total int

	// Missing and Extra are rows one side has and the other does not, named. A run that
	// stopped early is Missing; a longer run is Extra.
	Missing, Extra []string
}

// RowDiff is one row's disagreement.
type RowDiff struct {
	Name       string
	Column     string
	Want, Got  string
	Columns    []string
	WantHashes []string
	GotHashes  []string
}

// Compare reports how a fresh run differs from the reference, or nil for no difference.
//
// The receiver is the reference -- what is checked in, the "want" -- and the argument is the
// run. That direction is fixed because the error text says "want" and "got" and getting it
// backwards in a test would send the reader looking for the change in the wrong build.
func (r *Reference) Compare(got *Reference) *Diff {
	d := &Diff{}
	if r.Kind != got.Kind {
		d.Header = append(d.Header, fmt.Sprintf("kind: want %q, got %q", r.Kind, got.Kind))
	}
	d.Header = append(d.Header, headerDiff(r.Head, got.Head)...)
	if strings.Join(r.Columns, " ") != strings.Join(got.Columns, " ") {
		d.Header = append(d.Header, fmt.Sprintf("columns: want [%s], got [%s]",
			strings.Join(r.Columns, " "), strings.Join(got.Columns, " ")))
	}

	// By name and not by position: a run that lost a frame in the middle would otherwise
	// report every row after it as changed, which is true and useless.
	index := func(rows []Row) map[string]Row {
		m := make(map[string]Row, len(rows))
		for _, row := range rows {
			m[row.Name] = row
		}
		return m
	}
	wantRows, gotRows := index(r.Rows), index(got.Rows)
	for _, row := range r.Rows {
		g, ok := gotRows[row.Name]
		if !ok {
			d.Missing = append(d.Missing, row.Name)
			continue
		}
		d.Total++
		if strings.Join(row.Hashes, " ") == strings.Join(g.Hashes, " ") {
			continue
		}
		d.Rows++
		if d.First == nil {
			rd := &RowDiff{
				Name:       row.Name,
				Columns:    r.Columns,
				WantHashes: row.Hashes,
				GotHashes:  g.Hashes,
			}
			for i, h := range row.Hashes {
				if i < len(g.Hashes) && g.Hashes[i] == h {
					continue
				}
				rd.Column, rd.Want = column(r.Columns, i), h
				if i < len(g.Hashes) {
					rd.Got = g.Hashes[i]
				}
				break
			}
			d.First = rd
		}
	}
	for _, row := range got.Rows {
		if _, ok := wantRows[row.Name]; !ok {
			d.Extra = append(d.Extra, row.Name)
		}
	}

	if len(d.Header) == 0 && d.Rows == 0 && len(d.Missing) == 0 && len(d.Extra) == 0 {
		return nil
	}
	return d
}

// headerDiff compares two ordered header lists, reporting changed, missing and added keys.
// Repeated keys -- `deviation` is one -- are compared as a set of values under the key.
func headerDiff(want, got []KV) []string {
	group := func(kvs []KV) (map[string][]string, []string) {
		m := make(map[string][]string)
		var order []string
		for _, kv := range kvs {
			if _, seen := m[kv.Key]; !seen {
				order = append(order, kv.Key)
			}
			m[kv.Key] = append(m[kv.Key], kv.Value)
		}
		return m, order
	}
	wm, worder := group(want)
	gm, gorder := group(got)

	var out []string
	for _, k := range worder {
		gv, ok := gm[k]
		if !ok {
			out = append(out, fmt.Sprintf("%s: want %q, absent from the run", k, strings.Join(wm[k], ", ")))
			continue
		}
		w := append([]string(nil), wm[k]...)
		g := append([]string(nil), gv...)
		sort.Strings(w)
		sort.Strings(g)
		if strings.Join(w, "\x00") != strings.Join(g, "\x00") {
			out = append(out, fmt.Sprintf("%s: want %q, got %q", k, strings.Join(wm[k], ", "), strings.Join(gv, ", ")))
		}
	}
	for _, k := range gorder {
		if _, ok := wm[k]; !ok {
			out = append(out, fmt.Sprintf("%s: the run says %q, absent from the reference", k, strings.Join(gm[k], ", ")))
		}
	}
	return out
}

func column(names []string, i int) string {
	if i < len(names) {
		return names[i]
	}
	return fmt.Sprintf("column %d", i)
}

// String is the failure message: the header first, then the first diverging row, then the
// scale of it.
func (d *Diff) String() string {
	var b strings.Builder
	for _, h := range d.Header {
		fmt.Fprintf(&b, "header %s\n", h)
	}
	if d.First != nil {
		f := d.First
		fmt.Fprintf(&b, "first difference at %s: %s want %s, got %s\n", f.Name, f.Column, f.Want, f.Got)
		fmt.Fprintf(&b, "  reference %s: %s\n", f.Name, strings.Join(f.WantHashes, " "))
		fmt.Fprintf(&b, "  this run  %s: %s\n", f.Name, strings.Join(f.GotHashes, " "))
	}
	if d.Rows > 0 {
		fmt.Fprintf(&b, "%d of %d rows differ\n", d.Rows, d.Total)
	}
	if n := len(d.Missing); n > 0 {
		fmt.Fprintf(&b, "%d rows in the reference the run never produced (first %s)\n", n, d.Missing[0])
	}
	if n := len(d.Extra); n > 0 {
		fmt.Fprintf(&b, "%d rows the run produced that the reference does not have (first %s)\n", n, d.Extra[0])
	}
	return strings.TrimRight(b.String(), "\n")
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
