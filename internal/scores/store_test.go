package scores

// The side-car: where it goes, what it is called, and -- the part that matters -- what it
// does with a file that is not what it expected.
//
// The original's reader is the reason half of this file exists. `ReadScoresFromDisk` asks
// the filesystem how big the file is and reads that many bytes into the middle of the house
// structure in memory (docs/analysis/scoring.md 7.14), so a side-car that had grown by a
// byte overwrote the saved game behind it and a large one walked off the end of the
// allocation. That is not a hypothetical: the file lives in the player's own data
// directory, where a full disk, a half-finished restore from backup, or somebody with a hex
// editor and an opinion about the leaderboard can all reach it.

import (
	"bytes"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/internal/datadir"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/prefs"
)

// ------------------------------------------------------------------ directory

// The overrides, in order. Each is checked with the others cleared, because "GLIDERGO_DATA
// wins" is only meaningful if GLIDERGO_CONFIG was set at the same time.
func TestDirHonoursTheOverridesInOrder(t *testing.T) {
	t.Setenv("GLIDERGO_DATA", "/tmp/glider-data")
	t.Setenv("GLIDERGO_CONFIG", "/tmp/glider-config")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	if got, err := Dir(); err != nil || got != "/tmp/glider-data" {
		t.Errorf("with GLIDERGO_DATA set, Dir() = %q, %v", got, err)
	}

	t.Setenv("GLIDERGO_DATA", "")
	want := filepath.Join("/tmp/glider-config", SubDir)
	if got, err := Dir(); err != nil || got != want {
		t.Errorf("with GLIDERGO_CONFIG set, Dir() = %q, %v; want %q", got, err, want)
	}

	t.Setenv("GLIDERGO_CONFIG", "")
	if runtime.GOOS == "linux" {
		want = filepath.Join("/tmp/xdg", "glidergo", SubDir)
		if got, err := Dir(); err != nil || got != want {
			t.Errorf("with XDG_DATA_HOME set, Dir() = %q, %v; want %q", got, err, want)
		}
	}
}

// A player who points GLIDERGO_CONFIG at one directory gets one directory. This checks the
// two packages agree about that variable, so if prefs ever stops honouring it the mismatch
// is a failure here rather than settings in one place and scores in another.
func TestDirAndPrefsAgreeAboutGLIDERGO_CONFIG(t *testing.T) {
	t.Setenv("GLIDERGO_DATA", "")
	t.Setenv("GLIDERGO_CONFIG", "/tmp/glider-portable")

	pd, err := prefs.Dir()
	if err != nil {
		t.Fatalf("prefs.Dir: %v", err)
	}
	if pd != "/tmp/glider-portable" {
		t.Fatalf("prefs.Dir() = %q; it no longer uses GLIDERGO_CONFIG verbatim, so "+
			"scores.Dir is deriving its path from the wrong root", pd)
	}
	sd, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if !strings.HasPrefix(sd, pd+string(filepath.Separator)) {
		t.Errorf("scores go to %q, which is not inside the configured %q", sd, pd)
	}
}

// The default is a *data* directory, not a config one: on Linux ~/.local/share, so a
// dotfile setup that syncs ~/.config does not pick up a game's high scores.
func TestDirDefaultsToTheDataDirectory(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the hand-rolled XDG path is the Linux branch")
	}
	t.Setenv("GLIDERGO_DATA", "")
	t.Setenv("GLIDERGO_CONFIG", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/tester")

	want := filepath.Join("/home/tester", ".local", "share", "glidergo", SubDir)
	if got, err := Dir(); err != nil || got != want {
		t.Errorf("Dir() = %q, %v; want %q", got, err, want)
	}

	// The spec says a relative $XDG_DATA_HOME is to be treated as unset, which matters
	// because a relative path would put the boards wherever the game happened to be
	// started from.
	t.Setenv("XDG_DATA_HOME", "relative/path")
	if got, err := Dir(); err != nil || got != want {
		t.Errorf("with a relative XDG_DATA_HOME, Dir() = %q, %v; want %q", got, err, want)
	}
}

// ------------------------------------------------------------------ file name

// The names a house can actually have. Three of them come from the shipped set (a space, an
// apostrophe, a Mac Roman high byte); the rest are the cases a filesystem cares about.
var trickyNames = []string{
	"", " ", "  ", "Art Museum", "Ozma's Revenge", "Slumberland",
	"CON", "con", "Con", "NUL", "com1", "LPT9", "CONS", "CO",
	".", "..", "...", "%", "%41", "A", "a",
	"a/b", `a\b`, "a:b", "C:house", "a*b", "a?b", `a"b`, "a<b", "a>b", "a|b",
	"trailing ", " leading", "trailing.", "tab\there", "nl\nhere", "nul\x00here",
	"Caf\xE9", "\xC4", "\xA5\xA5\xA5", // Mac Roman: e-acute, florin, three bullets
	strings.Repeat("x", 200), strings.Repeat("x", 201),
	strings.Repeat("\xC4", 200),
}

