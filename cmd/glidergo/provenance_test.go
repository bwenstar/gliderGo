package main

// What `-version` owes a bug report.
//
// `-version` is the one command in this program whose output is written to be pasted somewhere
// else, which makes it the only place where a wrong string costs somebody else's time rather
// than the player's. Two kinds of wrongness are worth a test. A missing fact is the cheap kind:
// the report arrives and a question has to be asked. A *misstated* fact is the expensive kind --
// a path the game does not actually use sends the reader to look at a file that was never read.
//
// So statePaths gets the attention here. It mirrors three switch statements in prefs.go and
// play.go rather than calling them, because those three open files and print notes and -version
// must do neither, and a mirror is exactly the kind of code that is correct on the day it is
// written and silently wrong a stage later.

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// pathsOf runs statePaths and hands back a lookup, because what each test wants is one row.
func pathsOf(o *options) map[string]string {
	out := map[string]string{}
	for _, r := range statePaths(o) {
		out[r[0]] = r[1]
	}
	return out
}

// TestVersionReportsAllThreeStores is the shape assertion: three stores, always, named.
//
// Three and not one, because they are three directories in two roots -- prefs under
// os.UserConfigDir, scores and saves under the data directory -- and "where did my settings go"
// and "where did my saved game go" have different answers on every platform this builds for.
func TestVersionReportsAllThreeStores(t *testing.T) {
	got := pathsOf(&options{})
	for _, want := range []string{"prefs", "scores", "saves"} {
		if got[want] == "" {
			t.Errorf("-version does not say where %s live: %v", want, got)
		}
	}
}

// TestVersionQuotesTheDirectoryAFlagNamed is the case a bug report most often arrives in: the
// reporter ran with -scores or -saves pointing somewhere of their own, and the paths printed
// have to be those and not the defaults.
func TestVersionQuotesTheDirectoryAFlagNamed(t *testing.T) {
	got := pathsOf(&options{
		prefsPath: "/tmp/p.json",
		scoresDir: "/tmp/boards",
		savesDir:  "/tmp/games",
	})
	for what, want := range map[string]string{
		"prefs":  "/tmp/p.json",
		"scores": "/tmp/boards",
		"saves":  "/tmp/games",
	} {
		if got[what] != want {
			t.Errorf("%s reads %q, want %q", what, got[what], want)
		}
	}
}

// TestVersionSaysNowhereRatherThanAPathItWillNotUse covers the three opt-outs.
//
// This is the assertion with teeth. `-prefs none` is one word and it is what a bisect, a shared
// machine and a bug report's reproduction all use, so it is *not* a rare state -- and printing
// the default path underneath it would be the worst available answer: it names a real file, on
// the right machine, that this run never touched. The message says the flag back, so the reader
// can see why.
func TestVersionSaysNowhereRatherThanAPathItWillNotUse(t *testing.T) {
	for _, tc := range []struct {
		what string
		o    options
		flag string
	}{
		{"prefs", options{prefsPath: prefsNone}, "-prefs none"},
		{"scores", options{scoresDir: scoresNone}, "-scores none"},
		{"saves", options{savesDir: savesNone}, "-saves none"},
	} {
		got := pathsOf(&tc.o)[tc.what]
		if !strings.HasPrefix(got, "nowhere") {
			t.Errorf("with %s, %s reads %q; it has to say that nothing is read or written",
				tc.flag, tc.what, got)
		}
		if !strings.Contains(got, tc.flag) {
			t.Errorf("with %s, %s reads %q and does not name the flag that caused it",
				tc.flag, tc.what, got)
		}
	}
}

// TestVersionKnowsAMeasurementRunKeepsNothing is the fourth branch of loadPrefs, and the one a
// mirror is most likely to miss: -shot, -frames, -bench and -dump read this build's defaults and
// write nothing, without anybody having said -prefs none.
//
// It is here rather than left to prefs_test.go's TestHermeticRunsIgnoreThePlayersPreferences
// because that test asserts what the *game* does, and this one asserts that -version says the
// same thing. The two agreeing is the whole point; that is what makes the block quotable.
func TestVersionKnowsAMeasurementRunKeepsNothing(t *testing.T) {
	for _, o := range []options{
		{shot: "/tmp/x.png"},
		{frames: 300},
		{bench: true},
		{dump: "/tmp/frames"},
	} {
		if got := pathsOf(&o)["prefs"]; !strings.HasPrefix(got, "nowhere") {
			t.Errorf("a measurement run (%+v) reports prefs at %q, but loadPrefs gives it "+
				"this build's defaults and saves nothing", o, got)
		}
	}

	// And the exception, which is the reason hermetic is not the whole test: an explicit file
	// wins even for a measurement, because `-shot -prefs testdata/x.json` is how a golden
	// screenshot of the settings screen gets settings to show.
	o := options{shot: "/tmp/x.png", prefsPath: "testdata/x.json"}
	if got := pathsOf(&o)["prefs"]; got != "testdata/x.json" {
		t.Errorf("-shot with an explicit -prefs reports %q, want the file that is read", got)
	}
}

