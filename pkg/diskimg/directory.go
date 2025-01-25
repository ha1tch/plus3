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
	Status           byte     // Status byte (0xe5 = deleted, 0x00 = unused)
	Name             [8]byte  // File name (padded with spaces)
	Extension        [3]byte  // File extension (padded with spaces)
	Extent           byte     // Extent number (for file parts)
	Reserved1        byte     // Reserved for attributes
	Reserved2        byte     // Reserved
	RecordCount      byte     // Number of 128-byte records in this extent
	AllocationBlocks [16]byte // Block numbers used by file
	FirstRecord      byte     // First record number
	LogicalSize      uint16   // Logical size in 128-byte records
}

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

// AddEntry adds a new entry to the directory
func (d *Directory) AddEntry(entry DirectoryEntry) error {
	for i := range d.Entries {
		if d.Entries[i].Status == 0xE5 || d.Entries[i].Status == 0x00 {
			d.Entries[i] = entry
			d.Entries[i].Status = 0x01 // Mark as active
			return nil
		}
	}
	return errors.New("no free directory entry slots available")
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
	for i := range d.Entries {
		entryName := strings.TrimSpace(string(d.Entries[i].Name[:]))
		if entryName == filename && d.Entries[i].Status != 0xE5 && d.Entries[i].Status != 0x00 {
			return &d.Entries[i], nil
		}
	}
	return nil, fmt.Errorf("file %s not found", filename)
}

// AddFile adds a new file entry to the directory
func (d *Directory) AddFile(entry DirectoryEntry) error {
	for i := range d.Entries {
		if d.Entries[i].Status == 0xE5 || d.Entries[i].Status == 0x00 {
			d.Entries[i] = entry
			d.Entries[i].Status = 0x01 // Mark as active
			return nil
		}
	}
	return fmt.Errorf("no free directory entry slots available")
}
