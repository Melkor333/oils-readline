package main

//case shell.StdoutMsg:
//	var cmds []tea.Cmd

// hasNonColorEscape reports whether s contains ANSI escape sequences beyond
// basic SGR (Select Graphic Rendition) color codes. Used to distinguish between
// simple colored output (handled by StdoutViewer) and full terminal programs
// that need a Terminal widget with PTY passthrough (e.g., htop, vim, less).
//
// Recognized non-color escapes:
//   - CSI sequences ending in any letter other than 'm' (cursor movement, erase, scroll)
//   - OSC sequences (title changes, hyperlinks)
//   - DEC private sequences (save/restore cursor, keypad mode, character sets)
//   - Any other ESC + single char control sequence (0x40-0x5F range)
func hasNonColorEscape(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b { // ESC
			if i+1 >= len(s) {
				break
			}
			switch s[i+1] {
			case '[': // CSI sequence
				j := i + 2
				// Skip parameter bytes (0x30-0x3F: digits, semicolons, etc.)
				for j < len(s) && s[j] >= 0x30 && s[j] <= 0x3F {
					j++
				}
				// Skip intermediate bytes (0x20-0x2F)
				for j < len(s) && s[j] >= 0x20 && s[j] <= 0x2F {
					j++
				}
				if j < len(s) {
					if s[j] == 'm' {
						// SGR (Select Graphic Rendition) - coloring, skip past it
						i = j
					} else {
						// Non-SGR CSI sequence (cursor movement, erase display, etc.)
						return true
					}
				} else {
					i = len(s) - 1 // Skip to end, incomplete
				}

			case ']': // OSC (Operating System Command) sequence
				// Always non-color (title changes, hyperlinks, etc.)
				return true

			case '7': // DECSC - save cursor
				return true
			case '8': // DECRC - restore cursor
				return true
			case 'D': // IND - index
				return true
			case 'E': // NEL - next line
				return true
			case 'H': // HTS - horizontal tab set
				return true
			case 'M': // RI - reverse index
				return true
			case 'c': // RIS - full reset
				return true
			case '=': // DECKPAM - keypad application mode
				return true
			case '>': // DECKPNM - keypad numeric mode
				return true
			case '(': // SCS - designate G0 character set
				return true
			case ')': // SCS - designate G1 character set
				return true
			case '*': // SCS - designate G2 character set
				return true
			case '+': // SCS - designate G3 character set
				return true
			case '#': // DEC test sequences
				if i+2 < len(s) {
					return true
				}
			default:
				// Any other ESC + single char (0x40-0x5F) is a control sequence
				if s[i+1] >= 0x40 && s[i+1] <= 0x5F {
					return true
				}
			}
		}
	}
	return false
}
