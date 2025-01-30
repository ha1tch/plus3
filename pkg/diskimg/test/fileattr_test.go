// file: pkg/diskimg/fileattr_test.go

package diskimg

import (
	"testing"
)

func TestFileAttributes(t *testing.T) {
	attrs := &FileAttributes{
		ReadOnly: true,
		System:   true,
		Archived: false,
		UserF1:   true,
		UserF2:   false,
		UserF3:   true,
		UserF4:   false,
	}

	// Test type attributes
	typeAttrs := attrs.GetTypeAttributes()
	if typeAttrs != (AttrReadOnly | AttrSystem) {
		t.Errorf("GetTypeAttributes wrong value: got %02x, want %02x", 
			typeAttrs, AttrReadOnly|AttrSystem)
	}

	// Test setting type attributes
	newAttrs := &FileAttributes{}
	newAttrs.SetTypeAttributes(AttrReadOnly | AttrArchived)
	if !newAttrs.ReadOnly || newAttrs.System || !newAttrs.Archived {
		t.Error("SetTypeAttributes failed to set correct values")
	}

	// Test name attributes
	nameAttrs := attrs.GetNameAttributes()
	if nameAttrs[0]&AttrUserF1 == 0 || nameAttrs[2]&AttrUserF3 == 0 {
		t.Error("GetNameAttributes failed to set user flags")
	}
	if nameAttrs[1]&AttrUserF2 != 0 || nameAttrs[3]&AttrUserF4 != 0 {
		t.Error("GetNameAttributes set wrong user flags")
	}
	
	// Verify reserved bits (f5-f8) are always 0
	for i := 4; i < 8; i++ {
		if nameAttrs[i] != 0 {
			t.Errorf("Reserved attribute f%d is not zero", i+1)
		}
	}

	// Test setting name attributes
	newAttrs = &FileAttributes{}
	var testNameAttrs [8]byte
	testNameAttrs[0] = AttrUserF1
	testNameAttrs[2] = AttrUserF3
	newAttrs.SetNameAttributes(testNameAttrs)
	if !newAttrs.UserF1 || newAttrs.UserF2 || !newAttrs.UserF3 || newAttrs.UserF4 {
		t.Error("SetNameAttributes failed to set correct values")
	}
}

func TestDirectoryEntryAttributes(t *testing.T) {
	entry := &DirectoryEntry{}
	copy(entry.Name[:], "TEST    ")
	copy(entry.Extension[:], "BAS")

	attrs := &FileAttributes{
		ReadOnly: true,
		System:   false,
		Archived: true,
		UserF1:   true,
		UserF2:   false,
		UserF3:   true,
		UserF4:   false,
	}

	// Apply attributes to directory entry
	attrs.ApplyToDirectoryEntry(entry)

	// Verify type field attributes (extension high bits)
	if (entry.Extension[0] & 0x80) == 0 { // ReadOnly
		t.Error("ReadOnly attribute not set in extension")
	}
	if (entry.Extension[1] & 0x80) != 0 { // System
		t.Error("System attribute incorrectly set in extension")
	}
	if (entry.Extension[2] & 0x80) == 0 { // Archived
		t.Error("Archived attribute not set in extension")
	}

	// Verify name field attributes (name high bits)
	if (entry.Name[0] & 0x80) == 0 { // UserF1
		t.Error("UserF1 attribute not set in name")
	}
	if (entry.Name[1] & 0x80) != 0 { // UserF2
		t.Error("UserF2 attribute incorrectly set in name")
	}
	if (entry.Name[2] & 0x80) == 0 { // UserF3
		t.Error("UserF3 attribute not set in name")
	}
	if (entry.Name[3] & 0x80) != 0 { // UserF4
		t.Error("UserF4 attribute incorrectly set in name")
	}

	// Test reading attributes back
	readAttrs := &FileAttributes{}
	readAttrs.ReadFromDirectoryEntry(entry)

	if !readAttrs.ReadOnly || readAttrs.System || !readAttrs.Archived {
		t.Error("ReadFromDirectoryEntry failed to read type attributes correctly")
	}
	if !readAttrs.UserF1 || readAttrs.UserF2 || !readAttrs.UserF3 || readAttrs.UserF4 {
		t.Error("ReadFromDirectoryEntry failed to read name attributes correctly")
	}
}