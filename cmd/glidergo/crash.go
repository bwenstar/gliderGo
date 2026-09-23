package main

// What a run leaves behind when it dies (docs/IMPROVEMENTS.md 4.35).
//
// Double-clicking glidergo.exe is a documented way to start it, and the console Windows opens for
// it closes when the process exits, taking the panic trace or the error with it. A Linux desktop
// launcher sends stderr nowhere at all. So a run a player started keeps one file, crash.log in the
// data directory: the -version block at the top, a rule under it, and below the rule whatever
// stopped the run -- what the runtime printed as it died (runtime/debug.SetCrashOutput), or the
// error main stopped with. The next run a player starts looks below the rule. A crash there is
// kept as crash-last.log and said on stderr and on the status band; anything else is overwritten.
//
// The file is rewritten at every start rather than appended to. A log that gained a header on
// every clean run would make "it grew" mean nothing, and a player asked for it would send a year
// of starts to report one crash. Measurement runs (-frames, -bench, -shot, -dump) keep none, for
// 2.53's reason: a run that measures must not depend on, or change, the machine it measures.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bwenstar/gliderGo/internal/datadir"
	"github.com/bwenstar/gliderGo/internal/project"
)

const (
	crashName     = "crash.log"
	crashLastName = "crash-last.log"

	// crashRule ends the header. It is written for the person who opens the file, and it is how
	// the next run finds where the header stops.
	crashRule = "-- anything below this line is what stopped the run above --"

	// errorPrefix begins every line main writes below the rule, as it begins them on stderr. It
	// is what tells an error apart from a crash: nothing the runtime prints as it dies starts
	// with it.
	errorPrefix = "glidergo: "

	// crashReadMax is as much of the last run's file as is read to decide. A crash trace from
	// this program is a few dozen kilobytes; one larger than this is kept whole by the rename
	// and only its first part is copied if the rename fails.
	crashReadMax = 4 << 20
)

// crashLog is this run's crash file.
type crashLog struct {
	f *os.File // open for appending

	// last is crash-last.log's path when the run before this one crashed, and "" when it did not.
	last string
}

// crash is this run's crash file, from the moment run opens it; nil before then, on a
// measurement run, and on a machine with nowhere to keep one. main reads it after run returns.
var crash *crashLog

// crashDir is where both files go: the data directory's root, beside the scores and saves
// directories and not inside either, because a crash belongs to neither.
func crashDir() (string, error) { return datadir.Dir("") }

// openCrashLog keeps the last run's crash, if it left one, and starts this run's file.
//
// It never fails. A machine with no data directory, or a read-only one, plays without a crash
// file, as it plays without saves; -version still says where the file would be.
func openCrashLog(o *options) *crashLog {
	dir, err := crashDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(dir, crashName)
	c := &crashLog{}
	if old, err := readUpTo(path, crashReadMax); err == nil && crashed(old) {
		last := filepath.Join(dir, crashLastName)
		// A rename keeps all of it. It can fail where a copy would not -- on Windows, when
		// another copy of the game still has the file open -- so a copy is the fallback.
		if os.Rename(path, last) == nil || os.WriteFile(last, old, 0o644) == nil {
			c.last = last
		}
	}

	// The header is written with a plain truncating write, and the file then opened again for
	// appending, rather than one O_TRUNC|O_APPEND open: on Windows Go opens an O_APPEND file
	// with append access only, and whether CREATE_ALWAYS may truncate with that is not a
	// question worth an unrun platform. Two copies of the game on one machine, as a local race
	// is, each rewrite the header when they start and then both append, so neither loses a
	// crash to the other.
	var head bytes.Buffer
	crashHeader(&head, o)
	if os.MkdirAll(dir, 0o755) != nil || os.WriteFile(path, head.Bytes(), 0o644) != nil {
		return c
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return c
	}
	if debug.SetCrashOutput(f, debug.CrashOptions{}) != nil {
		f.Close()
		return c
	}
	c.f = f
	return c
}

// crashHeader is the top of the file: what -version prints, when this run started and what it
// was asked for, and the rule. The -version block is what the bug-report footer asks for, so a
// player who sends this file has sent it -- including, in its bugs row, where to send it.
func crashHeader(w io.Writer, o *options) {
	printVersion(w, o)
	fmt.Fprintf(w, "  started   %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, "  args      %q\n", os.Args[1:])
	fmt.Fprintf(w, "\n%s\n", crashRule)
}

// crashed reports whether a crash file holds a crash: a line below its rule that main did not
// write. A file with no rule is not one -- it is from a run whose header never got written,
// which is a full disk rather than a crash.
func crashed(b []byte) bool {
	i := bytes.Index(b, []byte(crashRule))
	if i < 0 {
		return false
	}
	for _, line := range strings.Split(string(b[i+len(crashRule):]), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, errorPrefix) {
			return true
		}
	}
	return false
}

// stopped writes the error run returned below the rule, a line at a time and each with the
// prefix, so that an error with a newline in it is still not mistaken for a crash.
//
// It is not kept by the next run. An error is almost always one the player was shown and can
// fix -- a house that is not there, a display that is not -- and a status band that called
// every one of those a crash would be wrong most of the time. It is here for the run whose
// stderr went nowhere, until the next start.
func (c *crashLog) stopped(err error) {
	if c == nil || c.f == nil || err == nil {
		return
	}
	var b strings.Builder
	for _, line := range strings.Split(err.Error(), "\n") {
		b.WriteString(errorPrefix + line + "\n")
	}
	c.f.WriteString(b.String())
}

// close stops the runtime writing to the file and closes it. A run never needs to -- the
// process ending closes both -- but a test that opened one does, because Windows will not remove
// a directory with a file open in it.
func (c *crashLog) close() {
	if c == nil || c.f == nil {
		return
	}
	debug.SetCrashOutput(nil, debug.CrashOptions{})
	c.f.Close()
	c.f = nil
}

// tell says, on stderr, that the last run crashed and where its report is. The full path, and
// the tracker, because a terminal has the room the status band does not.
func (c *crashLog) tell() {
	if c == nil || c.last == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "%sthe last run crashed; its report is in %s\n", errorPrefix, c.last)
	fmt.Fprintf(os.Stderr, "%sif you can, attach that file to an issue at %s -- "+
		"it begins with what -version prints\n", errorPrefix, project.Issues)
}

// notice is the status band's line for the same thing, or "". The band has room for about ninety
// characters, so the path is written the way a player would type it: %AppData%\... on Windows,
// which Explorer's address bar and Win+R both expand, and ~/... elsewhere.
func (c *crashLog) notice() string {
	if c == nil || c.last == "" {
		return ""
	}
	return "the last run crashed; its report is in " + typedPath(c.last)
}

// typedPath is path as a player would type it: under %AppData% on Windows, under ~ elsewhere,
// and unchanged when it is under neither.
func typedPath(path string) string {
	sep := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		if d := os.Getenv("APPDATA"); d != "" && strings.HasPrefix(strings.ToLower(path), strings.ToLower(d)+sep) {
			return "%AppData%" + path[len(d):]
		}
		return path
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(path, home+sep) {
		return "~" + path[len(home):]
	}
	return path
}

// readUpTo is os.ReadFile, stopping at max bytes.
func readUpTo(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, max))
}
