package replay_test

// The hostile-house soak (docs/IMPROVEMENTS.md 4.30).
//
// The fuzz targets in internal/house prove a damaged house loads, or is refused, without a panic.
// This is the other half: a damaged house that loads is then *played*, with the whole asset tree
// behind it, which is how a player meets one. Each play takes a shipped house, overwrites a few
// bytes of one room, starts the glider there, and flies 200 frames with the keys changing.
//
// A play fails on a panic, on a run that does not come back, and on any error Run returns. That
// last one is the reason the soak exists. A sticky asset error is what a wrong name in a table
// looks like, and one of those went unseen for as long as nothing that loads art had shredded a
// glider (2.73). A play does not fail for tripping a guard (Diag.Guarded): a guard is the
// designed, reported answer to a house that makes no sense, and a mutator that is any good trips
// them.
//
// Every play is its own subtest and is decided by its number alone, so a failure is reproduced by
// name: `go test ./internal/replay -run 'TestADamagedHouseStillPlays/play017'`. The default is a
// few dozen plays, about a second; -soak runs more, and `make fuzz` asks for a few thousand.

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/internal/game/player"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/replay"
)

var soakPlays = flag.Int("soak", 32, "how many plays TestADamagedHouseStillPlays makes")

// soakFrames is how long each play flies, and soakWait how long it may take. 200 frames is about
// 20 ms of work, so a play that has not returned in soakWait has hung.
const (
	soakFrames = 200
	soakWait   = 30 * time.Second
)

// A damage is one change a play made to its house, reported so a failure says what it was.
type damage struct {
	at       int // offset in the file
	was, now []byte
}

func (d damage) String() string { return fmt.Sprintf("@%d %x->%x", d.at, d.was, d.now) }

// extremes are the 16-bit values a damaged field is most likely to have and least likely to have
// been tested with: the ends of a short, and the two values either side of zero.
var extremes = []uint16{0, 1, 0x7FFF, 0x8000, 0xFFFF}

// damageRoom overwrites a few bytes of room r of the file b, in place, and says which. The name
// is left alone, since nothing the game does with a name is a hazard, and one change in four goes
// to a room the composer draws beside it instead.
func damageRoom(rng *rand.Rand, b []byte, r, rooms int) []damage {
	var out []damage
	for range 1 + rng.IntN(12) {
		room := r
		if rng.IntN(4) == 0 {
			room = rng.IntN(rooms)
		}
		const skipName = 28
		at := house.SizeofHouseHeader + room*house.SizeofRoom + skipName +
			rng.IntN(house.SizeofRoom-skipName)
		d := damage{at: at}
		if rng.IntN(5) < 2 {
			at &^= 1 // every field is aligned to two bytes
			if at+2 > len(b) {
				continue
			}
			v := extremes[rng.IntN(len(extremes))]
			d = damage{at: at, was: append([]byte(nil), b[at:at+2]...), now: []byte{byte(v >> 8), byte(v)}}
		} else {
			d.was, d.now = []byte{b[at]}, []byte{byte(rng.Uint32())}
		}
		copy(b[d.at:], d.now)
		out = append(out, d)
	}
	return out
}

// soakKeys is a keystroke timeline that changes every few frames. Pause, delete and command are
// left out: the first stops the simulation and the other two end the game or ask to.
func soakKeys(rng *rand.Rand) []replay.Hold {
	var holds []replay.Hold
	for f := int64(0); f < soakFrames; f += 5 + int64(rng.IntN(25)) {
		n := rng.Uint32()
		holds = append(holds, replay.Hold{Frame: f, P1: player.Keys{
			Left: n&1 != 0, Right: n&2 != 0, Batt: n&4 != 0, Band: n&8 != 0}})
	}
	return holds
}

func TestADamagedHouseStillPlays(t *testing.T) {
	requireAssets(t, "houses")
	requireAssets(t, "art")
	names, err := filepath.Glob(filepath.Join(assetRoot, "houses", "*.house"))
	if err != nil || len(names) == 0 {
		t.Fatalf("no shipped houses to damage (%v)", err)
	}
	guarded := 0
	for play := range *soakPlays {
		t.Run(fmt.Sprintf("play%03d", play), func(t *testing.T) {
			rng := rand.New(rand.NewPCG(uint64(play), 0x6C69646572)) // "lider"
			name := names[rng.IntN(len(names))]
			stem := strings.TrimSuffix(filepath.Base(name), ".house")
			orig, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			h, err := house.Load(orig)
			if err != nil {
				t.Fatal(err)
			}
			var lived []int
			for i, r := range h.Rooms {
				if r.Suite != -1 {
					lived = append(lived, i)
				}
			}
			room := lived[rng.IntN(len(lived))]

			// Damage until Load takes it: a house Load refuses is FuzzLoad's business.
			var b []byte
			var hurt []damage
			for try := 0; ; try++ {
				if try == 50 {
					t.Skipf("%s room %d: 50 damaged copies and Load refused every one", stem, room)
				}
				b = append(b[:0], orig...)
				hurt = damageRoom(rng, b, room, len(h.Rooms))
				if _, err := house.Load(b); err == nil {
					break
				}
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, stem+".house"), b, 0o644); err != nil {
				t.Fatal(err)
			}

			s := replay.NewScript(stem, soakFrames)
			localAssets(t, s)
			s.HouseDir = dir // after localAssets, which points it at the extracted tree
			s.Sound, s.Music = false, false
			s.Seed = int32(rng.Uint32())
			s.Room = int16(room)
			if rng.IntN(2) == 0 {
				// Half the plays resume at a point in the room, which is the other way a
				// house is entered and the only one that puts the glider near the damage.
				s.Where = house.Point{H: int16(rng.IntN(512)), V: int16(rng.IntN(342))}
				s.Facing = byte(rng.IntN(2))
			}
			s.Input = soakKeys(rng)
			what := fmt.Sprintf("%s room %d, %v", stem, room, hurt)

			type outcome struct {
				res   *replay.Result
				err   error
				panic string
			}
			done := make(chan outcome, 1)
			go func() {
				defer func() {
					if p := recover(); p != nil {
						done <- outcome{panic: fmt.Sprintf("%v\n%s", p, debug.Stack())}
					}
				}()
				res, err := replay.Run(s)
				done <- outcome{res: res, err: err}
			}()
			select {
			case o := <-done:
				switch {
				case o.panic != "":
					t.Fatalf("%s: panicked: %s", what, o.panic)
				case o.err != nil:
					t.Fatalf("%s: %v", what, o.err)
				case o.res.Diag.Guarded > 0:
					guarded++
				}
			case <-time.After(soakWait):
				t.Fatalf("%s: still running after %v", what, soakWait)
			}
		})
	}
	t.Logf("%d plays, %d tripped a guard", *soakPlays, guarded)
}
