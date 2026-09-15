package game

// Randomness: RandomInt (Utilities.c:72-81) and the Toolbox Random() it is built on.
//
// Three statements of arithmetic over one 16-bit draw, and everything in the game that
// is not deterministic goes through it: which way a balloon drifts, when the telephone
// rings, which of the two wind chimes strikes, where a shred of glider flies, how long a
// sparkle lasts. So this file is where a fidelity replay either works or does not.
//
// **The generator itself is not in the vendored sources.** `Random()` lives in the Mac
// Toolbox -- ROM on a 68k Mac, InterfaceLib on PowerPC -- and Glider PRO only calls it.
// What is below is the published algorithm for it: the minimal standard Lehmer generator
// (Park & Miller, multiplier 16807, modulus 2^31-1) evaluated by Schrage's method, with
// the low sixteen bits of the new seed returned as a signed short. That is what Apple
// documented and what every reimplementation of the Toolbox uses.
//
// It is deterministic and seedable, which is what the port actually needs, but it has not
// been verified against a real Mac's output because there is no Mac here to ask. Until it
// is, a recorded demo from 1994 may drift. Stage 1.8 owns that check; see
// docs/IMPROVEMENTS.md 2.18.

// Random is the Toolbox's Random(): advance the seed and return its low word, signed.
//
// The range is [-32768, 32767] and it is *not* uniform over a power of two -- the low
// sixteen bits of a mod-(2^31-1) sequence are very slightly biased, and the generator
// never produces a seed of 0, so the sequence has period 2^31-2 rather than 2^32.
//
// Schrage's method is what keeps the intermediate product inside 31 bits: 16807 * 127772
// is 2,147,464,004, four hundred thousand short of int32's ceiling. That is the whole
// reason for the 127773 split, and it is why this is written in int32 rather than int64 --
// widening it would work and would stop the arithmetic being the thing it is a
// transcription of.
func (w *World) Random() int16 {
	// A zero seed is a fixed point of the recurrence and would return 0 forever. The
	// Toolbox seeds from the clock (Utilities.c:60, GetDateTime into qd.randSeed) and
	// cannot hit it; here the zero value of the field can, so it is nudged. See
	// World.RandSeed for why the default is 1 and not the wall clock.
	if w.RandSeed <= 0 || w.RandSeed == 0x7FFFFFFF {
		w.RandSeed = 1
	}

	hi := w.RandSeed / 127773
	lo := w.RandSeed % 127773
	t := 16807*lo - 2836*hi
	if t <= 0 {
		t += 0x7FFFFFFF
	}
	w.RandSeed = t

	return int16(uint16(t & 0xFFFF))
}

// RandomInt is Utilities.c:72-81: a random integer "within range".
//
// Three statements, and the third has a bug worth being precise about:
//
//	rawResult = Random();                              // [-32768, 32767]
//	if (rawResult < 0L) rawResult *= -1L;              // [0, 32768]  <-- note the 32768
//	rawResult = (rawResult * (long)range) / 32768L;    // [0, range]  <-- inclusive
//
// Negating -32768 as a *long* gives 32768, not the overflow a short would give, so the
// folded range has 32769 values and the last of them makes the division come out at
// exactly `range`. **RandomInt(n) can return n**, with probability 1/65536.
//
// That is one draw in 65536 landing one past the end of whatever array the caller is
// indexing. The original gets away with it everywhere it matters by luck and by the
// tables being longer than they need to be; PourScreenOn is the one place it is visibly
// load-bearing, which is part of why that function is not transcribed (see
// screen.go). The bug is reproduced because removing it changes the sequence of every
// subsequent draw, and a port whose RNG stream differs from the original's cannot replay
// anything.
//
// Callers that cannot tolerate the overshoot must clamp at the call site, and the port's
// do -- each with a comment saying so, rather than being quietly protected from here.
//
// A negative `range` returns a value in [range, 0]; nothing in the game passes one.
func (w *World) RandomInt(rng int16) int16 {
	rawResult := int32(w.Random())
	if rawResult < 0 {
		rawResult *= -1
	}
	rawResult = (rawResult * int32(rng)) / 32768
	return int16(rawResult)
}
