package audio

// The sample bank: Sound.c's LoadBufferSounds (:316-347), Music.c's LoadMusicSounds
// (:220-251) and Sound.c's LoadTriggerSound (:265-303), reading the extraction that
// tools/probe_snd.py wrote instead of a resource fork.
//
// ---------------------------------------------------------------------------
// What a 'snd ' resource is, and what is left of it here
// ---------------------------------------------------------------------------
//
// Every sound in Glider PRO is a format-1 'snd ' resource holding one standard sampled
// sound header: unsigned 8-bit mono PCM, and for every one of the application's own sounds a
// sample rate of 0x56EE8BA3 in Fixed 16.16, which is 22254.5454... Hz -- exactly 244800/11, the
// Macintosh's audio clock. **The houses' own trigger sounds are not all at that rate**, and
// Sound.Step is what the mixer does about it. The
// extractor strips the 20-byte header and writes the frames raw, so a `.pcm` file here is
// one byte per frame and its length in bytes *is* its length in frames. `LoadBufferSounds`
// does the same thing in the same way: `BlockMove((Ptr)(*theSound + 20L), ...)` with a size
// of `GetHandleSize(theSound) - 20L`. The port's bank is therefore the same bytes the 1994
// build held in memory, and the same total: 1.1 MB for the 63 effects and the 7 pieces of
// music, all of it resident, because the C loads every sound at launch and never pages one in.
//
// The extracted tree on disk is three times that -- 3.4 MB -- and the difference is the
// twenty-two houses' own trigger sounds, which the extractor pulls out of every house's fork at
// once. Only ever one of those is in memory: LoadTriggerSound owns a single slot and frees it on
// every room change. See LoadHouse.
//
// The tree arrives as an fs.FS rather than as a directory name, so one loader reads both the
// copy built into the executable and a directory somebody named on the command line. See
// assets/assets.go and internal/assetfs.
//
// ---------------------------------------------------------------------------
// The loop points are ignored, and the samples prove it is right to ignore them
// ---------------------------------------------------------------------------
//
// Each header carries a loopStart/loopEnd pair and the extractor transcribes them, but Glider
// PRO plays every sound with `bufferCmd`, which plays a buffer once through: the Sound Manager
// honours loop points only for the pitched commands (`freqDurationCmd` and friends). Exactly
// four of the 63 sounds carry a full-range `0 .. frames-1` loop, and every other one carries
// the degenerate `frames-2 .. frames-1` pair a sound editor writes when there is no loop. The
// four are the game's four *continuous* sounds -- and what they are is the argument:
//
//	slot  sound   frames  game frames  requested
//	 18   Thrust    5759     7.78       every 4th frame the battery key is held (Input.c:148)
//	 62   Hiss      2960     4.00       every 4th frame the helium key is held (Input.c:174)
//	 47   Sizzle    2240     3.03       every frame over a lit floor vent (Interactions.c:1365)
//	 26   Shred     1934     2.61       every frame of a glider being shredded (Player.c:1287)
//
// Read the last two columns together. Hiss is four game frames long *to four decimal places*
// -- 2960 samples is 4 x 740 exactly -- and the game asks for it every fourth frame, so one
// channel plays it back to back with no seam and no loop point needed. Sizzle is three frames
// long and is asked for every frame, so it occupies three channels and covers itself the same
// way. Thrust and Shred are cut slightly short of their cadence, so their copies overlap and
// thicken rather than butt together.
//
// These lengths are not accidents; they are a composer writing to a mixer with three channels
// and a 30 Hz retrigger, and they are why the loop points in the headers were never needed.
// They are also why looping here would be actively wrong: a looped Thrust would keep firing
// after the player let go of the key, and its channel's completion callback would never run, so
// its priority would never fall back to 0 and the channel would be lost for the session.
//
// bank_test.go's TestContinuousSoundsCoverTheirCadence checks the table above against the
// extracted samples, so the argument fails loudly if the assets ever disagree with it. See
// docs/IMPROVEMENTS.md 2.45.

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

