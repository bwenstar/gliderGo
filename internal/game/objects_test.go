package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"glidergo/internal/house"
	"glidergo/internal/render"
)

var update = flag.Bool("update", false, "rewrite the golden files from the current output")

const (
	assetRoot  = "../../assets/extracted"
	goldenFile = "testdata/objects_golden.txt"
)

// requireAssets skips rather than fails when the extracted art is absent, the same
// way internal/render's tests do. The houses are derived from a copyrighted 1994
// application and are not in the repository.
func requireAssets(t *testing.T, sub string) string {
	t.Helper()
	dir := filepath.Join(assetRoot, sub)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("no extracted assets at %s (run `make assets`)", dir)
	}
	return dir
}

// newTestWorld builds a headless world on a house: a default 640x480 view, an asset
// loader that may or may not find art, and a fixed random seed.
func newTestWorld(h *house.House, artDir, forkDir string) *World {
	a := render.NewAssets(artDir)
	if forkDir != "" {
		a.OpenHouseResFork(forkDir)
	}
	sc := render.NewScene(render.DefaultView(), a, h)
	w := NewWorld(h, sc, 1)
	// A sound system that always succeeds, so the kSoundTrigger path is exercised
	// and the one-slot limit is what the corpus measures. A nil hook would make every
	// sound trigger inert and the test would prove nothing about it.
	w.TriggerSoundExists = func(int16) bool { return true }
	return w
}

// ---------------------------------------------------------------------------
// The object vocabulary
// ---------------------------------------------------------------------------

// TestObjectCodesMatchHouse pins this package's 117 constants against
// internal/house's name table.
//
// The two exist separately -- house needs names for its text codec and game needs
// switchable constants -- and a disagreement between them would be invisible: a
// mistyped constant here still compiles and still switches, it just switches on the
// wrong object. Every code is checked in both directions.
func TestObjectCodesMatchHouse(t *testing.T) {
	codes := map[int16]string{
		FloorVent: "FloorVent", CeilingVent: "CeilingVent", FloorBlower: "FloorBlower",
		CeilingBlower: "CeilingBlower", SewerGrate: "SewerGrate", LeftFan: "LeftFan",
		RightFan: "RightFan", Taper: "Taper", Candle: "Candle", Stubby: "Stubby",
		Tiki: "Tiki", BBQ: "BBQ", InvisBlower: "InvisBlower", GrecoVent: "GrecoVent",
		SewerBlower: "SewerBlower", LiftArea: "LiftArea",

		Table: "Table", Shelf: "Shelf", Cabinet: "Cabinet",
		FilingCabinet: "FilingCabinet", WasteBasket: "WasteBasket",
		MilkCrate: "MilkCrate", Counter: "Counter", Dresser: "Dresser",
		DeckTable: "DeckTable", Stool: "Stool", Trunk: "Trunk",
		InvisObstacle: "InvisObstacle", Manhole: "Manhole", Books: "Books",
		InvisBounce: "InvisBounce",

		RedClock: "RedClock", BlueClock: "BlueClock", YellowClock: "YellowClock",
		Cuckoo: "Cuckoo", Paper: "Paper", Battery: "Battery", Bands: "Bands",
		GreaseRt: "GreaseRt", GreaseLf: "GreaseLf", Foil: "Foil",
		InvisBonus: "InvisBonus", Star: "Star", Sparkle: "Sparkle", Helium: "Helium",
		Slider: "Slider",

		UpStairs: "UpStairs", DownStairs: "DownStairs", MailboxLf: "MailboxLf",
		MailboxRt: "MailboxRt", FloorTrans: "FloorTrans", CeilingTrans: "CeilingTrans",
		DoorInLf: "DoorInLf", DoorInRt: "DoorInRt", DoorExRt: "DoorExRt",
		DoorExLf: "DoorExLf", WindowInLf: "WindowInLf", WindowInRt: "WindowInRt",
		WindowExRt: "WindowExRt", WindowExLf: "WindowExLf", InvisTrans: "InvisTrans",
		DeluxeTrans: "DeluxeTrans",

		LightSwitch: "LightSwitch", MachineSwitch: "MachineSwitch",
		Thermostat: "Thermostat", PowerSwitch: "PowerSwitch",
		KnifeSwitch: "KnifeSwitch", InvisSwitch: "InvisSwitch", Trigger: "Trigger",
		LgTrigger: "LgTrigger", SoundTrigger: "SoundTrigger",

		CeilingLight: "CeilingLight", LightBulb: "LightBulb", TableLamp: "TableLamp",
		HipLamp: "HipLamp", DecoLamp: "DecoLamp", Flourescent: "Flourescent",
		TrackLight: "TrackLight", InvisLight: "InvisLight",

		Shredder: "Shredder", Toaster: "Toaster", MacPlus: "MacPlus",
		Guitar: "Guitar", TV: "TV", Coffee: "Coffee", Outlet: "Outlet", VCR: "VCR",
		Stereo: "Stereo", Microwave: "Microwave", CinderBlock: "CinderBlock",
		FlowerBox: "FlowerBox", CDs: "CDs", CustomPict: "CustomPict",

		Balloon: "Balloon", CopterLf: "CopterLf", CopterRt: "CopterRt",
		DartLf: "DartLf", DartRt: "DartRt", Ball: "Ball", Drip: "Drip", Fish: "Fish",
		Cobweb: "Cobweb",

		Ozma: "Ozma", Mirror: "Mirror", Mousehole: "Mousehole",
		Fireplace: "Fireplace", Flower: "Flower", WallWindow: "WallWindow",
		Bear: "Bear", Calendar: "Calendar", Vase1: "Vase1", Vase2: "Vase2",
		Bulletin: "Bulletin", Cloud: "Cloud", Faucet: "Faucet", Rug: "Rug",
		Chimes: "Chimes",
	}

	if len(codes) != 117 {
		t.Errorf("the table lists %d codes, want 117", len(codes))
	}
	// house spells them with the original's k prefix; this package drops it, since
	// game.FloorVent already reads as a name and game.kFloorVent would not be exported.
	for code, name := range codes {
		if got := house.ObjectName(code); got != "k"+name {
			t.Errorf("code 0x%02X: this package calls it %s, house calls it %s",
				code, name, got)
		}
		if got, ok := house.ObjectCode("k" + name); !ok || got != code {
			t.Errorf("%s: this package has 0x%02X, house has 0x%02X (ok=%v)",
				name, code, got, ok)
		}
	}
}

// TestActionNames checks the action table covers exactly the 28 actions and has no
// gaps, since a gap would silently rename every action after it.
func TestActionNames(t *testing.T) {
	for a := int16(0); a < NumHotSpotActions; a++ {
		n := ActionName(a)
		if n == "kUnknownAction" || !strings.HasPrefix(n, "k") {
			t.Errorf("action %d: name %q", a, n)
		}
	}
	if got := ActionName(NumHotSpotActions); got != "kUnknownAction" {
		t.Errorf("action %d: got %q, want kUnknownAction", NumHotSpotActions, got)
	}
	seen := map[string]int16{}
	for a := int16(0); a < NumHotSpotActions; a++ {
		if prev, dup := seen[ActionName(a)]; dup {
			t.Errorf("actions %d and %d share the name %s", prev, a, ActionName(a))
		}
		seen[ActionName(a)] = a
	}
}

// ---------------------------------------------------------------------------
// SetObjectState
// ---------------------------------------------------------------------------

// switchableTypes is the set SetObjectState can actually change: the eleven blowers,
// the fourteen prizes, kDeluxeTrans, the eight lights, the eight appliances, kStereo
// and the eight enemies. 51 of the 117, and the other 66 are cases that exist only to
// return false.
func switchableTypes() map[int16]bool {
	m := map[int16]bool{}
	for _, what := range []int16{
		FloorVent, CeilingVent, FloorBlower, CeilingBlower, LeftFan, RightFan,
		SewerGrate, InvisBlower, GrecoVent, SewerBlower, LiftArea,
		RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery, Bands,
		GreaseRt, GreaseLf, Foil, InvisBonus, Star, Sparkle, Helium,
		DeluxeTrans,
		CeilingLight, LightBulb, TableLamp, HipLamp, DecoLamp, Flourescent,
		TrackLight, InvisLight,
		Stereo,
		Shredder, Toaster, MacPlus, TV, Coffee, Outlet, VCR, Microwave,
		Balloon, CopterLf, CopterRt, DartLf, DartRt, Ball, Drip, Fish,
	} {
		m[what] = true
	}
	return m
}

