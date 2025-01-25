package diskimg

import (
	"errors"
	"fmt"
)

// DiskCheck performs a consistency check for a +3DOS disk image.
func (di *DiskImage) DiskCheck() error {
	if err := di.checkBootSector(); err != nil {
		return fmt.Errorf("boot sector check failed: %w", err)
	}
	if err := di.checkDirectoryEntries(); err != nil {
		return fmt.Errorf("directory entries check failed: %w", err)
	}
	if err := di.checkSectorAllocation(); err != nil {
		return fmt.Errorf("sector allocation check failed: %w", err)
	}
	return nil
}

// checkBootSector validates the boot sector.
func (di *DiskImage) checkBootSector() error {
	bootSector, err := di.GetSectorData(0, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to read boot sector: %w", err)
	}

	sum := 0
	for i, b := range bootSector {
		if i == 15 {
			continue
		}
		sum += int(b)
	}
	if (sum+int(bootSector[15]))%256 != 3 {
		return errors.New("boot sector checksum validation failed")
	}
	return nil
}

// checkDirectoryEntries validates the directory structure.
func (di *DiskImage) checkDirectoryEntries() error {
	dirData, err := di.readDirectory()
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for i := 0; i < MaxDirectoryEntries; i++ {
		offset := i * DirectoryEntrySize
		entryData := dirData[offset : offset+DirectoryEntrySize]
		if entryData[0] == 0xE5 || entryData[0] == 0x00 {
			continue
		}

		entry := DirectoryEntry{}
		copy(entry.Name[:], entryData[:8])
		copy(entry.Extension[:], entryData[8:11])
		entry.Status = entryData[0]
		entry.RecordCount = entryData[12]
		// Add other fields based on +3DOS directory entry structure...

		if !isValidFilename(entry.Name[:], entry.Extension[:]) {
			return fmt.Errorf("invalid filename: %s.%s", entry.Name, entry.Extension)
		}
	}
	return nil
}

// checkSectorAllocation ensures no overlapping or invalid sectors.
func (di *DiskImage) checkSectorAllocation() error {
	bitmap := make([]bool, di.TotalSectors())

	for _, entry := range di.directory {
		if entry.Status == 0xE5 || entry.Status == 0x00 {
			continue
		}
		for _, block := range entry.AllocationBlocks {
			if block == 0x00 {
				break
			}
			sector := int(block)
			if sector >= len(bitmap) {
				return fmt.Errorf("invalid sector: %d", sector)
			}
			if bitmap[sector] {
				return fmt.Errorf("sector %d allocated multiple times", sector)
			}
			bitmap[sector] = true
		}
	}
	return nil
}

// isValidFilename validates filenames.
func isValidFilename(name []byte, ext []byte) bool {
	return len(name) <= 8 && len(ext) <= 3
}
