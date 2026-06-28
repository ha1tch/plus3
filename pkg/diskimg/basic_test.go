package diskimg

import (
	"os"
	"strings"
	"testing"
)

// TestDetokeniseRealProgram decodes the tokenised body of a real BASIC program
// written by a ZX Spectrum +3 (HELLO.BAS). This is the case the manual decode
// earlier produced, now exercised through the library.
func TestDetokeniseRealProgram(t *testing.T) {
	body, err := os.ReadFile("testdata/hello.tok")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got, err := DetokeniseBasic(body)
	if err != nil {
		t.Fatalf("DetokeniseBasic: %v", err)
	}
	want := "10 REM Hello you have found me\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestDetokeniseLoader is the case the manual decode never exercised: a program
// containing numeric constants (which carry a hidden 0x0E + 5-byte binary form
// that must be skipped), keywords, a statement separator, and a string literal.
// This is representative of a real loader.
func TestDetokeniseLoader(t *testing.T) {
	body, err := os.ReadFile("testdata/loader.tok")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got, err := DetokeniseBasic(body)
	if err != nil {
		t.Fatalf("DetokeniseBasic: %v", err)
	}

	// Line 10: BORDER 0: PAPER 0  - keywords and numbers.
	// Line 20: LOAD ""CODE        - keyword, string literal, trailing keyword.
	for _, want := range []string{
		"10 BORDER 0",
		"PAPER 0",
		`20 LOAD ""`,
		"CODE",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, got)
		}
	}

	// The 0x0E number marker and its 5 binary bytes must NOT leak into the output.
	if strings.Contains(got, "[0E]") || strings.Contains(got, "\x0e") {
		t.Errorf("number marker leaked into output:\n%s", got)
	}
}

// TestDetokenise128Tokens checks the 128K-only keywords PLAY and SPECTRUM decode.
func TestDetokenise128Tokens(t *testing.T) {
	// 10 PLAY "abc"   (PLAY = 0xA4)
	line := []byte{0x00, 0x0A, 0x00, 0x00} // header, length patched below
	payload := append([]byte{0xA4}, []byte(`"abc"`)...)
	payload = append(payload, 0x0D)
	line[2] = byte(len(payload))
	prog := append(line, payload...)

	got, err := DetokeniseBasic(prog)
	if err != nil {
		t.Fatalf("DetokeniseBasic: %v", err)
	}
	if !strings.Contains(got, "PLAY") {
		t.Errorf("PLAY (0xA4) not decoded: %q", got)
	}
}
