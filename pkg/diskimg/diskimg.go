// file: pkg/diskimg/diskimg.go

package diskimg

import (
	"errors"
)

const (
	TracksPerSide   = 40
	SectorsPerTrack = 9
	BytesPerSector  = 512
	DiskSizeInBytes = 184320 // ZX Spectrum +3 single-sided disk
	MaxDirectoryEntries = 112
	BlockSize = 1024
	SectorsPerBlock = BlockSize / BytesPerSector
)

// DiskImage represents a ZX Spectrum +3 disk image
type DiskImage struct {
	directory       Directory
	diskSizeInBytes int
	allocation      AllocationMap
}

// NewDiskImage initializes a new disk image
func NewDiskImage() *DiskImage {
	totalSectors := DiskSizeInBytes / BytesPerSector
	return &DiskImage{
		directory:       Directory{Entries: make([]DirectoryEntry, MaxDirectoryEntries)},
		diskSizeInBytes: DiskSizeInBytes,
		allocation:      NewAllocationMap(totalSectors),
	}
}

// GetSectorData retrieves data from a specific track and sector
func (di *DiskImage) GetSectorData(track, sector, side int) ([]byte, error) {
	if track < 0 || track >= TracksPerSide || sector < 0 || sector >= SectorsPerTrack {
		return nil, errors.New("invalid track or sector")
	}

	// Placeholder logic for retrieving sector data; needs proper implementation
	return make([]byte, BytesPerSector), nil
}

// SetSectorData writes data to a specific track and sector
func (di *DiskImage) SetSectorData(track, sector, side int, data []byte) error {
	if track < 0 || track >= TracksPerSide || sector < 0 || sector >= SectorsPerTrack {
		return errors.New("invalid track or sector")
	}
	if len(data) != BytesPerSector {
		return errors.New("data length does not match sector size")
	}

	// Placeholder logic for setting sector data; needs proper implementation
	return nil
}

// AllocateFileSpace allocates space for a file on the disk
func (di *DiskImage) AllocateFileSpace(size int) ([]int, error) {
	return di.allocation.AllocateFileSpace(size)
}

// AllocationMap tracks sector usage on the disk
type AllocationMap struct {
	sectors []bool
}

// NewAllocationMap creates a new allocation map
func NewAllocationMap(totalSectors int) AllocationMap {
	return AllocationMap{
		sectors: make([]bool, totalSectors),
	}
}

// AllocateFileSpace allocates space for a file
func (am *AllocationMap) AllocateFileSpace(size int) ([]int, error) {
	neededSectors := (size + BytesPerSector - 1) / BytesPerSector
	allocated := []int{}

	for i := 0; i < len(am.sectors) && neededSectors > 0; i++ {
		if !am.sectors[i] {
			am.sectors[i] = true
			allocated = append(allocated, i)
			neededSectors--
		}
	}

	if neededSectors > 0 {
		return nil, errors.New("not enough space on disk")
	}
	return allocated, nil
}

// FreeSectors releases allocated sectors
func (am *AllocationMap) FreeSectors(sectors []int) {
	for _, sector := range sectors {
		if sector >= 0 && sector < len(am.sectors) {
			am.sectors[sector] = false
		}
	}
}