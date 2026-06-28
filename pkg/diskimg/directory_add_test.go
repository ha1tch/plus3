package diskimg

import "testing"

// These tests pin down the behaviour of the directory add operation: placement in
// the first free slot, user-area-0 enforcement, no overwrite of an existing file,
// and an error on a full directory.

// namedEntry builds a directory entry with a recognisable name in user area 0.
func namedEntry(name string) DirectoryEntry {
	var e DirectoryEntry
	e.Status = 0x00
	copy(e.Name[:], name)
	return e
}

// emptyDir returns a directory of n entries, all unused (0xE5).
func emptyDir(n int) *Directory {
	d := &Directory{Entries: make([]DirectoryEntry, n)}
	for i := range d.Entries {
		d.Entries[i].Status = 0xE5
	}
	return d
}

// addFn lets the same tests run against the add operation. It remains a table so
// additional entry points can be covered here if any are introduced.
type addFn struct {
	name string
	add  func(*Directory, DirectoryEntry) error
}

func addFns() []addFn {
	return []addFn{
		{"AddFile", (*Directory).AddFile},
	}
}

// A new file goes into the first free slot.
func TestAddFile_UsesFirstFreeSlot(t *testing.T) {
	for _, f := range addFns() {
		t.Run(f.name, func(t *testing.T) {
			d := emptyDir(4)
			if err := f.add(d, namedEntry("ALPHA")); err != nil {
				t.Fatalf("add: %v", err)
			}
			if got := d.Entries[0].GetFilename(); got != "ALPHA" {
				t.Errorf("slot 0 = %q, want ALPHA", got)
			}
			// A second file takes the next slot, not slot 0.
			if err := f.add(d, namedEntry("BETA")); err != nil {
				t.Fatalf("add 2: %v", err)
			}
			if got := d.Entries[1].GetFilename(); got != "BETA" {
				t.Errorf("slot 1 = %q, want BETA", got)
			}
			if got := d.Entries[0].GetFilename(); got != "ALPHA" {
				t.Errorf("slot 0 overwritten: %q, want ALPHA", got)
			}
		})
	}
}

// The stored entry is forced to user area 0, even if the caller passes a
// different status. (A non-zero user area would hide the file from the +3.)
func TestAddFile_ForcesUserZero(t *testing.T) {
	for _, f := range addFns() {
		t.Run(f.name, func(t *testing.T) {
			d := emptyDir(4)
			e := namedEntry("GAMMA")
			e.Status = 0x05 // caller asks for user area 5
			if err := f.add(d, e); err != nil {
				t.Fatalf("add: %v", err)
			}
			if d.Entries[0].Status != 0x00 {
				t.Errorf("status = %#x, want 0x00 (user 0)", d.Entries[0].Status)
			}
		})
	}
}

// Adding does not overwrite an existing user-0 file: a real file (status 0x00
// with a name) is not a free slot.
func TestAddFile_DoesNotOverwriteExistingFile(t *testing.T) {
	for _, f := range addFns() {
		t.Run(f.name, func(t *testing.T) {
			d := emptyDir(4)
			d.Entries[0] = namedEntry("EXISTS") // occupied, user 0
			if err := f.add(d, namedEntry("NEW")); err != nil {
				t.Fatalf("add: %v", err)
			}
			if got := d.Entries[0].GetFilename(); got != "EXISTS" {
				t.Errorf("existing file clobbered: slot 0 = %q, want EXISTS", got)
			}
			if got := d.Entries[1].GetFilename(); got != "NEW" {
				t.Errorf("new file misplaced: slot 1 = %q, want NEW", got)
			}
		})
	}
}

// A full directory returns an error rather than silently dropping the file.
func TestAddFile_FullDirectoryErrors(t *testing.T) {
	for _, f := range addFns() {
		t.Run(f.name, func(t *testing.T) {
			d := &Directory{Entries: make([]DirectoryEntry, 2)}
			d.Entries[0] = namedEntry("ONE")
			d.Entries[1] = namedEntry("TWO")
			if err := f.add(d, namedEntry("THREE")); err == nil {
				t.Error("expected error on full directory, got nil")
			}
		})
	}
}
