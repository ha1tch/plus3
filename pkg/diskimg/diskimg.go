// file: pkg/diskimg/diskimg.go

package diskimg

import (
	"errors"

	"github.com/ha1tch/plus3/internal"
)

// Standard +3 disk parameters
const (
	TracksPerSide   = 40
	SectorsPerTrack = 9
	BytesPerSector  = 512
	SidesPerDisk    = 1 // +3 uses single-sided disks
	DirectorySector = 0 // In +3DOS dir starts at sector 0 of dir track
	DiskSizeInBytes = 184320
)

// DSK format header structure based on standard CPCEMU disk image format
type DiskHeader struct {
	Signature [34]byte  // "EXTENDED CPC DSK File\r\nDisk-Info\r\n"
	Creator   [14]byte  // Name of the creator
	TracksNum uint8     // Number of tracks
	SidesNum  uint8     // Number of sides
	TrackSize uint16    // Size of each track in bytes
	Unused    [204]byte // Unused space to make header 256 bytes
}

// Track information block
type TrackInfo struct {
	Signature  [13]byte // "Track-Info\r\n"
	Unused1    [3]byte
	TrackNum   uint8
	SideNum    uint8
	Unused2    [2]byte
	SectorSize uint8         // sector size = 128 << SectorSize
	SectorsNum uint8         // number of sectors in track
	GapLength  uint8         // gap#3 length
	FillerByte uint8         // filler byte
	SectorInfo [9]SectorInfo // For +3, always 9 sectors per track
}

// Sector information
type SectorInfo struct {
	Track      uint8
	Side       uint8
	SectorID   uint8
	SectorSize uint8
	FDCStatus1 uint8
	FDCStatus2 uint8
	DataLength uint16
}

// DiskImage represents a complete disk image in memory
type DiskImage struct {
	Header          DiskHeader
	Tracks          [][]byte // Raw track data
	Modified        bool
	sectorMap       *internal.SectorMap
	allocation      *SectorAllocation
	directory       []DirectoryEntry
	diskSizeInBytes int
}

// NewDiskImage creates a new empty disk image with standard +3 format
func NewDiskImage() *DiskImage {
	di := &DiskImage{
		Modified:        true,
		sectorMap:       internal.NewSectorMap(),
		directory:       make([]DirectoryEntry, MaxDirectoryEntries),
		diskSizeInBytes: DiskSizeInBytes,
	}

	// Initialize header with standard +3 parameters
	copy(di.Header.Signature[:], "EXTENDED CPC DSK File\r\nDisk-Info\r\n")
	copy(di.Header.Creator[:], "plus3 utility")
	di.Header.TracksNum = TracksPerSide
	di.Header.SidesNum = SidesPerDisk
	di.Header.TrackSize = BytesPerSector * SectorsPerTrack

	// Initialize empty tracks
	di.Tracks = make([][]byte, TracksPerSide*SidesPerDisk)
	for i := range di.Tracks {
		di.Tracks[i] = make([]byte, BytesPerSector*SectorsPerTrack)
	}

	// Initialize sector allocation
	totalSectors := int(di.Header.TracksNum) * int(di.Header.SidesNum) * SectorsPerTrack
	di.allocation = newSectorAllocation(totalSectors)

	return di
}

// GetSectorData retrieves data for a specific sector using proper mapping
func (di *DiskImage) GetSectorData(track, sector, side int) ([]byte, error) {
	// Convert to linear sector number
	linear, err := di.sectorMap.PhysicalToLinear(track, sector, side)
	if err != nil {
		return nil, err
	}

	// Get offset in disk image
	offset, err := di.sectorMap.GetSectorOffset(track, sector, side)
	if err != nil {
		return nil, err
	}

	// Check allocation
	allocated, err := di.allocation.IsSectorAllocated(linear)
	if err != nil {
		return nil, err
	}
	if !allocated {
		return nil, errors.New("attempting to read unallocated sector")
	}

	// Get track data
	trackIdx := track + (side * int(di.Header.TracksNum))
	if trackIdx >= len(di.Tracks) {
		return nil, errors.New("track index out of range")
	}

	// Calculate sector position within track
	sectorStart := int(offset) % len(di.Tracks[trackIdx])
	if sectorStart+BytesPerSector > len(di.Tracks[trackIdx]) {
		return nil, errors.New("sector extends beyond track boundary")
	}

	// Copy sector data
	sectorData := make([]byte, BytesPerSector)
	copy(sectorData, di.Tracks[trackIdx][sectorStart:sectorStart+BytesPerSector])

	return sectorData, nil
}

// SetSectorData writes data to a specific sector using proper mapping
func (di *DiskImage) SetSectorData(track, sector, side int, data []byte) error {
	if len(data) != BytesPerSector {
		return errors.New("invalid sector data size")
	}

	// Convert to linear sector number
	linear, err := di.sectorMap.PhysicalToLinear(track, sector, side)
	if err != nil {
		return err
	}

	// Get offset in disk image
	offset, err := di.sectorMap.GetSectorOffset(track, sector, side)
	if err != nil {
		return err
	}

	// Allocate sector if not already allocated
	if !di.allocation.allocated[linear] {
		err = di.allocation.AllocateSectors(linear, 1)
		if err != nil {
			return err
		}
	}

	// Get track data
	trackIdx := track + (side * int(di.Header.TracksNum))
	if trackIdx >= len(di.Tracks) {
		return errors.New("track index out of range")
	}

	// Calculate sector position within track
	sectorStart := int(offset) % len(di.Tracks[trackIdx])
	if sectorStart+BytesPerSector > len(di.Tracks[trackIdx]) {
		return errors.New("sector extends beyond track boundary")
	}

	// Write sector data
	copy(di.Tracks[trackIdx][sectorStart:sectorStart+BytesPerSector], data)
	di.Modified = true

	return nil
}

func (di *DiskImage) TotalSectors() int {
	return di.diskSizeInBytes / BytesPerSector
}

// GetTrackData returns the raw data for a specific track
func (di *DiskImage) GetTrackData(track, side int) ([]byte, error) {
	trackIdx, err := di.sectorMap.PhysicalToLinear(track, 0, side)
	if err != nil {
		return nil, err
	}

	if trackIdx >= len(di.Tracks) {
		return nil, errors.New("track index out of range")
	}

	trackData := make([]byte, len(di.Tracks[trackIdx]))
	copy(trackData, di.Tracks[trackIdx])
	return trackData, nil
}

// SetTrackData sets the raw data for a specific track
func (di *DiskImage) SetTrackData(track, side int, data []byte) error {
	if len(data) != int(di.Header.TrackSize) {
		return errors.New("invalid track data size")
	}

	trackIdx, err := di.sectorMap.PhysicalToLinear(track, 0, side)
	if err != nil {
		return err
	}

	if trackIdx >= len(di.Tracks) {
		return errors.New("track index out of range")
	}

	copy(di.Tracks[trackIdx], data)
	di.Modified = true

	// Update sector allocation for this track
	start, end, err := di.sectorMap.GetTrackBounds(track, side)
	if err != nil {
		return err
	}

	err = di.allocation.AllocateSectors(start, end-start+1)
	if err != nil {
		return err
	}

	return nil
}
