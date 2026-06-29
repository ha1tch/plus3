// file: pkg/diskimg/convert_test.go

package diskimg

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zentools/pkg/tap"
)

// TestConvertDiskToTAP_CodeFile imports a CODE file, exports it to TAP, and
// verifies the TAP bytes are well-formed: exactly a header block and a data
// block, both with correct checksums, correct load address, and data identical
// to the original. This guards against the historic convert.go bugs (stray
// checksum byte, malformed header, wrong checksum computation).
func TestConvertDiskToTAP_CodeFile(t *testing.T) {
	dir := t.TempDir()
	payload := []byte("Hello, +3DOS to TAP. " + string(make([]byte, 300)))
	for i := range payload {
		if payload[i] == 0 {
			payload[i] = byte(i & 0xFF)
		}
	}
	binPath := filepath.Join(dir, "GAME.BIN")
	if err := os.WriteFile(binPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	di := NewDiskImage()
	if err := di.ImportCode(binPath, 0x8000); err != nil {
		t.Fatalf("ImportCode: %v", err)
	}

	var buf bytes.Buffer
	if err := di.ConvertDiskToTAP("GAME.BIN", &buf); err != nil {
		t.Fatalf("ConvertDiskToTAP: %v", err)
	}

	// Independently decode the TAP bytes.
	blocks, err := tap.Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("the produced TAP does not decode: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("TAP has %d blocks, want 2 (header + data)", len(blocks))
	}
	hdr, data := blocks[0], blocks[1]
	if !hdr.IsHeader || !hdr.ChecksumOK {
		t.Errorf("header block: IsHeader=%v ChecksumOK=%v", hdr.IsHeader, hdr.ChecksumOK)
	}
	if !data.ChecksumOK {
		t.Error("data block checksum mismatch")
	}
	if hdr.Type != tap.TypeCode {
		t.Errorf("type = %d, want Code(%d)", hdr.Type, tap.TypeCode)
	}
	if hdr.Param1 != 0x8000 {
		t.Errorf("load address = 0x%04X, want 0x8000", hdr.Param1)
	}
	if int(hdr.DataLength) != len(payload) {
		t.Errorf("header DataLength = %d, want %d", hdr.DataLength, len(payload))
	}
	if !bytes.Equal(data.Data, payload) {
		t.Errorf("TAP data differs from original payload (%d vs %d bytes)", len(data.Data), len(payload))
	}
}

// TestTAPDiskRoundTrip exports a CODE file to TAP and re-imports it to a fresh
// disk, confirming the data survives a full disk->TAP->disk cycle.
func TestTAPDiskRoundTrip(t *testing.T) {
	dir := t.TempDir()
	payload := make([]byte, 512)
	for i := range payload {
		payload[i] = byte((i*7 + 1) & 0xFF)
	}
	binPath := filepath.Join(dir, "DATA.BIN")
	if err := os.WriteFile(binPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	di := NewDiskImage()
	if err := di.ImportCode(binPath, 0x9000); err != nil {
		t.Fatalf("ImportCode: %v", err)
	}
	var buf bytes.Buffer
	if err := di.ConvertDiskToTAP("DATA.BIN", &buf); err != nil {
		t.Fatalf("ConvertDiskToTAP: %v", err)
	}

	// Import the TAP into a new disk, then re-export to TAP. Comparing at the
	// TAP layer (where the data block carries an exact length) verifies the
	// conversion round-trips without depending on ExportFile's record padding.
	di2 := NewDiskImage()
	if err := di2.ConvertTAPtoDisk(&buf, "RELOAD.BIN"); err != nil {
		t.Fatalf("ConvertTAPtoDisk: %v", err)
	}

	var buf2 bytes.Buffer
	if err := di2.ConvertDiskToTAP("RELOAD.BIN", &buf2); err != nil {
		t.Fatalf("ConvertDiskToTAP (reloaded): %v", err)
	}
	blocks, err := tap.Decode(buf2.Bytes())
	if err != nil {
		t.Fatalf("re-exported TAP does not decode: %v", err)
	}
	if len(blocks) != 2 || !blocks[1].ChecksumOK {
		t.Fatalf("re-exported TAP malformed: %d blocks", len(blocks))
	}
	if !bytes.Equal(blocks[1].Data, payload) {
		t.Errorf("round-trip data mismatch: got %d bytes, want %d", len(blocks[1].Data), len(payload))
	}
}
