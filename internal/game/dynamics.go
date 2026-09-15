package game

// Dynamics: the dinahs table and its two per-frame loops. All of Dynamics3.c
// (555 lines), plus the dynaNum write-back at ObjectDrawAll.c:952-960.
//
// "dinahs" is the author's name for the table and it is kept. Seventeen object codes
// register into it, fifteen handlers advance them and seven renderers draw them; the
// difference between fifteen and seven is the subject of the comment on RenderDynamics.
//
// Two facts about this table shape everything else in 1.5c, and both are easier to see
// here than in any individual handler.
//
// **The cap is shared across the whole locale, not per room.** kMaxDynamicObs is 18
// and NumDynamics is one counter, incremented in *drawing* order, which is NW, NE, N,
// SW, SE, S, W, E, central (RoomGraphics.c:80-120). That is not master-index order --
// see the warning at ListAllLocalObjects -- so slot 0 belongs to the north-west room,
// and a saturated locale drops the *central* room's objects, the ones the player can
// see.
//
// **Only seven of the seventeen codes can register from a room other than the central
// one**, and they are the seven appliances. Every other registration is gated on
// `neighbor == kCentralRoom` at the call site (internal/render/locale.go:748, :837,
// :881, :889, :897, matching ObjectDrawAll.c:452, :648, :797-857, :868, :884). That
// single split explains four otherwise unrelated things: why those seven and only
// those seven put dest back into screen space below; why they draw into backSrcMap and
// therefore need no renderer; why CheckDynamicCollision needs a doOffset flag at all;
// and why dinahs[].room is nearly dead state (see UpdateOutletsLighting).

import (
	"glidergo/internal/house"
	"glidergo/internal/render"
)

// MaxDynamicObs is kMaxDynamicObs (GliderDefines.h:265): the whole locale's budget for
// things that move or blink.
//
// Eighteen across nine rooms is under two apiece, which is why saturation is reachable
// in the shipped houses at all. AddDynamicObject returns -1 past it, before touching
// the table.
const MaxDynamicObs = 18

// The per-type frame counts (GliderDefines.h:446-457). Every one is the wrap point of a
// `frame++` in some handler, so they belong beside the handlers rather than in
// consts.go with the table caps.
const (
	NumOutletPicts   int16 = 4  // kNumOutletPicts: frame 0 is the idle socket, 1..3 the spark
	NumBreadPicts    int16 = 6  // kNumBreadPicts
	NumBalloonFrames int16 = 8  // kNumBalloonFrames
	NumCopterFrames  int16 = 10 // kNumCopterFrames
	NumDartFrames    int16 = 4  // kNumDartFrames -- two directions, not an animation
	NumBallFrames    int16 = 2  // kNumBallFrames
	NumDripFrames    int16 = 6  // kNumDripFrames
	NumFishFrames    int16 = 8  // kNumFishFrames
	NumSparkleModes  int16 = 5  // kNumSparkleModes
)

// The three constants Dynamics3.c #defines for itself (Dynamics3.c:13-15) rather than
// in GliderDefines.h with the rest.
//
// BalloonStart and CopterStart are why a balloon or a copter ignores the y the author
// placed it at: both launch from a fixed height, the floor and the ceiling. Only the x
// survives. A house editor that lets you drag a balloon vertically is lying to the
// author, which is worth knowing before Stage 5.
const (
	BalloonStart int16 = 310 // kBalloonStart: every balloon starts at the floor
	CopterStart  int16 = 8   // kCopterStart: every copter starts at the ceiling
	DartVelocity int16 = 6   // kDartVelocity: and every dart crosses at six a frame
)

// LengthOfZap is kLengthOfZap (GliderDefines.h:546): how long an outlet sparks. Thirty
// frames, i.e. one second at TicksPerFrame = 2.
const LengthOfZap int16 = 30

// StartSparkle is kStartSparkle (GliderDefines.h:545): the timer value at which an
// enemy about to launch puffs. Four frames of warning, which is exactly how long
// RenderSparkles takes to run one puff down -- the two systems agree by construction,
// and TriggerBalloon's `+ 1` is what preserves the agreement.
const StartSparkle int16 = 4

