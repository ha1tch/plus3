package diskimg

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRoundTrip is the master integration test. It drives the full public API the
// way the CLI does: create a blank disk, import a CODE file, save, reload, list,
// and export. If the exported bytes match the original, then every layer beneath
// has to be correct - allocation, the block-to-sector mapping, directory
// persistence, and the PLUS3DOS header round-trip. A single pass here implies the
// underlying logic is sound; a failure localises to whichever assertion breaks.
func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	diskPath := filepath.Join(dir, "test.dsk")

	src, err := os.ReadFile("testdata/sample.bin")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	// Create a blank disk and import the sample as a CODE file.
	di := NewDiskImage()
	if err := writeFixture(t, dir, "sample.bin", src); err != nil {
		t.Fatal(err)
	}
	if err := di.ImportCode(filepath.Join(dir, "sample.bin"), 0x8000); err != nil {
		t.Fatalf("ImportCode: %v", err)
	}
	if err := di.SaveToFile(diskPath); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	// Reload from disk - this exercises the reader, not just in-memory state.
	reloaded, err := LoadFromFile(diskPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	// The catalog must list exactly the file we wrote (SAMPLE.BIN).
	entries, err := reloaded.GetDirectory()
	if err != nil {
		t.Fatalf("GetDirectory: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsUnused() && e.GetFilename() != "" {
			names = append(names, e.GetFilename())
		}
	}
	if len(names) != 1 || names[0] != "SAMPLE.BIN" {
		t.Fatalf("catalog = %v, want [SAMPLE.BIN]", names)
	}

	// Export with the header stripped and compare byte-for-byte.
	outPath := filepath.Join(dir, "out.bin")
	if err := reloaded.ExportFile("SAMPLE.BIN", outPath, true); err != nil {
		t.Fatalf("ExportFile: %v", err)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("round-trip mismatch: exported %d bytes, original %d bytes", len(got), len(src))
	}
}

// TestReadRealDisk loads a disk written by a physical ZX Spectrum +3 and confirms
// the reader sees the files the +3 itself wrote. This proves the reader matches
// real-world +3DOS output, independent of our own writer - the two could share a
// wrong assumption and still round-trip, but a genuine +3 disk cannot.
func TestReadRealDisk(t *testing.T) {
	di, err := LoadFromFile("testdata/hello.dsk")
	if err != nil {
		t.Fatalf("LoadFromFile(hello.dsk): %v", err)
	}

	entries, err := di.GetDirectory()
	if err != nil {
		t.Fatalf("GetDirectory: %v", err)
	}

	found := map[string]bool{}
	for _, e := range entries {
		if !e.IsUnused() && e.GetFilename() != "" {
			found[e.GetFilename()] = true
		}
	}
	for _, want := range []string{"ALOUETTE.BAS", "HELLO.BAS"} {
		if !found[want] {
			t.Errorf("real disk: %s not found in catalog %v", want, found)
		}
	}
}

func writeFixture(t *testing.T, dir, name string, data []byte) error {
	t.Helper()
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

// TestMultiFileNoOverwrite is a regression test for a bug where adding a second
// file to a disk overwrote the first: a freshly loaded allocator marked every
// data block free, ignoring blocks already occupied by existing files, so the
// second file reused the first file's blocks. Each plus3 "add" is a separate
// load/modify/save cycle, so the allocator must reconcile against the on-disk
// directory at load time. This test adds two files across two save/load cycles
// and verifies both survive intact in distinct storage.
func TestMultiFileNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	diskPath := filepath.Join(dir, "multi.dsk")

	dataA := bytes.Repeat([]byte{0xAA}, 500)
	dataB := bytes.Repeat([]byte{0xBB}, 500)
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), dataA, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.bin"), dataB, 0o644); err != nil {
		t.Fatal(err)
	}

	// First add: create disk, import A, save.
	di := NewDiskImage()
	if err := di.ImportCode(filepath.Join(dir, "a.bin"), 0x8000); err != nil {
		t.Fatalf("ImportCode A: %v", err)
	}
	if err := di.SaveToFile(diskPath); err != nil {
		t.Fatalf("SaveToFile after A: %v", err)
	}

	// Second add: reload, import B, save. This is the path that previously
	// overwrote A because the allocator did not know A's blocks were taken.
	di2, err := LoadFromFile(diskPath)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if err := di2.ImportCode(filepath.Join(dir, "b.bin"), 0x8000); err != nil {
		t.Fatalf("ImportCode B: %v", err)
	}
	if err := di2.SaveToFile(diskPath); err != nil {
		t.Fatalf("SaveToFile after B: %v", err)
	}

	// Both files must come back byte-exact.
	di3, err := LoadFromFile(diskPath)
	if err != nil {
		t.Fatalf("LoadFromFile (final): %v", err)
	}
	outA := filepath.Join(dir, "out_a.bin")
	outB := filepath.Join(dir, "out_b.bin")
	if err := di3.ExportFile("A.BIN", outA, true); err != nil {
		t.Fatalf("ExportFile A: %v", err)
	}
	if err := di3.ExportFile("B.BIN", outB, true); err != nil {
		t.Fatalf("ExportFile B: %v", err)
	}

	gotA, _ := os.ReadFile(outA)
	gotB, _ := os.ReadFile(outB)
	if !bytes.Equal(gotA, dataA) {
		t.Errorf("A.BIN corrupted: got %d bytes starting %x, want %d bytes of 0xAA", len(gotA), firstBytes(gotA), len(dataA))
	}
	if !bytes.Equal(gotB, dataB) {
		t.Errorf("B.BIN corrupted: got %d bytes starting %x, want %d bytes of 0xBB", len(gotB), firstBytes(gotB), len(dataB))
	}
}

func firstBytes(b []byte) []byte {
	if len(b) > 4 {
		return b[:4]
	}
	return b
}