// TestTheAudioRowSurvivesEveryWayOfAskingForNoSound is the same argument one row down.
//
// audioRoute is the only line in the block that predicts rather than reports, so it is the line
// most able to be confidently wrong. The four cases here are the four ways a session ends up
// without a player, and each has a different fix: the flag said so, the flag said list, the
// recording was the point, or the machine has nothing installed. A row that gave the same answer
// to all four would send three of those four readers the wrong way.
func TestTheAudioRowSurvivesEveryWayOfAskingForNoSound(t *testing.T) {
	for _, tc := range []struct {
		why  string
		o    options
		says string
	}{
		{"-sound=false", options{}, "-sound=false"},
		{"-audio list", options{sound: true, audioOut: "list"}, "-audio list"},
		{"-wav alone", options{sound: true, wav: "/tmp/s.wav"}, "/tmp/s.wav"},
	} {
		got := audioRoute(&tc.o)
		if !strings.Contains(got, tc.says) {
			t.Errorf("with %s the audio row reads %q and does not name the reason", tc.why, got)
		}
	}

	// -wav alone opens no player, and the row must not name one: that is openSink's middle
	// case, and it is the one a reader would otherwise assume wrong in either direction.
	if got := audioRoute(&options{sound: true, wav: "/tmp/s.wav"}); strings.Contains(got, "+") {
		t.Errorf("-wav alone reports two outputs (%q); openSink opens only the file", got)
	}
	// With both, both, because `-audio x -wav f` is play-and-keep and a report about the file
	// and a report about the sound are different reports.
	got := audioRoute(&options{sound: true, wav: "/tmp/s.wav", audioOut: "aplay"})
	for _, want := range []string{"/tmp/s.wav", "aplay"} {
		if !strings.Contains(got, want) {
			t.Errorf("-audio aplay -wav /tmp/s.wav reports %q, which omits %s", got, want)
		}
	}
}

// TestTheAudioRowDoesNotClaimAFlaggedOutputIsInstalled is the misstatement worth a test of its
// own: audio.Open insists on a name given with -audio, so an absent one is a fatal error and not
// a fallback. The row is written before that error can happen, so it has to hedge -- and the
// hedge is load-bearing, because a reader who sees a bare player name in a version block will
// read it as "this machine has that player".
func TestTheAudioRowDoesNotClaimAFlaggedOutputIsInstalled(t *testing.T) {
	got := audioRoute(&options{sound: true, audioOut: "no-such-player"})
	if !strings.Contains(got, "no-such-player") {
		t.Fatalf("the audio row does not quote the -audio value back: %q", got)
	}
	if !strings.Contains(got, "-audio") {
		t.Errorf("the audio row reads %q without saying a flag chose it", got)
	}
}

// TestTheUsageErrorIsNotABugReport pins the one distinction the fatal footer makes.
//
// main prints a second line under every error, and which line depends on this: a command line
// that was rejected is told about -h, and everything else is pointed at the tracker. The value
// of the footer is entirely in that split -- a footer under `-neighbors must be 1, 3 or 9`
// saying "if that is not something you can fix, file an issue" is noise, and a footer that is
// noise on the commonest error is one nobody reads by the time it matters.
func TestTheUsageErrorIsNotABugReport(t *testing.T) {
	isUsage := func(err error) bool {
		var u usageErr
		return errors.As(err, &u)
	}

	inner := errors.New("-neighbors must be 1, 3 or 9")
	wrapped := error(usageErr{inner})

	// The message is the specific one, not a generic "bad usage": the wrap adds a
	// classification and no words.
	if wrapped.Error() != inner.Error() {
		t.Errorf("usageErr changed the message to %q, want %q", wrapped, inner)
	}
	if !isUsage(wrapped) {
		t.Error("a wrapped command-line error does not read as one")
	}
	if isUsage(inner) {
		t.Error("an unwrapped error reads as a command-line error, so every failure would " +
			"be told to look at -h instead of being reported")
	}

	// And it stays classified through another layer, because a check that wraps with %w for
	// context is the ordinary way one of these grows.
	if !isUsage(fmt.Errorf("reading the flags: %w", wrapped)) {
		t.Error("usageErr does not survive being wrapped again")
	}
}
