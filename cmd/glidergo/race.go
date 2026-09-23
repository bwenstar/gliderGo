package main

// The race mode's host half: two machines, two houses, one result.
//
// internal/netplay is the protocol and the driver. This file is everything a *player* meets --
// the flags, the screen that waits for the other machine, the line that says how the opponent is
// doing, and the screen that says who won -- and it is the same division play.go draws for a
// single game: nothing in here knows what a Standing is worth, and nothing in internal/netplay
// knows what a window is.
//
// Three things about the mode are decisions rather than mechanics, and this is where a player
// meets the consequences of all three:
//
//   - **The two peers simulate separate worlds and each one only flies its own glider.** So this
//     file has no input to send and none to receive: it sends a progress record and draws the
//     one it got back. A race is therefore playable over a link that a lock-step match could
//     not survive, which is why docs/PLAN.md Stage 3 is a race and §10.2's lock-step mode is
//     still a document.
//   - **There is no starting gun.** Meet returns as soon as the match is agreed, so one machine
//     may be flying while the other is still loading art. Nothing the race measures is measured
//     in wall clock (netplay.Winner), so that costs nothing -- and it is why the waiting screen
//     below waits for a *connection* and never for a countdown.
//   - **A race is not the original's two-player game.** That one is two gliders in one room on
//     one keyboard and it is World.TwoPlayer; this one is two people who agreed to fly the same
//     house and see who gets further. The original had no network mode at all, so there is no
//     1994 behaviour to be faithful to here -- only a 1994 house, hashed and agreed, and a
//     metric the original already counted (CountRoomsVisited).
//
// The port's own restraint: **the game loop is not allowed to wait for the network, ever.** Every
// blocking call in this file happens on a screen whose whole job is waiting, with an Escape on
// it. Between those screens the frame loop touches netplay exactly twice per frame -- one
// comparison in Report and one read in Opponent -- and neither can block on the other machine.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bwenstar/gliderGo/assets"
	"github.com/bwenstar/gliderGo/internal/game"
	"github.com/bwenstar/gliderGo/internal/house"
	"github.com/bwenstar/gliderGo/internal/netplay"
	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
	"github.com/bwenstar/gliderGo/internal/render"
	"github.com/bwenstar/gliderGo/internal/replay"
	"github.com/bwenstar/gliderGo/internal/shell"
)

// raceRequested reports whether the *flags* asked for a race. One function rather than the
// condition written out at its two call sites -- run()'s dispatch and parseFlags' refusals --
// because the two must agree about what a race is.
//
// play does not use it any more, and that is the shape of the Race screen's change: a race can
// now be arranged from the title screen as well as from a command line, so what play is handed
// is a shell.Race (Wanted() is the same question asked of the arrangement rather than of the
// flags) and the flags' only remaining job is to make one before the shell ever opens.
func raceRequested(o *options) bool { return o.host || o.join != "" }

// asRace is the command line's arrangement, in the form the shell would have produced. It is
// what keeps playDirect and the shell's Play on the same path through play(): one race type,
// made in two places, understood in one.
func asRace(o *options) shell.Race { return shell.Race{Host: o.host, Join: o.join} }

// The two ways the connecting screens end without a match, both of them the player's choice and
// neither of them a failure. play turns each into a return rather than into a message on stderr:
// somebody who pressed Escape has been told what happened by the screen they pressed it on.
var (
	errRaceGaveUp = errors.New("the race was abandoned before it started")
	errRaceClosed = errors.New("the window closed before the race started")
)

// How long one dial attempt waits, and how long the guest sits between attempts.
//
// **The guest retries, and that is the one piece of this mode that is a convenience rather than
// a mechanism.** The commonest way a race fails to start is the guest pressing join before the
// host pressed host, which arrives as "connection refused" within a millisecond on a LAN. The
// alternative to retrying is a program that exits, and two people on the telephone counting to
// three.
//
// **Three seconds, because Windows takes two to say "refused".** A Windows guest dialling a
// closed port tries again twice before it reports the refusal (Go's fd_windows.go), so with the
// one second this used to be, every refusal on Windows arrived as a timeout, and a timeout
// cannot tell "not hosting yet" from a firewall (joinWords). The wait costs Escape nothing: the
// dial runs under rc's context, and Escape ends that (docs/IMPROVEMENTS.md 4.33).
const (
	joinTimeout = 3 * time.Second
	joinRetry   = time.Second / 2
)

// The two deadlines a race keeps, and they exist for the same reason: **a run with nobody in
// front of it has no Escape key.**
//
// A player waiting for the other machine waits as long as they like, because the screen they are
// looking at says how to stop. A `-frames`, `-bench` or `-dump` run is a Makefile or a test, its
// window is the null backend and it will never see a keystroke, so every wait in this file that a
// player can end has to end by itself there instead -- otherwise `-join` an address that will
// never answer is a hang, and a hang is the one failure a script cannot report. That is the same
// argument endlessHeadlessRun in main.go makes about a null build with no frame limit, arrived at
// from the other end.
//
// hermetic (cmd/glidergo/prefs.go) is the test for which kind of run this is. It is that
// function and not a fresh condition because "is this a measurement" already had one name.
//
// Variables rather than constants for one reader, loopback_test.go, whose races have nobody at
// them either and which should not spend thirty seconds learning that a host built to say nothing
// has said nothing.
var (
	raceConnectWait = 30 * time.Second
	raceSettleWait  = 10 * time.Second
)

// How long each end gives one connection to finish the handshake (netplay.MeetWithin). These
// hold in every run, a player's as well as a measurement's, because what they end is not a wait
// anybody chose: a connection that never speaks is not going to race.
//
// **The guest's is the longer, and it has to be.** The host deals with one connection at a time,
// so a real guest can sit in the kernel's backlog while the host waits out a stray connection
// ahead of it. The guest's handshake clock is running all that time, and fifteen seconds covers
// two strays at five. Five is far more than a real handshake takes, which is two messages each
// way and no arithmetic, since the engine fingerprint is worked out before either end connects.
//
// Variables for loopback_test.go, which has no five seconds to spend on each silent connection.
var (
	hostHandshakeWait  = 5 * time.Second
	guestHandshakeWait = 15 * time.Second
)