// The three resource-ID bases, from the C's #defines.
//
// The port keeps the C's *slot* numbering rather than the resource IDs, because every
// sound in the game is named by slot: kHitWallSound is 0 and lives in 'snd ' 1000. That
// indirection is Sound.c's (`GetResource('snd ', i + kBaseBufferSoundID)`) and dropping it
// would mean renumbering all 64 constants in GliderDefines.h.
const (
	BaseSoundID = 1000 // kBaseBufferSoundID (Sound.c:14)
	BaseMusicID = 2000 // kBaseBufferMusicID (Music.c:15)

	// MaxSounds is kMaxSounds (Sound.c:15): 64 slots, of which LoadBufferSounds fills
	// the first 63 and the last is reserved for the house's own trigger sound.
	MaxSounds = 64

	// MaxMusic is kMaxMusic (Music.c:16): seven pieces of music, all resident.
	MaxMusic = 7

	// TriggerSlot is kMaxSounds-1, the one slot LoadTriggerSound owns. It is also the
	// value of kTriggerSound (GliderDefines.h:118), which is what the game passes to
	// PlayPrioritySound when a sound trigger fires -- the slot and the sound ID are the
	// same number, and that is not a coincidence but the whole mechanism.
	TriggerSlot = MaxSounds - 1
)

// RateNum and RateDen are the Macintosh sample rate as the exact fraction it is:
// 244800/11 Hz = 22254.5454... Hz, which the resource header carries as the Fixed value
// 0x56EE8BA3.
//
// Rate is what the port *labels* streams with, and it is the rounded integer because every
// audio API worth writing to takes an integer rate. The 0.4545 Hz difference is 35 parts per
// million, or 0.06 cents of pitch -- four hundred times smaller than the just-noticeable
// difference -- and accepting it is what lets the mixer resample nothing at all. See
// docs/IMPROVEMENTS.md 2.46 for the alternative and why it is not worth its complexity.
const (
	RateNum = 244800
	RateDen = 11
	Rate    = 22255
)

// Sound is one entry of theSoundData[]: a sample and the header fields worth keeping.
type Sound struct {
	// Slot is the index the game uses -- kShredSound, kTikSound and so on. For the
	// application's sounds it is ID-BaseSoundID; for a house's trigger sound it is
	// TriggerSlot, whatever the resource ID was.
	Slot int16

	// ID is the resource ID the fork held it under: 1000+ for the application's
	// effects, 2000+ for music, 3000+ for a house's custom trigger sounds.
	ID int16

	// Name is the resource's name -- "Wall Hit", "Cuckoo", "A-hem!". The application's
	// names are only ever seen by a person reading a report, but the *houses'* names are
	// content: a sound trigger in Art Museum names "Shhhh!" and the name is the only
	// description of what the author meant.
	Name string

	// Data is unsigned 8-bit PCM, one byte a frame, at RateHz. 128 is silence, which is why
	// the mixer subtracts it rather than sign-extending.
	Data []byte

	// RateHz is the header's sampleRate, and it is **not** the same for every sound. All 63
	// effects and all 7 pieces of music are at the Macintosh rate, but fourteen of the 58
	// extracted house sounds are not:
	//
	//	11127.2727  half the Macintosh rate      6  In The Mirror, Leviathan, SpacePods, Titanic
	//	 7418.1818  a third                      3  CD Demo House, Nemo's Market
	//	 5563.6364  a quarter                    1  Leviathan's "Ricochet"
	//	22255.0000  the rounded rate, verbatim   1  Grand Prix's "V8 startup"
	//	22050.0000  the CD rate                  1  Grand Prix's "Goodbye!"
	//	11127.5000  half, rounded                1  Rainbow's End's "Curly Woob-Woob"
	//	 9779.0000  a sound editor's own number  1  SpacePods' "Organ"
	//
	// The three exact fractions are a house author downsampling to save disk, which halved or
	// thirded the rate exactly; the four odd ones are whatever tool they happened to use. Either
	// way the number is the author's decision about how their sound should sound. See Step.
	RateHz float64

	// Step is how far the mixer advances through Data per output sample, in 16.16 fixed
	// point, so 1<<16 is "one sample per sample" and 1<<15 is a sound played at half the
	// output rate -- each byte held for two.
	//
	// This is drop-sample rate conversion, and it is what the original asked the Sound
	// Manager for: every channel is opened with `initNoInterp` (Sound.c:379, Music.c:287),
	// which means convert the header's rate to the hardware's by stepping the sample pointer
	// and taking whatever it lands on, with no interpolation between neighbours. So a port
	// that ignored RateHz would play CD Demo House's "Fly Buzz" -- 7418 Hz -- three times too
	// fast, and a house author's fly would come out a mosquito.
	//
	// It is computed against the exact Macintosh rate rather than the rounded output Rate, so
	// every sound recorded at 22254.5455 Hz gets exactly 1<<16 and the whole application bank
	// is copied rather than resampled. See RateNum and the note on Rate.
	Step int64

	// LoopStart and LoopEnd are transcribed from the header and never read. See the file
	// comment for why that is faithful rather than lazy.
	LoopStart, LoopEnd int32
}

