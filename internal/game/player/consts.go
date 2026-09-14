package player

// The constants that decide how the glider moves. Values and citations were
// checked against the original headers and sources rather than copied from the
// analysis: every #define below was read out of the file named beside it.
//
// Names drop the original's leading `k` because Go's exported identifiers are
// already qualified by the package, so `player.Gravity` reads better than
// `player.KGravity`. Nothing else is renamed -- where the original has two
// constants with the same value and different names, both are kept, because they
// document which code path uses which.

// Core integrator (GliderPRO/Sources/Player.c:13-17).
const (
	// Gravity is the value VDesiredVel is reset to every frame, which makes it
	// the free-fall terminal speed rather than an acceleration. Reached from
	// rest in two frames: 0, +2, +3.
	Gravity int16 = 3

	// HImpulse and VImpulse are the maximum change in HVel and VVel per frame:
	// the whole of the game's damping is this ramp pulling toward the desired
	// velocity. There is no friction term and no drag coefficient.
	HImpulse int16 = 2
	VImpulse int16 = 2

	// MaxHVel clamps horizontal speed to +/-16. There is deliberately no
	// vertical equivalent; see MoveGlider.
	MaxHVel int16 = 16

	// ShredderCountdown is stored in Frame as a negative countdown, and is the
	// delay between the confetti spawning and the life being lost.
	ShredderCountdown int16 = -68
)

// Input (GliderPRO/Sources/Input.c:14-16).
const (
	// NormalThrust is what a held direction key contributes to HDesiredVel, so
	// cruise speed is 5 px/frame: the ramp reaches it in three frames and the
	// overshoot clamp stops it at exactly 5 rather than 6.
	NormalThrust int16 = 5

	// HyperThrust is the battery, and it is added straight to HVel, bypassing
	// the ramp entirely. With a direction key also held the clamp pins the
	// result at MaxHVel, so steady-state battery flight is exactly 16 px/frame.
	HyperThrust int16 = 8

	// HeliumLift is assigned to VDesiredVel while helium is engaged, so rising
	// is slightly faster than falling (4 against 3).
	HeliumLift int16 = 4
)

// Blowers, fans and shoves (Interactions.c:13-15, Dynamics.c:17).
const (
	// FloorVentLift is assigned to VDesiredVel, so a floor vent's lift does not
	// stack with anything: the last blower to write it wins.
	FloorVentLift int16 = -6

	// CeilingVentDrop is likewise assigned to VDesiredVel, so it is a ramp target
	// and never a one-frame kick: VVel climbs 3, 5, 7, 8 over three frames under
	// the vent, is re-asserted every frame the hot spot overlaps, and decays
	// 8, 6, 4, 3 only from the first frame the glider is clear. There is no
	// vertical clamp, so the 8 is reached in full.
	CeilingVentDrop int16 = 8

	// FanStrength is added to HDesiredVel rather than assigned, which is the one
	// asymmetry in the blower set: fans stack with the direction keys and with
	// each other.
	FanStrength int16 = 12

	// ShoveVelocity is assigned to HDesiredVel when a foil-clad player collides
	// with a dynamic object.
	ShoveVelocity int16 = 8
)

// Transit speeds (all in GliderPRO/Sources/Player.c). These move Dest directly;
// the modes that use them never call MoveGlider. Several share a value and are
// still kept apart, because the original defines them separately per handler and
// a future correction to one should not silently move the others.
const (
	ClimbStairsSpeed  int16 = -4 // MoveGliderUpStairs, vertical (Player.c:317)
	VClimbStairsSpeed int16 = -4 // FinishGliderUpStairs, vertical (:385)
	HClimbStairsSpeed int16 = -4 // FinishGliderUpStairs, horizontal (:386)
	VDropStairsSpeed  int16 = 4  // down-stairs vertical (:433, :514)
	HDropStairsSpeed  int16 = 4  // down-stairs horizontal (:434, :515)
	VDropDuctSpeed    int16 = 4  // MoveGliderDownDuct (:659), FinishGliderDuctingIn (:936)
	VRiseDuctSpeed    int16 = -4 // MoveGliderUpDuct (:756)
	HPushMailSpeed    int16 = -4 // FinishGliderMailingLeft (:853)
	HPushMailRtSpeed  int16 = 4  // FinishGliderMailingRight (:891)
	HMailPullSpeed    int16 = 4  // MoveGliderInMailLeft (:962)
	HMailPullRtSpeed  int16 = -4 // MoveGliderInMailRight (:1053)
	VMailDropSpeed    int16 = 2  // settling rate inside a mail slot (:963, :1054)
	DropShredSlow     int16 = 1  // shredding descent once the shredder has hold (:1262)
	DropShredFast     int16 = 4  // shredding descent before reaching it (:1263)
)

