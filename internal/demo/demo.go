// Package demo is the attract-mode input stream: the 1994 game's `'demo'` resource, read,
// written, replayed and recorded.
//
// Glider PRO's attract mode is not a movie. It is a **recorded keystroke log** replayed
// through the ordinary physics -- `GetDemoInput` (Input.c:186-277) drives player one from the
// stream instead of from the keyboard, and everything downstream of that call is the same code
// a player's game runs. So the demo is worth having for a reason that has nothing to do with
// showing the game off on a shop counter: replaying one is a **frame-exact determinism test**.
// If the port's physics, its random stream or its frame clock drift by one step, the recorded
// glider stops flying the recorded path, and the trace says at which frame.
//
// That is why this package is a codec and a cursor and a recorder, and why the screen that
// plays a demo when the splash screen goes idle is not here: the attract mode is shell work
// (internal/shell, 1.7), the determinism harness is internal/replay, and both of them want the
// same 30 lines of stream handling.
//
// # The format
//
// One record is six bytes, big-endian:
//
//	long frame     4 bytes at +0    the gameFrame the key was down on
//	char key       1 byte  at +4    0 right, 1 left, 2 battery/helium, 3 rubber band
//	char padding   1 byte  at +5    never written; uninitialised heap residue
//
// `demoType`, GliderStructs.h:334-339, `sizeof` 6 under the Mac's `pack(2)` with a 4-byte
// `long`. The original loads it with `BlockMove` straight into a `NewPtr` block
// (StructuresInit2.c:280-298) -- **no byte swapping**, so the on-disk `long` is big-endian
// because the 68k and the PPC were, and a little-endian port has to swap it or read garbage.
// docs/analysis/input.md §14 has the derivation, including the elimination argument that
// proves the stride is 6 and not 8 (only 6-with-big-endian yields a monotonic frame sequence).
//
// # Two hazards this package exists to hold still
//
// **One record per frame, or playback dies.** Playback compares `gameFrame == recs[i].frame`
// and advances the cursor only on a match (Input.c:224-266). It is an equality test, so if two
// records share a frame the second one can never match: the cursor is left pointing at a frame
// number the game has already gone past, and **every later record is unreachable too**. The
// shipped resource has no repeated frame, which is what makes it playable. The original's own
// recorder could produce one anyway -- `LogDemoKey` is called from four independent branches of
// `GetInput` (Input.c:304, :321, :334, :348) and a player holding right while firing a band
// logs two records on one frame -- so a recording made with the shipped code could silently
// stop replaying a few seconds in. Recorder refuses to write that stream; see its Dropped.
//
// **The stream does not end the demo; the game does.** There is no terminator and no length
// field in the data, and the original does not bounds-check the cursor: when the records run
// out it keeps reading past the end of the block, where the comparison against heap slack
// almost always fails and playback quietly stops producing input (docs/analysis/input.md
// §14.4). Benign on a Mac `NewPtr`, a panic in Go. Cursor answers "no input this frame"
// instead and counts the refusals, so the port is safe and can still say how far past the end
// a run went. What actually stops the shipped demo is the glider dying: the last record is
// frame 3414, about 114 seconds in, and the death countdown ends the game some frames later.
package demo

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// RecordSize is `sizeof(demoType)`: four bytes of frame, a key and the padding byte.
const RecordSize = 6

// ShippedLength is `kDemoLength` (GliderDefines.h:625), the byte count the original allocates
// and BlockMoves.
//
// 6702, which is exactly 1117 records -- and it is 1117 records because it is the length of
// **one particular recording** that John Calhoun made and then hard-coded, not a buffer size
// (docs/analysis/input.md §14.8: the recording buffer was 2000 records and the dump wrote
// `sizeof(demoType) * demoIndex`). A port that treats it as a maximum would be wrong for any
// other stream, which is why nothing here reads this constant except the test that checks the
// shipped resource is still the file we think it is.
const ShippedLength = 6702

// ShippedRecords is ShippedLength in records: 1117.
const ShippedRecords = ShippedLength / RecordSize

// ShippedPath is where `make assets` puts `'demo'` 128, relative to assets/extracted.
//
// The extractor writes every resource as <type>/<id>.bin, so this is one instance of a general
// rule rather than a special case; it is named here because this is the only Go code that
// reads a raw resource out of that tree, and a string literal in three test files would be
// three places to fix when the extractor's layout changes.
const ShippedPath = "res/demo/128.bin"