// FixedOne is one output sample in Step's 16.16 fixed point.
const FixedOne = 1 << 16

// stepFor converts a header sample rate into Step.
//
// Rounded, not truncated, and that is the whole reason this is a function: the manifest carries
// the rate as four decimal places of a Fixed 16.16 value, so the Macintosh rate arrives as
// 22254.5455 or 22254.5454 depending on which way the extractor's printf went. Both must land on
// exactly 1<<16 or the application's bank would resample by a part in ten million -- inaudible,
// but enough to make the mixer's inner loop do fixed-point arithmetic on every sound in the game
// and enough to break the byte-for-byte equality a recorded replay depends on.
//
// A rate of zero -- a manifest with no rate column, or a Sound built by hand in a test -- is the
// Macintosh rate, because that is what every sound the game shipped with is.
//
// The conversion around the product is deliberate and must stay. Written `x*FixedOne + 0.5` the
// compiler may contract the multiply and the add into one FMA -- the Go spec allows it, arm64
// codegen takes it -- and an FMA rounds once where the two operations round twice. That is a
// last-bit difference in a value this function then truncates, so a rate sitting on a boundary
// would step by one 65536th more on an Apple Silicon Mac than on an x86 one, and every mixed
// sample after it would differ. An explicit float64 conversion forces the intermediate rounding
// and forbids the fusion. See docs/IMPROVEMENTS.md 4.9.
func stepFor(rateHz float64) int64 {
	if rateHz <= 0 {
		return FixedOne
	}
	return int64(float64(rateHz*RateDen/RateNum*FixedOne) + 0.5)
}

// Frames is the sample's length. It is the byte length, the sound being 8-bit mono, and it
// is named for the header field (`numFrames`) rather than for the slice so that the arithmetic
// in the mixer reads like the Sound Manager's.
func (s *Sound) Frames() int { return len(s.Data) }

// Seconds is how long the sample takes to play, at its own rate.
//
// Its own and not the Macintosh's, because the two differ for thirteen of the houses' trigger
// sounds and the number wanted is always the audible duration: Grand Prix's "V8 startup" is
// 198,239 frames at 22255 Hz, which is nine seconds of engine and not nine seconds of anything
// else.
//
// Only ever used in reports and in tests, but it is the number that makes a priority argument
// concrete: Shred is 1934 frames, which is 87 milliseconds -- two and a half game frames -- and
// the game asks for it again on every frame of the shredding, so it holds all three channels at
// priority 903 for as long as the glider is coming apart and everything quieter is dropped for
// the duration.
func (s *Sound) Seconds() float64 {
	if s.RateHz > 0 {
		return float64(len(s.Data)) / s.RateHz
	}
	return float64(len(s.Data)) * RateDen / RateNum
}

// Bank is every sample the game can play: the application's 63 effects, its 7 pieces of
// music, and the current house's custom trigger sounds.
//
// It is loaded once and never mutated by the mixer, which is what lets the live path mix on
// one goroutine while the game requests sounds on another without copying samples around.
type Bank struct {
	// Effects is theSoundData[]. Index by slot. Entry TriggerSlot is always nil here --
	// the trigger sample lives on the Engine, because it is loaded and freed per room and
	// the bank is per session.
	Effects [MaxSounds]*Sound

	// Music is theMusicData[]. Index by piece, 0..6, which is what the score tables in
	// internal/game/music.go hold.
	Music [MaxMusic]*Sound

	// Triggers is the open house's 'snd ' resources by ID, the pool LoadTriggerSound
	// draws from. Empty until LoadHouse, and empty for a house that has no custom
	// sounds -- which is most of them, and which makes every sound trigger in such a
	// house fail to get a hot spot at all. See game/hotspots.go's loadTriggerSound.
	Triggers map[int16]*Sound

	// House is whose Triggers those are, for reports.
	House string

	fsys fs.FS
}

