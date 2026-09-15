package audio

// The music channel: Music.c's fourth sound channel, the queue StartMusic fills and the
// callback that keeps it full.
//
// ---------------------------------------------------------------------------
// What the C actually does, because the shape is not obvious
// ---------------------------------------------------------------------------
//
// Music is a fourth `sampledSynth` channel opened exactly like the three effect channels
// (`initNoInterp + initMono`, Music.c:287) and playing samples out of the same kind of bank.
// It has no priority and it never contends: the effects channels cannot displace it and it
// cannot displace them. What it has instead is *continuity*, and the whole of Music.c is about
// keeping a sample queued ahead of the one playing so the score does not stop between pieces:
//
//	StartMusic (:44-94)     queues piece musicSoundID, advances musicCursor through
//	                        musicScore[], queues *that* piece too, then queues a callBackCmd.
//	                        Two buffers deep, deliberately.
//	MusicCallBack (:170-216) fires when the queue drains to the callback, walks the score for
//	                        one more piece, queues it and queues another callBackCmd behind it.
//	StopTheMusic (:98-121)  flushCmd then quietCmd, and isMusicOn = false.
//
// So the queue in this file is the C's command queue, two pieces deep because StartMusic makes
// it two pieces deep, and NextPiece is the callBackCmd. The port's queue empties to silence
// and then asks for one more piece, where the C's asks *before* it runs dry; the difference is
// the interrupt-latency gap between pieces in the original, which the port does not reproduce
// because it cannot and would not want to.
//
// ---------------------------------------------------------------------------
// Where the score lives, and why not here
// ---------------------------------------------------------------------------
//
// musicScore[16], gameScore[6], musicMode and musicCursor are all in internal/game, on the
// World, and only NextPiece crosses over. That split is not tidiness: the score walk *reads
// and writes the cursor*, the cursor is part of a saved session's state, and the transit code
// nudges it from six room-change handlers (SetMusicalMode). Putting the walk in the mixer
// would put a piece of game state behind an audio interface, and putting it behind a mutex --
// which is what mixing on a device thread would force -- would put a lock on the frame loop.
// One function pointer, called from Mix on Mix's own goroutine, and the state stays where the
// rest of the game's state is. See the engine.go file comment.
//
// The one consequence to keep in mind: because NextPiece writes World.MusicCursor, a session
// with music on walks the cursor and a session with music off does not. Nothing in the
// simulation reads the cursor, so no room, object or player behaviour can differ -- but the
// replay harness records audio settings in the script header for exactly this class of reason,
// and it is why it does.
//
// ---------------------------------------------------------------------------
// StopTheMusic flushes its callback too, and here it is harmless
// ---------------------------------------------------------------------------
//
// Note the shape of StopTheMusic (:109-118): flushCmd and quietCmd, which discards the queued
// callBackCmd the same way FlushAnyTriggerPlaying does -- and then it sets isMusicOn = false
// *by hand*, so nothing is left stale. That contrast is the strongest evidence for the reading
// in FlushTriggerSound's note: flushing a channel loses its callback, the author knew to
// repair the state by hand where the state mattered, and the one place he did not is the one
// place it broke.

// The music-channel piece indices that double as modes (GliderDefines.h:51-53).
//
// They are indices into theMusicData[], 0..6, and they appear in the score tables -- but
// MusicCallBack's `default` branch also assigns `musicSoundID = musicMode`, so a mode that is
// not one of the two negative score modes *is* a piece number and the score stops walking.
// That is how the game asks for one piece on repeat.
const (
	PlayChorus         int16 = 4
	PlayRefrainSparse1 int16 = 5
	PlayRefrainSparse2 int16 = 6
)

// NoPiece is what NextPiece returns to decline: the score has nothing more to play.
//
// The C has no way to say this -- MusicCallBack always queues something -- so it is the port's
// answer for the cases the C reaches by not being called at all: music turned off, no bank, or
// a cursor pointing outside the score. It stalls the channel rather than silencing it, so that
// a later QueueMusic starts things again.
const NoPiece int16 = -1

