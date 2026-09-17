package game

// The nine appliance handlers: Dynamics.c:293-775.
//
// HandleToast is here rather than with the movers because it lives in Dynamics.c and
// shares the toaster's slot fields; RenderToast is with the other six renderers.
//
// ---------------------------------------------------------------------------
// The one thing to know: appliances edit the background
// ---------------------------------------------------------------------------
//
// Six of the nine -- Mac Plus, TV, coffee maker, VCR, stereo, microwave -- are the same
// function with different art, different sounds and different timer arithmetic:
//
//	if (timer > 0) {
//	    timer--;
//	    if (active) {
//	        if      (timer == 0) AddRectToWorkRects(dest);                  // reveal
//	        else if (timer == 1) { sound; blit onArt -> backSrcMap;
//	                               AddRectToBackRects(dest); }             // stage
//	        ...type-specific extra arms...
//	    } else {
//	        the same two arms with offArt
//	    }
//	}
//
// **They blit into backSrcMap, not workSrcMap.** That is the whole design. The back map
// is the room's clean background, so switching a TV on *permanently repaints the
// background*, and the appliance then needs no renderer and no per-frame work -- which is
// why only seven of the seventeen types appear in RenderDynamics.
//
// It also explains the two-arm timer, read against the dirty-rect semantics at
// render_frame.go's CopyRectsQD (work->screen first, then back->work):
//
//	frame  timer  what happens                      net effect
//	N      1      new art into back; back-rect      end of frame back->work, so work holds it
//	N+1    0      work-rect                         end of frame work->screen, so it is seen
//
// **A state change takes exactly two frames and costs two rect entries, once.** That is
// why ToggleCoffee, ToggleVCR, ToggleMicrowave and ToggleTV set timer = 4 and not 2: two
// frames of lead-in, then stage, then reveal. And it is why the whole body is `timer > 0`
// gated rather than run every frame -- a settled appliance costs nothing at all.
//
// The exception is the outlet, which draws into the *work* map every frame of a zap, hurts
// you, and can kill. It is the only one with a renderer's shape and no renderer.

import (
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/render"
)

// The appliance sounds (GliderDefines.h:79-91, :126-147). Declared here rather than in a
// central table, following the convention interactions.go and telephone.go set: a sound
// constant lives beside its one or two call sites so that grepping the name finds the
// behaviour and not a list.
//
// CoffeeSound and CoffeePriority are not here -- triggers.go already declares them for
// FireTrigger's kCoffee arm, and HandleCoffee is the second of the two call sites.
const (
	MacOnSound   int16 = 29 // the boot chime, shared by five appliances switching on
	MacBeepSound int16 = 30 // and the Mac Plus's own beep ten frames later
	MacOffSound  int16 = 31 // shared by every appliance switching off
	TVOnSound    int16 = 32
	TVOffSound   int16 = 33
	MysticSound  int16 = 35 // the sparkle emitter's chime
	ZapSound     int16 = 36
	VCRSound     int16 = 24

	ToastLaunchSound int16 = 27
	ToastLandSound   int16 = 28
)

const (
	MysticPriority      int16 = 202
	VCRPriority         int16 = 303
	ToastLaunchPriority int16 = 304
	ToastLandPriority   int16 = 305
	MacOnPriority       int16 = 401
	MacOffPriority      int16 = 402
	MacBeepPriority     int16 = 403
	TVOnPriority        int16 = 404
	TVOffPriority       int16 = 405
	ZapPriority         int16 = 406
)

// applianceToBack is the CopyBits every one of the six state changes performs:
// applianceSrcMap -> backSrcMap, srcCopy, opaque.
//
// Opaque and not masked, and that is right for all seven overlays -- a screen, a clock
// face, an LED and a microwave door are meant to cover what is under them. It is the same
// blit internal/render's opaqueSheet does for the *static* draw of the same art
// (objectdraw2.go), which is what makes the two agree: the composition draws the off
// state, and the handler paints the on state over it.
func (w *World) applianceToBack(src, dst Rect) {
	if art := w.R.A.Sheet("appliance"); art != nil {
		w.R.Back.Copy(art, src, dst, render.SrcCopy)
	}
}

