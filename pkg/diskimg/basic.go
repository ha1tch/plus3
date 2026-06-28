package diskimg

import (
	"fmt"
	"io"
	"strings"
)

// ReadBasicText reads a BASIC program file from the disk and returns its
// detokenised text. The file must have a PLUS3DOS header identifying it as a
// BASIC program (file type 0); the 128-byte header is skipped before decoding.
func (di *DiskImage) ReadBasicText(diskPath string) (string, error) {
	f, err := di.OpenFile(diskPath, false)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if !f.isHeadered {
		return "", fmt.Errorf("%s has no PLUS3DOS header; not a BASIC program", diskPath)
	}
	if ftype, _, _, _ := f.header.GetBasicHeader(); ftype != FileTypeProgram {
		return "", fmt.Errorf("%s is not a BASIC program (file type %d)", diskPath, ftype)
	}

	if _, err := f.Seek(HeaderSize, io.SeekStart); err != nil {
		return "", err
	}
	body, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return DetokeniseBasic(body)
}

// basicTokens maps Spectrum BASIC keyword token bytes (0xA3-0xFF) to their text.
// 0xA3 (SPECTRUM) and 0xA4 (PLAY) are 128K-only; the rest are common to 48K/128K.
// Verified against the ROM token order (final code = main code + 0xA5) and the
// tokenised-file format reference.
var basicTokens = map[byte]string{
	0xA3: "SPECTRUM", 0xA4: "PLAY", 0xA5: "RND", 0xA6: "INKEY$", 0xA7: "PI",
	0xA8: "FN", 0xA9: "POINT", 0xAA: "SCREEN$", 0xAB: "ATTR", 0xAC: "AT",
	0xAD: "TAB", 0xAE: "VAL$", 0xAF: "CODE", 0xB0: "VAL", 0xB1: "LEN",
	0xB2: "SIN", 0xB3: "COS", 0xB4: "TAN", 0xB5: "ASN", 0xB6: "ACS",
	0xB7: "ATN", 0xB8: "LN", 0xB9: "EXP", 0xBA: "INT", 0xBB: "SQR",
	0xBC: "SGN", 0xBD: "ABS", 0xBE: "PEEK", 0xBF: "IN", 0xC0: "USR",
	0xC1: "STR$", 0xC2: "CHR$", 0xC3: "NOT", 0xC4: "BIN", 0xC5: "OR",
	0xC6: "AND", 0xC7: "<=", 0xC8: ">=", 0xC9: "<>", 0xCA: "LINE",
	0xCB: "THEN", 0xCC: "TO", 0xCD: "STEP", 0xCE: "DEF FN", 0xCF: "CAT",
	0xD0: "FORMAT", 0xD1: "MOVE", 0xD2: "ERASE", 0xD3: "OPEN #", 0xD4: "CLOSE #",
	0xD5: "MERGE", 0xD6: "VERIFY", 0xD7: "BEEP", 0xD8: "CIRCLE", 0xD9: "INK",
	0xDA: "PAPER", 0xDB: "FLASH", 0xDC: "BRIGHT", 0xDD: "INVERSE", 0xDE: "OVER",
	0xDF: "OUT", 0xE0: "LPRINT", 0xE1: "LLIST", 0xE2: "STOP", 0xE3: "READ",
	0xE4: "DATA", 0xE5: "RESTORE", 0xE6: "NEW", 0xE7: "BORDER", 0xE8: "CONTINUE",
	0xE9: "DIM", 0xEA: "REM", 0xEB: "FOR", 0xEC: "GO TO", 0xED: "GO SUB",
	0xEE: "INPUT", 0xEF: "LOAD", 0xF0: "LIST", 0xF1: "LET", 0xF2: "PAUSE",
	0xF3: "NEXT", 0xF4: "POKE", 0xF5: "PRINT", 0xF6: "PLOT", 0xF7: "RUN",
	0xF8: "SAVE", 0xF9: "RANDOMIZE", 0xFA: "IF", 0xFB: "CLS", 0xFC: "DRAW",
	0xFD: "CLEAR", 0xFE: "RETURN", 0xFF: "COPY",
}

// DetokeniseBasic converts a tokenised Sinclair BASIC program (the raw program
// bytes, with no PLUS3DOS header) into readable text.
//
// Program structure: a sequence of lines, each
//
//	[line number: 2 bytes big-endian][length: 2 bytes little-endian][text...][0x0D]
//
// Within the text, keyword tokens (0xA3-0xFF) expand to keywords; a numeric
// constant appears as its visible ASCII digits followed by a 0x0E marker and a
// 5-byte binary form, which is skipped (the visible digits are what we print).
//
// This handles the cases a loader needs: keywords, numbers, strings, and the
// statement separator. It does not attempt to reproduce embedded colour/AT/TAB
// control-code arguments beyond passing printable bytes through.
func DetokeniseBasic(prog []byte) (string, error) {
	var out strings.Builder
	i := 0
	for i < len(prog) {
		if i+4 > len(prog) {
			break // trailing bytes shorter than a line header; stop cleanly
		}
		lineNo := int(prog[i])<<8 | int(prog[i+1])
		length := int(prog[i+2]) | int(prog[i+3])<<8
		i += 4
		if i+length > len(prog) {
			return "", fmt.Errorf("line %d claims %d bytes but only %d remain",
				lineNo, length, len(prog)-i)
		}
		text := prog[i : i+length]
		i += length

		out.WriteString(fmt.Sprintf("%d ", lineNo))
		out.WriteString(detokeniseLine(text))
		out.WriteByte('\n')
	}
	return out.String(), nil
}

// detokeniseLine renders a single line's text (up to and including its 0x0D).
func detokeniseLine(text []byte) string {
	var b strings.Builder
	for j := 0; j < len(text); j++ {
		c := text[j]
		switch {
		case c == 0x0D:
			// End-of-line marker; nothing to emit.
		case c == 0x0E:
			// Number marker: the 5 binary bytes that follow are the value already
			// shown as ASCII digits just before this marker. Skip them.
			j += 5
		case c >= 0xA3:
			kw := basicTokens[c]
			// Keywords need surrounding spaces so tokens don't run together, but
			// the Spectrum's own listing spacing is contextual; a single trailing
			// space after the keyword is the safe, readable choice.
			b.WriteString(kw)
			b.WriteByte(' ')
		case c >= 0x20 && c < 0x7F:
			// Printable ASCII (includes digits, letters, punctuation, quotes).
			b.WriteByte(c)
		default:
			// Non-printable / control byte we don't specifically handle: show it
			// as a hex escape so output stays lossless and obvious.
			b.WriteString(fmt.Sprintf("[%02X]", c))
		}
	}
	return strings.TrimRight(b.String(), " ")
}
