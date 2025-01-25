// file: pkg/diskimg/directory_ops.go

package diskimg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Constants for +3DOS directory handling
const (
	DirectoryTrack         = 0   // Fixed track for directory
	DirectoryStartSector   = 0   // Starting sector of the directory
	DirectorySizeInSectors = 4   // Directory occupies 4 sectors
	DirectoryEntrySize     = 32  // Size of a single directory entry in bytes
	MaxDirectoryEntries    = 128 // Maximum entries in the directory
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
			// Skip unused/deleted entries
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

// FlushDirectory is not required for +3DOS as changes are written immediately
// Stub this function for compatibility
func (di *DiskImage) FlushDirectory() error {
	return nil
}
