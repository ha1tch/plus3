// file: pkg/diskimg/writer.go

package diskimg

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// SaveToFile saves the disk image to a file
func (di *DiskImage) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return di.Save(file)
}

// Save writes the disk image to an io.Writer
func (di *DiskImage) Save(w io.Writer) error {
	// Validate disk image before saving
	if err := di.ValidateFormat(); err != nil {
		return err
	}

	// Write the main disk header
	if err := binary.Write(w, binary.LittleEndian, &di.Header); err != nil {
		return errors.New("failed to write disk header")
	}

	// Write each track with its header
	trackCount := int(di.Header.TracksNum * di.Header.SidesNum)
	for i := 0; i < trackCount; i++ {
		track := i % int(di.Header.TracksNum)
		side := i / int(di.Header.TracksNum)

		// Create and write track info
		trackInfo, err := di.createTrackInfo(track, side)
		if err != nil {
			return err
		}

		if err := binary.Write(w, binary.LittleEndian, trackInfo); err != nil {
			return errors.New("failed to write track info")
		}

        // Get the track data using sector mapping
        start, end, err := di.sectorMap.GetTrackBounds(track, side)
        if err != nil {
            return err
        }

        // Check if track is allocated
        trackAllocated, err := di.allocation.GetTrackAllocation(track, side)
        if err != nil {
            return err
        }

        // Write track data
        trackData := di.Tracks[i]
        if !containsTrue(trackAllocated) {
            // If track is not allocated, write empty (formatted) sectors
            trackData = make([]byte, di.Header.TrackSize)
            for i := range trackData {
                trackData[i] = 0xE5 // Standard format filler byte
            }
        }

        if _, err := w.Write(trackData); err != nil {
            return errors.New("failed to write track data")
        }
	}

	di.Modified = false
	return nil
}

// containsTrue checks if a boolean slice contains any true values
func containsTrue(slice []bool) bool {
    for _, v := range slice {
        if v {
            return true
        }
    }
    return false
}

// createTrackInfo generates track information for a specific track
func (di *DiskImage) createTrackInfo(track, side int) (*TrackInfo, error) {
	// Validate track and side
	if track >= int(di.Header.TracksNum) || track < 0 {
		return nil, errors.New("track number out of range")
	}
	if side >= int(di.Header.SidesNum) || side < 0 {
		return nil, errors.New("side number out of range")
	}

	info := &TrackInfo{
		TrackNum:   uint8(track),
		SideNum:    uint8(side),
		SectorSize: 2, // 512 bytes = 128 << 2
		SectorsNum: SectorsPerTrack,
		GapLength:  0x52, // Standard gap length for +3 format
		FillerByte: 0xE5, // Standard filler byte
	}
	copy(info.Signature[:], "Track-Info\r\n")

	// Set up sector information using sector mapping
	for i := 0; i < SectorsPerTrack; i++ {
		linear, err := di.sectorMap.PhysicalToLinear(track, i, side)
		if err != nil {
			return nil, err
		}

		allocated, err := di.allocation.IsSectorAllocated(linear)
		if err != nil {
			return nil, err
		}

		info.SectorInfo[i] = SectorInfo{
			Track:      uint8(track),
			Side:       uint8(side),
			SectorID:   uint8(i + 1), // Sectors typically numbered from 1
			SectorSize: 2,            // 512 bytes
			DataLength: BytesPerSector,
			// Set FDC status based on allocation
			FDCStatus1: boolToByte(!allocated), // 0 if allocated, non-zero if not
			FDCStatus2: 0,
		}
	}

	return info, nil
}

// boolToByte converts a boolean to a byte (0 or 1)
func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}