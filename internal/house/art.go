package house

// WantsOwnArt reports whether this house names a picture that only its own resource fork could
// supply: a room background at or above FirstUserBackground, or a kCustomPict anywhere.
//
// It exists so that the two places which mount a house's fork can tell the difference between
// "there is no fork here" and "there is no fork here *and* this house needed one"
// (docs/IMPROVEMENTS.md 4.17). Both used to warn on the first, which meant the one house in the
// library that provably needs no warning -- `Open House`, drawn entirely with built-in
// backgrounds -- was the only house to get one on a complete asset tree, and the first thing its
// author saw was a message saying they had done something wrong.
//
// **kCustomPict counts whatever id it names, and that is not laziness.** A kCustomPict's `pict`
// may be a house resource or an application one, and both happen: Metropolis carries its own PICT
// 1999 and Fun House its own 2014 and 2015, all three of which shadow application art
// (internal/render/housepict.go). So an id below FirstUserBackground is no evidence that the fork
// is unwanted, and the only test that would be exact is "is this id in the application chain and
// not in the house's" -- which needs the very fork whose absence is being reported.
//
// **kTV and kSoundTrigger deliberately do not count.** They are the two objects whose resources a
// house also carries, and neither comes out of the fork this predicate is about. A kTV wants a
// QuickTime movie, which the port has no support for and draws built-in art instead
// (internal/game/dynamics_appliances.go); a kSoundTrigger wants a `snd `, which arrives through
// the sound root's per-house manifest (internal/audio.Bank.LoadHouse) rather than through
// houseart/. A house of nothing but sound triggers is not a house with missing pictures, and the
// message this feeds says pictures.
func (h *House) WantsOwnArt() bool {
	for i := range h.Rooms {
		rm := &h.Rooms[i]
		if rm.Background >= FirstUserBackground {
			return true
		}
		for j := range rm.Objects {
			if rm.Objects[j].What == codeCustomPict {
				return true
			}
		}
	}
	return false
}
