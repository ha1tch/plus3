// file: pkg/diskimg/convert.go

package diskimg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// TAPHeader represents a TAP file header block
type TAPHeader struct {
	Type      byte
	Filename  [10]byte
	DataLen   uint16
	Param1    uint16
	Param2    uint16
	Checksum  byte
}

// ConvertTAPtoDisk converts TAP data to +3DOS format
func (di *DiskImage) ConvertTAPtoDisk(tap io.Reader, diskPath string) error {
	// Read header block
	var length uint16
	if err := binary.Read(tap, binary.LittleEndian, &length); err != nil {
		return err
	}

	if length != 19 {
		return errors.New("invalid TAP header block length")
	}

	var header TAPHeader
	if err := binary.Read(tap, binary.LittleEndian, &header); err != nil {
		return err
	}

	// Read data block
	if err := binary.Read(tap, binary.LittleEndian, &length); err != nil {
		return err
	}

	data := make([]byte, length-1) // -1 for checksum
	if _, err := io.ReadFull(tap, data); err != nil {
		return err
	}

	// Skip checksum byte
	_, err := tap.Read(make([]byte, 1))
	if err != nil {
		return err
	}

	// Create +3DOS file
	f, err := di.OpenFile(diskPath, true)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create +3DOS header
	plus3Header := NewPlus3DosHeader()
	switch header.Type {
	case 0: // Program
		err = plus3Header.SetBasicHeader(FileTypeProgram, header.DataLen, header.Param1, header.Param2)
	case 3: // Code
		err = plus3Header.SetBasicHeader(FileTypeCode, header.DataLen, header.Param1, 0)
	default:
		return errors.New("unsupported TAP file type")
	}
	if err != nil {
		return err
	}

	// Write header and data
	if err := f.Write(plus3Header.toBytes()); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}

	return nil
}

// ConvertDiskToTAP converts a +3DOS file to TAP format
func (di *DiskImage) ConvertDiskToTAP(diskPath string, tap io.Writer) error {
	f, err := di.OpenFile(diskPath, false)
	if err != nil {
		return err
	}
	defer f.Close()

	if !f.isHeadered {
		return errors.New("file has no header")
	}

	fileType, length, param1, param2 := f.header.GetBasicHeader()
	
	// Create TAP header block
	header := TAPHeader{
		Type:     byte(fileType),
		DataLen:  length,
		Param1:   param1,
		Param2:   param2,
	}

	// Copy filename
	copy(header.Filename[:], bytes.TrimRight(f.entry.Name[:], " "))

	// Write header block
	binary.Write(tap, binary.LittleEndian, uint16(19))
	binary.Write(tap, binary.LittleEndian, header)
	
	// Calculate and write header checksum
	var checksum byte
	headerBytes := make([]byte, 18)
	binary.LittleEndian.PutUint16(headerBytes[0:], uint16(19))
	copy(headerBytes[2:], bytes.Trim(f.entry.Name[:], " "))
	binary.LittleEndian.PutUint16(headerBytes[12:], length)
	binary.LittleEndian.PutUint16(headerBytes[14:], param1)
	binary.LittleEndian.PutUint16(headerBytes[16:], param2)
	
	for _, b := range headerBytes {
		checksum ^= b
	}
	tap.Write([]byte{checksum})

	// Read file data (skipping header)
	data := make([]byte, length)
	f.Seek(HeaderSize, io.SeekStart)
	if _, err := io.ReadFull(f, data); err != nil {
		return err
	}

	// Write data block
	binary.Write(tap, binary.LittleEndian, uint16(len(data)+1)) // +1 for checksum
	tap.Write(data)

	// Calculate and write data checksum
	checksum = 0
	for _, b := range data {
		checksum ^= b
	}
	tap.Write([]byte{checksum})

	return nil
}