// Dynamic is dynaType (GliderStructs.h): one entry in the dinahs table.
//
// Field order and field *names* are the original's, and four of the numeric names are
// lies in at least one type. That is not tidied, and the reason is worth stating once:
// renaming them per type needs seventeen structs and a tagged union, and then no
// handler is a statement-for-statement transcription of its C any more. The meaning is
// documented at each registration case instead.
//
//	Field      its name means      is also used as
//	HVel       horizontal velocity toaster: the launch clip line
//	                               outlet:  the room's light count
//	                               drip:    the rest position's top
//	                               fish:    the respawn delay, in frames
//	Count      a countdown         every timed type: the reload period
//	                               toaster/ball/fish: the initial velocity
//	Position   a coordinate        toaster/outlet: a launch/idle enum, 0 or 1
//	                               copter: the home x
//	                               dart:   the home y
//	                               ball/fish: the floor
//	Frame      a strip index       toaster: the idle countdown
//	                               dart:   0 or 2, a direction
//	                               VCR:    a two-state toggle, flipped as 1 - frame
//
// Byte0 and Byte1 are Byte in the C and are written through a (Byte) cast from a short
// index, so an object in slot >255 would truncate; MaxRoomObs is 24, so it cannot
// happen. The narrow type is kept anyway, because it is what makes the cast visible.
type Dynamic struct {
	Type        int16 // one of the 17 registrable codes, or ObjectIsEmpty
	Dest, Whole Rect  // this frame's blit; the union with last frame's
	HVel, VVel  int16
	Count       int16
	Frame       int16
	Timer       int16
	Position    int16
	Room        int16 // exactly one reader in the whole game: UpdateOutletsLighting
	Byte0       byte  // the object's slot within its room
	Byte1       byte
	Moving      bool // animating
	Active      bool // the switchable state
}

// ---------------------------------------------------------------------------
// ZeroDinahs (Dynamics3.c:160-180)
// ---------------------------------------------------------------------------

// ZeroDinahs empties the table. Part of DrawLocale's reset head (RoomGraphics.c:55),
// so every room change discards every moving object -- including one in mid-flight.
//
// **It writes twelve of the fourteen fields.** Byte1 and Moving are not touched, and in
// the original the table is NewPtr'd rather than NewPtrClear'd (StructuresInit2.c:261),
// so before the first registration those two hold whatever the Mac's memory manager
// left. Nothing reads a slot whose Type is ObjectIsEmpty, so it never mattered. Go's
// zero value covers them, which makes the port *stricter* than the original in a way
// nothing can observe -- said here so nobody later finds the omission and reports it as
// a bug.
//
// Transcribed as twelve assignments rather than `w.Dinahs[i] = Dynamic{}` for the same
// reason: the struct literal would hide which two are missing.
func (w *World) ZeroDinahs() {
	for i := 0; i < MaxDynamicObs; i++ {
		w.Dinahs[i].Type = house.ObjectIsEmpty
		w.Dinahs[i].Dest = render.SetRect(0, 0, 0, 0)
		w.Dinahs[i].Whole = render.SetRect(0, 0, 0, 0)
		w.Dinahs[i].HVel = 0
		w.Dinahs[i].VVel = 0
		w.Dinahs[i].Count = 0
		w.Dinahs[i].Frame = 0
		w.Dinahs[i].Timer = 0
		w.Dinahs[i].Position = 0
		w.Dinahs[i].Room = 0
		w.Dinahs[i].Byte0 = 0
		w.Dinahs[i].Active = false
	}
	w.NumDynamics = 0
}

// ---------------------------------------------------------------------------
// AddDynamicObject (Dynamics3.c:187-554)
// ---------------------------------------------------------------------------