// raceConnect is the connecting half of a race, as the *screen* sees it: a handle on whatever
// the connecting goroutine is parked in, so that Escape can get it back.
//
// It exists because two of the three blocking calls before a match cannot be cancelled by
// asking. Accept is in the kernel, and Meet's first Recv is a read on a socket nobody has written
// to yet. What releases each of them is closing the thing underneath, from another goroutine --
// which is netplay.Listener's own documented answer for Accept and Conn's for Recv -- so the
// goroutine hands each one over as it gets it and the screen closes whatever it is holding. The
// third, the dial, takes a context, and cancel ends that.
//
// cancelled is checked before every hand-over as well as being acted on in cancel, and both
// halves are needed: a cancel that lands between the dial returning and the hand-over would
// otherwise leave a connected socket with nobody to close it.
type raceConnect struct {
	ctx  context.Context
	stop context.CancelFunc

	mu        sync.Mutex
	cancelled bool
	l         *netplay.Listener
	c         net.Conn
	last      raceNote // the most recent failure, for the waiting screen to show
}

// raceNote is a failure the waiting screen shows, and how many dials in a row have gone
// unanswered, which decides how loudly a firewall is named (joinWords).
type raceNote struct {
	err      error
	timeouts int
}

func newRaceConnect() *raceConnect {
	rc := &raceConnect{}
	rc.ctx, rc.stop = context.WithCancel(context.Background())
	return rc
}

// keep hands the screen something to close. It reports false if Escape has already happened, in
// which case the caller owns the thing it was about to hand over and must close it itself.
func (rc *raceConnect) keep(l *netplay.Listener, c net.Conn) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.cancelled {
		return false
	}
	if l != nil {
		rc.l = l
	}
	if c != nil {
		rc.c = c
	}
	return true
}

// cancel closes whatever the connecting goroutine is parked in, whichever of the three it is.
func (rc *raceConnect) cancel() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cancelled = true
	rc.stop()
	if rc.l != nil {
		rc.l.Close()
	}
	if rc.c != nil {
		rc.c.Close()
	}
}

func (rc *raceConnect) stopped() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.cancelled
}

// note records a failed dial or a turned-away connection for the screen, and lastNote reads
// it. A refusal the player cannot see is a screen that says "joining" forever: a typed-wrong host
// name fails with "no such host" on every attempt, and that sentence is the difference between
// pressing Escape now and waiting to find out. On the host's side it is the guest who opened
// another house, and it is the hosting player who can tell them so.
func (rc *raceConnect) note(err error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	var je *netplay.JoinError
	if errors.As(err, &je) && je.Why == netplay.JoinTimedOut {
		rc.last = raceNote{err: err, timeouts: rc.last.timeouts + 1}
		return
	}
	rc.last = raceNote{err: err}
}

func (rc *raceConnect) lastNote() raceNote {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.last
}

// raceMeeting is what the connecting goroutine returns: an agreed match, or why there is none.
type raceMeeting struct {
	c     *netplay.Conn
	trans net.Conn
	m     netplay.Match
	err   error
}

// openRace is everything between the flags and a Race the frame loop can talk to: bind or dial,
// shake hands, and hand back the driver together with the transport the caller has to close.
//
// The house is a parameter and not a name because the gate is the house's *content*: HouseHash
// over the canonical encoding, so two players who typed the same house name and have different
// files are told so before either of them flies. Called from play after -room has been applied
// and a resume refused, which is why it is called from there and not from playDirect -- -room
// rewrites h.FirstRoom and the hash covers it, so two players who disagree about the starting
// room are not racing the same house either.
//
// Which end this machine is comes in as an argument, because since the Race screen existed it is
// no longer a property of the flags: `-host` and the title screen's Host row arrive here as the
// same shell.Race and get the same handshake. The port is still the flag's, because a port is a
// property of the machine rather than of one arrangement made on it -- see the -port help text.
func (a *app) openRace(name string, h *house.House, race shell.Race) (*netplay.Race, net.Conn, error) {
	o := a.o

	canon, err := h.Save()
	if err != nil {
		// A house that cannot be re-encoded cannot be hashed, and a race without the hash is
		// a race with no gate at all. Refusing is the only honest answer; the same house
		// still plays perfectly well on its own.
		return nil, nil, fmt.Errorf("%s cannot be hashed for a race: %w", name, err)
	}
	nonce, err := netplay.Nonce()
	if err != nil {
		return nil, nil, err
	}
	// Engine is left for meetRace, which works it out on the connecting goroutine: it takes
	// half a second, and this goroutine is the one drawing the window.
	local := netplay.Hello{
		Nonce:     nonce,
		HouseName: name,
		HouseHash: netplay.HouseHash(canon),
		Neighbors: uint8(a.p.Neighbors),
		Rules:     raceRules(a.p.Fixes),
		Release:   version,
	}

	// **Bind before the waiting screen goes up.** A port already in use is the commonest way
	// hosting fails and netplay.Listen is separate from Accept so that it can be reported here
	// -- as a refusal to start, which is what it is -- rather than behind a screen that says
	// the other player has not arrived yet.
	var l *netplay.Listener
	var heading string
	var where []string
	if race.Host {
		if l, err = a.hostListener(o.port); err != nil {
			return nil, nil, err
		}
		heading = "WAITING FOR THE OTHER PLAYER"
		where = hostingLines(l.Addr(), name)
	} else {
		heading = "JOINING"
		where = []string{
			"connecting to " + netplay.Address(race.Join, o.port),
			"the other machine has to be hosting " + name,
		}
	}

	rc := newRaceConnect()
	if l != nil {
		rc.keep(l, nil) // nothing can have cancelled yet; the screen is not up
	}
	res := make(chan raceMeeting, 1)
	go func() { res <- a.meetRace(rc, l, race.Join, local) }()

	met, err := a.awaitRace(rc, race, heading, where, res)
	if errors.Is(err, errRaceGaveUp) || errors.Is(err, errRaceClosed) {
		return nil, nil, err
	}
	if err != nil {
		// The terminal gets the sentence and the status band the line (briefErr): a
		// refusal ends the race and the band is where a player reads why.
		return nil, nil, briefErr{brief: raceWords(err, race.Host, 0), err: err}
	}

	r := netplay.Start(met.c, met.m)
	if !o.quiet {
		// The line a bug report about a race has to quote, and the one place both halves of
		// the agreement are visible at once: which player this side is, the match the two
		// nonces produced, the seed the house will actually run from, what the other side
		// calls the house it hashed the same as this one, and which release it is and what
		// it plays with -- the rules that do not decide a race, and so were let through.
		fmt.Printf("glidergo: race: player %d, match 0x%08X, seed %d, opponent %q with %d-room "+
			"view, release %q, rules: %v\n",
			met.m.Slot+1, met.m.ID, met.m.RandSeed(), met.m.PeerHouseName, met.m.PeerNeighbors,
			met.m.PeerRelease, met.m.PeerRules)
	}
	return r, met.trans, nil
}

