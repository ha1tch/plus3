// file: pkg/diskimg/diskcheck.go

package diskimg

import (
	"bytes"
	"errors"
	"fmt"
)

// ValidationLevel determines how strict the validation should be
type ValidationLevel int

const (
	// ValidationBasic checks only essential disk structure
	ValidationBasic ValidationLevel = iota
	// ValidationStrict enforces all +3DOS format rules
	ValidationStrict
)

// DiskCheck performs comprehensive disk validation
type DiskCheck struct {
	disk    *DiskImage
	level   ValidationLevel
	errors  []error
}

// NewDiskCheck creates a new disk validator
func NewDiskCheck(disk *DiskImage, level ValidationLevel) *DiskCheck {
	return &DiskCheck{
		disk:    disk,
		level:   level,
		errors:  make([]error, 0),
	}
}

// Validate performs all validation checks
func (dc *DiskCheck) Validate() []error {
	// Basic structure checks
	dc.checkDiskHeader()
	dc.checkTrackCount()
	dc.checkSectorSizes()
	
	// Directory validation
	dc.checkDirectoryStructure()
	
	// If strict validation requested
	if dc.level == ValidationStrict {
		dc.checkBootSector()
		dc.checkUnusedSectors()
		dc.checkFileConsistency()
		dc.checkDirectoryEntries()
	}
	
	return dc.errors
}

func (dc *DiskCheck) addError(err error) {
	dc.errors = append(dc.errors, err)
}

// Basic structure validation

func (dc *DiskCheck) checkDiskHeader() {
	// Check disk signature
	expectedSig := []byte("EXTENDED CPC DSK File\r\nDisk-Info\r\n")
	if !bytes.Equal(dc.disk.Header.Signature[:len(expectedSig)], expectedSig) {
		dc.addError(errors.New("invalid disk image signature"))
	}

	// Check +3 format parameters
	if dc.disk.Header.TracksNum != TracksPerSide {
		dc.addError(fmt.Errorf("invalid track count: expected %d, got %d", 
			TracksPerSide, dc.disk.Header.TracksNum))
	}

	if dc.disk.Header.SidesNum != SidesPerDisk {
		dc.addError(fmt.Errorf("invalid side count: expected %d, got %d", 
			SidesPerDisk, dc.disk.Header.SidesNum))
	}

	expectedTrackSize := BytesPerSector * SectorsPerTrack
	if int(dc.disk.Header.TrackSize) != expectedTrackSize {
		dc.addError(fmt.Errorf("invalid track size: expected %d, got %d", 
			expectedTrackSize, dc.disk.Header.TrackSize))
	}
}

func (dc *DiskCheck) checkTrackCount() {
	expectedTracks := int(dc.disk.Header.TracksNum * dc.disk.Header.SidesNum)
	if len(dc.disk.Tracks) != expectedTracks {
		dc.addError(fmt.Errorf("track data mismatch: expected %d tracks, found %d", 
			expectedTracks, len(dc.disk.Tracks)))
	}

	// Check each track's size
	for i, track := range dc.disk.Tracks {
		if len(track) != int(dc.disk.Header.TrackSize) {
			dc.addError(fmt.Errorf("track %d size mismatch: expected %d bytes, found %d", 
				i, dc.disk.Header.TrackSize, len(track)))
		}
	}
}

func (dc *DiskCheck) checkSectorSizes() {
	for t := 0; t < int(dc.disk.Header.TracksNum); t++ {
		for s := 0; s < SectorsPerTrack; s++ {
			data, err := dc.disk.GetSectorData(t, s, 0)
			if err != nil {
				dc.addError(fmt.Errorf("failed to read sector %d on track %d: %v", s, t, err))
				continue
			}
			if len(data) != BytesPerSector {
				dc.addError(fmt.Errorf("invalid sector size at track %d, sector %d: got %d bytes", 
					t, s, len(data)))
			}
		}
	}
}

// Directory validation

func (dc *DiskCheck) checkDirectoryStructure() {
	// Check directory location (should be at track 0, after boot sector)
	dirSector, err := dc.disk.GetSectorData(0, DirectorySector, 0)
	if err != nil {
		dc.addError(errors.New("cannot read directory sector"))
		return
	}

	// Check directory size
	dirSize := MaxDirectoryEntries * DirectoryEntrySize
	sectorsNeeded := (dirSize + BytesPerSector - 1) / BytesPerSector
	
	for s := 0; s < sectorsNeeded; s++ {
		_, err := dc.disk.GetSectorData(0, DirectorySector+s, 0)
		if err != nil {
			dc.addError(fmt.Errorf("directory sector %d not readable", DirectorySector+s))
		}
	}
}