// ignoresAction is the two types whose write is not driven by `action` at all: a prize
// is always cleared and a stereo always toggles.
func ignoresAction(what int16) bool {
	switch what {
	case RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery, Bands,
		GreaseRt, GreaseLf, Foil, InvisBonus, Star, Sparkle, Helium, Stereo:
		return true
	}
	return false
}

// TestSetObjectStateEveryType drives SetObjectState over all 117 types with all three
// actions, from both starting states, in and out of the central room, and with and
// without a master index. 2,808 cases.
//
// It is a robustness sweep first -- the original indexes rooms[], objects[],
// masterObjects[] and hotSpots[] with no bounds test anywhere, and each of those is a
// panic in Go rather than a garbage read -- and an exact behavioural check second: the
// return value is predicted for every case from the type's family and the action, not
// merely sampled.
func TestSetObjectStateEveryType(t *testing.T) {
	switchable := switchableTypes()
	if len(switchable) != 51 {
		t.Fatalf("the switchable set has %d entries, want 51", len(switchable))
	}

	// What `changed` must be, worked out from the C rather than from the port: a
	// switch reports a change only when the state it would write differs from the
	// state that is there, except for the two families that ignore `action`.
	want := func(what, action int16, on bool) bool {
		if !switchable[what] {
			return false
		}
		if what == Stereo {
			return true // toggles a global, always reports a change
		}
		if ignoresAction(what) {
			return on // a prize is cleared, so it changes only if uncollected
		}
		switch action {
		case Toggle:
			return true
		case ForceOn:
			return !on
		case ForceOff:
			return on
		}
		return false
	}

	cases := 0
	for what := int16(0x01); what <= 0x8F; what++ {
		if house.ObjectName(what) == "" {
			continue // one of the undefined codes in the range
		}
		for _, action := range []int16{Toggle, ForceOn, ForceOff} {
			for _, on := range []bool{false, true} {
				for _, local := range []int16{0, -1} {
					for _, central := range []bool{true, false} {
						cases++
						h := oneRoomHouse(what)
						obj := &h.Rooms[0].Objects[0]
						if off := stateOffset(what); off >= 0 && on {
							obj.Data[off] = 1
						}
						if what == DeluxeTrans {
							obj.Data[8] = 3 // linked, so it composes
						}

						w := newTestWorld(h, "", "")
						w.R.PlayMusicGame = on
						composeRoom(w, 0)
						if !central {
							// Pretend the glider is elsewhere, so the prize family's
							// nested `room == thisRoomNumber` test fails.
							w.R.RoomNumber = 7
						}

						got := w.SetObjectState(0, 0, action, local)
						if exp := want(what, action, on); got != exp {
							t.Errorf("0x%02X %s action=%d on=%v local=%d central=%v: "+
								"changed=%v, want %v", what, house.ObjectName(what),
								action, on, local, central, got, exp)
						}
						// A second identical call must flip again for a toggle, and be
						// a no-op for a force and for the write-once prize family --
						// which is what makes a collected prize uncollectable.
						second := w.SetObjectState(0, 0, action, local)
						wantSecond := switchable[what] &&
							(what == Stereo || (action == Toggle && !ignoresAction(what)))
						if second != wantSecond {
							t.Errorf("%s action=%d on=%v: second call changed=%v, want %v",
								house.ObjectName(what), action, on, second, wantSecond)
						}
					}
				}
			}
		}
	}
	if cases != 2808 {
		t.Errorf("%d cases, want 2808 (117 types x 3 actions x 2 states x 2 locals x 2 rooms)",
			cases)
	}
}

// TestSetObjectStateUnknownAction pins what an out-of-range action does, which is not
// nothing.
//
// The C's inner `switch (action)` has no default, so an unrecognised action leaves both
// locals alone -- and then the unconditional `data.x.state = newState` after the switch
// writes the *false* that newState was initialised to. So an unknown action silently
// forces the object off while reporting that nothing changed.
//
// Unreachable from the shipped callers, which only ever pass the three. Worth a test
// because it is the one place a wrong `action` is worse than a no-op, and because the
// two action-ignoring families behave differently again.
func TestSetObjectStateUnknownAction(t *testing.T) {
	const bogus int16 = 99

	for what := range switchableTypes() {
		h := oneRoomHouse(what)
		obj := &h.Rooms[0].Objects[0]
		off := stateOffset(what)
		if off >= 0 {
			obj.Data[off] = 1
		}
		if what == DeluxeTrans {
			obj.Data[8] = 3 // linked, so it composes
		}

		w := newTestWorld(h, "", "")
		w.R.PlayMusicGame = true
		composeRoom(w, 0)

		changed := w.SetObjectState(0, 0, bogus, 0)

		if ignoresAction(what) {
			if !changed {
				t.Errorf("%s: an action-ignoring type should still report a change",
					house.ObjectName(what))
			}
			continue
		}
		if changed {
			t.Errorf("%s: an unknown action should report no change",
				house.ObjectName(what))
		}
		if off >= 0 && obj.Data[off]&0x0F != 0 {
			t.Errorf("%s: state is %#02x after an unknown action, want the forced 0",
				house.ObjectName(what), obj.Data[off])
		}
	}
}

// stateOffset is the payload offset of a type's state byte, or -1 for the types that
// have none. It is the test's own copy of the mapping setstate.go switches over, kept
// separate on purpose: a test that derived it from the code under test would agree with
// a wrong answer.
func stateOffset(what int16) int {
	switch what {
	case FloorVent, CeilingVent, FloorBlower, CeilingBlower, LeftFan, RightFan,
		SewerGrate, InvisBlower, GrecoVent, SewerBlower, LiftArea:
		return offBlowerState
	case RedClock, BlueClock, YellowClock, Cuckoo, Paper, Battery, Bands,
		GreaseRt, GreaseLf, Foil, InvisBonus, Star, Sparkle, Helium:
		return offBonusState
	case DeluxeTrans:
		return offTransportWide
	case CeilingLight, LightBulb, TableLamp, HipLamp, DecoLamp, Flourescent,
		TrackLight, InvisLight:
		return offLightState
	case Shredder, Toaster, MacPlus, TV, Coffee, Outlet, VCR, Microwave:
		return offApplianceState
	case Balloon, CopterLf, CopterRt, DartLf, DartRt, Ball, Drip, Fish:
		return offEnemyState
	}
	return -1 // kStereo, which writes no object byte at all
}

// TestSetObjectStateOutOfRange checks the four indices the original never guards.
func TestSetObjectStateOutOfRange(t *testing.T) {
	h := oneRoomHouse(FloorVent)
	w := newTestWorld(h, "", "")
	composeRoom(w, 0)

	for _, c := range []struct{ room, object, local int16 }{
		{-1, 0, 0}, {99, 0, 0}, {0, -1, 0}, {0, MaxRoomObs, 0},
		{0, 0, -1}, {0, 0, 9999},
	} {
		// The point is that none of these panics.
		w.SetObjectState(c.room, c.object, Toggle, c.local)
	}
}

