package main

// The race's words: what its screens say when something between two machines goes wrong.
//
// Every error here already has a sentence, and the sentence is written for a terminal. It names
// both sides, quotes hashes and fingerprints, and runs to two hundred characters. That is right
// for a bug report and wrong for a plate 88 characters wide, or for a status band that fits one
// line (docs/IMPROVEMENTS.md 4.33). So each one is worded again here, shorter, and **the advice
// comes first**: a line that gets cut should lose its explanation and keep what to do. The full
// sentence still goes to the terminal, where it always went.

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/bwenstar/gliderGo/internal/netplay"
	"github.com/bwenstar/gliderGo/internal/render"
)

// briefErr is an error with a line for a screen as well as its sentence for a terminal. Error is
// the sentence, so stderr and a bug report get all of it. Brief is what the shell's status band
// shows (internal/shell's brief), and errors.Is and As see through both to the cause.
type briefErr struct {
	brief string
	err   error
}

func (e briefErr) Error() string { return e.err.Error() }
func (e briefErr) Unwrap() error { return e.err }
func (e briefErr) Brief() string { return e.brief }

// turnedAway is a connection the host dropped because it did not become a race (hostRace): which
// machine it was, without its port (remoteHost), and why.
type turnedAway struct {
	host string
	err  error
}

func (e *turnedAway) Error() string { return fmt.Sprintf("turned away %s: %v", e.host, e.err) }
func (e *turnedAway) Unwrap() error { return e.err }

// firewallAfter is how many dials in a row have to go unanswered before the screen names a
// firewall. See joinWords.
const firewallAfter = 3

// raceWords is the line a screen shows for why a race has not started. That covers a failed dial,
// a connection the host turned away, and a handshake that ended without a match. hosting is which
// end this is. timeouts is how many dials in a row have had no answer.
func raceWords(err error, hosting bool, timeouts int) string {
	var ta *turnedAway
	var je *netplay.JoinError
	switch {
	case errors.As(err, &ta):
		return "turned away " + ta.host + ": " + meetWords(ta.err, true)
	case errors.As(err, &je):
		return joinWords(je, timeouts)
	}
	return meetWords(err, hosting)
}

// joinWords is a failed dial, sorted by cause (netplay.JoinFailure).
//
// **No answer is not worded as a firewall until it has happened a few times in a row.** A machine
// that has not started hosting yet looks exactly the same from here if it runs Windows, or Linux
// with ufw or firewalld: all three drop a connection to a closed port instead of refusing it. So
// the first few timeouts say the likelier thing. The firewall comes in once the likelier thing
// has had time to stop being true, which at three seconds a dial is about ten seconds.
func joinWords(je *netplay.JoinError, timeouts int) string {
	host, port, err := net.SplitHostPort(je.Addr)
	if err != nil {
		host, port = je.Addr, netplay.DefaultPort
	}
	switch je.Why {
	case netplay.JoinRefused:
		return "that machine answered, but nothing is hosting on port " + port + " yet"
	case netplay.JoinNoSuchHost:
		return "check the spelling: no machine called " + host + " could be found"
	case netplay.JoinUnreachable:
		return "check the address: this network has no way to reach " + host
	case netplay.JoinTimedOut:
		if timeouts < firewallAfter {
			return "no answer yet -- the other machine may not be hosting"
		}
		return "still no answer. If the other machine is hosting, a firewall is dropping port " +
			port + " -- on Windows, the host has to allow gliderGo when Windows asks"
	}
	return je.Err.Error()
}

// meetWords is a handshake that ended without a match. When this end is hosting, the other end
// is a guest, and the words are a note about somebody who was turned away. When this end is
// joining, the other end is the host, and the words say what to change.
func meetWords(err error, hosting bool) string {
	var refused *netplay.RefusalError
	var stranger netplay.StrangerError
	var silent *netplay.SilenceError
	switch {
	case errors.As(err, &refused):
		return refusalWords(refused, hosting)
	case errors.As(err, &stranger):
		if hosting {
			return fmt.Sprintf("not gliderGo (it opened with %q)", stranger[:])
		}
		return fmt.Sprintf("check the address and port: that is some other program, not "+
			"gliderGo (it opened with %q)", stranger[:])
	case errors.Is(err, netplay.ErrMagic):
		if hosting {
			return "not gliderGo"
		}
		return "check the address and port: that is some other program, not gliderGo"
	case errors.As(err, &silent) && silent.Partway:
		if hosting {
			return "it stopped answering partway through"
		}
		return "the host stopped answering partway through -- try joining again"
	case errors.As(err, &silent):
		if hosting {
			return fmt.Sprintf("it said nothing for %v", silent.Wait)
		}
		return "check the address and port: something answered there, but it is not a " +
			"gliderGo race"
	case errors.Is(err, io.EOF):
		if hosting {
			return "it hung up before the race started"
		}
		return "the host hung up before the race started"
	case errors.Is(err, io.ErrUnexpectedEOF):
		// The connection ended inside a message. Without this case the line was Go's
		// sentence, "unexpected EOF" and all (docs/IMPROVEMENTS.md 4.30).
		if hosting {
			return "it hung up partway through the handshake"
		}
		return "the host hung up partway through -- try joining again"
	case errors.Is(err, netplay.ErrVersion):
		// The envelope's version rather than the hello's: a build too far apart to read
		// the other's hello at all, so there are no releases to name.
		return "install the same release on both: the other game speaks another version of " +
			"the race"
	case errors.Is(err, netplay.ErrProtocol), errors.Is(err, netplay.ErrShort):
		if hosting {
			return "it sent something this game cannot read"
		}
		return "the host sent something this game cannot read"
	}
	return err.Error()
}

