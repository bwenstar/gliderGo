package house

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Pascal strings, the way the Toolbox stored them: a fixed-size byte array whose
// first byte is the length. Str27 is 28 bytes, Str31 is 32, and so on -- the
// number in the type name is the maximum length, not the array size.
//
// These are kept as arrays rather than Go strings for one reason: the bytes past
// the length byte are uninitialised memory in the shipped files, they differ
// house to house, and a loader that drops them cannot round-trip. Text() decodes
// the meaningful prefix; Residue() exposes the rest so the text format can carry
// it explicitly instead of losing it.
type (
	PStr16  [16]byte  // Str15:  high-score names
	PStr28  [28]byte  // Str27:  room names
	PStr32  [32]byte  // Str31:  high-score board banner
	PStr256 [256]byte // Str255: house banner and trailer
)

// Text decodes the length-prefixed part of a Pascal string as Mac OS Roman.
//
// A length byte greater than the array's capacity is clamped rather than
// rejected: this is used on user-supplied files, and the original's own
// validation clamps room names the same way (name[0] <= 27,
// docs/analysis/house-format.md 11.2).
func pstrText(b []byte) string {
	n := int(b[0])
	if n > len(b)-1 {
		n = len(b) - 1
	}
	return MacRomanToUTF8(b[1 : 1+n])
}

// pstrResidue returns the bytes after the declared length: uninitialised in the
// shipped files, meaningless, and preserved anyway.
func pstrResidue(b []byte) []byte {
	n := int(b[0])
	if n > len(b)-1 {
		n = len(b) - 1
	}
	return b[1+n:]
}

// pstrSet writes s into a Pascal string, zeroing the residue. It reports whether
// the text fit; a caller that cares (the text parser does) should treat a false
// as an error rather than accept a silent truncation.
func pstrSet(b []byte, s string) bool {
	enc, ok := UTF8ToMacRoman(s)
	fit := ok && len(enc) <= len(b)-1
	if len(enc) > len(b)-1 {
		enc = enc[:len(b)-1]
	}
	b[0] = byte(len(enc))
	copy(b[1:], enc)
	for i := 1 + len(enc); i < len(b); i++ {
		b[i] = 0
	}
	return fit
}

func (p PStr16) Text() string  { return pstrText(p[:]) }
func (p PStr28) Text() string  { return pstrText(p[:]) }
func (p PStr32) Text() string  { return pstrText(p[:]) }
func (p PStr256) Text() string { return pstrText(p[:]) }

func (p PStr16) Residue() []byte  { return pstrResidue(p[:]) }
func (p PStr28) Residue() []byte  { return pstrResidue(p[:]) }
func (p PStr32) Residue() []byte  { return pstrResidue(p[:]) }
func (p PStr256) Residue() []byte { return pstrResidue(p[:]) }

func (p *PStr16) SetText(s string) bool  { return pstrSet(p[:], s) }
func (p *PStr28) SetText(s string) bool  { return pstrSet(p[:], s) }
func (p *PStr32) SetText(s string) bool  { return pstrSet(p[:], s) }
func (p *PStr256) SetText(s string) bool { return pstrSet(p[:], s) }

// HasResidue reports whether any byte past the length is non-zero, i.e. whether
// a text round-trip has to record the residue explicitly to stay exact.
func hasResidue(b []byte) bool {
	for _, c := range pstrResidue(b) {
		if c != 0 {
			return true
		}
	}
	return false
}

func (p PStr16) HasResidue() bool  { return hasResidue(p[:]) }
func (p PStr28) HasResidue() bool  { return hasResidue(p[:]) }
func (p PStr32) HasResidue() bool  { return hasResidue(p[:]) }
func (p PStr256) HasResidue() bool { return hasResidue(p[:]) }

