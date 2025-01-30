// file: pkg/diskimg/host_test.go

package diskimg

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHostIntegration(t *testing.T) {
	disk := NewDiskImage()
	tmpDir := t.TempDir()

	t.Run("BASIC program round trip", func(t *testing.T) {
		// Create test BASIC program
		program := []byte("10 PRINT \"Hello\"\n20 GOTO 10\n")
		srcPath := filepath.Join(tmpDir, "test.bas")
		os.WriteFile(srcPath, program, 0644)

		// Import to disk
		err := disk.ImportBasicProgram(srcPath, 10)
		if err != nil {
			t.Fatal(err)
		}

		// Export back
		outPath := filepath.Join(tmpDir, "out.bas")
		err = disk.ExtractBasic("TEST.BAS", outPath)
		if err != nil {
			t.Fatal(err)
		}

		// Verify data
		data, _ := os.ReadFile(outPath)
		if !bytes.Equal(data, program) {
			t.Error("Program data mismatch after round trip")
		}
	})

	t.Run("Screen$ file handling", func(t *testing.T) {
		// Create test screen data
		screen := make([]byte, 6912)
		for i := range screen {
			screen[i] = byte(i)
		}
		srcPath := filepath.Join(tmpDir, "screen.scr")
		os.WriteFile(srcPath, screen, 0644)

		// Import as screen$
		err := disk.ImportScreen(srcPath)
		if err != nil {
			t.Fatal(err)
		}

		// Verify load address
		f, _ := disk.OpenFile("SCREEN.SCR", false)
		_, _, addr, _ := f.header.GetBasicHeader()
		if addr != 16384 {
			t.Error("Wrong screen load address")
		}
		f.Close()

		// Export and verify
		outPath := filepath.Join(tmpDir, "out.scr")
		err = disk.ExportScreen("SCREEN.SCR", outPath)
		if err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(outPath)
		if !bytes.Equal(data, screen) {
			t.Error("Screen data mismatch")
		}
	})

	t.Run("Format conversion", func(t *testing.T) {
		// Create TAP data
		tapBuf := new(bytes.Buffer)
		content := []byte{0xF3, 0x3C, 0xAF, 0x32} // Sample machine code
		writeTAPBlock(tapBuf, 3, "CODE", content, 32768)

		// Convert to +3DOS
		err := disk.ConvertTAPtoDisk(tapBuf, "CODE.BIN")
		if err != nil {
			t.Fatal(err)
		}

		// Verify header
		f, _ := disk.OpenFile("CODE.BIN", false)
		typ, _, addr, _ := f.header.GetBasicHeader()
		if typ != FileTypeCode || addr != 32768 {
			t.Error("Wrong file type or load address")
		}
		f.Close()

		// Convert back to TAP
		outTAP := new(bytes.Buffer)
		err = disk.ConvertDiskToTAP("CODE.BIN", outTAP)
		if err != nil {
			t.Fatal(err)
		}

		// Verify content in TAP
		block := readTAPBlock(outTAP)
		if !bytes.Equal(block, content) {
			t.Error("Code data mismatch in TAP")
		}
	})
}

// Helper to write TAP block
func writeTAPBlock(w *bytes.Buffer, typ byte, name string, data []byte, param uint16) {
	// Header
	w.Write([]byte{19, 0, typ})
	namePadded := make([]byte, 10)
	copy(namePadded, name)
	w.Write(namePadded)
	binary.Write(w, binary.LittleEndian, uint16(len(data)))
	binary.Write(w, binary.LittleEndian, param)
	w.Write([]byte{0, 0, 0}) // Checksum etc

	// Data
	binary.Write(w, binary.LittleEndian, uint16(len(data)+1))
	w.Write(data)
	w.Write([]byte{0}) // Checksum
}

// Helper to read TAP block
func readTAPBlock(r *bytes.Buffer) []byte {
	r.Next(21) // Skip header
	length := int(binary.LittleEndian.Uint16(r.Next(2))) - 1
	data := make([]byte, length)
	r.Read(data)
	return data
}