// raceEngine is this build's engine fingerprint, which a race's hello carries, or the stand-in a
// test has put in a.engine.
func (a *app) raceEngine() (uint64, error) {
	if a.engine != nil {
		return a.engine()
	}
	return engineFingerprint()
}

// engineFingerprint is replay.Engine over the built-in tree, worked out the first time a race
// asks for it and kept for the rest of the process: it cannot change while the binary runs, and a
// second race in one session should not pay for it again.
//
// On the built-in tree and never on -assets or -houses, because what the fingerprint stands for
// is the binary. A race over a house from a directory is still gated on that house by its own
// hash.
var engineFingerprint = sync.OnceValues(func() (uint64, error) {
	return replay.Engine(assets.Tree())
})

// raceRules is a player's fixes as the race's hello carries them. A function and not a literal at
// the call site for the reason gameFixes is one: TestEveryFixHasARaceRule reaches it, and a fix
// added to prefs.Fixes and not here would be one the other player is never told about.
func raceRules(f prefs.Fixes) netplay.Rules {
	var r netplay.Rules
	for _, fix := range []struct {
		on   bool
		rule netplay.Rules
	}{
		{f.MirrorFlame, netplay.RuleMirrorFlame},
		{f.MirrorFoil, netplay.RuleMirrorFoil},
		{f.SwitchSparkle, netplay.RuleSwitchSparkle},
		{f.Player2GiveUp, netplay.RulePlayer2GiveUp},
	} {
		if fix.on {
			r |= fix.rule
		}
	}
	return r
}

// hostListener binds the port a hosted race waits on: every interface, because the other player
// is on another machine -- unless a test has set a.listen, which binds loopback instead.
func (a *app) hostListener(port string) (*netplay.Listener, error) {
	if a.listen != nil {
		return a.listen(port)
	}
	return netplay.Listen("", port)
}

// meetRace is the connecting goroutine: accept or dial, then shake hands. Everything in it
// blocks, which is why it is not on the frame loop's goroutine and why every blocking call has
// been handed to rc first.
func (a *app) meetRace(rc *raceConnect, l *netplay.Listener, join string, local netplay.Hello) raceMeeting {
	if l != nil {
		// Shut however this ends: a match is two players, and a host that has gone back to
		// the title screen is not hosting.
		defer l.Close()
	}

	// The engine before the connection, so that once there is one the hello goes out at once
	// and the other machine is not left waiting on this one's arithmetic. A player who gives
	// up meanwhile is heard below, when the listener or the dial finds rc cancelled.
	eng, err := a.raceEngine()
	if err != nil {
		// A build that cannot say what its engine is cannot be told apart from one with
		// another engine, so it does not race -- and nothing but a broken build gets here,
		// since the fingerprint is made from what the binary carries.
		return raceMeeting{err: fmt.Errorf("this build cannot race: %w", err)}
	}
	local.Engine = eng

	if l != nil {
		return a.hostRace(rc, l, local)
	}
	trans, err := a.dialRace(rc, join)
	if err != nil {
		return raceMeeting{err: err}
	}
	return shake(rc, trans, local, guestHandshakeWait)
}

// hostRace is the host's half: take connections until one of them is a race.
//
// **A connection that does not become a match is turned away, and the host goes on waiting**
// (docs/IMPROVEMENTS.md 4.32). The commonest one is a guest who opened another house, and the
// fix is on their machine: this one should still be hosting when they have opened the right one.
// So every handshake failure turns the connection away, whatever it was: another house, another
// release, a browser, a port scanner, a connection that never spoke, or the nonce tie. There is
// no list of which errors count, because a new refusal in netplay should not end hosting because
// nobody added it to one. The one that says why goes on the waiting screen, and on stdout for a
// bug report.
//
// A refusal is decided from the two hellos, so the guest refused this host at the same moment
// (netplay.Meet), and turning it away cannot leave a guest flying a race the host has left.
func (a *app) hostRace(rc *raceConnect, l *netplay.Listener, local netplay.Hello) raceMeeting {
	for {
		trans, err := l.Accept()
		if err != nil {
			if rc.stopped() {
				// The error is this port closing the listener out from under a blocked
				// Accept, so it describes the cancellation and not a failure. Saying so is
				// what keeps "Accept: use of closed network connection" off a player's
				// screen.
				return raceMeeting{err: errRaceGaveUp}
			}
			return raceMeeting{err: err}
		}
		met := shake(rc, trans, local, hostHandshakeWait)
		if met.err == nil || errors.Is(met.err, errRaceGaveUp) {
			return met
		}
		turned := &turnedAway{host: remoteHost(trans), err: met.err}
		rc.note(turned)
		if !a.o.quiet {
			fmt.Printf("glidergo: race: %v\n", turned)
		}
	}
}

// shake is the handshake on one connection, with the connection handed to rc first so that
// Escape can close it.
func shake(rc *raceConnect, trans net.Conn, local netplay.Hello, wait time.Duration) raceMeeting {
	if !rc.keep(nil, trans) {
		trans.Close()
		return raceMeeting{err: errRaceGaveUp}
	}
	c := netplay.NewConn(trans)
	m, err := netplay.MeetWithin(c, local, wait)
	if err != nil {
		trans.Close()
		if rc.stopped() {
			return raceMeeting{err: errRaceGaveUp}
		}
		// Not wrapped. netplay.Meet's refusals -- another house, another engine, another
		// rule -- are the messages in this package a player is meant to read, and each one
		// already says what both sides have and what to do about it.
		return raceMeeting{err: err}
	}
	return raceMeeting{c: c, trans: trans, m: m}
}

