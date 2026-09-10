// Package house is gliderGo's model of a Glider PRO house -- the file format the
// 1994 game called a "house" and everyone else calls a level.
//
// The governing design rule is that this package is byte-exact. Load(Save(h))
// and Save(Load(b)) are both identities, for all 22 shipped houses, including
// the parts of the file the original never meant anyone to see. That is not
// fastidiousness for its own sake: the shipped houses are the only authority on
// their own contents, they are vendored read-only, and a loader that silently
// normalises them destroys the evidence a fidelity bug would be diagnosed from.
//
// Three consequences shape the types below, each learned from the corpus rather
// than assumed (docs/analysis/house-format.md, and the audit in TestCorpus*):
//
//   - Fixed-size byte arrays, not Go strings. A Str27 is 28 bytes of which
//     byte 0 is the length; the bytes past the length are uninitialised residue
//     and they differ between houses. PStr28 keeps all 28 and decodes on demand.
//
//   - Booleans are bytes. Mac `Boolean` is one byte, conventionally 1, but
//     three blowers in the corpus store 23 and houseType.unusedBoolean holds
//     seven distinct values including 255. Truthiness is "non-zero"; the byte
//     is preserved.
//
//   - The objectType union is 10 raw bytes with typed views over it, not nine
//     Go structs. Empty slots (what == ObjectIsEmpty) carry live residue in
//     those 10 bytes because CreateNewRoom only ever set `what`
//     (GliderPRO/Sources/Room.c:181-182), and object slot indices are
//     load-bearing -- links address objects by slot, so nothing may be
//     renumbered or compacted. Six shipped rooms genuinely have holes.
//
// All multi-byte fields are big-endian, and classic Mac field order is kept as
// the original declared it: Point is {V, H} -- vertical first -- and Rect is
// {Top, Left, Bottom, Right}. Getting either backwards produces a house that
// loads without complaint and plays wrong, which is exactly the class of bug
// this package is written to make impossible.
package house

// Sizes of the on-disk structures, from GliderPRO/Headers/GliderStructs.h.
// SizeofHouseHeader is offsetof(houseType, rooms), not sizeof(houseType): a
// PowerPC build padded the struct to 868 and left two slack bytes at the end of
// the file it saved, which is why Sampler is 1564 bytes rather than 1562
// (docs/analysis/house-format.md 12.4).
const (
	SizeofHouseHeader = 866 // houseType up to rooms[]
	SizeofRoom        = 348 // roomType
	SizeofObject      = 12  // objectType: short what + 10-byte union
	SizeofScores      = 292 // scoresType
	SizeofGame        = 40  // gameType

	MaxRoomObs = 24 // kMaxRoomObs: objects[] slots per room
	MaxScores  = 10 // kMaxScores: high-score rows per house
	NumTiles   = 8  // kNumTiles: background tiles across a room

	// PowerPCSlack is the two bytes a PowerPC build could append past the last
	// room. Tolerated on load, reproduced on save, never interpreted.
	PowerPCSlack = 2
)

// House versions (GliderPRO/Headers/GliderDefines.h).
const (
	HouseVersion    = 0x0200 // kHouseVersion: every shipped house
	NewHouseVersion = 0x0300 // kNewHouseVersion: never written by 1.1.2
)

// ObjectIsEmpty marks an unused objects[] slot. It is -1, not 0: a slot whose
// `what` is 0 would be a legal-looking object of undefined type.
const ObjectIsEmpty int16 = -1

// Point is a classic QuickDraw Point: **vertical first**. Keeping the on-disk
// order in the Go type means a mis-ordered read is a compile-time field-name
// error rather than a silent transposition.
type Point struct {
	V, H int16
}

// Rect is a classic QuickDraw Rect, in the order the Toolbox stored it.
type Rect struct {
	Top, Left, Bottom, Right int16
}

// Wide and Tall match RectWide/RectTall in GliderPRO/Sources/RectUtils.c.
func (r Rect) Wide() int16 { return r.Right - r.Left }
func (r Rect) Tall() int16 { return r.Bottom - r.Top }