// Strict validation checks

func (dc *DiskCheck) checkBootSector() {
	bootSector, err := dc.disk.GetSectorData(0, 0, 0)
	if err != nil {
		dc.addError(errors.New("cannot read boot sector"))
		return
	}

	// Calculate checksum (byte 15 should make sum of all bytes = 3 mod 256)
	var sum byte
	for i, b := range bootSector[:15] {
		sum += b
	}
	for _, b := range bootSector[16:] {
		sum += b
	}

	if (sum+bootSector[15])%256 != 3 {
		dc.addError(errors.New("invalid boot sector checksum"))
	}
}

func (dc *DiskCheck) checkUnusedSectors() {
	// Check if unallocated sectors contain format filler (0xE5)
	for t := 0; t < int(dc.disk.Header.TracksNum); t++ {
		for s := 0; s < SectorsPerTrack; s++ {
			linear, err := dc.disk.sectorMap.PhysicalToLinear(t, s, 0)
			if err != nil {
				continue
			}

			allocated, err := dc.disk.allocation.IsSectorAllocated(linear)
			if err != nil {
				continue
			}

			if !allocated {
				data, err := dc.disk.GetSectorData(t, s, 0)
				if err != nil {
					continue
				}

				// Check if unallocated sector contains anything other than filler
				for _, b := range data {
					if b != 0xE5 {
						dc.addError(fmt.Errorf("unallocated sector contains non-filler data at track %d, sector %d", t, s))
						break
					}
				}
			}
		}
	}
}

func (dc *DiskCheck) checkFileConsistency() {
	dir, err := dc.disk.GetDirectory()
	if err != nil {
		dc.addError(errors.New("cannot read directory for file consistency check"))
		return
	}

	// Check each file entry
	for i, entry := range dir.Entries {
		if !entry.IsDeleted() && !entry.IsUnused() {
			// Validate allocation blocks
			for _, block := range entry.AllocationBlocks {
				if block != 0 {
					if int(block) >= len(dc.disk.allocation.allocated) {
						dc.addError(fmt.Errorf("file entry %d references invalid allocation block %d", i, block))
					}
				}
			}

			// Check file size consistency
			expectedSize := int(entry.LogicalSize) * 128 // LogicalSize is in 128-byte records
			if expectedSize > BytesPerSector*len(entry.AllocationBlocks) {
				dc.addError(fmt.Errorf("file entry %d size inconsistency: claims %d bytes but has %d bytes allocated",
					i, expectedSize, BytesPerSector*len(entry.AllocationBlocks)))
			}
		}
	}
}

func (dc *DiskCheck) checkDirectoryEntries() {
	dir, err := dc.disk.GetDirectory()
	if err != nil {
		dc.addError(errors.New("cannot read directory for entry validation"))
		return
	}

	// Check for duplicate filenames
	seen := make(map[string]bool)
	for i, entry := range dir.Entries {
		if !entry.IsDeleted() && !entry.IsUnused() {
			name := entry.GetFilename()
			if seen[name] {
				dc.addError(fmt.Errorf("duplicate filename found: %s at entry %d", name, i))
			}
			seen[name] = true
		}
	}

	// Validate individual entries
	for i, entry := range dir.Entries {
		if !entry.IsDeleted() && !entry.IsUnused() {
			if err := dc.validateDirectoryEntry(&entry, i); err != nil {
				dc.addError(err)
			}
		}
	}
}

func (dc *DiskCheck) validateDirectoryEntry(entry *DirectoryEntry, index int) error {
	// Check filename characters (valid CP/M characters)
	for _, c := range entry.Name {
		if c != 0x20 && (c < 0x21 || c > 0x7E) {
			return fmt.Errorf("invalid character in filename at entry %d", index)
		}
	}

	// Check extension characters
	for _, c := range entry.Extension {
		if c != 0x20 && (c < 0x21 || c > 0x7E) {
			return fmt.Errorf("invalid character in extension at entry %d", index)
		}
	}

	// Basic sanity checks
	if entry.Extent > 31 {
		return fmt.Errorf("invalid extent number at entry %d", index)
	}

	if entry.RecordCount > 128 {
		return fmt.Errorf("invalid record count at entry %d", index)
	}

	return nil
}