package main

// Both ends of a race in one process, over 127.0.0.1, through the same a.play a player's race
// goes through (docs/IMPROVEMENTS.md 4.34).
//
// internal/netplay tests the protocol and the driver, and race_test.go the lines a player reads.
// Nothing tested the join between them -- the waiting screens, the hand-over of the seed, the
// report once a frame, the wait at the end and the result -- and four planned changes (4.31 to
// 4.33, and 4.28's redial) are about to edit exactly that. These are the tests they edit against.
//
// Every test here is named Loopback, so that `make race` can pick them out and run them under the
// race detector without the rest of this package's twenty seconds. They are where this package's
// goroutines meet: race.go shakes hands on one behind the waiting screen, and netplay reads and
// writes on two more for the whole race.
//
// The house is Grand Prix, because an idle glider dies in it: in about 350 frames with nobody
// at the keys, and further on holding right. A frame-limited run would not do, because a run that
// -frames stops has not ended -- it has been left (finalStanding), and two runs that were both
// left make a race nobody won.

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/assets"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/netplay"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/shell"
)

// idleWin is a window with nobody at it: no events ever, and it never closes. fakeWin cannot be
// one, because past the end of its script it reports the window closed -- right for a test that
// should fail rather than hang, and wrong for a race, whose waiting screens poll for as long as
// the other machine takes. What stops a hang here is what stops one in a real headless race: the
// two deadlines a bench run keeps.
//
// held is keys held down for the whole run, which is as much flying as these tests need.
type idleWin struct {
	held map[platform.Key]bool
}

func (w *idleWin) Present(*platform.Framebuffer) error { return nil }
func (w *idleWin) PollEvents() []platform.Event        { return nil }
func (w *idleWin) KeyDown(k platform.Key) bool         { return w.held[k] }
func (w *idleWin) SetTitle(string) error               { return nil }
func (w *idleWin) Close() error                        { return nil }

// raceHouse is the house every race here is over, from the built-in tree, which every build has.
func raceHouse() houseRef {
	return houseRef{Name: "Grand Prix", FS: assets.Tree(), Rel: "houses/Grand Prix.house"}
}

// raceApp is one end of a race: playApp's bench run -- unpaced, quiet, no scores and no saves,
// and hermetic, so both of the race's deadlines are kept -- in front of an idle window. flying
// holds player 1's right key for the whole run; otherwise nobody touches anything.
//
// The framebuffer has no pixels, because the window throws its frames away and converting one
// is most of what a present costs: ToBGRX leaves alone any row the destination has no room for,
// which here is all of them. Under the race detector, where every byte stored is a call, that
// is the difference between these tests taking seconds and taking a minute and a half.
//
// The engine fingerprint is a stand-in, loopbackEngine, because the real one is half a second of
// flying that under the race detector becomes twenty-five, and every race here would pay it.
// What the fingerprint is has its own test (TestTheEngineFingerprintIsPinned); what these check is
// that the one there is gets sent, compared and refused on.
func raceApp(t *testing.T, flying bool) *app {
	t.Helper()
	a, _ := playApp(t, savesNone)
	win := &idleWin{held: map[platform.Key]bool{}}
	if flying {
		win.held[a.p.Player1.Keys().Right] = true
	}
	a.win = win
	a.fb = &platform.Framebuffer{}
	a.engine = func() (uint64, error) { return loopbackEngine, nil }
	return a
}

// loopbackEngine is the engine every raceApp claims.
const loopbackEngine = 0x6C6F6F706261636B

// hostOnLoopback makes a host bind 127.0.0.1 on a port the kernel picks, and returns where to
// dial it. The address arrives once the listener exists, which is before the waiting screen goes
// up, so a guest started on it never meets a refusal.
func hostOnLoopback(a *app) <-chan string {
	addr := make(chan string, 1)
	a.listen = func(string) (*netplay.Listener, error) {
		l, err := netplay.Listen("127.0.0.1", "0")
		if err == nil {
			addr <- l.Addr()
		}
		return l, err
	}
	return addr
}