// TestPrizeStateBugIsReproduced pins the union-aliasing bug in the prize family:
// the house's state byte is cleared and the master copy's is not.
//
// This is the one test in the file that would fail if someone "fixed" the port, which
// is exactly why it exists. The behaviour is Objects.c:463 writing data.a.state --
// payload offset 7 -- inside the bonus family, whose state byte is at offset 8.
func TestPrizeStateBugIsReproduced(t *testing.T) {
	h := oneRoomHouse(InvisBonus)
	// Give the bonus a point value whose low byte is non-zero, so the collateral
	// damage to `points` is visible too. points is at offsets 6-7, big-endian.
	h.Rooms[0].Objects[0].Data[6] = 0x01
	h.Rooms[0].Objects[0].Data[7] = 0xF4 // 500
	h.Rooms[0].Objects[0].Data[offBonusState] = 1

	w := newTestWorld(h, "", "")
	composeRoom(w, 0)

	if !w.SetObjectState(0, 0, Toggle, 0) {
		t.Fatal("collecting an uncollected prize should report a change")
	}
	if got := h.Rooms[0].Objects[0].Data[offBonusState]; got != 0 {
		t.Errorf("house state byte = %d, want 0: the house write is the correct one", got)
	}
	m := w.R.Master[0].TheObject
	if got := m.Data[offBonusState]; got != 1 {
		t.Errorf("master copy state byte = %d, want 1 (still set): "+
			"the C writes offset 7, not 8, so the master copy is never cleared", got)
	}
	if got := m.Data[7]; got != 0 {
		t.Errorf("master copy offset 7 = %d, want 0: that is the byte the C clears", got)
	}
	if got := m.Bonus().Points; got != 256 {
		t.Errorf("master copy points = %d, want 256: 500 with its low byte cleared", got)
	}
	// The hot spot is switched off correctly, because that write uses hotNum and not
	// a union member.
	if len(w.R.Hot) == 0 {
		t.Fatal("an invisible bonus should have made a reward rect")
	}
	if w.R.Hot[0].IsOn {
		t.Error("the collected prize's hot spot should be off")
	}
}

// TestDeluxeTransNibbles checks the packed state field in both directions: the low
// nibble is the state, the high nibble is the default, and SetObjectsToDefaults
// restores one from the other.
func TestDeluxeTransNibbles(t *testing.T) {
	for _, initial := range []byte{0x00, 0x10} {
		h := oneRoomHouse(DeluxeTrans)
		h.Rooms[0].Objects[0].Data[offTransportWide] = initial
		h.Rooms[0].Objects[0].Data[8] = 0 // who: linked, so a rect is made

		w := newTestWorld(h, "", "")
		composeRoom(w, 0)

		wide := &h.Rooms[0].Objects[0].Data[offTransportWide]
		if !w.SetObjectState(0, 0, ForceOn, 0) && initial == 0x00 {
			t.Errorf("initial %#02x: ForceOn on an off transporter should change it", initial)
		}
		if *wide&0x0F == 0 {
			t.Errorf("initial %#02x: after ForceOn the low nibble is 0", initial)
		}
		if *wide&0xF0 != initial&0xF0 {
			t.Errorf("initial %#02x: ForceOn changed the high nibble to %#02x", initial, *wide&0xF0)
		}

		w.SetObjectState(0, 0, ForceOff, 0)
		if *wide&0x0F != 0 {
			t.Errorf("initial %#02x: after ForceOff the low nibble is %d", initial, *wide&0x0F)
		}

		w.SetObjectsToDefaults()
		wantOn := initial&0xF0 != 0
		if gotOn := *wide&0x0F != 0; gotOn != wantOn {
			t.Errorf("initial %#02x: after defaults, on=%v want %v (wide=%#02x)",
				initial, gotOn, wantOn, *wide)
		}
	}
}

// TestSetObjectsToDefaultsRestoresEveryStatefulType checks that every type whose
// state is read is restored, by flipping each one away from its initial and asking
// for it back.
//
// It is the positive form of the claim in defaults.go's header: the types the function
// omits are exactly the types whose state nothing reads.
func TestSetObjectsToDefaultsRestoresEveryStatefulType(t *testing.T) {
	type spec struct {
		what              int16
		stateOff, initOff int
	}
	specs := []spec{
		{FloorVent, offBlowerState, offBlowerInitial},
		{LiftArea, offBlowerState, offBlowerInitial},
		{RedClock, offBonusState, offBonusInitial},
		{GreaseRt, offBonusState, offBonusInitial},
		{Sparkle, offBonusState, offBonusInitial},
		{CeilingLight, offLightState, offLightInitial},
		{InvisLight, offLightState, offLightInitial},
		{Shredder, offApplianceState, offApplianceInitial},
		{Guitar, offApplianceState, offApplianceInitial},
		{Balloon, offEnemyState, offEnemyInitial},
		{Fish, offEnemyState, offEnemyInitial},
	}
	for _, s := range specs {
		for _, initial := range []byte{0, 1} {
			h := oneRoomHouse(s.what)
			obj := &h.Rooms[0].Objects[0]
			obj.Data[s.initOff] = initial
			obj.Data[s.stateOff] = 1 - initial
			h.Rooms[0].Visited = 1

			w := newTestWorld(h, "", "")
			w.SetObjectsToDefaults()

			if got := obj.Data[s.stateOff]; got != initial {
				t.Errorf("%s initial=%d: state came back as %d",
					house.ObjectName(s.what), initial, got)
			}
			if h.Rooms[0].Visited != 0 {
				t.Errorf("%s: visited not cleared", house.ObjectName(s.what))
			}
		}
	}
}

// TestStereoIgnoresAction pins the one type that answers a switch by toggling a
// global: every action toggles, so a trigger's ForceOn turns music off if it was on.
func TestStereoIgnoresAction(t *testing.T) {
	h := oneRoomHouse(Stereo)
	w := newTestWorld(h, "", "")
	w.R.PlayMusicGame = true
	composeRoom(w, 0)

	for _, action := range []int16{ForceOn, ForceOn, ForceOff, Toggle} {
		before := w.R.PlayMusicGame
		if !w.SetObjectState(0, 0, action, 0) {
			t.Errorf("action %d: a stereo always reports a change", action)
		}
		if w.R.PlayMusicGame == before {
			t.Errorf("action %d: music flag did not toggle", action)
		}
	}
}

// ---------------------------------------------------------------------------
// The room predicates
// ---------------------------------------------------------------------------

// TestShadowVisibleDiffersFromHasFloor pins the one-label difference between
// IsShadowVisible and DoesRoomHaveFloor, which is the pair of functions most likely
// to be "simplified" into one.
func TestShadowVisibleDiffersFromHasFloor(t *testing.T) {
	cases := []struct {
		background            int16
		shadow, floor, ceilng bool
	}{
		{SimpleRoom, true, true, true},
		{Basement, true, true, true},
		{Garden, true, true, false},
		{Meadow, true, true, false},
		{Field, true, true, false},
		{Roof, false, true, false}, // the difference: no shadow, but a floor
		{Sky, false, false, false},
		{Stratosphere, false, false, false},
		{Stars, false, false, false},
	}
	for _, c := range cases {
		h := oneRoomHouse(house.ObjectIsEmpty)
		h.Rooms[0].Background = c.background
		w := newTestWorld(h, "", "")
		w.R.RoomNumber = 0

		if got := w.IsShadowVisible(); got != c.shadow {
			t.Errorf("background %d: IsShadowVisible=%v want %v", c.background, got, c.shadow)
		}
		if got := w.DoesRoomHaveFloor(); got != c.floor {
			t.Errorf("background %d: DoesRoomHaveFloor=%v want %v", c.background, got, c.floor)
		}
		if got := w.DoesRoomHaveCeiling(); got != c.ceilng {
			t.Errorf("background %d: DoesRoomHaveCeiling=%v want %v", c.background, got, c.ceilng)
		}
	}
	// And the claim in room.go's header table, as an assertion rather than a comment.
	h := oneRoomHouse(house.ObjectIsEmpty)
	h.Rooms[0].Background = Roof
	w := newTestWorld(h, "", "")
	w.R.RoomNumber = 0
	if w.IsShadowVisible() == w.DoesRoomHaveFloor() {
		t.Error("on a roof the two predicates must disagree")
	}
}

