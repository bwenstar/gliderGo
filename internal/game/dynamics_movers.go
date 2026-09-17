package game

// The six movers: all of Dynamics2.c (588 lines).
//
// Balloon, copter, dart, ball, drip and fish -- the six dynamic types that are hazards
// rather than scenery, and the only six whose Dest changes every frame.
//
// ---------------------------------------------------------------------------
// Three shapes, not six
// ---------------------------------------------------------------------------
//
// Every one is `if (moving) { ...fly... } else { ...wait... }`, and what "done" means
// sorts them into three families:
//
//	fly-and-respawn   balloon, copter, dart   leaves the room, sparkles out, and is put
//	                                          back at its start line to wait out Count
//	bounce            ball                    never leaves; an active ball bounces at
//	                                          constant energy forever
//	drop-and-rest     drip, fish              falls to a fixed line and sits there
//
// The three in the first family share their whole idle arm verbatim (enemyWaiting) and
// the head of their retire block (enemyRetire). The other three each wait differently,
// and the differences are the interesting part: a ball's idle arm is three statements
// long and one of them writes a global, a drip's is a four-way switch on the timer that
// animates a swelling drop, and a fish's runs its bob animation *outside* the Active
// test.
//
// ---------------------------------------------------------------------------
// Two asymmetries worth knowing before reading any of them
// ---------------------------------------------------------------------------
//
// **EvenFrame gating is per handler and follows no rule.** The balloon animates on even
// frames, the copter every frame; the ball, drip and fish gain gravity on even frames,
// the toast every frame; the drip flips its two falling cels on even frames, the dart
// never animates at all. There is no principle behind the pattern -- it is what each
// animation happened to look right at -- so every gate is transcribed where it is and
// none is factored out.
//
// **Whole is rebuilt from Dest every frame, by pushing the trailing edge back one
// frame's travel.** Five of the six do it; the drip does not do it in its idle arm,
// which leaves a stale union on screen (see HandleDrip). The renderers then blit at
// Dest and register Whole, so the union is what covers the trail -- and because it is
// recomputed rather than accumulated it cannot drift.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The four remaining file-local #defines from Dynamics2.c:11-17. BalloonStart,
// CopterStart and DartVelocity are the other three and are in dynamics.go with the
// registration that reads them.
//
// **Four names, two values.** BalloonStop and CopterStart are both 8 and both mean
// "the ceiling"; BalloonStart, CopterStop and DartStop are all 310 and all mean "the
// floor". A balloon rises from the floor to the ceiling and a copter descends from the
// ceiling to the floor, so each type needs both lines under its own two names, and the
// author wrote five #defines for two numbers rather than share them. Kept apart,
// because a house whose room height changed would need them apart.
const (
	BalloonStop    int16 = 8   // kBalloonStop: a rising balloon is retired here
	CopterStop     int16 = 310 // kCopterStop: and a descending copter here
	DartStop       int16 = 310 // kDartStop: a dart that never reaches a side wall
	EnemyDropSpeed int16 = 8   // kEnemyDropSpeed: how fast a shot balloon or copter drops
)

// The nine mover sounds (GliderDefines.h:92-100, :136-138, :148-153).
//
// EnemyIn and EnemyOut are shared by all three respawning types, which is why a dart
// and a balloon leaving the room sound identical. The other seven belong to one handler
// each.
const (
	PopSound         int16 = 37 // a balloon hit by a rubber band
	EnemyInSound     int16 = 38 // the puff four frames before any enemy appears
	EnemyOutSound    int16 = 39 // and the puff as it leaves
	PaperCrunchSound int16 = 40 // a copter or a dart hit by a rubber band
	BounceSound      int16 = 41
	DripSound        int16 = 42 // the drop letting go of the ceiling
	DropSound        int16 = 43 // and landing; also the fish hitting the water
	FishOutSound     int16 = 44 // a fish leaping
	FishInSound      int16 = 45 // and landing, on top of DropSound -- see HandleFish
)

const (
	BouncePriority      int16 = 307
	DripPriority        int16 = 308
	DropPriority        int16 = 309
	PopPriority         int16 = 407
	EnemyInPriority     int16 = 408
	EnemyOutPriority    int16 = 409
	PaperCrunchPriority int16 = 410
	FishOutPriority     int16 = 411
	FishInPriority      int16 = 412
)

