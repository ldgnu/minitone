package models

import (
	"strings"
	"unicode"
)

// maxDisplayLen caps any single remote string we are willing to render.
const maxDisplayLen = 512

// Sanitize makes an untrusted string safe to print to a terminal and to keep.
//
// Remote metadata (YouTube titles, Radio Browser station names, Navidrome
// albums) is attacker-influenced: Radio Browser in particular is a public,
// community-editable database. Any control sequence — especially OSC
// (ESC ] … BEL/ST), which can rewrite the terminal title or, via OSC 52, write
// to the system clipboard — is removed here rather than trusted.
//
// Newlines, tabs and carriage returns collapse to a single space so one record
// always stays on one row, and the length is bounded so a hostile payload cannot
// blow up the layout.
func Sanitize(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))

	// A minimal VT500 parser: enough to consume a whole escape sequence and
	// discard it, including OSC payloads whose body is arbitrary text or
	// base64 (a naive "stop at the first letter" rule leaks the payload as
	// visible garbage).
	const (
		stNormal    = iota
		stEsc       // saw ESC
		stCSI       // ESC [ … final byte in 0x40..0x7E
		stOSC       // ESC ] … terminated by BEL or ST
		stOSCEsc    // ESC ] … ESC  (expecting '\\')
		stStringSeq // ESC P / _ / ^ / X … terminated by ST
		stStringEsc
	)
	state := stNormal

	for _, r := range s {
		switch state {
		case stEsc:
			switch r {
			case '[':
				state = stCSI
			case ']':
				state = stOSC
			case 'P', '_', '^', 'X':
				state = stStringSeq
			default:
				// Two-character sequence: drop it and carry on.
				state = stNormal
			}
			continue

		case stCSI:
			// Parameter and intermediate bytes run 0x30..0x3F; the final byte
			// is 0x40..0x7E.
			if r == 0x1b {
				state = stEsc
			} else if r >= 0x40 && r <= 0x7E {
				state = stNormal
			}
			continue

		case stOSC:
			switch r {
			case 0x07: // BEL
				state = stNormal
			case 0x1b:
				state = stOSCEsc
			}
			continue

		case stOSCEsc:
			// ST is ESC \ ; anything else keeps us inside the string.
			if r == '\\' {
				state = stNormal
			} else if r != 0x1b {
				state = stOSC
			}
			continue

		case stStringSeq:
			if r == 0x1b {
				state = stStringEsc
			}
			continue

		case stStringEsc:
			if r == '\\' {
				state = stNormal
			} else if r != 0x1b {
				state = stStringSeq
			}
			continue
		}

		// stNormal
		switch {
		case r == 0x1b:
			state = stEsc
		case r == 0x07: // stray BEL
		case r == '\n' || r == '\t' || r == '\r':
			b.WriteRune(' ')
		case unicode.IsControl(r):
			// Dropped: remaining C0/C1 controls.
		case r == unicode.ReplacementChar:
			// Already-invalid UTF-8; render nothing.
		default:
			b.WriteRune(r)
		}

		if b.Len() >= maxDisplayLen {
			b.WriteString("…")
			return b.String()
		}
	}

	// An unterminated sequence is simply dropped: the loop above consumed it
	// and nothing was written.
	return b.String()
}

// Sanitized returns a copy of the song with every text field cleaned.
func (s Song) Sanitized() Song {
	s.Title = Sanitize(s.Title)
	s.Artist = Sanitize(s.Artist)
	s.Album = Sanitize(s.Album)
	s.Genre = Sanitize(s.Genre)
	s.URL = Sanitize(s.URL)
	s.FilePath = Sanitize(s.FilePath)
	return s
}
