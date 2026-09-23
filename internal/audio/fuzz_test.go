package audio

// Hostile bytes, on purpose (docs/IMPROVEMENTS.md 4.30).
//
// A sound tree is the one asset tree the other fuzz targets do not reach, and its per-house half is
// the part a house author makes: a manifest row and a .pcm file for each of a house's own sounds.
// So FuzzHouseSounds puts both through LoadHouse and then plays what loaded. Plain `go test` runs
// its seeds; `make fuzz` runs the engine.

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

// houseManifest is a per-house manifest of one row, in the extractor's columns.
func houseManifest(id, frames, rate, status string) string {
	return "file\thouse\tid\tname\tframes\trate_hz\tloop_start\tloop_end\tbase_freq\tencode\tstatus\n" +
		strings.Join([]string{"a.pcm", "fuzz", id, "A-hem!", frames, rate, "0", "0", "60", "0x00", status}, "\t") + "\n"
}

// FuzzHouseSounds is a house's own sounds from anywhere: its manifest, and the one sample file the
// manifest can name.
//
// What must hold:
//   - LoadHouse loads the house's sounds or none of them, and says why when it is none;
//   - every sound it loads is in the trigger slot under its own ID, and has at least one frame;
//   - each one, played, holds its channel for exactly as many samples as its length and step say,
//     because a sample's length is part of the port's behaviour (channel.next), and then lets go.
//
// And nothing panics, loading or mixing. An empty file used to load, and the mixer's first
// sample read past its end.
func FuzzHouseSounds(f *testing.F) {
	pcm := []byte{0x80, 0xFF, 0x00, 0x80, 0x7F}
	f.Add(houseManifest("3000", "5", "22254.5455", "ok"), pcm)
	f.Add(houseManifest("3001", "5", "7418.1818", "ok"), pcm)
	f.Add(houseManifest("3002", "", "", "ok"), pcm)
	f.Add(houseManifest("3003", "5", "22254.5455", "mace6"), pcm)
	f.Add(houseManifest("3000", "0", "22254.5455", "ok"), []byte{})
	f.Add(houseManifest("68536", "5", "22254.5455", "ok"), pcm)
	f.Add(houseManifest("3000", "5", "+Inf", "ok"), pcm)
	f.Add(houseManifest("3000", "5", "NaN", "ok"), pcm)
	f.Add(houseManifest("3000", "5", "65535.9999", "ok"), pcm)
	f.Add(houseManifest("3000", "5", "1", "ok"), pcm)
	f.Add("", pcm)
	f.Fuzz(func(t *testing.T, manifest string, pcm []byte) {
		b := &Bank{Triggers: map[int16]*Sound{}, fsys: fstest.MapFS{
			"houses/manifest.tsv": &fstest.MapFile{Data: []byte(manifest)},
			"houses/a.pcm":        &fstest.MapFile{Data: pcm},
		}}
		if err := b.LoadHouse("fuzz"); err != nil {
			if len(b.Triggers) != 0 || len(b.Unreadable) != 0 {
				t.Fatalf("LoadHouse failed (%v) and kept %d sounds and %d unreadable ones",
					err, len(b.Triggers), len(b.Unreadable))
			}
			return
		}
		e := New(b)
		for id, s := range b.Triggers {
			what := fmt.Sprintf("sound %d (%d frames at %g Hz)", id, len(s.Data), s.RateHz)
			switch {
			case s.ID != id || s.Slot != TriggerSlot:
				t.Fatalf("%s: keyed %d, slot %d", what, s.ID, s.Slot)
			case len(s.Data) == 0:
				t.Fatalf("%s: loaded with no frames", what)
			}
			e.FlushTriggerSound()
			if !e.LoadTriggerSound(id) {
				t.Fatalf("%s: loaded, and LoadTriggerSound refuses it", what)
			}
			e.PlayPrioritySound(TriggerSlot, TriggerPriority)

			// start plays a step of zero or less at the Macintosh rate.
			step := s.Step
			if step <= 0 {
				step = FixedOne
			}
			long := (int64(len(s.Data))*FixedOne + step - 1) / step
			if long > 1<<16 {
				// A slow enough rate plays one byte for a very long time, as it did in
				// 1994. Mix a little of it, and leave the arithmetic to the fast ones.
				e.Mix(make([]int16, 1<<10))
				continue
			}
			if e.Mix(make([]int16, long-1)); !e.Busy() {
				t.Fatalf("%s: over after %d samples, and its length and step say %d", what, long-1, long)
			}
			if e.Mix(make([]int16, 1)); e.Busy() {
				t.Fatalf("%s: still playing after the %d samples its length and step say", what, long)
			}
		}
	})
}