// refusalWords is one of Meet's four refusals.
//
// **A guest is told the host's house** (docs/IMPROVEMENTS.md 4.28). The refusal is the one moment
// a guest can learn it, because the hellos have already been exchanged. The terminal's sentence
// has it too, after "yours is", which is past where the status band cuts off.
func refusalWords(r *netplay.RefusalError, hosting bool) string {
	mine, theirs := r.Local, r.Peer
	switch r.Reason {
	case netplay.ErrHouse:
		same := theirs.HouseName == mine.HouseName
		switch {
		case hosting && same:
			return "they have a different copy of " + strconv.Quote(mine.HouseName)
		case hosting:
			return "they opened " + peerText(theirs.HouseName) + ", not " +
				strconv.Quote(mine.HouseName)
		case same:
			return "get the host's copy of " + strconv.Quote(mine.HouseName) +
				": yours is a different file"
		}
		return "the host is racing " + peerText(theirs.HouseName) + ": open it to join"
	case netplay.ErrVersion, netplay.ErrEngine:
		if theirs.Release == mine.Release {
			both := "both say they are " + releaseText(mine.Release)
			if mine.Release == "" {
				both = "neither names its release"
			}
			return "build both from the same source: " + both + ", but they fly differently"
		}
		if hosting {
			return "they have " + peerRelease(theirs.Release) + ", and this is " +
				releaseText(mine.Release)
		}
		return "install the same release on both: the host has " +
			peerRelease(theirs.Release) + ", and this is " + releaseText(mine.Release)
	case netplay.ErrRules:
		diff := (mine.Rules ^ theirs.Rules) & netplay.RulesGated
		var who []string
		if on := theirs.Rules & diff; on != 0 {
			who = append(who, "the host plays with "+on.String())
		}
		if on := mine.Rules & diff; on != 0 {
			who = append(who, "you play with "+on.String())
		}
		if hosting {
			return "their rules differ from this side's over " + diff.String()
		}
		return "set the same rules as the host: " + strings.Join(who, ", and ")
	}
	return r.Error()
}

// endWords is a race's connection failing after the match was agreed, for the result screen.
//
// After the handshake, a reset, a refused write and an end of stream all mean the same thing: the
// other game stopped without sending its goodbye. Windows and Linux word each of the three
// differently. The Go error goes to stdout (finishRace), where a bug report quotes it. The one
// failure that is not the other game stopping is the other game sending nonsense, and that one is
// a bug somewhere.
func endWords(err error) string {
	for _, e := range []error{netplay.ErrProtocol, netplay.ErrShort, netplay.ErrMagic,
		netplay.ErrVersion} {
		if errors.Is(err, e) {
			return "the other game sent something this one cannot read"
		}
	}
	return "the other player's game ended without saying goodbye"
}

// smallPrint is what the other side plays with that differs from this side and did not refuse
// the race. That is its release, when it is another one that flies the same, and any fix it has
// on or off where this side does not (docs/IMPROVEMENTS.md 4.33, from 4.31). A player beaten by a
// glider with fixes.mirror_foil on can see that it had it.
func smallPrint(m netplay.Match, mine netplay.Rules) []string {
	var out []string
	if m.PeerRelease != version {
		out = append(out, "the other player has "+peerRelease(m.PeerRelease)+
			", which flies the same as "+releaseText(version))
	}
	var diffs []string
	for bit := range 16 {
		b := netplay.Rules(1) << bit
		switch {
		case m.PeerRules&b != 0 && mine&b == 0:
			diffs = append(diffs, b.String()+" on")
		case m.PeerRules&b == 0 && mine&b != 0:
			diffs = append(diffs, b.String()+" off")
		}
	}
	if len(diffs) > 0 {
		out = append(out, "they play with "+strings.Join(diffs, ", "))
	}
	return out
}

// peerText is a string the other machine sent, made fit for a line of the plate. It is quoted, so
// that a control character arrives as an escape and not as whatever the font makes of it. It is
// also cut, because a house name can be 255 bytes and the other end is not obliged to be kind.
func peerText(s string) string {
	const most = 40
	if r := []rune(s); len(r) > most {
		s = string(r[:most]) + "..."
	}
	return strconv.Quote(s)
}

// releaseText is this build's release as the words name it, and peerRelease is the other side's.
func releaseText(r string) string {
	if r == "" {
		return "an unnamed build"
	}
	return "release " + strconv.Quote(r)
}

func peerRelease(r string) string {
	if r == "" {
		return "an unnamed build"
	}
	return "release " + peerText(r)
}

// wrapRace breaks a line into lines no wider than px. It breaks at spaces where it can and inside
// a word where it has to. An address or a quoted "SSH-" is one word, and a word wider than the
// plate is better cut than drawn over the frame.
func wrapRace(text string, px int16) []string {
	if render.StringWidth(text) <= px {
		return []string{text}
	}
	var out []string
	line := ""
	for _, word := range strings.Fields(text) {
		for render.StringWidth(word) > px {
			if line != "" {
				out = append(out, line)
				line = ""
			}
			r := []rune(word)
			n := len(r)
			for n > 1 && render.StringWidth(string(r[:n])) > px {
				n--
			}
			out = append(out, string(r[:n]))
			word = string(r[n:])
		}
		switch {
		case line == "":
			line = word
		case render.StringWidth(line+" "+word) <= px:
			line += " " + word
		default:
			out = append(out, line)
			line = word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