// AddDynamicObject registers one object into the dinahs table and returns its slot, or
// -1.
//
// Reached from seventeen places inside DrawARoomsObjects, through the
// render.Scene.AddDynamicObject hook rather than a direct call: only the composition
// knows the rect, and only this package owns the table. The caller writes the returned
// slot back into the object graph -- see SetDynaNum.
//
// Three things about the shape:
//
// The Type write at the top happens *before* the default arm's -1 return, so an unknown
// code leaves Type set on the slot NumDynamics still points at. Harmless -- the next
// successful registration overwrites it, and nothing reads past NumDynamics -- and
// transcribed where it is anyway.
//
// The return is `NumDynamics - 1` after the increment, i.e. the slot just filled.
//
// **`where` is room-local, and seven cases add the origin straight back.** All
// seventeen call sites subtract playOriginH/V immediately before calling
// (ObjectDrawAll.c:455 and sixteen others), and then MacPlus, TV, Coffee, Outlet, VCR,
// Stereo and Microwave -- and only those seven -- write `where.Left + OriginH + dx`,
// putting dest back into *screen* space. Those seven are exactly the seven not gated on
// the central room, so their art has to land in backSrcMap, which spans the whole
// locale rather than one room. The other eight stay room-local and have the origin
// added at draw time instead. CheckDynamicCollision's doOffset flag is what reconciles
// the two spaces.
func (w *World) AddDynamicObject(what int16, where Rect, who house.Object, room, index int16, isOn bool) int16 {
	if w.NumDynamics >= MaxDynamicObs {
		return -1
	}

	d := &w.Dinahs[w.NumDynamics]
	d.Type = what

	// playOriginH/V, added back by the seven appliance cases only.
	originH := w.R.V.OriginH
	originV := w.R.V.OriginV

	switch what {
	case Sparkle:
		// A sparkle *emitter*. It draws nothing itself: HandleSparkleObject pushes
		// entries into the transient sparkles table and this slot is only the timer
		// behind them.
		d.Dest = render.ZeroCorner(render.SparkleSrc[0])
		d.Dest = render.Offset(d.Dest, where.Left, where.Top)
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = 0
		d.Count = 0
		d.Frame = 0
		// The only RandomInt in this function, and it is per registration -- so the
		// RNG stream position after a room change is a function of how many sparkles
		// the locale registered, which is a function of what intersected the window,
		// which is a function of resolution. Hazard H6; see IMPROVEMENTS 2.18.
		d.Timer = w.RandomInt(60) + 15
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case Toaster:
		// dest is the *bread*, not the toaster: centred horizontally in the object and
		// then pushed down until its top sits on the object's top, so the slice starts
		// fully hidden inside the slot.
		d.Dest = render.BreadSrc[0]
		d.Dest = render.CenterIn(d.Dest, where)
		d.Dest = render.Offset(d.Dest, 0, where.Top-d.Dest.Top)
		d.Whole = d.Dest
		d.HVel = where.Top + 2 // hVel used as clip
		velocity := launchVelocity(who.Appliance().Height)
		d.VVel = -velocity
		d.Count = velocity // count = initial velocity
		// frame is the idle countdown here and the strip index while airborne; timer
		// is the reload period it is refilled from. Both g.delay * 3, which is
		// (delay * 6) / TicksPerFrame written the short way.
		d.Frame = int16(who.Appliance().Delay) * 3
		d.Timer = d.Frame
		d.Position = 0 // launch/idle state
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case MacPlus:
		d.Dest = render.ZeroCorner(render.PlusScreen1)
		d.Dest = render.Offset(d.Dest, where.Left+originH+10, where.Top+originV+7)
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = 0
		d.Count = 0
		d.Frame = 0
		d.Timer = 0
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case TV:
		d.Dest = render.ZeroCorner(render.TVScreen1)
		d.Dest = render.Offset(d.Dest, where.Left+originH+17, where.Top+originV+10)
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = 0
		d.Count = 0
		d.Frame = 0
		d.Timer = 0
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case Coffee:
		d.Dest = render.ZeroCorner(render.CoffeeLight1)
		d.Dest = render.Offset(d.Dest, where.Left+originH+32, where.Top+originV+57)
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = 0
		d.Count = 0
		d.Frame = 0
		// 200 is not a reload period. HandleCoffee overwrites it with
		// 200 + RandomInt(200) the first time it reaches zero, and never settles.
		if isOn {
			d.Timer = 200
		} else {
			d.Timer = 0
		}
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case Outlet:
		// The one registration that reads the room's lighting, and the reason
		// Scene.NumLights has to be live per-room state rather than a lookup:
		// DrawLocale reassigns it before each of the nine draws, so each room's
		// outlets record their own room's count. UpdateOutletsLighting maintains it
		// afterwards and HandleOutlet is the single reader.
		d.Dest = render.ZeroCorner(render.OutletSrc[0])
		d.Dest = render.Offset(d.Dest, where.Left+originH, where.Top+originV)
		d.Whole = d.Dest
		d.HVel = int16(w.R.NumLights)
		d.VVel = 0
		d.Count = (int16(who.Appliance().Delay) * 6) / TicksPerFrame
		d.Frame = 0
		d.Timer = d.Count
		d.Position = 0 // launch/idle state
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case VCR:
		d.Dest = render.ZeroCorner(render.VCRTime1)
		d.Dest = render.Offset(d.Dest, where.Left+originH+64, where.Top+originV+6)
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = 0
		d.Count = 0
		d.Frame = 0
		// Like the coffee maker's 200: HandleVCR bounces the timer between 115 and 100
		// for ever, fifteen frames per half-blink.
		if isOn {
			d.Timer = 115
		} else {
			d.Timer = 0
		}
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case Stereo:
		d.Dest = render.ZeroCorner(render.StereoLight1)
		d.Dest = render.Offset(d.Dest, where.Left+originH+56, where.Top+originV+20)
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = 0
		d.Count = 0
		d.Frame = 0
		d.Timer = 0
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case Microwave:
		// The slot is forced to 48 wide over 16-wide art, and HandleMicrowave pays
		// that back by laying the window down three times. Neither half makes sense
		// without the other. Hazard H5.
		d.Dest = render.ZeroCorner(render.MicroOn)
		d.Dest = render.Offset(d.Dest, where.Left+originH+14, where.Top+originV+13)
		d.Dest.Right = d.Dest.Left + 48
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = 0
		d.Count = 0
		d.Frame = 0
		d.Timer = 0
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case Balloon:
		// **The authored y is discarded.** Only where.Left survives; the balloon is
		// placed at BalloonStart, the floor, and rises from there. Same for the copter
		// below, at the ceiling. A dart discards the authored x instead.
		d.Dest = render.ZeroCorner(render.BalloonSrc[0])
		d.Dest = render.Offset(d.Dest, where.Left, 0)
		d.Dest.Bottom = BalloonStart
		d.Dest.Top = d.Dest.Bottom - render.BalloonSrc[0].Tall()
		d.Whole = d.Dest
		d.HVel = 0
		d.VVel = -2
		d.Count = (int16(who.Enemy().Delay) * 6) / TicksPerFrame
		d.Frame = 0
		d.Timer = d.Count
		d.Position = 0
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn // initially idle

	case CopterLf, CopterRt:
		d.Dest = render.ZeroCorner(render.CopterSrc[0])
		d.Dest = render.Offset(d.Dest, where.Left, 0)
		d.Dest.Top = CopterStart
		d.Dest.Bottom = d.Dest.Top + render.CopterSrc[0].Tall()
		d.Whole = d.Dest
		// hVel is the drift, one pixel a frame, and HandleCopter clears it to zero
		// when a rubber band knocks the copter out of the sky. So hVel == 0 on a
		// copter means "shot", not "stationary".
		if what == CopterLf {
			d.HVel = -1
		} else {
			d.HVel = 1
		}
		d.VVel = 2
		d.Count = (int16(who.Enemy().Delay) * 6) / TicksPerFrame
		d.Frame = 0
		d.Timer = d.Count
		d.Position = d.Dest.Left // the home x
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn // initially idle

	case DartLf, DartRt:
		// **The authored x is discarded** -- a dart always enters from the wall it
		// flies away from. Only where.Top survives. And note the left edge: a leftward
		// dart starts at RoomWide *minus its own width*, flush against the right wall
		// and fully on screen, not one width past it.
		d.Dest = render.ZeroCorner(render.DartSrc[0])
		if what == DartLf {
			d.Dest = render.Offset(d.Dest, RoomWide-render.DartSrc[0].Wide(), where.Top)
			d.HVel = -DartVelocity
			d.Frame = 0
		} else {
			d.Dest = render.Offset(d.Dest, 0, where.Top)
			d.HVel = DartVelocity
			d.Frame = 2
		}
		d.Whole = d.Dest
		d.VVel = 2
		d.Count = (int16(who.Enemy().Delay) * 6) / TicksPerFrame
		d.Timer = d.Count
		d.Position = d.Dest.Top // the home y
		// **No Room assignment.** The only registration case that omits it, and
		// harmless by construction rather than by luck: Room has exactly one reader,
		// UpdateOutletsLighting, which filters on Type == Outlet, and an outlet is one
		// of the seven multi-room appliances. A dart only ever registers from the
		// central room, so its Room could hold nothing the filter needs to tell apart.
		// Do not "complete" this case; see TestDartRegistrationLeavesRoomZero.
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn // initially idle

	case Ball:
		d.Dest = render.ZeroCorner(render.BallSrc[0])
		d.Dest = render.Offset(d.Dest, where.Left, where.Top)
		d.Whole = d.Dest
		d.HVel = 0
		// The half-rate loop, and the vestigial EvenFrame write with it. See
		// launchVelocityHalfRate.
		velocity := w.launchVelocityHalfRate(who.Enemy().Length)
		d.VVel = -velocity
		d.Moving = false
		d.Count = -velocity // count = initial velocity -- negative here, unlike the toaster's
		d.Frame = 0
		d.Timer = 0
		d.Position = d.Dest.Bottom // the floor
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Active = isOn

	case Drip:
		d.Dest = render.DripSrc[0]
		d.Dest = render.CenterIn(d.Dest, where)
		d.Dest = render.Offset(d.Dest, 0, where.Top-d.Dest.Top)
		d.Whole = d.Dest
		d.HVel = d.Dest.Top // remember where to hang back up
		d.VVel = 0
		d.Count = (int16(who.Enemy().Delay) * 6) / TicksPerFrame
		// Frame 3 is the hanging drop, which is also what DrawDrip paints into the
		// static background -- so a drip at rest and a drip not yet registered look
		// the same, which is what makes the composition's static draw sufficient.
		d.Frame = 3
		d.Timer = d.Count
		d.Position = d.Dest.Top + int16(who.Enemy().Length) // the floor it falls to
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	case Fish:
		// The one case with no ZeroRectCorner. FishSrc[0] is already at the origin, so
		// the author's omission is a no-op rather than a bug -- and it is why the
		// offset here is a plain +10,+8 from where's top-left.
		d.Dest = render.FishSrc[0]
		d.Dest = render.Offset(d.Dest, where.Left+10, where.Top+8)
		d.Whole = d.Dest
		d.HVel = (int16(who.Enemy().Delay) * 6) / TicksPerFrame // the respawn delay
		// **The C reads who->data.g.height here, not h.length**, and it is not a bug:
		// applianceType.height and enemyType.length are both the short at payload
		// offset 4 (GliderStructs.h:64-82), so the wrong union arm names the right two
		// bytes. Enemy().Length is used because that is the family this object belongs
		// to; Appliance().Height would read the same bytes, which is exactly why the
		// choice needs a comment rather than being left to whoever edits this next.
		// The two delay fields do *not* alias -- see launchVelocity's note.
		velocity := w.launchVelocityHalfRate(who.Enemy().Length)
		d.VVel = -velocity
		d.Count = -velocity // count = initial velocity, negative like the ball's
		d.Frame = 0
		d.Timer = d.HVel
		d.Position = d.Dest.Bottom // the water
		d.Room = room
		d.Byte0 = byte(index)
		d.Byte1 = 0
		d.Moving = false
		d.Active = isOn

	default:
		return -1
	}

	w.NumDynamics++
	return w.NumDynamics - 1
}

