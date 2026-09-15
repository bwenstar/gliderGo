package main

import (
	"flag"
	"fmt"
	"io"
	"os"

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
		frames    = fs.Int("frames", 0, "frames to run (overrides the script)")
		seed      = fs.Int("seed", -1, "random seed (overrides the script)")
		neighbors = fs.Int("neighbors", 0, "compose 1, 3 or 9 rooms (overrides the script)")
		two       = fs.Bool("two", false, "two players")
		room      = fs.Int("room", -1, "start room (overrides the script)")
		where     = fs.String("where", "", "start the glider at h,v in -room: a resume")
		trace     = fs.Bool("trace", false, "print the per-frame trace, not just the summary")
		digest    = fs.Bool("digest", false, "print only the digest, for scripting")
		writeBack = fs.String("script", "", "write the resolved script here (- for stdout) and exit")
		out       = fs.String("o", "", "write the output to this file instead of stdout")
	)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `usage: %s replay [flags] [script]

Runs the game with no window and no sound, from a script, and reports what each
frame did. With no script file, the flags alone describe the run.

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

`, prog, prog)
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
	if *two {
		s.TwoPlayer = true
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
	s.HouseDir, s.ArtDir, s.HouseArtDir = *houseDir, *artDir, *houseArt

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

	res, err := replay.Run(s)
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
	fmt.Fprintf(w, "  dropped rects %d work, %d back   guarded reads %d\n",
		res.Diag.DroppedWorkRects, res.Diag.DroppedBackRects, res.Diag.Guarded)
	for _, d := range res.Diag.Seen {
		fmt.Fprintf(w, "    deviation: %s\n", d)
	}
	fmt.Fprintf(w, "  digest %s\n", res.Digest)
}

func playerCount(s *replay.Script) int {
	if s.TwoPlayer {
		return 2
	}
	return 1
}