// macRomanHigh maps bytes 0x80-0xFF to Unicode. Generated from Python's
// `mac_roman` codec so that this package, the extraction tools and the analysis
// documents all agree on the same 128 characters -- including 0xF0, the Apple
// logo, which has no standard Unicode home and lives at U+F8FF by convention.
var macRomanHigh = [128]rune{
	0x00C4, 0x00C5, 0x00C7, 0x00C9, 0x00D1, 0x00D6, 0x00DC, 0x00E1, // 80 Ä Å Ç É Ñ Ö Ü á
	0x00E0, 0x00E2, 0x00E4, 0x00E3, 0x00E5, 0x00E7, 0x00E9, 0x00E8, // 88 à â ä ã å ç é è
	0x00EA, 0x00EB, 0x00ED, 0x00EC, 0x00EE, 0x00EF, 0x00F1, 0x00F3, // 90 ê ë í ì î ï ñ ó
	0x00F2, 0x00F4, 0x00F6, 0x00F5, 0x00FA, 0x00F9, 0x00FB, 0x00FC, // 98 ò ô ö õ ú ù û ü
	0x2020, 0x00B0, 0x00A2, 0x00A3, 0x00A7, 0x2022, 0x00B6, 0x00DF, // A0 † ° ¢ £ § • ¶ ß
	0x00AE, 0x00A9, 0x2122, 0x00B4, 0x00A8, 0x2260, 0x00C6, 0x00D8, // A8 ® © ™ ´ ¨ ≠ Æ Ø
	0x221E, 0x00B1, 0x2264, 0x2265, 0x00A5, 0x00B5, 0x2202, 0x2211, // B0 ∞ ± ≤ ≥ ¥ µ ∂ ∑
	0x220F, 0x03C0, 0x222B, 0x00AA, 0x00BA, 0x03A9, 0x00E6, 0x00F8, // B8 ∏ π ∫ ª º Ω æ ø
	0x00BF, 0x00A1, 0x00AC, 0x221A, 0x0192, 0x2248, 0x2206, 0x00AB, // C0 ¿ ¡ ¬ √ ƒ ≈ ∆ «
	0x00BB, 0x2026, 0x00A0, 0x00C0, 0x00C3, 0x00D5, 0x0152, 0x0153, // C8 » … NBSP À Ã Õ Œ œ
	0x2013, 0x2014, 0x201C, 0x201D, 0x2018, 0x2019, 0x00F7, 0x25CA, // D0 – — “ ” ‘ ’ ÷ ◊
	0x00FF, 0x0178, 0x2044, 0x20AC, 0x2039, 0x203A, 0xFB01, 0xFB02, // D8 ÿ Ÿ ⁄ € ‹ › ﬁ ﬂ
	0x2021, 0x00B7, 0x201A, 0x201E, 0x2030, 0x00C2, 0x00CA, 0x00C1, // E0 ‡ · ‚ „ ‰ Â Ê Á
	0x00CB, 0x00C8, 0x00CD, 0x00CE, 0x00CF, 0x00CC, 0x00D3, 0x00D4, // E8 Ë È Í Î Ï Ì Ó Ô
	0xF8FF, 0x00D2, 0x00DA, 0x00DB, 0x00D9, 0x0131, 0x02C6, 0x02DC, // F0 APPLE Ò Ú Û Ù ı ˆ ˜
	0x00AF, 0x02D8, 0x02D9, 0x02DA, 0x00B8, 0x02DD, 0x02DB, 0x02C7, // F8 ¯ ˘ ˙ ˚ ¸ ˝ ˛ ˇ
}

// macRomanRev is the inverse of macRomanHigh, built once at init.
var macRomanRev = func() map[rune]byte {
	m := make(map[rune]byte, 128)
	for i, r := range macRomanHigh {
		m[r] = byte(0x80 + i)
	}
	return m
}()

// MacRomanToUTF8 decodes Mac OS Roman bytes. Every one of the 256 byte values
// has a mapping, so this cannot fail; control bytes below 0x20 pass through as
// the corresponding code points, which is what the original's text drawing did
// with them too.
func MacRomanToUTF8(b []byte) string {
	ascii := true
	for _, c := range b {
		if c >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return string(b)
	}
	var sb strings.Builder
	sb.Grow(len(b) + 8)
	for _, c := range b {
		if c < 0x80 {
			sb.WriteByte(c)
		} else {
			sb.WriteRune(macRomanHigh[c-0x80])
		}
	}
	return sb.String()
}

// UTF8ToMacRoman encodes back. It reports false if any rune has no Mac Roman
// equivalent, in which case that rune is replaced with '?' -- the text-format
// parser turns a false into an error rather than writing a house that would not
// round-trip.
func UTF8ToMacRoman(s string) ([]byte, bool) {
	out := make([]byte, 0, len(s))
	ok := true
	for _, r := range s {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		default:
			if b, found := macRomanRev[r]; found {
				out = append(out, b)
			} else {
				out = append(out, '?')
				ok = false
			}
		}
	}
	return out, ok
}

// QuoteMacRoman renders a Pascal string's text for the text format: a Go-style
// quoted string, so that a name containing a quote, a backslash or a control
// byte survives a round trip.
func QuoteMacRoman(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7F {
				fmt.Fprintf(&sb, `\x%02x`, r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// UnquoteMacRoman is the inverse of QuoteMacRoman. It is deliberately strict:
// an unknown escape is an error, not a literal backslash, because a house that
// silently changes meaning on reload is worse than one that fails to load.
func UnquoteMacRoman(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("not a quoted string: %s", s)
	}
	body := s[1 : len(s)-1]
	var sb strings.Builder
	for i := 0; i < len(body); {
		c := body[i]
		if c != '\\' {
			r, n := utf8.DecodeRuneInString(body[i:])
			sb.WriteRune(r)
			i += n
			continue
		}
		if i+1 >= len(body) {
			return "", fmt.Errorf("trailing backslash in %s", s)
		}
		switch body[i+1] {
		case '"':
			sb.WriteByte('"')
		case '\\':
			sb.WriteByte('\\')
		case 'n':
			sb.WriteByte('\n')
		case 'r':
			sb.WriteByte('\r')
		case 't':
			sb.WriteByte('\t')
		case 'x':
			if i+3 >= len(body) {
				return "", fmt.Errorf("truncated \\x escape in %s", s)
			}
			var v int
			if _, err := fmt.Sscanf(body[i+2:i+4], "%02x", &v); err != nil {
				return "", fmt.Errorf("bad \\x escape in %s: %v", s, err)
			}
			sb.WriteRune(rune(v))
			i += 4
			continue
		default:
			return "", fmt.Errorf("unknown escape \\%c in %s", body[i+1], s)
		}
		i += 2
	}
	return sb.String(), nil
}
