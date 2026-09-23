package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bwenstar/gliderGo/internal/netplay"
	"github.com/bwenstar/gliderGo/internal/render"
)

// joinErr is a failed dial as netplay.Join returns one, sorted and all.
func joinErr(addr string, why netplay.JoinFailure) *netplay.JoinError {
	return &netplay.JoinError{Addr: addr, Why: why, Err: errors.New("dial tcp " + addr + ": the cause")}
}

// Each cause a dial can fail with is said as what to check, and names the thing to check it on.
// "no such host" and "network is unreachable" are the kernel's words, and neither says what a
// player typed wrong.
func TestAFailedDialSaysWhatToCheck(t *testing.T) {
	for _, tc := range []struct {
		why  netplay.JoinFailure
		want []string
	}{
		{netplay.JoinRefused, []string{"nothing is hosting", "port 1138"}},
		{netplay.JoinNoSuchHost, []string{"check the spelling", "macintosh"}},
		{netplay.JoinUnreachable, []string{"check the address", "macintosh"}},
		{netplay.JoinTimedOut, []string{"no answer yet", "may not be hosting"}},
		{netplay.JoinOther, []string{"the cause"}},
	} {
		got := raceWords(joinErr("macintosh:1138", tc.why), false, 1)
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("cause %d reads %q, which does not say %q", tc.why, got, w)
			}
		}
	}
}

// No answer is worded as a firewall only once it has gone on long enough that the host not
// having started yet is the less likely reason (joinWords). Before that, a player who is early
// would be sent looking for a firewall that is not there.
func TestAFirewallIsNamedOnlyAfterSeveralSilences(t *testing.T) {
	je := joinErr("10.0.0.5:1994", netplay.JoinTimedOut)
	for n := 1; n < firewallAfter; n++ {
		if got := raceWords(je, false, n); strings.Contains(got, "firewall") {
			t.Errorf("after %d unanswered dials the screen says %q", n, got)
		}
	}
	got := raceWords(je, false, firewallAfter)
	for _, w := range []string{"firewall", "port 1994", "Windows"} {
		if !strings.Contains(got, w) {
			t.Errorf("after %d unanswered dials the screen says %q, which does not say %q",
				firewallAfter, got, w)
		}
	}
}

// And the count is the screen's, kept by raceConnect: any answer at all, even a refusal, is
// proof that the path is open, so it starts the count again.
func TestAnAnswerResetsTheSilenceCount(t *testing.T) {
	rc := newRaceConnect()
	defer rc.cancel()
	timeout := joinErr("10.0.0.5:1994", netplay.JoinTimedOut)
	for range firewallAfter {
		rc.note(timeout)
	}
	if n := rc.lastNote().timeouts; n != firewallAfter {
		t.Fatalf("after %d timeouts the count is %d", firewallAfter, n)
	}
	rc.note(joinErr("10.0.0.5:1994", netplay.JoinRefused))
	rc.note(timeout)
	if n := rc.lastNote().timeouts; n != 1 {
		t.Errorf("a refusal between timeouts left the count at %d, want 1", n)
	}
}

// hellos is two sides that disagree about one thing each time, as Meet sees them.
func hellos() (mine, theirs netplay.Hello) {
	mine = netplay.Hello{Nonce: 7, HouseName: "Slumberland", Release: "0.4.0"}
	theirs = netplay.Hello{Nonce: 9, HouseName: "Fun House", Release: "0.4.0"}
	return mine, theirs
}

func refusal(reason error, mine, theirs netplay.Hello) error {
	return &netplay.RefusalError{Reason: reason, Local: mine, Peer: theirs}
}

// A guest turned away over the house is told which house the host is racing, because the refusal
// is the one moment it can be known (docs/IMPROVEMENTS.md 4.28). The host is told what the guest
// opened. Neither is told only that they differ.
func TestAGuestIsToldTheHostsHouse(t *testing.T) {
	mine, theirs := hellos()
	if got := raceWords(refusal(netplay.ErrHouse, mine, theirs), false, 0); !strings.HasPrefix(got,
		`the host is racing "Fun House"`) {
		t.Errorf("a guest with the wrong house reads %q", got)
	}
	if got := raceWords(refusal(netplay.ErrHouse, mine, theirs), true, 0); got !=
		`they opened "Fun House", not "Slumberland"` {
		t.Errorf("the host of a guest with the wrong house reads %q", got)
	}

	// The same name and a different hash is a different copy, and saying "open Slumberland"
	// to somebody who has Slumberland open is advice they cannot take.
	theirs.HouseName = mine.HouseName
	got := raceWords(refusal(netplay.ErrHouse, mine, theirs), false, 0)
	if !strings.Contains(got, "copy") || strings.Contains(got, "open it") {
		t.Errorf("a guest with another copy of the same house reads %q", got)
	}
}