// applianceToWork is the outlet's blit, and the outlet's alone: the spark is transient,
// so it goes into the work map to be erased next frame rather than into the background.
func (w *World) applianceToWork(src, dst Rect) {
	if art := w.R.A.Sheet("appliance"); art != nil {
		w.R.Work.Copy(art, src, dst, render.SrcCopy)
	}
}

// ---------------------------------------------------------------------------
// HandleSparkleObject (Dynamics.c:293-317)
// ---------------------------------------------------------------------------

// HandleSparkleObject is a sparkle *emitter* and draws nothing at all.
//
// It has no renderer -- the seven in RenderDynamics are toast, balloon, copter, dart,
// ball, drip and fish -- and all it does is push an entry into the transient sparkles
// free list every 60..299 frames. Everything visible about a kSparkle object belongs to
// RenderSparkles.
//
// **Frame is not an animation index here; it is a five-frame lockout** so the emitter
// cannot queue a second puff while the first is still on screen. It is NumSparkleModes
// because that is exactly how long RenderSparkles takes to run one puff down, which makes
// the two systems agree by construction rather than by a matching literal.
//
// The C's empty `else` is transcribed as an empty branch, because what it does is
// observable: a switched-off emitter freezes mid-lockout -- Frame stops decrementing --
// so switching it back on *resumes* the countdown rather than restarting it.
//
// Two RNG draws per emitter: RandomInt(60)+15 at registration and RandomInt(240)+60 here.
// The second one draws forever, so a room's sparkle count shifts the whole downstream
// stream; any determinism story (1.8's replays, Stage 3's networked race) has to account
// for that.
func (w *World) HandleSparkleObject(who int16) {
	if w.Dinahs[who].Active {
		if w.Dinahs[who].Frame <= 0 { // idle
			w.Dinahs[who].Timer--
			if w.Dinahs[who].Timer <= 0 {
				w.Dinahs[who].Timer = w.RandomInt(240) + 60
				w.Dinahs[who].Frame = NumSparkleModes
				tempRect := w.Dinahs[who].Dest
				w.AddSparkle(tempRect)
				w.PlayPrioritySound(MysticSound, MysticPriority)
			}
		} else { // sparkling
			w.Dinahs[who].Frame--
		}
	} else {
		// Deliberately empty, as the C's is. See the note above: the lockout freezes
		// rather than resetting.
	}
}

// ---------------------------------------------------------------------------
// HandleToast (Dynamics.c:321-384)
// ---------------------------------------------------------------------------