// Two different houses must never share a board. Everything else about the encoding is
// negotiable; this is not.
func TestFileNameIsInjective(t *testing.T) {
	seen := map[string]string{}
	for _, name := range trickyNames {
		f := FileName(name)
		if prev, dup := seen[f]; dup {
			t.Errorf("%q and %q both map to %q", prev, name, f)
			continue
		}
		seen[f] = name
	}
	// And on a case-folding filesystem, which is where injectivity is easiest to lose.
	// "CON" and "con" must not collapse just because the reserved-name escape only fires
	// for one of them.
	folded := map[string]string{}
	for _, name := range trickyNames {
		f := strings.ToLower(FileName(name))
		if prev, dup := folded[f]; dup && !strings.EqualFold(prev, name) {
			t.Errorf("%q and %q differ by more than case but collide on a case-folding "+
				"filesystem (%q)", prev, name, f)
		}
		folded[f] = name
	}
}

// Every name a house can have has to produce a file name all three target platforms will
// open. The escape set is the intersection of what they accept, so this is a check that
// nothing got added to it.
func TestFileNameIsPortable(t *testing.T) {
	for _, name := range trickyNames {
		f := FileName(name)
		if !strings.HasSuffix(f, Ext) {
			t.Errorf("FileName(%q) = %q, which has no %s", name, f, Ext)
		}
		if f == Ext {
			t.Errorf("FileName(%q) = %q, which is an extension and no name", name, f)
		}
		if f == "." || f == ".." {
			t.Errorf("FileName(%q) = %q", name, f)
		}
		for i := 0; i < len(f); i++ {
			c := f[i]
			if !datadir.SafeByte(c) && c != '%' {
				t.Errorf("FileName(%q) = %q, which holds byte %02X at %d", name, f, c, i)
			}
		}
		if base := strings.TrimSuffix(f, Ext); datadir.Reserved(base) {
			t.Errorf("FileName(%q) = %q, a Windows device name", name, f)
		}
		if len(f) > 120 {
			t.Errorf("FileName(%q) is %d bytes long", name, len(f))
		}
	}
}

// Spot checks, so that a rewrite of the escaper that stayed injective but changed the
// spelling is caught: the file names are user-visible and a rename would orphan every board
// a player already has.
func TestFileNameSpotChecks(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Art Museum", "Art%20Museum.scores"},
		{"Slumberland", "Slumberland.scores"},
		{"Ozma's Revenge", "Ozma%27s%20Revenge.scores"},
		{"", "%.scores"},
		{"CON", "%43ON.scores"},
		{"con", "%63on.scores"},
		{"a/b", "a%2Fb.scores"},
		{"Caf\xE9", "Caf%E9.scores"},
	} {
		if got := FileName(tc.in); got != tc.want {
			t.Errorf("FileName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ----------------------------------------------------------------- round trip

func TestSaveThenLoadReturnsTheSameBoard(t *testing.T) {
	st := OpenDir(t.TempDir())
	want := board(900, 800, 700)
	want.Banner.SetText("Well played")

	if err := st.Save("Art Museum", &want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, notes := st.Load("Art Museum", house.Scores{})
	if len(notes) != 0 {
		t.Errorf("Load complained about a board it had just written: %v", notes)
	}
	if got != want {
		t.Errorf("the board came back different")
	}

	// Byte-exact, residue included: a board read and written back must not be a diff, or
	// every launch rewrites the file.
	if err := st.Save("Art Museum", &got); err != nil {
		t.Fatalf("Save: %v", err)
	}
	again, _ := st.Load("Art Museum", house.Scores{})
	if again != want {
		t.Error("a second save/load cycle changed the board")
	}
}

// Exactly the original's 292 bytes and nothing else in the file, so a board written here
// could be read by a 1994 build (with its houseIsReadOnly bug fixed) and vice versa.
func TestSaveWritesExactlyTheOriginalsBytes(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)
	s := board(900, 800)
	if err := st.Save("House", &s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "House"+Ext))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != house.SizeofScores {
		t.Errorf("the side-car is %d bytes, want %d", len(data), house.SizeofScores)
	}
	if !bytes.Equal(data, house.EncodeScores(&s)) {
		t.Error("the file is not the board's own encoding")
	}
}

// The write is atomic, so an interrupted save leaves the previous board rather than half of
// one -- and it leaves no litter behind, which the original's in-place FSWrite plus SetEOF
// did not have to think about.
func TestSaveLeavesNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)
	s := board(900)
	for i := 0; i < 3; i++ {
		if err := st.Save("House", &s); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != "House"+Ext {
		var got []string
		for _, e := range ents {
			got = append(got, e.Name())
		}
		t.Errorf("the directory holds %v, want just the board", got)
	}
}

// Save creates its directory on the way, which is CreateScoresFolder's job in the original
// (HighScores.c:642-661). A player's first high score must not need them to have made a
// folder.
func TestSaveCreatesTheDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not", "yet", "there")
	st := OpenDir(dir)
	s := board(900)
	if err := st.Save("House", &s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "House"+Ext)); err != nil {
		t.Error(err)
	}
}