// played is what a.play returned, from the goroutine it ran on.
type played struct {
	out shell.Outcome
	err error
}

func playInBackground(a *app, race shell.Race) <-chan played {
	done := make(chan played, 1)
	go func() {
		out, err := a.play(raceHouse(), false, false, race)
		done <- played{out, err}
	}()
	return done
}

// A minute is far past anything these take -- well under a second each, under the race detector
// too -- and far short of go test's ten-minute default, so a hang fails with a name on it.
const loopbackPatience = time.Minute

func await(t *testing.T, what string, done <-chan played) played {
	t.Helper()
	select {
	case p := <-done:
		return p
	case <-time.After(loopbackPatience):
		t.Fatalf("the %s's game had not returned after %v", what, loopbackPatience)
	}
	return played{}
}

func awaitAddr(t *testing.T, addr <-chan string, host <-chan played) string {
	t.Helper()
	select {
	case s := <-addr:
		return s
	case p := <-host:
		t.Fatalf("the host returned before it listened: %+v", p)
	case <-time.After(loopbackPatience):
		t.Fatal("the host never listened")
	}
	return ""
}

// raceWaits shortens the race's two deadlines for one test, and puts them back after it.
func raceWaits(t *testing.T, connect, settle time.Duration) {
	t.Helper()
	oldConnect, oldSettle := raceConnectWait, raceSettleWait
	raceConnectWait, raceSettleWait = connect, settle
	t.Cleanup(func() { raceConnectWait, raceSettleWait = oldConnect, oldSettle })
}

// raceBoth runs a whole race: a host, and a guest dialling the address the host got.
func raceBoth(t *testing.T, hostFlies, guestFlies bool) (host, guest *app, hp, gp played) {
	t.Helper()
	host, guest = raceApp(t, hostFlies), raceApp(t, guestFlies)
	addr := hostOnLoopback(host)
	hostDone := playInBackground(host, shell.Race{Host: true})
	guestDone := playInBackground(guest, shell.Race{Join: awaitAddr(t, addr, hostDone)})

	hp, gp = await(t, "host", hostDone), await(t, "guest", guestDone)
	for _, s := range []struct {
		what string
		a    *app
		p    played
	}{{"host", host, hp}, {"guest", guest, gp}} {
		if s.p.err != nil {
			t.Fatalf("the %s's race failed: %v", s.what, s.p.err)
		}
		if s.a.raced == nil {
			t.Fatalf("the %s played a game and recorded no race", s.what)
		}
	}
	return host, guest, hp, gp
}