// TestDirtOpeningsAreInconsistent pins the kDirt branch of DetermineRoomOpenings,
// where the threshold and the flag beside it are computed from different digits.
//
// The threshold tests `leftTile == 1` and the flag tests `leftTile != 0`, so *two* of
// the eight tiles disagree and they disagree in opposite directions:
//
//	tile 0  -- no wall by the threshold, not open by the flag
//	tile 1  -- a wall by the threshold, open by the flag
//	tile 2+ -- no wall, open: the two agree
//
// Tile 1 is the one that lets the glider out: CheckGliderInRoom reads the threshold and
// refuses to let it fly through, while the escape functions read the flag and hand it to
// the next room if something pushes it there. Tile 0 is the mirror case and merely traps
// it against an invisible edge.
func TestDirtOpeningsAreInconsistent(t *testing.T) {
	h := oneRoomHouse(house.ObjectIsEmpty)
	h.Rooms[0].Background = Dirt
	h.Rooms[0].Tiles[NumTiles-1] = 3

	w := newTestWorld(h, "", "")
	w.R.RoomNumber = 0

	cases := []struct {
		tile   int16
		thresh int16
		open   bool
	}{
		{0, NoLeftWallLimit, false}, // no wall, not open
		{1, LeftWallLimit, true},    // a wall, open
		{2, NoLeftWallLimit, true},  // the two agree
		{5, NoLeftWallLimit, true},
	}
	for _, c := range cases {
		h.Rooms[0].Tiles[0] = c.tile
		w.DetermineRoomOpenings()
		if w.R.LeftThresh != c.thresh || w.R.LeftOpen != c.open {
			t.Errorf("kDirt leftTile %d: thresh=%d open=%v, want %d and %v",
				c.tile, w.R.LeftThresh, w.R.LeftOpen, c.thresh, c.open)
		}
	}

	// An interior with the same tiles agrees on both, which is what makes this a
	// one-digit divergence rather than a general property of the tile test.
	h.Rooms[0].Background = SimpleRoom
	for _, tile := range []int16{0, 1, 2, 5} {
		h.Rooms[0].Tiles[0] = tile
		w.DetermineRoomOpenings()
		walled := w.R.LeftThresh == LeftWallLimit
		if walled == w.R.LeftOpen {
			t.Errorf("kSimpleRoom leftTile %d: thresh and flag disagree "+
				"(walled=%v open=%v)", tile, walled, w.R.LeftOpen)
		}
	}
}

// TestDirtInconsistencyIsReachableInShippedHouses counts the rooms where the kDirt
// disagreement actually bites, so that the divergence documented in room.go is backed
// by a number rather than an assumption.
//
// The census is pinned. If it moves, either the predicate changed or the houses did.
func TestDirtInconsistencyIsReachableInShippedHouses(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	paths, _ := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	dirtRooms, dirtHouses, disagree := 0, 0, 0
	perHouse := map[string]int{}
	for _, path := range paths {
		h, err := house.LoadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		w := newTestWorld(h, "", "")
		hits := 0
		for r := range h.Rooms {
			if h.Rooms[r].Background != Dirt {
				continue
			}
			dirtRooms++
			w.R.RoomNumber = int16(r)
			w.DetermineRoomOpenings()
			// The left edge is the only one the two digits differ on; the right edge
			// uses the same test in both branches.
			if (w.R.LeftThresh == LeftWallLimit) == w.R.LeftOpen {
				disagree++
				hits++
			}
		}
		if hits > 0 {
			perHouse[name] = hits
			dirtHouses++
		}
	}

	names := make([]string, 0, len(perHouse))
	for n := range perHouse {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t.Logf("%s: %d kDirt rooms where the threshold and the flag disagree", n, perHouse[n])
	}
	t.Logf("%d kDirt rooms in the corpus, %d of them inconsistent, across %d houses",
		dirtRooms, disagree, dirtHouses)

	if dirtRooms == 0 {
		t.Error("no shipped room uses kDirt, so room.go's note about six houses is wrong")
	}
}

// TestOpeningInvariantsOverCorpus asserts the structural properties of
// DetermineRoomOpenings over all 4,070 rooms, which the golden hash pins but does not
// explain.
//
// Three claims, each from the C's shape rather than from the port's:
// the thresholds only ever take one of two values per side; the five wall-less
// backgrounds are open regardless of their tiles; and top and bottom depend on the
// background alone, so two rooms sharing a background always agree on them.
func TestOpeningInvariantsOverCorpus(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	paths, _ := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	wallLess := map[int16]bool{
		Garden: true, Skywalk: true, Field: true, Stratosphere: true, Stars: true,
	}
	type vert struct{ top, bottom bool }
	byBackground := map[int16]vert{}

	rooms := 0
	for _, path := range paths {
		h, err := house.LoadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		w := newTestWorld(h, "", "")
		for r := range h.Rooms {
			rooms++
			w.R.RoomNumber = int16(r)
			w.DetermineRoomOpenings()

			if w.R.LeftThresh != LeftWallLimit && w.R.LeftThresh != NoLeftWallLimit {
				t.Fatalf("%s room %d: LeftThresh=%d is neither limit",
					name, r, w.R.LeftThresh)
			}
			if w.R.RightThresh != RightWallLimit && w.R.RightThresh != NoRightWallLimit {
				t.Fatalf("%s room %d: RightThresh=%d is neither limit",
					name, r, w.R.RightThresh)
			}

			bg := h.Rooms[r].Background
			if wallLess[bg] {
				if !w.R.LeftOpen || !w.R.RightOpen ||
					w.R.LeftThresh != NoLeftWallLimit ||
					w.R.RightThresh != NoRightWallLimit {
					t.Errorf("%s room %d: background %d has no side walls but came "+
						"out left=(%d,%v) right=(%d,%v)", name, r, bg,
						w.R.LeftThresh, w.R.LeftOpen, w.R.RightThresh, w.R.RightOpen)
				}
			}

			// Only the built-in backgrounds: a house background's openings come from
			// the room's own bounds field, so two rooms can legitimately differ.
			if bg < UserBackground {
				got := vert{w.R.TopOpen, w.R.BottomOpen}
				if prev, seen := byBackground[bg]; seen && prev != got {
					t.Errorf("%s room %d: background %d gave top=%v bottom=%v, but "+
						"another room with the same background gave top=%v bottom=%v",
						name, r, bg, got.top, got.bottom, prev.top, prev.bottom)
				}
				byBackground[bg] = got
			}
		}
	}
	t.Logf("%d rooms, %d distinct built-in backgrounds", rooms, len(byBackground))
	if rooms != 4070 {
		t.Errorf("%d rooms in the corpus, want 4070", rooms)
	}
}

// TestInteriorOpenings walks the three tile-driven branches.
func TestInteriorOpenings(t *testing.T) {
	cases := []struct {
		background          int16
		left, right         int16
		lThresh, rThresh    int16
		leftOpen, rightOpen bool
	}{
		{SimpleRoom, 0, 7, LeftWallLimit, RightWallLimit, false, false},
		{SimpleRoom, 3, 3, NoLeftWallLimit, NoRightWallLimit, true, true},
		{Sky, 0, 7, LeftWallLimit, RightWallLimit, false, false},
		{Meadow, 6, 7, LeftWallLimit, RightWallLimit, false, false},
		{Meadow, 0, 0, NoLeftWallLimit, NoRightWallLimit, true, true},
		{Garden, 0, 7, NoLeftWallLimit, NoRightWallLimit, true, true},
		{Stars, 0, 7, NoLeftWallLimit, NoRightWallLimit, true, true},
	}
	for _, c := range cases {
		h := oneRoomHouse(house.ObjectIsEmpty)
		h.Rooms[0].Background = c.background
		h.Rooms[0].Tiles[0] = c.left
		h.Rooms[0].Tiles[NumTiles-1] = c.right

		w := newTestWorld(h, "", "")
		w.R.RoomNumber = 0
		w.DetermineRoomOpenings()

		if w.R.LeftThresh != c.lThresh || w.R.RightThresh != c.rThresh ||
			w.R.LeftOpen != c.leftOpen || w.R.RightOpen != c.rightOpen {
			t.Errorf("background %d tiles %d..%d: got (%d,%d,%v,%v) want (%d,%d,%v,%v)",
				c.background, c.left, c.right,
				w.R.LeftThresh, w.R.RightThresh, w.R.LeftOpen, w.R.RightOpen,
				c.lThresh, c.rThresh, c.leftOpen, c.rightOpen)
		}
	}
}

// ---------------------------------------------------------------------------
// The hot-spot table
// ---------------------------------------------------------------------------

