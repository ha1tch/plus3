// file: pkg/diskimg/allocation.go

package diskimg

import (
	"errors"
	"fmt"
)

// SectorAllocation tracks the allocation status of disk sectors
type SectorAllocation struct {
	allocated []bool  // true if sector is allocated
	sectorMap *internal.SectorMap
}

// newSectorAllocation creates a new sector allocation tracker
func newSectorAllocation(totalSectors int) *SectorAllocation {
	return &SectorAllocation{
		allocated: make([]bool, totalSectors),
		sectorMap: internal.NewSectorMap(),
	}
}

// AllocateSectors marks a range of sectors as allocated
func (sa *SectorAllocation) AllocateSectors(start, count int) error {
	if start < 0 || start+count > len(sa.allocated) {
		return errors.New("sector range out of bounds")
	}

	// Check if any sectors in range are already allocated
	for i := start; i < start+count; i++ {
		if sa.allocated[i] {
			return fmt.Errorf("sector %d already allocated", i)
		}
	}

	// Mark sectors as allocated
	for i := start; i < start+count; i++ {
		sa.allocated[i] = true
	}

	return nil
}

// FreeSectors marks a range of sectors as free
func (sa *SectorAllocation) FreeSectors(start, count int) error {
	if start < 0 || start+count > len(sa.allocated) {
		return errors.New("sector range out of bounds")
	}

	for i := start; i < start+count; i++ {
		sa.allocated[i] = false
	}

	return nil
}

// FindFreeSectors looks for a contiguous range of free sectors
func (sa *SectorAllocation) FindFreeSectors(count int) (int, error) {
	if count <= 0 {
		return 0, errors.New("invalid sector count requested")
	}
	if count > len(sa.allocated) {
		return 0, errors.New("requested sector count exceeds disk size")
	}

	start := 0
	consecutive := 0

	for i := 0; i < len(sa.allocated); i++ {
		if !sa.allocated[i] {
			if consecutive == 0 {
				start = i
			}
			consecutive++
			if consecutive == count {
				return start, nil
			}
		} else {
			consecutive = 0
		}
	}

	return 0, errors.New("not enough contiguous free sectors available")
}

// IsSectorAllocated checks if a specific sector is allocated
func (sa *SectorAllocation) IsSectorAllocated(sector int) (bool, error) {
	if sector < 0 || sector >= len(sa.allocated) {
		return false, errors.New("sector number out of range")
	}
	return sa.allocated[sector], nil
}

// GetFreeSpace returns the number of free sectors
func (sa *SectorAllocation) GetFreeSpace() int {
	free := 0
	for _, allocated := range sa.allocated {
		if !allocated {
			free++
		}
	}
	return free
}

// GetTrackAllocation returns allocation status for all sectors in a track
func (sa *SectorAllocation) GetTrackAllocation(track, side int) ([]bool, error) {
	start, end, err := sa.sectorMap.GetTrackBounds(track, side)
	if err != nil {
		return nil, err
	}

	trackAlloc := make([]bool, end-start+1)
	for i := range trackAlloc {
		trackAlloc[i] = sa.allocated[start+i]
	}

	return trackAlloc, nil
}

// AllocateTrack marks all sectors in a track as allocated
func (sa *SectorAllocation) AllocateTrack(track, side int) error {
	start, end, err := sa.sectorMap.GetTrackBounds(track, side)
	if err != nil {
		return err
	}

	return sa.AllocateSectors(start, end-start+1)
}

// FreeTrack marks all sectors in a track as free
func (sa *SectorAllocation) FreeTrack(track, side int) error {
	start, end, err := sa.sectorMap.GetTrackBounds(track, side)
	if err != nil {
		return err
	}

	return sa.FreeSectors(start, end-start+1)
}

// ResetAllocation clears all allocation information
func (sa *SectorAllocation) ResetAllocation() {
	for i := range sa.allocated {
		sa.allocated[i] = false
	}
}