// HandleToast is two states in one function: airborne bread, and a toaster waiting to
// fire.
//
// **Frame means two different things, switched by Moving.** Airborne it is an index into
// the six-frame bread strip; idle it is the number of frames until the next launch. Timer
// is the reload *period* it is refilled from. One field, two meanings -- the single most
// confusing thing in Dynamics.c, and the reason the landing writes `Frame = Timer`.
//
// **The landing test is `VVel > Count`, not a position test.** Count is the launch speed
// the registration solved for, stored positive; the bread leaves at -Count and gains 1 per
// frame, so it lands after exactly 2*Count+1 frames wherever it happens to be. The toast
// never looks at the floor. Fire a toaster next to a wall and the bread flies through it
// and lands in mid-air, at the height it started, on schedule.
//
// The Whole union is built by pushing the *trailing* edge back one frame's travel -- the
// mirror image of RenderFlyingPoints, which pushes the leading edge forward. Note both
// arms subtract VVel, so the sign does the work and the two lines differ only in which
// edge moves. It is recomputed from Dest every frame, so it cannot drift.
func (w *World) HandleToast(who int16) {
	if w.Dinahs[who].Moving {
		if w.EvenFrame {
			w.Dinahs[who].Frame++
			if w.Dinahs[who].Frame >= NumBreadPicts {
				w.Dinahs[who].Frame = 0
			}
		}
		w.checkGliders(who, false)

		w.Dinahs[who].Dest = render.Offset(w.Dinahs[who].Dest, 0, w.Dinahs[who].VVel)
		w.Dinahs[who].Whole = w.Dinahs[who].Dest
		if w.Dinahs[who].VVel > 0 {
			w.Dinahs[who].Whole.Top -= w.Dinahs[who].VVel
		} else {
			w.Dinahs[who].Whole.Bottom -= w.Dinahs[who].VVel
		}
		w.Dinahs[who].VVel++ // falls
		if w.Dinahs[who].VVel > w.Dinahs[who].Count {
			dest := render.Offset(w.Dinahs[who].Whole, w.R.V.OriginH, w.R.V.OriginV)
			w.AddRectToWorkRects(player.Rect(dest))
			w.Dinahs[who].Moving = false
			w.Dinahs[who].Frame = w.Dinahs[who].Timer
			w.PlayPrioritySound(ToastLandSound, ToastLandPriority)
		}
	} else {
		if w.Dinahs[who].Active {
			w.Dinahs[who].Frame--
		}
		// **Outside the Active test**, so an inactive toaster whose countdown had
		// already expired rewrites Frame = Timer every single frame. Harmless -- the
		// same value -- but it means a switched-off toaster is not quite idle, and it
		// is why a toaster switched back on always waits a full period rather than
		// firing at once.
		if w.Dinahs[who].Frame <= 0 {
			if w.Dinahs[who].Active {
				w.Dinahs[who].VVel = -w.Dinahs[who].Count
				w.Dinahs[who].Frame = 0
				w.Dinahs[who].Moving = true
				w.PlayPrioritySound(ToastLaunchSound, ToastLaunchPriority)
			} else {
				w.Dinahs[who].Frame = w.Dinahs[who].Timer
			}
		}
	}
}

// ---------------------------------------------------------------------------
// HandleMacPlus (Dynamics.c:388-424)
// ---------------------------------------------------------------------------

