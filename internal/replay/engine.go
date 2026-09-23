package replay

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io/fs"
	"runtime"
	"sync"

	"github.com/bwenstar/gliderGo/internal/demo"
	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
)

// Engine is this build's engine fingerprint: a number two builds agree on when their simulations
// agree, and the one the race's handshake compares before either player flies
// (docs/IMPROVEMENTS.md 4.31). tree is the built-in asset tree, assets.Tree(), and has to be:
// what is being fingerprinted is the binary, and a directory somebody pointed -assets at is not
// part of it.
//
// **It is measured, not declared.** It is a hash of what this build's simulation does on a fixed
// set of runs, and nobody has to remember to bump it: a physics change that alters any of them
// alters it, and a change that alters none of them -- a picture, a sound, a screen, a comment --
// leaves it alone, so two releases with the same simulation go on racing each other. The
// alternative on file was a hand-bumped constant with a test tying it to the golden traces. It
// costs nothing at run time and is right only as long as the tests were run before the build
// that ships, and a build nobody tested is precisely the one a stranger races with.
//
// The runs are the configuration the race cannot vary and the player cannot choose: the built-in
// tree, seed 1, sound on, nine neighbours, none of the opt-in fixes, and no preferences at all.
// Sound is on because a build with sound composes rooms differently from one without (see
// Script.Sound); which of the two a player has is not a gate, because a kSoundIt hot spot only
// plays a sound, but the fingerprint has to pick one.
//
//   - **The 1994 demo, through Demo House.** 1117 recorded keystrokes flown by somebody who meant
//     to go somewhere, which is ten rooms, three deaths and 3,432 frames -- the one run here with
//     a player in it.
//   - **The duct run** from testdata/duct.script: a transition on frame 9, then 590 frames of a
//     glider held against a ceiling by a floor vent, which is the air-flow pass on every frame.
//   - **Right held down, from the start of each of the other 20 houses 1994 shipped with** (Empty
//     House has nothing to fly into), until the game is over or 900 frames have gone. The demo
//     flies one route, so a physics change anywhere off it would pass on the demo alone; these
//     fly the opening rooms of every house, and most of them end in a game over, which puts
//     three deaths and three restarts per house in the hash.
//
// Only the 1994 houses, because they are vendored and never edited, so a fingerprint built on
// them changes when the engine does and at no other time. The port's own houses are the port's
// to change, and the house hash is already their gate.
//
// 16,596 frames: two seconds on one core and about half of one across eight, because the runs
// go side by side. The caller spends it on the goroutine that is waiting for the other player
// anyway.
//
// **Changing anything in this file changes every build's fingerprint from here on**, and so
// refuses races between this build and every release before it. That is the right cost for a
// change to what the fingerprint means and the wrong one for tidying it; see Sample.simulation
// for what is in the hash.
func Engine(tree fs.FS) (uint64, error) {
	if tree == nil {
		return 0, fmt.Errorf("replay: an engine fingerprint needs the built-in asset tree")
	}
	// From the tree and never from a disk: loadDemo tries the disk first, and a stray
	// res/demo/ in whatever directory the player started the game from is no part of the
	// binary.
	stream, err := demo.LoadFS(tree, demo.ShippedPath)
	if err != nil {
		return 0, fmt.Errorf("replay: engine fingerprint: %w", err)
	}

	// The runs share nothing -- each has its own World, its own assets and its own mixer, and
	// the packages under them keep no state between Worlds but tables their init functions
	// build -- so they run side by side. Most of a run is decoding the pictures of the rooms
	// it flies through, which is the same work whichever order it happens in; the hash is taken
	// afterwards, in the runs' order, so how they were scheduled is not in it.
	runs := engineRuns(tree, stream)
	parts := make([][]byte, len(runs))
	errs := make([]error, len(runs))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(runs)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				parts[i], errs[i] = engineRun(runs[i])
			}
		}()
	}
	for i := range runs {
		next <- i
	}
	close(next)
	wg.Wait()

	h := fnv.New64a()
	for i, part := range parts {
		if errs[i] != nil {
			return 0, fmt.Errorf("replay: engine fingerprint, %s: %w", runs[i].House, errs[i])
		}
		h.Write(part)
	}
	return h.Sum64(), nil
}

