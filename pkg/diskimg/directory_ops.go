// file: pkg/diskimg/directory_ops.go

package diskimg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Constants for +3DOS directory handling
const (
	DirectoryTrack         = 1  // Directory track (XDPB OFF=1: track 0 is the reserved system track)
	DirectoryStartSector   = 0  // First data sector index of the directory within the track
	DirectorySizeInSectors = 4  // Directory occupies 4 sectors
	DirectoryEntrySize     = 32 // Size of a single directory entry in bytes
	MaxDirectoryEntries    = 64 // +3 standard format: 2K dir / 32 bytes = 64 entries
)

// readDirectory reads all directory sectors from the disk
func (di *DiskImage) readDirectory() ([]byte, error) {
	// Allocate buffer for directory data
	dirData := make([]byte, DirectorySizeInSectors*BytesPerSector)

	// Read each sector
	for sector := 0; sector < DirectorySizeInSectors; sector++ {
		sectorData, err := di.GetSectorData(DirectoryTrack, sector+DirectoryStartSector, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory sector %d: %w", sector, err)
		}

		// Copy sector data into buffer
		offset := sector * BytesPerSector
		copy(dirData[offset:], sectorData)
	}

	return dirData, nil
}

// writeDirectory writes directory data back to disk
func (di *DiskImage) writeDirectory(dirData []byte) error {
	if len(dirData) > DirectorySizeInSectors*BytesPerSector {
		return errors.New("directory data exceeds maximum size")
	}

	// Write each sector
	for sector := 0; sector < DirectorySizeInSectors; sector++ {
		offset := sector * BytesPerSector
		sectorData := dirData[offset : offset+BytesPerSector]

		err := di.SetSectorData(DirectoryTrack, sector+DirectoryStartSector, 0, sectorData)
		if err != nil {
			return fmt.Errorf("failed to write directory sector %d: %w", sector, err)
		}
	}

	return nil
}

// InitializeDirectory creates an empty directory on the disk
func (di *DiskImage) InitializeDirectory() error {
	// Create empty directory data
	dirData := make([]byte, DirectorySizeInSectors*BytesPerSector)
	for i := range dirData {
		dirData[i] = 0xE5 // Mark all entries as deleted
	}

	// Write the empty directory
	err := di.writeDirectory(dirData)
	if err != nil {
		return fmt.Errorf("failed to initialize directory: %w", err)
	}

	return nil
}

// GetDirectory returns the directory data as a slice of entries
func (di *DiskImage) GetDirectory() ([]DirectoryEntry, error) {
	dirData, err := di.readDirectory()
	if err != nil {
		return nil, err
	}

	// Parse directory entries
	entries := make([]DirectoryEntry, MaxDirectoryEntries)
	for i := 0; i < MaxDirectoryEntries; i++ {
		offset := i * DirectoryEntrySize
		if dirData[offset] == 0xE5 {
			// Unused/deleted entry - preserve the 0xE5 marker so callers can
			// identify free slots (IsUnused) and reuse them.
			entries[i].Status = 0xE5
			continue
		}

		entryData := dirData[offset : offset+DirectoryEntrySize]
		entry := DirectoryEntry{}
		err := binary.Read(bytes.NewReader(entryData), binary.LittleEndian, &entry)
		if err != nil {
			return nil, fmt.Errorf("failed to parse directory entry %d: %w", i, err)
		}
		entries[i] = entry
	}

	return entries, nil
}

// FlushDirectory serializes the in-memory directory and writes it to the
// directory sectors (track 1). Empty entries are stored with the 0xE5 marker.
func (di *DiskImage) FlushDirectory() error {
	dirData, err := di.directory.Save()
	if err != nil {
		return err
	}
	// Pad/trim to the directory area size and ensure empty entries are 0xE5.
	want := DirectorySizeInSectors * BytesPerSector
	if len(dirData) < want {
		pad := make([]byte, want-len(dirData))
		for i := range pad {
			pad[i] = 0xE5
		}
		dirData = append(dirData, pad...)
	} else if len(dirData) > want {
		dirData = dirData[:want]
	}
	return di.writeDirectory(dirData)
}

// DeleteFile removes a file from the disk: it frees the file's allocation blocks,
// marks its directory entry unused (0xE5), and flushes the directory to disk.
func (di *DiskImage) DeleteFile(filename string) error {
	idx := -1
	for i := range di.directory.Entries {
		e := &di.directory.Entries[i]
		if e.IsUnused() {
			continue
		}
		if strings.EqualFold(e.GetFilename(), strings.ToUpper(strings.TrimSpace(filename))) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("file not found: %s", filename)
	}

	// Free the allocation blocks listed in the entry.
	entry := &di.directory.Entries[idx]
	var blocks []int
	for _, b := range entry.AllocationBlocks {
		if b != 0 {
			blocks = append(blocks, int(b))
		}
	}
	if di.fileAlloc != nil && len(blocks) > 0 {
		_ = di.fileAlloc.FreeBlocks(blocks)
	}

	// Mark the entry unused.
	di.directory.Entries[idx] = DirectoryEntry{Status: 0xE5}

	di.Modified = true
	return di.FlushDirectory()
}
