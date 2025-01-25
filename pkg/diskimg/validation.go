// file: pkg/diskimg/validation.go

package diskimg

import (
	"bytes"
	"errors"
	"fmt"
)

// ValidationError represents a specific disk format validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error - %s: %s", e.Field, e.Message)
}

// ValidateFormat performs comprehensive validation of the disk image format
func (di *DiskImage) ValidateFormat() error {
	// Validate header
	if err := di.validateHeaderFormat(); err != nil {
		return err
	}

	// Validate track data
	if err := di.validateTrackData(); err != nil {
		return err
	}

	// Validate disk parameters
	if err := di.validateDiskParameters(); err != nil {
		return err
	}

	return nil
}

// validateHeaderFormat checks if the disk header is valid for +3DOS format
func (di *DiskImage) validateHeaderFormat() error {
	// Check disk image signature
	expectedSig := []byte("EXTENDED CPC DSK File\r\nDisk-Info\r\n")
	if !bytes.Equal(di.Header.Signature[:len(expectedSig)], expectedSig) {
		return &ValidationError{
			Field:   "Header.Signature",
			Message: "invalid disk image signature",
		}
	}

	// Verify creator field is not empty and properly terminated
	if bytes.Equal(di.Header.Creator[:], make([]byte, 14)) {
		return &ValidationError{
			Field:   "Header.Creator",
			Message: "creator field is empty",
		}
	}

	// Basic parameter validation
	if di.Header.TracksNum == 0 {
		return &ValidationError{
			Field:   "Header.TracksNum",
			Message: "number of tracks cannot be zero",
		}
	}

	if di.Header.SidesNum == 0 {
		return &ValidationError{
			Field:   "Header.SidesNum",
			Message: "number of sides cannot be zero",
		}
	}

	return nil
}

// validateTrackData verifies all track data structures
func (di *DiskImage) validateTrackData() error {
	expectedTracks := int(di.Header.TracksNum * di.Header.SidesNum)
	
	// Check track array size
	if len(di.Tracks) != expectedTracks {
		return &ValidationError{
			Field:   "Tracks",
			Message: fmt.Sprintf("expected %d tracks, found %d", expectedTracks, len(di.Tracks)),
		}
	}

	// Verify each track's data
	for i, track := range di.Tracks {
		trackNum := i % int(di.Header.TracksNum)
		side := i / int(di.Header.TracksNum)

		// Check track size
		if len(track) != int(di.Header.TrackSize) {
			return &ValidationError{
				Field:   fmt.Sprintf("Track[%d]", i),
				Message: fmt.Sprintf("invalid track size: expected %d, got %d", di.Header.TrackSize, len(track)),
			}
		}

		// Verify track can be properly sectored
		if len(track)%BytesPerSector != 0 {
			return &ValidationError{
				Field:   fmt.Sprintf("Track[%d]", i),
				Message: "track size is not a multiple of sector size",
			}
		}

		// Check track information
		if _, err := di.GetTrackInfo(trackNum, side); err != nil {
			return &ValidationError{
				Field:   fmt.Sprintf("Track[%d]Info", i),
				Message: err.Error(),
			}
		}
	}

	return nil
}

// validateDiskParameters checks if disk parameters match +3DOS requirements
func (di *DiskImage) validateDiskParameters() error {
	// Validate parameters against +3DOS standard format
	if di.Header.TracksNum != TracksPerSide {
		return &ValidationError{
			Field:   "DiskParameters.TracksNum",
			Message: fmt.Sprintf("invalid number of tracks for +3DOS format: expected %d, got %d", 
				TracksPerSide, di.Header.TracksNum),
		}
	}

	if di.Header.SidesNum != SidesPerDisk {
		return &ValidationError{
			Field:   "DiskParameters.SidesNum",
			Message: fmt.Sprintf("invalid number of sides for +3DOS format: expected %d, got %d", 
				SidesPerDisk, di.Header.SidesNum),
		}
	}

	expectedTrackSize := BytesPerSector * SectorsPerTrack
	if int(di.Header.TrackSize) != expectedTrackSize {
		return &ValidationError{
			Field:   "DiskParameters.TrackSize",
			Message: fmt.Sprintf("invalid track size for +3DOS format: expected %d, got %d", 
				expectedTrackSize, di.Header.TrackSize),
		}
	}

	return nil
}

// IsPlus3Format checks if the disk image is in +3DOS format
func (di *DiskImage) IsPlus3Format() bool {
	return di.Header.TracksNum == TracksPerSide &&
		di.Header.SidesNum == SidesPerDisk &&
		int(di.Header.TrackSize) == BytesPerSector*SectorsPerTrack
}

// ValidateBootSector checks if the disk has a valid boot sector
func (di *DiskImage) ValidateBootSector() error {
	// Get the boot sector (Track 0, Side 0, Sector 1)
	bootSector, err := di.GetSectorData(0, 0, 0)
	if err != nil {
		return errors.New("failed to read boot sector")
	}

	// Basic boot sector validation (minimal check)
	if len(bootSector) != BytesPerSector {
		return errors.New("invalid boot sector size")
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
		return errors.New("invalid boot sector checksum")
	}

	return nil
}