// TestSoundTriggerIsOnePerRoom pins the single reserved sound slot: the first
// kSoundTrigger in a room gets a kSoundIt rect and every later one gets nothing.
//
// It is not an artefact of the port. LoadTriggerSound fails outright when
// theSoundData[kMaxSounds-1] is already occupied (Sound.c:269), and only
// DrawLocale's DumpTriggerSound frees it.
func TestSoundTriggerIsOnePerRoom(t *testing.T) {
	h := oneRoomHouse(SoundTrigger)
	for i := 0; i < 3; i++ {
		h.Rooms[0].Objects[i] = h.Rooms[0].Objects[0]
	}
	h.Rooms[0].NumObjects = 3

	w := newTestWorld(h, "", "")
	composeRoom(w, 0)

	n := 0
	for _, hs := range w.R.Hot {
		if hs.Action == SoundIt {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d kSoundIt rects for three sound triggers, want 1", n)
	}

	// And the slot is re-armed by the reset head, so the next room's first trigger
	// works again.
	w.R.TriggerSoundHeld = false
	w.ListAllLocalObjects()
	n = 0
	for _, hs := range w.R.Hot {
		if hs.Action == SoundIt {
			n++
		}
	}
	if n != 1 {
		t.Errorf("after re-arming: %d kSoundIt rects, want 1", n)
	}

	// With no sound system at all, no trigger gets a rect -- the dontLoadSounds path.
	w.TriggerSoundExists = nil
	w.R.TriggerSoundHeld = false
	w.ListAllLocalObjects()
	for _, hs := range w.R.Hot {
		if hs.Action == SoundIt {
			t.Error("a silent build should make no kSoundIt rects")
		}
	}

	// A trigger whose sound does not exist leaves the slot free, because
	// LoadTriggerSound's `theSound == nil` branch returns -1 without allocating. So
	// the *second* trigger is the one that works, and which of the three fires
	// depends on the objects[] slot order.
	h.Rooms[0].Objects[0].Data[6], h.Rooms[0].Objects[0].Data[7] = 0, 1 // where = 1
	h.Rooms[0].Objects[1].Data[6], h.Rooms[0].Objects[1].Data[7] = 0, 2 // where = 2
	h.Rooms[0].Objects[2].Data[6], h.Rooms[0].Objects[2].Data[7] = 0, 3 // where = 3
	w.TriggerSoundExists = func(id int16) bool { return id != 1 }
	w.R.TriggerSoundHeld = false
	w.ListAllLocalObjects()

	var owners []int16
	for _, hs := range w.R.Hot {
		if hs.Action == SoundIt {
			owners = append(owners, hs.Who)
		}
	}
	if len(owners) != 1 {
		t.Fatalf("%d kSoundIt rects, want 1", len(owners))
	}
	if owners[0] != 1 {
		t.Errorf("the working rect belongs to slot %d, want slot 1: slot 0's sound is "+
			"missing, which leaves the reserved slot free for slot 1", owners[0])
	}
}

// TestSparkleMakesNoRect pins the one type that computes a rect and discards it.
func TestSparkleMakesNoRect(t *testing.T) {
	h := oneRoomHouse(Sparkle)
	h.Rooms[0].Objects[0].Data[offBonusState] = 1
	w := newTestWorld(h, "", "")
	composeRoom(w, 0)

	if len(w.R.Hot) != 0 {
		t.Errorf("a sparkle made %d hot spots, want 0", len(w.R.Hot))
	}
	if w.R.Master[0].HotNum != -1 {
		t.Errorf("HotNum=%d, want -1", w.R.Master[0].HotNum)
	}
}

// TestMultiRectObjectsKeepTheLast pins the HotNum convention, which decides what a
// switch actually switches.
func TestMultiRectObjectsKeepTheLast(t *testing.T) {
	cases := []struct {
		what       int16
		wantRects  int
		wantAction int16
	}{
		// A fan makes a lethal blade box then a push column; the column is last, so
		// switching the fan switches the column and the blades stay lethal.
		{LeftFan, 2, PushItLeft},
		{RightFan, 2, PushItRight},
		// A microwave makes the oven then the beam; the beam is last.
		{Microwave, 2, MicrowaveIt},
		// A single-rect type for contrast.
		{Table, 1, DissolveIt},
	}
	for _, c := range cases {
		h := oneRoomHouse(c.what)
		w := newTestWorld(h, "", "")
		composeRoom(w, 0)

		if len(w.R.Hot) != c.wantRects {
			t.Errorf("%s: %d rects, want %d", house.ObjectName(c.what),
				len(w.R.Hot), c.wantRects)
			continue
		}
		hn := w.R.Master[0].HotNum
		if int(hn) != c.wantRects-1 {
			t.Errorf("%s: HotNum=%d, want %d (the last rect)",
				house.ObjectName(c.what), hn, c.wantRects-1)
			continue
		}
		if got := w.R.Hot[hn].Action; got != c.wantAction {
			t.Errorf("%s: last rect is %s, want %s", house.ObjectName(c.what),
				ActionName(got), ActionName(c.wantAction))
		}
	}
}

// TestFanBladesStayLethalWhenOff is the behavioural consequence of the previous
// test, asserted directly because it is surprising enough to be "fixed" by accident.
func TestFanBladesStayLethalWhenOff(t *testing.T) {
	h := oneRoomHouse(LeftFan)
	h.Rooms[0].Objects[0].Data[offBlowerState] = 1
	w := newTestWorld(h, "", "")
	composeRoom(w, 0)

	w.SetObjectState(0, 0, ForceOff, 0)

	if w.R.Hot[0].Action != DissolveIt || !w.R.Hot[0].IsOn {
		t.Errorf("blade box: action=%s isOn=%v, want kDissolveIt and on",
			ActionName(w.R.Hot[0].Action), w.R.Hot[0].IsOn)
	}
	if w.R.Hot[1].Action != PushItLeft || w.R.Hot[1].IsOn {
		t.Errorf("push column: action=%s isOn=%v, want kPushItLeft and off",
			ActionName(w.R.Hot[1].Action), w.R.Hot[1].IsOn)
	}
}

// TestFlameSplit pins the lift/burn split, which is arithmetic on one rect and is the
// kind of thing a transcription gets subtly wrong.
func TestFlameSplit(t *testing.T) {
	// A long reach: taller than DeadlyFlameHeight, so the column splits.
	h := oneRoomHouse(Taper)
	h.Rooms[0].Objects[0].Data[4] = 0
	h.Rooms[0].Objects[0].Data[5] = 100 // distance
	w := newTestWorld(h, "", "")
	composeRoom(w, 0)

	if len(w.R.Hot) != 3 {
		t.Fatalf("a tall taper made %d rects, want 3 (lift, burn, dissolve)", len(w.R.Hot))
	}
	lift, burn := w.R.Hot[0], w.R.Hot[1]
	if lift.Action != LiftIt || burn.Action != BurnIt || w.R.Hot[2].Action != DissolveIt {
		t.Errorf("actions %s/%s/%s, want kLiftIt/kBurnIt/kDissolveIt",
			ActionName(lift.Action), ActionName(burn.Action), ActionName(w.R.Hot[2].Action))
	}
	// The burn rect is 22 pixels tall, not 24: its top is bottom - 24 + 2.
	if got := burn.Bounds.Tall(); got != DeadlyFlameHeight-2 {
		t.Errorf("burn rect is %d tall, want %d", got, DeadlyFlameHeight-2)
	}
	// The lift rect ends where the burn rect's nominal 24-pixel band begins, so the
	// two overlap by exactly 2.
	if lift.Bounds.Bottom != burn.Bounds.Top-2 {
		t.Errorf("lift bottom %d, burn top %d: want a 2px overlap",
			lift.Bounds.Bottom, burn.Bounds.Top)
	}

	// A short reach: no lift at all.
	h2 := oneRoomHouse(Taper)
	h2.Rooms[0].Objects[0].Data[5] = 10
	w2 := newTestWorld(h2, "", "")
	composeRoom(w2, 0)
	if len(w2.R.Hot) != 2 {
		t.Fatalf("a short taper made %d rects, want 2 (burn, dissolve)", len(w2.R.Hot))
	}
	if w2.R.Hot[0].Action != BurnIt {
		t.Errorf("short taper's first rect is %s, want kBurnIt",
			ActionName(w2.R.Hot[0].Action))
	}
}

// TestUnlinkedTransportsAreInert pins the who != 255 gate: an unwired mailbox, duct
// or transporter makes no rect at all.
func TestUnlinkedTransportsAreInert(t *testing.T) {
	for _, what := range []int16{MailboxLf, MailboxRt, FloorTrans, CeilingTrans,
		InvisTrans, DeluxeTrans} {
		h := oneRoomHouse(what)
		h.Rooms[0].Objects[0].Data[8] = house.UnlinkedWho
		w := newTestWorld(h, "", "")
		composeRoom(w, 0)
		if len(w.R.Hot) != 0 {
			t.Errorf("unlinked %s made %d rects, want 0",
				house.ObjectName(what), len(w.R.Hot))
		}

		h.Rooms[0].Objects[0].Data[8] = 3
		w.ListAllLocalObjects()
		if len(w.R.Hot) == 0 {
			t.Errorf("linked %s made no rects", house.ObjectName(what))
		}
	}
}

// TestUnwiredSwitchesAreInert is the same gate on the switch family, where the
// sentinel is -1 in a short rather than 255 in a byte.
func TestUnwiredSwitchesAreInert(t *testing.T) {
	for _, what := range []int16{LightSwitch, MachineSwitch, Thermostat,
		PowerSwitch, KnifeSwitch, InvisSwitch, Trigger, LgTrigger} {
		h := oneRoomHouse(what)
		// where = -1, at offsets 6-7.
		h.Rooms[0].Objects[0].Data[6] = 0xFF
		h.Rooms[0].Objects[0].Data[7] = 0xFF
		w := newTestWorld(h, "", "")
		composeRoom(w, 0)
		if len(w.R.Hot) != 0 {
			t.Errorf("unwired %s made %d rects, want 0",
				house.ObjectName(what), len(w.R.Hot))
		}

		h.Rooms[0].Objects[0].Data[6] = 0
		h.Rooms[0].Objects[0].Data[7] = 1
		w.ListAllLocalObjects()
		if len(w.R.Hot) != 1 {
			t.Errorf("wired %s made %d rects, want 1",
				house.ObjectName(what), len(w.R.Hot))
			continue
		}
		want := SwitchIt
		if what == Trigger || what == LgTrigger {
			want = TriggerIt
		}
		if got := w.R.Hot[0].Action; got != want {
			t.Errorf("%s: action %s, want %s", house.ObjectName(what),
				ActionName(got), ActionName(want))
		}
	}
}

// TestInvisBlowerVectorGate pins the nibble switch with no default: a vector of 0, or
// one with several bits set, makes no rect.
func TestInvisBlowerVectorGate(t *testing.T) {
	cases := []struct {
		vector byte
		action int16
		rects  int
	}{
		{1, LiftIt, 1},
		{2, PushItRight, 1},
		{4, DropIt, 1},
		{8, PushItLeft, 1},
		{0, 0, 0},
		{3, 0, 0},         // two bits: no case matches
		{15, 0, 0},        // all four
		{0x11, LiftIt, 1}, // the high nibble is masked off
	}
	for _, c := range cases {
		h := oneRoomHouse(InvisBlower)
		h.Rooms[0].Objects[0].Data[8] = c.vector // blowerType.vector
		w := newTestWorld(h, "", "")
		composeRoom(w, 0)

		if len(w.R.Hot) != c.rects {
			t.Errorf("vector %#02x: %d rects, want %d", c.vector, len(w.R.Hot), c.rects)
			continue
		}
		if c.rects == 1 && w.R.Hot[0].Action != c.action {
			t.Errorf("vector %#02x: action %s, want %s", c.vector,
				ActionName(w.R.Hot[0].Action), ActionName(c.action))
		}
	}
}

// TestScrutinizedActions pins the five actions that use the tight hit box. It is a
// closed set in the original and getting it wrong changes how hard every hazard in
// the game is to touch.
func TestScrutinizedActions(t *testing.T) {
	want := map[int16]bool{
		DissolveIt: true, BounceIt: true, ShredIt: true, MicrowaveIt: true, WebIt: true,
	}
	seen := map[int16]bool{}

	for what := int16(0x01); what <= 0x8F; what++ {
		if house.ObjectName(what) == "" {
			continue
		}
		h := oneRoomHouse(what)
		obj := &h.Rooms[0].Objects[0]
		obj.Data[offBonusState] = 1 // keep prizes valid
		obj.Data[8] = 3             // link byte, for the who != 255 gates
		obj.Data[6], obj.Data[7] = 0, 1
		if what == InvisBlower || what == LiftArea {
			obj.Data[8] = 1 // a vector the switch handles
		}

		w := newTestWorld(h, "", "")
		composeRoom(w, 0)

		for _, hs := range w.R.Hot {
			seen[hs.Action] = seen[hs.Action] || hs.DoScrutinize
			if hs.DoScrutinize != want[hs.Action] {
				t.Errorf("%s: %s has DoScrutinize=%v, want %v",
					house.ObjectName(what), ActionName(hs.Action),
					hs.DoScrutinize, want[hs.Action])
			}
		}
	}
	for a := range want {
		if !seen[a] {
			t.Errorf("%s was never produced by any type, so its scrutinize flag is untested",
				ActionName(a))
		}
	}
}

// ---------------------------------------------------------------------------
// The corpus
// ---------------------------------------------------------------------------

// TestObjectGraphEveryRoom builds the object graph and the hot-spot table for every
// room of every shipped house and compares each house against a checked-in hash.
//
// The hash covers every field that later stages read: each master entry's seven
// indices, and each hot spot's bounds, action, owner, state and scrutinize flag. So a
// one-pixel change in any of the 117 CreateActiveRects cases moves it, and so does a
// change in the listing order that renumbers the master table.
//
// Like internal/render's corpus test this is a regression guard rather than a fidelity
// test -- the hashes are of our output, since there is no 1994 table to hash against.
// Fidelity comes from the transcription being checked case by case against the C. What
// the hash freezes is that it stays transcribed.
func TestObjectGraphEveryRoom(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	artDir := filepath.Join(assetRoot, "art")

	paths, err := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	var lines []string
	totalRooms, totalHot, totalMaster := 0, 0, 0

	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		h, err := house.LoadFile(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		forkDir := filepath.Join(assetRoot, "houseart", name)
		if _, err := os.Stat(forkDir); err != nil {
			forkDir = ""
		}
		w := newTestWorld(h, artDir, forkDir)

		sum := sha256.New()
		hot, master := 0, 0
		for r := int16(0); r < w.NumberRooms(); r++ {
			composeRoom(w, r)
			w.DetermineRoomOpenings()

			hash16(sum, r, int16(len(w.R.Master)), int16(w.R.NumLocalMaster),
				int16(len(w.R.Hot)), int16(w.R.NumChimes),
				w.R.LeftThresh, w.R.RightThresh,
				b16(w.R.LeftOpen), b16(w.R.RightOpen),
				b16(w.R.TopOpen), b16(w.R.BottomOpen))

			for _, m := range w.R.Master {
				hash16(sum, m.RoomNum, m.ObjectNum, m.RoomLink, m.ObjectLink,
					m.LocalLink, m.HotNum, m.DynaNum, m.TheObject.What)
			}
			for _, hs := range w.R.Hot {
				hash16(sum, hs.Bounds.Top, hs.Bounds.Left, hs.Bounds.Bottom,
					hs.Bounds.Right, hs.Action, hs.Who,
					b16(hs.IsOn), b16(hs.DoScrutinize))
			}
			hot += len(w.R.Hot)
			master += len(w.R.Master)
			totalRooms++
		}
		totalHot += hot
		totalMaster += master
		lines = append(lines, fmt.Sprintf("%-28s %4d rooms %6d master %5d hot  %s",
			name, w.NumberRooms(), master, hot,
			hex.EncodeToString(sum.Sum(nil)[:16])))
	}

	lines = append(lines, fmt.Sprintf("%-28s %4d rooms %6d master %5d hot",
		"TOTAL", totalRooms, totalMaster, totalHot))
	got := strings.Join(lines, "\n") + "\n"

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", goldenFile)
		return
	}
	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("%v (run `go test ./internal/game -update`)", err)
	}
	if got != string(want) {
		t.Errorf("the object graph changed. Diff:\n%s", firstDiff(string(want), got))
	}
}