// Key is one record's key byte: which of player one's four controls was down.
//
// The values are the recorder's, not the playback switch's. `GetDemoInput`'s comments on cases
// 0 and 1 say "left key" and "right key" and are **backwards** -- case 0's body applies
// rightward thrust and sets `heldRight` -- while `LogDemoKey(0)` is called from the right-key
// branch. The call sites win because they produced the shipped bytes. Do not "fix" the
// comments into code; docs/analysis/input.md §14.2 spells this out at length because it is the
// single easiest way to mirror-image a port of the demo.
type Key byte

// The four codes. The identical set is spelled out again in internal/game/player as the type
// of the recorder hook, so that the player package can stay import-free; a test in
// internal/game pins the two together.
const (
	KeyRight Key = 0
	KeyLeft  Key = 1
	KeyBatt  Key = 2 // battery or helium, decided by the sign of the shared counter
	KeyBand  Key = 3
)

func (k Key) String() string {
	switch k {
	case KeyRight:
		return "right"
	case KeyLeft:
		return "left"
	case KeyBatt:
		return "batt"
	case KeyBand:
		return "band"
	}
	return fmt.Sprintf("key(%d)", byte(k))
}

// Valid reports whether k is one of the four codes playback acts on.
//
// Anything else is not an error in the data: `GetDemoInput`'s switch has no default, so a
// record with key 7 is consumed, does nothing, and -- this is the part worth knowing -- does
// **not** clear `fireHeld`, because the clear lives in the `else` of the frame test rather
// than in the switch. Decode keeps such records; Validate reports them.
func (k Key) Valid() bool { return k <= KeyBand }

// Record is one `demoType`.
type Record struct {
	// Frame is the C's `long frame`: the value of gameFrame when the key was logged. The
	// game's frame counter is an int64 here (World.Frame) and a 32-bit long on disk, so
	// Validate is what refuses a frame that would not survive being written back.
	Frame int64

	// Key is the control that was down.
	Key Key

	// Pad is `padding`, which `LogDemoKey` never writes (Input.c:46-47). In the shipped
	// resource it is uninitialised heap: 109 distinct values including runs of ASCII
	// (`'r'`, `'o'`, `'E'`), i.e. whatever the Mac's heap happened to hold at that address.
	//
	// It is kept and round-tripped for the reason internal/house keeps its residue: a
	// decoder that dropped it could not prove it had read the file correctly, because
	// re-encoding would not reproduce the bytes. Nothing reads it as data -- a port must
	// ignore it, and a recorder should write zero.
	Pad byte
}

// Stream is a whole demo, in record order.
type Stream []Record

// Decode reads a stream from the raw resource bytes.
//
// The only structural error possible is a length that is not a multiple of six: there is no
// header, no magic and no version, so a file of the wrong kind that happens to divide by six
// decodes into nonsense records rather than failing. Validate is the check that catches that,
// and the shipped-resource test in this package is what pins the one file that matters.
func Decode(b []byte) (Stream, error) {
	if len(b)%RecordSize != 0 {
		return nil, fmt.Errorf("demo: %d bytes is not a whole number of %d-byte records (%d left over)",
			len(b), RecordSize, len(b)%RecordSize)
	}
	s := make(Stream, 0, len(b)/RecordSize)
	for i := 0; i < len(b); i += RecordSize {
		s = append(s, Record{
			Frame: int64(int32(binary.BigEndian.Uint32(b[i : i+4]))),
			Key:   Key(b[i+4]),
			Pad:   b[i+5],
		})
	}
	return s, nil
}

// Read decodes a stream from a reader.
func Read(r io.Reader) (Stream, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Decode(b)
}