// ---------------------------------------------------------------------------
// The two blocks the three respawning enemies share
// ---------------------------------------------------------------------------

// enemyWaiting is the idle arm of HandleBalloon, HandleCopter and HandleDart
// (Dynamics2.c:103-129, :215-236, :329-352), which are byte-for-byte identical.
//
// Count is the reload period and Timer counts it down. Two ways out, and the second is
// the interesting one:
//
//	Timer <= 0            launch, and puff *only if* Count < StartSparkle
//	Timer == StartSparkle puff, four frames early
//
// **The Count test is what guarantees at most one puff.** With a reload period of five
// frames or more the timer passes through 4 on its way down, so the warning puff fires
// then and the launch itself is silent. With a period of three or less the timer never
// equals 4, so the puff is emitted at launch instead. Neither case can produce two,
// which is the whole point -- a dart that appeared unannounced would be unavoidable, and
// a doubled puff would look like two darts. TriggerBalloon's `+ 1` (see trip.go) exists
// to keep this true for a switch-launched enemy too.
//
// **Count == 4 is a hole and gets no puff at all.** Timer is reset *to* Count and then
// decremented before it is compared, so a four-frame reload sees 3, 2, 1, 0 and never 4 --
// while `Count < StartSparkle` is also false. It is unreachable in any house, because all
// four registration arms compute `Count = (Delay * 6) / TicksPerFrame` with TicksPerFrame
// 2, i.e. `Delay * 3`, and 4 is not a multiple of three. Do not "fix" the boundary to close
// it: the arithmetic is the guarantee, and changing either test would change a reachable
// period's behaviour to close an unreachable one's. See
// TestEnemyWaitingHasOneSilentPeriodAndItIsUnreachable and docs/IMPROVEMENTS.md 2.36, which
// says why an editor or a multiplayer tuning knob could still reach it.
//
// Nothing happens at all while Active is false: a switched-off enemy's timer freezes,
// so switching it back on resumes the countdown rather than restarting it. Same
// behaviour as the sparkle emitter's lockout, and for the same reason -- there is no
// `else`.
func (w *World) enemyWaiting(who int16) {
	if w.Dinahs[who].Active {
		w.Dinahs[who].Timer--
		if w.Dinahs[who].Timer <= 0 {
			w.Dinahs[who].Moving = true
			if w.Dinahs[who].Count < StartSparkle {
				w.AddSparkle(w.Dinahs[who].Dest)
				w.PlayPrioritySound(EnemyInSound, EnemyInPriority)
			}
		} else if w.Dinahs[who].Timer == StartSparkle {
			w.AddSparkle(w.Dinahs[who].Dest)
			w.PlayPrioritySound(EnemyInSound, EnemyInPriority)
		}
	}
}

// enemyRetire is the head of the retire block, shared by the same three
// (Dynamics2.c:88-95, :193-200, :297-304): erase the trail, puff where the enemy was,
// play the leaving sound. Everything after it is per type -- where the start line is
// and which fields the reset touches -- so only the head is shared.
//
// **The work rect is Whole offset by playOrigin; the sparkle is Dest *un-offset*.** The
// two adjacent lines disagree and both are right: AddSparkle adds playOrigin itself
// (DynamicMaps.c:176-179), so handing it a screen rect would put the puff one whole
// origin down and to the right. The C makes the asymmetry look like a slip by reusing
// one local named `dest` for both.
func (w *World) enemyRetire(who int16) {
	dest := render.Offset(w.Dinahs[who].Whole, w.R.V.OriginH, w.R.V.OriginV)
	w.AddRectToWorkRects(player.Rect(dest))
	w.AddSparkle(w.Dinahs[who].Dest)
	w.PlayPrioritySound(EnemyOutSound, EnemyOutPriority)
}

// ---------------------------------------------------------------------------
// HandleBalloon (Dynamics2.c:27-130)
// ---------------------------------------------------------------------------