// launchVelocity is the toaster's arm of the author's "reverse engineer init. vel."
// (Dynamics3.c:224-231): the house stores how *high* the thing should go, and this
// solves for the launch speed by simulating gravity until the steps cover it. Returns
// the speed positive; the caller applies the sign.
//
// **The post-test is load-bearing.** The C is a do/while, so a height of 0 still runs
// one iteration and yields 1 -- a toaster with height 0 launches, one pixel. Go has no
// do/while, so the break has to be at the bottom.
//
// Not clamped, and height is a short straight out of the house file, so a hand-edited
// 32767 spins about 181 times. Not a hang; left alone.
//
// Note which union arm each caller reads. applianceType.height and enemyType.length
// alias -- both the short at payload offset 4 -- but applianceType.delay and
// enemyType.delay do *not*: offset 7 versus offset 6 (GliderStructs.h:69, :78).
// internal/house has this right already, and every registration case above reads the
// delay arm matching its own family.
func launchVelocity(height int16) int16 {
	position := height
	velocity := int16(0)
	for {
		velocity++
		position -= velocity
		if position <= 0 {
			break
		}
	}
	return velocity
}

// launchVelocityHalfRate is the ball and fish arm of the same derivation
// (Dynamics3.c:472-483, :522-533), where velocity rises only on alternate steps because
// HandleBall and HandleFish apply gravity at half rate.
//
// **It writes EvenFrame, and that is hazard H1.** In AddDynamicObject `lilFrame` is a
// local (Dynamics3.c:191) while `evenFrame` is an extern global (:23); the author
// seeded what he took to be two loop-local toggles and one of the names resolved to a
// global. The proof that it is vestigial here is that nothing in AddDynamicObject reads
// evenFrame, and that the toaster's loop -- full rate, needing no toggle -- writes
// neither flag. The line was copied from HandleBall's launch arm (Dynamics2.c:420),
// where the same write *is* deliberate: it phase-locks the half-rate gravity to the
// launch so the reverse-engineered velocity actually reaches the height the author
// typed.
//
// Transcribed anyway, because the collateral effect is visible: EvenFrame selects
// RenderFlames versus RenderStars (Render.c:649-652) and gates every half-rate mover,
// so entering a room whose composition registers a ball resynchronises every flame and
// star in the locale. That is the original's behaviour and it is the kind of thing
// 1.8's fidelity replays exist to hold.
func (w *World) launchVelocityHalfRate(height int16) int16 {
	position := height
	velocity := int16(0)
	w.EvenFrame = true // H1: vestigial here, deliberate at Dynamics2.c:420
	lilFrame := true
	for {
		if lilFrame {
			velocity++
		}
		lilFrame = !lilFrame
		position -= velocity
		if position <= 0 {
			break
		}
	}
	return velocity
}