// remoteHost is the machine at the other end of a connection, without its port. The port is a
// number the other machine's kernel picked and says nothing to a player. The machine is what
// tells the guest they were waiting for apart from a stray.
func remoteHost(c net.Conn) string {
	addr := c.RemoteAddr().String()
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}

// refusedByPeer reports whether a race failed to start because of what is on the other end,
// rather than because of anything this program got wrong: another release, house or rule, or
// another program altogether (netplay.ErrMagic). main leaves the bug-report footer off these.
// Each message already names both sides and what to change. A footer that invites a bug report
// about a friend's other house is the kind of wrong main's comment says makes a footer
// invisible.
//
// A handshake that timed out is not here. The other end could be a gliderGo host that is stuck,
// and that one is worth a report.
func refusedByPeer(err error) bool {
	for _, e := range []error{netplay.ErrMagic, netplay.ErrVersion, netplay.ErrEngine,
		netplay.ErrHouse, netplay.ErrRules} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// dialRace is the guest's half: dial until somebody answers or the player gives up. Unlike the
// host, a guest does not try again once it has connected. A refusal from the host is about
// something on this machine or that one, and dialling again would only be refused again.
func (a *app) dialRace(rc *raceConnect, join string) (net.Conn, error) {
	for {
		c, err := netplay.Join(rc.ctx, join, a.o.port, joinTimeout)
		if err == nil {
			return c, nil
		}
		if rc.stopped() {
			return nil, errRaceGaveUp
		}
		rc.note(err)
		select {
		case <-rc.ctx.Done():
			return nil, errRaceGaveUp
		case <-time.After(joinRetry):
		}
	}
}

// awaitRace is the waiting screen: it pumps the window, draws, and returns when the connecting
// goroutine does or when the player has had enough.
//
// Escape and the close box, and nothing else. A race is a thing two people arranged, so an
// accidental keystroke must not undo it -- which is the opposite of the high-score screens, where
// any key means "I have read this". The way out is written on the screen, which is the same rule
// the pause hint follows.
func (a *app) awaitRace(rc *raceConnect, race shell.Race, heading string, where []string, res <-chan raceMeeting) (raceMeeting, error) {
	scr := raceSurface()
	var deadline <-chan time.Time
	if hermetic(a.o) {
		deadline = time.After(raceConnectWait)
	}
	for {
		// The goroutine first, so that a match that is already agreed is not delayed by a
		// poll, and so that a window closing in the same pass as a successful handshake does
		// not throw the match away.
		select {
		case met := <-res:
			return met, met.err
		case <-deadline:
			// A nil channel never fires, so a player's wait never comes through here. See
			// raceConnectWait.
			rc.cancel()
			return raceMeeting{}, raceTimedOut(race, rc)
		default:
		}

		for _, ev := range a.win.PollEvents() {
			switch {
			case ev.Kind == platform.EventQuit:
				rc.cancel()
				return raceMeeting{}, errRaceClosed
			case ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyEscape:
				rc.cancel()
				return raceMeeting{}, errRaceGaveUp
			}
		}

		lines := append([]string{}, where...)
		if note := rc.lastNote(); note.err != nil {
			lines = append(lines, "", raceWords(note.err, race.Host, note.timeouts))
		}
		lines = append(lines, "", "press Esc to give up")
		raceScreen(scr, heading, lines, nil)
		if err := a.presentSurface(scr); err != nil {
			rc.cancel()
			return raceMeeting{}, errRaceClosed
		}
		time.Sleep(pollWait)
	}
}

// raceTimedOut is the message a measurement run gets instead of an Escape key. Both versions
// carry the last note, because after thirty seconds of "connection refused", or of turning away
// a guest with another house, the note is the whole of what went wrong.
func raceTimedOut(race shell.Race, rc *raceConnect) error {
	err := rc.lastNote().err
	if race.Host {
		if err != nil {
			return fmt.Errorf("no race started within %v; %w", raceConnectWait, err)
		}
		return fmt.Errorf("nobody joined the race within %v", raceConnectWait)
	}
	if err != nil {
		return fmt.Errorf("could not join a race within %v: %w", raceConnectWait, err)
	}
	return fmt.Errorf("could not join a race within %v", raceConnectWait)
}

// hostingLines is what the waiting host reads out to the other player.
//
// The port comes from the listener rather than from the flag, because a host that asked for port
// 0 -- or whose 1994 was taken -- still has to be able to say where to connect. The addresses are
// this machine's, and they are here for one reason: the sentence a hosting player needs is the
// thing the other player has to type, and "`-join` my address" was not that sentence even when a
// command line was the only way in. A machine with none to offer says so rather than printing an
// empty line.
//
// **The address is on its own line now, and the command line has moved underneath it.** That is
// this screen's half of the Race screen's arrival: the other player opens Race..., puts the cursor
// on Join and types what is on this screen, so what this screen has to say is an address and
// nothing else. The shell's version is still printed, because the player who typed `-host` may
// well have a partner at a shell too -- and it is quoted, which it was not, because a house name
// with a space in it is two arguments and parseFlags answers two arguments with "one house at a
// time". An example a player can paste and be refused by is worse than no example at all.
func hostingLines(addr, name string) []string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = netplay.DefaultPort
	}
	lines := []string{"the other player opens Race... and types:"}
	got := localAddresses()
	first := "THIS-MACHINE:" + port
	if len(got) == 0 {
		// Nothing to offer, so the shape of the answer is offered instead: somebody who can
		// see their own address in an operating system's own settings can still read this.
		lines = append(lines, first)
	}
	for i, ip := range got {
		if i == 0 {
			first = net.JoinHostPort(ip.String(), port)
		}
		lines = append(lines, addressLine(ip, port))
	}
	lines = append(lines, "", "or, from a shell:",
		fmt.Sprintf("glidergo -join %s %s", first, shellQuote(name)))
	if runtime.GOOS == "windows" {
		// The moment this screen goes up is the moment Windows asks, because listening on
		// every interface is what its firewall asks about. The dialog does not say what
		// the program wants the network for, and cancelling it blocks the port.
		lines = append(lines, "", "if Windows asks about gliderGo, allow it on the network "+
			"you are on")
	}
	return lines
}

