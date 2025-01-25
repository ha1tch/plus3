// file: pkg/diskimg/fileheader.go

package diskimg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// Header constants
	HeaderSignature = "PLUS3DOS"
	HeaderSoftEOF   = 0x1A
	HeaderSize      = 128
	HeaderVersion   = 1
	HeaderIssue     = 1

	// File types for +3 BASIC header data
	FileTypeProgram      = 0
	FileTypeNumericArray = 1
	FileTypeCharArray    = 2
	FileTypeCode         = 3
)

// Plus3DosHeader represents the 128-byte header structure
type Plus3DosHeader struct {
	Signature  [8]byte   // "PLUS3DOS"
	SoftEOF    byte      // 0x1A
	Issue      byte      // Issue number
	Version    byte      // Version number
	FileLength uint32    // Length of the file in bytes
	HeaderData [8]byte   // BASIC header data
	Reserved   [104]byte // Reserved space
	Checksum   byte      // Sum of bytes 0-126 modulo 256
}

// NewPlus3DosHeader creates a new header with standard values
func NewPlus3DosHeader() *Plus3DosHeader {
	header := &Plus3DosHeader{
		SoftEOF: HeaderSoftEOF,
		Issue:   HeaderIssue,
		Version: HeaderVersion,
	}
	copy(header.Signature[:], HeaderSignature)
	return header
}

// SetBasicHeader sets the BASIC-specific header data
func (h *Plus3DosHeader) SetBasicHeader(fileType byte, length uint16, param1, param2 uint16) error {
	if fileType > FileTypeCode {
		return fmt.Errorf("invalid file type: %d", fileType)
	}

	h.HeaderData[0] = fileType

	// Set file length (2 bytes)
	binary.LittleEndian.PutUint16(h.HeaderData[1:3], length)

	// Set parameters based on file type
	switch fileType {
	case FileTypeProgram:
		// param1 = LINE (autostart), param2 = program length
		binary.LittleEndian.PutUint16(h.HeaderData[3:5], param1) // LINE
		binary.LittleEndian.PutUint16(h.HeaderData[5:7], param2) // Length
	case FileTypeNumericArray, FileTypeCharArray:
		// param1 = variable name, param2 unused
		h.HeaderData[3] = byte(param1) // Variable name
	case FileTypeCode:
		// param1 = load address, param2 unused
		binary.LittleEndian.PutUint16(h.HeaderData[3:5], param1) // Load address
	}

	return nil
}

// GetBasicHeader retrieves the BASIC-specific header information
func (h *Plus3DosHeader) GetBasicHeader() (fileType byte, length uint16, param1, param2 uint16) {
	fileType = h.HeaderData[0]
	length = binary.LittleEndian.Uint16(h.HeaderData[1:3])

	switch fileType {
	case FileTypeProgram:
		param1 = binary.LittleEndian.Uint16(h.HeaderData[3:5]) // LINE
		param2 = binary.LittleEndian.Uint16(h.HeaderData[5:7]) // Length
	case FileTypeNumericArray, FileTypeCharArray:
		param1 = uint16(h.HeaderData[3]) // Variable name
	case FileTypeCode:
		param1 = binary.LittleEndian.Uint16(h.HeaderData[3:5]) // Load address
	}

	return
}

// Validate checks if the header is valid
func (h *Plus3DosHeader) Validate() error {
	// Check signature
	if !bytes.Equal(h.Signature[:], []byte(HeaderSignature)) {
		return errors.New("invalid PLUS3DOS header signature")
	}

	// Check soft-EOF
	if h.SoftEOF != HeaderSoftEOF {
		return errors.New("invalid soft-EOF marker")
	}

	// Check version compatibility
	if h.Issue != HeaderIssue {
		return fmt.Errorf("incompatible header issue number: %d", h.Issue)
	}
	if h.Version > HeaderVersion {
		return fmt.Errorf("incompatible header version: %d", h.Version)
	}

	// Validate file type
	fileType := h.HeaderData[0]
	if fileType > FileTypeCode {
		return fmt.Errorf("invalid file type: %d", fileType)
	}

	// Verify checksum
	if !h.verifyChecksum() {
		return errors.New("header checksum verification failed")
	}

	return nil
}

// UpdateChecksum calculates and sets the header checksum
func (h *Plus3DosHeader) UpdateChecksum() {
	var sum uint16 // Use uint16 to handle values > 255
	headerBytes := h.toBytes()
	for i := 0; i < HeaderSize-1; i++ {
		sum += uint16(headerBytes[i]) // Accumulate using uint16
	}
	h.Checksum = byte(sum % 256) // Final checksum fits in a byte
}

// verifyChecksum checks if the current checksum is valid
func (h *Plus3DosHeader) verifyChecksum() bool {
	var sum uint16 // Use uint16 for accumulation
	headerBytes := h.toBytes()
	for i := 0; i < HeaderSize-1; i++ {
		sum += uint16(headerBytes[i]) // Accumulate using uint16
	}
	return byte(sum%256) == h.Checksum // Compare byte-sized checksum
}

// toBytes converts the header to a byte slice
func (h *Plus3DosHeader) toBytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, h)
	return buf.Bytes()
}

// FromBytes populates the header from a byte slice
func (h *Plus3DosHeader) FromBytes(data []byte) error {
	if len(data) < HeaderSize {
		return errors.New("data too short for header")
	}

	buf := bytes.NewReader(data)
	return binary.Read(buf, binary.LittleEndian, h)
}

// GetFileType returns a string description of the file type
func (h *Plus3DosHeader) GetFileType() string {
	switch h.HeaderData[0] {
	case FileTypeProgram:
		return "BASIC Program"
	case FileTypeNumericArray:
		return "Numeric Array"
	case FileTypeCharArray:
		return "Character Array"
	case FileTypeCode:
		return "Code/Screen$"
	default:
		return "Unknown"
	}
}

// String returns a human-readable representation of the header
func (h *Plus3DosHeader) String() string {
	fileType, length, param1, param2 := h.GetBasicHeader()

	typeStr := h.GetFileType()
	info := fmt.Sprintf("Type: %s, Length: %d", typeStr, length)

	switch fileType {
	case FileTypeProgram:
		if param1 != 0 {
			info += fmt.Sprintf(", LINE %d", param1)
		}
		info += fmt.Sprintf(", Program length: %d", param2)
	case FileTypeNumericArray, FileTypeCharArray:
		if param1 != 0 {
			info += fmt.Sprintf(", Variable: %c", param1)
		}
	case FileTypeCode:
		info += fmt.Sprintf(", Load address: %d", param1)
	}

	return info
}