// HandleBalloon rises a balloon from the floor to the ceiling, or drops a popped one.
//
// **VVel < 0 means "intact".** There is no separate popped flag: a rising balloon has a
// negative velocity, and popping it writes EnemyDropSpeed into VVel, which is how the
// next frame takes the other arm. That is why the popped arm has no band test -- a
// balloon cannot be shot twice -- and why it does not call CheckDynamicCollision: **a
// popped balloon is harmless**, and falls through the player without touching them.
//
// The two frame ranges are the two states: 0..5 is the balloon, 6..7 the burst. Both
// wraps are gated on EvenFrame, so the animation runs at 15 fps against the copter's
// 30. The popped arm's wrap looks like a loop and is one -- frames 6 and 7 alternate
// all the way down -- because there is no "gone" cel.
//
// The retire test fires at either end, so a balloon that reaches the ceiling and a
// popped one that reaches the floor take the same exit and both sparkle out. The reset
// puts it back on the floor with VVel = -2, and **only Dest.Bottom is placed** --
// Dest.Left and Dest.Right are left wherever the balloon was, which is where it started,
// since nothing ever moves a balloon sideways.
func (w *World) HandleBalloon(who int16) {
	if w.Dinahs[who].Moving {
		if w.Dinahs[who].VVel < 0 { // intact, rising
			if w.EvenFrame {
				w.Dinahs[who].Frame++
				if w.Dinahs[who].Frame >= 6 {
					w.Dinahs[who].Frame = 0
				}
			}
			w.checkGliders(who, false)
			if w.NumBands > 0 && w.DidBandHitDynamic(who) {
				w.Dinahs[who].Frame = 6
				w.Dinahs[who].VVel = EnemyDropSpeed
				w.PlayPrioritySound(PopSound, PopPriority)
			} else {
				w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
				w.Dinahs[who].Whole = w.Dinahs[who].Dest
				w.Dinahs[who].Whole.Bottom -= w.Dinahs[who].VVel
			}
		} else { // popped, falling
			if w.EvenFrame {
				w.Dinahs[who].Frame++
				if w.Dinahs[who].Frame >= 8 {
					w.Dinahs[who].Frame = 6
				}
			}
			w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
		}

		if w.Dinahs[who].Dest.Top <= BalloonStop || w.Dinahs[who].Dest.Bottom >= BalloonStart {
			w.enemyRetire(who)
			w.Dinahs[who].Moving = false
			w.Dinahs[who].VVel = -2
			w.Dinahs[who].Timer = w.Dinahs[who].Count
			w.Dinahs[who].Dest.Bottom = BalloonStart
			w.Dinahs[who].Dest.Top = w.Dinahs[who].Dest.Bottom - render.BalloonSrc[0].Tall()
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
		}
	} else {
		w.enemyWaiting(who)
	}
}

// ---------------------------------------------------------------------------
// HandleCopter (Dynamics2.c:134-238)
// ---------------------------------------------------------------------------

