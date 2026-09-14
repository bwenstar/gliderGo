// Package player is the glider: its state machine, its integrator, and the
// twenty-four modes it can be in.
//
// Everything here is 16-bit integer arithmetic in pixels per frame, because that
// is what the original is. There is no fixed point, no sub-pixel accumulator and
// no delta time anywhere in Glider PRO's player code: hVel, vVel and their
// desired counterparts are all `short` and are added directly to rectangle
// edges. The simulation is frame-locked at kTicksPerFrame = 2 Mac ticks, i.e.
// 30.07 fps, and every constant in the game was tuned against that. Introducing
// floats or a variable timestep here would change the feel of the game even if
// every constant were kept, so the port keeps the integers and leaves any
// interpolation to the renderer.
//
// The authority for this package is docs/analysis/player-physics.md, which was
// reverse-documented from GliderPRO/Sources/Player.c, Modes.c, Input.c and
// Interactions.c. Ported functions carry the original's name and its file:line.
package player

// Rect is the QuickDraw rectangle, in the Toolbox's field order.
//
// Position lives entirely in these. The original has no x/y pair for the player
// at all: movement mutates dest.left/right/top/bottom in place, and the sprite,
// the hit box and the dirty rect are all read off the same four numbers. A port
// that introduced a separate position and derived the rect from it would have to
// reproduce the rounding of every place the original mutates one edge alone
// (Player.c:342, :375, :411, :469, :503, :540, :746, :843, :873, :911, :944,
// :1042, :1133, :1308), so the rect stays primary.
type Rect struct {
	Top, Left, Bottom, Right int16
}

