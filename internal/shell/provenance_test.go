package shell

// What the About box owes somebody who only has the binary.
//
// A released build travels without this repository. The player who downloaded it has no
// README, no `git log` and no `go.mod`; the About box is the entire documentation they have, and
// the two facts that decide whether their bug report is actionable are which build they are
// running and where to send it. Neither is inferable from the screen if it is not printed on
// the screen.
//
// So these are not cosmetic assertions. internal/project holds the strings and its own tests
// pin them against go.mod and the README; what is left, and what is here, is that they reach a
// screen a player can actually get to.

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/project"
	"github.com/bwenstar/gliderGo/internal/render"
)

// aboutText joins the box into one string, because what is being asserted is that a fact is
// legible somewhere in it and not that it is on any particular row.
func aboutText(s *Shell) string {
	var b strings.Builder
	for _, l := range s.aboutLines() {
		b.WriteString(l.text)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestTheAboutBoxNamesTheProjectAndWhereItLives(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	text := aboutText(s)

	if !strings.Contains(text, project.Name) {
		t.Errorf("the About box does not name %s:\n%s", project.Name, text)
	}

	// The scheme is dropped on screen -- nobody types https:// off a photograph of a
	// monitor -- but the rest has to be exact, because a URL that is nearly right is worse
	// than none: it looks authoritative and goes nowhere.
	short := strings.TrimPrefix(project.Home, "https://")
	if !strings.Contains(text, short) {
		t.Errorf("the About box does not say where the project lives (%s):\n%s", short, text)
	}

	// And the original's credit, which is a licence obligation and not a courtesy.
	for _, want := range []string{project.Original, project.OriginalAuthor,
		project.OriginalPublisher, project.OriginalYear} {
		if !strings.Contains(text, want) {
			t.Errorf("the About box does not mention %q:\n%s", want, text)
		}
	}

	// And which revision of the C it is a transcription of. Without the commit the box makes
	// an unfalsifiable claim: "a port of Glider PRO" cannot be checked against anything,
	// while "github.com/softdorothy/glider_pro @ 94fed96" can be cloned and read. That is
	// the difference between a credit and provenance, and it is the whole reason a player
	// can settle "is this bug 1994's or theirs" without asking anybody.
	for _, want := range []string{strings.TrimPrefix(project.Upstream, "https://"),
		project.UpstreamCommit} {
		if !strings.Contains(text, want) {
			t.Errorf("the About box does not say which source it was transcribed from "+
				"(%s):\n%s", want, text)
		}
	}
}

// TestTheAboutBoxNamesTheBuild pins the other half. The version is a host field rather than a
// constant, so a host that does not set it must degrade to the project name alone rather than
// to a dangling row -- which is the state `-shot` and every test here run in.
func TestTheAboutBoxNamesTheBuild(t *testing.T) {
	s, f := shellOver(t, []string{"Slumberland"})
	if got := aboutText(s); !strings.Contains(got, project.Name+"\n") {
		t.Errorf("with no version on the host the About box should name the project alone, got:\n%s", got)
	}

	// With one, it is on the same row, so that a photograph of the box carries it.
	host := f.host()
	host.Version = "v9.9.9-test"
	withVer, err := New(host, s.lib)
	if err != nil {
		t.Fatal(err)
	}
	if got := aboutText(withVer); !strings.Contains(got, project.Name+" v9.9.9-test") {
		t.Errorf("the About box does not name the build:\n%s", got)
	}
}

// TestEveryAboutLineFits is the reason the URL is short. The panel is 640 pixels wide at 1x and
// the font is fixed-pitch, so a line that overflows is silently clipped by `fit` at draw time --
// a truncated URL is exactly the failure this whole file exists to prevent, and it would not
// show up in any of the assertions above.
func TestEveryAboutLineFits(t *testing.T) {
	s, _ := shellOver(t, []string{"Slumberland"})
	const room = aboutRight - aboutLeft - 16 // the panel, less an inset on each side
	for _, l := range s.aboutLines() {
		if l.text == "" {
			continue
		}
		scale := l.scale
		if scale == 0 {
			scale = 1
		}
		if w := render.StringWidthScaled(l.text, scale); w > room {
			t.Errorf("%q is %d pixels wide and the box has %d", l.text, w, room)
		}
	}
}

// ditlPicture matches a row of the DITL 150 item table in docs/analysis/ui-dialogs.md 10.6:
//
//	| 4 | picture (disabled) | (5,6,105,378) | 372x100 | `PICT` **153** -- the Glider PRO logo art |
//
// The id is sometimes bolded and sometimes not, which is why the asterisks are optional rather
// than tidied up in the document: the table is a transcription and it is left as it reads.
var ditlPicture = regexp.MustCompile(
	"^\\| *[0-9]+ *\\| *picture[^|]*\\|[^|]*\\| *([0-9]+)×([0-9]+) *\\| *`PICT` *\\*{0,2}([0-9]+)")

// TestTheAboutHeadingIsTheOriginalsTitleArt is the assertion that four stages of this port
// needed and did not have.
//
// The About box drew PICT 151 from 1.7a until now. 150 and 151 are the two states of the fake
// Okay button the original's dialog draws -- 63x63, a diamond on white -- so the heading of the
// only screen a downloaded binary carries was a small button, which reads as a draw that failed.
// The picture the dialog actually puts at its top is item 4, the 372x100 title art, and the cost
// of the mistake was not only cosmetic: that art is the one place the 1994 game credits Paul
// Finn, Steve Sullivan and Ward Hartenstein on screen.
//
// So the number is checked against the document that records the dialog rather than against
// itself. Among DITL 150's picture items the heading is the largest -- the others are a button --
// which is a rule about the layout and not a restatement of the answer.
func TestTheAboutHeadingIsTheOriginalsTitleArt(t *testing.T) {
	spec := filepath.Join(repoRoot(t), "docs", "analysis", "ui-dialogs.md")
	b, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}

	inAbout := false
	best, bestArea := 0, 0
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "### ") {
			inAbout = strings.Contains(line, "10.6") && strings.Contains(line, "About box")
			continue
		}
		if !inAbout {
			continue
		}
		m := ditlPicture.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		id, _ := strconv.Atoi(m[3])
		if w*h > bestArea {
			best, bestArea = id, w*h
		}
	}

	if bestArea == 0 {
		t.Fatalf("no picture items found in %s 10.6; the table this test reads has moved", spec)
	}
	if best != aboutPlate {
		t.Errorf("ui-dialogs.md 10.6 gives PICT %d as the About dialog's largest picture item "+
			"and aboutPlate is %d; 150 and 151 are the fake Okay button's two states, so the "+
			"box would be headed by a 63x63 diamond", best, aboutPlate)
	}
}

// repoRoot walks up to the directory holding go.mod, so a test can read a document that lives
// outside its own package. internal/project has the same helper for the same reason.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the package directory")
		}
		dir = parent
	}
}
