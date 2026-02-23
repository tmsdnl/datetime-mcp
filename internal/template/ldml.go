package template

import "strings"

// ldmlToken maps an LDML field symbol to its Go time layout equivalent.
type ldmlToken struct {
	ldml string
	go_  string
}

// ldmlTokens is ordered longest-first to prevent partial matches
// (e.g. MMMM before MMM before MM, EEEE before EEE, yyyy before yy, ZZZZ before Z).
var ldmlTokens = []ldmlToken{
	{"yyyy", "2006"},
	{"MMMM", "January"},
	{"EEEE", "Monday"},
	{"ZZZZ", "-07:00"},
	{"MMM", "Jan"},
	{"EEE", "Mon"},
	{"HH", "15"},
	{"MM", "01"},
	{"dd", "02"},
	{"yy", "06"},
	{"mm", "04"},
	{"ss", "05"},
	{"h", "3"},
	{"a", "PM"},
	{"z", "MST"},
	{"Z", "-0700"},
}

// isLetter reports whether b is an ASCII letter.
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// ldmlToGoLayout converts an LDML format expression to a Go time layout string.
//
// Returns (goLayout, true) if at least one LDML token was found.
// Returns (original, false) if no LDML tokens were found — caller should treat
// the string as a raw Go time layout.
//
// Boundary rule: an LDML token is only matched when its left boundary is valid:
// the token must be at the start of the string, immediately after a non-letter
// separator character, or immediately after a previously matched LDML token.
// This prevents false matches of single-character tokens (a, h, z, Z) inside
// Go layout keywords like "Monday" or "January".
//
// Single-quote escaping (per LDML / Unicode UTS #35):
//   - Text enclosed in single quotes is treated as literal and passed through unchanged.
//   - The enclosing quotes themselves are stripped.
//   - A doubled single quote ('') produces a literal single-quote character.
func ldmlToGoLayout(s string) (string, bool) {
	hasLDML := false
	var out strings.Builder
	prevWasToken := false
	i := 0

	for i < len(s) {
		// Handle single-quote escaping.
		if s[i] == '\'' {
			prevWasToken = false
			i++ // skip the opening quote
			for i < len(s) {
				if s[i] == '\'' {
					if i+1 < len(s) && s[i+1] == '\'' {
						// Doubled single quote → literal single quote.
						out.WriteByte('\'')
						i += 2
					} else {
						// Closing quote — exit quote mode.
						i++
						break
					}
				} else {
					out.WriteByte(s[i])
					i++
				}
			}
			continue
		}

		// Try to match an LDML token at this position (longest-first).
		//
		// Left boundary rule: a token is only valid if:
		//   - we're at position 0 (start of string), OR
		//   - the preceding character is a non-letter separator, OR
		//   - the preceding character was the end of a matched LDML token.
		//
		// This prevents single-char tokens from matching inside English words
		// (e.g. 'a' inside "Monday", 'h' inside "Thursday").
		matched := false
		for _, tok := range ldmlTokens {
			if strings.HasPrefix(s[i:], tok.ldml) {
				leftOK := i == 0 || !isLetter(s[i-1]) || prevWasToken
				if !leftOK {
					// All shorter tokens starting with the same char share the
					// same left boundary, so none will match — stop trying.
					break
				}
				out.WriteString(tok.go_)
				i += len(tok.ldml)
				hasLDML = true
				matched = true
				prevWasToken = true
				break
			}
		}
		if !matched {
			out.WriteByte(s[i])
			i++
			prevWasToken = false
		}
	}

	if !hasLDML {
		return s, false
	}
	return out.String(), true
}