// Glider is gliderType (GliderPRO/Headers/GliderStructs.h:200-216).
//
// Field order and grouping follow the C so the two can be read side by side. The
// C struct is 112 bytes under natural alignment and 110 under pack(2), with
// identical member offsets either way; nothing serialises it, so the layout is
// documentation rather than a constraint (player-physics.md §2.1, §16.4).
//
// Three fields are overloaded in the original and are kept overloaded here,
// because the aliasing is observable:
//
//   - Frame is a fade index, a sprite index, an animation phase, a "was burning"
//     sentinel, a y coordinate, and a negative countdown, depending on Mode.
//   - WasMode is a burn fuse, a spider-web counter, and a saved mode.
//   - HVel is the idle countdown while Mode is GliderIdle.
//
// Splitting them into separate honest fields would be tidier and would silently
// drop the cases where one mode reads what another mode wrote. See §5.2 and §5.3.
type Glider struct {
	// Src is the source rect in the 48x668 glider atlas. Mask is the rect in
	// the 1-bit mask atlas and is assigned the same value as Src at every one
	// of the ~40 assignments in Player.c and Modes.c; they are separate fields
	// only because CopyMask takes independent source and mask rects. Both are
	// kept because several handlers trim one edge of each in lockstep, and a
	// single collapsed rect would hide a place where only one was trimmed.
	Src, Mask Rect

	// Dest is the glider's position in room-local pixels. Whole is the swept
	// union of the pre-move and post-move Dest, which the dirty-rect renderer
	// uses to restore background. Whole is not computed as a min/max union:
	// MoveGlider writes the trailing edge before the move and the leading edge
	// after, which equals the union only because Dest never changes size during
	// a move. See §7.2 consequence 4.
	Dest, Whole Rect

	// DestShadow is the ground shadow, 48x9, and WholeShadow its swept union.
	// The shadow tracks the glider horizontally only: MoveGlider never touches
	// DestShadow.Top or .Bottom. It is a fixed-height blob sliding along the
	// floor, with no scaling or fading with altitude.
	DestShadow, WholeShadow Rect

	// Clip is a mode-specific clipping or target rect (the mail slot, the duct).
	// EnteredRect is the respawn position: where Dest was when the player last
	// became normal in this room.
	Clip, EnteredRect Rect

	// RightClip and LeftClip are the x the glider is trimmed against while it
	// walks behind a staircase. In the original these are two file-scope globals
	// in Player.c (:52) shared by both players; here they are per-glider.
	//
	// That is a deviation, and it is behaviour-preserving in the original's own
	// configuration: Glider PRO's two players are always in the same room, so both
	// would read the same value, and each is written by the mode-entry function
	// immediately before the handler that reads it. Stage 3's separate-worlds race
	// puts the two players in different rooms, where a shared global would be
	// wrong, so the state is stored where it belongs instead of being fixed later.
	RightClip, LeftClip int16

	// Transit is where the transit object this glider just entered leads. The
	// original keeps this in three more Player.c globals -- transRoom, transRect
	// and linkedToWhat -- written by the Start* functions and read afterwards by
	// the room-transition code. Per-glider for the same reason as the clips.
	Transit Link

	// The four raw key-map bit indices this glider answers to. Player 2's are
	// hard-coded modifier keys in the original, which a modern OS may swallow;
	// keeping them as data rather than constants is what will let 1.7 remap them.
	LeftKey, RightKey, BattKey, BandKey int32

	// HVel and VVel are the current velocity in px/frame, positive being right
	// and down. HVel is clamped to +/-kMaxHVel by the integrator; VVel is not
	// clamped at all, and its terminal value is emergent (see MoveGlider).
	HVel, VVel int16

	// WasHVel and WasVVel are the velocities as of the last completed move.
	// GliderHitTop reads WasHVel to un-sweep the hit box (Interactions.c:65).
	// WasVVel is written by MoveGlider and read nowhere in the original tree; it
	// is kept so that the transcription of MoveGlider stays statement for
	// statement, and so that a later reader does not have to wonder whether it
	// was dropped by accident. See §22 open question 5.
	WasHVel, WasVVel int16

	// The per-frame velocity targets. The integrator resets these itself --
	// HDesiredVel to 0 and VDesiredVel to Gravity -- so every accelerating
	// influence (keys, fans, vents, helium) must be re-applied every frame and
	// nothing persists. This is the single most important structural fact about
	// the original's physics.
	VDesiredVel, HDesiredVel int16

	// Mode is the state-machine state; Frame and WasMode are the overloaded
	// counters described in the type comment.
	Mode, Frame, WasMode int16

	// Facing is FaceRight (true) or FaceLeft (false), and changes only through
	// the three-frame about-face tumble in modes GliderFaceLeft/FaceRight.
	// MoveGliderNormal reads it and never writes it.
	Facing bool

	// Tipped means the player is holding the direction key opposite to Facing:
	// the bank. Recomputed by GetInput every frame.
	Tipped bool

	// Sliding is set by the kSlideIt hot spot -- spilt grease -- which also snaps
	// VVel so the glider is seated on the spill (Interactions.c:1376-1378).
	//
	// It looks like a fourth Ignore flag and is not one, in two ways. The Ignores
	// are cleared unconditionally at the end of HandleGlider for every mode;
	// Sliding is cleared inside MoveGliderNormal (Player.c:155-159, :177-181),
	// which HandleGlider reaches only in mode Normal (Player.c:1339). So it is a
	// one-frame flag only while the glider is walking. A glider that slips and
	// then turns around or catches fire carries it for the whole of that mode.
	//
	// And it is read outside the mover: CheckRoofCollision skips its entire
	// tile-1 fall-through test while it is set (Interactions.c:455), which is
	// what makes grease on a roof a slide rather than a hole. That read happens
	// in the same frame as the set -- HandleInteraction is CheckForHotSpots then
	// CheckGliderInRoom (Interactions.c:1691-1710), both before HandleGlider --
	// and CheckGliderInRoom runs in Normal, FaceLeft, FaceRight and Burning
	// (:690-694). Only the first of those four clears the flag, so a glider that
	// slips on a roof and then tumbles about-face is immune to the roof for those
	// three frames too. See escape.go's CheckRoofCollision.
	Sliding bool

	// IgnoreLeft, IgnoreRight and IgnoreGround are one-frame flags set by the
	// interaction pass -- "you are standing in a doorway", "there is a manhole
	// under you" -- and cleared unconditionally at the end of HandleGlider for
	// every mode. They are the whole channel by which the world tells the mover
	// that a boundary is passable this frame.
	IgnoreLeft, IgnoreRight, IgnoreGround bool

	// FireHeld debounces the band key so that holding it fires once.
	FireHeld bool

	// Which is Player1 (true) or Player2 (false).
	Which bool

	// HeldLeft and HeldRight record that a direction key was down this frame;
	// they block the down-stairs and up-stairs transitions respectively. GetInput
	// recomputes them every frame except while burning, when they keep their old
	// values.
	HeldLeft, HeldRight bool

	// DontDraw suppresses rendering. It is set by vanish() on the five transit
	// exits -- transporter, both ducts, both mail slots -- and by the death paths,
	// and cleared by every Flag*/Start* entry function, by UndoGliderLimbo and by
	// HandleIdleGlider.
	//
	// It is *not* implied by GliderInLimbo. FlagGliderInLimbo never touches the
	// field (Modes.c:458-468), so a glider that entered limbo from a wall or a
	// staircase -- raceForExit in escape.go, escapeOrWait in handle.go -- is frozen
	// but still drawn, sitting half out of the room for the whole wait. Those paths
	// hide the glider, when they hide it at all, by collapsing Dest/Src/Mask
	// instead. The only writer outside this package is Play.c:190, at NewGame.
	DontDraw bool
}

