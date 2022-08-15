package progress

import "strconv"

// XTerm Control Sequences
// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html

// ANSI CSI sequences
// https://en.wikipedia.org/wiki/ANSI.SYS

// 8-Bit Coded Character Set Structure and Rules
// https://www.ecma-international.org/publications-and-standards/standards/ecma-43/

// https://en.wikipedia.org/wiki/ANSI_escape_code#ESC
const ESC = 0x1b

// https://en.wikipedia.org/wiki/ANSI_escape_code#CSIsection
var CSI = []byte{ESC, '['}

// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_

// Cursor Up
func CUU(n uint8) []byte { return csi(n, 'A') }

// Cursor Down
func CUD(n uint8) []byte { return csi(n, 'B') }

// Cursor Forward
func CUF(n uint8) []byte { return csi(n, 'C') }

// Cursor Backward
func CUB(n uint8) []byte { return csi(n, 'D') }

// Cursor Next Line
func CNL(n uint8) []byte { return csi(n, 'E') }

// Cursor Previous Line
func CPL(n uint8) []byte { return csi(n, 'F') }

// Cursor Horizontal Absolute
func CHA(n uint8) []byte { return csi(n, 'G') }

// Cursor Position
func CUP(row, col uint8) []byte { return append(CSI, byte(row), ';', byte(col), 'H') }

// Cursor Forward Tabulation Ps tab stops (default = 1)
func CHT(n uint8) []byte { return csi(n, 'I') }

// Erase in Display
// Clears part of the screen.
// If n is 0 (or missing), clear from cursor to end of screen.
// If n is 1, clear from cursor to beginning of the screen.
// If n is 2, clear entire screen (and moves cursor to upper left on DOS ANSI.SYS).
// If n is 3, clear entire screen and delete all lines saved in the scrollback buffer (this feature was added for xterm and is supported by other terminal applications).
func ED(n uint8) []byte { return csi(n, 'J') }

// Erase in Line
// Erases part of the line.
// If n is 0 (or missing), clear from cursor to the end of the line.
// If n is 1, clear from cursor to beginning of the line.
// If n is 2, clear entire line. Cursor position does not change.
func EL(n uint8) []byte { return csi(n, 'K') }

// Scroll Up
// Scroll whole page up by n (default 1) lines. New lines are added at the bottom. (not ANSI.SYS)
func SU(n uint8) []byte { return csi(n, 'S') }

// Scroll Down
// Scroll whole page down by n (default 1) lines. New lines are added at the top. (not ANSI.SYS)
func SD(n uint8) []byte { return csi(n, 'T') }

// Horizontal Vertical Position
// Same as CUP, but counts as a format effector function (like CR or LF) rather than an editor function (like CUD or CNL).
// This can lead to different handling in certain terminal modes.
func HVP(row, col uint8) []byte { return csi2(row, col, 'f') }

// Select Graphic Rendition
// Sets colors and style of the characters following this code
// https://en.wikipedia.org/wiki/ANSI_escape_code#SGR