// The ordinary race, and the property the mode rests on: **two machines with no referee compute
// the same result.** Each side scored the other's run exactly as the other ran it, both hold the
// same match from opposite slots, and both arrive at the same winner for the same reason.
//
// One side flies and the other does not, so that there is a winner to agree about. Which one wins
// is not asserted: the seed comes from two nonces, so it is different every run, and holding right
// is not a strategy anybody should have to defend in a test.
func TestLoopbackRaceIsDecidedTheSameOnBothSides(t *testing.T) {
	raceWaits(t, 20*time.Second, 20*time.Second)
	host, guest, hp, gp := raceBoth(t, false, true)
	h, g := host.raced, guest.raced

	if h.match.ID != g.match.ID || h.match.Seed != g.match.Seed {
		t.Errorf("the two sides agreed different matches: %08X/%016X and %08X/%016X",
			h.match.ID, h.match.Seed, g.match.ID, g.match.Seed)
	}
	if h.match.Slot == g.match.Slot {
		t.Errorf("both sides are player %d", h.match.Slot+1)
	}
	if h.theirs != g.mine || g.theirs != h.mine {
		t.Errorf("a side scored the other's run as something it did not fly:\n"+
			"  host:  mine %+v, theirs %+v\n  guest: mine %+v, theirs %+v",
			h.mine, h.theirs, g.mine, g.theirs)
	}
	for _, s := range []struct {
		what string
		r    *raceResult
	}{{"host", h}, {"guest", g}} {
		if !s.r.settled {
			t.Errorf("the %s gave up waiting for the other run (raceSettleWait) rather than "+
				"hearing it end", s.what)
		}
		if s.r.err != nil {
			t.Errorf("the %s's connection failed: %v", s.what, s.r.err)
		}
		if st := s.r.mine.State; st != netplay.Died && st != netplay.Finished {
			t.Errorf("the %s's run ended %v; in Grand Prix an idle glider dies, so the run "+
				"should have ended by itself", s.what, st)
		}
	}

	if h.out != g.out {
		t.Fatalf("the two sides disagree about the result: host %v, guest %v", h.out, g.out)
	}
	if h.out.Slot < 0 || h.out.Reason == netplay.ByForfeit {
		t.Fatalf("the race came out %v; a glider that flew against one that did not should "+
			"have beaten it or lost to it in the air", h.out)
	}
	if v := h.verdict() + " / " + g.verdict(); v != "you win / you lose" && v != "you lose / you win" {
		t.Errorf("the two screens say %q", v)
	}
	if hp.out.Race != h.band() || gp.out.Race != g.band() || hp.out.Race == "" {
		t.Errorf("play handed the shell %q and %q for the band, want %q and %q",
			hp.out.Race, gp.out.Race, h.band(), g.band())
	}
}

// Two gliders nobody touches, from one agreed seed, fly the same run to the frame: a draw on
// every comparison Winner makes. This is the check that the seed the handshake agreed is the seed
// both Worlds ran from, because the two apps start from different streams, and a side that ignored
// the match would carry on from its own.
//
// It is also the draw docs/analysis/determinism-networking.md §10.4.9 proposes as a test vector,
// arrived at with no recording at all.
func TestLoopbackRaceOfTwoIdleGlidersIsADraw(t *testing.T) {
	raceWaits(t, 20*time.Second, 20*time.Second)
	host, guest := raceApp(t, false), raceApp(t, false)
	host.randSeed, guest.randSeed = 16807, 282475249 // the first two draws of the default stream

	addr := hostOnLoopback(host)
	hostDone := playInBackground(host, shell.Race{Host: true})
	guestDone := playInBackground(guest, shell.Race{Join: awaitAddr(t, addr, hostDone)})
	hp, gp := await(t, "host", hostDone), await(t, "guest", guestDone)
	if hp.err != nil || gp.err != nil {
		t.Fatalf("the race failed: host %v, guest %v", hp.err, gp.err)
	}

	h, g := host.raced, guest.raced
	if h.mine != g.mine {
		t.Errorf("two idle gliders on one seed flew different runs:\n  host  %+v\n  guest %+v",
			h.mine, g.mine)
	}
	if host.randSeed != guest.randSeed {
		t.Errorf("the random streams ended at %d and %d; both Worlds should have run from the "+
			"match's seed", host.randSeed, guest.randSeed)
	}
	if want := (netplay.Outcome{Slot: -1, Reason: netplay.Drawn}); h.out != want || g.out != want {
		t.Errorf("the race came out %v and %v, want %v", h.out, g.out, want)
	}
	if h.verdict() != "a draw" || hp.out.Race != "race: a draw" {
		t.Errorf("a draw is headed %q and reported as %q", h.verdict(), hp.out.Race)
	}
}

// guestHello is what the other machine says about itself when it is played by hand: the same
// house, the same engine and release, and no fixes.
func guestHello(t *testing.T) netplay.Hello {
	t.Helper()
	h, err := raceHouse().open()
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := netplay.Nonce()
	if err != nil {
		t.Fatal(err)
	}
	return netplay.Hello{
		Nonce:     nonce,
		HouseName: "Grand Prix",
		HouseHash: netplay.HouseHash(canonical(t, h)),
		Neighbors: uint8(prefs.Default().Neighbors),
		Engine:    loopbackEngine,
		Release:   version,
	}
}