// ---------------------------------------------------------------------------
// SetDynaNum -- the write-back (ObjectDrawAll.c:952-960)
// ---------------------------------------------------------------------------

// SetDynaNum records what a room object's registration returned, for every one of the
// room's 24 object slots.
//
// This is the piece that makes every Trigger* and Toggle* reachable at all. Without it
// AddDynamicObject returns a slot number nobody stores and MasterObject.DynaNum stays
// -1 for ever.
//
// **DynaNum is a union over three tables**, and the port keeps it as one field because
// the C's write-back is one loop over one field:
//
//	assigned at              for                  the number is
//	ObjectDrawAll.c:374-400  kGreaseRt, kGreaseLf a grease slot
//	:456, :652-857           the 17 dynamic codes a dinahs slot
//	:518-574                 the 6 switch types   a hotSpots slot
//
// So TriggerSwitch(dynaNum) -> HandleSwitches(&hotSpots[who]) is not a type confusion:
// for a switch, DynaNum *is* the hot-spot index. Splitting the field into three would
// force the write-back to know which kind it wrote, which is exactly the information
// the C's local `dynamicNum` deliberately discards.
//
// It must be called with -1 too, not only with a real slot. The C resets
// `dynamicNum = -1` at the top of every object iteration (:45), and that reset is what
// clears a stale slot number left by the previous room; skipping the -1 call leaves the
// graph pointing into the last room's table. The C's write-back is also *outside* its
// IsThisValid guard, so empty slots get the -1 as well.
func (w *World) SetDynaNum(room, obj, dyna int16) {
	for n := range w.R.Master {
		if w.R.Master[n].ObjectNum == obj && w.R.Master[n].RoomNum == room {
			w.R.Master[n].DynaNum = dyna
		}
	}
}