// A release is named when the two differ, because installing it is something a player can do.
// When they do not differ there is nothing to install, and the advice is the one that is left.
func TestAnEngineRefusalNamesBothReleases(t *testing.T) {
	mine, theirs := hellos()
	theirs.Release = "0.5.0"
	got := raceWords(refusal(netplay.ErrEngine, mine, theirs), false, 0)
	for _, w := range []string{"install the same release", `"0.5.0"`, `"0.4.0"`} {
		if !strings.Contains(got, w) {
			t.Errorf("an engine refusal between releases reads %q, which does not say %q", got, w)
		}
	}
	theirs.Release = mine.Release
	if got := raceWords(refusal(netplay.ErrEngine, mine, theirs), false, 0); !strings.HasPrefix(got,
		"build both from the same source") {
		t.Errorf("an engine refusal within one release reads %q", got)
	}
}

func TestARulesRefusalSaysWhoPlaysWithWhat(t *testing.T) {
	mine, theirs := hellos()
	theirs.Rules = netplay.RuleAssisted
	got := raceWords(refusal(netplay.ErrRules, mine, theirs), false, 0)
	if !strings.HasPrefix(got, "set the same rules as the host") ||
		!strings.Contains(got, "the host plays with an assist") {
		t.Errorf("a guest refused over the host's assist reads %q", got)
	}
}

// A house name and a release are the other machine's strings, up to 255 bytes each, and the font
// draws what it has no glyph for as a box. So they arrive quoted, with a control character as its
// escape, and cut.
func TestThePeersOwnWordsAreQuotedAndCut(t *testing.T) {
	mine, theirs := hellos()
	theirs.HouseName = "Fun\x1b[2J House" + strings.Repeat("!", 250)
	got := raceWords(refusal(netplay.ErrHouse, mine, theirs), false, 0)
	if strings.ContainsRune(got, 0x1b) || !strings.Contains(got, `\x1b`) {
		t.Errorf("a house name with an escape in it reads %q", got)
	}
	if len(got) > 120 {
		t.Errorf("a 264-byte house name made a %d-byte line: %q", len(got), got)
	}
}

// The ways a handshake ends that are not refusals. Each says, to a guest, what to do; and a host
// turning a connection away says which machine it was.
func TestAHandshakeThatFailsSaysWhy(t *testing.T) {
	stranger := netplay.StrangerError{'S', 'S', 'H', '-'}
	for _, tc := range []struct {
		err     error
		hosting bool
		want    string
	}{
		{stranger, false, `some other program, not gliderGo (it opened with "SSH-")`},
		{&turnedAway{host: "10.0.0.9", err: stranger}, true,
			`turned away 10.0.0.9: not gliderGo (it opened with "SSH-")`},
		{&netplay.SilenceError{Wait: 15 * time.Second}, false, "it is not a gliderGo race"},
		{&netplay.SilenceError{Wait: 15 * time.Second, Partway: true}, false, "try joining again"},
		{&turnedAway{host: "10.0.0.9", err: &netplay.SilenceError{Wait: 5 * time.Second}}, true,
			"turned away 10.0.0.9: it said nothing for 5s"},
		{fmt.Errorf("meeting: %w", io.EOF), false, "the host hung up"},
	} {
		if got := raceWords(tc.err, tc.hosting, 0); !strings.Contains(got, tc.want) {
			t.Errorf("%v reads %q, which does not say %q", tc.err, got, tc.want)
		}
	}
}

// The line a guest's race ends on goes to the title screen's status band, after the house's name,
// and the band is one line. So each of them has to fit it with a house name of an ordinary length
// in front -- the long explanations are the terminal's.
func TestEveryWayAGuestIsTurnedAwayFitsTheBand(t *testing.T) {
	const band = 640 - 8 // internal/shell's screenWide-8
	mine, theirs := hellos()
	theirs.Release = "0.5.0"
	theirs.Rules = netplay.RuleAssisted
	for _, err := range []error{
		refusal(netplay.ErrHouse, mine, theirs),
		refusal(netplay.ErrEngine, mine, theirs),
		refusal(netplay.ErrVersion, mine, theirs),
		refusal(netplay.ErrRules, mine, theirs),
		netplay.StrangerError{'G', 'E', 'T', ' '},
		&netplay.SilenceError{Wait: 15 * time.Second},
		&netplay.SilenceError{Wait: 15 * time.Second, Partway: true},
		io.EOF,
		netplay.ErrVersion,
		netplay.ErrProtocol,
	} {
		line := "Slumberland: " + raceWords(err, false, 0)
		if w := render.StringWidth(line); w > band {
			t.Errorf("%q is %d pixels, and the band is %d", line, w, band)
		}
	}
}