// MoveGlider is Player.c:64-147, the integrator. It ramps the velocities toward
// their desired values, resets those desired values, clamps horizontal speed,
// snapshots WasHVel/WasVVel, moves Dest and DestShadow, and computes the swept
// Whole/WholeShadow -- all of it, in that order, in one function.
//
// It is called only by the six modes in which the player has momentum: normal,
// burning, the two about-faces, and the two foil dissolves. Every other mode
// moves Dest by a hard-coded speed and never comes here.
//
// Five details are load-bearing and are transcribed rather than tidied:
//
//  1. The desired velocities are reset here, at the end of the frame's use of
//     them, not at the start of the next frame. HDesiredVel becomes 0 and
//     VDesiredVel becomes Gravity, which is why free fall settles at exactly
//     +3 px/frame and why releasing a key decays HVel 5, 3, 1, 0.
//
//  2. There is no vertical clamp. HVel is clamped to +/-MaxHVel inside each sign
//     branch; VVel is not clamped anywhere, so however large a VVel survives the
//     ramp is applied to Dest in full.
//
//     But nothing in the world gets a vertical kick applied whole, and it is easy
//     to think otherwise. A ceiling vent does not write VVel at all: kDropIt
//     assigns VDesiredVel (Interactions.c:1206-1207), so VVel ramps 3, 5, 7, 8,
//     holds at 8 while the hot spot keeps overlapping, and decays 8, 6, 4, 3 only
//     once the glider is clear. And the writes that do target VVel directly -- a
//     kSlideIt grease snap (Interactions.c:1378), the ceiling stops and floor
//     crashes in escape.go -- are still attenuated, because HandleInteraction runs
//     before HandleGlider in the same frame (Play.c:482 then :487) and the ramp
//     above eats up to VImpulse of them before the move below. A grease snap of
//     -12 displaces -10; a written +/-2 is snapped straight to VDesiredVel.
//
//     Adding a symmetric vertical clamp "for safety" would still be wrong -- it
//     would cap the vent's steady state and the escape.go writes -- but the reason
//     is the missing clamp itself, not a full-magnitude kick.
//
//  3. Both move blocks are `if vel < 0 { ... } else { ... }`, not `else if
//     vel > 0`. A zero-velocity axis therefore takes the positive branch, so
//     WasHVel and WasVVel are refreshed to 0 and all four edges of Whole are
//     rewritten on every single call. A stationary glider un-sweeps its hit box
//     by 0, not by its last non-zero velocity.
//
//  4. The clamp is inside the sign branches, and the branch is chosen on the sign
//     of the velocity rather than on whether the ramp moved. So a velocity pushed
//     past the limit by something that bypasses the ramp -- the battery adds
//     +/-8 directly to HVel -- is still trimmed to +/-16 here.
//
//  5. Whole is built one edge at a time, trailing before the move and leading
//     after, and collapses to exactly Dest when both velocities are 0. It does
//     not accumulate across frames.
func (g *Glider) MoveGlider() {
	// ---- horizontal ramp (Player.c:66-78). ">" is tested first. ----
	if g.HVel > g.HDesiredVel {
		g.HVel -= HImpulse
		if g.HVel < g.HDesiredVel {
			g.HVel = g.HDesiredVel // never overshoot
		}
	} else if g.HVel < g.HDesiredVel {
		g.HVel += HImpulse
		if g.HVel > g.HDesiredVel {
			g.HVel = g.HDesiredVel
		}
	}
	g.HDesiredVel = 0 // Player.c:78 -- reset, every frame

	// ---- vertical ramp (Player.c:80-92) ----
	if g.VVel > g.VDesiredVel {
		g.VVel -= VImpulse
		if g.VVel < g.VDesiredVel {
			g.VVel = g.VDesiredVel
		}
	} else if g.VVel < g.VDesiredVel {
		g.VVel += VImpulse
		if g.VVel > g.VDesiredVel {
			g.VVel = g.VDesiredVel
		}
	}
	g.VDesiredVel = Gravity // Player.c:92 -- reset to +3, every frame

	// ---- horizontal move and sweep (Player.c:94-127) ----
	if g.HVel < 0 { // moving left
		if g.HVel < -MaxHVel {
			g.HVel = -MaxHVel
		}
		g.WasHVel = g.HVel

		g.Whole.Right = g.Dest.Right // trailing edge, before the move
		g.Dest.Left += g.HVel
		g.Dest.Right += g.HVel
		g.Whole.Left = g.Dest.Left // leading edge, after

		g.WholeShadow.Right = g.DestShadow.Right
		g.DestShadow.Left += g.HVel
		g.DestShadow.Right += g.HVel
		g.WholeShadow.Left = g.DestShadow.Left
	} else { // moving right, or not at all: HVel == 0 lands here
		if g.HVel > MaxHVel {
			g.HVel = MaxHVel
		}
		g.WasHVel = g.HVel // Player.c:116 -- runs even for 0

		g.Whole.Left = g.Dest.Left
		g.Dest.Left += g.HVel
		g.Dest.Right += g.HVel
		g.Whole.Right = g.Dest.Right

		g.WholeShadow.Left = g.DestShadow.Left
		g.DestShadow.Left += g.HVel
		g.DestShadow.Right += g.HVel
		g.WholeShadow.Right = g.DestShadow.Right
	}

	// ---- vertical move and sweep (Player.c:129-146) ----
	// DestShadow is never moved vertically. Its Top is set once by InitGlider
	// and thereafter only re-derived as Top+ShadowHigh by the mode-entry
	// functions, which never rewrite Top itself.
	if g.VVel < 0 { // moving up
		g.WasVVel = g.VVel

		g.Whole.Bottom = g.Dest.Bottom
		g.Dest.Top += g.VVel
		g.Dest.Bottom += g.VVel
		g.Whole.Top = g.Dest.Top
	} else { // moving down, or not at all: VVel == 0 lands here
		g.WasVVel = g.VVel // Player.c:140 -- runs even for 0

		g.Whole.Top = g.Dest.Top
		g.Dest.Top += g.VVel
		g.Dest.Bottom += g.VVel
		g.Whole.Bottom = g.Dest.Bottom
	}
}