// TestHotSpotTypeCoverage reports which of the hot-spot-producing types the shipped
// houses actually instantiate, and fails if the corpus stops covering one it used to.
//
// The point is to know what the corpus test above is and is not exercising. A type no
// shipped house uses is transcribed against the C and nothing else, and that is worth
// knowing per type rather than in aggregate.
func TestHotSpotTypeCoverage(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	artDir := filepath.Join(assetRoot, "art")

	paths, _ := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	produced := map[int16]int{} // object type -> rects seen
	actions := map[int16]int{}  // action -> rects seen

	for _, path := range paths {
		h, err := house.LoadFile(path)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(path), ".house")
		forkDir := filepath.Join(assetRoot, "houseart", name)
		if _, err := os.Stat(forkDir); err != nil {
			forkDir = ""
		}
		w := newTestWorld(h, artDir, forkDir)
		for r := int16(0); r < w.NumberRooms(); r++ {
			composeRoom(w, r)
			for _, hs := range w.R.Hot {
				actions[hs.Action]++
				if int(hs.Who) < len(w.R.Master) {
					produced[w.R.Master[hs.Who].TheObject.What]++
				}
			}
		}
	}

	// The 93 types that can produce a rect, from the case analysis in hotspots.go.
	var uncovered []string
	for _, what := range producingTypes() {
		if produced[what] == 0 {
			uncovered = append(uncovered, house.ObjectName(what))
		}
	}
	sort.Strings(uncovered)
	t.Logf("%d of %d hot-spot-producing types instantiated by the shipped houses",
		len(producingTypes())-len(uncovered), len(producingTypes()))
	if len(uncovered) > 0 {
		t.Logf("never instantiated, so transcribed against the C alone: %s",
			strings.Join(uncovered, " "))
	}

	var unusedActions []string
	for a := int16(0); a < NumHotSpotActions; a++ {
		if actions[a] == 0 {
			unusedActions = append(unusedActions, ActionName(a))
		}
	}
	if len(unusedActions) > 0 {
		t.Logf("actions no shipped room produces: %s", strings.Join(unusedActions, " "))
	}

	// A floor is asserted rather than logged, so that a regression which stopped
	// producing rects entirely would fail rather than quietly log a shorter list.
	if len(produced) < 60 {
		t.Errorf("only %d types produced a rect across all 22 houses; "+
			"something has stopped composing", len(produced))
	}
}