// meetAsGuest is the other machine played by hand: dial, and shake hands saying hello.
func meetAsGuest(t *testing.T, addr string, hello netplay.Hello) (*netplay.Conn, io.Closer, netplay.Match, error) {
	t.Helper()
	trans, err := netplay.Join(addr, "", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	c := netplay.NewConn(trans)
	m, err := netplay.Meet(c, hello)
	if err != nil {
		trans.Close()
	}
	return c, trans, m, err
}

func canonical(t *testing.T, h *house.House) []byte {
	t.Helper()
	b, err := h.Save()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A guest whose process ends mid-race -- no goodbye, just a socket closing -- has forfeited, and
// the host is told so without waiting out anything: PLAN Stage 3's "killing the guest mid-race
// leaves the host in a defined state". A process cannot be killed in-process, so the guest is a
// raw peer that meets, hears the host flying, reports one room and hangs up.
//
// The host flies on to the end of its own run first. A forfeit does not stop the other player's
// game, and a host whose glider froze when the guest's machine crashed would be a worse bug than
// the one this is checking for.
func TestLoopbackRaceGuestWhoLeavesMidRaceForfeits(t *testing.T) {
	raceWaits(t, 20*time.Second, 20*time.Second)
	host := raceApp(t, false)
	addr := hostOnLoopback(host)
	hostDone := playInBackground(host, shell.Race{Host: true})

	c, trans, m, err := meetAsGuest(t, awaitAddr(t, addr, hostDone), guestHello(t))
	if err != nil {
		t.Fatal(err)
	}
	msg, err := c.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := msg.(*netplay.Report); !ok || r.State.Ended() {
		t.Fatalf("the host's first word after the handshake was %T %+v, want a standing of a "+
			"run in progress", msg, msg)
	}
	left := netplay.Standing{Rooms: 1, Frame: 1}
	if err := c.SendStanding(m.Slot, left); err != nil {
		t.Fatal(err)
	}
	trans.Close()

	hp := await(t, "host", hostDone)
	if hp.err != nil {
		t.Fatal(hp.err)
	}
	r := host.raced
	if r == nil {
		t.Fatal("the host recorded no race")
	}
	if !r.settled {
		t.Error("the host waited out raceSettleWait for a guest that had already gone")
	}
	if r.mine.State != netplay.Died {
		t.Errorf("the host's run ended %v; its glider should have flown on and died", r.mine.State)
	}
	if want := netplay.Abandoned(left); r.theirs != want {
		t.Errorf("the guest is scored as %+v, want its last standing folded as a departure, %+v",
			r.theirs, want)
	}
	if want := (netplay.Outcome{Slot: int8(r.match.Slot), Reason: netplay.ByForfeit}); r.out != want {
		t.Errorf("the race came out %v, want %v", r.out, want)
	}
	if r.verdict() != "you win" {
		t.Errorf("the host's screen says %q", r.verdict())
	}
	// r.err is not checked, and is nearly always set. A process that ends with the host's
	// standings unread in its socket -- which, mid-race, is every process that ends -- resets
	// the connection rather than closing it, so the host's next write is refused and its result
	// screen adds "the connection failed". From the host's end that is what happened. What it
	// does not do is say so in words: the line is Go's, "write: broken pipe" here and
	// something else on Windows (docs/IMPROVEMENTS.md 4.33).
	t.Logf("the host reports: %v", r.err)
}

// A guest on another engine is refused by the host, and the host by it: **both** sides refuse,
// before either flies, and both name the other's release. docs/IMPROVEMENTS.md 4.31 is the reason
// both matters. A refusal only one side makes leaves the other -- whichever is player 1 -- with a
// match, a glider in the air, and a peer that hung up, which it scores as a forfeit: a win shown
// to a player who never raced anybody.
//
// Which of the two is player 1 is the nonces' to decide and differs from run to run, so the
// property is checked whichever way round it came out.
func TestLoopbackRaceWithAnotherEngineIsRefusedByBothSides(t *testing.T) {
	raceWaits(t, 20*time.Second, 20*time.Second)
	host := raceApp(t, false)
	addr := hostOnLoopback(host)
	hostDone := playInBackground(host, shell.Race{Host: true})

	hello := guestHello(t)
	hello.Engine, hello.Release = loopbackEngine+1, "9.9.9"
	_, _, _, gerr := meetAsGuest(t, awaitAddr(t, addr, hostDone), hello)
	hp := await(t, "host", hostDone)

	for _, side := range []struct {
		who, names string
		err        error
	}{{"host", `release "9.9.9"`, hp.err}, {"guest", `release "` + version + `"`, gerr}} {
		if !errors.Is(side.err, netplay.ErrEngine) {
			t.Errorf("the %s's handshake ended %v, want %v", side.who, side.err, netplay.ErrEngine)
			continue
		}
		if !strings.Contains(side.err.Error(), side.names) {
			t.Errorf("the %s's refusal does not name the other side, %s: %v",
				side.who, side.names, side.err)
		}
	}
	if host.raced != nil {
		t.Errorf("the host recorded a race it refused: %+v", host.raced)
	}
}

// A host that accepts and then says nothing leaves a guest parked in the handshake, and a bench
// run has nobody to press Escape. raceConnectWait ends it: play returns an error rather than
// hanging, and lets go of the socket, which the silent host sees as the guest leaving.
//
// The guest does speak first -- Meet sends its hello before it reads one -- so the silent end
// hears that, and then the end of the stream.
//
// What the error *says* is 4.32 and 4.33's business, and this does not pin it: today it is
// raceTimedOut's "could not join a race", for a guest that joined perfectly well.
func TestLoopbackRaceHostThatNeverSpeaksTimesOut(t *testing.T) {
	const wait = 500 * time.Millisecond
	raceWaits(t, wait, 20*time.Second)

	l, err := netplay.Listen("127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	heard := make(chan []error, 1)
	go func() {
		trans, err := l.Accept()
		if err != nil {
			heard <- []error{err}
			return
		}
		defer trans.Close()
		c := netplay.NewConn(trans)
		var got []error
		for i := 0; i < 2; i++ {
			msg, err := c.Recv()
			if err == nil {
				if _, ok := msg.(*netplay.Hello); !ok {
					err = errors.New("the guest sent something other than a hello")
				}
			}
			got = append(got, err)
		}
		heard <- got
	}()

	guest := raceApp(t, false)
	start := time.Now()
	gp := await(t, "guest", playInBackground(guest, shell.Race{Join: l.Addr()}))
	took := time.Since(start)

	switch {
	case gp.err == nil:
		t.Fatalf("a host that never spoke started a race: %+v", gp.out)
	case errors.Is(gp.err, errRaceGaveUp), errors.Is(gp.err, errRaceClosed):
		t.Errorf("the guest says %q, which is a player's choice, and nobody chose", gp.err)
	}
	if took > wait+10*time.Second {
		t.Errorf("the guest took %v to give up, with raceConnectWait at %v", took, wait)
	}
	if guest.raced != nil {
		t.Errorf("a race that never started recorded a result: %+v", guest.raced)
	}

	select {
	case got := <-heard:
		if len(got) != 2 || got[0] != nil {
			t.Fatalf("the silent host heard %v, want the guest's hello first", got)
		}
		if !errors.Is(got[1], io.EOF) {
			t.Errorf("after the hello the silent host got %v, want the end of the stream: "+
				"the guest should have closed its socket when it gave up", got[1])
		}
	case <-time.After(loopbackPatience):
		t.Fatal("the guest gave up and never closed its socket")
	}
}
