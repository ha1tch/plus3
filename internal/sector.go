// file: internal/sector.go

package internal

import (
	"errors"
	"fmt"
)

// SectorMap handles track/sector mapping for +3DOS disk format
type SectorMap struct {
	TracksPerSide   int
	SectorsPerTrack int
	SidesPerDisk    int
	BytesPerSector  int
}

// NewSectorMap creates a new sector map with standard +3 parameters
func NewSectorMap() *SectorMap {
	return &SectorMap{
		TracksPerSide:   40,
		SectorsPerTrack: 9,
		SidesPerDisk:    1,
		BytesPerSector:  512,
	}
}

// LinearToPhysical converts a linear sector number to physical track/sector/side coordinates
func (sm *SectorMap) LinearToPhysical(linear int) (track, sector, side int, err error) {
	// Validate linear sector number
	maxSectors := sm.TracksPerSide * sm.SectorsPerTrack * sm.SidesPerDisk
	if linear < 0 || linear >= maxSectors {
		return 0, 0, 0, fmt.Errorf("sector number %d out of range (0-%d)", linear, maxSectors-1)
	}

	// Calculate track, sector, and side
	sectorsPerSide := sm.TracksPerSide * sm.SectorsPerTrack
	side = linear / sectorsPerSide
	remainder := linear % sectorsPerSide
	track = remainder / sm.SectorsPerTrack
	sector = remainder % sm.SectorsPerTrack

	return track, sector, side, nil
}

// PhysicalToLinear converts physical track/sector/side coordinates to a linear sector number
func (sm *SectorMap) PhysicalToLinear(track, sector, side int) (int, error) {
	// Validate parameters
	if track < 0 || track >= sm.TracksPerSide {
		return 0, fmt.Errorf("track %d out of range (0-%d)", track, sm.TracksPerSide-1)
	}
	if sector < 0 || sector >= sm.SectorsPerTrack {
		return 0, fmt.Errorf("sector %d out of range (0-%d)", sector, sm.SectorsPerTrack-1)
	}
	if side < 0 || side >= sm.SidesPerDisk {
		return 0, fmt.Errorf("side %d out of range (0-%d)", side, sm.SidesPerDisk-1)
	}

	// Calculate linear sector number
	linear := (side * sm.TracksPerSide * sm.SectorsPerTrack) +
		(track * sm.SectorsPerTrack) +
		sector

	return linear, nil
}

// GetTrackOffset returns the byte offset for the start of a track
func (sm *SectorMap) GetTrackOffset(track, side int) (int64, error) {
	if track < 0 || track >= sm.TracksPerSide {
		return 0, fmt.Errorf("track %d out of range (0-%d)", track, sm.TracksPerSide-1)
	}
	if side < 0 || side >= sm.SidesPerDisk {
		return 0, fmt.Errorf("side %d out of range (0-%d)", side, sm.SidesPerDisk-1)
	}

	offset := int64((side*sm.TracksPerSide + track) * sm.SectorsPerTrack * sm.BytesPerSector)
	return offset, nil
}

// GetSectorOffset returns the byte offset for a specific sector
func (sm *SectorMap) GetSectorOffset(track, sector, side int) (int64, error) {
	// First get the track offset
	trackOffset, err := sm.GetTrackOffset(track, side)
	if err != nil {
		return 0, err
	}

	// Validate sector
	if sector < 0 || sector >= sm.SectorsPerTrack {
		return 0, fmt.Errorf("sector %d out of range (0-%d)", sector, sm.SectorsPerTrack-1)
	}

	// Add sector offset to track offset
	offset := trackOffset + int64(sector*sm.BytesPerSector)
	return offset, nil
}

// GetTrackBounds returns the start and end sector numbers for a track
func (sm *SectorMap) GetTrackBounds(track, side int) (start, end int, err error) {
	linear, err := sm.PhysicalToLinear(track, 0, side)
	if err != nil {
		return 0, 0, err
	}

	start = linear
	end = linear + sm.SectorsPerTrack - 1

	return start, end, nil
}

// ValidateSectorRange checks if a sector number is within valid bounds
func (sm *SectorMap) ValidateSectorRange(linear int) error {
	maxSectors := sm.TracksPerSide * sm.SectorsPerTrack * sm.SidesPerDisk
	if linear < 0 || linear >= maxSectors {
		return fmt.Errorf("sector %d out of range (0-%d)", linear, maxSectors-1)
	}
	return nil
}

// AdjustTrackSide converts track/side for double-sided disk access if needed
func (sm *SectorMap) AdjustTrackSide(track int, alternateSides bool) (adjustedTrack, side int, err error) {
	if track < 0 || track >= (sm.TracksPerSide*sm.SidesPerDisk) {
		return 0, 0, errors.New("track number out of range")
	}

	if alternateSides {
		// Alternating sides mode
		side = track % 2
		adjustedTrack = track / 2
	} else {
		// Sequential sides mode
		side = track / sm.TracksPerSide
		adjustedTrack = track % sm.TracksPerSide
	}

	return adjustedTrack, side, nil
}