// LoadBank reads the extracted sound tree: manifest.tsv and the .pcm files beside it.
//
// It is InitSound's LoadBufferSounds plus InitMusic's LoadMusicSounds, and it has their
// failure behaviour: **any missing sample is an error for the whole bank**. The C returns
// MemError() from the loop and InitSound answers a non-zero error by setting `failedSound`,
// which silences the entire game (Sound.c:458-463) -- it does not play the sounds it managed
// to load. A half-loaded bank is not a state the original has, so it is not one the port
// invents; a caller that wants to play anyway wants a silent Engine, which is what nil gives
// it.
func LoadBank(fsys fs.FS) (*Bank, error) {
	if fsys == nil {
		return nil, errors.New("audio: no sound tree")
	}
	b := &Bank{Triggers: map[int16]*Sound{}, fsys: fsys}

	rows, err := readManifest(fsys, "manifest.tsv")
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		snd, err := b.load(".", r)
		if err != nil {
			return nil, err
		}
		switch {
		case snd.ID >= BaseMusicID && snd.ID < BaseMusicID+MaxMusic:
			snd.Slot = snd.ID - BaseMusicID
			b.Music[snd.Slot] = snd
		case snd.ID >= BaseSoundID && snd.ID < BaseSoundID+TriggerSlot:
			snd.Slot = snd.ID - BaseSoundID
			b.Effects[snd.Slot] = snd
		default:
			return nil, fmt.Errorf("audio: %s: resource id %d is neither an effect (%d..%d) nor music (%d..%d)",
				r["file"], snd.ID, BaseSoundID, BaseSoundID+TriggerSlot-1, BaseMusicID, BaseMusicID+MaxMusic-1)
		}
	}

	// The two completeness checks the C makes by returning early from its loops. They are
	// separate messages because they mean different things to whoever hit them: a missing
	// effect is a broken extraction, and a missing piece of music is usually a bank built
	// from a Glider PRO Lite fork, which shipped without the score.
	for i := int16(0); i < TriggerSlot; i++ {
		if b.Effects[i] == nil {
			return nil, fmt.Errorf("audio: no sound for slot %d ('snd ' %d)", i, BaseSoundID+i)
		}
	}
	for i := 0; i < MaxMusic; i++ {
		if b.Music[i] == nil {
			return nil, fmt.Errorf("audio: no music for piece %d ('snd ' %d)", i, BaseMusicID+i)
		}
	}
	return b, nil
}

// HasBank reports whether fsys looks like an extracted sound tree.
//
// It exists for the callers whose sound is on by *default* rather than by request -- the game
// and `glidertool replay` -- so that they can degrade to silence on a checkout where `make
// assets` has not been run instead of refusing to start. It is a cheap probe and not a
// validation: it says a manifest is there, and LoadBank still decides whether the tree behind
// it is complete. The distinction is the point. A caller that asked for sound and cannot have it
// gets LoadBank's error, which names the missing file; a caller that never asked gets silence.
func HasBank(fsys fs.FS) bool {
	if fsys == nil {
		return false
	}
	st, err := fs.Stat(fsys, "manifest.tsv")
	return err == nil && !st.IsDir()
}

// LoadHouse opens a house's custom trigger sounds, which is what HouseIO.c's resource-fork
// swap does for sounds: for as long as the house is open its 'snd ' resources are the ones
// GetResource finds first.
//
// A house with no sounds is not an error and neither is a missing directory. Nine of the
// twenty-two shipped houses have no custom sounds at all, and in five resources across four of
// the other thirteen the sound is present but MACE 6:1 compressed and so unreadable -- the
// extractor marks such a row `status` other than `ok` and this skips it, which lands in exactly
// the C's behaviour: GetResource returns nil, LoadTriggerSound returns -1, and the sound trigger
// silently gets no hot spot. Demo House's only sound is one of the five, so twelve houses arrive
// here with anything to load.
func (b *Bank) LoadHouse(name string) error {
	b.Triggers = map[int16]*Sound{}
	b.House = name

	rows, err := readManifest(b.fsys, "houses/manifest.tsv")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, r := range rows {
		if r["house"] != name || (r["status"] != "" && r["status"] != "ok") {
			continue
		}
		snd, err := b.load("houses", r)
		if err != nil {
			return err
		}
		snd.Slot = TriggerSlot
		b.Triggers[snd.ID] = snd
	}
	return nil
}

