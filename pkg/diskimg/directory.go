package diskimg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// DirectoryEntry represents a single directory entry in +3DOS format
type DirectoryEntry struct {
	Status           byte     // St: 0xE5 = unused/deleted, 0-15 = user number
	Name             [8]byte  // F0-F7: file name (padded with spaces)
	Extension        [3]byte  // E0-E2: extension (padded; high bits are attributes)
	Extent           byte     // Xl: extent number low byte
	Reserved1        byte     // Bc: byte count (last record byte count)
	Reserved2        byte     // Xh: extent number high byte
	RecordCount      byte     // Rc: number of 128-byte records in this extent
	AllocationBlocks [16]byte // Al: block numbers used by this extent
}

// This struct is exactly 32 bytes, matching the CP/M directory entry layout.

// SetAttributes sets file attributes using constants from fileattr.go
func (de *DirectoryEntry) SetAttributes(readOnly, hidden, system bool) {
	var attr byte
	if readOnly {
		attr |= AttrReadOnly
	}
	if hidden {
		attr |= AttrHidden
	}
	if system {
		attr |= AttrSystem
	}
	de.Reserved1 = (de.Reserved1 & 0xD8) | attr
}

// GetAttributes retrieves file attributes using constants from fileattr.go
func (de *DirectoryEntry) GetAttributes() (readOnly, hidden, system bool) {
	return (de.Reserved1 & AttrReadOnly) != 0,
		(de.Reserved1 & AttrHidden) != 0,
		(de.Reserved1 & AttrSystem) != 0
}

// Load reads directory entries from raw disk data
func (d *Directory) Load(data []byte) error {
	if len(data)%32 != 0 {
		return errors.New("invalid directory size: must be a multiple of 32 bytes")
	}
	numEntries := len(data) / 32
	d.Entries = make([]DirectoryEntry, numEntries)
	for i := 0; i < numEntries; i++ {
		offset := i * 32
		entryData := data[offset : offset+32]
		if err := binary.Read(bytes.NewReader(entryData), binary.LittleEndian, &d.Entries[i]); err != nil {
			return fmt.Errorf("failed to read directory entry %d: %w", i, err)
		}
	}
	return nil
}

// Save writes the directory entries to raw disk data
func (d *Directory) Save() ([]byte, error) {
	var buffer bytes.Buffer
	for i, entry := range d.Entries {
		// An empty slot (status 0x00 with no name, or already 0xE5) is written as a
		// full 0xE5 entry, the CP/M unused-entry marker.
		if entry.isFree() {
			filler := make([]byte, DirectoryEntrySize)
			for j := range filler {
				filler[j] = 0xE5
			}
			buffer.Write(filler)
			continue
		}
		if err := binary.Write(&buffer, binary.LittleEndian, entry); err != nil {
			return nil, fmt.Errorf("failed to write directory entry %d: %w", i, err)
		}
	}
	return buffer.Bytes(), nil
}

// FindEntryByName searches for a directory entry by its name
func (d *Directory) FindEntryByName(name string) (*DirectoryEntry, error) {
	name = strings.ToUpper(name)
	for i := range d.Entries {
		entryName := strings.TrimSpace(string(d.Entries[i].Name[:]))
		if entryName == name {
			return &d.Entries[i], nil
		}
	}
	return nil, errors.New("file not found")
}

// DeleteEntry marks a directory entry as deleted
func (d *Directory) DeleteEntry(name string) error {
	entry, err := d.FindEntryByName(name)
	if err != nil {
		return err
	}
	entry.Status = 0xE5 // Mark as deleted
	return nil
}

// Directory is a wrapper for managing directory entries
type Directory struct {
	Entries []DirectoryEntry
}

// FindFile searches for a file by name in the directory
func (d *Directory) FindFile(filename string) (*DirectoryEntry, error) {
	target := strings.ToUpper(strings.TrimSpace(filename))
	for i := range d.Entries {
		if d.Entries[i].IsUnused() {
			continue
		}
		if strings.EqualFold(d.Entries[i].GetFilename(), target) {
			return &d.Entries[i], nil
		}
	}
	return nil, fmt.Errorf("file %s not found", filename)
}

// AddFile adds a new file entry to the directory
func (d *Directory) AddFile(entry DirectoryEntry) error {
	for i := range d.Entries {
		if d.Entries[i].isFree() {
			d.Entries[i] = entry
			d.Entries[i].Status = 0x00 // user 0 (default user area)
			return nil
		}
	}
	return fmt.Errorf("no free directory entry slots available")
}

// IsUnused reports whether this directory entry is empty (CP/M marks empty and
// deleted entries alike with status 0xE5).
func (de *DirectoryEntry) IsUnused() bool {
	return de.Status == 0xE5
}

// IsDeleted reports whether this entry has been deleted. In CP/M a deleted entry
// uses the same 0xE5 marker as an unused one, so the two are indistinguishable.
func (de *DirectoryEntry) IsDeleted() bool {
	return de.Status == 0xE5
}

// GetFilename returns the file name as "NAME.EXT", trimmed of padding spaces and
// with the high (attribute) bits of each character stripped.
func (de *DirectoryEntry) GetFilename() string {
	name := make([]byte, 0, 12)
	for _, b := range de.Name {
		b &= 0x7F // strip attribute bit
		if b == ' ' || b == 0 {
			continue
		}
		name = append(name, b)
	}
	ext := make([]byte, 0, 3)
	for _, b := range de.Extension {
		b &= 0x7F
		if b == ' ' || b == 0 {
			continue
		}
		ext = append(ext, b)
	}
	if len(ext) == 0 {
		return string(name)
	}
	return string(name) + "." + string(ext)
}

// isFree reports whether this entry is an empty/reusable slot: either the CP/M
// unused marker (0xE5) or an uninitialised zero entry with no name. A real file
// in user area 0 has status 0x00 but a non-blank name and is NOT free.
func (de *DirectoryEntry) isFree() bool {
	if de.Status == 0xE5 {
		return true
	}
	if de.Status == 0x00 {
		blank := true
		for _, b := range de.Name {
			if b != 0 && b != ' ' {
				blank = false
				break
			}
		}
		return blank
	}
	return false
}
