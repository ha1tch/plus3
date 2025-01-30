// file: pkg/diskimg/fileheader_test.go

package diskimg

import (
	"bytes"
	"testing"
)

func TestNewPlus3DosHeader(t *testing.T) {
	header := NewPlus3DosHeader()

	// Check signature
	expectedSig := []byte(HeaderSignature)
	if !bytes.Equal(header.Signature[:], expectedSig) {
		t.Errorf("Wrong signature. Expected %s, got %s", 
			expectedSig, header.Signature)
	}

	// Check standard values
	if header.SoftEOF != HeaderSoftEOF {
		t.Errorf("Wrong soft-EOF. Expected 0x%02X, got 0x%02X", 
			HeaderSoftEOF, header.SoftEOF)
	}

	if header.Issue != HeaderIssue {
		t.Errorf("Wrong issue number. Expected %d, got %d", 
			HeaderIssue, header.Issue)
	}

	if header.Version != HeaderVersion {
		t.Errorf("Wrong version. Expected %d, got %d", 
			HeaderVersion, header.Version)
	}
}

func TestBasicHeaderProgram(t *testing.T) {
	header := NewPlus3DosHeader()

	// Set up a BASIC program header
	length := uint16(1000)
	line := uint16(10)
	progLen := uint16(950)
	
	err := header.SetBasicHeader(FileTypeProgram, length, line, progLen)
	if err != nil {
		t.Fatalf("SetBasicHeader failed: %v", err)
	}

	// Read back and verify
	fileType, gotLength, gotLine, gotProgLen := header.GetBasicHeader()
	
	if fileType != FileTypeProgram {
		t.Errorf("Wrong file type. Expected %d, got %d", FileTypeProgram, fileType)
	}
	if gotLength != length {
		t.Errorf("Wrong length. Expected %d, got %d", length, gotLength)
	}
	if gotLine != line {
		t.Errorf("Wrong LINE. Expected %d, got %d", line, gotLine)
	}
	if gotProgLen != progLen {
		t.Errorf("Wrong program length. Expected %d, got %d", progLen, gotProgLen)
	}
}

func TestBasicHeaderCode(t *testing.T) {
	header := NewPlus3DosHeader()

	// Set up a CODE file header
	length := uint16(6912) // Screen$ size
	loadAddr := uint16(16384) // Screen memory address
	
	err := header.SetBasicHeader(FileTypeCode, length, loadAddr, 0)
	if err != nil {
		t.Fatalf("SetBasicHeader failed: %v", err)
	}

	// Read back and verify
	fileType, gotLength, gotLoadAddr, _ := header.GetBasicHeader()
	
	if fileType != FileTypeCode {
		t.Errorf("Wrong file type. Expected %d, got %d", FileTypeCode, fileType)
	}
	if gotLength != length {
		t.Errorf("Wrong length. Expected %d, got %d", length, gotLength)
	}
	if gotLoadAddr != loadAddr {
		t.Errorf("Wrong load address. Expected %d, got %d", loadAddr, gotLoadAddr)
	}
}

func TestHeaderChecksum(t *testing.T) {
	header := NewPlus3DosHeader()

	// Set some data
	err := header.SetBasicHeader(FileTypeProgram, 1000, 10, 950)
	if err != nil {
		t.Fatalf("SetBasicHeader failed: %v", err)
	}

	// Update checksum
	header.UpdateChecksum()

	// Verify checksum
	if !header.verifyChecksum() {
		t.Error("Checksum verification failed")
	}

	// Corrupt data and verify checksum fails
	header.HeaderData[0] = 255
	if header.verifyChecksum() {
		t.Error("Checksum verification should fail with corrupted data")
	}
}

func TestHeaderByteConversion(t *testing.T) {
	header := NewPlus3DosHeader()

	// Set some test data
	header.SetBasicHeader(FileTypeProgram, 1000, 10, 950)
	header.UpdateChecksum()

	// Convert to bytes
	data := header.toBytes()
	if len(data) != HeaderSize {
		t.Errorf("Wrong data size. Expected %d, got %d", HeaderSize, len(data))
	}

	// Create new header from bytes
	newHeader := &Plus3DosHeader{}
	err := newHeader.FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes failed: %v", err)
	}

	// Verify headers match
	if !bytes.Equal(header.Signature[:], newHeader.Signature[:]) {
		t.Error("Signatures don't match after conversion")
	}
	if header.Checksum != newHeader.Checksum {
		t.Error("Checksums don't match after conversion")
	}
}

func TestHeaderValidation(t *testing.T) {
	header := NewPlus3DosHeader()
	
	// Valid header should pass validation
	err := header.Validate()
	if err != nil {
		t.Errorf("Valid header failed validation: %v", err)
	}

	// Test invalid signature
	badHeader := *header
	copy(badHeader.Signature[:], "INVALID")
	if err := badHeader.Validate(); err == nil {
		t.Error("Validation should fail with invalid signature")
	}

	// Test invalid soft-EOF
	badHeader = *header
	badHeader.SoftEOF = 0x00
	if err := badHeader.Validate(); err == nil {
		t.Error("Validation should fail with invalid soft-EOF")
	}

	// Test invalid file type
	badHeader = *header
	badHeader.HeaderData[0] = 255
	if err := badHeader.Validate(); err == nil {
		t.Error("Validation should fail with invalid file type")
	}
}

func TestHeaderString(t *testing.T) {
	header := NewPlus3DosHeader()

	// Test BASIC program
	header.SetBasicHeader(FileTypeProgram, 1000, 10, 950)
	str := header.String()
	if str == "" {
		t.Error("String representation should not be empty")
	}

	// Test CODE file
	header.SetBasicHeader(FileTypeCode, 6912, 16384, 0)
	str = header.String()
	if str == "" {
		t.Error("String representation should not be empty")
	}
}