package house

// The most a house can be, and what happens to a file that is more (docs/IMPROVEMENTS.md 4.36).
// nRooms is a short, so a house is at most MaxFileSize bytes. Past that nothing can load, and the
// point of these is that nothing tries to: a file is refused before it is read into memory, and a
// text before its rooms are.

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestTheLargestAHouseCanBeIsWhatNRoomsCanCount(t *testing.T) {
	if MaxFileSize != 11_403_784 {
		t.Errorf("MaxFileSize = %d, want 866 + 348*32767 + 2 = 11,403,784", MaxFileSize)
	}
}

// The largest house there can be still opens: every room nRooms can count, and the PowerPC slack.
func TestTheLargestHouseStillLoads(t *testing.T) {
	path := peekFixture(t, "Largest.house", MaxRooms, func(b []byte) []byte {
		return append(b, 0, 0)
	})
	if st, err := os.Stat(path); err != nil || st.Size() != MaxFileSize {
		t.Fatalf("the fixture is %v bytes (%v), want %d", st.Size(), err, MaxFileSize)
	}
	if _, err := PeekFile(path); err != nil {
		t.Errorf("PeekFile: %v", err)
	}
	if h, err := LoadFile(path); err != nil || len(h.Rooms) != MaxRooms {
		t.Errorf("LoadFile: %v", err)
	}
}

// A byte more is refused from its size, by the picker's sniff as well as by Load. The file is
// sparse, so the test writes a header and the operating system makes up the rest.
func TestAFileLargerThanAHouseIsRefusedBeforeItIsRead(t *testing.T) {
	path := peekFixture(t, "Huge.house", 3, nil)
	if err := os.Truncate(path, MaxFileSize+1); err != nil {
		t.Skipf("cannot make a sparse file here: %v", err)
	}
	if _, err := PeekFile(path); err == nil || !strings.Contains(err.Error(),
		"larger than a house can be") {
		t.Errorf("PeekFile on a %d-byte file: %v", MaxFileSize+1, err)
	}
	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(),
		"larger than a house can be") {
		t.Errorf("LoadFile on a %d-byte file: %v", MaxFileSize+1, err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"Huge.house": {Data: b}}
	if _, err := PeekFS(fsys, "Huge.house"); err == nil || !strings.Contains(err.Error(),
		"larger than a house can be") {
		t.Errorf("PeekFS: %v", err)
	}
	if _, err := LoadFS(fsys, "Huge.house"); err == nil || !strings.Contains(err.Error(),
		"larger than a house can be") {
		t.Errorf("LoadFS: %v", err)
	}
}

// endless is a filesystem with one file that never ends and says it is empty, which is what a
// pipe, a device or somebody's fs.FS can look like. Stat is no help with it; only a limit on the
// read is.
type endless struct{}

func (endless) Open(name string) (fs.File, error) { return endlessFile{}, nil }

type endlessFile struct{}

func (endlessFile) Stat() (fs.FileInfo, error) { return endlessInfo{}, nil }
func (endlessFile) Close() error               { return nil }
func (endlessFile) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

type endlessInfo struct{}

func (endlessInfo) Name() string       { return "endless.house" }
func (endlessInfo) Size() int64        { return 0 }
func (endlessInfo) Mode() fs.FileMode  { return 0o444 }
func (endlessInfo) ModTime() time.Time { return time.Time{} }
func (endlessInfo) IsDir() bool        { return false }
func (endlessInfo) Sys() any           { return nil }

func TestAFileThatNeverEndsIsReadOnlyAsFarAsAHouseCouldGo(t *testing.T) {
	if _, err := LoadFS(endless{}, "endless.house"); err == nil || !strings.Contains(err.Error(),
		"larger than a house can be") {
		t.Errorf("LoadFS on a file that never ends: %v", err)
	}
	// And the same through a path, where /dev/zero is one.
	if _, err := os.Stat("/dev/zero"); err != nil {
		t.Skip("no /dev/zero here")
	}
	if _, err := LoadFile("/dev/zero"); err == nil || !strings.Contains(err.Error(),
		"larger than a house can be") {
		t.Errorf("LoadFile(/dev/zero): %v", err)
	}
}

// rooms is a house text that goes on numbering rooms for ever.
type rooms struct {
	buf bytes.Buffer
	n   int
}

func (r *rooms) Read(p []byte) (int, error) {
	for r.buf.Len() < len(p) {
		if r.n == 0 {
			fmt.Fprintf(&r.buf, "format %d\n", TextFormatVersion)
		}
		fmt.Fprintf(&r.buf, "room %d\n", r.n)
		r.n++
	}
	return r.buf.Read(p)
}

// The parser stops at the first room a house cannot have, rather than at the end of the text. An
// endless text is the proof: it can only stop there.
func TestATextStopsAtTheFirstRoomAHouseCannotHave(t *testing.T) {
	_, err := ParseText(&rooms{})
	if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("room %d is past the last",
		MaxRooms)) {
		t.Fatalf("ParseText on rooms without end: %v", err)
	}
	// The format directive is line 1, and room 0 is line 2.
	if want := fmt.Sprintf("line %d:", MaxRooms+2); !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %q, want it to start %q", err, want)
	}
}