// musicChannel is the fourth channel plus the queue in front of it.
type musicChannel struct {
	channel

	// queue is the pieces already committed, in order. Two deep in normal play, because
	// StartMusic queues two.
	queue []int16

	// chain is whether the callBackCmd is armed: whether an empty queue should ask
	// NextPiece for more. False from the start and after SilenceMusic, so a stopped score
	// stays stopped.
	chain bool

	// stalled latches a NextPiece that declined. Without it an idle chained channel would
	// call back 22,255 times a second -- into game code, from inside Mix -- which would be
	// both a waste and a very effective way to make a stopped score expensive. The C's
	// equivalent is simply not having a callback queued.
	stalled bool
}

// next advances the music channel one frame, starting the next piece when the current one ends.
func (m *musicChannel) next(e *Engine) int32 {
	if m.data == nil {
		m.advance(e)
	}
	return m.channel.next()
}

// advance is the callBackCmd's body: take the next committed piece, or ask for one.
func (m *musicChannel) advance(e *Engine) {
	for len(m.queue) > 0 {
		piece := m.queue[0]
		m.queue = m.queue[1:]
		if m.begin(e, piece) {
			return
		}
	}
	if !m.chain || m.stalled || e.NextPiece == nil {
		return
	}
	// Exactly one attempt. A NextPiece that answers with a piece the bank cannot play is
	// treated as a decline rather than retried, because the alternative is a loop inside a
	// single sample of the mix.
	if piece := e.NextPiece(); piece != NoPiece && m.begin(e, piece) {
		return
	}
	m.stalled = true
}

// begin loads a piece, reporting whether it could.
func (m *musicChannel) begin(e *Engine, piece int16) bool {
	s := e.bank.Piece(piece)
	if s == nil || len(s.Data) == 0 {
		return false
	}
	m.channel.start(s, piece, 0)
	e.stats.MusicStarted++
	return true
}

// MusicAvailable reports whether starting the score would do anything -- the two guards
// StartMusic makes before it queues a note (Music.c:52-57): `dontLoadMusic` and a system
// volume of zero, plus `failedMusic`, which for the port is the same condition as having no
// bank.
//
// It exists because the game's StartMusic sets isMusicOn from the answer, and a MusicOn that
// was true with nothing behind it would make every later ToggleMusicWhilePlaying lie.
func (e *Engine) MusicAvailable() bool {
	return e != nil && !e.dontLoad && e.volume != 0 && e.bank.Piece(0) != nil
}

// QueueMusic is one of StartMusic's bufferCmds: commit a piece and arm the callback.
//
// The game calls it twice in a row, as the C queues twice, with the cursor advanced in
// between. Calling it while the score is playing appends rather than interrupting, which is
// what SndDoCommand's `false` (do not flush) means at Music.c:62.
func (e *Engine) QueueMusic(piece int16) {
	if e == nil || e.dontLoad {
		return
	}
	e.music.chain = true
	e.music.stalled = false
	e.music.queue = append(e.music.queue, piece)
}

// SilenceMusic is StopTheMusic's flushCmd and quietCmd: drop what is queued and stop what is
// playing.
//
// It disarms the chain, which is the port's stand-in for the discarded callBackCmd -- and
// here, unlike in FlushTriggerSound, discarding it is the *intent* rather than an accident, so
// this one is the C's behaviour and not a deviation from it.
func (e *Engine) SilenceMusic() {
	if e == nil {
		return
	}
	e.music.queue = e.music.queue[:0]
	e.music.chain = false
	e.music.stalled = false
	e.music.quiet()
}

// MusicPlaying is the piece on the music channel, or NoPiece. For reports and tests; the game
// has nothing to ask this with, because the C's answer would be inside the Sound Manager.
func (e *Engine) MusicPlaying() int16 {
	if e == nil || e.music.data == nil {
		return NoPiece
	}
	return e.music.playing
}

// MusicQueued is how many pieces are committed behind the one playing. Two is the number
// StartMusic leaves; anything larger means something is queueing faster than the score plays.
func (e *Engine) MusicQueued() int {
	if e == nil {
		return 0
	}
	return len(e.music.queue)
}
