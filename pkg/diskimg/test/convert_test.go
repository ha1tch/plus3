// file: pkg/diskimg/convert_test.go

package diskimg

import (
	"bytes"
	"testing"
)

func TestTAPConversion(t *testing.T) {
	disk := NewDiskImage()

	// Create test TAP data
	tapData := new(bytes.Buffer)
	program := []byte("10 PRINT \"TEST\"")
	
	// Write header block
	tapData.Write([]byte{19, 0}) // Length
	tapData.Write([]byte{0})     // Header type (PROGRAM)
	tapData.Write([]byte("TEST      ")) // Filename
	tapData.Write([]byte{byte(len(program)), 0}) // Length
	tapData.Write([]byte{10, 0})  // LINE parameter
	tapData.Write([]byte{0, 0})   // Unused
	tapData.Write([]byte{0})      // Checksum

	// Write data block
	tapData.Write([]byte{byte(len(program) + 1), 0}) // Length + checksum
	tapData.Write(program)
	tapData.Write([]byte{0}) // Checksum

	// Test TAP to disk conversion
	t.Run("TAP to disk", func(t *testing.T) {
		err := disk.ConvertTAPtoDisk(bytes.NewReader(tapData.Bytes()), "TEST.BAS")
		if err != nil {
			t.Fatalf("Conversion failed: %v", err)
		}

		// Verify file
		f, err := disk.OpenFile("TEST.BAS", false)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		if !f.isHeadered {
			t.Error("Converted file should have header")
		}

		data := make([]byte, len(program))
		f.Seek(HeaderSize, 0)
		f.Read(data)
		if !bytes.Equal(data, program) {
			t.Error("Program data mismatch")
		}
	})

	// Test disk to TAP conversion
	t.Run("Disk to TAP", func(t *testing.T) {
		outTAP := new(bytes.Buffer)
		err := disk.ConvertDiskToTAP("TEST.BAS", outTAP)
		if err != nil {
			t.Fatalf("Conversion failed: %v", err)
		}

		// Basic validation of TAP format
		headerLen := outTAP.Next(2)
		if !bytes.Equal(headerLen, []byte{19, 0}) {
			t.Error("Invalid header block length")
		}

		fileType := outTAP.Next(1)
		if fileType[0] != 0 {
			t.Error("Wrong file type")
		}

		// Skip to data block and verify
		outTAP.Next(16) // Skip rest of header
		dataLen := outTAP.Next(2)
		if !bytes.Equal(dataLen, []byte{byte(len(program) + 1), 0}) {
			t.Error("Invalid data block length")
		}

		data := make([]byte, len(program))
		outTAP.Read(data)
		if !bytes.Equal(data, program) {
			t.Error("Program data mismatch in TAP output")
		}
	})
}