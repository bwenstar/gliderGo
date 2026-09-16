package house

// Recognising a house file without reading it.
//
// The original does not have this problem. It finds houses by asking the File
// Manager for every file whose Finder type is 'gliH' and whose creator is 'ozm5'
// (SelectHouse.c:557-648, docs/analysis/ui-dialogs.md 7.2), which is metadata the
// file system kept beside the fork and which no other operating system has. A
// port therefore has to decide from the bytes, and docs/analysis/ui-dialogs.md P5
// works the options through: an extension convention plus a magic number, or a
// pure sniff of the header.
//
// This is the sniff, and it is the whole of the test:
//
//	version == kHouseVersion (0x0200)   and   866 + 348*nRooms <= file length
//
// Both halves matter. All 22 shipped houses carry exactly 0x0200 and 1.1.2 never
// wrote the 0x0300 that GliderDefines.h reserves, so the version is a two-byte
// magic number in practice. The arithmetic is what makes it a *house* rather than
// any file that happens to start 0x02 0x00: nRooms sits at offset 864 and the
// rooms are fixed-size records, so the length is predicted from two bytes of the
// header, and a random file passing both tests by coincidence is a remote
// prospect.
//
// The comparison is `<=` and not `==` deliberately. Twenty-one of the shipped
// houses satisfy the equality; Sampler is 1564 bytes where 866 + 348*2 predicts
// 1562, because a PowerPC build padded houseType to 868 and wrote two slack bytes
// past the last room (house.go's PowerPCSlack). A port that insisted on equality
// would refuse to list a house the original opens without complaint.
//
// What this deliberately does *not* do is validate. Load is strict about
// structure and permissive about content and PeekFile is looser still: it says
// "this is a house file and here is its header", and whether the house is
// playable is Load's answer and then the game's. The picker in internal/shell
// relies on that split -- it lists what sniffs as a house and reports the failure
// when a listed house will not load, which is a message a player can act on
// rather than a file that silently vanished from the list.

import (
	"fmt"
	"io"
	"os"
)

// Summary is what a house picker can learn from a house's header alone: enough to
// list it, sort it, show its size and read its high-score board, and not one room.
//
// Every field is the header's, except Path and Size which are the file's. Rooms is
// nRooms as stored, which Load treats as authoritative -- so it is the number the
// list should show even though nothing has counted the rooms.
type Summary struct {
	Path      string
	Size      int64
	Version   int16
	NRooms    int16
	FirstRoom int16
	TimeStamp int32
	Flags     int32
	Unlocked  bool   // TimeStamp's low bit, clear: the editor may change it
	Banner    string // the author's opening message, as BringUpBanner shows it
	Scores    Scores // the ten-row board stored at offset 528

	// The saved game the house carries: `hasGame` at offset 860 and the 40-byte
	// `gameType` at 820. Both are in the header this already reads, and they are here so
	// that a menu can offer "resume the game this house shipped with" without opening
	// every room -- see EmbeddedGame, and internal/shell's saved-game row. Twenty of the
	// twenty-two shipped houses carry a stale block with HasGame clear, so the flag is
	// the only thing that says whether the block means anything.
	HasGame bool
	Game    Game
}

// PeekFile reads a house's header and reports whether the file is a house at all.
//
// The error says which of the two tests failed and with what numbers, because
// this is the function a picker uses to explain why a file it can see is not in
// the list -- "version 0x4D5A" for a Windows executable someone dropped in the
// houses directory, "866 + 348*40 = 14786 bytes of rooms in a 500-byte file" for
// a truncated download.
func PeekFile(path string) (*Summary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("%s: is a directory", path)
	}

	buf := make([]byte, SizeofHouseHeader)
	if _, err := io.ReadFull(f, buf); err != nil {
		// ErrUnexpectedEOF for a short file, which is the common case and is not
		// worth a different message: too small to hold a house header is the
		// reason either way.
		return nil, fmt.Errorf("%s: %d bytes, too small for a %d-byte house header",
			path, st.Size(), SizeofHouseHeader)
	}

	d := &decoder{b: buf}
	var h House
	if err := d.header(&h); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if h.Version != HouseVersion {
		return nil, fmt.Errorf("%s: version is 0x%04X, not a house (kHouseVersion is 0x%04X)",
			path, uint16(h.Version), HouseVersion)
	}
	if need := int64(SizeofHouseHeader) + int64(SizeofRoom)*int64(h.NRooms); need > st.Size() {
		return nil, fmt.Errorf("%s: header claims %d rooms, which needs %d bytes; the file is %d",
			path, h.NRooms, need, st.Size())
	}

	return &Summary{
		Path:      path,
		Size:      st.Size(),
		Version:   h.Version,
		NRooms:    h.NRooms,
		FirstRoom: h.FirstRoom,
		TimeStamp: h.TimeStamp,
		Flags:     h.Flags,
		Unlocked:  h.Unlocked(),
		Banner:    h.Banner.Text(),
		Scores:    h.HighScores,
		HasGame:   h.HasGame != 0,
		Game:      h.SavedGame,
	}, nil
}
