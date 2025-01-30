// file: pkg/diskimg/track.go

package diskimg

// TrackInfo contains track-level metadata
type TrackInfo struct {
	Signature   [13]byte    // "Track-Info\r\n"
	Unused1     [3]byte
	TrackNum    uint8
	SideNum     uint8
	Unused2     [2]byte
	SectorSize  uint8       // Sector size = 128 << SectorSize
	SectorsNum  uint8       // Number of sectors in track
	GapLength   uint8       // Gap#3 length
	FillerByte  uint8       // Filler byte
	SectorInfo  []SectorInfo
}

// SectorInfo contains per-sector metadata
type SectorInfo struct {
	Track      uint8
	Side       uint8
	SectorID   uint8
	Size       uint8  // Size = 128 << Size
	Status1    uint8  // FDC status register 1
	Status2    uint8  // FDC status register 2
	ActualSize uint16 // Actual data length
}

// NewTrackInfo creates a new track info block
func NewTrackInfo(track, side int) *TrackInfo {
	ti := &TrackInfo{
		TrackNum:   uint8(track),
		SideNum:    uint8(side),
		SectorSize: 2,          // 512 bytes (128 << 2)
		SectorsNum: SectorsPerTrack,
		GapLength:  0x52,       // Standard gap for +3DOS
		FillerByte: 0xE5,       // Standard filler byte
		SectorInfo: make([]SectorInfo, SectorsPerTrack),
	}
	copy(ti.Signature[:], "Track-Info\r\n")

	// Initialize sector info
	for i := range ti.SectorInfo {
		ti.SectorInfo[i] = SectorInfo{
			Track:      uint8(track),
			Side:       uint8(side),
			SectorID:   uint8(i + 1),
			Size:       2,                // 512 bytes
			ActualSize: BytesPerSector,
		}
	}

	return ti
}

// GetTrackInfo returns track information for given track
func (di *DiskImage) GetTrackInfo(track, side int) (*TrackInfo, error) {
	if track < 0 || track >= int(di.Header.TracksNum) {
		return nil, ErrInvalidTrack
	}
	if side < 0 || side >= int(di.Header.SidesNum) {
		return nil, ErrInvalidSide
	}

	return NewTrackInfo(track, side), nil
}

// ValidateTrackInfo verifies track information
func (ti *TrackInfo) Validate() error {
	if string(ti.Signature[:12]) != "Track-Info\r\n" {
		return ErrInvalidTrackSignature
	}

	if ti.SectorSize != 2 { // 512 bytes
		return ErrInvalidSectorSize
	}

	if ti.SectorsNum != SectorsPerTrack {
		return ErrInvalidSectorCount
	}

	for i, si := range ti.SectorInfo {
		if si.Size != 2 {
			return ErrInvalidSectorSize
		}
		if si.ActualSize != BytesPerSector {
			return ErrInvalidSectorSize
		}
		if si.Track != ti.TrackNum {
			return ErrInvalidTrackNum
		}
		if si.Side != ti.SideNum {
			return ErrInvalidSide
		}
		if si.SectorID != uint8(i+1) {
			return ErrInvalidSectorID
		}
	}

	return nil
}