// engineRun plays one of Engine's runs and returns what of it goes into the hash.
//
// A run is its house, its length and its end, and then its frames. The length is what keeps two
// runs from running together: without it, a frame moved from the end of one run to the start of
// the next would hash the same.
func engineRun(s *Script) ([]byte, error) {
	res, err := Run(s)
	if err != nil {
		return nil, err
	}
	b := append([]byte(nil), s.House...)
	b = binary.BigEndian.AppendUint32(b, uint32(len(res.Samples)))
	b = append(b, byte(b2i(res.GameOver)))
	for _, sm := range res.Samples {
		b = sm.simulation(b)
	}
	return b, nil
}

// engineHouses are the 1994 houses Engine holds right in: all of them but Empty House.
var engineHouses = []string{
	"Art Museum", "CD Demo House", "California or Bust!", "Castle o' the Air", "Davis Station",
	"Demo House", "Fun House", "Grand Prix", "ImagineHouse PRO II", "In The Mirror",
	"Land of Illusion", "Leviathan", "Metropolis", "Nemo's Market", "Rainbow's End", "Sampler",
	"Slumberland", "SpacePods", "Teddy World", "The Asylum Pro", "Titanic",
}

// engineRuns are Engine's scripts, in the order they are hashed.
func engineRuns(tree fs.FS, stream demo.Stream) []*Script {
	fixed := func(name string, frames int) *Script {
		s := NewScript(name, frames)
		s.Tree = tree
		s.Seed = 1
		// Music off, which leaves the simulation alone (Script.Music) and saves mixing a
		// score that nothing here listens to.
		s.Music = false
		return s
	}

	d := fixed("Demo House", 3500) // the run ends itself at 3,432
	d.Demo = demo.ShippedPath
	d.demo = stream

	// testdata/duct.script, restated rather than read: testdata is not in the binary, and a
	// script edited for a test's sake should not change which races are allowed.
	duct := fixed("CD Demo House", 600)
	duct.Room = 4
	duct.Where = house.Point{H: 420, V: 20}

	runs := []*Script{d, duct}
	for _, name := range engineHouses {
		s := fixed(name, 900)
		s.Input = []Hold{{Frame: 0, P1: player.Keys{Right: true}}}
		runs = append(runs, s)
	}
	return runs
}

// simulation appends the part of a sample a race can feel, and none of what it only sees.
//
// In: the frame and its parity, the room, the glider's mode and where it is, the score, the
// gliders and stars left, and the random stream. Out: every render counter (w2m, b2w, rend,
// pend, the pendulum clock, guarded reads and dropped rects), because those move when a
// picture is drawn differently and a race does not; and the sounds, which are decisions the
// game made that change nothing it does afterwards. The random stream stays in although some of
// its draws are made while composing a room -- a flame's phase, a pendulum's start -- because
// the balloons and the toasters draw from the same stream afterwards, so a build that draws once
// more while drawing a candle flies a different house.
//
// The two-player fields are not here because Engine's runs are one-player, and so is a race.
func (s Sample) simulation(b []byte) []byte {
	be := binary.BigEndian
	b = be.AppendUint64(b, uint64(s.Frame))
	b = append(b, byte(b2i(s.Even)))
	b = be.AppendUint16(b, uint16(s.Room))
	b = be.AppendUint16(b, uint16(s.Mode))
	for _, v := range []int16{s.Dest.Top, s.Dest.Left, s.Dest.Bottom, s.Dest.Right} {
		b = be.AppendUint16(b, uint16(v))
	}
	b = be.AppendUint32(b, uint32(s.Score))
	b = be.AppendUint16(b, uint16(s.Mortals))
	b = be.AppendUint16(b, uint16(s.Stars))
	return be.AppendUint32(b, uint32(s.Rand))
}