func TestTheEndOfARaceIsNotTheSocketsWords(t *testing.T) {
	reset := &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read",
		errors.New("connection reset by peer"))}
	for _, err := range []error{io.EOF, reset} {
		if got := endWords(err); !strings.Contains(got, "without saying goodbye") {
			t.Errorf("endWords(%v) = %q", err, got)
		}
	}
	if got := endWords(fmt.Errorf("reading: %w", netplay.ErrProtocol)); !strings.Contains(got,
		"cannot read") {
		t.Errorf("a protocol violation at the end of a race reads %q", got)
	}
}

// What the race let through is still worth a line: a player beaten by a glider with a fix on can
// see that it had one (docs/IMPROVEMENTS.md 4.31).
func TestTheSmallPrintIsWhatTheOtherSidePlaysWith(t *testing.T) {
	m := netplay.Match{PeerRelease: version, PeerRules: netplay.RuleMirrorFoil}
	got := strings.Join(smallPrint(m, netplay.RuleMirrorFlame), "\n")
	if got != "they play with fixes.mirror_flame off, fixes.mirror_foil on" {
		t.Errorf("small print = %q", got)
	}
	if got := smallPrint(netplay.Match{PeerRelease: version}, 0); len(got) != 0 {
		t.Errorf("two identical sides have small print: %q", got)
	}
	m = netplay.Match{PeerRelease: version + "-other"}
	if got := strings.Join(smallPrint(m, 0), "\n"); !strings.Contains(got, "flies the same") {
		t.Errorf("another release's small print = %q", got)
	}
}

// Every line the plate draws is within it, and none of the text is lost on the way: a word too
// wide for a line is split, which is the one case wrapping has to change the words.
func TestAWrappedLineStaysOnThePlate(t *testing.T) {
	const px = 200
	for _, text := range []string{
		"short",
		"still no answer. If the other machine is hosting, a firewall is dropping port 1994",
		strings.Repeat("x", 90),
		"connecting to [2001:db8:85a3::8a2e:370:7334]:1994 which is an address with no spaces",
	} {
		lines := wrapRace(text, px)
		for _, l := range lines {
			if w := render.StringWidth(l); w > px {
				t.Errorf("wrapRace(%q) gave %q, %d pixels wide", text, l, w)
			}
		}
		strip := func(s string) string { return strings.ReplaceAll(s, " ", "") }
		if got := strip(strings.Join(lines, "")); got != strip(text) {
			t.Errorf("wrapRace(%q) lost text: %q", text, lines)
		}
	}
}

// The default route's address first, whatever order the interfaces came in. A Docker bridge that
// sorts before the LAN is the case this is for.
func TestTheDefaultRoutesAddressIsReadOutFirst(t *testing.T) {
	ips := []net.IP{net.ParseIP("172.17.0.1"), net.ParseIP("192.168.1.20"),
		net.ParseIP("10.8.0.2"), net.ParseIP("fd00::20")}
	got := orderAddresses(ips, net.ParseIP("192.168.1.20"))
	if len(got) != 3 || !got[0].Equal(net.ParseIP("192.168.1.20")) {
		t.Errorf("orderAddresses = %v, want 192.168.1.20 first and three in all", got)
	}
	if got := orderAddresses(ips, nil); !got[0].Equal(ips[0]) {
		t.Errorf("with no default route, orderAddresses = %v, want the interfaces' order", got)
	}
}

// A private address is marked, so that a host whose friend is across the internet can see
// before anyone types it that this one will not do. Tailscale's range is not marked, because on
// a player's machine it is Tailscale's, and that reaches across.
func TestAnAddressOnlyThisNetworkCanReachIsMarked(t *testing.T) {
	for _, tc := range []struct {
		ip     string
		want   string
		marked bool
	}{
		{"192.168.1.20", "192.168.1.20:1994", true},
		{"10.0.0.5", "10.0.0.5:1994", true},
		{"fd00::20", "[fd00::20]:1994", true},
		{"100.101.102.103", "100.101.102.103:1994", false},
		{"203.0.113.7", "203.0.113.7:1994", false},
		{"2001:db8::7", "[2001:db8::7]:1994", false},
	} {
		got := addressLine(net.ParseIP(tc.ip), "1994")
		addr, mark, _ := strings.Cut(got, " ")
		if addr != tc.want || (mark != "") != tc.marked {
			t.Errorf("addressLine(%s) = %q, want %s, marked %v", tc.ip, got, tc.want, tc.marked)
		}
	}
}

// Join's context is the raceConnect's, so Escape ends a dial where it stands rather than when
// its timeout would have.
func TestCancelEndsADialInProgress(t *testing.T) {
	rc := newRaceConnect()
	done := make(chan error, 1)
	go func() {
		c, err := netplay.Join(rc.ctx, "203.0.113.1", "1994", time.Minute)
		if c != nil {
			c.Close()
		}
		done <- err
	}()
	time.Sleep(100 * time.Millisecond)
	rc.cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Skipf("this network failed TEST-NET-3 before the cancel (%v)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled dial was still waiting five seconds later")
	}
}