// Trigger is the GetResource('snd ', soundID) inside LoadTriggerSound: the house's sound of
// that ID, or nil if it has none.
func (b *Bank) Trigger(id int16) *Sound {
	if b == nil {
		return nil
	}
	return b.Triggers[id]
}

// Effect is theSoundData[slot], range-checked. Out of range answers nil rather than
// panicking, because the one caller that can pass an unvetted number is the trigger path and
// its bad number is a house's, not a programmer's.
func (b *Bank) Effect(slot int16) *Sound {
	if b == nil || slot < 0 || int(slot) >= MaxSounds {
		return nil
	}
	return b.Effects[slot]
}

// Piece is theMusicData[i], range-checked for the same reason: the score tables are indexed
// by a cursor the C lets go negative (Music.c:159, kProdGameScoreMode sets it to -1) and the
// port would rather answer silence than stop the game.
func (b *Bank) Piece(i int16) *Sound {
	if b == nil || i < 0 || int(i) >= MaxMusic {
		return nil
	}
	return b.Music[i]
}

// Bytes is what SoundBytesNeeded and MusicBytesNeeded together answer (Sound.c:491-512,
// Music.c:386-407): how much memory the bank costs. The original asked because it had to
// decide whether to load sounds at all on a 4 MB Mac; the port asks so that a report can say.
func (b *Bank) Bytes() int {
	n := 0
	for _, s := range b.Effects {
		if s != nil {
			n += len(s.Data)
		}
	}
	for _, s := range b.Music {
		if s != nil {
			n += len(s.Data)
		}
	}
	for _, s := range b.Triggers {
		n += len(s.Data)
	}
	return n
}

// load reads one manifest row's .pcm file.
//
// The row's `frames` is checked against the file's length rather than trusted, because the
// two coming apart is the one extraction bug this loader can detect: a truncated write leaves
// a plausible file whose header says otherwise, and a sound that is quietly 40 bytes short is
// a click nobody can trace.
func (b *Bank) load(dir string, r map[string]string) (*Sound, error) {
	name := r["file"]
	id, err := strconv.Atoi(r["id"])
	if err != nil {
		return nil, fmt.Errorf("audio: %s: bad id %q", name, r["id"])
	}
	data, err := fs.ReadFile(b.fsys, path.Join(dir, name))
	if err != nil {
		return nil, err
	}
	if want := atoiOr(r["frames"], -1); want >= 0 && want != len(data) {
		return nil, fmt.Errorf("audio: %s: manifest says %d frames, file holds %d", name, want, len(data))
	}
	rate, _ := strconv.ParseFloat(r["rate_hz"], 64)
	return &Sound{
		ID:        int16(id),
		Name:      r["name"],
		Data:      data,
		RateHz:    rate,
		Step:      stepFor(rate),
		LoopStart: int32(atoiOr(r["loop_start"], 0)),
		LoopEnd:   int32(atoiOr(r["loop_end"], 0)),
	}, nil
}

// readManifest reads one of the extractor's tab-separated manifests into rows keyed by the
// header's column names.
//
// By name and not by position, because the two manifests do not have the same columns -- the
// per-house one carries `house` and `status` and drops `rate_fixed` -- and because the
// extractor is free to add a column without breaking a loader that asks for what it needs.
func readManifest(fsys fs.FS, name string) ([]map[string]string, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var head []string
	var rows []map[string]string
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if head == nil {
			head = fields
			continue
		}
		row := make(map[string]string, len(head))
		for i, h := range head {
			if i < len(fields) {
				row[h] = fields[i]
			}
		}
		rows = append(rows, row)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if head == nil {
		return nil, fmt.Errorf("audio: %s: empty manifest", name)
	}
	return rows, nil
}

func atoiOr(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
