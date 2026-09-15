package audio

// The sound table: GliderDefines.h:55-118 (the 64 slots) and :120-180 (the 60 priorities),
// transcribed so that a report can name a number.
//
// Nothing in the mixer needs either table -- a slot is an index and a priority is an int16 --
// and that is the point of keeping them here rather than scattering them through the packages
// that play sounds. `internal/game` and `internal/game/player` declare only the constants
// their own code names, which is right for them and useless for a bug report: "sound 26 at
// priority 903 displaced sound 18" is a line nobody can read, and "shred (903) displaced
// thrust (300)" is one anybody can.
//
// ---------------------------------------------------------------------------
// The priority scale is banded, and the bands are the design
// ---------------------------------------------------------------------------
//
// Larger wins. The 60 priorities are not 60 distinct opinions; they are eight bands, and
// within a band the exact number almost never matters because two sounds in one band rarely
// contend for the same channel. Reading them as bands is what makes the scale make sense:
//
//	100s  incidental      a wall bump, a score tick, a band rebounding, a refused exit
//	200s  timekeeping     the clock's tik and tok, the wind chimes, a mystic's hum
//	300s  continuous      thrust, a fired band, a VCR, toast, coffee, a drip -- the
//	                      sounds a player causes and expects to hear over the room
//	400s  appliances      a Mac, a TV, a fish, a zap, a pop, an enemy arriving
//	500   the telephone   alone in its band, and louder than every appliance
//	700s  switches        a switch thrown, blowers on or off, a glider fizzling
//	800s  loud appliances the microwave, the buzzer, the bird, the cuckoo, the typewriter
//	900s  the glider      fade in, fade out, catching fire, being shredded, following,
//	                      and the two transporter sounds -- everything that happens *to*
//	                      the player, which is what should never be masked
//	999   the trigger     a house's own sound, and the one priority with a rule of its
//	                      own: PlayPrioritySound refuses a second trigger while one is
//	                      playing on any channel (Sound.c:47-51), so a room full of
//	                      sound triggers cannot drown itself out.
//
// The consequence worth knowing when reading a trace: the loud sounds are also the *repeated*
// ones. A shredding glider asks for slot 26 at priority 903 on every single frame, and because
// PlayPrioritySound sends each request to the quietest channel it takes all three in three
// frames and keeps them until the shredding stops -- so a glider coming apart silences the rest
// of the room, by construction rather than by accident. That is why the port's channel policy
// has to be Sound.c's exactly, and why the trace records which channel each grant went to.

// Names is the 64 slots in order, named as GliderDefines.h names them with the k-prefix and
// the -Sound suffix dropped.
//
// The order is load-bearing: this table's index *is* the resource ID minus 1000, so a name in
// the wrong row would misname every report about that sound.
//
// The check on it is testdata/sound_golden.txt, which lists every slot with the header's name
// and the *resource's* own name side by side. They cannot be compared mechanically -- the
// resource fork calls slot 8 "Miked", slot 16 "Yow!" and slot 58 "Twunk", where the header calls
// them kMicrowavedSound, kCaughtFireSound and kWebTwangSound -- but they can be compared by a
// person, once, and then held still by a golden file. Which is what that file is for.
var Names = [MaxSounds]string{
	"hit wall",     // 0
	"fade in",      // 1
	"fade out",     // 2
	"beeps",        // 3
	"buzzer",       // 4
	"ding",         // 5
	"energize",     // 6
	"follow",       // 7
	"microwaved",   // 8
	"switch",       // 9
	"bird",         // 10
	"cuckoo",       // 11
	"tik",          // 12
	"tok",          // 13
	"blower on",    // 14
	"blower off",   // 15
	"caught fire",  // 16
	"score tik",    // 17
	"thrust",       // 18
	"fizzle",       // 19
	"fire band",    // 20
	"band rebound", // 21
	"grease spill", // 22
	"chord",        // 23
	"VCR",          // 24
	"foil hit",     // 25
	"shred",        // 26
	"toast launch", // 27
	"toast land",   // 28
	"mac on",       // 29
	"mac beep",     // 30
	"mac off",      // 31
	"TV on",        // 32
	"TV off",       // 33
	"coffee",       // 34
	"mystic",       // 35
	"zap",          // 36
	"pop",          // 37
	"enemy in",     // 38
	"enemy out",    // 39
	"paper crunch", // 40
	"bounce",       // 41
	"drip",         // 42
	"drop",         // 43
	"fish out",     // 44
	"fish in",      // 45
	"dont exit",    // 46
	"sizzle",       // 47
	"paper 1",      // 48
	"paper 2",      // 49
	"paper 3",      // 50
	"paper 4",      // 51
	"typing",       // 52
	"carriage",     // 53
	"chord 2",      // 54
	"phone ring",   // 55
	"chime 1",      // 56
	"chime 2",      // 57
	"web twang",    // 58
	"trans out",    // 59
	"trans in",     // 60
	"bonus",        // 61
	"hiss",         // 62
	"trigger",      // 63
}

// Name is the slot's name, or a bracketed number for a slot outside the table. It never
// returns the empty string, because its callers are format strings.
func Name(slot int16) string {
	if slot < 0 || int(slot) >= len(Names) {
		return "[" + itoa(int(slot)) + "]"
	}
	return Names[slot]
}

// TriggerPriority is kTriggerPriority (GliderDefines.h:180), the one priority with a rule of
// its own. The mixer needs the value, which is why this constant is here and the other 59 are
// not.
const TriggerPriority int16 = 999

// Band names the priority's decade group. See the file comment.
func Band(priority int16) string {
	switch {
	case priority >= 999:
		return "trigger"
	case priority >= 900:
		return "glider"
	case priority >= 800:
		return "loud appliance"
	case priority >= 700:
		return "switch"
	case priority >= 500:
		return "telephone"
	case priority >= 400:
		return "appliance"
	case priority >= 300:
		return "continuous"
	case priority >= 200:
		return "timekeeping"
	case priority >= 100:
		return "incidental"
	}
	// Priority 0 is not a sound's; it is what a channel's priority falls back to when its
	// completion callback runs (Sound.c:225). A *request* at 0 would be granted by any
	// idle channel and refused by every busy one, and no call site makes one.
	return "idle"
}

// itoa is strconv.Itoa without the import, so that this file names no dependency at all. It
// handles the negative case because Name's argument comes from a house file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