const (
	SGR_RESET      = 0 // Reset (default)
	SGR_BOLD       = 1 // Bold
	SGR_HB         = 2 // Halfbright
	SGR_ITALIC     = 3 // Italic
	SGR_UNDER      = 4 // Underline
	SGR_BLINK      = 5 // Blink
	SGR_RAPIDBLINK = 6 // Rapid blink
	SGR_REV        = 7 // Reverse video
	SGR_INVIS      = 8 // Invisible
	SGR_STRIK      = 9 // Strikethrough

	SGR_PRI_FONT   = 10 // Primary font
	SGR_ALT_FONT_1 = 11 // Alternative font 1
	SGR_ALT_FONT_2 = 12 // Alternative font 2
	SGR_ALT_FONT_3 = 13 // Alternative font 3
	SGR_ALT_FONT_4 = 14 // Alternative font 4
	SGR_ALT_FONT_5 = 15 // Alternative font 5
	SGR_ALT_FONT_6 = 16 // Alternative font 6
	SGR_ALT_FONT_7 = 17 // Alternative font 7
	SGR_ALT_FONT_8 = 18 // Alternative font 8
	SGR_ALT_FONT_9 = 19 // Alternative font 9

	SGR_FRAKTUR  = 20 // Fraktur
	SGR_DBLUNDER = 21 // Doubly underlined
	SGR_NORMAL   = 22 // Normal intensity
	SGR_NITALIC  = 23 // Neither italic nor blackletter
	SGR_NUDER    = 24 // Not underlined
	SGR_NBLK     = 25 // Not blinking
	SGR_PROP     = 26 // Proportional spacing
	SGR_NREV     = 27 // Not reverse video
	SGR_REVEAL   = 28 // Reveal
	SGR_NCROSS   = 29 // Not crossed out

	SGR_FG_BLACK   = 30 // Black foreground
	SGR_FG_RED     = 31 // Red foreground
	SGR_FG_GREEN   = 32 // Green foreground
	SGR_FG_YELLOW  = 33 // Yellow foreground
	SGR_FG_BLUE    = 34 // Blue foreground
	SGR_FG_MAGENTA = 35 // Magenta foreground
	SGR_FG_CYAN    = 36 // Cyan foreground
	SGR_FG_WHITE   = 37 // White foreground
	SGR_FG_SET     = 38 // Set foreground color
	SGR_FG_DEFAULT = 39 // Default foreground

	SGR_BG_BLACK   = 40 // Black background
	SGR_BG_RED     = 41 // Red background
	SGR_BG_GREEN   = 42 // Green background
	SGR_BG_YELLOW  = 43 // Yellow background
	SGR_BG_BLUE    = 44 // Blue background
	SGR_BG_MAGENTA = 45 // Magenta background
	SGR_BG_CYAN    = 46 // Cyan background
	SGR_BG_WHITE   = 47 // White background
	SGR_BG_SET     = 48 // Set background color
	SGR_BG_DEFAULT = 49 // Default background

	SGR_NOPROP      = 50 // Disable proportional spacing
	SGR_FRAMED      = 51 // Framed
	SGR_ENCIRCLED   = 52 // Encircled
	SGR_OVERLINED   = 53 // Overlined
	SGR_NFRAME      = 54 // Not framed or encircled
	SGR_NOVERLINED  = 57 // Not overlined
	SGR_UNDERCOLOR  = 58 // Set underline color
	SGR_NUNDERCOLOR = 59 // Default underline color

	SGR_FG_BRIGHT_BLACK   = 90 // Bright black foreground
	SGR_FG_BRIGHT_RED     = 91 // Bright red foreground
	SGR_FG_BRIGHT_GREEN   = 92 // Bright green foreground
	SGR_FG_BRIGHT_YELLOW  = 93 // Bright yellow foreground
	SGR_FG_BRIGHT_BLUE    = 94 // Bright blue foreground
	SGR_FG_BRIGHT_MAGENTA = 95 // Bright magenta foreground
	SGR_FG_BRIGHT_CYAN    = 96 // Bright cyan foreground
	SGR_FG_BRIGHT_WHITE   = 97 // Bright white foreground

	SGR_BG_BRIGHT_BLACK   = 100 // Bright black background
	SGR_BG_BRIGHT_RED     = 101 // Bright red background
	SGR_BG_BRIGHT_GREEN   = 102 // Bright green background
	SGR_BG_BRIGHT_YELLOW  = 103 // Bright yellow background
	SGR_BG_BRIGHT_BLUE    = 104 // Bright blue background
	SGR_BG_BRIGHT_MAGENTA = 105 // Bright magenta background
	SGR_BG_BRIGHT_CYAN    = 106 // Bright cyan background
	SGR_BG_BRIGHT_WHITE   = 107 // Bright white background
)

func SGR(n uint8) []byte { return csi(n, 'm') }

// https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit
// ESC[38;5;231m
func SGR256_FG(n uint8) []byte { return csi3(SGR_FG_SET, 5, n, 'm') }

func SGR256_BG(n uint8) []byte { return csi3(SGR_BG_SET, 5, n, 'm') }

// Device Status Report
func DSR() []byte { return csi(6, 'n') }

func csi(n uint8, i byte) []byte {
	// example: [32m
	buf := make([]byte, 5)
	buf = strconv.AppendUint(buf[:copy(buf, CSI)], uint64(n), 10)
	return append(buf, i)
}

func csi2(n1 uint8, n2 uint8, i byte) []byte {
	// example: [14;1f
	buf := make([]byte, 8)
	buf = strconv.AppendUint(buf[:copy(buf, CSI)], uint64(n1), 10)
	buf = append(buf, ';')
	buf = strconv.AppendUint(buf, uint64(n2), 10)
	return append(buf, i)
}

func csi3(n1 uint8, n2 uint8, n3 uint8, i byte) []byte {
	// example: [38;5;231m
	buf := make([]byte, 11)
	buf = strconv.AppendUint(buf[:copy(buf, CSI)], uint64(n1), 10)
	buf = append(buf, ';')
	buf = strconv.AppendUint(buf, uint64(n2), 10)
	buf = append(buf, ';')
	buf = strconv.AppendUint(buf, uint64(n3), 10)
	return append(buf, i)
}