// HandleCopter flies a paper helicopter down and across the room, or drops a shot one.
//
// **HVel != 0 means "not shot".** The same trick as the balloon's VVel, on the other
// axis: shooting a copter zeroes HVel, so it stops drifting sideways and the next frame
// takes the falling arm. And as with the balloon, the falling arm neither tests for
// bands nor checks the gliders -- **a crumpled copter is harmless.**
//
// Its animation is *not* EvenFrame-gated where the balloon's is, so the rotor spins at
// the full 30 fps. Frames 0..7 fly, 8..9 crumple.
//
// The union needs both edges here, and the horizontal one has to pick a side: the copter
// drifts left or right depending on its type, so the trailing edge is Right for a
// leftward copter and Left for a rightward one. Note that both lines *subtract* HVel,
// exactly as the toast's two vertical lines both subtract VVel -- the sign does the
// work.
//
// The reset places three edges from CopterStart and Position and the fourth as
// `Left + 32`, a bare literal where the vertical uses RectTall(copterSrc[0]). Both are
// 32 and 30 respectively for the shipped art, so the mixture is harmless; it is
// transcribed as written because that is what a copter sheet of a different width would
// have to match.
func (w *World) HandleCopter(who int16) {
	if w.Dinahs[who].Moving {
		if w.Dinahs[who].HVel != 0 { // not shot
			w.Dinahs[who].Frame++
			if w.Dinahs[who].Frame >= 8 {
				w.Dinahs[who].Frame = 0
			}
			w.checkGliders(who, false)
			if w.NumBands > 0 && w.DidBandHitDynamic(who) {
				w.Dinahs[who].Frame = 8
				w.Dinahs[who].HVel = 0
				w.Dinahs[who].VVel = EnemyDropSpeed
				w.PlayPrioritySound(PaperCrunchSound, PaperCrunchPriority)
			} else {
				w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, w.Dinahs[who].HVel, w.Dinahs[who].VVel)
				w.Dinahs[who].Whole = w.Dinahs[who].Dest
				w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
				if w.Dinahs[who].HVel < 0 {
					w.Dinahs[who].Whole.Right -= w.Dinahs[who].HVel
				} else {
					w.Dinahs[who].Whole.Left -= w.Dinahs[who].HVel
				}
			}
		} else { // shot
			w.Dinahs[who].Frame++
			if w.Dinahs[who].Frame >= 10 {
				w.Dinahs[who].Frame = 8
			}
			w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
		}

		if w.Dinahs[who].Dest.Top <= CopterStart || w.Dinahs[who].Dest.Bottom >= CopterStop {
			w.enemyRetire(who)
			w.Dinahs[who].Moving = false
			w.Dinahs[who].VVel = 2
			if w.Dinahs[who].Type == CopterLf {
				w.Dinahs[who].HVel = -1
			} else {
				w.Dinahs[who].HVel = 1
			}
			w.Dinahs[who].Timer = w.Dinahs[who].Count
			w.Dinahs[who].Dest.Top = CopterStart
			w.Dinahs[who].Dest.Bottom = w.Dinahs[who].Dest.Top + render.CopterSrc[0].Tall()
			w.Dinahs[who].Dest.Left = w.Dinahs[who].Position
			w.Dinahs[who].Dest.Right = w.Dinahs[who].Dest.Left + 32
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
		}
	} else {
		w.enemyWaiting(who)
	}
}

// ---------------------------------------------------------------------------
// HandleDart (Dynamics2.c:242-354)
// ---------------------------------------------------------------------------

// HandleDart flies a paper dart across the room at a constant six pixels a frame.
//
// The third of the HVel/VVel state tricks: **HVel != 0 means "not shot"**, and a shot
// dart falls straight down with no collision test and no band test, harmless like the
// popped balloon and the crumpled copter.
//
// **A dart has no animation.** Frames 0..1 are the leftward dart and 2..3 the rightward
// one, nothing ever advances Frame, and the only writes to it are the direction at reset
// (0 or 2) and the crumple (1 or 3). Four cels, two directions, no frame counter -- which
// is why NumDartFrames appears nowhere in this file.
//
// **It also has no gravity.** Neither arm touches VVel, so the dart holds whatever it
// was reset with -- 2, from the reset below -- and crosses the room on a shallow constant
// slope rather than an arc. A dart therefore always arrives at the same height on the far
// wall, which is what makes rooms full of them learnable.
//
// The retire test is the only three-way one in the file, because a dart can leave by
// either side wall or by the floor. RoomWide is the right-hand test even though the dart
// is drawn from a 64-wide cel: it is Dest.Right that has to reach 512, so the visible
// dart is already outside the room by the time it retires.
func (w *World) HandleDart(who int16) {
	if w.Dinahs[who].Moving {
		if w.Dinahs[who].HVel != 0 { // not shot
			w.checkGliders(who, false)
			if w.NumBands > 0 && w.DidBandHitDynamic(who) {
				if w.Dinahs[who].Type == DartLf {
					w.Dinahs[who].Frame = 1
				} else {
					w.Dinahs[who].Frame = 3
				}
				w.Dinahs[who].HVel = 0
				w.Dinahs[who].VVel = EnemyDropSpeed
				w.PlayPrioritySound(PaperCrunchSound, PaperCrunchPriority)
			} else {
				w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, w.Dinahs[who].HVel, w.Dinahs[who].VVel)
				w.Dinahs[who].Whole = w.Dinahs[who].Dest
				w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
				if w.Dinahs[who].HVel < 0 {
					w.Dinahs[who].Whole.Right -= w.Dinahs[who].HVel
				} else {
					w.Dinahs[who].Whole.Left -= w.Dinahs[who].HVel
				}
			}
		} else { // shot, falling straight down
			w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
		}

		if w.Dinahs[who].Dest.Left <= 0 || w.Dinahs[who].Dest.Right >= RoomWide ||
			w.Dinahs[who].Dest.Bottom >= DartStop {
			w.enemyRetire(who)
			w.Dinahs[who].Moving = false
			w.Dinahs[who].VVel = 2
			if w.Dinahs[who].Type == DartLf {
				w.Dinahs[who].Frame = 0
				w.Dinahs[who].HVel = -DartVelocity
				w.Dinahs[who].Dest.Right = RoomWide
				w.Dinahs[who].Dest.Left = w.Dinahs[who].Dest.Right - render.DartSrc[0].Wide()
			} else {
				w.Dinahs[who].Frame = 2
				w.Dinahs[who].HVel = DartVelocity
				w.Dinahs[who].Dest.Left = 0
				w.Dinahs[who].Dest.Right = w.Dinahs[who].Dest.Left + render.DartSrc[0].Wide()
			}
			w.Dinahs[who].Timer = w.Dinahs[who].Count
			w.Dinahs[who].Dest.Top = w.Dinahs[who].Position
			w.Dinahs[who].Dest.Bottom = w.Dinahs[who].Dest.Top + render.DartSrc[0].Tall()
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
		}
	} else {
		w.enemyWaiting(who)
	}
}