// addressLine is one address as the hosting screen reads it out, with its port, and marked
// when only a machine on the same network can reach it.
//
// **net.IP.IsPrivate's ranges and nothing else**: 10/8, 172.16/12, 192.168/16 and fc00::/7.
// 100.64/10 is not marked, although it is carrier-grade NAT's range too, because an interface in
// it on a player's own machine is almost always Tailscale. That is the overlay the README
// suggests for racing past a router, and it is reachable from wherever the other player's
// Tailscale is.
func addressLine(ip net.IP, port string) string {
	line := net.JoinHostPort(ip.String(), port)
	if ip.IsPrivate() {
		line += "   (this network only)"
	}
	return line
}

// shellQuote wraps a house name so that a shell hands it to the program as one argument.
//
// Double quotes rather than single, and that is the only real decision in it: this line is read on
// Windows at least as often as on a shell that understands both, and cmd.exe knows only double
// quotes. Names are quoted only when they need it, so that the ordinary case reads as a name and
// not as a string literal. A name containing a double quote is left exactly as it is, because it
// cannot be quoted correctly for both shells at once, no 1994 house has one, and the line above
// this one is the line the player is meant to use anyway.
func shellQuote(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, `"`) || !strings.ContainsAny(s, " \t'\\$&|<>()^;,=%!`*?~#") {
		return s
	}
	return `"` + s + `"`
}

// localAddresses is this machine's addresses, best first, for the lines above.
//
// Interfaces that are down and loopback are dropped: neither is an address the other player can
// reach, and offering 127.0.0.1 to somebody on another machine is worse than offering nothing.
// The address the default route leaves from goes first, and then IPv4 before IPv6, because a
// LAN's IPv4 address is the one a person can read out loud. At most three, because this is a
// line on a screen and not an inventory. A machine with a dozen interfaces is a machine whose
// owner knows which one to use.
func localAddresses() []net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var v4, v6 []net.IP
	for _, in := range ifaces {
		if in.Flags&net.FlagUp == 0 || in.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := in.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.IsLinkLocalUnicast() {
				// A link-local address needs a zone to be usable and the zone is the
				// *other* machine's interface name, which this one cannot know.
				continue
			}
			if ip4 := ipn.IP.To4(); ip4 != nil {
				v4 = append(v4, ip4)
			} else {
				v6 = append(v6, ipn.IP)
			}
		}
	}
	return orderAddresses(append(v4, v6...), defaultRoute())
}