// Mode timing and sprite sequencing (GliderPRO/Headers/GliderDefines.h:560-564,
// Modes.c:408).
const (
	LastFadeSequence    int16 = 16 // length of fadeInSequence; fade-in ends at Frame >= 16
	LeftFadeOffset      int16 = 7  // added to a fade sprite index for the left-facing variant
	FirstAboutFaceFrame int16 = 18 // first about-face sprite, and the Frame seed facing right
	LastAboutFaceFrame  int16 = 20 // last about-face sprite, and the Frame seed facing left
	WasBurning          int16 = 2  // Frame sentinel meant to carry fire through the stairs; vestigial, never written -- see StartGliderGoingUpStairs
	FramesToBurn        int16 = 60 // frames a burning glider survives: two seconds
	TicksPerFrame             = 2  // Mac ticks per frame, i.e. 30.07 fps
)

// Room geometry (GliderPRO/Headers/GliderDefines.h:496-511). Room-local
// coordinates: x runs 0..511 and y runs 0..321, and values outside that range
// are legal and meaningful -- they are how the player leaves a room.
const (
	NumTiles        int16 = 8
	TileWide        int16 = 64
	TileHigh        int16 = 322
	RoomWide        int16 = 512
	VertLocalOffset int16 = 322

	CeilingLimit int16 = 8   // Dest.Top below this triggers the ceiling test
	FloorLimit   int16 = 312 // Dest.Bottom past this triggers the floor test
	RoofLimit    int16 = 122 // on a roof background, Dest.Bottom past this hits the roof

	LeftWallLimit    int16 = 12  // the left wall plane in a walled room
	RightWallLimit   int16 = 500 // the right wall plane in a walled room
	NoLeftWallLimit  int16 = -24 // 0 - GliderWide/2: fully out through the left
	NoRightWallLimit int16 = 536 // RoomWide + GliderWide/2: fully out through the right
	NoCeilingLimit   int16 = -10 // fully out through the top
	NoFloorLimit     int16 = 332 // fully out through the bottom
)

// Glider geometry (GliderPRO/Headers/GliderDefines.h:548-553, :569).
const (
	// GliderWide and GliderHigh are the *nominal* dimensions, not invariants of
	// Dest. They are what the mode-entry functions install (FlagGliderNormal,
	// Modes.c:332-334) and what the reveal/hide tests compare against. Dest is
	// progressively narrowed or shortened by the clip helpers in the stairs, duct,
	// mail and shredder handlers (handle.go clipRight/clipLeft/clipTop/clipBottom;
	// Player.c:341, :374, :410, :468, :502, :539, :745, :842, :872, :910, :943,
	// :1041, :1132, :1307), and it stays clipped across frames because the next
	// frame's move is applied to the already-clipped edge and then re-clipped.
	GliderWide        int16 = 48 // nominal Dest width, installed at mode entry
	GliderHigh        int16 = 20 // nominal Dest height; 26 while burning
	HalfGliderWide    int16 = 24 // the roof tile test and the band spawn point
	GliderBurningHigh int16 = 26 // Dest height while burning: a 6px flame plume above
	ShadowHigh        int16 = 9
	ShadowTop         int16 = 306 // DestShadow.Top, fixed
	GliderStartsDown  int16 = 32  // base y for a side-entry respawn rect
)

// Facing and player identity (GliderDefines.h:554-557). These are Booleans in
// the original, so they are bools here and not a named type: the C compares them
// directly against TRUE and FALSE in places, and a distinct type would invite a
// port to make them an enum with three states.
const (
	FaceRight = true
	FaceLeft  = false
	Player1   = true
	Player2   = false
)

