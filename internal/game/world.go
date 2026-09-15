// Package game is the running Glider PRO game: the live state the original kept
// in file-scope globals across a dozen translation units, and the code that reads
// and writes it.
//
// It exists as one package for the reason the original's globals were one segment.
// Play.c's frame loop, Interactions.c's hot-spot dispatcher, Dynamics*.c's movers,
// Objects.c's state engine and Transit.c's room changes are mutually recursive:
// MoveGliderUpStairs calls MoveRoomToRoom, which rebuilds the room, which calls
// CreateActiveRects, which reads the same masterObjects table HandleSwitches
// writes. Splitting that into packages would need an interface at every cut, and
// the cuts are not where the seams are. So it is one package, and the seams that
// *are* real -- the renderer, the house codec, the player's own state machine --
// stay in their own packages behind narrow surfaces.
//
// Two types carry everything. World is the game; Room is the composed locale the
// glider is standing in. The split is not tidiness: Room is everything DrawLocale
// rebuilds from scratch on a room change, and World is everything that survives
// one. Putting a field on the wrong side of that line is the bug class this
// package is arranged to make visible, and there is one field in the original that
// is on the wrong side -- tvOn, see Room.TVInRoom.
package game

import (
	"math/rand"

	"glidergo/internal/game/player"
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// Rect is the QuickDraw rect, shared with internal/house and internal/render so no
// conversion is needed across those two boundaries. internal/game/player declares
// its own identical struct; convert with player.Rect(r) at that boundary, which is
// a no-op the compiler checks.
type Rect = house.Rect

// Point is the QuickDraw point: vertical first.
type Point = house.Point

// World is the game. Every field names a file-scope global of the original, and
// the comment gives the C identifier where the Go name differs.
//
// It is deliberately one flat struct rather than a nest of subsystems. In the C
// these are 40-odd globals that any function may touch, and a nested arrangement
// would suggest an encapsulation that does not exist -- HandleInteraction writes
// theScore, batteryTotal, bandsTotal, foilTotal, starsLeft, mortals and the room
// itself, all in one dispatcher. Flat and honest beats nested and wrong.
type World struct {
	// H is thisHouse: the house handle. Object state lives here and nowhere else,
	// which is why SetObjectState writes through it and every other copy is a
	// cache that has to be invalidated by hand.
	H *house.House

	// R is the current locale. It is a field, not a pointer, and it is *rebuilt in
	// place* by Rebuild rather than replaced -- see the note on Rebuild for why
	// that has to be a method.
	R Room

	// Rand is the RandomInt stream. It is a field rather than the global source
	// because the order of draws is observable: the flame phases, the pendulum
	// starts and the balloon jitter all pull from it in composition order, so two
	// runs that consume the same sequence look identical and a replay can be
	// pinned. Seeded once per game, never re-seeded on a room change.
	Rand *rand.Rand

	// P1 and P2 are theGlider and theGlider2. Values, not pointers, so that
	// &w.P1 is a stable address for the whole game -- the C passes &theGlider
	// into every handler and stores no gliders anywhere else, and a slice would
	// invalidate those addresses on growth.
	P1, P2 player.Glider

	// TwoPlayer is twoPlayerGame. OneLeft is onePlayerLeft: one player has used
	// their last mortal and the survivor is finishing alone. DeadWhich is
	// playerDead, whose true value is player.Player1 -- and which names the player
	// who *is* dead, so every read of it selects the *other* glider.
	TwoPlayer bool
	OneLeft   bool
	DeadWhich bool

	// FirstPlayer (Transit.c:18) is which of the two triggered the pending room
	// change, and is what decides who is placed first on arrival.
	FirstPlayer bool

	// Escaped is otherPlayerEscaped: the boundary the other player left through
	// while this one was still in the room, or NoOneEscaped.
	Escaped int16

	// Suicide is playerSuicide: the player pressed the give-up key, which skips
	// the death animation but still spends a mortal.
	Suicide bool

	// Score is theScore, a long in the C and therefore int32 here. Rooms visited,
	// not points, is what the high-score board sorts on; see internal/house.Scores.
	Score int32

	// The inventory. Battery is signed and is one counter for two power-ups:
	// positive is battery charges, negative is helium (player.Env.BatteryTotal has
	// the full note). Foil is sheets remaining and ShowFoil is whether the foil
	// sprite sheet is loaded, which are separate because the sheet swap is a
	// graphics change that lags the count by a frame.
	Mortals   int16
	Battery   int16
	Bands     int16
	Foil      int16
	ShowFoil  bool
	StarsLeft int16

	// SaidFollowCount is saidFollow (Modes.c:13): how many times the two-player
	// "follow me" prompt has played, capped at three for the whole game rather than
	// per room (Modes.c:462-466).
	//
	// The C's name is `saidFollow` and the Env accessor keeps it; the field carries
	// the -Count suffix because Go will not let a struct have a field and a method of
	// the same name, and the method is the one whose name is fixed by the interface.
	// The same applies to SoundPlayer below.
	SaidFollowCount int16

	// Frame is gameFrame: frames since the game began, the clock every timer and
	// animation phase is measured against. EvenFrame halves it for the animations
	// that run at 15 fps. NextFrame is the TickCount the next frame is due at,
	// which InitGarbageRects seeds (Render.c:690).
	Frame     int64
	EvenFrame bool
	NextFrame int64

	// TickCount is the 60.15 Hz tick source. nil derives it from Frame; see Ticks.
	TickCount func() int64

	// Work2Main and Back2Work are the two dirty-rect lists (Render.c): what has to
	// be copied from the work map to the screen this frame, and what has to be
	// restored from the clean background to the work map first. Both are cleared by
	// InitGarbageRects and consumed by RenderFrame.
	//
	// Overflow is observable and 1.5d has to reproduce it: AddRectToWorkRects
	// (Render.c) guards on `numWork2Main < kMaxGarbageRects - 1` and, when that
	// fails, **silently drops the rect**. Nothing merges it into a neighbour. A
	// dropped rect is a patch of screen that is never copied forward, so a very busy
	// frame leaves visible litter until something else dirties the same pixels.
	//
	// Note the `- 1`: the guard stops at 47 of the 48 slots, so the last one is
	// unreachable. That is the original's off-by-one and the effective cap is 47.
	Work2Main []Rect
	Back2Work []Rect

	// Pending is the resolved destination of the transit the glider is currently
	// inside: transRect, transRoom and linkedToWhat as one value. The player code
	// takes it as an argument rather than reaching for the object graph, which is
	// why player.Link exists at all.
	Pending player.Link

	// PrevRoom is previousRoom, which ForceThisRoom sets and the map window reads.
	PrevRoom int16

	// Ward and Phone are the house's two flag bits, cached because they are read
	// per frame. HasMovie is whether this build found a QuickTime movie for the
	// house at all.
	Ward     bool
	Phone    bool
	HasMovie bool

	// TVOn is tvOn, and it is the one global in the original that is on the wrong
	// side of the World/Room line: nothing ever resets it on a room change, so it
	// stays true after the player leaves the room whose TV they switched on.
	//
	// That is not a bug in the C, because every one of the seven sites that reads
	// it also tests tvInRoom, which *is* room-scoped and *is* reset. The port
	// keeps the field here, keeps Room.TVInRoom there, and keeps both conjuncts at
	// all seven read sites. Dropping either one -- "simplifying" TVOn into the
	// room, or dropping the TVInRoom test as redundant -- changes behaviour.
	TVOn bool

	// TriggerSoundExists stands in for the 'snd ' resource lookup inside
	// LoadTriggerSound (Sound.c:265-303). It exists as a hook rather than a call
	// because whether a sound trigger gets a hot spot at all depends on whether
	// its sound loads, so a silent build composes measurably different rooms from
	// a build with audio -- and the fidelity corpus has to be able to choose.
	//
	// nil means "no sound system", which is exactly the original's dontLoadSounds
	// short circuit: LoadTriggerSound returns -1 and no kSoundIt rect is made.
	// That is the state of this stage; 1.6 supplies a real one.
	TriggerSoundExists func(soundID int16) bool

	// SoundPlayer is the injected implementation of PlayPrioritySound (Sound.c:40-85):
	// request a sound, which is granted only if nothing of higher priority is already
	// playing. A hook for the same reason as TriggerSoundExists -- 1.6 fills it in --
	// and nil means silence.
	//
	// It is not called `PlayPrioritySound`, even though that is the C's name, because
	// the method that satisfies player.Env owns that name (see env.go). The field is
	// the backend; the method is the call site's view of it.
	//
	// Larger is higher: PlayPrioritySound finds the quietest of the three channels
	// and plays only if `priority >= lowestPriority` (Sound.c:53-67). The scale runs
	// from 100 for a wall bump to the 800s for the noisy appliances, so a blower's
	// 701 displaces most things and is displaced by few.
	SoundPlayer func(sound, priority int16)
}

// Room is the composed locale: the central room the glider is in, the eight
// neighbours drawn around it, and every table DrawLocale rebuilds from scratch.
//
// Everything here is discarded and recomputed on a room change. That is the
// definition of the type: if a field survives a room change it belongs on World.
type Room struct {
	// Scene is the renderer's half of the same locale -- the surfaces, the nine
	// room numbers, the animation tables and the draw order. It is embedded rather
	// than referenced because in the original there is no distinction: numLights
	// and thisTiles are globals that both the drawing code and the collision code
	// read, and the embedding reproduces that shared reach without duplicating the
	// fields into two places that can disagree.
	*render.Scene

	// Hot is hotSpots[], the collision table. Cap MaxHotSpots; AddActiveRect
	// returns -1 rather than growing it.
	Hot []HotObject

	// Master is masterObjects[], the object graph: every object of all nine local
	// rooms, with its links resolved into indices into this same slice. Cap
	// MaxMasterObjects.
	Master []MasterObject

	// NumLocalMaster is numLocalMasterObjects: how many of Master belong to the
	// central room. The central room is listed first, so Master[:NumLocalMaster]
	// is exactly the central room's 24 slots -- which is what the dispatchers that
	// only care about the room the glider is in iterate over.
	NumLocalMaster int

	// LeftThresh and RightThresh are each both a coordinate and a flag: the escape
	// checks compare the glider against them, and compare them against
	// LeftWallLimit/RightWallLimit to decide whether there is a wall at all.
	LeftThresh, RightThresh int16

	// The four openings, from DetermineRoomOpenings. Note that these are not
	// symmetric in how they are derived: left and right come from the first and
	// last background tile, top and bottom from the background alone.
	TopOpen, BottomOpen, LeftOpen, RightOpen bool

	// ShadowVisible is shadowVisible: the *cached* flag, set once by ReadyLevel
	// and by RedrawRoomLighting, not the IsShadowVisible predicate. The two differ
	// in the original and the difference is load-bearing; see IsShadowVisible.
	ShadowVisible bool

	// TakingTheStairs suppresses the glider's own drawing for one frame while a
	// staircase transition completes.
	TakingTheStairs bool

	// HasMirror is true in a room holding a kMirror, which doubles every dirty
	// rect at a fixed offset (Modes.c:92-97).
	HasMirror bool

	// TVInRoom and TVMovieNumber are room-scoped and are reset here on every room
	// change, which is what makes World.TVOn safe to leave stale. See that field.
	TVInRoom      bool
	TVMovieNumber int16

	// NumChimes is numChimes: how many kChimes objects the *central* room holds.
	// It is incremented inside CreateActiveRects, which only runs for the central
	// room, so a chime in a neighbouring room does not shorten the interval --
	// a detail worth stating because the count reads like a locale-wide total.
	NumChimes int

	// TriggerSoundHeld is theSoundData[kMaxSounds-1] != nil, i.e. whether the one
	// reserved sound slot is occupied. It lives on Room and not on World because
	// DrawLocale calls DumpTriggerSound() in its reset head (RoomGraphics.c:58),
	// which frees the slot on every room change -- so its lifetime is exactly a
	// locale's even though the C's array is a file-scope global.
	//
	// It is what makes a room's *second* kSoundTrigger inert: the first one takes
	// the slot and every later one fails LoadTriggerSound's `!= nil` test and gets
	// no hot spot. That is a real one-per-room limit and not an artefact of the
	// port; see loadTriggerSound.
	TriggerSoundHeld bool
}

// NoOneEscaped is the sentinel for World.Escaped (GliderDefines.h).
const NoOneEscaped int16 = -1

// NewWorld starts a world on a house. Nothing is composed until Rebuild.
//
// The Scene is supplied rather than created here because a Scene needs a View and
// an Assets, and a headless test wants to choose both. rnd is the RandomInt stream;
// pass a fixed seed to make a run reproducible.
func NewWorld(h *house.House, sc *render.Scene, rnd *rand.Rand) *World {
	w := &World{
		H:        h,
		Rand:     rnd,
		Ward:     h.Ward(),
		Phone:    h.Phone(),
		Escaped:  NoOneEscaped,
		PrevRoom: -1,
	}
	w.R.Scene = sc
	w.R.Hot = make([]HotObject, 0, MaxHotSpots)
	w.R.Master = make([]MasterObject, 0, MaxMasterObjects)
	return w
}

// ThisRoom is thisRoom: the central room's record. It is read out of the house
// rather than kept as a separate copy, which is a deliberate divergence -- see
// SetObjectState for why the original's separate thisRoom copy exists and why not
// having it removes a whole class of cache-coherence bug rather than creating one.
func (w *World) ThisRoom() *house.Room {
	return w.Room(w.R.RoomNumber)
}

// Room returns a room by index, or nil if the index is out of range.
//
// The nil is not defensive padding. GetNeighborRoomNumber returns -1 for a room
// that does not exist, and the original then indexes rooms[-1] -- which on 68k
// read the tail of the preceding room's struct and, for the object slots, aliased
// tiles[6] and the low byte of openings (see the note in hotspots.go). Returning
// nil and having callers test it is the port's chosen divergence, and every caller
// that tests it says which of the C's out-of-range reads it is standing in for.
func (w *World) Room(n int16) *house.Room {
	if n < 0 || int(n) >= len(w.H.Rooms) {
		return nil
	}
	return &w.H.Rooms[n]
}

// NumberRooms is the numberRooms global: how many rooms the house really has.
// The stored NRooms field is advisory and RoomExists walks this count.
func (w *World) NumberRooms() int16 { return int16(len(w.H.Rooms)) }