// MasterHotNum answers the six switch cases' `dynamicNum = masterObjects[i].hotNum`
// (ObjectDrawAll.c:518, :531, :544, :557, :570, :574).
//
// A read hook rather than a call, for the same reason SetDynaNum is a write hook: the
// master table is this package's and the switch statement that needs it is
// internal/render's.
//
// **The index is wrong for eight of the nine rooms, and that is transcribed, not
// fixed** -- hazard H2. The C indexes the master table with `i`, the object's slot
// within its room (0..23), rather than with the master index the write-back loop below
// it then searches for. ListOneRoomsObjects lists the central room first and appends
// all 24 slots in order (objects.go:231-271), so for the central room Master[i] *is*
// object i and the line is accidentally right. For the other eight rooms it reads the
// central room's object i instead, whose HotNum is some unrelated hot spot or -1.
//
// Consequence: a trigger wired to a switch in a neighbouring room throws whichever
// central-room hot spot sits at the same slot number, or reads hotSpots[-1]. Both are
// the original's behaviour, and a shipped house with a cross-room-triggered switch
// depends on the wrong number, so 1.8's fidelity replays are the only thing that could
// tell us which houses those are. Do not "fix" it to the master index.
func (w *World) MasterHotNum(obj int16) int16 {
	if w.badIndex(devMasterObject, int(obj), len(w.R.Master)) {
		return -1
	}
	return w.R.Master[obj].HotNum
}

