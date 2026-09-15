package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"glidergo/internal/audio"
	"glidergo/internal/replay"
)

// replayCmd runs the game headlessly from a script and prints what happened, one line per
// frame.
//
// This is the bug-report format. A released game gets reports like "sometimes the glider
// sticks in the third room", which is not reproducible by anybody; a script is a house, a
// seed and a keystroke log, and it reproduces the frame exactly on any machine, with no
// display, no sound and no timing dependency. `-trace` is the evidence and `-digest` is the
// one-line answer to "do we still agree".
//
// The flags shadow every script keyword and win over the file, so a script on disk can be
// re-run longer or from a different room without editing it -- which is what triage looks
// like in practice.
func replayCmd(args []string) error {
	fs := flag.NewFlagSet(prog+" replay", flag.ContinueOnError)
	var (
		houseName = fs.String("house", "", "house name or path (overrides the script)")
		houseDir  = fs.String("houses", "assets/extracted/houses", "directory of extracted houses")
		artDir    = fs.String("art", "assets/extracted/art", "extracted application art tree")
		houseArt  = fs.String("houseart", "assets/extracted/houseart", "extracted per-house resource forks")
		soundDir  = fs.String("sounds", "assets/extracted/sound", "extracted sound bank")
		sound     = fs.Bool("sound", true, "mix the sound; -sound=false replays in silence")
		music     = fs.Bool("music", true, "play the music (needs sound)")
		wav       = fs.String("wav", "", "write the mix to this WAV file")
		frames    = fs.Int("frames", 0, "frames to run (overrides the script)")
		seed      = fs.Int("seed", -1, "random seed (overrides the script)")
		neighbors = fs.Int("neighbors", 0, "compose 1, 3 or 9 rooms (overrides the script)")
		two       = fs.Bool("two", false, "two players")
		demoPath  = fs.String("demo", "", "replay this recorded input stream instead of the keyboard")
		room      = fs.Int("room", -1, "start room (overrides the script)")
		where     = fs.String("where", "", "start the glider at h,v in -room: a resume")
		trace     = fs.Bool("trace", false, "print the per-frame trace, not just the summary")
		digest    = fs.Bool("digest", false, "print only the digest, for scripting")
		writeBack = fs.String("script", "", "write the resolved script here (- for stdout) and exit")
		out       = fs.String("o", "", "write the output to this file instead of stdout")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `usage: %s replay [flags] [script]

Runs the game with no window, from a script, and reports what each frame did.
With no script file, the flags alone describe the run.

A script is line-oriented; %s replay -script - prints one to copy:

  house CD Demo House
  seed 1
  frames 600
  room 4                 # start here...
  where 423 20           # ...at this point, which resumes rather than starts
  at 0 right             # from frame 0, hold right
  at 45 -                # let go

Keys are left, right, batt, band, command, delete and pause, joined with commas;
- is none. An 'at' line takes player one's keys then optionally player two's.

The 1994 attract mode is a script too -- a recorded keystroke stream replayed
through the ordinary physics, which is the strictest determinism test there is:

  %s replay -house 'Demo House' -demo assets/extracted/res/demo/128.bin -frames 3500

'%s demo' describes such a stream without running it.

The sound is mixed even with nowhere to send it, so every run reports which
sounds were asked for and a digest of the samples. -wav keeps them, which on a
machine with no sound card is the only way to hear what a replay sounded like:

  %s replay -wav bug.wav -trace -o bug.txt bug.script

`, prog, prog, prog, prog, prog)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return fmt.Errorf("expected at most one script file")
	}

	// A script file is the base; the flags are an overlay. Without a file the base is
	// NewScript's defaults, so -house alone is a runnable command.
	s := replay.NewScript("", 600)
	if fs.NArg() == 1 {
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			return err
		}
		parsed, err := replay.Parse(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", fs.Arg(0), err)
		}
		s = parsed
	}

	// Only flags the caller actually set may override, which is why the numeric
	// defaults above are out-of-range sentinels rather than the real defaults: a script
	// that says `seed 7` must not be silently reset to 0 by a flag nobody typed.
	//
	// A bool has no spare value to use as a sentinel -- an unset -sound and -sound=false are
	// the same false -- so the three booleans ask the flag package which ones were typed. That
	// also makes the override work in both directions: -sound=false silences a script, -music
	// on its own re-enables the score in a script that turned it off, and -two=false runs a
	// two-player script with one glider.
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if *houseName != "" {
		s.House = *houseName
	}
	if *frames > 0 {
		s.Frames = *frames
	}
	if *seed >= 0 {
		s.Seed = int32(*seed)
	}
	if *neighbors != 0 {
		s.Neighbors = *neighbors
	}
	if set["two"] {
		s.TwoPlayer = *two
	}
	if set["sound"] {
		s.Sound = *sound
	}
	if set["music"] {
		s.Music = *music
	}
	if *demoPath != "" {
		s.Demo = *demoPath
	}
	if *room >= 0 {
		s.Room = int16(*room)
	}
	if *where != "" {
		var h, v int
		if _, err := fmt.Sscanf(*where, "%d,%d", &h, &v); err != nil {
			return fmt.Errorf("-where: want h,v: %w", err)
		}
		s.Where.H, s.Where.V = int16(h), int16(v)
		if s.Room < 0 {
			return fmt.Errorf("-where needs -room")
		}
	}
	// The directories always come from the flags, because they describe this machine
	// rather than the run. A script mailed in by a player names a house, not a path on
	// the reporter's disk.
	s.HouseDir, s.ArtDir, s.HouseArtDir, s.SoundDir = *houseDir, *artDir, *houseArt, *soundDir

	// A directed sentence rather than fs.Usage(): the caller knows what a replay is and has
	// left out one thing, so fifteen flag descriptions bury the answer.
	if s.House == "" {
		return fmt.Errorf("no house to replay: pass a script file, or -house %q (see `%s replay -h`)",
			"CD Demo House", prog)
	}

	if *writeBack != "" {
		w, closeOut, err := openOut(*writeBack)
		if err != nil {
			return err
		}
		if err := s.Write(w); err != nil {
			closeOut()
			return err
		}
		return closeOut()
	}

	// Sound is on by default, and on a checkout with no extracted sound tree that default has to
	// give way: `make assets` is a separate step, and a tool that refused to replay a house until
	// it had been run would be answering a question nobody asked. Typing -sound makes it a
	// request instead, and a request that cannot be met is replay.RunTo's error, which names the
	// file that is missing.
	//
	// This is deliberately below -script: the fallback is a fact about *this machine*, the way
	// the directory flags are, and a resolved script written out here is going to somebody else's
	// machine. Writing `sound off` into it because the developer had not run `make assets` would
	// mail them the wrong run.
	//
	// The note goes to stderr because stdout is the trace, and it is worth printing: "the sound
	// column is empty" and "this machine has no sounds extracted" look identical in the output
	// otherwise, and the first is a bug while the second is a Makefile target.
	if s.Sound && !set["sound"] && !audio.HasBank(*soundDir) {
		s.Sound = false
		fmt.Fprintf(os.Stderr, "%s: no sound bank in %s; replaying in silence (make assets, or -sounds dir)\n", prog, *soundDir)
	}

	// -wav is not exclusive with anything: the mix goes to the file and to the digest at the
	// same time, so the recording and the trace beside it describe one run rather than two.
	// replay.RunTo closes it, on the error paths too.
	//
	// There is deliberately no flag to play a replay through the speakers. The headless path
	// mixes one frame's samples per frame as fast as the machine can simulate -- ten minutes of
	// audio in a few seconds -- so a live player would drop almost all of it. Recording and
	// then playing the file is the same sound, correctly paced.
	var sink audio.Sink
	if *wav != "" {
		if !s.Sound {
			why := "the script says `sound off`"
			switch {
			case set["sound"]:
				why = "-sound=false"
			case !audio.HasBank(*soundDir):
				why = "there is no sound bank in " + *soundDir
			}
			return fmt.Errorf("-wav has nothing to write: the sound is off (%s)", why)
		}
		f, err := audio.CreateWAV(*wav)
		if err != nil {
			return err
		}
		sink = f
	}

	res, err := replay.RunTo(s, sink)
	// A sticky asset error still comes back with a usable result: the trace is worth
	// printing before the complaint, because "every room composed empty" is the symptom
	// the missing PICT explains.
	if res == nil {
		return err
	}

	// -o picks where the output goes and nothing else. It would be tempting to have it
	// imply -trace, on the grounds that nobody redirects six lines to a file, but then
	// there is no way to save a summary and no way to tell by reading the command line
	// which of the two a file contains.
	w, closeOut, cerr := openOut(*out)
	if cerr != nil {
		return cerr
	}
	switch {
	case *digest:
		fmt.Fprintln(w, res.Digest)
	case *trace:
		if terr := res.Trace(w); terr != nil {
			closeOut()
			return terr
		}
	default:
		summary(w, res)
	}
	if cerr := closeOut(); cerr != nil {
		return cerr
	}
	return err
}

