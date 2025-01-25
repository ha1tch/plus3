// file: pkg/diskimg/directory_ops.go

package diskimg

import (
	"errors"
)

// readDirectorySectors reads the directory sectors from the disk image
func (di *DiskImage) readDirectorySectors() ([]byte, error) {
	// Calculate how many sectors we need for the directory
	directorySectors := (MaxDirectoryEntries * DirectoryEntrySize + BytesPerSector - 1) / BytesPerSector

	// Allocate buffer for directory data
	dirData := make([]byte, directorySectors*BytesPerSector)

	// Read each sector
	for sector := 0; sector < directorySectors; sector++ {
		sectorData, err := di.GetSectorData(DirectoryTrack, sector+DirectorySector, 0)
		if err != nil {
			return nil, errors.New("failed to read directory sector")
		}
		
		// Copy sector data into buffer
		offset := sector * BytesPerSector
		copy(dirData[offset:], sectorData)
	}

	return dirData, nil
}

// writeDirectorySectors writes the directory sectors to the disk image
func (di *DiskImage) writeDirectorySectors(dirData []byte) error {
	// Calculate number of sectors needed
	directorySectors := (len(dirData) + BytesPerSector - 1) / BytesPerSector

	// Validate directory size
	maxDirectorySectors := (MaxDirectoryEntries * DirectoryEntrySize + BytesPerSector - 1) / BytesPerSector
	if directorySectors > maxDirectorySectors {
		return errors.New("directory data exceeds maximum size")
	}

	// Write each sector
	for sector := 0; sector < directorySectors; sector++ {
		// Get data for this sector
		offset := sector * BytesPerSector
		sectorData := make([]byte, BytesPerSector)
		
		// Copy data, ensuring we don't read past the end of dirData
		remaining := len(dirData) - offset
		if remaining > BytesPerSector {
			remaining = BytesPerSector
		}
		copy(sectorData, dirData[offset:offset+remaining])

		// Write sector
		err := di.SetSectorData(DirectoryTrack, sector+DirectorySector, 0, sectorData)
		if err != nil {
			return errors.New("failed to write directory sector")
		}
	}

	return nil
}

// InitializeDirectory initializes an empty directory on the disk
func (di *DiskImage) InitializeDirectory() error {
	// Calculate number of sectors needed for directory
	directorySectors := (MaxDirectoryEntries * DirectoryEntrySize + BytesPerSector - 1) / BytesPerSector

	// Create empty sector data
	emptyData := make([]byte, BytesPerSector)
	for i := range emptyData {
		emptyData[i] = 0xE5 // Fill with deleted entry markers
	}

	// Write empty sectors
	for sector := 0; sector < directorySectors; sector++ {
		err := di.SetSectorData(DirectoryTrack, sector+DirectorySector, 0, emptyData)
		if err != nil {
			return fmt.Errorf("failed to initialize directory sector %d: %v", sector, err)
		}
	}

	// Create new directory structure
	dir := &Directory{
		disk: di,
	}

	// Initialize all entries as unused
	for i := range dir.Entries {
		dir.Entries[i].Status = 0xE5 // Marked as deleted/unused
	}

	di.directory = dir
	return nil
}

// GetDirectory returns the disk's directory, reading it if necessary
func (di *DiskImage) GetDirectory() (*Directory, error) {
	if di.directory == nil {
		dir, err := di.readDirectory()
		if err != nil {
			return nil, err
		}
		di.directory = dir
	}
	return di.directory, nil
}

// FlushDirectory ensures all directory changes are written to disk
func (di *DiskImage) FlushDirectory() error {
	if di.directory != nil && di.directory.modified {
		return di.directory.write()
	}
	return nil
}