// ---------------------------------------------------------------------------
// HandleBall (Dynamics2.c:358-423)
// ---------------------------------------------------------------------------

// HandleBall bounces a ball on a fixed floor.
//
// Three things make it the odd one out, and all three are observable.
//
// **The collision test is outside the moving check**, at the very top of the function,
// so **a ball at rest still kills.** Every other mover only touches you while it is
// moving. A ball that has run down its bounces sits on the floor as a permanent hazard
// that looks like scenery -- which is exactly what the rooms that use one are built
// around.
//
// **It writes EvenFrame.** The idle arm's `evenFrame = true` (Dynamics2.c:420) is a
// write to the global frame-parity flag from inside an object handler, and nothing ever
// resynchronises it: a ball kicked into motion mid-frame permanently shifts the parity
// every other animation in the game reads, so the candles in that room flicker on the
// frames a room without a ball flickers its stars. See World.EvenFrame. The write also
// makes the ball's own first frame of gravity land immediately, which is presumably what
// it was for.
//
// **Active means "bounces forever".** At each bounce an active ball's velocity is reset
// to Count -- the launch speed -- so it never loses energy. An inactive one keeps three
// quarters of its speed, with C's truncation toward zero, so the sequence from -8 is
// 6, -4, 3, -2, 1, 0, and the ball stops after six bounces. `vVel == 0` is the stop
// test rather than `>= 0`, and it is reached exactly because 3/4 of 1 truncates to 0.
//
// One line of dead code is transcribed: `if (whole.bottom < dest.bottom)` after both
// have been set to Position can never be true. It is left in because it is the sort of
// line whose absence looks like an omission.
func (w *World) HandleBall(who int16) {
	w.checkGliders(who, false)

	if w.Dinahs[who].Moving { // bouncing
		w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
		if w.Dinahs[who].Dest.Bottom >= w.Dinahs[who].Position { // bounce
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
			w.Dinahs[who].Whole.Bottom = w.Dinahs[who].Position
			w.Dinahs[who].Dest.Bottom = w.Dinahs[who].Position
			w.Dinahs[who].Dest.Top = w.Dinahs[who].Dest.Bottom - 32
			if w.Dinahs[who].Active {
				w.Dinahs[who].VVel = w.Dinahs[who].Count
			} else {
				w.Dinahs[who].VVel = -((w.Dinahs[who].VVel * 3) / 4)
				if w.Dinahs[who].VVel == 0 {
					w.Dinahs[who].Moving = false // stop bounce
				}
			}
			if w.Dinahs[who].Whole.Bottom < w.Dinahs[who].Dest.Bottom {
				w.Dinahs[who].Whole.Bottom = w.Dinahs[who].Dest.Bottom
			}
			w.PlayPrioritySound(BounceSound, BouncePriority)
			if w.Dinahs[who].Moving {
				w.Dinahs[who].Frame = 1 // squashed, for one frame
			}
		} else {
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			if w.Dinahs[who].VVel > 0 {
				w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
			} else {
				w.Dinahs[who].Whole.Bottom -= w.Dinahs[who].VVel
			}
			if w.EvenFrame {
				w.Dinahs[who].VVel++
			}
			w.Dinahs[who].Frame = 0
		}
	} else {
		if w.Dinahs[who].Active {
			w.Dinahs[who].VVel = w.Dinahs[who].Count
			w.Dinahs[who].Moving = true
			w.EvenFrame = true // the global write; see above
		}
	}
}

