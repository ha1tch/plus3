// file: pkg/diskimg/reader.go

package diskimg

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// LoadFromFile loads a DSK image from a file
func LoadFromFile(filename string) (*DiskImage, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return Load(file)
}

// Load reads a DSK image from an io.Reader
func Load(r io.Reader) (*DiskImage, error) {
	di := &DiskImage{
		sectorMap: internal.NewSectorMap(),
	}

	// Read the header
	if err := binary.Read(r, binary.LittleEndian, &di.Header); err != nil {
		return nil, errors.New("failed to read disk header")
	}

	// Validate the header
	if err := di.validateHeader(); err != nil {
		return nil, err
	}

	// Initialize sector allocation
	totalSectors := int(di.Header.TracksNum) * int(di.Header.SidesNum) * SectorsPerTrack
	di.allocation = newSectorAllocation(totalSectors)

	// Read track data
	trackCount := int(di.Header.TracksNum * di.Header.SidesNum)
	di.Tracks = make([][]byte, trackCount)

	for i := 0; i < trackCount; i++ {
		// Calculate physical track and side
		track := i % int(di.Header.TracksNum)
		side := i / int(di.Header.TracksNum)

		// Read track header
		var trackInfo TrackInfo
		if err := binary.Read(r, binary.LittleEndian, &trackInfo); err != nil {
			return nil, errors.New("failed to read track info")
		}

		// Validate track info
		if err := di.validateTrackInfo(&trackInfo, track, side); err != nil {
			return nil, err
		}

		// Allocate and read track data
		trackData := make([]byte, di.Header.TrackSize)
		if _, err := io.ReadFull(r, trackData); err != nil {
			return nil, errors.New("failed to read track data")
		}
		di.Tracks[i] = trackData

		// Mark sectors in this track as allocated if they contain data
		start, end, err := di.sectorMap.GetTrackBounds(track, side)
		if err != nil {
			return nil, err
		}
		
		// Check if track contains any non-zero data
		hasData := false
		for _, b := range trackData {
			if b != 0 {
				hasData = true
				break
			}
		}
		
		if hasData {
			err = di.allocation.AllocateSectors(start, end-start+1)
			if err != nil {
				return nil, err
			}
		}
	}

	di.Modified = false
	return di, nil
}

// validateTrackInfo validates track information against expected values
func (di *DiskImage) validateTrackInfo(info *TrackInfo, track, side int) error {
	// Check track signature
	if string(info.Signature[:12]) != "Track-Info\r\n" {
		return errors.New("invalid track signature")
	}

	// Validate track numbering
	if int(info.TrackNum) != track {
		return errors.New("track number mismatch")
	}
	if int(info.SideNum) != side {
		return errors.New("side number mismatch")
	}

	// Check sector parameters
	if info.SectorSize != 2 { // 512 bytes = 2 (128 << 2)
		return errors.New("unsupported sector size")
	}
	if info.SectorsNum != SectorsPerTrack {
		return errors.New("invalid sectors per track")
	}

	// Validate each sector info
	for i := range info.SectorInfo {
		if err := di.validateSectorInfo(&info.SectorInfo[i], track, side); err != nil {
			return err
		}
	}

	return nil
}

// validateSectorInfo checks individual sector information
func (di *DiskImage) validateSectorInfo(info *SectorInfo, track, side int) error {
	if int(info.Track) != track {
		return errors.New("sector track number mismatch")
	}
	if int(info.Side) != side {
		return errors.New("sector side number mismatch")
	}
	if info.SectorSize != 2 { // 512 bytes
		return errors.New("invalid sector size in sector info")
	}
	if info.DataLength != BytesPerSector {
		return errors.New("invalid sector data length")
	}
	return nil
}

// validateHeader checks if the disk header is valid for a +3 disk
func (di *DiskImage) validateHeader() error {
	// Check signature
	sig := string(di.Header.Signature[:])
	if sig[:34] != "EXTENDED CPC DSK File\r\nDisk-Info\r\n" {
		return errors.New("invalid disk image signature")
	}

	// Validate disk parameters for +3 format
	if di.Header.TracksNum != TracksPerSide {
		return errors.New("invalid number of tracks for +3 format")
	}

	if di.Header.SidesNum != SidesPerDisk {
		return errors.New("invalid number of sides for +3 format")
	}

	expectedTrackSize := BytesPerSector * SectorsPerTrack
	if int(di.Header.TrackSize) != expectedTrackSize {
		return errors.New("invalid track size for +3 format")
	}

	return nil
}