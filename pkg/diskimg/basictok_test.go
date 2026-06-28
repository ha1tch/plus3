package diskimg

import (
	"strings"
	"testing"
)

// canonicalBasic removes all spaces so that round-trip comparisons are not
// defeated by the detokeniser's display spacing around tokens (e.g. it renders
// the CODE token as "CODE " and ">=" as ">= "). Spacing is a display choice in
// the detokeniser, not part of the tokenised bytes, so it is not compared here.
// Note spaces inside string literals would also be stripped; the round-trip
// cases below deliberately avoid relying on intra-string spacing for equality
// (string contents are checked separately in TestTokeniseKeywordsInStringsAndRem).
func canonicalBasic(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func TestTokeniseRoundTrip(t *testing.T) {
	cases := []string{
		`10 CLEAR 32767: LOAD "detect"CODE: RANDOMIZE USR 32768`,
		`20 PRINT "HELLO WORLD"`,
		`30 FOR I=1 TO 10: NEXT I`,
		`40 IF A>=5 THEN GO TO 100`,
		`50 BORDER 0: PAPER 0: INK 7: CLS`,
		`70 POKE 23606,0: RANDOMIZE USR 15619`,
	}
	for _, src := range cases {
		tok, err := TokeniseBasic(src)
		if err != nil {
			t.Errorf("TokeniseBasic(%q): %v", src, err)
			continue
		}
		back, err := DetokeniseBasic(tok)
		if err != nil {
			t.Errorf("DetokeniseBasic round-trip of %q: %v", src, err)
			continue
		}
		got := canonicalBasic(back)
		want := canonicalBasic(src)
		if got != want {
			t.Errorf("round-trip mismatch:\n  src:  %q\n  got:  %q", want, got)
		}
	}
}

func TestTokeniseKeywordsInStringsAndRem(t *testing.T) {
	// Keywords inside strings and after REM must stay literal, not be tokenised.
	src := `10 REM PRINT and GO TO are just words here`
	tok, err := TokeniseBasic(src)
	if err != nil {
		t.Fatalf("TokeniseBasic: %v", err)
	}
	// After the REM token (0xEA) the bytes must be the literal ASCII text, so
	// the word PRINT must appear as ASCII, not as the 0xF5 token.
	if strings.Contains(string(tok), string([]byte{0xF5})) {
		t.Error("PRINT was tokenised inside a REM comment; should be literal")
	}

	src2 := `20 PRINT "GO TO PRINT"`
	tok2, err := TokeniseBasic(src2)
	if err != nil {
		t.Fatalf("TokeniseBasic: %v", err)
	}
	back2, _ := DetokeniseBasic(tok2)
	if !strings.Contains(back2, `"GO TO PRINT"`) {
		t.Errorf("keywords inside string literal were altered: %q", back2)
	}
}

func TestTokeniseRejectsBadLineNumber(t *testing.T) {
	if _, err := TokeniseBasic("PRINT 1"); err == nil {
		t.Error("expected error for a line without a line number")
	}
	if _, err := TokeniseBasic("70000 PRINT 1"); err == nil {
		t.Error("expected error for a line number above 9999")
	}
}

func TestLooksTokenised(t *testing.T) {
	// Real tokenised BASIC (line 10: PRINT "HI").
	tok, err := TokeniseBasic(`10 PRINT "HI"`)
	if err != nil {
		t.Fatal(err)
	}
	if !LooksTokenised(tok) {
		t.Error("tokenised BASIC not recognised as tokenised")
	}

	// Plain-text source must NOT look tokenised.
	if LooksTokenised([]byte(`10 PRINT "HI"`)) {
		t.Error("plain-text source misidentified as tokenised")
	}

	// Arbitrary binary must not look tokenised.
	if LooksTokenised([]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA}) {
		t.Error("binary data misidentified as tokenised")
	}

	// Multi-line tokenised program with ascending line numbers.
	multi, _ := TokeniseBasic("10 PRINT \"A\"\n20 PRINT \"B\"")
	if !LooksTokenised(multi) {
		t.Error("multi-line tokenised program not recognised")
	}
}