// ---------------------------------------------------------------------------
// HandleDrip (Dynamics2.c:427-497)
// ---------------------------------------------------------------------------

// HandleDrip swells a drop on the ceiling, drops it, and hangs the next one up.
//
// HVel is where to hang it back up -- the ceiling Y the registration recorded -- and
// Position is the floor it falls to. Both are absolute room-local coordinates, so a
// drip's fall length is fixed by the author's rect and not by the room.
//
// **`frame = 9 - frame` is a two-cel flip**, not arithmetic on an index: 4 becomes 5 and
// 5 becomes 4, so the falling drop alternates between the two stretched cels on even
// frames. Frames 0..2 are the swelling drop, 3 is the hanging one -- which is also what
// the static room draw paints (objectdraw2.go's DrawDrip) -- and 4..5 are in flight.
//
// **The idle arm never touches Whole**, so from the frame after a splashdown until the
// next launch, Whole still holds the union of the last fall: the full column from ceiling
// to floor. RenderDrip is not gated on Moving, so it registers that stale column as a
// work rect on every single resting frame. Harmless -- the work map there is a clean copy
// of the background, so the extra work->screen copy changes no pixel -- but it is a large
// blit and two of the 47 rect slots, held for the whole reload period. Transcribed, and
// noted in docs/IMPROVEMENTS.md as a candidate for a later efficiency pass rather than
// something to fix inside a fidelity stage.
//
// The four-way idle switch is a chain of `==` tests on the timer, so a drip whose reload
// period is shorter than 6 frames skips the cels it steps over. There is no clamping.
func (w *World) HandleDrip(who int16) {
	if w.Dinahs[who].Moving {
		if w.EvenFrame {
			w.Dinahs[who].Frame = 9 - w.Dinahs[who].Frame
		}
		w.checkGliders(who, false)

		w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
		if w.Dinahs[who].Dest.Bottom >= w.Dinahs[who].Position {
			dest := render.Offset(w.Dinahs[who].Whole, w.R.V.OriginH, w.R.V.OriginV)
			w.AddRectToWorkRects(player.Rect(dest))
			w.Dinahs[who].Dest.Top = w.Dinahs[who].HVel
			w.Dinahs[who].Dest.Bottom = w.Dinahs[who].Dest.Top + 12
			w.PlayPrioritySound(DropSound, DropPriority)
			w.Dinahs[who].VVel = 0
			w.Dinahs[who].Timer = w.Dinahs[who].Count
			w.Dinahs[who].Frame = 3
			w.Dinahs[who].Moving = false
		} else {
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
			if w.EvenFrame {
				w.Dinahs[who].VVel++
			}
		}
	} else {
		if w.Dinahs[who].Active {
			w.Dinahs[who].Timer--

			switch {
			case w.Dinahs[who].Timer == 6:
				w.Dinahs[who].Frame = 0
			case w.Dinahs[who].Timer == 4:
				w.Dinahs[who].Frame = 1
			case w.Dinahs[who].Timer == 2:
				w.Dinahs[who].Frame = 2
			case w.Dinahs[who].Timer <= 0:
				w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, 3)
				w.Dinahs[who].Whole = w.Dinahs[who].Dest
				w.Dinahs[who].Moving = true
				w.Dinahs[who].Frame = 4
				w.PlayPrioritySound(DripSound, DripPriority)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// HandleFish (Dynamics2.c:501-588)
// ---------------------------------------------------------------------------

// HandleFish leaps a fish out of the water and lets it fall back in.
//
// HVel is the respawn delay in frames (Timer is refilled from it), Count is the launch
// velocity and Position is the waterline. The leap is the ball's arc with a different
// idle arm.
//
// **The animation only advances on the way down.** `vVel >= 0 && frame < 7` walks frames
// 4..7 as the fish falls, so the rise holds cel 4 -- one pose going up, four coming down.
// It stops at 7 rather than wrapping, so the last cel is held until splashdown.
//
// **Two sounds on splashdown**, DropSound then FishInSound, one after the other with
// nothing between them. Not a mistake to tidy: which one the player hears is
// PlayPrioritySound's decision, and 1.6 will make that call. Transcribed as two calls.
//
// **The idle bob runs outside the Active test.** `whole = dest` and the `timer & 3`
// animation are unconditional, and only the countdown is gated -- so a switched-off fish
// still animates, using a frozen timer. If that frozen timer happens to be 3 modulo 4,
// the bob condition is true on *every* frame and the fish twitches up and down forever,
// one pixel at a time, in place. That is reachable in the shipped game by switching a
// fish off at the wrong moment, and it is the closest thing in Dynamics2.c to a visible
// bug.
//
// The bob keeps its own union: cels 1 and 2 nudge the fish down and extend Whole's
// bottom, cels 3 and 0 nudge it back up and extend Whole's top, so a one-pixel move
// always has a two-pixel rect behind it.
func (w *World) HandleFish(who int16) {
	if w.Dinahs[who].Moving { // leaping
		if w.Dinahs[who].VVel >= 0 && w.Dinahs[who].Frame < 7 {
			w.Dinahs[who].Frame++
		}
		w.checkGliders(who, false)

		w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
		if w.Dinahs[who].Dest.Bottom >= w.Dinahs[who].Position { // splash down
			dest := render.Offset(w.Dinahs[who].Whole, w.R.V.OriginH, w.R.V.OriginV)
			w.AddRectToWorkRects(player.Rect(dest))
			w.Dinahs[who].Dest.Bottom = w.Dinahs[who].Position
			w.Dinahs[who].Dest.Top = w.Dinahs[who].Dest.Bottom - 16
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			w.Dinahs[who].Whole.Top -= 2
			w.PlayPrioritySound(DropSound, DropPriority)
			w.Dinahs[who].VVel = w.Dinahs[who].Count
			w.Dinahs[who].Timer = w.Dinahs[who].HVel
			w.Dinahs[who].Frame = 0
			w.Dinahs[who].Moving = false
			w.PlayPrioritySound(FishInSound, FishInPriority)
		} else {
			w.Dinahs[who].Whole = w.Dinahs[who].Dest
			if w.Dinahs[who].VVel > 0 {
				w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
			} else {
				w.Dinahs[who].Whole.Bottom -= w.Dinahs[who].VVel
			}
			if w.EvenFrame {
				w.Dinahs[who].VVel++
			}
		}
	} else { // idle, bobbing in the water
		w.Dinahs[who].Whole = w.Dinahs[who].Dest
		if w.Dinahs[who].Timer&0x0003 == 0x0003 {
			w.Dinahs[who].Frame++
			if w.Dinahs[who].Frame > 3 {
				w.Dinahs[who].Frame = 0
			}
			if w.Dinahs[who].Frame == 1 || w.Dinahs[who].Frame == 2 {
				w.Dinahs[who].Dest.Top++
				w.Dinahs[who].Dest.Bottom++
				w.Dinahs[who].Whole.Bottom++
			} else {
				w.Dinahs[who].Dest.Top--
				w.Dinahs[who].Dest.Bottom--
				w.Dinahs[who].Whole.Top--
			}
		}
		if w.Dinahs[who].Active {
			w.Dinahs[who].Timer--
			if w.Dinahs[who].Timer <= 0 { // fish leaps
				w.Dinahs[who].Whole = w.Dinahs[who].Dest
				w.Dinahs[who].Moving = true
				w.Dinahs[who].Frame = 4
				w.PlayPrioritySound(FishOutSound, FishOutPriority)
			}
		}
	}
}