// Load reads a stream from a file.
func Load(path string) (Stream, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := Decode(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// LoadFS reads a stream out of a filesystem: the copy built into the executable, or a
// directory somebody named.
func LoadFS(fsys fs.FS, name string) (Stream, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	s, err := Decode(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return s, nil
}

// LoadShipped reads `'demo'` 128 out of an extracted asset tree.
//
// tree is the assets/extracted directory, or the built-in copy of it. A missing file is
// returned as it comes from the filesystem, so a caller can test it with errors.Is against
// fs.ErrNotExist and carry on without a demo -- which is what a tree with the assets removed
// has, and what the shell's attract mode has to survive.
func LoadShipped(tree fs.FS) (Stream, error) {
	return LoadFS(tree, ShippedPath)
}

// Encode writes the stream in the original's on-disk form.
//
// Frames are truncated to 32 bits, because the field is a 32-bit big-endian long and this
// function has no way to report anything. Validate is what refuses a stream that would not
// survive the trip, and WriteTo calls it -- so the truncation is reachable only by a caller
// who deliberately skipped the check.
func (s Stream) Encode() []byte {
	b := make([]byte, len(s)*RecordSize)
	for i, r := range s {
		binary.BigEndian.PutUint32(b[i*RecordSize:], uint32(int32(r.Frame)))
		b[i*RecordSize+4] = byte(r.Key)
		b[i*RecordSize+5] = r.Pad
	}
	return b
}

// WriteTo validates the stream and writes it.
//
// Validating on the way out and not on the way in is deliberate. Reading is forensics -- the
// shipped resource is what it is, and a decoder that refused a file could not be used to find
// out what was wrong with it. Writing is publication: a stream with two records on one frame
// is a file that stops replaying halfway through with no error anywhere, and the last moment
// anybody can be told about it is here.
func (s Stream) WriteTo(w io.Writer) (int64, error) {
	if err := s.Validate(); err != nil {
		return 0, err
	}
	n, err := w.Write(s.Encode())
	return int64(n), err
}

// WriteFile writes the stream to a path, validated.
func (s Stream) WriteFile(path string) error {
	if err := s.Validate(); err != nil {
		return err
	}
	return os.WriteFile(path, s.Encode(), 0o644)
}

// Validate reports the first thing about a stream that would make playback go wrong.
//
// Three rules, and only the first is about the file being well formed:
//
//   - Frames are non-negative and fit in the 32-bit field they are written to.
//   - Frames strictly increase. This is the rule that matters: playback tests
//     `gameFrame == recs[i].frame` and never rewinds, so a repeated or descending frame
//     strands the cursor and silently kills every record after it (see the package comment).
//   - Keys are one of the four. Not fatal -- an unknown key is consumed and ignored -- so it
//     is reported last and only after the ordering, because a stream with both problems has
//     the ordering problem.
func (s Stream) Validate() error {
	for i, r := range s {
		if r.Frame < 0 || r.Frame > 0x7fffffff {
			return fmt.Errorf("demo: record %d: frame %d does not fit the 32-bit field", i, r.Frame)
		}
		if i > 0 && r.Frame <= s[i-1].Frame {
			return fmt.Errorf("demo: record %d: frame %d does not follow %d -- playback stalls "+
				"at the first repeat (one action per frame)", i, r.Frame, s[i-1].Frame)
		}
	}
	for i, r := range s {
		if !r.Key.Valid() {
			return fmt.Errorf("demo: record %d: key %d is not one of 0..3", i, byte(r.Key))
		}
	}
	return nil
}

// LastFrame is the frame of the last record, or -1 for an empty stream.
//
// It is the length of the *input*, not of the demo: what ends the shipped demo is the glider
// dying, some frames after the last key. Do not use this as a run length.
func (s Stream) LastFrame() int64 {
	if len(s) == 0 {
		return -1
	}
	return s[len(s)-1].Frame
}

// Keys counts the records of each key code, for a tool that wants to describe a stream.
func (s Stream) Keys() map[Key]int {
	m := make(map[Key]int, 4)
	for _, r := range s {
		m[r.Key]++
	}
	return m
}

// ---------------------------------------------------------------------------
// Playback
// ---------------------------------------------------------------------------

// Cursor is `demoIndex` (Input.c:33): a stream and a position in it.
//
// It is a separate type from the stream because the position is per *game* -- `NewGame` resets
// it (Play.c:114) -- while the stream is loaded once and could in principle be replayed by two
// worlds at once. It is also the only place the original's out-of-bounds read is stood in for,
// which is a thing worth being able to point at.
type Cursor struct {
	s    Stream
	i    int
	past int
}

// Cursor returns a cursor at the start of the stream.
func (s Stream) Cursor() *Cursor { return &Cursor{s: s} }

// Key is Input.c:224-266's test and advance: the key for this frame, if the next record is
// this frame's.
//
// The equality test is the original's and is load-bearing, not defensive. A `>=` here would be
// a kinder cursor and a different game: it would make a stream with a repeated frame limp on
// instead of stalling, and it would let a demo whose recording skipped a frame number apply a
// stale key on the next one. Both are observable, so the compare stays as it is and Validate
// is what keeps the streams this port writes out of that territory.
//
// A frame with no record is (0, false) and the caller must clear `fireHeld` -- which is why the
// bool is returned rather than a valid-looking key.
func (c *Cursor) Key(frame int64) (Key, bool) {
	if c.i >= len(c.s) {
		// The original reads demoData[demoIndex] here regardless, off the end of the
		// allocation. Counting instead is the whole of the port's deviation.
		c.past++
		return 0, false
	}
	if c.s[c.i].Frame != frame {
		return 0, false
	}
	k := c.s[c.i].Key
	c.i++
	return k, true
}

// Reset puts the cursor back to the first record: `demoIndex = 0` (Play.c:114).
//
// The past-the-end count is **not** reset, because it is a diagnostic about the process and
// not about the game -- a run that played the demo twice and went off the end twice should say
// two, not one. Nothing in the simulation reads it.
func (c *Cursor) Reset() { c.i = 0 }

// Index is the cursor's position: how many records have been consumed.
func (c *Cursor) Index() int { return c.i }

// Len is the number of records in the stream.
func (c *Cursor) Len() int { return len(c.s) }

// Done reports whether every record has been consumed.
func (c *Cursor) Done() bool { return c.i >= len(c.s) }

// PastEnd is how many frames asked for a key after the stream ran out -- i.e. how many reads
// past the end of the block the original would have performed.
//
// Nonzero is normal for a demo that outlives its recording, and it is exactly what a released
// build wants to be able to say instead of crashing.
func (c *Cursor) PastEnd() int { return c.past }

// Stream is the records the cursor is walking.
func (c *Cursor) Stream() Stream { return c.s }

// ---------------------------------------------------------------------------
// Recording
// ---------------------------------------------------------------------------

// Recorder is `LogDemoKey` (Input.c:44-49) with the original's two footguns removed.
//
// The C writes into a fixed 2000-record block with no bounds check and appends a record for
// every branch of `GetInput` that fires, so a session longer than 2000 keystrokes corrupts the
// heap and a session where two keys were down on one frame produces a stream that stops
// replaying at that frame (see the package comment). This grows, and it keeps **one record per
// frame** and counts what it dropped, so that a recording either replays to its end or says
// why not.
//
// Which of two same-frame actions survives is "the first one logged", and that is not
// arbitrary: the hook is called from the four sites the C calls `LogDemoKey` from, in the C's
// order -- direction, then battery, then band -- so a player holding right and firing records
// as holding right, exactly as the original's own recorder would have if it could only write
// one. It is a real loss of information and the reason Dropped exists rather than being
// silent.
type Recorder struct {
	recs    Stream
	dropped int
	last    int64
	started bool
}

// NewRecorder returns an empty recorder.
func NewRecorder() *Recorder { return &Recorder{} }

// Log records that key was down on frame, unless this frame already has a record.
//
// A frame at or before the previous record's is dropped and counted too. That cannot happen
// from a running game -- World.Frame only goes up -- so it is here to keep the invariant a
// property of the type rather than of its callers, and because the recorder is the one thing
// that decides whether a stream is playable.
func (r *Recorder) Log(frame int64, k Key) {
	if r.started && frame <= r.last {
		r.dropped++
		return
	}
	r.recs = append(r.recs, Record{Frame: frame, Key: k})
	r.last, r.started = frame, true
}

// Stream is what has been recorded so far. It shares memory with the recorder, so a caller
// that keeps it across further Log calls should copy it.
func (r *Recorder) Stream() Stream { return r.recs }

// Dropped is how many logged actions were discarded because the frame already had one.
//
// Nonzero means the recorded stream is a simplification of what the player did: the second key
// of a same-frame pair is gone. That is a fidelity fact about the *recording*, not a bug in
// the recorder, and it is reported so that a tool can print it beside the record count.
func (r *Recorder) Dropped() int { return r.dropped }