// ---------------------------------------------------------------------------
// HandleDynamics (Dynamics3.c:31-105) and RenderDynamics (:112-154)
// ---------------------------------------------------------------------------

// HandleDynamics advances every registered object one frame. Fifteen handlers over
// seventeen labels -- CopterLf/CopterRt share HandleCopter and DartLf/DartRt share
// HandleDart -- and the C has no default arm.
//
// Called from PlayGame's loop body ungated by GameOver, so the room keeps moving
// through the death countdown. Without it every ball, fish, dart, balloon, toaster and
// enemy stands still, and the rooms that are pure obstacle courses are walkable.
//
// The loop bounds on NumDynamics and does not look at Room: **every registered object
// is simulated every frame, wherever it is.** There is nothing to cull and nothing to
// gate -- only the seven appliances can be in a neighbouring room in the first place,
// and they are drawn into the shared background where the player can see them through
// an open doorway.
func (w *World) HandleDynamics() {
	for i := int16(0); i < w.NumDynamics; i++ {
		switch w.Dinahs[i].Type {
		case Sparkle:
			w.HandleSparkleObject(i)
		case Toaster:
			w.HandleToast(i)
		case MacPlus:
			w.HandleMacPlus(i)
		case TV:
			w.HandleTV(i)
		case Coffee:
			w.HandleCoffee(i)
		case Outlet:
			w.HandleOutlet(i)
		case VCR:
			w.HandleVCR(i)
		case Stereo:
			w.HandleStereo(i)
		case Microwave:
			w.HandleMicrowave(i)
		case Balloon:
			w.HandleBalloon(i)
		case CopterLf, CopterRt:
			w.HandleCopter(i)
		case DartLf, DartRt:
			w.HandleDart(i)
		case Ball:
			w.HandleBall(i)
		case Drip:
			w.HandleDrip(i)
		case Fish:
			w.HandleFish(i)
		default:
			// Includes ObjectIsEmpty. The C has no default at all.
		}
	}
}

// RenderDynamics draws the seven types that have a renderer, from RenderFrame between
// RenderFlames/RenderStars and RenderFlyingPoints.
//
// The other ten -- the sparkle emitter and the nine appliance arms -- are absent
// because their *handler* draws them, into backSrcMap, once per state change. That is
// the whole reason this switch is shorter than HandleDynamics'.
func (w *World) RenderDynamics() {
	for i := int16(0); i < w.NumDynamics; i++ {
		switch w.Dinahs[i].Type {
		case Toaster:
			w.RenderToast(i)
		case Balloon:
			w.RenderBalloon(i)
		case CopterLf, CopterRt:
			w.RenderCopter(i)
		case DartLf, DartRt:
			w.RenderDart(i)
		case Ball:
			w.RenderBall(i)
		case Drip:
			w.RenderDrip(i)
		case Fish:
			w.RenderFish(i)
		default:
		}
	}
}

// dinah resolves a DynaNum to a table slot, refusing -1 and out-of-range.
//
// The C does not: all fourteen Toggle* and all eight Trigger* index dinahs[index]
// directly, and reach it from masterObjects[].dynaNum, which is -1 for any object that
// never registered -- one past the 18-slot cap, or in a locale composed with
// redraw = true. So dinahs[-1] is reachable in the original from a switch wired to a
// nineteenth appliance. Refused here and counted; see IMPROVEMENTS 2.33.
//
// One helper for twenty-two call sites rather than a badIndex call in each: the
// fourteen Toggle* are one line long apiece and three lines of guard would bury them.
func (w *World) dinah(index int16) *Dynamic {
	if w.badIndex(devDynamic, int(index), int(w.NumDynamics)) {
		return nil
	}
	return &w.Dinahs[index]
}
