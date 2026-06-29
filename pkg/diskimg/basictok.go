package diskimg

import (
	zbasic "github.com/ha1tch/zentools/pkg/basic"
)

// LooksTokenised reports whether data appears to be already-tokenised BASIC
// rather than plain-text source. It is a structural check (the buffer must parse
// cleanly as a sequence of tokenised lines with ascending line numbers),
// suitable for an advisory warning. The implementation is provided by
// zentools/pkg/basic, the verified interchange tokeniser.
func LooksTokenised(data []byte) bool {
	return zbasic.LooksTokenised(data)
}

// TokeniseBasic converts plain-text Sinclair BASIC source into its on-disk
// tokenised byte form. It is the inverse of DetokeniseBasic.
//
// Input is one BASIC line per text line, each beginning with a line number, for
// example:
//
//	10 CLEAR 32767: LOAD "game"CODE: RANDOMIZE USR 32768
//	20 PRINT "DONE"
//
// Scope: keywords, integer numeric constants, string literals, and REM comments.
// Keywords are matched longest-first and only outside string literals and REM
// text. Numeric constants are emitted in the ROM integer form (the visible
// digits followed by 0x0E and a 5-byte value), for constants in 0..65535;
// floating-point literals, DEF FN calculator slots, and embedded colour-control
// argument bytes are not produced.
//
// The result is the raw tokenised program (no PLUS3DOS header). Tokenisation is
// performed by zentools/pkg/basic; matching is case-insensitive, as before.
func TokeniseBasic(src string) ([]byte, error) {
	return zbasic.Tokenise(src)
}
