package main

// presenter's one rule, on its own and then in a real game: a present may be put off but never past
// the next sleep, and putting presents off never changes what the window ends up showing.

import (
	"crypto/sha256"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/shell"
)

// answers is a Coalescer that says what it is told to, in order, and then no.
type answers []bool

func (a *answers) Coalesce() bool {
	if len(*a) == 0 {
		return false
	}
	yes := (*a)[0]
	*a = (*a)[1:]
	return yes
}

// sends is a send func that records the first pixel of every frame it is given.
type sends []uint8

func (s *sends) send(f *render.Surface) { *s = append(*s, f.Pix[0]) }

func frame(v uint8) *render.Surface {
	f := render.NewSurface(4, 2)
	for i := range f.Pix {
		f.Pix[i] = v
	}
	return f
}

func TestAPresentIsPutOffUntilTheNextSleep(t *testing.T) {
	var got sends
	yes := answers{true, true, true, true}
	p := &presenter{co: &yes, send: got.send}

	p.present(frame(1))
	p.present(frame(2))
	p.present(frame(3))
	if len(got) != 0 {
		t.Fatalf("%d presents were sent that the window asked to have later", len(got))
	}
	p.sleep(0)
	if len(got) != 1 || got[0] != 3 {
		t.Fatalf("a sleep with presents owed sent %v, want the last of them, [3]", got)
	}
	p.sleep(0)
	if len(got) != 1 {
		t.Fatalf("a sleep with nothing owed sent another: %v", got)
	}
	p.present(frame(4))
	p.flush()
	p.flush()
	if len(got) != 2 || got[1] != 4 {
		t.Fatalf("flush sent %v in all, want [3 4]: one owed, and nothing for the second flush", got)
	}
}

// TestWhatIsOwedIsTheFrameAsItWas: the game draws on Main between presents and does not always
// present what it drew, so the debt is a copy, and a change after the present is not sent with it.
func TestWhatIsOwedIsTheFrameAsItWas(t *testing.T) {
	var got sends
	yes := answers{true}
	p := &presenter{co: &yes, send: got.send}
	main := frame(7)
	p.present(main)
	main.Pix[0] = 9 // an erase nobody presents
	p.sleep(0)
	if len(got) != 1 || got[0] != 7 {
		t.Errorf("the sleep sent %v, want [7]: the frame as presented, not the screen as it is now", got)
	}
}

// TestAPresentAroundThePresenterCancelsTheDebt: the waits and the dialogs present without it, and
// what they send is newer than anything owed, so paying the debt afterwards would put an older
// frame back on the screen.
func TestAPresentAroundThePresenterCancelsTheDebt(t *testing.T) {
	var got sends
	yes := answers{true}
	p := &presenter{co: &yes, send: got.send}
	p.present(frame(1)) // the end of a game's opening burst, owed
	p.superseded()      // its banner, presented by the wait
	p.sleep(0)
	if len(got) != 0 {
		t.Errorf("the sleep after a newer present sent %v, which would put the old frame back", got)
	}
	var none *presenter
	none.superseded() // between games there is no presenter, and app.present still calls this
}

func TestASentPresentPaysWhatIsOwed(t *testing.T) {
	var got sends
	asked := answers{true, true, false}
	p := &presenter{co: &asked, send: got.send}
	p.present(frame(1))
	p.present(frame(2))
	p.present(frame(3)) // sent, and it is the latest frame, so nothing is owed now
	p.sleep(0)
	if len(got) != 1 || got[0] != 3 {
		t.Errorf("sent %v, want [3]: the present the window took pays for the two before it", got)
	}
}

func TestAWindowThatDoesNotAskIsSentEveryPresent(t *testing.T) {
	var got sends
	p := newPresenter(&fakeWin{}, got.send)
	if p.co != nil {
		t.Fatal("fakeWin is not a Coalescer, and newPresenter found one")
	}
	for i := uint8(1); i <= 3; i++ {
		p.present(frame(i))
	}
	p.sleep(0)
	if len(got) != 3 {
		t.Errorf("sent %v to a window that never asked, want all three", got)
	}
	// And a nil window -- the hooks exist before there is always one -- is the same.
	if newPresenter(nil, got.send).co != nil {
		t.Error("newPresenter found a Coalescer in a nil window")
	}
}

// recordWin is fakeWin that keeps a hash of every frame it is sent.
type recordWin struct {
	fakeWin
	frames [][32]byte
}

func (r *recordWin) Present(fb *platform.Framebuffer) error {
	r.frames = append(r.frames, sha256.Sum256(fb.Pix))
	return nil
}

// askingWin asks to have every present later, which is the most any window could ask.
type askingWin struct{ recordWin }

func (*askingWin) Coalesce() bool { return true }

// TestCoalescingOnlyLeavesFramesOut plays the same paced game twice, to a window that takes every
// present and to one that puts every present off, and requires the second to have been sent a
// subsequence of what the first was sent, ending on the same frame. That is the whole of what
// coalescing may do: leave frames out. It may not change one, reorder two, or lose the last.
//
// Paced, because the rule is about sleeps and only a paced game sleeps; 20 frames take two thirds
// of a second. This is the test that found the debt had to be a copy: the game's teardown draws
// after its last frame, and a flush of Main as it stood sent a frame no present ever made.
func TestCoalescingOnlyLeavesFramesOut(t *testing.T) {
	play := func(win platform.Window) {
		t.Helper()
		path, _ := houseWithGame(t, t.TempDir(), "Slumberland", false, 0, 0)
		o := &options{frames: 20, quiet: true, roomNum: -1, savesDir: savesNone, scoresDir: scoresNone}
		a := newApp(o, prefs.Default(), false)
		a.win = win
		a.fb = platform.NewFramebuffer(640, 480)
		if _, err := a.play(diskHouse(path), false, false, shell.Race{}); err != nil {
			t.Fatal(err)
		}
	}
	every := &recordWin{fakeWin: fakeWin{script: idlePolls(1000), held: map[platform.Key]bool{}}}
	put := &askingWin{recordWin{fakeWin: fakeWin{script: idlePolls(1000), held: map[platform.Key]bool{}}}}
	play(every)
	play(put)

	if len(every.frames) == 0 || len(put.frames) == 0 {
		t.Fatalf("frames sent: %d to the plain window, %d to the asking one", len(every.frames), len(put.frames))
	}
	if len(put.frames) >= len(every.frames) {
		t.Errorf("the asking window was sent %d frames and the plain one %d; putting presents off "+
			"left none out", len(put.frames), len(every.frames))
	}
	i := 0
	for n, f := range put.frames {
		for i < len(every.frames) && every.frames[i] != f {
			i++
		}
		if i == len(every.frames) {
			t.Fatalf("the asking window's frame %d of %d is not one the plain window was sent, "+
				"or not in the same order", n+1, len(put.frames))
		}
		i++
	}
	if put.frames[len(put.frames)-1] != every.frames[len(every.frames)-1] {
		t.Error("the two windows were left showing different frames")
	}
}