// producingTypes is the 93 types with at least one AddActiveRect in their
// CreateActiveRects case. The 24 that produce none are the eight lights, kCustomPict,
// kSparkle and the fourteen plain clutter types.
func producingTypes() []int16 {
	silent := map[int16]bool{
		CeilingLight: true, LightBulb: true, TableLamp: true, HipLamp: true,
		DecoLamp: true, Flourescent: true, TrackLight: true, InvisLight: true,
		CustomPict: true, Sparkle: true,
		Ozma: true, Mirror: true, Mousehole: true, Fireplace: true, Flower: true,
		WallWindow: true, Bear: true, Calendar: true, Vase1: true, Vase2: true,
		Bulletin: true, Cloud: true, Faucet: true, Rug: true,
	}
	var out []int16
	for what := int16(0x01); what <= 0x8F; what++ {
		if house.ObjectName(what) == "" || silent[what] {
			continue
		}
		out = append(out, what)
	}
	return out
}

// TestProducingTypeCountIs93 pins the split from hotspots.go's header, so that adding
// a case without updating the count is caught.
func TestProducingTypeCountIs93(t *testing.T) {
	if n := len(producingTypes()); n != 93 {
		t.Errorf("%d producing types, want 93", n)
	}
}

// TestListAllLocalObjectsIsDeterministic runs the build twice on the same room and
// requires identical tables, since the corpus hash depends on it and the O(n^2) link
// pass keeps the last match rather than the first.
func TestListAllLocalObjectsIsDeterministic(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	path := filepath.Join(houseDir, "Demo House.house")
	h, err := house.LoadFile(path)
	if err != nil {
		t.Skipf("%v", err)
	}
	w := newTestWorld(h, filepath.Join(assetRoot, "art"), "")

	snap := func(r int16) string {
		composeRoom(w, r)
		sum := sha256.New()
		for _, m := range w.R.Master {
			hash16(sum, m.RoomNum, m.ObjectNum, m.RoomLink, m.ObjectLink,
				m.LocalLink, m.HotNum)
		}
		for _, hs := range w.R.Hot {
			hash16(sum, hs.Bounds.Top, hs.Bounds.Left, hs.Bounds.Bottom,
				hs.Bounds.Right, hs.Action, hs.Who)
		}
		return hex.EncodeToString(sum.Sum(nil))
	}
	for r := int16(0); r < w.NumberRooms(); r++ {
		a, b := snap(r), snap(r)
		if a != b {
			t.Fatalf("room %d: two builds disagree", r)
		}
	}
}

// TestTablesRespectTheirCaps checks the two hard caps over the whole corpus. Neither
// is expected to bind in a shipped house, and knowing that is the point: it is what
// makes the -1 return paths untested-by-the-corpus rather than untested-by-anything.
func TestTablesRespectTheirCaps(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	paths, _ := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	maxHot, maxMaster := 0, 0
	for _, path := range paths {
		h, err := house.LoadFile(path)
		if err != nil {
			continue
		}
		w := newTestWorld(h, filepath.Join(assetRoot, "art"), "")
		for r := int16(0); r < w.NumberRooms(); r++ {
			composeRoom(w, r)
			if len(w.R.Hot) > maxHot {
				maxHot = len(w.R.Hot)
			}
			if len(w.R.Master) > maxMaster {
				maxMaster = len(w.R.Master)
			}
			if len(w.R.Hot) > MaxHotSpots {
				t.Fatalf("room %d of %s: %d hot spots exceeds the cap of %d",
					r, path, len(w.R.Hot), MaxHotSpots)
			}
			if len(w.R.Master) > MaxMasterObjects {
				t.Fatalf("room %d of %s: %d master entries exceeds the cap of %d",
					r, path, len(w.R.Master), MaxMasterObjects)
			}
		}
	}
	t.Logf("busiest room in the corpus: %d/%d hot spots, %d/%d master entries",
		maxHot, MaxHotSpots, maxMaster, MaxMasterObjects)
}

// TestHotSpotTableOverflowReturnsMinusOne exercises the cap that no shipped house
// reaches, since it is the only way that path is covered.
func TestHotSpotTableOverflowReturnsMinusOne(t *testing.T) {
	h := oneRoomHouse(Table)
	// 24 slots each making one rect is 24; three rects each would be 72. Fill the
	// room with tall tapers, which make three apiece.
	for i := 0; i < MaxRoomObs; i++ {
		h.Rooms[0].Objects[i] = house.Object{What: Taper}
		h.Rooms[0].Objects[i].Data[5] = 100 // a reach that splits
	}
	h.Rooms[0].NumObjects = MaxRoomObs

	w := newTestWorld(h, "", "")
	composeRoom(w, 0)

	if len(w.R.Hot) != MaxHotSpots {
		t.Errorf("%d hot spots, want exactly the cap of %d", len(w.R.Hot), MaxHotSpots)
	}
	sawMinusOne := false
	for _, m := range w.R.Master {
		if m.HotNum == -1 {
			sawMinusOne = true
		}
	}
	if !sawMinusOne {
		t.Error("no object reported HotNum -1, so the overflow path did not run")
	}
	// And SetObjectState on an object with no hot spot must not panic -- this is the
	// unguarded hotSpots[hotNum] in the appliance family.
	for i := int16(0); i < MaxRoomObs; i++ {
		w.SetObjectState(0, i, Toggle, i)
	}
}