// HandleMacPlus is the plain skeleton plus one extra arm: MacOnSound at Timer == 30.
//
// With ToggleMacPlus' asymmetric 40-on/10-off the on-sequence is *chime at 30, stage at 1,
// reveal at 0* -- a ten-frame gap between the chime and the screen lighting up, which is a
// Mac Plus booting. The off-sequence never reaches 30, which is why 10 is enough for it.
func (w *World) HandleMacPlus(who int16) {
	if w.Dinahs[who].Timer <= 0 {
		return
	}
	w.Dinahs[who].Timer--
	if w.Dinahs[who].Active {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(MacBeepSound, MacBeepPriority)
			w.applianceToBack(render.PlusScreen2, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		case 30:
			w.PlayPrioritySound(MacOnSound, MacOnPriority)
		}
	} else {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(MacOffSound, MacOffPriority)
			w.applianceToBack(render.PlusScreen1, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	}
}

// ---------------------------------------------------------------------------
// HandleTV (Dynamics.c:428-478)
// ---------------------------------------------------------------------------

// HandleTV is the plain skeleton with the four-condition QuickTime test wrapped around
// *both* arms of the active branch, each as an empty `if` with the real work in the
// `else`:
//
//	if ((thisMac.hasQT) && (hasMovie) && (tvInRoom) && (who == tvWithMovieNumber)) { }
//	else { ...blit tvScreen2 / AddRectToWorkRects... }
//
// So on a machine with QuickTime *and* a house shipping a movie, switching the TV on does
// nothing -- no blit, no reveal -- because the movie is expected to draw over that rect
// instead. The off branch has no such test and always blits tvScreen1, which is how the
// movie gets painted over when the set is switched off.
//
// **The port has no movie support, so it always takes the else branch, which is the
// behaviour of a 1994 Mac without QuickTime.** That is a deliberate, bounded fidelity
// target rather than an oversight: Room.TVMovieNumber stays at the -1 Rebuild resets it to
// (see internal/render/locale.go's kTV case), so the identity test is unreachable even in
// principle. docs/IMPROVEMENTS.md 2.37 has the scope -- 15 of the 22 shipped houses did ship
// a movie, and all 15 are already extracted under assets/extracted/movie -- and names the
// four no-op sites 1.5e has to fill together.
func (w *World) HandleTV(who int16) {
	if w.Dinahs[who].Timer <= 0 {
		return
	}
	w.Dinahs[who].Timer--
	if w.Dinahs[who].Active {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(TVOnSound, TVOnPriority)
			w.applianceToBack(render.TVScreen2, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	} else {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(TVOffSound, TVOffPriority)
			w.applianceToBack(render.TVScreen1, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	}
}

// ---------------------------------------------------------------------------
// HandleCoffee (Dynamics.c:482-524)
// ---------------------------------------------------------------------------

// HandleCoffee is the skeleton plus two self-restarts, and the result is an appliance that
// never settles.
//
// From ToggleCoffee's 4 it stages at 1, reveals at 0, jumps to 200..399, counts down to
// 100, gurgles, and jumps again. **The gurgle period is Timer-100 = 100..299 frames**,
// 3.3 to 10 seconds, and Timer == 1 and Timer == 0 are unreachable for as long as it stays
// on.
//
// Switching it off leaves Timer somewhere in 101..399, and the inactive branch waits for it
// to walk all the way down to 1 before playing MacOffSound -- so **a coffee maker could
// take up to thirteen seconds to visibly switch off.** ToggleCoffee forcing Timer = 4 is
// the only thing that prevents it; the delay is reachable only if something clears Active
// without touching Timer, which nothing in the game does.
//
// A second per-frame RNG consumer, drawing once per gurgle forever. Same determinism note
// as HandleSparkleObject.
func (w *World) HandleCoffee(who int16) {
	if w.Dinahs[who].Timer <= 0 {
		return
	}
	w.Dinahs[who].Timer--
	if w.Dinahs[who].Active {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
			w.Dinahs[who].Timer = 200 + w.RandomInt(200)
		case 1:
			w.PlayPrioritySound(MacOnSound, MacOnPriority)
			w.applianceToBack(render.CoffeeLight2, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		case 100:
			w.PlayPrioritySound(CoffeeSound, CoffeePriority)
			w.Dinahs[who].Timer = 200 + w.RandomInt(200)
		}
	} else {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(MacOffSound, MacOffPriority)
			w.applianceToBack(render.CoffeeLight1, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	}
}

// ---------------------------------------------------------------------------
// HandleOutlet (Dynamics.c:528-599)
// ---------------------------------------------------------------------------

// HandleOutlet is the only appliance that hurts you, the only one that draws into the work
// map, and the only one with a real bug in its fan-out.
//
// The zap runs LengthOfZap = 30 frames cycling Frame 1,2,3 -- frame 0 is the idle socket
// and the wrap is to **1**, not 0, so the idle art never appears mid-zap -- with ZapSound
// at Timer 25, 20, 15, 10 and 5, five repeats on top of the one TriggerOutlet or
// ToggleOutlet already played.
//
// **The draw test reads the post-reset Position.** On the final frame the block above has
// already set Position = 0, so the test falls through to `HVel > 0`, and HVel is the room's
// light count (UpdateOutletsLighting). In a lit room the outlet redraws OutletSrc[0], its
// idle socket; **in a dark room it is painted out instead.** That is what the numLights
// plumbing is for, and it is one HVel, one reader, one frame per zap.
//
// The PaintRect that does the painting-out has **no destination in the shipped Carbon
// source**: the `SetPort((GrafPtr)workSrcMap)` on the line above is commented out with
// nothing put in its place, so the fill lands in whichever GWorld the previous drawing
// call left current. The port fills the work map, which is what the commented line names
// and what the correctly-converted Grease.c does. docs/IMPROVEMENTS.md 2.34.
//
// The idle arm has the toaster's outside-the-guard shape and the same harmless repeated
// write; Position is the launch/idle enum where the toaster uses Moving, and the inactive
// arm resets Timer from Count where the toaster resets Frame from Timer.
func (w *World) HandleOutlet(who int16) {
	if w.Dinahs[who].Position != 0 {
		w.Dinahs[who].Timer--

		// **The fan-out is written out longhand here, and the booleans are
		// deliberately inconsistent.** Dynamics.c:539 and :541 pass false; :545, :546
		// and :550 pass true. The outlet is an appliance, so its Dest is in screen
		// coordinates -- which means the OneLeft arm compares a screen rect against a
		// room-local glider and the zap misses by exactly the scroll offset. Zero in a
		// room at the house origin, non-zero everywhere else.
		//
		// This is a real bug in the original and it is transcribed, not fixed. Calling
		// checkGliders would fix it; giving checkGliders two booleans would hide it.
		if w.TwoPlayer {
			if w.OneLeft {
				if w.DeadWhich == w.P1.Which {
					w.CheckDynamicCollision(who, &w.P2, false)
				} else {
					w.CheckDynamicCollision(who, &w.P1, false)
				}
			} else {
				w.CheckDynamicCollision(who, &w.P1, true)
				w.CheckDynamicCollision(who, &w.P2, true)
			}
		} else {
			w.CheckDynamicCollision(who, &w.P1, true)
		}

		if w.Dinahs[who].Timer <= 0 {
			w.Dinahs[who].Frame = 0
			w.Dinahs[who].Position = 0
			w.Dinahs[who].Timer = w.Dinahs[who].Count
		} else {
			if w.Dinahs[who].Timer%5 == 0 {
				w.PlayPrioritySound(ZapSound, ZapPriority)
			}
			w.Dinahs[who].Frame++
			if w.Dinahs[who].Frame >= NumOutletPicts {
				w.Dinahs[who].Frame = 1
			}
		}

		if w.Dinahs[who].Position != 0 || w.Dinahs[who].HVel > 0 {
			w.applianceToWork(render.OutletSrc[w.Dinahs[who].Frame], w.Dinahs[who].Dest)
		} else {
			w.R.Work.Fill(w.Dinahs[who].Dest, render.Black8)
		}
		w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
	} else {
		if w.Dinahs[who].Active {
			w.Dinahs[who].Timer--
		}
		if w.Dinahs[who].Timer <= 0 {
			if w.Dinahs[who].Active {
				w.Dinahs[who].Position = 1
				w.Dinahs[who].Timer = LengthOfZap
				w.PlayPrioritySound(ZapSound, ZapPriority)
			} else {
				w.Dinahs[who].Timer = w.Dinahs[who].Count
			}
		}
	}
}

// ---------------------------------------------------------------------------
// HandleVCR (Dynamics.c:603-667)
// ---------------------------------------------------------------------------

// HandleVCR is the skeleton plus a self-restarting blink:
//
//	timer  active branch
//	101    stage vcrTime2 if Frame == 0 else vcrTime1
//	100    reveal; timer = 115; Frame = 1 - Frame
//	5      MacOnSound
//	1      VCRSound; stage vcrTime2
//	0      reveal; timer = 115
//
// From ToggleVCR's 4 it stages at 1 and reveals at 0, then jumps to 115 -- and from then
// on it bounces between 115 and 100, **fifteen frames per half-blink**, half a second,
// forever. Timer == 5 and Timer == 1 are unreachable after the first cycle, exactly like
// the coffee maker's. It is a VCR flashing 12:00, and it is the most 1994 thing in the
// game.
//
// **Frame here is a two-state toggle, not a strip index** -- `1 - Frame` rather than
// `Frame++`. Third distinct meaning of Frame in this file.
func (w *World) HandleVCR(who int16) {
	if w.Dinahs[who].Timer <= 0 {
		return
	}
	w.Dinahs[who].Timer--
	if w.Dinahs[who].Active {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
			w.Dinahs[who].Timer = 115
		case 5:
			w.PlayPrioritySound(MacOnSound, MacOnPriority)
		case 1:
			w.PlayPrioritySound(VCRSound, VCRPriority)
			w.applianceToBack(render.VCRTime2, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		case 100:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
			w.Dinahs[who].Timer = 115
			w.Dinahs[who].Frame = 1 - w.Dinahs[who].Frame
		case 101:
			if w.Dinahs[who].Frame == 0 {
				w.applianceToBack(render.VCRTime2, w.Dinahs[who].Dest)
			} else {
				w.applianceToBack(render.VCRTime1, w.Dinahs[who].Dest)
			}
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	} else {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(MacOffSound, MacOffPriority)
			w.applianceToBack(render.VCRTime1, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	}
}

// ---------------------------------------------------------------------------
// HandleStereo (Dynamics.c:671-711)
// ---------------------------------------------------------------------------

// HandleStereo is the plain skeleton, and the only handler that reaches outside the
// dynamics system: ToggleMusicWhilePlaying at Timer == 0 in **both** branches.
//
// So a stereo does not switch the music on and off -- it *toggles* it, twice per switch
// throw, once on the way on and once on the way off. Two stereos in one room wired to the
// same switch therefore cancel each other out, and a stereo switched on in a house whose
// music is already playing switches the music off. ToggleStereos' unique `Timer == 0`
// guard exists precisely to stop a mashed switch from toggling the music several times a
// second.
func (w *World) HandleStereo(who int16) {
	if w.Dinahs[who].Timer <= 0 {
		return
	}
	w.Dinahs[who].Timer--
	if w.Dinahs[who].Active {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
			w.ToggleMusicWhilePlaying()
		case 1:
			w.PlayPrioritySound(MacOnSound, MacOnPriority)
			w.applianceToBack(render.StereoLight2, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	} else {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
			w.ToggleMusicWhilePlaying()
		case 1:
			w.PlayPrioritySound(MacOffSound, MacOffPriority)
			w.applianceToBack(render.StereoLight1, w.Dinahs[who].Dest)
			w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
		}
	}
}

// ---------------------------------------------------------------------------
// HandleMicrowave (Dynamics.c:715-775)
// ---------------------------------------------------------------------------

// HandleMicrowave is the plain skeleton with the blit tiled three times.
//
// The registration forces the slot 48 wide over 16-wide art
// (AddDynamicObject's kMicrowave case: `dest.right = dest.left + 48`), and the handler
// pays that back by laying the 16px window down three times:
//
//	dest = dinahs[who].dest; dest.right = dest.left + 16;
//	CopyBits(microOn -> back, dest);  QOffsetRect(&dest, 16, 0);   // x3
//	AddRectToBackRects(&dinahs[who].dest);                          // the full 48
//
// The C unrolls it; a three-iteration loop here says the same thing. Note the dirty rect
// registered is the full 48-wide slot rect, **not** the 16-wide scratch, so the staging is
// correct even though the scratch has been walked off the right-hand end.
func (w *World) HandleMicrowave(who int16) {
	if w.Dinahs[who].Timer <= 0 {
		return
	}
	w.Dinahs[who].Timer--
	if w.Dinahs[who].Active {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(MacOnSound, MacOnPriority)
			w.tileMicrowave(who, render.MicroOn)
		}
	} else {
		switch w.Dinahs[who].Timer {
		case 0:
			w.AddRectToWorkRects(player.Rect(w.Dinahs[who].Dest))
		case 1:
			w.PlayPrioritySound(MacOffSound, MacOffPriority)
			w.tileMicrowave(who, render.MicroOff)
		}
	}
}

// tileMicrowave is the six lines HandleMicrowave's two arms share, differing only in the
// art. internal/render's DrawMicrowave does the same three-blit walk for the static draw.
func (w *World) tileMicrowave(who int16, src Rect) {
	dest := w.Dinahs[who].Dest
	dest.Right = dest.Left + 16
	for i := 0; i < 3; i++ {
		w.applianceToBack(src, dest)
		dest = render.Offset(dest, 16, 0)
	}
	w.AddRectToBackRects(player.Rect(w.Dinahs[who].Dest))
}
