// file: pkg/diskimg/diskimg_test.go

package diskimg

import (
	"bytes"
	"testing"
)

func TestNewDiskImage(t *testing.T) {
	di := NewDiskImage()
	
	// Test disk header initialization
	if di == nil {
		t.Fatal("NewDiskImage returned nil")
	}
	
	// Check signature
	expectedSig := "EXTENDED CPC DSK File\r\nDisk-Info\r\n"
	if string(di.Header.Signature[:len(expectedSig)]) != expectedSig {
		t.Errorf("Wrong signature. Expected %s, got %s", 
			expectedSig, string(di.Header.Signature[:]))
	}

	// Check basic disk parameters
	if di.Header.TracksNum != TracksPerSide {
		t.Errorf("Wrong track count. Expected %d, got %d", 
			TracksPerSide, di.Header.TracksNum)
	}
	if di.Header.SidesNum != SidesPerDisk {
		t.Errorf("Wrong side count. Expected %d, got %d", 
			SidesPerDisk, di.Header.SidesNum)
	}
	
	// Check track initialization
	expectedTracks := int(di.Header.TracksNum * di.Header.SidesNum)
	if len(di.Tracks) != expectedTracks {
		t.Errorf("Wrong number of tracks. Expected %d, got %d", 
			expectedTracks, len(di.Tracks))
	}

	// Check track sizes
	expectedTrackSize := BytesPerSector * SectorsPerTrack
	for i, track := range di.Tracks {
		if len(track) != expectedTrackSize {
			t.Errorf("Track %d wrong size. Expected %d, got %d", 
				i, expectedTrackSize, len(track))
		}
	}
}

func TestSectorOperations(t *testing.T) {
	di := NewDiskImage()

	// Test data
	testData := []byte("Test sector data")
	paddedData := make([]byte, BytesPerSector)
	copy(paddedData, testData)

	// Test writing sector
	err := di.SetSectorData(0, 0, 0, paddedData)
	if err != nil {
		t.Errorf("SetSectorData failed: %v", err)
	}

	// Test reading sector
	readData, err := di.GetSectorData(0, 0, 0)
	if err != nil {
		t.Errorf("GetSectorData failed: %v", err)
	}

	if !bytes.Equal(readData, paddedData) {
		t.Error("Read data doesn't match written data")
	}

	// Test invalid sector access
	_, err = di.GetSectorData(TracksPerSide+1, 0, 0)
	if err == nil {
		t.Error("Expected error for invalid track access")
	}

	_, err = di.GetSectorData(0, SectorsPerTrack+1, 0)
	if err == nil {
		t.Error("Expected error for invalid sector access")
	}
}

func TestTrackOperations(t *testing.T) {
	di := NewDiskImage()

	// Create test track data
	trackSize := BytesPerSector * SectorsPerTrack
	testTrack := make([]byte, trackSize)
	for i := range testTrack {
		testTrack[i] = byte(i & 0xFF)
	}

	// Test writing track
	err := di.SetTrackData(0, 0, testTrack)
	if err != nil {
		t.Errorf("SetTrackData failed: %v", err)
	}

	// Test reading track
	readTrack, err := di.GetTrackData(0, 0)
	if err != nil {
		t.Errorf("GetTrackData failed: %v", err)
	}

	if !bytes.Equal(readTrack, testTrack) {
		t.Error("Read track doesn't match written track")
	}

	// Test invalid track access
	_, err = di.GetTrackData(TracksPerSide+1, 0)
	if err == nil {
		t.Error("Expected error for invalid track access")
	}
}

func TestDiskValidation(t *testing.T) {
	di := NewDiskImage()

	// Create validator with basic checks
	validator := NewDiskCheck(di, ValidationBasic)
	errors := validator.Validate()
	
	if len(errors) > 0 {
		t.Errorf("Validation failed for new disk: %v", errors)
	}

	// Corrupt disk header and test validation
	di.Header.TracksNum = 0
	errors = validator.Validate()
	
	if len(errors) == 0 {
		t.Error("Validation should fail with corrupt header")
	}
}

func TestDirectoryOperations(t *testing.T) {
	di := NewDiskImage()

	// Test directory initialization
	err := di.InitializeDirectory()
	if err != nil {
		t.Errorf("Directory initialization failed: %v", err)
	}

	dir, err := di.GetDirectory()
	if err != nil {
		t.Errorf("GetDirectory failed: %v", err)
	}

	// Test file creation
	entry, err := dir.AddFile("TEST.BAS")
	if err != nil {
		t.Errorf("AddFile failed: %v", err)
	}

	// Verify file entry
	if entry.GetFilename() != "TEST.BAS" {
		t.Errorf("Wrong filename. Expected TEST.BAS, got %s", entry.GetFilename())
	}

	// Test file search
	found, _, err := dir.FindFile("TEST.BAS")
	if err != nil {
		t.Errorf("FindFile failed: %v", err)
	}
	if found == nil {
		t.Error("File not found after creation")
	}

	// Test file deletion
	err = dir.DeleteFile("TEST.BAS")
	if err != nil {
		t.Errorf("DeleteFile failed: %v", err)
	}

	// Verify deletion
	found, _, err = dir.FindFile("TEST.BAS")
	if err == nil {
		t.Error("File still exists after deletion")
	}
}

func TestSectorAllocation(t *testing.T) {
	di := NewDiskImage()

	// Test sector allocation tracking
	linear, err := di.sectorMap.PhysicalToLinear(0, 0, 0)
	if err != nil {
		t.Errorf("PhysicalToLinear failed: %v", err)
	}

	// Allocate a sector
	err = di.allocation.AllocateSectors(linear, 1)
	if err != nil {
		t.Errorf("AllocateSectors failed: %v", err)
	}

	// Verify allocation
	allocated, err := di.allocation.IsSectorAllocated(linear)
	if err != nil {
		t.Errorf("IsSectorAllocated failed: %v", err)
	}
	if !allocated {
		t.Error("Sector should be allocated")
	}

	// Test freeing sectors
	err = di.allocation.FreeSectors(linear, 1)
	if err != nil {
		t.Errorf("FreeSectors failed: %v", err)
	}

	// Verify sector is free
	allocated, err = di.allocation.IsSectorAllocated(linear)
	if err != nil {
		t.Errorf("IsSectorAllocated failed: %v", err)
	}
	if allocated {
		t.Error("Sector should be free")
	}
}