// summary is the default output: enough to say whether the run went where it was meant to,
// and the digest to compare against somebody else's machine.
func summary(w io.Writer, res *replay.Result) {
	s := res.Script
	fmt.Fprintf(w, "%s  seed %d  %d frames  %d-room  %d player(s)\n",
		s.House, s.Seed, res.Frames, s.Neighbors, playerCount(s))
	if s.Room >= 0 {
		fmt.Fprintf(w, "  started room %d at %d,%d\n", s.Room, s.Where.H, s.Where.V)
	}
	fmt.Fprintf(w, "  ended   room %d  score %d  stars %d  mortals %d  gameOver %v\n",
		res.Room, res.Score, res.StarsLeft, res.Mortals, res.GameOver)
	if len(res.Samples) > 0 {
		last := res.Samples[len(res.Samples)-1]
		fmt.Fprintf(w, "  last frame  renders %d  work2main %d  back2work %d  pendulums %d  mode %d\n",
			last.Renders, last.Work2Main, last.Back2Work, last.Pendulums, last.Mode)
	}
	// The demo, when there is one, above the diagnostics: "consumed 300 of 1117" is the first
	// thing to look at in an attract-mode replay, because a run that ended early consumed a
	// prefix and its digest is a digest of a shorter demo than the one asked for.
	if s.Demo != "" {
		fmt.Fprintf(w, "  demo    %s\n", s.Demo)
		fmt.Fprintf(w, "          consumed %d of %d records, %d frames past the end\n",
			res.Demo.Consumed, res.Demo.Records, res.Demo.PastEnd)
	}
	fmt.Fprintf(w, "  dropped rects %d work, %d back   guarded reads %d\n",
		res.Diag.DroppedWorkRects, res.Diag.DroppedBackRects, res.Diag.Guarded)
	for _, d := range res.Diag.Seen {
		fmt.Fprintf(w, "    deviation: %s\n", d)
	}
	// The sound, in the same shape as the rest: what was asked for, what was heard, and a
	// digest to compare with. Refused and cut-off are on the first line because they are the
	// two numbers that explain "a sound did not play" without anybody having to listen -- the
	// first means three louder sounds were already going, the second means one was interrupted
	// mid-sample by a louder one on its channel.
	if s.Sound {
		a := res.Audio
		fmt.Fprintf(w, "  sound   %d asked, %d played, %d refused, %d cut off, %d trigger refused   music %d\n",
			a.Requests, a.Granted, a.Refused, a.Displaced, a.TriggerRefused, a.MusicStarted)
		fmt.Fprintf(w, "  mix     %d samples (%.1fs)  %d clipped  digest %s   bank %d KiB, %d house sounds\n",
			a.Samples, float64(a.Samples)/float64(audio.Rate), a.Clipped, a.Digest,
			(a.Bank+512)/1024, a.Triggers)
	}
	fmt.Fprintf(w, "  digest %s\n", res.Digest)
	// The picture, beside the behaviour and clearly labelled as a different claim. Two
	// reporters whose digests agree and whose screens do not have a machine-dependent
	// *renderer*, which is the one class of bug the trace cannot describe at all -- and the
	// three hashes say which stage of the pipeline it entered at. See replay.Planes.
	fmt.Fprintf(w, "  screen  back %s  work %s  main %s\n",
		res.Planes.Back, res.Planes.Work, res.Planes.Main)
}

func playerCount(s *replay.Script) int {
	if s.TwoPlayer {
		return 2
	}
	return 1
}
