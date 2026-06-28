// file: pkg/diskimg/diskimg.go

package diskimg

import (
	"github.com/ha1tch/plus3/internal"
)

const (
	TracksPerSide    = 40 // standard +3 logical track count
	MaxTracksPerSide = 45 // physical maximum carried by real .dsk images
	SectorsPerTrack  = 9
	BytesPerSector   = 512
	DiskSizeInBytes  = 184320 // ZX Spectrum +3 single-sided disk (40*9*512)
	BlockSize        = 1024   // +3DOS uses 1K allocation blocks
	SectorsPerBlock  = BlockSize / BytesPerSector
	SidesPerDisk     = 1
)

// DiskHeader is the Disc Information Block at offset 0 of a .dsk image.
type DiskHeader struct {
	Signature [34]byte // "MV - CPCEMU Disk-File\r\nDisk-Info\r\n"
	Creator   [14]byte // name of the creating utility
	TracksNum uint8    // number of tracks
	SidesNum  uint8    // number of sides
	TrackSize uint16   // size of a track (incl. 256-byte track info block)
	Unused    [204]byte
}

// DiskImage represents a ZX Spectrum +3 disk image.
type DiskImage struct {
	Header   DiskHeader
	Tracks   [][]byte // raw track data (track info block + sector data) per track
	Modified bool
	DiskType uint8 // intended CP/M format: 0=+3 standard, 1=CPC system, 2=CPC data

	directory  Directory
	allocation *SectorAllocation
	fileAlloc  *FileAllocation
	sectorMap  *internal.SectorMap
}

// TotalSectors returns the total number of sectors on the disk.
func (di *DiskImage) TotalSectors() int {
	return int(di.Header.TracksNum) * int(di.Header.SidesNum) * SectorsPerTrack
}

// NewDiskImage initializes a new, formatted, blank +3 disk image with standard
// geometry. Each track is built with a proper track information block and its
// nine 512-byte sectors filled with the format filler byte (0xE5), so the disk
// is immediately usable - matching what a real +3 format produces.
func NewDiskImage() *DiskImage {
	di := &DiskImage{
		sectorMap: internal.NewSectorMap(),
		directory: Directory{Entries: make([]DirectoryEntry, MaxDirectoryEntries)},
	}
	di.Header.TracksNum = TracksPerSide
	di.Header.SidesNum = 1
	di.Header.TrackSize = uint16(SectorsPerTrack*BytesPerSector + 256)
	copy(di.Header.Signature[:], "MV - CPCEMU Disk-File\r\nDisk-Info\r\n")
	copy(di.Header.Creator[:], "plus3")
	di.allocation = newSectorAllocation(di.TotalSectors())
	di.fileAlloc = newFileAllocation(di)

	// Format every track: build the track info block + 0xE5-filled sectors.
	trackBytes := 256 + SectorsPerTrack*BytesPerSector
	di.Tracks = make([][]byte, int(di.Header.TracksNum)*int(di.Header.SidesNum))
	for t := range di.Tracks {
		block := make([]byte, trackBytes)
		// Track information block.
		copy(block[0:], "Track-Info\r\n")
		block[0x10] = byte(t % int(di.Header.TracksNum)) // track number
		block[0x11] = byte(t / int(di.Header.TracksNum)) // side number
		block[0x14] = 2                                  // sector size code (512)
		block[0x15] = SectorsPerTrack                    // sectors per track
		block[0x16] = 0x4E                               // gap3 length (78)
		block[0x17] = 0xE5                               // filler byte
		// Sector information list (8 bytes per sector), IDs R=1..9.
		for sct := 0; sct < SectorsPerTrack; sct++ {
			si := 0x18 + sct*8
			block[si+0] = byte(t % int(di.Header.TracksNum)) // C
			block[si+1] = byte(t / int(di.Header.TracksNum)) // H
			block[si+2] = byte(sct + 1)                      // R (sector ID, from 1)
			block[si+3] = 2                                  // N (512)
			block[si+6] = byte(BytesPerSector & 0xFF)        // actual length lo
			block[si+7] = byte(BytesPerSector >> 8)          // actual length hi
		}
		// Fill sector data area with the format filler (0xE5).
		for i := 256; i < trackBytes; i++ {
			block[i] = 0xE5
		}
		di.Tracks[t] = block
	}
	return di
}

// trackIndex returns the index into di.Tracks for a given track and side.
func (di *DiskImage) trackIndex(track, side int) int {
	return side*int(di.Header.TracksNum) + track
}

// GetSectorData retrieves the 512-byte data for a track/sector/side.
// Sector data follows the 256-byte track information block in each track.
func (di *DiskImage) GetSectorData(track, sector, side int) ([]byte, error) {
	if track < 0 || track >= int(di.Header.TracksNum) ||
		sector < 0 || sector >= SectorsPerTrack ||
		side < 0 || side >= int(di.Header.SidesNum) {
		return nil, ErrInvalidSector
	}
	idx := di.trackIndex(track, side)
	if idx >= len(di.Tracks) || di.Tracks[idx] == nil {
		return nil, ErrInvalidSector
	}
	off := 256 + sector*BytesPerSector
	td := di.Tracks[idx]
	if off+BytesPerSector > len(td) {
		return nil, ErrInvalidSector
	}
	out := make([]byte, BytesPerSector)
	copy(out, td[off:off+BytesPerSector])
	return out, nil
}

// SetSectorData writes 512 bytes into a track/sector/side, marking the disk modified.
func (di *DiskImage) SetSectorData(track, sector, side int, data []byte) error {
	if len(data) != BytesPerSector {
		return ErrInvalidSectorSize
	}
	if track < 0 || track >= int(di.Header.TracksNum) ||
		sector < 0 || sector >= SectorsPerTrack ||
		side < 0 || side >= int(di.Header.SidesNum) {
		return ErrInvalidSector
	}
	idx := di.trackIndex(track, side)
	if idx >= len(di.Tracks) {
		return ErrInvalidSector
	}
	if di.Tracks[idx] == nil {
		di.Tracks[idx] = make([]byte, di.Header.TrackSize)
	}
	off := 256 + sector*BytesPerSector
	copy(di.Tracks[idx][off:off+BytesPerSector], data)
	di.Modified = true
	return nil
}