// Sprite indices into the 31-rect glider atlas (kNumGliderSrcRects, :558). The
// original indexes gliderSrc[] with bare numbers everywhere; these names are the
// one place the port adds vocabulary the C does not have, because a reader cannot
// otherwise tell gliderSrc[29] from gliderSrc[30].
//
// The layout was derived by collecting every gliderSrc[] subscript in Player.c and
// Modes.c and grouping them by the branch that produces them.
const (
	SpriteRight        = 0  // facing right, level
	SpriteRightTipped  = 1  // facing right, banked into a leftward key
	SpriteLeft         = 2  // facing left, level
	SpriteLeftTipped   = 3  // facing left, banked into a rightward key
	SpriteFirstFade    = 4  // 4..10: the seven right-facing dissolve stages
	SpriteLastFade     = 10 // + LeftFadeOffset gives 11..17 for facing left
	SpriteFirstBurning = 21 // 21..24 facing right, 25..28 facing left
	SpriteBurningLeft  = 25
	SpriteSlideRight   = 29 // standing on grease, facing right
	SpriteSlideLeft    = 30
)

// FadeInSequence is fadeInSequence[], filled in at startup by
// InterfaceInit.c:167-182. It maps a fade frame counter to a dissolve sprite, and
// it is not a straight ramp: it is four overlapping runs of four, each starting one
// stage further along, so the sixteen frames visit only stages 4..10 and linger.
//
// Read forwards it is a fade in; the same table read backwards, by decrementing
// Frame, is the fade out -- which is why there is no second table.
var FadeInSequence = [LastFadeSequence]int16{
	4, 5, 6, 7,
	5, 6, 7, 8,
	6, 7, 8, 9,
	7, 8, 9, 10,
}

// Sounds and their priorities, for the calls this package makes
// (GliderDefines.h:55-118, :120-180). Priority decides which sound wins a busy
// channel, and lower numbers win.
const (
	FadeInSound     int16 = 1
	FadeOutSound    int16 = 2
	FollowSound     int16 = 7
	CaughtFireSound int16 = 16
	ThrustSound     int16 = 18
	FizzleSound     int16 = 19
	ShredSound      int16 = 26
	TransOutSound   int16 = 59
	TransInSound    int16 = 60
	HissSound       int16 = 62

	ThrustPriority     int16 = 300
	HissPriority       int16 = 311
	FizzlePriority     int16 = 703
	FadeInPriority     int16 = 900
	FadeOutPriority    int16 = 901
	CaughtFirePriority int16 = 902
	ShredPriority      int16 = 903
	FollowPriority     int16 = 904
	TransInPriority    int16 = 905
	TransOutPriority   int16 = 906
)

// MaxSaidFollow caps the "follow me" prompt at three per game (Modes.c:462).
const MaxSaidFollow int16 = 3

// Where a room transition is headed (GliderDefines.h:210-213).
const (
	Above   int16 = 1
	ToRight int16 = 2
	Below   int16 = 3
	ToLeft  int16 = 4
)

// Scoreboard title modes (GliderDefines.h:619-621).
const (
	NormalTitleMode  int16 = 0
	EscapedTitleMode int16 = 1
)

// How the other player got out, stored in one signed slot
// (GliderDefines.h:596-608). Only the values the player code writes are here.
const (
	NoOneEscaped            int16 = -1
	PlayerEscapedUpStairs   int16 = -6
	PlayerEscapedDownStairs int16 = -7
	PlayerTransportedOut    int16 = -10
	PlayerDuckedOut         int16 = -11
	PlayerMailedOut         int16 = -12
)

// Clipping planes and offsets the transit handlers measure against. Each is a
// bare number in the C, at the line cited.
const (
	// CeilingTransTop is the ceiling transporter's top edge
	// (GliderDefines.h:472); the duct handlers clip against CeilingTransTop+1.
	CeilingTransTop int16 = 6

	// StairsClipOffset is subtracted from Dest.Bottom to find how much of the
	// glider is still below the top of the staircase (Player.c:336). The nearby
	// kStairsTop is 28, one less; the C uses the literal 29 here and does not
	// reference the constant, so the port keeps the literal too.
	StairsClipOffset int16 = 29

	// DuctFloorLimit is the y a glider descending a floor duct disappears at
	// (Player.c:704).
	DuctFloorLimit int16 = 315

	// ShredderInset places the glider horizontally in the shredder's mouth,
	// measured from the object's left edge (Modes.c:369).
	ShredderInset int16 = 36

	// ShredderMouthInset is subtracted from the shredder's bottom to get the y
	// stored in Frame as the grind target (Modes.c:400).
	ShredderMouthInset int16 = 3

	// GliderAppearsComing is the y a glider is placed at when it emerges from a
	// staircase in the next room (Modes.c:521, :552 -- separate #defines with the
	// same value, one per direction).
	GliderAppearsComingUp   int16 = 100
	GliderAppearsComingDown int16 = 100

	// BandSpawnH and BandSpawnV are added to Dest's top-left to get the muzzle
	// (Input.c:352-353). BandSpawnH is HalfGliderWide and BandSpawnV is half the
	// glider's height, but both are literals in the C.
	BandSpawnH int16 = 24
	BandSpawnV int16 = 10

	// MirrorOffsetH and MirrorOffsetV place the reflected dirty rect in a room
	// with a mirror (Modes.c:95).
	MirrorOffsetH int16 = -20
	MirrorOffsetV int16 = -16
)