// TestNumNeighborsChangesTheGraph pins the effect of the window-size preference on the
// object graph, which is not obvious from either end.
//
// numNeighbors is 1, 3 or 9 (Settings.c:888 defaults it to 9, and Main.c:191 forces it
// to 1 on a 512-pixel screen). It selects how many rooms ListAllLocalObjects walks, so
// it changes the *size* of masterObjects -- and therefore whether a link into a
// neighbouring room resolves. On a small screen, a switch wired next door gets
// LocalLink = -1 and cannot update the cached copy of what it switches.
//
// The hot-spot table is unaffected, because only the central room makes rects.
func TestNumNeighborsChangesTheGraph(t *testing.T) {
	houseDir := requireAssets(t, "houses")
	paths, _ := filepath.Glob(filepath.Join(houseDir, "*.house"))
	if len(paths) == 0 {
		t.Skipf("no houses in %s", houseDir)
	}
	sort.Strings(paths)

	// One mid-sized house with plenty of cross-room links.
	h, err := house.LoadFile(filepath.Join(houseDir, "CD Demo House.house"))
	if err != nil {
		t.Skipf("%v", err)
	}

	type tally struct{ master, hot, resolved int }
	got := map[int]tally{}
	for _, n := range []int{1, 3, 9} {
		w := newTestWorld(h, "", "")
		w.R.NumNeighbors = n
		var tl tally
		for r := int16(0); r < w.NumberRooms(); r++ {
			composeRoom(w, r)
			tl.master += len(w.R.Master)
			tl.hot += len(w.R.Hot)
			for _, m := range w.R.Master {
				if m.RoomLink != -1 && m.ObjectLink != -1 && m.LocalLink != -1 {
					tl.resolved++
				}
			}
		}
		got[n] = tl
		t.Logf("numNeighbors %d: %d master entries, %d hot spots, %d links resolved",
			n, tl.master, tl.hot, tl.resolved)
	}

	// A one-room locale lists exactly 24 slots per room; three rooms list at most 72.
	if want := len(h.Rooms) * MaxRoomObs; got[1].master != want {
		t.Errorf("numNeighbors 1: %d master entries, want %d (24 per room)",
			got[1].master, want)
	}
	if !(got[1].master < got[3].master && got[3].master < got[9].master) {
		t.Errorf("the table should grow with the window: %d, %d, %d",
			got[1].master, got[3].master, got[9].master)
	}
	if got[1].hot != got[9].hot || got[3].hot != got[9].hot {
		t.Errorf("hot spots differ by window size (%d, %d, %d) but only the central "+
			"room makes rects", got[1].hot, got[3].hot, got[9].hot)
	}
	if !(got[1].resolved < got[9].resolved) {
		t.Errorf("a wider window must resolve more links: %d at 1, %d at 9",
			got[1].resolved, got[9].resolved)
	}
}

// TestSwitchingALightRecountsTheRoom pins the game-side half of the lighting engine:
// SetObjectState writes a lamp's state and RedrawRoomLighting is what turns that into a
// dark room -- and only on the zero-to-one boundary.
//
// A room with two lamps stays lit when the first is switched off, so the recount runs
// and the recompose does not. That asymmetry is the whole point of RedrawRoomLighting's
// wasLit/isLit test, and it is what makes a two-lamp room feel unswitchable.
func TestSwitchingALightRecountsTheRoom(t *testing.T) {
	h := oneRoomHouse(TableLamp)
	h.Rooms[0].Objects[0].Data[offLightState] = 1
	h.Rooms[0].Objects[1] = h.Rooms[0].Objects[0]
	h.Rooms[0].NumObjects = 2

	w := newTestWorld(h, "", "")
	composeRoom(w, 0)
	w.R.NumLights = w.R.GetNumberOfLights(0)
	if w.R.NumLights != 2 {
		t.Fatalf("two lamps counted as %d", w.R.NumLights)
	}

	// First lamp off: the count drops and the room is still lit, so nothing recomposes.
	if !w.SetObjectState(0, 0, ForceOff, 0) {
		t.Fatal("switching a lit lamp off should report a change")
	}
	w.RedrawRoomLighting()
	if w.R.NumLights != 1 {
		t.Errorf("after one lamp off, NumLights=%d, want 1", w.R.NumLights)
	}

	// Second lamp off: the count reaches zero, which is the transition.
	if !w.SetObjectState(0, 1, ForceOff, 1) {
		t.Fatal("switching the second lamp off should report a change")
	}
	w.RedrawRoomLighting()
	if w.R.NumLights != 0 {
		t.Errorf("after both lamps off, NumLights=%d, want 0", w.R.NumLights)
	}

	// And back on.
	w.SetObjectState(0, 0, ForceOn, 0)
	w.RedrawRoomLighting()
	if w.R.NumLights != 1 {
		t.Errorf("after one lamp back on, NumLights=%d, want 1", w.R.NumLights)
	}
}

// TestLightsMakeNoHotSpot pins the one family with no rect: a lamp is switched by
// something else's hot spot, never by its own.
func TestLightsMakeNoHotSpot(t *testing.T) {
	for _, what := range []int16{CeilingLight, LightBulb, TableLamp, HipLamp,
		DecoLamp, Flourescent, TrackLight, InvisLight} {
		h := oneRoomHouse(what)
		h.Rooms[0].Objects[0].Data[offLightState] = 1
		w := newTestWorld(h, "", "")
		composeRoom(w, 0)

		if len(w.R.Hot) != 0 {
			t.Errorf("%s made %d hot spots, want 0", house.ObjectName(what), len(w.R.Hot))
		}
		if w.R.Master[0].HotNum != -1 {
			t.Errorf("%s: HotNum=%d, want -1", house.ObjectName(what), w.R.Master[0].HotNum)
		}
		// But it is still counted, which is the only thing a light does.
		if got := w.R.GetNumberOfLights(0); got != 1 {
			t.Errorf("%s: GetNumberOfLights=%d, want 1", house.ObjectName(what), got)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// composeRoom is DrawLocale's head (RoomGraphics.c:44-70) with the drawing left out:
// re-arm the reserved sound slot, forget the chime count, resolve the nine neighbours
// and build the object graph.
//
// The corpus tests use this rather than Rebuild because the graph is what they measure
// and Rebuild would also need art for all nine rooms. It has to stay in step with
// Rebuild's own reset head, and the order matters: LocalNumbers must be resolved before
// ListAllLocalObjects, which walks it.
func composeRoom(w *World, room int16) {
	w.R.TriggerSoundHeld = false
	w.R.NumChimes = 0
	w.R.RoomNumber = room
	for i := 0; i < 9; i++ {
		w.R.LocalNumbers[i] = w.R.GetNeighborRoomNumber(i)
		w.R.IsStructure[i] = w.R.IsRoomAStructure(w.R.LocalNumbers[i])
	}
	w.ListAllLocalObjects()
}

// oneRoomHouse makes the smallest house that will compose: one room at floor 0,
// suite 0, holding one object of the given type at a position clear of the edges.
func oneRoomHouse(what int16) *house.House {
	h := &house.House{
		Version:   0x0200,
		NRooms:    1,
		FirstRoom: 0,
		Rooms: []house.Room{{
			Background: SimpleRoom,
			Floor:      0,
			Suite:      0,
			NumObjects: 1,
		}},
	}
	for i := range h.Rooms[0].Objects {
		h.Rooms[0].Objects[i] = house.Object{What: house.ObjectIsEmpty}
	}
	if what != house.ObjectIsEmpty {
		obj := house.Object{What: what}
		// topLeft = (100, 100), which for the rect-shaped layouts is also the top
		// left of bounds. Offsets 0-1 are v and 2-3 are h -- vertical first.
		obj.Data[0], obj.Data[1] = 0, 100
		obj.Data[2], obj.Data[3] = 0, 100
		// A plausible non-zero size/reach for the layouts that carry one, at 4-5.
		obj.Data[4], obj.Data[5] = 0, 48
		h.Rooms[0].Objects[0] = obj
	}
	return h
}

// hash16 feeds a run of int16s to a hash in big-endian order, which is the order the
// original stored them in.
func hash16(sum interface{ Write([]byte) (int, error) }, vs ...int16) {
	var b [2]byte
	for _, v := range vs {
		binary.BigEndian.PutUint16(b[:], uint16(v))
		sum.Write(b[:])
	}
}

// b16 is a bool as the byte the C would have stored.
func b16(v bool) int16 {
	if v {
		return 1
	}
	return 0
}

// firstDiff reports the first differing line of two golden texts, which is more use
// in a failure message than either whole file.
func firstDiff(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var a, b string
		if i < len(wl) {
			a = wl[i]
		}
		if i < len(gl) {
			b = gl[i]
		}
		if a != b {
			return fmt.Sprintf("line %d:\n  want %s\n   got %s", i+1, a, b)
		}
	}
	return "(identical)"
}