// ----------------------------------------------------------------------- load

// The ordinary first run: no side-car, so the board is the one the house file carries. This
// is why a house that has had a score in it since 1995 shows that score the first time it
// is opened here.
func TestLoadOfAMissingFileIsTheSeed(t *testing.T) {
	st := OpenDir(t.TempDir())
	seed := board(1000, 900)
	got, notes := st.Load("Never Played", seed)
	if len(notes) != 0 {
		t.Errorf("a first run produced notes: %v", notes)
	}
	if got != seed {
		t.Error("the seed board was not returned unchanged")
	}
}

// The original's worst bug, and the reason Overlay exists. A 400-byte side-car was read as
// 400 bytes into a 292-byte member, over the top of savedGame, hasGame, firstRoom, nRooms
// and the start of the room array.
func TestLoadClampsAnOverlongFile(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)
	want := board(900, 800, 700)

	data := house.EncodeScores(&want)
	data = append(data, bytes.Repeat([]byte{0xEE}, 4096)...)
	if err := os.WriteFile(filepath.Join(dir, "House"+Ext), data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, notes := st.Load("House", house.Scores{})
	if got != want {
		t.Error("the board was not the first 292 bytes of the file")
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "ignored the rest") {
		t.Errorf("notes are %v, want one saying the tail was ignored", notes)
	}
}

// A short file is an overlay, not a truncation: what the file holds wins and the seed
// supplies the rest. That is the original's own behaviour, since it read *into* the house's
// board, and it is worth keeping -- a file cut short by a full disk still yields a board.
func TestLoadOverlaysATruncatedFile(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)

	full := board(900, 800, 700, 600, 500, 400, 300, 200, 100, 50)
	seed := board(1000, 950)

	// The banner and the first two names, and nothing else: 32 + 2*16 bytes.
	if err := os.WriteFile(filepath.Join(dir, "House"+Ext),
		house.EncodeScores(&full)[:64], 0o644); err != nil {
		t.Fatal(err)
	}

	got, notes := st.Load("House", seed)
	if len(notes) == 0 || !strings.Contains(notes[0], "only 64 of 292") {
		t.Errorf("notes are %v, want one about the length", notes)
	}
	if got.Banner != full.Banner {
		t.Errorf("the banner came from the seed, not the file")
	}
	if got.Names[0] != full.Names[0] || got.Names[1] != full.Names[1] {
		t.Error("the first two names did not come from the file")
	}
	// Everything past byte 64 is the seed's, so the scores are the seed's -- which means
	// the result is still a sorted, playable board rather than a mixture that crashes.
	if got.Scores != seed.Scores {
		t.Errorf("scores are %v, want the seed's %v", got.Scores, seed.Scores)
	}
	if !Sorted(&got) {
		t.Errorf("a truncated file produced an unsorted board: %v", got.Scores)
	}
}

// A hand-edited file. Somebody puts themselves at the top by typing a big number in, gets
// the byte order wrong, and ends up with a negative score -- which is the sort's own
// "consumed" marker and would otherwise make the row invisible to it.
func TestLoadRepairsATamperedFile(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)

	s := board(900, 800, 700)
	s.Scores[0] = -1
	s.Scores[1] = 5
	s.Levels[2] = -30
	if err := os.WriteFile(filepath.Join(dir, "House"+Ext),
		house.EncodeScores(&s), 0o644); err != nil {
		t.Fatal(err)
	}

	got, notes := st.Load("House", house.Scores{})
	if len(notes) < 2 {
		t.Errorf("notes are %v, want one per repair", notes)
	}
	for _, n := range notes {
		if !strings.HasPrefix(n, "House"+Ext+": ") {
			t.Errorf("note %q does not say which file it is about", n)
		}
	}
	if !Sorted(&got) {
		t.Errorf("the repaired board is unsorted: %v", got.Scores)
	}
	for i, v := range got.Scores {
		if v < 0 {
			t.Errorf("row %d still holds %d", i, v)
		}
	}
	if got.Levels[0] < 0 {
		t.Errorf("a negative room count survived: %d", got.Levels[0])
	}
}