// Animation bounds that only appear as literals in the C.
const (
	BurningFrames  int16 = 4  // MoveGliderBurning cycles Frame 0..3 (Player.c:206)
	FoilLastFrame  int16 = 8  // the foil dissolve ends after Frame 8 (Player.c:1173)
	FoilDeckFrame  int16 = 5  // from Frame 5 the foil sprite is shown (Player.c:1179)
	FoilFadeTop    int16 = 10 // sprite index is 10-Frame while dissolving (:1183)
	IdleFrames     int16 = 30 // TagGliderIdle's countdown, stored in HVel (Modes.c:638)
	BatteryFrames  int16 = 4  // battery/helium sound re-triggers every 4th frame (Input.c:152)
	AboutFaceStart int16 = 18 // == FirstAboutFaceFrame, for readability at use sites
)

// Mode is the glider's state-machine state (GliderDefines.h:571-594).
//
// The values are consecutive and explicit in the original and are preserved
// exactly, because they are written into a saved game.
type Mode = int16

const (
	GliderNormal         Mode = 0  // the only fully interactive mode
	GliderFadingIn       Mode = 1  // materialising after a respawn
	GliderFadingOut      Mode = 2  // dying
	GliderGoingUp        Mode = 3  // walking up out of the top of the room
	GliderComingUp       Mode = 4  // emerging at the bottom right of the room above
	GliderGoingDown      Mode = 5  // walking down out of the room
	GliderComingDown     Mode = 6  // emerging at the left of the room below
	GliderFaceLeft       Mode = 7  // about-face tumble, three frames
	GliderFaceRight      Mode = 8  // about-face tumble, three frames
	GliderBurning        Mode = 9  // on fire, with a 60-frame fuse
	GliderTransporting   Mode = 10 // dissolving into a transporter
	GliderDuctingDown    Mode = 11 // sliding into a floor duct
	GliderDuctingUp      Mode = 12 // rising into a ceiling duct
	GliderDuctingIn      Mode = 13 // dropping out of a ceiling duct
	GliderMailInLeft     Mode = 14 // being sucked into a left-facing mail slot
	GliderMailOutLeft    Mode = 15 // being pushed out of a left-facing mail slot
	GliderMailInRight    Mode = 16
	GliderMailOutRight   Mode = 17
	GliderGoingFoil      Mode = 18 // dissolve out and in, gaining foil
	GliderLosingFoil     Mode = 19 // dissolve out and in, losing foil
	GliderShredding      Mode = 20 // being eaten by a paper shredder
	GliderInLimbo        Mode = 21 // two-player only: waiting for the other player
	GliderIdle           Mode = 22 // two-player only: a 30-frame freeze
	GliderTransportingIn Mode = 23 // materialising out of a transporter
)

// InRoom reports whether a mode counts as "in the room" for the wall, floor and
// ceiling logic (Interactions.c:691-694). Only four of the twenty-four do, which
// is why a glider on the stairs or in a duct cannot also fall out of the world.
func InRoom(m Mode) bool {
	switch m {
	case GliderNormal, GliderFaceLeft, GliderFaceRight, GliderBurning:
		return true
	}
	return false
}

// HasMomentum reports whether a mode runs the integrator. These are exactly the
// six modes whose handlers end in a MoveGlider call; every other mode moves Dest
// by one of the hard-coded transit speeds instead.
func HasMomentum(m Mode) bool {
	switch m {
	case GliderNormal, GliderBurning, GliderFaceLeft, GliderFaceRight,
		GliderGoingFoil, GliderLosingFoil:
		return true
	}
	return false
}

// CollidesWithDynamics reports whether a mode is tested against dynamic objects
// (Dynamics.c:45-50). It is InRoom's four plus the two foil dissolves, because a
// player mid-dissolve can still be shoved.
func CollidesWithDynamics(m Mode) bool {
	switch m {
	case GliderNormal, GliderFaceLeft, GliderFaceRight, GliderBurning,
		GliderGoingFoil, GliderLosingFoil:
		return true
	}
	return false
}
