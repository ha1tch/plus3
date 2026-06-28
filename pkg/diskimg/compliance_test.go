package diskimg

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// These tests lock in the +3DOS-compliance invariants that a real ZX Spectrum +3
// caught during bring-up. Each corresponds to a specific bug that was invisible to
// software round-tripping (the reader shared the writer's assumptions) and only
// surfaced on hardware or by diffing against a +3DOS-written disk. They are cheap
// guards against silent regression of behaviour that took real hardware to find.

// buildTestDisk imports the sample fixture as a CODE file and returns the saved
// image bytes plus the first directory entry, so assertions can inspect both the
// on-disk layout and the parsed entry.
func buildTestDisk(t *testing.T, loadAddr uint16) (image []byte, entry DirectoryEntry) {
	t.Helper()
	dir := t.TempDir()

	src, err := os.ReadFile("testdata/sample.bin")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srcPath := filepath.Join(dir, "sample.bin")
	if err := os.WriteFile(srcPath, src, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	di := NewDiskImage()
	if err := di.ImportCode(srcPath, loadAddr); err != nil {
		t.Fatalf("ImportCode: %v", err)
	}
	diskPath := filepath.Join(dir, "test.dsk")
	if err := di.SaveToFile(diskPath); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	image, err = os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("read disk: %v", err)
	}

	reloaded, err := LoadFromFile(diskPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	entries, err := reloaded.GetDirectory()
	if err != nil {
		t.Fatalf("GetDirectory: %v", err)
	}
	for _, e := range entries {
		if !e.IsUnused() && e.GetFilename() != "" {
			return image, e
		}
	}
	t.Fatal("no file entry found after reload")
	return nil, DirectoryEntry{}
}

// firstHeader locates the PLUS3DOS header on the disk image (it begins the file's
// data) and returns its 128 bytes.
func firstHeader(t *testing.T, image []byte) []byte {
	t.Helper()
	i := bytes.Index(image, []byte("PLUS3DOS"))
	if i < 0 {
		t.Fatal("no PLUS3DOS header on disk")
	}
	return image[i : i+128]
}

// Bug 1: files must be written in user area 0. The +3 catalogs user 0 by default;
// a file in user area 1 is invisible ("disk empty").
func TestComplianceUserAreaZero(t *testing.T) {
	_, entry := buildTestDisk(t, 0x8000)
	if entry.Status != 0x00 {
		t.Errorf("directory status = %#x, want 0x00 (user 0)", entry.Status)
	}
}

// Bug 2: the directory's allocation field must hold block NUMBERS, not the block
// count. The first allocation entry must point at the block where the data
// actually lives (the block whose first bytes are the PLUS3DOS header).
func TestComplianceAllocationHoldsBlockNumbers(t *testing.T) {
	image, entry := buildTestDisk(t, 0x8000)

	// Collect the non-zero block numbers; they must be sequential and plausible,
	// not a single small count value.
	var blocks []int
	for _, b := range entry.AllocationBlocks {
		if b != 0 {
			blocks = append(blocks, int(b))
		}
	}
	if len(blocks) == 0 {
		t.Fatal("no allocation blocks recorded")
	}

	// The first block must contain the PLUS3DOS header. Map block -> file offset
	// using the same geometry the writer uses (data area starts at track 1).
	ts := int(image[0x32]) | int(image[0x33])<<8
	const sec, spt = 512, 9
	linear := blocks[0] * 2 // two 512-byte sectors per 1K block
	track := 1 + linear/spt
	sector := linear % spt
	off := 0x100 + track*ts + 0x100 + sector*sec
	if !bytes.HasPrefix(image[off:], []byte("PLUS3DOS")) {
		t.Errorf("first allocation block %d does not point at the file header", blocks[0])
	}
}

// Bug 3: allocation must not over-reserve. The sample is 600 bytes + 128-byte
// header = 728 bytes -> ceil(728/1024) = 1 block. Allow no more than that.
func TestComplianceNoOverAllocation(t *testing.T) {
	_, entry := buildTestDisk(t, 0x8000)
	count := 0
	for _, b := range entry.AllocationBlocks {
		if b != 0 {
			count++
		}
	}
	// 728 bytes fits in a single 1K block.
	if count != 1 {
		t.Errorf("allocated %d blocks for a 728-byte file, want 1", count)
	}
}

// Bug 4 and 5: the PLUS3DOS header must use version 0 (a higher version is
// rejected by +3DOS as "wrong file type"), declare file type 3 (CODE), put the
// load address in the first CODE parameter, and 0x8000 in the second - matching
// what +3DOS itself writes. The checksum must be valid.
func TestComplianceHeaderMatchesPlus3DOS(t *testing.T) {
	const loadAddr = 30000
	image, _ := buildTestDisk(t, loadAddr)
	h := firstHeader(t, image)

	if h[10] != 0 {
		t.Errorf("header version = %d, want 0", h[10])
	}
	// BASIC sub-header begins at offset 15: [15]=type, [16:18]=len, [18:20]=p1, [20:22]=p2.
	if h[15] != 3 {
		t.Errorf("file type = %d, want 3 (CODE)", h[15])
	}
	if p1 := binary.LittleEndian.Uint16(h[18:20]); p1 != loadAddr {
		t.Errorf("CODE p1 (load address) = %d, want %d", p1, loadAddr)
	}
	if p2 := binary.LittleEndian.Uint16(h[20:22]); p2 != 0x8000 {
		t.Errorf("CODE p2 = %#x, want 0x8000", p2)
	}
	var sum int
	for _, b := range h[:127] {
		sum += int(b)
	}
	if byte(sum%256) != h[127] {
		t.Errorf("header checksum invalid: stored %#x, computed %#x", h[127], byte(sum%256))
	}
}

// The directory entry must serialise to exactly 32 bytes - the canonical CP/M
// layout. A struct of any other size shifts every subsequent entry and corrupts
// the directory (this was a real 35-byte bug).
func TestComplianceDirectoryEntryIs32Bytes(t *testing.T) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, DirectoryEntry{}); err != nil {
		t.Fatalf("serialise: %v", err)
	}
	if buf.Len() != 32 {
		t.Errorf("DirectoryEntry serialises to %d bytes, want 32", buf.Len())
	}
}