// The blanket property, which is the one worth having: whatever is in that file, Load
// returns a board that is safe to sort, safe to insert into, and safe to draw. No panic, no
// error, no negative score, no unsorted board.
func TestLoadSurvivesArbitraryGarbage(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)
	path := filepath.Join(dir, "House"+Ext)
	seed := board(900, 800)
	rng := rand.New(rand.NewSource(1994))

	for i := 0; i < 500; i++ {
		data := make([]byte, rng.Intn(600))
		rng.Read(data)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}

		got, _ := st.Load("House", seed)
		if !Sorted(&got) {
			t.Fatalf("%d bytes of garbage produced an unsorted board: %v",
				len(data), got.Scores)
		}
		for row, v := range got.Scores {
			if v < 0 {
				t.Fatalf("%d bytes of garbage left row %d at %d", len(data), row, v)
			}
		}
		if got.Levels[0] < 0 {
			t.Fatalf("%d bytes of garbage left %d rooms", len(data), got.Levels[0])
		}
		// And the board still works: a new score goes on it, lands in the row Insert
		// returns, and leaves the board sorted. (Not necessarily row 0 -- garbage can
		// contain scores near MaxInt32, and a tie does not displace the row it tied.)
		placing := Insert(&got, "Player", math.MaxInt32, 5, time.Unix(0, 0))
		if placing < 0 {
			if got.Scores[0] != math.MaxInt32 {
				t.Fatalf("%d bytes of garbage: MaxInt32 did not qualify against %v",
					len(data), got.Scores)
			}
		} else if got.Scores[placing] != math.MaxInt32 {
			t.Fatalf("%d bytes of garbage: Insert returned row %d, which holds %d",
				len(data), placing, got.Scores[placing])
		}
		if !Sorted(&got) {
			t.Fatalf("%d bytes of garbage: Insert left the board unsorted: %v",
				len(data), got.Scores)
		}
	}
}

// An unreadable file is not a crash and not an empty board: it is the house's own board,
// plus a note for the log. A directory where the file should be is the easiest way to
// produce a read error without needing root or a permission trick that CI might not allow.
func TestLoadOfAnUnreadableFileFallsBackToTheSeed(t *testing.T) {
	dir := t.TempDir()
	st := OpenDir(dir)
	if err := os.Mkdir(filepath.Join(dir, "House"+Ext), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := board(900)
	got, notes := st.Load("House", seed)
	if got != seed {
		t.Error("the seed was not returned")
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "cannot read") {
		t.Errorf("notes are %v, want one saying it could not read the file", notes)
	}
}

// ------------------------------------------------------------------ nil store

// A build with nowhere to keep scores can still show them. `-shot` and the tests are both
// this case, and so is a read-only installation.
func TestANilStoreLoadsTheSeedAndRefusesToSave(t *testing.T) {
	var st *Store
	seed := board(900, 800)

	got, notes := st.Load("House", seed)
	if got != seed || len(notes) != 0 {
		t.Errorf("a nil store did not return the seed quietly: %v", notes)
	}
	if st.Dir() != "" || st.Path("House") != "" {
		t.Errorf("a nil store claims a path: %q %q", st.Dir(), st.Path("House"))
	}
	err := st.Save("House", &seed)
	if err == nil {
		t.Fatal("a nil store accepted a save")
	}
	if !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("the error is %q, which does not explain itself", err)
	}
}

// ------------------------------------------------------------------- seeding

// The end-to-end story for a shipped house: the board in the file is what a player sees
// first, their own game goes on top, and what comes back is the merged board.
func TestAShippedBoardIsSeededThenAddedTo(t *testing.T) {
	all := shippedBoards(t)
	var pick shipped
	for _, c := range all {
		if Occupied(&c.board) >= 3 {
			pick = c
			break
		}
	}
	if pick.name == "" {
		t.Skip("no shipped house has three scores on its board")
	}

	st := OpenDir(t.TempDir())
	got, notes := st.Load(pick.name, pick.board)
	if len(notes) != 0 {
		t.Errorf("%s: %v", pick.name, notes)
	}
	if got != pick.board {
		t.Fatalf("%s: the seeded board differs from the house's own", pick.name)
	}

	best := got.Scores[0]
	placing := Insert(&got, "gliderGo", best+100, 99,
		time.Date(2026, time.September, 16, 9, 0, 0, 0, time.UTC))
	if placing != 0 {
		t.Fatalf("beating the house record placed at row %d", placing)
	}
	if err := st.Save(pick.name, &got); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// And the house file's own board is still what it was -- the whole point of the
	// side-car is that the vendored houses are never written to.
	back, _ := st.Load(pick.name, pick.board)
	if back != got {
		t.Error("the merged board did not survive the round trip")
	}
	if back.Scores[1] != best {
		t.Errorf("the house's old record is gone: row 1 holds %d, want %d",
			back.Scores[1], best)
	}
}