// House is houseType: the 866-byte header plus the rooms.
//
// Field names, order and spelling are the original's, including the unused ones.
// `UnusedShort`, `UnusedBoolean` and the two unused longs inside SavedGame are
// residue in the shipped files, and they are carried so that a round-trip is a
// round-trip.
type House struct {
	Version       int16
	UnusedShort   int16
	TimeStamp     int32 // Mac epoch: seconds since 1904-01-01 local
	Flags         int32 // bit 0 = wardBit, bit 1 = phoneBit, bit 2 = suppress star count
	Initial       Point // where the glider starts, in room-local coordinates
	Banner        PStr256
	Trailer       PStr256
	HighScores    Scores
	SavedGame     Game
	HasGame       byte
	UnusedBoolean byte
	FirstRoom     int16
	NRooms        int16 // authoritative on load; recomputed on save
	Rooms         []Room

	// Slack is the trailing bytes past the last room: empty for 21 of the 22
	// shipped houses, two bytes for Sampler. Preserved verbatim.
	Slack []byte
}

// Flag bits of House.Flags (docs/analysis/house-format.md 4.4). The star-count
// bit is inverted: the banner shows the star count when bit 2 is *clear*.
const (
	FlagWard        int32 = 1 << 0
	FlagPhone       int32 = 1 << 1
	FlagNoStarCount int32 = 1 << 2
)

func (h *House) Ward() bool  { return h.Flags&FlagWard != 0 }
func (h *House) Phone() bool { return h.Flags&FlagPhone != 0 }

// BannerStarCount reports whether the banner displays the star count, which is
// true when FlagNoStarCount is *clear* -- the sense the original tests, not the
// one the flag's position suggests.
func (h *House) BannerStarCount() bool { return h.Flags&FlagNoStarCount == 0 }

// Unlocked mirrors the original's odd overload of TimeStamp's low bit as a
// read-only marker (docs/analysis/house-format.md 4.3).
func (h *House) Unlocked() bool { return h.TimeStamp&1 == 0 }

// Scores is scoresType: the ten-row high-score board stored inside the house
// file at offset 528. See docs/analysis/scoring.md 7.2. Note that Levels is
// *rooms visited*, despite its name -- the port relies on that reading for both
// its own score board and the Stage 3 race metric.
type Scores struct {
	Banner     PStr32
	Names      [MaxScores]PStr16
	Scores     [MaxScores]int32
	TimeStamps [MaxScores]uint32 // Mac epoch
	Levels     [MaxScores]int16  // rooms visited, not a level number
}

// Game is gameType: a saved game embedded in the house file. Only two shipped
// houses set HasGame; the other twenty carry a stale 40-byte block at +820,
// which is why two pairs of houses share identical bytes there.
type Game struct {
	Version      int16
	WasStarsLeft int16
	TimeStamp    int32
	Where        Point
	Score        int32
	UnusedLong   int32
	UnusedLong2  int32
	Energy       int16
	Bands        int16
	RoomNumber   int16
	GliderState  int16
	NumGliders   int16
	Foil         int16
	UnusedShort  int16
	Facing       byte
	ShowFoil     byte
}

// Room is roomType, 348 bytes. Objects is always MaxRoomObs long, holes and all.
type Room struct {
	Name       PStr28
	Bounds     int16 // bit flags: which sides are room boundaries
	LeftStart  byte
	RightStart byte
	UnusedByte byte
	Visited    byte
	Background int16 // PICT id, or 0 for "no background"
	Tiles      [NumTiles]int16
	Floor      int16
	Suite      int16
	Openings   int16
	NumObjects int16 // advisory: the original recomputes it from the slots
	Objects    [MaxRoomObs]Object
}

// LiveObjects counts slots whose what is not ObjectIsEmpty. This, not
// NumObjects, is the authority -- MakeSureNumObjectsJives overwrites the stored
// field from exactly this count (GliderPRO/Sources/HouseLegal.c:899-907).
// It happens to agree in all 4,070 shipped rooms, which the corpus test pins.
func (r *Room) LiveObjects() int {
	n := 0
	for i := range r.Objects {
		if r.Objects[i].What != ObjectIsEmpty {
			n++
		}
	}
	return n
}

// Compacted reports whether the live slots run 0..LiveObjects-1 with no holes.
// Six shipped rooms return false. A port must not "fix" them: links address
// objects by slot index.
func (r *Room) Compacted() bool {
	seenEmpty := false
	for i := range r.Objects {
		if r.Objects[i].What == ObjectIsEmpty {
			seenEmpty = true
		} else if seenEmpty {
			return false
		}
	}
	return true
}
