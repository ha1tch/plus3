// file: pkg/diskimg/directory.go

package diskimg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	// Directory entry size and offsets according to +3DOS spec
	DirectoryEntrySize = 32
	MaxDirectoryEntries = 64  // Maximum number of entries for +3DOS
	
	// File attributes
	AttrReadOnly  = 0x01
	AttrHidden    = 0x02
	AttrSystem    = 0x04
	AttrArchived  = 0x20
	
	// Special values
	DirectoryTrack = 0  // Directory starts at track 0
	DirectorySector = 1 // After boot sector
)

// DirectoryEntry represents a single directory entry in +3DOS format
type DirectoryEntry struct {
	Status     byte      // Status byte (0xe5 = deleted, 0x00 = unused)
	Name       [8]byte   // File name (padded with spaces)
	Extension  [3]byte   // File extension (padded with spaces)
	Extent     byte      // Extent number (for file parts)
	Reserved1  byte      // Reserved
	Reserved2  byte      // Reserved
	RecordCount byte    // Number of 128-byte records in this extent
	AllocationBlocks [16]byte // Block numbers used by file
	FirstRecord byte     // First record number
	LogicalSize uint16   // Logical size in 128-byte records
}

// NewDirectoryEntry creates a new directory entry with the given filename
func NewDirectoryEntry(filename string) (*DirectoryEntry, error) {
	if len(filename) == 0 {
		return nil, errors.New("empty filename")
	}

	entry := &DirectoryEntry{
		Status: 0x00,
	}

	// Split filename into name and extension
	parts := strings.Split(filename, ".")
	name := parts[0]
	ext := ""
	if len(parts) > 1 {
		ext = parts[1]
	}

	// Validate and copy name (max 8 chars)
	if len(name) > 8 {
		return nil, errors.New("filename too long (max 8 chars)")
	}
	copy(entry.Name[:], padRight(strings.ToUpper(name), 8))

	// Validate and copy extension (max 3 chars)
	if len(ext) > 3 {
		return nil, errors.New("extension too long (max 3 chars)")
	}
	if ext != "" {
		copy(entry.Extension[:], padRight(strings.ToUpper(ext), 3))
	} else {
		copy(entry.Extension[:], padRight("", 3))
	}

	return entry, nil
}

// IsDeleted returns true if the entry is marked as deleted
func (de *DirectoryEntry) IsDeleted() bool {
	return de.Status == 0xe5
}

// IsUnused returns true if the entry is unused
func (de *DirectoryEntry) IsUnused() bool {
	return de.Status == 0x00
}

// GetFilename returns the full filename including extension
func (de *DirectoryEntry) GetFilename() string {
	name := strings.TrimRight(string(de.Name[:]), " ")
	ext := strings.TrimRight(string(de.Extension[:]), " ")
	if ext != "" {
		return fmt.Sprintf("%s.%s", name, ext)
	}
	return name
}

// SetAttributes sets the file attributes
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
	// Don't modify other attribute bits
	de.Reserved1 = (de.Reserved1 & 0xD8) | attr
}

// GetAttributes returns the file attributes
func (de *DirectoryEntry) GetAttributes() (readOnly, hidden, system bool) {
	return (de.Reserved1 & AttrReadOnly) != 0,
		(de.Reserved1 & AttrHidden) != 0,
		(de.Reserved1 & AttrSystem) != 0
}

// Directory represents the +3DOS directory structure
type Directory struct {
	Entries    [MaxDirectoryEntries]DirectoryEntry
	disk       *DiskImage
	modified   bool
}

// readDirectory reads the directory from the disk image
func (di *DiskImage) readDirectory() (*Directory, error) {
	dir := &Directory{
		disk: di,
	}

	// Read directory sectors
	dirData, err := di.readDirectorySectors()
	if err != nil {
		return nil, err
	}

	// Parse directory entries
	for i := 0; i < MaxDirectoryEntries; i++ {
		offset := i * DirectoryEntrySize
		if offset+DirectoryEntrySize > len(dirData) {
			return nil, errors.New("directory data truncated")
		}

		entryData := dirData[offset : offset+DirectoryEntrySize]
		entry := &dir.Entries[i]
		
		// Use binary.Read with a buffer to parse the entry
		buf := bytes.NewReader(entryData)
		if err := binary.Read(buf, binary.LittleEndian, entry); err != nil {
			return nil, fmt.Errorf("error parsing directory entry %d: %v", i, err)
		}
	}

	return dir, nil
}

// writeDirectory writes the directory back to the disk image
func (dir *Directory) write() error {
	if !dir.modified {
		return nil
	}

	// Create buffer for directory data
	dirData := make([]byte, MaxDirectoryEntries*DirectoryEntrySize)

	// Write each entry to the buffer
	for i, entry := range dir.Entries {
		offset := i * DirectoryEntrySize
		buf := bytes.NewBuffer(nil)
		if err := binary.Write(buf, binary.LittleEndian, entry); err != nil {
			return fmt.Errorf("error encoding directory entry %d: %v", i, err)
		}
		copy(dirData[offset:], buf.Bytes())
	}

	// Write to disk sectors
	if err := dir.disk.writeDirectorySectors(dirData); err != nil {
		return err
	}

	dir.modified = false
	return nil
}

// FindFile looks for a file in the directory
func (dir *Directory) FindFile(filename string) (*DirectoryEntry, int, error) {
	filename = strings.ToUpper(filename)
	for i, entry := range dir.Entries {
		if !entry.IsDeleted() && !entry.IsUnused() {
			if strings.ToUpper(entry.GetFilename()) == filename {
				return &dir.Entries[i], i, nil
			}
		}
	}
	return nil, -1, errors.New("file not found")
}

// AddFile adds a new file entry to the directory
func (dir *Directory) AddFile(filename string) (*DirectoryEntry, error) {
	// First check if file already exists
	if existing, _, err := dir.FindFile(filename); err == nil {
		return nil, fmt.Errorf("file %s already exists", existing.GetFilename())
	}

	// Find first free entry
	entryIndex := -1
	for i := range dir.Entries {
		if dir.Entries[i].IsUnused() || dir.Entries[i].IsDeleted() {
			entryIndex = i
			break
		}
	}

	if entryIndex == -1 {
		return nil, errors.New("directory full")
	}

	// Create new entry
	entry, err := NewDirectoryEntry(filename)
	if err != nil {
		return nil, err
	}

	dir.Entries[entryIndex] = *entry
	dir.modified = true

	return &dir.Entries[entryIndex], nil
}

// DeleteFile marks a file as deleted in the directory
func (dir *Directory) DeleteFile(filename string) error {
	entry, index, err := dir.FindFile(filename)
	if err != nil {
		return err
	}

	entry.Status = 0xe5 // Mark as deleted
	dir.modified = true

	return nil
}

// Helper functions

// padRight pads a string with spaces to the specified length
func padRight(str string, length int) string {
	if len(str) >= length {
		return str[:length]
	}
	return str + strings.Repeat(" ", length-len(str))
}