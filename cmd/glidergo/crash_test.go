package main

// The crash file (crash.go): that a run which dies leaves what it printed, that the next run keeps
// it and says so, and that nothing else is ever called a crash.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/assets"
)

// crashChild is the environment variable that makes this test binary the run that crashes.
const crashChild = "GLIDERGO_TEST_CRASH_CHILD"

func crashOptions() *options { return &options{tree: assets.Tree(), levelTree: assets.Levels()} }

// openedCrashLog is openCrashLog, closed again when the test ends.
func openedCrashLog(t *testing.T) *crashLog {
	t.Helper()
	c := openCrashLog(crashOptions())
	t.Cleanup(c.close)
	if c == nil || c.f == nil {
		t.Fatalf("openCrashLog kept no file in %s", os.Getenv("GLIDERGO_DATA"))
	}
	return c
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The whole path, with a real crash: this binary is run again as a game that opens its crash file
// and then panics on a goroutine of its own, as the sound or the network could. The runtime's own
// report has to be in the file, under the header -- nothing here writes it but the runtime -- and
// the next start has to keep it and say so.
func TestACrashIsKeptForTheNextRun(t *testing.T) {
	if os.Getenv(crashChild) != "" {
		openCrashLog(crashOptions())
		done := make(chan struct{})
		go func() { panic("the test's own crash") }()
		<-done
	}

	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestACrashIsKeptForTheNextRun$")
	cmd.Env = append(os.Environ(), crashChild+"=1", "GLIDERGO_DATA="+dir, "GLIDERGO_CONFIG="+dir)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("the crashing run did not fail (%v):\n%s", err, out)
	}

	path := filepath.Join(dir, crashName)
	got := readFile(t, path)
	head, tail, ok := strings.Cut(got, crashRule)
	switch {
	case !ok:
		t.Fatalf("%s has no rule:\n%s", path, got)
	case !strings.HasPrefix(head, "glidergo ") || !strings.Contains(head, "  bugs "):
		t.Errorf("the header is not the -version block:\n%s", head)
	case !strings.Contains(tail, "panic: the test's own crash") || !strings.Contains(tail, "goroutine "):
		t.Errorf("below the rule is not the runtime's report:\n%s", tail)
	}

	t.Setenv("GLIDERGO_DATA", dir)
	next := openedCrashLog(t)
	last := filepath.Join(dir, crashLastName)
	if next.last != last {
		t.Fatalf("the next start kept the crash as %q, want %q", next.last, last)
	}
	if kept := readFile(t, last); kept != got {
		t.Errorf("%s is not the crashed run's file:\n%s", last, kept)
	}
	if _, tail, _ := strings.Cut(readFile(t, path), crashRule); strings.TrimSpace(tail) != "" {
		t.Errorf("the new %s already has something below its rule: %q", crashName, tail)
	}
	if n := next.notice(); !strings.Contains(n, "crashed") || !strings.HasSuffix(n, crashLastName) {
		t.Errorf("the status band says %q, want the crash and the file", n)
	}
	next.close()

	// A clean run is not kept, and does not take the crash-last.log it did not write away.
	again := openedCrashLog(t)
	if again.last != "" || again.notice() != "" {
		t.Errorf("a start after a clean run kept %q and says %q", again.last, again.notice())
	}
	if readFile(t, last) != got {
		t.Error("a clean run changed crash-last.log")
	}
}

// What main writes is an error, and an error is not a crash: it is almost always the player's to
// fix, and they were shown it. A newline inside one must not make its second line look like the
// runtime's.
func TestAnErrorIsNotACrash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLIDERGO_DATA", dir)
	c := openedCrashLog(t)
	c.stopped(errors.New("house \"Nowhere\" is not in the library\nand here is a second line"))
	c.close()

	got := readFile(t, filepath.Join(dir, crashName))
	if !strings.Contains(got, errorPrefix+"and here is a second line\n") {
		t.Errorf("the error's second line was not written with the prefix:\n%s", got)
	}
	if next := openedCrashLog(t); next.last != "" {
		t.Errorf("an error was kept as a crash, in %s", next.last)
	}
	if _, err := os.Stat(filepath.Join(dir, crashLastName)); err == nil {
		t.Errorf("an error left a %s", crashLastName)
	}
}

func TestWhatCountsAsACrash(t *testing.T) {
	head := "glidergo dev\n  backend   x11\n\n" + crashRule + "\n"
	for _, tc := range []struct {
		name string
		file string
		want bool
	}{
		{"a clean run", head, false},
		{"an error", head + errorPrefix + "no display\n", false},
		{"a panic", head + "panic: runtime error: index out of range\n\ngoroutine 1 [running]:\n", true},
		{"a fatal error", head + "fatal error: all goroutines are asleep - deadlock!\n", true},
		{"a Windows exception", head + "Exception 0xc0000005 0x0 0x0 0x7ff6\nPC=0x7ff6\n", true},
		{"a signal in C", head + "SIGSEGV: segmentation violation\nPC=0x0 m=0 sigcode=1\n", true},
		{"an error, then a crash", head + errorPrefix + "x\npanic: y\n", true},
		{"no rule at all", "panic: from a file that is not ours\n", false},
		{"nothing", "", false},
	} {
		if got := crashed([]byte(tc.file)); got != tc.want {
			t.Errorf("%s: crashed = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The band has room for about ninety characters, so the path on it is the one a player would
// type, and it has to still name the file.
func TestTheBandWritesThePathAsAPlayerWouldTypeIt(t *testing.T) {
	var home, want string
	if runtime.GOOS == "windows" {
		home = `C:\Users\someone\AppData\Roaming`
		t.Setenv("APPDATA", home)
		want = `%AppData%\glidergo\crash-last.log`
	} else {
		home = "/home/someone"
		t.Setenv("HOME", home)
		want = "~/.local/share/glidergo/crash-last.log"
		if runtime.GOOS == "darwin" {
			want = "~/Library/Application Support/glidergo/crash-last.log"
		}
	}
	path := filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(want, "%AppData%"), "~"))
	if got := typedPath(path); got != want {
		t.Errorf("typedPath(%q) = %q, want %q", path, got, want)
	}
	if elsewhere := filepath.Join(t.TempDir(), crashLastName); typedPath(elsewhere) != elsewhere {
		t.Errorf("a path under neither was rewritten: %q", typedPath(elsewhere))
	}
}

// -version names the file, and says so when a measurement run would keep none.
func TestVersionSaysWhereACrashIsKept(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLIDERGO_DATA", dir)
	if got, want := pathsOf(&options{})["crash"], filepath.Join(dir, crashName); got != want {
		t.Errorf("the crash row is %q, want %q", got, want)
	}
	for _, o := range []options{{shot: "/tmp/x.png"}, {frames: 300}, {bench: true}, {dump: "/tmp/f"}} {
		if got := pathsOf(&o)["crash"]; !strings.HasPrefix(got, "nowhere") {
			t.Errorf("a measurement run (%+v) reports a crash file at %q, and keeps none", o, got)
		}
	}
}