// orderAddresses moves first to the front of ips, if it is there, and keeps three.
//
// **The default route's address first, because interface order is the operating system's and not
// the player's.** On a machine with Docker, a VM host or a VPN client, the first interface up is
// often a bridge nobody else can reach. It used to be the address the pasteable command line
// used. The address the default route leaves from is the one the machine's own traffic uses, and
// on a home network that is the LAN address the other player needs.
func orderAddresses(ips []net.IP, first net.IP) []net.IP {
	var out []net.IP
	for _, ip := range ips {
		if first != nil && ip.Equal(first) {
			out = append([]net.IP{ip}, out...)
		} else {
			out = append(out, ip)
		}
	}
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

// defaultRoute is the address this machine would send from to somewhere off its own networks. It
// asks the kernel by "connecting" a UDP socket, which picks a route and a source address and
// sends nothing: a UDP connect is only bookkeeping. 192.0.2.1 is TEST-NET-1 (RFC 5737), an
// address that will never be anybody's, so that this line cannot be read as the game phoning
// anywhere. A machine with no default route gets nil, and the interfaces' own order.
func defaultRoute() net.IP {
	c, err := net.Dial("udp", "192.0.2.1:9")
	if err != nil {
		return nil
	}
	defer c.Close()
	if a, ok := c.LocalAddr().(*net.UDPAddr); ok {
		return a.IP
	}
	return nil
}

// ---------------------------------------------------------------------------
// The game's own numbers
// ---------------------------------------------------------------------------

// standingOf is the mapping from a World to what the other player is shown. Four of the six
// fields are the World's own; the two that are not are documented on netplay.Standing.
//
// State is left Racing, because a standing made during play is a standing of a run in progress
// and finalStanding is the one place that decides how a run ended.
func standingOf(w *game.World) netplay.Standing {
	s := netplay.Standing{
		// The metric, and the reason it is a walk of the house rather than a counter: it is
		// the original's own CountRoomsVisited, which is what the 1994 high-score table
		// stores. Cheap once a frame over a few thousand rooms; see wrapPresentRace for why
		// it is only called once a frame and not once a blit.
		Rooms:   w.CountRoomsVisited(),
		Room:    w.R.RoomNumber,
		Mortals: w.Mortals,
		Score:   w.Score,

		// The sender's own gameFrame, which is also the ordering rule on the wire: a
		// standing whose Frame is behind one already received is a protocol error. Present
		// is wrapped in frame order and nothing else reports, so this only ever rises.
		Frame: uint32(w.Frame),
	}
	// Floor and Suite are display only and a room index can be out of range for one frame
	// during a transition, which is what Room returning nil means. Leaving them at zero for
	// that frame is better than either guessing or refusing to report progress that did
	// happen.
	if rm := w.ThisRoom(); rm != nil {
		s.Floor, s.Suite = rm.Floor, rm.Suite
	}
	return s
}

// finalStanding is standingOf with the one field play has an answer for: how the run ended.
//
// The three-way split is internal/game/play.go:408 plus the case that line does not reach.
// GameOver is the latch FlagGameOver sets, so an unset one means the loop ended some other way --
// the player gave up from a pause, or the window closed -- and that is a forfeit rather than a
// death. Within a game over, Mortals < 0 is the C's own test: DoDiedGameOver is running out of
// gliders and DoGameOver is finishing the house.
func finalStanding(w *game.World) netplay.Standing {
	s := standingOf(w)
	switch {
	case !w.GameOver:
		s.State = netplay.Quit
	case w.Mortals < 0:
		s.State = netplay.Died
	default:
		s.State = netplay.Finished
	}
	return s
}

// wrapPresentRace reports this side's progress and draws the other side's, around whatever
// Present was already doing.
//
// **Once a frame for the report and every present for the panel**, and the asymmetry is the whole
// reason this is a wrapper and not two lines inside play's Present. Present is called once per
// frame during play and once per *wipe strip* during a room transition -- 116 or 160 times inside
// one frame (wrapPresentLimit has the long version) -- so:
//
//   - The report is gated on World.Frame changing. netplay.Race.Report would drop the duplicates
//     anyway, but standingOf walks every room in the house to count the visited ones, and doing
//     that a hundred and sixty times to compute the same answer is a waste the gate makes free.
//   - The panel is drawn on every present, because it has to be. The game composes each frame
//     from the work map and copies only the rects that changed, so the pixels under the panel are
//     restored whenever something moves there. Redrawing immediately before the blit is what
//     keeps it on screen without the panel having to be a thing the game knows about -- nothing
//     here ever writes to the work map, so the *game's* idea of the screen is untouched.
//
// The order matters in one place: the panel is drawn before prev(), because prev is the blit.
func wrapPresentRace(w *game.World, r *netplay.Race) func() {
	prev := w.Present
	last := int64(-1)
	return func() {
		if w.Frame != last {
			last = w.Frame
			r.Report(standingOf(w))
		}
		drawOpponent(w, r)
		if prev != nil {
			prev()
		}
	}
}

// The panel's geometry, in the shape internal/game/pause.go's hint row uses and for the same
// reasons: a box exactly as wide as its text, three rows of padding above and below a 9-row font,
// and a baseline measured from the box's top.
const (
	racePanelPad  = 6
	racePanelTall = 15
	racePanelBase = 10
	racePanelEdge = 4 // from the screen's top-left corner
)

// drawOpponent draws the one line this mode adds to the game's screen.
//
// **It is over the top band and that is a considered trade.** There is no free space on a
// 640x480 Glider PRO screen: the scoreboard takes twenty rows at one end or the other
// (internal/game/scoreboard.go's two modes), and with nine neighbours the bands above and below
// the central room hold slivers of the rooms next door. The top-left corner is the least costly
// of the bad options -- black in a 1- or 3-room view, and in a 9-room view a corner of the room
// to the north-west, which is decoration and not a hazard the player can fly into from here. The
// alternative was to shrink the play area, which would change every rect in the renderer and make
// a race a different game.
//
// Nothing is drawn once this side's run is over: the endings paint the whole window
// (DoGameOver floods it, DoDiedGameOver animates over it) and a progress line over the top of
// them would be this port talking during the original's last word. The result screen says
// everything the panel was saying, a moment later and in full.
func drawOpponent(w *game.World, r *netplay.Race) {
	if w.GameOver {
		return
	}
	line := opponentLine(r.Opponent())
	wide := render.StringWidth(line) + 2*racePanelPad
	box := render.SetRect(racePanelEdge, racePanelEdge,
		racePanelEdge+wide, racePanelEdge+racePanelTall)

	// Black with a white frame, deliberately unlike anything in the house: this is the port
	// talking, which is the same argument pause.go makes for the hint row's colours.
	w.Main.Fill(box, render.Black8)
	w.Main.FrameRect(box, render.White8)
	w.Main.DrawString(box.Left+racePanelPad, box.Top+racePanelBase, line, render.White8)
}

// opponentLine is the panel's text: what the other player has reached, and whether they are still
// reaching.
//
// gone is used for the words and never for the numbers, which is netplay.Race.Opponent's own
// division: the figures are what that player actually got to, and the judgement about whether
// leaving mid-flight is a forfeit belongs to netplay.Winner and to the result screen. So a peer
// that vanished while flying reads "left" here with its last real progress beside it, rather than
// having its standing rewritten on the way to the screen.
func opponentLine(s netplay.Standing, gone bool) string {
	switch {
	case gone && !s.State.Ended():
		return fmt.Sprintf("them: left -- %s, score %d", counted(int(s.Rooms), "room"), s.Score)
	case s.State.Ended():
		return fmt.Sprintf("them: %s -- %s, score %d", s.State, counted(int(s.Rooms), "room"), s.Score)
	}
	return fmt.Sprintf("them: %s, floor %d suite %d, score %d, %s",
		counted(int(s.Rooms), "room"), s.Floor, s.Suite, s.Score, counted(int(s.Mortals), "glider"))
}

// ---------------------------------------------------------------------------
// After the run
// ---------------------------------------------------------------------------

// raceResult is how one race came out, as this side saw it: the match the handshake agreed, both
// runs as the result scored them, and the result.
//
// **One value, taken once, and read by everything after the run** -- the stdout line, the result
// screen, and the title screen's status band. The standings used to be read from the Race at each
// of those, and a peer whose report arrived between two of the reads (one that had not settled
// when raceSettleWait ran out, say) could be printed as one thing and scored as another.
type raceResult struct {
	match   netplay.Match
	mine    netplay.Standing
	theirs  netplay.Standing // folded through Abandoned; see theirsAsScored
	out     netplay.Outcome
	settled bool  // the other side's fate was heard, rather than given up waiting for
	err     error // why the connection failed, if it did; see netplay.Race.Err
}

// verdict is the result in this side's words, which the result screen shows as its heading and
// the status band repeats.
//
// Nobody winning is two situations, and only one of them is a draw: two runs that compared equal
// on everything. Both players leaving is a forfeit with nobody to award it to, which the result
// screen used to head "A DRAW" as well -- a result nobody earned, announced as one both did.
func (res raceResult) verdict() string {
	switch {
	case !res.out.Decided():
		return "no result"
	case res.out.Slot == int8(res.match.Slot):
		return "you win"
	case res.out.Slot >= 0:
		return "you lose"
	case res.out.Reason == netplay.Drawn:
		return "a draw"
	}
	return "no result"
}

// band is the line the title screen's status band shows after a race, instead of the score it
// shows after a game: a race's score is one of its tie-breaks, and the result is what the player
// came back to the menu knowing.
func (res raceResult) band() string {
	if !res.out.Decided() || res.out.Reason == netplay.Drawn {
		return "race: " + res.verdict()
	}
	return "race: " + res.verdict() + ", " + res.out.Reason.String()
}

// finishRace is everything after NewGame returns: say how this run ended, wait to hear how the
// other one did, and show the result.
//
// The order is forced and each step is the reason for the next. Report then Close, because Close
// flushes the standing it is about to say goodbye after -- the last standing of a run is the one
// the result is computed from. Then wait, because **finishing first does not mean winning**: the
// other player is still flying and may yet get further, so neither side has a result until both
// runs have ended. Then compute, on both machines, from the same two standings -- which is what
// makes a race with no referee agree with itself.
func (a *app) finishRace(w *game.World, r *netplay.Race, mine netplay.Standing, closed bool) raceResult {
	r.Report(mine)
	// Close gives up on a writer the other machine has stopped reading from (netplay.ErrStalled)
	// rather than freezing the game before the waiting screen is up. The stall decides nothing,
	// so it is only said: the wait below and its Esc are still what end the race.
	stalled := r.Close()

	// A player, or a measurement -- plus the third case neither flag covers, a window that has
	// already gone. There is nothing to draw a result on and nobody to press a key, so a closed
	// window takes the same path a benchmark does: report to stdout and finish.
	live := !closed && !hermetic(a.o)

	settled := a.awaitSettled(w, r, mine, live)
	res := raceResult{match: r.Match(), mine: mine, theirs: theirsAsScored(r),
		settled: settled, err: r.Err()}
	// From the snapshot rather than from r.Result, which would read the opponent a second
	// time. See raceResult.
	res.out = res.match.Result(res.mine, res.theirs)

	if !a.o.quiet {
		// stdout gets the result whether or not anybody is looking at the window, because a
		// headless race is how the mode gets tested and because this is the line a bug
		// report quotes.
		fmt.Printf("glidergo: race: %s -- you %s, them %s\n",
			res.out, standingWords(res.mine), standingWords(res.theirs))
		if res.err != nil {
			fmt.Printf("glidergo: race: the connection failed: %v\n", res.err)
		}
		if stalled != nil {
			fmt.Printf("glidergo: race: %v\n", stalled)
		}
		if !res.settled {
			fmt.Println("glidergo: race: the other player never finished, so there is no result")
		}
	}
	if live {
		a.showRaceResult(res)
	}
	return res
}

// awaitSettled waits for the other side's fate behind a screen that says so.
//
// Escape leaves without a result, and that is a real answer rather than a failure: a player whose
// opponent has walked away from their machine should not have to close the window to get out. The
// result screen then says there is no result, which is true and is better than a number invented
// to fill the space.
func (a *app) awaitSettled(w *game.World, r *netplay.Race, mine netplay.Standing, live bool) bool {
	select {
	case <-r.Settled():
		return true
	default:
	}

	scr := raceSurface()
	var deadline <-chan time.Time
	if !live {
		deadline = time.After(raceSettleWait)
	}
	for {
		select {
		case <-r.Settled():
			return true
		case <-deadline:
			return false
		default:
		}
		if !live {
			// No window to pump and nobody to pump it for. Sleeping is the whole of the
			// wait, and the deadline above is what ends it.
			time.Sleep(pollWait)
			continue
		}

		for _, ev := range a.win.PollEvents() {
			switch {
			case ev.Kind == platform.EventQuit:
				w.Quitting = true
				return false
			case ev.Kind == platform.EventKeyDown && ev.Key == platform.KeyEscape:
				return false
			}
		}
		raceScreen(scr, "WAITING FOR THE OTHER PLAYER", []string{
			"your run: " + standingWords(mine),
			"",
			"the race is not over until both runs are,",
			"however far ahead you finished",
			"",
			"press Esc to stop waiting",
		}, smallPrint(r.Match(), raceRules(a.p.Fixes)))
		if err := a.presentSurface(scr); err != nil {
			return false
		}
		time.Sleep(pollWait)
	}
}

// showRaceResult is the last screen: who won, why, and both runs side by side.
func (a *app) showRaceResult(res raceResult) {
	lines := []string{
		"you:  " + standingWords(res.mine),
		"them: " + standingWords(res.theirs),
		"",
		res.out.String(),
	}
	if !res.settled {
		lines = append(lines, "", "the other player never finished")
	}
	if res.err != nil {
		// In words and not in Go's: which of reset, broken pipe and EOF the socket said is
		// the operating system's business, and stdout has it (finishRace).
		lines = append(lines, "", endWords(res.err))
	}
	lines = append(lines, "", "press a key to finish")

	scr := raceSurface()
	raceScreen(scr, strings.ToUpper(res.verdict()), lines,
		smallPrint(res.match, raceRules(a.p.Fixes)))
	if err := a.presentSurface(scr); err != nil {
		return
	}
	// Any key, and no timeout. Nothing waits behind this screen -- the process exits after it,
	// or the title screen comes back with the result on its status band -- so there is nothing
	// for a timeout to protect, and unlike the waiting screens above, a player who presses a
	// key here has read the one sentence they were waiting for.
	for {
		for _, ev := range a.win.PollEvents() {
			switch {
			case ev.Kind == platform.EventQuit:
				return
			case ev.Kind == platform.EventKeyDown && !ev.Repeat:
				return
			}
		}
		if err := a.presentSurface(scr); err != nil {
			return
		}
		time.Sleep(pollWait)
	}
}

// theirsAsScored is the opponent's standing as the *result* saw it -- folded through Abandoned,
// so a peer that vanished mid-flight reads "left" rather than "racing".
//
// The panel does the opposite (see opponentLine) and both are right: a panel during play is a
// progress report and this is a scorecard. Abandoned is netplay's own fold and is applied here
// rather than reimplemented, because both machines have to apply the same one or they print
// different scorecards for a result they agree about.
func theirsAsScored(r *netplay.Race) netplay.Standing {
	s, gone := r.Opponent()
	if gone {
		return netplay.Abandoned(s)
	}
	return s
}

// standingWords is one run in a sentence, for the two screens and the stdout line.
func standingWords(s netplay.Standing) string {
	return fmt.Sprintf("%s -- %s, score %d, %d frames", s.State, counted(int(s.Rooms), "room"), s.Score, s.Frame)
}

// ---------------------------------------------------------------------------
// The screens
// ---------------------------------------------------------------------------

// The panel every race screen is drawn in: centred, and as wide as the screen leaves room for.
// Sixteen lines tall, which is the hosting screen's three addresses, its shell line, a Windows
// hint and a note that wraps, with the way out still under them (docs/IMPROVEMENTS.md 4.33). A
// line wider than the plate is wrapped rather than centred off both edges.
const (
	racePlateLeft   = 40
	racePlateTop    = 90
	racePlateRight  = 600
	racePlateBottom = 390
	racePlateMargin = 16 // the least room between a line and the plate's frame
	raceHeadingBase = 40 // baseline of the scale-2 heading, from the plate's top
	raceLineBase    = 74 // baseline of the first body line
	raceLineStep    = 14
)

// raceSurface is a screen of this port's own, not a World's.
//
// The race's three screens are drawn from nothing rather than over the game, and the reason is
// that two of the three happen when there is no game: the waiting screen runs before NewWorld,
// because the seed the World is built with is the one the handshake agreed. So they get a bare
// surface and go through presentSurface, which is the one path in this program that blits
// something that is not a World's Main.
func raceSurface() *render.Surface {
	view := render.DefaultView()
	return render.NewSurface(int(view.Screen.Wide()), int(view.Screen.Tall()))
}

// raceScreen draws a heading and a column of lines into a plate on a black screen, and under
// them, dimmer, the small print: what the other side plays with that this side does not, when the
// race let it through (smallPrint).
//
// Deliberately plain, and deliberately not dressed up as 1994 artwork. Every pixel of the
// original's own screens came out of a PICT (internal/render), and a hand-drawn imitation of one
// would be this port claiming the original had a network mode. So: black, a framed plate, the
// port's own font. The same argument the pause hint's colours are chosen by.
func raceScreen(s *render.Surface, heading string, lines, small []string) {
	s.Fill(s.Bounds(), render.Black8)

	plate := render.SetRect(racePlateLeft, racePlateTop, racePlateRight, racePlateBottom)
	s.Fill(plate, render.Black8)
	s.FrameRect(plate, render.White8)
	s.FrameRect(render.Inset(plate, 2, 2), render.Gray8)

	raceCenter(s, plate, plate.Top+raceHeadingBase, heading, render.White8, 2)
	v := plate.Top + raceLineBase
	room := plate.Wide() - 2*racePlateMargin
	draw := func(lines []string, idx uint8) {
		for _, line := range lines {
			if line == "" {
				v += raceLineStep
				continue
			}
			for _, part := range wrapRace(line, room) {
				if v > plate.Bottom-4 {
					// A plate that has run out of room drops the rest rather than
					// drawing over its own frame. The lines are in the order they
					// matter, so what falls off the bottom is the least of them.
					return
				}
				raceCenter(s, plate, v, part, idx, 1)
				v += raceLineStep
			}
		}
	}
	draw(lines, render.LtGray8)
	if len(small) > 0 {
		v += raceLineStep
		draw(small, render.Gray8)
	}
}

// raceCenter draws one line centred in a rect, clamped to its left edge.
//
// The third copy of these four lines in this repository -- internal/game/pause.go and
// internal/shell/screens.go have the others -- and, as those two say about each other, it is
// still not worth a package. This one is four lines used by one function, and a shared text
// helper would be a package boundary carrying a division.
func raceCenter(s *render.Surface, r render.Rect, v int16, text string, idx uint8, scale int) {
	h := r.Left + (r.Wide()-render.StringWidthScaled(text, scale))/2
	if h < r.Left {
		h = r.Left
	}
	s.DrawStringScaled(h, v, text, idx, scale)
}

// presentSurface blits a surface that is not a World's Main.
//
// app.present is the same two steps for a World and cannot be used here, because two of the three
// race screens are drawn when there is no World at all. The error is returned rather than
// swallowed for the same reason: those two screens have no World to set Quitting on, so a dead
// window has to come back as a value their loops can act on.
func (a *app) presentSurface(s *render.Surface) error {
	if a.win == nil || a.fb == nil {
		return nil
	}
	s.ToBGRX(a.fb.Pix, a.fb.Stride)
	if err := a.win.Present(a.fb); err != nil {
		return err
	}
	// The mixer's pacer, as on every other present in this program: the title score is still
	// playing behind the waiting screen, and a screen that did not clock the pump would be a
	// gap in the sound for as long as the other player took to arrive.
	a.pump.ClockTick()
	return nil
}

// raceRefusals are the four command lines that name a race and something it cannot be, checked in
// parseFlags. Each one is two answers to one question, and the house rule for those is to refuse
// rather than to guess -- see the -resume/-room pair, whose failure mode (an hour wondering why a
// flag did nothing) is the one this is written to avoid.
//
// There used to be a fifth, refusing `-port` without `-host` or `-join`, and the Race screen
// retired it: a port given on its own is no longer a flag with nothing to act on, it is the port
// the title screen's Race row will host and dial on. That is not two answers to one question, so
// it is not this function's business.
func raceRefusals(o *options) error {
	race := raceRequested(o)
	switch {
	case o.host && o.join != "":
		return errors.New("-host and -join cannot both be given: one machine hosts and the " +
			"other joins it")
	case race && o.two:
		// Two different modes wearing one word. -two is two gliders in one room on one
		// keyboard, which is the original's own two-player game; a race is two machines with
		// a world each. A race whose peer was running two gliders would also have no way to
		// say whose standing it was sending.
		return errors.New("-two and a race cannot both be given: -two is two players at one " +
			"keyboard, and a race is one glider on each machine")
	case race && o.resume:
		// A resume rewrites the house's rooms -- the visited flags, which the hash covers and
		// which the race's own metric counts -- so two sides resuming different saves would
		// be refused by the house gate, and two sides resuming the *same* save would be
		// racing from a position one of them has already flown. Neither is a race.
		return errors.New("-resume and a race cannot both be given: a race starts both " +
			"players at the house's beginning")
	case race && o.seed != 1:
		// The match seed comes from the two nonces (netplay.Meet), so -seed would be
		// silently overridden -- and silently overriding the flag that decides what the
		// house does is exactly the kind of quiet disagreement the gate exists to catch.
		return errors.New("-seed and a race cannot both be given: the two machines agree a " +
			"seed between themselves so that the house behaves the same on both")
	}
	return nil
}
